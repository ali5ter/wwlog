package tui

import (
	"fmt"
	"strings"

	"github.com/ali5ter/wwlog/internal/api"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// logModel is the food log tab — a date list on the left, meal detail on the right.
type logModel struct {
	list     list.Model
	detail   viewport.Model
	logs     []*api.DayLog
	width    int
	height   int
	selected int
}

type dateItem struct {
	log *api.DayLog
}

func (d dateItem) Title() string       { return d.log.Date }
func (d dateItem) Description() string { return mealSummary(d.log) }
func (d dateItem) FilterValue() string { return d.log.Date }

func newLogModel(logs []*api.DayLog, width, height int) logModel {
	listWidth := width / 3
	detailWidth := width - listWidth

	items := make([]list.Item, len(logs))
	for i, l := range logs {
		items[i] = dateItem{log: l}
	}

	l := list.New(items, list.NewDefaultDelegate(), listWidth, height)
	l.Title = "Dates"
	l.Styles.Title = styleMealHeading
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	vp := viewport.New(detailWidth, height)

	m := logModel{
		list:   l,
		detail: vp,
		logs:   logs,
		width:  width,
		height: height,
	}
	if len(logs) > 0 {
		m.detail.SetContent(renderDay(logs[0]))
	}
	return m
}

func (m logModel) update(msg tea.Msg) (logModel, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	if i := m.list.Index(); i != m.selected && i < len(m.logs) {
		m.selected = i
		m.detail.SetContent(renderDay(m.logs[i]))
		m.detail.GotoTop()
	}

	m.detail, cmd = m.detail.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m logModel) view() string {
	listPane := stylePanelBorder.Render(m.list.View())
	detailPane := m.detail.View()
	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, detailPane)
}

func (m *logModel) resize(width, height int) {
	m.width = width
	m.height = height
	listWidth := width / 3
	m.list.SetSize(listWidth, height)
	m.detail.Width = width - listWidth
	m.detail.Height = height
}

func renderDay(day *api.DayLog) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", styleMealHeading.Render(day.Date))
	renderMeal(&b, "Breakfast", day.Meals.Morning)
	renderMeal(&b, "Lunch", day.Meals.Midday)
	renderMeal(&b, "Dinner", day.Meals.Evening)
	renderMeal(&b, "Snacks", day.Meals.Anytime)
	return b.String()
}

func renderMeal(b *strings.Builder, name string, entries []api.FoodEntry) {
	fmt.Fprintf(b, "%s\n", styleDetailLabel.Render(name))
	if len(entries) == 0 {
		fmt.Fprintf(b, "  %s\n", styleFoodPortion.Render("Nothing logged"))
	}
	for _, e := range entries {
		portion := ""
		if e.PortionName != "" {
			portion = styleFoodPortion.Render(fmt.Sprintf("  %.4g %s", e.PortionSize, e.PortionName))
		}
		fmt.Fprintf(b, "  %s%s\n", styleFoodItem.Render(e.Name), portion)
	}
	fmt.Fprintln(b)
}

func mealSummary(day *api.DayLog) string {
	total := len(day.Meals.Morning) + len(day.Meals.Midday) +
		len(day.Meals.Evening) + len(day.Meals.Anytime)
	return fmt.Sprintf("%d items logged", total)
}
