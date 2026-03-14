package migrate

import (
	"fmt"
	"os"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

// MigrateBatchMapping describes one source-to-target migration mapping loaded from TOML.
type MigrateBatchMapping struct {
	FromCWD string `toml:"from"`
	ToCWD   string `toml:"to"`
	Branch  string `toml:"branch"`
}

// MigrateBatchOptions controls file-driven batch migration execution.
type MigrateBatchOptions struct {
	FilePath     string
	SessionsRoot string
	StateDBPath  string
	Limit        int
	Since        time.Time
	HasSince     bool
	DryRun       bool
	Confirm      bool
	PrintCreated bool
}

// MigrateBatchItemResult records the outcome of one mapping in file order.
type MigrateBatchItemResult struct {
	Mapping MigrateBatchMapping
	Result  MigrateResult
	Err     error
}

// MigrateBatchResult is the aggregate result for a file-driven batch migration run.
type MigrateBatchResult struct {
	DryRun        bool
	PrintCreated  bool
	TotalMappings int
	Succeeded     int
	Failed        int
	Matched       int
	Created       int
	Skipped       int
	Items         []MigrateBatchItemResult
}

type migrateBatchFile struct {
	Mappings []MigrateBatchMapping `toml:"mapping"`
}

// MigrateSessionsBatch executes one migration file in declaration order.
func MigrateSessionsBatch(opts MigrateBatchOptions) (MigrateBatchResult, error) {
	opts.FilePath = strings.TrimSpace(opts.FilePath)
	opts.SessionsRoot = strings.TrimSpace(opts.SessionsRoot)

	opts.StateDBPath = strings.TrimSpace(opts.StateDBPath)
	if opts.FilePath == "" {
		return MigrateBatchResult{}, fmt.Errorf("migration file path is required")
	}

	if opts.SessionsRoot == "" {
		return MigrateBatchResult{}, fmt.Errorf("sessions root is required")
	}

	if opts.StateDBPath == "" {
		return MigrateBatchResult{}, fmt.Errorf("codex state db path is required")
	}

	if !opts.DryRun && !opts.Confirm {
		return MigrateBatchResult{}, fmt.Errorf("real migration requires --confirm")
	}

	mappings, err := loadMigrateBatchMappings(opts.FilePath)
	if err != nil {
		return MigrateBatchResult{}, err
	}

	result := MigrateBatchResult{
		DryRun:        opts.DryRun,
		PrintCreated:  opts.PrintCreated,
		TotalMappings: len(mappings),
		Items:         make([]MigrateBatchItemResult, 0, len(mappings)),
	}

	var firstErr error

	for _, mapping := range mappings {
		item := MigrateBatchItemResult{Mapping: mapping}
		item.Result, item.Err = MigrateSessions(MigrateOptions{
			FromCWD:      mapping.FromCWD,
			ToCWD:        mapping.ToCWD,
			Branch:       mapping.Branch,
			SessionsRoot: opts.SessionsRoot,
			StateDBPath:  opts.StateDBPath,
			Limit:        opts.Limit,
			Since:        opts.Since,
			HasSince:     opts.HasSince,
			DryRun:       opts.DryRun,
			Confirm:      opts.Confirm,
			PrintCreated: opts.PrintCreated,
		})

		result.Items = append(result.Items, item)
		if item.Err != nil {
			result.Failed++

			if firstErr == nil {
				firstErr = item.Err
			}

			if !opts.DryRun {
				break
			}

			continue
		}

		result.Succeeded++
		result.Matched += item.Result.Matched
		result.Created += item.Result.Created
		result.Skipped += item.Result.Skipped
	}

	if firstErr != nil {
		if opts.DryRun {
			return result, fmt.Errorf("%d migration mapping(s) failed", result.Failed)
		}

		return result, firstErr
	}

	return result, nil
}

func loadMigrateBatchMappings(path string) ([]MigrateBatchMapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg migrateBatchFile
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse migrate file: %w", err)
	}

	if len(cfg.Mappings) == 0 {
		return nil, fmt.Errorf("migration file must contain at least one [[mapping]] entry")
	}

	mappings := make([]MigrateBatchMapping, 0, len(cfg.Mappings))
	for i, mapping := range cfg.Mappings {
		mapping.FromCWD = strings.TrimSpace(mapping.FromCWD)
		mapping.ToCWD = strings.TrimSpace(mapping.ToCWD)

		mapping.Branch = strings.TrimSpace(mapping.Branch)
		if mapping.FromCWD == "" {
			return nil, fmt.Errorf("mapping %d: from is required", i+1)
		}

		if mapping.ToCWD == "" {
			return nil, fmt.Errorf("mapping %d: to is required", i+1)
		}

		mappings = append(mappings, mapping)
	}

	return mappings, nil
}
