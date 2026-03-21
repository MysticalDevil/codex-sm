package runtime

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MysticalDevil/codexsm/tui/preview"
)

type stubEngine struct {
	lastEvent Event
	effects   []Effect
	view      string
}

func (s *stubEngine) HandleEvent(ev Event) []Effect {
	s.lastEvent = ev
	return s.effects
}

func (s *stubEngine) View() string {
	return s.view
}

func TestEffectsToCmdLoadPreview(t *testing.T) {
	root := t.TempDir()

	p := filepath.Join(root, "rollout.jsonl")
	if err := os.WriteFile(p, []byte(`{"type":"session_meta","payload":{"id":"x"}}`+"\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cmd := effectsToCmd([]Effect{
		LoadPreviewEffect{
			Request: preview.Request{
				RequestID: 1,
				Key:       "k1",
				Path:      p,
				Width:     40,
				Lines:     8,
			},
		},
	})
	if cmd == nil {
		t.Fatal("expected cmd")
	}

	msg := cmd()

	loaded, ok := msg.(preview.LoadedMsg)
	if !ok {
		t.Fatalf("unexpected msg type: %T", msg)
	}

	if loaded.Key != "k1" {
		t.Fatalf("unexpected loaded key: %q", loaded.Key)
	}
}

func TestBubbleTeaModelMapsWindowSize(t *testing.T) {
	engine := &stubEngine{
		effects: []Effect{QuitEffect{}},
		view:    "ok",
	}

	model := &bubbleTeaModel{engine: engine}
	if got := model.View(); got != "ok" {
		t.Fatalf("unexpected view: %q", got)
	}

	_, cmd := model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if cmd == nil {
		t.Fatal("expected cmd")
	}

	if _, ok := engine.lastEvent.(WindowSizeEvent); !ok {
		t.Fatalf("unexpected event type: %T", engine.lastEvent)
	}
}

func TestBubbleTeaModelMapsQuitEffect(t *testing.T) {
	engine := &stubEngine{effects: []Effect{QuitEffect{}}}
	model := &bubbleTeaModel{engine: engine}

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
}
