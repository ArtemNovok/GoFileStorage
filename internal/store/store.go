package store

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"gofilesystem/internal/encrypt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaulRootDirectoryName = "pupalupa"
)

// PathTransFormFunc is a function that generate PathKey from the given key string
type PathTransformFunc func(string) PathKey

// PathKey represents the path to the file and the file name
type PathKey struct {
	// PathName is path to the dir where file is
	PathName string
	// FileName is name of the file in witch data is stored
	FileName string
}

// FullPath name returns full path to the file
func (p *PathKey) FullPath() string {
	return filepath.Join(p.PathName, p.FileName)
}

// RootPathDIr returns root directory of the path to the file
func (p *PathKey) RootPathDir() string {
	paths := strings.Split(p.PathName, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

// CASPathTransformFunc return PathKey struct generated from given key
func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])
	blockSize := 5
	sliceLen := len(hashStr) / blockSize
	paths := make([]string, sliceLen)
	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, i*blockSize+blockSize
		paths[i] = hashStr[from:to]
	}

	return PathKey{
		PathName: strings.Join(paths, "/"),
		FileName: hashStr,
	}
}

// StoreOpts are options for store struct
type StoreOpts struct {
	// Root is a name of the directory where all dirs/files are stored
	Root              string
	PathTransformFunc PathTransformFunc
	Log               *slog.Logger
}

// Store represents the struct responsible for storing and managing data on the lowest level
type Store struct {
	StoreOpts
}

// NewStore generate new store with given options
func NewStore(opts StoreOpts) *Store {
	if len(opts.Root) == 0 {
		opts.Root = defaulRootDirectoryName
	}
	return &Store{
		StoreOpts: opts,
	}
}

func DefaultPathTransformFunc(key string) PathKey {
	return PathKey{
		PathName: key,
		FileName: key,
	}
}

// Clear deletes whole root directory of the store with all content
func (s *Store) Clear() error {
	const op = "store.Clean"
	log := s.Log.With(slog.String("op", op))
	err := os.RemoveAll(s.Root)
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return err
	}
	log.Info("root dir removed", slog.String("dir", s.Root))
	return nil
}

// Has returns true if given key is exists or false if doesn't
func (s *Store) Has(key string) bool {
	const op = "store.Has"
	log := s.Log.With(slog.String("op", op))
	pathKey := s.PathTransformFunc(key)
	fullPath := filepath.Join(s.Root, pathKey.FullPath())
	_, err := os.Stat(fullPath)
	log.Info("checking if key is exists", slog.String("key", key))
	return !errors.Is(err, fs.ErrNotExist)
}

// Delete deletes data and all path for given key
func (s *Store) Delete(key string) error {
	const op = "store.Delete"
	log := s.Log.With(slog.String("op", op))
	pathKey := s.PathTransformFunc(key)
	log.Info("deleting key", slog.String("key", key))
	fullPathToRoot := filepath.Join(s.Root, pathKey.RootPathDir())
	return os.RemoveAll(fullPathToRoot)
}

// Read returns io.Reader for further logic with out need to close file
func (s *Store) Read(key string) (int64, io.Reader, error) {
	const op = "store.Read"
	log := s.Log.With(slog.String("op", op))
	log.Info("reading key", slog.String("key", key))
	f, err := s.readStream(key)
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return 0, nil, err
	}
	defer f.Close()
	buf := new(bytes.Buffer)
	n, err := io.Copy(buf, f)
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return 0, nil, err
	}
	return n, buf, nil
}

// readStream returns file stored at given key
func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPath := filepath.Join(s.Root, pathKey.FullPath())
	return os.Open(fullPath)
}

// Write data from io.Reader and store it at path generated from given key
func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}
func (s *Store) WriteDecrypt(enKey []byte, key string, r io.Reader) (int64, error) {
	return s.writeDecrypt(enKey, key, r)
}
func (s *Store) writeDecrypt(enKey []byte, key string, r io.Reader) (int64, error) {
	const op = "store.writeDecrypt"
	log := s.Log.With(slog.String("op", op))
	f, pathAndFileName, err := s.openFileForWriting(key)
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return 0, err
	}
	defer f.Close()
	n, err := encrypt.CopyDecrypt(enKey, r, f)
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return 0, err
	}
	log.Info("written bytes to disk", slog.Int64("bytes", int64(n)), slog.String("disk", pathAndFileName))
	return int64(n), nil
}

// writeStream writes data from io.Reader and stores it at path generated from given key
func (s *Store) writeStream(key string, r io.Reader) (int64, error) {
	const op = "store.writeStream"
	log := s.Log.With(slog.String("op", op))
	f, pathAndFileName, err := s.openFileForWriting(key)
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return 0, err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return 0, err
	}
	log.Info("written bytes to disk", slog.Int64("bytes", n), slog.String("disk", pathAndFileName))
	return n, nil
}

// openFIleForWriting create if needs and opens the file for writing,
// you must manually close file in top level func, after you logic with this file
// is executed
func (s *Store) openFileForWriting(key string) (*os.File, string, error) {
	const op = "store.openFileForWriting"
	log := s.Log.With(slog.String("op", op))
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := filepath.Join(s.Root, pathKey.PathName)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return nil, "", err
	}

	pathAndFileName := filepath.Join(s.Root, pathKey.FullPath())
	f, err := os.Create(pathAndFileName)
	if err != nil {
		log.Error("got error", slog.String("error", err.Error()))
		return nil, "", err
	}
	return f, pathAndFileName, nil
}
