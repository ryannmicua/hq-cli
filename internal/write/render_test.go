package write_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/write"
)

func TestRendererContractAssumptions_ProjectStateSectionOrder(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "hq", "projects", "example", "STATE.md"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	content := string(data)
	sections := []string{
		"# Example Project State",
		"## Executive Summary",
		"## Current Outcome",
		"## Current State",
		"## Next Action",
		"## Open Decisions",
		"## Blockers",
		"## Material Risks",
		"## Evidence",
	}

	lastIdx := -1
	for _, s := range sections {
		idx := strings.Index(content, s)
		if idx == -1 {
			t.Fatalf("missing section: %s", s)
		}
		if idx <= lastIdx {
			t.Fatalf("section %q appears before previous section", s)
		}
		lastIdx = idx
	}
}

func TestRendererContractAssumptions_SessionLogFormat(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "hq", "SESSION-LOG.md"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "# Session Log") {
		t.Fatal("SESSION-LOG.md must start with # Session Log")
	}
	if !strings.Contains(content, "---") {
		t.Fatal("SESSION-LOG.md must contain separator ---")
	}

	parts := strings.Split(content, "---")
	if len(parts) < 2 {
		t.Fatal("SESSION-LOG.md must have content after separator")
	}

	entries := strings.TrimSpace(parts[len(parts)-1])
	lines := strings.Split(entries, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.Contains(line, ":") {
			t.Fatalf("entry does not contain timestamp: %q", line)
		}
	}
}

func TestRendererContractAssumptions_CurrentWorkFormat(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "hq", "CURRENT-WORK.md"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "# Current Work") {
		t.Fatal("CURRENT-WORK.md must start with # Current Work")
	}
	if !strings.Contains(content, "## Active") {
		t.Fatal("CURRENT-WORK.md must contain ## Active section")
	}
	if !strings.Contains(content, "## Warm") {
		t.Fatal("CURRENT-WORK.md must contain ## Warm section")
	}
	if !strings.Contains(content, "- Workspace:") {
		t.Fatal("entry must contain Workspace field")
	}
	if !strings.Contains(content, "- Load first:") {
		t.Fatal("entry must contain Load first field")
	}
}

func TestRendererContractAssumptions_DraftRecordFixture(t *testing.T) {
	t.Log("No existing draft-record fixture; contract defines inbox/, projects/*/reports/, projects/*/source/ as destinations")
}

