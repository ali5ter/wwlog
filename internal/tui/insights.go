package tui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ali5ter/wwlog/internal/api"
)

type insightsModel struct {
	viewport    viewport.Model
	logs        []*api.DayLog
	width       int
	height      int
	initialized bool
}

func newInsightsModel(logs []*api.DayLog, width, height int) insightsModel {
	vp := viewport.New(viewport.WithWidth(width-2), viewport.WithHeight(height))
	vp.MouseWheelEnabled = true
	m := insightsModel{
		viewport:    vp,
		logs:        logs,
		width:       width,
		height:      height,
		initialized: true,
	}
	m.viewport.SetContent(m.render())
	return m
}

func (m insightsModel) update(msg tea.Msg) (insightsModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, keys.ScrollUp):
			m.viewport.ScrollUp(3)
			return m, nil
		case key.Matches(msg, keys.ScrollDown):
			m.viewport.ScrollDown(3)
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m insightsModel) view() string {
	return lipgloss.NewStyle().Padding(0, 1).Render(m.viewport.View())
}

func (m *insightsModel) resize(width, height int) {
	if !m.initialized {
		return
	}
	m.width = width
	m.height = height
	m.viewport.SetWidth(width - 2)
	m.viewport.SetHeight(height)
	m.viewport.SetContent(m.render())
}

