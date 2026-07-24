// Package hq provides safe path resolution and markdown collection
// adapters for HQ workspaces.
package hq

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolver canonicalizes an HQ root and resolves relative paths safely
// within that root. It rejects traversal, symlink escape, and platform
// path hazards.
type Resolver struct {
	root string // canonical root path (cleaned, absolute, symlink-evaluated)
}

// NewResolver creates a Resolver for the given root directory. It
// canonicalizes the root by resolving to an absolute path, evaluating
// symlinks, and cleaning the result. An error is returned if the root
// does not exist or resolves to a non-directory.
func NewResolver(root string) (*Resolver, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("abs root %q: %w", root, err)
	}
	abs = filepath.Clean(abs)

	evaled, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("eval symlinks for root %q: %w", abs, err)
	}
	evaled = filepath.Clean(evaled)

	info, err := os.Stat(evaled)
	if err != nil {
		return nil, fmt.Errorf("stat resolved root %q: %w", evaled, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root %q is not a directory", evaled)
	}

	return &Resolver{root: evaled}, nil
}

// Root returns the canonical root path.
func (r *Resolver) Root() string {
	return r.root
}

// Resolve resolves the given relative path within the HQ root. It
// verifies containment: the resolved absolute path must be within the
// canonical root after symlink evaluation. It rejects:
//   - empty paths
//   - paths with ".." segments that escape
//   - absolute paths (must be relative)
//   - paths whose final resolved location is outside root
//
// It returns the resolved absolute path on success.
func (r *Resolver) Resolve(relative string) (string, error) {
	if relative == "" {
		return "", fmt.Errorf("path is empty")
	}

	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("path %q is absolute; must be relative", relative)
	}

	// Join and clean.
	joined := filepath.Join(r.root, relative)
	joined = filepath.Clean(joined)

	// Check that joined starts with root + separator (or equals root).
	if !strings.HasPrefix(joined, r.root+string(filepath.Separator)) && joined != r.root {
		return "", fmt.Errorf("path %q escapes HQ root", relative)
	}

	// Check for pre-eval traversal attempts (e.g., ".." that would leave root).
	rel := filepath.ToSlash(relative)
	if strings.Contains(rel, "../") || strings.HasPrefix(rel, "../") || rel == ".." {
		return "", fmt.Errorf("path %q contains traversal", relative)
	}
	// Backslash form on Windows.
	if strings.Contains(relative, "..\\") || strings.HasPrefix(relative, "..\\") {
		return "", fmt.Errorf("path %q contains traversal", relative)
	}

	// Resolve symlinks in the final target and re-check containment.
	evaled, err := filepath.EvalSymlinks(joined)
	if err != nil {
		if os.IsNotExist(err) {
			// Target doesn't exist; the cleaned path is still acceptable
			// as long as it's within root.
			return joined, nil
		}
		return "", fmt.Errorf("eval symlinks for %q: %w", joined, err)
	}
	evaled = filepath.Clean(evaled)

	if !strings.HasPrefix(evaled, r.root+string(filepath.Separator)) && evaled != r.root {
		return "", fmt.Errorf("resolved path %q escapes HQ root via symlink", relative)
	}

	return evaled, nil
}
