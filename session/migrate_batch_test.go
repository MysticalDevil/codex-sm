package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMigrateBatchMappings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migrate.toml")
	content := `
[[mapping]]
from = "/old/a"
to = "/new/a"
branch = "main"

[[mapping]]
from = "/old/b"
to = "/new/b"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := loadMigrateBatchMappings(path)
	if err != nil {
		t.Fatalf("load mappings: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected mapping count: %d", len(got))
	}
	if got[0].FromCWD != "/old/a" || got[0].ToCWD != "/new/a" || got[0].Branch != "main" {
		t.Fatalf("unexpected first mapping: %+v", got[0])
	}
	if got[1].FromCWD != "/old/b" || got[1].ToCWD != "/new/b" || got[1].Branch != "" {
		t.Fatalf("unexpected second mapping: %+v", got[1])
	}
}

func TestLoadMigrateBatchMappingsRejectsInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migrate.toml")
	if err := os.WriteFile(path, []byte("[[mapping]]\nfrom = \"/old\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := loadMigrateBatchMappings(path)
	if err == nil || !strings.Contains(err.Error(), "to is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMigrateSessionsBatchDryRunContinuesAfterFailures(t *testing.T) {
	root := t.TempDir()
	sessionsRoot := filepath.Join(root, "sessions")
	srcRollout := filepath.Join(sessionsRoot, "2026", "03", "10", "rollout-2026-03-10-old-id.jsonl")
	if err := os.MkdirAll(filepath.Dir(srcRollout), 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"old-id","timestamp":"2026-03-10T01:00:00Z","cwd":"/old"}}`,
		`{"type":"turn_context","payload":{"cwd":"/old"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(srcRollout, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(root, "state.sqlite")
	createMigrationTestDB(t, dbPath)
	insertThreadRow(t, dbPath, testThreadRow("old-id", srcRollout, "/old", "main", 1773072000))

	filePath := filepath.Join(root, "migrate.toml")
	fileContent := `
[[mapping]]
from = "/old"
to = "/target/a"

[[mapping]]
from = "/missing"
to = "/target/b"
`
	if err := os.WriteFile(filePath, []byte(fileContent), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := MigrateSessionsBatch(MigrateBatchOptions{
		FilePath:     filePath,
		SessionsRoot: sessionsRoot,
		StateDBPath:  dbPath,
		DryRun:       true,
	})
	if err == nil || !strings.Contains(err.Error(), "1 migration mapping(s) failed") {
		t.Fatalf("unexpected batch error: %v", err)
	}
	if result.TotalMappings != 2 || result.Succeeded != 1 || result.Failed != 1 {
		t.Fatalf("unexpected batch result: %+v", result)
	}
	if len(result.Items) != 2 || result.Items[1].Err == nil {
		t.Fatalf("expected second mapping failure: %+v", result.Items)
	}
}

func TestMigrateSessionsBatchRealRunStopsOnFirstFailure(t *testing.T) {
	root := t.TempDir()
	sessionsRoot := filepath.Join(root, "sessions")
	srcRollout := filepath.Join(sessionsRoot, "2026", "03", "10", "rollout-2026-03-10-old-id.jsonl")
	if err := os.MkdirAll(filepath.Dir(srcRollout), 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"old-id","timestamp":"2026-03-10T01:00:00Z","cwd":"/old"}}`,
		`{"type":"turn_context","payload":{"cwd":"/old"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(srcRollout, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(root, "state.sqlite")
	createMigrationTestDB(t, dbPath)
	insertThreadRow(t, dbPath, testThreadRow("old-id", srcRollout, "/old", "main", 1773072000))

	filePath := filepath.Join(root, "migrate.toml")
	fileContent := `
[[mapping]]
from = "/old"
to = "` + filepath.Join(root, "dest-a") + `"

[[mapping]]
from = "/missing"
to = "` + filepath.Join(root, "dest-b") + `"

[[mapping]]
from = "/old"
to = "` + filepath.Join(root, "dest-c") + `"
`
	if err := os.WriteFile(filePath, []byte(fileContent), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := MigrateSessionsBatch(MigrateBatchOptions{
		FilePath:     filePath,
		SessionsRoot: sessionsRoot,
		StateDBPath:  dbPath,
		DryRun:       false,
		Confirm:      true,
	})
	if err == nil || !strings.Contains(err.Error(), `no sessions matched source cwd "/missing"`) {
		t.Fatalf("unexpected batch error: %v", err)
	}
	if result.Succeeded != 1 || result.Failed != 1 || len(result.Items) != 2 {
		t.Fatalf("unexpected partial result: %+v", result)
	}
}
