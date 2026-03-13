package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeleteRestoreDoNotDirectlyScanOrFilterSessions(t *testing.T) {
	for _, name := range []string{"delete.go", "restore.go"} {
		path := filepath.Join(".", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		src := string(data)
		if strings.Contains(src, "session.ScanSessions(") {
			t.Fatalf("%s should not call session.ScanSessions directly", name)
		}
		if strings.Contains(src, "session.FilterSessions(") {
			t.Fatalf("%s should not call session.FilterSessions directly", name)
		}
	}
}
