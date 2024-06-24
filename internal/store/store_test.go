package store

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewSore(t *testing.T) {
	store := newStore()
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

func Test_MuiliReadWrites(t *testing.T) {
	tests := []struct {
		name  string
		store Store
		key1  string
		key2  string
		data1 string
		data2 string
	}{
		{
			name:  "test1",
			store: *NewStore(StoreOpts{PathTransformFunc: CASPathTransformFunc}),
			key1:  "key1",
			key2:  "key2",
			data1: "data for key1",
			data2: "data for key2",
		},
	}
	for _, tt := range tests {
		// writing, checking and reading
		payLoad1 := bytes.NewReader([]byte(tt.data1))
		err := tt.store.writeStream(tt.key1, payLoad1)
		require.Nil(t, err)
		exist := tt.store.Has(tt.key1)
		require.Equal(t, exist, true)
		bytes1, err := tt.store.Read(tt.key1)
		require.Nil(t, err)
		src, err := io.ReadAll(bytes1)
		require.Nil(t, err)
		require.Equal(t, string(src), tt.data1)
		payLoad2 := bytes.NewReader([]byte(tt.data2))
		err = tt.store.writeStream(tt.key2, payLoad2)
		require.Nil(t, err)
		exist2 := tt.store.Has(tt.key2)
		require.Equal(t, exist2, true)
		bytes2, err := tt.store.Read(tt.key2)
		require.Nil(t, err)
		src2, err := io.ReadAll(bytes2)
		require.Nil(t, err)
		require.Equal(t, string(src2), tt.data2)
		// deleting
		err = tt.store.Delete(tt.key1)
		require.Nil(t, err)
		notExist1 := tt.store.Has(tt.key1)
		require.Equal(t, notExist1, false)
		exist2 = tt.store.Has(tt.key2)
		require.Equal(t, exist2, true)
		err = tt.store.Delete(tt.key2)
		require.Nil(t, err)
		notExist2 := tt.store.Has(tt.key2)
		require.Equal(t, notExist2, false)
		tt.store.Clear()
	}
}
func Test_Write_MultipleTimes(t *testing.T) {
	store := newStore()
	count := 100
	for i := 0; i < count; i++ {
		payLoad := bytes.NewReader([]byte("test bytes"))
		key := fmt.Sprintf("key:%v", i)
		err := store.writeStream(key, payLoad)
		require.Nil(t, err)
		isExist := store.Has(key)
		require.Equal(t, isExist, true)
		reder, err := store.Read(key)
		require.Nil(t, err)
		bytes, err := io.ReadAll(reder)
		require.Nil(t, err)
		require.Equal(t, bytes, []byte("test bytes"))
		err = store.Delete(key)
		require.Nil(t, err)
		isExist = store.Has(key)
		require.Equal(t, isExist, false)
	}
	err := store.Clear()
	require.Nil(t, err)
}
func Test_Clear(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	store := NewStore(opts)
	byte := bytes.NewReader([]byte("test"))
	err := store.writeStream("test", byte)
	require.Nil(t, err)
	err = store.Clear()
	require.Nil(t, err)
}

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	return NewStore(opts)

}
