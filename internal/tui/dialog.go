package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

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
	return styleDialogBox.Render(rebackgroundResets(inner))
}

// rebackgroundResets keeps colorPanel painted through every embedded ANSI
// reset in s. lipgloss styles each render as a fully self-contained span —
// open code, content, then a bare reset — so a nested style that sets only a
// foreground (huh's text-input value when its field is blurred, the spinner
// glyph) still closes with a reset that wipes the panel background an outer
// Background(colorPanel) wrapper had established, for everything after it on
// that line. huh v2.0.3's Input field only threads the active theme's colour
// into the underlying textinput's *focused* style state — field_input.go's
// View sets textinput.Styles().Focused but never touches .Blurred — so a
// form's non-active fields can't be fixed by styling the theme alone;
// re-opening colorPanel immediately after each reset is the workaround.
func rebackgroundResets(s string) string {
	open := bgOpenCode(colorPanel)
	if open == "" {
		return s
	}
	parts := strings.Split(s, ansi.ResetStyle)
	for i := 0; i < len(parts)-1; i++ {
		parts[i] += ansi.ResetStyle + open
	}
	return strings.Join(parts, "")
}

// bgOpenCode returns the raw ANSI SGR sequence lipgloss emits to paint bg as
// a background colour, with no content or trailing reset — used to
// re-establish a background immediately after a reset embedded mid-string.
func bgOpenCode(bg color.Color) string {
	rendered := lipgloss.NewStyle().Background(bg).Render("\x00")
	if i := strings.IndexByte(rendered, 0); i >= 0 {
		return rendered[:i]
	}
	return ""
}
