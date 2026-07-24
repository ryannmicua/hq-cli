package hq_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/hq"
)

func TestNewResolver_AcceptsExistingDir(t *testing.T) {
	dir := t.TempDir()
	r, err := hq.NewResolver(dir)
	if err != nil {
		t.Fatalf("NewResolver failed: %v", err)
	}
	if r.Root() == "" {
		t.Fatal("expected non-empty root")
	}
}

func TestNewResolver_RejectsMissingDir(t *testing.T) {
	_, err := hq.NewResolver("C:\\nonexistent-resolver-test-xyz")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestNewResolver_RejectsFileRoot(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := hq.NewResolver(f)
	if err == nil {
		t.Fatal("expected error for file root")
	}
}

func TestResolve_AcceptsPathWithinRoot(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}

	r, err := hq.NewResolver(dir)
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := r.Resolve("subdir")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if resolved != filepath.Clean(sub) {
		t.Fatalf("Resolve = %q, want %q", resolved, filepath.Clean(sub))
	}
}

func TestResolve_RejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	r, err := hq.NewResolver(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for traversal path")
	}
}

func TestResolve_RejectsAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	r, err := hq.NewResolver(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve("C:\\Windows\\System32")
	if err == nil {
		t.Fatal("expected error for absolute path")
	}
}

func TestResolve_RejectsEmptyPath(t *testing.T) {
	dir := t.TempDir()
	r, err := hq.NewResolver(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestResolve_AcceptsRootItself(t *testing.T) {
	dir := t.TempDir()
	r, err := hq.NewResolver(dir)
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := r.Resolve(".")
	if err != nil {
		t.Fatalf("Resolve root itself failed: %v", err)
	}
	if resolved != r.Root() {
		t.Fatalf("Resolve('.') = %q, want root %q", resolved, r.Root())
	}
}
