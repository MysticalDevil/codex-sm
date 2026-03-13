package tui

import "testing"

func TestTUICommandFlagsUseScanAndViewLimit(t *testing.T) {
	cmd := NewCommand(CommandDeps{})

	if f := cmd.Flags().Lookup("scan-limit"); f == nil {
		t.Fatal("expected --scan-limit flag")
	}
	if f := cmd.Flags().Lookup("view-limit"); f == nil {
		t.Fatal("expected --view-limit flag")
	}
	if f := cmd.Flags().Lookup("limit"); f != nil {
		t.Fatal("did not expect legacy --limit flag")
	}
}
