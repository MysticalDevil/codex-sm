// Package deleteexec provides execution entrypoint for delete operations.
package deleteexec

import "github.com/MysticalDevil/codexsm/session"

// Summary is delete execution summary.
type Summary = session.DeleteSummary

// Options controls delete behavior.
type Options = session.DeleteOptions

// Execute runs delete operation.
func Execute(candidates []session.Session, sel session.Selector, opts Options) (Summary, error) {
	return session.DeleteSessions(candidates, sel, opts)
}
