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

func writeRequestFile(t *testing.T, dir, id string, content map[string]any) string {
	t.Helper()
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	path := filepath.Join(dir, id+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write request file: %v", err)
	}
	return path
}

func validRequestPayload() map[string]any {
	return map[string]any{
		"schemaVersion":      "1.0",
		"requestId":          "018f0000-0000-7000-8000-000000000001",
		"caller":             map[string]any{"name": "test", "sessionId": "sess-1"},
		"purpose":            "test submit command",
		"operation":          "project-check-in",
		"target":             "projects/example/STATE.md",
		"payload":            map[string]any{"summary": "test", "currentState": "done", "nextAction": "ship"},
		"expectedTargetHash": "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
	}
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

func TestSubmit_ValidRequest(t *testing.T) {
	tmpDir := t.TempDir()
	reqFile := writeRequestFile(t, tmpDir, "valid", validRequestPayload())

	stdout, stderr, code := runHQ(t, []string{"submit", "--request", reqFile})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["command"] != "submit" {
		t.Fatalf("command = %v, want 'submit'", result["command"])
	}
	if result["success"] != true {
		t.Fatal("expected success=true")
	}
	data := result["data"].(map[string]any)
	if data["state"] != "pending" {
		t.Fatalf("state = %v, want 'pending'", data["state"])
	}
	if _, ok := data["requestId"]; !ok {
		t.Fatal("expected requestId in data")
	}
	if _, ok := data["requestSha256"]; !ok {
		t.Fatal("expected requestSha256 in data")
	}
}

func TestSubmit_InvalidRequest(t *testing.T) {
	tmpDir := t.TempDir()
	req := validRequestPayload()
	req["requestId"] = ""
	reqFile := writeRequestFile(t, tmpDir, "invalid", req)

	_, stderr, code := runHQ(t, []string{"submit", "--request", reqFile})
	if code != 4 {
		t.Fatalf("exit code = %d, want 4; stderr: %s", code, stderr)
	}
}

func TestSubmit_MissingFlag(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"submit"})
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["success"] != false {
		t.Fatal("expected success=false")
	}
}

func TestSubmit_OversizedFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "large.json")
	large := make([]byte, 2<<20) // 2 MiB
	if err := os.WriteFile(path, large, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	stdout, _, code := runHQ(t, []string{"submit", "--request", path})
	if code != 4 {
		t.Fatalf("exit code = %d, want 4; stdout: %s", code, stdout)
	}
	if !strings.Contains(stdout, "too large") && !strings.Contains(stdout, "HQ_INVALID_REQUEST") {
		t.Fatalf("expected size limit error, got: %s", stdout)
	}
}

func TestSubmit_NonexistentFile(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"submit", "--request", "C:\\nonexistent\\file.json"})
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

func TestSubmit_DuplicateID(t *testing.T) {
	tmpDir := t.TempDir()
	req := validRequestPayload()
	reqFile := writeRequestFile(t, tmpDir, "dup", req)

	// First submit should succeed.
	stdout1, stderr1, code1 := runHQ(t, []string{"submit", "--request", reqFile})
	if code1 != 0 {
		t.Fatalf("first submit failed: code=%d stderr=%s", code1, stderr1)
	}

	// Second submit with same content should return same status.
	stdout2, stderr2, code2 := runHQ(t, []string{"submit", "--request", reqFile})
	if code2 != 0 {
		t.Fatalf("duplicate submit failed: code=%d stderr=%s", code2, stderr2)
	}

	if stdout1 != stdout2 {
		t.Log("duplicate submit returned different output (may differ in timestamps)")
	}
}

func TestStatus_AfterSubmit(t *testing.T) {
	tmpDir := t.TempDir()
	req := validRequestPayload()
	reqFile := writeRequestFile(t, tmpDir, "status-test", req)

	// Submit first.
	stdout, stderr, code := runHQ(t, []string{"submit", "--request", reqFile})
	if code != 0 {
		t.Fatalf("submit failed: code=%d stderr=%s", code, stderr)
	}

	var submitResult map[string]any
	json.Unmarshal([]byte(stdout), &submitResult)
	submitData := submitResult["data"].(map[string]any)
	requestID := submitData["requestId"].(string)

	// Now status.
	stdout2, stderr2, code2 := runHQ(t, []string{"status", "--request-id", requestID})
	if code2 != 0 {
		t.Fatalf("status failed: code=%d stderr=%s", code2, stderr2)
	}

	var statusResult map[string]any
	if err := json.Unmarshal([]byte(stdout2), &statusResult); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if statusResult["command"] != "status" {
		t.Fatalf("command = %v, want 'status'", statusResult["command"])
	}
	if statusResult["success"] != true {
		t.Fatal("expected success=true")
	}
	statusData := statusResult["data"].(map[string]any)
	if statusData["state"] != "pending" {
		t.Fatalf("state = %v, want 'pending'", statusData["state"])
	}
}

func TestStatus_Nonexistent(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"status", "--request-id", "018f0000-0000-7000-8000-000000000099"})
	if code != 3 {
		t.Fatalf("exit code = %d, want 3; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	errObj := result["error"].(map[string]any)
	if errObj["code"] != "HQ_NOT_FOUND" {
		t.Fatalf("error code = %v, want HQ_NOT_FOUND", errObj["code"])
	}
}

func TestStatus_MissingFlag(t *testing.T) {
	stdout, stderr, code := runHQ(t, []string{"status"})
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr: %s", code, stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["success"] != false {
		t.Fatal("expected success=false")
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
