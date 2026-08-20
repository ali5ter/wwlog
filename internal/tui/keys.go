package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	// Up/Down navigate the date list by default. When a tab's detail pane
	// has focus (see FocusNext/keys.FocusPrev, logModel.detailFocused /
	// nutriModel.detailFocused), they scroll that pane instead — the same
	// behavior ScrollUp/ScrollDown always provide, just reachable without
	// leaving list focus. Both are still bubbles' own list.DefaultKeyMap
	// bindings underneath; nothing here rebinds list navigation itself.
	Up           key.Binding
	Down         key.Binding
	GotoTop      key.Binding // "g"/"home" — matches bubbles/list's own default binding
	GotoBottom   key.Binding // "G"/"end"  — matches bubbles/list's own default binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	FocusPrev    key.Binding // "left"  — focus the date list
	FocusNext    key.Binding // "right" — focus the detail pane
	ScrollUp     key.Binding
	ScrollDown   key.Binding
	Filter       key.Binding
	Export       key.Binding
	DateRange    key.Binding
	Sort         key.Binding
	TabNext      key.Binding
	TabPrev      key.Binding
	Help         key.Binding
	Quit         key.Binding
}

var keys = keyMap{
	Up:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	// The date list already jumps top/bottom on these keys for free — it's
	// bubbles/list's own DefaultKeyMap.GoToStart/GoToEnd, never overridden
	// by newDateList. These bindings exist so the detail viewport (which has
	// no such default) can match the same keys, and so the help panel can
	// document behavior that previously had no visible binding at all.
	GotoTop:      key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g/home", "go to top")),
	GotoBottom:   key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G/end", "go to bottom")),
	PageUp:       key.NewBinding(key.WithKeys("b", "pgup"), key.WithHelp("b/pgup", "page up")),
	PageDown:     key.NewBinding(key.WithKeys("f", "pgdown"), key.WithHelp("f/pgdn", "page down")),
	HalfPageUp:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "½ page up")),
	HalfPageDown: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "½ page down")),
	FocusPrev:    key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "focus list")),
	FocusNext:    key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "focus detail")),
	// Secondary affordance: scroll the detail pane without moving focus off
	// the list — kept even though Up/Down do this too once the detail pane
	// has focus. Also the only way Insights (no list, no focus concept)
	// scrolls in 3-line steps rather than the viewport's own line-at-a-time
	// default; see insights.go.
	ScrollUp:   key.NewBinding(key.WithKeys("shift+up"), key.WithHelp("⇧↑", "scroll detail")),
	ScrollDown: key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("⇧↓", "scroll detail")),
	Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	Export:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export")),
	DateRange:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "range")),
	Sort:       key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
	TabNext:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
	TabPrev:    key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("⇧tab", "prev tab")),
	Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// helpKeyMap adapts keys to bubbles/help's KeyMap interface, tailoring the
// full help view to what's actually usable on the active tab — Insights has
// no list pane, so no pane-focus, filter, or sort bindings apply there.
type helpKeyMap struct {
	activeTab tab
}

// ShortHelp returns the footer's permanently pinned bindings.
func (h helpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{keys.DateRange, keys.Export, keys.Help}
}

// FullHelp returns the full help panel's columns: navigation, pane/tab
// controls, and actions. Columns are built per tab rather than by mutating
// the shared keys map, since keys is a package-level value.
func (h helpKeyMap) FullHelp() [][]key.Binding {
	navigate := []key.Binding{
		keys.Up, keys.Down,
		keys.PageUp, keys.PageDown,
		keys.HalfPageUp, keys.HalfPageDown,
		keys.GotoTop, keys.GotoBottom,
	}

	var panes []key.Binding
	actions := []key.Binding{}
	if h.activeTab != tabInsights {
		panes = append(panes, keys.FocusPrev, keys.FocusNext, keys.ScrollUp, keys.ScrollDown)
		actions = append(actions, keys.Filter)
		if h.activeTab == tabLog {
			actions = append(actions, keys.Sort)
		}
	}
	panes = append(panes, keys.TabNext, keys.TabPrev)
	actions = append(actions, keys.DateRange, keys.Export, keys.Help, keys.Quit)

	return [][]key.Binding{navigate, panes, actions}
}
