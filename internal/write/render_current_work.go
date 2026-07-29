package write

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/ryannmicua/hq-cli/internal/contract"
)

type CurrentWorkUpdateRenderer struct{}

func NewCurrentWorkUpdateRenderer() *CurrentWorkUpdateRenderer {
	return &CurrentWorkUpdateRenderer{}
}

var validSections = map[string]bool{
	"Active": true,
	"Warm":   true,
}

func (r *CurrentWorkUpdateRenderer) Render(ctx context.Context, req contract.Request, current []byte) (RenderedTarget, error) {
	var update contract.CurrentWorkUpdate
	if err := contract.DecodeStrict(req.Payload, &update); err != nil {
		return RenderedTarget{}, fmt.Errorf("unmarshal current-work-update payload: %w", err)
	}

	if update.WorkName == "" {
		return RenderedTarget{}, fmt.Errorf("current-work-update: workName is required")
	}

	if update.Section != "" && !validSections[update.Section] {
		return RenderedTarget{}, fmt.Errorf("current-work-update: invalid section %q, must be Active or Warm", update.Section)
	}

	if current == nil {
		return RenderedTarget{}, fmt.Errorf("current-work-update: current content is required")
	}

	content := string(current)
	lines := strings.Split(content, "\n")

	entryLineIdx := -1
	entrySection := ""

	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			sec := strings.TrimPrefix(line, "## ")
			if validSections[sec] {
				entrySection = sec
			}
		}

		if strings.HasPrefix(line, "### ") {
			name := strings.TrimPrefix(line, "### ")
			if name == update.WorkName {
				entryLineIdx = i
				if update.Section != "" && update.Section != entrySection {
					return RenderedTarget{}, fmt.Errorf("current-work-update: entry %q found in section %q, but request specifies section %q", update.WorkName, entrySection, update.Section)
				}
				break
			}
		}
	}

	if entryLineIdx == -1 {
		return r.createEntry(req.Target, content, update)
	}

	entryEndIdx := findEntryFieldsEnd(lines, entryLineIdx+1)

	var result strings.Builder
	for i := 0; i < len(lines); i++ {
		if i == entryLineIdx {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(fmt.Sprintf("### %s\n", update.WorkName))
			writeCWField(&result, "Workspace", update.Workspace)
			writeCWField(&result, "Load first", update.LoadFirst)
			writeCWField(&result, "Supporting context", formatSupportingContext(update.SupportingContext))
			writeCWField(&result, "Current outcome", update.CurrentOutcome)
			writeCWField(&result, "Current state", update.CurrentState)
			writeCWField(&result, "Next action", update.NextAction)
			writeCWField(&result, "Last touched", update.LastTouched)
			i = entryEndIdx - 1
			continue
		}

		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(lines[i])
	}

	out := []byte(result.String())
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(out))
	return RenderedTarget{
		Path:    req.Target,
		Content: out,
		SHA256:  hash,
	}, nil
}

func findEntryFieldsEnd(lines []string, start int) int {
	fieldsFound := 0
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			if fieldsFound > 0 {
				return i + 1
			}
			continue
		}
		if strings.HasPrefix(lines[i], "### ") || strings.HasPrefix(lines[i], "## ") {
			return i
		}
		if strings.HasPrefix(trimmed, "- ") {
			fieldsFound++
		}
	}
	return len(lines)
}

func writeCWField(b *strings.Builder, name, value string) {
	b.WriteString(fmt.Sprintf("- %s: %s\n", name, value))
}

func formatSupportingContext(ctx []string) string {
	if len(ctx) == 0 {
		return ""
	}
	return strings.Join(ctx, ", ")
}

func (r *CurrentWorkUpdateRenderer) createEntry(target, content string, update contract.CurrentWorkUpdate) (RenderedTarget, error) {
	if update.Section == "" {
		return RenderedTarget{}, fmt.Errorf("current-work-update: entry %q not found and no section provided for creation", update.WorkName)
	}
	if update.Workspace == "" || update.LoadFirst == "" || update.LastTouched == "" {
		return RenderedTarget{}, fmt.Errorf("current-work-update: cannot create entry %q: workspace, loadFirst, and lastTouched are required", update.WorkName)
	}

	lines := strings.Split(content, "\n")
	var result strings.Builder
	foundSection := false
	inserted := false

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(line)

		if !foundSection && strings.HasPrefix(line, "## "+update.Section) {
			foundSection = true
			continue
		}

		if foundSection && !inserted {
			nextIsEntry := i+1 < len(lines) && (strings.HasPrefix(lines[i+1], "### ") || strings.HasPrefix(lines[i+1], "## "))
			atEnd := i+1 >= len(lines)
			if nextIsEntry || atEnd {
				result.WriteString("\n")
				result.WriteString(formatNewEntry(update))
				inserted = true
			}
		}
	}

	if !foundSection {
		return RenderedTarget{}, fmt.Errorf("current-work-update: section %q not found", update.Section)
	}

	if !inserted {
		result.WriteString("\n")
		result.WriteString(formatNewEntry(update))
	}

	out := []byte(result.String())
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(out))
	return RenderedTarget{
		Path:    target,
		Content: out,
		SHA256:  hash,
	}, nil
}

func formatNewEntry(update contract.CurrentWorkUpdate) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("### %s\n", update.WorkName))
	writeCWField(&b, "Workspace", update.Workspace)
	writeCWField(&b, "Load first", update.LoadFirst)
	writeCWField(&b, "Supporting context", formatSupportingContext(update.SupportingContext))
	writeCWField(&b, "Current outcome", update.CurrentOutcome)
	writeCWField(&b, "Current state", update.CurrentState)
	writeCWField(&b, "Next action", update.NextAction)
	writeCWField(&b, "Last touched", update.LastTouched)
	return b.String()
}
