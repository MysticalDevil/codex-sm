package cli

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/spf13/cobra"
)

type doctorLevel string

const (
	doctorPass doctorLevel = "PASS"
	doctorWarn doctorLevel = "WARN"
	doctorFail doctorLevel = "FAIL"
)

type doctorCheck struct {
	Name   string
	Level  doctorLevel
	Detail string
}

func newDoctorCmd() *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run local environment and configuration checks",
		Long: "Run local checks for codexsm runtime prerequisites.\n\n" +
			"This command validates config and storage paths.",
		Example: "  codexsm doctor\n" +
			"  codexsm doctor --strict",
		RunE: func(cmd *cobra.Command, args []string) error {
			checks := runDoctorChecks()
			out := renderDoctorChecks(checks, shouldUseColor("auto", cmd.OutOrStdout()))
			if _, err := fmt.Fprint(cmd.OutOrStdout(), out); err != nil {
				return err
			}
			if strict {
				for _, c := range checks {
					if c.Level == doctorFail || c.Level == doctorWarn {
						return WithExitCode(fmt.Errorf("doctor check failed: %s (%s)", c.Name, c.Level), 1)
					}
				}
			}
			for _, c := range checks {
				if c.Level == doctorFail {
					return WithExitCode(fmt.Errorf("doctor check failed: %s", c.Name), 1)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false, "treat warnings as failures")
	cmd.AddCommand(newDoctorRiskCmd())
	return cmd
}

func newDoctorRiskCmd() *cobra.Command {
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
				v, err := runtimeSessionsRoot()
				if err != nil {
					return WithExitCode(err, 2)
				}
				root = v
			} else {
				v, err := config.ResolvePath(root)
				if err != nil {
					return WithExitCode(err, 2)
				}
				root = v
			}
			items, err := session.ScanSessions(root)
			if err != nil {
				return WithExitCode(err, 2)
			}

			if sampleLimit <= 0 {
				sampleLimit = 10
			}
			type riskyItem struct {
				Session session.Session
				Risk    session.Risk
			}
			var checker session.IntegrityChecker
			if integrity {
				checker = session.SHA256SidecarChecker
			}
			risky := make([]riskyItem, 0, len(items))
			highCount := 0
			mediumCount := 0
			for _, s := range items {
				r := session.EvaluateRisk(s, checker)
				if r.Level == session.RiskNone {
					continue
				}
				risky = append(risky, riskyItem{Session: s, Risk: r})
				switch r.Level {
				case session.RiskHigh:
					highCount++
				case session.RiskMedium:
					mediumCount++
				}
			}
			sort.SliceStable(risky, func(i, j int) bool {
				ri := risky[i].Risk.Level
				rj := risky[j].Risk.Level
				if ri != rj {
					return riskRank(ri) > riskRank(rj)
				}
				c := risky[j].Session.UpdatedAt.Compare(risky[i].Session.UpdatedAt)
				if c != 0 {
					return c < 0
				}
				return risky[i].Session.SessionID < risky[j].Session.SessionID
			})

			rate := 0.0
			if len(items) > 0 {
				rate = float64(len(risky)) / float64(len(items)) * 100
			}
			usedFormat := strings.ToLower(strings.TrimSpace(format))
			if usedFormat == "" {
				usedFormat = "text"
			}
			if usedFormat != "text" && usedFormat != "json" {
				return WithExitCode(fmt.Errorf("invalid --format %q (allowed: text, json)", format), 2)
			}
			if usedFormat == "json" {
				type riskSample struct {
					Level     session.RiskLevel  `json:"level"`
					Reason    session.RiskReason `json:"reason"`
					Health    session.Health     `json:"health"`
					SessionID string             `json:"session_id"`
					Path      string             `json:"path"`
					Detail    string             `json:"detail,omitempty"`
				}
				type riskReport struct {
					SessionsTotal  int          `json:"sessions_total"`
					RiskTotal      int          `json:"risk_total"`
					RiskRate       float64      `json:"risk_rate"`
					High           int          `json:"high"`
					Medium         int          `json:"medium"`
					IntegrityCheck bool         `json:"integrity_check"`
					SampleLimit    int          `json:"sample_limit"`
					Samples        []riskSample `json:"samples"`
				}
				rep := riskReport{
					SessionsTotal:  len(items),
					RiskTotal:      len(risky),
					RiskRate:       rate,
					High:           highCount,
					Medium:         mediumCount,
					IntegrityCheck: integrity,
					SampleLimit:    sampleLimit,
					Samples:        make([]riskSample, 0, minInt(sampleLimit, len(risky))),
				}
				for i, item := range risky {
					if i >= sampleLimit {
						break
					}
					rep.Samples = append(rep.Samples, riskSample{
						Level:     item.Risk.Level,
						Reason:    item.Risk.Reason,
						Health:    item.Session.Health,
						SessionID: item.Session.SessionID,
						Path:      item.Session.Path,
						Detail:    item.Risk.Detail,
					})
				}
				b, err := json.Marshal(rep)
				if err != nil {
					return WithExitCode(err, 2)
				}
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(b)); err != nil {
					return err
				}
				if len(risky) > 0 {
					return WithExitCode(fmt.Errorf("risk sessions detected: %d", len(risky)), 1)
				}
				return nil
			}

			var buf bytes.Buffer
			_, _ = fmt.Fprintf(&buf, "RISK SUMMARY\n")
			_, _ = fmt.Fprintf(
				&buf,
				"sessions_total=%d risk_total=%d risk_rate=%.1f%% high=%d medium=%d integrity_check=%v\n",
				len(items), len(risky), rate, highCount, mediumCount, integrity,
			)
			if len(risky) == 0 {
				_, _ = fmt.Fprintln(&buf, "no risky sessions found")
				if _, err := fmt.Fprint(cmd.OutOrStdout(), buf.String()); err != nil {
					return err
				}
				return nil
			}

			_, _ = fmt.Fprintf(&buf, "samples(limit=%d)\n", sampleLimit)
			_, _ = fmt.Fprintln(&buf, "LEVEL   HEALTH        SESSION_ID    PATH")
			for i, item := range risky {
				if i >= sampleLimit {
					break
				}
				sid := item.Session.SessionID
				if len(sid) > 12 {
					sid = sid[:12]
				}
				_, _ = fmt.Fprintf(
					&buf,
					"%-6s  %-12s  %-12s  %s\n",
					strings.ToUpper(string(item.Risk.Level)),
					string(item.Session.Health),
					sid,
					compactDoctorPath(item.Session.Path, 72),
				)
			}
			if _, err := fmt.Fprint(cmd.OutOrStdout(), buf.String()); err != nil {
				return err
			}
			return WithExitCode(fmt.Errorf("risk sessions detected: %d", len(risky)), 1)
		},
	}
	cmd.SilenceUsage = true
	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().IntVar(&sampleLimit, "sample-limit", 10, "max risky sessions to print")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text|json")
	cmd.Flags().BoolVar(&integrity, "integrity-check", true, "enable SHA256 sidecar integrity check")
	return cmd
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func riskRank(level session.RiskLevel) int {
	switch level {
	case session.RiskHigh:
		return 2
	case session.RiskMedium:
		return 1
	default:
		return 0
	}
}

