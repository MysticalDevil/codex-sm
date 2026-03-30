package audit

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/session"
)

func TestWriteActionLog(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "logs", "actions.log")
	rec := ActionRecord{
		BatchID:    "b-20260305T010203Z-abcdef123456",
		Timestamp:  time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC),
		Action:     "soft-delete",
		Simulation: true,
		Selector: session.Selector{
			IDPrefix:     "019c",
			HostContains: "/workspace",
			PathContains: "rollout",
			HeadContains: "fixture",
			OlderThan:    30 * time.Minute,
			HasOlderThan: true,
			Health:       session.HealthOK,
			HasHealth:    true,
		},
		MatchedCount:  1,
		AffectedBytes: 123,
		Sessions:      []SessionRef{{SessionID: "s1", Path: "/tmp/s1.jsonl"}},
	}

	if err := WriteActionLog(logFile, rec); err != nil {
		t.Fatalf("WriteActionLog #1: %v", err)
	}

	if err := WriteActionLog(logFile, rec); err != nil {
		t.Fatalf("WriteActionLog #2: %v", err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(lines))
	}

	var got actionRecordJSON
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("unmarshal first log line: %v", err)
	}

	if got.BatchID != rec.BatchID {
		t.Fatalf("batch_id mismatch: %#v", got.BatchID)
	}

	if got.Selector.OlderThan != "30m0s" {
		t.Fatalf("older_than should be duration string, got: %#v", got.Selector.OlderThan)
	}

	if got.Selector.IDPrefix != "019c" {
		t.Fatalf("id_prefix mismatch: %#v", got.Selector.IDPrefix)
	}

	if got.Selector.HostContains != "/workspace" || got.Selector.PathContains != "rollout" || got.Selector.HeadContains != "fixture" {
		t.Fatalf("contains selectors mismatch: %#v", got.Selector)
	}
}

func TestSessionIDsForBatchRollback(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "logs", "actions.log")

	base := ActionRecord{
		Timestamp:  time.Date(2026, 3, 5, 8, 0, 0, 0, time.UTC),
		Action:     "soft-delete",
		Simulation: false,
		BatchID:    "b-1",
		Results: []session.DeleteResult{
			{SessionID: "a", Status: "deleted", Destination: "/trash/a.jsonl"},
			{SessionID: "b", Status: "deleted", Destination: "/trash/b.jsonl"},
		},
	}
	if err := WriteActionLog(logFile, base); err != nil {
		t.Fatalf("WriteActionLog base: %v", err)
	}

	if err := WriteActionLog(logFile, ActionRecord{
		Timestamp:  base.Timestamp.Add(time.Minute),
		Action:     "soft-delete",
		Simulation: true,
		BatchID:    "b-2",
		Results:    []session.DeleteResult{{SessionID: "c", Status: "simulated"}},
	}); err != nil {
		t.Fatalf("WriteActionLog dry-run: %v", err)
	}

	ids, err := SessionIDsForBatchRollback(logFile, "b-1")
	if err != nil {
		t.Fatalf("SessionIDsForBatchRollback(b-1): %v", err)
	}

	if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Fatalf("unexpected ids: %#v", ids)
	}

	if _, err := SessionIDsForBatchRollback(logFile, "missing"); err == nil {
		t.Fatal("expected missing batch id error")
	}

	if _, err := SessionIDsForBatchRollback(logFile, "b-2"); err == nil {
		t.Fatal("expected no restorable results error")
	}
}

func TestNewBatchID(t *testing.T) {
	a, err := NewBatchID()
	if err != nil {
		t.Fatalf("NewBatchID #1: %v", err)
	}

	b, err := NewBatchID()
	if err != nil {
		t.Fatalf("NewBatchID #2: %v", err)
	}

	if a == "" || b == "" {
		t.Fatalf("batch ids must be non-empty: a=%q b=%q", a, b)
	}

	if a == b {
		t.Fatalf("batch ids should differ: %q", a)
	}
}

func TestBuildActionRecord(t *testing.T) {
	ts := time.Date(2026, 3, 9, 10, 20, 30, 0, time.UTC)
	sel := session.Selector{ID: "s1"}
	items := []session.Session{{SessionID: "s1", Path: "/tmp/s1.jsonl"}}
	results := []session.DeleteResult{{SessionID: "s1", Path: "/tmp/s1.jsonl", Status: "simulated"}}

	rec := BuildActionRecord(
		"b-1",
		ts,
		"dry-run",
		true,
		sel,
		items,
		123,
		results,
		"",
	)
	if rec.BatchID != "b-1" || rec.Timestamp != ts || rec.Action != "dry-run" {
		t.Fatalf("unexpected basic fields: %+v", rec)
	}

	if rec.MatchedCount != 1 || len(rec.Sessions) != 1 || rec.Sessions[0].SessionID != "s1" {
		t.Fatalf("unexpected session fields: %+v", rec)
	}

	if rec.AffectedBytes != 123 || len(rec.Results) != 1 {
		t.Fatalf("unexpected payload fields: %+v", rec)
	}
}
