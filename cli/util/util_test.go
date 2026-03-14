package util

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/session"
)

func TestExitErrorAndWithExitCode(t *testing.T) {
	base := errors.New("boom")

	if got := WithExitCode(nil, 7); got != nil {
		t.Fatalf("WithExitCode(nil) = %v, want nil", got)
	}

	err := WithExitCode(base, 7)

	var ex *ExitError

	if !errors.As(err, &ex) {
		t.Fatalf("expected ExitError wrapper, got %T", err)
	}

	if ex.ExitCode() != 7 {
		t.Fatalf("ExitCode()=%d, want 7", ex.ExitCode())
	}

	if ex.Error() != "boom" {
		t.Fatalf("Error()=%q, want boom", ex.Error())
	}

	if !errors.Is(ex.Unwrap(), base) {
		t.Fatalf("Unwrap()=%v, want wrapping base error", ex.Unwrap())
	}
}

func TestResolveOrDefault(t *testing.T) {
	fallbackCalled := false
	fallback := func() (string, error) {
		fallbackCalled = true
		return "/tmp/default", nil
	}

	got, err := ResolveOrDefault("   ", fallback)
	if err != nil {
		t.Fatalf("ResolveOrDefault empty: %v", err)
	}

	if !fallbackCalled || got != "/tmp/default" {
		t.Fatalf("fallback path=%q called=%v", got, fallbackCalled)
	}

	fallbackCalled = false
	absPath := filepath.Join(t.TempDir(), "x")

	got, err = ResolveOrDefault(absPath, fallback)
	if err != nil {
		t.Fatalf("ResolveOrDefault explicit: %v", err)
	}

	if fallbackCalled {
		t.Fatal("fallback should not be called for explicit path")
	}

	if got != absPath {
		t.Fatalf("explicit path got %q, want %q", got, absPath)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.json")

	if err := WriteFileAtomic(path, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic initial: %v", err)
	}

	if err := WriteFileAtomic(path, []byte(`{"a":2}`), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic replace: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if strings.TrimSpace(string(b)) != `{"a":2}` {
		t.Fatalf("unexpected file contents: %q", string(b))
	}
}

func TestShouldUseColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")

	if !ShouldUseColor("always", &bytes.Buffer{}) {
		t.Fatal("always should force color")
	}

	if ShouldUseColor("never", os.Stdout) {
		t.Fatal("never should disable color")
	}

	t.Setenv("NO_COLOR", "1")

	if ShouldUseColor("auto", os.Stdout) {
		t.Fatal("auto should disable color when NO_COLOR is set")
	}
}

func TestBuildSelectorAndParseHealth(t *testing.T) {
	sel, err := BuildSelector(" id ", " pref ", "host", "path", "head", "24h", "corrupted")
	if err != nil {
		t.Fatalf("BuildSelector valid: %v", err)
	}

	if sel.ID != "id" || sel.IDPrefix != "pref" || !sel.HasOlderThan || !sel.HasHealth {
		t.Fatalf("unexpected selector: %+v", sel)
	}

	if sel.Health != session.HealthCorrupted {
		t.Fatalf("unexpected health: %s", sel.Health)
	}

	if _, err := BuildSelector("", "", "", "", "", "", "bad-health"); err == nil {
		t.Fatal("expected invalid health error")
	}

	if _, err := ParseHealth("missing-meta"); err != nil {
		t.Fatalf("ParseHealth missing-meta: %v", err)
	}
}
