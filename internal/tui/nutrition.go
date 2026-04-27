package tui

import (
	"fmt"
	"strings"

	"github.com/ali5ter/wwlog/internal/api"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// nutriModel is the nutrition summary tab.
// TODO: Add per-day bar charts using lipgloss once nutrition fetch is wired in.
type nutriModel struct {
	viewport viewport.Model
	logs     []*api.DayLog
	width    int
	height   int
}

func newNutriModel(logs []*api.DayLog, width, height int) nutriModel {
	vp := viewport.New(width, height)
	vp.SetContent(renderNutritionPlaceholder(logs))
	return nutriModel{viewport: vp, logs: logs, width: width, height: height}
}

func (m nutriModel) update(msg tea.Msg) (nutriModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m nutriModel) view() string {
	return m.viewport.View()
}

func (m *nutriModel) resize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
}

func renderNutritionPlaceholder(logs []*api.DayLog) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", styleMealHeading.Render("Nutrition Summary"))
	fmt.Fprintf(&b, "%s\n\n",
		styleDetailValue.Render("Nutritional data requires fetching each food item individually."),
	)
	fmt.Fprintf(&b, "%s\n",
		styleFoodPortion.Render("Run with --nutrition flag to enable, or press ^E to export a nutrition CSV."),
	)
	fmt.Fprintf(&b, "\n%s\n", styleDim.Render(strings.Repeat("─", 40)))
	for _, day := range logs {
		total := len(day.Meals.Morning) + len(day.Meals.Midday) +
			len(day.Meals.Evening) + len(day.Meals.Anytime)
		fmt.Fprintf(&b, "  %s  %s\n",
			styleDetailLabel.Render(day.Date),
			styleFoodPortion.Render(fmt.Sprintf("%d items", total)),
		)
	}
	return b.String()
}
