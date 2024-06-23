package main

import (
	"gofilesystem/p2p"
	"log"
)

func main() {
	tc := p2p.NewTCPTransport(":3000")
	if err := tc.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}
	select {}
}
