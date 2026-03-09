package cli

import "fmt"

// ExitError carries a user-facing error plus an intended process exit code.
type ExitError struct {
	// Code is the process exit code that should be returned to the shell.
	Code int
	// Err is the wrapped error exposed to callers and users.
	Err error
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
