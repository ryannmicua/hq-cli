package hq

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Record represents a single HQ markdown record from an allowlisted
// collection.
type Record struct {
	Collection string            `json:"collection"`
	ID         string            `json:"id"`
	Path       string            `json:"path"`      // relative to HQ root
	Content    string            `json:"content"`   // full file content
	SHA256     string            `json:"sha256"`    // hex-encoded SHA-256
	GitCommit  *string           `json:"gitCommit"` // null when unavailable
	Metadata   map[string]string `json:"metadata"`  // collection-specific
}

// CollectionQuery specifies which records to list and optional filters.
type CollectionQuery struct {
	Collection string
	Filter     map[string]string // e.g. {"section": "active"}
	Limit      int               // 0 means no limit
}

// GetSelector identifies a single record by collection+ID or by
// allowlisted relative path.
type GetSelector struct {
	Collection string // empty when using Path
	ID         string // empty when using Path
	Path       string // allowlisted relative path
}

// CollectionAllowlist defines which collection names are valid.
var CollectionAllowlist = map[string]bool{
	"current-work": true,
	"projects":     true,
	"decisions":    true,
	"work-types":   true,
	"people":       true,
	"references":   true,
}

// collectionDirs maps collection names to their directory roots relative
// to the HQ root.
var collectionDirs = map[string]string{
	"current-work": ".",
	"projects":     "projects",
	"decisions":    "decisions",
	"work-types":   "work-types",
	"people":       "people",
	"references":   "references",
}

// AllowlistedPaths are explicit file paths (relative to HQ root) that
// are retrievable via get --path.
var AllowlistedPaths = map[string]bool{
	"CURRENT-WORK.md":      true,
	"SESSION-LOG.md":       true,
	"AGENTS.md":            true,
	"safety-boundaries.md": true,
}

// IsAllowlistedPath checks if a relative path is explicitly allowlisted.
func IsAllowlistedPath(path string) bool {
	normalized := filepath.ToSlash(path)
	return AllowlistedPaths[normalized]
}

// Get retrieves a single record by collection+ID or allowlisted path.
// It expects a valid Resolver for path containment checks.
func Get(ctx context.Context, resolver *Resolver, sel GetSelector) (*Record, error) {
	var targetPath string

	if sel.Path != "" {
		// Resolve by explicit allowlisted path.
		normalized := filepath.ToSlash(sel.Path)
		if !IsAllowlistedPath(normalized) {
			// Check if it matches a collection path.
			isCollection := false
			for _, dir := range collectionDirs {
				if strings.HasPrefix(normalized, filepath.ToSlash(dir)) {
					isCollection = true
					break
				}
			}
			if !isCollection {
				return nil, fmt.Errorf("path %q is not allowlisted", sel.Path)
			}
		}
		targetPath = sel.Path
	} else {
		// Resolve by collection + ID
		if !CollectionAllowlist[sel.Collection] {
			return nil, fmt.Errorf("unknown collection %q", sel.Collection)
		}
		p, err := collectionPath(sel.Collection, sel.ID)
		if err != nil {
			return nil, err
		}
		targetPath = p
	}

	// Resolve through the path resolver.
	absPath, err := resolver.Resolve(targetPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	// Read the file.
	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, notFoundErr(targetPath)
		}
		return nil, fmt.Errorf("read file %q: %w", absPath, err)
	}

	rec := buildRecord(sel.Collection, sel.ID, targetPath, string(content), resolver.Root())
	if sel.Path != "" && rec.ID == "" {
		// Derive collection and ID from path.
		rec.Collection, rec.ID = deriveFromPath(targetPath)
	}
	return rec, nil
}

