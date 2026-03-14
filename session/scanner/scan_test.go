package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
)

func TestScanSessionsHealthAndID(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")
	okFile := filepath.Join(root, "2026", "03", "02", "rollout-scanner-ok.jsonl")
	missingMeta := filepath.Join(root, "missing.jsonl")
	corrupted := filepath.Join(root, "bad.jsonl")

	list, err := ScanSessions(root)
	if err != nil {
		t.Fatalf("ScanSessions: %v", err)
	}

	foundOK := false
	foundMissing := false
	foundCorrupted := false

	for _, s := range list {
		switch s.Path {
		case okFile:
			foundOK = true

			if s.Health != session.HealthOK {
				t.Fatalf("ok file health=%s", s.Health)
			}

			if s.SessionID != "019cadee-e315-7b91-8b5d-c0b52770cca6" {
				t.Fatalf("unexpected session id: %s", s.SessionID)
			}

			if s.Head != "hello codex session manager" {
				t.Fatalf("unexpected session head: %q", s.Head)
			}

			if s.HostDir != "/workspace/proj" {
				t.Fatalf("unexpected host dir: %q", s.HostDir)
			}
		case missingMeta:
			foundMissing = true

			if s.Health != session.HealthMissingMeta {
				t.Fatalf("missing file health=%s", s.Health)
			}
		case corrupted:
			foundCorrupted = true

			if s.Health != session.HealthCorrupted {
				t.Fatalf("bad file health=%s", s.Health)
			}
		}
	}

	if !foundOK || !foundMissing || !foundCorrupted {
		t.Fatalf("did not find all expected files")
	}
}

func TestScanSessionsHeadSkipsInstructionNoise(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")
	p := filepath.Join(root, "2026", "03", "03", "rollout-noise.jsonl")

	list, err := ScanSessions(root)
	if err != nil {
		t.Fatalf("ScanSessions: %v", err)
	}

	found := false

	for _, s := range list {
		if s.Path != p {
			continue
		}

		found = true

		if s.Head != "default list output should hide filename" {
			t.Fatalf("unexpected head: %q", s.Head)
		}

		break
	}

	if !found {
		t.Fatalf("noise fixture not found: %s", p)
	}
}

func TestScanSessionsMarksOverlongMetaLineCorrupted(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "oversize.jsonl")

	overlong := `{"type":"session_meta","payload":{"id":"x","timestamp":"` + strings.Repeat("1", maxSessionMetaLineBytes) + `"}}`
	if err := os.WriteFile(p, []byte(overlong), 0o644); err != nil {
		t.Fatalf("write oversize fixture: %v", err)
	}

	list, err := ScanSessions(root)
	if err != nil {
		t.Fatalf("ScanSessions: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("expected 1 scanned session, got %d", len(list))
	}

	if list[0].Health != session.HealthCorrupted {
		t.Fatalf("expected corrupted health for oversize meta line, got %s", list[0].Health)
	}
}

func TestScanSessionsLimitedKeepsTopNByComparator(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")

	items, err := ScanSessionsLimited(root, 2, func(a, b session.Session) bool {
		if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
			return c < 0
		}

		return a.SessionID < b.SessionID
	})
	if err != nil {
		t.Fatalf("ScanSessionsLimited: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(items))
	}

	if items[0].UpdatedAt.Before(items[1].UpdatedAt) {
		t.Fatalf("expected descending updated order, got %+v", items)
	}
}

func TestScanSessionsExtremeStaticFixtureHealth(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "extreme-static")
	root := filepath.Join(workspace, "sessions")

	list, err := ScanSessions(root)
	if err != nil {
		t.Fatalf("ScanSessions: %v", err)
	}

	if len(list) != 6 {
		t.Fatalf("expected 6 sessions, got %d", len(list))
	}

	byBase := make(map[string]session.Session, len(list))
	for _, item := range list {
		byBase[filepath.Base(item.Path)] = item
	}

	if got := byBase["mixed-corrupt-and-huge-001.jsonl"].Health; got != session.HealthCorrupted {
		t.Fatalf("mixed-corrupt health=%s", got)
	}

	if got := byBase["single-line-no-newline-001.jsonl"].Health; got != session.HealthMissingMeta {
		t.Fatalf("single-line-no-newline health=%s", got)
	}

	if got := byBase["oversize-meta-line-001.jsonl"].Health; got != session.HealthOK {
		t.Fatalf("oversize-meta health=%s", got)
	}

	if head := byBase["unicode-wide-long-001.jsonl"].Head; !strings.Contains(head, "超长宽字符会话") {
		t.Fatalf("unexpected unicode head: %q", head)
	}

	if head := byBase["oversize-user-message-001.jsonl"].Head; !strings.Contains(head, "U-LONG-START") {
		t.Fatalf("unexpected oversize user head: %q", head)
	}
}
