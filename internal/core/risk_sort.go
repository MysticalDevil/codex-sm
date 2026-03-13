package core

import (
	"slices"
	"strings"

	"github.com/MysticalDevil/codexsm/session"
)

// SortSessionsByRisk applies risk-priority ordering:
// risk desc, updated_at desc, session_id asc.
func SortSessionsByRisk(items []session.Session, evaluator RiskEvaluator, checker session.IntegrityChecker) {
	if len(items) <= 1 {
		return
	}
	if evaluator == nil {
		evaluator = SessionRiskEvaluator{}
	}
	slices.SortStableFunc(items, func(a, b session.Session) int {
		ra := evaluator.Evaluate(a, checker)
		rb := evaluator.Evaluate(b, checker)
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
