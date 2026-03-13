package usecase

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/audit"
	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session"
)

const (
	DefaultMaxBatchReal   = 50
	DefaultMaxBatchDryRun = 500
)

type DeleteCandidatesInput struct {
	SessionsRoot string
	Selector     session.Selector
	Now          time.Time
	Repository   core.SessionRepository
}

type DeleteCandidatesResult struct {
	Candidates    []session.Session
	AffectedBytes int64
}

func SelectDeleteCandidates(in DeleteCandidatesInput) (DeleteCandidatesResult, error) {
	if !in.Selector.HasAnyFilter() {
		return DeleteCandidatesResult{}, errors.New("delete requires at least one selector (--id/--id-prefix/--host-contains/--path-contains/--head-contains/--older-than/--health)")
	}
	items, err := core.QuerySessions(in.Repository, in.SessionsRoot, core.QuerySpec{
		Selector: in.Selector,
		Now:      in.Now,
	})
	if err != nil {
		return DeleteCandidatesResult{}, err
	}
	var affected int64
	for _, s := range items {
		affected += s.SizeBytes
	}
	return DeleteCandidatesResult{
		Candidates:    items,
		AffectedBytes: affected,
	}, nil
}

type RestoreCandidatesInput struct {
	TrashSessionsRoot string
	Selector          session.Selector
	BatchID           string
	LogFile           string
	Now               time.Time
	Repository        core.SessionRepository
	IDsForBatch       func(logFile, batchID string) ([]string, error)
}

type RestoreCandidatesResult struct {
	Candidates    []session.Session
	AffectedBytes int64
}

func SelectRestoreCandidates(in RestoreCandidatesInput) (RestoreCandidatesResult, error) {
	batchID := strings.TrimSpace(in.BatchID)
	if batchID != "" && in.Selector.HasAnyFilter() {
		return RestoreCandidatesResult{}, fmt.Errorf("restore --batch-id cannot be combined with selector flags")
	}

	if batchID != "" {
		loadIDs := in.IDsForBatch
		if loadIDs == nil {
			loadIDs = audit.SessionIDsForBatchRollback
		}
		ids, err := loadIDs(in.LogFile, batchID)
		if err != nil {
			return RestoreCandidatesResult{}, err
		}
		idSet := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
		all, err := core.QuerySessions(in.Repository, in.TrashSessionsRoot, core.QuerySpec{
			Now: in.Now,
		})
		if err != nil {
			return RestoreCandidatesResult{}, err
		}
		candidates := make([]session.Session, 0, len(all))
		var affected int64
		for _, s := range all {
			if _, ok := idSet[s.SessionID]; !ok {
				continue
			}
			candidates = append(candidates, s)
			affected += s.SizeBytes
		}
		if len(candidates) == 0 {
			return RestoreCandidatesResult{}, fmt.Errorf("batch id %q has no sessions currently restorable from trash", batchID)
		}
		return RestoreCandidatesResult{Candidates: candidates, AffectedBytes: affected}, nil
	}

	if !in.Selector.HasAnyFilter() {
		return RestoreCandidatesResult{}, errors.New("restore requires at least one selector (--id/--id-prefix/--host-contains/--path-contains/--head-contains/--older-than/--health or --batch-id)")
	}
	items, err := core.QuerySessions(in.Repository, in.TrashSessionsRoot, core.QuerySpec{
		Selector: in.Selector,
		Now:      in.Now,
	})
	if err != nil {
		return RestoreCandidatesResult{}, err
	}
	var affected int64
	for _, s := range items {
		affected += s.SizeBytes
	}
	return RestoreCandidatesResult{
		Candidates:    items,
		AffectedBytes: affected,
	}, nil
}

func EffectiveMaxBatch(flagChanged bool, configured int, dryRun bool) int {
	if flagChanged {
		return configured
	}
	if dryRun {
		return DefaultMaxBatchDryRun
	}
	return DefaultMaxBatchReal
}
