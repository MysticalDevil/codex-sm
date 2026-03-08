package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
)

func TestDoctorCommandNonStrict(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	t.Setenv("SESSIONS_ROOT", sessionsRoot)
	t.Setenv("CSM_CONFIG", filepath.Join(workspace, "missing-config.json"))

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"doctor"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor execute: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "CHECK") || !strings.Contains(out, "sessions_root") {
		t.Fatalf("unexpected doctor output: %q", out)
	}
}

func TestDoctorCommandStrictFailsOnWarn(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	t.Setenv("SESSIONS_ROOT", sessionsRoot)
	t.Setenv("CSM_CONFIG", filepath.Join(workspace, "missing-config.json"))

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "--strict"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected strict doctor failure")
	}
}

func TestDoctorRiskCommandReturnsFailureWhenRiskFound(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")
	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "risk", "--sessions-root", root, "--sample-limit", "5"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected risk command to fail when risky sessions exist")
	}
	var ex *ExitError
	if !errors.As(err, &ex) || ex.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got err=%v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "RISK SUMMARY") || !strings.Contains(out, "risk_total=") {
		t.Fatalf("unexpected risk output: %q", out)
	}
}

func TestDoctorRiskCommandPassesWhenNoRiskFound(t *testing.T) {
	sessionsRoot := t.TempDir()
	writeDoctorSessionFixture(t, sessionsRoot, "ok1", t.TempDir())
	writeDoctorSessionFixture(t, sessionsRoot, "ok2", t.TempDir())

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "risk", "--sessions-root", sessionsRoot})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor risk execute: %v", err)
	}
	if !strings.Contains(stdout.String(), "no risky sessions found") {
		t.Fatalf("expected no-risk output, got: %q", stdout.String())
	}
}

func TestCheckSessionHostPathsWarnsWhenHostMissing(t *testing.T) {
	sessionsRoot := t.TempDir()
	existingHost := t.TempDir()
	missingHost := filepath.Join(t.TempDir(), "missing-host-dir")

	writeDoctorSessionFixture(t, sessionsRoot, "s1", existingHost)
	writeDoctorSessionFixture(t, sessionsRoot, "s2", missingHost)

	got := checkSessionHostPaths(sessionsRoot, nil)
	if got.Level != doctorWarn {
		t.Fatalf("expected warn, got %s detail=%q", got.Level, got.Detail)
	}
	if !strings.Contains(got.Detail, "recommended_actions:") {
		t.Fatalf("expected action block in detail, got: %q", got.Detail)
	}
	if !strings.Contains(got.Detail, "migrate (soft-delete): codexsm delete --host-contains") {
		t.Fatalf("expected delete suggestion in detail, got: %q", got.Detail)
	}
}

func TestCheckSessionHostPathsPassWhenAllHostsExist(t *testing.T) {
	sessionsRoot := t.TempDir()
	hostA := t.TempDir()
	hostB := t.TempDir()

	writeDoctorSessionFixture(t, sessionsRoot, "s1", hostA)
	writeDoctorSessionFixture(t, sessionsRoot, "s2", hostB)

	got := checkSessionHostPaths(sessionsRoot, nil)
	if got.Level != doctorPass {
		t.Fatalf("expected pass, got %s detail=%q", got.Level, got.Detail)
	}
	if !strings.Contains(got.Detail, "all host paths exist") {
		t.Fatalf("unexpected pass detail: %q", got.Detail)
	}
}

func writeDoctorSessionFixture(t *testing.T, sessionsRoot, id, host string) {
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
