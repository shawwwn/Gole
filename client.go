package main

import (
	"fmt"
	"os"
	"net"
	"time"

	kcp "github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

// wrapper for StartClientTCP(), StartClientKCP(), and StartClientUDP()
func StartClient(conn net.Conn, conf Config) {
	switch conf.getMode() {
	case "tcp":
		StartClientTCP(conn, conf.(*TCPConfig))
	case "udp":
		if conf.(*UDPConfig).Proto == "kcp" {
			StartClientKCP(conn.(net.PacketConn), conf.(*UDPConfig))
		} else {
			StartClientUDP(conn.(net.PacketConn), conf.(*UDPConfig))
		}
	}
}

func StartClientTCP(conn net.Conn, conf *TCPConfig) {

	// encrypt socket
	if conf.Key != "" {
		switch conn.(type) {
		case *net.TCPConn:
			conn = NewEConn(conn, conf.Enc, conf.Key)
		}
	}

	// Setup client side of smux
	var interval int = g_timeout/3
	interval = bound(interval, 1, 10)
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = 1
	smuxConfig.MaxReceiveBuffer = 4194304
	smuxConfig.MaxStreamBuffer = 2097152
	smuxConfig.KeepAliveInterval = time.Duration(interval) * time.Second
	smuxConfig.KeepAliveTimeout = time.Duration(g_timeout) * time.Second
	if err := smux.VerifyConfig(smuxConfig); err != nil {
		perror("smux.VerifyConfig() failed.", err)
		os.Exit(1)
	}

	sess, err := smux.Client(conn, smuxConfig)
	if err != nil {
		perror("smux.Client() failed.", err)
		os.Exit(1)
	}
	fmt.Printf("tunnel created: [local]%v <--> [remote]%v\n", sess.LocalAddr(), sess.RemoteAddr())

	// listen for connections from forward address
	lis, err := net.ListenTCP("tcp", conf.FwdAddr.(*net.TCPAddr))
	if err != nil {
		perror("net.Listen() failed.", err)
		os.Exit(1)
	}
	defer lis.Close()
	fmt.Printf("Waiting for new connections from %s ...\n", conf.FwdAddr.String())

	// periodic check if smux session is still alive
	go func() {
		for {
			time.Sleep(2*time.Second)
			if sess.IsClosed() {
				lis.Close()
				fmt.Printf("tunnel is closed\n")
				break
			}
		}
	}()

	for {
		fwd_conn, err := lis.Accept()
		if err != nil {
			perror("lis.Accept() failed.", err)
			break
		}

		stream, err := sess.OpenStream()
		if err != nil {
			perror("sess.OpenStream() failed.", err)
			fwd_conn.Close()
			break
		}
		PrintDbgf("stream open(%d): %v --> tunnel\n", stream.ID(), fwd_conn.RemoteAddr())

		go conn2stream(fwd_conn, stream)
	}

	// clean up
	fmt.Printf("...\n")
	fmt.Printf("tunnel collapsed: [local]%v <--> [remote]%v\n", sess.LocalAddr(), sess.RemoteAddr())
	sess.Close()
	conn.Close()
	time.Sleep(time.Second)
}

func StartClientKCP(conn net.PacketConn, conf *UDPConfig) {

	// encrypt socket
	if conf.Key != "" {
		switch conn.(type) {
		case *net.UDPConn:
			conn = NewEPacketConn(conn, conf.Enc, conf.Key)
		}
	}

	// setup kcp
	kconf := getKCPConfig(conf.KConf)
	PrintDbgf("%T: %v\n", kconf, kconf)
	block := getKCPBlockCipher(kconf)
	kconn, err := kcp.NewConn2(conf.RAddr, block, kconf.DataShard, kconf.ParityShard, conn)
	if err != nil {
		perror("kcp.NewConn2() failed.", err)
		os.Exit(1)
	}

	kconn.SetStreamMode(true)
	kconn.SetWriteDelay(false)
	kconn.SetNoDelay(kconf.NoDelay, kconf.Interval, kconf.Resend, kconf.NoCongestion)
	kconn.SetMtu(kconf.MTU)
	kconn.SetWindowSize(kconf.SndWnd, kconf.RcvWnd)
	kconn.SetACKNoDelay(kconf.AckNodelay)
	if err := SetDSCP(conn.(net.Conn), kconf.DSCP); err != nil {
		perror("SetDSCP() failed.", err)
	}
	if err := kconn.SetReadBuffer(kconf.SockBuf); err != nil {
		perror("kconn.SetReadBuffer() failed.", err)
	}
	if err := kconn.SetWriteBuffer(kconf.SockBuf); err != nil {
		perror("kconn.SetWriteBuffer() failed.", err)
	}
	kconn.Write([]byte{1,3,0,0,0,0,0,0}) // smux cmdNOP, let remote know we are connected

	// Setup client side of smux
	var interval int = g_timeout/3
	interval = bound(interval, 1, 10)
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = 1
	smuxConfig.MaxReceiveBuffer = 4194304
	smuxConfig.MaxStreamBuffer = 2097152
	smuxConfig.KeepAliveInterval = time.Duration(interval) * time.Second
	smuxConfig.KeepAliveTimeout = time.Duration(g_timeout) * time.Second
	if err := smux.VerifyConfig(smuxConfig); err != nil {
		perror("smux.VerifyConfig() failed.", err)
		os.Exit(1)
	}

	sess, err := smux.Client(kconn, smuxConfig)
	if err != nil {
		perror("smux.Client() failed.", err)
		os.Exit(1)
	}
	fmt.Printf("tunnel created: [local]%v <--> [remote]%v\n", sess.LocalAddr(), sess.RemoteAddr())

	// listen from forward
	lis, err := net.ListenTCP("tcp", conf.FwdAddr.(*net.TCPAddr))
	if err != nil {
		perror("net.Listen() failed.", err)
		os.Exit(1)
	}
	defer lis.Close()
	fmt.Printf("Waiting for new connections from %s ...\n", conf.FwdAddr.String())

	// periodic check if smux session is still alive
	go func() {
		for {
			time.Sleep(2*time.Second)
			if sess.IsClosed() {
				lis.Close()
				fmt.Printf("tunnel is closed\n")
				break
			}
		}
	}()

	for {
		fwd_conn, err := lis.Accept()
		if err != nil {
			perror("lis.Accept() failed.", err)
			break
		}

		stream, err := sess.OpenStream()
		if err != nil {
			perror("sess.OpenStream() failed.", err)
			fwd_conn.Close()
			break
		}
		PrintDbgf("stream open(%d): %v --> tunnel\n", stream.ID(), fwd_conn.RemoteAddr())

		go conn2stream(fwd_conn, stream)
	} // AcceptTCP()

	// clean up
	fmt.Printf("...\n")
	fmt.Printf("tunnel collapsed: [local]%v <--> [remote]%v\n", sess.LocalAddr(), sess.RemoteAddr())
	sess.Close()
	kconn.Close()
	conn.Close()
	time.Sleep(time.Second)
}

func StartClientUDP(conn net.PacketConn, conf *UDPConfig) {

	// encrypt socket
	if conf.Key != "" {
		switch conn.(type) {
		case *net.UDPConn:
			conn = NewEPacketConn(conn, conf.Enc, conf.Key)
		}
	}

	fmt.Printf("tunnel created: [local]%v <--> [remote]%v\n", conf.LocalAddr(), conf.RemoteAddr())
	fmt.Printf("Listen on forward address: %s\n", conf.FwdAddr)

	fwd_conn, err := net.ListenUDP("udp", conf.FwdAddr.(*net.UDPAddr))
	if err != nil {
		perror("net.ListenUDP() failed.", err)
		os.Exit(1)
	}

	var client_addr *net.UDPAddr // address originates from client app
	var sent chan struct{} = make(chan struct{}, 1)

	// TODO: Below code for udp forwarding is just a prototype and should never 
	//       be used in a production system as it only allows one connection.
	//       Needs udp muxing.

	// conn --> fwd_conn
	go func() {
		defer conn.Close()
		defer fwd_conn.Close()
		buf := make([]byte, 4096)
		for {
			conn.SetDeadline(time.Now().Add(time.Duration(g_timeout) * time.Second))
			n, _, err := conn.ReadFrom(buf)
			if err != nil {
				fmt.Println("conn.ReadFromUDP() failed.", err)
				return
			}
			<-sent // wait until client first send something
			_, err = fwd_conn.WriteToUDP(buf[:n], client_addr)
			if err != nil {
				fmt.Println("fwd_conn.WriteToUDP() failed.", err)
				return
			}
		}
	}()

	// fwd_conn --> conn
	func() {
		defer conn.Close()
		defer fwd_conn.Close()
		buf := make([]byte, 4096)
		for {
			n, c_addr, err := fwd_conn.ReadFromUDP(buf)
			if err != nil {
				fmt.Println("fwd_conn.ReadFromUDP() failed.", err)
				return
			}

			// use the most recent client address
			if client_addr==nil  {
				client_addr = c_addr
				PrintDbgf("connection from: %s\n", client_addr)
				close(sent)
			} else if !UDPAddrEqual(c_addr, client_addr) {
				client_addr = c_addr
				PrintDbgf("new connection from: %s\n", client_addr)
			}

			_, err = conn.WriteTo(buf[:n], conf.RAddr)
			if err != nil {
				fmt.Println("conn.Write() failed.", err)
				return
			}
			conn.SetDeadline(time.Now().Add(time.Duration(g_timeout) * time.Second))
		}
	}()

	fmt.Println("...")
	fmt.Printf("tunnel collapsed: [local]%v <--> [remote]%v\n", conf.LocalAddr(), conf.RemoteAddr())
	time.Sleep(time.Second)
}
