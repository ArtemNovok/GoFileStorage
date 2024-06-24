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

// PathTransFormFunc is a function that generate PathKey from the given key string
type PathTransformFunc func(string) PathKey

// PathKey represents the path to the file and the file name
type PathKey struct {
	PathName string
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
	PathTransformFunc PathTransformFunc
}

// Store represents the struct responsible for storing and managing data on the lowest level
type Store struct {
	StoreOpts
}

// NewStore generate new store with given options
func NewStore(opts StoreOpts) *Store {
	return &Store{
		StoreOpts: opts,
	}
}

func DefaultPathTransformFunc(key string) string {
	return key
}

// Has returns true if given key is exists or false if doesn't
func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	_, err := os.Stat(pathKey.FullPath())
	return !errors.Is(err, fs.ErrNotExist)
}

// Delete deletes data and all path for given key
func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)
	return os.RemoveAll(pathKey.RootPathDir())
}

// Read returns io.Reader for further logic with out need to close file
func (s *Store) Read(key string) (io.Reader, error) {
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
	return os.Open(pathKey.FullPath())
}

// writeStream writes data from io.Reader and stores it at path generated from given key
func (s *Store) writeStream(key string, r io.Reader) error {
	pathName := s.PathTransformFunc(key)
	if err := os.MkdirAll(pathName.PathName, os.ModePerm); err != nil {
		return err
	}

	pathAndFileName := pathName.FullPath()
	f, err := os.Create(pathAndFileName)
	if err != nil {
		return nil
	}
	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}
	fmt.Printf("written (%d) bytes to disk: %s", n, pathAndFileName)
	return nil
}
