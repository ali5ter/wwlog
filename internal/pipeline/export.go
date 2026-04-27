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
	return json.NewEncoder(os.Stdout).Encode(logs)
}

// EmitMarkdown writes a Markdown food log report to w.
func EmitMarkdown(w io.Writer, logs []*api.DayLog) error {
	fmt.Fprintf(w, "# Weight Watchers Food Log\n\n")
	for _, day := range logs {
		fmt.Fprintf(w, "## %s\n\n", day.Date)
		writeMealMD(w, "Breakfast", day.Meals.Morning)
		writeMealMD(w, "Lunch", day.Meals.Midday)
		writeMealMD(w, "Dinner", day.Meals.Evening)
		writeMealMD(w, "Snacks", day.Meals.Anytime)
	}
	return nil
}

// EmitCSV writes nutritional data as CSV to w.
func EmitCSV(w io.Writer, rows []*api.NutritionData) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"Date", "When", "Food",
		"Calories", "Fat", "Saturated Fat", "Sodium",
		"Carbohydrates", "Fiber", "Sugars", "Added Sugar", "Protein",
	})
	for _, r := range rows {
		if r == nil {
			continue
		}
		_ = cw.Write([]string{
			r.TrackedDate, r.TimeOfDay,
			fmt.Sprintf("%s, %.1f %s", r.Name, r.PortionSize, r.PortionName),
			fmt.Sprintf("%.0f", r.Calories),
			fmt.Sprintf("%.1f", r.Fat),
			fmt.Sprintf("%.1f", r.SaturatedFat),
			fmt.Sprintf("%.1f", r.Sodium),
			fmt.Sprintf("%.1f", r.Carbs),
			fmt.Sprintf("%.1f", r.Fiber),
			fmt.Sprintf("%.1f", r.Sugar),
			fmt.Sprintf("%.1f", r.AddedSugar),
			fmt.Sprintf("%.1f", r.Protein),
		})
	}
	cw.Flush()
	return cw.Error()
}

func writeMealMD(w io.Writer, name string, entries []api.FoodEntry) {
	fmt.Fprintf(w, "### %s\n\n", name)
	if len(entries) == 0 {
		fmt.Fprintf(w, "_Nothing logged._\n\n")
		return
	}
	for _, e := range entries {
		suffix := ""
		if e.PortionName != "" {
			suffix = fmt.Sprintf(", %.4g %s", e.PortionSize, e.PortionName)
		}
		fmt.Fprintf(w, "- %s%s\n", e.Name, strings.TrimRight(suffix, "0."))
	}
	fmt.Fprintln(w)
}
