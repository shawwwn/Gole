package main

import (
	"os"
	"net"
	"fmt"
	"time"
	"math/rand"
	"sync"
	"context"
	"errors"

	"golang.org/x/net/ipv4"
)

// wrapper for PunchTCP() and PunchUDP()
func Punch(conf Config) (net.Conn, error) {
	var conn net.Conn
	var err error
	switch conf.getMode() {
	case "tcp":
		conn, err = PunchTCP(conf.(*TCPConfig))
	case "udp":
		conn, err = PunchUDP(conf.(*UDPConfig))
	}
	return conn, err
}

func PunchTCP(conf *TCPConfig) (net.Conn, error) {
	var conn net.Conn
	var err error
	var thru bool = false

	// ~2mins timeout on retries
	for i:=0; i<60; i++ {
		conn, err = net.DialTCP("tcp", conf.LAddr, conf.RAddr)
		if (err != nil) {
			ms := 1000+rand.Intn(2000)
			perror(fmt.Sprintf("connect: failed, retry in %.2fs.", float32(ms)/1000), err);
			time.Sleep(time.Duration(ms)*time.Millisecond);
			continue
		}

		// encrypt socket
		if conf.Key != "" {
			conn = NewEConn(conn, conf.Enc, conf.Key)
		}

		msg := fmt.Sprintf("HELO-%d", os.Getpid())
		PrintDbgf("send: %s\n", msg);
		_, err = conn.Write([]byte(msg))
		if (err != nil) {
			perror("send() failed.", err)
			return nil, err
		}

		data := make([]byte, 1024)
		n, err := conn.Read(data)
		if err != nil {
			perror("recv() failed.", err)
			return nil, err
		}
		fmt.Printf("recv: %s\n", data[:n])

		// check authentication
		if string(data[:4]) != "HELO" {
			conn.Close()
			return nil, errors.New("auth failed")
		}

		thru = true
		break
	}

	if !thru {
		return nil, errors.New("timeout punching holes")
	}
	return conn, err
}

func sendMsgUDP(conn net.PacketConn, msg string, to_addr net.Addr) error {
	PrintDbgf("send: %s\n", msg);
	_, err := conn.WriteTo([]byte(msg), to_addr)
	if (err != nil) {
		perror("send() failed.", err)
	}
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	return err
}

func PunchUDP(conf *UDPConfig) (net.Conn, error) {
	var conn net.PacketConn
	var err error
	var ttl0 int
	var opts *ipv4.Conn

	// conn, err = net.DialUDP("udp", conf.LAddr, conf.RAddr)
	conn, err = net.ListenUDP("udp", conf.LAddr)
	if err != nil {
		perror("net.DialUDP() failed.", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	var fail error = nil
	recv_done := make(chan struct{})
	ctx1, sendDone := context.WithCancel(context.Background())

	// set ttl
	if conf.TTL != 0 {
		opts = ipv4.NewConn(conn.(*net.UDPConn))
		ttl0, err = opts.TTL()
		if err != nil {
			perror("Get TTL failed.", err)
		}
		err = opts.SetTTL(conf.TTL)
		if err != nil {
			perror("Set TTL failed.", err)
		} else {
			fmt.Printf("Set ttl to %d\n", conf.TTL)
		}
	}

	// encrypt socket
	if conf.Key != "" {
		switch conn.(type) {
		case *net.UDPConn:
			conn = NewEPacketConn(conn, conf.Enc, conf.Key)
		}
	}

	// sender
	msg := fmt.Sprintf("HELO-%d", os.Getpid())
	wg.Add(1)
	go func() {
		defer PrintDbgf("sender stopped\n")
		defer wg.Done()

		// ~2mins timeout on retries
		for i:=1; i<=60; i++ {
			err = sendMsgUDP(conn, msg, conf.RAddr)
			ms := time.Duration(1000+rand.Intn(2000))
			select {
			case <-time.After(ms*time.Millisecond):
			case <-ctx1.Done():
				return
			}
			if i == 60 {
				PrintDbgf("send: failed.\n")
				fail = errors.New("timeout punching holes")
				close(recv_done)
				conn.SetReadDeadline(time.Now())
				return
			} else {
				PrintDbgf("send: timeout, retry\n")
			}
		}
	}()

	// receiver
	wg.Add(2)
	go func() {
		helo, okay := false, false
		defer func() {
			PrintDbgf("receiver stopped\n")
			if ! helo { wg.Done() }
			if ! okay { wg.Done() }
			conn.SetReadDeadline(time.Time{})
		}()

		for {
			data := make([]byte, 1024)
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, raddr, err := conn.ReadFrom(data)
			select {
			case <-recv_done:
				return
			default:
			}
			if raddr.String() != conf.RAddr.String() {
				fmt.Printf("ignore unsolicited message from %s\n", raddr)
				continue
			}
			if err != nil {
				perror("recv: failed.", err)
				fail = err
				sendDone()
				return
			}
			fmt.Printf("recv: %s\n", data[:n])

			if n < 4 {
				continue
			}

			// check authentication
			if !contains(string(data[:4]), []string{"HELO", "OKAY"}) {
				fail = errors.New("auth failed")
				sendDone()
				return
			}

			// restore ttl
			if conf.TTL != 0 {
				err = opts.SetTTL(ttl0)
				if err != nil {
					perror("Restore TTL failed.", err)
				}
			}

			switch string(data[:4]) {
			case "HELO":
				sendMsgUDP(conn, "OKAY", conf.RAddr)
				if ! helo {
					helo = true
					wg.Done()
				}
			case "OKAY":
				sendDone()
				if ! okay {
					okay = true
					wg.Done()
				}
			}
		} // for
	}()

	wg.Wait()
	close(recv_done) // stop receiver
	conn.SetReadDeadline(time.Now()) // cancel ReadFrom()
	time.Sleep(10)
	PrintDbgf("Wait for remaining packets to clear ...\n")
	time.Sleep(2*time.Second) // wait for packets to clear
	if fail != nil {
		conn.Close()
		return nil, fail
	}
	return conn.(net.Conn), nil
}


