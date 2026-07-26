package write

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/fsx"
)

var subdirs = []string{
	"pending",
	"applied",
	"rejected",
	"conflicted",
	"recovery-required",
}

func requestsDir(root string) string {
	return filepath.Join(root, ".hq-interface", "requests")
}

func EnsureLayout(fsys fsx.FS, root string) error {
	requestsRoot := filepath.Join(root, ".hq-interface")
	if err := fsys.MkdirPrivate(requestsRoot, 0700); err != nil {
		return fmt.Errorf("HQ_UNSUPPORTED_FILESYSTEM: create interface dir: %w", err)
	}

	reqDir := requestsDir(root)
	if err := fsys.MkdirPrivate(reqDir, 0700); err != nil {
		return fmt.Errorf("HQ_UNSUPPORTED_FILESYSTEM: create requests dir: %w", err)
	}

	for _, sub := range subdirs {
		subPath := filepath.Join(reqDir, sub)
		if err := fsys.MkdirPrivate(subPath, 0700); err != nil {
			return fmt.Errorf("HQ_UNSUPPORTED_FILESYSTEM: create %s dir: %w", sub, err)
		}
	}
	return nil
}

type LayoutHealth struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func LayoutHealthCheck(fsys fsx.FS, root string) []LayoutHealth {
	var checks []LayoutHealth

	reqDir := requestsDir(root)

	for _, sub := range subdirs {
		subPath := filepath.Join(reqDir, sub)
		info, err := os.Stat(subPath)
		check := LayoutHealth{Name: fmt.Sprintf("layout-%s", sub)}
		if err == nil && info.IsDir() {
			check.Status = "pass"
			check.Message = fmt.Sprintf("%s directory exists and is accessible", sub)
		} else if os.IsNotExist(err) {
			check.Status = "warn"
			check.Message = fmt.Sprintf("%s directory does not exist", sub)
		} else {
			check.Status = "fail"
			check.Message = fmt.Sprintf("%s directory error: %v", sub, err)
		}
		checks = append(checks, check)
	}
	return checks
}

type RequestStore struct {
	fsys   fsx.FS
	root   string
	policy *Policy
}

func NewRequestStore(fsys fsx.FS, root string, policy *Policy) *RequestStore {
	return &RequestStore{fsys: fsys, root: root, policy: policy}
}

func (rs *RequestStore) Policy() *Policy {
	return rs.policy
}

func (rs *RequestStore) Submit(req contract.Request) (contract.RequestStatus, error) {
	if err := EnsureLayout(rs.fsys, rs.root); err != nil {
		return contract.RequestStatus{}, err
	}

	if err := contract.ValidateRequest(req); err != nil {
		return contract.RequestStatus{}, err
	}

	if rs.policy != nil {
		class, err := rs.policy.Classify(req.Operation, req.Target)
		if err != nil {
			return contract.RequestStatus{}, fmt.Errorf("HQ_INTERNAL_ERROR: policy classify: %w", err)
		}
		if class == PolicyDenied {
			return contract.RequestStatus{}, fmt.Errorf("HQ_POLICY_DENIED: operation %q target %q is denied by policy", req.Operation, req.Target)
		}
	}

	reqJSON, err := contract.MarshalRequest(req)
	if err != nil {
		return contract.RequestStatus{}, fmt.Errorf("HQ_INTERNAL_ERROR: marshal: %w", err)
	}

	reqSha256, err := contract.CanonicalRequestHash(req)
	if err != nil {
		return contract.RequestStatus{}, fmt.Errorf("HQ_INTERNAL_ERROR: compute hash: %w", err)
	}

	existing, err := rs.findRequest(req.RequestID)
	if err == nil && existing != "" {
		return rs.buildStatusFromFile(existing)
	}

	pendingDir := filepath.Join(requestsDir(rs.root), "pending")
	filename := req.RequestID + ".json"

	tmpDir := filepath.Join(rs.root, ".hq-interface", "tmp")
	tmpName := req.RequestID + ".tmp"

	tmpPath, err := rs.fsys.WriteDurable(tmpDir, tmpName, reqJSON)
	if err != nil {
		return contract.RequestStatus{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: write temp: %w", err)
	}

	dstPath := filepath.Join(pendingDir, filename)
	if err := rs.fsys.RenameAtomic(tmpPath, dstPath); err != nil {
		return contract.RequestStatus{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: rename: %w", err)
	}

	if err := rs.fsys.SyncParent(pendingDir); err != nil {
		return contract.RequestStatus{}, fmt.Errorf("HQ_WRITE_INTERRUPTED: sync: %w", err)
	}

	return contract.RequestStatus{
		RequestID:     req.RequestID,
		State:         contract.StatePending,
		RequestSha256: reqSha256,
		Receipt:       nil,
		Recovery:      nil,
	}, nil
}

func (rs *RequestStore) Status(id string) (contract.RequestStatus, error) {
	if err := contract.ValidateRequest(contract.Request{
		SchemaVersion:      "1.0",
		RequestID:          id,
		Caller:             contract.Caller{Name: "internal"},
		Purpose:            "status lookup",
		Operation:          "project-check-in",
		Target:             "dummy",
		Payload:            []byte(`{}`),
		ExpectedTargetHash: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
	}); err != nil {
		return contract.RequestStatus{}, err
	}

	foundPath, err := rs.findRequest(id)
	if err != nil {
		if os.IsNotExist(err) {
			return contract.RequestStatus{}, fmt.Errorf("HQ_NOT_FOUND: request %q not found", id)
		}
		return contract.RequestStatus{}, fmt.Errorf("HQ_INTERNAL_ERROR: find request: %w", err)
	}
	if foundPath == "" {
		return contract.RequestStatus{}, fmt.Errorf("HQ_NOT_FOUND: request %q not found", id)
	}

	return rs.buildStatusFromFile(foundPath)
}

func (rs *RequestStore) findRequest(id string) (string, error) {
	reqDir := requestsDir(rs.root)
	for _, sub := range subdirs {
		subPath := filepath.Join(reqDir, sub)
		filePath := filepath.Join(subPath, id+".json")
		if _, err := os.Stat(filePath); err == nil {
			return filePath, nil
		}
	}
	return "", nil
}

func (rs *RequestStore) buildStatusFromFile(path string) (contract.RequestStatus, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return contract.RequestStatus{}, fmt.Errorf("HQ_INTERNAL_ERROR: read request file: %w", err)
	}

	req, err := contract.UnmarshalRequest(data)
	if err != nil {
		return contract.RequestStatus{}, fmt.Errorf("HQ_INTERNAL_ERROR: unmarshal request: %w", err)
	}

	sha256, err := contract.CanonicalRequestHash(req)
	if err != nil {
		return contract.RequestStatus{}, fmt.Errorf("HQ_INTERNAL_ERROR: compute hash: %w", err)
	}

	state := stateFromPath(path)

	return contract.RequestStatus{
		RequestID:     req.RequestID,
		State:         state,
		RequestSha256: sha256,
		Receipt:       nil,
		Recovery:      nil,
	}, nil
}

func stateFromPath(path string) contract.RequestState {
	dir := filepath.Base(filepath.Dir(path))
	switch dir {
	case "pending":
		return contract.StatePending
	case "applied":
		return contract.StateApplied
	case "rejected":
		return contract.StateRejected
	case "conflicted":
		return contract.StateConflicted
	case "recovery-required":
		return contract.StateRecoveryRequired
	default:
		return contract.StatePending
	}
}


