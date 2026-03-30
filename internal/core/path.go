package core

import (
	"os"
	"strings"
)

// CompactHomePath replaces home prefix with "~" when possible.
func CompactHomePath(path, home string) string {
	if home == "" {
		return path
	}

	if path == home {
		return "~"
	}

	prefix := home + string(os.PathSeparator)
	if rest, ok := strings.CutPrefix(path, prefix); ok {
		return "~" + string(os.PathSeparator) + rest
	}

	return path
}
