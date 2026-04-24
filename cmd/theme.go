package cmd

import (
	"image/color"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// resolveTheme returns the huh.Theme for the given name.
// Falls back to ThemeBase16 for unknown names.
func resolveTheme(name string) huh.Theme {
	switch name {
	case "tech":
		return huh.ThemeFunc(techTheme)
	case "charm":
		return huh.ThemeFunc(huh.ThemeCharm)
	case "dracula":
		return huh.ThemeFunc(huh.ThemeDracula)
	case "catppuccin":
		return huh.ThemeFunc(huh.ThemeCatppuccin)
	default: // "base16" and anything else
		return huh.ThemeFunc(huh.ThemeBase16)
	}
}

// techTheme is a custom cyan/green tech-style theme with dark/light variants.
func techTheme(isDark bool) *huh.Styles {
	t := huh.ThemeBase(isDark)

	var (
		cyan   color.Color
		green  color.Color
		accent color.Color
		muted  color.Color
		fg     color.Color
	)

	if isDark {
		cyan   = lipgloss.Color("#0097AF")
		green  = lipgloss.Color("#00C97A")
		accent = lipgloss.Color("#00D4FF")
		muted  = lipgloss.Color("#4A5568")
		fg     = lipgloss.Color("#E2E8F0")
	} else {
		cyan   = lipgloss.Color("#006B80")
		green  = lipgloss.Color("#007A4A")
		accent = lipgloss.Color("#0088AA")
		muted  = lipgloss.Color("#718096")
		fg     = lipgloss.Color("#1A202C")
	}

	t.Focused.Base = t.Focused.Base.BorderForeground(cyan)
	t.Focused.Card = t.Focused.Base

	t.Focused.Title = t.Focused.Title.Foreground(cyan).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(cyan).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(muted)
	t.Focused.Directory = t.Focused.Directory.Foreground(cyan)

	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(accent)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(accent)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(accent)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(accent)

	t.Focused.Option = t.Focused.Option.Foreground(fg)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green).Bold(true)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(fg)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(muted)

	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(lipgloss.Color("#000000")).Background(cyan)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(fg).Background(muted)

	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(lipgloss.Color("#FF5555"))
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(lipgloss.Color("#FF5555"))

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(accent)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(muted)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(cyan)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.Title = t.Blurred.Title.Foreground(muted)
	t.Blurred.NoteTitle = t.Blurred.NoteTitle.Foreground(muted)

	t.Group.Title = t.Focused.Title

	return t
}
