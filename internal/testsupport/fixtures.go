// Package testsupport provides helpers for fixture-based tests.
package testsupport

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// PrepareFixtureSandbox copies testdata fixtures into an isolated writable
// sandbox under testdata/_sandbox and returns the sandbox path.
func PrepareFixtureSandbox(t testing.TB, fixtureName string) string {
	t.Helper()

	src := filepath.Join(TestdataRoot(), "fixtures", fixtureName)
	if st, err := os.Stat(src); err != nil || !st.IsDir() {
		t.Fatalf("fixture %q not found at %s: %v", fixtureName, src, err)
	}

	sandboxRoot := filepath.Join(TestdataRoot(), "_sandbox")
	if err := os.MkdirAll(sandboxRoot, 0o755); err != nil {
		t.Fatalf("create sandbox root: %v", err)
	}

	name := sanitizeName(t.Name())

	dst := filepath.Join(sandboxRoot, name+"-"+time.Now().Format("20060102-150405.000000000"))
	if err := copyTree(src, dst); err != nil {
		t.Fatalf("copy fixture %q to sandbox: %v", fixtureName, err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(dst)
	})

	return dst
}

// TestdataRoot returns the absolute path to repository testdata/.
func TestdataRoot() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed in testsupport.TestdataRoot")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata"))
}

func sanitizeName(s string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_")

	out := replacer.Replace(s)
	if out == "" {
		return "test"
	}

	return out
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}

	defer func() {
		_ = in.Close()
	}()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()

	if copyErr != nil {
		return copyErr
	}

	if closeErr != nil {
		return closeErr
	}

	return nil
}
