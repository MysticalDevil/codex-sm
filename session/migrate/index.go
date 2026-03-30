package migrate

import (
	"database/sql"
	"database/sql/driver"
	"errors"
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
		return nil, fmt.Errorf("open migration db %q: %w", path, err)
	}

	if err := db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping db: %w (close db: %w)", err, closeErr)
		}

		return nil, fmt.Errorf("ping migration db %q: %w", path, err)
	}

	if err := verifyThreadsSchema(db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("verify schema: %w (close db: %w)", err, closeErr)
		}

		return nil, fmt.Errorf("verify threads schema in %q: %w", path, err)
	}

	return db, nil
}

func verifyThreadsSchema(db *sql.DB) (err error) {
	rows, err := db.Query(sqlThreadsTableInfo)
	if err != nil {
		return fmt.Errorf("inspect threads schema: %w", err)
	}

	defer func() {
		closeErr := rows.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close threads schema rows: %w", closeErr)
		}
	}()

	cols := map[string]bool{}

	for rows.Next() {
		var (
			cid         int
			name, typ   string
			notnull, pk int
			dflt        sql.NullString
		)

		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("scan threads schema: %w", err)
		}

		cols[name] = true
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate threads schema rows: %w", err)
	}

	required := []string{
		"id", "rollout_path", "created_at", "updated_at", "source", "model_provider", "cwd", "title",
		"sandbox_policy", "approval_mode", "tokens_used", "has_user_event", "archived", "archived_at",
		"git_sha", "git_branch", "git_origin_url", "cli_version", "first_user_message",
		"agent_nickname", "agent_role", "memory_mode",
	}
	for _, name := range required {
		if !cols[name] {
			return fmt.Errorf("codex state schema incompatible: missing threads.%s", name)
		}
	}

	return nil
}

func listThreadRowsByCWD(dbPath, cwd string) (out []threadRow, err error) {
	db, err := openMigrationDB(dbPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		closeErr := db.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close migration db: %w", closeErr)
		}
	}()

	rows, err := db.Query(sqlListThreadsByCWD, cwd)
	if err != nil {
		return nil, fmt.Errorf("query thread rows for cwd %q: %w", cwd, err)
	}

	defer func() {
		closeErr := rows.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close thread rows: %w", closeErr)
		}
	}()

	for rows.Next() {
		var (
			createdAt, updatedAt int64
			row                  threadRow
		)

		if err := rows.Scan(
			&row.ID, &row.RolloutPath, &createdAt, &updatedAt, &row.Source, &row.ModelProvider, &row.Cwd, &row.Title,
			&row.SandboxPolicy, &row.ApprovalMode, &row.TokensUsed, &row.HasUserEvent, &row.Archived, &row.ArchivedAt,
			&row.GitSHA, &row.GitBranch, &row.GitOriginURL, &row.CLIVersion, &row.FirstUserMessage,
			&row.AgentNickname, &row.AgentRole, &row.MemoryMode,
		); err != nil {
			return nil, fmt.Errorf("scan thread row for cwd %q: %w", cwd, err)
		}

		row.CreatedAt = time.Unix(createdAt, 0).UTC()
		row.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		out = append(out, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate thread rows for cwd %q: %w", cwd, err)
	}

	return out, nil
}

func insertMigratedThreads(dbPath string, migrations []executedMigration) (err error) {
	db, err := openMigrationDB(dbPath)
	if err != nil {
		return err
	}

	defer func() {
		closeErr := db.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close migration db: %w", closeErr)
		}
	}()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}

	defer func() {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) && err == nil {
			err = fmt.Errorf("rollback migration tx: %w", rollbackErr)
		}
	}()

	stmt, err := tx.Prepare(sqlInsertThread)
	if err != nil {
		return fmt.Errorf("prepare insert thread statement: %w", err)
	}

	defer func() {
		closeErr := stmt.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close insert stmt: %w", closeErr)
		}
	}()

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

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration transaction: %w", err)
	}

	return nil
}

func nullableString(v sql.NullString) driver.Value {
	if v.Valid {
		return v.String
	}

	return nil
}

func nullableInt64(v sql.NullInt64) driver.Value {
	if v.Valid {
		return v.Int64
	}

	return nil
}
