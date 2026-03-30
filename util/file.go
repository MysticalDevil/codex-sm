package util

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// MoveFile moves a file with cross-device fallback.
func MoveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	if err := CopyFile(src, dst); err != nil {
		return fmt.Errorf("copy %q to %q: %w", src, dst, err)
	}

	if err := os.Remove(src); err != nil {
		return fmt.Errorf("remove source file %q after copy: %w", src, err)
	}

	return nil
}

// CopyFile copies a file to destination and fsyncs output.
func CopyFile(src, dst string) (retErr error) {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file %q: %w", src, err)
	}

	defer func() {
		if closeErr := in.Close(); closeErr != nil {
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination file %q: %w", dst, err)
	}

	defer func() {
		if closeErr := out.Close(); closeErr != nil {
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		retErr = fmt.Errorf("copy file data from %q to %q: %w", src, dst, err)
		return
	}

	if err := out.Sync(); err != nil {
		retErr = fmt.Errorf("sync destination file %q: %w", dst, err)
		return
	}

	return
}
