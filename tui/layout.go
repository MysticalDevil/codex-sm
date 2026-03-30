package tui

const (
	// MinWidth is the minimal terminal width required by TUI in ultra mode.
	MinWidth = 64
	// MinWidthCompact is the preferred minimal terminal width for compact mode.
	MinWidthCompact = 80
	// MinWidthMedium is the preferred minimal terminal width for medium mode.
	MinWidthMedium = 96
	// MinWidthNormal is the preferred minimal terminal width for normal (non-compact) mode.
	MinWidthNormal = 118
	// MinHeight is the minimal terminal height required by TUI.
	MinHeight = 24
)

type layoutTier int

const (
	layoutTierUltra layoutTier = iota
	layoutTierCompact
	layoutTierMedium
	layoutTierFull
)

// Metrics describes the top-level dimensions used by TUI panels.
type Metrics struct {
	TotalW        int
	TotalH        int
	Compact       bool
	Tier          layoutTier
	KeysOuterH    int
	MainAreaH     int
	TreeOuterH    int
	GapW          int
	LeftOuterW    int
	RightOuterW   int
	InfoOuterH    int
	PreviewOuterH int
}

// NormalizeSize applies fallback values when terminal size is unavailable.
func NormalizeSize(width, height int) (int, int) {
	if width <= 0 {
		width = 120
	}

	if height <= 0 {
		height = 32
	}

	return width, height
}

// RenderWidth returns a width safe for rendering without hitting the terminal's last column.
func RenderWidth(width int) int {
	if width <= 1 {
		return width
	}

	return width - 1
}

func tierByWidth(width int) layoutTier {
	w := RenderWidth(width)
	switch {
	case w >= MinWidthNormal:
		return layoutTierFull
	case w >= MinWidthMedium:
		return layoutTierMedium
	case w >= MinWidthCompact:
		return layoutTierCompact
	default:
		return layoutTierUltra
	}
}

// IsUltraWidth reports whether current terminal width should use ultra rendering.
func IsUltraWidth(width int) bool {
	if width <= 0 {
		return false
	}

	return tierByWidth(width) == layoutTierUltra
}

// IsCompactWidth reports whether current terminal width should use compact rendering.
func IsCompactWidth(width int) bool {
	if width <= 0 {
		return false
	}

	tier := tierByWidth(width)

	return tier == layoutTierCompact || tier == layoutTierUltra
}

// IsTooSmall reports whether current terminal size is below supported bounds.
func IsTooSmall(width, height int) bool {
	if width <= 0 || height <= 0 {
		return false
	}

	return RenderWidth(width) < MinWidth || height < MinHeight
}

// Compute calculates panel dimensions for a normalized terminal size.
func Compute(width, height int) Metrics {
	totalW, totalH := NormalizeSize(width, height)
	totalW = RenderWidth(totalW)

	tier := tierByWidth(width)
	compact := tier == layoutTierCompact || tier == layoutTierUltra
	keysOuterH := 3
	mainAreaH := max(8, totalH-keysOuterH)

	if compact {
		gapW := 0

		leftMin := 22
		rightMin := 42

		if tier == layoutTierUltra {
			leftMin = 20
			rightMin = 20
		}

		leftOuterW := int(float64(totalW) * 0.35)
		leftOuterW = max(leftOuterW, leftMin)

		if leftOuterW > totalW-rightMin-gapW {
			leftOuterW = max(leftMin, totalW-rightMin-gapW)
		}

		rightOuterW := totalW - leftOuterW - gapW
		if rightOuterW < rightMin {
			rightOuterW = rightMin
			leftOuterW = max(leftMin, totalW-rightOuterW-gapW)
		}

		if leftOuterW+gapW+rightOuterW > totalW {
			rightOuterW = max(rightMin, totalW-leftOuterW-gapW)
		}

		infoOuterH := 4
		if infoOuterH >= mainAreaH-4 {
			infoOuterH = max(3, mainAreaH/3)
		}

		previewOuterH := mainAreaH - infoOuterH
		if previewOuterH < 5 {
			previewOuterH = 5
			infoOuterH = max(3, mainAreaH-previewOuterH)
		}

		return Metrics{
			TotalW:        totalW,
			TotalH:        totalH,
			Compact:       true,
			Tier:          tier,
			KeysOuterH:    keysOuterH,
			MainAreaH:     mainAreaH,
			TreeOuterH:    mainAreaH,
			GapW:          gapW,
			LeftOuterW:    leftOuterW,
			RightOuterW:   rightOuterW,
			InfoOuterH:    infoOuterH,
			PreviewOuterH: previewOuterH,
		}
	}

	gapW := 1
	if totalW < 132 {
		gapW = 0
	}

	leftOuterW := int(float64(totalW) * 0.28)
	leftOuterW = max(leftOuterW, 28)

	if leftOuterW > totalW-36-gapW {
		leftOuterW = max(28, totalW-36-gapW)
	}

	rightOuterW := totalW - leftOuterW - gapW
	if rightOuterW < 36 {
		rightOuterW = 36
		leftOuterW = max(28, totalW-rightOuterW-gapW)
	}

	if leftOuterW+gapW+rightOuterW > totalW {
		rightOuterW = max(36, totalW-leftOuterW-gapW)
	}

	infoOuterH := 4
	if infoOuterH >= mainAreaH-4 {
		infoOuterH = max(3, mainAreaH/4)
	}

	previewOuterH := mainAreaH - infoOuterH
	if previewOuterH < 5 {
		previewOuterH = 5
		infoOuterH = max(3, mainAreaH-previewOuterH)
	}

	return Metrics{
		TotalW:        totalW,
		TotalH:        totalH,
		Compact:       false,
		Tier:          tier,
		KeysOuterH:    keysOuterH,
		MainAreaH:     mainAreaH,
		TreeOuterH:    mainAreaH,
		GapW:          gapW,
		LeftOuterW:    leftOuterW,
		RightOuterW:   rightOuterW,
		InfoOuterH:    infoOuterH,
		PreviewOuterH: previewOuterH,
	}
}