func (m *insightsModel) render() string {
	if len(m.logs) == 0 {
		return styleFoodPortion.Render("No data available.")
	}

	vw := m.width - 4
	if vw < 40 {
		vw = 40
	}
	barWidth := 22
	if barWidth > vw-30 {
		barWidth = vw - 30
	}
	if barWidth < 8 {
		barWidth = 8
	}
	sep := styleDim.Render(strings.Repeat("─", vw-2))

	summary := api.ComputeRangeSummary(m.logs)
	meals := api.MealStats(m.logs)
	macros := api.AvgMacroBreakdown(m.logs)
	// TopFoods returns all foods sorted by total points; filter zeros so only
	// point-costing foods appear here (zero-point foods get their own section).
	var foods []api.FoodStat
	for _, f := range api.TopFoods(m.logs, 0) {
		if f.TotalPts <= 0 {
			break
		}
		foods = append(foods, f)
		if len(foods) == 20 {
			break
		}
	}

	var b strings.Builder

	// ── Points Heatmap ─────────────────────────────────────────────
	fmt.Fprintf(&b, "%s\n%s\n\n", styleMealHeading.Render("Points Budget  (day by day)"), sep)
	fmt.Fprintf(&b, "%s\n\n", renderHeatmap(m.logs, vw))

	// ── Range Summary ──────────────────────────────────────────────
	fmt.Fprintf(&b, "%s\n%s\n\n", styleMealHeading.Render("Range Summary"), sep)

	daysStr := fmt.Sprintf("%d days  ·  %d food items logged", summary.Days, summary.TotalItems)
	fmt.Fprintf(&b, "  %s\n\n", styleDetailValue.Render(daysStr))

	if summary.AvgDailyTarget > 0 {
		ptsBar := makeBar(summary.AvgDailyPts, summary.AvgDailyTarget, barWidth)
		fmt.Fprintf(&b, "  %s  %s  %s\n",
			lipgloss.NewStyle().Width(10).Render(styleDetailLabel.Render("Points")),
			ptsBar,
			styleDetailValue.Render(fmt.Sprintf("avg %.0fpt / %.0fpt target", summary.AvgDailyPts, summary.AvgDailyTarget)),
		)
		budgetStr := fmt.Sprintf("%d days on/under budget  ·  %d days over", summary.DaysUnderBudget, summary.DaysOverBudget)
		fmt.Fprintf(&b, "  %s\n", styleFoodPortion.Render(budgetStr))
	}
	if summary.AvgDailyCals > 0 {
		calBar := makeBar(summary.AvgDailyCals, 2000, barWidth)
		fmt.Fprintf(&b, "  %s  %s  %s\n",
			lipgloss.NewStyle().Width(10).Render(styleDetailLabel.Render("Calories")),
			calBar,
			styleDetailValue.Render(fmt.Sprintf("avg %.0f kcal / day", summary.AvgDailyCals)),
		)
		fmt.Fprintf(&b, "  %s\n", styleFoodPortion.Render("bar shows % of 2000 kcal reference"))
	}

	// ── Points by Meal ─────────────────────────────────────────────
	fmt.Fprintf(&b, "\n%s\n%s\n\n", styleMealHeading.Render("Points by Meal  (average per day)"), sep)

	var maxMealPts float64
	for _, ms := range meals {
		if ms.AvgPts > maxMealPts {
			maxMealPts = ms.AvgPts
		}
	}
	if maxMealPts == 0 {
		maxMealPts = 1
	}
	for _, ms := range meals {
		bar := makeBar(ms.AvgPts, maxMealPts, barWidth)
		label := lipgloss.NewStyle().Width(12).Render(styleDetailLabel.Render(ms.Symbol + "  " + ms.Name))
		val := styleDetailValue.Render(fmt.Sprintf("%.1fpt", ms.AvgPts))
		cal := styleFoodPortion.Render(fmt.Sprintf("  %.0f kcal", ms.AvgCals))
		fmt.Fprintf(&b, "  %s  %s  %s%s\n", label, bar, val, cal)
	}

	// ── Macro Distribution ─────────────────────────────────────────
	fmt.Fprintf(&b, "\n%s\n%s\n\n", styleMealHeading.Render("Macro Distribution  (average daily, % of calories)"), sep)

	if macros.ProteinG+macros.CarbsG+macros.FatG == 0 {
		fmt.Fprintf(&b, "  %s\n", styleFoodPortion.Render("No nutrition data available."))
	} else {
		writeMacroBar(&b, "Protein", macros.ProteinPct, macros.ProteinG, "g", barWidth)
		writeMacroBar(&b, "Carbs", macros.CarbsPct, macros.CarbsG, "g", barWidth)
		writeMacroBar(&b, "Fat", macros.FatPct, macros.FatG, "g", barWidth)
		if macros.AlcoholG > 0 {
			writeMacroBar(&b, "Alcohol", macros.AlcoholPct, macros.AlcoholG, "g", barWidth)
		}
		fmt.Fprintf(&b, "\n  %s\n", styleFoodPortion.Render("Recommended: ~20% protein  ·  ~50% carbs  ·  ~30% fat"))
	}

	// ── Top Foods by Points ─────────────────────────────────────────
	fmt.Fprintf(&b, "\n%s\n%s\n\n", styleMealHeading.Render("Top Foods by Points"), sep)

	if len(foods) == 0 {
		fmt.Fprintf(&b, "  %s\n", styleFoodPortion.Render("No food data available."))
	} else {
		nameW := 32
		for _, fs := range foods {
			name := truncate(fs.Name, nameW)
			countStr := styleFoodPortion.Render(fmt.Sprintf("%d×", fs.Count))
			totalPts := styleDetailValue.Render(fmt.Sprintf("%.0fpt total", fs.TotalPts))
			avgPts := styleFoodPortion.Render(fmt.Sprintf("%.0fpt avg", fs.AvgPts))
			avgCals := ""
			if fs.AvgCals > 0 {
				avgCals = styleFoodPortion.Render(fmt.Sprintf("  %.0f kcal avg", fs.AvgCals))
			}
			nameCol := lipgloss.NewStyle().Width(nameW + 2).Render(styleFoodItem.Render(name))
			countCol := lipgloss.NewStyle().Width(5).Render(countStr)
			totalCol := lipgloss.NewStyle().Width(14).Render(totalPts)
			avgCol := lipgloss.NewStyle().Width(12).Render(avgPts)
			fmt.Fprintf(&b, "  %s%s%s%s%s\n", nameCol, countCol, totalCol, avgCol, avgCals)
		}
	}

	// ── All Foods (zero-point) ───────────────────────────────────────
	zpFoods := zeroPointFoods(m.logs)
	if len(zpFoods) > 0 {
		fmt.Fprintf(&b, "\n%s\n%s\n\n", styleMealHeading.Render("Zero-Point Foods Logged"), sep)
		for _, fs := range zpFoods {
			name := truncate(fs.Name, 32)
			countStr := styleFoodPortion.Render(fmt.Sprintf("%d×", fs.Count))
			calsStr := styleFoodPortion.Render(fmt.Sprintf("  %.0f kcal avg", fs.AvgCals))
			nameCol := lipgloss.NewStyle().Width(34).Render(styleFoodItem.Render(name))
			fmt.Fprintf(&b, "  %s%s%s\n", nameCol, countStr, calsStr)
		}
	}

	return b.String()
}

