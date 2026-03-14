package tui

import (
	"strings"

	themepkg "github.com/MysticalDevil/codexsm/tui/theme"
)

type tuiTheme struct {
	Name   string
	Colors map[string]string
}

func (t tuiTheme) hex(key, fallback string) string {
	return themepkg.Theme{Name: t.Name, Colors: t.Colors}.Hex(key, fallback)
}

func (t tuiTheme) merge(overrides map[string]string) tuiTheme {
	if len(overrides) == 0 {
		return t
	}
	if t.Colors == nil {
		t.Colors = make(map[string]string, len(overrides))
	}
	for k, v := range overrides {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		t.Colors[strings.TrimSpace(strings.ToLower(k))] = strings.TrimSpace(v)
	}
	return t
}

func defaultTUIThemeName() string {
	return themepkg.DefaultThemeName()
}

func availableTUIThemes() []string {
	return themepkg.AvailableThemeNames()
}

func resolveTUITheme(cfgName string, cfgColors map[string]string, flagName string, flagColors []string) (tuiTheme, error) {
	out, err := themepkg.Resolve(cfgName, cfgColors, flagName, flagColors)
	if err != nil {
		return tuiTheme{}, err
	}
	return tuiTheme{
		Name:   out.Name,
		Colors: out.Colors,
	}, nil
}

func parseThemeOverrides(items []string) (map[string]string, error) {
	return themepkg.ParseOverrides(items)
}

func cloneColorMap(m map[string]string) map[string]string {
	return themepkg.CloneColorMap(m)
}

func DefaultThemeName() string {
	return themepkg.DefaultThemeName()
}

func ValidateTheme(cfgName string, cfgColors map[string]string, flagName string, flagColors []string) error {
	return themepkg.Validate(cfgName, cfgColors, flagName, flagColors)
}

var builtinThemes = themepkg.BuiltinThemes

func toTheme(t tuiTheme) themepkg.Theme {
	return themepkg.Theme{Name: t.Name, Colors: t.Colors}
}
