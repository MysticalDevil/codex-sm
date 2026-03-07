package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionCommandBash(t *testing.T) {
	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"completion", "bash"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("completion bash execute: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "__start_codexsm") {
		t.Fatalf("unexpected bash completion output: %q", out)
	}
}

func TestCompletionCommandInvalidShell(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"completion", "invalid-shell"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for invalid completion shell")
	}
}
