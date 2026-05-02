// Package pipeline handles non-TUI output modes (JSON, Markdown, CSV).
package pipeline

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ali5ter/wwlog/internal/api"
)

// EmitJSON writes all day logs as a JSON array to stdout.
func EmitJSON(logs []*api.DayLog) error {
	return WriteJSON(os.Stdout, logs)
}

// WriteJSON encodes logs as a JSON array to w.
func WriteJSON(w io.Writer, logs []*api.DayLog) error {
	return json.NewEncoder(w).Encode(logs)
}

// WriteLogCSV writes a CSV of food log entries with points and calories to w.
func WriteLogCSV(w io.Writer, logs []*api.DayLog) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"Date", "Meal", "Food", "Serving", "Points", "Calories", "Protein (g)", "Carbs (g)", "Fat (g)"})
	for _, day := range logs {
		for meal, entries := range map[string][]api.FoodEntry{
			"Breakfast": day.Meals.Morning,
			"Lunch":     day.Meals.Midday,
			"Dinner":    day.Meals.Evening,
			"Snacks":    day.Meals.Anytime,
		} {
			for _, e := range entries {
				serving := e.ServingDesc
				if serving == "" && e.PortionName != "" {
					serving = fmt.Sprintf("%g %s", e.PortionSize, e.PortionName)
				}
				n := e.Nutrition()
				_ = cw.Write([]string{
					day.Date, meal, e.Name, serving,
					fmt.Sprintf("%.0f", e.PointsPrecise),
					fmt.Sprintf("%.0f", n.Calories),
					fmt.Sprintf("%.1f", n.Protein),
					fmt.Sprintf("%.1f", n.Carbs),
					fmt.Sprintf("%.1f", n.Fat),
				})
			}
		}
	}
	cw.Flush()
	return cw.Error()
}

// EmitMarkdown writes a Markdown food log report to w.
func EmitMarkdown(w io.Writer, logs []*api.DayLog) error {
	fmt.Fprintf(w, "# Food Log\n\n")
	for _, day := range logs {
		fmt.Fprintf(w, "## %s\n\n", day.Date)
		p := day.Points
		if p.DailyTarget > 0 {
			fmt.Fprintf(w, "**Points:** %.0f / %.0f used  ·  Weekly bank: %.0f\n\n",
				p.DailyUsed, p.DailyTarget, p.WeeklyAllowanceRemaining)
		}
		writeMealMD(w, "Breakfast", day.Meals.Morning)
		writeMealMD(w, "Lunch", day.Meals.Midday)
		writeMealMD(w, "Dinner", day.Meals.Evening)
		writeMealMD(w, "Snacks", day.Meals.Anytime)
	}
	return nil
}

func writeMealMD(w io.Writer, name string, entries []api.FoodEntry) {
	fmt.Fprintf(w, "### %s\n\n", name)
	if len(entries) == 0 {
		fmt.Fprintf(w, "_Nothing logged._\n\n")
		return
	}
	for _, e := range entries {
		serving := e.ServingDesc
		if serving == "" && e.PortionName != "" {
			serving = fmt.Sprintf("%.4g %s", e.PortionSize, e.PortionName)
			serving = strings.TrimRight(serving, "0.")
		}
		cal := e.Nutrition().Calories
		meta := ""
		if serving != "" {
			meta += ", " + serving
		}
		if e.PointsPrecise > 0 {
			meta += fmt.Sprintf(", %.0fpt", e.PointsPrecise)
		}
		if cal > 0 {
			meta += fmt.Sprintf(", %.0f kcal", cal)
		}
		fmt.Fprintf(w, "- %s%s\n", e.Name, meta)
	}
	fmt.Fprintln(w)
}
