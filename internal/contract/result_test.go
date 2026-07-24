package contract_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/contract"
)

func TestNewSuccess_ProducesValidJSON(t *testing.T) {
	data := map[string]string{"version": "0.1.0"}
	r := contract.NewSuccess("version", data)

	var buf bytes.Buffer
	if err := contract.WriteJSON(&buf, r); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	checkField(t, decoded, "schemaVersion", "1.0")
	checkField(t, decoded, "command", "version")
	checkField(t, decoded, "success", true)
	checkField(t, decoded, "mutation", "noMutation")

	if ts, ok := decoded["timestamp"].(string); !ok || ts == "" {
		t.Fatal("timestamp should be a non-empty string")
	}
	if decoded["error"] != nil {
		t.Fatal("success result should have null error")
	}
}

func TestNewError_ProducesValidJSON(t *testing.T) {
	errDetail := contract.ErrDetail(contract.CodeNotFound, "project was not found", false, map[string]any{"collection": "projects"})
	r := contract.NewError("get", errDetail)

	var buf bytes.Buffer
	if err := contract.WriteJSON(&buf, r); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	checkField(t, decoded, "schemaVersion", "1.0")
	checkField(t, decoded, "command", "get")
	checkField(t, decoded, "success", false)
	checkField(t, decoded, "mutation", "noMutation")

	errObj, ok := decoded["error"].(map[string]any)
	if !ok {
		t.Fatal("error field should be present")
	}
	checkField(t, errObj, "code", contract.CodeNotFound)
	checkField(t, errObj, "message", "project was not found")
	checkField(t, errObj, "retryable", false)
}

func TestWarnings_ArrayEmptyByDefault(t *testing.T) {
	r := contract.NewSuccess("list", nil)

	var buf bytes.Buffer
	_ = contract.WriteJSON(&buf, r)

	var decoded map[string]any
	_ = json.Unmarshal(buf.Bytes(), &decoded)

	warnings, ok := decoded["warnings"].([]any)
	if !ok {
		t.Fatal("warnings should be an array")
	}
	if len(warnings) != 0 {
		t.Fatalf("expected empty warnings, got %d", len(warnings))
	}
}

func TestJSON_NoGoInternalFields(t *testing.T) {
	r := contract.NewSuccess("health", nil)
	var buf bytes.Buffer
	_ = contract.WriteJSON(&buf, r)

	output := buf.String()
	for _, unwanted := range []string{"json:", "};\n", "uint64", "struct {", "&Result"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("output contains Go-internal string %q", unwanted)
		}
	}
}

func TestExitCode_Mapping(t *testing.T) {
	tests := []struct {
		code string
		want int
	}{
		{contract.CodeInvalidArgument, 2},
		{contract.CodeNotFound, 3},
		{contract.CodeInvalidRequest, 4},
		{contract.CodePathDenied, 5},
		{contract.CodeApprovalRequired, 6},
		{contract.CodePermissionDenied, 7},
		{contract.CodeVersionConflict, 8},
		{contract.CodeLockTimeout, 9},
		{contract.CodeUnsupportedFS, 10},
		{contract.CodeWriteInterrupted, 11},
		{contract.CodePolicyDenied, 12},
		{contract.CodeInternalError, 70},
		{"UNKNOWN", 1},
		{"", 1},
	}

	for _, tt := range tests {
		got := contract.ExitCode(tt.code)
		if got != tt.want {
			t.Errorf("ExitCode(%q) = %d, want %d", tt.code, got, tt.want)
		}
	}
}

func checkField(t *testing.T, obj map[string]any, key string, want any) {
	t.Helper()
	got, ok := obj[key]
	if !ok {
		t.Fatalf("missing field %q", key)
	}
	if got != want {
		t.Errorf("field %q = %v (%T), want %v (%T)", key, got, got, want, want)
	}
}
