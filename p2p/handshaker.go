package p2p

// HandshakeFunc is a func :)
type HandshakeFunc func(Peer) error

func NoHandshakeFunc(Peer) error {
	return nil
}
