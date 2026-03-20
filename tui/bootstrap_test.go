package tui

import "testing"

func TestResolveSource(t *testing.T) {
	t.Run("defaults to sessions", func(t *testing.T) {
		got, err := resolveSource("", "")
		if err != nil {
			t.Fatalf("resolveSource: %v", err)
		}

		if got != "sessions" {
			t.Fatalf("expected sessions, got %q", got)
		}
	})

	t.Run("uses config when flag empty", func(t *testing.T) {
		got, err := resolveSource("", "trash")
		if err != nil {
			t.Fatalf("resolveSource: %v", err)
		}

		if got != "trash" {
			t.Fatalf("expected trash, got %q", got)
		}
	})

	t.Run("rejects invalid source", func(t *testing.T) {
		if _, err := resolveSource("bad", ""); err == nil {
			t.Fatal("expected invalid source error")
		}
	})
}
