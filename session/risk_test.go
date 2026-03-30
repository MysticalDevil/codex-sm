package session

import (
	"errors"
	"testing"
)

func TestEvaluateRiskFromHealth(t *testing.T) {
	cases := []struct {
		name   string
		health Health
		level  RiskLevel
		reason RiskReason
	}{
		{name: "ok", health: HealthOK, level: RiskNone, reason: RiskReasonNone},
		{name: "missing-meta", health: HealthMissingMeta, level: RiskMedium, reason: RiskReasonMissingMeta},
		{name: "corrupted", health: HealthCorrupted, level: RiskHigh, reason: RiskReasonCorrupted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateRisk(Session{Health: tc.health}, nil)
			if got.Level != tc.level || got.Reason != tc.reason {
				t.Fatalf("EvaluateRisk(%s)=%+v, want level=%s reason=%s", tc.health, got, tc.level, tc.reason)
			}
		})
	}
}

func TestEvaluateRiskWithIntegrityChecker(t *testing.T) {
	s := Session{Health: HealthOK, SessionID: "s1"}

	errRisk := EvaluateRisk(s, func(Session) IntegrityCheckResult {
		return IntegrityCheckResult{Err: errors.New("io failed")}
	})
	if errRisk.Level != RiskMedium || errRisk.Reason != RiskReasonIntegrityCheckError {
		t.Fatalf("expected integrity error risk, got %+v", errRisk)
	}

	mismatchRisk := EvaluateRisk(s, func(Session) IntegrityCheckResult {
		return IntegrityCheckResult{Verified: true, Match: false, Detail: "sha mismatch"}
	})
	if mismatchRisk.Level != RiskHigh || mismatchRisk.Reason != RiskReasonIntegrityMismatch {
		t.Fatalf("expected integrity mismatch risk, got %+v", mismatchRisk)
	}
}
