package usecase

import (
	"time"

	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session"
)

type ListInput struct {
	SessionsRoot string
	Selector     session.Selector
	Now          time.Time
	SortBy       string
	Order        string
	Offset       int
	Limit        int
	Repository   core.SessionRepository
}

type ListResult struct {
	Total int
	Items []session.Session
}

func ListSessions(in ListInput) (ListResult, error) {
	q, err := core.QuerySessions(in.Repository, in.SessionsRoot, core.QuerySpec{
		Selector: in.Selector,
		SortBy:   in.SortBy,
		Order:    in.Order,
		Offset:   in.Offset,
		Limit:    in.Limit,
		Now:      in.Now,
	})
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Total: q.Total, Items: q.Items}, nil
}
