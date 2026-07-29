package write

import (
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