func runDoctorChecks() []doctorCheck {
	checks := make([]doctorCheck, 0, 10)

	checks = append(checks, checkConfigFile())

	sessionsRoot, sessionsErr := runtimeSessionsRoot()
	checks = append(checks, checkDir("sessions_root", sessionsRoot, sessionsErr))
	checks = append(checks, checkSessionHostPaths(sessionsRoot, sessionsErr))

	trashRoot, trashErr := runtimeTrashRoot()
	checks = append(checks, checkDir("trash_root", trashRoot, trashErr))

	logFile, logErr := runtimeLogFile()
	checks = append(checks, checkLogFile(logFile, logErr))
	return checks
}

func checkSessionHostPaths(sessionsRoot string, sessionsErr error) doctorCheck {
	if sessionsErr != nil {
		return doctorCheck{Name: "session_host_paths", Level: doctorWarn, Detail: "skipped: sessions_root unresolved"}
	}
	if _, err := os.Stat(sessionsRoot); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorCheck{Name: "session_host_paths", Level: doctorWarn, Detail: "skipped: sessions_root missing"}
		}
		return doctorCheck{Name: "session_host_paths", Level: doctorFail, Detail: err.Error()}
	}

	items, err := session.ScanSessions(sessionsRoot)
	if err != nil {
		return doctorCheck{Name: "session_host_paths", Level: doctorFail, Detail: err.Error()}
	}
	if len(items) == 0 {
		return doctorCheck{Name: "session_host_paths", Level: doctorPass, Detail: "no sessions found"}
	}

	withHost := 0
	missingCountByHost := make(map[string]int)
	for _, s := range items {
		host := strings.TrimSpace(s.HostDir)
		if host == "" {
			continue
		}
		withHost++
		if _, statErr := os.Stat(host); statErr == nil {
			continue
		} else if errors.Is(statErr, os.ErrNotExist) {
			missingCountByHost[host]++
		} else {
			return doctorCheck{Name: "session_host_paths", Level: doctorFail, Detail: fmt.Sprintf("stat host %s: %v", host, statErr)}
		}
	}
	if len(missingCountByHost) == 0 {
		return doctorCheck{
			Name:   "session_host_paths",
			Level:  doctorPass,
			Detail: fmt.Sprintf("all host paths exist (sessions=%d with_host=%d)", len(items), withHost),
		}
	}

	hosts := make([]string, 0, len(missingCountByHost))
	for host := range missingCountByHost {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	displayHosts := hosts
	if len(displayHosts) > 3 {
		displayHosts = displayHosts[:3]
	}
	hostLines := make([]string, 0, len(displayHosts))
	for _, host := range displayHosts {
		hostLines = append(hostLines, fmt.Sprintf("- %s (%d)", compactDoctorPath(host, 56), missingCountByHost[host]))
	}

	suggestHost := displayHosts[0]
	return doctorCheck{
		Name:  "session_host_paths",
		Level: doctorWarn,
		Detail: fmt.Sprintf(
			"missing_hosts=%d impacted_sessions=%d\nsample_hosts:\n%s\nrecommended_actions:\n1. review: codexsm list --host-contains %s\n2. migrate (soft-delete): codexsm delete --host-contains %s\n3. optional hard-delete: codexsm delete --host-contains %s --dry-run=false --confirm --hard",
			len(missingCountByHost),
			withHost,
			strings.Join(hostLines, "\n"),
			suggestHost,
			suggestHost,
			suggestHost,
		),
	}
}

