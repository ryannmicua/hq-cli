//go:build !windows

package fsx

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

func (f *posixFS) Backup(target, backupPath string) (string, error) {
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

func (f *posixFS) ReplaceDurable(tempPath, targetPath string) error {
	if err := os.Rename(tempPath, targetPath); err != nil {
		return fmt.Errorf("rename replace %q -> %q: %w", tempPath, targetPath, err)
	}
	return nil
}

func (f *posixFS) Capabilities(root string) (Capabilities, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return Capabilities{}, fmt.Errorf("resolve root: %w", err)
	}

	return Capabilities{
		SupportAtomicReplace:        true,
		SupportFileLocking:          true,
		FilesystemType:              "unknown",
		RootPath:                    rootPath,
		SupportOwnerOnlyPermissions: true,
	}, nil
}
