package audit

import (
	"time"

	"github.com/MysticalDevil/codexsm/session"
)

// BuildActionRecord creates a normalized audit record shared by CLI/TUI callers.
func BuildActionRecord(
	batchID string,
	ts time.Time,
	action string,
	simulation bool,
	selector session.Selector,
	items []session.Session,
	affectedBytes int64,
	results []session.DeleteResult,
	errorSummary string,
) ActionRecord {
	rec := ActionRecord{
		BatchID:       batchID,
		Timestamp:     ts,
		Action:        action,
		Simulation:    simulation,
		Selector:      selector,
		MatchedCount:  len(items),
		AffectedBytes: affectedBytes,
		Results:       results,
		ErrorSummary:  errorSummary,
		Sessions:      make([]SessionRef, 0, len(items)),
	}
	for _, s := range items {
		rec.Sessions = append(rec.Sessions, SessionRef{
			SessionID: s.SessionID,
			Path:      s.Path,
		})
	}
	return rec
}
