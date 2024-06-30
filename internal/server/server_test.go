package server

import (
	"bytes"
	"fmt"
	"gofilesystem/internal/encrypt"
	"gofilesystem/internal/logger/mylogger"
	"gofilesystem/internal/p2p"
	"gofilesystem/internal/store"
	"io"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	defaultLoggerLevel = "DEV"
)

// This test tests main functions of the server
func Test_Server(t *testing.T) {
	logger := setUpLogger()
	s := makeServer(":3000", "3000", logger)
	s2 := makeServer(":7070", "7070", logger, ":3000")
	go func() {
		log.Fatal(s.Start())
	}()
	time.Sleep(20 * time.Millisecond)
	go s2.Start()
	time.Sleep(10 * time.Millisecond)
	start := time.Now()
	db := "myDB"
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%v", i)
		data := fmt.Sprintf("Data%v", i)
		payload := bytes.NewReader([]byte(data))
		err := s.StoreData(key, db, payload)
		require.Nil(t, err)
	}

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%v", i)
		has := s2.Store.Has(key, db)
		fmt.Println(has)
		require.Equal(t, has, true)
	}
	s.Store.Clear()
	require.Error(t, os.Chdir("3000"))
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%v", i)
		_, reader, err := s.Get(key, db)
		require.Nil(t, err)
		bytes, err := io.ReadAll(reader)
		require.Nil(t, err)
		_, reader2, err := s2.Get(key, db)
		require.Nil(t, err)
		bytes2, err := io.ReadAll(reader2)
		require.Nil(t, err)
		require.Equal(t, bytes, bytes2)
	}
	s2.Store.Clear()
	s.Store.Clear()
	require.Error(t, os.Chdir("3000"))
	require.Error(t, os.Chdir("7070"))
	logger.Info("done")
	logger.Info("took", slog.Duration("time", time.Duration(time.Since(start).Seconds())))
}

func Test_3Servers(t *testing.T) {
	logger := setUpLogger()
	s := makeServer(":3000", "3000", logger)
	s2 := makeServer(":7070", "7070", logger, ":3000")
	s3 := makeServer(":8000", "8000", logger, ":3000", ":7070")
	go func() {
		log.Fatal(s.Start())
	}()
	time.Sleep(10 * time.Millisecond)
	go func() {
		log.Fatal(s2.Start())
	}()
	time.Sleep(10 * time.Millisecond)
	go func() {
		log.Fatal(s3.Start())
	}()
	time.Sleep(10 * time.Millisecond)
	fmt.Println(s.peers)
	fmt.Println(s2.peers)
	fmt.Println(s3.peers)
	db := "myDB"
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%v", i)
		data := fmt.Sprintf("Data%v", i)
		payload := bytes.NewReader([]byte(data))
		err := s.StoreData(key, db, payload)
		require.Nil(t, err)
	}
	time.Sleep(10 * time.Millisecond)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%v", i)
		has2 := s2.Store.Has(key, db)
		require.Equal(t, has2, true)
		has3 := s3.Store.Has(key, db)
		require.Equal(t, has3, true)
	}
	s.Store.Clear()
	s2.Store.Clear()
	require.Error(t, os.Chdir("3000"))
	require.Error(t, os.Chdir("7070"))
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%v", i)
		_, reader, err := s.Get(key, db)
		require.Nil(t, err)
		bytes, err := io.ReadAll(reader)
		require.Nil(t, err)
		_, reader2, err := s2.Get(key, db)
		require.Nil(t, err)
		bytes2, err := io.ReadAll(reader2)
		require.Nil(t, err)
		require.Equal(t, bytes, bytes2)
		_, reader3, err := s3.Get(key, db)
		require.Nil(t, err)
		bytes3, err := io.ReadAll(reader3)
		require.Equal(t, bytes, bytes3)
	}

	s3.Store.Clear()
	s2.Store.Clear()
	s.Store.Clear()
	require.Error(t, os.Chdir("8000"))
	require.Error(t, os.Chdir("3000"))
	require.Error(t, os.Chdir("7070"))
	logger.Info("done")
}

