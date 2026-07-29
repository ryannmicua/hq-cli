//go:build windows

package fsx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

var (
	modkernel32Lock                  = syscall.NewLazyDLL("kernel32.dll")
	procCreateFileW                  = modkernel32Lock.NewProc("CreateFileW")
	procLockFileEx                   = modkernel32Lock.NewProc("LockFileEx")
	procUnlockFileEx                 = modkernel32Lock.NewProc("UnlockFileEx")
	procGetFileInformationByHandleEx = modkernel32Lock.NewProc("GetFileInformationByHandleEx")
)

const (
	lockFileFlags     = 0x00000000
	lockExclusive     = 0x00000002
	lockShared        = 0x00000000
	lockFailImmediate = 0x00000001

	fileShareRead   = 0x00000001
	fileShareWrite  = 0x00000002
	fileShareDelete = 0x00000004
	openExisting    = 3
	openAlways      = 4
	fileAttrNormal  = 0x00000080
	invalidHandle   = ^uintptr(0)

	fileNameInfo = 0x00000002
)

func (f *windowsFS) Lock(ctx context.Context, target string, timeout time.Duration, exclusive bool) (UnlockFunc, error) {
	hqDir := filepath.Dir(filepath.Dir(target))
	lockDir := filepath.Join(hqDir, ".hq-interface", "locks")
	if err := os.MkdirAll(lockDir, 0700); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}

	lockName := filepath.Base(target) + ".lock"
	lockPath := filepath.Join(lockDir, lockName)

	hostname, _ := os.Hostname()
	lockInfo := fmt.Sprintf("%s|%d|%d", hostname, os.Getpid(), time.Now().UnixNano())

	if err := os.WriteFile(lockPath, []byte(lockInfo), 0600); err != nil {
		return nil, fmt.Errorf("write lock file: %w", err)
	}

	lockPathPtr, err := syscall.UTF16PtrFromString(lockPath)
	if err != nil {
		return nil, fmt.Errorf("convert lock path: %w", err)
	}

	handle, _, err := procCreateFileW.Call(
		uintptr(unsafe.Pointer(lockPathPtr)),
		uintptr(0x001001FF),
		fileShareRead|fileShareWrite|fileShareDelete,
		0,
		openAlways,
		fileAttrNormal,
		0,
	)
	if handle == invalidHandle {
		return nil, fmt.Errorf("CreateFileW lock %q: %v", lockPath, err)
	}

	lockMode := uint32(lockExclusive)
	if !exclusive {
		lockMode = lockShared
	}

	flags := uint32(0)
	if timeout == 0 {
		flags = lockFailImmediate
	}

	var overlapped syscall.Overlapped
	ret, _, err := procLockFileEx.Call(
		handle,
		uintptr(lockMode|flags),
		0,
		0x7FFFFFFF,
		0,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if ret == 0 {
		syscall.CloseHandle(syscall.Handle(handle))
		if timeout == 0 {
			return nil, fmt.Errorf("HQ_LOCK_TIMEOUT: could not acquire lock on %q", target)
		}
		return nil, fmt.Errorf("HQ_LOCK_TIMEOUT: lock acquisition failed on %q: %v", target, err)
	}

	unlock := func() error {
		var ov syscall.Overlapped
		ret, _, err := procUnlockFileEx.Call(
			handle,
			0,
			0x7FFFFFFF,
			0,
			uintptr(unsafe.Pointer(&ov)),
		)
		syscall.CloseHandle(syscall.Handle(handle))
		if ret == 0 {
			return fmt.Errorf("unlock file: %v", err)
		}
		return nil
	}

	return unlock, nil
}