func checkConfigFile() doctorCheck {
	p, err := config.AppConfigPath()
	if err != nil {
		return doctorCheck{Name: "config", Level: doctorFail, Detail: err.Error()}
	}
	_, err = os.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorCheck{Name: "config", Level: doctorWarn, Detail: fmt.Sprintf("missing (optional): %s", p)}
		}
		return doctorCheck{Name: "config", Level: doctorFail, Detail: err.Error()}
	}
	cfg, err := config.LoadAppConfig()
	if err != nil {
		return doctorCheck{Name: "config", Level: doctorFail, Detail: err.Error()}
	}
	if strings.TrimSpace(cfg.SessionsRoot) == "" && strings.TrimSpace(cfg.TrashRoot) == "" && strings.TrimSpace(cfg.LogFile) == "" {
		return doctorCheck{Name: "config", Level: doctorPass, Detail: "loaded (no overrides)"}
	}
	return doctorCheck{Name: "config", Level: doctorPass, Detail: "loaded"}
}

func checkDir(name, path string, pathErr error) doctorCheck {
	if pathErr != nil {
		return doctorCheck{Name: name, Level: doctorFail, Detail: pathErr.Error()}
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorCheck{Name: name, Level: doctorWarn, Detail: fmt.Sprintf("missing: %s", path)}
		}
		return doctorCheck{Name: name, Level: doctorFail, Detail: err.Error()}
	}
	if !info.IsDir() {
		return doctorCheck{Name: name, Level: doctorFail, Detail: fmt.Sprintf("not a directory: %s", path)}
	}
	if writable, msg := isWritableDir(path); !writable {
		return doctorCheck{Name: name, Level: doctorWarn, Detail: msg}
	}
	return doctorCheck{Name: name, Level: doctorPass, Detail: path}
}

