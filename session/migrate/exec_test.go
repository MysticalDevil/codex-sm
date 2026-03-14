package migrate

import (
	"database/sql"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestRewriteMigrationLineUpdatesMetadataOnly(t *testing.T) {
	line := []byte(`{"type":"session_meta","payload":{"id":"old-id","cwd":"/old","other":"keep"}}`)
	got, err := rewriteMigrationLine(line, "old-id", "new-id", "/new")
	if err != nil {
		t.Fatalf("rewrite line: %v", err)
	}
	text := string(got)
	if !strings.Contains(text, `"id":"new-id"`) || !strings.Contains(text, `"cwd":"/new"`) || !strings.Contains(text, `"other":"keep"`) {
		t.Fatalf("unexpected rewritten line: %s", text)
	}

	raw := []byte(`{"type":"response_item","payload":{"type":"message","role":"user","text":"keep raw"}}`)
	got, err = rewriteMigrationLine(raw, "old-id", "new-id", "/new")
	if err != nil {
		t.Fatalf("rewrite response item: %v", err)
	}
	var wantObj, gotObj map[string]any
	if err := json.Unmarshal(raw, &wantObj); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if err := json.Unmarshal(got, &gotObj); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if gotObj["type"] != wantObj["type"] {
		t.Fatalf("response item type changed: %#v", gotObj)
	}
}

func TestBuildMigratedRolloutPath(t *testing.T) {
	got := buildMigratedRolloutPath("/tmp/rollout-2026-03-10-abc.jsonl", "main", "dest-id")
	if !strings.HasSuffix(got, "rollout-2026-03-10-abc-main-dest-id.jsonl") {
		t.Fatalf("unexpected rollout path: %s", got)
	}
}

func TestMigrateSessionsSkipsAlreadyMigrated(t *testing.T) {
	root := t.TempDir()
	srcRollout := filepath.Join(root, "2026", "03", "10", "rollout-2026-03-10-old-id.jsonl")
	if err := os.MkdirAll(filepath.Dir(srcRollout), 0o755); err != nil {
		t.Fatal(err)
	}
	srcContent := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"old-id","timestamp":"2026-03-10T01:00:00Z","cwd":"/old"}}`,
		`{"type":"turn_context","payload":{"cwd":"/old"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(srcRollout, []byte(srcContent), 0o644); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(root, "state.sqlite")
	createMigrationTestDB(t, dbPath)
	insertThreadRow(t, dbPath, testThreadRow("old-id", srcRollout, "/old", "main", 1773072000))
	insertThreadRow(t, dbPath, testThreadRow("migrated-id", buildMigratedRolloutPath(srcRollout, "main", "migrated-id"), "/target/main", "main", 1773072000))

	result, err := MigrateSessions(MigrateOptions{
		FromCWD:      "/old",
		ToCWD:        "/target/main",
		SessionsRoot: root,
		StateDBPath:  dbPath,
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("migrate sessions: %v", err)
	}
	if result.Matched != 0 || result.Skipped != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func createMigrationTestDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	schema := `
CREATE TABLE threads (
    id TEXT PRIMARY KEY,
    rollout_path TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    source TEXT NOT NULL,
    model_provider TEXT NOT NULL,
    cwd TEXT NOT NULL,
    title TEXT NOT NULL,
    sandbox_policy TEXT NOT NULL,
    approval_mode TEXT NOT NULL,
    tokens_used INTEGER NOT NULL DEFAULT 0,
    has_user_event INTEGER NOT NULL DEFAULT 0,
    archived INTEGER NOT NULL DEFAULT 0,
    archived_at INTEGER,
    git_sha TEXT,
    git_branch TEXT,
    git_origin_url TEXT,
    cli_version TEXT NOT NULL DEFAULT '',
    first_user_message TEXT NOT NULL DEFAULT '',
    agent_nickname TEXT,
    agent_role TEXT,
    memory_mode TEXT NOT NULL DEFAULT 'enabled'
);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}
}

func testThreadRow(id, rolloutPath, cwd, branch string, updatedAt int64) threadRow {
	ts := time.Unix(updatedAt, 0).UTC()
	return threadRow{
		ID:               id,
		RolloutPath:      rolloutPath,
		CreatedAt:        ts,
		UpdatedAt:        ts,
		Source:           "cli",
		ModelProvider:    "openai",
		Cwd:              cwd,
		Title:            "title",
		SandboxPolicy:    "{}",
		ApprovalMode:     "on-request",
		TokensUsed:       42,
		HasUserEvent:     1,
		Archived:         0,
		GitBranch:        branch,
		CLIVersion:       "0.112.0",
		FirstUserMessage: "hello",
		MemoryMode:       "enabled",
	}
}

func insertThreadRow(t *testing.T, path string, row threadRow) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
INSERT INTO threads (
    id, rollout_path, created_at, updated_at, source, model_provider, cwd, title,
    sandbox_policy, approval_mode, tokens_used, has_user_event, archived, archived_at,
    git_sha, git_branch, git_origin_url, cli_version, first_user_message, agent_nickname, agent_role, memory_mode
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.ID, row.RolloutPath, row.CreatedAt.Unix(), row.UpdatedAt.Unix(), row.Source, row.ModelProvider, row.Cwd, row.Title,
		row.SandboxPolicy, row.ApprovalMode, row.TokensUsed, row.HasUserEvent, row.Archived, nil,
		nil, row.GitBranch, nil, row.CLIVersion, row.FirstUserMessage, nil, nil, row.MemoryMode,
	)
	if err != nil {
		t.Fatal(err)
	}
}
