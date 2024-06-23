package p2p

import (
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents tcp remote node over established connection
type TCPPeer struct {
	// connection to a peer
	conn net.Conn

	// if we dial to a connection - outbound : true
	// if we we accept connection - outbound : false

	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransport struct {
	// listener address on which we accept connections
	listenerAddress string

	// listener for our listener address
	listener net.Listener

	// handshakeFunc is a func to make "handshake" before proceed the peer
	shakeHandsFunc HandshakeFunc
	decoder        Decoder
	mu             sync.RWMutex
	peers          map[net.Addr]Peer
}

func NewTCPTransport(listenerAddr string) *TCPTransport {
	return &TCPTransport{
		listenerAddress: listenerAddr,
		shakeHandsFunc:  NoHandshakeFunc,
	}
}

func (tc *TCPTransport) ListenAndAccept() error {
	var err error
	tc.listener, err = net.Listen("tcp", tc.listenerAddress)
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

type Temp struct {
}

func (tc *TCPTransport) handleConn(con net.Conn) {
	const op = "p2p.tcp_transport.handleConn"
	peer := NewTCPPeer(con, true)
	if err := tc.shakeHandsFunc(con); err != nil {
		fmt.Printf("%s:%s", op, err.Error())
	}
	var temp Temp
	for {
		tc.decoder.Decode(peer.conn, temp)
	}
}
