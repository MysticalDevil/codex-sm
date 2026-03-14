package doctor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/usecase"
)

func runChecks(
	resolveSessionsRoot func() (string, error),
	resolveTrashRoot func() (string, error),
	resolveLogFile func() (string, error),
) []check {
	checks := make([]check, 0, 10)

	checks = append(checks, checkConfigFile())

	sessionsRoot, sessionsErr := resolveSessionsRoot()
	checks = append(checks, checkDir("sessions_root", sessionsRoot, sessionsErr))
	checks = append(checks, usecase.CheckSessionHostPaths(usecase.DoctorHostPathInput{
		SessionsRoot: sessionsRoot,
		SessionsErr:  sessionsErr,
		CompactPath:  CompactPath,
	}))

	trashRoot, trashErr := resolveTrashRoot()
	checks = append(checks, checkDir("trash_root", trashRoot, trashErr))

	logFile, logErr := resolveLogFile()
	checks = append(checks, checkLogFile(logFile, logErr))

	return checks
}

func checkConfigFile() check {
	p, err := config.AppConfigPath()
	if err != nil {
		return check{Name: "config", Level: Fail, Detail: err.Error()}
	}

	_, err = os.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return check{Name: "config", Level: Warn, Detail: fmt.Sprintf("missing (optional): %s", p)}
		}

		return check{Name: "config", Level: Fail, Detail: err.Error()}
	}

	cfg, err := config.LoadAppConfig()
	if err != nil {
		return check{Name: "config", Level: Fail, Detail: err.Error()}
	}

	if strings.TrimSpace(cfg.SessionsRoot) == "" && strings.TrimSpace(cfg.TrashRoot) == "" && strings.TrimSpace(cfg.LogFile) == "" {
		return check{Name: "config", Level: Pass, Detail: "loaded (no overrides)"}
	}

	return check{Name: "config", Level: Pass, Detail: "loaded"}
}

func checkDir(name, path string, pathErr error) check {
	if pathErr != nil {
		return check{Name: name, Level: Fail, Detail: pathErr.Error()}
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return check{Name: name, Level: Warn, Detail: fmt.Sprintf("missing: %s", path)}
		}

		return check{Name: name, Level: Fail, Detail: err.Error()}
	}

	if !info.IsDir() {
		return check{Name: name, Level: Fail, Detail: fmt.Sprintf("not a directory: %s", path)}
	}

	if writable, msg := isWritableDir(path); !writable {
		return check{Name: name, Level: Warn, Detail: msg}
	}

	return check{Name: name, Level: Pass, Detail: path}
}

func checkLogFile(path string, pathErr error) check {
	if pathErr != nil {
		return check{Name: "log_file", Level: Fail, Detail: pathErr.Error()}
	}

	dir := filepath.Dir(path)

	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return check{Name: "log_file", Level: Warn, Detail: fmt.Sprintf("parent dir missing: %s", dir)}
		}

		return check{Name: "log_file", Level: Fail, Detail: err.Error()}
	}

	if !info.IsDir() {
		return check{Name: "log_file", Level: Fail, Detail: fmt.Sprintf("parent is not directory: %s", dir)}
	}

	if writable, msg := isWritableDir(dir); !writable {
		return check{Name: "log_file", Level: Warn, Detail: msg}
	}

	return check{Name: "log_file", Level: Pass, Detail: path}
}

func isWritableDir(path string) (bool, string) {
	f, err := os.CreateTemp(path, ".codexsm-doctor-*")
	if err != nil {
		return false, fmt.Sprintf("not writable: %s (%v)", path, err)
	}

	name := f.Name()
	if err := f.Close(); err != nil {
		return false, fmt.Sprintf("close temp file failed: %v", err)
	}

	if err := os.Remove(name); err != nil {
		return false, fmt.Sprintf("cleanup temp file failed: %v", err)
	}

	return true, path
}
