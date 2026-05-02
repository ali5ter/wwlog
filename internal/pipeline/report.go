package pipeline

import (
	"fmt"
	"io"
	"strings"

	"github.com/ali5ter/wwlog/internal/api"
)

// EmitTextReport writes a human-readable insights report to w.
func EmitTextReport(w io.Writer, logs []*api.DayLog) error {
	if len(logs) == 0 {
		fmt.Fprintln(w, "No data.")
		return nil
	}

	start := logs[0].Date
	end := logs[len(logs)-1].Date
	fmt.Fprintf(w, "Food Log — %s → %s\n", start, end)
	fmt.Fprintln(w, strings.Repeat("─", 60))

	// Range summary
	s := api.ComputeRangeSummary(logs)
	fmt.Fprintf(w, "\nSUMMARY\n")
	fmt.Fprintf(w, "  %d days  ·  %d food items logged\n", s.Days, s.TotalItems)
	if s.AvgDailyTarget > 0 {
		fmt.Fprintf(w, "  Points:    avg %.0fpt / %.0fpt target  (%d on/under budget, %d over)\n",
			s.AvgDailyPts, s.AvgDailyTarget, s.DaysUnderBudget, s.DaysOverBudget)
	}
	if s.AvgDailyCals > 0 {
		fmt.Fprintf(w, "  Calories:  avg %.0f kcal / day\n", s.AvgDailyCals)
	}

	// Points per day table
	fmt.Fprintf(w, "\nDAILY POINTS\n")
	fmt.Fprintf(w, "  %-12s  %6s  %6s  %6s\n", "Date", "Used", "Target", "Left")
	fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 36))
	for _, day := range logs {
		p := day.Points
		if p.DailyTarget == 0 {
			continue
		}
		fmt.Fprintf(w, "  %-12s  %6.0f  %6.0f  %6.0f\n",
			day.Date, p.DailyUsed, p.DailyTarget, p.DailyRemaining)
	}

	// Points by meal
	meals := api.MealStats(logs)
	fmt.Fprintf(w, "\nPOINTS BY MEAL  (average per day)\n")
	for _, ms := range meals {
		fmt.Fprintf(w, "  %s %-12s  %.1fpt  ·  %.0f kcal\n",
			ms.Symbol, ms.Name, ms.AvgPts, ms.AvgCals)
	}

	// Macro distribution
	macros := api.AvgMacroBreakdown(logs)
	if macros.ProteinG+macros.CarbsG+macros.FatG > 0 {
		fmt.Fprintf(w, "\nMACRO DISTRIBUTION  (average daily)\n")
		fmt.Fprintf(w, "  %-14s  %5.1f%%  %6.0fg avg\n", "Protein", macros.ProteinPct, macros.ProteinG)
		fmt.Fprintf(w, "  %-14s  %5.1f%%  %6.0fg avg\n", "Carbohydrates", macros.CarbsPct, macros.CarbsG)
		fmt.Fprintf(w, "  %-14s  %5.1f%%  %6.0fg avg\n", "Fat", macros.FatPct, macros.FatG)
		if macros.AlcoholG > 0 {
			fmt.Fprintf(w, "  %-14s  %5.1f%%  %6.0fg avg\n", "Alcohol", macros.AlcoholPct, macros.AlcoholG)
		}
	}

	// Top foods by points
	foods := api.TopFoods(logs, 20)
	fmt.Fprintf(w, "\nTOP FOODS BY POINTS\n")
	fmt.Fprintf(w, "  %-34s  %3s  %8s  %8s  %10s\n", "Food", "N", "Tot pts", "Avg pts", "Avg kcal")
	fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 70))
	for _, fs := range foods {
		calStr := "—"
		if fs.AvgCals > 0 {
			calStr = fmt.Sprintf("%.0f", fs.AvgCals)
		}
		fmt.Fprintf(w, "  %-34s  %3d  %8.0f  %8.1f  %10s\n",
			truncateStr(fs.Name, 34), fs.Count, fs.TotalPts, fs.AvgPts, calStr)
	}

	// Zero-point foods
	zp := zeroPointList(logs)
	if len(zp) > 0 {
		fmt.Fprintf(w, "\nZERO-POINT FOODS LOGGED\n")
		for _, fs := range zp {
			calStr := ""
			if fs.AvgCals > 0 {
				calStr = fmt.Sprintf("  (%.0f kcal avg)", fs.AvgCals)
			}
			fmt.Fprintf(w, "  %-34s  %d×%s\n", truncateStr(fs.Name, 34), fs.Count, calStr)
		}
	}

	return nil
}

func truncateStr(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func zeroPointList(logs []*api.DayLog) []api.FoodStat {
	all := api.TopFoods(logs, 0)
	var zp []api.FoodStat
	for _, fs := range all {
		if fs.TotalPts == 0 {
			zp = append(zp, fs)
		}
	}
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
