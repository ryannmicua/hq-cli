package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/fsx"
	"github.com/ryannmicua/hq-cli/internal/write"
)

type WriteService struct {
	store *write.RequestStore
	fsys  fsx.FS
	root  string
}

func NewWriteService(cfgRoot string, policy *write.Policy) *WriteService {
	f := fsx.NewFS()
	store := write.NewRequestStore(f, cfgRoot, policy)
	return &WriteService{store: store, fsys: f, root: cfgRoot}
}

func (s *WriteService) Store() *write.RequestStore {
	return s.store
}

func (s *WriteService) SharedLock(ctx context.Context) (fsx.UnlockFunc, error) {
	lockPath := filepath.Join(s.root, ".hq-interface", "locks", "global.lock")
	return s.fsys.LockFile(ctx, lockPath, 0, false)
}

const maxRequestSize = 1 << 20 // 1 MiB

func handleSubmit(ctx context.Context, args []string, writeSvc *WriteService, stdout, stderr io.Writer) int {
	flags, _ := parseFlags(args)

	requestFile, ok := flags["request"]
	if !ok {
		r := contract.NewError("submit", contract.ErrDetail(contract.CodeInvalidArgument, "submit requires --request <file.json>", false, nil))
		return writeResult(stdout, r)
	}

	info, err := os.Stat(requestFile)
	if err != nil {
		if os.IsNotExist(err) {
			r := contract.NewError("submit", contract.ErrDetail(contract.CodeNotFound, fmt.Sprintf("request file not found: %s", requestFile), false, nil))
			return writeResult(stdout, r)
		}
		if os.IsPermission(err) {
			r := contract.NewError("submit", contract.ErrDetail(contract.CodePermissionDenied, fmt.Sprintf("cannot stat request file: %s", requestFile), false, nil))
			return writeResult(stdout, r)
		}
		r := contract.NewError("submit", contract.ErrDetail(contract.CodeInternalError, fmt.Sprintf("stat request file: %v", err), false, nil))
		return writeResult(stdout, r)
	}
	if info.Size() > maxRequestSize {
		r := contract.NewError("submit", contract.ErrDetail(contract.CodeInvalidRequest, fmt.Sprintf("request file too large: %d bytes (max %d)", info.Size(), maxRequestSize), false, nil))
		return writeResult(stdout, r)
	}

	data, err := os.ReadFile(requestFile)
	if err != nil {
		if os.IsPermission(err) {
			r := contract.NewError("submit", contract.ErrDetail(contract.CodePermissionDenied, fmt.Sprintf("cannot read request file: %s", requestFile), false, nil))
			return writeResult(stdout, r)
		}
		r := contract.NewError("submit", contract.ErrDetail(contract.CodeInternalError, fmt.Sprintf("read request file: %v", err), false, nil))
		return writeResult(stdout, r)
	}

	var req contract.Request
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		r := contract.NewError("submit", contract.ErrDetail(contract.CodeInvalidRequest, fmt.Sprintf("invalid JSON request: %v", err), false, nil))
		return writeResult(stdout, r)
	}

	status, err := writeSvc.store.Submit(req)
	if err != nil {
		msg := err.Error()
		code := classifyError(msg, []string{
			contract.CodeInvalidRequest,
			contract.CodePolicyDenied,
			contract.CodeNotFound,
			contract.CodePermissionDenied,
			contract.CodeUnsupportedFS,
			contract.CodeWriteInterrupted,
		})
		if code == "" {
			code = contract.CodeInternalError
		}
		r := contract.NewError("submit", contract.ErrDetail(code, msg, false, nil))
		return writeResult(stdout, r)
	}

	dataOut := map[string]any{
		"requestId":     status.RequestID,
		"state":         status.State,
		"requestSha256": status.RequestSha256,
		"statusCommand": fmt.Sprintf("hq status --request-id %s", status.RequestID),
	}

	r := contract.NewSuccess("submit", dataOut)
	return writeResult(stdout, r)
}

