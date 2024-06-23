package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewSore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: DefaultPathTransformFunc,
	}
	store := NewStore(opts)
	key := "key"
	tr := store.PathTransformFunc(key)
	require.Equal(t, tr, key)
}

func Test_writeStream(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: DefaultPathTransformFunc,
	}
	store := NewStore(opts)
	key := "specialKey"
	payLoad := bytes.NewReader([]byte("some jpg bytes"))
	require.Nil(t, store.writeStream(key, payLoad))

}
