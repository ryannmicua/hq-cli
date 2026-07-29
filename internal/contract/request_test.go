package contract_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/contract"
)

func TestRequestMarshalUnmarshalRoundTrip(t *testing.T) {
	payload := json.RawMessage(`{"summary":"test","currentState":"done","nextAction":"ship"}`)
	orig := contract.Request{
		SchemaVersion:      "1.0",
		RequestID:          "018f0000-0000-7000-8000-000000000001",
		Caller:             contract.Caller{Name: "tester", SessionID: "sess-1"},
		Purpose:            "Testing round trip",
		Operation:          "project-check-in",
		Target:             "projects/test/STATE.md",
		Payload:            payload,
		SubmittedAt:        "2026-07-26T00:00:00Z",
		ExpectedTargetHash: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
		CreateOnly:         false,
	}

	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded contract.Request
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.RequestID != orig.RequestID {
		t.Fatalf("RequestID = %q, want %q", decoded.RequestID, orig.RequestID)
	}
	if decoded.Caller.Name != orig.Caller.Name {
		t.Fatalf("Caller.Name = %q, want %q", decoded.Caller.Name, orig.Caller.Name)
	}
	if decoded.Operation != orig.Operation {
		t.Fatalf("Operation = %q, want %q", decoded.Operation, orig.Operation)
	}
	if decoded.Target != orig.Target {
		t.Fatalf("Target = %q, want %q", decoded.Target, orig.Target)
	}
}

