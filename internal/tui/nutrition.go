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
	"github.com/ali5ter/wwlog/internal/api"
	"github.com/guptarohit/asciigraph"
)

// rdv holds reference daily values used as bar maximums.
var rdv = &api.DayNutrition{
	Calories:     2000,
	Fat:          65,
	SaturatedFat: 20,
	Sodium:       2300,
	Carbs:        300,
	Fiber:        28,
	Sugar:        50,
	Protein:      50,
	Alcohol:      28,
}

type nutriModel struct {
	list        list.Model
	filter      textinput.Model
	filtering   bool
	detail      viewport.Model
	allLogs     []*api.DayLog
	logs        []*api.DayLog // filtered view
	allData     map[string]*api.DayNutrition
	data        map[string]*api.DayNutrition // filtered view
	avgs        *api.DayNutrition
	width       int
	height      int
	selected    int
	locale      locale
	initialized bool
}

func newNutriModel(logs []*api.DayLog, data map[string]*api.DayNutrition, width, height int, loc locale) nutriModel {
	listWidth := width / 3
	listHeight := height - 2

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

	vp := viewport.New(viewport.WithWidth(width-listWidth), viewport.WithHeight(height))
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
				return m, textinput.Blink
			case key.Matches(msg, keys.ScrollUp):
				m.detail.ScrollUp(3)
				return m, nil
			case key.Matches(msg, keys.ScrollDown):
				m.detail.ScrollDown(3)
				return m, nil
			}
		}
	}

	// Mouse wheel scrolls the detail viewport, not the date list.
	if _, ok := msg.(tea.MouseWheelMsg); ok {
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	}

	// Click on a date row in the list pane selects that row.
	if click, ok := msg.(tea.MouseClickMsg); ok {
		if idx, ok := m.dateRowAtPoint(click.X, click.Y); ok {
			m.list.Select(idx)
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
	listWidth := m.width / 3
	detailWidth := m.width - listWidth

	var filterBar string
	if m.filtering {
		filterBar = m.filter.View()
	} else if m.filter.Value() != "" {
		filterBar = styleFilterPrompt.Render("> ") + styleFilterText.Render(m.filter.Value())
	} else {
		filterBar = styleDim.Render("> filter by date…")
	}
	filterSep := styleDim.Render(strings.Repeat("─", listWidth-1))

	listPane := stylePanelBorder.Width(listWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, filterBar, filterSep, m.list.View()),
	)
	detailPane := lipgloss.NewStyle().Width(detailWidth).Padding(0, 1).Render(m.detail.View())
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

// dateRowAtPoint mirrors logModel.dateRowAtPoint — returns the absolute
// list index at the given terminal coordinate, or false if the point is
// not on a list row.
func (m nutriModel) dateRowAtPoint(x, y int) (int, bool) {
	listWidth := m.width / 3
	if x < 0 || x >= listWidth {
		return 0, false
	}
	const headerRows = 2
	const filterRows = 2
	rowStride := defaultDelegateRowStride()
	rowsTop := headerRows + filterRows
	if y < rowsTop {
		return 0, false
	}
	first := m.list.Paginator.Page * m.list.Paginator.PerPage
	offset := (y - rowsTop) / rowStride
	idx := first + offset
	items := m.list.Items()
	if idx < 0 || idx >= len(items) {
		return 0, false
	}
	return idx, true
}

func (m *nutriModel) resize(width, height int) {
	if !m.initialized {
		return
	}
	m.width = width
	m.height = height
	listWidth := width / 3
	m.list.SetSize(listWidth, height-2) // -2 for filter bar + separator
	detailWidth := width - listWidth
	m.detail.SetWidth(detailWidth)
	m.detail.SetHeight(height)
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

	vw := m.detail.Width() - 2
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
	renderPointsSummary(&b, day.Points, vw, m.locale)

	fmt.Fprintf(&b, "%s\n\n", styleDim.Render(strings.Repeat("─", sepWidth)))

	// Nutrition bars vs recommended daily values.
	fmt.Fprintf(&b, "%s\n\n", styleDetailLabel.Render("Nutrition  (bars show % of daily reference)"))
	writeNutriBar(&b, "Calories", "kcal", dn.Calories, rdv.Calories, m.avgs.Calories, barWidth)
	writeNutriBar(&b, "Protein", "g", dn.Protein, rdv.Protein, m.avgs.Protein, barWidth)
	writeNutriBar(&b, "Carbs", "g", dn.Carbs, rdv.Carbs, m.avgs.Carbs, barWidth)
	writeNutriBar(&b, "Fat", "g", dn.Fat, rdv.Fat, m.avgs.Fat, barWidth)
	writeNutriBar(&b, "Sat Fat", "g", dn.SaturatedFat, rdv.SaturatedFat, m.avgs.SaturatedFat, barWidth)
	writeNutriBar(&b, "Fiber", "g", dn.Fiber, rdv.Fiber, m.avgs.Fiber, barWidth)
	writeNutriBar(&b, "Sodium", "mg", dn.Sodium, rdv.Sodium, m.avgs.Sodium, barWidth)
	writeNutriBar(&b, "Sugar", "g", dn.Sugar, rdv.Sugar, m.avgs.Sugar, barWidth)
	if dn.Alcohol > 0 || m.avgs.Alcohol > 0 {
		writeNutriBar(&b, "Alcohol", "g", dn.Alcohol, rdv.Alcohol, m.avgs.Alcohol, barWidth)
	}

	if len(m.logs) > 1 {
		fmt.Fprintf(&b, "\n%s\n\n", styleDim.Render(strings.Repeat("─", sepWidth)))
		fmt.Fprintf(&b, "%s\n\n", styleDetailLabel.Render("Trends across date range"))
		writeTrendTable(&b, m.logs, m.data, vw)
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

func writeNutriBar(b *strings.Builder, label, unit string, value, max, avg float64, barWidth int) {
	bar := makeBar(value, max, barWidth)
	labelCol := lipgloss.NewStyle().Width(11).Render(styleDetailLabel.Render(label))
	valCol := lipgloss.NewStyle().Width(14).Render(styleDetailValue.Render(fmt.Sprintf("%s %s", formatNutriValue(value), unit)))
	avgCol := styleFoodPortion.Render(fmt.Sprintf("avg %s", formatNutriValue(avg)))
	fmt.Fprintf(b, "  %s%s%s  %s\n", labelCol, valCol, bar, avgCol)
}

func writeTrendTable(b *strings.Builder, logs []*api.DayLog, data map[string]*api.DayNutrition, vw int) {
	type series struct {
		label string
		unit  string
		get   func(*api.DayNutrition) float64
	}
	metrics := []series{
		{"Calories", "kcal", func(d *api.DayNutrition) float64 { return d.Calories }},
		{"Protein", "g", func(d *api.DayNutrition) float64 { return d.Protein }},
		{"Carbs", "g", func(d *api.DayNutrition) float64 { return d.Carbs }},
		{"Fat", "g", func(d *api.DayNutrition) float64 { return d.Fat }},
	}

	n := len(logs)
	// Y-axis labels for Calories are up to 4 digits wide + " │" = ~8 chars offset.
	// Leave a generous margin so chart fits within vw.
	plotW := vw - 14
	if plotW < 8 {
		plotW = 8
	}

	dateLabel := func(v float64) string {
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
	ticks := min(n, 7) // cap so labels don't crowd on wide date ranges

	for _, m := range metrics {
		vals := make([]float64, n)
		for i, day := range logs {
			if dn, ok := data[day.Date]; ok {
				vals[i] = m.get(dn)
			}
		}
		vals = clampOutliers(vals)
		chart := asciigraph.Plot(vals,
			asciigraph.Height(4),
			asciigraph.Width(plotW),
			asciigraph.SeriesColors(asciigraph.MediumAquamarine),
			asciigraph.LabelColor(asciigraph.SlateGray),
			asciigraph.AxisColor(asciigraph.SlateGray),
			asciigraph.Caption(fmt.Sprintf("%s (%s)", m.label, m.unit)),
			asciigraph.CaptionColor(asciigraph.LightSlateGray),
			asciigraph.XAxisRange(0, float64(n-1)),
			asciigraph.XAxisTickCount(ticks),
			asciigraph.XAxisValueFormatter(dateLabel),
		)
		fmt.Fprintf(b, "%s\n\n", chart)
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

func makeBar(value, max float64, width int) string {
	if width == 0 {
		return ""
	}
	filled := 0
	if max > 0 {
		filled = int(math.Round(value / max * float64(width)))
	}
	if filled > width {
		filled = width
	}
	empty := width - filled
	barColor := color.Color(colorTeal)
	if max > 0 && value > max {
		barColor = colorPurple
	}
	return lipgloss.NewStyle().Foreground(barColor).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(colorSteel).Render(strings.Repeat("░", empty))
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
		cals, fat, satFat, sodium, carbs, fiber, sugar, protein, alcohol []float64
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
		Protein:      meanOf(f.protein),
		Alcohol:      meanOf(f.alcohol),
	}
}
