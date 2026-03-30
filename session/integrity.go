package session

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SHA256SidecarChecker verifies session integrity via "<session>.sha256" sidecar file.
// Sidecar formats accepted:
//   - "<hexhash>"
//   - "<hexhash>  <filename>"
func SHA256SidecarChecker(s Session) IntegrityCheckResult {
	p := strings.TrimSpace(s.Path)
	if p == "" {
		return IntegrityCheckResult{}
	}

	sidecar := p + ".sha256"

	raw, err := os.ReadFile(sidecar)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return IntegrityCheckResult{} // not verified is not a risk by itself
		}

		return IntegrityCheckResult{Verified: true, Match: false, Err: fmt.Errorf("read sidecar %s: %w", sidecar, err)}
	}

	expected, err := parseSHA256Sidecar(string(raw), filepath.Base(p))
	if err != nil {
		return IntegrityCheckResult{Verified: true, Match: false, Err: err}
	}

	got, err := fileSHA256Hex(p)
	if err != nil {
		return IntegrityCheckResult{Verified: true, Match: false, Err: fmt.Errorf("hash file %s: %w", p, err)}
	}

	return IntegrityCheckResult{
		Verified: true,
		Match:    strings.EqualFold(expected, got),
		Detail:   fmt.Sprintf("expected=%s got=%s", expected, got),
	}
}

func parseSHA256Sidecar(raw, baseName string) (string, error) {
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		sum := strings.TrimSpace(parts[0])
		if len(parts) != sha256.Size*2 {
			// allow lines like "<sum>  filename"
			if len(parts[0]) != sha256.Size*2 {
				return "", errors.New("invalid sha256 sidecar format")
			}
		}

		if _, err := hex.DecodeString(sum); err != nil {
			return "", fmt.Errorf("invalid sha256 hex in sidecar: %w", err)
		}

		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[len(parts)-1])

			name = strings.TrimPrefix(name, "*")
			if baseName != "" && name != "" && name != baseName {
				continue
			}
		}

		return strings.ToLower(sum), nil
	}

	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("scan sha256 sidecar: %w", err)
	}

	return "", errors.New("no valid sha256 entry in sidecar")
}

func fileSHA256Hex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for checksum %q: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash file %q: %w", path, err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
