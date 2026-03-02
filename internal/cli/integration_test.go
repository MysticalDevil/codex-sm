package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestList_DefaultLimitShowsLatest10(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 12; i++ {
		id := fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
		p := filepath.Join(root, "2026", "03", "02", fmt.Sprintf("rollout-2026-03-02T17-39-%02d-%s.jsonl", i, id))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		meta := fmt.Sprintf(`{"type":"session_meta","payload":{"id":"%s","timestamp":"2026-03-02T09:44:00.024Z"}}\n`, id)
		if err := os.WriteFile(p, []byte(meta), 0o644); err != nil {
			t.Fatal(err)
		}
		mod := time.Now().Add(-time.Duration(12-i) * time.Minute)
		if err := os.Chtimes(p, mod, mod); err != nil {
			t.Fatal(err)
		}
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"list", "--sessions-root", root, "--color", "never"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("list execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "showing 10 of 12") {
		t.Fatalf("unexpected list footer: %q", out)
	}
	if count := strings.Count(out, "rollout-"); count != 10 {
		t.Fatalf("expected 10 rows, got %d", count)
	}
}

func TestDelete_DryRunWritesAuditAndKeepsFile(t *testing.T) {
	root := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "actions.log")

	id := "11111111-1111-1111-1111-111111111111"
	sessionPath := filepath.Join(root, "2026", "03", "02", "rollout-2026-03-02T17-44-00-11111111-1111-1111-1111-111111111111.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"type":"session_meta","payload":{"id":"11111111-1111-1111-1111-111111111111","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n"
	if err := os.WriteFile(sessionPath, []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"delete", "--sessions-root", root, "--id", id, "--dry-run", "--log-file", logFile})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete execute: %v", err)
	}
	if _, err := os.Stat(sessionPath); err != nil {
		t.Fatalf("session file should remain on dry-run: %v", err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	line := strings.TrimSpace(string(data))
	if line == "" {
		t.Fatal("expected one audit log line")
	}

	var rec map[string]any
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("invalid audit json: %v", err)
	}
	sim, ok := rec["simulation"].(bool)
	if !ok || !sim {
		t.Fatalf("expected simulation=true, got: %v", rec["simulation"])
	}
}

func TestDelete_RealDeleteRequiresConfirm(t *testing.T) {
	root := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "actions.log")
	id := "22222222-2222-2222-2222-222222222222"
	p := filepath.Join(root, "2026", "03", "02", "rollout-2026-03-02T17-44-00-22222222-2222-2222-2222-222222222222.jsonl")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"type":"session_meta","payload":{"id":"22222222-2222-2222-2222-222222222222","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n"
	if err := os.WriteFile(p, []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"delete", "--sessions-root", root, "--id", id, "--dry-run=false", "--log-file", logFile})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when real delete misses --confirm")
	}
	if !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(p); statErr != nil {
		t.Fatalf("session file should remain when validation fails: %v", statErr)
	}
}