// List enumerates records from an allowed collection with optional
// filters. Results are sorted deterministically by normalized path.
func List(ctx context.Context, resolver *Resolver, query CollectionQuery) ([]*Record, error) {
	if !CollectionAllowlist[query.Collection] {
		return nil, fmt.Errorf("unknown collection %q", query.Collection)
	}

	dir, ok := collectionDirs[query.Collection]
	if !ok {
		return nil, fmt.Errorf("no directory mapping for collection %q", query.Collection)
	}

	absDir := filepath.Join(resolver.Root(), dir)

	var records []*Record

	switch query.Collection {
	case "current-work":
		recs, err := listCurrentWork(ctx, resolver, absDir, query)
		if err != nil {
			return nil, err
		}
		records = recs
	default:
		recs, err := listMarkdownFiles(ctx, resolver, absDir, query.Collection, dir, query)
		if err != nil {
			return nil, err
		}
		records = recs
	}

	// Apply filters.
	records = applyFilters(records, query.Filter)

	// Sort by normalized path deterministically.
	sort.Slice(records, func(i, j int) bool {
		return filepath.ToSlash(records[i].Path) < filepath.ToSlash(records[j].Path)
	})

	// Apply limit.
	if query.Limit > 0 && len(records) > query.Limit {
		records = records[:query.Limit]
	}

	return records, nil
}

// ============================================================
// Internal helpers
// ============================================================

func listCurrentWork(ctx context.Context, resolver *Resolver, absDir string, query CollectionQuery) ([]*Record, error) {
	// CURRENT-WORK.md is a single file at the root.
	cwPath := filepath.Join(absDir, "CURRENT-WORK.md")
	content, err := os.ReadFile(cwPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read CURRENT-WORK.md: %w", err)
	}

	rec := buildRecord("current-work", "current", "CURRENT-WORK.md", string(content), resolver.Root())

	// Parse sections (Active / Warm) into metadata.
	sections := parseSections(string(content))
	if sections != nil {
		rec.Metadata = sections
	}

	return []*Record{rec}, nil
}

func listMarkdownFiles(ctx context.Context, resolver *Resolver, absDir, collection, relDir string, query CollectionQuery) ([]*Record, error) {
	var records []*Record

	err := filepath.WalkDir(absDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip .git, .hq-interface, hidden dirs, and templates.
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "templates" {
				return filepath.SkipDir
			}
			// Skip parent directory itself.
			if path == absDir {
				return nil
			}
			return nil
		}

		// Only .md files.
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		// Read content.
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Determine relative path.
		relPath, err := filepath.Rel(resolver.Root(), path)
		if err != nil {
			return err
		}

		// Determine ID.
		id := determineID(relPath, collection)

		rec := buildRecord(collection, id, filepath.ToSlash(relPath), string(content), resolver.Root())
		records = append(records, rec)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk %q: %w", absDir, err)
	}

	return records, nil
}

func collectionPath(collection, id string) (string, error) {
	switch collection {
	case "current-work":
		if id != "current" {
			return "", fmt.Errorf("current-work ID must be 'current', got %q", id)
		}
		return "CURRENT-WORK.md", nil
	case "projects":
		return fmt.Sprintf("projects/%s/STATE.md", id), nil
	case "decisions":
		return fmt.Sprintf("decisions/%s.md", id), nil
	case "work-types":
		// ID is a normalized relative path without .md
		return fmt.Sprintf("work-types/%s.md", id), nil
	case "people":
		return fmt.Sprintf("people/%s.md", id), nil
	case "references":
		return fmt.Sprintf("references/%s.md", id), nil
	default:
		return "", fmt.Errorf("unknown collection %q", collection)
	}
}

