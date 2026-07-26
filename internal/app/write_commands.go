package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/fsx"
	"github.com/ryannmicua/hq-cli/internal/write"
)

type WriteService struct {
	store *write.RequestStore
}

func NewWriteService(cfgRoot string, policy *write.Policy) *WriteService {
	f := fsx.NewFS()
	store := write.NewRequestStore(f, cfgRoot, policy)
	return &WriteService{store: store}
}

func (s *WriteService) Store() *write.RequestStore {
	return s.store
}

func handleSubmit(ctx context.Context, args []string, writeSvc *WriteService, stdout, stderr io.Writer) int {
	flags, _ := parseFlags(args)

	requestFile, ok := flags["request"]
	if !ok {
		r := contract.NewError("submit", contract.ErrDetail(contract.CodeInvalidArgument, "submit requires --request <file.json>", false, nil))
		return writeResult(stdout, r)
	}

	data, err := os.ReadFile(requestFile)
	if err != nil {
		if os.IsNotExist(err) {
			r := contract.NewError("submit", contract.ErrDetail(contract.CodeNotFound, fmt.Sprintf("request file not found: %s", requestFile), false, nil))
			return writeResult(stdout, r)
		}
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
		code := contract.CodeInternalError
		msg := err.Error()

		if isErrorCode(msg, contract.CodeInvalidRequest) {
			code = contract.CodeInvalidRequest
		} else if isErrorCode(msg, contract.CodePolicyDenied) {
			code = contract.CodePolicyDenied
		} else if isErrorCode(msg, contract.CodeNotFound) {
			code = contract.CodeNotFound
		} else if isErrorCode(msg, contract.CodePermissionDenied) {
			code = contract.CodePermissionDenied
		} else if isErrorCode(msg, contract.CodeUnsupportedFS) {
			code = contract.CodeUnsupportedFS
		} else if isErrorCode(msg, contract.CodeWriteInterrupted) {
			code = contract.CodeWriteInterrupted
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

	status, err := writeSvc.store.Status(requestID)
	if err != nil {
		code := contract.CodeInternalError
		msg := err.Error()

		if isErrorCode(msg, contract.CodeNotFound) {
			code = contract.CodeNotFound
		} else if isErrorCode(msg, contract.CodeInvalidRequest) {
			code = contract.CodeInvalidRequest
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

func isErrorCode(msg, code string) bool {
	return strings.HasPrefix(msg, code+":") || strings.HasPrefix(msg, code)
}
