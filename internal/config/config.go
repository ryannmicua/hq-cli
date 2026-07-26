// Package config resolves the HQ root and runtime configuration.
// Priority: --root flag > $HQ_ROOT > current working directory.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the resolved HQ runtime configuration.
type Config struct {
	Root string // Canonical path to the HQ root directory
}

// Load resolves the HQ root from the given flag value, HQ_ROOT environment
// variable, or the current working directory, in that order. It validates
// that the resolved path exists and is a directory.
//
// lookupEnv allows test injection; use os.LookupEnv in production.
func Load(rootFlag string, lookupEnv func(string) (string, bool)) (*Config, error) {
	root := rootFlag
	if root == "" {
		if v, ok := lookupEnv("HQ_ROOT"); ok && v != "" {
			root = v
		}
	}
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("determine root from current directory: %w", err)
		}
	}

	// Resolve to an absolute, clean path.
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root %q: %w", root, err)
	}
	abs = filepath.Clean(abs)

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("HQ root %q does not exist", abs)
		}
		return nil, fmt.Errorf("stat HQ root %q: %w", abs, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("HQ root %q is not a directory", abs)
	}

	return &Config{Root: abs}, nil
}
