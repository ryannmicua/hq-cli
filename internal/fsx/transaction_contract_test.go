package fsx_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ryannmicua/hq-cli/internal/fsx"
)

func TestLock_ExclusiveAcquireAndRelease(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")

	if err := os.WriteFile(target, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}

	unlock, err := f.Lock(context.Background(), target, time.Second, true)
	if err != nil {
		t.Fatalf("Lock exclusive failed: %v", err)
	}

	if unlock == nil {
		t.Fatal("Lock returned nil unlock function")
	}

	if err := unlock(); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}
}

func TestLock_SharedAcquireAndRelease(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")

	if err := os.WriteFile(target, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}

	unlock, err := f.Lock(context.Background(), target, time.Second, false)
	if err != nil {
		t.Fatalf("Lock shared failed: %v", err)
	}

	if unlock == nil {
		t.Fatal("Lock returned nil unlock function")
	}

	if err := unlock(); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}
}

func TestLock_Timeout(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")

	if err := os.WriteFile(target, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}

	unlock1, err := f.Lock(context.Background(), target, time.Second, true)
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	defer unlock1()

	_, err = f.Lock(context.Background(), target, 0, true)
	if err == nil {
		t.Fatal("expected lock timeout/error for immediate exclusive on held lock")
	}
}

func TestBackup_CreatesWithHash(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")
	backup := filepath.Join(dir, "backup.bak")

	content := []byte("original content")
	if err := os.WriteFile(target, content, 0600); err != nil {
		t.Fatal(err)
	}

	hash, err := f.Backup(target, backup)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	if hash == "" {
		t.Fatal("Backup returned empty hash")
	}

	backupData, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}

	if string(backupData) != string(content) {
		t.Fatalf("backup content = %q, want %q", string(backupData), string(content))
	}
}

func TestBackup_HashMatchesContent(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")
	backup := filepath.Join(dir, "backup.bak")

	content := []byte("test content for hash")
	if err := os.WriteFile(target, content, 0600); err != nil {
		t.Fatal(err)
	}

	hash, err := f.Backup(target, backup)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	expectedHash := fmt.Sprintf("%x", sha256.Sum256(content))
	if hash != expectedHash {
		t.Fatalf("hash = %q, want %q", hash, expectedHash)
	}
}

func TestBackup_FileNotExist(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()
	target := filepath.Join(dir, "nonexistent.md")
	backup := filepath.Join(dir, "backup.bak")

	_, err := f.Backup(target, backup)
	if err == nil {
		t.Fatal("expected error for nonexistent target")
	}
}

func TestReplaceDurable_ReplacesContent(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")
	temp := filepath.Join(dir, "temp.md")

	original := []byte("original")
	replacement := []byte("replacement content")

	if err := os.WriteFile(target, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(temp, replacement, 0600); err != nil {
		t.Fatal(err)
	}

	if err := f.ReplaceDurable(temp, target); err != nil {
		t.Fatalf("ReplaceDurable failed: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target after replace: %v", err)
	}

	if string(data) != string(replacement) {
		t.Fatalf("target content = %q, want %q", string(data), string(replacement))
	}
}

func TestReplaceDurable_TempNotExist(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")
	temp := filepath.Join(dir, "nonexistent.md")

	if err := os.WriteFile(target, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}

	err := f.ReplaceDurable(temp, target)
	if err == nil {
		t.Fatal("expected error when temp does not exist")
	}
}

func TestCapabilities_ReturnsStruct(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()

	caps, err := f.Capabilities(dir)
	if err != nil {
		t.Fatalf("Capabilities failed: %v", err)
	}

	if caps.RootPath == "" {
		t.Fatal("expected non-empty RootPath")
	}
	if caps.FilesystemType == "" {
		t.Fatal("expected non-empty FilesystemType")
	}
}
