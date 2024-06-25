package p2p

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTCPTransport(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	listenerAddr := ":4646"
	opts := TCPTransportOpts{
		ListenerAddress: listenerAddr,
		ShakeHandsFunc:  NoHandshakeFunc,
		Decoder:         DefaultDecoder{},
		Log:             logger,
	}
	tr := NewTCPTransport(opts)
	require.Equal(t, tr.TCPTransportOpts.ListenerAddress, listenerAddr)

	// Server
	assert.Nil(t, tr.ListenAndAccept())
}
