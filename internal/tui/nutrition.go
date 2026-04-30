package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/ali5ter/wwlog/internal/api"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	detail      viewport.Model
	logs        []*api.DayLog
	data        map[string]*api.DayNutrition
	avgs        *api.DayNutrition
	width       int
	height      int
	selected    int
	initialized bool
}

func newNutriModel(logs []*api.DayLog, data map[string]*api.DayNutrition, width, height int) nutriModel {
	listWidth := width / 3
	listHeight := height - 2

	items := make([]list.Item, len(logs))
	for i, l := range logs {
		items[i] = dateItem{log: l}
	}

	l := list.New(items, list.NewDefaultDelegate(), listWidth, listHeight)
	l.Title = "Dates"
	l.Styles.Title = styleMealHeading
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	vp := viewport.New(width-listWidth, height)

	m := nutriModel{
		list:        l,
		detail:      vp,
		logs:        logs,
		data:        data,
		avgs:        computeAverages(data, logs),
		width:       width,
		height:      height,
		initialized: true,
	}
	m.detail.SetContent(m.renderDetail())
	return m
}

func (m nutriModel) update(msg tea.Msg) (nutriModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, keys.ScrollUp):
			m.detail.LineUp(3)
			return m, nil
		case key.Matches(msg, keys.ScrollDown):
			m.detail.LineDown(3)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	selChanged := false
	if i := m.list.Index(); i != m.selected && i < len(m.logs) {
		m.selected = i
		m.detail.SetContent(m.renderDetail())
		m.detail.GotoTop()
		selChanged = true
	}
	var cmd2 tea.Cmd
	if !selChanged {
		// Only forward non-key messages to the viewport — same reasoning as logModel.
		if _, isKey := msg.(tea.KeyMsg); !isKey {
			m.detail, cmd2 = m.detail.Update(msg)
		}
	}
	return m, tea.Batch(cmd, cmd2)
}

func (m nutriModel) view() string {
	listWidth := m.width / 3
	detailWidth := m.width - listWidth

	label := styleDim.Render("> dates")
	sep := styleDim.Render(strings.Repeat("─", listWidth-1))

	listPane := stylePanelBorder.Width(listWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, label, sep, m.list.View()),
	)
	detailPane := lipgloss.NewStyle().Width(detailWidth).Padding(0, 1).Render(m.detail.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, detailPane)
}

func (m *nutriModel) resize(width, height int) {
	if !m.initialized {
		return
	}
	m.width = width
	m.height = height
	listWidth := width / 3
	m.list.SetSize(listWidth, height-2)
	detailWidth := width - listWidth
	m.detail.Width = detailWidth
	m.detail.Height = height
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

	vw := m.detail.Width - 2
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
	fmt.Fprintf(&b, "%s\n", styleMealHeading.Render(formatDateLong(day.Date)))
	fmt.Fprintf(&b, "%s\n\n", styleDim.Render(strings.Repeat("─", sepWidth)))

	// Points summary.
	renderPointsSummary(&b, day.Points, vw)

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

	for _, m := range metrics {
		vals := make([]float64, n)
		for i, day := range logs {
			if dn, ok := data[day.Date]; ok {
				vals[i] = m.get(dn)
			}
		}
		chart := asciigraph.Plot(vals,
			asciigraph.Height(4),
			asciigraph.Width(plotW),
			asciigraph.SeriesColors(asciigraph.MediumAquamarine),
			asciigraph.LabelColor(asciigraph.SlateGray),
			asciigraph.AxisColor(asciigraph.SlateGray),
			asciigraph.Caption(fmt.Sprintf("%s (%s)", m.label, m.unit)),
			asciigraph.CaptionColor(asciigraph.LightSlateGray),
			asciigraph.XAxisRange(0, float64(n-1)),
			asciigraph.XAxisTickCount(n),
			asciigraph.XAxisValueFormatter(dateLabel),
		)
		fmt.Fprintf(b, "%s\n\n", chart)
	}
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
	barColor := colorTeal
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
	avg := &api.DayNutrition{}
	n := 0
	for _, day := range logs {
		dn, ok := data[day.Date]
		if !ok {
			continue
		}
		avg.Calories += dn.Calories
		avg.Fat += dn.Fat
		avg.SaturatedFat += dn.SaturatedFat
		avg.Sodium += dn.Sodium
		avg.Carbs += dn.Carbs
		avg.Fiber += dn.Fiber
		avg.Sugar += dn.Sugar
		avg.Protein += dn.Protein
		avg.Alcohol += dn.Alcohol
		n++
	}
	if n > 0 {
		f := float64(n)
		avg.Calories /= f
		avg.Fat /= f
		avg.SaturatedFat /= f
		avg.Sodium /= f
		avg.Carbs /= f
		avg.Fiber /= f
		avg.Sugar /= f
		avg.Protein /= f
		avg.Alcohol /= f
	}
	return avg
}
