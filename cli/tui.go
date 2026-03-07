package cli

import (
	"github.com/MysticalDevil/codexsm/internal/tui/browser"
	"github.com/spf13/cobra"
)

func newTUICmd() *cobra.Command {
	return browser.NewCommand(browser.CommandDeps{
		ResolveSessionsRoot: runtimeSessionsRoot,
		ResolveTrashRoot:    runtimeTrashRoot,
		ResolveLogFile:      runtimeLogFile,
		TUIConfig:           runtimeConfig.TUI,
	})
}
