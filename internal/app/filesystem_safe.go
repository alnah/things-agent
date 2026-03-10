package app

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	backupDirPerm  = 0o700
	backupFilePerm = 0o600
)

type rootedFile struct {
	*os.File
	root *os.Root
}

func (f *rootedFile) Close() error {
	fileErr := f.File.Close()
	rootErr := f.root.Close()
	if fileErr != nil {
		return fileErr
	}
	return rootErr
}

func openRootedFileRead(path string) (*rootedFile, error) {
	return openRootedFile(path, os.O_RDONLY, 0)
}

func openRootedFileWrite(path string, perm os.FileMode) (*rootedFile, error) {
	return openRootedFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
}

func openRootedFile(path string, flag int, perm os.FileMode) (*rootedFile, error) {
	dir := filepath.Dir(path)
	name := filepath.Base(path)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open root %s: %w", dir, err)
	}
	file, err := root.OpenFile(name, flag, perm)
	if err != nil {
		_ = root.Close()
		return nil, err
	}
	return &rootedFile{File: file, root: root}, nil
}
