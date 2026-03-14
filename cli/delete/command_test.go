package del

import (
	"bytes"
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/ops"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/spf13/cobra"
)

func TestPrintDeleteSummaryIncludesResultError(t *testing.T) {
	cmd := &cobra.Command{}
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)

	PrintDeleteSummary(cmd, session.DeleteSummary{
		Action:      "delete",
		Simulation:  true,
		MatchedCount: 1,
		Succeeded:   0,
		Failed:      1,
		Skipped:     0,
		Results: []session.DeleteResult{
			{
				Status:    "failed",
				SessionID: "abc123",
				Path:      "/tmp/a.jsonl",
				Error:     "boom",
			},
		},
	})

	out := stdout.String()
	if !strings.Contains(out, "action=delete simulation=true matched=1 succeeded=0 failed=1") {
		t.Fatalf("unexpected summary line: %q", out)
	}

	if !strings.Contains(out, "failed abc123 /tmp/a.jsonl err=boom") {
		t.Fatalf("expected error result line, got: %q", out)
	}
}

func TestPrintDeletePreviewSampleShowsRemainingCount(t *testing.T) {
	cmd := &cobra.Command{}
	stderr := &bytes.Buffer{}
	cmd.SetErr(stderr)

	candidates := []session.Session{
		{SessionID: "11111111-1111-1111-1111-111111111111", Path: "/tmp/1.jsonl", SizeBytes: 1024},
		{SessionID: "22222222-2222-2222-2222-222222222222", Path: "/tmp/2.jsonl", SizeBytes: 2048},
		{SessionID: "33333333-3333-3333-3333-333333333333", Path: "/tmp/3.jsonl", SizeBytes: 4096},
	}

	PrintDeletePreview(cmd, candidates, false, ops.PreviewSample, 1)

	out := stderr.String()
	if !strings.Contains(out, "preview action=soft-delete matched=3") {
		t.Fatalf("unexpected preview header: %q", out)
	}

	if !strings.Contains(out, "... and 2 more") {
		t.Fatalf("expected remaining count line, got: %q", out)
	}
}

func TestInteractiveConfirmDeleteRequiresTerminalStdin(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("y\n"))
	cmd.SetErr(&bytes.Buffer{})

	ok, err := InteractiveConfirmDelete(cmd, 2, false)
	if err == nil {
		t.Fatal("expected non-terminal stdin error")
	}

	if ok {
		t.Fatal("expected confirmation=false for non-terminal stdin")
	}

	if !strings.Contains(err.Error(), "requires a terminal stdin") {
		t.Fatalf("unexpected error: %v", err)
	}
}