func writeMacroBar(b *strings.Builder, label string, pct, grams float64, unit string, barWidth int) {
	bar := makeBar(pct, 100, barWidth)
	labelCol := lipgloss.NewStyle().Width(11).Render(styleDetailLabel.Render(label))
	pctCol := lipgloss.NewStyle().Width(6).Render(styleDetailValue.Render(fmt.Sprintf("%.0f%%", pct)))
	gramCol := styleFoodPortion.Render(fmt.Sprintf("%.0f%s avg", grams, unit))
	fmt.Fprintf(b, "  %s%s  %s  %s\n", labelCol, pctCol, bar, gramCol)
}

// renderHeatmap draws a GitHub-contributions-style calendar grid: months
// labelled along the top, sparse weekday labels (Mon/Wed/Fri) down the left,
// weeks as columns, days as rows, double-width cells for square-ish visuals,
// quantised 4-step gradient.
func renderHeatmap(logs []*api.DayLog, vw int) string {
	const layout = "2006-01-02"

	type entry struct {
		ratio     float64
		hasTarget bool
	}
	days := make(map[string]entry, len(logs))
	for _, log := range logs {
		if log.Points.DailyTarget > 0 {
			// Use TotalPointsConsumed (sum of food entries' PointsPrecise)
			// rather than Points.DailyUsed — the latter caps at DailyTarget,
			// hiding over-budget days that flowed into the weekly allowance.
			days[log.Date] = entry{
				ratio:     log.TotalPointsConsumed() / log.Points.DailyTarget,
				hasTarget: true,
			}
		} else {
			days[log.Date] = entry{hasTarget: false}
		}
	}

	// Fixed grid: span the WW my-day endpoint's ~89-day backwards window so
	// the heatmap is the same width regardless of the queried range.
	// Anchored to the Monday of the current week on the right; cells outside
	// the queried [first..last] window render as no-data grey.
	today := time.Now().UTC().Truncate(24 * time.Hour)
	todayStr := today.Format(layout)
	todayMon := today.AddDate(0, 0, -((int(today.Weekday())+6)%7))
	earliestMon := today.AddDate(0, 0, -89)
	earliestMon = earliestMon.AddDate(0, 0, -((int(earliestMon.Weekday())+6)%7))
	nWeeks := int(todayMon.Sub(earliestMon).Hours()/(24*7)) + 1

	var weeks []time.Time
	for i := 0; i < nWeeks; i++ {
		weeks = append(weeks, earliestMon.AddDate(0, 0, 7*i))
	}

	const dayLabelW = 4 // "Mon "
	cellW := 2
	gap := 1
	totalW := dayLabelW + nWeeks*cellW + (nWeeks-1)*gap
	if totalW > vw {
		gap = 0
		totalW = dayLabelW + nWeeks*cellW
	}
	if totalW > vw && cellW > 1 {
		cellW = 1
	}

	// 5 discrete shades from "nothing logged" to "fully on budget", plus
	// purple for over budget. Gives the GitHub-contributions stepped feel
	// instead of a continuous gradient.
	darkTeal := lipgloss.Color("#003d30")
	palette := [5]color.Color{
		lerpColor(darkTeal, colorTeal, 0.10),
		lerpColor(darkTeal, colorTeal, 0.35),
		lerpColor(darkTeal, colorTeal, 0.60),
		lerpColor(darkTeal, colorTeal, 0.85),
		colorTeal,
	}

	// Cells use the lower half block ▄ rather than the full block █ so the
	// top half of each terminal row stays empty — that gives a visible gap
	// between consecutive day-rows in the same week column, breaking up the
	// "vertical bar" appearance of solid stacked cells.
	noDataStyle := lipgloss.NewStyle().Foreground(colorLine)
	cellFor := func(dateStr string) string {
		blocks := strings.Repeat("▄", cellW)
		// Future days within the rightmost (current) week haven't happened
		// yet — render blank rather than no-data grey.
		if dateStr > todayStr {
			return strings.Repeat(" ", cellW)
		}
		e, inRange := days[dateStr]
		if !inRange || !e.hasTarget {
			return noDataStyle.Render(blocks)
		}
		var c color.Color
		switch {
		case e.ratio > 1.02:
			c = colorPurple
		case e.ratio >= 0.85:
			c = palette[4]
		case e.ratio >= 0.60:
			c = palette[3]
		case e.ratio >= 0.35:
			c = palette[2]
		case e.ratio >= 0.10:
			c = palette[1]
		default:
			c = palette[0]
		}
		return lipgloss.NewStyle().Foreground(c).Render(blocks)
	}

	var b strings.Builder

	// Month label row: place each month's 3-letter abbreviation at the
	// position of the first week starting in that month. Labels may
	// overflow the (cellW+gap) slot they nominally occupy — that's fine
	// because the slot to their right is blank until the next month.
	// +3 trailing chars so a label on the rightmost week column has room to
	// render its 3-letter month abbreviation without being truncated.
	hdrLen := nWeeks*cellW + (nWeeks-1)*gap + 3
	hdr := []rune(strings.Repeat(" ", hdrLen))
	var lastMonth time.Month
	for i, w := range weeks {
		if w.Month() != lastMonth {
			lastMonth = w.Month()
			label := w.Format("Jan")
			pos := i * (cellW + gap)
			for j, r := range label {
				if pos+j < len(hdr) {
					hdr[pos+j] = r
				}
			}
		}
	}
	fmt.Fprintf(&b, "%s%s\n",
		strings.Repeat(" ", dayLabelW),
		styleFoodPortion.Render(string(hdr)))

	// Day rows Mon→Sun, only Mon/Wed/Fri labelled (GitHub convention).
	// Inter-cell gap rendered as a middle dot in the no-data colour so the
	// boundary between adjacent cells stays visible even when neighbours
	// share a colour (e.g. two no-data cells in a row).
	gapStr := lipgloss.NewStyle().Foreground(colorLine).Render(strings.Repeat("·", gap))
	dayNames := [7]string{"Mon", "", "Wed", "", "Fri", "", ""}
	for row, name := range dayNames {
		label := fmt.Sprintf("%-*s", dayLabelW, name)
		fmt.Fprintf(&b, "%s", styleDetailLabel.Render(label))
		for i, weekMon := range weeks {
			if i > 0 && gap > 0 {
				fmt.Fprintf(&b, "%s", gapStr)
			}
			date := weekMon.AddDate(0, 0, row)
			fmt.Fprintf(&b, "%s", cellFor(date.Format(layout)))
		}
		fmt.Fprintln(&b)
	}

	// Legend: GitHub-style "Less □ ▓ ▒ ▓ ▓ More" + over-budget caveat.
	fmt.Fprintln(&b)
	swatch := func(c color.Color) string {
		return lipgloss.NewStyle().Foreground(c).Render(strings.Repeat("▄", cellW))
	}
	fmt.Fprintf(&b, "%sLess %s %s %s %s %s More  %s over budget  %s no data",
		strings.Repeat(" ", dayLabelW),
		swatch(palette[0]), swatch(palette[1]), swatch(palette[2]),
		swatch(palette[3]), swatch(palette[4]),
		swatch(colorPurple),
		swatch(colorLine))

	return b.String()
}

// zeroPointFoods returns foods with 0 total points, sorted by frequency.
func zeroPointFoods(logs []*api.DayLog) []api.FoodStat {
	all := api.TopFoods(logs, 0)
	var zp []api.FoodStat
	for _, fs := range all {
		if fs.TotalPts == 0 {
			zp = append(zp, fs)
		}
	}
	// sort by count descending
	for i := 1; i < len(zp); i++ {
		for j := i; j > 0 && zp[j].Count > zp[j-1].Count; j-- {
			zp[j], zp[j-1] = zp[j-1], zp[j]
		}
	}
	if len(zp) > 15 {
		zp = zp[:15]
	}
	return zp
}
