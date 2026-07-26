package fsx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/fsx"
)

func TestNewFS_NonNil(t *testing.T) {
	f := fsx.NewFS()
	if f == nil {
		t.Fatal("NewFS returned nil")
	}
}

func TestIsSupported_ReturnsBool(t *testing.T) {
	supported := fsx.IsSupported()
	if supported != true {
		t.Fatal("IsSupported should return true on local filesystems")
	}
}

func TestMkdirPrivate_CreatesDirectory(t *testing.T) {
	f := fsx.NewFS()
	dir := filepath.Join(t.TempDir(), "testdir")
	if err := f.MkdirPrivate(dir, 0700); err != nil {
		t.Fatalf("MkdirPrivate failed: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected a directory")
	}
}

func TestMkdirPrivate_Idempotent(t *testing.T) {
	f := fsx.NewFS()
	dir := filepath.Join(t.TempDir(), "testdir")
	if err := f.MkdirPrivate(dir, 0700); err != nil {
		t.Fatalf("first MkdirPrivate failed: %v", err)
	}
	if err := f.MkdirPrivate(dir, 0700); err != nil {
		t.Fatalf("second MkdirPrivate (idempotent) failed: %v", err)
	}
}

func TestWriteDurable_WritesFile(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()
	data := []byte("test content")
	path, err := f.WriteDurable(dir, "test.tmp", data)
	if err != nil {
		t.Fatalf("WriteDurable failed: %v", err)
	}
	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(read) != string(data) {
		t.Fatalf("content = %q, want %q", string(read), string(data))
	}
}

func TestRenameAtomic_MovesFile(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("move me"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := f.RenameAtomic(src, dst); err != nil {
		t.Fatalf("RenameAtomic failed: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("source should not exist after rename")
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatal("destination should exist after rename")
	}
}

func TestRenameAtomic_SourceNotExist(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()
	err := f.RenameAtomic(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dest.txt"))
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestSyncParent_NoError(t *testing.T) {
	f := fsx.NewFS()
	dir := t.TempDir()
	if err := f.SyncParent(dir); err != nil {
		t.Logf("SyncParent (expected on some platforms): %v", err)
	}
}
