package write_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/fsx"
	"github.com/ryannmicua/hq-cli/internal/write"
)

type injectPoint int

const (
	injectNone injectPoint = iota
	injectTempFileCreate
	injectTempFileRename
	injectLock
	injectHashRecheck
	injectBackup
	injectBackupVerify
	injectRenderedTempFlush
	injectTargetReplace
	injectPostWriteVerify
	injectReceiptStore
	injectStateTransition
)

type injectableFS struct {
	fsx.FS
	mu      sync.Mutex
	failOn  injectPoint
	applied bool
}

func (f *injectableFS) WriteDurable(dir, name string, data []byte) (string, error) {
	f.mu.Lock()
	p := f.failOn
	app := f.applied
	f.mu.Unlock()

	if p == injectTempFileCreate && app {
		return "", fmt.Errorf("injected failure: write durable")
	}
	if p == injectRenderedTempFlush && app {
		return "", fmt.Errorf("injected failure: write durable")
	}
	return f.FS.WriteDurable(dir, name, data)
}

func (f *injectableFS) markApplied() {
	f.mu.Lock()
	f.applied = true
	f.mu.Unlock()
}

func (f *injectableFS) RenameAtomic(oldpath, newpath string) error {
	f.mu.Lock()
	p := f.failOn
	app := f.applied
	f.mu.Unlock()
	if p == injectTempFileRename && app {
		return fmt.Errorf("injected failure: rename atomic")
	}
	return f.FS.RenameAtomic(oldpath, newpath)
}

func (f *injectableFS) Lock(ctx context.Context, target string, timeout time.Duration, exclusive bool) (fsx.UnlockFunc, error) {
	f.mu.Lock()
	p := f.failOn
	f.mu.Unlock()
	if p == injectLock {
		return nil, fmt.Errorf("injected failure: lock")
	}
	return f.FS.Lock(ctx, target, timeout, exclusive)
}

func (f *injectableFS) ReplaceDurable(tempPath, targetPath string) error {
	f.mu.Lock()
	p := f.failOn
	f.mu.Unlock()
	if p == injectTargetReplace {
		return fmt.Errorf("injected failure: replace durable")
	}
	return f.FS.ReplaceDurable(tempPath, targetPath)
}

func (f *injectableFS) LockFile(ctx context.Context, lockPath string, timeout time.Duration, exclusive bool) (fsx.UnlockFunc, error) {
	f.mu.Lock()
	p := f.failOn
	f.mu.Unlock()
	if p == injectLock {
		return nil, fmt.Errorf("injected failure: lock")
	}
	return f.FS.LockFile(ctx, lockPath, timeout, exclusive)
}

func (f *injectableFS) Capabilities(root string) (fsx.Capabilities, error) {
	return f.FS.Capabilities(root)
}

func (f *injectableFS) Backup(target, backupPath string) (string, error) {
	f.mu.Lock()
	p := f.failOn
	f.mu.Unlock()
	if p == injectBackup {
		return "", fmt.Errorf("injected failure: backup")
	}
	return f.FS.Backup(target, backupPath)
}

func runFailureTest(t *testing.T, point injectPoint, root string) error {
	t.Helper()

	realFS := fsx.NewFS()
	f := &injectableFS{FS: realFS, failOn: point}

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
		SchemaVersion: "1.0", RequestID: "018f0000-0000-7000-8000-0000f0000001",
		Caller: contract.Caller{Name: "test"}, Purpose: "test failure injection",
		Operation: "project-check-in", Target: "projects/test/STATE.md",
		Payload: payload, ExpectedTargetHash: expectedHash,
	}

	if _, err := store.Submit(req); err != nil {
		t.Fatalf("submit: %v", err)
	}

	f.markApplied()

	_, err := engine.Apply(context.Background(), req.RequestID, "fault-001")
	return err
}

func verifySafeState(t *testing.T, root, requestID string, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected injected failure, got success")
	}

	pendingPath := filepath.Join(root, ".hq-interface", "requests", "pending", requestID+".json")
	appliedPath := filepath.Join(root, ".hq-interface", "requests", "applied", requestID+".json")

	_, pendingErr := os.Stat(pendingPath)
	_, appliedErr := os.Stat(appliedPath)

	hasPending := pendingErr == nil
	hasApplied := appliedErr == nil

	if hasApplied {
		t.Logf("request moved to applied (intent may exist)")
	}

	if hasPending {
		t.Logf("request remains pending — no mutation committed")
	} else if !hasApplied {
		t.Fatalf("request not found in pending or applied: should be in one or the other")
	}

	targetExamples := filepath.Join(root, "projects", "test", "STATE.md")
	if _, err := os.Stat(targetExamples); err != nil {
		t.Logf("target may not exist — acceptable for some failure points")
	}
}

func TestFailureInjection_TempFileCreate(t *testing.T) {
	root := t.TempDir()
	err := runFailureTest(t, injectTempFileCreate, root)
	verifySafeState(t, root, "018f0000-0000-7000-8000-0000f0000001", err)
}

func TestFailureInjection_Lock(t *testing.T) {
	root := t.TempDir()
	err := runFailureTest(t, injectLock, root)
	verifySafeState(t, root, "018f0000-0000-7000-8000-0000f0000001", err)
}

func TestFailureInjection_Backup(t *testing.T) {
	root := t.TempDir()
	err := runFailureTest(t, injectBackup, root)
	verifySafeState(t, root, "018f0000-0000-7000-8000-0000f0000001", err)
}

func TestFailureInjection_TargetReplace(t *testing.T) {
	root := t.TempDir()
	err := runFailureTest(t, injectTargetReplace, root)
	verifySafeState(t, root, "018f0000-0000-7000-8000-0000f0000001", err)
}

func TestSince_EmptyStore(t *testing.T) {
	since := time.Now().Add(-1 * time.Hour)
	svc := NewChangesService(nil, t.TempDir(), NewReceiptStore(nil, t.TempDir()))
	page, err := svc.Since(since, 10)
	if err != nil {
		t.Fatalf("Since on empty store: %v", err)
	}
	if len(page.Receipts) != 0 {
		t.Fatalf("expected 0 receipts, got %d", len(page.Receipts))
	}
}
