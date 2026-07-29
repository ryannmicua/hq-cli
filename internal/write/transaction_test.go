package write_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/fsx"
	"github.com/ryannmicua/hq-cli/internal/write"
)

func sha256hex(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func testReceiptPolicy(t *testing.T) *write.Policy {
	t.Helper()
	return testPolicy(t)
}

func TestTransactionEngine_ApplyProjectCheckIn(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	policy := testReceiptPolicy(t)
	store := write.NewRequestStore(f, root, policy)
	receipts := write.NewReceiptStore(f, root)
	engine := write.NewTransactionEngine(f, root, store, receipts, policy)

	targetDir := filepath.Join(root, "projects", "test")
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		t.Fatal(err)
	}

	targetContent := []byte("# Test Project State\n\n## Executive Summary\n\nInitial.\n\n## Current Outcome\n\nInitial.\n\n## Current State\n\nInitial.\n\n## Next Action\n\nInitial.\n\n## Open Decisions\n\n- None.\n\n## Blockers\n\n- None.\n\n## Material Risks\n\n- None.\n\n## Evidence\n\n- Initial source, verified 2026-07-24.\n")
	targetPath := filepath.Join(root, "projects", "test", "STATE.md")
	if err := os.WriteFile(targetPath, targetContent, 0600); err != nil {
		t.Fatal(err)
	}

	expectedHash := sha256hex(targetContent)

	payload, _ := json.Marshal(contract.ProjectCheckIn{
		Summary:        "Applied summary",
		CurrentOutcome: "Applied outcome",
		CurrentState:   "Applied state",
		NextAction:     "Applied action",
		Blockers:       []string{"None"},
		Risks:          []string{"None"},
		Evidence:       []string{"Applied evidence, verified 2026-07-29"},
		VerifiedAt:     "2026-07-29",
	})

	req := contract.Request{
		SchemaVersion:      "1.0",
		RequestID:          "018f0000-0000-7000-8000-000000000010",
		Caller:             contract.Caller{Name: "test"},
		Purpose:            "test apply",
		Operation:          "project-check-in",
		Target:             "projects/test/STATE.md",
		Payload:            payload,
		ExpectedTargetHash: expectedHash,
	}

	status, err := store.Submit(req)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if status.State != contract.StatePending {
		t.Fatalf("state = %q, want pending", status.State)
	}

	receipt, err := engine.Apply(context.Background(), req.RequestID, "approval-001")
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if receipt.RequestID != req.RequestID {
		t.Fatalf("receipt.RequestID = %q, want %q", receipt.RequestID, req.RequestID)
	}
	if receipt.Cursor == 0 {
		t.Fatal("receipt.Cursor should be non-zero")
	}
	if receipt.TargetSha256 == "" {
		t.Fatal("receipt.TargetSha256 should be non-empty")
	}
}

func TestTransactionEngine_AlreadyAppliedIdempotent(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	policy := testReceiptPolicy(t)
	store := write.NewRequestStore(f, root, policy)
	receipts := write.NewReceiptStore(f, root)
	engine := write.NewTransactionEngine(f, root, store, receipts, policy)

	targetDir := filepath.Join(root, "projects", "test")
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		t.Fatal(err)
	}

	targetContent := []byte("# Test\n\n## Executive Summary\n\nInit.\n\n## Current Outcome\n\nInit.\n\n## Current State\n\nInit.\n\n## Next Action\n\nInit.\n\n## Open Decisions\n\n- None.\n\n## Blockers\n\n- None.\n\n## Material Risks\n\n- None.\n\n## Evidence\n\n- Src.\n")
	targetPath := filepath.Join(root, "projects", "test", "STATE.md")
	if err := os.WriteFile(targetPath, targetContent, 0600); err != nil {
		t.Fatal(err)
	}

	expectedHash := sha256hex(targetContent)

	payload, _ := json.Marshal(contract.ProjectCheckIn{
		Summary: "S", CurrentOutcome: "O", CurrentState: "S",
		NextAction: "N", Blockers: []string{"None"}, Risks: []string{"None"},
		Evidence: []string{"E"}, VerifiedAt: "2026-07-29",
	})

	req := contract.Request{
		SchemaVersion: "1.0", RequestID: "018f0000-0000-7000-8000-000000000020",
		Caller: contract.Caller{Name: "test"}, Purpose: "test",
		Operation: "project-check-in", Target: "projects/test/STATE.md",
		Payload: payload, ExpectedTargetHash: expectedHash,
	}

	if _, err := store.Submit(req); err != nil {
		t.Fatal(err)
	}

	if _, err := engine.Apply(context.Background(), req.RequestID, "approval-002"); err != nil {
		t.Fatalf("first apply: %v", err)
	}

	_, err := engine.Apply(context.Background(), req.RequestID, "approval-002")
	if err == nil {
		t.Fatal("expected error for already-applied request")
	}
}

