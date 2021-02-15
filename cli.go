package main

import (
	"flag"
	"fmt"
	"os"
	"net"
	"strings"
)

type Config interface {
	getMode() string
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	getOp() string
}

type TCPConfig struct {
	Op string
	LAddr *net.TCPAddr
	RAddr *net.TCPAddr
	FwdAddr net.Addr
}
func (c TCPConfig) getMode() string {
	return "tcp"
}
func (c TCPConfig) LocalAddr() net.Addr {
	return c.LAddr
}
func (c TCPConfig) RemoteAddr() net.Addr {
	return c.RAddr
}
func (c TCPConfig) getOp() string {
	return c.Op
}

type UDPConfig struct {
	Op string
	LAddr *net.UDPAddr
	RAddr *net.UDPAddr
	FwdAddr net.Addr
	Proto string
	KConf string
	TTL int
}
func (c UDPConfig) getMode() string {
	return "udp"
}
func (c UDPConfig) LocalAddr() net.Addr {
	return c.LAddr
}
func (c UDPConfig) RemoteAddr() net.Addr {
	return c.RAddr
}
func (c UDPConfig) getOp() string {
	return c.Op
}

func ParseConfig(args []string) Config {
	tcp_cmd := flag.NewFlagSet("tcp", flag.ExitOnError)
	tcp_op := tcp_cmd.String("op", "holepunch", "operation to perform")
	tcp_fwd := tcp_cmd.String("fwd", "", "forward to/from address in server/client mode")

	udp_cmd := flag.NewFlagSet("udp", flag.ExitOnError)
	udp_ttl := udp_cmd.Int("ttl", 0, "ttl value used in holepunching")
	udp_op := udp_cmd.String("op", "holepunch", "operation to perform")
	udp_proto := udp_cmd.String("proto", "udp", "tunnel's transport layer protocol")
	udp_fwd := udp_cmd.String("fwd", "", "forward to/from address in server/client mode")

	print_usage := func() {
		fmt.Println("usage:")
		fmt.Println("gole MODE local_addr remote_addr MODE_OPTIONS...")
		fmt.Println("\nMODE 'tcp' OPTIONS:")
		tcp_cmd.PrintDefaults()
		fmt.Println("\nMODE 'udp' OPTIONS:")
		udp_cmd.PrintDefaults()
	}

	if len(args) <= 1 {
		print_usage()
		os.Exit(1)
	}

	arg1 := strings.ReplaceAll(args[1], "-", "")
	if len(args)==1 || arg1=="h" || arg1=="help" {
		print_usage()
		os.Exit(1)
	} else if len(args) < 2 {
		fmt.Println("must select a mode (tcp|udp)\n")
		os.Exit(1)
	} else if len(args) < 4 {
		fmt.Println("must specify local/remote endpoints for tunnel\n")
		print_usage()
		os.Exit(1)
	}
	l_endpt := args[2]
	r_endpt := args[3]

	switch args[1] {
	case "tcp":
		conf := new(TCPConfig)
		conf.LAddr, _ = net.ResolveTCPAddr("tcp", l_endpt)
		conf.RAddr, _ = net.ResolveTCPAddr("tcp", r_endpt)
		tcp_cmd.Parse(args[4:])
		conf.FwdAddr, _ = net.ResolveTCPAddr("tcp", *tcp_fwd)
		conf.Op = *tcp_op
		if ! contains(conf.Op, []string{"holepunch", "server", "client"}) {
			perror("Unknown operation:", conf.Op)
			os.Exit(1)
		}
		return conf

	case "udp":
		conf := new(UDPConfig)
		conf.LAddr, _ = net.ResolveUDPAddr("udp", l_endpt)
		conf.RAddr, _ = net.ResolveUDPAddr("udp", r_endpt)
		udp_cmd.Parse(args[4:])
		conf.TTL = *udp_ttl
		conf.Op = *udp_op
		if ! contains(conf.Op, []string{"holepunch", "server", "client"}) {
			perror("Unknown operation:", conf.Op)
			os.Exit(1)
		}

		// parse "-proto"
		ps := strings.Split(*udp_proto, ",") // parms: kcp,conf=<path>
		conf.Proto = ps[0]
		if conf.Proto == "udp" {
			conf.FwdAddr, _ = net.ResolveUDPAddr("udp", *udp_fwd)
		} else if conf.Proto == "kcp" {
			conf.FwdAddr, _ = net.ResolveTCPAddr("tcp", *udp_fwd)
			conf.KConf = "kcp.conf" // default path
			if len(ps) > 1 {
				for _, v := range ps[1:] {
					ks := strings.SplitN(v, "=", 2)
					key := ks[0]
					val := ""
					if len(ks)>1 {
						val = ks[1]
					}
					if key == "conf" {
						conf.KConf = val
					} else {
						perror("unknown proto parameters")
						os.Exit(1)
					}
				}
			}
		} else {
			perror("unknown protocol")
			os.Exit(1)
		}

		if len(udp_cmd.Args()) != 0 {
			perror("Unknown option:", udp_cmd.Args()[0])
			os.Exit(1)
		}
		return conf

	default:
		fmt.Println("must select a mode (tcp|udp)\n")
		os.Exit(1)
	} // switch MODE

	return nil
}
