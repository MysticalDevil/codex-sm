package doctor

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"strings"

	cliutil "github.com/MysticalDevil/codexsm/cli/util"
	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/usecase"
	"github.com/spf13/cobra"
)

func newRiskCmd(resolveSessionsRoot func() (string, error)) *cobra.Command {
	var (
		sessionsRoot string
		sampleLimit  int
		format       string
		integrity    bool
	)

	cmd := &cobra.Command{
		Use:   "risk",
		Short: "Scan sessions and report risk candidates",
		Long: "Scan sessions and report RISK candidates.\n\n" +
			"Current risk policy:\n" +
			"  - high: health=corrupted\n" +
			"  - medium: health=missing-meta\n" +
			"  - extension point reserved for integrity checks",
		Example: "  codexsm doctor risk\n" +
			"  codexsm doctor risk --sessions-root ~/.codex/sessions\n" +
			"  codexsm doctor risk --sample-limit 20\n" +
			"  codexsm doctor risk --format json --integrity-check",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := strings.TrimSpace(sessionsRoot)
			if root == "" {
				v, err := resolveSessionsRoot()
				if err != nil {
					return cliutil.WithExitCode(err, 2)
				}

				root = v
			} else {
				v, err := config.ResolvePath(root)
				if err != nil {
					return cliutil.WithExitCode(err, 2)
				}

				root = v
			}

			usedFormat := strings.ToLower(strings.TrimSpace(format))
			if usedFormat == "" {
				usedFormat = "text"
			}

			if usedFormat != "text" && usedFormat != "json" {
				return cliutil.WithExitCode(fmt.Errorf("invalid --format %q (allowed: text, json)", format), 2)
			}

			report, err := usecase.DoctorRisk(usecase.DoctorRiskInput{
				SessionsRoot:   root,
				SampleLimit:    sampleLimit,
				IntegrityCheck: integrity,
			})
			if err != nil {
				return cliutil.WithExitCode(err, 2)
			}

			if usedFormat == "json" {
				b, err := json.Marshal(report)
				if err != nil {
					return cliutil.WithExitCode(err, 2)
				}

				if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(b)); err != nil {
					return err
				}

				if report.RiskTotal > 0 {
					return cliutil.WithExitCode(fmt.Errorf("risk sessions detected: %d", report.RiskTotal), 1)
				}

				return nil
			}

			out := renderRiskText(report)
			if _, err := fmt.Fprint(cmd.OutOrStdout(), out); err != nil {
				return err
			}

			if report.RiskTotal > 0 {
				return cliutil.WithExitCode(fmt.Errorf("risk sessions detected: %d", report.RiskTotal), 1)
			}

			return nil
		},
	}
	cmd.SilenceUsage = true
	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().IntVar(&sampleLimit, "sample-limit", 10, "max risky sessions to print")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text|json")
	cmd.Flags().BoolVar(&integrity, "integrity-check", true, "enable SHA256 sidecar integrity check")

	return cmd
}

func renderRiskText(report usecase.DoctorRiskReport) string {
	var buf bytes.Buffer

	_, _ = fmt.Fprintf(&buf, "RISK SUMMARY\n")
	_, _ = fmt.Fprintf(
		&buf,
		"sessions_total=%d risk_total=%d risk_rate=%.1f%% high=%d medium=%d integrity_check=%v\n",
		report.SessionsTotal, report.RiskTotal, report.RiskRate, report.High, report.Medium, report.IntegrityCheck,
	)

	if report.RiskTotal == 0 {
		_, _ = fmt.Fprintln(&buf, "no risky sessions found")
		return buf.String()
	}

	_, _ = fmt.Fprintf(&buf, "samples(limit=%d)\n", report.SampleLimit)
	_, _ = fmt.Fprintln(&buf, "LEVEL   HEALTH        SESSION_ID    PATH")

	for _, item := range report.Samples {
		sid := item.SessionID
		if len(sid) > 12 {
			sid = sid[:12]
		}

		_, _ = fmt.Fprintf(
			&buf,
			"%-6s  %-12s  %-12s  %s\n",
			strings.ToUpper(string(item.Level)),
			string(item.Health),
			sid,
			CompactPath(item.Path, 72),
		)
	}

	return buf.String()
}
