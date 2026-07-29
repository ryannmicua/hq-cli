package write

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/ryannmicua/hq-cli/internal/contract"
)

type DraftRecordRenderer struct{}

func NewDraftRecordRenderer() *DraftRecordRenderer {
	return &DraftRecordRenderer{}
}

func (r *DraftRecordRenderer) Render(ctx context.Context, req contract.Request, current []byte) (RenderedTarget, error) {
	if !req.CreateOnly {
		return RenderedTarget{}, fmt.Errorf("draft-record requires createOnly=true")
	}

	if current != nil {
		return RenderedTarget{}, fmt.Errorf("draft-record target %q already exists", req.Target)
	}

	var draft contract.DraftRecord
	if err := contract.DecodeStrict(req.Payload, &draft); err != nil {
		return RenderedTarget{}, fmt.Errorf("unmarshal draft-record payload: %w", err)
	}

	if draft.Title == "" {
		return RenderedTarget{}, fmt.Errorf("draft-record: title is required")
	}
	if draft.RecordDate == "" {
		return RenderedTarget{}, fmt.Errorf("draft-record: recordDate is required")
	}
	if draft.Classification == "" {
		return RenderedTarget{}, fmt.Errorf("draft-record: classification is required")
	}

	if !contract.AllowedClassifications[draft.Classification] {
		return RenderedTarget{}, fmt.Errorf("draft-record: invalid classification %q", draft.Classification)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s\n\n", draft.Title))
	b.WriteString(fmt.Sprintf("- Date: %s\n", draft.RecordDate))
	b.WriteString("- Status: Draft\n")
	b.WriteString(fmt.Sprintf("- Source: Submitted through hq-cli request %s\n\n", req.RequestID))
	b.WriteString(draft.Body)
	b.WriteString("\n")

	out := []byte(b.String())
	hash := fmt.Sprintf("%x", sha256.Sum256(out))

	return RenderedTarget{
		Path:       req.Target,
		Content:    out,
		SHA256:     hash,
		CreateOnly: true,
	}, nil
}
