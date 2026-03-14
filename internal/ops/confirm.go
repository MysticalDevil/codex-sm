package ops

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// IsInteractiveReader checks whether reader is a terminal-like file descriptor.
func IsInteractiveReader(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}

	fi, err := f.Stat()
	if err != nil {
		return false
	}

	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ConfirmDelete asks the user to confirm delete action.
func ConfirmDelete(in io.Reader, out io.Writer, count int, hard bool) (bool, error) {
	if !IsInteractiveReader(in) {
		return false, fmt.Errorf("interactive confirm requires a terminal stdin; use --yes to continue non-interactively")
	}

	reader := bufio.NewReader(in)

	if hard {
		if _, err := fmt.Fprintf(out, "Hard delete %d session(s). Type DELETE to continue: ", count); err != nil {
			return false, err
		}

		text, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}

		return strings.TrimSpace(text) == "DELETE", nil
	}

	if _, err := fmt.Fprintf(out, "Delete %d session(s) to trash? [y/N]: ", count); err != nil {
		return false, err
	}

	text, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	v := strings.ToLower(strings.TrimSpace(text))

	return v == "y" || v == "yes", nil
}

// ConfirmRestore asks the user to confirm restore action.
func ConfirmRestore(in io.Reader, out io.Writer, count int) (bool, error) {
	if !IsInteractiveReader(in) {
		return false, fmt.Errorf("interactive confirm requires a terminal stdin; use --yes to continue non-interactively")
	}

	if _, err := fmt.Fprintf(out, "Restore %d session(s) from trash? [y/N]: ", count); err != nil {
		return false, err
	}

	reader := bufio.NewReader(in)

	text, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	v := strings.ToLower(strings.TrimSpace(text))

	return v == "y" || v == "yes", nil
}
