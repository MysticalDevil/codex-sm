// Package restoreexec contains restore operation execution logic.
package restoreexec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MysticalDevil/codexsm/internal/fileutil"
	"github.com/MysticalDevil/codexsm/session"
)

// Summary describes restore operation result.
type Summary struct {
	Action        string
	Simulation    bool
	MatchedCount  int
	Succeeded     int
	Failed        int
	Skipped       int
	AffectedBytes int64
	Results       []session.DeleteResult
	ErrorSummary  string
}

// Options controls restore behavior.
type Options struct {
	DryRun             bool
	Confirm            bool
	Yes                bool
	AllowEmptySelector bool
	MaxBatch           int
	SessionsRoot       string
	TrashSessionsRoot  string
}

// Execute runs restore over selected candidates.
func Execute(candidates []session.Session, sel session.Selector, opts Options) (Summary, error) {
	summary := Summary{
		Action:       ActionName(opts.DryRun),
		Simulation:   opts.DryRun,
		MatchedCount: len(candidates),
		Results:      make([]session.DeleteResult, 0, len(candidates)),
	}
	if !sel.HasAnyFilter() && !opts.AllowEmptySelector {
		summary.ErrorSummary = "restore requires at least one selector (--id/--id-prefix/--host-contains/--path-contains/--head-contains/--older-than/--health or --batch-id)"
		return summary, errors.New(summary.ErrorSummary)
	}
	if opts.MaxBatch <= 0 {
		opts.MaxBatch = 50
	}
	if !opts.DryRun {
		if !opts.Confirm {
			summary.ErrorSummary = "real restore requires --confirm"
			return summary, errors.New(summary.ErrorSummary)
		}
		if len(candidates) > 1 && !opts.Yes {
			summary.ErrorSummary = "batch restore requires --yes"
			return summary, errors.New(summary.ErrorSummary)
		}
		if len(candidates) > opts.MaxBatch {
			summary.ErrorSummary = fmt.Sprintf("matched %d sessions; exceeds --max-batch=%d", len(candidates), opts.MaxBatch)
			return summary, errors.New(summary.ErrorSummary)
		}
	}

	for _, s := range candidates {
		summary.AffectedBytes += s.SizeBytes
		rel, err := filepath.Rel(opts.TrashSessionsRoot, s.Path)
		if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
			rel = filepath.Base(s.Path)
		}
		dst := filepath.Join(opts.SessionsRoot, rel)

		if opts.DryRun {
			summary.Skipped++
			summary.Results = append(summary.Results, session.DeleteResult{
				SessionID:   s.SessionID,
				Path:        s.Path,
				Destination: dst,
				Status:      "simulated",
			})
			continue
		}

		if _, err := os.Stat(dst); err == nil {
			summary.Failed++
			summary.Results = append(summary.Results, session.DeleteResult{
				SessionID: s.SessionID,
				Path:      s.Path,
				Status:    "failed",
				Error:     fmt.Sprintf("destination exists: %s", dst),
			})
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			summary.Failed++
			summary.Results = append(summary.Results, session.DeleteResult{
				SessionID: s.SessionID,
				Path:      s.Path,
				Status:    "failed",
				Error:     err.Error(),
			})
			continue
		}
		if err := fileutil.MoveFile(s.Path, dst); err != nil {
			summary.Failed++
			summary.Results = append(summary.Results, session.DeleteResult{
				SessionID: s.SessionID,
				Path:      s.Path,
				Status:    "failed",
				Error:     err.Error(),
			})
			continue
		}
		summary.Succeeded++
		summary.Results = append(summary.Results, session.DeleteResult{
			SessionID:   s.SessionID,
			Path:        s.Path,
			Destination: dst,
			Status:      "restored",
		})
	}

	if summary.Failed > 0 {
		summary.ErrorSummary = fmt.Sprintf("%d failed", summary.Failed)
	}
	return summary, nil
}

// ActionName returns action label used in audit output.
func ActionName(dryRun bool) string {
	if dryRun {
		return "restore-dry-run"
	}
	return "restore"
}
