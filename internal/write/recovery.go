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

type RecoveryInspection struct {
	RequestID      string `json:"requestId"`
	State          string `json:"state"`
	Target         string `json:"target"`
	RequestHash    string `json:"requestSha256"`
	TargetExists   bool   `json:"targetExists"`
	TargetHash     string `json:"targetHash,omitempty"`
	BackupExists   bool   `json:"backupExists"`
	BackupPath     string `json:"backupPath,omitempty"`
	BackupHash     string `json:"backupHash,omitempty"`
	HasReceipt     bool   `json:"hasReceipt"`
	IntentExists   bool   `json:"intentExists"`
	RecoveryAction string `json:"recoveryAction"`
}

type RecoveryService struct {
	fsys     fsx.FS
	root     string
	store    *RequestStore
	receipts *ReceiptStore
	policy   *Policy
}

func NewRecoveryService(fsys fsx.FS, root string, store *RequestStore, receipts *ReceiptStore, policy *Policy) *RecoveryService {
	return &RecoveryService{
		fsys:     fsys,
		root:     root,
		store:    store,
		receipts: receipts,
		policy:   policy,
	}
}

func (rs *RecoveryService) Inspect(ctx context.Context, requestID string) (RecoveryInspection, error) {
	if err := contract.ValidateRequestID(requestID); err != nil {
		return RecoveryInspection{}, fmt.Errorf("HQ_INVALID_REQUEST: %w", err)
	}

	status, err := rs.store.Status(requestID)
	if err != nil {
		return RecoveryInspection{}, err
	}

	reqPath := filepath.Join(rs.root, ".hq-interface", "requests", string(status.State), requestID+".json")
	reqData, err := os.ReadFile(reqPath)
	if err != nil {
		return RecoveryInspection{}, fmt.Errorf("HQ_INTERNAL_ERROR: read request: %w", err)
	}

	req, err := contract.UnmarshalRequest(reqData)
	if err != nil {
		return RecoveryInspection{}, fmt.Errorf("HQ_INTERNAL_ERROR: unmarshal: %w", err)
	}

	targetPath := filepath.Join(rs.root, req.Target)
	targetExists := true
	targetHash := ""
	if _, err := os.Stat(targetPath); err != nil {
		targetExists = false
	} else {
		data, err := os.ReadFile(targetPath)
		if err == nil {
			targetHash = fmt.Sprintf("%x", sha256.Sum256(data))
		}
	}

	inspection := RecoveryInspection{
		RequestID:    requestID,
		State:        string(status.State),
		Target:       req.Target,
		RequestHash:  status.RequestSha256,
		TargetExists: targetExists,
		TargetHash:   targetHash,
	}

	receipt, err := rs.receipts.FindByRequestID(requestID)
	if err == nil {
		inspection.HasReceipt = true
		inspection.BackupPath = receipt.BackupPath
		inspection.BackupHash = receipt.BackupSha256
		if receipt.BackupPath != "" {
			if _, err := os.Stat(receipt.BackupPath); err == nil {
				inspection.BackupExists = true
			}
		}
	}

	intent, err := rs.receipts.ReadIntent(requestID)
	if err == nil {
		inspection.IntentExists = true
		_ = intent
	}

	switch {
	case status.State == contract.StateApplied && inspection.HasReceipt:
		inspection.RecoveryAction = "no-action-needed: request is already applied with a receipt"
	case status.State == contract.StateRecoveryRequired:
		if inspection.BackupExists {
			inspection.RecoveryAction = "restore-available: backup exists, use recover restore to revert"
		} else {
			inspection.RecoveryAction = "manual-review: no backup found, inspect target manually"
		}
	case inspection.IntentExists && !inspection.HasReceipt:
		if targetHash == "" {
			inspection.RecoveryAction = "create-not-completed: intent found but target does not exist, recreate request"
		} else {
			inspection.RecoveryAction = "incomplete-apply: intent exists without receipt, check target integrity"
		}
	default:
		inspection.RecoveryAction = "inspect-manually: review request state and target manually"
	}

	return inspection, nil
}

func (rs *RecoveryService) Restore(ctx context.Context, requestID string, approvalReference string) (contract.Receipt, error) {
	if err := contract.ValidateRequestID(requestID); err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_INVALID_REQUEST: %w", err)
	}

	if approvalReference == "" {
		return contract.Receipt{}, fmt.Errorf("HQ_APPROVAL_REQUIRED: restore requires an approval reference")
	}

	if rs.policy != nil {
		class, err := rs.policy.Classify("*", "*")
		if err == nil && class == PolicyApprovalRequired {
		}
	}

	inspection, err := rs.Inspect(ctx, requestID)
	if err != nil {
		return contract.Receipt{}, err
	}

	if !inspection.BackupExists {
		return contract.Receipt{}, fmt.Errorf("HQ_NOT_FOUND: no backup found for request %q", requestID)
	}

	receipt, err := rs.receipts.FindByRequestID(requestID)
	if err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_NOT_FOUND: no receipt found for request %q", requestID)
	}

	if receipt.ApprovalReference != "" && receipt.ApprovalReference != approvalReference {
		return contract.Receipt{}, fmt.Errorf("HQ_APPROVAL_REQUIRED: approval reference %q does not match original receipt reference %q", approvalReference, receipt.ApprovalReference)
	}

	targetPath := filepath.Join(rs.root, inspection.Target)
	currentHash := ""
	if data, err := os.ReadFile(targetPath); err == nil {
		currentHash = fmt.Sprintf("%x", sha256.Sum256(data))
	}

	if currentHash != "" && currentHash != receipt.TargetSha256 {
		return contract.Receipt{}, fmt.Errorf("HQ_VERSION_CONFLICT: target %q hash %q does not match receipt hash %q; target was modified after apply", inspection.Target, currentHash, receipt.TargetSha256)
	}

	backupData, err := os.ReadFile(receipt.BackupPath)
	if err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: read backup: %w", err)
	}

	if receipt.BackupSha256 != "" {
		backupHash := fmt.Sprintf("%x", sha256.Sum256(backupData))
		if backupHash != receipt.BackupSha256 {
			return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: backup hash mismatch: expected %q, got %q", receipt.BackupSha256, backupHash)
		}
	}

	if err := os.WriteFile(targetPath, backupData, 0600); err != nil {
		return contract.Receipt{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: restore write: %w", err)
	}

	recoveryReceipt := contract.Receipt{
		SchemaVersion:     "1.0",
		Cursor:            0,
		RequestID:         requestID,
		Target:            inspection.Target,
		TargetSha256:      fmt.Sprintf("%x", sha256.Sum256(backupData)),
		RenderedSha256:    receipt.TargetSha256,
		BackupPath:        receipt.BackupPath,
		BackupSha256:      receipt.BackupSha256,
		ApprovalReference: approvalReference,
		AppliedAt:         time.Now().UTC().Format(time.RFC3339),
	}

	return recoveryReceipt, nil
}
