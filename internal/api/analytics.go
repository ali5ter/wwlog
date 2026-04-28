package api

import (
	"sort"
	"strings"
)

// FoodStat holds aggregate analytics for a single food across a date range.
type FoodStat struct {
	Name      string
	Count     int
	TotalPts  float64
	AvgPts    float64
	TotalCals float64
	AvgCals   float64
}

// MealStat holds average points and calories for a meal period across a date range.
type MealStat struct {
	Name    string
	Symbol  string
	AvgPts  float64
	AvgCals float64
}

// MacroBreakdown holds the average daily grams and % of calories from each macronutrient.
type MacroBreakdown struct {
	ProteinPct float64
	ProteinG   float64
	CarbsPct   float64
	CarbsG     float64
	FatPct     float64
	FatG       float64
	AlcoholPct float64
	AlcoholG   float64
}

// RangeSummary holds high-level statistics for a date range.
type RangeSummary struct {
	Days            int
	TotalItems      int
	AvgDailyPts     float64
	AvgDailyTarget  float64
	DaysUnderBudget int
	DaysOverBudget  int
	AvgDailyCals    float64
}

// TopFoods returns food items sorted by total points then calories, up to limit items.
// Pass limit ≤ 0 to return all.
func TopFoods(logs []*DayLog, limit int) []FoodStat {
	acc := make(map[string]*FoodStat)
	for _, day := range logs {
		for _, e := range day.AllEntries() {
			key := strings.ToLower(e.Name)
			stat, ok := acc[key]
			if !ok {
				stat = &FoodStat{Name: e.Name}
				acc[key] = stat
			}
			nut := e.Nutrition()
			stat.Count++
			stat.TotalPts += e.PointsPrecise
			stat.TotalCals += nut.Calories
		}
	}
	stats := make([]FoodStat, 0, len(acc))
	for _, s := range acc {
		if s.Count > 0 {
			s.AvgPts = s.TotalPts / float64(s.Count)
			s.AvgCals = s.TotalCals / float64(s.Count)
		}
		stats = append(stats, *s)
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].TotalPts != stats[j].TotalPts {
			return stats[i].TotalPts > stats[j].TotalPts
		}
		return stats[i].TotalCals > stats[j].TotalCals
	})
	if limit > 0 && len(stats) > limit {
		stats = stats[:limit]
	}
	return stats
}

// MealStats returns average daily points and calories for each meal period.
func MealStats(logs []*DayLog) []MealStat {
	type acc struct{ pts, cals float64 }
	totals := map[string]*acc{
		"morning": {},
		"midday":  {},
		"evening": {},
		"anytime": {},
	}
	for _, day := range logs {
		meals := map[string][]FoodEntry{
			"morning": day.Meals.Morning,
			"midday":  day.Meals.Midday,
			"evening": day.Meals.Evening,
			"anytime": day.Meals.Anytime,
		}
		for period, entries := range meals {
			for _, e := range entries {
				totals[period].pts += e.PointsPrecise
				totals[period].cals += e.Nutrition().Calories
			}
		}
	}
	days := float64(len(logs))
	if days == 0 {
		days = 1
	}
	return []MealStat{
		{Name: "Breakfast", Symbol: "☀", AvgPts: totals["morning"].pts / days, AvgCals: totals["morning"].cals / days},
		{Name: "Lunch", Symbol: "☁", AvgPts: totals["midday"].pts / days, AvgCals: totals["midday"].cals / days},
		{Name: "Dinner", Symbol: "☽", AvgPts: totals["evening"].pts / days, AvgCals: totals["evening"].cals / days},
		{Name: "Snacks", Symbol: "✦", AvgPts: totals["anytime"].pts / days, AvgCals: totals["anytime"].cals / days},
	}
}

// AvgMacroBreakdown computes the average daily macro distribution across all logs.
// Calories per gram: protein=4, carbs=4, fat=9, alcohol=7.
func AvgMacroBreakdown(logs []*DayLog) MacroBreakdown {
	if len(logs) == 0 {
		return MacroBreakdown{}
	}
	var protein, carbs, fat, alcohol float64
	for _, day := range logs {
		for _, e := range day.AllEntries() {
			nut := e.Nutrition()
			protein += nut.Protein
			carbs += nut.Carbs
			fat += nut.Fat
			alcohol += nut.Alcohol
		}
	}
	n := float64(len(logs))
	protein /= n
	carbs /= n
	fat /= n
	alcohol /= n

	proteinCals := protein * 4
	carbsCals := carbs * 4
	fatCals := fat * 9
	alcoholCals := alcohol * 7
	total := proteinCals + carbsCals + fatCals + alcoholCals
	if total == 0 {
		return MacroBreakdown{}
	}
	return MacroBreakdown{
		ProteinPct: proteinCals / total * 100,
		ProteinG:   protein,
		CarbsPct:   carbsCals / total * 100,
		CarbsG:     carbs,
		FatPct:     fatCals / total * 100,
		FatG:       fat,
		AlcoholPct: alcoholCals / total * 100,
		AlcoholG:   alcohol,
	}
}

// ComputeRangeSummary derives high-level statistics across all day logs.
func ComputeRangeSummary(logs []*DayLog) RangeSummary {
	s := RangeSummary{Days: len(logs)}
	if len(logs) == 0 {
		return s
	}
	var totalPts, totalTarget, totalCals float64
	for _, day := range logs {
		p := day.Points
		totalPts += p.DailyUsed
		totalTarget += p.DailyTarget
		if p.DailyTarget > 0 {
			if p.DailyUsed <= p.DailyTarget {
				s.DaysUnderBudget++
			} else {
				s.DaysOverBudget++
			}
		}
		for _, e := range day.AllEntries() {
			totalCals += e.Nutrition().Calories
			s.TotalItems++
		}
	}
	n := float64(len(logs))
	s.AvgDailyPts = totalPts / n
	s.AvgDailyTarget = totalTarget / n
	s.AvgDailyCals = totalCals / n
	return s
}
