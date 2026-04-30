package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

// FetchDay retrieves the food log and points summary for a single date (YYYY-MM-DD).
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
			PointsDetails struct {
				Weight                   float64 `json:"weight"`
				WeightUnit               string  `json:"weightUnit"`
				DailyPointTarget         float64 `json:"dailyPointTarget"`
				DailyPointsUsed          float64 `json:"dailyPointsUsed"`
				DailyPointsRemaining     float64 `json:"dailyPointsRemaining"`
				WeeklyPointAllowance     float64 `json:"weeklyPointAllowance"`
				WeeklyAllowanceRemaining float64 `json:"weeklyPointAllowanceRemaining"`
				WeeklyAllowanceUsed      float64 `json:"weeklyPointAllowanceUsed"`
				ActivityEarned           float64 `json:"dailyActivityPointsEarned"`
				ActivityRemaining        float64 `json:"dailyActivityPointsRemaining"`
				VeggieServings           float64 `json:"dailyVeggieServings"`
			} `json:"pointsDetails"`
			TrackedFoods Meals `json:"trackedFoods"`
		} `json:"today"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode day %s: %w", date, err)
	}

	pd := raw.Today.PointsDetails
	return &DayLog{
		Date:  date,
		Meals: raw.Today.TrackedFoods,
		Points: DayPoints{
			Date:                     date,
			DailyTarget:              pd.DailyPointTarget,
			DailyUsed:                pd.DailyPointsUsed,
			DailyRemaining:           pd.DailyPointsRemaining,
			WeeklyAllowance:          pd.WeeklyPointAllowance,
			WeeklyAllowanceRemaining: pd.WeeklyAllowanceRemaining,
			WeeklyAllowanceUsed:      pd.WeeklyAllowanceUsed,
			ActivityEarned:           pd.ActivityEarned,
			ActivityRemaining:        pd.ActivityRemaining,
			VeggieServings:           pd.VeggieServings,
			Weight:                   pd.Weight,
			WeightUnit:               strings.TrimSuffix(pd.WeightUnit, "s"),
		},
	}, nil
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

// FetchLatestVersion hits the GitHub Releases API and returns the latest
// published version tag (without the leading "v"), e.g. "1.2.3".
// Returns an empty string on any error so callers can treat it as optional.
func FetchLatestVersion() string {
	resp, err := http.Get("https://api.github.com/repos/ali5ter/wwlog/releases/latest")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var v struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return ""
	}
	return strings.TrimPrefix(v.TagName, "v")
}

// FetchDayRaw returns the raw JSON response body for a single date. Used for
// API inspection — run with --raw to see all available fields.
func (c *Client) FetchDayRaw(date string) ([]byte, error) {
	url := fmt.Sprintf(
		"https://cmx.weightwatchers.%s/api/v3/cmx/operations/composed/members/~/my-day/%s",
		c.tld, date,
	)
	resp, err := c.get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch raw day %s: %w", date, err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Client) get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	return c.client.Do(req)
}
