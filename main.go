package main

import (
	"fmt"
	"gofilesystem/internal/p2p"
	"log"
)

func OnPeer(p p2p.Peer) error {
	fmt.Println("execute OnPeer function")
	p.Close()
	return nil
}
func main() {
	tcpTrOpts := p2p.TCPTransportOpts{
		ListenerAddress: ":3000",
		ShakeHandsFunc:  p2p.NoHandshakeFunc,
		Decoder:         &p2p.DefaultDecoder{},
		OnPeer:          OnPeer,
	}
	tc := p2p.NewTCPTransport(tcpTrOpts)

	go func() {
		for {
			msg := <-tc.Consume()
			fmt.Printf("%+v\n", msg)
		}
	}()

	if err := tc.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}
	select {}
}
