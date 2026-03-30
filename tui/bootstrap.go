package tui

import (
	"container/list"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/tui/runtime"
	"github.com/MysticalDevil/codexsm/usecase"
)

type commandInput struct {
	SessionsRoot    string
	TrashRoot       string
	LogFile         string
	ScanLimit       int
	ViewLimit       int
	GroupBy         string
	Source          string
	ThemeName       string
	ThemeColors     []string
	DryRun          bool
	Confirm         bool
	Yes             bool
	HardDelete      bool
	MaxBatch        int
	MaxBatchChanged bool
}

func runCommand(deps CommandDeps, in commandInput) error {
	sessionsRoot, err := resolvePathOrDefault(in.SessionsRoot, deps.ResolveSessionsRoot)
	if err != nil {
		return err
	}

	trashRoot, err := resolvePathOrDefault(in.TrashRoot, deps.ResolveTrashRoot)
	if err != nil {
		return err
	}

	logFile, err := resolvePathOrDefault(in.LogFile, deps.ResolveLogFile)
	if err != nil {
		return err
	}

	source, err := resolveSource(in.Source, deps.TUIConfig.Source)
	if err != nil {
		return err
	}

	if in.ScanLimit < 0 {
		return fmt.Errorf("invalid --scan-limit value %d", in.ScanLimit)
	}

	if in.ViewLimit < 0 {
		return fmt.Errorf("invalid --view-limit value %d", in.ViewLimit)
	}

	scanRoot := sessionsRoot
	if source == "trash" {
		scanRoot = filepath.Join(trashRoot, "sessions")
	}

	result, err := usecase.LoadTUISessions(usecase.LoadTUISessionsInput{
		SessionsRoot: scanRoot,
		ScanLimit:    in.ScanLimit,
		ViewLimit:    in.ViewLimit,
	})
	if err != nil {
		return err
	}

	home, _ := config.ResolvePath("~")

	groupBy := in.GroupBy
	if strings.TrimSpace(groupBy) == "" {
		groupBy = strings.TrimSpace(deps.TUIConfig.GroupBy)
	}

	mode, err := normalizeTUIGroupBy(groupBy)
	if err != nil {
		return err
	}

	theme, err := resolveTUITheme(deps.TUIConfig.Theme, deps.TUIConfig.Colors, in.ThemeName, in.ThemeColors)
	if err != nil {
		return err
	}

	previewIndex, err := config.ResolvePath("~/.codex/codexsm/index/preview.v1.jsonl")
	if err != nil {
		previewIndex = ""
	}

	m := tuiModel{
		sessions:           result.Items,
		collapsedGroups:    make(map[string]bool),
		home:               home,
		sessionsRoot:       sessionsRoot,
		status:             "Ready. Press q to quit.",
		previewCache:       make(map[string][]string),
		previewNodes:       make(map[string]*list.Element),
		previewBytesBudget: 8 << 20,
		focus:              focusTree,
		ultraPane:          ultraPaneTree,
		groupBy:            mode,
		source:             source,
		theme:              theme,
		previewIndex:       previewIndex,
		indexCap:           5000,
		trashRoot:          trashRoot,
		logFile:            logFile,
		dryRun:             in.DryRun,
		confirm:            in.Confirm,
		yes:                in.Yes,
		hardDelete:         in.HardDelete,
		maxBatch:           in.MaxBatch,
		maxBatchChanged:    in.MaxBatchChanged,
	}
	m.rebuildTree()

	runtimeFactory := deps.NewRuntime
	if runtimeFactory == nil {
		runtimeFactory = runtime.NewBubbleTea
	}

	err = runtimeFactory().Run(&m)

	return err
}

func resolvePathOrDefault(raw string, fallback func() (string, error)) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback()
	}

	return config.ResolvePath(raw)
}

func resolveSource(flagValue, configValue string) (string, error) {
	source := strings.TrimSpace(flagValue)
	if source == "" {
		source = strings.TrimSpace(configValue)
	}

	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		source = "sessions"
	}

	if source != "sessions" && source != "trash" {
		return "", fmt.Errorf("invalid --source %q (allowed: sessions, trash)", flagValue)
	}

	return source, nil
}
