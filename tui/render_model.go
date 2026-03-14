package tui

import (
	"strings"

	renderpkg "github.com/MysticalDevil/codexsm/tui/render"
)

func renderKeysLine(width int, th tuiTheme) string {
	return renderpkg.RenderKeysLine(width, toTheme(th))
}

func buildPreviewScrollBar(start, end, total, width int) string {
	return renderpkg.BuildPreviewScrollBar(start, end, total, width)
}

func (m tuiModel) colorHex(key string) string {
	th := m.theme
	if strings.TrimSpace(th.Name) == "" || len(th.Colors) == 0 {
		th = tuiTheme{
			Name:   defaultTUIThemeName(),
			Colors: cloneColorMap(builtinThemes[defaultTUIThemeName()]),
		}
	}
	fallback := builtinThemes[defaultTUIThemeName()][key]
	return th.hex(key, fallback)
}
