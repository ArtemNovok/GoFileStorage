package p2p

import (
	"errors"
	"fmt"
	"io"
	"net"
)

// TCPPeer represents tcp remote node over established connection
type TCPPeer struct {
	// connection to a peer
	conn net.Conn

	// if we dial to a connection - outbound : true
	// if we we accept connection - outbound : false

	outbound bool
}

// Close implement peer interface, close closes peer connection
func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransportOpts struct {
	// listener address on which we accept connections
	ListenerAddress string
	// handshakeFunc is a func to make "handshake" before proceed the peer
	ShakeHandsFunc HandshakeFunc
	Decoder        Decoder
	OnPeer         func(Peer) error
}

type TCPTransport struct {
	// listener for our listener address
	listener net.Listener

	TCPTransportOpts

	rpcch chan RPC
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcch:            make(chan RPC),
	}
}

// Consume implements transport  interface which returns read-only chanel
// that contains messages received from another peer
func (tc *TCPTransport) Consume() <-chan RPC {
	return tc.rpcch
}

func (tc *TCPTransport) ListenAndAccept() error {
	var err error
	tc.listener, err = net.Listen("tcp", tc.ListenerAddress)
	if err != nil {
		return err
	}
	go tc.startAcceptLoop()
	return nil
}

func (tc *TCPTransport) startAcceptLoop() {
	const op = "p2p.tcp_transport.acceptLoop"

	for {
		conn, err := tc.listener.Accept()
		if err != nil {
			//TODO Don't know what to do here for now
			fmt.Printf("%s: %s\n", op, err.Error())
		}
		go tc.handleConn(conn)
	}
}

func (tc *TCPTransport) handleConn(con net.Conn) {
	var err error
	const op = "p2p.tcp_transport.handleConn"
	defer func() {
		fmt.Printf("dropping peer connection %s: %s\n", op, err)
		con.Close()
	}()

	peer := NewTCPPeer(con, true)
	if err := tc.ShakeHandsFunc(peer); err != nil {
		con.Close()
		fmt.Printf("%s: error: %s\n", op, err)
		return
	}
	if tc.OnPeer != nil {
		if err = tc.OnPeer(peer); err != nil {
			return
		}
	}
	rpc := RPC{}
	for {
		if err := tc.Decoder.Decode(con, &rpc); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Printf("%s: error: %s\n", op, err)
			continue
		}
		rpc.From = peer.conn.RemoteAddr()
		tc.rpcch <- rpc
	}
}
