//go:build !windows

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
)

func staleTimeout() time.Time {
	d := 5 * time.Minute
	if s := os.Getenv("HQ_LOCK_STALE_TIMEOUT"); s != "" {
		if v, err := time.ParseDuration(s); err == nil && v > 0 {
			d = v
		}
	}
	return time.Now().Add(-d)
}

func (f *posixFS) Lock(ctx context.Context, target string, timeout time.Duration, exclusive bool) (UnlockFunc, error) {
	hqDir := filepath.Dir(filepath.Dir(target))
	lockDir := filepath.Join(hqDir, ".hq-interface", "locks")
	lockName := filepath.Base(target) + ".lock"
	lockPath := filepath.Join(lockDir, lockName)
	return f.LockFile(ctx, lockPath, timeout, exclusive)
}

func (f *posixFS) LockFile(ctx context.Context, lockPath string, timeout time.Duration, exclusive bool) (UnlockFunc, error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0700); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}

	hostname, _ := os.Hostname()
	pid := os.Getpid()
	now := time.Now()
	lockContent := fmt.Sprintf("%s|%d|%d\n", hostname, pid, now.UnixNano())

	owner, err := syscall.Open(lockPath, syscall.O_RDWR|syscall.O_CREAT|syscall.O_EXCL, 0600)
	if err == nil {
		syscall.Write(owner, []byte(lockContent))
		syscall.Close(owner)
	}

	fd, err := syscall.Open(lockPath, syscall.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	how := syscall.LOCK_EX
	if !exclusive {
		how = syscall.LOCK_SH
	}

	deadline := time.Now().Add(timeout)
	for {
		err := syscall.Flock(fd, how|syscall.LOCK_NB)
		if err == nil {
			break
		}

		if err != syscall.EWOULDBLOCK {
			syscall.Close(fd)
			return nil, fmt.Errorf("flock error: %v", err)
		}

		if timeout == 0 {
			syscall.Close(fd)
			return nil, fmt.Errorf("HQ_LOCK_TIMEOUT: could not acquire lock on %q", lockPath)
		}

		if checkStaleLock(lockPath) {
			os.Remove(lockPath)
			continue
		}

		if time.Now().After(deadline) {
			syscall.Close(fd)
			return nil, fmt.Errorf("HQ_LOCK_TIMEOUT: could not acquire lock on %q", lockPath)
		}

		select {
		case <-ctx.Done():
			syscall.Close(fd)
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}

	unlock := func() error {
		syscall.Flock(fd, syscall.LOCK_UN)
		syscall.Close(fd)
		return nil
	}

	return unlock, nil
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

	proc, err := os.FindProcess(pid)
	if err == nil {
		if err := proc.Signal(syscall.Signal(0)); err == nil {
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
