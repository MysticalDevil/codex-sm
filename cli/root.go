// Package cli wires codex-sm commands to the internal session and audit services.
package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Version is the application version and is usually injected at build time.
var Version = "dev"

// NewRootCmd builds the top-level codex-sm command and registers all subcommands.
func NewRootCmd() *cobra.Command {
	var (
		logLevel  string
		logFormat string
	)

	cmd := &cobra.Command{
		Use:   "codex-sm",
		Short: "Codex session manager",
		Long: "codex-sm manages local Codex sessions.\n\n" +
			"Build/install requires GOEXPERIMENT=jsonv2.\n\n" +
			"Use `codex-sm help <command>` to view details for a subcommand.\n" +
			"Examples: `codex-sm help delete`, `codex-sm help list`, `codex-sm help group`, `codex-sm help doctor`, `codex-sm help version`.",
		Example: "  codex-sm list\n" +
			"  codex-sm tui\n" +
			"  codex-sm group --by day\n" +
			"  codex-sm delete --id <session_id>\n" +
			"  codex-sm doctor\n" +
			"  codex-sm version --short\n" +
			"  codex-sm help delete",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogger(logFormat, logLevel, cmd.ErrOrStderr()); err != nil {
				return err
			}
			return loadRuntimeConfig()
		},
	}
	cmd.PersistentFlags().StringVar(&logLevel, "log-level", "warn", "log level: debug|info|warn|error")
	cmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format: text|json")

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGroupCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newRestoreCmd())
	cmd.AddCommand(newTUICmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newDoctorCmd())
	applyHelpStyles(cmd)
	return cmd
}

func applyHelpStyles(root *cobra.Command) {
	helpTemplate := buildHelpTemplate()
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		c.SetHelpTemplate(helpTemplate)
		for _, sc := range c.Commands() {
			walk(sc)
		}
	}
	walk(root)
}

func buildHelpTemplate() string {
	cyan := ansiCyanBold
	dim := ansiDim
	reset := ansiReset

	section := func(title string) string {
		return fmt.Sprintf("%s%s%s", cyan, title, reset)
	}

	var b strings.Builder
	b.WriteString("{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}\n\n{{end}}")
	b.WriteString("{{if or .Runnable .HasSubCommands}}")
	b.WriteString(section("Usage:"))
	b.WriteString("\n  {{.UseLine}}\n{{end}}")

	b.WriteString("{{if .HasAvailableSubCommands}}\n")
	b.WriteString(section("Available Commands:"))
	b.WriteString("\n{{range .Commands}}{{if (and .IsAvailableCommand (not .Hidden))}}  {{rpad .Name .NamePadding }} {{.Short}}\n{{end}}{{end}}{{end}}")

	b.WriteString("{{if .HasAvailableLocalFlags}}\n")
	b.WriteString(section("Flags:"))
	b.WriteString("\n{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}\n{{end}}")

	b.WriteString("{{if .HasAvailableInheritedFlags}}\n")
	b.WriteString(section("Global Flags:"))
	b.WriteString("\n{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}\n{{end}}")

	b.WriteString("{{if .HasExample}}\n")
	b.WriteString(section("Examples:"))
	b.WriteString("\n{{.Example}}\n{{end}}")

	b.WriteString("{{if .HasHelpSubCommands}}\n")
	b.WriteString(section("Additional Help Topics:"))
	b.WriteString("\n{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}\n{{end}}{{end}}{{end}}")

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%sUse \"%s{{.CommandPath}} [command] --help%s\" for more information about a command.\n", dim, cyan, reset))
	return b.String()
}
