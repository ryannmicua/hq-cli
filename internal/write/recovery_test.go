package write_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/fsx"
	"github.com/ryannmicua/hq-cli/internal/write"
)

func TestRecoveryInspect_PendingRequest(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	policy := testReceiptPolicy(t)
	store := write.NewRequestStore(f, root, policy)
	receipts := write.NewReceiptStore(f, root)
	svc := write.NewRecoveryService(f, root, store, receipts, policy)

	targetDir := filepath.Join(root, "projects", "test")
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		t.Fatal(err)
	}

	targetContent := []byte("# Test State\n\n## Executive Summary\n\nTest.\n\n## Current Outcome\n\nTest.\n\n## Current State\n\nTest.\n\n## Next Action\n\nTest.\n\n## Open Decisions\n\n- None.\n\n## Blockers\n\n- None.\n\n## Material Risks\n\n- None.\n\n## Evidence\n\n- Src.\n")
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
		SchemaVersion: "1.0", RequestID: "018f0000-0000-7000-8000-000000000050",
		Caller: contract.Caller{Name: "test"}, Purpose: "test",
		Operation: "project-check-in", Target: "projects/test/STATE.md",
		Payload:            payload,
		ExpectedTargetHash: sha256hex(targetContent),
	}

	if _, err := store.Submit(req); err != nil {
		t.Fatal(err)
	}

	ins, err := svc.Inspect(context.Background(), req.RequestID)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if ins.RequestID != req.RequestID {
		t.Fatalf("RequestID = %q, want %q", ins.RequestID, req.RequestID)
	}
	if ins.State != "pending" {
		t.Fatalf("State = %q, want pending", ins.State)
	}
}

func TestRecoveryInspect_AppliedWithReceipt(t *testing.T) {
	root := t.TempDir()
	f := fsx.NewFS()
	policy := testReceiptPolicy(t)
	store := write.NewRequestStore(f, root, policy)
	receipts := write.NewReceiptStore(f, root)
	engine := write.NewTransactionEngine(f, root, store, receipts, policy)
	svc := write.NewRecoveryService(f, root, store, receipts, policy)

	targetDir := filepath.Join(root, "projects", "test")
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		t.Fatal(err)
	}

	targetContent := []byte("# Test State\n\n## Executive Summary\n\nInit.\n\n## Current Outcome\n\nInit.\n\n## Current State\n\nInit.\n\n## Next Action\n\nInit.\n\n## Open Decisions\n\n- None.\n\n## Blockers\n\n- None.\n\n## Material Risks\n\n- None.\n\n## Evidence\n\n- Src.\n")
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
		SchemaVersion: "1.0", RequestID: "018f0000-0000-7000-8000-000000000060",
		Caller: contract.Caller{Name: "test"}, Purpose: "test",
		Operation: "project-check-in", Target: "projects/test/STATE.md",
		Payload:            payload,
		ExpectedTargetHash: sha256hex(targetContent),
	}

	if _, err := store.Submit(req); err != nil {
		t.Fatal(err)
	}

	if _, err := engine.Apply(context.Background(), req.RequestID, "approval-recovery"); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	ins, err := svc.Inspect(context.Background(), req.RequestID)
	if err != nil {
		t.Fatalf("Inspect after apply: %v", err)
	}

	if ins.State != "applied" {
		t.Fatalf("State = %q, want applied", ins.State)
	}
	if !ins.HasReceipt {
		t.Fatal("expected HasReceipt after apply")
	}
	if ins.RecoveryAction != "no-action-needed: request is already applied with a receipt" {
		t.Fatalf("RecoveryAction = %q, want no-action-needed", ins.RecoveryAction)
	}
}
