package tui

import (
	"fmt"
	"strings"

	"github.com/ali5ter/wwlog/internal/api"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type insightsModel struct {
	viewport    viewport.Model
	logs        []*api.DayLog
	width       int
	height      int
	initialized bool
}

func newInsightsModel(logs []*api.DayLog, width, height int) insightsModel {
	vp := viewport.New(width-2, height)
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
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, keys.ScrollUp):
			m.viewport.LineUp(3)
			return m, nil
		case key.Matches(msg, keys.ScrollDown):
			m.viewport.LineDown(3)
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
	m.viewport.Width = width - 2
	m.viewport.Height = height
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
	foods := api.TopFoods(m.logs, 20)

	var b strings.Builder

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
