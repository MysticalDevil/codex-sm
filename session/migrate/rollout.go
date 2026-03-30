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

	"github.com/MysticalDevil/codexsm/util"
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

		return nil, fmt.Errorf("walk migration rollouts under %q: %w", root, err)
	}

	return out, nil
}

func readMigrationMeta(path string) (meta migrateMeta, ok bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return migrateMeta{}, false, fmt.Errorf("open rollout file %q: %w", path, err)
	}

	defer func() {
		closeErr := f.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close rollout file: %w", closeErr)
		}
	}()

	r := bufio.NewReader(f)

	line, truncated, err := util.ReadBoundedLine(r, maxSessionMetaLineBytes)
	if err != nil && !errors.Is(err, io.EOF) {
		return migrateMeta{}, false, fmt.Errorf("read session meta line from %q: %w", path, err)
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
		return fmt.Errorf("open source rollout %q: %w", sourcePath, err)
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
		return fmt.Errorf("create destination rollout %q: %w", destPath, err)
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
			return fmt.Errorf("read rollout line from %q: %w", sourcePath, err)
		}

		if len(line) == 0 && errors.Is(err, io.EOF) {
			break
		}

		line = bytes.TrimSuffix(line, []byte{'\n'})

		rewritten, err := rewriteMigrationLine(line, sourceID, destID, destCWD)
		if err != nil {
			return fmt.Errorf("rewrite rollout line for %q: %w", sourcePath, err)
		}

		if _, err := w.Write(rewritten); err != nil {
			return fmt.Errorf("write rewritten rollout data to %q: %w", destPath, err)
		}

		if err := w.WriteByte('\n'); err != nil {
			return fmt.Errorf("write rewritten rollout newline to %q: %w", destPath, err)
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush rewritten rollout %q: %w", destPath, err)
	}

	return nil
}

func rewriteMigrationLine(line []byte, sourceID, destID, destCWD string) ([]byte, error) {
	if !jsontext.Value(line).IsValid() {
		return append([]byte(nil), line...), nil
	}

	var payload map[string]jsontext.Value
	if err := json.Unmarshal(line, &payload); err != nil {
		return append([]byte(nil), line...), nil
	}

	typ, err := decodeJSONString(payload["type"])
	if err != nil {
		return append([]byte(nil), line...), nil
	}

	switch typ {
	case "session_meta":
		p, err := decodeJSONObject(payload["payload"])
		if err != nil {
			return append([]byte(nil), line...), nil
		}

		if id, err := decodeJSONString(p["id"]); err == nil && id == sourceID {
			p["id"], err = encodeJSONString(destID)
			if err != nil {
				return nil, err
			}
		}

		if _, ok := p["cwd"]; ok {
			p["cwd"], err = encodeJSONString(destCWD)
			if err != nil {
				return nil, err
			}
		}

		payload["payload"], err = marshalJSONObject(p)
		if err != nil {
			return nil, err
		}
	case "turn_context":
		p, err := decodeJSONObject(payload["payload"])
		if err != nil {
			return append([]byte(nil), line...), nil
		}

		if _, ok := p["cwd"]; ok {
			p["cwd"], err = encodeJSONString(destCWD)
			if err != nil {
				return nil, err
			}
		}

		payload["payload"], err = marshalJSONObject(p)
		if err != nil {
			return nil, err
		}
	}

	out, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal rewritten migration line: %w", err)
	}

	return out, nil
}

func decodeJSONObject(v jsontext.Value) (map[string]jsontext.Value, error) {
	var out map[string]jsontext.Value
	if err := json.Unmarshal(v, &out); err != nil {
		return nil, fmt.Errorf("decode json object: %w", err)
	}

	return out, nil
}

func decodeJSONString(v jsontext.Value) (string, error) {
	var out string
	if err := json.Unmarshal(v, &out); err != nil {
		return "", fmt.Errorf("decode json string: %w", err)
	}

	return out, nil
}

func encodeJSONString(v string) (jsontext.Value, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encode json string: %w", err)
	}

	return jsontext.Value(b), nil
}

func marshalJSONObject(v map[string]jsontext.Value) (jsontext.Value, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encode json object: %w", err)
	}

	return jsontext.Value(b), nil
}
