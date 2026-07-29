package write

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/fsx"
)

type TransactionEngine struct {
	fsys     fsx.FS
	root     string
	store    *RequestStore
	receipts *ReceiptStore
	policy   *Policy
}

func NewTransactionEngine(fsys fsx.FS, root string, store *RequestStore, receipts *ReceiptStore, policy *Policy) *TransactionEngine {
	return &TransactionEngine{
		fsys:     fsys,
		root:     root,
		store:    store,
		receipts: receipts,
		policy:   policy,
	}
}

func (te *TransactionEngine) Apply(ctx context.Context, requestID string, approvalReference string) (contract.Receipt, error) {
	if err := contract.ValidateRequestID(requestID); err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_INVALID_REQUEST: %w", err)
	}

	status, err := te.store.Status(requestID)
	if err != nil {
		return contract.Receipt{}, err
	}

	if status.State != contract.StatePending {
		return contract.Receipt{}, fmt.Errorf("HQ_VERSION_CONFLICT: request %q is in state %q, not pending", requestID, status.State)
	}

	reqPath := filepath.Join(te.root, ".hq-interface", "requests", "pending", requestID+".json")
	reqData, err := os.ReadFile(reqPath)
	if err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: read request: %w", err)
	}

	req, err := contract.UnmarshalRequest(reqData)
	if err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_INTERNAL_ERROR: unmarshal stored request: %w", err)
	}

	canonicalHash, err := contract.CanonicalRequestHash(req)
	if err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_INTERNAL_ERROR: compute hash: %w", err)
	}

	if canonicalHash != status.RequestSha256 {
		return contract.Receipt{}, fmt.Errorf("HQ_VERSION_CONFLICT: request %q has been modified since submission", requestID)
	}

	if te.policy != nil {
		class, err := te.policy.Classify(req.Operation, req.Target)
		if err != nil {
			return contract.Receipt{}, fmt.Errorf("HQ_INTERNAL_ERROR: policy classify: %w", err)
		}
		if class == PolicySubmitOnly {
			return contract.Receipt{}, fmt.Errorf("HQ_POLICY_DENIED: operation %q target %q is submit-only, cannot be applied", req.Operation, req.Target)
		}
		if class == PolicyApprovalRequired || class == PolicyDenied {
			if class == PolicyDenied {
				return contract.Receipt{}, fmt.Errorf("HQ_POLICY_DENIED: operation %q target %q is denied by policy", req.Operation, req.Target)
			}
			if approvalReference == "" {
				return contract.Receipt{}, fmt.Errorf("HQ_APPROVAL_REQUIRED: operation %q on target %q requires an approval reference", req.Operation, req.Target)
			}
		}
	}

	var renderer Renderer
	switch req.Operation {
	case "project-check-in":
		renderer = NewProjectCheckInRenderer()
	case "session-entry":
		renderer = NewSessionEntryRenderer()
	case "draft-record":
		renderer = NewDraftRecordRenderer()
	case "current-work-update":
		renderer = NewCurrentWorkUpdateRenderer()
	default:
		return contract.Receipt{}, fmt.Errorf("HQ_INTERNAL_ERROR: unknown operation %q", req.Operation)
	}

	targetPath := filepath.Join(te.root, req.Target)

	if err := os.MkdirAll(filepath.Dir(targetPath), 0700); err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: create target dir: %w", err)
	}

	unlock, err := te.fsys.Lock(ctx, targetPath, 0, true)
	if err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_LOCK_TIMEOUT: cannot acquire lock on %q: %w", req.Target, err)
	}
	defer unlock()

	current, err := os.ReadFile(targetPath)
	if err != nil && !os.IsNotExist(err) {
		return contract.Receipt{}, fmt.Errorf("HQ_INTERNAL_ERROR: read target: %w", err)
	}
	if os.IsNotExist(err) {
		current = nil
	}

	if !req.CreateOnly {
		if current == nil {
			return contract.Receipt{}, fmt.Errorf("HQ_NOT_FOUND: target %q does not exist", req.Target)
		}
		currentHash := fmt.Sprintf("%x", sha256.Sum256(current))
		if currentHash != req.ExpectedTargetHash {
			return contract.Receipt{}, fmt.Errorf("HQ_VERSION_CONFLICT: target %q hash mismatch: expected %q, got %q", req.Target, req.ExpectedTargetHash, currentHash)
		}
	}

	rendered, err := renderer.Render(ctx, req, current)
	if err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_INVALID_REQUEST: render failed: %w", err)
	}

	cursor, err := te.receipts.NextCursor()
	if err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_INTERNAL_ERROR: allocate cursor: %w", err)
	}

	var backupPath string
	var backupHash string

	intent := IntentRecord{
		RequestID:           requestID,
		Cursor:              cursor,
		Target:              targetPath,
		PreHash:             fmt.Sprintf("%x", sha256.Sum256(current)),
		RenderedContentHash: rendered.SHA256,
		Timestamp:           time.Now().UTC().Format(time.RFC3339),
	}

	if !req.CreateOnly {
		backupDir := filepath.Join(te.root, ".hq-interface", "backups")
		backupName := fmt.Sprintf("%s-%s.bak", filepath.Base(req.Target), requestID)
		backupPath = filepath.Join(backupDir, backupName)
		intent.BackupPath = backupPath
	}

	if err := te.receipts.WriteIntent(intent); err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: write intent: %w", err)
	}
	defer te.receipts.RemoveIntent(requestID)

	if !req.CreateOnly {
		if current != nil {
			hash, err := te.fsys.Backup(targetPath, backupPath)
			if err != nil {
				return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: backup: %w", err)
			}
			backupHash = hash

			readBack, err := os.ReadFile(backupPath)
			if err != nil {
				return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: verify backup: %w", err)
			}
			if fmt.Sprintf("%x", sha256.Sum256(readBack)) != backupHash {
				return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: backup verification failed")
			}
		}
	}

	tmpDir := filepath.Join(te.root, ".hq-interface", "tmp")
	tmpName := fmt.Sprintf("apply-%s.tmp", requestID)
	tmpPath, err := te.fsys.WriteDurable(tmpDir, tmpName, rendered.Content)
	if err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: write rendered temp: %w", err)
	}

	if err := te.fsys.ReplaceDurable(tmpPath, targetPath); err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: replace target: %w", err)
	}

	appliedData, err := os.ReadFile(targetPath)
	if err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: verify applied target: %w", err)
	}
	appliedHash := fmt.Sprintf("%x", sha256.Sum256(appliedData))

	if appliedHash != rendered.SHA256 {
		return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: applied target hash mismatch")
	}

	receipt := contract.Receipt{
		SchemaVersion:     "1.0",
		Cursor:            cursor,
		RequestID:         requestID,
		Target:            req.Target,
		TargetSha256:      rendered.SHA256,
		RenderedSha256:    rendered.SHA256,
		BackupPath:        backupPath,
		BackupSha256:      backupHash,
		ApprovalReference: approvalReference,
		AppliedAt:         time.Now().UTC().Format(time.RFC3339),
	}

	appliedDir := filepath.Join(te.root, ".hq-interface", "requests", "applied")
	newPath := filepath.Join(appliedDir, requestID+".json")
	if err := os.Rename(reqPath, newPath); err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: move to applied: %w", err)
	}

	if err := te.receipts.Store(receipt); err != nil {
		os.Rename(newPath, reqPath)
		return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: store receipt: %w", err)
	}

	return receipt, nil
}
