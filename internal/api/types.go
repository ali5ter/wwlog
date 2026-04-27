// Package api contains the WW API client and data types.
package api

// FoodEntry represents a single tracked food item.
type FoodEntry struct {
	ID          string  `json:"_id"`
	EntryID     string  `json:"entryId"`
	Name        string  `json:"name"`
	PortionName string  `json:"portionName"`
	PortionSize float64 `json:"portionSize"`
	SourceType  string  `json:"sourceType"`
	TrackedDate string  `json:"trackedDate"`
	TimeOfDay   string  `json:"timeOfDay"`
}

// Meals groups food entries by meal period.
type Meals struct {
	Morning []FoodEntry `json:"morning"`
	Midday  []FoodEntry `json:"midday"`
	Evening []FoodEntry `json:"evening"`
	Anytime []FoodEntry `json:"anytime"`
}

// DayLog is the food log for a single date.
type DayLog struct {
	Date  string
	Meals Meals
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
}

// SourceType constants for food entry source types.
const (
	SourceWWFood         = "WWFOOD"
	SourceMemberFood     = "MEMBERFOOD"
	SourceWWVendorFood   = "WWVENDORFOOD"
	SourceMemberRecipe   = "MEMBERRECIPE"
	SourceWWRecipe       = "WWRECIPE"
	SourceMemberFoodQuick = "MEMBERFOODQUICK"
)
