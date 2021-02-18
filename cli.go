package main

import (
	"flag"
	"fmt"
	"os"
	"net"
	"strings"
	"strconv"

	"github.com/shawwwn/gole/s5"
)

type Config interface {
	getMode() string
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	getOp() string
}

type S5Config struct {
	bind *net.TCPAddr
	fwmark int
	dscp int
	dialer *net.Dialer
}

type TCPConfig struct {
	Op string
	LAddr *net.TCPAddr
	RAddr *net.TCPAddr
	FwdAddr net.Addr
	Enc string
	Key string
	S5Conf *S5Config
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
	Enc string
	Key string
	S5Conf *S5Config
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

var g_timeout int
var g_verbose bool
func ParseConfig(args []string) Config {
	g_cmd := flag.NewFlagSet("tcp", flag.ExitOnError)
	g_cmd.BoolVar(&g_verbose, "verbose", false, "turn on debug output")
	g_cmd.BoolVar(&g_verbose, "v", false, "")
	g_help := g_cmd.Bool("help", false, "usage information")
	g_cmd.BoolVar(g_help, "h", false, "")
	g_cmd.IntVar(&g_timeout, "timeout", 30, "how long in seconds an idle connection timeout and exit")
	g_enc := g_cmd.String("enc", "xor", "encryption method")
	g_key := g_cmd.String("key", "", "encryption key (leave empty to disable encryption)")

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
		fmt.Println("gole [GLOBAL_OPTIONS] MODE local_addr remote_addr MODE_OPTIONS")
		fmt.Println("\nGLOBAL OPTIONS:")
		g_cmd.PrintDefaults()
		fmt.Println("\nMODE 'tcp' OPTIONS:")
		tcp_cmd.PrintDefaults()
		fmt.Println("\nMODE 'udp' OPTIONS:")
		udp_cmd.PrintDefaults()
	}

	g_cmd.Parse(args[1:])
	if *g_help {
		print_usage()
		os.Exit(0)
	}
	args = g_cmd.Args()

	if len(args) <= 0 {
		print_usage()
		os.Exit(1)
	}

	if len(args) < 1 {
		fmt.Printf("must select a mode (tcp|udp)\n")
		os.Exit(1)
	} else if len(args) < 3 {
		fmt.Printf("must specify both local & remote endpoints for tunnel\n")
		print_usage()
		os.Exit(1)
	}
	l_endpt := args[1]
	r_endpt := args[2]

	mode := strings.ToLower(args[0]) 
	switch mode {
	case "tcp":
		conf := new(TCPConfig)
		conf.LAddr, _ = net.ResolveTCPAddr("tcp", l_endpt)
		conf.RAddr, _ = net.ResolveTCPAddr("tcp", r_endpt)
		tcp_cmd.Parse(args[3:])
		conf.Op = *tcp_op
		if ! contains(conf.Op, []string{"holepunch", "server", "client"}) {
			perror("Unknown operation:", conf.Op)
			os.Exit(1)
		}
		if strings.HasPrefix(*tcp_fwd, "socks5") {
			if conf.Op != "server" {
				perror("SOCKS5 proxy only works in server mode")
				os.Exit(1)
			}
			conf.FwdAddr = nil
			conf.S5Conf = parseSocks5(*tcp_fwd)
			s5.Dialer = conf.S5Conf.dialer
			s5.Verbose = g_verbose
		} else {
			conf.FwdAddr, _ = net.ResolveTCPAddr("tcp", *tcp_fwd)
		}
		conf.Enc = *g_enc
		conf.Key = *g_key
		return conf

	case "udp":
		conf := new(UDPConfig)
		conf.LAddr, _ = net.ResolveUDPAddr("udp", l_endpt)
		conf.RAddr, _ = net.ResolveUDPAddr("udp", r_endpt)
		udp_cmd.Parse(args[3:])
		conf.TTL = *udp_ttl
		conf.Op = *udp_op
		if ! contains(conf.Op, []string{"holepunch", "server", "client"}) {
			perror("Unknown operation:", conf.Op)
			os.Exit(1)
		}
		conf.Enc = *g_enc
		conf.Key = *g_key

		parseProto(*udp_proto, conf)
		if conf.Proto == "udp" {
			conf.FwdAddr, _ = net.ResolveUDPAddr("udp", *udp_fwd)
		} else if conf.Proto == "kcp" {
			if strings.HasPrefix(*udp_fwd, "socks5") {
				if conf.Op != "server" {
					perror("SOCKS5 proxy only works in server mode")
					os.Exit(1)
				}
				conf.FwdAddr = nil
				conf.S5Conf = parseSocks5(*udp_fwd)
				s5.Dialer = conf.S5Conf.dialer
				s5.Verbose = g_verbose
			} else {
				conf.FwdAddr, _ = net.ResolveTCPAddr("tcp", *udp_fwd)
			}
		}

		if len(udp_cmd.Args()) != 0 {
			perror("Unknown option:", udp_cmd.Args()[0])
			os.Exit(1)
		}
		return conf

	default:
		fmt.Printf("must select a mode (tcp|udp)\n")
		os.Exit(1)
	} // switch MODE

	return nil
}

// Params: -fwd="socks5"
//         -fwd="socks5,bind=192.168.1.64,fwmark=10,dscp=46"
func parseSocks5(ss string) *S5Config {
	ps := strings.Split(ss, ",")
	// ps[0] == "socks5"
	s5conf := &S5Config{nil, 0, 0, nil}
	if len(ps) > 1 {
		for _, v := range ps[1:] {
			ks := strings.SplitN(v, "=", 2)
			key := ks[0]
			val := ""
			if len(ks)>1 {
				val = ks[1]
			}
			switch key {
			case "bind":
				s5conf.bind = &net.TCPAddr{ IP: parseIP(val) }
			case "fwmark":
				s5conf.fwmark, _ = strconv.Atoi(val)
			case "dscp":
				s5conf.dscp, _ = strconv.Atoi(val)
			default:
				perror("Unknown SOCKS5 parameters: %s", v)
				os.Exit(1)
			}
		}
	}

	s5conf.dialer = s5.CreateDialer(s5conf.bind, s5conf.fwmark, s5conf.dscp)
	// fmt.Printf("s5: %v\n", s5conf)
	return s5conf
}

// Params: -proto="kcp,conf=<path>"
//         -proto="udp"
func parseProto(ss string, conf *UDPConfig) {
	ps := strings.Split(ss, ",")
	conf.Proto = ps[0]
	if conf.Proto == "kcp" {
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
	} else if conf.Proto == "udp" {
		// ...
	} else {
		perror("unknown protocol")
		os.Exit(1)
	}
}
