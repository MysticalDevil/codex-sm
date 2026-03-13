package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
)

func TestSelectDeleteCandidates(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")

	_, err := SelectDeleteCandidates(DeleteCandidatesInput{
		SessionsRoot: root,
		Selector:     session.Selector{},
		Now:          time.Now(),
	})
	if err == nil || !strings.Contains(err.Error(), "requires at least one selector") {
		t.Fatalf("expected selector error, got: %v", err)
	}

	res, err := SelectDeleteCandidates(DeleteCandidatesInput{
		SessionsRoot: root,
		Selector: session.Selector{
			ID: "11111111-1111-1111-1111-111111111111",
		},
		Now: time.Now(),
	})
	if err != nil {
		t.Fatalf("SelectDeleteCandidates: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res.Candidates))
	}
	if res.AffectedBytes <= 0 {
		t.Fatalf("expected affected bytes > 0, got %d", res.AffectedBytes)
	}
}

func TestSelectRestoreCandidates(t *testing.T) {
	trashSessionsRoot := t.TempDir()
	writeSessionFixture(t, trashSessionsRoot, "a-1", "/tmp/a")
	writeSessionFixture(t, trashSessionsRoot, "b-2", "/tmp/b")

	_, err := SelectRestoreCandidates(RestoreCandidatesInput{
		TrashSessionsRoot: trashSessionsRoot,
		Selector:          session.Selector{},
		BatchID:           "b-1",
		LogFile:           "/tmp/log",
		IDsForBatch: func(_ string, _ string) ([]string, error) {
			return []string{"a-1"}, nil
		},
		Now: time.Now(),
	})
	if err != nil {
		t.Fatalf("batch-id restore candidates: %v", err)
	}

	_, err = SelectRestoreCandidates(RestoreCandidatesInput{
		TrashSessionsRoot: trashSessionsRoot,
		Selector: session.Selector{
			ID: "a-1",
		},
		BatchID: "b-1",
		LogFile: "/tmp/log",
		IDsForBatch: func(_ string, _ string) ([]string, error) {
			return []string{"a-1"}, nil
		},
		Now: time.Now(),
	})
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("expected conflict error, got: %v", err)
	}

	_, err = SelectRestoreCandidates(RestoreCandidatesInput{
		TrashSessionsRoot: trashSessionsRoot,
		Selector:          session.Selector{},
		BatchID:           "",
		Now:               time.Now(),
	})
	if err == nil || !strings.Contains(err.Error(), "requires at least one selector") {
		t.Fatalf("expected missing selector error, got: %v", err)
	}
}

func TestEffectiveMaxBatch(t *testing.T) {
	if got := EffectiveMaxBatch(false, 777, true); got != DefaultMaxBatchDryRun {
		t.Fatalf("unexpected dry-run default max-batch: %d", got)
	}
	if got := EffectiveMaxBatch(false, 777, false); got != DefaultMaxBatchReal {
		t.Fatalf("unexpected real default max-batch: %d", got)
	}
	if got := EffectiveMaxBatch(true, 123, true); got != 123 {
		t.Fatalf("expected configured max-batch override, got %d", got)
	}
}

func writeSessionFixture(t *testing.T, sessionsRoot, id, host string) {
	t.Helper()
	dir := filepath.Join(sessionsRoot, "2026", "03", "08")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir sessions fixture: %v", err)
	}
	path := filepath.Join(dir, id+".jsonl")
	line := fmt.Sprintf(
		`{"type":"session_meta","payload":{"id":"%s","cwd":"%s","timestamp":"%s"}}`+"\n",
		id,
		host,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatalf("write session fixture: %v", err)
	}
}
