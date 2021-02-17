package main

import (
	"errors"
	"time"
	"net"
	"syscall"
	"crypto/sha1"

	xor "github.com/templexxx/xorsimd"
	"golang.org/x/crypto/pbkdf2"
)



type EConnXor struct {
	conn net.Conn
	key []byte
	key_ri int // key read index
	key_wi int // key write index
}
func (econn *EConnXor) Conn() net.Conn {
	return econn.conn
}
func (econn *EConnXor) Read(b []byte) (n int, err error) {
	n, err = econn.conn.Read(b)

	// stream decrypt
	sz := n
	for i:=0; i<sz; {
		ct := xor.Bytes(b[i:n], b[i:n], econn.key[econn.key_ri:])
		if ct == 0 {
			break
		}
		econn.key_ri = (econn.key_ri+ct) % len(econn.key)
		i += ct
	}

	return n, err
}
func (econn *EConnXor) Write(b []byte) (n int, err error) {

	// stream encrypt
	sz := len(b)
	for i:=0; i<sz; {
		ct := xor.Bytes(b[i:], b[i:], econn.key[econn.key_wi:])
		if ct == 0 {
			break
		}
		econn.key_wi = (econn.key_wi+ct) % len(econn.key)
		i += ct
	}

	return econn.conn.Write(b)
}
func (econn *EConnXor) Close() error {
	return econn.conn.Close()
}
func (econn *EConnXor) LocalAddr() net.Addr {
	return econn.conn.LocalAddr()
}
func (econn *EConnXor) RemoteAddr() net.Addr {
	return econn.conn.RemoteAddr()
}
func (econn *EConnXor) SetDeadline(t time.Time) error {
	return econn.conn.SetDeadline(t)
}
func (econn *EConnXor) SetReadDeadline(t time.Time) error {
	return econn.conn.SetReadDeadline(t)
}
func (econn *EConnXor) SetWriteDeadline(t time.Time) error {
	return econn.conn.SetWriteDeadline(t)
}
func (econn *EConnXor) SetReadBuffer(bytes int) error {
	if nc, ok := econn.conn.(*net.TCPConn); ok {
		return nc.SetReadBuffer(bytes)
	}
	return errors.New("not implemented")
}
func (econn *EConnXor) SetWriteBuffer(bytes int) error {
	if nc, ok := econn.conn.(*net.TCPConn); ok {
		return nc.SetWriteBuffer(bytes)
	}
	return errors.New("not implemented")
}
func (econn *EConnXor) SyscallConn() (syscall.RawConn, error) {
	if nc, ok := econn.conn.(*net.TCPConn); ok {
		return nc.SyscallConn()
	}
	return nil, errors.New("not implemented")
}

type EPacketConnXor struct {
	conn net.PacketConn
	key []byte
}
func (econn *EPacketConnXor) Conn() net.Conn {
	if nc, ok := econn.conn.(net.Conn); ok {
		return nc
	}
	return nil
}
func (econn *EPacketConnXor) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, addr, err = econn.conn.ReadFrom(b)
	xor.Bytes(b, b[:n], []byte(econn.key))
	return n, addr, err
}
func (econn *EPacketConnXor) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	xor.Bytes(b, b, []byte(econn.key))
	return econn.conn.WriteTo(b, addr)
}
func (econn *EPacketConnXor) Close() error {
	return econn.conn.Close()
}
func (econn *EPacketConnXor) LocalAddr() net.Addr {
	return econn.conn.LocalAddr()
}
func (econn *EPacketConnXor) SetDeadline(t time.Time) error {
	return econn.conn.SetDeadline(t)
}
func (econn *EPacketConnXor) SetReadDeadline(t time.Time) error {
	return econn.conn.SetReadDeadline(t)
}
func (econn *EPacketConnXor) SetWriteDeadline(t time.Time) error {
	return econn.conn.SetWriteDeadline(t)
}
func (econn *EPacketConnXor) SetReadBuffer(bytes int) error {
	if nc, ok := econn.conn.(*net.UDPConn); ok {
		return nc.SetReadBuffer(bytes)
	}
	return errors.New("not implemented")
}
func (econn *EPacketConnXor) SetWriteBuffer(bytes int) error {
	if nc, ok := econn.conn.(*net.UDPConn); ok {
		return nc.SetWriteBuffer(bytes)
	}
	return errors.New("not implemented")
}
func (econn *EPacketConnXor) SyscallConn() (syscall.RawConn, error) {
	if nc, ok := econn.conn.(*net.UDPConn); ok {
		return nc.SyscallConn()
	}
	return nil, errors.New("not implemented")
}

// for compatibility with net.Conn
func (econn *EPacketConnXor) Read(b []byte) (n int, err error) {
	conn, ok := interface{}(econn.conn).(net.Conn)
	if ok {
		n, err = conn.Read(b)
		xor.Bytes(b, b[:n], []byte(econn.key))
		return n, err
	}
	return 0, errors.New("not implemented")
}
func (econn *EPacketConnXor) Write(b []byte) (n int, err error) {
	conn, ok := interface{}(econn.conn).(net.Conn)
	if ok {
		xor.Bytes(b, b, []byte(econn.key))
		return conn.Write(b)
	}
	return 0, errors.New("not implemented")
}
func (econn *EPacketConnXor) RemoteAddr() net.Addr {
	conn, ok := interface{}(econn.conn).(net.Conn)
	if ok {
		return conn.RemoteAddr()
	}
	return nil
}


func NewEConn(conn net.Conn, enc, key string) net.Conn {
	k := pbkdf2.Key([]byte(key), []byte("saltybiscuit"), 64, 4096, sha1.New)
	return &EConnXor{conn, k, 0, 0}
}

func NewEPacketConn(conn net.PacketConn, enc, key string) net.PacketConn {
	k := pbkdf2.Key([]byte(key), []byte("saltybiscuit"), 64, 4096, sha1.New)
	return &EPacketConnXor{conn, k}
}
