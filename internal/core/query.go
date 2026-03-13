package core

import (
	"time"

	"github.com/MysticalDevil/codexsm/session"
)

// QuerySessions scans root via repo, then applies selector and optional limit.
// Ordering follows session.FilterSessions (UpdatedAt desc).
func QuerySessions(repo SessionRepository, root string, spec QuerySpec) ([]session.Session, error) {
	if repo == nil {
		repo = ScannerRepository{}
	}
	items, err := repo.ScanSessions(root)
	if err != nil {
		return nil, err
	}
	now := spec.Now
	if now.IsZero() {
		now = time.Now()
	}
	filtered := session.FilterSessions(items, spec.Selector, now)
	if spec.Limit > 0 && len(filtered) > spec.Limit {
		filtered = filtered[:spec.Limit]
	}
	return filtered, nil
}
