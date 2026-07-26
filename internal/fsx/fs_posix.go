//go:build !windows

package fsx

import (
	"fmt"
	"os"
	"path/filepath"
)

func NewFS() FS {
	return &posixFS{}
}

func IsSupported() bool {
	return true
}

type posixFS struct{}

func (f *posixFS) MkdirPrivate(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return fmt.Errorf("mkdir private %q: %w", path, err)
	}
	return nil
}

func (f *posixFS) WriteDurable(dir, name string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create temp dir %q: %w", dir, err)
	}
	tmpPath := filepath.Join(dir, name)
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return "", fmt.Errorf("write temp file %q: %w", tmpPath, err)
	}
	return tmpPath, nil
}

func (f *posixFS) RenameAtomic(oldpath, newpath string) error {
	if err := os.Rename(oldpath, newpath); err != nil {
		return fmt.Errorf("rename %q -> %q: %w", oldpath, newpath, err)
	}
	return nil
}

func (f *posixFS) SyncParent(dir string) error {
	if dir == "" {
		dir = "."
	}
	parent, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open parent dir %q for sync: %w", dir, err)
	}
	defer parent.Close()
	if err := parent.Sync(); err != nil {
		return fmt.Errorf("sync parent dir %q: %w", dir, err)
	}
	return nil
}
