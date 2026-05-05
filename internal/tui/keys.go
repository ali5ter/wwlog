package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
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
	Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	ScrollUp:   key.NewBinding(key.WithKeys("shift+up"), key.WithHelp("⇧↑", "scroll up")),
	ScrollDown: key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("⇧↓", "scroll down")),
	Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	Export:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export")),
	DateRange:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "range")),
	Sort:       key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
	TabNext:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
	TabPrev:    key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("⇧tab", "prev tab")),
	Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
