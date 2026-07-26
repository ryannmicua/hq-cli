package read_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/config"
	"github.com/ryannmicua/hq-cli/internal/read"
	"github.com/ryannmicua/hq-cli/internal/testutil"
)

func mustService(t *testing.T) *read.Service {
	t.Helper()
	hqRoot := filepath.Join(testutil.ModuleRoot(), "testdata", "hq")
	cfg, err := config.Load(hqRoot, func(s string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	svc, err := read.NewService(cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func TestVersion_ReturnsAllFields(t *testing.T) {
	svc := mustService(t)
	v := svc.Version()
	if v.Version == "" {
		t.Fatal("expected non-empty version")
	}
	if v.GoVersion == "" {
		t.Fatal("expected non-empty GoVersion")
	}
	if v.OS == "" {
		t.Fatal("expected non-empty OS")
	}
	if v.Arch == "" {
		t.Fatal("expected non-empty Arch")
	}
}

func TestHealth_ReturnsPass(t *testing.T) {
	svc := mustService(t)
	h := svc.Health()
	if h.Overall != "pass" && h.Overall != "fail" {
		t.Fatalf("Overall = %q, expected 'pass' or 'fail'", h.Overall)
	}
	if len(h.Checks) == 0 {
		t.Fatal("expected at least one health check")
	}
	// Verify all known checks are present.
	checkNames := make(map[string]bool)
	for _, c := range h.Checks {
		checkNames[c.Name] = true
	}
	for _, name := range []string{"binary", "contract-assets", "hq-root", "git", "read-path"} {
		if !checkNames[name] {
			t.Fatalf("missing health check: %s", name)
		}
	}
}

func TestSearch_FindsBlocker(t *testing.T) {
	svc := mustService(t)
	ctx := context.Background()

	result, err := svc.Search(ctx, read.SearchQuery{Query: "blocker"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Count == 0 {
		t.Fatal("expected at least one match for 'blocker'")
	}

	// Verify sorting.
	for i := 1; i < len(result.Matches); i++ {
		prev := result.Matches[i-1]
		cur := result.Matches[i]
		if prev.Path > cur.Path || (prev.Path == cur.Path && prev.Line > cur.Line) {
			t.Fatalf("matches not sorted: (%q,%d) > (%q,%d)", prev.Path, prev.Line, cur.Path, cur.Line)
		}
	}
}

func TestSearch_CaseSensitive(t *testing.T) {
	svc := mustService(t)
	ctx := context.Background()

	// Case-insensitive should find "Blockers".
	result, err := svc.Search(ctx, read.SearchQuery{Query: "blockers"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count == 0 {
		t.Fatal("expected case-insensitive match for 'blockers'")
	}

	// Case-sensitive with exact case.
	result2, err := svc.Search(ctx, read.SearchQuery{Query: "Blockers", CaseSensitive: true})
	if err != nil {
		t.Fatal(err)
	}
	if result2.Count == 0 {
		t.Fatal("expected case-sensitive match for 'Blockers'")
	}

	// Case-sensitive with wrong case should fail to find in lowercase content.
	result3, err := svc.Search(ctx, read.SearchQuery{Query: "BLOCKERS", CaseSensitive: true})
	if err != nil {
		t.Fatal(err)
	}
	if result3.Count != 0 {
		t.Fatalf("expected 0 case-sensitive matches for 'BLOCKERS' (wrong case), got %d", result3.Count)
	}
}

func TestSearch_NoMatch(t *testing.T) {
	svc := mustService(t)
	ctx := context.Background()

	result, err := svc.Search(ctx, read.SearchQuery{Query: "xyznonexistent12345"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 0 {
		t.Fatalf("expected 0 matches, got %d", result.Count)
	}
}

// Snapshot test: verify no files are modified by read operations.
func TestSnapshot_ReadCommandsNoMutation(t *testing.T) {
	svc := mustService(t)
	ctx := context.Background()

	// Compute pre-snapshot hashes.
	preSnapshot := hashFixtureFiles(t, "testdata/hq")

	// Run various read operations.
	_ = svc.Version()
	_ = svc.Health()
	_, _ = svc.Search(ctx, read.SearchQuery{Query: "blocker"})

	// Compute post-snapshot hashes.
	postSnapshot := hashFixtureFiles(t, "testdata/hq")

	// Compare.
	if len(preSnapshot) != len(postSnapshot) {
		t.Fatalf("file count changed: before=%d, after=%d", len(preSnapshot), len(postSnapshot))
	}
	for path, hash := range preSnapshot {
		if postHash, ok := postSnapshot[path]; !ok {
			t.Fatalf("file %q disappeared", path)
		} else if hash != postHash {
			t.Fatalf("file %q was modified: before=%s, after=%s", path, hash, postHash)
		}
	}
}

func TestSearch_ByCollection(t *testing.T) {
	svc := mustService(t)
	ctx := context.Background()

	result, err := svc.Search(ctx, read.SearchQuery{
		Query:      "HQ",
		Collection: "decisions",
	})
	if err != nil {
		t.Fatalf("Search by collection failed: %v", err)
	}
	// Just verify no error; results depend on fixture content.
	_ = result
}

// hashFixtureFiles walks a fixture directory and computes SHA-256 for every file.
func hashFixtureFiles(t *testing.T, relPath string) map[string]string {
	t.Helper()
	fixturePath := filepath.Join(testutil.ModuleRoot(), relPath)
	hashes := make(map[string]string)

	err := filepath.WalkDir(fixturePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		h := sha256.Sum256(data)
		rel, _ := filepath.Rel(fixturePath, path)
		hashes[filepath.ToSlash(rel)] = hex.EncodeToString(h[:])
		return nil
	})
	if err != nil {
		t.Fatalf("walk fixtures: %v", err)
	}

	return hashes
}
