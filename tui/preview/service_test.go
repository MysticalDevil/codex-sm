package preview

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleLoadedRejectsStaleMessage(t *testing.T) {
	out := HandleLoaded(10, "k1", LoadedMsg{
		RequestID: 9,
		Key:       "k1",
		Lines:     []string{"x"},
	}, "", 50)

	if out.Accepted {
		t.Fatal("expected stale message to be rejected")
	}

	if out.NextWait != "k1" {
		t.Fatalf("NextWait=%q, want k1", out.NextWait)
	}
}

func TestHandleLoadedAcceptsAndBuildsPersistCmd(t *testing.T) {
	record := IndexRecord{
		Key:           "k1",
		Path:          "/tmp/a",
		Width:         80,
		SizeBytes:     10,
		UpdatedAtUnix: 1,
		TouchedAtUnix: 2,
		Lines:         []string{"line"},
	}

	out := HandleLoaded(10, "k1", LoadedMsg{
		RequestID: 10,
		Key:       "k1",
		Lines:     []string{"line"},
		Record:    record,
	}, "/tmp/index.jsonl", 100)

	if !out.Accepted {
		t.Fatal("expected message to be accepted")
	}

	if out.NextWait != "" {
		t.Fatalf("NextWait=%q, want empty", out.NextWait)
	}

	if out.PersistCmd == nil {
		t.Fatal("expected persist cmd")
	}
}

func TestHandleLoadedErrorMapsToFriendlyCacheLine(t *testing.T) {
	out := HandleLoaded(1, "k1", LoadedMsg{
		RequestID: 1,
		Key:       "k1",
		Err:       "read failed",
	}, "/tmp/index.jsonl", 100)

	if !out.Accepted {
		t.Fatal("expected accepted=true for matching request")
	}

	if len(out.CacheLines) != 1 || !strings.Contains(out.CacheLines[0], "preview load failed: read failed") {
		t.Fatalf("unexpected cache lines: %+v", out.CacheLines)
	}
}

func TestPersistIndexCmdWritesIndex(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "preview-index.jsonl")

	cmd := PersistIndexCmd(indexPath, 10, IndexRecord{
		Key:           "k1",
		Path:          "/tmp/a",
		Width:         80,
		SizeBytes:     10,
		UpdatedAtUnix: 1,
		TouchedAtUnix: 2,
		Lines:         []string{"hello"},
	})
	if cmd == nil {
		t.Fatal("expected persist cmd")
	}

	msg := cmd()

	persisted, ok := msg.(IndexPersistedMsg)

	if !ok {
		t.Fatalf("unexpected message type: %T", msg)
	}

	if persisted.Err != "" {
		t.Fatalf("unexpected persist error: %s", persisted.Err)
	}

	lines, found, err := LoadIndexEntry(indexPath, "k1")
	if err != nil {
		t.Fatalf("LoadIndexEntry: %v", err)
	}

	if !found || len(lines) != 1 || lines[0] != "hello" {
		t.Fatalf("unexpected indexed lines: found=%v lines=%v", found, lines)
	}
}
