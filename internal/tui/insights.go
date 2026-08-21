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
	// twoCol tracks the last layout tier render() used, so resize can detect
	// a crossing and reset scroll — see resize.
	twoCol bool
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
	_, m.twoCol = insightsColumns(insightsViewWidth(m.width))
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
		case key.Matches(msg, keys.GotoTop):
			m.viewport.GotoTop()
			return m, nil
		case key.Matches(msg, keys.GotoBottom):
			m.viewport.GotoBottom()
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
	prevTwoCol := m.twoCol
	m.width = width
	m.height = height
	m.viewport.SetWidth(width - 2)
	m.viewport.SetHeight(height)
	_, m.twoCol = insightsColumns(insightsViewWidth(m.width))
	m.viewport.SetContent(m.render())
	// Crossing the two-column breakpoint roughly halves (or doubles) the
	// document's height. Without this, a user scrolled partway down would
	// get silently snapped to the bottom by the viewport's own offset clamp.
	if m.twoCol != prevTwoCol {
		m.viewport.GotoTop()
	}
}

// insightsViewWidth converts the tab's total width to the content width its
// sections render at — matches the -4 the viewport (-2) and its Padding(0,1)
// (-2) already cost, floored so a very narrow terminal can't drive section
// widths negative.
func insightsViewWidth(width int) int {
	vw := width - 4
	if vw < 40 {
		vw = 40
	}
	return vw
}

func (m *insightsModel) render() string {
	if len(m.logs) == 0 {
		return styleFoodPortion.Render("No data available.")
	}

	vw := insightsViewWidth(m.width)
	colW, twoCol := insightsColumns(vw)

	summary := api.ComputeRangeSummary(m.logs)
	meals := api.MealStats(m.logs)
	macros := api.AvgMacroBreakdown(m.logs)
	micro, hasMicro := computeMicroAverages(m.logs)
	zpFoods := zeroPointFoods(m.logs)

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

	// The heatmap doesn't stretch to fill vw (see renderHeatmap) — measure
	// its actual rendered width to decide whether Range Summary fits beside
	// it on the same row, rather than assuming a fixed natural width.
	rawHeatmap := renderHeatmap(m.logs, vw)
	hmW := blockWidth(rawHeatmap)
	wideRow := twoCol && vw-hmW-insightsGutter >= insightsMinColWidth

	headingWidth := vw
	if wideRow {
		headingWidth = hmW
	}
	heatmap := sectionHeading("Points Budget  (day by day)", "Points Budget", headingWidth) +
		"\n\n" + rawHeatmap

	var rows []string
	switch {
	case wideRow:
		rightW := vw - hmW - insightsGutter
		rows = append(rows,
			joinColumns(heatmap, hmW, rangeSummarySection(summary, rightW), rightW),
			joinColumns(mealsSection(meals, colW), colW, macrosSection(macros, colW), colW),
			joinColumns(microsSection(micro, hasMicro, colW), colW, topFoodsSection(foods, colW), colW),
			zeroPointSection(zpFoods, vw),
		)
	case twoCol:
		rows = append(rows,
			heatmap,
			joinColumns(rangeSummarySection(summary, colW), colW, mealsSection(meals, colW), colW),
			joinColumns(macrosSection(macros, colW), colW, microsSection(micro, hasMicro, colW), colW),
			joinColumns(topFoodsSection(foods, colW), colW, zeroPointSection(zpFoods, colW), colW),
		)
	default:
		rows = append(rows,
			heatmap,
			rangeSummarySection(summary, vw),
			mealsSection(meals, vw),
			macrosSection(macros, vw),
			microsSection(micro, hasMicro, vw),
			topFoodsSection(foods, vw),
			zeroPointSection(zpFoods, vw),
		)
	}

	var out []string
	for _, r := range rows {
		if strings.TrimSpace(r) != "" {
			out = append(out, r)
		}
	}
	return strings.Join(out, "\n\n")
}

