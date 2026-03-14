package core

import (
	"fmt"
	"time"
)

// ShortID returns a stable short view for long session IDs.
func ShortID(id string) string {
	const maxLen = 12
	if len(id) <= maxLen {
		return id
	}
	return id[:maxLen]
}

// FormatDisplayTime formats timestamps for human-readable table output.
func FormatDisplayTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

// FormatBytesIEC formats byte counts using IEC units.
func FormatBytesIEC(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(size)
	unit := -1
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	return fmt.Sprintf("%.1f%s", value, units[unit])
}
