package cli

import (
	"io"

	"github.com/MysticalDevil/codexsm/internal/ops"
)

type previewMode = ops.PreviewMode

const (
	previewFull   previewMode = ops.PreviewFull
	previewSample previewMode = ops.PreviewSample
	previewNone   previewMode = ops.PreviewNone
)

func parsePreviewMode(v string) (previewMode, error) {
	return ops.ParsePreviewMode(v)
}

func isInteractiveReader(r io.Reader) bool {
	return ops.IsInteractiveReader(r)
}
