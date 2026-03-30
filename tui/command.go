// Package tui implements the interactive terminal UI for browsing and managing sessions.
package tui

import (
	"container/list"

	"github.com/spf13/cobra"

	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/MysticalDevil/codexsm/tui/runtime"
)

type treeItemKind int

const (
	treeItemMonth treeItemKind = iota
	treeItemSession
)

type treeItem struct {
	Kind   treeItemKind
	Label  string
	Month  string
	Index  int
	Indent int
	// HostMissing marks sessions whose host path does not exist on local filesystem.
	HostMissing bool
}

type tuiFocus int

const (
	focusTree tuiFocus = iota
	focusPreview
)

type ultraPane int

const (
	ultraPaneTree ultraPane = iota
	ultraPanePreview
)

type tuiModel struct {
	sessions           []session.Session
	tree               []treeItem
	collapsedGroups    map[string]bool
	cursor             int
	offset             int
	previewOffset      int
	width              int
	height             int
	home               string
	sessionsRoot       string
	status             string
	previewCache       map[string][]string
	previewLRU         *list.List
	previewNodes       map[string]*list.Element
	previewBytesBudget int64
	previewBytesUsed   int64
	previewReqSeq      uint64
	previewReqID       uint64
	previewWait        string
	previewIndex       string
	indexCap           int
	lastPath           string
	focus              tuiFocus
	ultraPane          ultraPane
	groupBy            string
	source             string
	theme              tuiTheme
	trashRoot          string
	logFile            string
	dryRun             bool
	confirm            bool
	yes                bool
	hardDelete         bool
	maxBatch           int
	maxBatchChanged    bool
	pendingAction      string
	pendingStep        int
	pendingID          string
	pendingHost        string
	pendingGroup       string
	pendingCount       int
}

type CommandDeps struct {
	ResolveSessionsRoot func() (string, error)
	ResolveTrashRoot    func() (string, error)
	ResolveLogFile      func() (string, error)
	NewRuntime          func() runtime.Runtime
	TUIConfig           config.TUIConfig
}

func NewCommand(deps CommandDeps) *cobra.Command {
	if deps.ResolveSessionsRoot == nil {
		deps.ResolveSessionsRoot = config.DefaultSessionsRoot
	}

	if deps.ResolveTrashRoot == nil {
		deps.ResolveTrashRoot = config.DefaultTrashRoot
	}

	if deps.ResolveLogFile == nil {
		deps.ResolveLogFile = config.DefaultLogFile
	}

	var (
		sessionsRoot string
		trashRoot    string
		logFile      string
		scanLimit    int
		viewLimit    int
		groupBy      string
		source       string
		themeName    string
		themeColors  []string
		dryRun       bool
		confirm      bool
		yes          bool
		hardDelete   bool
		maxBatch     int
	)

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI session browser",
		Long: "Interactive session browser (optional).\n\n" +
			"Keys:\n" +
			"  j/k or Down/Up: move cursor\n" +
			"  g/G: first/last\n" +
			"  z: collapse/expand selected session group\n" +
			"  Z: expand all groups\n" +
			"  Ctrl+d / Ctrl+u: scroll preview\n" +
			"  d: delete current session, or selected group on a group header\n" +
			"  m: migrate sessions with missing selected host to trash\n" +
			"  r: restore current session (only when --source=trash)\n" +
			"  y/n: confirm/cancel pending action (group real delete requires 3 confirms)\n" +
			"  q: quit",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(deps, commandInput{
				SessionsRoot:    sessionsRoot,
				TrashRoot:       trashRoot,
				LogFile:         logFile,
				ScanLimit:       scanLimit,
				ViewLimit:       viewLimit,
				GroupBy:         groupBy,
				Source:          source,
				ThemeName:       themeName,
				ThemeColors:     themeColors,
				DryRun:          dryRun,
				Confirm:         confirm,
				Yes:             yes,
				HardDelete:      hardDelete,
				MaxBatch:        maxBatch,
				MaxBatchChanged: cmd.Flags().Changed("max-batch"),
			})
		},
	}

	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().StringVar(&trashRoot, "trash-root", "", "trash root directory")
	cmd.Flags().StringVar(&logFile, "log-file", "", "action log file")
	cmd.Flags().IntVar(&scanLimit, "scan-limit", 2000, "max sessions scanned then sorted for TUI (0 means unlimited)")
	cmd.Flags().IntVarP(&viewLimit, "view-limit", "l", 100, "max sessions rendered in TUI after sorting (0 means unlimited)")
	cmd.Flags().StringVar(&groupBy, "group-by", "", "tree group key: host|day|month")
	cmd.Flags().StringVar(&source, "source", "", "session source: sessions|trash")
	cmd.Flags().StringVar(&themeName, "theme", "", "TUI theme: tokyonight|catppuccin|gruvbox|onedark|nord|dracula")
	cmd.Flags().StringArrayVar(&themeColors, "theme-color", nil, "custom theme override (key=value), repeatable")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", true, "simulate delete/restore from TUI")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "required for real delete/restore from TUI")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip TUI confirmation prompts")
	cmd.Flags().BoolVar(&hardDelete, "hard", false, "hard delete on session source")
	cmd.Flags().IntVar(&maxBatch, "max-batch", 100, "max sessions allowed for one real TUI action")

	return cmd
}
