// Package config provides the `codexsm config` command tree.
package config

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	cliutil "github.com/MysticalDevil/codexsm/cli/util"
	appconfig "github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/tui"
	"github.com/spf13/cobra"
)

type showOutput struct {
	Path      string              `json:"path"`
	Exists    bool                `json:"exists"`
	Config    appconfig.AppConfig `json:"config"`
	Effective *effectiveRuntime   `json:"effective,omitempty"`
}

type effectiveRuntime struct {
	SessionsRoot string `json:"sessions_root"`
	TrashRoot    string `json:"trash_root"`
	LogFile      string `json:"log_file"`
}

// NewCommand builds the config command tree.
func NewCommand(
	resolveSessionsRoot func() (string, error),
	resolveTrashRoot func() (string, error),
	resolveLogFile func() (string, error),
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage codexsm config file",
		Long: "Inspect and manage the user config file.\n\n" +
			"Config path:\n" +
			"  - $CSM_CONFIG when set\n" +
			"  - ~/.config/codexsm/config.json by default",
		Example: "  codexsm config show\n" +
			"  codexsm config show --resolved\n" +
			"  codexsm config init\n" +
			"  codexsm config init --force\n" +
			"  codexsm config validate",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newShowCmd(resolveSessionsRoot, resolveTrashRoot, resolveLogFile))
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newValidateCmd())

	return cmd
}

func newShowCmd(
	resolveSessionsRoot func() (string, error),
	resolveTrashRoot func() (string, error),
	resolveLogFile func() (string, error),
) *cobra.Command {
	var resolved bool

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Print config content",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := appconfig.AppConfigPath()
			if err != nil {
				return err
			}

			cfg, err := appconfig.LoadAppConfig()
			if err != nil {
				return err
			}

			_, statErr := os.Stat(p)

			exists := statErr == nil
			if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
				return fmt.Errorf("stat config %s: %w", p, statErr)
			}

			out := showOutput{
				Path:   p,
				Exists: exists,
				Config: cfg,
			}

			if resolved {
				sessionsRoot, err := resolveSessionsRoot()
				if err != nil {
					return err
				}

				trashRoot, err := resolveTrashRoot()
				if err != nil {
					return err
				}

				logFile, err := resolveLogFile()
				if err != nil {
					return err
				}

				out.Effective = &effectiveRuntime{
					SessionsRoot: sessionsRoot,
					TrashRoot:    trashRoot,
					LogFile:      logFile,
				}
			}

			data, err := cliutil.MarshalPrettyJSON(out)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))

			return err
		},
	}
	cmd.Flags().BoolVar(&resolved, "resolved", false, "include effective runtime paths after applying defaults")

	return cmd
}

func newInitCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write a starter config template",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := appconfig.AppConfigPath()
			if err != nil {
				return err
			}

			if !force {
				if _, err := os.Stat(p); err == nil {
					return fmt.Errorf("config file already exists: %s (use --force to overwrite)", p)
				} else if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("stat config %s: %w", p, err)
				}
			}

			data, err := cliutil.MarshalPrettyJSON(DefaultAppConfigTemplate())
			if err != nil {
				return err
			}

			data = append(data, '\n')

			if dryRun {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "dry-run: would write %s\n", p); err != nil {
					return err
				}

				_, err := cmd.OutOrStdout().Write(data)

				return err
			}

			if err := appconfig.EnsureConfigDir(); err != nil {
				return err
			}

			if err := cliutil.WriteFileAtomic(p, data, 0o644); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "initialized config: %s\n", p)

			return err
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing config file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print template without writing file")

	return cmd
}

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate config schema and key values",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := appconfig.AppConfigPath()
			if err != nil {
				return err
			}

			raw, err := os.ReadFile(p)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("config file does not exist: %s", p)
				}

				return fmt.Errorf("read config %s: %w", p, err)
			}

			var cfg appconfig.AppConfig
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return fmt.Errorf("parse config %s: %w", p, err)
			}

			if err := ValidateAppConfig(cfg); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "valid: %s\n", p)

			return err
		},
	}

	return cmd
}

// DefaultAppConfigTemplate returns the default template for `config init`.
func DefaultAppConfigTemplate() appconfig.AppConfig {
	return appconfig.AppConfig{
		SessionsRoot: "~/.codex/sessions",
		TrashRoot:    "~/.codex/trash",
		LogFile:      "~/.codex/codexsm/logs/actions.log",
		TUI: appconfig.TUIConfig{
			GroupBy: "host",
			Theme:   tui.DefaultThemeName(),
			Source:  "sessions",
			Colors:  map[string]string{},
		},
	}
}

// ValidateAppConfig validates config fields and enum values.
func ValidateAppConfig(cfg appconfig.AppConfig) error {
	var errs []error

	checkPath := func(name, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}

		if _, err := appconfig.ResolveConfigPath(value); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	checkPath("sessions_root", cfg.SessionsRoot)
	checkPath("trash_root", cfg.TrashRoot)
	checkPath("log_file", cfg.LogFile)

	if v := strings.ToLower(strings.TrimSpace(cfg.TUI.GroupBy)); v != "" {
		allowed := []string{"host", "day", "month"}
		if !slices.Contains(allowed, v) {
			errs = append(errs, fmt.Errorf("tui.group_by: invalid value %q (allowed: %s)", cfg.TUI.GroupBy, strings.Join(allowed, ", ")))
		}
	}

	if v := strings.ToLower(strings.TrimSpace(cfg.TUI.Source)); v != "" && v != "sessions" && v != "trash" {
		errs = append(errs, fmt.Errorf("tui.source: invalid value %q (allowed: sessions, trash)", cfg.TUI.Source))
	}

	if err := tui.ValidateTheme(cfg.TUI.Theme, cfg.TUI.Colors, "", nil); err != nil {
		errs = append(errs, fmt.Errorf("tui.theme/tui.colors: %w", err))
	}

	for k, v := range cfg.TUI.Colors {
		if strings.TrimSpace(k) == "" {
			errs = append(errs, errors.New("tui.colors: key must not be empty"))
			continue
		}

		if strings.TrimSpace(v) == "" {
			errs = append(errs, fmt.Errorf("tui.colors.%s: value must not be empty", k))
		}
	}

	return errors.Join(errs...)
}
