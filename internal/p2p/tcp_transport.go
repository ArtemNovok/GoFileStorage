package p2p

import (
	"errors"
	"io"
	"log/slog"
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
	Log            *slog.Logger
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
	const op = "p2p.ListenAndAccept"
	log := tc.Log.With(slog.String("op", op))
	var err error
	tc.listener, err = net.Listen("tcp", tc.ListenerAddress)
	if err != nil {
		return err
	}
	go tc.startAcceptLoop()
	log.Info("TCP transport listening on", slog.String("port", tc.ListenerAddress))
	return nil
}

func (tc *TCPTransport) startAcceptLoop() {
	const op = "p2p.tcp_transport.acceptLoop"
	log := tc.Log.With(slog.String("op", op))
	for {
		conn, err := tc.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			//TODO Don't know what to do here for now
			log.Error("got error", slog.String("error", err.Error()))
		}
		go tc.handleConn(conn)
	}
}

// Close implements Transport interface
func (tc *TCPTransport) Close() error {
	return tc.listener.Close()
}

func (tc *TCPTransport) handleConn(con net.Conn) {
	var err error
	const op = "p2p.tcp_transport.handleConn"
	log := tc.Log.With(slog.String("op", op))
	defer func() {
		log.Info("dropping peer connections", slog.String("error", err.Error()))
		con.Close()
	}()

	peer := NewTCPPeer(con, true)
	if err := tc.ShakeHandsFunc(peer); err != nil {
		con.Close()
		log.Error("got error", slog.String("error", err.Error()))
		return
	}
	if tc.OnPeer != nil {
		if err = tc.OnPeer(peer); err != nil {
			return
		}
	}
	rpc := RPC{}
	for {
		if err = tc.Decoder.Decode(con, &rpc); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			log.Error("got error", slog.String("error", err.Error()))

			continue
		}
		rpc.From = peer.conn.RemoteAddr()
		tc.rpcch <- rpc
	}
}
