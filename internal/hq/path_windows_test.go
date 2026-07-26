//go:build windows

package hq_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/hq"
)

func TestResolve_Windows_DriveChange(t *testing.T) {
	dir := t.TempDir()
	r, err := hq.NewResolver(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve("C:\\Windows")
	if err == nil {
		t.Fatal("expected error for drive-relative absolute path; C:\\Windows is absolute")
	}
}

func TestResolve_Windows_ReservedNames(t *testing.T) {
	dir := t.TempDir()
	r, err := hq.NewResolver(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve("CON")
	if err != nil {
		t.Logf("Reserved name CON rejected (expected on some systems): %v", err)
	}
}

func TestResolve_Windows_AltDataStreamSyntax(t *testing.T) {
	dir := t.TempDir()
	r, err := hq.NewResolver(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve("some.md:stream")
	if err != nil {
		t.Logf("ADS path resolved with error (expected on some systems): %v", err)
	}
}

func TestResolve_Windows_TraversalBackslash(t *testing.T) {
	dir := t.TempDir()
	r, err := hq.NewResolver(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve("..\\..\\Windows")
	if err == nil {
		t.Fatal("expected error for backslash traversal")
	}
}

func TestResolve_Windows_Junction(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	junctionPath := filepath.Join(root, "escape-link")

	// Create junction using cmd /c mklink /J
	cmd := exec.Command("cmd", "/c", "mklink", "/J", junctionPath, outside)
	if err := cmd.Run(); err != nil {
		t.Skip("skipping junction test: system does not support mklink or permission denied")
	}

	r, err := hq.NewResolver(root)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Resolve("escape-link")
	if err == nil {
		t.Fatal("expected error for junction escaping root")
	}
}