func checkLogFile(path string, pathErr error) doctorCheck {
	if pathErr != nil {
		return doctorCheck{Name: "log_file", Level: doctorFail, Detail: pathErr.Error()}
	}
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorCheck{Name: "log_file", Level: doctorWarn, Detail: fmt.Sprintf("parent dir missing: %s", dir)}
		}
		return doctorCheck{Name: "log_file", Level: doctorFail, Detail: err.Error()}
	}
	if !info.IsDir() {
		return doctorCheck{Name: "log_file", Level: doctorFail, Detail: fmt.Sprintf("parent is not directory: %s", dir)}
	}
	if writable, msg := isWritableDir(dir); !writable {
		return doctorCheck{Name: "log_file", Level: doctorWarn, Detail: msg}
	}
	return doctorCheck{Name: "log_file", Level: doctorPass, Detail: path}
}

func isWritableDir(path string) (bool, string) {
	f, err := os.CreateTemp(path, ".codexsm-doctor-*")
	if err != nil {
		return false, fmt.Sprintf("not writable: %s (%v)", path, err)
	}
	name := f.Name()
	if closeErr := f.Close(); closeErr != nil {
		return false, fmt.Sprintf("close temp file failed: %v", closeErr)
	}
	if rmErr := os.Remove(name); rmErr != nil {
		return false, fmt.Sprintf("cleanup temp file failed: %v", rmErr)
	}
	return true, path
}

func renderDoctorChecks(checks []doctorCheck, color bool) string {
	var buf bytes.Buffer

	checkW := len("CHECK")
	statusW := len("STATUS")
	for _, c := range checks {
		if len(c.Name) > checkW {
			checkW = len(c.Name)
		}
		if len(c.Level) > statusW {
			statusW = len(c.Level)
		}
	}

	headCheck := padRight("CHECK", checkW)
	headStatus := padRight("STATUS", statusW)
	headDetail := "DETAIL"
	if color {
		headCheck = colorize(headCheck, ansiCyanBold, true)
		headStatus = colorize(headStatus, ansiCyanBold, true)
		headDetail = colorize(headDetail, ansiCyanBold, true)
	}
	_, _ = fmt.Fprintf(&buf, "%s  %s  %s\n", headCheck, headStatus, headDetail)

	for _, c := range checks {
		status := padRight(string(c.Level), statusW)
		if color {
			switch c.Level {
			case doctorPass:
				status = colorize(status, ansiGreen, true)
			case doctorWarn:
				status = colorize(status, ansiYellow, true)
			case doctorFail:
				status = colorize(status, ansiRed, true)
			}
		}
		lines := doctorDetailLines(c.Detail)
		if len(lines) == 0 {
			lines = []string{""}
		}
		_, _ = fmt.Fprintf(&buf, "%s  %s  %s\n", padRight(c.Name, checkW), status, lines[0])
		for _, line := range lines[1:] {
			_, _ = fmt.Fprintf(&buf, "%s  %s  %s\n", strings.Repeat(" ", checkW), strings.Repeat(" ", statusW), line)
		}
	}
	return buf.String()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func doctorDetailLines(detail string) []string {
	d := strings.TrimSpace(detail)
	if d == "" {
		return nil
	}
	raw := strings.Split(d, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func compactDoctorPath(path string, maxLen int) string {
	p := strings.TrimSpace(path)
	if p == "" || maxLen <= 0 || len(p) <= maxLen {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		if p == home {
			p = "~"
		} else if strings.HasPrefix(p, home+string(os.PathSeparator)) {
			p = "~" + string(os.PathSeparator) + strings.TrimPrefix(p, home+string(os.PathSeparator))
		}
	}
	if len(p) <= maxLen {
		return p
	}
	segs := strings.Split(strings.Trim(p, string(os.PathSeparator)), string(os.PathSeparator))
	if len(segs) < 3 {
		if len(p) <= maxLen {
			return p
		}
		head := maxLen / 2
		tail := maxLen - head - 3
		if tail < 1 {
			return p[:maxLen]
		}
		return p[:head] + "..." + p[len(p)-tail:]
	}
	last := segs[len(segs)-1]
	prev := segs[len(segs)-2]
	prefix := "/"
	if strings.HasPrefix(p, "~/") {
		prefix = "~/"
	}
	short := prefix + ".../" + prev + "/" + last
	if len(short) <= maxLen {
		return short
	}
	if len(last)+4 > maxLen {
		start := len(last) - (maxLen - 3)
		if start < 0 {
			start = 0
		}
		return "..." + last[start:]
	}
	return short[:maxLen]
}
