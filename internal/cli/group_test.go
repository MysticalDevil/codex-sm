package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGroupByHealthJSON(t *testing.T) {
	root := t.TempDir()
	mustWriteSession(t, root, "a", "{\"type\":\"session_meta\",\"payload\":{\"id\":\"a\",\"timestamp\":\"2026-03-02T09:44:00.024Z\"}}\n", time.Now().Add(-1*time.Hour))
	mustWriteSession(t, root, "b", "{\"type\":\"session_meta\",\"payload\":{\"id\":\"b\",\"timestamp\":\"2026-03-02T09:44:00.024Z\"}}\n", time.Now().Add(-2*time.Hour))
	bad := filepath.Join(root, "2026", "03", "02", "bad.jsonl")
	if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("{bad-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"group", "--sessions-root", root, "--by", "health", "--format", "json", "--color", "never"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("group execute: %v", err)
	}

	var got []groupStat
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(got))
	}
	if got[0].Group != "ok" || got[0].Count != 2 {
		t.Fatalf("unexpected first group: %+v", got[0])
	}
}

func mustWriteSession(t *testing.T, root, id, firstLine string, mod time.Time) {
	t.Helper()
	path := filepath.Join(root, "2026", "03", "02", "rollout-"+id+".jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(firstLine), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}