func Test_Delete(t *testing.T) {
	logger := setUpLogger()
	s := makeServer(":3000", "3000", logger)
	s2 := makeServer(":7070", "7070", logger, ":3000")
	go func() {
		log.Fatal(s.Start())
	}()
	time.Sleep(10 * time.Millisecond)
	go func() {
		log.Fatal(s2.Start())
	}()
	time.Sleep(10 * time.Millisecond)
	db := "myDB"
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%v", i)
		data := fmt.Sprintf("Data%v", i)
		payload := bytes.NewReader([]byte(data))
		err := s.StoreData(key, db, payload)
		require.Nil(t, err)
	}
	time.Sleep(10 * time.Millisecond)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%v", i)
		has2 := s2.Store.Has(key, db)
		require.Equal(t, has2, true)
	}
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%v", i)
		err := s2.Delete(key, db)
		require.Nil(t, err)
		require.False(t, s2.Store.Has(key, db))
	}
	time.Sleep(10 * time.Millisecond)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%v", i)
		require.False(t, s.Store.Has(key, db))
	}
	s.Store.Clear()
	s2.Store.Clear()
	require.Error(t, os.Chdir("3000"))
	require.Error(t, os.Chdir("7070"))
}

func Test_DataBassesSupport(t *testing.T) {
	logger := setUpLogger()
	s := makeServer(":3000", "3000", logger)
	s2 := makeServer(":7070", "7070", logger, ":3000")
	go func() {
		log.Fatal(s.Start())
	}()
	time.Sleep(10 * time.Millisecond)
	go func() {
		log.Fatal(s2.Start())
	}()
	time.Sleep(10 * time.Millisecond)
	db1 := "MyDataBase1"
	db2 := "MyDataBase2"
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%v", i)
		data := fmt.Sprintf("Data%v", i)
		payload := bytes.NewReader([]byte(data))
		// need second payload because after StoreData payload is empty
		payload2 := bytes.NewReader([]byte(data))
		err := s.StoreData(key, db1, payload)
		require.Nil(t, err)
		time.Sleep(50 * time.Millisecond)
		err = s.StoreData(key, db2, payload2)
		require.Nil(t, err)
	}
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%v", i)
		err := s.Delete(key, db1)
		require.Nil(t, err)
		require.True(t, s.Store.Has(key, db2))
		time.Sleep(50 * time.Millisecond)
		require.False(t, s2.Store.Has(key, db1))
		require.True(t, s2.Store.Has(key, db2))
	}
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%v", i)
		data := fmt.Sprintf("Data%v", i)
		payload := bytes.NewReader([]byte(data))
		err := s.StoreData(key, db1, payload)
		require.Nil(t, err)
	}
	s.Store.DeleteDB(db1)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%v", i)
		data := fmt.Sprintf("Data%v", i)
		_, reader, err := s.Get(key, db1)
		require.Nil(t, err)
		b, err := io.ReadAll(reader)
		require.Nil(t, err)
		require.Equal(t, b, []byte(data))
	}
	s.Store.Clear()
	s2.Store.Clear()
	require.Error(t, os.Chdir("3000"))
	require.Error(t, os.Chdir("7070"))

}

func makeServer(Addr, root string, logger *slog.Logger, nodes ...string) *FileServer {
	tcpTrOpts := p2p.TCPTransportOpts{
		ListenerAddress: Addr,
		ShakeHandsFunc:  p2p.NoHandshakeFunc,
		Decoder:         &p2p.DefaultDecoder{},
		Log:             logger,
	}
	tcpTransport := p2p.NewTCPTransport(tcpTrOpts)
	serverOpts := FileServerOpts{
		EncKey:            encrypt.NewEcryptionKey(),
		StorageRoot:       root,
		PathTransformFunc: store.CASPathTransformFunc,
		Transport:         tcpTransport,
		Log:               logger,
		BootstrapNodes:    nodes,
	}
	s := NewFileServer(serverOpts)
	tcpTransport.OnPeer = s.OnPeer
	return s
}
func setUpLogger() *slog.Logger {
	level := os.Getenv("LOGGER_LEVEL")
	if len(level) == 0 {
		level = defaultLoggerLevel
	}
	switch level {
	case "DEV":
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		return logger
	case "PROD":
		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
		return logger
	default:
		logger := SetupPrettyLogger()
		return logger
	}
}
func SetupPrettyLogger() *slog.Logger {
	opts := mylogger.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}
	handler := opts.NewPrettyHandler(os.Stdout)
	return slog.New(handler)
}
