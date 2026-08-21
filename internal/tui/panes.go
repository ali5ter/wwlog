package tui

import "charm.land/lipgloss/v2"

// paneFrameWidth/paneFrameHeight are how much space paneBox's border and
// padding cost a pane's content budget: 1 border + 1 padding cell per
// horizontal side, 1 border row per vertical side (no vertical padding —
// that would cost rows for no real benefit here).
const (
	paneFrameWidth  = 4
	paneFrameHeight = 2

	// paneFilterRows is the filter bar + separator occupying the top of the
	// list pane's content, above the date list itself.
	paneFilterRows = 2

	// detailScrollStep is how many lines up/down moves the detail pane's
	// content per press — shift+up/down (always available) or plain
	// up/down while the detail pane has keyboard focus.
	detailScrollStep = 3
)

// paneBox wraps content in a focus-aware rounded box — colorPurple border
// when this pane has keyboard focus, colorLine otherwise — so which pane
// (date list or detail) currently receives up/down is always visually
// unambiguous. Mirrors styleDialogBox's established RoundedBorder
// convention (export/range dialogs) rather than introducing a new visual
// language. Matches colorPurple deliberately: a list's own selected row
// also uses colorPurple (see newDateList's SelectedTitle) — selection and
// focus share one "you are here" colour, distinct from colorTeal's general
// brand/chrome role (badges, tabs, gauges).
func paneBox(content string, outerWidth, outerHeight int, focused bool) string {
	c := colorLine
	if focused {
		c = colorPurple
	}
	return lipgloss.NewStyle().
		Width(outerWidth).
		Height(outerHeight).
		MaxHeight(outerHeight).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c).
		Render(content)
}

// paneContentWidth/paneContentHeight return how large a pane's own content
// (a list.Model's SetSize, or a viewport's Width/Height) needs to be so it
// fits exactly inside paneBox's border and padding, clamped at 0 so a very
// narrow/short terminal can't drive either negative.
func paneContentWidth(outerWidth int) int {
	w := outerWidth - paneFrameWidth
	if w < 0 {
		return 0
	}
	return w
}

func paneContentHeight(outerHeight int) int {
	h := outerHeight - paneFrameHeight
	if h < 0 {
		return 0
	}
	return h
}

// listPaneWidth/detailPaneWidth split a tab's total width between the
// date-list pane and the detail pane — single source of truth, previously
// a bare width/3 literal duplicated in both tabs' view()/resize().
func listPaneWidth(totalWidth int) int {
	return totalWidth / 3
}

func detailPaneWidth(totalWidth int) int {
	return totalWidth - listPaneWidth(totalWidth)
}

// dateRowAtPoint returns the absolute list index at the given terminal
// coordinate, or false if the point is not on a list row. Layout: the
// global header + separator (2 rows), the pane box's own top border (1
// row), then the filter bar + separator (2 rows) inside the list pane's
// content — the list's title is hidden (see newDateList), so the date
// rows start immediately after. Date rows follow with the default
// delegate's item height + 1 row of spacing between items.
func dateRowAtPoint(x, y, totalWidth, listPage, perPage, itemCount int) (int, bool) {
	lw := listPaneWidth(totalWidth)
	if x < 0 || x >= lw {
		return 0, false
	}
	const headerRows = 2     // global header + separator
	const paneBorderRows = 1 // pane box top border
	rowStride := defaultDelegateRowStride()
	rowsTop := headerRows + paneBorderRows + paneFilterRows
	if y < rowsTop {
		return 0, false
	}
	first := listPage * perPage
	offset := (y - rowsTop) / rowStride
	idx := first + offset
	if idx < 0 || idx >= itemCount {
		return 0, false
	}
	return idx, true
}

// defaultDelegateRowStride returns the on-screen rows occupied by one item
// plus the spacing below it under bubbles' default delegate (height 2 +
// spacing 1 = 3 rows per item slot).
func defaultDelegateRowStride() int { return 3 }
