package tui

import (
	"bufio"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func loadPreviewIndexEntry(path, key string) ([]string, bool, error) {
	entries, corrupted, err := readPreviewIndex(path)
	if err != nil {
		return nil, false, err
	}
	if corrupted {
		// Best-effort corruption recovery: compact to valid entries only.
		_ = rewritePreviewIndex(path, entries, maxInt(1, len(entries)))
	}
	rec, ok := entries[key]
	if !ok {
		return nil, false, nil
	}
	return append([]string(nil), rec.Lines...), true, nil
}

func upsertPreviewIndex(path string, cap int, record previewIndexRecord) error {
	if cap <= 0 {
		cap = 5000
	}
	lockPath := path + ".lock"
	return withPreviewIndexLock(lockPath, 2*time.Second, func() error {
		entries, _, err := readPreviewIndex(path)
		if err != nil {
			return err
		}
		if record.Key != "" {
			entries[record.Key] = record
		}
		return rewritePreviewIndex(path, entries, cap)
	})
}

func readPreviewIndex(path string) (map[string]previewIndexRecord, bool, error) {
	out := make(map[string]previewIndexRecord)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return out, false, nil
		}
		return nil, false, err
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	corrupted := false
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec previewIndexRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			corrupted = true
			continue
		}
		if rec.Key == "" {
			corrupted = true
			continue
		}
		old, ok := out[rec.Key]
		if !ok || rec.TouchedAtUnix >= old.TouchedAtUnix {
			out[rec.Key] = rec
		}
	}
	if err := sc.Err(); err != nil {
		return nil, false, err
	}
	return out, corrupted, nil
}

func rewritePreviewIndex(path string, entries map[string]previewIndexRecord, cap int) error {
	if cap <= 0 {
		cap = 5000
	}
	list := make([]previewIndexRecord, 0, len(entries))
	for _, rec := range entries {
		list = append(list, rec)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].TouchedAtUnix == list[j].TouchedAtUnix {
			return list[i].Key < list[j].Key
		}
		return list[i].TouchedAtUnix > list[j].TouchedAtUnix
	})
	if len(list) > cap {
		list = list[:cap]
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".preview-index-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	encErr := error(nil)
	for _, rec := range list {
		b, err := json.Marshal(rec)
		if err != nil {
			encErr = err
			break
		}
		if _, err := tmp.Write(b); err != nil {
			encErr = err
			break
		}
		if _, err := tmp.Write([]byte{'\n'}); err != nil {
			encErr = err
			break
		}
	}
	closeErr := tmp.Close()
	if encErr != nil {
		_ = os.Remove(tmpPath)
		return encErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func withPreviewIndexLock(lockPath string, timeout time.Duration, fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for {
		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_ = lockFile.Close()
			defer func() { _ = os.Remove(lockPath) }()
			return fn()
		}
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting preview index lock: %s", lockPath)
		}
		time.Sleep(15 * time.Millisecond)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
