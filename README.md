<img src="goler.png" alt="goler" height="150px" />

# Gole
A p2p hole-punching tool wrriten in Go, allowing two computers behind NAT to communicate with each other.

## Features
* TCP/UDP hole punching even when both sides are behind symmetric NATs (no guarantee :wink:)
* TCP/UDP tunneling
* KCP[*](#References) tunneling for tcp-over-udp support
* STUN-less, command line driven
* Traffic encryption, anti-gfw

## Quickstart
Suppose:
* A has a web server listening at `127.0.0.1:8888`
* A is behind NAT and has a public ip of `4.4.4.4`
* B also behind NAT and have a public ip of `3.3.3.3`
* B want to access Peer's web server from my local machine at `127.0.0.1:1111`
* They agreed on a pair of tcp ports `:4444`(A) and `:3333`(B)

B run: 
```sh
gole -v tcp 0.0.0.0:4444 3.3.3.3:3333 -op server -fwd=127.0.0.1:8888
```

A run:
```sh
gole -v tcp 0.0.0.0:3333 4.4.4.4:4444 -op client -fwd=127.0.0.1:1111
```

After successfully punching through both NATs, a TCP tunnel between the two ports will be created.

A can then access B's web server from his localhost at `127.0.0.1:1111`:
```
127.0.0.1:1111 --> (1.2.3.4:4444 <--> 1.2.3.4:3333) --> 127.0.0.1:8888
```

## Usage
```
gole [GLOBAL_OPTIONS] MODE local_addr remote_addr MODE_OPTIONS...

    GLOBAL OPTIONS:
      -h
      -help
            Usage information
      -timeout=30
            How long in seconds an idle connection timeout and exit
            Please refer to wiki for more info
      -v
      -verbose
            Turn on debug output
      -enc=xor
            Encryption method
      -key=
            Encryption key (leave empty to disable encryption)

    MODE 'tcp' OPTIONS:
      -fwd=IP:PORT
            Forward to address in server mode
            Forward from address in client mode
      -op=holepunch|server|client
            Operation to perform (default "holepunch")
            NOTE: "server" means first holepunch and start tunnel server

    MODE 'udp' OPTIONS:
      -fwd=IP:PORT
            <same as in 'tcp'>
      -op=holepunch|server|client
            <same as in 'tcp'>
      -proto=udp|kcp[,conf=path-to-kcp-config-file]
            Tunnel's transport layer protocol (default "udp")
            NOTE: When using the KCP protocol, forward address on both sides must be TCP address.
      -ttl=0
            TTL value used in holepunching (0 to disable setting ttl)
            Should only be used when both sides are under symmetric NATs. For the full rationale, please refer to wiki.
            NOTE: Only one side needs to set it!
```

## Building
```sh
make
./gole -h
```

## Documentation
TODO: wiki

## References
* Bryan Ford, the UDP hole punching part of Gole is loosely based on [his paper](https://bford.info/pub/net/p2pnat/)
* xtaci, for [kcp-go](https://github.com/xtaci/kcp-go)
* xtaci, for [smux](https://github.com/xtaci/smux)
* templexxx, for [xorsimd](https://github.com/templexxx/xorsimd)
