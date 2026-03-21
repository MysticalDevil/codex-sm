package runtime

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MysticalDevil/codexsm/tui/preview"
)

type bubbleTeaRuntime struct{}

func NewBubbleTea() Runtime {
	return bubbleTeaRuntime{}
}

func (bubbleTeaRuntime) Run(engine Engine) error {
	_, err := tea.NewProgram(&bubbleTeaModel{engine: engine}, tea.WithAltScreen()).Run()
	return err
}

type bubbleTeaModel struct {
	engine Engine
}

func (m *bubbleTeaModel) Init() tea.Cmd {
	return nil
}

func (m *bubbleTeaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var event Event

	switch in := msg.(type) {
	case tea.WindowSizeMsg:
		event = WindowSizeEvent{Width: in.Width, Height: in.Height}
	case tea.KeyMsg:
		event = KeyPressedEvent{Key: in.String()}
	case preview.LoadedMsg:
		event = PreviewLoadedEvent{Message: in}
	case preview.IndexPersistedMsg:
		event = PreviewPersistedEvent{Message: in}
	default:
		return m, nil
	}

	return m, effectsToCmd(m.engine.HandleEvent(event))
}

func (m *bubbleTeaModel) View() string {
	return m.engine.View()
}

func effectsToCmd(effects []Effect) tea.Cmd {
	if len(effects) == 0 {
		return nil
	}

	cmds := make([]tea.Cmd, 0, len(effects))
	for _, effect := range effects {
		switch e := effect.(type) {
		case QuitEffect:
			return tea.Quit
		case LoadPreviewEffect:
			req := e.Request

			cmds = append(cmds, func() tea.Msg {
				return preview.Load(req)
			})
		case PersistPreviewEffect:
			indexPath := e.IndexPath
			capacity := e.Cap
			record := e.Record

			cmds = append(cmds, func() tea.Msg {
				return preview.PersistIndex(indexPath, capacity, record)
			})
		}
	}

	if len(cmds) == 0 {
		return nil
	}

	return tea.Batch(cmds...)
}
