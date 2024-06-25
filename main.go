package main

import (
	"fmt"
	"gofilesystem/internal/p2p"
	"gofilesystem/internal/server"
	"gofilesystem/internal/store"
	"log"
	"time"
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
		// OnPeer:          OnPeer,
	}
	tcpTransport := p2p.NewTCPTransport(tcpTrOpts)
	serverOpts := server.FileServerOpts{
		StorageRoot:       "3000_files",
		PathTransformFunc: store.CASPathTransformFunc,
		Transport:         tcpTransport,
	}
	s := server.NewFileServer(serverOpts)
	go func() {
		time.Sleep(10 * time.Second)
		fmt.Println("quitting server")
		s.Stop()
	}()
	if err := s.Start(); err != nil {
		log.Fatalf("got fatal error: %s", err)
	}
}
