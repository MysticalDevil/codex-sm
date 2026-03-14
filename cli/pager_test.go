package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestApplyPagerChoice(t *testing.T) {
	type tc struct {
		name     string
		page     int
		pages    int
		in       string
		wantPage int
		wantAct  pagerAction
	}

	cases := []tc{
		{name: "quit", page: 2, pages: 5, in: "q\n", wantPage: 2, wantAct: pagerActionQuit},
		{name: "all", page: 2, pages: 5, in: "a\n", wantPage: 2, wantAct: pagerActionAll},
		{name: "first", page: 2, pages: 5, in: "g\n", wantPage: 0, wantAct: pagerActionContinue},
		{name: "last", page: 0, pages: 5, in: "G\n", wantPage: 4, wantAct: pagerActionContinue},
		{name: "next default", page: 1, pages: 5, in: "x\n", wantPage: 2, wantAct: pagerActionContinue},
		{name: "back clamp", page: 0, pages: 5, in: "k\n", wantPage: 0, wantAct: pagerActionContinue},
		{name: "empty pages", page: 0, pages: 0, in: "\n", wantPage: 0, wantAct: pagerActionQuit},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotPage, gotAct := applyPagerChoice(c.page, c.pages, c.in)
			if gotPage != c.wantPage || gotAct != c.wantAct {
				t.Fatalf("applyPagerChoice(%d,%d,%q)=(%d,%d), want (%d,%d)",
					c.page, c.pages, c.in, gotPage, gotAct, c.wantPage, c.wantAct)
			}
		})
	}
}

func TestWriteWithPagerBypassWhenPagerDisabled(t *testing.T) {
	out := &bytes.Buffer{}
	text := "line1\nline2\n"

	if err := writeWithPager(out, text, false, 2, false); err != nil {
		t.Fatalf("writeWithPager: %v", err)
	}

	if out.String() != text {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestWriteWithPagerAllMode(t *testing.T) {
	prevTerminal := isTerminalWriterForPager

	isTerminalWriterForPager = func(_ io.Writer) bool { return true }

	defer func() { isTerminalWriterForPager = prevTerminal }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	if _, err := w.WriteString("a\n"); err != nil {
		t.Fatalf("write stdin: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close stdin writer: %v", err)
	}

	origStdin := os.Stdin

	os.Stdin = r

	defer func() { os.Stdin = origStdin }()
	defer func() { _ = r.Close() }()

	out := &bytes.Buffer{}
	text := "HEADER\nrow1\nrow2\nrow3\nshowing 3 sessions\n"

	if err := writeWithPager(out, text, true, 2, true); err != nil {
		t.Fatalf("writeWithPager: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "-- Page 1/2 --") {
		t.Fatalf("expected pager prompt, got %q", got)
	}

	if !strings.Contains(got, "row3") || !strings.Contains(got, "showing 3 sessions") {
		t.Fatalf("expected streamed remaining rows and footer, got %q", got)
	}
}
