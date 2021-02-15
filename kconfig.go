package main
//
// KCP-related settings
//

import (
	"encoding/json"
	"fmt"
	"os"

	kcp "github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
	"crypto/sha1"
)

type KCPConfig struct {
	Key          string `json:"key"`
	Crypt        string `json:"crypt"`
	Mode         string `json:"mode"`
	MTU          int    `json:"mtu"`
	SndWnd       int    `json:"sndwnd"`
	RcvWnd       int    `json:"rcvwnd"`
	DataShard    int    `json:"datashard"`
	ParityShard  int    `json:"parityshard"`
	DSCP         int    `json:"dscp"`
	AckNodelay   bool   `json:"acknodelay"`
	NoDelay      int    `json:"nodelay"`
	Interval     int    `json:"interval"`
	Resend       int    `json:"resend"`
	NoCongestion int    `json:"nc"`
	SockBuf      int    `json:"sockbuf"`
}

func getKCPConfig(path string) *KCPConfig {
	kconf := KCPConfig{
		"somekey",	// key
		"xor",		// crypt
		"fast2",	// mode
		1350,		// mtu
		1024,		// sndwnd
		1024,		// rcvwnd
		10,			// datashard
		3,			// parityshard
		0,			// dscp
		false,		// acknodelay
		0,			// nodelay
		50,			// interval
		0,			// resend
		0,			// nc
		4194304,	// sockbuf
		// 1,		// smuxver
		// 4194304	// smuxbuf
		// 2097152,	// streambuf
		// 2,		// keepalive
		// 10,		// timeout
	}

	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		fmt.Println("could not load kcp config file, use default values instead")
	} else {
		fmt.Printf("load %s\n", path)
		json.NewDecoder(file).Decode(&kconf)
	}

	switch kconf.Mode {
	case "normal":
		kconf.NoDelay, kconf.Interval, kconf.Resend, kconf.NoCongestion = 0, 40, 2, 1
	case "fast":
		kconf.NoDelay, kconf.Interval, kconf.Resend, kconf.NoCongestion = 0, 30, 2, 1
	case "fast2":
		kconf.NoDelay, kconf.Interval, kconf.Resend, kconf.NoCongestion = 1, 20, 2, 1
	case "fast3":
		kconf.NoDelay, kconf.Interval, kconf.Resend, kconf.NoCongestion = 1, 10, 2, 1
	}

	return &kconf
}

func getKCPBlockCipher(kconf *KCPConfig) kcp.BlockCrypt {
	pass := pbkdf2.Key([]byte(kconf.Key), []byte("some salt"), 4096, 32, sha1.New)
	var block kcp.BlockCrypt
	switch kconf.Crypt {
	case "sm4":
		block, _ = kcp.NewSM4BlockCrypt(pass[:16])
	case "tea":
		block, _ = kcp.NewTEABlockCrypt(pass[:16])
	case "xor":
		block, _ = kcp.NewSimpleXORBlockCrypt(pass)
	case "none":
		block, _ = kcp.NewNoneBlockCrypt(pass)
	case "aes-128":
		block, _ = kcp.NewAESBlockCrypt(pass[:16])
	case "aes-192":
		block, _ = kcp.NewAESBlockCrypt(pass[:24])
	case "blowfish":
		block, _ = kcp.NewBlowfishBlockCrypt(pass)
	case "twofish":
		block, _ = kcp.NewTwofishBlockCrypt(pass)
	case "cast5":
		block, _ = kcp.NewCast5BlockCrypt(pass[:16])
	case "3des":
		block, _ = kcp.NewTripleDESBlockCrypt(pass[:24])
	case "xtea":
		block, _ = kcp.NewXTEABlockCrypt(pass[:16])
	case "salsa20":
		block, _ = kcp.NewSalsa20BlockCrypt(pass)
	default:
		kconf.Crypt = "aes"
		block, _ = kcp.NewAESBlockCrypt(pass)
	}
	return block
}

// func main() {
// 	conf := getKCPConfig("kcp.conf")

// 	fmt.Printf("%T: %v\n", conf, conf)
// }
