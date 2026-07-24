//go:build !windows

package hq_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/hq"
)

func TestResolve_POSIX_AbsolutePathInjection(t *testing.T) {
	dir := t.TempDir()
	r, err := hq.NewResolver(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Even if joined with root, an absolute path argument is rejected.
	_, err = r.Resolve("/etc/passwd")
	if err == nil {
		t.Fatal("expected error for absolute path injection")
	}
}

func TestResolve_POSIX_SymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(root, "escape-link")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skip("symlink creation not supported:", err)
	}

	r, err := hq.NewResolver(root)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve("escape-link")
	if err == nil {
		t.Fatal("expected error for symlink escaping root")
	}
}
