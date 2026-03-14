package ops

import (
	"bytes"
	"testing"
)

func TestIsInteractiveReader(t *testing.T) {
	if IsInteractiveReader(&bytes.Buffer{}) {
		t.Fatal("bytes buffer must not be treated as interactive reader")
	}
}

func TestConfirmDeleteNonInteractive(t *testing.T) {
	ok, err := ConfirmDelete(&bytes.Buffer{}, &bytes.Buffer{}, 2, false)
	if err == nil {
		t.Fatal("expected non-interactive error")
	}

	if ok {
		t.Fatal("non-interactive confirm must not approve")
	}
}

func TestConfirmRestoreNonInteractive(t *testing.T) {
	ok, err := ConfirmRestore(&bytes.Buffer{}, &bytes.Buffer{}, 2)
	if err == nil {
		t.Fatal("expected non-interactive error")
	}

	if ok {
		t.Fatal("non-interactive confirm must not approve")
	}
}
