package p2p

// Peer is a representation of remote node
type Peer interface {
	Close() error
}

// Transport is a handler for communication
// between the nodes in the network
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
