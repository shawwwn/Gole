package main

import (
	"fmt"
	"os"
	"io"
	"net"
	"bytes"

	"golang.org/x/net/ipv4"
	"github.com/xtaci/smux"
)

func perror(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
}

func contains(s string, ss []string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func bound(val int, low int , high int) int {
	if val < low {
		val = low
	} else if val > high {
		val = high
	}
	return val
}

func PrintDbgf(format string, a ...interface{}) (n int, err error) {
	if g_verbose {
		return fmt.Printf(format, a...)
	}
	return 0, nil
}

// Forward data between @conn and @stream util one of them
// calls close() or error out.
func conn2stream(conn net.Conn, stream *smux.Stream) {
	var n_recv = make(chan int64, 1)
	var n_send = make(chan int64, 1)

	var copyIO = func(dst, src io.ReadWriteCloser, count chan int64) {
		defer src.Close()
		defer dst.Close()
		buf := make([]byte, 4096)
		c, err := io.CopyBuffer(dst, src, buf)
		count<- c
		if err != nil {
			return
		}
	}

	go copyIO(stream, conn, n_send)
	go copyIO(conn, stream, n_recv)

	PrintDbgf("stream close(%d): send:%d, recv:%d\n", stream.ID(), <- n_send, <- n_recv)
}

// Forward data between @conn and @stream util one of them
// calls close() or error out.
func stream2conn(stream *smux.Stream, conn net.Conn) {
	var n_recv = make(chan int64, 1)
	var n_send = make(chan int64, 1)

	var copyIO = func (dst, src io.ReadWriteCloser, count chan int64) {
		defer src.Close()
		defer dst.Close()
		buf := make([]byte, 4096)
		c, err := io.CopyBuffer(dst, src, buf)
		count<- c
		if err != nil {
			return
		}
	}

	go copyIO(stream, conn, n_send)
	go copyIO(conn, stream, n_recv)

	PrintDbgf("stream close(%d): send:%d, recv:%d\n", stream.ID(), <- n_send, <- n_recv)
}

func conn2conn(fwd_conn net.Conn, conn net.Conn) {
	var n_recv = make(chan int64, 1)
	var n_send = make(chan int64, 1)

	var copyIO = func (dst, src io.ReadWriteCloser, count chan int64) {
		defer src.Close()
		defer dst.Close()
		c, err := io.Copy(dst, src)
		count<- c
		if err != nil {
			fmt.Println(err)
			return 
		}
	}

	go copyIO(conn, fwd_conn, n_recv)
	go copyIO(fwd_conn, conn, n_send)

	fmt.Printf("stream close: send:%d, recv:%d\n", <- n_send, <- n_recv)
}

// check if two UDPAddr is equal
func UDPAddrEqual(a, b *net.UDPAddr) bool {
	return a.Port==b.Port && bytes.Equal(a.IP, b.IP) && a.Zone==b.Zone
}

// setDSCP for EConn and EPacketConn
func SetDSCP(conn net.Conn, dscp int) error {
	switch nc := conn.(type) {
	case *EConnXor:
		return ipv4.NewConn(nc.Conn()).SetTOS(dscp << 2)
	case *EPacketConnXor:
		return ipv4.NewConn(nc.Conn()).SetTOS(dscp << 2)
	}
	return ipv4.NewConn(conn).SetTOS(dscp << 2)
}
