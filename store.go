package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type PathTransformFunc func(string) string

type StoreOpts struct {
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	return &Store{
		StoreOpts: opts,
	}
}
func DefaultPathTransformFunc(key string) string {
	return key
}
func (s *Store) writeStream(key string, r io.Reader) error {
	pathName := s.PathTransformFunc(key)
	if err := os.MkdirAll(pathName, os.ModePerm); err != nil {
		return err
	}
	fileName := "someFileName"
	pathToFile := filepath.Join(pathName, fileName)
	f, err := os.Create(pathToFile)
	if err != nil {
		return nil
	}
	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}
	fmt.Printf("written (%d) bytes to disk: %s", n, pathToFile)
	return nil
}
