package main

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"sync"
)

type DataPacket struct {
	Buffer     []byte
	Sender     netip.AddrPort
	FromServer bool
}

func bail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var wg sync.WaitGroup
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	packets := make(chan DataPacket, 2)

	_target_host, err := net.LookupHost("play.hyperlandsmc.net")
	bail(err)

	target_host := _target_host[0]
	target_port := uint16(19132)

	target_addr := netip.AddrPortFrom(netip.AddrFrom4([4]byte(net.ParseIP(target_host).To4())), target_port)

	laddr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:19132")
	bail(err)

	conn, err := net.ListenUDP("udp4", laddr)
	bail(err)

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	wg.Add(1)

	go func() {
		var client *net.UDPAddr

		for {
			select {
			case packet := <-packets:
				target_addr, err := net.ResolveUDPAddr("udp4", target_addr.String())
				bail(err)

				fmt.Printf("Received a packet from %s\n", packet.Sender.String())

				if packet.FromServer {
					fmt.Printf("Sending to %s a buffer with %d size\n", client.String(), len(packet.Buffer))
					conn.WriteToUDP(packet.Buffer, client)
				} else {
					if client == nil {
						client = net.UDPAddrFromAddrPort(packet.Sender)
					}

					conn.WriteTo(packet.Buffer, target_addr)
					fmt.Printf("Sending to %s a buffer with %d size\n", target_addr.String(), len(packet.Buffer))
				}
			case <-ctx.Done():
				wg.Done()
				return
			}
		}
	}()

	for ctx.Err() == nil {
		buffer := make([]byte, 1024)

		count, addr, err := conn.ReadFromUDPAddrPort(buffer)
		if err != nil && ctx.Err() == nil {
			bail(err)
		}

		if count < 0 {
			continue
		}

		packets <- DataPacket{
			Buffer:     buffer[:count],
			Sender:     addr,
			FromServer: target_addr.String() == addr.String(),
		}
	}

	<-ctx.Done()
	wg.Wait()
}
