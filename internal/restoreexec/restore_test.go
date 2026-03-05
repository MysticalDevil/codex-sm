package restoreexec

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
)

func TestActionName(t *testing.T) {
	if got := ActionName(true); got != "restore-dry-run" {
		t.Fatalf("unexpected dry action: %q", got)
	}
	if got := ActionName(false); got != "restore" {
		t.Fatalf("unexpected real action: %q", got)
	}
}

func TestExecuteValidations(t *testing.T) {
	_, err := Execute(nil, session.Selector{}, Options{})
	if err == nil || !strings.Contains(err.Error(), "requires at least one selector") {
		t.Fatalf("expected selector validation error, got: %v", err)
	}

	sel := session.Selector{ID: "x"}
	_, err = Execute([]session.Session{{SessionID: "x", Path: "/tmp/x"}}, sel, Options{DryRun: false})
	if err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("expected confirm validation error, got: %v", err)
	}

	_, err = Execute(
		[]session.Session{{SessionID: "x", Path: "/tmp/x"}, {SessionID: "y", Path: "/tmp/y"}},
		sel,
		Options{DryRun: false, Confirm: true, MaxBatch: 50},
	)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected yes validation error, got: %v", err)
	}
}

func TestExecuteDryRunAndReal(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	trashRoot := filepath.Join(workspace, "trash", "sessions")
	src := filepath.Join(trashRoot, "2026", "03", "02", "rollout-r1.jsonl")

	candidates := []session.Session{{
		SessionID: "99990000-1111-2222-3333-444444444444",
		Path:      src,
		SizeBytes: 10,
	}}
	sel := session.Selector{ID: candidates[0].SessionID}

	dry, err := Execute(candidates, sel, Options{
		DryRun:            true,
		SessionsRoot:      sessionsRoot,
		TrashSessionsRoot: trashRoot,
	})
	if err != nil {
		t.Fatalf("dry-run execute: %v", err)
	}
	if dry.Skipped != 1 || dry.Succeeded != 0 || dry.Failed != 0 {
		t.Fatalf("unexpected dry summary: %+v", dry)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source should remain after dry-run: %v", err)
	}

	real, err := Execute(candidates, sel, Options{
		DryRun:            false,
		Confirm:           true,
		Yes:               true,
		SessionsRoot:      sessionsRoot,
		TrashSessionsRoot: trashRoot,
	})
	if err != nil {
		t.Fatalf("real execute: %v", err)
	}
	if real.Succeeded != 1 || real.Failed != 0 {
		t.Fatalf("unexpected real summary: %+v", real)
	}
	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("trash source should be moved, err=%v", err)
	}
	dst := filepath.Join(sessionsRoot, "2026", "03", "02", "rollout-r1.jsonl")
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("destination should exist after restore: %v", err)
	}
}