func TestCanonicalRequestHashDeterministic(t *testing.T) {
	payload := json.RawMessage(`{"summary":"x"}`)
	req1 := contract.Request{
		SchemaVersion: "1.0",
		RequestID:     "018f0000-0000-7000-8000-000000000001",
		Caller:        contract.Caller{Name: "tester"},
		Purpose:       "test",
		Operation:     "project-check-in",
		Target:        "projects/test/STATE.md",
		Payload:       payload,
		SubmittedAt:   "2026-07-26T00:00:00Z",
	}
	req2 := req1

	h1, err := contract.CanonicalRequestHash(req1)
	if err != nil {
		t.Fatalf("hash1: %v", err)
	}
	h2, err := contract.CanonicalRequestHash(req2)
	if err != nil {
		t.Fatalf("hash2: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("hashes differ for identical requests: %q vs %q", h1, h2)
	}
}

func TestCanonicalRequestHashDiffersOnChange(t *testing.T) {
	payload := json.RawMessage(`{"summary":"x"}`)
	base := contract.Request{
		SchemaVersion: "1.0",
		RequestID:     "018f0000-0000-7000-8000-000000000001",
		Caller:        contract.Caller{Name: "tester"},
		Purpose:       "test",
		Operation:     "project-check-in",
		Target:        "projects/test/STATE.md",
		Payload:       payload,
		SubmittedAt:   "2026-07-26T00:00:00Z",
	}

	h1, _ := contract.CanonicalRequestHash(base)

	modified := base
	modified.Purpose = "different"
	h2, _ := contract.CanonicalRequestHash(modified)

	if h1 == h2 {
		t.Fatal("hashes should differ when purpose changes")
	}
}

func TestCanonicalRequestHashDiffersOnPayload(t *testing.T) {
	base := contract.Request{
		SchemaVersion: "1.0",
		RequestID:     "018f0000-0000-7000-8000-000000000001",
		Caller:        contract.Caller{Name: "tester"},
		Purpose:       "test",
		Operation:     "project-check-in",
		Target:        "projects/test/STATE.md",
		Payload:       json.RawMessage(`{"summary":"x"}`),
	}

	h1, _ := contract.CanonicalRequestHash(base)

	modified := base
	modified.Payload = json.RawMessage(`{"summary":"y"}`)
	h2, _ := contract.CanonicalRequestHash(modified)

	if h1 == h2 {
		t.Fatal("hashes should differ when payload changes")
	}
}

func TestOperationPayloadProjectCheckIn(t *testing.T) {
	data := json.RawMessage(`{"summary":"s","currentState":"cs","nextAction":"na"}`)
	p, err := contract.OperationPayload("project-check-in", data)
	if err != nil {
		t.Fatalf("OperationPayload failed: %v", err)
	}
	v, ok := p.(*contract.ProjectCheckIn)
	if !ok {
		t.Fatalf("expected *ProjectCheckIn, got %T", p)
	}
	if v.Summary != "s" {
		t.Fatalf("Summary = %q, want %q", v.Summary, "s")
	}
}

func TestOperationPayloadSessionEntry(t *testing.T) {
	data := json.RawMessage(`{"timestamp":"2026-07-26T00:00:00Z","tags":["test"],"summary":"test session"}`)
	p, err := contract.OperationPayload("session-entry", data)
	if err != nil {
		t.Fatalf("OperationPayload failed: %v", err)
	}
	v, ok := p.(*contract.SessionEntry)
	if !ok {
		t.Fatalf("expected *SessionEntry, got %T", p)
	}
	if v.Summary != "test session" {
		t.Fatalf("Summary = %q, want %q", v.Summary, "test session")
	}
}

func TestOperationPayloadDraftRecord(t *testing.T) {
	data := json.RawMessage(`{"title":"Draft","body":"Body","recordDate":"2026-07-26","classification":"inbox"}`)
	p, err := contract.OperationPayload("draft-record", data)
	if err != nil {
		t.Fatalf("OperationPayload failed: %v", err)
	}
	v, ok := p.(*contract.DraftRecord)
	if !ok {
		t.Fatalf("expected *DraftRecord, got %T", p)
	}
	if v.Title != "Draft" {
		t.Fatalf("Title = %q, want %q", v.Title, "Draft")
	}
	if v.Classification != "inbox" {
		t.Fatalf("Classification = %q, want %q", v.Classification, "inbox")
	}
}

func TestOperationPayloadCurrentWorkUpdate(t *testing.T) {
	data := json.RawMessage(`{"workName":"HQ CLI","workspace":"projects/hq-cli/","loadFirst":"STATE.md","currentState":"done","nextAction":"next","lastTouched":"2026-07-26"}`)
	p, err := contract.OperationPayload("current-work-update", data)
	if err != nil {
		t.Fatalf("OperationPayload failed: %v", err)
	}
	v, ok := p.(*contract.CurrentWorkUpdate)
	if !ok {
		t.Fatalf("expected *CurrentWorkUpdate, got %T", p)
	}
	if v.WorkName != "HQ CLI" {
		t.Fatalf("WorkName = %q, want %q", v.WorkName, "HQ CLI")
	}
}

func TestOperationPayloadUnknown(t *testing.T) {
	_, err := contract.OperationPayload("unknown-op", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for unknown operation")
	}
	if !strings.Contains(err.Error(), "unknown operation") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOperationPayloadInvalidJSON(t *testing.T) {
	_, err := contract.OperationPayload("project-check-in", json.RawMessage(`{invalid}`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestOperationPayloadRoundTrip(t *testing.T) {
	orig := contract.ProjectCheckIn{
		Summary:      "s",
		CurrentState: "cs",
		NextAction:   "na",
		Blockers:     []string{"b1"},
	}
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	p, err := contract.OperationPayload("project-check-in", b)
	if err != nil {
		t.Fatalf("OperationPayload: %v", err)
	}
	v := p.(*contract.ProjectCheckIn)
	if v.Summary != orig.Summary {
		t.Fatalf("Summary = %q, want %q", v.Summary, orig.Summary)
	}
	if len(v.Blockers) != 1 || v.Blockers[0] != "b1" {
		t.Fatalf("Blockers = %v, want [b1]", v.Blockers)
	}
}

func TestDraftRecordClassificationAllowed(t *testing.T) {
	allowed := []string{"inbox", "project-report", "project-source"}
	for _, c := range allowed {
		if !contract.AllowedClassifications[c] {
			t.Fatalf("classification %q should be allowed", c)
		}
	}
}

func TestRequestStateConstants(t *testing.T) {
	if contract.StatePending != "pending" {
		t.Fatalf("StatePending = %q, want %q", contract.StatePending, "pending")
	}
	if contract.StateApplied != "applied" {
		t.Fatalf("StateApplied = %q, want %q", contract.StateApplied, "applied")
	}
	if contract.StateRejected != "rejected" {
		t.Fatalf("StateRejected = %q, want %q", contract.StateRejected, "rejected")
	}
	if contract.StateConflicted != "conflicted" {
		t.Fatalf("StateConflicted = %q, want %q", contract.StateConflicted, "conflicted")
	}
	if contract.StateRecoveryRequired != "recovery-required" {
		t.Fatalf("StateRecoveryRequired = %q, want %q", contract.StateRecoveryRequired, "recovery-required")
	}
}

func TestEmptyRequestIDProducesValidationError(t *testing.T) {
	req := contract.Request{
		SchemaVersion:      "1.0",
		RequestID:          "",
		Caller:             contract.Caller{Name: "t"},
		Purpose:            "test",
		Operation:          "project-check-in",
		Target:             "projects/test/STATE.md",
		Payload:            json.RawMessage(`{}`),
		ExpectedTargetHash: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
	}
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected validation error for empty RequestID")
	}
	if !strings.Contains(err.Error(), "requestId") {
		t.Fatalf("error should mention requestId: %v", err)
	}
}

func TestDisallowUnknownFields(t *testing.T) {
	data := `{"schemaVersion":"1.0","requestId":"018f0000-0000-7000-8000-000000000001","caller":{"name":"t"},"purpose":"p","operation":"project-check-in","target":"t","payload":{},"unknownField":"x","expectedTargetHash":"abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"}`
	var req contract.Request
	dec := json.NewDecoder(strings.NewReader(data))
	dec.DisallowUnknownFields()
	err := dec.Decode(&req)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func validRequest() contract.Request {
	return contract.Request{
		SchemaVersion:      "1.0",
		RequestID:          "018f0000-0000-7000-8000-000000000001",
		Caller:             contract.Caller{Name: "tester"},
		Purpose:            "test",
		Operation:          "project-check-in",
		Target:             "projects/test/STATE.md",
		Payload:            json.RawMessage(`{"summary":"s","currentState":"cs","nextAction":"na"}`),
		ExpectedTargetHash: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
	}
}

// U3 validation tests

func TestValidateRequestValid(t *testing.T) {
	req := validRequest()
	if err := contract.ValidateRequest(req); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateRequestMissingRequestID(t *testing.T) {
	req := validRequest()
	req.RequestID = ""
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HQ_INVALID_REQUEST") {
		t.Fatalf("expected HQ_INVALID_REQUEST, got: %v", err)
	}
}

func TestValidateRequestNonUUID(t *testing.T) {
	req := validRequest()
	req.RequestID = "not-a-uuid"
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HQ_INVALID_REQUEST") {
		t.Fatalf("expected HQ_INVALID_REQUEST, got: %v", err)
	}
}

func TestValidateRequestUppercaseUUID(t *testing.T) {
	req := validRequest()
	req.RequestID = "018F0000-0000-7000-8000-000000000001"
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error for uppercase UUID")
	}
	if !strings.Contains(err.Error(), "HQ_INVALID_REQUEST") {
		t.Fatalf("expected HQ_INVALID_REQUEST, got: %v", err)
	}
}

func TestValidateRequestMissingOperation(t *testing.T) {
	req := validRequest()
	req.Operation = ""
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HQ_INVALID_REQUEST") {
		t.Fatalf("expected HQ_INVALID_REQUEST, got: %v", err)
	}
}

func TestValidateRequestUnknownOperation(t *testing.T) {
	req := validRequest()
	req.Operation = "unknown-op"
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HQ_INVALID_REQUEST") {
		t.Fatalf("expected HQ_INVALID_REQUEST, got: %v", err)
	}
}

func TestValidateRequestMissingTarget(t *testing.T) {
	req := validRequest()
	req.Target = ""
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRequestTargetTraversal(t *testing.T) {
	req := validRequest()
	req.Target = "../escape.md"
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HQ_PATH_DENIED") {
		t.Fatalf("expected HQ_PATH_DENIED, got: %v", err)
	}
}

func TestValidateRequestTargetAbsolute(t *testing.T) {
	req := validRequest()
	req.Target = "/etc/passwd"
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HQ_PATH_DENIED") {
		t.Fatalf("expected HQ_PATH_DENIED, got: %v", err)
	}
}

