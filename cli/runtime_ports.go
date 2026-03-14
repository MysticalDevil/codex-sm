package cli

import (
	"time"

	"github.com/MysticalDevil/codexsm/audit"
)

type clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

type auditSink interface {
	NewBatchID() (string, error)
	WriteActionLog(logFile string, rec audit.ActionRecord) error
}

type defaultAuditSink struct{}

func (defaultAuditSink) NewBatchID() (string, error) {
	return audit.NewBatchID()
}

func (defaultAuditSink) WriteActionLog(logFile string, rec audit.ActionRecord) error {
	return audit.WriteActionLog(logFile, rec)
}

var runtimeClock clock = systemClock{}
var runtimeAuditSink auditSink = defaultAuditSink{}
