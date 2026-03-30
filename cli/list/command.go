// Package list provides the `codexsm list` command.
package list

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	cliutil "github.com/MysticalDevil/codexsm/cli/util"
	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/MysticalDevil/codexsm/usecase"
	"github.com/spf13/cobra"
)

type RenderOptions struct {
	NoHeader  bool
	ColorMode string
	Out       io.Writer
	Columns   []Column
	HeadWidth int
}

// NewCommand builds the list command.
func NewCommand(resolveSessionsRoot func() (string, error)) *cobra.Command {
	var (
		sessionsRoot string
		id           string
		idPrefix     string
		hostContains string
		pathContains string
		headContains string
		olderThan    string
		health       string
		format       string
		limit        int
		detailed     bool
		pager        bool
		pageSize     int
		colorMode    string
		noHeader     bool
		columnInput  string
		headWidth    int
		sortBy       string
		order        string
		offset       int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Codex sessions",
		Example: "  codexsm list\n" +
			"  codexsm list --detailed\n" +
			"  codexsm list --head-width 48\n" +
			"  codexsm list --limit 0 --pager\n" +
			"  codexsm list --sort size --order asc --limit 20\n" +
			"  codexsm list --host-contains /workspace --head-contains fixture\n" +
			"  codexsm list --id-prefix 019ca9 --format json\n" +
			"  codexsm list --format csv --column session_id,updated_at,size",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if strings.TrimSpace(sessionsRoot) == "" {
				sessionsRoot, err = resolveSessionsRoot()
				if err != nil {
					return err
				}
			} else {
				sessionsRoot, err = config.ResolvePath(sessionsRoot)
				if err != nil {
					return err
				}
			}

			sel, err := cliutil.BuildSelector(id, idPrefix, hostContains, pathContains, headContains, olderThan, health)
			if err != nil {
				return err
			}

			if pager && !cmd.Flags().Changed("limit") {
				limit = 0
			}

			result, err := usecase.ListSessions(usecase.ListInput{
				SessionsRoot: sessionsRoot,
				Selector:     sel,
				SortBy:       sortBy,
				Order:        order,
				Offset:       offset,
				Limit:        limit,
			})
			if err != nil {
				return err
			}

			filtered := result.Items
			total := result.Total

			formatMode := strings.ToLower(strings.TrimSpace(format))
			if formatMode == "" {
				formatMode = "table"
			}

			if formatMode == "json" && (noHeader || strings.TrimSpace(columnInput) != "") {
				return errors.New("--no-header and --column are only supported with table/csv/tsv")
			}

			columns, err := ParseColumns(columnInput, detailed, formatMode)
			if err != nil {
				return err
			}

			switch formatMode {
			case "table":
				out := cmd.OutOrStdout()

				table, err := RenderTable(filtered, total, RenderOptions{
					NoHeader:  noHeader,
					ColorMode: colorMode,
					Out:       out,
					Columns:   columns,
					HeadWidth: headWidth,
				})
				if err != nil {
					return err
				}

				return WriteWithPager(out, table, pager, pageSize, !noHeader)
			case "json":
				b, err := json.Marshal(filtered)
				if err != nil {
					return err
				}

				if _, err := cmd.OutOrStdout().Write(append(b, '\n')); err != nil {
					return err
				}

				return nil
			case "csv":
				return WriteDelimited(cmd.OutOrStdout(), filtered, ',', noHeader, columns)
			case "tsv":
				return WriteDelimited(cmd.OutOrStdout(), filtered, '\t', noHeader, columns)
			default:
				return fmt.Errorf("unsupported format %q", format)
			}
		},
	}

	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().StringVarP(&id, "id", "i", "", "exact session id")
	cmd.Flags().StringVarP(&idPrefix, "id-prefix", "p", "", "session id prefix")
	cmd.Flags().StringVar(&hostContains, "host-contains", "", "case-insensitive substring match against host path")
	cmd.Flags().StringVar(&pathContains, "path-contains", "", "case-insensitive substring match against session file path")
	cmd.Flags().StringVar(&headContains, "head-contains", "", "case-insensitive substring match against preview head text")
	cmd.Flags().StringVarP(&olderThan, "older-than", "o", "", "select sessions older than duration (e.g. 30d, 12h)")
	cmd.Flags().StringVarP(&health, "health", "H", "", "health filter: ok|corrupted|missing-meta")
	cmd.Flags().StringVarP(&format, "format", "f", "table", "output format: table|json|csv|tsv")
	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "max rows to print (0 means unlimited)")
	cmd.Flags().BoolVar(&detailed, "detailed", false, "show detailed columns")
	cmd.Flags().BoolVar(&pager, "pager", false, "enable interactive pager")
	cmd.Flags().IntVar(&pageSize, "page-size", 10, "rows per page when --pager is enabled")
	cmd.Flags().StringVar(&colorMode, "color", "always", "color mode: auto|always|never")
	cmd.Flags().BoolVar(&noHeader, "no-header", false, "hide header row for table/csv/tsv")
	cmd.Flags().StringVar(&columnInput, "column", "", "comma-separated columns (e.g. session_id,updated_at,size)")
	cmd.Flags().IntVar(&headWidth, "head-width", 36, "max HEAD width in table format (0 means no truncation)")
	cmd.Flags().StringVarP(&sortBy, "sort", "s", "updated_at", "sort field: updated_at|created_at|size|health|id|session_id")
	cmd.Flags().StringVar(&order, "order", "desc", "sort order: asc|desc")
	cmd.Flags().IntVar(&offset, "offset", 0, "skip first N rows before printing")

	return cmd
}

func RenderTable(sessions []session.Session, total int, opts RenderOptions) (string, error) {
	useColor := cliutil.ShouldUseColor(opts.ColorMode, opts.Out)
	home, _ := os.UserHomeDir()

	var buf bytes.Buffer

	w := tabwriter.NewWriter(&buf, 2, 4, 2, ' ', 0)

	if !opts.NoHeader {
		headers := make([]string, 0, len(opts.Columns))
		for _, c := range opts.Columns {
			headers = append(headers, c.Header)
		}

		_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))
	}

	for _, s := range sessions {
		values := make([]string, 0, len(opts.Columns))
		for _, c := range opts.Columns {
			values = append(values, ColumnValue(c.Key, s, home, opts.HeadWidth, true))
		}

		_, _ = fmt.Fprintln(w, strings.Join(values, "\t"))
	}

	if err := w.Flush(); err != nil {
		return "", err
	}

	footer := fmt.Sprintf("showing %d of %d", len(sessions), total)
	if len(sessions) < total {
		footer += " (use --limit 0 for all)"
	}

	_, _ = fmt.Fprintf(&buf, "%s\n", footer)

	rendered := buf.String()
	if useColor {
		rendered = ColorizeRenderedTable(rendered, sessions, opts.NoHeader, HasHealthColumn(opts.Columns))
	}

	return rendered, nil
}
