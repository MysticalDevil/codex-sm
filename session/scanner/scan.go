package scanner

import (
	"bufio"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/session"
)

// ScanSessions walks the sessions root and parses each .jsonl file into Session metadata.
func ScanSessions(root string) ([]session.Session, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("sessions root is empty")
	}

	var out []session.Session
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".jsonl" {
			return nil
		}
		s, err := scanOne(path)
		if err != nil {
			return err
		}
		out = append(out, s)
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []session.Session{}, nil
		}
		return nil, err
	}
	return out, nil
}

// ScanSessionsLimited scans a root and retains only the best limit sessions
// according to less. A limit <= 0 falls back to full ScanSessions behavior.
func ScanSessionsLimited(root string, limit int, less func(a, b session.Session) bool) ([]session.Session, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("sessions root is empty")
	}
	if limit <= 0 {
		return ScanSessions(root)
	}
	if less == nil {
		return nil, fmt.Errorf("limited scan comparator is nil")
	}

	out := make([]session.Session, 0, limit)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		s, err := scanOne(path)
		if err != nil {
			return err
		}
		if len(out) < limit {
			out = append(out, s)
			return nil
		}
		worst := 0
		for i := 1; i < len(out); i++ {
			if less(out[worst], out[i]) {
				worst = i
			}
		}
		if less(s, out[worst]) {
			out[worst] = s
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []session.Session{}, nil
		}
		return nil, err
	}
	slices.SortStableFunc(out, func(a, b session.Session) int {
		switch {
		case less(a, b):
			return -1
		case less(b, a):
			return 1
		default:
			return 0
		}
	})
	return out, nil
}

func scanOne(path string) (session.Session, error) {
	info, err := os.Stat(path)
	if err != nil {
		return session.Session{}, err
	}

	s := session.Session{
		Path:      path,
		UpdatedAt: info.ModTime(),
		SizeBytes: info.Size(),
		Health:    session.HealthOK,
	}

	fallbackID := sessionIDFromFilename(filepath.Base(path))
	if fallbackID != "" {
		s.SessionID = fallbackID
	}

	f, err := os.Open(path)
	if err != nil {
		s.Health = session.HealthCorrupted
		if s.CreatedAt.IsZero() {
			s.CreatedAt = s.UpdatedAt
		}
		return s, nil
	}
	closeScanFile := func() {
		if closeErr := f.Close(); closeErr != nil {
			s.Health = session.HealthCorrupted
			if s.CreatedAt.IsZero() {
				s.CreatedAt = s.UpdatedAt
			}
		}
	}

	r := bufio.NewReader(f)
	line, truncated, err := readBoundedLine(r, maxSessionMetaLineBytes)
	if err != nil && !errors.Is(err, io.EOF) {
		s.Health = session.HealthCorrupted
		if s.CreatedAt.IsZero() {
			s.CreatedAt = s.UpdatedAt
		}
		closeScanFile()
		return s, nil
	}
	if truncated {
		s.Health = session.HealthCorrupted
		s.CreatedAt = s.UpdatedAt
		closeScanFile()
		return s, nil
	}
	if len(line) == 0 {
		s.Health = session.HealthMissingMeta
		s.CreatedAt = s.UpdatedAt
		closeScanFile()
		return s, nil
	}

	var m metaLine
	if !jsontext.Value(line).IsValid() {
		s.Health = session.HealthCorrupted
		s.CreatedAt = s.UpdatedAt
		closeScanFile()
		return s, nil
	}
	if err := json.Unmarshal(line, &m); err != nil {
		s.Health = session.HealthCorrupted
		s.CreatedAt = s.UpdatedAt
		closeScanFile()
		return s, nil
	}

	if m.Type != "session_meta" || strings.TrimSpace(m.Payload.ID) == "" {
		s.Health = session.HealthMissingMeta
		s.CreatedAt = s.UpdatedAt
		closeScanFile()
		return s, nil
	}

	s.SessionID = m.Payload.ID
	s.HostDir = strings.TrimSpace(m.Payload.Cwd)
	if ts, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(m.Payload.Timestamp)); err == nil {
		s.CreatedAt = ts
	} else {
		s.CreatedAt = s.UpdatedAt
	}
	s.Head = readConversationHead(r)

	closeScanFile()
	return s, nil
}
