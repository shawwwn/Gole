package s5

import "net"
import "syscall"
import "fmt"
import "time"

func CreateDialer(bind net.Addr, fwmark int, dscp int) *net.Dialer {

	// callback from dialer creation
	var controlCallback = func(network, address string, c syscall.RawConn) error {
		return c.Control(func(fd uintptr) {
			f := int(fd)
			// log.Printf("open %s socket (fd=%d)", network, f)
			err := syscall.SetsockoptInt(f, syscall.SOL_SOCKET, syscall.SO_MARK, fwmark)
			if err != nil {
				fmt.Println("set fwmark failed:", err)
			}
		})
	}

	dialer := &net.Dialer{
		LocalAddr: bind,
		Timeout: time.Duration(time.Second*30),
	}

	if fwmark != 0 {
		dialer.Control = controlCallback
	}

	return dialer
}
