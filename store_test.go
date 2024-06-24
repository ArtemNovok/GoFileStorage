package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewSore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	store := NewStore(opts)
	key := "key"
	wantPath := "a62f2/225bf/70bfa/ccbc7/f1ef2/a3978/36717/377de"
	wantOrigin := "a62f2225bf70bfaccbc7f1ef2a397836717377de"
	tr := store.PathTransformFunc(key)
	require.Equal(t, tr.PathName, wantPath)
	require.Equal(t, tr.FileName, wantOrigin)
}

func Test_writeStream_Has_Delete(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	store := NewStore(opts)
	key := "fatcatpicture"
	data := "some jpg bytes"
	payLoad := bytes.NewReader([]byte(data))
	require.Nil(t, store.writeStream(key, payLoad))
	bytes, err := store.Read(key)
	require.Nil(t, err)
	src, err := io.ReadAll(bytes)
	require.Nil(t, err)
	require.Equal(t, string(src), data)
	exists := store.Has(key)
	require.Equal(t, exists, true)
	err = store.Delete(key)
	require.Nil(t, err)
}
