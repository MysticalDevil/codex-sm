package tui

import (
	"strings"

	"github.com/MysticalDevil/codexsm/tui/preview"
)

type angleTagTone int

const (
	angleTagToneDefault angleTagTone = iota
	angleTagToneSystem
	angleTagToneLifecycle
	angleTagToneDanger
	angleTagToneSuccess
)

func (m *tuiModel) previewFor(path string, width, lines int) []string {
	key := preview.CacheKeyForSession(path, width, 0, 0)
	if cached, ok := m.previewCacheGet(key); ok {
		return cached
	}
	out := buildPreviewLines(path, width, lines, m.theme)
	m.previewCachePut(key, out)
	return out
}

func buildPreviewLines(path string, width, lines int, theme tuiTheme) []string {
	return preview.BuildLines(path, width, lines, previewPalette(theme))
}

func previewPalette(theme tuiTheme) preview.ThemePalette {
	def := builtinThemes[defaultTUIThemeName()]
	return preview.ThemePalette{
		PrefixDefault:   theme.hex("prefix_default", def["prefix_default"]),
		PrefixUser:      theme.hex("prefix_user", def["prefix_user"]),
		PrefixAssistant: theme.hex("prefix_assistant", def["prefix_assistant"]),
		PrefixOther:     theme.hex("prefix_other", def["prefix_other"]),
		TagDanger:       theme.hex("tag_danger", def["tag_danger"]),
		TagDefault:      theme.hex("tag_default", def["tag_default"]),
		TagSystem:       theme.hex("tag_system", def["tag_system"]),
		TagLifecycle:    theme.hex("tag_lifecycle", def["tag_lifecycle"]),
		TagSuccess:      theme.hex("tag_success", def["tag_success"]),
	}
}

func classifyAngleTag(tag string) angleTagTone {
	name := strings.TrimSpace(tag)
	name = strings.TrimPrefix(name, "<")
	name = strings.TrimSuffix(name, ">")
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "/")
	if i := strings.IndexAny(name, " \t"); i >= 0 {
		name = name[:i]
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return angleTagToneDefault
	}
	if strings.Contains(name, "error") || strings.Contains(name, "fail") || strings.Contains(name, "abort") || strings.Contains(name, "panic") {
		return angleTagToneDanger
	}
	if strings.Contains(name, "ok") || strings.Contains(name, "success") || strings.Contains(name, "done") {
		return angleTagToneSuccess
	}
	if strings.Contains(name, "mode") || strings.Contains(name, "context") || strings.Contains(name, "permission") || strings.Contains(name, "sandbox") || strings.Contains(name, "instruction") {
		return angleTagToneSystem
	}
	if strings.Contains(name, "turn") || strings.Contains(name, "session") || strings.Contains(name, "meta") || strings.Contains(name, "event") {
		return angleTagToneLifecycle
	}
	return angleTagToneDefault
}
