package main

import (
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
	logger := setUpLogger()
	tcpTrOpts := p2p.TCPTransportOpts{
		ListenerAddress: ":3000",
		ShakeHandsFunc:  p2p.NoHandshakeFunc,
		Decoder:         &p2p.DefaultDecoder{},
		Log:             logger,
		// OnPeer:          OnPeer,
	}
	tcpTransport := p2p.NewTCPTransport(tcpTrOpts)
	serverOpts := server.FileServerOpts{
		StorageRoot:       "3000_files",
		PathTransformFunc: store.CASPathTransformFunc,
		Transport:         tcpTransport,
		Log:               logger,
	}
	s := server.NewFileServer(serverOpts)
	go func() {
		time.Sleep(10 * time.Second)
		logger.Info("quitting server")
		s.Stop()
	}()
	if err := s.Start(); err != nil {
		log.Fatalf("got fatal error: %s", err)
	}
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
