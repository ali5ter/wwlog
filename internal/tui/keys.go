package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	// Up/Down navigate the date list by default. When a tab's detail pane
	// has focus (see FocusNext/keys.FocusPrev, logModel.detailFocused /
	// nutriModel.detailFocused), they scroll that pane instead — the same
	// behavior ScrollUp/ScrollDown always provide, just reachable without
	// leaving list focus. Both are still bubbles' own list.DefaultKeyMap
	// bindings underneath; nothing here rebinds list navigation itself.
	Up         key.Binding
	Down       key.Binding
	FocusPrev  key.Binding // "left"  — focus the date list
	FocusNext  key.Binding // "right" — focus the detail pane
	ScrollUp   key.Binding
	ScrollDown key.Binding
	Filter     key.Binding
	Export     key.Binding
	DateRange  key.Binding
	Sort       key.Binding
	TabNext    key.Binding
	TabPrev    key.Binding
	Help       key.Binding
	Quit       key.Binding
}

var keys = keyMap{
	Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	FocusPrev: key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "focus list")),
	FocusNext: key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "focus detail")),
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
