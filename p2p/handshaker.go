package p2p

// HandshakeFunc is a func :)
type HandshakeFunc func(any) error

func NoHandshakeFunc(any) error {
	return nil
}
