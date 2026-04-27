package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client is an authenticated WW API client.
type Client struct {
	token  string
	tld    string
	client *http.Client
}

// New creates an authenticated API client.
func New(token, tld string) *Client {
	return &Client{
		token:  token,
		tld:    tld,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchDay retrieves the food log for a single date (YYYY-MM-DD).
func (c *Client) FetchDay(date string) (*DayLog, error) {
	url := fmt.Sprintf(
		"https://cmx.weightwatchers.%s/api/v3/cmx/operations/composed/members/~/my-day/%s",
		c.tld, date,
	)

	resp, err := c.get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch day %s: %w", date, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch day %s: server returned %d", date, resp.StatusCode)
	}

	var raw struct {
		Today struct {
			TrackedFoods Meals `json:"trackedFoods"`
		} `json:"today"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode day %s: %w", date, err)
	}

	return &DayLog{Date: date, Meals: raw.Today.TrackedFoods}, nil
}

// FetchNutrition retrieves and calculates nutritional data for a food entry.
// Quick-add entries (MEMBERFOODQUICK) return nil.
func (c *Client) FetchNutrition(entry *FoodEntry) (*NutritionData, error) {
	if entry.SourceType == SourceMemberFoodQuick {
		return nil, nil
	}

	prefixes := map[string]string{
		SourceWWFood:       fmt.Sprintf("https://cmx.weightwatchers.%s/api/v3/public/foods/", c.tld),
		SourceMemberFood:   fmt.Sprintf("https://cmx.weightwatchers.%s/api/v3/cmx/members/~/custom-foods/foods/", c.tld),
		SourceWWVendorFood: fmt.Sprintf("https://cmx.weightwatchers.%s/api/v3/public/foods/", c.tld),
		SourceMemberRecipe: fmt.Sprintf("https://cmx.weightwatchers.%s/api/v3/cmx/members/~/custom-foods/recipes/", c.tld),
		SourceWWRecipe:     fmt.Sprintf("https://cmx.weightwatchers.%s/api/v3/public/recipes/", c.tld),
	}

	prefix, ok := prefixes[entry.SourceType]
	if !ok {
		return nil, fmt.Errorf("unknown source type: %s", entry.SourceType)
	}

	data := &NutritionData{
		Name:        entry.Name,
		PortionName: entry.PortionName,
		PortionSize: entry.PortionSize,
		TrackedDate: entry.TrackedDate,
		TimeOfDay:   entry.TimeOfDay,
	}

	isRecipe := entry.SourceType == SourceMemberRecipe || entry.SourceType == SourceWWRecipe
	url := fmt.Sprintf("%s%s?fullDetails=true", prefix, entry.ID)

	resp, err := c.get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch nutrition for %s: %w", entry.Name, err)
	}
	defer resp.Body.Close()

	if isRecipe {
		var recipe struct {
			ServingSize float64 `json:"servingSize"`
			Ingredients []struct {
				Quantity   float64 `json:"quantity"`
				ItemDetail struct {
					Portions []struct {
						Size      float64            `json:"size"`
						Nutrition map[string]float64 `json:"nutrition"`
					} `json:"portions"`
				} `json:"itemDetail"`
			} `json:"ingredients"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&recipe); err != nil {
			return nil, fmt.Errorf("decode recipe nutrition: %w", err)
		}
		data.PortionName = "serving(s)"
		for _, ing := range recipe.Ingredients {
			if len(ing.ItemDetail.Portions) == 0 || recipe.ServingSize == 0 {
				continue
			}
			p := ing.ItemDetail.Portions[0]
			scale := ing.Quantity / recipe.ServingSize * entry.PortionSize / p.Size
			data.Calories += p.Nutrition["calories"] * scale
			data.Fat += p.Nutrition["fat"] * scale
			data.SaturatedFat += p.Nutrition["saturatedFat"] * scale
			data.Sodium += p.Nutrition["sodium"] * scale
			data.Carbs += p.Nutrition["carbs"] * scale
			data.Fiber += p.Nutrition["fiber"] * scale
			data.Sugar += p.Nutrition["sugar"] * scale
			data.AddedSugar += p.Nutrition["addedSugar"] * scale
			data.Protein += p.Nutrition["protein"] * scale
		}
	} else {
		var portions []struct {
			Name      string             `json:"name"`
			Size      float64            `json:"size"`
			Nutrition map[string]float64 `json:"nutrition"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&portions); err != nil {
			return nil, fmt.Errorf("decode nutrition: %w", err)
		}
		for _, p := range portions {
			if p.Name != entry.PortionName || p.Size == 0 {
				continue
			}
			scale := entry.PortionSize / p.Size
			data.Calories = p.Nutrition["calories"] * scale
			data.Fat = p.Nutrition["fat"] * scale
			data.SaturatedFat = p.Nutrition["saturatedFat"] * scale
			data.Sodium = p.Nutrition["sodium"] * scale
			data.Carbs = p.Nutrition["carbs"] * scale
			data.Fiber = p.Nutrition["fiber"] * scale
			data.Sugar = p.Nutrition["sugar"] * scale
			data.AddedSugar = p.Nutrition["addedSugar"] * scale
			data.Protein = p.Nutrition["protein"] * scale
			break
		}
	}

	return data, nil
}

// DateRange returns a slice of YYYY-MM-DD strings from start to end inclusive.
func DateRange(start, end string) ([]string, error) {
	const layout = "2006-01-02"
	s, err := time.Parse(layout, start)
	if err != nil {
		return nil, fmt.Errorf("invalid start date %q: %w", start, err)
	}
	e, err := time.Parse(layout, end)
	if err != nil {
		return nil, fmt.Errorf("invalid end date %q: %w", end, err)
	}
	if s.After(e) {
		return nil, fmt.Errorf("start date %s is after end date %s", start, end)
	}
	var dates []string
	for d := s; !d.After(e); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format(layout))
	}
	return dates, nil
}

func (c *Client) get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	return c.client.Do(req)
}
