package core

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/MysticalDevil/codexsm/session/scanner"
)

func TestParseSortSpec(t *testing.T) {
	spec, err := ParseSortSpec("size", "asc")
	if err != nil {
		t.Fatalf("ParseSortSpec(size,asc): %v", err)
	}

	if spec.Field != SortFieldSize || spec.Order != SortOrderAsc {
		t.Fatalf("unexpected spec: %+v", spec)
	}

	spec, err = ParseSortSpec("", "")
	if err != nil {
		t.Fatalf("ParseSortSpec(default): %v", err)
	}

	if spec.Field != SortFieldUpdatedAt || spec.Order != SortOrderDesc {
		t.Fatalf("unexpected default spec: %+v", spec)
	}

	if _, err := ParseSortSpec("invalid", "asc"); err == nil {
		t.Fatal("expected invalid sort error")
	}

	if _, err := ParseSortSpec("size", "invalid"); err == nil {
		t.Fatal("expected invalid order error")
	}
}

func TestQuerySessionsSortOffsetLimit(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")

	out, err := QuerySessions(scanner.ScanSessions, root, QuerySpec{
		Selector: session.Selector{},
		SortBy:   "size",
		Order:    "asc",
		Offset:   1,
		Limit:    2,
		Now:      time.Now(),
	})
	if err != nil {
		t.Fatalf("QuerySessions: %v", err)
	}

	if out.Total == 0 {
		t.Fatalf("expected non-zero total: %+v", out)
	}

	if len(out.Items) == 0 || len(out.Items) > 2 {
		t.Fatalf("expected 1..2 items after offset+limit, got %d", len(out.Items))
	}

	for i := 1; i < len(out.Items); i++ {
		if out.Items[i-1].SizeBytes > out.Items[i].SizeBytes {
			t.Fatalf("expected size asc order, got %+v", out.Items)
		}
	}
}

func TestQuerySessionsInvalidOffset(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")

	root := filepath.Join(workspace, "sessions")
	if _, err := QuerySessions(scanner.ScanSessions, root, QuerySpec{Offset: -1}); err == nil {
		t.Fatal("expected invalid offset error")
	}
}

func TestQuerySessionsNilRepository(t *testing.T) {
	if _, err := QuerySessions(nil, "/tmp", QuerySpec{}); err == nil {
		t.Fatal("expected nil repository error")
	}
}
