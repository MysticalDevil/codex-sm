package core

import (
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/session"
)

func TestSortSessionsByRisk(t *testing.T) {
	now := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)
	items := []session.Session{
		{SessionID: "ok-new", UpdatedAt: now.Add(2 * time.Hour), Health: session.HealthOK},
		{SessionID: "mid-new", UpdatedAt: now.Add(1 * time.Hour), Health: session.HealthMissingMeta},
		{SessionID: "high-old", UpdatedAt: now.Add(-3 * time.Hour), Health: session.HealthCorrupted},
		{SessionID: "mid-old", UpdatedAt: now.Add(-2 * time.Hour), Health: session.HealthMissingMeta},
	}

	SortSessionsByRisk(items, nil, nil)

	want := []string{"high-old", "mid-new", "mid-old", "ok-new"}
	for i, id := range want {
		if items[i].SessionID != id {
			t.Fatalf("unexpected risk sort at %d: got=%q want=%q", i, items[i].SessionID, id)
		}
	}
}

func TestCompactHomePath(t *testing.T) {
	home := "/home/test-user"

	cases := []struct {
		in   string
		want string
	}{
		{in: "/home/test-user", want: "~"},
		{in: "/home/test-user/Project/codexsm", want: "~/Project/codexsm"},
		{in: "/tmp/other", want: "/tmp/other"},
	}
	for _, tc := range cases {
		if got := CompactHomePath(tc.in, home); got != tc.want {
			t.Fatalf("CompactHomePath(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