// rangeSummarySection renders the days/items line plus the Points, Calories,
// and Activity bars/lines, sized to width w.
func rangeSummarySection(s api.RangeSummary, w int) string {
	pointsRow := func(bw int) string {
		return fmt.Sprintf("  %s  %s  %s",
			lipgloss.NewStyle().Width(10).Render(styleDetailLabel.Render("Points")),
			makeBar(s.AvgDailyPts, s.AvgDailyTarget, bw, false),
			styleDetailValue.Render(fmt.Sprintf("avg %.0fpt / %.0fpt target", s.AvgDailyPts, s.AvgDailyTarget)),
		)
	}
	calsRow := func(bw int) string {
		return fmt.Sprintf("  %s  %s  %s",
			lipgloss.NewStyle().Width(10).Render(styleDetailLabel.Render("Calories")),
			makeBar(s.AvgDailyCals, 2000, bw, false),
			styleDetailValue.Render(fmt.Sprintf("avg %.0f kcal / day", s.AvgDailyCals)),
		)
	}
	var rows []func(int) string
	if s.AvgDailyTarget > 0 {
		rows = append(rows, pointsRow)
	}
	if s.AvgDailyCals > 0 {
		rows = append(rows, calsRow)
	}
	barWidth := barWidthForRows(w, rows...)

	var b strings.Builder
	b.WriteString(sectionHeading("Range Summary", "Range Summary", w))
	b.WriteString("\n\n")

	daysStr := fmt.Sprintf("%d days  ·  %d food items logged", s.Days, s.TotalItems)
	fmt.Fprintf(&b, "  %s\n\n", styleDetailValue.Render(daysStr))

	if s.AvgDailyTarget > 0 {
		fmt.Fprintf(&b, "%s\n", pointsRow(barWidth))
		budgetStr := fmt.Sprintf("%d days on/under budget  ·  %d days over", s.DaysUnderBudget, s.DaysOverBudget)
		fmt.Fprintf(&b, "  %s\n", styleFoodPortion.Render(budgetStr))
	}
	if s.AvgDailyCals > 0 {
		fmt.Fprintf(&b, "%s\n", calsRow(barWidth))
		fmt.Fprintf(&b, "  %s\n", styleFoodPortion.Render("bar shows % of 2000 kcal reference"))
	}
	activityStr := fmt.Sprintf("%.0fpt earned across %d days", s.TotalActivityEarned, s.DaysWithActivity)
	fmt.Fprintf(&b, "  %s  %s\n",
		lipgloss.NewStyle().Width(10).Render(styleDetailLabel.Render("Activity")),
		styleDetailValue.Render(activityStr),
	)
	return strings.TrimRight(b.String(), "\n")
}

// mealsSection renders one bar row per meal period, sized to width w.
func mealsSection(meals []api.MealStat, w int) string {
	var maxMealPts float64
	for _, ms := range meals {
		if ms.AvgPts > maxMealPts {
			maxMealPts = ms.AvgPts
		}
	}
	if maxMealPts == 0 {
		maxMealPts = 1
	}

	mealRow := func(ms api.MealStat) func(int) string {
		return func(bw int) string {
			bar := makeBar(ms.AvgPts, maxMealPts, bw, false)
			label := lipgloss.NewStyle().Width(12).Render(styleDetailLabel.Render(ms.Symbol + "  " + ms.Name))
			val := styleDetailValue.Render(fmt.Sprintf("%.1fpt", ms.AvgPts))
			cal := styleFoodPortion.Render(fmt.Sprintf("  %.0f kcal", ms.AvgCals))
			return fmt.Sprintf("  %s  %s  %s%s", label, bar, val, cal)
		}
	}
	rows := make([]func(int) string, len(meals))
	for i, ms := range meals {
		rows[i] = mealRow(ms)
	}
	barWidth := barWidthForRows(w, rows...)

	var b strings.Builder
	b.WriteString(sectionHeading("Points by Meal  (average per day)", "Points by Meal", w))
	b.WriteString("\n\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "%s\n", row(barWidth))
	}
	return strings.TrimRight(b.String(), "\n")
}

// macrosSection renders the Protein/Carbs/Fat/Alcohol bars, sized to width w.
func macrosSection(macros api.MacroBreakdown, w int) string {
	var b strings.Builder
	b.WriteString(sectionHeading("Macro Distribution  (average daily, % of calories)", "Macro Distribution", w))
	b.WriteString("\n\n")

	if macros.ProteinG+macros.CarbsG+macros.FatG == 0 {
		fmt.Fprintf(&b, "  %s\n", styleFoodPortion.Render("No nutrition data available."))
		return strings.TrimRight(b.String(), "\n")
	}

	rows := []func(int) string{
		macroBarRow("Protein", macros.ProteinPct, macros.ProteinG, "g"),
		macroBarRow("Carbs", macros.CarbsPct, macros.CarbsG, "g"),
		macroBarRow("Fat", macros.FatPct, macros.FatG, "g"),
	}
	if macros.AlcoholG > 0 {
		rows = append(rows, macroBarRow("Alcohol", macros.AlcoholPct, macros.AlcoholG, "g"))
	}
	barWidth := barWidthForRows(w, rows...)
	for _, row := range rows {
		fmt.Fprintf(&b, "%s\n", row(barWidth))
	}

	footnote := "Recommended: ~20% protein  ·  ~50% carbs  ·  ~30% fat"
	if lipgloss.Width(footnote) > w {
		footnote = "Rec: 20% protein · 50% carbs · 30% fat"
	}
	fmt.Fprintf(&b, "\n  %s\n", styleFoodPortion.Render(footnote))
	return strings.TrimRight(b.String(), "\n")
}

