package preview

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

// MaxIndexBytes is the byte budget for preview index storage.
const MaxIndexBytes = 8 << 20

// LoadIndexEntry loads one cached preview by key.
func LoadIndexEntry(path, key string) ([]string, bool, error) {
	entries, corrupted, err := ReadIndex(path)
	if err != nil {
		return nil, false, err
	}

	if corrupted {
		_ = RewriteIndex(path, entries, max(1, len(entries)), MaxIndexBytes)
	}

	rec, ok := entries[key]
	if !ok {
		return nil, false, nil
	}

	return append([]string(nil), rec.Lines...), true, nil
}

// UpsertIndex inserts or updates one preview index record.
func UpsertIndex(path string, cap int, record IndexRecord) error {
	if cap <= 0 {
		cap = 5000
	}

	lockPath := path + ".lock"

	return WithIndexLock(lockPath, 2*time.Second, func() error {
		entries, _, err := ReadIndex(path)
		if err != nil {
			return err
		}

		if record.Key != "" {
			entries[record.Key] = record
		}

		return RewriteIndex(path, entries, cap, MaxIndexBytes)
	})
}

// ReadIndex reads all records. Corrupted lines are skipped and flagged.
func ReadIndex(path string) (map[string]IndexRecord, bool, error) {
	out := make(map[string]IndexRecord)
	totalBytes := int64(0)

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

		var rec IndexRecord
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
			if ok {
				totalBytes -= RecordBytes(old)
			}

			out[rec.Key] = rec
			totalBytes += RecordBytes(rec)
			totalBytes = TrimIndexBytes(out, totalBytes, MaxIndexBytes)
		}
	}

	if err := sc.Err(); err != nil {
		return nil, false, err
	}

	return out, corrupted, nil
}

// RewriteIndex rewrites index with cap and byte-limit trimming.
func RewriteIndex(path string, entries map[string]IndexRecord, cap int, maxBytes int64) error {
	if cap <= 0 {
		cap = 5000
	}

	if maxBytes <= 0 {
		maxBytes = MaxIndexBytes
	}

	list := make([]IndexRecord, 0, len(entries))
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

	totalBytes := int64(0)

	trimmed := list[:0]
	for _, rec := range list {
		recBytes := RecordBytes(rec)
		if len(trimmed) > 0 && totalBytes+recBytes > maxBytes {
			break
		}

		trimmed = append(trimmed, rec)
		totalBytes += recBytes
	}

	list = trimmed

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

// WithIndexLock runs fn under an index lock file with timeout.
func WithIndexLock(lockPath string, timeout time.Duration, fn func() error) error {
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

// RecordBytes estimates one index record byte footprint.
func RecordBytes(rec IndexRecord) int64 {
	total := int64(len(rec.Key) + len(rec.Path) + 64)
	for _, line := range rec.Lines {
		total += int64(len(line))
	}

	return total
}

// TrimIndexBytes evicts oldest records until byte budget is satisfied.
func TrimIndexBytes(entries map[string]IndexRecord, totalBytes, maxBytes int64) int64 {
	if maxBytes <= 0 || totalBytes <= maxBytes {
		return totalBytes
	}

	for totalBytes > maxBytes && len(entries) > 1 {
		oldestKey := ""

		var oldest IndexRecord
		for k, rec := range entries {
			if oldestKey == "" || rec.TouchedAtUnix < oldest.TouchedAtUnix || (rec.TouchedAtUnix == oldest.TouchedAtUnix && rec.Key < oldest.Key) {
				oldestKey = k
				oldest = rec
			}
		}

		if oldestKey == "" {
			break
		}

		delete(entries, oldestKey)

		totalBytes -= RecordBytes(oldest)
	}

	if totalBytes < 0 {
		return 0
	}

	return totalBytes
}
