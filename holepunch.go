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
		c := conf.(*TCPConfig)
		conn, err = PunchTCP(c.LAddr, c.RAddr)
	case "udp":
		c := conf.(*UDPConfig)
		conn, err = PunchUDP(c.LAddr, c.RAddr)
	}
	return conn, err
}

func PunchTCP(laddr, raddr *net.TCPAddr) (*net.TCPConn, error) {
	var conn *net.TCPConn
	var err error
	var thru bool = false

	// ~2mins timeout
	for i:=0; i<60; i++ {
		conn, err = net.DialTCP("tcp", laddr, raddr)
		if (err != nil) {
			ms := 1000+rand.Intn(2000)
			perror(fmt.Sprintf("connect: failed, retry in %.2fs.", float32(ms)/1000), err);
			time.Sleep(time.Duration(ms)*time.Millisecond);
			continue
		}

		msg := fmt.Sprintf("HELO-%d", os.Getpid())
		fmt.Printf("send: %s\n", msg);
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

		thru = true
		break
	}

	if !thru {
		return nil, errors.New("timeout punching holes")
	}
	return conn, err
}



// simple udp
type sudp struct {
	conn net.UDPConn
}

func sendMsgUDP(conn *net.UDPConn, msg string, to_addr *net.UDPAddr) error {
	fmt.Printf("send: %s\n", msg);
	_, err := conn.WriteToUDP([]byte(msg), to_addr)
	if (err != nil) {
		perror("send() failed.", err)
	}
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	return err
}


func PunchUDP(laddr, raddr *net.UDPAddr, ttl ...int) (*net.UDPConn, error) {
	var conn *net.UDPConn
	var err error
	var ttl0 int
	var opts *ipv4.Conn

	// conn, err = net.DialUDP("udp", laddr, raddr)
	conn, err = net.ListenUDP("udp", laddr)
	if err != nil {
		perror("net.DialUDP() failed.", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	var fail bool = false
	recv_done := make(chan struct{})
	ctx1, sendDone := context.WithCancel(context.Background())

	// set ttl
	if len(ttl) != 0 {
		opts = ipv4.NewConn(conn)
		ttl0, err = opts.TTL()
		if err != nil {
			perror("Get TTL failed.", err)
		}
		err = opts.SetTTL(ttl[0])
		if err != nil {
			perror("Set TTL failed.", err)
		} else {
			fmt.Printf("Set ttl to %d\n", ttl[0])
		}
	}

	// sender
	msg := fmt.Sprintf("HELO-%d", os.Getpid())
	wg.Add(1)
	go func() {
		defer fmt.Println("sender stopped")
		defer wg.Done()

		// ~2mins timeout
		for i:=1; i<=60; i++ {
			err = sendMsgUDP(conn, msg, raddr)
			ms := time.Duration(1000+rand.Intn(2000))
			select {
			case <-time.After(ms*time.Millisecond):
			case <-ctx1.Done():
				return
			}
			if i == 60 {
				fmt.Println("send: failed.")
				fail = true
				close(recv_done)
				conn.SetReadDeadline(time.Now())
				return
			} else {
				fmt.Println("send: timeout, retry")
			}
		}
	}()

	// receiver
	wg.Add(2)
	go func() {
		helo, okay := false, false
		defer func() {
			fmt.Println("receiver stopped")
			if ! helo { wg.Done() }
			if ! okay { wg.Done() }
			conn.SetReadDeadline(time.Time{})
		}()

		for {
			data := make([]byte, 1024)
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, _, err := conn.ReadFromUDP(data)
			select {
			case <-recv_done:
				return
			default:
			}
			if err != nil {
				perror("recv: failed.", err)
				fail = true
				sendDone()
				return
			}
			fmt.Printf("recv: %s\n", data[:n])

			if n < 4 {
				continue
			}

			// restore ttl
			if len(ttl) != 0 {
				err = opts.SetTTL(ttl0)
				if err != nil {
					perror("Restore TTL failed.", err)
				}
			}

			switch string(data[:4]) {
			case "HELO":
				sendMsgUDP(conn, "OKAY", raddr)
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
	conn.SetReadDeadline(time.Now()) // cancel ReadFromUDP()
	time.Sleep(10)
	fmt.Println("Wait for remaining packets to clear ...")
	time.Sleep(2*time.Second) // wait for packets to clear
	if fail {
		return nil, errors.New("timeout punching holes")
	}
	return conn, nil
}


