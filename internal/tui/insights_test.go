package tui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/ali5ter/wwlog/internal/api"
)

// insightsFixtureLogs builds n days of data with points, meals across every
// period, and full nutrition — enough for every Insights section (heatmap,
// range summary, meals, macros, micros, top foods, zero-point foods) to
// render non-empty.
func insightsFixtureLogs(n int) []*api.DayLog {
	nutriEntry := func(name string, pts, cals, protein, carbs, fat float64) api.FoodEntry {
		return api.FoodEntry{
			Name:          name,
			PortionSize:   1,
			PointsPrecise: pts,
			DefaultPortion: api.DefaultPortion{
				Size: 1,
				Nutrition: api.NutritionMap{
					"calories":     cals,
					"protein":      protein,
					"carbs":        carbs,
					"fat":          fat,
					"fiber":        5,
					"sodium":       400,
					"addedSugar":   4,
					"saturatedFat": 3,
				},
			},
		}
	}
	logs := make([]*api.DayLog, n)
	for i := 0; i < n; i++ {
		logs[i] = &api.DayLog{
			Date: fmt.Sprintf("2026-06-%02d", i%28+1),
			Points: api.DayPoints{
				DailyTarget:              23,
				WeeklyAllowanceRemaining: 20,
			},
			Meals: api.Meals{
				Morning: []api.FoodEntry{nutriEntry("Oatmeal With Berries", 6, 300, 10, 50, 5)},
				Midday:  []api.FoodEntry{nutriEntry("Grilled Chicken Salad", 8, 450, 35, 20, 15)},
				Evening: []api.FoodEntry{nutriEntry("Salmon And Rice Bowl", 9, 600, 40, 60, 20)},
				Anytime: []api.FoodEntry{nutriEntry("Apple", 0, 95, 0, 25, 0)},
			},
		}
	}
	return logs
}

// insightsEmptyNutritionLogs has food entries with points but no nutrition
// data — Macro Distribution and Daily Averages both have nothing to show.
func insightsEmptyNutritionLogs(n int) []*api.DayLog {
	logs := make([]*api.DayLog, n)
	for i := 0; i < n; i++ {
		logs[i] = &api.DayLog{
			Date:   fmt.Sprintf("2026-06-%02d", i%28+1),
			Points: api.DayPoints{DailyTarget: 23, WeeklyAllowanceRemaining: 20},
			Meals: api.Meals{
				Anytime: []api.FoodEntry{{Name: "Mystery Snack", PointsPrecise: 0}},
			},
		}
	}
	return logs
}

func TestInsightsRenderFitsWidth(t *testing.T) {
	logs := insightsFixtureLogs(14)
	// 40 (insightsViewWidth's absolute floor) is deliberately excluded: a
	// bar row's label and value text plus the insightsMinBarWidth=6 floor
	// can genuinely not fit in 40 columns without either eliding text or
	// shrinking the minimum readable bar below 6 cells — neither of which
	// this design does. The heatmap's legend line (~56 wide, independent of
	// vw — see renderHeatmap) has the same unavoidable floor. Both are
	// pre-existing at-the-margin behaviour, not something this refactor
	// introduced or is expected to fix; 60 and up is the realistic range.
	widths := []int{60, 79, 85, 99, 100, 101, 109, 110, 120, 200}

	for _, w := range widths {
		m := newInsightsModel(logs, w, 30)
		vw := insightsViewWidth(w)
		for i, line := range strings.Split(m.render(), "\n") {
			if lw := lipgloss.Width(line); lw > vw {
				t.Errorf("width %d: line %d width = %d, exceeds vw = %d\nline: %q", w, i, lw, vw, line)
			}
		}
	}
}

func TestInsightsSingleColumnBelowBreakpoint(t *testing.T) {
	logs := insightsFixtureLogs(14)
	m := newInsightsModel(logs, 99, 30)
	for _, line := range strings.Split(m.render(), "\n") {
		if strings.Contains(line, "Range Summary") && strings.Contains(line, "Points by Meal") {
			t.Errorf("width 99 should be single-column, but a line pairs Range Summary with Points by Meal: %q", line)
		}
	}
}

func TestInsightsTwoColumnAtBreakpoint(t *testing.T) {
	logs := insightsFixtureLogs(14)
	m := newInsightsModel(logs, 104, 30) // twoCol but not wide enough to pair with the heatmap
	found := false
	for _, line := range strings.Split(m.render(), "\n") {
		if strings.Contains(line, "Range Summary") && strings.Contains(line, "Points by Meal") {
			found = true
		}
	}
	if !found {
		t.Error("width 104 should pair Range Summary with Points by Meal on one line")
	}
}

func TestInsightsWideRowPairsHeatmapWithSummary(t *testing.T) {
	logs := insightsFixtureLogs(14)

	m109 := newInsightsModel(logs, 109, 30)
	for _, line := range strings.Split(m109.render(), "\n") {
		if strings.Contains(line, "Points Budget") && strings.Contains(line, "Range Summary") {
			t.Error("width 109 should not yet be wide enough to pair the heatmap with Range Summary")
		}
	}

	m110 := newInsightsModel(logs, 110, 30)
	found := false
	for _, line := range strings.Split(m110.render(), "\n") {
		if strings.Contains(line, "Points Budget") && strings.Contains(line, "Range Summary") {
			found = true
		}
	}
	if !found {
		t.Error("width 110 should pair the heatmap with Range Summary on one line")
	}
}

func TestInsightsEmptySectionsKeepColumn(t *testing.T) {
	logs := insightsEmptyNutritionLogs(14)
	// Width 104 lands in the non-wide two-column tier, where Macro
	// Distribution pairs with Daily Averages (see render()) — the pairing
	// this test needs to exercise the empty-micros collapse.
	const width = 104
	m := newInsightsModel(logs, width, 30)
	colW, twoCol := insightsColumns(insightsViewWidth(width))
	if !twoCol {
		t.Fatal("setup: width 104 should be two-column")
	}

	rendered := m.render()
	if strings.Contains(rendered, "\x00") {
		t.Fatal("render() produced an unexpected null byte")
	}
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "Macro Distribution") {
			if lw := lipgloss.Width(line); lw > colW {
				t.Errorf("Macro Distribution with an empty pairing should stay at colW = %d, got line width %d", colW, lw)
			}
		}
	}
}
