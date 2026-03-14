package preview

// ThemePalette stores semantic preview colors already resolved by the caller.
type ThemePalette struct {
	PrefixDefault   string
	PrefixUser      string
	PrefixAssistant string
	PrefixOther     string
	TagDanger       string
	TagDefault      string
	TagSystem       string
	TagLifecycle    string
	TagSuccess      string
}

// IndexRecord is one preview index row.
type IndexRecord struct {
	Key           string   `json:"key"`
	Path          string   `json:"path"`
	Width         int      `json:"width"`
	SizeBytes     int64    `json:"size_bytes"`
	UpdatedAtUnix int64    `json:"updated_at_unix"`
	TouchedAtUnix int64    `json:"touched_at_unix"`
	Lines         []string `json:"lines"`
}
