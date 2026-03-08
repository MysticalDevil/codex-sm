package session

import "strings"

type RiskLevel string

const (
	RiskNone   RiskLevel = "none"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type RiskReason string

const (
	RiskReasonNone                RiskReason = "none"
	RiskReasonCorrupted           RiskReason = "corrupted-session"
	RiskReasonMissingMeta         RiskReason = "missing-meta"
	RiskReasonIntegrityCheckError RiskReason = "integrity-check-error"
	RiskReasonIntegrityMismatch   RiskReason = "integrity-mismatch"
)

type Risk struct {
	Level  RiskLevel  `json:"level"`
	Reason RiskReason `json:"reason"`
	Detail string     `json:"detail,omitempty"`
}

type IntegrityCheckResult struct {
	Verified bool
	Match    bool
	Err      error
	Detail   string
}

type IntegrityChecker func(Session) IntegrityCheckResult

func EvaluateRisk(s Session, checker IntegrityChecker) Risk {
	switch s.Health {
	case HealthCorrupted:
		return Risk{Level: RiskHigh, Reason: RiskReasonCorrupted, Detail: "session file is corrupted or unreadable"}
	case HealthMissingMeta:
		return Risk{Level: RiskMedium, Reason: RiskReasonMissingMeta, Detail: "session_meta missing or invalid"}
	}

	if checker == nil {
		return Risk{Level: RiskNone, Reason: RiskReasonNone}
	}
	res := checker(s)
	if res.Err != nil {
		return Risk{
			Level:  RiskMedium,
			Reason: RiskReasonIntegrityCheckError,
			Detail: strings.TrimSpace(res.Err.Error()),
		}
	}
	if res.Verified && !res.Match {
		return Risk{
			Level:  RiskHigh,
			Reason: RiskReasonIntegrityMismatch,
			Detail: strings.TrimSpace(res.Detail),
		}
	}
	return Risk{Level: RiskNone, Reason: RiskReasonNone}
}

func IsRisky(s Session, checker IntegrityChecker) bool {
	return EvaluateRisk(s, checker).Level != RiskNone
}
