package server

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"gofilesystem/internal/logger/mylogger"
	"gofilesystem/internal/p2p"
	"gofilesystem/internal/store"
	"io"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultLoggerLevel = "DEV"
)

// This test tests main functions of the server
func Test_Server(t *testing.T) {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
	logger := setUpLogger()
	s := makeServer(":3000", "3000", logger)
	s2 := makeServer(":4000", "4000", logger, ":3000")
	go func() {
		log.Fatal(s.Start())
	}()
	go s2.Start()
	time.Sleep(2 * time.Second)
	start := time.Now()
	// Create keys for one server and than it will create them for other
	for i := 0; i < 45; i++ {
		key := fmt.Sprintf("mydata%v", i)
		payload := fmt.Sprintf("so much data %v", i)
		data := bytes.NewReader([]byte(payload))
		s2.StoreData(key, data)
		time.Sleep(5 * time.Millisecond)
	}
	// delete one server data and check if data is actually removed
	s2.Store.Clear()
	require.Error(t, os.Chdir("4000"))
	// Start requesting deleting keys to force server to look up for them in other servers
	for i := 0; i < 45; i++ {
		key := fmt.Sprintf("mydata%v", i)
		_, r, err := s2.Get(key)
		if err != nil {
			logger.Error("got error", slog.String("error", err.Error()))
		}
		b, err := io.ReadAll(r)
		if err != nil {
			logger.Error("got error", slog.String("error", err.Error()))
		}
		fmt.Println(string(b))
	}
	// Make sure that both servers has all files and that files are equal
	for i := 0; i < 45; i++ {
		key := fmt.Sprintf("mydata%v", i)
		n2, r2, err := s2.Get(key)
		require.Nil(t, err)
		n, r, err := s.Get(key)
		require.Nil(t, err)
		assert.Equal(t, n, n2)
		b2, err := io.ReadAll(r2)
		require.Nil(t, err)
		b, err := io.ReadAll(r)
		require.Nil(t, err)
		require.Equal(t, b, b2)
	}
	// Removing all data from servers and making sure that data is actually removed
	s2.Store.Clear()
	s.Store.Clear()
	require.Error(t, os.Chdir("3000"))
	require.Error(t, os.Chdir("4000"))
	logger.Info("done")
	logger.Info("took", slog.Duration("time", time.Duration(time.Since(start).Seconds())))
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
