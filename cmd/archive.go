package cmd

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/ali5ter/wwlog/internal/api"
	"github.com/ali5ter/wwlog/internal/auth"
	"github.com/ali5ter/wwlog/internal/store"
	"golang.org/x/sync/errgroup"
)

func runArchive(authenticator *auth.Auth, tld string, s *store.Store) error {
	token, err := authenticator.Token()
	if err != nil {
		return fmt.Errorf("%w\nRun 'wwlog --login' to authenticate", err)
	}
	client := api.New(token, tld)

	end := time.Now().Format("2006-01-02")
	start := time.Now().AddDate(0, 0, -api.APIWindowDays).Format("2006-01-02")

	dates, err := api.DateRange(start, end)
	if err != nil {
		return err
	}
	total := len(dates)

	existing := make(map[string]bool, total)
	for _, d := range s.Dates() {
		existing[d] = true
	}

	fmt.Fprintf(os.Stderr, "Archiving %d days (%s → %s)\n", total, start, end)

	type result struct {
		isNew  bool
		updated bool
		missed  bool
	}
	results := make([]result, total)

	var fetched atomic.Int64

	const maxConcurrent = 10
	var g errgroup.Group
	sem := make(chan struct{}, maxConcurrent)

	for i, date := range dates {
		i, date := i, date
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			day, err := client.FetchDay(date)
			n := fetched.Add(1)
			fmt.Fprintf(os.Stderr, "\r  fetched %d / %d...", n, total)

			if err != nil {
				if errors.Is(err, api.ErrOutOfWindow) {
					results[i].missed = true
					return nil
				}
				results[i].missed = true
				return nil
			}
			if err := s.Save(day); err != nil {
				return fmt.Errorf("save %s: %w", date, err)
			}
			if existing[date] {
				results[i].updated = true
			} else {
				results[i].isNew = true
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Fprintln(os.Stderr)
		return err
	}
	fmt.Fprintln(os.Stderr)

	var newCount, updatedCount, missedCount int
	for _, r := range results {
		switch {
		case r.isNew:
			newCount++
		case r.updated:
			updatedCount++
		case r.missed:
			missedCount++
		}
	}

	stats := s.Stats()
	fmt.Fprintln(os.Stderr, "Archive complete.")
	if newCount > 0 {
		fmt.Fprintf(os.Stderr, "  New      %3d days\n", newCount)
	}
	if updatedCount > 0 {
		fmt.Fprintf(os.Stderr, "  Updated  %3d days\n", updatedCount)
	}
	if missedCount > 0 {
		fmt.Fprintf(os.Stderr, "  Missed   %3d days (outside API window or unavailable)\n", missedCount)
	}
	fmt.Fprintf(os.Stderr, "  Stored   %3d days total (%s → %s)\n",
		stats.DayCount, stats.FirstDate, stats.LastDate)
	if stats.GapCount > 0 {
		fmt.Fprintf(os.Stderr, "  Gaps     %3d\n", stats.GapCount)
	}
	return nil
}

func runStatus(s *store.Store) {
	stats := s.Stats()
	if stats.DayCount == 0 {
		fmt.Fprintf(os.Stderr, "Store     %s\n", stats.Dir)
		fmt.Fprintln(os.Stderr, "          (empty — run 'wwlog --archive' to populate)")
		return
	}
	gapStr := ""
	if stats.GapCount > 0 {
		gapStr = fmt.Sprintf(" · %d gap(s)", stats.GapCount)
	}
	fmt.Fprintf(os.Stderr, "Store     %s\n", stats.Dir)
	fmt.Fprintf(os.Stderr, "Coverage  %s → %s  (%d days%s)\n",
		stats.FirstDate, stats.LastDate, stats.DayCount, gapStr)

	if stats.GapCount == 0 {
		return
	}

	// List each gap range so the user knows exactly which dates are missing.
	const layout = "2006-01-02"
	dates := s.Dates()
	first := true
	for i := 1; i < len(dates); i++ {
		prev, _ := time.Parse(layout, dates[i-1])
		curr, _ := time.Parse(layout, dates[i])
		if curr.Sub(prev) <= 24*time.Hour {
			continue
		}
		gapStart := prev.AddDate(0, 0, 1)
		gapEnd := curr.AddDate(0, 0, -1)
		label := "Gaps"
		if !first {
			label = "    "
		}
		if gapStart.Equal(gapEnd) {
			fmt.Fprintf(os.Stderr, "%s      %s\n", label, gapStart.Format(layout))
		} else {
			fmt.Fprintf(os.Stderr, "%s      %s → %s\n", label, gapStart.Format(layout), gapEnd.Format(layout))
		}
		first = false
	}
}
