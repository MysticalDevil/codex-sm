package ops

import "testing"

func TestParsePreviewMode(t *testing.T) {
	tests := []struct {
		in   string
		want PreviewMode
		ok   bool
	}{
		{in: "", want: PreviewSample, ok: true},
		{in: "sample", want: PreviewSample, ok: true},
		{in: "full", want: PreviewFull, ok: true},
		{in: "none", want: PreviewNone, ok: true},
		{in: "bad", ok: false},
	}
	for _, tt := range tests {
		got, err := ParsePreviewMode(tt.in)
		if tt.ok && err != nil {
			t.Fatalf("ParsePreviewMode(%q) unexpected error: %v", tt.in, err)
		}

		if !tt.ok && err == nil {
			t.Fatalf("ParsePreviewMode(%q) expected error", tt.in)
		}

		if tt.ok && got != tt.want {
			t.Fatalf("ParsePreviewMode(%q)=%q, want=%q", tt.in, got, tt.want)
		}
	}
}
