package session_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/MysticalDevil/codexsm/session/scanner"
)

func BenchmarkScanSessions(b *testing.B) {
	root := filepath.Join(testsupport.TestdataRoot(), "fixtures", "rich", "sessions")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		sessions, err := scanner.ScanSessions(root)
		if err != nil {
			b.Fatalf("ScanSessions: %v", err)
		}

		if len(sessions) == 0 {
			b.Fatal("expected non-empty sessions")
		}
	}
}

func BenchmarkFilterSessions(b *testing.B) {
	root := filepath.Join(testsupport.TestdataRoot(), "fixtures", "rich", "sessions")

	sessions, err := scanner.ScanSessions(root)
	if err != nil {
		b.Fatalf("ScanSessions setup: %v", err)
	}

	if len(sessions) == 0 {
		b.Fatal("expected non-empty sessions")
	}

	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		sel  session.Selector
	}{
		{
			name: "all",
			sel:  session.Selector{},
		},
		{
			name: "host_head_health",
			sel: session.Selector{
				HostContains: "workspace",
				HeadContains: "session",
				Health:       session.HealthOK,
				HasHealth:    true,
			},
		},
		{
			name: "older_than",
			sel: session.Selector{
				OlderThan:    24 * time.Hour,
				HasOlderThan: true,
			},
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				out := session.FilterSessions(sessions, tc.sel, now)
				if len(out) == 0 && tc.name == "all" {
					b.Fatal("unexpected empty result for all selector")
				}
			}
		})
	}
}

func BenchmarkScanSessionsLimited_3k(b *testing.B) {
	root := prepareBenchSessionsRoot(b, 3000, false)
	less := func(a, b session.Session) bool {
		if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
			return c < 0
		}

		return a.SessionID < b.SessionID
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		sessions, err := scanner.ScanSessionsLimited(root, 100, less)
		if err != nil {
			b.Fatalf("ScanSessionsLimited: %v", err)
		}

		if len(sessions) != 100 {
			b.Fatalf("expected 100 retained sessions, got %d", len(sessions))
		}
	}
}

func BenchmarkScanSessions_AllVsLimited_3k(b *testing.B) {
	root := prepareBenchSessionsRoot(b, 3000, false)
	less := func(a, b session.Session) bool {
		if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
			return c < 0
		}

		return a.SessionID < b.SessionID
	}

	b.Run("all", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			sessions, err := scanner.ScanSessions(root)
			if err != nil {
				b.Fatalf("ScanSessions: %v", err)
			}

			if len(sessions) != 3000 {
				b.Fatalf("expected 3000 sessions, got %d", len(sessions))
			}
		}
	})

	b.Run("limited_100", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for range b.N {
			sessions, err := scanner.ScanSessionsLimited(root, 100, less)
			if err != nil {
				b.Fatalf("ScanSessionsLimited: %v", err)
			}

			if len(sessions) != 100 {
				b.Fatalf("expected 100 retained sessions, got %d", len(sessions))
			}
		}
	})
}

func BenchmarkScanSessions_ExtremeMix(b *testing.B) {
	root := prepareBenchSessionsRoot(b, 1200, true)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		sessions, err := scanner.ScanSessions(root)
		if err != nil {
			b.Fatalf("ScanSessions: %v", err)
		}

		if len(sessions) < 1200 {
			b.Fatalf("expected sessions from mixed dataset, got %d", len(sessions))
		}
	}
}

func prepareBenchSessionsRoot(b *testing.B, count int, includeExtreme bool) string {
	b.Helper()
	root := b.TempDir()

	base := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	for i := range count {
		created := base.Add(time.Duration(i) * time.Minute)

		dayDir := filepath.Join(root, created.Format("2006"), created.Format("01"), created.Format("02"))
		if err := os.MkdirAll(dayDir, 0o755); err != nil {
			b.Fatalf("mkdir bench day dir: %v", err)
		}

		sessionID := fmt.Sprintf("%08x-1111-2222-3333-%012x", i, i)
		path := filepath.Join(dayDir, fmt.Sprintf("bench-%04d-%s.jsonl", i, sessionID))

		body := strings.Join([]string{
			fmt.Sprintf(`{"type":"session_meta","payload":{"id":"%s","timestamp":"%s","cwd":"/workspace/bench/%02d"}}`, sessionID, created.Format(time.RFC3339Nano), i%32),
			fmt.Sprintf(`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"benchmark session %d keeps scan paths warm"}]}}`, i),
			"",
		}, "\n")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			b.Fatalf("write bench session: %v", err)
		}
	}

	if includeExtreme {
		writeExtremeBenchFixtures(b, root, base.Add(24*time.Hour))
	}

	return root
}

func writeExtremeBenchFixtures(b *testing.B, root string, created time.Time) {
	b.Helper()

	dayDir := filepath.Join(root, created.Format("2006"), created.Format("01"), created.Format("02"))
	if err := os.MkdirAll(dayDir, 0o755); err != nil {
		b.Fatalf("mkdir extreme bench day dir: %v", err)
	}

	files := map[string]string{
		"oversize-meta.jsonl": fmt.Sprintf(
			`{"type":"session_meta","payload":{"id":"extreme-meta","timestamp":"%s","cwd":"%s"}}`+"\n"+
				`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"oversize meta follow-up"}]}}`+"\n",
			created.Format(time.RFC3339Nano),
			strings.Repeat("/segment", 16000),
		),
		"oversize-user.jsonl": fmt.Sprintf(
			`{"type":"session_meta","payload":{"id":"extreme-user","timestamp":"%s","cwd":"/workspace/extreme-user"}}`+"\n"+
				`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"%s"}]}}`+"\n",
			created.Format(time.RFC3339Nano),
			strings.Repeat("U-LONG ", 12000),
		),
		"unicode-wide.jsonl": fmt.Sprintf(
			`{"type":"session_meta","payload":{"id":"extreme-unicode","timestamp":"%s","cwd":"/workspace/extreme-unicode"}}`+"\n"+
				`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"%s"}]}}`+"\n",
			created.Format(time.RFC3339Nano),
			strings.Repeat("请处理宽字符 👨‍👩‍👧‍👦 مرحبا שלום セッション ", 512),
		),
		"corrupted.jsonl":  `{"type":"session_meta","payload":{"id":"broken"` + "\n" + `not-json-line` + "\n",
		"no-newline.jsonl": `{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"single line without newline"}]}}`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dayDir, name), []byte(content), 0o644); err != nil {
			b.Fatalf("write extreme bench fixture %s: %v", name, err)
		}
	}
}
