package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	var noDescriptions bool

	cmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Long: "Generate shell completion scripts for codexsm.\n\n" +
			"Examples:\n" +
			"  codexsm completion bash > ~/.local/share/bash-completion/completions/codexsm\n" +
			"  codexsm completion zsh > ${fpath[1]}/_codexsm\n" +
			"  codexsm completion fish > ~/.config/fish/completions/codexsm.fish\n" +
			"  codexsm completion powershell > codexsm.ps1",
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := strings.ToLower(strings.TrimSpace(args[0]))
			root := cmd.Root()
			out := cmd.OutOrStdout()

			switch shell {
			case "bash":
				return root.GenBashCompletionV2(out, !noDescriptions)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, !noDescriptions)
			case "powershell":
				if noDescriptions {
					return root.GenPowerShellCompletion(out)
				}

				return root.GenPowerShellCompletionWithDesc(out)
			default:
				return fmt.Errorf("unsupported shell %q (allowed: bash, zsh, fish, powershell)", shell)
			}
		},
	}
	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "disable completion descriptions where supported")

	return cmd
}
