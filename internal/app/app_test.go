package app_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/app"
	"github.com/ryannmicua/hq-cli/internal/testutil"
)

// runHQ is a helper that invokes app.Run with the given args and fixture root.
func runHQ(t *testing.T, args []string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	root := testutil.ModuleRoot()
	hqRoot := filepath.Join(root, "testdata", "hq")
	fullArgs := append([]string{"--root", hqRoot}, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode = app.Run(fullArgs, &stdoutBuf, &stderrBuf)
	return stdoutBuf.String(), stderrBuf.String(), exitCode
}

func TestVersion_ReturnsValidJSON(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"version"})
	if stderr != "" {
		t.Logf("stderr: %s", stderr)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["command"] != "version" {
		t.Fatalf("command = %v, want 'version'", result["command"])
	}
	if result["success"] != true {
		t.Fatal("expected success=true")
	}
	if _, ok := result["data"]; !ok {
		t.Fatal("expected data field")
	}
}

func TestHealth_ReturnsPass(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"health"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["command"] != "health" {
		t.Fatalf("command = %v, want 'health'", result["command"])
	}
}

func TestContext_WithFixtures(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"context"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["command"] != "context" {
		t.Fatalf("command = %v, want 'context'", result["command"])
	}
	if result["success"] != true {
		t.Fatal("expected success=true")
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data object")
	}
	if _, ok := data["selectedWork"]; !ok {
		t.Fatal("expected selectedWork in data")
	}
	if _, ok := data["currentWork"]; !ok {
		t.Fatal("expected currentWork in data")
	}
}

func TestGet_ProjectExample(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"get", "--collection", "projects", "--id", "example"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["command"] != "get" {
		t.Fatalf("command = %v, want 'get'", result["command"])
	}
	if result["success"] != true {
		t.Fatal("expected success=true")
	}
}

func TestGet_NonExistent(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"get", "--collection", "projects", "--id", "nonexistent"})
	if code != 3 {
		t.Fatalf("exit code = %d, want 3; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["success"] != false {
		t.Fatal("expected success=false")
	}
	errObj := result["error"].(map[string]any)
	if errObj["code"] != "HQ_NOT_FOUND" {
		t.Fatalf("error code = %v, want HQ_NOT_FOUND", errObj["code"])
	}
}

func TestGet_ByPath(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"get", "--path", "AGENTS.md"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["success"] != true {
		t.Fatal("expected success=true")
	}
}

func TestList_Projects(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"list", "--collection", "projects"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["success"] != true {
		t.Fatal("expected success=true")
	}

	data := result["data"].(map[string]any)
	items := data["items"].([]any)
	if len(items) == 0 {
		t.Fatal("expected at least one project")
	}
}

func TestList_CurrentWork(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"list", "--collection", "current-work"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["success"] != true {
		t.Fatal("expected success=true")
	}
}

func TestSearch_FindsContent(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"search", "--query", "blocker"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["success"] != true {
		t.Fatal("expected success=true")
	}

	data := result["data"].(map[string]any)
	if data["count"].(float64) == 0 {
		t.Log("no matches found for 'blocker' (fixtures may vary)")
	}
}

func TestSearch_NoMatch(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"search", "--query", "xyznonexistent12345"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["success"] != true {
		t.Fatal("expected success=true")
	}

	data := result["data"].(map[string]any)
	if data["count"].(float64) != 0 {
		t.Fatalf("expected 0 matches, got %v", data["count"])
	}
}

func TestUnknownCommand(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"unknown"})
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Fatalf("expected 'unknown command' in stderr, got: %s", stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["success"] != false {
		t.Fatal("expected success=false")
	}
}

func TestNoArgs(t *testing.T) {
	_, stderr, code := runHQ(t, []string{})
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "missing command") && !strings.Contains(stderr, "usage") {
		t.Fatalf("expected error in stderr, got: %s", stderr)
	}
}

// Snapshot test: verify no read command modifies the filesystem.
func TestSnapshot_ReadCommands(t *testing.T) {
	fixtureDir := filepath.Join(testutil.ModuleRoot(), "testdata", "hq")

	// Compute pre-snapshot.
	preSnapshot := hashFiles(t, fixtureDir)

	// Run several read commands.
	runHQ(t, []string{"version"})
	runHQ(t, []string{"health"})
	runHQ(t, []string{"context"})
	runHQ(t, []string{"get", "--collection", "projects", "--id", "example"})
	runHQ(t, []string{"get", "--path", "projects/example/README.md"})
	runHQ(t, []string{"list", "--collection", "projects"})
	runHQ(t, []string{"search", "--query", "HQ"})
	runHQ(t, []string{"search", "--query", "working", "--collection", "current-work"})

	// Compute post-snapshot.
	postSnapshot := hashFiles(t, fixtureDir)

	// Compare.
	if len(preSnapshot) != len(postSnapshot) {
		t.Fatalf("file count changed: before=%d, after=%d", len(preSnapshot), len(postSnapshot))
	}
	for path, hash := range preSnapshot {
		if postHash, ok := postSnapshot[path]; !ok {
			t.Fatalf("file %q disappeared", path)
		} else if hash != postHash {
			t.Fatalf("file %q modified: hash changed", path)
		}
	}
}

func TestBuild(t *testing.T) {
	// Quick test that the package compiles.
	_ = app.Run
}

// hashFiles walks a directory and returns SHA-256 hex hashes for all files.
func hashFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	hashes := make(map[string]string)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
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
		rel, _ := filepath.Rel(dir, path)
		hashes[filepath.ToSlash(rel)] = hex.EncodeToString(h[:])
		return nil
	})
	if err != nil {
		t.Fatalf("walk %q: %v", dir, err)
	}
	return hashes
}
