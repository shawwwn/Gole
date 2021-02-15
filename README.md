<img src="goler.png" alt="goler" height="100px" />

# Gole
A p2p hole punching/NAT traversal/tunneling tool written in Go.

## Features
* TCP/UDP hole punching even when both sides are behind symmetric NATs (but not guaranteed).
* TCP/UDP Tunneling.
* KCP Tunneling for tcp-over-udp support.
* STUN-less, command line driven.

## To Get Started
Suppose:
* Remote Peer has a web server listening at `127.0.0.1:8888`
* Peer is behind NAT and has a public ip of `4.4.4.4`
* I'm also behind NAT and have a public ip of `3.3.3.3`
* I want to access Peer's web server from my local machine at `127.0.0.1:1111`
* We agreed on a pair of tcp ports `:4444`(peer) and `:3333`(me)

Peer run: 
```sh
gole tcp 0.0.0.0:4444 3.3.3.3:3333 -op server -fwd=127.0.0.1:8888
```

I run:
```sh
gole tcp 0.0.0.0:3333 4.4.4.4:4444 -op client -fwd=127.0.0.1:1111
```

After successfully punching through both NATs, a TCP tunnel will be created on punched holes.

I can access Peer's web server from my `127.0.0.1:1111`:
```
127.0.0.1:1111 --> (1.2.3.4:4444 <--> 1.2.3.4:3333) --> 127.0.0.1:8888
```

## Usage
```
gtun MODE local_addr remote_addr MODE_OPTIONS...

        MODE 'tcp' OPTIONS:
          -fwd=IP:PORT
                Forward to address in server mode
                Forward from address in client mode
          -op=holepunch|server|client
                Operation to perform (default "holepunch")
                NOTE: "server" means first holepunch and start tunnel server

        MODE 'udp' OPTIONS:
          -fwd=IP:PORT
                Forward to/from address in server/client mode
          -op=holepunch|server|client
                Operation to perform (default "holepunch")
          -proto=udp|kcp[,conf=path-to-kcp-config-file]
                Tunnel's transport layer protocol (default "udp")
                NOTE: When using KCP protocol, forward address must be TCP address.
          -ttl=0
                TTL value used in holepunching (0 to disable setting ttl)
                Should only be used when both sides are under symmetric NATs. For full rationale, please refer to Wiki.
                NOTE: Only one side needs to set it!
          }
```

## Building
```sh
make
./gole -h
```

## Documentation
TODO: wiki

## Credits
* xtaci, for kcp-go(https://github.com/xtaci/kcp-go)
* xtaci, for smux(https://github.com/xtaci/smux)
