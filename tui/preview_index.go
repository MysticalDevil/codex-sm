package tui

import (
	"strings"
	"time"

	previewpkg "github.com/MysticalDevil/codexsm/tui/preview"
)

const maxPreviewIndexBytes = previewpkg.MaxIndexBytes

func loadPreviewIndexEntry(path, key string) ([]string, bool, error) {
	return previewpkg.LoadIndexEntry(path, key)
}

func upsertPreviewIndex(path string, cap int, record previewIndexRecord) error {
	return previewpkg.UpsertIndex(path, cap, toPreviewIndexRecord(record))
}

func readPreviewIndex(path string) (map[string]previewIndexRecord, bool, error) {
	raw, corrupted, err := previewpkg.ReadIndex(path)
	if err != nil {
		return nil, false, err
	}
	out := make(map[string]previewIndexRecord, len(raw))
	for k, v := range raw {
		out[k] = fromPreviewIndexRecord(v)
	}
	return out, corrupted, nil
}

func rewritePreviewIndex(path string, entries map[string]previewIndexRecord, cap int, maxBytes int64) error {
	raw := make(map[string]previewpkg.IndexRecord, len(entries))
	for k, v := range entries {
		raw[k] = toPreviewIndexRecord(v)
	}
	return previewpkg.RewriteIndex(path, raw, cap, maxBytes)
}

func withPreviewIndexLock(lockPath string, timeout time.Duration, fn func() error) error {
	return previewpkg.WithIndexLock(lockPath, timeout, fn)
}

func previewIndexRecordBytes(rec previewIndexRecord) int64 {
	return previewpkg.RecordBytes(toPreviewIndexRecord(rec))
}

func trimPreviewIndexBytes(entries map[string]previewIndexRecord, totalBytes, maxBytes int64) int64 {
	raw := make(map[string]previewpkg.IndexRecord, len(entries))
	for k, v := range entries {
		raw[k] = toPreviewIndexRecord(v)
	}
	total := previewpkg.TrimIndexBytes(raw, totalBytes, maxBytes)
	// Keep wrapper map in sync when caller relies on in-place trimming.
	if len(raw) != len(entries) {
		for k := range entries {
			delete(entries, k)
		}
		for k, v := range raw {
			entries[k] = fromPreviewIndexRecord(v)
		}
	}
	return total
}

func toPreviewIndexRecord(v previewIndexRecord) previewpkg.IndexRecord {
	return previewpkg.IndexRecord{
		Key:           strings.TrimSpace(v.Key),
		Path:          v.Path,
		Width:         v.Width,
		SizeBytes:     v.SizeBytes,
		UpdatedAtUnix: v.UpdatedAtUnix,
		TouchedAtUnix: v.TouchedAtUnix,
		Lines:         append([]string(nil), v.Lines...),
	}
}

func fromPreviewIndexRecord(v previewpkg.IndexRecord) previewIndexRecord {
	return previewIndexRecord{
		Key:           v.Key,
		Path:          v.Path,
		Width:         v.Width,
		SizeBytes:     v.SizeBytes,
		UpdatedAtUnix: v.UpdatedAtUnix,
		TouchedAtUnix: v.TouchedAtUnix,
		Lines:         append([]string(nil), v.Lines...),
	}
}
