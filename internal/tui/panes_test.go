package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/ali5ter/wwlog/internal/api"
)

func TestPaneContentWidth(t *testing.T) {
	tests := []struct {
		outer int
		want  int
	}{
		{outer: 40, want: 36},
		{outer: paneFrameWidth, want: 0},
		{outer: 0, want: 0},
		{outer: -5, want: 0},
	}
	for _, tt := range tests {
		if got := paneContentWidth(tt.outer); got != tt.want {
			t.Errorf("paneContentWidth(%d) = %d, want %d", tt.outer, got, tt.want)
		}
	}
}

func TestPaneContentHeight(t *testing.T) {
	tests := []struct {
		outer int
		want  int
	}{
		{outer: 30, want: 28},
		{outer: paneFrameHeight, want: 0},
		{outer: 0, want: 0},
		{outer: -5, want: 0},
	}
	for _, tt := range tests {
		if got := paneContentHeight(tt.outer); got != tt.want {
			t.Errorf("paneContentHeight(%d) = %d, want %d", tt.outer, got, tt.want)
		}
	}
}

func TestDateRowAtPoint(t *testing.T) {
	const totalWidth = 90 // listPaneWidth = 30
	const rowsTop = 5     // headerRows(2) + paneBorderRows(1) + paneFilterRows(2)
	const rowStride = 3

	tests := []struct {
		name      string
		x, y      int
		listPage  int
		perPage   int
		itemCount int
		wantIdx   int
		wantOK    bool
	}{
		{"above rowsTop", 5, rowsTop - 1, 0, 10, 20, 0, false},
		{"first row", 5, rowsTop, 0, 10, 20, 0, true},
		{"second row via stride", 5, rowsTop + rowStride, 0, 10, 20, 1, true},
		{"second page first row", 5, rowsTop, 1, 5, 20, 5, true},
		{"x negative outside list pane", -1, rowsTop, 0, 10, 20, 0, false},
		{"x at list pane edge (excluded)", totalWidth / 3, rowsTop, 0, 10, 20, 0, false},
		{"idx beyond itemCount", 5, rowsTop + 50*rowStride, 0, 10, 3, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, ok := dateRowAtPoint(tt.x, tt.y, totalWidth, tt.listPage, tt.perPage, tt.itemCount)
			if ok != tt.wantOK || (ok && idx != tt.wantIdx) {
				t.Errorf("dateRowAtPoint(%d,%d,...) = (%d,%v), want (%d,%v)", tt.x, tt.y, idx, ok, tt.wantIdx, tt.wantOK)
			}
		})
	}
}

// manyEntriesLog builds a DayLog with enough Morning entries that renderDay's
// output overflows a small viewport, so ScrollUp/ScrollDown have visible
// effect in tests below.
func manyEntriesLog(date string) *api.DayLog {
	entries := make([]api.FoodEntry, 20)
	for i := range entries {
		entries[i] = api.FoodEntry{Name: fmt.Sprintf("Test Item %d", i)}
	}
	return &api.DayLog{Date: date, Meals: api.Meals{Morning: entries}}
}

func TestLogModelFocusAndScroll(t *testing.T) {
	logs := []*api.DayLog{manyEntriesLog("2026-06-10"), manyEntriesLog("2026-06-11")}
	loc := newLocale("com", "")
	m := newLogModel(logs, 100, 12, loc)

	if m.detailFocused {
		t.Fatal("detailFocused should start false")
	}

	m, _ = m.update(tea.KeyPressMsg{Code: tea.KeyRight})
	if !m.detailFocused {
		t.Fatal("FocusNext (right) should set detailFocused")
	}

	startOffset := m.detail.YOffset()
	startIdx := m.list.Index()
	m, _ = m.update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.detail.YOffset() == startOffset {
		t.Error("Down while detail-focused should scroll the detail pane")
	}
	if m.list.Index() != startIdx {
		t.Error("Down while detail-focused should not move the date selection")
	}

	m, _ = m.update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.detailFocused {
		t.Fatal("FocusPrev (left) should clear detailFocused")
	}

	m, _ = m.update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.list.Index() != startIdx+1 {
		t.Errorf("Down while list-focused should navigate dates, list.Index() = %d, want %d", m.list.Index(), startIdx+1)
	}
}

func TestLogModelFilterClearsDetailFocus(t *testing.T) {
	logs := []*api.DayLog{manyEntriesLog("2026-06-10")}
	loc := newLocale("com", "")
	m := newLogModel(logs, 100, 12, loc)

	m, _ = m.update(tea.KeyPressMsg{Code: tea.KeyRight})
	if !m.detailFocused {
		t.Fatal("setup: detailFocused should be true before filtering")
	}

	m, _ = m.update(tea.KeyPressMsg{Code: '/', Text: "/"})
	if !m.filtering {
		t.Fatal("\"/\" should enter filtering mode")
	}
	if m.detailFocused {
		t.Error("entering filter mode should clear detailFocused")
	}
}
