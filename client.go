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
		StartClientTCP(conn.(*net.TCPConn), conf.(*TCPConfig))
	case "udp":
		if conf.(*UDPConfig).Proto == "kcp" {
			StartClientKCP(conn.(*net.UDPConn), conf.(*UDPConfig))
		} else {
			StartClientUDP(conn.(*net.UDPConn), conf.(*UDPConfig))
		}
	}
}

func StartClientTCP(conn *net.TCPConn, conf *TCPConfig) {

	// Setup client side of smux
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = 1
	smuxConfig.MaxReceiveBuffer = 4194304
	smuxConfig.MaxStreamBuffer = 2097152
	// smuxConfig.KeepAliveInterval = time.Duration(2) * time.Second
	// smuxConfig.KeepAliveTimeout = time.Duration(10) * time.Second
	if err := smux.VerifyConfig(smuxConfig); err != nil {
		perror("smux.VerifyConfig() failed.", err)
		os.Exit(1)
	}

	session, err := smux.Client(conn, smuxConfig)
	if err != nil {
		perror("smux.Client() failed.", err)
		os.Exit(1)
	}
	fmt.Printf("tunnel created: [local]%v <--> [remote]%v\n", session.LocalAddr(), session.RemoteAddr())

	// listen for connections from forward address
	listener, err := net.ListenTCP("tcp", conf.FwdAddr.(*net.TCPAddr))
	if err != nil {
		perror("net.Listen() failed.", err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Printf("Waiting for new connections from %s ...\n", conf.FwdAddr.String())

	// periodic check if smux.Client is still alive
	go func() {
		for {
			time.Sleep(2*time.Second)
			if session.IsClosed() {
				listener.Close()
				fmt.Printf("tunnel is closed\n")
				break
			}
		}
	}()

	for {
		fwd_conn, err := listener.Accept()
		if err != nil {
			perror("listener.Accept() failed.", err)
			break
		}

		stream, err := session.OpenStream()
		if err != nil {
			perror("session.OpenStream() failed.", err)
			fwd_conn.Close()
			break
		}
		fmt.Printf("stream open(%d): %v --> tunnel\n", stream.ID(), fwd_conn.RemoteAddr())

		go conn2stream(fwd_conn, stream)
	}

	// clean up
	fmt.Printf("...\n")
	fmt.Printf("tunnel collapsed: [local]%v <--> [remote]%v\n", session.LocalAddr(), session.RemoteAddr())
	session.Close()
	conn.Close()
	time.Sleep(time.Second)
	fmt.Printf("Done\n")
}

func StartClientKCP(conn *net.UDPConn, conf *UDPConfig) {

	// setup kcp
	kconf := getKCPConfig(conf.KConf)
	fmt.Printf("%T: %v\n", kconf, kconf)
	block := getKCPBlockCipher(kconf)
	kconn, err := kcp.NewConn2(conf.RAddr, block, kconf.DataShard, kconf.ParityShard, conn)
	if err != nil {
		perror("kcp.NewConn2() failed.", err)
		os.Exit(1)
	}
	fmt.Println("kcp remote address:", kconn.RemoteAddr())
	kconn.SetStreamMode(true)
	kconn.SetWriteDelay(false)
	kconn.SetNoDelay(kconf.NoDelay, kconf.Interval, kconf.Resend, kconf.NoCongestion)
	kconn.SetMtu(kconf.MTU)
	kconn.SetWindowSize(kconf.SndWnd, kconf.RcvWnd)
	kconn.SetACKNoDelay(kconf.AckNodelay)
	if err := kconn.SetDSCP(kconf.DSCP); err != nil {
		perror("kconn.SetDSCP() failed.", err)
	}
	if err := kconn.SetReadBuffer(kconf.SockBuf); err != nil {
		perror("kconn.SetReadBuffer() failed.", err)
	}
	if err := kconn.SetWriteBuffer(kconf.SockBuf); err != nil {
		perror("kconn.SetWriteBuffer() failed.", err)
	}

	// Setup client side of smux
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = 1
	smuxConfig.MaxReceiveBuffer = 4194304
	smuxConfig.MaxStreamBuffer = 2097152
	// smuxConfig.KeepAliveInterval = time.Duration(2) * time.Second
	// smuxConfig.KeepAliveTimeout = time.Duration(10) * time.Second
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
		fmt.Printf("stream open(%d): %v --> tunnel\n", stream.ID(), fwd_conn.RemoteAddr())

		go conn2stream(fwd_conn, stream)
	} // AcceptTCP()

	// clean up
	fmt.Printf("...\n")
	fmt.Printf("tunnel collapsed: [local]%v <--> [remote]%v\n", sess.LocalAddr(), sess.RemoteAddr())
	sess.Close()
	kconn.Close()
	conn.Close()
	time.Sleep(time.Second)
	fmt.Printf("Done\n")
}

func StartClientUDP(conn *net.UDPConn, conf *UDPConfig) {
	fmt.Printf("Listen on fwd address: %s\n", conf.FwdAddr)
	fwd_conn, err := net.ListenUDP("udp", conf.FwdAddr.(*net.UDPAddr))
	if err != nil {
		perror("net.ListenUDP() failed.", err)
		os.Exit(1)
	}

	// recreate socket with sendto() on tunnel endpoints
	conn.Close()
	conn, err = net.DialUDP("udp", conf.LAddr, conf.RAddr)

	var client_addr *net.UDPAddr // address originates from client app
	var sent chan struct{} = make(chan struct{}, 1)

	// TODO: Below code for udp forwarding is just a prototype and should never 
	//       be used in a production system as it only allows one connection.
	//       Needs udp mux.

	// conn --> fwd_conn
	go func() {
		defer conn.Close()
		defer fwd_conn.Close()
		buf := make([]byte, 4096)
		for {
			conn.SetDeadline(time.Now().Add(time.Duration(g_timeout) * time.Second))
			n, _, err := conn.ReadFromUDP(buf)
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
				fmt.Printf("connection from: %s\n", client_addr)
				close(sent)
			} else if !UDPAddrEqual(c_addr, client_addr) {
				client_addr = c_addr
				fmt.Printf("new connection from: %s\n", client_addr)
			}

			_, err = conn.Write(buf[:n])
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
	fmt.Printf("Done\n")
}
