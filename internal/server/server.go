package server

import (
	"bytes"
	"encoding/gob"
	"errors"
	"gofilesystem/internal/p2p"
	"gofilesystem/internal/store"
	"io"
	"log/slog"
	"sync"
	"time"
)

var (
	ErrPeerNotExists = errors.New("peer doesn't exists")
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

type MessageStoreFile struct {
	Key  string
	Size int64
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
	var (
		log        = fs.Log.With(slog.String("op", op))
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	log.Info("storing data with key", slog.String("key", key))
	size, err := fs.Store.Write(key, tee)
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return err
	}
	msg := Message{
		PayLoad: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}
	msgBuf := new(bytes.Buffer)
	if err := gob.NewEncoder(msgBuf).Encode(msg); err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return err
	}
	for _, peer := range fs.peers {
		if err := peer.Send(msgBuf.Bytes()); err != nil {
			log.Error("got error", slog.String("error", err.Error()))
			return err
		}
	}
	time.Sleep(4 * time.Second)
	for _, peer := range fs.peers {
		n, err := io.Copy(peer, fileBuffer)
		if err != nil {
			return err
		}
		log.Info("received and written", slog.Int64("bytes", n))
	}
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
			if err := fs.handleMessage(msg.From, &m); err != nil {
				log.Error("got error", slog.String("error", err.Error()))
				return
			}
		case <-fs.quitch:
			return
		}
	}
}

func (fs *FileServer) handleMessage(form string, msg *Message) error {
	const op = "server.handleMessage"
	log := fs.Log.With(slog.String("op", op))
	log.Info("starting handling message", slog.String("from", form))
	switch v := msg.PayLoad.(type) {
	case MessageStoreFile:
		if err := fs.handleMessageStoreFile(form, v); err != nil {
			log.Error("got error", slog.String("error", err.Error()))
			return err
		}
	}
	log.Info("message handled", slog.String("from", form))
	return nil
}

func (fs *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	const op = "server.handleMessageStoreFile"
	log := fs.Log.With(slog.String("op", op))
	peer, ok := fs.peers[from]
	if !ok {
		log.Error("got error", slog.String("error", ErrPeerNotExists.Error()))
		return ErrPeerNotExists
	}
	if _, err := fs.Store.Write(msg.Key, io.LimitReader(peer, msg.Size)); err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return err
	}
	log.Info("handled message from", slog.String("from", from), slog.Any("message", msg))
	peer.(*p2p.TCPPeer).Wg.Done()
	return nil
}

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
