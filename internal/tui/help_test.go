package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/ali5ter/wwlog/internal/api"
)

func flattenBindings(cols [][]key.Binding) []key.Binding {
	var out []key.Binding
	for _, c := range cols {
		out = append(out, c...)
	}
	return out
}

func containsBinding(bindings []key.Binding, target key.Binding) bool {
	for _, b := range bindings {
		if b.Help() == target.Help() {
			return true
		}
	}
	return false
}

func TestHelpKeyMapShortHelp(t *testing.T) {
	got := helpKeyMap{activeTab: tabLog}.ShortHelp()
	want := []key.Binding{keys.DateRange, keys.Export, keys.Help}
	if len(got) != len(want) {
		t.Fatalf("ShortHelp() returned %d bindings, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Help() != want[i].Help() {
			t.Errorf("ShortHelp()[%d] = %+v, want %+v", i, got[i].Help(), want[i].Help())
		}
	}
}

func TestHelpKeyMapFullHelp(t *testing.T) {
	tests := []struct {
		name          string
		tab           tab
		wantHasFilter bool
		wantHasSort   bool
		wantHasFocus  bool
	}{
		{"log", tabLog, true, true, true},
		{"nutrition", tabNutrition, true, false, true},
		{"insights", tabInsights, false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flat := flattenBindings(helpKeyMap{activeTab: tt.tab}.FullHelp())
			if has := containsBinding(flat, keys.Filter); has != tt.wantHasFilter {
				t.Errorf("Filter present = %v, want %v", has, tt.wantHasFilter)
			}
			if has := containsBinding(flat, keys.Sort); has != tt.wantHasSort {
				t.Errorf("Sort present = %v, want %v", has, tt.wantHasSort)
			}
			if has := containsBinding(flat, keys.FocusPrev); has != tt.wantHasFocus {
				t.Errorf("FocusPrev present = %v, want %v", has, tt.wantHasFocus)
			}
			// Navigation and quit/help/range/export are universal, regardless of tab.
			for _, b := range []key.Binding{keys.Up, keys.Down, keys.GotoTop, keys.GotoBottom, keys.Quit, keys.Help, keys.DateRange, keys.Export} {
				if !containsBinding(flat, b) {
					t.Errorf("expected %q to always be present in FullHelp()", b.Help().Key)
				}
			}
		})
	}
}

// TestHeaderTabsOffsetMatchesTabAtPoint guards against the render and the
// click hit-test drifting apart — the same class of bug v1.14.0 fixed in
// dateRowAtPoint (see panes.go).
func TestHeaderTabsOffsetMatchesTabAtPoint(t *testing.T) {
	m := Model{width: 100, activeTab: tabLog}
	offset := headerTabsOffset()

	if _, ok := m.tabAtPoint(offset-1, 0); ok {
		t.Errorf("tabAtPoint(%d,0) should still be on the badge/gap, not a tab", offset-1)
	}
	if idx, ok := m.tabAtPoint(offset, 0); !ok || idx != 0 {
		t.Errorf("tabAtPoint(%d,0) = (%d,%v), want (0,true) — first tab starts at headerTabsOffset()", offset, idx, ok)
	}
}

// newTestModel returns a minimal Model on the Log screen with real data
// loaded, sufficient to exercise Update's key-handling paths.
func newTestModel() Model {
	logs := []*api.DayLog{manyEntriesLog("2026-06-10")}
	loc := newLocale("com", "")
	return Model{
		screen:   screenLog,
		width:    100,
		height:   30,
		logs:     logs,
		logModel: newLogModel(logs, 100, 24, loc),
	}
}

func TestHelpToggle(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	m = updated.(Model)
	if !m.helpOpen {
		t.Fatal("\"?\" should open the help panel")
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(Model)
	if m.helpOpen {
		t.Error("esc should close the help panel")
	}
}

func TestHelpToggleSuppressedWhileFiltering(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(Model)
	if !m.logModel.filtering {
		t.Fatal("setup: \"/\" should enter filtering mode")
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	m = updated.(Model)
	if m.helpOpen {
		t.Error("\"?\" while filtering should be a literal character, not open help")
	}
}

func TestHelpOpenSwallowsOtherKeys(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m = updated.(Model)
	if m.dialog != dialogNone {
		t.Error("keys other than ?/esc/q should be swallowed while help is open")
	}
	if !m.helpOpen {
		t.Error("help should remain open until ?/esc")
	}
}

func TestLogModelGotoTopBottomGatedByFocus(t *testing.T) {
	logs := []*api.DayLog{manyEntriesLog("2026-06-10")}
	loc := newLocale("com", "")
	m := newLogModel(logs, 100, 12, loc)

	m, _ = m.update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if m.detail.YOffset() != 0 {
		t.Error("G while list-focused belongs to the date list, not the detail viewport")
	}

	m, _ = m.update(tea.KeyPressMsg{Code: tea.KeyRight})
	if !m.detailFocused {
		t.Fatal("setup: expected detail focus")
	}

	m, _ = m.update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if m.detail.YOffset() == 0 {
		t.Error("G while detail-focused should jump to the bottom of the detail pane")
	}

	m, _ = m.update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.detail.YOffset() != 0 {
		t.Error("g while detail-focused should jump to the top of the detail pane")
	}
}
