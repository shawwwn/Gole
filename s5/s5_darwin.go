package s5

import "net"
import "time"

func CreateDialer(bind net.Addr, fwmark int, dscp int) *net.Dialer {

	dialer := &net.Dialer{
		LocalAddr: bind,
		Timeout: time.Duration(time.Second*30),
	}

	// TODO: Set fwmark,dscp on OSX

	return dialer
}
