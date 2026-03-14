package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSHA256SidecarChecker(t *testing.T) {
	dir := t.TempDir()

	p := filepath.Join(dir, "x.jsonl")
	if err := os.WriteFile(p, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write session: %v", err)
	}

	sum, err := fileSHA256Hex(p)
	if err != nil {
		t.Fatalf("fileSHA256Hex: %v", err)
	}

	// No sidecar => not verified, not risky by itself.
	res := SHA256SidecarChecker(Session{Path: p})
	if res.Verified {
		t.Fatalf("expected not verified without sidecar, got %+v", res)
	}

	// Matching sidecar.
	if err := os.WriteFile(p+".sha256", []byte(sum+"  "+filepath.Base(p)+"\n"), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	res = SHA256SidecarChecker(Session{Path: p})
	if !res.Verified || !res.Match {
		t.Fatalf("expected verified match, got %+v", res)
	}

	// Mismatch.
	bad := strings.Repeat("0", 64)
	if err := os.WriteFile(p+".sha256", []byte(bad+"\n"), 0o644); err != nil {
		t.Fatalf("write bad sidecar: %v", err)
	}

	res = SHA256SidecarChecker(Session{Path: p})
	if !res.Verified || res.Match {
		t.Fatalf("expected verified mismatch, got %+v", res)
	}
}
