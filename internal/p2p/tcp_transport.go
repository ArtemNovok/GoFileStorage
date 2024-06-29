package p2p

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
)

// TCPPeer represents tcp remote node over established connection
type TCPPeer struct {
	// connection to a peer
	net.Conn

	// if we dial to a connection - outbound : true
	// if we we accept connection - outbound : false

	outbound bool

	wg *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}
func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

// Send write payload to the peer connection and implements Peer interface
func (p *TCPPeer) Send(payload []byte) error {
	_, err := p.Conn.Write(payload)
	return err
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
		rpcch:            make(chan RPC, 1024),
	}
}
func (tc *TCPTransport) NetAddr() net.Addr {
	return tc.listener.Addr()
}

// Addr returns transport's address and implements Transport interface
func (tc *TCPTransport) Addr() string {
	return tc.listener.Addr().String()
}

// Dial handles outbound connections and implements Transport interface
func (tc *TCPTransport) Dial(addr string) error {
	const op = "p2p.Dial"
	log := tc.Log.With(slog.String("op", op))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return err
	}
	go tc.handleConn(conn, true)
	return nil
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
		log.Error("got error", slog.String("error", err.Error()))
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
		log.Info("got peer connection from", slog.String("address", conn.RemoteAddr().String()))
		go tc.handleConn(conn, false)
	}
}

// Close implements Transport interface
func (tc *TCPTransport) Close() error {
	return tc.listener.Close()
}

func (tc *TCPTransport) handleConn(con net.Conn, isOutBound bool) {
	var err error
	const op = "p2p.tcp_transport.handleConn"
	log := tc.Log.With(slog.String("op", op))
	defer func() {
		log.Info("dropping peer connections", slog.Any("error", err))
		con.Close()
	}()

	peer := NewTCPPeer(con, isOutBound)
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
	for {
		rpc := RPC{}
		if err = tc.Decoder.Decode(con, &rpc); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			log.Error("got error", slog.String("error", err.Error()))

			continue
		}
		rpc.From = peer.Conn.RemoteAddr().String()
		if rpc.Stream {
			log.Info("waiting till stream is done", slog.String("from", rpc.From))
			peer.wg.Add(1)
			peer.wg.Wait()
			log.Info("stream is done", slog.String("from", rpc.From))
			continue
		}
		tc.rpcch <- rpc
	}
}
