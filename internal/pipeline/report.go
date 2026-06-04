package pipeline

import (
	"fmt"
	"io"
	"strings"

	"github.com/ali5ter/wwlog/config"
	"github.com/ali5ter/wwlog/internal/api"
)

// EmitTextReport writes a human-readable insights report to w.
// When targets is non-nil and has configured values, the DAILY AVERAGES section
// includes ✓/✗ hit/miss indicators per metric.
func EmitTextReport(w io.Writer, logs []*api.DayLog, targets *config.Targets) error {
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
	fmt.Fprintf(w, "  Activity:  %.0fpt earned  (%d days with activity)\n", s.TotalActivityEarned, s.DaysWithActivity)

	// Points per day table
	fmt.Fprintf(w, "\nDAILY POINTS\n")
	fmt.Fprintf(w, "  %-12s  %6s  %6s  %6s  %8s\n", "Date", "Used", "Target", "Left", "Activity")
	fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 46))
	for _, day := range logs {
		p := day.Points
		if p.DailyTarget == 0 {
			continue
		}
		fmt.Fprintf(w, "  %-12s  %6.0f  %6.0f  %6.0f  %+8.0f\n",
			day.Date, p.DailyUsed, p.DailyTarget, p.DailyRemaining, p.ActivityEarned)
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
		if targets != nil && targets.HasAny() {
			proteinMark, proteinRef := hitMarkRange(macros.ProteinG, targets.ProteinG)
			carbsMark, carbsRef := hitMarkRange(macros.CarbsG, targets.CarbsG)
			fatMark, fatRef := hitMarkRange(macros.FatG, targets.FatG)
			fmt.Fprintf(w, "  %-14s  %5.1f%%  %6.0fg avg  %s%s\n", "Protein", macros.ProteinPct, macros.ProteinG, proteinRef, proteinMark)
			fmt.Fprintf(w, "  %-14s  %5.1f%%  %6.0fg avg  %s%s\n", "Carbohydrates", macros.CarbsPct, macros.CarbsG, carbsRef, carbsMark)
			fmt.Fprintf(w, "  %-14s  %5.1f%%  %6.0fg avg  %s%s\n", "Fat", macros.FatPct, macros.FatG, fatRef, fatMark)
		} else {
			fmt.Fprintf(w, "  %-14s  %5.1f%%  %6.0fg avg\n", "Protein", macros.ProteinPct, macros.ProteinG)
			fmt.Fprintf(w, "  %-14s  %5.1f%%  %6.0fg avg\n", "Carbohydrates", macros.CarbsPct, macros.CarbsG)
			fmt.Fprintf(w, "  %-14s  %5.1f%%  %6.0fg avg\n", "Fat", macros.FatPct, macros.FatG)
		}
		if macros.AlcoholG > 0 {
			fmt.Fprintf(w, "  %-14s  %5.1f%%  %6.0fg avg\n", "Alcohol", macros.AlcoholPct, macros.AlcoholG)
		}
	}

	// Daily averages — fiber, sodium, added sugar
	nutrition := api.ComputeAllNutrition(logs)
	var fiberSum, sodiumSum, addedSugarSum float64
	daysWithData := 0
	for _, dn := range nutrition {
		if dn.ItemCount > 0 {
			fiberSum += dn.Fiber
			sodiumSum += dn.Sodium
			addedSugarSum += dn.AddedSugar
			daysWithData++
		}
	}
	if daysWithData > 0 {
		n := float64(daysWithData)
		avgFiber := fiberSum / n
		avgSodium := sodiumSum / n
		avgAddedSugar := addedSugarSum / n

		fmtMicro := func(v float64) string {
			if v == 0 {
				return "—"
			}
			return fmt.Sprintf("%.0f", v)
		}

		fmt.Fprintf(w, "\nDAILY AVERAGES (additional)\n")

		if targets != nil && targets.HasAny() {
			// With configured targets: show target and hit/miss indicator.
			fiberMark, fiberRef := hitMarkFloor(avgFiber, targets.FiberGMin, 28.0)
			sodiumMark, sodiumRef := hitMarkCeil(avgSodium, targets.SodiumMgMax, 2300.0)
			addedSugarMark, addedSugarRef := hitMarkCeil(avgAddedSugar, targets.AddedSugarGMax, 35.0)
			fmt.Fprintf(w, "  %-14s  %s g avg   target ≥%.0fg   %s\n", "Fiber", fmtMicro(avgFiber), fiberRef, fiberMark)
			fmt.Fprintf(w, "  %-14s  %s mg avg  target ≤%.0fmg  %s\n", "Sodium", fmtMicro(avgSodium), sodiumRef, sodiumMark)
			fmt.Fprintf(w, "  %-14s  %s g avg   target ≤%.0fg   %s\n", "Added Sugar", fmtMicro(avgAddedSugar), addedSugarRef, addedSugarMark)
		} else {
			// No configured targets: show generic reference values.
			fmt.Fprintf(w, "  %-14s  %s g avg   (ref ≥%.0fg)\n", "Fiber", fmtMicro(avgFiber), 28.0)
			fmt.Fprintf(w, "  %-14s  %s mg avg  (ref ≤%.0fmg)\n", "Sodium", fmtMicro(avgSodium), 2300.0)
			fmt.Fprintf(w, "  %-14s  %s g avg   (ref ≤%.0fg)\n", "Added Sugar", fmtMicro(avgAddedSugar), 35.0)
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

// hitMarkFloor returns a ✓/✗ indicator and the effective floor value.
// When configured is 0, falls back to fallback.
func hitMarkFloor(avg, configured, fallback float64) (mark string, ref float64) {
	ref = fallback
	if configured > 0 {
		ref = configured
	}
	if avg >= ref {
		return "✓", ref
	}
	return "✗", ref
}

// hitMarkCeil returns a ✓/✗ indicator and the effective ceiling value.
// When configured is 0, falls back to fallback.
func hitMarkCeil(avg, configured, fallback float64) (mark string, ref float64) {
	ref = fallback
	if configured > 0 {
		ref = configured
	}
	if avg <= ref {
		return "✓", ref
	}
	return "✗", ref
}

// hitMarkRange returns a ✓/✗ indicator and a formatted target string for a range band.
// Returns empty strings when band is unconfigured (len < 2).
func hitMarkRange(avg float64, band []float64) (mark, ref string) {
	if len(band) < 2 {
		return "", ""
	}
	ref = fmt.Sprintf("target %.0f–%.0fg  ", band[0], band[1])
	if avg >= band[0] {
		return "✓", ref
	}
	return "✗", ref
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
