package read

import (
	"context"
	"fmt"
	"strings"

	"github.com/ryannmicua/hq-cli/internal/hq"
)

func (s *Service) BuildContextResult(ctx context.Context, opts ContextOpts) any {
	currentWork, err := hq.Get(ctx, s.resolver, hq.GetSelector{Collection: "current-work", ID: "current"})
	if err != nil {
		return nil
	}

	selectedWork := ""
	projectSlug := opts.Project

	lines := strings.Split(currentWork.Content, "\n")
	inActive := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "## Active" {
			inActive = true
			continue
		}
		if inActive && strings.HasPrefix(trimmed, "## ") {
			break
		}
		if inActive && strings.HasPrefix(trimmed, "### ") {
			if selectedWork == "" {
				selectedWork = strings.TrimSpace(strings.TrimPrefix(trimmed, "###"))
			}
			continue
		}
		if inActive && projectSlug == "" && strings.HasPrefix(trimmed, "- Workspace:") {
			ws := strings.TrimSpace(strings.TrimPrefix(trimmed, "- Workspace:"))
			ws = strings.Trim(ws, "`")
			parts := strings.Split(ws, "/")
			projectSlug = parts[len(parts)-1]
		}
	}

	if projectSlug == "" {
		return nil
	}

	stateRec, _ := hq.Get(ctx, s.resolver, hq.GetSelector{Collection: "projects", ID: projectSlug})
	readmeRec, _ := hq.Get(ctx, s.resolver, hq.GetSelector{Path: fmt.Sprintf("projects/%s/README.md", projectSlug)})

	projectCtx := map[string]any{
		"slug": projectSlug,
	}
	if stateRec != nil {
		projectCtx["state"] = stateRec
	}
	if readmeRec != nil {
		projectCtx["readme"] = readmeRec
	}

	var sessionEntries []map[string]any
	if sessionRec, err := hq.Get(ctx, s.resolver, hq.GetSelector{Path: "SESSION-LOG.md"}); err == nil {
		sessionEntries = parseSessionLogEntries(sessionRec.Content, opts.SessionCount)
	}

	operatingRules := []string{}
	if _, err := s.resolver.Resolve("AGENTS.md"); err == nil {
		operatingRules = append(operatingRules, "AGENTS.md")
	}
	if _, err := s.resolver.Resolve("safety-boundaries.md"); err == nil {
		operatingRules = append(operatingRules, "safety-boundaries.md")
	}

	return map[string]any{
		"selectedWork":   selectedWork,
		"currentWork":    currentWork,
		"project":        projectCtx,
		"sessionEntries": sessionEntries,
		"operatingRules": operatingRules,
	}
}

func parseSessionLogEntries(content string, max int) []map[string]any {
	var entries []map[string]any
	lines := strings.Split(content, "\n")
	afterSeparator := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			afterSeparator = true
			continue
		}
		if !afterSeparator || trimmed == "" {
			continue
		}

		entry := parseSessionLine(trimmed)
		if entry != nil {
			entries = append(entries, entry)
			if len(entries) >= max {
				break
			}
		}
	}
	return entries
}

func parseSessionLine(line string) map[string]any {
	if len(line) < 16 {
		return nil
	}
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return nil
	}
	datePart := parts[0]
	timePart := parts[1]
	rest := parts[2]

	timestamp := datePart + " " + timePart

	var tags []string
	summary := rest
	for strings.HasPrefix(strings.TrimSpace(summary), "#") {
		parts := strings.SplitN(strings.TrimSpace(summary), " ", 2)
		if len(parts) == 2 {
			tag := strings.TrimPrefix(parts[0], "#")
			tags = append(tags, tag)
			summary = parts[1]
		} else {
			tag := strings.TrimPrefix(parts[0], "#")
			tags = append(tags, tag)
			summary = ""
			break
		}
	}

	return map[string]any{
		"timestamp": timestamp,
		"tags":      tags,
		"summary":   strings.TrimSpace(summary),
	}
}
