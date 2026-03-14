package cli

import (
	"github.com/MysticalDevil/codexsm/audit"
	"github.com/MysticalDevil/codexsm/usecase"
)

type defaultAuditSink struct{}

func (defaultAuditSink) NewBatchID() (string, error) {
	return audit.NewBatchID()
}

func (defaultAuditSink) WriteActionLog(logFile string, rec audit.ActionRecord) error {
	return audit.WriteActionLog(logFile, rec)
}

var runtimeAuditSink usecase.AuditSink = defaultAuditSink{}
