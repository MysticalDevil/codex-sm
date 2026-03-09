package session

import "strings"

// RiskLevel classifies how severe a session issue is.
type RiskLevel string

const (
	// RiskNone means the session has no detected risk.
	RiskNone RiskLevel = "none"
	// RiskMedium means the session needs attention but is still partially usable.
	RiskMedium RiskLevel = "medium"
	// RiskHigh means the session is corrupted or unsafe to trust.
	RiskHigh RiskLevel = "high"
)

// RiskReason identifies the specific policy that made a session risky.
type RiskReason string

const (
	// RiskReasonNone means no risk policy matched.
	RiskReasonNone RiskReason = "none"
	// RiskReasonCorrupted marks unreadable or malformed session files.
	RiskReasonCorrupted RiskReason = "corrupted-session"
	// RiskReasonMissingMeta marks files without usable session metadata.
	RiskReasonMissingMeta RiskReason = "missing-meta"
	// RiskReasonIntegrityCheckError marks sidecar verification failures.
	RiskReasonIntegrityCheckError RiskReason = "integrity-check-error"
	// RiskReasonIntegrityMismatch marks hash mismatches against a sidecar.
	RiskReasonIntegrityMismatch RiskReason = "integrity-mismatch"
)

// Risk describes the highest-priority issue detected for a session.
type Risk struct {
	// Level is the severity assigned by the current risk policy.
	Level RiskLevel `json:"level"`
	// Reason identifies which policy produced Level.
	Reason RiskReason `json:"reason"`
	// Detail carries human-readable context for the detected issue.
	Detail string `json:"detail,omitempty"`
}

// IntegrityCheckResult records the outcome of verifying a session sidecar.
type IntegrityCheckResult struct {
	// Verified reports whether a sidecar check was attempted.
	Verified bool
	// Match reports whether the verified content matched the sidecar digest.
	Match bool
	// Err reports why verification failed when the check could not complete.
	Err error
	// Detail carries additional mismatch or verification context.
	Detail string
}

// IntegrityChecker verifies a session and returns an integrity result.
type IntegrityChecker func(Session) IntegrityCheckResult

// EvaluateRisk returns the highest-priority risk detected for s.
//
// Health-based risks are evaluated first.
// If checker is nil or s is already unhealthy, integrity verification is skipped.
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

// IsRisky reports whether EvaluateRisk returns a non-none level for s.
func IsRisky(s Session, checker IntegrityChecker) bool {
	return EvaluateRisk(s, checker).Level != RiskNone
}
