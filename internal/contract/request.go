package contract

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type RequestState string

const (
	StatePending          RequestState = "pending"
	StateApplied          RequestState = "applied"
	StateRejected         RequestState = "rejected"
	StateConflicted       RequestState = "conflicted"
	StateRecoveryRequired RequestState = "recovery-required"
)

type Caller struct {
	Name      string `json:"name"`
	SessionID string `json:"sessionId"`
}

type Request struct {
	SchemaVersion      string          `json:"schemaVersion"`
	RequestID          string          `json:"requestId"`
	Caller             Caller          `json:"caller"`
	Purpose            string          `json:"purpose"`
	Operation          string          `json:"operation"`
	Target             string          `json:"target"`
	Payload            json.RawMessage `json:"payload"`
	SubmittedAt        string          `json:"submittedAt"`
	ExpectedTargetHash string          `json:"expectedTargetHash"`
	CreateOnly         bool            `json:"createOnly"`
	ApprovalReference  *string         `json:"approvalReference"`
}

type RequestStatus struct {
	RequestID     string       `json:"requestId"`
	State         RequestState `json:"state"`
	RequestSha256 string       `json:"requestSha256"`
	Receipt       any          `json:"receipt"`
	Recovery      any          `json:"recovery"`
}

type ProjectCheckIn struct {
	Summary        string   `json:"summary"`
	CurrentOutcome string   `json:"currentOutcome"`
	CurrentState   string   `json:"currentState"`
	NextAction     string   `json:"nextAction"`
	Blockers       []string `json:"blockers"`
	Risks          []string `json:"risks"`
	Evidence       []string `json:"evidence"`
	VerifiedAt     string   `json:"verifiedAt"`
}

type SessionEntry struct {
	Timestamp string   `json:"timestamp"`
	Tags      []string `json:"tags"`
	Summary   string   `json:"summary"`
}

type DraftRecord struct {
	Title          string `json:"title"`
	Body           string `json:"body"`
	RecordDate     string `json:"recordDate"`
	Classification string `json:"classification"`
}

var AllowedClassifications = map[string]bool{
	"inbox":           true,
	"project-report": true,
	"project-source": true,
}

type CurrentWorkUpdate struct {
	WorkName          string   `json:"workName"`
	Workspace         string   `json:"workspace"`
	LoadFirst         string   `json:"loadFirst"`
	SupportingContext []string `json:"supportingContext"`
	CurrentOutcome    string   `json:"currentOutcome"`
	CurrentState      string   `json:"currentState"`
	NextAction        string   `json:"nextAction"`
	LastTouched       string   `json:"lastTouched"`
	Section           string   `json:"section"`
}

