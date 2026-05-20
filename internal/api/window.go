package api

import (
	"errors"
	"fmt"
	"time"
)

// APIWindowDays is the WW my-day endpoint's hard backwards retention window
// in days. Verified empirically: 89 days back from today still returns 200,
// 90 days back returns 400. Older accounts hit the same wall — it's a
// server-side policy, not a per-account limit.
const APIWindowDays = 89

// DayStore is an optional local persistence layer for day logs.
// A nil DayStore disables persistence; LoadRange falls back to API-only
// behaviour identical to the pre-store implementation.
type DayStore interface {
	Load(date string) (*DayLog, bool)
	Save(day *DayLog) error
}

// ClampToWindow checks start and end against the WW my-day endpoint's
// retention window and returns a usable range. If start is older than the
// window, it is clamped forward and a user-visible notice is returned. If
// end is also older than the window, no fetchable data exists and an error
// is returned. Callers that maintain a local DayStore should use LoadRange
// directly instead, which handles pre-window dates transparently.
func ClampToWindow(start, end string) (clampedStart, clampedEnd, notice string, err error) {
	const layout = "2006-01-02"
	today, _ := time.Parse(layout, time.Now().Format(layout))
	earliest := today.AddDate(0, 0, -APIWindowDays)

	s, err := time.Parse(layout, start)
	if err != nil {
		return start, end, "", fmt.Errorf("invalid start date %q: %w", start, err)
	}
	e, err := time.Parse(layout, end)
	if err != nil {
		return start, end, "", fmt.Errorf("invalid end date %q: %w", end, err)
	}
	if e.Before(earliest) {
		return start, end, "", fmt.Errorf(
			"end date %s is older than WW's ~90-day retention window (earliest queryable: %s)",
			end, earliest.Format(layout),
		)
	}
	if s.Before(earliest) {
		earliestStr := earliest.Format(layout)
		return earliestStr, end,
			fmt.Sprintf("--start clamped to %s (WW retains ~90 days)", earliestStr), nil
	}
	return start, end, "", nil
}

// LoadRange fetches every date in [start, end] from the WW API and, when ds
// is non-nil, from the local store.
//
// Dates within the API retention window are fetched from the API and saved to
// the store (best-effort). Dates older than the window are served from the
// store only. Per-day 400 responses (ErrOutOfWindow) are skipped; the store
// is consulted as a fallback for those dates too.
//
// When ds is nil the behaviour is identical to the pre-store implementation:
// start is clamped to the window and a notice is emitted, or an error is
// returned if end is also outside the window.
//
// Notices are user-visible info messages — print to stderr in pipeline mode
// or surface in the TUI status bar.
func LoadRange(client *Client, ds DayStore, start, end string) ([]*DayLog, []string, error) {
	const layout = "2006-01-02"
	today, _ := time.Parse(layout, time.Now().Format(layout))
	earliest := today.AddDate(0, 0, -APIWindowDays)

	allDates, err := DateRange(start, end)
	if err != nil {
		return nil, nil, err
	}

	// Partition dates into those the API can serve and those only the store can.
	var apiDates, storeDates []string
	for _, d := range allDates {
		t, _ := time.Parse(layout, d)
		if t.Before(earliest) {
			storeDates = append(storeDates, d)
		} else {
			apiDates = append(apiDates, d)
		}
	}

	var logs []*DayLog
	var notices []string

	// Pre-window dates — store only.
	if len(storeDates) > 0 {
		if ds == nil {
			notices = append(notices, fmt.Sprintf(
				"--start clamped to %s (WW retains ~90 days)", earliest.Format(layout),
			))
		} else {
			var hits, misses int
			for _, date := range storeDates {
				if day, ok := ds.Load(date); ok {
					logs = append(logs, day)
					hits++
				} else {
					misses++
				}
			}
			if hits > 0 {
				notices = append(notices, fmt.Sprintf("%d day(s) loaded from local store", hits))
			}
			if misses > 0 {
				notices = append(notices, fmt.Sprintf(
					"%d day(s) outside retention window and not in local store", misses,
				))
			}
		}
	}

	// If the entire requested range is outside the window:
	if len(apiDates) == 0 {
		if ds == nil {
			return nil, nil, fmt.Errorf(
				"end date %s is older than WW's ~90-day retention window (earliest queryable: %s)",
				end, earliest.Format(layout),
			)
		}
		return logs, notices, nil
	}

	// In-window dates — fetch from API, save to store.
	var skipped int
	for _, date := range apiDates {
		day, err := client.FetchDay(date)
		if err != nil {
			if errors.Is(err, ErrOutOfWindow) {
				skipped++
				if ds != nil {
					if d, ok := ds.Load(date); ok {
						logs = append(logs, d)
					}
				}
				continue
			}
			return nil, notices, fmt.Errorf("fetch %s: %w", date, err)
		}
		logs = append(logs, day)
		if ds != nil {
			_ = ds.Save(day)
		}
	}
	if skipped > 0 {
		notices = append(notices, fmt.Sprintf(
			"skipped %d day(s) outside WW retention window", skipped,
		))
	}

	return logs, notices, nil
}
