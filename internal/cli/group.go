package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/MysticalDevil/codex-sm/internal/config"
	"github.com/MysticalDevil/codex-sm/internal/session"

	"github.com/spf13/cobra"
)

type groupStat struct {
	Group     string `json:"group"`
	Count     int    `json:"count"`
	SizeBytes int64  `json:"size_bytes"`
	Latest    string `json:"latest"`
}

func newGroupCmd() *cobra.Command {
	var (
		sessionsRoot string
		id           string
		idPrefix     string
		olderThan    string
		health       string
		by           string
		format       string
		pager        bool
		pageSize     int
		colorMode    string
	)

	cmd := &cobra.Command{
		Use:   "group",
		Short: "Group sessions by day or health",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			sessionsRoot, err = resolveOrDefault(sessionsRoot, config.DefaultSessionsRoot)
			if err != nil {
				return err
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

			stats, err := buildGroupStats(filtered, by)
			if err != nil {
				return err
			}

			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "table":
				table, err := renderGroupTable(stats, by, colorMode, cmd.OutOrStdout())
				if err != nil {
					return err
				}
				return writeWithPager(cmd.OutOrStdout(), table, pager, pageSize)
			case "json":
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(stats)
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
	cmd.Flags().StringVar(&by, "by", "day", "group key: day|health")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	cmd.Flags().BoolVar(&pager, "pager", false, "enable interactive pager")
	cmd.Flags().IntVar(&pageSize, "page-size", 10, "rows per page when --pager is enabled")
	cmd.Flags().StringVar(&colorMode, "color", "always", "color mode: auto|always|never")

	return cmd
}

func buildGroupStats(sessions []session.Session, by string) ([]groupStat, error) {
	mode := strings.ToLower(strings.TrimSpace(by))
	if mode == "" {
		mode = "day"
	}
	if mode != "day" && mode != "health" {
		return nil, fmt.Errorf("invalid --by %q (allowed: day, health)", by)
	}

	type agg struct {
		count  int
		size   int64
		latest time.Time
	}
	m := make(map[string]*agg)
	for _, s := range sessions {
		var key string
		switch mode {
		case "day":
			key = s.UpdatedAt.Local().Format("2006-01-02")
		case "health":
			key = string(s.Health)
		}
		if key == "" {
			key = "-"
		}
		a := m[key]
		if a == nil {
			a = &agg{}
			m[key] = a
		}
		a.count++
		a.size += s.SizeBytes
		if s.UpdatedAt.After(a.latest) {
			a.latest = s.UpdatedAt
		}
	}

	out := make([]groupStat, 0, len(m))
	for key, a := range m {
		latest := "-"
		if !a.latest.IsZero() {
			latest = formatDisplayTime(a.latest)
		}
		out = append(out, groupStat{Group: key, Count: a.count, SizeBytes: a.size, Latest: latest})
	}

	if mode == "day" {
		sort.Slice(out, func(i, j int) bool { return out[i].Group > out[j].Group })
	} else {
		sort.Slice(out, func(i, j int) bool {
			if out[i].Count == out[j].Count {
				return out[i].Group < out[j].Group
			}
			return out[i].Count > out[j].Count
		})
	}

	return out, nil
}

func renderGroupTable(stats []groupStat, by, colorMode string, out io.Writer) (string, error) {
	useColor := shouldUseColor(colorMode, out)
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 2, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "GROUP\tCOUNT\tSIZE\tLATEST")
	for _, g := range stats {
		_, _ = fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", g.Group, g.Count, formatBytesIEC(g.SizeBytes), g.Latest)
	}
	if err := w.Flush(); err != nil {
		return "", err
	}
	_, _ = fmt.Fprintf(&buf, "groups=%d by=%s\n", len(stats), strings.ToLower(strings.TrimSpace(by)))
	text := buf.String()
	if !useColor {
		return text, nil
	}
	return colorizeGroupedTable(text), nil
}

func colorizeGroupedTable(text string) string {
	if text == "" {
		return text
	}
	hasTrailingNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if i == 0 {
			lines[i] = colorize(line, ansiCyanBold, true)
			continue
		}
		if strings.HasPrefix(line, "groups=") {
			lines[i] = colorize(line, ansiDim, true)
		}
	}
	out := strings.Join(lines, "\n")
	if hasTrailingNewline {
		out += "\n"
	}
	return out
}
