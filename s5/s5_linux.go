package s5

import "net"
import "syscall"
import "fmt"
import "time"

func CreateDialer(bind net.Addr, fwmark int, dscp int) *net.Dialer {

	// callback from dialer creation
	var ctrlCallback = func(network, address string, c syscall.RawConn) error {
		return c.Control(func(fd uintptr) {
			var err error
			if fwmark != 0 {
				err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_MARK, fwmark)
				if err != nil {
					fmt.Println("set fwmark failed:", err)
				}
			}
			if dscp != 0 {
				err = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TOS, dscp << 2)
				if err != nil {
					fmt.Println("set dscp failed:", err)
				}
			}
		})
	}

	dialer := &net.Dialer{
		LocalAddr: bind,
		Timeout: time.Duration(time.Second*30),
	}

	dialer.Control = ctrlCallback

	return dialer
}
