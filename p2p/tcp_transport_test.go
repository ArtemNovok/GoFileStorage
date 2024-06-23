package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTCPTransport(t *testing.T) {
	listenerAddr := ":4646"
	tr := NewTCPTransport(listenerAddr)
	require.Equal(t, tr.listenerAddress, listenerAddr)

	// Server
	assert.Nil(t, tr.ListenAndAccept())
}
