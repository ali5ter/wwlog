package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// renderShortHelp lays out the footer's pinned bindings ("r range  e
// export  ? help") as a single line. Hand-rolled for the same reason as
// renderHelpColumns: bubbles/help.Model.ShortHelpView joins each key and
// description with a literal unstyled " ", which drops the background
// between them.
func renderShortHelp(bindings []key.Binding) string {
	keyStyle := lipgloss.NewStyle().Background(colorPanel).Foreground(colorTeal).Bold(true)
	descStyle := lipgloss.NewStyle().Background(colorPanel).Foreground(colorMuted)
	gap := lipgloss.NewStyle().Background(colorPanel).Render(" ")
	sep := lipgloss.NewStyle().Background(colorPanel).Render("  ")

	var parts []string
	for i, b := range bindings {
		if i > 0 {
			parts = append(parts, sep)
		}
		parts = append(parts, keyStyle.Render(b.Help().Key), gap, descStyle.Render(b.Help().Desc))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderHelpColumns lays out FullHelp()'s groups as a grid, one column per
// group. Deliberately does not use help.Model.FullHelpView: that method
// joins each row's key and description with a literal unstyled " ", and
// lipgloss.JoinVertical/JoinHorizontal pad ragged row widths/column heights
// with unstyled space too — all three drop the background back to the
// terminal default wherever they apply, which reads as a black rectangle
// against colorPanel. Same root cause as the huh theme fix in splash.go.
//
// The fix here is the same one dialog.go's renderDialog already uses for a
// single line, generalized to a grid: every fragment — key, gap, description,
// trailing pad, blank filler row — is its own Style.Render() call with
// Background(colorPanel) set, and no fragment is ever re-wrapped in a second
// enclosing style (nesting a Width()-padding style around text that already
// contains another style's ANSI reset reintroduces the exact same bug).
func renderHelpColumns(groups [][]key.Binding) string {
	keyStyle := lipgloss.NewStyle().Background(colorPanel).Foreground(colorTeal).Bold(true)
	descStyle := lipgloss.NewStyle().Background(colorPanel).Foreground(colorMuted)
	fill := lipgloss.NewStyle().Background(colorPanel)

	maxRows := 0
	for _, g := range groups {
		if len(g) > maxRows {
			maxRows = len(g)
		}
	}

	var cols []string
	for _, g := range groups {
		if len(g) == 0 {
			continue
		}
		keyWidth := 0
		for _, b := range g {
			if w := lipgloss.Width(b.Help().Key); w > keyWidth {
				keyWidth = w
			}
		}

		rows := make([]string, len(g))
		for i, b := range g {
			rows[i] = lipgloss.JoinHorizontal(lipgloss.Top,
				keyStyle.Width(keyWidth).Render(b.Help().Key),
				fill.Render(" "),
				descStyle.Render(b.Help().Desc),
			)
		}

		colWidth := 0
		for _, r := range rows {
			if w := lipgloss.Width(r); w > colWidth {
				colWidth = w
			}
		}
		for i, r := range rows {
			if pad := colWidth - lipgloss.Width(r); pad > 0 {
				rows[i] = lipgloss.JoinHorizontal(lipgloss.Top, r, fill.Render(strings.Repeat(" ", pad)))
			}
		}
		for len(rows) < maxRows {
			rows = append(rows, fill.Render(strings.Repeat(" ", colWidth)))
		}

		cols = append(cols, lipgloss.JoinVertical(lipgloss.Left, rows...))
	}

	// Built as its own maxRows-tall block, not a single line — a
	// single-line block placed alongside maxRows-tall columns would force
	// JoinHorizontal to vertically stretch it, and the rows it synthesizes
	// to do that are unstyled, reintroducing the same bug one level up.
	sepLines := make([]string, maxRows)
	for i := range sepLines {
		sepLines[i] = fill.Render("    ")
	}
	sep := lipgloss.JoinVertical(lipgloss.Left, sepLines...)

	var parts []string
	for i, c := range cols {
		if i > 0 {
			parts = append(parts, sep)
		}
		parts = append(parts, c)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// helpPanel renders the full keyboard-shortcut reference shown when the user
// presses "?", reusing renderDialog's rounded-border box — the same visual
// language as the export and date-range dialogs (dialog.go) — rather than
// introducing a new one. Composited on screen via overlayDialog (model.go),
// same as those two, so it's centred rather than anchored to an edge.
func helpPanel(hkm helpKeyMap) string {
	body := renderHelpColumns(hkm.FullHelp())
	return renderDialog("Keyboard shortcuts", body, "esc/? close")
}
