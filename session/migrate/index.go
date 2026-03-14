package migrate

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type threadRow struct {
	ID               string
	RolloutPath      string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Source           string
	ModelProvider    string
	Cwd              string
	Title            string
	SandboxPolicy    string
	ApprovalMode     string
	TokensUsed       int64
	HasUserEvent     int64
	Archived         int64
	ArchivedAt       sql.NullInt64
	GitSHA           sql.NullString
	GitBranch        string
	GitOriginURL     sql.NullString
	CLIVersion       string
	FirstUserMessage string
	AgentNickname    sql.NullString
	AgentRole        sql.NullString
	MemoryMode       string
}

type executedMigration struct {
	source threadRow
	dest   threadRow
}

func openMigrationDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if err := verifyThreadsSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func verifyThreadsSchema(db *sql.DB) error {
	rows, err := db.Query(sqlThreadsTableInfo)
	if err != nil {
		return fmt.Errorf("inspect threads schema: %w", err)
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("scan threads schema: %w", err)
		}
		cols[name] = true
	}
	required := []string{
		"id", "rollout_path", "created_at", "updated_at", "source", "model_provider", "cwd", "title",
		"sandbox_policy", "approval_mode", "tokens_used", "has_user_event", "archived", "archived_at",
		"git_sha", "git_branch", "git_origin_url", "cli_version", "first_user_message",
		"agent_nickname", "agent_role", "memory_mode",
	}
	for _, name := range required {
		if !cols[name] {
			return fmt.Errorf("Codex state schema incompatible: missing threads.%s", name)
		}
	}
	return nil
}

func listThreadRowsByCWD(dbPath, cwd string) ([]threadRow, error) {
	db, err := openMigrationDB(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(sqlListThreadsByCWD, cwd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []threadRow
	for rows.Next() {
		var createdAt, updatedAt int64
		var row threadRow
		if err := rows.Scan(
			&row.ID, &row.RolloutPath, &createdAt, &updatedAt, &row.Source, &row.ModelProvider, &row.Cwd, &row.Title,
			&row.SandboxPolicy, &row.ApprovalMode, &row.TokensUsed, &row.HasUserEvent, &row.Archived, &row.ArchivedAt,
			&row.GitSHA, &row.GitBranch, &row.GitOriginURL, &row.CLIVersion, &row.FirstUserMessage,
			&row.AgentNickname, &row.AgentRole, &row.MemoryMode,
		); err != nil {
			return nil, err
		}
		row.CreatedAt = time.Unix(createdAt, 0).UTC()
		row.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		out = append(out, row)
	}
	return out, rows.Err()
}

func insertMigratedThreads(dbPath string, migrations []executedMigration) error {
	db, err := openMigrationDB(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(sqlInsertThread)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, migration := range migrations {
		dest := migration.dest
		if _, err := stmt.Exec(
			dest.ID, dest.RolloutPath, dest.CreatedAt.Unix(), dest.UpdatedAt.Unix(), dest.Source, dest.ModelProvider, dest.Cwd, dest.Title,
			dest.SandboxPolicy, dest.ApprovalMode, dest.TokensUsed, dest.HasUserEvent, dest.Archived, nullableInt64(dest.ArchivedAt),
			nullableString(dest.GitSHA), dest.GitBranch, nullableString(dest.GitOriginURL), dest.CLIVersion, dest.FirstUserMessage,
			nullableString(dest.AgentNickname), nullableString(dest.AgentRole), dest.MemoryMode,
		); err != nil {
			return fmt.Errorf("insert cloned thread for %s: %w", filepath.Base(dest.RolloutPath), err)
		}
	}
	return tx.Commit()
}

func nullableString(v sql.NullString) any {
	if v.Valid {
		return v.String
	}
	return nil
}

func nullableInt64(v sql.NullInt64) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}
