package doctor

import (
	"os"
	"strings"
)

// CompactPath compacts path strings for doctor output.
func CompactPath(path string, maxLen int) string {
	p := strings.TrimSpace(path)
	if p == "" || maxLen <= 0 || len(p) <= maxLen {
		return p
	}

	if home, err := os.UserHomeDir(); err == nil {
		if p == home {
			p = "~"
		} else if rest, ok := strings.CutPrefix(p, home+string(os.PathSeparator)); ok {
			p = "~" + string(os.PathSeparator) + rest
		}
	}

	if len(p) <= maxLen {
		return p
	}

	segs := strings.Split(strings.Trim(p, string(os.PathSeparator)), string(os.PathSeparator))
	if len(segs) < 3 {
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
		start = max(start, 0)

		return "..." + last[start:]
	}

	return short[:maxLen]
}
