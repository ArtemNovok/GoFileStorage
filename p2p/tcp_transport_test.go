package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTCPTransport(t *testing.T) {
	listenerAddr := ":4646"
	opts := TCPTransportOpts{
		ListenerAddress: listenerAddr,
		ShakeHandsFunc:  NoHandshakeFunc,
		Decoder:         DefaultDecoder{},
	}
	tr := NewTCPTransport(opts)
	require.Equal(t, tr.TCPTransportOpts.ListenerAddress, listenerAddr)

	// Server
	assert.Nil(t, tr.ListenAndAccept())
}