// macroBarRow returns a row builder for one macronutrient's bar — deferred
// like this so barWidthForRows can measure the row's non-bar width (which
// varies with the gram/percentage values) before a shared bar width for the
// whole section is settled on.
func macroBarRow(label string, pct, grams float64, unit string) func(int) string {
	return func(bw int) string {
		bar := makeBar(pct, 100, bw, false)
		labelCol := lipgloss.NewStyle().Width(11).Render(styleDetailLabel.Render(label))
		pctCol := lipgloss.NewStyle().Width(6).Render(styleDetailValue.Render(fmt.Sprintf("%.0f%%", pct)))
		gramCol := styleFoodPortion.Render(fmt.Sprintf("%.0f%s avg", grams, unit))
		return fmt.Sprintf("  %s%s  %s  %s", labelCol, pctCol, bar, gramCol)
	}
}

// microAverages holds the range-wide daily averages microsSection bars against
// their reference values.
type microAverages struct {
	fiber, sodium, addedSugar, saturatedFat float64
}

// computeMicroAverages averages fiber/sodium/added-sugar/saturated-fat across
// days that have any logged nutrition data. The bool return is false when no
// day in the range has nutrition data, so microsSection can omit itself
// entirely rather than showing an all-zero section.
func computeMicroAverages(logs []*api.DayLog) (microAverages, bool) {
	nutrition := api.ComputeAllNutrition(logs)
	var fiberSum, sodiumSum, addedSugarSum, saturatedFatSum float64
	daysWithData := 0
	for _, dn := range nutrition {
		if dn.ItemCount > 0 {
			fiberSum += dn.Fiber
			sodiumSum += dn.Sodium
			addedSugarSum += dn.AddedSugar
			saturatedFatSum += dn.SaturatedFat
			daysWithData++
		}
	}
	if daysWithData == 0 {
		return microAverages{}, false
	}
	n := float64(daysWithData)
	return microAverages{
		fiber:        fiberSum / n,
		sodium:       sodiumSum / n,
		addedSugar:   addedSugarSum / n,
		saturatedFat: saturatedFatSum / n,
	}, true
}

// microsSection renders the fiber/sodium/added-sugar/saturated-fat bars,
// sized to width w. Returns "" when has is false, so callers can drop it
// from the layout the same way an empty food table is dropped.
func microsSection(avg microAverages, has bool, w int) string {
	if !has {
		return ""
	}
	rows := []func(int) string{
		microBarRow("Fiber", "g", avg.fiber, rdv.Fiber),
		microBarRow("Sodium", "mg", avg.sodium, rdv.Sodium),
		microBarRow("Added Sugar", "g", avg.addedSugar, rdv.AddedSugar),
		microBarRow("Sat Fat", "g", avg.saturatedFat, rdv.SaturatedFat),
	}
	barWidth := barWidthForRows(w, rows...)

	var b strings.Builder
	heading := "Daily Averages  (fiber · sodium · added sugar · saturated fat)"
	b.WriteString(sectionHeading(heading, "Daily Averages", w))
	b.WriteString("\n\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "%s\n", row(barWidth))
	}
	return strings.TrimRight(b.String(), "\n")
}

// microBarRow returns a row builder for one micronutrient's bar — see
// macroBarRow for why this is deferred rather than returning a string.
func microBarRow(label, unit string, avg, ref float64) func(int) string {
	return func(bw int) string {
		bar := makeBar(avg, ref, bw, false)
		labelCol := lipgloss.NewStyle().Width(13).Render(styleDetailLabel.Render(label))
		valCol := lipgloss.NewStyle().Width(12).Render(styleDetailValue.Render(fmt.Sprintf("%s %s", formatNutriValue(avg), unit)))
		refStr := styleFoodPortion.Render(fmt.Sprintf("ref %s", formatNutriValue(ref)))
		return fmt.Sprintf("  %s%s%s  %s", labelCol, valCol, bar, refStr)
	}
}

