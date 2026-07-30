package write

import (
	"log"
	"sort"
	"time"

	"github.com/ryannmicua/hq-cli/internal/contract"
	"github.com/ryannmicua/hq-cli/internal/fsx"
)

type ChangesService struct {
	receipts *ReceiptStore
	fsys     fsx.FS
	root     string
}

type ChangesPage struct {
	Receipts   []contract.Receipt `json:"receipts"`
	NextCursor uint64             `json:"nextCursor"`
	HasMore    bool               `json:"hasMore"`
}

func NewChangesService(fsys fsx.FS, root string, receipts *ReceiptStore) *ChangesService {
	return &ChangesService{
		receipts: receipts,
		fsys:     fsys,
		root:     root,
	}
}

func (s *ChangesService) Since(since time.Time, limit int) (ChangesPage, error) {
	receipts, err := s.receipts.ListAll()
	if err != nil {
		return ChangesPage{}, err
	}

	var filtered []contract.Receipt
	var parseErrors int
	for _, r := range receipts {
		t, err := time.Parse(time.RFC3339, r.AppliedAt)
		if err != nil {
			parseErrors++
			continue
		}
		if !t.Before(since) {
			filtered = append(filtered, r)
		}
	}

	if parseErrors > 0 {
		log.Printf("warn: %d receipts with unparseable AppliedAt timestamp skipped in Since query", parseErrors)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Cursor < filtered[j].Cursor
	})

	hasMore := limit > 0 && len(filtered) > limit
	if limit > 0 && len(filtered) == limit && len(filtered) > 0 {
		remaining, _, err := s.receipts.ListAfter(filtered[len(filtered)-1].Cursor, 1)
		if err == nil && len(remaining) > 0 {
			hasMore = true
		}
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	nextCursor := uint64(0)
	if len(filtered) > 0 {
		nextCursor = filtered[len(filtered)-1].Cursor
	}

	return ChangesPage{
		Receipts:   filtered,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *ChangesService) After(cursor uint64, limit int) (ChangesPage, error) {
	if limit <= 0 {
		limit = 100
	}

	receipts, nextCursor, err := s.receipts.ListAfter(cursor, limit)
	if err != nil {
		return ChangesPage{}, err
	}

	lastCursor := uint64(0)
	if len(receipts) > 0 {
		lastCursor = receipts[len(receipts)-1].Cursor
	}

	remaining, _, err := s.receipts.ListAfter(lastCursor, 1)
	if err != nil {
		return ChangesPage{}, err
	}

	return ChangesPage{
		Receipts:   receipts,
		NextCursor: nextCursor,
		HasMore:    len(remaining) > 0,
	}, nil
}
