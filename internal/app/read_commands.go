package app

import (
	"context"
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
	result := svc.BuildContextResult(ctx, opts)
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



func writeResult(stdout io.Writer, r contract.Result) int {
	if err := contract.WriteJSON(stdout, r); err != nil {
		return 70
	}
	if r.Error != nil {
		return contract.ExitCode(r.Error.Code)
	}
	return 0
}
