//go:build windows

package fsx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	modkernel32Lock                  = syscall.NewLazyDLL("kernel32.dll")
	procCreateFileW                  = modkernel32Lock.NewProc("CreateFileW")
	procLockFileEx                   = modkernel32Lock.NewProc("LockFileEx")
	procUnlockFileEx                 = modkernel32Lock.NewProc("UnlockFileEx")
	procOpenProcess                  = modkernel32Lock.NewProc("OpenProcess")
	procWaitForSingleObject          = modkernel32Lock.NewProc("WaitForSingleObject")
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

	processQueryLimitedInfo = 0x00001000
	errorAccessDenied       = 5
	errorInvalidParameter   = 87
	waitObject0 = 0
)

func (f *windowsFS) Lock(ctx context.Context, target string, timeout time.Duration, exclusive bool) (UnlockFunc, error) {
	hqDir := filepath.Dir(filepath.Dir(target))
	lockDir := filepath.Join(hqDir, ".hq-interface", "locks")
	lockName := filepath.Base(target) + ".lock"
	lockPath := filepath.Join(lockDir, lockName)
	return f.LockFile(ctx, lockPath, timeout, exclusive)
}

func (f *windowsFS) LockFile(ctx context.Context, lockPath string, timeout time.Duration, exclusive bool) (UnlockFunc, error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0700); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}

	hostname := resolveHostname()
	lockInfo := func() string {
		return fmt.Sprintf("%s|%d|%d\n", hostname, os.Getpid(), time.Now().UnixNano())
	}

	lockPathPtr, err := syscall.UTF16PtrFromString(lockPath)
	if err != nil {
		return nil, fmt.Errorf("convert lock path: %w", err)
	}

	deadline := time.Now().Add(timeout)
	attempt := 0
	for {
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
		if ret != 0 {
			os.WriteFile(lockPath, []byte(lockInfo()), 0600)

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

		syscall.CloseHandle(syscall.Handle(handle))

		if timeout == 0 {
			return nil, fmt.Errorf("HQ_LOCK_TIMEOUT: could not acquire lock on %q", lockPath)
		}

		if checkStaleLock(lockPath) {
			os.Remove(lockPath)
			continue
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("HQ_LOCK_TIMEOUT: could not acquire lock on %q", lockPath)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryDelay(attempt)):
			attempt++
		}
	}
}

func checkStaleLock(lockPath string) bool {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return false
	}

	parts := strings.SplitN(strings.TrimSpace(string(data)), "|", 3)
	if len(parts) < 3 {
		return false
	}

	pid, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	ret, _, sysErr := procOpenProcess.Call(
		uintptr(processQueryLimitedInfo),
		0,
		uintptr(pid),
	)
	if ret != 0 {
		status, _, _ := procWaitForSingleObject.Call(ret, 0)
		syscall.CloseHandle(syscall.Handle(ret))
		if status == waitObject0 {
			ts, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				return false
			}
			lockTime := time.Unix(0, ts)
			return lockTime.Before(staleTimeout())
		}
		return false
	}

	if sysErr != nil {
		if errno, ok := sysErr.(syscall.Errno); ok {
			if errno == errorAccessDenied {
				return false
			}
			if errno != errorInvalidParameter {
				return false
			}
		} else {
			return false
		}
	}

	ts, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return false
	}

	lockTime := time.Unix(0, ts)
	return lockTime.Before(staleTimeout())
}