// topFoodsSection renders the Top Foods by Points leaderboard, sized to
// width w — see topFoodsLayout for how its columns shed as w shrinks.
func topFoodsSection(foods []api.FoodStat, w int) string {
	var b strings.Builder
	b.WriteString(sectionHeading("Top Foods by Points", "Top Foods", w))
	b.WriteString("\n\n")

	if len(foods) == 0 {
		fmt.Fprintf(&b, "  %s\n", styleFoodPortion.Render("No food data available."))
		return strings.TrimRight(b.String(), "\n")
	}

	layout := topFoodsLayout(w)
	for _, fs := range foods {
		name := truncate(fs.Name, layout.nameW-2)
		nameCol := lipgloss.NewStyle().Width(layout.nameW).Render(styleFoodItem.Render(name))
		countCol := lipgloss.NewStyle().Width(foodCountW).Render(styleFoodPortion.Render(fmt.Sprintf("%d×", fs.Count)))
		totalCol := lipgloss.NewStyle().Width(foodTotalPtsW).Render(styleDetailValue.Render(fmt.Sprintf("%.0fpt total", fs.TotalPts)))
		fmt.Fprintf(&b, "  %s%s%s", nameCol, countCol, totalCol)
		if layout.showAvgPts {
			avgCol := lipgloss.NewStyle().Width(foodAvgPtsW).Render(styleFoodPortion.Render(fmt.Sprintf("%.0fpt avg", fs.AvgPts)))
			fmt.Fprint(&b, avgCol)
		}
		if layout.showKcal && fs.AvgCals > 0 {
			fmt.Fprint(&b, styleFoodPortion.Render(fmt.Sprintf("  %.0f kcal avg", fs.AvgCals)))
		}
		fmt.Fprintln(&b)
	}
	return strings.TrimRight(b.String(), "\n")
}

// zeroPointSection renders the Zero-Point Foods table, sized to width w.
// Returns "" when there's nothing to show, matching topFoodsSection's
// sibling behaviour of dropping empty sections from the layout entirely.
func zeroPointSection(foods []api.FoodStat, w int) string {
	if len(foods) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(sectionHeading("Zero-Point Foods Logged", "Zero-Point Foods", w))
	b.WriteString("\n\n")

	layout := zeroPointLayout(w)
	for _, fs := range foods {
		name := truncate(fs.Name, layout.nameW-2)
		nameCol := lipgloss.NewStyle().Width(layout.nameW).Render(styleFoodItem.Render(name))
		countCol := lipgloss.NewStyle().Width(foodCountW).Render(styleFoodPortion.Render(fmt.Sprintf("%d×", fs.Count)))
		fmt.Fprintf(&b, "  %s%s", nameCol, countCol)
		if layout.showKcal {
			fmt.Fprint(&b, styleFoodPortion.Render(fmt.Sprintf("  %.0f kcal avg", fs.AvgCals)))
		}
		fmt.Fprintln(&b)
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderHeatmap draws a GitHub-contributions-style calendar grid: months
// labelled along the top, sparse weekday labels (Mon/Wed/Fri) down the left,
// weeks as columns, days as rows, double-width cells for square-ish visuals,
// quantised 4-step gradient.
func renderHeatmap(logs []*api.DayLog, vw int) string {
	const layout = "2006-01-02"

	type entry struct {
		ratio           float64
		hasTarget       bool
		weeklyExhausted bool
	}
	days := make(map[string]entry, len(logs))
	for _, log := range logs {
		if log.Points.DailyTarget > 0 {
			consumed := log.TotalPointsConsumed()
			ratio := consumed / log.Points.DailyTarget
			// Purple only when both daily target and weekly bank are exhausted.
			// Dipping into the weekly allowance is on-budget by WW rules.
			weeklyExhausted := consumed > log.Points.DailyTarget && log.Points.WeeklyAllowanceRemaining < 0
			if !weeklyExhausted && ratio > 1.0 {
				ratio = 1.0
			}
			days[log.Date] = entry{
				ratio:           ratio,
				hasTarget:       true,
				weeklyExhausted: weeklyExhausted,
			}
		} else {
			days[log.Date] = entry{hasTarget: false}
		}
	}

	// Fixed grid: span the WW my-day endpoint's retention window so the
	// heatmap is the same width regardless of the queried range. Anchored
	// to the Monday of the current week on the right; cells outside the
	// queried [first..last] window render as no-data grey. Uses the local
	// calendar date — matches api.ClampToWindow's boundary logic.
	today, _ := time.Parse(layout, time.Now().Format(layout))
	todayStr := today.Format(layout)
	todayMon := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	earliestMon := today.AddDate(0, 0, -api.APIWindowDays)
	earliestMon = earliestMon.AddDate(0, 0, -((int(earliestMon.Weekday()) + 6) % 7))
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
		case e.weeklyExhausted:
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
	// Plain-space gap between weeks — the half-height ▄ cells already
	// provide a visible vertical gap (empty top half of each row), so a
	// styled · separator on top read as visual noise.
	gapStr := strings.Repeat(" ", gap)
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
