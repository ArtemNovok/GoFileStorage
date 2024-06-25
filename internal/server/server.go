package server

import (
	"gofilesystem/internal/p2p"
	"gofilesystem/internal/store"
	"log/slog"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc store.PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
	Log               *slog.Logger
}
type FileServer struct {
	FileServerOpts
	Store  *store.Store
	quitch chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := store.StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
		Log:               opts.Log,
	}
	return &FileServer{
		FileServerOpts: opts,
		Store:          store.NewStore(storeOpts),
		quitch:         make(chan struct{}),
	}
}

func (fs *FileServer) loop() {
	const op = "server.loop"
	log := fs.Log.With(slog.String("op", op))
	defer func() {
		log.Info("server stopped dur to user stop action")
	}()
	for {
		select {
		case msg := <-fs.Transport.Consume():
			log.Info("got message", slog.Any("msg", msg))
		case <-fs.quitch:
			return
		}
	}
}

func (fs *FileServer) Stop() {
	close(fs.quitch)
}

func (fs *FileServer) bootstrapNetwork() error {
	return nil
}

func (fs *FileServer) Start() error {
	const op = "server.Start"
	log := fs.Log.With(slog.String("op", op))
	err := fs.Transport.ListenAndAccept()
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return err
	}
	fs.loop()
	return nil
}
