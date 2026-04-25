package cmd

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette holds the content-level colours for a named theme.
// These are used for all non-huh output: song list labels,
// progress lines, download summary, config prompts.
type Palette struct {
	Primary color.Color // accent / headings / ▶ label
	OK      color.Color // success, format labels (FLAC/320k)
	Fail    color.Color // errors, failed tracks
	Text    color.Color // bright foreground for titles / track names
}

func paletteFor(theme string) Palette {
	switch theme {
	case "tech":
		return Palette{
			Primary: lipgloss.Color("#0097AF"),
			OK:      lipgloss.Color("#00C97A"),
			Fail:    lipgloss.Color("#FF5555"),
			Text:    lipgloss.Color("#E2E8F0"),
		}
	case "charm":
		return Palette{
			Primary: lipgloss.Color("#F780E2"),
			OK:      lipgloss.Color("#02BA84"),
			Fail:    lipgloss.Color("#FF4672"),
			Text:    lipgloss.Color("#FFFDF5"),
		}
	case "dracula":
		return Palette{
			Primary: lipgloss.Color("#BD93F9"),
			OK:      lipgloss.Color("#50FA7B"),
			Fail:    lipgloss.Color("#FF5555"),
			Text:    lipgloss.Color("#F8F8F2"),
		}
	case "catppuccin":
		return Palette{
			Primary: lipgloss.Color("#89DCEB"),
			OK:      lipgloss.Color("#A6E3A1"),
			Fail:    lipgloss.Color("#F38BA8"),
			Text:    lipgloss.Color("#CDD6F4"),
		}
	default: // base16 — use ANSI terminal colours so it works on any palette
		return Palette{
			Primary: lipgloss.Color("12"), // bright blue
			OK:      lipgloss.Color("10"), // bright green
			Fail:    lipgloss.Color("9"),  // bright red
			Text:    lipgloss.Color("15"), // bright white
		}
	}
}
