package server

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"gofilesystem/internal/p2p"
	"gofilesystem/internal/store"
	"io"
	"log/slog"
	"sync"
	"time"
)

var (
	ErrPeerNotExists = errors.New("peer doesn't exists")
	ErrKeyNotExists  = errors.New("key doesn't exists")
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
type MessageGetFile struct {
	Key string
}

func (fs *FileServer) stream(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

func (fs *FileServer) copyStream(buffer io.Reader) (int64, error) {
	const op = "server.copyStream"
	peers := []io.Writer{}
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	n, err := io.Copy(mw, buffer)
	if err != nil {
		return 0, fmt.Errorf("%s:%w", op, err)
	}
	return n, nil
}

func (fs *FileServer) broadcast(msg *Message) error {
	const op = "server.broadcast"
	msgBuf := new(bytes.Buffer)
	if err := gob.NewEncoder(msgBuf).Encode(msg); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	for _, peer := range fs.peers {
		if err := peer.Send(msgBuf.Bytes()); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}
	return nil
}

func (fs *FileServer) Get(key string) (io.Reader, error) {
	const op = "server.Get"
	log := fs.Log.With(slog.String("op", op))
	if fs.Store.Has(key) {
		return fs.Store.Read(key)
	}
	log.Info("didn't find key locally", slog.String("key", key))

	msg := Message{
		PayLoad: MessageGetFile{
			Key: key,
		},
	}
	if err := fs.broadcast(&msg); err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return nil, err
	}
	time.Sleep(4 * time.Second)
	for _, peer := range fs.peers {
		buf := new(bytes.Buffer)
		n, err := io.CopyN(buf, peer, 10)
		if err != nil {
			log.Error("got error", slog.String("error", err.Error()))
			return nil, err
		}
		log.Info("received bytes from peer", slog.Int64("bytes", n), slog.String("from", peer.RemoteAddr().String()))
	}
	select {}
	return nil, ErrKeyNotExists
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

	if err := fs.broadcast(&msg); err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return err
	}

	time.Sleep(4 * time.Second)
	n, err := fs.copyStream(fileBuffer)
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return err
	}
	log.Info("received and written", slog.Int64("bytes", n))
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
		log.Info("server stopped due to error or user stop action")
	}()
	for {
		select {
		case msg := <-fs.Transport.Consume():
			var m Message
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&m); err != nil {
				log.Error("got decoding error", slog.String("error", err.Error()))
			}
			if err := fs.handleMessage(msg.From, &m); err != nil {
				log.Error("got error", slog.String("error", err.Error()))
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
	case MessageGetFile:
		if err := fs.handleGetMessageFile(form, v); err != nil {
			log.Error("got error", slog.String("error", err.Error()))
			return err
		}
	}

	log.Info("message handled", slog.String("from", form))
	return nil
}

func (fs *FileServer) handleGetMessageFile(from string, msg MessageGetFile) error {
	const op = "server.handleGetMessageFile"
	log := fs.Log.With(slog.String("op", op))
	log.Info("need to find file and if it exists send it over wire")
	if !fs.Store.Has(msg.Key) {
		log.Info("key doesn't exists locally")
		return ErrKeyNotExists
	}
	log.Info("found file for peer", slog.String("peer", from))
	r, err := fs.Store.Read(msg.Key)
	if err != nil {
		return err
	}
	peer, ok := fs.peers[from]
	if !ok {
		return ErrPeerNotExists
	}
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}
	log.Info("sended file to a peer", slog.Int64("bytes", n))
	return nil
}

func (fs *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	const op = "server.handleMessageStoreFile"
	log := fs.Log.With(slog.String("op", op))
	peer, ok := fs.peers[from]
	if !ok {
		return ErrPeerNotExists
	}
	if _, err := fs.Store.Write(msg.Key, io.LimitReader(peer, msg.Size)); err != nil {
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