func TestValidateRequestMissingHashAndCreate(t *testing.T) {
	req := validRequest()
	req.ExpectedTargetHash = ""
	req.CreateOnly = false
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HQ_INVALID_REQUEST") {
		t.Fatalf("expected HQ_INVALID_REQUEST, got: %v", err)
	}
}

func TestValidateRequestBothHashAndCreate(t *testing.T) {
	req := validRequest()
	req.CreateOnly = true
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually exclusive error, got: %v", err)
	}
}

func TestValidateRequestHashNonHex(t *testing.T) {
	req := validRequest()
	req.ExpectedTargetHash = "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "must be hex") {
		t.Fatalf("expected hex error, got: %v", err)
	}
}

func TestValidateRequestHashShort(t *testing.T) {
	req := validRequest()
	req.ExpectedTargetHash = "abcd"
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "64 hex") {
		t.Fatalf("expected length error, got: %v", err)
	}
}

func TestValidateRequestValidCreateOnly(t *testing.T) {
	req := validRequest()
	req.ExpectedTargetHash = ""
	req.CreateOnly = true
	if err := contract.ValidateRequest(req); err != nil {
		t.Fatalf("expected no error for valid createOnly, got: %v", err)
	}
}

func TestValidateRequestValidHash(t *testing.T) {
	req := validRequest()
	req.CreateOnly = false
	if err := contract.ValidateRequest(req); err != nil {
		t.Fatalf("expected no error for valid hash, got: %v", err)
	}
}

