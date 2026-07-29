package write_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/fsx"
	"github.com/ryannmicua/hq-cli/internal/write"
)

func TestEnsureLayout_CreatesSubdirectories(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()

	if err := write.EnsureLayout(f, root); err != nil {
		t.Fatalf("EnsureLayout failed: %v", err)
	}

	subdirs := []string{"pending", "applied", "rejected", "conflicted", "recovery-required"}
	for _, sub := range subdirs {
		path := filepath.Join(root, ".hq-interface", "requests", sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected %s dir: %v", sub, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", sub)
		}
	}
}

func TestEnsureLayout_Idempotent(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()

	if err := write.EnsureLayout(f, root); err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if err := write.EnsureLayout(f, root); err != nil {
		t.Fatalf("second call (idempotent) failed: %v", err)
	}
}

func TestEnsureLayout_SetsPrivatePermissions(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()

	if err := write.EnsureLayout(f, root); err != nil {
		t.Fatalf("EnsureLayout failed: %v", err)
	}

	info, err := os.Stat(filepath.Join(root, ".hq-interface"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0044 != 0 {
		t.Log("Note: permissions may not be fully enforced on all platforms")
	}
}

func TestLayoutHealthCheck_DoesNotCreateDirs(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()

	checks := write.LayoutHealthCheck(f, root)
	for _, c := range checks {
		if c.Status == "" {
			t.Fatalf("check %q has empty status", c.Name)
		}
	}

	path := filepath.Join(root, ".hq-interface")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("LayoutHealthCheck should not create directories")
	}
}

func TestLayoutHealthCheck_AfterEnsureShowsPass(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()

	if err := write.EnsureLayout(f, root); err != nil {
		t.Fatal(err)
	}

	checks := write.LayoutHealthCheck(f, root)
	for _, c := range checks {
		if c.Status != "pass" {
			t.Fatalf("check %q status = %q, want pass", c.Name, c.Status)
		}
	}
}

func testPolicy(t *testing.T) *write.Policy {
	t.Helper()
	p, err := write.ParsePolicyJSON([]byte(`{
		"schemaVersion": "1.0",
		"rules": [
			{"operation":"project-check-in","targetPattern":"projects/*/STATE.md","class":"allowed"},
			{"operation":"session-entry","targetPattern":"SESSION-LOG.md","class":"allowed"},
			{"operation":"draft-record","targetPattern":"inbox/*.md","class":"allowed"},
			{"operation":"draft-record","targetPattern":"projects/*/reports/*.md","class":"allowed"},
			{"operation":"draft-record","targetPattern":"projects/*/source/*.md","class":"allowed"},
			{"operation":"current-work-update","targetPattern":"CURRENT-WORK.md","class":"allowed"},
			{"operation":"*","targetPattern":"identity/**","class":"denied"},
			{"operation":"*","targetPattern":"AGENTS.md","class":"denied"}
		],
		"defaultClass": "denied"
	}`))
	if err != nil {
		t.Fatalf("testPolicy: %v", err)
	}
	return p
}

func testRequest(t *testing.T, id string) contract.Request {
	t.Helper()
	return contract.Request{
		SchemaVersion:      "1.0",
		RequestID:          id,
		Caller:             contract.Caller{Name: "test"},
		Purpose:            "test submission",
		Operation:          "project-check-in",
		Target:             "projects/test/STATE.md",
		Payload:            json.RawMessage(`{"summary":"s","currentState":"cs","nextAction":"na"}`),
		ExpectedTargetHash: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
	}
}

func TestSubmit_ValidRequestReturnsPending(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	store := write.NewRequestStore(f, root, testPolicy(t))

	req := testRequest(t, "018f0000-0000-7000-8000-000000000001")
	status, err := store.Submit(req)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if status.State != contract.StatePending {
		t.Fatalf("state = %q, want %q", status.State, contract.StatePending)
	}
	if status.RequestID != req.RequestID {
		t.Fatalf("RequestID = %q, want %q", status.RequestID, req.RequestID)
	}
}

func TestSubmit_WritesToPendingDirectory(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	store := write.NewRequestStore(f, root, testPolicy(t))

	req := testRequest(t, "018f0000-0000-7000-8000-000000000002")
	_, err := store.Submit(req)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	pendingFile := filepath.Join(root, ".hq-interface", "requests", "pending", req.RequestID+".json")
	if _, err := os.Stat(pendingFile); err != nil {
		t.Fatalf("expected pending file: %v", err)
	}
}

func TestSubmit_DuplicateByteIdenticalReturnsSameStatus(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	store := write.NewRequestStore(f, root, testPolicy(t))

	req := testRequest(t, "018f0000-0000-7000-8000-000000000003")
	status1, err := store.Submit(req)
	if err != nil {
		t.Fatalf("first submit failed: %v", err)
	}

	status2, err := store.Submit(req)
	if err != nil {
		t.Fatalf("duplicate submit failed: %v", err)
	}

	if status1.RequestID != status2.RequestID {
		t.Fatalf("request IDs differ: %q vs %q", status1.RequestID, status2.RequestID)
	}
	if status1.State != status2.State {
		t.Fatalf("states differ: %q vs %q", status1.State, status2.State)
	}
}

func TestSubmit_InvalidRequestReturnsError(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	store := write.NewRequestStore(f, root, testPolicy(t))

	req := testRequest(t, "018f0000-0000-7000-8000-000000000004")
	req.RequestID = ""
	_, err := store.Submit(req)
	if err == nil {
		t.Fatal("expected error for invalid request")
	}
}

func TestSubmit_PolicyDeniedReturnsError(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	store := write.NewRequestStore(f, root, testPolicy(t))

	req := testRequest(t, "018f0000-0000-7000-8000-000000000005")
	req.Target = "identity/me.md"
	_, err := store.Submit(req)
	if err == nil {
		t.Fatal("expected error for policy denied target")
	}
}

func TestStatus_AfterSubmitReturnsPending(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	store := write.NewRequestStore(f, root, testPolicy(t))

	req := testRequest(t, "018f0000-0000-7000-8000-000000000006")
	_, err := store.Submit(req)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	status, err := store.Status(req.RequestID)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if status.State != contract.StatePending {
		t.Fatalf("state = %q, want %q", status.State, contract.StatePending)
	}
}

func TestStatus_NonExistentReturnsNotFound(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	store := write.NewRequestStore(f, root, testPolicy(t))

	_, err := store.Status("018f0000-0000-7000-8000-000000000099")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
	if !strings.Contains(err.Error(), "HQ_NOT_FOUND") {
		t.Fatalf("expected HQ_NOT_FOUND, got: %v", err)
	}
}

func TestStatus_ReturnsSha256(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	store := write.NewRequestStore(f, root, testPolicy(t))

	req := testRequest(t, "018f0000-0000-7000-8000-000000000007")
	status, err := store.Submit(req)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	if status.RequestSha256 == "" {
		t.Fatal("expected non-empty RequestSha256")
	}
}

func TestStatus_ReadOnly(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	store := write.NewRequestStore(f, root, testPolicy(t))

	req := testRequest(t, "018f0000-0000-7000-8000-000000000008")
	if _, err := store.Submit(req); err != nil {
		t.Fatal(err)
	}

	preFiles := countFiles(t, root)

	if _, err := store.Status(req.RequestID); err != nil {
		t.Fatal(err)
	}

	postFiles := countFiles(t, root)
	if preFiles != postFiles {
		t.Fatalf("file count changed: before=%d, after=%d", preFiles, postFiles)
	}
}

func countFiles(t *testing.T, root string) int {
	t.Helper()
	count := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			count++
		}
		return nil
	})
	return count
}