func TestTransactionEngine_CreateOnlyDraftRecord(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	policy := testReceiptPolicy(t)
	store := write.NewRequestStore(f, root, policy)
	receipts := write.NewReceiptStore(f, root)
	engine := write.NewTransactionEngine(f, root, store, receipts, policy)

	payload, _ := json.Marshal(contract.DraftRecord{
		Title: "Test Draft", Body: "Body content",
		RecordDate: "2026-07-29", Classification: "inbox",
	})

	req := contract.Request{
		SchemaVersion: "1.0", RequestID: "018f0000-0000-7000-8000-000000000030",
		Caller: contract.Caller{Name: "test"}, Purpose: "test",
		Operation: "draft-record", Target: "inbox/2026-07-29-test-draft.md",
		Payload: payload, CreateOnly: true,
	}

	if _, err := store.Submit(req); err != nil {
		t.Fatal(err)
	}

	receipt, err := engine.Apply(context.Background(), req.RequestID, "approval-003")
	if err != nil {
		t.Fatalf("Apply draft-record: %v", err)
	}

	if receipt.Cursor == 0 {
		t.Fatal("receipt.Cursor should be non-zero")
	}

	draftPath := filepath.Join(root, "inbox", "2026-07-29-test-draft.md")
	if _, err := os.Stat(draftPath); err != nil {
		t.Fatalf("draft file not created: %v", err)
	}
}

func TestTransactionEngine_LockTimeout(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	policy := testReceiptPolicy(t)
	store := write.NewRequestStore(f, root, policy)
	receipts := write.NewReceiptStore(f, root)
	engine := write.NewTransactionEngine(f, root, store, receipts, policy)

	targetDir := filepath.Join(root, "projects", "test")
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		t.Fatal(err)
	}

	targetContent := []byte("# Test\n\n## Executive Summary\n\nInit.\n\n## Current Outcome\n\nInit.\n\n## Current State\n\nInit.\n\n## Next Action\n\nInit.\n\n## Open Decisions\n\n- None.\n\n## Blockers\n\n- None.\n\n## Material Risks\n\n- None.\n\n## Evidence\n\n- Src.\n")
	targetPath := filepath.Join(root, "projects", "test", "STATE.md")
	if err := os.WriteFile(targetPath, targetContent, 0600); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(contract.ProjectCheckIn{
		Summary: "S", CurrentOutcome: "O", CurrentState: "S",
		NextAction: "N", Blockers: []string{"None"}, Risks: []string{"None"},
		Evidence: []string{"E"}, VerifiedAt: "2026-07-29",
	})

	req := contract.Request{
		SchemaVersion: "1.0", RequestID: "018f0000-0000-7000-8000-000000000040",
		Caller: contract.Caller{Name: "test"}, Purpose: "test",
		Operation: "project-check-in", Target: "projects/test/STATE.md",
		Payload: payload,
		ExpectedTargetHash: sha256hex(targetContent),
	}

	if _, err := store.Submit(req); err != nil {
		t.Fatal(err)
	}

	unlock, err := f.Lock(context.Background(), targetPath, 0, true)
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	defer unlock()

	_, err = engine.Apply(context.Background(), req.RequestID, "approval-004")
	if err == nil {
		t.Fatal("expected error when lock is held")
	}
}