func handleStatus(ctx context.Context, args []string, writeSvc *WriteService, stdout, stderr io.Writer) int {
	flags, _ := parseFlags(args)

	requestID, ok := flags["request-id"]
	if !ok {
		r := contract.NewError("status", contract.ErrDetail(contract.CodeInvalidArgument, "status requires --request-id <uuid>", false, nil))
		return writeResult(stdout, r)
	}

	unlock, err := writeSvc.SharedLock(ctx)
	if err != nil {
		r := contract.NewError("status", contract.ErrDetail(contract.CodeLockTimeout, fmt.Sprintf("acquire read lock: %v", err), false, nil))
		return writeResult(stdout, r)
	}
	defer unlock()

	status, err := writeSvc.store.Status(requestID)
	if err != nil {
		msg := err.Error()
		code := classifyError(msg, []string{
			contract.CodeNotFound,
			contract.CodeInvalidRequest,
		})
		if code == "" {
			code = contract.CodeInternalError
		}
		r := contract.NewError("status", contract.ErrDetail(code, msg, false, nil))
		return writeResult(stdout, r)
	}

	dataOut := map[string]any{
		"requestId":     status.RequestID,
		"state":         status.State,
		"requestSha256": status.RequestSha256,
	}

	r := contract.NewSuccess("status", dataOut)
	return writeResult(stdout, r)
}

const maxApprovalRefSize = 1 << 12 // 4 KiB

func resolveApprovalReference(flags map[string]string, stderr io.Writer) string {
	if ref := flags["approval-reference"]; ref != "" {
		fmt.Fprintf(stderr, "WARNING: --approval-reference on command line is deprecated; use HQ_APPROVAL_REFERENCE environment variable or stdin instead\n")
		return ref
	}
	if ref := os.Getenv("HQ_APPROVAL_REFERENCE"); ref != "" {
		return ref
	}
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
		return ""
	}
	data, err := io.ReadAll(io.LimitReader(os.Stdin, maxApprovalRefSize))
	if err == nil {
		ref := strings.TrimSpace(string(data))
		if ref != "" {
			return ref
		}
	}
	return ""
}

func handleApply(ctx context.Context, args []string, writeSvc *WriteService, engine *write.TransactionEngine, stdout, stderr io.Writer) int {
	flags, _ := parseFlags(args)

	requestID, ok := flags["request-id"]
	if !ok {
		r := contract.NewError("apply", contract.ErrDetail(contract.CodeInvalidArgument, "apply requires --request-id <uuid>", false, nil))
		return writeResult(stdout, r)
	}

	approvalRef := resolveApprovalReference(flags, stderr)

	receipt, err := engine.Apply(ctx, requestID, approvalRef)
	if err != nil {
		msg := err.Error()
		code := classifyError(msg, []string{
			contract.CodeNotFound,
			contract.CodeInvalidRequest,
			contract.CodeVersionConflict,
			contract.CodeLockTimeout,
			contract.CodeApprovalRequired,
			contract.CodePolicyDenied,
			contract.CodeWriteInterrupted,
			contract.CodeUnsupportedFS,
		})
		if code == "" {
			code = contract.CodeInternalError
		}
		r := contract.NewError("apply", contract.ErrDetail(code, msg, false, nil))
		return writeResult(stdout, r)
	}

	dataOut := map[string]any{
		"requestId":    receipt.RequestID,
		"cursor":       receipt.Cursor,
		"target":       receipt.Target,
		"targetSha256": receipt.TargetSha256,
		"appliedAt":    receipt.AppliedAt,
	}

	r := contract.NewSuccess("apply", dataOut)
	r.Mutation = contract.MutationApplied
	return writeResult(stdout, r)
}

