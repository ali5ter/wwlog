// Package api contains the WW API client and data types.
package api

import "encoding/json"

// DefaultPortion holds the canonical portion definition embedded in each tracked food entry.
type DefaultPortion struct {
	Name      string         `json:"name"`
	Size      float64        `json:"size"`
	Nutrition NutritionMap   `json:"nutrition"`
	Points    float64        `json:"points"`
}

// NutritionMap is a mixed-type map from the WW API — most values are floats but
// some keys (e.g. "isEstimatedSugar") are booleans. Only numeric values are kept.
type NutritionMap map[string]float64

func (n *NutritionMap) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m := make(NutritionMap, len(raw))
	for k, v := range raw {
		switch val := v.(type) {
		case float64:
			m[k] = val
		}
	}
	*n = m
	return nil
}

// FoodEntry represents a single tracked food item.
type FoodEntry struct {
	ID             string         `json:"_id"`
	EntryID        string         `json:"entryId"`
	Name           string         `json:"name"`
	PortionName    string         `json:"portionName"`
	PortionSize    float64        `json:"portionSize"`
	SourceType     string         `json:"sourceType"`
	TrackedDate    string         `json:"trackedDate"`
	TimeOfDay      string         `json:"timeOfDay"`
	ServingDesc    string         `json:"_servingDesc"`
	DefaultPortion DefaultPortion `json:"defaultPortion"`
	// PointsPrecise is the actual points charged to the budget for this entry.
	// For ZeroPoint foods this is 0. Use this for all points calculations.
	PointsPrecise float64 `json:"pointsPrecise"`
	IsZPF         bool    `json:"isZPF"`
	// PointsInfo.MaxPoints is the pre-ZPF calculated value. Only useful when
	// you want to show "saved X points" for a ZPF food.
	PointsInfo struct {
		MaxPoints float64 `json:"maxPoints"`
	} `json:"pointsInfo"`
}

// Nutrition computes scaled nutritional values for the tracked portion size.
func (e FoodEntry) Nutrition() NutritionData {
	p := e.DefaultPortion
	scale := 1.0
	if p.Size > 0 && e.PortionSize > 0 {
		scale = e.PortionSize / p.Size
	}
	cal := p.Nutrition["calories"]
	if cal == 0 {
		// Some WW API entries omit "calories"; derive from macros (Atwater factors).
		cal = 4*p.Nutrition["protein"] + 4*p.Nutrition["carbs"] + 9*p.Nutrition["fat"] + 7*p.Nutrition["alcohol"]
	}
	return NutritionData{
		Name:         e.Name,
		PortionName:  e.PortionName,
		PortionSize:  e.PortionSize,
		TrackedDate:  e.TrackedDate,
		TimeOfDay:    e.TimeOfDay,
		Calories:     cal * scale,
		Fat:          p.Nutrition["fat"] * scale,
		SaturatedFat: p.Nutrition["saturatedFat"] * scale,
		Sodium:       p.Nutrition["sodium"] * scale,
		Carbs:        p.Nutrition["carbs"] * scale,
		Fiber:        p.Nutrition["fiber"] * scale,
		Sugar:        p.Nutrition["sugar"] * scale,
		AddedSugar:   p.Nutrition["addedSugar"] * scale,
		Protein:      p.Nutrition["protein"] * scale,
		Alcohol:      p.Nutrition["alcohol"] * scale,
	}
}

// Meals groups food entries by meal period.
type Meals struct {
	Morning []FoodEntry `json:"morning"`
	Midday  []FoodEntry `json:"midday"`
	Evening []FoodEntry `json:"evening"`
	Anytime []FoodEntry `json:"anytime"`
}

// DayPoints holds the WW points summary for a single day.
type DayPoints struct {
	Date                     string
	DailyTarget              float64
	DailyUsed                float64
	DailyRemaining           float64
	WeeklyAllowance          float64
	WeeklyAllowanceRemaining float64
	WeeklyAllowanceUsed      float64
	ActivityEarned           float64
	ActivityRemaining        float64
	VeggieServings           float64
	Weight                   float64
	WeightUnit               string
}

// DayLog is the food log for a single date.
type DayLog struct {
	Date   string
	Meals  Meals
	Points DayPoints
}

// AllEntries returns all food entries across all meal periods.
func (d *DayLog) AllEntries() []FoodEntry {
	var entries []FoodEntry
	entries = append(entries, d.Meals.Morning...)
	entries = append(entries, d.Meals.Midday...)
	entries = append(entries, d.Meals.Evening...)
	entries = append(entries, d.Meals.Anytime...)
	return entries
}

// DayNutrition holds aggregated nutritional totals for a single day.
type DayNutrition struct {
	Date         string
	Calories     float64
	Fat          float64
	SaturatedFat float64
	Sodium       float64
	Carbs        float64
	Fiber        float64
	Sugar        float64
	Protein      float64
	Alcohol      float64
	ItemCount    int
}

// NutritionData holds calculated nutritional values for a single food entry.
type NutritionData struct {
	Name         string
	PortionName  string
	PortionSize  float64
	TrackedDate  string
	TimeOfDay    string
	Calories     float64
	Fat          float64
	SaturatedFat float64
	Sodium       float64
	Carbs        float64
	Fiber        float64
	Sugar        float64
	AddedSugar   float64
	Protein      float64
	Alcohol      float64
}

// SourceType constants for food entry source types.
const (
	SourceWWFood          = "WWFOOD"
	SourceMemberFood      = "MEMBERFOOD"
	SourceWWVendorFood    = "WWVENDORFOOD"
	SourceMemberRecipe    = "MEMBERRECIPE"
	SourceWWRecipe        = "WWRECIPE"
	SourceMemberFoodQuick = "MEMBERFOODQUICK"
)

// ComputeAllNutrition derives aggregated nutrition from the embedded portion data in each log.
// No extra API calls required — all data comes from the my-day response.
func ComputeAllNutrition(logs []*DayLog) map[string]*DayNutrition {
	result := make(map[string]*DayNutrition, len(logs))
	for _, day := range logs {
		dn := &DayNutrition{Date: day.Date}
		for _, e := range day.AllEntries() {
			if e.DefaultPortion.Size == 0 {
				continue
			}
			n := e.Nutrition()
			dn.Calories += n.Calories
			dn.Fat += n.Fat
			dn.SaturatedFat += n.SaturatedFat
			dn.Sodium += n.Sodium
			dn.Carbs += n.Carbs
			dn.Fiber += n.Fiber
			dn.Sugar += n.Sugar
			dn.Protein += n.Protein
			dn.Alcohol += n.Alcohol
			dn.ItemCount++
		}
		result[day.Date] = dn
	}
	return result
}
