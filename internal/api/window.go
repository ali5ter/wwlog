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

// ClampToWindow checks start and end against the WW my-day endpoint's
// retention window and returns a usable range. If start is older than the
// window, it is clamped forward and a user-visible notice is returned. If
// end is also older than the window, no fetchable data exists and an error
// is returned.
func ClampToWindow(start, end string) (clampedStart, clampedEnd, notice string, err error) {
	const layout = "2006-01-02"
	// Use the local calendar date — WW's retention boundary is measured in
	// the user's account locale, not UTC. Parsing the formatted date back
	// gives a date-only time at UTC midnight that arithmetic is safe on.
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

// LoadRange fetches every date in [start, end] from the WW API. The range
// is first clamped to the retention window via ClampToWindow. Per-day 400
// responses (ErrOutOfWindow) are skipped and aggregated into a single
// summary notice. Other errors abort.
//
// Notices are user-visible info messages — print to stderr in pipeline
// mode or surface in the TUI status bar.
func LoadRange(client *Client, start, end string) (logs []*DayLog, notices []string, err error) {
	cs, ce, clampNotice, err := ClampToWindow(start, end)
	if err != nil {
		return nil, nil, err
	}
	if clampNotice != "" {
		notices = append(notices, clampNotice)
	}

	dates, err := DateRange(cs, ce)
	if err != nil {
		return nil, notices, err
	}
	var skipped int
	for _, date := range dates {
		day, err := client.FetchDay(date)
		if err != nil {
			if errors.Is(err, ErrOutOfWindow) {
				skipped++
				continue
			}
			return nil, notices, fmt.Errorf("fetch %s: %w", date, err)
		}
		logs = append(logs, day)
	}
	if skipped > 0 {
		notices = append(notices, fmt.Sprintf(
			"skipped %d day(s) outside WW retention window", skipped,
		))
	}
	return logs, notices, nil
}