func handleChanges(ctx context.Context, args []string, changesSvc *write.ChangesService, writeSvc *WriteService, stdout, stderr io.Writer) int {
	flags, _ := parseFlags(args)

	unlock, err := writeSvc.SharedLock(ctx)
	if err != nil {
		r := contract.NewError("changes", contract.ErrDetail(contract.CodeLockTimeout, fmt.Sprintf("acquire read lock: %v", err), false, nil))
		return writeResult(stdout, r)
	}
	defer unlock()

	afterStr := flags["after"]
	sinceStr := flags["since"]

	if afterStr != "" && sinceStr != "" {
		r := contract.NewError("changes", contract.ErrDetail(contract.CodeInvalidArgument, "--since and --after are mutually exclusive", false, nil))
		return writeResult(stdout, r)
	}

	limit := 100
	limitStr := flags["limit"]
	if limitStr != "" {
		if v, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || v != 1 || limit <= 0 {
			r := contract.NewError("changes", contract.ErrDetail(contract.CodeInvalidArgument, fmt.Sprintf("invalid limit: %s", limitStr), false, nil))
			return writeResult(stdout, r)
		}
	}

	if sinceStr != "" {
		since, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			r := contract.NewError("changes", contract.ErrDetail(contract.CodeInvalidArgument, fmt.Sprintf("invalid --since value %q: expected RFC 3339 timestamp", sinceStr), false, nil))
			return writeResult(stdout, r)
		}
		page, err := changesSvc.Since(since, limit)
		if err != nil {
			r := contract.NewError("changes", contract.ErrDetail(contract.CodeInternalError, err.Error(), false, nil))
			return writeResult(stdout, r)
		}
		dataOut := map[string]any{
			"receipts":   page.Receipts,
			"nextCursor": page.NextCursor,
			"hasMore":    page.HasMore,
		}
		r := contract.NewSuccess("changes", dataOut)
		return writeResult(stdout, r)
	}

	var cursor uint64
	if afterStr != "" {
		if _, err := fmt.Sscanf(afterStr, "%d", &cursor); err != nil {
			r := contract.NewError("changes", contract.ErrDetail(contract.CodeInvalidArgument, fmt.Sprintf("invalid cursor value: %s", afterStr), false, nil))
			return writeResult(stdout, r)
		}
	}

	page, err := changesSvc.After(cursor, limit)
	if err != nil {
		r := contract.NewError("changes", contract.ErrDetail(contract.CodeInternalError, err.Error(), false, nil))
		return writeResult(stdout, r)
	}

	dataOut := map[string]any{
		"receipts":   page.Receipts,
		"nextCursor": page.NextCursor,
		"hasMore":    page.HasMore,
	}

	r := contract.NewSuccess("changes", dataOut)
	return writeResult(stdout, r)
}

func handleRecover(ctx context.Context, args []string, recoverySvc *write.RecoveryService, stdout, stderr io.Writer) int {
	flags, positional := parseFlags(args)

	subcommand := ""
	if len(positional) > 0 {
		subcommand = positional[0]
	}

	switch subcommand {
	case "inspect":
		requestID, ok := flags["request-id"]
		if !ok {
			r := contract.NewError("recover", contract.ErrDetail(contract.CodeInvalidArgument, "recover inspect requires --request-id <uuid>", false, nil))
			return writeResult(stdout, r)
		}

		inspection, err := recoverySvc.Inspect(ctx, requestID)
		if err != nil {
			msg := err.Error()
			code := classifyError(msg, []string{
				contract.CodeNotFound,
				contract.CodeInvalidRequest,
			})
			if code == "" {
				code = contract.CodeInternalError
			}
			r := contract.NewError("recover", contract.ErrDetail(code, msg, false, nil))
			return writeResult(stdout, r)
		}

		r := contract.NewSuccess("recover", inspection)
		return writeResult(stdout, r)

	case "restore":
		requestID, ok := flags["request-id"]
		if !ok {
			r := contract.NewError("recover", contract.ErrDetail(contract.CodeInvalidArgument, "recover restore requires --request-id <uuid>", false, nil))
			return writeResult(stdout, r)
		}

		approvalRef := resolveApprovalReference(flags, stderr)

		receipt, err := recoverySvc.Restore(ctx, requestID, approvalRef)
		if err != nil {
			msg := err.Error()
			code := classifyError(msg, []string{
				contract.CodeNotFound,
				contract.CodeInvalidRequest,
				contract.CodeVersionConflict,
				contract.CodeApprovalRequired,
				contract.CodeWriteInterrupted,
			})
			if code == "" {
				code = contract.CodeInternalError
			}
			r := contract.NewError("recover", contract.ErrDetail(code, msg, false, nil))
			return writeResult(stdout, r)
		}

		r := contract.NewSuccess("recover", receipt)
		r.Mutation = contract.MutationApplied
		return writeResult(stdout, r)

	default:
		r := contract.NewError("recover", contract.ErrDetail(contract.CodeInvalidArgument, "recover requires subcommand: inspect or restore", false, nil))
		return writeResult(stdout, r)
	}
}

func isErrorCode(msg, code string) bool {
	return strings.HasPrefix(msg, code+":")
}

func classifyError(msg string, codes []string) string {
	for _, code := range codes {
		if isErrorCode(msg, code) {
			return code
		}
	}
	return ""
}
