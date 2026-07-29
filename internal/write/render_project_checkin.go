package write

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/ryannmicua/hq-cli/internal/contract"
)

type ProjectCheckInRenderer struct{}

func NewProjectCheckInRenderer() *ProjectCheckInRenderer {
	return &ProjectCheckInRenderer{}
}

var ownedSections = map[string]bool{
	"Executive Summary": true,
	"Current Outcome":   true,
	"Current State":     true,
	"Next Action":       true,
	"Blockers":          true,
	"Material Risks":    true,
	"Evidence":          true,
}

func (r *ProjectCheckInRenderer) Render(ctx context.Context, req contract.Request, current []byte) (RenderedTarget, error) {
	var checkIn contract.ProjectCheckIn
	if err := contract.DecodeStrict(req.Payload, &checkIn); err != nil {
		return RenderedTarget{}, fmt.Errorf("unmarshal project-check-in payload: %w", err)
	}

	content := string(current)
	sections := splitAtSections(content)
	if len(sections) == 0 {
		return RenderedTarget{}, fmt.Errorf("cannot parse STATE.md: no sections found")
	}

	header := sections[0]
	body := sections[1:]

	ownedContent := map[string]string{
		"Executive Summary": checkIn.Summary,
		"Current Outcome":   checkIn.CurrentOutcome,
		"Current State":     checkIn.CurrentState,
		"Next Action":       checkIn.NextAction,
		"Blockers":          formatBulletList(checkIn.Blockers),
		"Material Risks":    formatBulletList(checkIn.Risks),
		"Evidence":          formatBulletList(checkIn.Evidence),
	}

	var result strings.Builder
	result.WriteString(strings.TrimRight(header, "\n"))
	result.WriteString("\n\n")

	for i, sec := range body {
		if i > 0 {
			result.WriteString("\n")
		}
		name := extractSectionName(sec)
		if ownedSections[name] {
			result.WriteString(fmt.Sprintf("## %s\n\n%s\n", name, ownedContent[name]))
		} else {
			result.WriteString(sec)
			result.WriteString("\n")
		}
	}

	out := []byte(result.String())
	hash := fmt.Sprintf("%x", sha256.Sum256(out))

	return RenderedTarget{
		Path:    req.Target,
		Content: out,
		SHA256:  hash,
	}, nil
}

func splitAtSections(content string) []string {
	lines := strings.Split(content, "\n")
	var result []string
	var current strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") && current.Len() > 0 {
			result = append(result, strings.TrimRight(current.String(), "\n"))
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(line)
	}

	if current.Len() > 0 {
		result = append(result, strings.TrimRight(current.String(), "\n"))
	}

	return result
}

func extractSectionName(section string) string {
	lines := strings.SplitN(section, "\n", 2)
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimPrefix(lines[0], "## ")
}

func formatBulletList(items []string) string {
	if len(items) == 0 {
		return "- None."
	}
	var b strings.Builder
	for i, item := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("- ")
		b.WriteString(item)
	}
	return b.String()
}
