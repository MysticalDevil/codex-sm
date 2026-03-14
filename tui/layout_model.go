package tui

import layoutpkg "github.com/MysticalDevil/codexsm/tui/layout"

const (
	MinWidth  = layoutpkg.MinWidth
	MinHeight = layoutpkg.MinHeight
)

type Metrics = layoutpkg.Metrics

func NormalizeSize(width, height int) (int, int) {
	return layoutpkg.NormalizeSize(width, height)
}

func RenderWidth(width int) int {
	return layoutpkg.RenderWidth(width)
}

func IsTooSmall(width, height int) bool {
	return layoutpkg.IsTooSmall(width, height)
}

func Compute(width, height int) Metrics {
	return layoutpkg.Compute(width, height)
}
