package usecase

import (
	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session/scanner"
)

func sessionRepositoryOrDefault(repo core.SessionRepository) core.SessionRepository {
	if repo != nil {
		return repo
	}

	return scanner.ScanSessions
}