func buildRecord(collection, id, relPath, content string, root string) *Record {
	hash := sha256.Sum256([]byte(content))
	hexHash := fmt.Sprintf("%x", hash)

	var gitCommit *string
	if gc := getGitCommit(root, relPath); gc != "" {
		gitCommit = &gc
	}

	metadata := make(map[string]string)

	// Add collection-specific metadata.
	switch collection {
	case "projects":
		if strings.HasSuffix(relPath, "STATE.md") {
			metadata["recordType"] = "project-state"
		} else if strings.HasSuffix(relPath, "README.md") {
			metadata["recordType"] = "project-readme"
		}
		// Extract slug from path.
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		if len(parts) >= 2 && collection == "projects" {
			metadata["name"] = parts[1] // the slug
		}
	case "people":
		name := strings.TrimSuffix(filepath.Base(relPath), ".md")
		metadata["name"] = name
	case "decisions":
		name := strings.TrimSuffix(filepath.Base(relPath), ".md")
		metadata["title"] = name
	}

	return &Record{
		Collection: collection,
		ID:         id,
		Path:       filepath.ToSlash(relPath),
		Content:    content,
		SHA256:     hexHash,
		GitCommit:  gitCommit,
		Metadata:   metadata,
	}
}

func getGitCommit(root, relPath string) string {
	absPath := filepath.Join(root, relPath)
	cmd := exec.Command("git", "log", "-1", "--format=%H", "--", absPath)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func determineID(relPath, collection string) string {
	normalized := filepath.ToSlash(relPath)
	switch collection {
	case "projects":
		parts := strings.SplitN(normalized, "/", 3)
		if len(parts) >= 2 {
			return parts[1]
		}
	case "decisions":
		base := filepath.Base(normalized)
		return strings.TrimSuffix(base, ".md")
	case "work-types":
		// Remove the "work-types/" prefix and ".md" suffix.
		trimmed := strings.TrimPrefix(normalized, "work-types/")
		trimmed = strings.TrimSuffix(trimmed, ".md")
		return strings.TrimSuffix(trimmed, "/example-work")
	case "people":
		base := filepath.Base(normalized)
		return strings.TrimSuffix(base, ".md")
	case "references":
		trimmed := strings.TrimPrefix(normalized, "references/")
		return strings.TrimSuffix(trimmed, ".md")
	}
	return ""
}

func deriveFromPath(relPath string) (string, string) {
	normalized := filepath.ToSlash(relPath)
	parts := strings.SplitN(normalized, "/", 2)
	if len(parts) < 2 {
		return "", ""
	}

	// Map directory name to collection.
	dirMap := map[string]string{
		"projects":   "projects",
		"decisions":  "decisions",
		"work-types": "work-types",
		"people":     "people",
		"references": "references",
	}

	collection, ok := dirMap[parts[0]]
	if !ok {
		return "", ""
	}

	id := determineID(relPath, collection)
	return collection, id
}

func notFoundErr(path string) error {
	return fmt.Errorf("record not found: %s", path)
}

func parseSections(content string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")
	currentSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			currentSection = strings.TrimSpace(strings.TrimPrefix(trimmed, "##"))
		}
		// Detect entries within a section.
		if strings.HasPrefix(trimmed, "### ") && currentSection != "" {
			entry := strings.TrimSpace(strings.TrimPrefix(trimmed, "###"))
			result[currentSection] = entry
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func applyFilters(records []*Record, filter map[string]string) []*Record {
	if len(filter) == 0 {
		return records
	}

	var filtered []*Record
	for _, rec := range records {
		match := true
		for k, v := range filter {
			switch k {
			case "section":
				// Match sections from CURRENT-WORK metadata.
				if rec.Metadata != nil {
					hasSection := false
					for section, entry := range rec.Metadata {
						if strings.EqualFold(section, v) && entry != "" {
							hasSection = true
							break
						}
					}
					if !hasSection {
						match = false
					}
				} else {
					match = false
				}
			case "name":
				if name, ok := rec.Metadata["name"]; !ok || !strings.Contains(strings.ToLower(name), strings.ToLower(v)) {
					match = false
				}
			case "id":
				if !strings.HasPrefix(strings.ToLower(rec.ID), strings.ToLower(v)) {
					match = false
				}
			}
		}
		if match {
			filtered = append(filtered, rec)
		}
	}
	return filtered
}
