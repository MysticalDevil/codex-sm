package cli

import (
	"bytes"
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
	return cmd
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
