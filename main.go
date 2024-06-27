package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"gofilesystem/internal/logger/mylogger"
	"gofilesystem/internal/p2p"
	"gofilesystem/internal/server"
	"gofilesystem/internal/store"
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
	logger := setUpLogger()
	s := makeServer(":3000", "", logger)
	s2 := makeServer(":4000", "", logger, ":3000")
	go func() {
		log.Fatal(s.Start())
	}()
	go s2.Start()
	time.Sleep(2 * time.Second)
	data := bytes.NewReader([]byte("so much data here"))
	s2.StoreData("mydata", data)
	select {}
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
