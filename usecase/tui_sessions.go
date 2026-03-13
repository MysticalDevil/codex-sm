package usecase

import (
	"slices"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session"
)

// LoadTUISessionsInput describes TUI session-loading constraints.
type LoadTUISessionsInput struct {
	SessionsRoot string
	ScanLimit    int
	ViewLimit    int
	Now          time.Time
	Repository   core.SessionRepository
	Evaluator    core.RiskEvaluator
}

// LoadTUISessionsResult is the normalized TUI session set.
type LoadTUISessionsResult struct {
	Total int
	Items []session.Session
}

// LoadTUISessions loads sessions for TUI and applies risk-first ordering.
func LoadTUISessions(in LoadTUISessionsInput) (LoadTUISessionsResult, error) {
	q, err := core.QuerySessions(in.Repository, in.SessionsRoot, core.QuerySpec{
		Now: in.Now,
	})
	if err != nil {
		return LoadTUISessionsResult{}, err
	}
	items := append([]session.Session(nil), q.Items...)
	SortTUISessionsByRisk(items, in.Evaluator)

	if in.ScanLimit > 0 && len(items) > in.ScanLimit {
		items = items[:in.ScanLimit]
	}
	if in.ViewLimit > 0 && len(items) > in.ViewLimit {
		items = items[:in.ViewLimit]
	}
	return LoadTUISessionsResult{
		Total: len(q.Items),
		Items: items,
	}, nil
}

// SortTUISessionsByRisk applies TUI priority ordering:
// risk desc, updated_at desc, session_id asc.
func SortTUISessionsByRisk(items []session.Session, evaluator core.RiskEvaluator) {
	if len(items) <= 1 {
		return
	}
	if evaluator == nil {
		evaluator = core.SessionRiskEvaluator{}
	}
	slices.SortStableFunc(items, func(a, b session.Session) int {
		ra := evaluator.Evaluate(a, nil)
		rb := evaluator.Evaluate(b, nil)
		if c := riskLevelRank(rb.Level) - riskLevelRank(ra.Level); c != 0 {
			return c
		}
		if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.SessionID, b.SessionID)
	})
}

func riskLevelRank(level session.RiskLevel) int {
	switch level {
	case session.RiskHigh:
		return 2
	case session.RiskMedium:
		return 1
	default:
		return 0
	}
}
