package server

import (
	"fmt"
	"gofilesystem/internal/p2p"
	"gofilesystem/internal/store"
	"log"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc store.PathTransformFunc
	Transport         p2p.Transport
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
	}
	return &FileServer{
		FileServerOpts: opts,
		Store:          store.NewStore(storeOpts),
		quitch:         make(chan struct{}),
	}
}

func (fs *FileServer) loop() {
	defer func() {
		log.Println("server stopped due to user stop action")
	}()
	for {
		select {
		case msg := <-fs.Transport.Consume():
			fmt.Println(msg)
		case <-fs.quitch:
			return
		}
	}
}

func (fs *FileServer) Stop() {
	close(fs.quitch)
}
func (fs *FileServer) Start() error {
	const op = "server.Start"
	err := fs.Transport.ListenAndAccept()
	if err != nil {
		return err
	}
	fs.loop()
	return nil
}
