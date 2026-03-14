package migrate

import (
	"bufio"
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type migrateMeta struct {
	ID   string
	Cwd  string
	Path string
}

func collectMigrationRollouts(root, fromCWD string) (map[string]migrateMeta, error) {
	out := map[string]migrateMeta{}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}

		meta, ok, err := readMigrationMeta(path)
		if err != nil {
			return err
		}

		if !ok || strings.TrimSpace(meta.Cwd) != fromCWD {
			return nil
		}

		out[meta.ID] = meta

		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}

		return nil, err
	}

	return out, nil
}

func readMigrationMeta(path string) (meta migrateMeta, ok bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return migrateMeta{}, false, err
	}

	defer func() {
		closeErr := f.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close rollout file: %w", closeErr)
		}
	}()

	r := bufio.NewReader(f)

	line, truncated, err := readBoundedLine(r, maxSessionMetaLineBytes)
	if err != nil && !errors.Is(err, io.EOF) {
		return migrateMeta{}, false, err
	}

	if truncated || len(line) == 0 || !jsontext.Value(line).IsValid() {
		return migrateMeta{}, false, nil
	}

	var m metaLine
	if err := json.Unmarshal(line, &m); err != nil {
		return migrateMeta{}, false, nil
	}

	if m.Type != "session_meta" || strings.TrimSpace(m.Payload.ID) == "" {
		return migrateMeta{}, false, nil
	}

	return migrateMeta{ID: m.Payload.ID, Cwd: strings.TrimSpace(m.Payload.Cwd), Path: path}, true, nil
}

func writeMigratedRollout(sourcePath, destPath, sourceID, destID, destCWD string) (err error) {
	in, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

	defer func() {
		closeErr := in.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close source rollout: %w", closeErr)
		}
	}()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create rollout dir: %w", err)
	}

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	defer func() {
		closeErr := out.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close destination rollout: %w", closeErr)
		}
	}()

	r := bufio.NewReader(in)
	w := bufio.NewWriter(out)

	for {
		line, err := r.ReadBytes('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}

		if len(line) == 0 && errors.Is(err, io.EOF) {
			break
		}

		line = bytes.TrimSuffix(line, []byte{'\n'})

		rewritten, err := rewriteMigrationLine(line, sourceID, destID, destCWD)
		if err != nil {
			return err
		}

		if _, err := w.Write(rewritten); err != nil {
			return err
		}

		if err := w.WriteByte('\n'); err != nil {
			return err
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	return w.Flush()
}

func rewriteMigrationLine(line []byte, sourceID, destID, destCWD string) ([]byte, error) {
	if !jsontext.Value(line).IsValid() {
		return append([]byte(nil), line...), nil
	}

	var payload map[string]any
	if err := json.Unmarshal(line, &payload); err != nil {
		return append([]byte(nil), line...), nil
	}

	typ, _ := payload["type"].(string)
	switch typ {
	case "session_meta":
		p, _ := payload["payload"].(map[string]any)
		if p != nil {
			if id, _ := p["id"].(string); id == sourceID {
				p["id"] = destID
			}

			if _, ok := p["cwd"]; ok {
				p["cwd"] = destCWD
			}
		}
	case "turn_context":
		p, _ := payload["payload"].(map[string]any)
		if p != nil {
			if _, ok := p["cwd"]; ok {
				p["cwd"] = destCWD
			}
		}
	}

	out, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return out, nil
}
