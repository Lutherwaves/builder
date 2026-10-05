package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme is near-monochrome and warm. Color is an event: the accent marks the
// selected row and the active panel only; status uses muted tints and always
// carries a glyph and a word, never hue alone.
type Theme struct {
	Text, Muted, Faint, Hair, Accent, AccentWash, Warn, Neg color.Color
}

var darkTheme = Theme{
	Text:       lipgloss.Color("#EBE6DC"),
	Muted:      lipgloss.Color("#A69D8D"),
	Faint:      lipgloss.Color("#776F60"),
	Hair:       lipgloss.Color("#2E2921"),
	Accent:     lipgloss.Color("#C3A56A"),
	AccentWash: lipgloss.Color("#221D14"),
	Warn:       lipgloss.Color("#B98F44"),
	Neg:        lipgloss.Color("#B0563F"),
}

var lightTheme = Theme{
	Text:       lipgloss.Color("#1C1813"),
	Muted:      lipgloss.Color("#5F5A4C"),
	Faint:      lipgloss.Color("#8A8170"),
	Hair:       lipgloss.Color("#DDD5C3"),
	Accent:     lipgloss.Color("#9A7B3F"),
	AccentWash: lipgloss.Color("#ECE3CF"),
	Warn:       lipgloss.Color("#8A6420"),
	Neg:        lipgloss.Color("#9E4A34"),
}

type styles struct {
	theme                                 Theme
	text, muted, faint, hair, accent      lipgloss.Style
	label, selected, warn, neg, tabActive lipgloss.Style
}

func newStyles(dark bool) styles {
	t := lightTheme
	if dark {
		t = darkTheme
	}
	fg := func(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }
	return styles{
		theme:     t,
		text:      fg(t.Text),
		muted:     fg(t.Muted),
		faint:     fg(t.Faint),
		hair:      fg(t.Hair),
		accent:    fg(t.Accent),
		label:     fg(t.Faint),
		selected:  lipgloss.NewStyle().Foreground(t.Text).Background(t.AccentWash),
		warn:      fg(t.Warn),
		neg:       fg(t.Neg),
		tabActive: fg(t.Accent).Underline(true),
	}
}
