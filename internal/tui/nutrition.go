package tui

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	"github.com/ali5ter/wwlog/config"
	"github.com/ali5ter/wwlog/internal/api"
)

// rdv holds reference daily values used as bar maximums when no user targets are configured.
var rdv = &api.DayNutrition{
	Calories:     2000,
	Fat:          65,
	SaturatedFat: 20,
	Sodium:       2300,
	Carbs:        300,
	Fiber:        28,
	Sugar:        50,
	AddedSugar:   35,
	Protein:      50,
	Alcohol:      28,
}

// nutriRef holds bar rendering parameters for a single nutrient.
type nutriRef struct {
	value float64 // 100% mark on the bar
	clamp bool    // true: bar caps at 100% (floor target — more is fine); false: bar turns purple if over (ceiling)
}

// resolveRef returns the bar reference for a named nutrient, using user targets when configured
// and falling back to generic RDV values otherwise.
func resolveRef(targets *config.Targets, metric string) nutriRef {
	if targets != nil {
		switch metric {
		case "calories":
			if len(targets.Calories) >= 2 {
				return nutriRef{targets.Calories[1], false}
			}
		case "protein":
			if len(targets.ProteinG) >= 2 {
				return nutriRef{targets.ProteinG[1], false}
			}
		case "carbs":
			if len(targets.CarbsG) >= 2 {
				return nutriRef{targets.CarbsG[1], false}
			}
		case "fat":
			if len(targets.FatG) >= 2 {
				return nutriRef{targets.FatG[1], false}
			}
		case "fiber":
			if targets.FiberGMin > 0 {
				return nutriRef{targets.FiberGMin, true}
			}
		case "sodium":
			if targets.SodiumMgMax > 0 {
				return nutriRef{targets.SodiumMgMax, false}
			}
		case "added_sugar":
			if targets.AddedSugarGMax > 0 {
				return nutriRef{targets.AddedSugarGMax, false}
			}
		}
	}
	switch metric {
	case "calories":
		return nutriRef{rdv.Calories, false}
	case "protein":
		return nutriRef{rdv.Protein, false}
	case "carbs":
		return nutriRef{rdv.Carbs, false}
	case "fat":
		return nutriRef{rdv.Fat, false}
	case "sat_fat":
		return nutriRef{rdv.SaturatedFat, false}
	case "fiber":
		return nutriRef{rdv.Fiber, false}
	case "sodium":
		return nutriRef{rdv.Sodium, false}
	case "sugar":
		return nutriRef{rdv.Sugar, false}
	case "added_sugar":
		return nutriRef{rdv.AddedSugar, false}
	case "alcohol":
		return nutriRef{rdv.Alcohol, false}
	}
	return nutriRef{1, false}
}

type nutriModel struct {
	list      list.Model
	filter    textinput.Model
	filtering bool
	detail    viewport.Model
	// detailFocused is true when the detail pane, rather than the date
	// list, has keyboard focus — see keys.FocusPrev/FocusNext. While true,
	// Up/Down scroll the detail pane instead of navigating dates.
	detailFocused bool
	allLogs       []*api.DayLog
	logs          []*api.DayLog // filtered view
	allData       map[string]*api.DayNutrition
	data          map[string]*api.DayNutrition // filtered view
	avgs          *api.DayNutrition
	width         int
	height        int
	selected      int
	locale        locale
	targets       *config.Targets
	initialized   bool
}

