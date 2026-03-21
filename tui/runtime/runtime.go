package runtime

import "github.com/MysticalDevil/codexsm/tui/preview"

// Engine is the runtime-agnostic TUI state machine.
type Engine interface {
	HandleEvent(Event) []Effect
	View() string
}

// Runtime runs one TUI engine using a concrete event loop implementation.
type Runtime interface {
	Run(engine Engine) error
}

// Event is an engine-level input event independent from UI frameworks.
type Event interface {
	event()
}

type WindowSizeEvent struct {
	Width  int
	Height int
}

func (WindowSizeEvent) event() {}

type KeyPressedEvent struct {
	Key string
}

func (KeyPressedEvent) event() {}

type PreviewLoadedEvent struct {
	Message preview.LoadedMsg
}

func (PreviewLoadedEvent) event() {}

type PreviewPersistedEvent struct {
	Message preview.IndexPersistedMsg
}

func (PreviewPersistedEvent) event() {}

// Effect is an engine output that must be executed by runtime adapter.
type Effect interface {
	effect()
}

type QuitEffect struct{}

func (QuitEffect) effect() {}

type LoadPreviewEffect struct {
	Request preview.Request
}

func (LoadPreviewEffect) effect() {}

type PersistPreviewEffect struct {
	IndexPath string
	Cap       int
	Record    preview.IndexRecord
}

func (PersistPreviewEffect) effect() {}
