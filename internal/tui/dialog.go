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
	titleR := styleDialogTitle.Render(title)
	hintR := styleDialogHint.Render(hint)

	// Force every row to the widest row's width with colorPanel background.
	// Without this, JoinVertical pads shorter rows with plain spaces (no
	// background), so terminal default bg shows through wherever a row is
	// narrower than the body — including the blank separator rows.
	width := lipgloss.Width(body)
	if w := lipgloss.Width(titleR); w > width {
		width = w
	}
	if w := lipgloss.Width(hintR); w > width {
		width = w
	}
	pad := lipgloss.NewStyle().Background(colorPanel).Width(width)

	inner := lipgloss.JoinVertical(lipgloss.Left,
		pad.Render(titleR),
		pad.Render(""),
		pad.Render(body),
		pad.Render(""),
		pad.Render(hintR),
	)
	return styleDialogBox.Render(inner)
}