func newNutriModel(logs []*api.DayLog, data map[string]*api.DayNutrition, width, height int, loc locale, targets *config.Targets) nutriModel {
	listOuter := listPaneWidth(width)
	detailOuter := detailPaneWidth(width)
	listWidth := paneContentWidth(listOuter)
	listHeight := paneContentHeight(height) - paneFilterRows
	detailWidth := paneContentWidth(detailOuter)
	detailHeight := paneContentHeight(height)

	items := make([]list.Item, len(logs))
	for i, l := range logs {
		items[i] = dateItem{log: l, locale: loc}
	}

	l := newDateList(items, listWidth, listHeight)

	fi := textinput.New()
	fi.Placeholder = "filter by date (e.g. Jan, 04)"
	fiStyles := fi.Styles()
	fiStyles.Focused.Prompt = styleFilterPrompt
	fiStyles.Focused.Text = styleFilterText
	fi.SetStyles(fiStyles)
	fi.Prompt = "> "

	vp := viewport.New(viewport.WithWidth(detailWidth), viewport.WithHeight(detailHeight))
	vp.MouseWheelEnabled = true

	m := nutriModel{
		list:        l,
		filter:      fi,
		detail:      vp,
		allLogs:     logs,
		logs:        logs,
		allData:     data,
		data:        data,
		avgs:        computeAverages(data, logs),
		width:       width,
		height:      height,
		locale:      loc,
		targets:     targets,
		initialized: true,
	}
	m.detail.SetContent(m.renderDetail())
	return m
}