func TestValidateRequestMissingCaller(t *testing.T) {
	req := validRequest()
	req.Caller.Name = ""
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "caller.name") {
		t.Fatalf("expected caller.name error, got: %v", err)
	}
}

func TestValidateRequestMissingPurpose(t *testing.T) {
	req := validRequest()
	req.Purpose = ""
	err := contract.ValidateRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "purpose") {
		t.Fatalf("expected purpose error, got: %v", err)
	}
}

func TestValidateRequestValidSessionEntry(t *testing.T) {
	req := validRequest()
	req.Operation = "session-entry"
	req.Target = "SESSION-LOG.md"
	req.Payload = json.RawMessage(`{"timestamp":"2026-07-26T00:00:00Z","tags":["t"],"summary":"test"}`)
	if err := contract.ValidateRequest(req); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateRequestValidDraftRecord(t *testing.T) {
	req := validRequest()
	req.Operation = "draft-record"
	req.Target = "projects/test/new-draft.md"
	req.ExpectedTargetHash = ""
	req.CreateOnly = true
	req.Payload = json.RawMessage(`{"title":"Draft","body":"Body","recordDate":"2026-07-26","classification":"inbox"}`)
	if err := contract.ValidateRequest(req); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateRequestIDValid(t *testing.T) {
	if err := contract.ValidateRequestID("018f0000-0000-7000-8000-000000000001"); err != nil {
		t.Fatalf("expected no error for valid UUID, got: %v", err)
	}
}

func TestValidateRequestIDInvalid(t *testing.T) {
	if err := contract.ValidateRequestID("not-a-uuid"); err == nil {
		t.Fatal("expected error for non-UUID string")
	}
}

func TestValidateRequestIDUppercase(t *testing.T) {
	if err := contract.ValidateRequestID("018F0000-0000-7000-8000-000000000001"); err == nil {
		t.Fatal("expected error for uppercase UUID")
	}
}

func TestValidateRequestIDEmpty(t *testing.T) {
	if err := contract.ValidateRequestID(""); err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestValidateRequestValidCurrentWorkUpdate(t *testing.T) {
	req := validRequest()
	req.Operation = "current-work-update"
	req.Target = "CURRENT-WORK.md"
	req.Payload = json.RawMessage(`{"workName":"HQ CLI","workspace":"projects/hq-cli/","loadFirst":"STATE.md","currentState":"done","nextAction":"next","lastTouched":"2026-07-26"}`)
	if err := contract.ValidateRequest(req); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}
