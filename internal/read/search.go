package read

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ryannmicua/hq-cli/internal/hq"
)

// SearchQuery represents the search command's parameters.
type SearchQuery struct {
	Query         string
	Collection    string // optional: restrict to a single collection
	Path          string // optional: restrict to a relative root
	CaseSensitive bool
	Limit         int // 0 = no limit
}

// SearchMatch represents one matching line in a search result.
type SearchMatch struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Text   string `json:"text"`
}

// SearchResult represents the search command data payload.
type SearchResult struct {
	Query     string        `json:"query"`
	Matches   []SearchMatch `json:"matches"`
	Count     int           `json:"count"`
	Truncated bool          `json:"truncated"`
}

// Search performs literal UTF-8 substring matching across allowlisted
// HQ files. Results are sorted by normalized path, line, then column.
func (s *Service) Search(ctx context.Context, query SearchQuery) (*SearchResult, error) {
	if query.Query == "" {
		return nil, fmt.Errorf("search query is empty")
	}

	// Collect files to search.
	files, err := s.collectSearchFiles(ctx, query)
	if err != nil {
		return nil, err
	}

	// Perform the search.
	var matches []SearchMatch
	for _, file := range files {
		fileMatches, err := searchFile(file, query)
		if err != nil {
			continue // skip unreadable files
		}
		relPath, err := filepath.Rel(s.config.Root, file)
		if err == nil {
			for i := range fileMatches {
				fileMatches[i].Path = relPath
			}
		}
		matches = append(matches, fileMatches...)
	}

	// Sort by path, line, column.
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Path != matches[j].Path {
			return matches[i].Path < matches[j].Path
		}
		if matches[i].Line != matches[j].Line {
			return matches[i].Line < matches[j].Line
		}
		return matches[i].Column < matches[j].Column
	})

	// Apply limit.
	truncated := false
	if query.Limit > 0 && len(matches) > query.Limit {
		matches = matches[:query.Limit]
		truncated = true
	}

	return &SearchResult{
		Query:     query.Query,
		Matches:   matches,
		Count:     len(matches),
		Truncated: truncated,
	}, nil
}

func (s *Service) collectSearchFiles(ctx context.Context, query SearchQuery) ([]string, error) {
	if query.Collection != "" {
		// Search within a single collection.
		recs, err := hq.List(ctx, s.resolver, hq.CollectionQuery{
			Collection: query.Collection,
		})
		if err != nil {
			return nil, err
		}
		var files []string
		for _, rec := range recs {
			abs, err := s.resolver.Resolve(rec.Path)
			if err != nil {
				continue
			}
			files = append(files, abs)
		}
		return files, nil
	}

	// Walk allowlisted directories.
	return s.walkAllowedFiles(ctx, query.Path)
}

func (s *Service) walkAllowedFiles(ctx context.Context, pathFilter string) ([]string, error) {
	searchRoot := s.config.Root
	if pathFilter != "" {
		resolved, err := s.resolver.Resolve(pathFilter)
		if err != nil {
			return nil, fmt.Errorf("resolve search path: %w", err)
		}
		searchRoot = resolved
	}

	// Directories to walk for full search.
	allowedDirs := []string{
		".",
		"projects",
		"decisions",
		"work-types",
		"people",
		"references",
	}

	var allFiles []string

	// Also check explicit allowlisted root files.
	for _, fp := range []string{"AGENTS.md", "CURRENT-WORK.md", "SESSION-LOG.md", "safety-boundaries.md"} {
		fullPath := filepath.Join(s.config.Root, fp)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			// Only include if it's within the path filter.
			if pathFilter == "" || strings.HasPrefix(fullPath, searchRoot) {
				allFiles = append(allFiles, fullPath)
			}
		}
	}

	for _, dir := range allowedDirs {
		absDir := filepath.Join(s.config.Root, dir)
		if pathFilter != "" && !strings.HasPrefix(absDir, searchRoot) {
			continue
		}

		_ = filepath.WalkDir(absDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // skip unreadable
			}
			if d.IsDir() {
				name := d.Name()
				if strings.HasPrefix(name, ".") || name == "templates" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				return nil
			}
			allFiles = append(allFiles, path)
			return nil
		})
	}

	return uniqueFiles(allFiles), nil
}

func searchFile(absPath string, query SearchQuery) ([]SearchMatch, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	q := query.Query
	if !query.CaseSensitive {
		q = strings.ToLower(q)
	}

	var matches []SearchMatch
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		text := scanner.Text()

		searchText := text
		if !query.CaseSensitive {
			searchText = strings.ToLower(text)
		}

		col := strings.Index(searchText, q)
		if col >= 0 {
			matches = append(matches, SearchMatch{
				Path:   absPath,
				Line:   lineNum,
				Column: col + 1, // 1-indexed
				Text:   text,
			})
		}
	}

	return matches, scanner.Err()
}

func uniqueFiles(files []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, f := range files {
		if !seen[f] {
			seen[f] = true
			f = filepath.ToSlash(f)
			result = append(result, f)
		}
	}
	return result
}
