package testsupport

import (
	"bufio"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRichFixtureJSONLShape(t *testing.T) {
	fixtureRoot := filepath.Join(TestdataRoot(), "fixtures", "rich")
	var jsonlFiles []string

	if err := filepath.WalkDir(fixtureRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".jsonl" {
			jsonlFiles = append(jsonlFiles, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk fixture root: %v", err)
	}

	if len(jsonlFiles) < 20 {
		t.Fatalf("expected rich fixture set, got only %d jsonl files", len(jsonlFiles))
	}

	for _, p := range jsonlFiles {
		p := p
		t.Run(strings.TrimPrefix(p, fixtureRoot+string(filepath.Separator)), func(t *testing.T) {
			checkFixtureFile(t, p)
		})
	}
}

func checkFixtureFile(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer func() { _ = f.Close() }()

	isExpectedCorrupted := strings.HasSuffix(path, string(filepath.Separator)+"bad.jsonl")
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNo := 0
	seenNonEmpty := false
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		seenNonEmpty = true

		if isExpectedCorrupted {
			if jsontext.Value([]byte(line)).IsValid() {
				t.Fatalf("corrupted fixture should contain invalid json at line %d", lineNo)
			}
			return
		}

		if !jsontext.Value([]byte(line)).IsValid() {
			t.Fatalf("invalid json at line %d", lineNo)
		}

		if lineNo == 1 && strings.Contains(filepath.Base(path), "rollout-") {
			var meta struct {
				Type    string `json:"type"`
				Payload struct {
					ID        string `json:"id"`
					Timestamp string `json:"timestamp"`
				} `json:"payload"`
			}
			if err := json.Unmarshal([]byte(line), &meta); err != nil {
				t.Fatalf("unmarshal first line: %v", err)
			}
			if meta.Type != "session_meta" {
				t.Fatalf("first line should be session_meta, got %q", meta.Type)
			}
			if strings.TrimSpace(meta.Payload.ID) == "" {
				t.Fatal("session_meta payload.id is required")
			}
			if strings.TrimSpace(meta.Payload.Timestamp) == "" {
				t.Fatal("session_meta payload.timestamp is required")
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan fixture: %v", err)
	}
	if !seenNonEmpty {
		t.Fatal("fixture file is empty")
	}
}
