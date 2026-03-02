// Package cli wires csm commands to the internal session and audit services.
package cli

import "github.com/spf13/cobra"

const version = "0.1.0"

// NewRootCmd builds the top-level csm command and registers all subcommands.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "csm",
		Short: "Codex session manager",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newVersionCmd())
	return cmd
}
