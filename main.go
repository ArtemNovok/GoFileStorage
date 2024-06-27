package main

import (
	"encoding/gob"
	"fmt"
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
	s2 := makeServer(":4000", "4000", logger, ":3000")
	go func() {
		log.Fatal(s.Start())
	}()
	go s2.Start()
	time.Sleep(2 * time.Second)
	// for i := 0; i < 45; i++ {
	i := 1
	key := fmt.Sprintf("mydata%v", i)
	// payload := fmt.Sprintf("so much data %v", i)
	// data := bytes.NewReader([]byte(payload))
	// s2.StoreData(key, data)
	// time.Sleep(5 * time.Millisecond)
	// }
	_, r, err := s2.Get(key)
	if err != nil {
		logger.Error("got error", slog.String("error", err.Error()))
	}
	b, err := io.ReadAll(r)
	if err != nil {
		logger.Error("got error", slog.String("error", err.Error()))
	}
	fmt.Println(string(b))
	// _, err = s.Get("mydata")
	// if err != nil {
	// 	logger.Error("got error", slog.String("error", err.Error()))
	// }
	logger.Info("done")
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
