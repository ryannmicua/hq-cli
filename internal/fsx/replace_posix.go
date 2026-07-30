//go:build !windows

package fsx

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
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
	// Type fields vary by platform; return the f_type name if recognizable.
	switch stat.Type {
	case 0xEF53: // ext4
		return "ext4"
	case 0x01021994: // tmpfs
		return "tmpfs"
	case 0x6969: // nfs
		return "nfs"
	case 0xADF5: // adfs
		return "adfs"
	case 0x137D: // exfat
		return "exfat"
	case 0x4d44: // fat/vfat
		return "vfat"
	case 0x58465342: // xfs
		return "xfs"
	case 0x9123683E: // btrfs
		return "btrfs"
	case 0x28CD3D45: // hammer2
		return "hammer2"
	case 0x65735546: // fuse
		return "fuse"
	case 0x1CD1: // devpts
		return "devpts"
	case 0x9fa0: // proc
		return "proc"
	case 0x1021994: // ramfs
		return "ramfs"
	case 0x2F: // cgroup
		return "cgroup"
	default:
		// Try macOS/BSD specific types. These are common on Apple filesystems.
		if stat.Type == 0x482D5341 { // APFS (macOS)
			return "apfs"
		}
		if stat.Type == 0x48666853 { // HFS+
			return "hfs+"
		}
		if stat.Type == 0x7734 { // ZFS
			return "zfs"
		}
		return "unknown"
	}
}

func probeAtomicReplacePOSIX(rootPath string) bool {
	probeDir := filepath.Join(rootPath, ".hq-interface", "probe")
	if err := os.MkdirAll(probeDir, 0700); err != nil {
		return false
	}

	src := filepath.Join(probeDir, "probe_src.tmp")
	dst := filepath.Join(probeDir, "probe_dst.tmp")

	if err := os.WriteFile(src, []byte("probe"), 0600); err != nil {
		os.RemoveAll(probeDir)
		return false
	}

	if err := os.Rename(src, dst); err != nil {
		os.Remove(src)
		os.RemoveAll(probeDir)
		return false
	}

	os.Remove(dst)
	os.RemoveAll(probeDir)
	return true
}

func parseCapOverride(override, rootPath string) Capabilities {
	parts := strings.SplitN(override, ",", 2)
	fsType := parts[0]
	supportReplace := len(parts) < 2 || parts[1] != "no-atomic"
	return Capabilities{
		SupportAtomicReplace:        supportReplace,
		SupportFileLocking:          true,
		FilesystemType:              fsType,
		RootPath:                    rootPath,
		SupportOwnerOnlyPermissions: true,
	}
}
