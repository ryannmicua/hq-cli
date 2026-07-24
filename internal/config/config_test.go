package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/config"
)

func TestLoad_FromRootFlag(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir, os.LookupEnv)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Root == "" {
		t.Fatal("expected non-empty Root")
	}
	abs, _ := filepath.Abs(dir)
	if cfg.Root != filepath.Clean(abs) {
		t.Fatalf("Root = %q, want %q", cfg.Root, filepath.Clean(abs))
	}
}

func TestLoad_RootFlagOverridesEnv(t *testing.T) {
	dirFlag := t.TempDir()
	dirEnv := t.TempDir()

	env := func(key string) (string, bool) {
		if key == "HQ_ROOT" {
			return dirEnv, true
		}
		return "", false
	}

	cfg, err := config.Load(dirFlag, env)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	absFlag, _ := filepath.Abs(dirFlag)
	if cfg.Root != filepath.Clean(absFlag) {
		t.Fatalf("Root = %q (should be from flag), want %q", cfg.Root, filepath.Clean(absFlag))
	}
}

func TestLoad_FromEnvVar(t *testing.T) {
	dir := t.TempDir()
	env := func(key string) (string, bool) {
		if key == "HQ_ROOT" {
			return dir, true
		}
		return "", false
	}

	cfg, err := config.Load("", env)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	abs, _ := filepath.Abs(dir)
	if cfg.Root != filepath.Clean(abs) {
		t.Fatalf("Root = %q, want %q", cfg.Root, filepath.Clean(abs))
	}
}

func TestLoad_FromCurrentDir(t *testing.T) {
	// Load from current dir when no flag and no env.
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir) //nolint:errcheck

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	env := func(key string) (string, bool) { return "", false }
	cfg, err := config.Load("", env)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	// Compare against os.Getwd (which config.Load uses internally),
	// not t.TempDir, because on Windows the temp dir may use 8.3
	// short names while os.Getwd returns the long form.
	wd, _ := os.Getwd()
	wd, _ = filepath.EvalSymlinks(wd)
	if cfg.Root != filepath.Clean(wd) {
		t.Fatalf("Root = %q, want %q", cfg.Root, filepath.Clean(wd))
	}
}

func TestLoad_MissingDir(t *testing.T) {
	env := func(key string) (string, bool) { return "", false }
	_, err := config.Load("C:\\nonexistent-hq-dir-12345", env)
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestLoad_FileNotDir(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "afile.txt")
	if err := os.WriteFile(tmpFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	env := func(key string) (string, bool) { return "", false }
	_, err := config.Load(tmpFile, env)
	if err == nil {
		t.Fatal("expected error for file root")
	}
}
