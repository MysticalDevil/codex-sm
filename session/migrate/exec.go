package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MigrateOptions controls session migration planning and execution.
type MigrateOptions struct {
	FromCWD      string
	ToCWD        string
	Branch       string
	SessionsRoot string
	StateDBPath  string
	Limit        int
	Since        time.Time
	HasSince     bool
	DryRun       bool
	Confirm      bool
	PrintCreated bool
}

// MigratePlan describes one source session copied to one destination session.
type MigratePlan struct {
	SourceID      string
	DestID        string
	SourceRollout string
	DestRollout   string
	SourceCWD     string
	DestCWD       string
	SourceBranch  string
	DestBranch    string
}

// MigrateResult is the user-facing migration summary.
type MigrateResult struct {
	FromCWD      string
	ToCWD        string
	DestBranch   string
	Matched      int
	Created      int
	Skipped      int
	DryRun       bool
	PrintCreated bool
	Planned      []MigratePlan
	Warnings     []string
}

type migrateCandidate struct {
	thread     threadRow
	sourceMeta migrateMeta
	destID     string
	destRoll   string
	destBranch string
}

// MigrateSessions copies sessions from one cwd to another and keeps the local thread index in sync.
func MigrateSessions(opts MigrateOptions) (MigrateResult, error) {
	opts.FromCWD = strings.TrimSpace(opts.FromCWD)
	opts.ToCWD = strings.TrimSpace(opts.ToCWD)
	opts.Branch = strings.TrimSpace(opts.Branch)
	opts.SessionsRoot = strings.TrimSpace(opts.SessionsRoot)
	opts.StateDBPath = strings.TrimSpace(opts.StateDBPath)
	if opts.FromCWD == "" {
		return MigrateResult{}, fmt.Errorf("source cwd is required")
	}
	if opts.ToCWD == "" {
		return MigrateResult{}, fmt.Errorf("destination cwd is required")
	}
	if opts.SessionsRoot == "" {
		return MigrateResult{}, fmt.Errorf("sessions root is required")
	}
	if opts.StateDBPath == "" {
		return MigrateResult{}, fmt.Errorf("codex state db path is required")
	}
	if !opts.DryRun && !opts.Confirm {
		return MigrateResult{}, fmt.Errorf("real migration requires --confirm")
	}

	result := MigrateResult{
		FromCWD:      opts.FromCWD,
		ToCWD:        opts.ToCWD,
		DestBranch:   opts.Branch,
		DryRun:       opts.DryRun,
		PrintCreated: opts.PrintCreated,
	}
	rollouts, err := collectMigrationRollouts(opts.SessionsRoot, opts.FromCWD)
	if err != nil {
		return result, err
	}

	sourceRows, err := listThreadRowsByCWD(opts.StateDBPath, opts.FromCWD)
	if err != nil {
		return result, err
	}
	targetRows, err := listThreadRowsByCWD(opts.StateDBPath, opts.ToCWD)
	if err != nil {
		return result, err
	}

	sourceByID := make(map[string]threadRow, len(sourceRows))
	for _, row := range sourceRows {
		sourceByID[row.ID] = row
	}
	for id, meta := range rollouts {
		if _, ok := sourceByID[id]; !ok {
			result.Warnings = append(result.Warnings, fmt.Sprintf("rollout exists without thread row: %s", meta.Path))
		}
	}

	targetLabel := targetRolloutLabel(opts.ToCWD)
	targetExisting := make(map[string]struct{}, len(targetRows))
	for _, row := range targetRows {
		base := filepath.Base(row.RolloutPath)
		targetExisting[base] = struct{}{}
	}

	candidates := make([]migrateCandidate, 0, len(sourceRows))
	for _, row := range sourceRows {
		meta, ok := rollouts[row.ID]
		if !ok {
			result.Warnings = append(result.Warnings, fmt.Sprintf("thread row exists without rollout file: %s", row.ID))
			continue
		}
		if opts.HasSince && row.UpdatedAt.Before(opts.Since) {
			continue
		}
		if existingRolloutForSource(row.RolloutPath, targetLabel, targetExisting) {
			result.Skipped++
			continue
		}
		destID := uuid.NewString()
		destRoll := buildMigratedRolloutPath(meta.Path, targetLabel, destID)
		destBranch := row.GitBranch
		if opts.Branch != "" {
			destBranch = opts.Branch
		}
		candidates = append(candidates, migrateCandidate{
			thread:     row,
			sourceMeta: meta,
			destID:     destID,
			destRoll:   destRoll,
			destBranch: destBranch,
		})
	}

	slices.SortStableFunc(candidates, func(a, b migrateCandidate) int {
		if a.thread.UpdatedAt.Equal(b.thread.UpdatedAt) {
			return strings.Compare(a.thread.ID, b.thread.ID)
		}
		if a.thread.UpdatedAt.After(b.thread.UpdatedAt) {
			return -1
		}
		return 1
	})

	if opts.Limit > 0 && len(candidates) > opts.Limit {
		candidates = candidates[:opts.Limit]
	}
	result.Matched = len(candidates)
	result.Planned = make([]MigratePlan, 0, len(candidates))
	for _, c := range candidates {
		result.Planned = append(result.Planned, MigratePlan{
			SourceID:      c.thread.ID,
			DestID:        c.destID,
			SourceRollout: c.sourceMeta.Path,
			DestRollout:   c.destRoll,
			SourceCWD:     opts.FromCWD,
			DestCWD:       opts.ToCWD,
			SourceBranch:  c.thread.GitBranch,
			DestBranch:    c.destBranch,
		})
	}

	if len(candidates) == 0 {
		if result.Skipped > 0 || len(result.Warnings) > 0 {
			return result, nil
		}
		if len(result.Warnings) == 0 {
			return result, fmt.Errorf("no sessions matched source cwd %q", opts.FromCWD)
		}
	}
	if opts.DryRun {
		return result, nil
	}
	if st, err := os.Stat(opts.ToCWD); err != nil || !st.IsDir() {
		result.Warnings = append(result.Warnings, fmt.Sprintf("target cwd does not currently exist on disk: %s", opts.ToCWD))
	}

	createdPaths := make([]string, 0, len(candidates))
	executed := make([]executedMigration, 0, len(candidates))
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate.destRoll); err == nil {
			return result, fmt.Errorf("destination rollout already exists: %s", candidate.destRoll)
		}
		if err := writeMigratedRollout(candidate.sourceMeta.Path, candidate.destRoll, candidate.thread.ID, candidate.destID, opts.ToCWD); err != nil {
			return result, err
		}
		createdPaths = append(createdPaths, candidate.destRoll)
		executed = append(executed, executedMigration{
			source: candidate.thread,
			dest: threadRow{
				ID:               candidate.destID,
				RolloutPath:      candidate.destRoll,
				CreatedAt:        candidate.thread.CreatedAt,
				UpdatedAt:        candidate.thread.UpdatedAt,
				Source:           candidate.thread.Source,
				ModelProvider:    candidate.thread.ModelProvider,
				Cwd:              opts.ToCWD,
				Title:            candidate.thread.Title,
				SandboxPolicy:    candidate.thread.SandboxPolicy,
				ApprovalMode:     candidate.thread.ApprovalMode,
				TokensUsed:       candidate.thread.TokensUsed,
				HasUserEvent:     candidate.thread.HasUserEvent,
				Archived:         candidate.thread.Archived,
				ArchivedAt:       candidate.thread.ArchivedAt,
				GitSHA:           candidate.thread.GitSHA,
				GitBranch:        candidate.destBranch,
				GitOriginURL:     candidate.thread.GitOriginURL,
				CLIVersion:       candidate.thread.CLIVersion,
				FirstUserMessage: candidate.thread.FirstUserMessage,
				AgentNickname:    candidate.thread.AgentNickname,
				AgentRole:        candidate.thread.AgentRole,
				MemoryMode:       candidate.thread.MemoryMode,
			},
		})
	}
	if err := insertMigratedThreads(opts.StateDBPath, executed); err != nil {
		return result, fmt.Errorf("wrote rollout files but failed to update Codex thread index: %w (created=%s)", err, strings.Join(createdPaths, ", "))
	}
	result.Created = len(executed)
	return result, nil
}

func existingRolloutForSource(sourceRollout, targetLabel string, targetExisting map[string]struct{}) bool {
	sourceStem := strings.TrimSuffix(filepath.Base(sourceRollout), filepath.Ext(sourceRollout))
	prefix := sourceStem + "-" + targetLabel + "-"
	for name := range targetExisting {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func buildMigratedRolloutPath(sourceRollout, targetLabel, destID string) string {
	dir := filepath.Dir(sourceRollout)
	stem := strings.TrimSuffix(filepath.Base(sourceRollout), filepath.Ext(sourceRollout))
	return filepath.Join(dir, stem+"-"+targetLabel+"-"+destID+".jsonl")
}

func targetRolloutLabel(dest string) string {
	label := filepath.Base(filepath.Clean(dest))
	label = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '-'
		}
	}, label)
	label = strings.Trim(label, "-_")
	if label == "" {
		return "migrated"
	}
	return label
}
