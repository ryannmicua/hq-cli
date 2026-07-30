package app

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestResolveApprovalReference_Flag(t *testing.T) {
	flags := map[string]string{"approval-reference": "ref-001"}
	var buf bytes.Buffer
	got := resolveApprovalReference(flags, &buf)
	if got != "ref-001" {
		t.Fatalf("expected ref-001, got %s", got)
	}
	if !strings.Contains(buf.String(), "WARNING") {
		t.Fatal("expected deprecation warning on stderr")
	}
}

func TestResolveApprovalReference_EnvVar(t *testing.T) {
	os.Setenv("HQ_APPROVAL_REFERENCE", "env-ref-001")
	defer os.Unsetenv("HQ_APPROVAL_REFERENCE")
	var buf bytes.Buffer
	got := resolveApprovalReference(map[string]string{}, &buf)
	if got != "env-ref-001" {
		t.Fatalf("expected env-ref-001, got %s", got)
	}
}

func TestResolveApprovalReference_None(t *testing.T) {
	os.Unsetenv("HQ_APPROVAL_REFERENCE")
	var buf bytes.Buffer
	got := resolveApprovalReference(map[string]string{}, &buf)
	if got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

func TestResolveApprovalReference_FlagWinsOverEnv(t *testing.T) {
	os.Setenv("HQ_APPROVAL_REFERENCE", "env-ref")
	defer os.Unsetenv("HQ_APPROVAL_REFERENCE")
	flags := map[string]string{"approval-reference": "flag-ref"}
	var buf bytes.Buffer
	got := resolveApprovalReference(flags, &buf)
	if got != "flag-ref" {
		t.Fatalf("expected flag-ref (flag wins), got %s", got)
	}
}
