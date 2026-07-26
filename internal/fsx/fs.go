package fsx

import (
	"os"
)

type FS interface {
	MkdirPrivate(path string, perm os.FileMode) error
	WriteDurable(dir, name string, data []byte) (string, error)
	RenameAtomic(oldpath, newpath string) error
	SyncParent(dir string) error
}
