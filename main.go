package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"gofilesystem/internal/encrypt"
	"gofilesystem/internal/logger/mylogger"
	"gofilesystem/internal/p2p"
	"gofilesystem/internal/server"
	"gofilesystem/internal/store"
	"io"
	"log"
	"log/slog"
	"os"
	"time"
)

const (
	defaultLoggerLevel = "DEBUG"
)

func OnPeer(p p2p.Peer) error {
	fmt.Println("execute OnPeer function")
	p.Close()
	return nil
}
func main() {
	gob.Register(server.MessageStoreFile{})
	gob.Register(server.MessageGetFile{})
	logger := setUpLogger()
	s := makeServer(":3000", "3000", logger)
	s2 := makeServer(":7070", "7070", logger, ":3000")
	s3 := makeServer(":8000", "8000", logger, ":3000", ":7070")
	go func() {
		log.Fatal(s.Start())
	}()
	time.Sleep(300 * time.Millisecond)
	go s2.Start()
	time.Sleep(1 * time.Second)
	go s3.Start()
	time.Sleep(2 * time.Second)
	start := time.Now()
	key := "mydata"
	payload := "so much data"
	data := bytes.NewReader([]byte(payload))
	s3.StoreData(key, data)
	s3.Store.Delete(key)
	_, r, err := s3.Get(key)
	if err != nil {
		panic(err)
	}
	b, err := io.ReadAll(r)
	fmt.Println(string(b))
	time.Sleep(10 * time.Millisecond)

	logger.Info("done")
	logger.Info("took", slog.Duration("time", time.Duration(time.Since(start).Seconds())))
}
func makeServer(Addr, root string, logger *slog.Logger, nodes ...string) *server.FileServer {
	tcpTrOpts := p2p.TCPTransportOpts{
		ListenerAddress: Addr,
		ShakeHandsFunc:  p2p.NoHandshakeFunc,
		Decoder:         &p2p.DefaultDecoder{},
		Log:             logger,
	}
	tcpTransport := p2p.NewTCPTransport(tcpTrOpts)
	serverOpts := server.FileServerOpts{
		EncKey:            encrypt.NewEcryptionKey(),
		StorageRoot:       root,
		PathTransformFunc: store.CASPathTransformFunc,
		Transport:         tcpTransport,
		Log:               logger,
		BootstrapNodes:    nodes,
	}
	s := server.NewFileServer(serverOpts)
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