func TestRendererProjectCheckIn_Golden(t *testing.T) {
	current, err := os.ReadFile(filepath.Join("..", "..", "testdata", "hq", "projects", "example", "STATE.md"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	payload, _ := json.Marshal(contract.ProjectCheckIn{
		Summary:        "Updated summary",
		CurrentOutcome: "Testing renderer output",
		CurrentState:   "Golden test phase",
		NextAction:     "Verify golden output",
		Blockers:       []string{"None"},
		Risks:          []string{"Golden fixture drift"},
		Evidence:       []string{"testdata/golden/project-check-in.md, verified 2026-07-29"},
		VerifiedAt:     "2026-07-29",
	})

	req := contract.Request{
		SchemaVersion:      "1.0",
		RequestID:          "018f0000-0000-7000-8000-000000000001",
		Operation:          "project-check-in",
		Target:             "projects/example/STATE.md",
		Payload:            payload,
		ExpectedTargetHash: fmt.Sprintf("%064x", sha256.Sum256(current)),
	}

	renderer := write.NewProjectCheckInRenderer()
	result, err := renderer.Render(context.Background(), req, current)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if result.Path != req.Target {
		t.Fatalf("Path = %q, want %q", result.Path, req.Target)
	}

	if result.SHA256 == "" {
		t.Fatal("expected non-empty SHA256")
	}

	if string(result.Content) == string(current) {
		t.Fatal("rendered content must differ from original")
	}

	goldenPath := filepath.Join("..", "..", "testdata", "golden", "project-check-in.md")
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Logf("golden file %s does not exist, creating from actual output", goldenPath)
		os.MkdirAll(filepath.Dir(goldenPath), 0700)
		os.WriteFile(goldenPath, result.Content, 0644)
		t.Skip("golden file created; rerun test to verify")
	}

	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if string(result.Content) != string(golden) {
		t.Fatalf("rendered content differs from golden.\nGot:\n%s\n\nWant:\n%s", string(result.Content), string(golden))
	}
}

func TestRendererSessionEntry_Golden(t *testing.T) {
	current, err := os.ReadFile(filepath.Join("..", "..", "testdata", "hq", "SESSION-LOG.md"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	payload, _ := json.Marshal(contract.SessionEntry{
		Timestamp: "2026-07-29T10:30:00+08:00",
		Tags:      []string{"test", "renderer"},
		Summary:   "Testing session entry renderer golden output",
	})

	req := contract.Request{
		SchemaVersion:      "1.0",
		RequestID:          "018f0000-0000-7000-8000-000000000002",
		Operation:          "session-entry",
		Target:             "SESSION-LOG.md",
		Payload:            payload,
		ExpectedTargetHash: fmt.Sprintf("%064x", sha256.Sum256(current)),
	}

	renderer := write.NewSessionEntryRenderer()
	result, err := renderer.Render(context.Background(), req, current)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if result.Path != req.Target {
		t.Fatalf("Path = %q, want %q", result.Path, req.Target)
	}

	if !strings.Contains(string(result.Content), "#test #renderer") {
		t.Fatal("expected tags in rendered output")
	}

	goldenPath := filepath.Join("..", "..", "testdata", "golden", "session-entry.md")
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Logf("golden file %s does not exist, creating from actual output", goldenPath)
		os.MkdirAll(filepath.Dir(goldenPath), 0700)
		os.WriteFile(goldenPath, result.Content, 0644)
		t.Skip("golden file created; rerun test to verify")
	}

	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if string(result.Content) != string(golden) {
		t.Fatalf("rendered content differs from golden.\nGot:\n%s\n\nWant:\n%s", string(result.Content), string(golden))
	}
}

func TestRendererDraftRecord_Golden(t *testing.T) {
	payload, _ := json.Marshal(contract.DraftRecord{
		Title:          "Test Draft Record",
		Body:           "This is the body of the draft record.",
		RecordDate:     "2026-07-29",
		Classification: "inbox",
	})

	req := contract.Request{
		SchemaVersion: "1.0",
		RequestID:     "018f0000-0000-7000-8000-000000000003",
		Operation:     "draft-record",
		Target:        "inbox/2026-07-29-test-draft.md",
		Payload:       payload,
		CreateOnly:    true,
	}

	renderer := write.NewDraftRecordRenderer()
	result, err := renderer.Render(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if result.CreateOnly != true {
		t.Fatal("draft-record renderer must set CreateOnly")
	}

	goldenPath := filepath.Join("..", "..", "testdata", "golden", "draft-record.md")
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Logf("golden file %s does not exist, creating from actual output", goldenPath)
		os.MkdirAll(filepath.Dir(goldenPath), 0700)
		os.WriteFile(goldenPath, result.Content, 0644)
		t.Skip("golden file created; rerun test to verify")
	}

	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if string(result.Content) != string(golden) {
		t.Fatalf("rendered content differs from golden.\nGot:\n%s\n\nWant:\n%s", string(result.Content), string(golden))
	}
}

func TestRendererCurrentWorkUpdate_Golden(t *testing.T) {
	current, err := os.ReadFile(filepath.Join("..", "..", "testdata", "hq", "CURRENT-WORK.md"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	payload, _ := json.Marshal(contract.CurrentWorkUpdate{
		WorkName:          "HQ CLI",
		Workspace:         "testdata/hq",
		LoadFirst:         "AGENTS.md",
		SupportingContext: []string{"docs/architecture.md", "docs/contracts/cli.md"},
		CurrentOutcome:    "Updated via renderer test",
		CurrentState:      "Golden test phase",
		NextAction:        "Verify golden output",
		LastTouched:       "2026-07-29",
		Section:           "Active",
	})

	req := contract.Request{
		SchemaVersion:      "1.0",
		RequestID:          "018f0000-0000-7000-8000-000000000004",
		Operation:          "current-work-update",
		Target:             "CURRENT-WORK.md",
		Payload:            payload,
		ExpectedTargetHash: fmt.Sprintf("%064x", sha256.Sum256(current)),
	}

	renderer := write.NewCurrentWorkUpdateRenderer()
	result, err := renderer.Render(context.Background(), req, current)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if result.Path != req.Target {
		t.Fatalf("Path = %q, want %q", result.Path, req.Target)
	}

	goldenPath := filepath.Join("..", "..", "testdata", "golden", "current-work-update.md")
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Logf("golden file %s does not exist, creating from actual output", goldenPath)
		os.MkdirAll(filepath.Dir(goldenPath), 0700)
		os.WriteFile(goldenPath, result.Content, 0644)
		t.Skip("golden file created; rerun test to verify")
	}

	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if string(result.Content) != string(golden) {
		t.Fatalf("rendered content differs from golden.\nGot:\n%s\n\nWant:\n%s", string(result.Content), string(golden))
	}
}

func TestRendererPreservesUnknownSections_ProjectCheckIn(t *testing.T) {
	original := []byte(`# Example Project State

## Executive Summary

Original summary.

## Current Outcome

Original outcome.

## Custom Section

This is an unknown section that must be preserved.

## Next Action

Original action.

## Open Decisions

- None.

## Blockers

- None.

## Material Risks

- None.

## Other Custom

More content.

## Evidence

- Original source, verified 2026-07-24.
`)

	payload, _ := json.Marshal(contract.ProjectCheckIn{
		Summary:        "New summary",
		CurrentOutcome: "New outcome",
		CurrentState:   "New state",
		NextAction:     "New action",
		Blockers:       []string{"None"},
		Risks:          []string{"None"},
		Evidence:       []string{"New source, verified 2026-07-29"},
		VerifiedAt:     "2026-07-29",
	})

	req := contract.Request{
		SchemaVersion:      "1.0",
		RequestID:          "018f0000-0000-7000-8000-000000000005",
		Operation:          "project-check-in",
		Target:             "projects/example/STATE.md",
		Payload:            payload,
		ExpectedTargetHash: fmt.Sprintf("%064x", sha256.Sum256(original)),
	}

	renderer := write.NewProjectCheckInRenderer()
	result, err := renderer.Render(context.Background(), req, original)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	content := string(result.Content)
	if !strings.Contains(content, "## Custom Section") {
		t.Fatal("unknown section 'Custom Section' not preserved")
	}
	if !strings.Contains(content, "## Other Custom") {
		t.Fatal("unknown section 'Other Custom' not preserved")
	}
	if !strings.Contains(content, "This is an unknown section that must be preserved.") {
		t.Fatal("content of unknown section not preserved")
	}
}

func TestRendererPreservesUnrelatedEntries_CurrentWorkUpdate(t *testing.T) {
	original := []byte(`# Current Work

## Active

### HQ CLI
- Workspace: testdata/hq
- Load first: AGENTS.md
- Supporting context: docs/
- Current outcome: Original
- Current state: Original
- Next action: Original
- Last touched: 2026-07-24

### Another Project
- Workspace: testdata/other
- Load first: README.md
- Supporting context: docs/
- Current outcome: Other
- Current state: Other
- Next action: Other
- Last touched: 2026-07-23

## Warm

### Old Project
- Workspace: testdata/old
- Load first: README.md
- Supporting context: docs/
- Current outcome: Old
- Current state: Old
- Next action: Old
- Last touched: 2026-07-20
`)

	payload, _ := json.Marshal(contract.CurrentWorkUpdate{
		WorkName:          "HQ CLI",
		Workspace:         "testdata/hq",
		LoadFirst:         "AGENTS.md",
		SupportingContext: []string{"docs/", "docs/contracts/"},
		CurrentOutcome:    "Updated outcome",
		CurrentState:      "Updated state",
		NextAction:        "Updated next action",
		LastTouched:       "2026-07-29",
		Section:           "Active",
	})

	req := contract.Request{
		SchemaVersion:      "1.0",
		RequestID:          "018f0000-0000-7000-8000-000000000006",
		Operation:          "current-work-update",
		Target:             "CURRENT-WORK.md",
		Payload:            payload,
		ExpectedTargetHash: fmt.Sprintf("%064x", sha256.Sum256(original)),
	}

	renderer := write.NewCurrentWorkUpdateRenderer()
	result, err := renderer.Render(context.Background(), req, original)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	content := string(result.Content)

	if !strings.Contains(content, "### Another Project") {
		t.Fatal("unrelated entry 'Another Project' not preserved")
	}
	if !strings.Contains(content, "### Old Project") {
		t.Fatal("unrelated entry 'Old Project' not preserved")
	}
	if !strings.Contains(content, "Updated outcome") {
		t.Fatal("HQ CLI entry not updated")
	}
	if !strings.Contains(content, "## Warm") {
		t.Fatal("Warm section not preserved")
	}
}

func TestRendererPreservesStateProjectSection(t *testing.T) {
	original := []byte(`# Example Project State

## Executive Summary

Test.

## Current Outcome

Test.

## Current State

Test.

## Next Action

Test.

## Open Decisions

- Decision one.

## Blockers

- None.

## Material Risks

- None.

## Evidence

- Source, verified 2026-07-24.
`)

	payload, _ := json.Marshal(contract.ProjectCheckIn{
		Summary:        "Updated",
		CurrentOutcome: "Updated",
		CurrentState:   "Updated",
		NextAction:     "Updated",
		Blockers:       []string{"None"},
		Risks:          []string{"None"},
		Evidence:       []string{"Updated, verified 2026-07-29"},
		VerifiedAt:     "2026-07-29",
	})

	req := contract.Request{
		SchemaVersion:      "1.0",
		Operation:          "project-check-in",
		Target:             "projects/example/STATE.md",
		Payload:            payload,
		ExpectedTargetHash: fmt.Sprintf("%064x", sha256.Sum256(original)),
	}

	renderer := write.NewProjectCheckInRenderer()
	result, err := renderer.Render(context.Background(), req, original)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	content := string(result.Content)
	if !strings.Contains(content, "- Decision one.") {
		t.Fatal("Open Decisions must be preserved")
	}
}

func TestRendererSessionEntry_RejectsEmbeddedNewlines(t *testing.T) {
	payload, _ := json.Marshal(contract.SessionEntry{
		Timestamp: "2026-07-29T10:30:00+08:00",
		Tags:      []string{"test"},
		Summary:   "line1\nline2",
	})

	req := contract.Request{
		SchemaVersion: "1.0",
		Operation:     "session-entry",
		Target:        "SESSION-LOG.md",
		Payload:       payload,
	}

	renderer := write.NewSessionEntryRenderer()
	_, err := renderer.Render(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected error for embedded newlines in summary")
	}
	if !strings.Contains(err.Error(), "newline") && !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRendererDraftRecord_RejectsExistingTarget(t *testing.T) {
	existing := []byte("# Existing file\n")
	payload, _ := json.Marshal(contract.DraftRecord{
		Title:          "Test",
		Body:           "Body",
		RecordDate:     "2026-07-29",
		Classification: "inbox",
	})

	req := contract.Request{
		SchemaVersion: "1.0",
		Operation:     "draft-record",
		Target:        "inbox/2026-07-29-test.md",
		Payload:       payload,
		CreateOnly:    true,
	}

	renderer := write.NewDraftRecordRenderer()
	_, err := renderer.Render(context.Background(), req, existing)
	if err == nil {
		t.Fatal("expected error for existing draft target")
	}
}

func TestRendererCurrentWorkUpdate_RejectsUnknownSection(t *testing.T) {
	original := []byte(`# Current Work

## Active

### Test
- Workspace: test
- Load first: test
- Supporting context: test
- Current outcome: test
- Current state: test
- Next action: test
- Last touched: 2026-07-29
`)

	payload, _ := json.Marshal(contract.CurrentWorkUpdate{
		WorkName:          "Test",
		Section:           "InvalidSection",
		Workspace:         "test",
		LoadFirst:         "test",
		SupportingContext: []string{},
		CurrentOutcome:    "test",
		CurrentState:      "test",
		NextAction:        "test",
		LastTouched:       "2026-07-29",
	})

	req := contract.Request{
		SchemaVersion:      "1.0",
		Operation:          "current-work-update",
		Target:             "CURRENT-WORK.md",
		Payload:            payload,
		ExpectedTargetHash: fmt.Sprintf("%064x", sha256.Sum256(original)),
	}

	renderer := write.NewCurrentWorkUpdateRenderer()
	_, err := renderer.Render(context.Background(), req, original)
	if err == nil {
		t.Fatal("expected error for invalid section")
	}
}
