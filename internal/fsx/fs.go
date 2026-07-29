package fsx

import (
	"context"
	"os"
	"time"
)

type UnlockFunc func() error

type Capabilities struct {
	SupportAtomicReplace        bool   `json:"supportAtomicReplace"`
	SupportFileLocking          bool   `json:"supportFileLocking"`
	FilesystemType              string `json:"filesystemType"`
	RootPath                    string `json:"rootPath"`
	SupportOwnerOnlyPermissions bool   `json:"supportOwnerOnlyPermissions"`
}

type FS interface {
	MkdirPrivate(path string, perm os.FileMode) error
	WriteDurable(dir, name string, data []byte) (string, error)
	RenameAtomic(oldpath, newpath string) error
	SyncParent(dir string) error

	Lock(ctx context.Context, target string, timeout time.Duration, exclusive bool) (UnlockFunc, error)
	Backup(target, backupPath string) (sha256 string, err error)
	ReplaceDurable(tempPath, targetPath string) error
	Capabilities(root string) (Capabilities, error)
}
