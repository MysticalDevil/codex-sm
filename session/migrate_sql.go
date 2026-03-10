package session

const (
	sqlThreadsTableInfo = `PRAGMA table_info(threads)`

	sqlListThreadsByCWD = `
		SELECT id, rollout_path, created_at, updated_at, source, model_provider, cwd, title,
		       sandbox_policy, approval_mode, tokens_used, has_user_event, archived, archived_at,
		       git_sha, git_branch, git_origin_url, cli_version, first_user_message,
		       agent_nickname, agent_role, memory_mode
		FROM threads
		WHERE cwd = ?
		ORDER BY updated_at DESC, id DESC
	`

	sqlInsertThread = `
		INSERT INTO threads (
			id, rollout_path, created_at, updated_at, source, model_provider, cwd, title,
			sandbox_policy, approval_mode, tokens_used, has_user_event, archived, archived_at,
			git_sha, git_branch, git_origin_url, cli_version, first_user_message,
			agent_nickname, agent_role, memory_mode
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
)
