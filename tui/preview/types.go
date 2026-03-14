package preview

// Request describes one async preview load.
type Request struct {
	RequestID     uint64
	Key           string
	Path          string
	Width         int
	Lines         int
	Palette       ThemePalette
	IndexPath     string
	SizeBytes     int64
	UpdatedAtUnix int64
}

// LoadedMsg is emitted after a preview load.
type LoadedMsg struct {
	RequestID uint64
	Key       string
	Lines     []string
	Err       string
	Record    IndexRecord
}

// IndexPersistedMsg is emitted after index persistence.
type IndexPersistedMsg struct {
	Err string
}
