package app

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/hq"
	"github.com/ryannmicua/hq-cli/internal/read"
)

func handleVersion(ctx context.Context, args []string, svc *read.Service, stdout, stderr io.Writer) int {
	v := svc.Version()
	r := contract.NewSuccess("version", v)
	return writeResult(stdout, r)
}

func handleHealth(ctx context.Context, args []string, svc *read.Service, stdout, stderr io.Writer) int {
	h := svc.Health()
	r := contract.NewSuccess("health", h)
	return writeResult(stdout, r)
}

func handleContext(ctx context.Context, args []string, svc *read.Service, stdout, stderr io.Writer) int {
	flags, _ := parseFlags(args)

	opts := read.ContextOpts{
		SessionCount: 20,
	}
	if p, ok := flags["project"]; ok {
		opts.Project = p
	}
	if n, ok := flags["session-entries"]; ok {
		if v, err := strconv.Atoi(n); err == nil && v > 0 {
			opts.SessionCount = v
		}
	}

	// Call the lowercased context method via a wrapped approach.
	// For Phase 0, we implement this inline using the read service's components.
	result := buildContextResult(ctx, svc, opts)
	if result == nil {
		r := contract.NewError("context", contract.ErrDetail(contract.CodeNotFound, "no active project found", false, nil))
		return writeResult(stdout, r)
	}

	r := contract.NewSuccess("context", result)
	return writeResult(stdout, r)
}

func handleGet(ctx context.Context, args []string, svc *read.Service, stdout, stderr io.Writer) int {
	flags, _ := parseFlags(args)

	collection := flags["collection"]
	id := flags["id"]
	path := flags["path"]

	if collection != "" && id != "" {
		// Get by collection + ID.
		sel := hq.GetSelector{Collection: collection, ID: id}
		rec, err := hq.Get(ctx, svc.Resolver(), sel)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				r := contract.NewError("get", contract.ErrDetail(contract.CodeNotFound, err.Error(), false, nil))
				return writeResult(stdout, r)
			}
			if strings.Contains(err.Error(), "unknown collection") {
				r := contract.NewError("get", contract.ErrDetail(contract.CodeInvalidArgument, err.Error(), false, nil))
				return writeResult(stdout, r)
			}
			details := map[string]any{"collection": collection, "id": id}
			r := contract.NewError("get", contract.ErrDetail(contract.CodeInternalError, err.Error(), false, details))
			return writeResult(stdout, r)
		}
		r := contract.NewSuccess("get", rec)
		return writeResult(stdout, r)
	}

	if path != "" {
		sel := hq.GetSelector{Path: path}
		rec, err := hq.Get(ctx, svc.Resolver(), sel)
		if err != nil {
			if strings.Contains(err.Error(), "not allowlisted") || strings.Contains(err.Error(), "HQ_PATH_DENIED") {
				r := contract.NewError("get", contract.ErrDetail(contract.CodePathDenied, err.Error(), false, nil))
				return writeResult(stdout, r)
			}
			if strings.Contains(err.Error(), "not found") {
				r := contract.NewError("get", contract.ErrDetail(contract.CodeNotFound, err.Error(), false, nil))
				return writeResult(stdout, r)
			}
			r := contract.NewError("get", contract.ErrDetail(contract.CodeInternalError, err.Error(), false, nil))
			return writeResult(stdout, r)
		}
		r := contract.NewSuccess("get", rec)
		return writeResult(stdout, r)
	}

	r := contract.NewError("get", contract.ErrDetail(contract.CodeInvalidArgument, "get requires --collection <name> --id <id> or --path <relative-path>", false, nil))
	return writeResult(stdout, r)
}

func handleList(ctx context.Context, args []string, svc *read.Service, stdout, stderr io.Writer) int {
	flags, _ := parseFlags(args)

	collection, ok := flags["collection"]
	if !ok {
		r := contract.NewError("list", contract.ErrDetail(contract.CodeInvalidArgument, "list requires --collection <name>", false, nil))
		return writeResult(stdout, r)
	}

	query := hq.CollectionQuery{
		Collection: collection,
		Filter:     make(map[string]string),
	}

	if filter, ok := flags["filter"]; ok {
		parts := strings.SplitN(filter, "=", 2)
		if len(parts) == 2 {
			query.Filter[parts[0]] = parts[1]
		}
	}
	if limit, ok := flags["limit"]; ok {
		if v, err := strconv.Atoi(limit); err == nil && v > 0 {
			query.Limit = v
		}
	}

	recs, err := hq.List(ctx, svc.Resolver(), query)
	if err != nil {
		if strings.Contains(err.Error(), "unknown collection") {
			r := contract.NewError("list", contract.ErrDetail(contract.CodeInvalidArgument, err.Error(), false, nil))
			return writeResult(stdout, r)
		}
		r := contract.NewError("list", contract.ErrDetail(contract.CodeInternalError, err.Error(), false, nil))
		return writeResult(stdout, r)
	}

	// Build list result items.
	items := make([]map[string]any, 0, len(recs))
	for _, rec := range recs {
		item := map[string]any{
			"id":       rec.ID,
			"path":     rec.Path,
			"metadata": rec.Metadata,
		}
		items = append(items, item)
	}

	data := map[string]any{
		"collection": collection,
		"items":      items,
		"count":      len(items),
	}

	r := contract.NewSuccess("list", data)
	return writeResult(stdout, r)
}

