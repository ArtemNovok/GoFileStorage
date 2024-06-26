package server

import (
	"bytes"
	"encoding/gob"
	"gofilesystem/internal/p2p"
	"gofilesystem/internal/store"
	"io"
	"log/slog"
	"sync"
	"time"
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
	Store    *store.Store
	peerLock sync.Mutex
	peers    map[string]p2p.Peer
	quitch   chan struct{}
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
		peers:          make(map[string]p2p.Peer),
		quitch:         make(chan struct{}),
	}
}

type Message struct {
	PayLoad any
}

func (fs *FileServer) boardCast(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}
func (fs *FileServer) StoreData(key string, r io.Reader) error {
	const op = "server.StoreData"
	log := fs.Log.With(slog.String("op", op))
	log.Info("storing data with key", slog.String("key", key))

	buf := new(bytes.Buffer)
	msg := Message{
		PayLoad: []byte("storagekey"),
	}
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return err
	}
	for _, peer := range fs.peers {
		if err := peer.Send(buf.Bytes()); err != nil {
			log.Error("got error", slog.String("error", err.Error()))
			return err
		}
	}
	time.Sleep(3 * time.Second)
	payload := []byte("THIS IS BIG FILE")
	for _, peer := range fs.peers {
		if err := peer.Send(payload); err != nil {
			log.Error("got error", slog.String("error", err.Error()))
			return err
		}
	}
	// buf := new(bytes.Buffer)
	// tee := io.TeeReader(r, buf)

	// if err := fs.Store.Write(key, tee); err != nil {
	// 	log.Error("got error", slog.String("error", err.Error()))
	// 	return err
	// }

	// p := &DataMessage{
	// 	Key:  key,
	// 	Data: buf.Bytes(),
	// }

	// log.Info("stored bytes", slog.Any("bytes", buf.Bytes()))
	// return fs.boardCast(&Message{
	// 	From:    fs.Transport.ListenAddress(),
	// 	PayLoad: p,
	// })
	return nil
}

func (fs *FileServer) OnPeer(p p2p.Peer) error {
	const op = "server.OnPeer"
	log := fs.Log.With(slog.String("op", op))
	fs.peerLock.Lock()
	defer fs.peerLock.Unlock()
	fs.peers[p.RemoteAddr().String()] = p
	log.Info("peer added to peers", slog.Any("address", p.RemoteAddr()))
	return nil
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
			var m Message
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&m); err != nil {
				log.Error("got error", slog.String("error", err.Error()))
			}
			log.Info("got message", slog.Any("message", m))
			peer, ok := fs.peers[msg.From]
			if !ok {
				panic("no peer in peers map")
			}
			b := make([]byte, 1024)
			n, err := peer.Read(b)
			if err != nil {
				panic(err)
			}
			log.Info("got big message", slog.String("message", string(b[:n])))
			// if err := fs.handleMessage(&m); err != nil {
			// 	log.Error("got error", slog.String("error", err.Error()))
			// }
		case <-fs.quitch:
			return
		}
	}
}

// func (fs *FileServer) handleMessage(msg *Message) error {
// 	switch v := msg.PayLoad.(type) {
// 	case *Message:
// 		fmt.Printf("recived data %+v\n", v)
// 	}
// 	return nil
// }

func (fs *FileServer) Stop() {
	close(fs.quitch)
}

func (fs *FileServer) bootstrapNetwork() error {
	const op = "server.bootstrapNetwork"
	log := fs.Log.With(slog.String("op", op))
	for _, addr := range fs.BootstrapNodes {
		go func(address string) {
			if err := fs.Transport.Dial(address); err != nil {
				log.Error("got error", slog.String("error", err.Error()))

			}
			log.Info("connected to peer", slog.String("address", addr))
		}(addr)
	}
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
	if len(fs.BootstrapNodes) != 0 {
		fs.bootstrapNetwork()
	}
	fs.loop()
	return nil
}
