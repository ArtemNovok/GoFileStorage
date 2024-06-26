package p2p

import "net"

// Peer is a representation of remote node
type Peer interface {
	net.Conn
	Send([]byte) error
}

// Transport is a handler for communication
// between the nodes in the network
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
	Dial(string) error
}
