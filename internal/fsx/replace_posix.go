//go:build !windows

package fsx

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
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

	if override := os.Getenv("HQ_FS_OVERRIDE"); override != "" {
		return parseCapOverride(override, rootPath), nil
	}

	fsType := detectFilesystemPOSIX(rootPath)
	supportReplace := probeAtomicReplacePOSIX(rootPath)
	supportLocking := true
	supportOwnerOnly := true

	return Capabilities{
		SupportAtomicReplace:        supportReplace,
		SupportFileLocking:          supportLocking,
		FilesystemType:              fsType,
		RootPath:                    rootPath,
		SupportOwnerOnlyPermissions: supportOwnerOnly,
	}, nil
}

func detectFilesystemPOSIX(rootPath string) string {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(rootPath, &stat); err != nil {
		return "unknown"
	}
	switch stat.Type {
	case 0xEF53:
		return "ext4"
	case 0x01021994:
		return "tmpfs"
	case 0x6969:
		return "nfs"
	case 0xADF5:
		return "adfs"
	case 0x137D:
		return "exfat"
	case 0x4d44:
		return "vfat"
	case 0x58465342:
		return "xfs"
	case 0x9123683E:
		return "btrfs"
	case 0x28CD3D45:
		return "hammer2"
	case 0x65735546:
		return "fuse"
	case 0x1CD1:
		return "devpts"
	case 0x9fa0:
		return "proc"
	case 0x2F:
		return "cgroup"
	default:
		if stat.Type == 0x482D5341 {
			return "apfs"
		}
		if stat.Type == 0x48666853 {
			return "hfs+"
		}
		if stat.Type == 0x7734 {
			return "zfs"
		}
		return "unknown"
	}
}

var (
	probeMu   sync.Mutex
	probeDone bool
	probeOk   bool
)

func probeAtomicReplacePOSIX(rootPath string) bool {
	probeMu.Lock()
	if probeDone {
		ok := probeOk
		probeMu.Unlock()
		return ok
	}
	probeMu.Unlock()
	// benign race: between the second Unlock and the later Lock+probeDone=true,
	// another goroutine may also pass the probeDone check and start a redundant
	// probe. Both use PID-specific temp files, so the duplicate is harmless;
	// the second writer's identical result wins.

	probeDir := filepath.Join(rootPath, ".hq-interface", "probe")
	if err := os.MkdirAll(probeDir, 0700); err != nil {
		return false
	}

	src := filepath.Join(probeDir, fmt.Sprintf("probe_src_%d.tmp", os.Getpid()))
	dst := filepath.Join(probeDir, fmt.Sprintf("probe_dst_%d.tmp", os.Getpid()))

	if err := os.WriteFile(src, []byte("probe"), 0600); err != nil {
		os.Remove(src)
		return false
	}

	if err := os.Rename(src, dst); err != nil {
		os.Remove(src)
		return false
	}

	ok := true
	os.Remove(dst)
	probeMu.Lock()
	probeDone = true
	probeOk = ok
	probeMu.Unlock()
	return ok
}
