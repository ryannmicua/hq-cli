package write

import (
	"context"

	"github.com/ryannmicua/hq-cli/internal/contract"
)

type RenderedTarget struct {
	Path       string
	Content    []byte
	SHA256     string
	CreateOnly bool
}

type Renderer interface {
	Render(ctx context.Context, request contract.Request, current []byte) (RenderedTarget, error)
}
