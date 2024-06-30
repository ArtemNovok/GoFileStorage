package server

import (
	"bytes"
	"encoding/binary"
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
	EncKey            []byte
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
	DB   string
	Size int64
}
type MessageGetFile struct {
	Key string
	DB  string
}
type MessageDeleteFile struct {
	Key string
	DB  string
}

func (fs *FileServer) copyStream(buffer io.Reader) (int64, error) {
	const op = "server.copyStream"
	peers := []io.Writer{}
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}
	Sreader := bytes.NewReader([]byte{p2p.IncomingStream})
	mw := io.MultiWriter(peers...)
	io.Copy(mw, Sreader)
	n, err := io.Copy(mw, buffer)
	if err != nil {
		return 0, fmt.Errorf("%s:%w", op, err)
	}
	return int64(n), nil
}

func (fs *FileServer) broadcast(msg *Message) error {
	const op = "server.broadcast"
	msgBuf := new(bytes.Buffer)
	if err := gob.NewEncoder(msgBuf).Encode(msg); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	for _, peer := range fs.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(msgBuf.Bytes()); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}
	return nil
}

func (fs *FileServer) Delete(key string, db string) error {
	const op = "server.Delete"
	log := fs.Log.With(slog.String("op", op))
	if !fs.Store.Has(key, db) {
		log.Info("Key doesn't exists")
		return nil
	}
	err := fs.Store.Delete(key, db)
	if err != nil {
		return fmt.Errorf("%s:%w", op, err)
	}
	log.Info("Key was deleted locally")
	msg := Message{
		PayLoad: MessageDeleteFile{
			Key: key,
			DB:  db,
		},
	}
	if err := fs.broadcast(&msg); err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return fmt.Errorf("%s:%w", op, err)
	}
	// time.Sleep(5 * time.Millisecond)
	return nil
}

func (fs *FileServer) Get(key string, db string) (int64, io.Reader, error) {
	const op = "server.Get"
	log := fs.Log.With(slog.String("op", op))
	if fs.Store.Has(key, db) {
		log.Info("found file locally", slog.String("key", key), slog.String("server address", fs.Transport.Addr()))
		return fs.Store.ReadDecrypt(fs.EncKey, key, db)
	}
	log.Info("didn't find key locally", slog.String("key", key), slog.String("server address", fs.Transport.Addr()))
	msg := Message{
		PayLoad: MessageGetFile{
			Key: key,
			DB:  db,
		},
	}
	if err := fs.broadcast(&msg); err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return 0, nil, err
	}
	time.Sleep(5 * time.Millisecond)
	for _, peer := range fs.peers {
		log.Info("Processing stream", slog.String("Remote address", peer.RemoteAddr().String()), slog.String("myAddress", fs.Transport.Addr()), slog.String("Local address", peer.LocalAddr().String()))
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)
		log.Debug("got file size", slog.Int64("fileSize", fileSize))
		if fileSize == 1 {
			log.Debug("pupa")
			peer.CloseStream()
			continue
		}
		n, err := fs.Store.WriteEncrypt(fs.EncKey, key, db, io.LimitReader(peer, fileSize))
		if err != nil {
			return 0, nil, err
		}
		log.Info("received bytes from peer", slog.Int64("bytes", n), slog.String("from", peer.RemoteAddr().String()))
		peer.CloseStream()
	}
	return fs.Store.ReadDecrypt(fs.EncKey, key, db)
}

func (fs *FileServer) StoreData(key string, db string, r io.Reader) error {
	const op = "server.StoreData"
	var (
		log        = fs.Log.With(slog.String("op", op))
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	log.Info("storing data with key", slog.String("key", key))
	size, err := fs.Store.WriteEncrypt(fs.EncKey, key, db, tee)
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return err
	}
	msg := Message{
		PayLoad: MessageStoreFile{
			Key:  key,
			Size: size - 16,
			DB:   db,
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return err
	}

	time.Sleep(5 * time.Millisecond)
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
	log := fs.Log.With(slog.String("op", op), slog.String("server address", fs.Transport.Addr()))
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
	case MessageDeleteFile:
		if err := fs.handleDeleteMessageFile(form, v); err != nil {
			log.Error("got error", slog.String("error", err.Error()))
			return err
		}
	}

	log.Info("message handled", slog.String("from", form))
	return nil
}

func (fs *FileServer) handleDeleteMessageFile(from string, msg MessageDeleteFile) error {
	const op = "server.handleDeleteMessageFile"
	log := fs.Log.With(slog.String("op", op), slog.String("server address", fs.Transport.Addr()), slog.String("from", from))
	log.Info("Checking if key is exist and deleting it")
	if fs.Store.Has(msg.Key, msg.DB) {
		if err := fs.Store.Delete(msg.Key, msg.DB); err != nil {
			return fmt.Errorf("%s:%w", op, err)
		}
		log.Info("key successfully deleted")
		return nil
	}
	log.Info("kye doesn't exist")
	return nil
}

func (fs *FileServer) handleGetMessageFile(from string, msg MessageGetFile) error {
	const op = "server.handleGetMessageFile"
	log := fs.Log.With(slog.String("op", op), slog.String("server address", fs.Transport.Addr()))
	log.Info("looking for file for the peer", slog.String("peer", from))
	if !fs.Store.Has(msg.Key, msg.DB) {
		peer, ok := fs.peers[from]
		if !ok {
			return ErrPeerNotExists
		}
		log.Info("key doesn't exists locally")
		// Sending empty steam so we can close our peer can close it
		err := peer.Send([]byte{p2p.IncomingStream})
		if err != nil {
			return err
		}
		err = binary.Write(peer, binary.LittleEndian, int64(1))
		if err != nil {
			return err
		}
		return ErrKeyNotExists
	}
	log.Info("found file for peer", slog.String("peer", from))
	fileSize, r, err := fs.Store.ReadDecrypt(fs.EncKey, msg.Key, msg.DB)
	if err != nil {
		return err
	}
	log.Debug("fileSize", slog.Int64("fileSize", fileSize))
	peer, ok := fs.peers[from]
	if !ok {
		return ErrPeerNotExists
	}
	// Sending Stream byte and then sending stream size
	err = peer.Send([]byte{p2p.IncomingStream})
	if err != nil {
		return err
	}
	err = binary.Write(peer, binary.LittleEndian, fileSize)
	if err != nil {
		return err
	}
	log.Debug("sended incoming Stream")
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}
	log.Info("sended file to a peer", slog.Int64("bytes", n), slog.String("peer", from))
	return nil
}

func (fs *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	const op = "server.handleMessageStoreFile"
	log := fs.Log.With(slog.String("op", op))
	peer, ok := fs.peers[from]
	if !ok {
		return ErrPeerNotExists
	}
	if _, err := fs.Store.WriteEncrypt(fs.EncKey, msg.Key, msg.DB, io.LimitReader(peer, msg.Size)); err != nil {
		return err
	}
	log.Info("handled message from", slog.String("from", from), slog.Any("message", msg))
	peer.CloseStream()
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
			log.Info("connected to peer", slog.String("address", address))
		}(addr)
	}
	return nil
}

func (fs *FileServer) Start() error {
	const op = "server.Start"
	log := fs.Log.With(slog.String("op", op))
	fs.init()
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

// init setting up gob register
func (fs *FileServer) init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
	gob.Register(MessageDeleteFile{})
}
