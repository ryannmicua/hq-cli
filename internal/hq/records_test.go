package hq_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/hq"
)

func TestGet_ProjectByID(t *testing.T) {
	ctx := context.Background()
	resolver := mustResolver(t, "testdata/hq")

	rec, err := hq.Get(ctx, resolver, hq.GetSelector{Collection: "projects", ID: "example"})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if rec.Collection != "projects" {
		t.Fatalf("Collection = %q, want %q", rec.Collection, "projects")
	}
	if rec.ID != "example" {
		t.Fatalf("ID = %q, want %q", rec.ID, "example")
	}
	if !strings.Contains(rec.Path, "example") {
		t.Fatalf("Path = %q, should contain 'example'", rec.Path)
	}
	if rec.Content == "" {
		t.Fatal("expected non-empty content")
	}
	if rec.SHA256 == "" {
		t.Fatal("expected non-empty SHA256")
	}
}

func TestGet_NonExistentID(t *testing.T) {
	ctx := context.Background()
	resolver := mustResolver(t, "testdata/hq")

	_, err := hq.Get(ctx, resolver, hq.GetSelector{Collection: "projects", ID: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestGet_ByPath(t *testing.T) {
	ctx := context.Background()
	resolver := mustResolver(t, "testdata/hq")

	rec, err := hq.Get(ctx, resolver, hq.GetSelector{Path: "AGENTS.md"})
	if err != nil {
		t.Fatalf("Get by path failed: %v", err)
	}
	if rec.Path != "AGENTS.md" {
		t.Fatalf("Path = %q, want %q", rec.Path, "AGENTS.md")
	}
	if rec.Content == "" {
		t.Fatal("expected non-empty content")
	}
}

func TestList_Projects(t *testing.T) {
	ctx := context.Background()
	resolver := mustResolver(t, "testdata/hq")

	recs, err := hq.List(ctx, resolver, hq.CollectionQuery{Collection: "projects"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(recs) == 0 {
		t.Fatal("expected at least one project record")
	}

	// Example project should be there.
	found := false
	for _, r := range recs {
		if r.ID == "example" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected project 'example' in list")
	}

	// Verify deterministic ordering.
	for i := 1; i < len(recs); i++ {
		if recs[i-1].Path > recs[i].Path {
			t.Fatalf("records not sorted by path: %q > %q", recs[i-1].Path, recs[i].Path)
		}
	}
}

func TestList_CurrentWork(t *testing.T) {
	ctx := context.Background()
	resolver := mustResolver(t, "testdata/hq")

	recs, err := hq.List(ctx, resolver, hq.CollectionQuery{Collection: "current-work"})
	if err != nil {
		t.Fatalf("List current-work failed: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	if recs[0].ID != "current" {
		t.Fatalf("ID = %q, want 'current'", recs[0].ID)
	}
}

func TestList_CurrentWorkFilterSection(t *testing.T) {
	ctx := context.Background()
	resolver := mustResolver(t, "testdata/hq")

	recs, err := hq.List(ctx, resolver, hq.CollectionQuery{
		Collection: "current-work",
		Filter:     map[string]string{"section": "active"},
	})
	if err != nil {
		t.Fatalf("List filtered failed: %v", err)
	}
	if len(recs) == 0 {
		t.Fatal("expected at least one record with section=active")
	}
}

func TestList_Decisions(t *testing.T) {
	ctx := context.Background()
	resolver := mustResolver(t, "testdata/hq")

	recs, err := hq.List(ctx, resolver, hq.CollectionQuery{Collection: "decisions"})
	if err != nil {
		t.Fatalf("List decisions failed: %v", err)
	}
	if len(recs) == 0 {
		t.Fatal("expected at least one decision record")
	}
}

func TestSHA256_Deterministic(t *testing.T) {
	ctx := context.Background()
	resolver := mustResolver(t, "testdata/hq")

	rec1, _ := hq.Get(ctx, resolver, hq.GetSelector{Path: "AGENTS.md"})
	rec2, _ := hq.Get(ctx, resolver, hq.GetSelector{Path: "AGENTS.md"})
	if rec1.SHA256 != rec2.SHA256 {
		t.Fatal("SHA-256 should be deterministic for identical content")
	}
}

func TestGitCommit_NullForFixtures(t *testing.T) {
	ctx := context.Background()
	resolver := mustResolver(t, "testdata/hq")

	rec, err := hq.Get(ctx, resolver, hq.GetSelector{Path: "AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}
	// Git may or may not be available; just verify it doesn't panic.
	_ = rec.GitCommit
}

func TestList_UnknownCollection(t *testing.T) {
	ctx := context.Background()
	resolver := mustResolver(t, "testdata/hq")

	_, err := hq.List(ctx, resolver, hq.CollectionQuery{Collection: "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown collection")
	}
}

func TestGet_UnknownCollection(t *testing.T) {
	ctx := context.Background()
	resolver := mustResolver(t, "testdata/hq")

	_, err := hq.Get(ctx, resolver, hq.GetSelector{Collection: "unknown", ID: "x"})
	if err == nil {
		t.Fatal("expected error for unknown collection")
	}
}

func TestCollectionAllowlist_ExcludesHidden(t *testing.T) {
	if hq.CollectionAllowlist[".git"] {
		t.Fatal(".git should not be in collection allowlist")
	}
	if hq.CollectionAllowlist[".hq-interface"] {
		t.Fatal(".hq-interface should not be in collection allowlist")
	}
}

// mustResolver creates a Resolver from a repo-relative path, failing the
// test on error. It finds the module root by looking for go.mod.
func mustResolver(t *testing.T, relPath string) *hq.Resolver {
	t.Helper()

	// Start from the test working directory and walk up to find go.mod.
	moduleRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(moduleRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(moduleRoot)
		if parent == moduleRoot {
			t.Fatal("could not find module root (go.mod)")
		}
		moduleRoot = parent
	}

	abs := filepath.Join(moduleRoot, relPath)
	r, err := hq.NewResolver(abs)
	if err != nil {
		t.Fatalf("NewResolver(%q): %v", abs, err)
	}
	return r
}
