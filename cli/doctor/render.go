package doctor

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	ansiReset    = "\x1b[0m"
	ansiGreen    = "\x1b[32m"
	ansiYellow   = "\x1b[33m"
	ansiRed      = "\x1b[31m"
	ansiCyanBold = "\x1b[1;36m"
)

func colorize(v, color string, enabled bool) string {
	if !enabled || color == "" {
		return v
	}

	return color + v + ansiReset
}

func renderChecks(checks []check, color bool) string {
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

	headCheck := fmt.Sprintf("%-*s", checkW, "CHECK")
	headStatus := fmt.Sprintf("%-*s", statusW, "STATUS")
	headDetail := "DETAIL"

	if color {
		headCheck = colorize(headCheck, ansiCyanBold, true)
		headStatus = colorize(headStatus, ansiCyanBold, true)
		headDetail = colorize(headDetail, ansiCyanBold, true)
	}

	_, _ = fmt.Fprintf(&buf, "%s  %s  %s\n", headCheck, headStatus, headDetail)

	for _, c := range checks {
		status := fmt.Sprintf("%-*s", statusW, string(c.Level))
		if color {
			switch c.Level {
			case Pass:
				status = colorize(status, ansiGreen, true)
			case Warn:
				status = colorize(status, ansiYellow, true)
			case Fail:
				status = colorize(status, ansiRed, true)
			}
		}

		lines := detailLines(c.Detail)
		if len(lines) == 0 {
			lines = []string{""}
		}

		_, _ = fmt.Fprintf(&buf, "%-*s  %s  %s\n", checkW, c.Name, status, lines[0])
		for _, line := range lines[1:] {
			_, _ = fmt.Fprintf(&buf, "%s  %s  %s\n", strings.Repeat(" ", checkW), strings.Repeat(" ", statusW), line)
		}
	}

	return buf.String()
}

func detailLines(detail string) []string {
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
