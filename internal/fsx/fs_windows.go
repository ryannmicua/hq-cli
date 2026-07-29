//go:build windows

package fsx

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	modkernel32     = syscall.NewLazyDLL("kernel32.dll")
	procMoveFileExW = modkernel32.NewProc("MoveFileExW")
)

const (
	moveFileReplaceExisting = 0x1
	moveFileWriteThrough    = 0x8
)

func NewFS() FS {
	return &windowsFS{}
}

func IsSupported() bool {
	return true
}

type windowsFS struct{}

func (f *windowsFS) MkdirPrivate(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return fmt.Errorf("mkdir private %q: %w", path, err)
	}
	return nil
}

func (f *windowsFS) WriteDurable(dir, name string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create temp dir %q: %w", dir, err)
	}
	tmpPath := filepath.Join(dir, name)
	fd, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("create temp file %q: %w", tmpPath, err)
	}
	_, err = fd.Write(data)
	if err != nil {
		fd.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("write temp file %q: %w", tmpPath, err)
	}
	if err := fd.Sync(); err != nil {
		fd.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("fsync temp file %q: %w", tmpPath, err)
	}
	if err := fd.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("close temp file %q: %w", tmpPath, err)
	}
	return tmpPath, nil
}

func (f *windowsFS) RenameAtomic(oldpath, newpath string) error {
	oldp, err := syscall.UTF16PtrFromString(oldpath)
	if err != nil {
		return fmt.Errorf("convert oldpath %q: %w", oldpath, err)
	}
	newp, err := syscall.UTF16PtrFromString(newpath)
	if err != nil {
		return fmt.Errorf("convert newpath %q: %w", newpath, err)
	}
	ret, _, err := procMoveFileExW.Call(
		uintptr(unsafe.Pointer(oldp)),
		uintptr(unsafe.Pointer(newp)),
		moveFileReplaceExisting|moveFileWriteThrough,
	)
	if ret == 0 {
		return fmt.Errorf("MoveFileExW %q -> %q: %v", oldpath, newpath, err)
	}
	return nil
}

func (f *windowsFS) SyncParent(dir string) error {
	// Windows does not support fsync on directories.
	// MoveFileExW with MOVEFILE_WRITE_THROUGH provides the durability guarantee.
	return nil
}
