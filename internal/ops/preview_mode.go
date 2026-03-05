// Package ops contains shared operational helpers for CLI actions.
package ops

import (
	"fmt"
	"strings"
)

// PreviewMode controls pre-action preview behavior.
type PreviewMode string

const (
	PreviewFull   PreviewMode = "full"
	PreviewSample PreviewMode = "sample"
	PreviewNone   PreviewMode = "none"
)

// ParsePreviewMode parses a preview mode string.
func ParsePreviewMode(v string) (PreviewMode, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", string(PreviewSample):
		return PreviewSample, nil
	case string(PreviewFull):
		return PreviewFull, nil
	case string(PreviewNone):
		return PreviewNone, nil
	default:
		return "", fmt.Errorf("invalid --preview %q (allowed: full, sample, none)", v)
	}
}