func handleSearch(ctx context.Context, args []string, svc *read.Service, stdout, stderr io.Writer) int {
	flags, _ := parseFlags(args)

	query, ok := flags["query"]
	if !ok {
		r := contract.NewError("search", contract.ErrDetail(contract.CodeInvalidArgument, "search requires --query <text>", false, nil))
		return writeResult(stdout, r)
	}

	searchQuery := read.SearchQuery{
		Query: query,
	}
	if c, ok := flags["collection"]; ok {
		searchQuery.Collection = c
	}
	if p, ok := flags["path"]; ok {
		searchQuery.Path = p
	}
	if v, ok := flags["case-sensitive"]; ok {
		searchQuery.CaseSensitive = v == "true" || v == ""
	}
	if limit, ok := flags["limit"]; ok {
		if v, err := strconv.Atoi(limit); err == nil && v > 0 {
			searchQuery.Limit = v
		}
	}

	result, err := svc.Search(ctx, searchQuery)
	if err != nil {
		code := contract.CodeInternalError
		if strings.Contains(err.Error(), "empty") {
			code = contract.CodeInvalidArgument
		}
		r := contract.NewError("search", contract.ErrDetail(code, err.Error(), false, nil))
		return writeResult(stdout, r)
	}

	r := contract.NewSuccess("search", result)
	return writeResult(stdout, r)
}

func buildContextResult(ctx context.Context, svc *read.Service, opts read.ContextOpts) any {
	// Simplified context builder for Phase 0.
	// Read current work.
	currentWork, err := hq.Get(ctx, svc.Resolver(), hq.GetSelector{Collection: "current-work", ID: "current"})
	if err != nil {
		return nil
	}

	// Extract active work name.
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

	// Read project state.
	stateRec, _ := hq.Get(ctx, svc.Resolver(), hq.GetSelector{Collection: "projects", ID: projectSlug})
	readmeRec, _ := hq.Get(ctx, svc.Resolver(), hq.GetSelector{Path: fmt.Sprintf("projects/%s/README.md", projectSlug)})

	projectCtx := map[string]any{
		"slug": projectSlug,
	}
	if stateRec != nil {
		projectCtx["state"] = stateRec
	}
	if readmeRec != nil {
		projectCtx["readme"] = readmeRec
	}

	// Parse session entries from SESSION-LOG.md.
	var sessionEntries []map[string]any
	if sessionRec, err := hq.Get(ctx, svc.Resolver(), hq.GetSelector{Path: "SESSION-LOG.md"}); err == nil {
		sessionEntries = parseSessionLogEntries(sessionRec.Content, opts.SessionCount)
	}

	// Operating rules.
	operatingRules := []string{}
	if _, err := svc.Resolver().Resolve("AGENTS.md"); err == nil {
		operatingRules = append(operatingRules, "AGENTS.md")
	}
	if _, err := svc.Resolver().Resolve("safety-boundaries.md"); err == nil {
		operatingRules = append(operatingRules, "safety-boundaries.md")
	}

	result := map[string]any{
		"selectedWork":   selectedWork,
		"currentWork":    currentWork,
		"project":        projectCtx,
		"sessionEntries": sessionEntries,
		"operatingRules": operatingRules,
	}

	return result
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

		// Parse entry: "YYYY-MM-DD HH:MM #tag1 #tag2 summary text"
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
	// Expected format: "YYYY-MM-DD HH:MM #tag1 #tag2 summary"
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

func writeResult(stdout io.Writer, r contract.Result) int {
	if err := contract.WriteJSON(stdout, r); err != nil {
		return 70
	}
	if r.Error != nil {
		return contract.ExitCode(r.Error.Code)
	}
	return 0
}