func DecodeStrict(data json.RawMessage, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func OperationPayload(operation string, data json.RawMessage) (any, error) {
	switch operation {
	case "project-check-in":
		var v ProjectCheckIn
		if err := DecodeStrict(data, &v); err != nil {
			return nil, fmt.Errorf("unmarshal project-check-in: %w", err)
		}
		return &v, nil
	case "session-entry":
		var v SessionEntry
		if err := DecodeStrict(data, &v); err != nil {
			return nil, fmt.Errorf("unmarshal session-entry: %w", err)
		}
		return &v, nil
	case "draft-record":
		var v DraftRecord
		if err := DecodeStrict(data, &v); err != nil {
			return nil, fmt.Errorf("unmarshal draft-record: %w", err)
		}
		return &v, nil
	case "current-work-update":
		var v CurrentWorkUpdate
		if err := DecodeStrict(data, &v); err != nil {
			return nil, fmt.Errorf("unmarshal current-work-update: %w", err)
		}
		return &v, nil
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

func CanonicalRequestHash(req Request) (string, error) {
	fields := map[string]any{
		"schemaVersion":      req.SchemaVersion,
		"requestId":          req.RequestID,
		"caller":             req.Caller,
		"purpose":            req.Purpose,
		"operation":          req.Operation,
		"target":             req.Target,
		"payload":            req.Payload,
		"submittedAt":        req.SubmittedAt,
		"expectedTargetHash": req.ExpectedTargetHash,
		"createOnly":         req.CreateOnly,
	}

	if err := validatePayloadForHash(req.Payload); err != nil {
		return "", err
	}

	ordered := marshalOrdered(fields)
	h := sha256.Sum256([]byte(ordered))
	return fmt.Sprintf("%x", h), nil
}

func validatePayloadForHash(payload json.RawMessage) error {
	if len(payload) == 0 {
		return fmt.Errorf("payload is empty")
	}
	if !json.Valid(payload) {
		return fmt.Errorf("payload is not valid JSON")
	}
	return nil
}

func marshalOrdered(fields map[string]any) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		v := fields[k]
		switch val := v.(type) {
		case string:
			parts = append(parts, fmt.Sprintf("%s:%s", k, val))
		case bool:
			parts = append(parts, fmt.Sprintf("%s:%t", k, val))
		case json.RawMessage:
			raw := string(val)
			if raw == "null" {
				raw = ""
			}
			parts = append(parts, fmt.Sprintf("%s:%s", k, raw))
		default:
			b, _ := json.Marshal(val)
			parts = append(parts, fmt.Sprintf("%s:%s", k, string(b)))
		}
	}
	return strings.Join(parts, "|")
}

var ValidOperations = map[string]bool{
	"project-check-in":    true,
	"session-entry":       true,
	"draft-record":        true,
	"current-work-update": true,
}

func ValidateRequest(req Request) error {
	if req.SchemaVersion == "" {
		return fmt.Errorf("HQ_INVALID_REQUEST: schemaVersion is required")
	}
	if req.RequestID == "" {
		return fmt.Errorf("HQ_INVALID_REQUEST: requestId is required")
	}
	if err := ValidateRequestID(req.RequestID); err != nil {
		return err
	}
	if req.Caller.Name == "" {
		return fmt.Errorf("HQ_INVALID_REQUEST: caller.name is required")
	}
	if req.Purpose == "" {
		return fmt.Errorf("HQ_INVALID_REQUEST: purpose is required")
	}
	if req.Operation == "" {
		return fmt.Errorf("HQ_INVALID_REQUEST: operation is required")
	}
	if !ValidOperations[req.Operation] {
		return fmt.Errorf("HQ_INVALID_REQUEST: unknown operation: %s", req.Operation)
	}
	if req.Target == "" {
		return fmt.Errorf("HQ_INVALID_REQUEST: target is required")
	}
	if err := validateTarget(req.Target); err != nil {
		return err
	}
	if len(req.Payload) == 0 {
		return fmt.Errorf("HQ_INVALID_REQUEST: payload is required")
	}
	if !json.Valid(req.Payload) {
		return fmt.Errorf("HQ_INVALID_REQUEST: payload is not valid JSON")
	}
	if err := validateHashOrCreate(req); err != nil {
		return err
	}
	return nil
}

func ValidateRequestID(id string) error {
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		return fmt.Errorf("HQ_INVALID_REQUEST: requestId %q is not a valid UUID (8-4-4-4-12)", id)
	}
	if len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
		return fmt.Errorf("HQ_INVALID_REQUEST: requestId %q is not a valid UUID length", id)
	}
	// Check lowercase hex only.
	for _, part := range parts {
		for _, c := range part {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return fmt.Errorf("HQ_INVALID_REQUEST: requestId %q must be lowercase hex", id)
			}
		}
	}
	return nil
}

func validateTarget(target string) error {
	if strings.HasPrefix(target, "/") {
		return fmt.Errorf("HQ_PATH_DENIED: target %q is absolute", target)
	}
	if strings.Contains(target, "../") || strings.HasPrefix(target, "../") || target == ".." {
		return fmt.Errorf("HQ_PATH_DENIED: target %q contains traversal", target)
	}
	if strings.Contains(target, "..\\") || strings.HasPrefix(target, "..\\") {
		return fmt.Errorf("HQ_PATH_DENIED: target %q contains traversal", target)
	}
	return nil
}

func MarshalRequest(req Request) ([]byte, error) {
	return json.Marshal(req)
}

func UnmarshalRequest(data []byte) (Request, error) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return req, err
	}
	return req, nil
}

func validateHashOrCreate(req Request) error {
	hasHash := req.ExpectedTargetHash != ""
	isCreate := req.CreateOnly

	if hasHash && isCreate {
		return fmt.Errorf("HQ_INVALID_REQUEST: expectedTargetHash and createOnly are mutually exclusive")
	}
	if !hasHash && !isCreate {
		return fmt.Errorf("HQ_INVALID_REQUEST: exactly one of expectedTargetHash or createOnly is required")
	}
	if hasHash {
		if len(req.ExpectedTargetHash) != 64 {
			return fmt.Errorf("HQ_INVALID_REQUEST: expectedTargetHash must be 64 hex characters")
		}
		for _, c := range req.ExpectedTargetHash {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return fmt.Errorf("HQ_INVALID_REQUEST: expectedTargetHash must be hex")
			}
		}
	}
	return nil
}
