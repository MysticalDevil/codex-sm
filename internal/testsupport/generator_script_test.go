package testsupport

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenSeededSessionsScriptExtremeModes(t *testing.T) {
	repoRoot := filepath.Join(TestdataRoot(), "..")
	outputRoot := t.TempDir()
	scriptPath := filepath.Join(repoRoot, "scripts", "gen_seeded_sessions.py")

	cmd := exec.Command(
		"python3",
		scriptPath,
		"--seed", "20260309",
		"--count", "2",
		"--large-file-count", "1",
		"--oversize-meta-count", "1",
		"--oversize-user-count", "1",
		"--oversize-assistant-count", "1",
		"--no-newline-count", "1",
		"--mixed-corrupt-huge-count", "1",
		"--unicode-wide-count", "1",
		"--risk-missing-meta-count", "1",
		"--risk-corrupted-count", "1",
		"--long-message-bytes", "2048",
		"--meta-line-bytes", "1024",
		"--large-file-target-bytes", "4096",
		"--payload-shape", "text-only",
		"--omit-final-newline",
		"--output-root", outputRoot,
	)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run generator: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "oversize_user=1") || !strings.Contains(text, "unicode_wide=1") {
		t.Fatalf("unexpected generator summary: %s", text)
	}

	filesByPrefix := map[string]string{}
	if err := filepath.WalkDir(outputRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		for _, prefix := range []string{
			"large-session-",
			"oversize-meta-",
			"oversize-user-",
			"oversize-assistant-",
			"no-newline-",
			"mixed-corrupt-huge-",
			"unicode-wide-",
		} {
			if strings.HasPrefix(base, prefix) {
				filesByPrefix[prefix] = path
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("walk output: %v", err)
	}

	for _, prefix := range []string{
		"large-session-",
		"oversize-meta-",
		"oversize-user-",
		"oversize-assistant-",
		"no-newline-",
		"mixed-corrupt-huge-",
		"unicode-wide-",
	} {
		if filesByPrefix[prefix] == "" {
			t.Fatalf("missing generated file for prefix %s", prefix)
		}
	}

	noNewlineBytes, err := os.ReadFile(filesByPrefix["no-newline-"])
	if err != nil {
		t.Fatalf("read no-newline file: %v", err)
	}
	if len(noNewlineBytes) == 0 || noNewlineBytes[len(noNewlineBytes)-1] == '\n' {
		t.Fatalf("expected no trailing newline in %s", filesByPrefix["no-newline-"])
	}

	oversizeUserBytes, err := os.ReadFile(filesByPrefix["oversize-user-"])
	if err != nil {
		t.Fatalf("read oversize-user file: %v", err)
	}
	oversizeUserText := string(oversizeUserBytes)
	if !strings.Contains(oversizeUserText, `"text":"USER-LONG-START`) {
		t.Fatalf("expected text-only payload in oversize-user fixture: %s", oversizeUserText)
	}
	if strings.Contains(oversizeUserText, `"content":[`) {
		t.Fatalf("did not expect content array in text-only payload: %s", oversizeUserText)
	}
}
