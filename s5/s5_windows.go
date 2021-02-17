package s5

import "net"
import "time"
import "syscall"
import "fmt"
import "golang.org/x/sys/windows"

func CreateDialer(bind net.Addr, fwmark int, dscp int) *net.Dialer {

	// callback from dialer creation
	var ctrlCallback = func(network, address string, c syscall.RawConn) error {
		return c.Control(func(fd uintptr) {
			var err error
			// do not support SO_MARK
			if dscp != 0 {
				err = syscall.SetsockoptInt(syscall.Handle(fd), windows.IPPROTO_IP, windows.IP_TOS, dscp << 2)
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
