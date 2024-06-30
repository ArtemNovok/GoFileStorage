package store

import (
	"bytes"
	"fmt"
	"gofilesystem/internal/encrypt"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_DecryptAndEncrypt(t *testing.T) {
	enKey := encrypt.NewEcryptionKey()
	store := newStore()
	db := "my_db"
	key := "my data"
	data := "my custom data"
	payLoad := bytes.NewBuffer([]byte(data))
	_, err := store.writeEncrypt(enKey, key, db, payLoad)
	require.Nil(t, err)
	isExists := store.Has(key, db)
	require.Equal(t, isExists, true)
	_, reader, err := store.ReadDecrypt(enKey, key, db)
	require.Nil(t, err)
	newData, err := io.ReadAll(reader)
	require.Nil(t, err)
	require.Equal(t, string(newData), data)
	store.Clear()
}

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
	store := newStore()
	db := "my_db"
	key := "fatcatpicture"
	data := "some jpg bytes"
	payLoad := bytes.NewReader([]byte(data))
	_, err := store.writeStream(key, db, payLoad)
	require.Nil(t, err)
	_, bytes, err := store.Read(key, db)
	require.Nil(t, err)
	src, err := io.ReadAll(bytes)
	require.Nil(t, err)
	require.Equal(t, string(src), data)
	exists := store.Has(key, db)
	require.Equal(t, exists, true)
	err = store.Delete(key, db)
	require.Nil(t, err)
	err = store.Clear()
	require.Nil(t, err)
}

func Test_MuiliReadWrites(t *testing.T) {
	tests := []struct {
		name  string
		store Store
		db    string
		key1  string
		key2  string
		data1 string
		data2 string
	}{
		{
			name:  "test1",
			store: *newStore(),
			db:    "my_db",
			key1:  "key1",
			key2:  "key2",
			data1: "data for key1",
			data2: "data for key2",
		},
	}
	for _, tt := range tests {
		// writing, checking and reading
		payLoad1 := bytes.NewReader([]byte(tt.data1))
		_, err := tt.store.writeStream(tt.key1, tt.db, payLoad1)
		require.Nil(t, err)
		exist := tt.store.Has(tt.key1, tt.db)
		require.Equal(t, exist, true)
		_, bytes1, err := tt.store.Read(tt.key1, tt.db)
		require.Nil(t, err)
		src, err := io.ReadAll(bytes1)
		require.Nil(t, err)
		require.Equal(t, string(src), tt.data1)
		payLoad2 := bytes.NewReader([]byte(tt.data2))
		_, err = tt.store.writeStream(tt.key2, tt.db, payLoad2)
		require.Nil(t, err)
		exist2 := tt.store.Has(tt.key2, tt.db)
		require.Equal(t, exist2, true)
		_, bytes2, err := tt.store.Read(tt.key2, tt.db)
		require.Nil(t, err)
		src2, err := io.ReadAll(bytes2)
		require.Nil(t, err)
		require.Equal(t, string(src2), tt.data2)
		// deleting
		err = tt.store.Delete(tt.key1, tt.db)
		require.Nil(t, err)
		notExist1 := tt.store.Has(tt.key1, tt.db)
		require.Equal(t, notExist1, false)
		exist2 = tt.store.Has(tt.key2, tt.db)
		require.Equal(t, exist2, true)
		err = tt.store.Delete(tt.key2, tt.db)
		require.Nil(t, err)
		notExist2 := tt.store.Has(tt.key2, tt.db)
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
		db := "my_db"
		_, err := store.writeStream(key, db, payLoad)
		require.Nil(t, err)
		isExist := store.Has(key, db)
		require.Equal(t, isExist, true)
		_, reder, err := store.Read(key, db)
		require.Nil(t, err)
		bytes, err := io.ReadAll(reder)
		require.Nil(t, err)
		require.Equal(t, bytes, []byte("test bytes"))
		err = store.Delete(key, db)
		require.Nil(t, err)
		isExist = store.Has(key, db)
		require.Equal(t, isExist, false)
	}
	err := store.Clear()
	require.Nil(t, err)
}
func Test_Clear(t *testing.T) {
	store := newStore()
	byte := bytes.NewReader([]byte("test"))

	db := "my_db"

	_, err := store.writeStream("test", db, byte)
	require.Nil(t, err)
	err = store.Clear()
	require.Nil(t, err)
}

func newStore() *Store {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
		Log:               logger,
	}
	return NewStore(opts)

}
