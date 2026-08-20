package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

// newHelpModel returns a help.Model styled with the wwlog palette instead of
// bubbles' own greyscale default. Every style carries an explicit
// Background(colorPanel) — the same convention styleStatusKey and friends
// use in styles.go — since both places this renders (the footer bar and the
// help panel's dialog box) share that background colour.
func newHelpModel() help.Model {
	h := help.New()
	h.ShortSeparator = "  "
	h.FullSeparator = "    "
	h.Styles.ShortKey = lipgloss.NewStyle().Background(colorPanel).Foreground(colorTeal).Bold(true)
	h.Styles.ShortDesc = lipgloss.NewStyle().Background(colorPanel).Foreground(colorMuted)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Background(colorPanel).Foreground(colorLine)
	h.Styles.FullKey = lipgloss.NewStyle().Background(colorPanel).Foreground(colorTeal).Bold(true)
	h.Styles.FullDesc = lipgloss.NewStyle().Background(colorPanel).Foreground(colorMuted)
	h.Styles.FullSeparator = lipgloss.NewStyle().Background(colorPanel).Foreground(colorLine)
	return h
}

// helpPanelWidth returns how wide the full help panel's key columns should
// render, given the terminal width — wider than the form dialogs
// (dialogContentWidth) since it lays out three columns side by side.
func helpPanelWidth(termWidth int) int {
	w := termWidth - 12
	if w > 76 {
		w = 76
	}
	if w < 50 {
		w = 50
	}
	return w
}

// helpPanel renders the full keyboard-shortcut reference shown when the user
// presses "?", reusing renderDialog's rounded-border box — the same visual
// language as the export and date-range dialogs (dialog.go) — rather than
// introducing a new one. Composited on screen via overlayDialog (model.go),
// same as those two, so it's centred rather than anchored to an edge.
func helpPanel(hkm helpKeyMap, termWidth int) string {
	hm := newHelpModel()
	hm.SetWidth(helpPanelWidth(termWidth))
	body := hm.FullHelpView(hkm.FullHelp())
	return renderDialog("Keyboard shortcuts", body, "esc/? close")
}
