package deleteexec

import (
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/session"
)

func TestExecuteValidationPassthrough(t *testing.T) {
	_, err := Execute(nil, session.Selector{}, Options{})
	if err == nil {
		t.Fatal("expected selector validation error")
	}
	if !strings.Contains(err.Error(), "requires at least one selector") {
		t.Fatalf("unexpected error: %v", err)
	}
}
