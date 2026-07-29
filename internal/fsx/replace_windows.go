//go:build windows

package fsx

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func (f *windowsFS) Backup(target, backupPath string) (string, error) {
	input, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("read target for backup: %w", err)
	}

	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}

	if err := os.WriteFile(backupPath, input, 0600); err != nil {
		return "", fmt.Errorf("write backup: %w", err)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(input))
	return hash, nil
}

func (f *windowsFS) ReplaceDurable(tempPath, targetPath string) error {
	oldp, err := syscall.UTF16PtrFromString(tempPath)
	if err != nil {
		return fmt.Errorf("convert temp path: %w", err)
	}
	newp, err := syscall.UTF16PtrFromString(targetPath)
	if err != nil {
		return fmt.Errorf("convert target path: %w", err)
	}

	ret, _, err := procMoveFileExW.Call(
		uintptr(unsafe.Pointer(oldp)),
		uintptr(unsafe.Pointer(newp)),
		moveFileReplaceExisting|moveFileWriteThrough,
	)
	if ret == 0 {
		return fmt.Errorf("MoveFileExW replace %q -> %q: %v", tempPath, targetPath, err)
	}
	return nil
}

func (f *windowsFS) Capabilities(root string) (Capabilities, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return Capabilities{}, fmt.Errorf("resolve root: %w", err)
	}

	return Capabilities{
		SupportAtomicReplace:        true,
		SupportFileLocking:          true,
		FilesystemType:              "ntfs",
		RootPath:                    rootPath,
		SupportOwnerOnlyPermissions: true,
	}, nil
}
