package tui

import "charm.land/lipgloss/v2"

// dialogContentWidth returns the inner content width (form width) for a dialog
// box, given the available terminal width. Bounded between 44 and 60 columns
// so dialogs feel compact regardless of terminal size.
func dialogContentWidth(termWidth int) int {
	w := termWidth / 2
	if w < 44 {
		w = 44
	}
	if w > 60 {
		w = 60
	}
	return w
}

// renderDialog wraps form content in a bordered, panel-coloured dialog box
// with a teal title at the top and a muted hint line at the bottom. The
// result is intended to be composited over the main TUI by overlayDialog.
func renderDialog(title, body, hint string) string {
	inner := lipgloss.JoinVertical(lipgloss.Left,
		styleDialogTitle.Render(title),
		"",
		body,
		"",
		styleDialogHint.Render(hint),
	)
	return styleDialogBox.Render(inner)
}
