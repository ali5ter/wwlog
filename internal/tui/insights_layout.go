package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Geometry for the Insights tab's responsive two-column layout (#74).
//
// insightsMinColWidth is Range Summary's own floor — its widest bar row is
// roughly "  Points      <bar>  avg 36pt / 23pt target" (indent 2 + label 10
// + gaps 4 + bar 6 + value ~24 = ~46; see barWidthForRows for how the exact
// value width is measured rather than assumed). That floor is what the
// two-column breakpoint derives from, rather than being a bare literal: a
// column only appears once it can fit Range Summary's minimum bar width,
// which in practice puts the breakpoint at a 100-column terminal (vw =
// width-4 >= 96, colW = (vw-4)/2).
const (
	insightsGutter      = 4  // columns between the two content columns
	insightsMinColWidth = 46 // Range Summary's floor, see comment above
	insightsMinBarWidth = 6
	insightsMaxBarWidth = 22 // today's shared bar-width ceiling
)

// insightsColumns reports the per-column content width and whether the
// two-column layout applies. Two columns only when each gets at least
// insightsMinColWidth.
func insightsColumns(vw int) (colW int, twoCol bool) {
	candidate := (vw - insightsGutter) / 2
	if candidate < insightsMinColWidth {
		return vw, false
	}
	return candidate, true
}

// blockWidth returns the width of a multi-line block's widest line — used to
// measure the heatmap's actual rendered width (it doesn't stretch to fill
// vw; see renderHeatmap) so render() can decide whether it fits beside Range
// Summary on the same row.
func blockWidth(s string) int {
	w := 0
	for _, line := range strings.Split(s, "\n") {
		if lw := lipgloss.Width(line); lw > w {
			w = lw
		}
	}
	return w
}

// barWidthForRows computes one bar width shared by every row builder passed
// in, wide enough that each row — including its own value text, which
// varies with the data (bigger numbers, longer units) — fits within w. Each
// builder is measured once at barWidth 1 rather than 0: makeBar (nutrition.go)
// always appends a trailing "│" reference marker outside the width it's
// given, but returns "" outright for width 0 — so a width-0 measurement
// would miss that marker entirely and every row would come out one column
// too wide. Measuring at 1 keeps the marker in the sample; since each
// further unit of bar width adds exactly one more fill/empty cell, the
// real bar width is a direct +1 offset from there.
func barWidthForRows(w int, rows ...func(barWidth int) string) int {
	maxAtOne := 0
	for _, row := range rows {
		if lw := lipgloss.Width(row(1)); lw > maxAtOne {
			maxAtOne = lw
		}
	}
	return clampBarWidth(w - maxAtOne + 1)
}

func clampBarWidth(bw int) int {
	if bw > insightsMaxBarWidth {
		return insightsMaxBarWidth
	}
	if bw < insightsMinBarWidth {
		return insightsMinBarWidth
	}
	return bw
}

// sectionHeading renders a "<title>\n<rule>" block at width w, falling back
// to the short title when the long one doesn't fit.
func sectionHeading(long, short string, w int) string {
	title := long
	if lipgloss.Width(title) > w {
		title = short
	}
	ruleWidth := w - 2
	if ruleWidth < 1 {
		ruleWidth = 1
	}
	sep := styleDim.Render(strings.Repeat("─", ruleWidth))
	return styleMealHeading.Render(title) + "\n" + sep
}

// padBlock pads every line of s to exactly w cells, ANSI-truncating any line
// that overruns. Safe to use with bare spaces here specifically because
// every style the Insights tab touches is foreground-only (see makeBar,
// renderHeatmap, and the style vars in styles.go) — the moment any Insights
// style gains a Background(), every pad in this file becomes a black gap
// (see CLAUDE.md's v1.15.1 notes) and must instead adopt help.go's
// per-fragment Background() treatment.
func padBlock(s string, w int) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		switch lw := lipgloss.Width(l); {
		case lw < w:
			lines[i] = l + strings.Repeat(" ", w-lw)
		case lw > w:
			lines[i] = ansi.Truncate(l, w, "")
		}
	}
	return strings.Join(lines, "\n")
}

// joinColumns places two section blocks side by side, left padded to leftW
// and right to rightW (independent widths so the heatmap, which keeps its
// own natural width rather than a shared colW, can still pair with a wider
// Range Summary column). The gutter is folded into the left column's pad.
// If one side is blank — an empty section, e.g. no micronutrient data — the
// surviving side keeps its own width rather than expanding to fill the row,
// so column boundaries stay aligned across the rows above and below it.
func joinColumns(left string, leftW int, right string, rightW int) string {
	leftBlank, rightBlank := strings.TrimSpace(left) == "", strings.TrimSpace(right) == ""
	switch {
	case leftBlank && rightBlank:
		return ""
	case rightBlank:
		return padBlock(left, leftW)
	case leftBlank:
		return padBlock(right, rightW)
	default:
		return lipgloss.JoinHorizontal(lipgloss.Top,
			padBlock(left, leftW+insightsGutter),
			padBlock(right, rightW),
		)
	}
}

// foodRowLayout describes which columns a food-table row has room for at a
// given allotted width — see topFoodsLayout/zeroPointLayout.
type foodRowLayout struct {
	nameW      int
	showAvgPts bool
	showKcal   bool
}

const (
	foodIndentW   = 2
	foodNameW     = 34 // Width() of the name column at full size
	foodCountW    = 5
	foodTotalPtsW = 14
	foodAvgPtsW   = 12
	foodKcalW     = 15 // "  1234 kcal avg"
	foodNameMinW  = 16 // floor below which a truncated name stops being useful
)

// topFoodsLayout tiers Top Foods' columns by allotted width, shedding the
// least essential ones first: kcal average, then points average, then
// finally shrinking the name column down to foodNameMinW.
func topFoodsLayout(w int) foodRowLayout {
	const fixed = foodIndentW + foodCountW + foodTotalPtsW // 21
	switch {
	case w >= fixed+foodNameW+foodAvgPtsW+foodKcalW: // >= 82: full row
		return foodRowLayout{foodNameW, true, true}
	case w >= fixed+foodNameW+foodAvgPtsW: // >= 67: drop kcal avg
		return foodRowLayout{foodNameW, true, false}
	case w >= fixed+foodNameW: // >= 55: drop pt avg too
		return foodRowLayout{foodNameW, false, false}
	default: // shrink the name column last
		nameW := w - fixed
		if nameW < foodNameMinW {
			nameW = foodNameMinW
		}
		return foodRowLayout{nameW, false, false}
	}
}

// zeroPointLayout mirrors topFoodsLayout for the Zero-Point Foods table,
// which has no points columns to shed — only the kcal average and, at the
// narrowest widths, the name column.
func zeroPointLayout(w int) foodRowLayout {
	const fixed = foodIndentW + foodCountW // 7
	switch {
	case w >= fixed+foodNameW+foodKcalW: // >= 56: full row
		return foodRowLayout{foodNameW, false, true}
	case w >= fixed+foodNameW: // >= 41: drop kcal avg
		return foodRowLayout{foodNameW, false, false}
	default:
		nameW := w - fixed
		if nameW < foodNameMinW {
			nameW = foodNameMinW
		}
		return foodRowLayout{nameW, false, false}
	}
}
