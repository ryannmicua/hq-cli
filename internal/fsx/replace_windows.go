//go:build windows

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

var (
	modkernel32Cap             = syscall.NewLazyDLL("kernel32.dll")
	procGetVolumeInformationW  = modkernel32Cap.NewProc("GetVolumeInformationW")
	procGetFileInformationByEx = modkernel32Cap.NewProc("GetFileInformationByHandleEx")
	procCreateFileWCap         = modkernel32Cap.NewProc("CreateFileW")
	procGetFileType            = modkernel32Cap.NewProc("GetFileType")
	diskQuery                  = uint32(0x00070000)
)

const (
	fileTypeDisk    = 0x0001
	fileTypeUnknown = 0x0000
	maxPath         = 260
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

	if override := os.Getenv("HQ_FS_OVERRIDE"); override != "" {
		return parseCapOverride(override, rootPath), nil
	}

	fsType := detectFilesystemWindows(rootPath)
	supportReplace := probeAtomicReplace(rootPath)
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

func detectFilesystemWindows(rootPath string) string {
	rootPtr, err := syscall.UTF16PtrFromString(rootPath)
	if err != nil {
		return "unknown"
	}

	var nameBuf [maxPath]uint16
	var fsNameBuf [maxPath]uint16
	var serial, maxComp, flags uint32

	ret, _, _ := procGetVolumeInformationW.Call(
		uintptr(unsafe.Pointer(rootPtr)),
		uintptr(unsafe.Pointer(&nameBuf[0])),
		uintptr(maxPath),
		uintptr(unsafe.Pointer(&serial)),
		uintptr(unsafe.Pointer(&maxComp)),
		uintptr(unsafe.Pointer(&flags)),
		uintptr(unsafe.Pointer(&fsNameBuf[0])),
		uintptr(maxPath),
	)
	if ret == 0 {
		return "unknown"
	}

	fsName := syscall.UTF16ToString(fsNameBuf[:])
	return strings.ToLower(fsName)
}

func probeAtomicReplace(rootPath string) bool {
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

	sp, _ := syscall.UTF16PtrFromString(src)
	dp, _ := syscall.UTF16PtrFromString(dst)
	ret, _, _ := procMoveFileExW.Call(
		uintptr(unsafe.Pointer(sp)),
		uintptr(unsafe.Pointer(dp)),
		moveFileReplaceExisting|moveFileWriteThrough,
	)

	ok := ret != 0
	os.Remove(src)
	os.Remove(dst)
	os.RemoveAll(probeDir)
	return ok
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
