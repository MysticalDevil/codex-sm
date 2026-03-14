// Package util contains shared helpers used by CLI command subpackages.
package util

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/session"
	rootutil "github.com/MysticalDevil/codexsm/util"
)

var ansiColorRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// ExitError carries a user-facing error plus an intended process exit code.
type ExitError struct {
	Code int
	Err  error
}

// Error implements error.
func (e *ExitError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}

	return e.Err.Error()
}

// Unwrap returns the wrapped error.
func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

// ExitCode returns Code, defaulting to 1 for nil or invalid values.
func (e *ExitError) ExitCode() int {
	if e == nil || e.Code <= 0 {
		return 1
	}

	return e.Code
}

// WithExitCode wraps an error with a process exit code for main() handling.
func WithExitCode(err error, code int) error {
	if err == nil {
		return nil
	}

	return &ExitError{Code: code, Err: fmt.Errorf("%w", err)}
}

// ResolveOrDefault resolves an explicit path or uses fallback.
func ResolveOrDefault(v string, fallback func() (string, error)) (string, error) {
	if strings.TrimSpace(v) == "" {
		return fallback()
	}

	return config.ResolvePath(v)
}

// MarshalPrettyJSON marshals v with multiline indentation.
func MarshalPrettyJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.MarshalWrite(&buf, v, jsontext.Multiline(true), jsontext.WithIndent("  ")); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// WriteFileAtomic writes data by temp file + rename.
func WriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".codexsm-config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}

	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(data); err != nil {
		if closeErr := tmp.Close(); closeErr != nil {
			return fmt.Errorf("write temp file: %w (close temp file: %w)", err, closeErr)
		}

		cleanup()

		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmp.Chmod(mode); err != nil {
		if closeErr := tmp.Close(); closeErr != nil {
			return fmt.Errorf("chmod temp file: %w (close temp file: %w)", err, closeErr)
		}

		cleanup()

		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("replace config %s: %w", path, err)
	}

	return nil
}

// StripANSI removes ANSI SGR sequences.
func StripANSI(v string) string {
	return ansiColorRe.ReplaceAllString(v, "")
}

// IsTerminalWriter reports whether out is a TTY char device.
func IsTerminalWriter(out io.Writer) bool {
	f, ok := out.(*os.File)
	if !ok {
		return false
	}

	fi, err := f.Stat()
	if err != nil {
		return false
	}

	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ShouldUseColor decides output coloring.
func ShouldUseColor(mode string, out io.Writer) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "always":
		return true
	case "never":
		return false
	case "", "auto":
		if strings.EqualFold(os.Getenv("NO_COLOR"), "1") || strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
			return false
		}

		return IsTerminalWriter(out)
	default:
		return IsTerminalWriter(out)
	}
}

// BuildSelector builds a session selector from common flags.
func BuildSelector(id, idPrefix, hostContains, pathContains, headContains, olderThan, health string) (session.Selector, error) {
	sel := session.Selector{
		ID:           strings.TrimSpace(id),
		IDPrefix:     strings.TrimSpace(idPrefix),
		HostContains: strings.TrimSpace(hostContains),
		PathContains: strings.TrimSpace(pathContains),
		HeadContains: strings.TrimSpace(headContains),
	}

	if strings.TrimSpace(olderThan) != "" {
		d, err := rootutil.ParseOlderThan(olderThan)
		if err != nil {
			return sel, err
		}

		sel.OlderThan = d
		sel.HasOlderThan = true
	}

	if strings.TrimSpace(health) != "" {
		h, err := ParseHealth(health)
		if err != nil {
			return sel, err
		}

		sel.Health = h
		sel.HasHealth = true
	}

	return sel, nil
}

// ParseHealth parses health flag value.
func ParseHealth(v string) (session.Health, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case string(session.HealthOK):
		return session.HealthOK, nil
	case string(session.HealthMissingMeta):
		return session.HealthMissingMeta, nil
	case string(session.HealthCorrupted):
		return session.HealthCorrupted, nil
	default:
		return "", fmt.Errorf("invalid --health %q (allowed: ok, corrupted, missing-meta)", v)
	}
}