func (m nutriModel) update(msg tea.Msg) (nutriModel, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if msg, ok := msg.(tea.KeyPressMsg); ok {
		if m.filtering {
			switch msg.String() {
			case "enter", "esc":
				m.filtering = false
				m.filter.Blur()
				m.applyFilter()
				return m, nil
			case "up", "k", "down", "j":
				m.filtering = false
				m.filter.Blur()
				m.applyFilter()
				// fall through to list navigation below
			default:
				m.filter, cmd = m.filter.Update(msg)
				cmds = append(cmds, cmd)
				m.applyFilter()
				return m, tea.Batch(cmds...)
			}
		} else {
			switch {
			case key.Matches(msg, keys.Filter):
				m.filtering = true
				m.filter.Focus()
				// "/" belongs to the list — see logModel's identical guard.
				m.detailFocused = false
				return m, textinput.Blink
			case key.Matches(msg, keys.FocusPrev):
				m.detailFocused = false
				return m, nil
			case key.Matches(msg, keys.FocusNext):
				m.detailFocused = true
				return m, nil
			case m.detailFocused && key.Matches(msg, keys.Up):
				m.detail.ScrollUp(detailScrollStep)
				return m, nil
			case m.detailFocused && key.Matches(msg, keys.Down):
				m.detail.ScrollDown(detailScrollStep)
				return m, nil
			case key.Matches(msg, keys.ScrollUp):
				m.detail.ScrollUp(detailScrollStep)
				return m, nil
			case key.Matches(msg, keys.ScrollDown):
				m.detail.ScrollDown(detailScrollStep)
				return m, nil
			// See logModel's identical guard: these keys belong to the date
			// list's own paging unless the detail pane has focus.
			case m.detailFocused && key.Matches(msg, keys.GotoTop):
				m.detail.GotoTop()
				return m, nil
			case m.detailFocused && key.Matches(msg, keys.GotoBottom):
				m.detail.GotoBottom()
				return m, nil
			case m.detailFocused && key.Matches(msg, keys.PageUp):
				m.detail.PageUp()
				return m, nil
			case m.detailFocused && key.Matches(msg, keys.PageDown):
				m.detail.PageDown()
				return m, nil
			case m.detailFocused && key.Matches(msg, keys.HalfPageUp):
				m.detail.HalfPageUp()
				return m, nil
			case m.detailFocused && key.Matches(msg, keys.HalfPageDown):
				m.detail.HalfPageDown()
				return m, nil
			}
		}
	}

	// Mouse wheel scrolls the detail viewport, not the date list, regardless
	// of pane focus — positional, like shift+up/down.
	if _, ok := msg.(tea.MouseWheelMsg); ok {
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	}

	// Click on a date row in the list pane selects that row and refocuses
	// the list; a click landing in the detail pane instead focuses it. See
	// logModel's identical handling for the click.Y >= 2 rationale.
	if click, ok := msg.(tea.MouseClickMsg); ok && click.Y >= 2 {
		if idx, ok := m.dateRowAtPoint(click.X, click.Y); ok {
			m.detailFocused = false
			m.list.Select(idx)
		} else if click.X >= listPaneWidth(m.width) {
			m.detailFocused = true
		}
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	selChanged := false
	if i := m.list.Index(); i != m.selected && i < len(m.logs) {
		m.selected = i
		m.detail.SetContent(m.renderDetail())
		m.detail.GotoTop()
		selChanged = true
	}

	if !selChanged {
		// Only forward non-key messages to the viewport — same reasoning as logModel.
		if _, isKey := msg.(tea.KeyPressMsg); !isKey {
			m.detail, cmd = m.detail.Update(msg)
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m nutriModel) view() string {
	listOuter := listPaneWidth(m.width)
	detailOuter := detailPaneWidth(m.width)

	var filterBar string
	if m.filtering {
		filterBar = m.filter.View()
	} else if m.filter.Value() != "" {
		filterBar = styleFilterPrompt.Render("> ") + styleFilterText.Render(m.filter.Value())
	} else {
		filterBar = styleDim.Render("> filter by date…")
	}
	filterSep := styleDim.Render(strings.Repeat("─", max(0, paneContentWidth(listOuter))))

	listPane := paneBox(
		lipgloss.JoinVertical(lipgloss.Left, filterBar, filterSep, m.list.View()),
		listOuter, m.height, !m.detailFocused,
	)
	detailPane := paneBox(m.detail.View(), detailOuter, m.height, m.detailFocused)
	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, detailPane)
}

func (m *nutriModel) applyFilter() {
	q := strings.ToLower(m.filter.Value())
	var filtered []*api.DayLog
	for _, l := range m.allLogs {
		if q == "" ||
			strings.Contains(strings.ToLower(l.Date), q) ||
			strings.Contains(strings.ToLower(m.locale.dateShort(l.Date)), q) {
			filtered = append(filtered, l)
		}
	}
	m.logs = filtered
	filteredData := make(map[string]*api.DayNutrition, len(filtered))
	for _, l := range filtered {
		if dn, ok := m.allData[l.Date]; ok {
			filteredData[l.Date] = dn
		}
	}
	m.data = filteredData
	m.avgs = computeAverages(filteredData, filtered)
	items := make([]list.Item, len(filtered))
	for i, l := range filtered {
		items[i] = dateItem{log: l, locale: m.locale}
	}
	m.list.SetItems(items)
	m.selected = 0
	if len(filtered) > 0 {
		m.detail.SetContent(m.renderDetail())
	} else {
		m.detail.SetContent(styleFoodPortion.Render("No matching dates."))
	}
	m.detail.GotoTop()
}

// dateRowAtPoint mirrors logModel.dateRowAtPoint — delegates to the shared
// implementation in panes.go.
func (m nutriModel) dateRowAtPoint(x, y int) (int, bool) {
	return dateRowAtPoint(x, y, m.width, m.list.Paginator.Page, m.list.Paginator.PerPage, len(m.list.Items()))
}

func (m *nutriModel) resize(width, height int) {
	if !m.initialized {
		return
	}
	m.width = width
	m.height = height
	listOuter := listPaneWidth(width)
	detailOuter := detailPaneWidth(width)
	m.list.SetSize(paneContentWidth(listOuter), paneContentHeight(height)-paneFilterRows)
	m.detail.SetWidth(paneContentWidth(detailOuter))
	m.detail.SetHeight(paneContentHeight(height))
	m.detail.SetContent(m.renderDetail())
}

func (m *nutriModel) renderDetail() string {
	if m.data == nil || len(m.data) == 0 {
		return styleFoodPortion.Render("No nutrition data available.")
	}
	if m.selected >= len(m.logs) || len(m.logs) == 0 {
		return ""
	}

	day := m.logs[m.selected]
	dn := m.data[day.Date]
	if dn == nil {
		return styleFoodPortion.Render("No data for this date.")
	}

	vw := m.detail.Width()
	if vw < 40 {
		vw = 40
	}
	sepWidth := vw - 2
	barWidth := 24
	if barWidth > vw-36 {
		barWidth = vw - 36
	}
	if barWidth < 8 {
		barWidth = 8
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", styleMealHeading.Render(m.locale.dateLong(day.Date)))
	fmt.Fprintf(&b, "%s\n\n", styleDim.Render(strings.Repeat("─", sepWidth)))

	// Points summary.
	renderPointsSummary(&b, day, vw, m.locale)

	fmt.Fprintf(&b, "%s\n\n", styleDim.Render(strings.Repeat("─", sepWidth)))

	// Nutrition bars vs configured targets (or generic RDV when no targets set).
	refLabel := "bars show % of daily reference"
	if m.targets != nil && m.targets.HasAny() {
		refLabel = "bars show % of configured target"
	}
	fmt.Fprintf(&b, "%s\n\n", styleDetailLabel.Render("Nutrition  ("+refLabel+")"))
	writeNutriBar(&b, "Calories", "kcal", dn.Calories, resolveRef(m.targets, "calories"), m.avgs.Calories, barWidth)
	writeNutriBar(&b, "Protein", "g", dn.Protein, resolveRef(m.targets, "protein"), m.avgs.Protein, barWidth)
	writeNutriBar(&b, "Carbs", "g", dn.Carbs, resolveRef(m.targets, "carbs"), m.avgs.Carbs, barWidth)
	writeNutriBar(&b, "Fat", "g", dn.Fat, resolveRef(m.targets, "fat"), m.avgs.Fat, barWidth)
	writeNutriBar(&b, "Sat Fat", "g", dn.SaturatedFat, resolveRef(m.targets, "sat_fat"), m.avgs.SaturatedFat, barWidth)
	writeNutriBar(&b, "Fiber", "g", dn.Fiber, resolveRef(m.targets, "fiber"), m.avgs.Fiber, barWidth)
	writeNutriBar(&b, "Sodium", "mg", dn.Sodium, resolveRef(m.targets, "sodium"), m.avgs.Sodium, barWidth)
	writeNutriBar(&b, "Sugar", "g", dn.Sugar, resolveRef(m.targets, "sugar"), m.avgs.Sugar, barWidth)
	if dn.AddedSugar > 0 || m.avgs.AddedSugar > 0 {
		writeNutriBar(&b, "Added Sugar", "g", dn.AddedSugar, resolveRef(m.targets, "added_sugar"), m.avgs.AddedSugar, barWidth)
	}
	if dn.Alcohol > 0 || m.avgs.Alcohol > 0 {
		writeNutriBar(&b, "Alcohol", "g", dn.Alcohol, resolveRef(m.targets, "alcohol"), m.avgs.Alcohol, barWidth)
	}

	if len(m.logs) > 1 {
		fmt.Fprintf(&b, "\n%s\n\n", styleDim.Render(strings.Repeat("─", sepWidth)))
		fmt.Fprintf(&b, "%s\n\n", styleDetailLabel.Render("Trends across date range"))
		writeTrendTable(&b, m.logs, m.data, vw, m.targets)
	}

	return b.String()
}

func (m *nutriModel) nutriSeries(fn func(*api.DayNutrition) float64) []float64 {
	vals := make([]float64, 0, len(m.logs))
	for _, day := range m.logs {
		if dn, ok := m.data[day.Date]; ok {
			vals = append(vals, fn(dn))
		}
	}
	return vals
}

func writeNutriBar(b *strings.Builder, label, unit string, value float64, ref nutriRef, avg float64, barWidth int) {
	bar := makeBar(value, ref.value, barWidth, ref.clamp)
	labelCol := lipgloss.NewStyle().Width(11).Render(styleDetailLabel.Render(label))
	valCol := lipgloss.NewStyle().Width(14).Render(styleDetailValue.Render(fmt.Sprintf("%s %s", formatNutriValue(value), unit)))
	avgCol := styleFoodPortion.Render(fmt.Sprintf("avg %s", formatNutriValue(avg)))
	fmt.Fprintf(b, "  %s%s%s  %s\n", labelCol, valCol, bar, avgCol)
}

func writeTrendTable(b *strings.Builder, logs []*api.DayLog, data map[string]*api.DayNutrition, vw int, targets *config.Targets) {
	type series struct {
		label string
		unit  string
		get   func(*api.DayNutrition) float64
		ref   float64 // reference daily value drawn as a threshold line
	}
	metrics := []series{
		{"Calories", "kcal", func(d *api.DayNutrition) float64 { return d.Calories }, resolveRef(targets, "calories").value},
		{"Protein", "g", func(d *api.DayNutrition) float64 { return d.Protein }, resolveRef(targets, "protein").value},
		{"Carbs", "g", func(d *api.DayNutrition) float64 { return d.Carbs }, resolveRef(targets, "carbs").value},
		{"Fat", "g", func(d *api.DayNutrition) float64 { return d.Fat }, resolveRef(targets, "fat").value},
	}

	n := len(logs)
	if n < 2 {
		return
	}

	plotW := vw - 2
	if plotW < 24 {
		plotW = 24
	}
	plotH := 8

	axisStyle := lipgloss.NewStyle().Foreground(colorSteel)
	labelStyle := lipgloss.NewStyle().Foreground(colorMuted)
	lineStyle := lipgloss.NewStyle().Foreground(colorTeal)

	xStep := 1
	if n > 7 {
		xStep = (n + 6) / 7
	}

	dateLabel := func(_ int, v float64) string {
		i := int(math.Round(v))
		if i < 0 || i >= len(logs) {
			return ""
		}
		d := logs[i].Date
		if len(d) >= 10 {
			return d[5:10] // "MM-DD"
		}
		return d
	}

	for _, m := range metrics {
		vals := make([]float64, n)
		for i, day := range logs {
			if dn, ok := data[day.Date]; ok {
				vals[i] = m.get(dn)
			}
		}
		vals = clampOutliers(vals)

		// Ensure Y range always includes the reference threshold so it is visible.
		maxY := m.ref
		for _, v := range vals {
			if v > maxY {
				maxY = v
			}
		}
		if maxY == 0 {
			maxY = 1
		}
		maxY *= 1.1 // headroom

		lc := linechart.New(
			plotW, plotH,
			0, float64(n-1), 0, maxY,
			linechart.WithXYSteps(xStep, 2),
			linechart.WithStyles(axisStyle, labelStyle, lineStyle),
			linechart.WithXLabelFormatter(dateLabel),
		)
		lc.DrawXYAxisAndLabel()
		// Draw reference threshold line before data so data renders on top.
		// Uses DrawRuneLineWithStyle with a dashed rune so it reads as a
		// background guide rather than a data series.
		if m.ref > 0 {
			refStyle := lipgloss.NewStyle().Foreground(colorSteel)
			lc.DrawRuneLineWithStyle(
				canvas.Float64Point{X: 0, Y: m.ref},
				canvas.Float64Point{X: float64(n - 1), Y: m.ref},
				'╌',
				refStyle,
			)
		}
		for i := 1; i < n; i++ {
			lc.DrawLineWithStyle(
				canvas.Float64Point{X: float64(i - 1), Y: vals[i-1]},
				canvas.Float64Point{X: float64(i), Y: vals[i]},
				runes.ArcLineStyle,
				lineStyle,
			)
		}

		caption := fmt.Sprintf("%s (%s)  ref: %.0f", m.label, m.unit, m.ref)
		fmt.Fprintf(b, "%s\n%s\n\n", styleDetailLabel.Render(caption), lc.View())
	}
}

// clampOutliers replaces statistical outliers (Tukey: Q3 + 3×IQR) with the
// upper fence value so a single bad WW API data point doesn't collapse the
// chart scale and make every other day look like zero.
func clampOutliers(vals []float64) []float64 {
	if len(vals) < 4 {
		return vals
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	n := len(sorted)
	q1 := sorted[n/4]
	q3 := sorted[n*3/4]
	iqr := q3 - q1
	if iqr <= 0 {
		return vals
	}
	fence := q3 + 3*iqr
	result := make([]float64, len(vals))
	for i, v := range vals {
		if v > fence {
			result[i] = fence
		} else {
			result[i] = v
		}
	}
	return result
}

// subcellBlocks indexes Unicode partial-block runes by eighths of a cell:
// 0 → empty, 1 → ▏ (1/8), 2 → ▎ (2/8), …, 8 → █ (8/8). The boundary cell of a
// bar uses one of these to show fractional fill at ~8× horizontal precision
// of a single full block.
var subcellBlocks = [9]rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

func makeBar(value, max float64, width int, clamp bool) string {
	if width == 0 {
		return ""
	}
	eighths := 0
	if max > 0 {
		eighths = int(math.Round(value / max * float64(width) * 8))
	}
	maxEighths := width * 8
	if eighths > maxEighths {
		eighths = maxEighths
	}
	full := eighths / 8
	rem := eighths % 8

	barColor := color.Color(colorTeal)
	if !clamp && max > 0 && value > max {
		barColor = colorPurple
	}

	fillStyle := lipgloss.NewStyle().Foreground(barColor)
	markerStyle := lipgloss.NewStyle().Foreground(colorLine)

	var b strings.Builder
	if full > 0 {
		b.WriteString(fillStyle.Render(strings.Repeat("█", full)))
	}
	emptyCells := width - full
	if rem > 0 && emptyCells > 0 {
		b.WriteString(fillStyle.Render(string(subcellBlocks[rem])))
		emptyCells--
	}
	if emptyCells > 0 {
		b.WriteString(strings.Repeat(" ", emptyCells))
	}
	// Right-edge marker showing the bar's upper limit (max). Spans the full
	// length regardless of fill, so the user can see "where 100% sits".
	b.WriteString(markerStyle.Render("│"))
	return b.String()
}

func formatNutriValue(v float64) string {
	if v == 0 {
		return "—"
	}
	if v >= 100 {
		return fmt.Sprintf("%.0f", v)
	}
	if v == math.Trunc(v) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

func computeAverages(data map[string]*api.DayNutrition, logs []*api.DayLog) *api.DayNutrition {
	type fields struct {
		cals, fat, satFat, sodium, carbs, fiber, sugar, addedSugar, protein, alcohol []float64
	}
	var f fields
	for _, day := range logs {
		dn, ok := data[day.Date]
		if !ok {
			continue
		}
		f.cals = append(f.cals, dn.Calories)
		f.fat = append(f.fat, dn.Fat)
		f.satFat = append(f.satFat, dn.SaturatedFat)
		f.sodium = append(f.sodium, dn.Sodium)
		f.carbs = append(f.carbs, dn.Carbs)
		f.fiber = append(f.fiber, dn.Fiber)
		f.sugar = append(f.sugar, dn.Sugar)
		f.addedSugar = append(f.addedSugar, dn.AddedSugar)
		f.protein = append(f.protein, dn.Protein)
		f.alcohol = append(f.alcohol, dn.Alcohol)
	}
	meanOf := func(vals []float64) float64 {
		clamped := clampOutliers(vals)
		var sum float64
		for _, v := range clamped {
			sum += v
		}
		if len(clamped) == 0 {
			return 0
		}
		return sum / float64(len(clamped))
	}
	return &api.DayNutrition{
		Calories:     meanOf(f.cals),
		Fat:          meanOf(f.fat),
		SaturatedFat: meanOf(f.satFat),
		Sodium:       meanOf(f.sodium),
		Carbs:        meanOf(f.carbs),
		Fiber:        meanOf(f.fiber),
		Sugar:        meanOf(f.sugar),
		AddedSugar:   meanOf(f.addedSugar),
		Protein:      meanOf(f.protein),
		Alcohol:      meanOf(f.alcohol),
	}
}
