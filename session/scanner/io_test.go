package scanner

import (
	"bufio"
	"bytes"
	"testing"
)

func TestReadBoundedLine_AllowsLinesLargerThanReaderBuffer(t *testing.T) {
	lineBytes := bytes.Repeat([]byte("a"), 12<<10)
	lineBytes = append(lineBytes, '\n')

	r := bufio.NewReaderSize(bytes.NewReader(lineBytes), 4<<10)
	line, truncated, err := readBoundedLine(r, maxSessionMetaLineBytes)
	if err != nil {
		t.Fatalf("readBoundedLine: %v", err)
	}
	if truncated {
		t.Fatal("expected non-truncated line")
	}
	if got, want := len(line), 12<<10; got != want {
		t.Fatalf("line length=%d want=%d", got, want)
	}
}
