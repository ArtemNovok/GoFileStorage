package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
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

// Has returns true if given key is exists or false if doesn't
func (s *Store) Has(key string) bool {
	const op = "store.Has"
	pathKey := s.PathTransformFunc(key)
	fullPath := filepath.Join(s.Root, pathKey.FullPath())
	_, err := os.Stat(fullPath)
	fmt.Printf("%s: checking if key %s exists\n", op, key)
	return !errors.Is(err, fs.ErrNotExist)
}

// Delete deletes data and all path for given key
func (s *Store) Delete(key string) error {
	const op = "store.Delete"
	pathKey := s.PathTransformFunc(key)
	fmt.Printf("%s: removing key %s\n", op, key)
	fullPathToRoot := filepath.Join(s.Root, pathKey.RootPathDir())
	return os.RemoveAll(fullPathToRoot)
}

// Read returns io.Reader for further logic with out need to close file
func (s *Store) Read(key string) (io.Reader, error) {
	const op = "store.Read"
	fmt.Printf("%s: reading key %s\n", op, key)
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// readStream returns file stored at given key
func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPath := filepath.Join(s.Root, pathKey.FullPath())
	return os.Open(fullPath)
}

// writeStream writes data from io.Reader and stores it at path generated from given key
func (s *Store) writeStream(key string, r io.Reader) error {
	const op = "store.writeStream"
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := filepath.Join(s.Root, pathKey.PathName)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return err
	}

	pathAndFileName := filepath.Join(s.Root, pathKey.FullPath())
	f, err := os.Create(pathAndFileName)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}
	fmt.Printf("%s: written (%d) bytes to disk: %s\n", op, n, pathAndFileName)
	return nil
}
