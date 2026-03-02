package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/MysticalDevil/codex-sm/internal/config"
	"github.com/MysticalDevil/codex-sm/internal/session"
	"github.com/MysticalDevil/codex-sm/internal/util"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		sessionsRoot string
		id           string
		idPrefix     string
		olderThan    string
		health       string
		format       string
		limit        int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Codex sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(sessionsRoot) == "" {
				v, err := config.DefaultSessionsRoot()
				if err != nil {
					return err
				}
				sessionsRoot = v
			} else {
				v, err := config.ResolvePath(sessionsRoot)
				if err != nil {
					return err
				}
				sessionsRoot = v
			}

			sel, err := buildSelector(id, idPrefix, olderThan, health)
			if err != nil {
				return err
			}

			sessions, err := session.ScanSessions(sessionsRoot)
			if err != nil {
				return err
			}
			filtered := session.FilterSessions(sessions, sel, time.Now())
			if limit > 0 && len(filtered) > limit {
				filtered = filtered[:limit]
			}

			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "table":
				return printTable(cmd, filtered)
			case "json":
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(filtered)
			default:
				return fmt.Errorf("unsupported format %q", format)
			}
		},
	}

	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().StringVar(&id, "id", "", "exact session id")
	cmd.Flags().StringVar(&idPrefix, "id-prefix", "", "session id prefix")
	cmd.Flags().StringVar(&olderThan, "older-than", "", "select sessions older than duration (e.g. 30d, 12h)")
	cmd.Flags().StringVar(&health, "health", "", "health filter: ok|corrupted|missing-meta")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	cmd.Flags().IntVar(&limit, "limit", 0, "max rows to print (0 means unlimited)")

	return cmd
}

func buildSelector(id, idPrefix, olderThan, health string) (session.Selector, error) {
	sel := session.Selector{
		ID:       strings.TrimSpace(id),
		IDPrefix: strings.TrimSpace(idPrefix),
	}

	if strings.TrimSpace(olderThan) != "" {
		d, err := util.ParseOlderThan(olderThan)
		if err != nil {
			return sel, err
		}
		sel.OlderThan = d
		sel.HasOlderThan = true
	}

	if strings.TrimSpace(health) != "" {
		h, err := parseHealth(health)
		if err != nil {
			return sel, err
		}
		sel.Health = h
		sel.HasHealth = true
	}

	return sel, nil
}

func parseHealth(v string) (session.Health, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case string(session.HealthOK):
		return session.HealthOK, nil
	case string(session.HealthCorrupted):
		return session.HealthCorrupted, nil
	case string(session.HealthMissingMeta):
		return session.HealthMissingMeta, nil
	default:
		return "", fmt.Errorf("invalid health %q", v)
	}
}

func printTable(cmd *cobra.Command, sessions []session.Session) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 2, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SESSION_ID\tCREATED_AT\tUPDATED_AT\tSIZE_BYTES\tHEALTH\tPATH")
	for _, s := range sessions {
		created := "-"
		if !s.CreatedAt.IsZero() {
			created = s.CreatedAt.Format(time.RFC3339)
		}
		updated := "-"
		if !s.UpdatedAt.IsZero() {
			updated = s.UpdatedAt.Format(time.RFC3339)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n", s.SessionID, created, updated, s.SizeBytes, s.Health, s.Path)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "rows=%d\n", len(sessions))
	return nil
}
