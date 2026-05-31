package api

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
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
// When offline is true, no API calls are made — all dates are served from the
// local store. client may be nil in this case.
//
// When a date fails with a non-window API error (network failure, server error),
// the store is consulted as a fallback and a notice is emitted rather than
// aborting. This allows the TUI and pipeline to serve archived data when the
// API is temporarily unavailable.
//
// When ds is nil the behaviour is identical to the pre-store implementation:
// start is clamped to the window and a notice is emitted, or an error is
// returned if end is also outside the window.
//
// Notices are user-visible info messages — print to stderr in pipeline mode
// or surface in the TUI status bar.
func LoadRange(client *Client, ds DayStore, start, end string, offline bool) ([]*DayLog, []string, error) {
	const layout = "2006-01-02"
	today, _ := time.Parse(layout, time.Now().Format(layout))
	earliest := today.AddDate(0, 0, -APIWindowDays)

	allDates, err := DateRange(start, end)
	if err != nil {
		return nil, nil, err
	}

	// Partition dates into those the API can serve and those only the store can.
	// In offline mode all dates go to storeDates regardless of the window.
	var apiDates, storeDates []string
	for _, d := range allDates {
		t, _ := time.Parse(layout, d)
		if offline || t.Before(earliest) {
			storeDates = append(storeDates, d)
		} else {
			apiDates = append(apiDates, d)
		}
	}

	var logs []*DayLog
	var notices []string

	// Pre-window or offline dates — store only.
	if len(storeDates) > 0 {
		if ds == nil && !offline {
			notices = append(notices, fmt.Sprintf(
				"--start clamped to %s (WW retains ~90 days)", earliest.Format(layout),
			))
		} else if ds == nil && offline {
			return nil, nil, fmt.Errorf("--offline requires a local store; check store_dir in config")
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
			if offline && hits == 0 && misses > 0 {
				return nil, nil, fmt.Errorf(
					"local archive has no data for the requested range — run 'wwlog --archive' to populate it",
				)
			}
			if offline && hits > 0 {
				notices = append(notices, fmt.Sprintf("offline — %d day(s) loaded from local archive", hits))
			} else if !offline && hits > 0 {
				notices = append(notices, fmt.Sprintf("%d day(s) loaded from local store", hits))
			}
			if misses > 0 {
				if offline {
					notices = append(notices, fmt.Sprintf(
						"%d day(s) not in local archive", misses,
					))
				} else {
					notices = append(notices, fmt.Sprintf(
						"%d day(s) outside retention window and not in local store", misses,
					))
				}
			}
		}
	}

	// If the entire requested range is offline or outside the window:
	if len(apiDates) == 0 {
		if ds == nil && !offline {
			return nil, nil, fmt.Errorf(
				"end date %s is older than WW's ~90-day retention window (earliest queryable: %s)",
				end, earliest.Format(layout),
			)
		}
		return logs, notices, nil
	}

	// In-window dates — fetch from API concurrently (bounded pool), save to store.
	type fetchResult struct {
		day      *DayLog
		skipped  bool
		apiError bool
	}
	results := make([]fetchResult, len(apiDates))

	const maxConcurrent = 10
	var g errgroup.Group
	sem := make(chan struct{}, maxConcurrent)

	for i, date := range apiDates {
		i, date := i, date
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			day, err := client.FetchDay(date)
			if err != nil {
				if errors.Is(err, ErrOutOfWindow) {
					results[i].skipped = true
					if ds != nil {
						if d, ok := ds.Load(date); ok {
							results[i].day = d
						}
					}
					return nil
				}
				// Network or server error — fall back to store rather than aborting.
				results[i].apiError = true
				if ds != nil {
					if d, ok := ds.Load(date); ok {
						results[i].day = d
					}
				}
				return nil
			}
			results[i].day = day
			if ds != nil {
				_ = ds.Save(day)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, notices, err
	}

	var skipped, apiErrors int
	for _, r := range results {
		if r.skipped {
			skipped++
		}
		if r.apiError {
			apiErrors++
		}
		if r.day != nil {
			logs = append(logs, r.day)
		}
	}
	if skipped > 0 {
		notices = append(notices, fmt.Sprintf(
			"skipped %d day(s) outside WW retention window", skipped,
		))
	}
	if apiErrors > 0 {
		notices = append(notices, fmt.Sprintf(
			"API unavailable for %d day(s) — showing archived data where available", apiErrors,
		))
	}

	return logs, notices, nil
}
