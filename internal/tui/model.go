// Package tui implements the Bubble Tea TUI for wwlog.
package tui

import (
	"fmt"
	"strings"

	"github.com/ali5ter/wwlog/internal/api"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tab int

const (
	tabLog tab = iota
	tabNutrition
)

var tabNames = []string{"Log", "Nutrition"}

type dataMsg struct {
	logs []*api.DayLog
	err  error
}

// Model is the top-level Bubble Tea model.
type Model struct {
	width     int
	height    int
	activeTab tab
	spinner   spinner.Model
	loading   bool
	err       error

	logs      []*api.DayLog
	logModel  logModel
	nutriModel nutriModel

	start     string
	end       string
	version   string
	fetchFn   func() ([]*api.DayLog, error)
	client    *api.Client
	nutrition bool
}

// Run initialises and starts the TUI, blocking until the user quits.
func Run(
	fetchFn func() ([]*api.DayLog, error),
	start, end string,
	nutrition bool,
	client *api.Client,
	version string,
) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorTeal)

	m := Model{
		spinner:   s,
		loading:   true,
		fetchFn:   fetchFn,
		start:     start,
		end:       end,
		nutrition: nutrition,
		client:    client,
		version:   version,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			logs, err := m.fetchFn()
			return dataMsg{logs: logs, err: err}
		},
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.logModel.resize(m.width, m.contentHeight())
		m.nutriModel.resize(m.width, m.contentHeight())
		return m, nil

	case dataMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.logs = msg.logs
		m.logModel = newLogModel(m.logs, m.width, m.contentHeight())
		m.nutriModel = newNutriModel(m.logs, m.width, m.contentHeight())
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.TabNext):
			m.activeTab = (m.activeTab + 1) % tab(len(tabNames))
		case key.Matches(msg, keys.TabPrev):
			if m.activeTab == 0 {
				m.activeTab = tab(len(tabNames) - 1)
			} else {
				m.activeTab--
			}
		}
	}

	// Delegate remaining key events to the active tab model
	var cmd tea.Cmd
	switch m.activeTab {
	case tabLog:
		m.logModel, cmd = m.logModel.update(msg)
	case tabNutrition:
		m.nutriModel, cmd = m.nutriModel.update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Loading food log…\n", m.spinner.View())
	}
	if m.err != nil {
		return styleError.Render(fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err))
	}

	header := m.headerView()
	status := m.statusView()
	content := m.contentView()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, status)
}

func (m Model) headerView() string {
	logo := styleHeaderAccent.Render("W—W")
	title := styleHeader.Render(fmt.Sprintf(" wwlog  %s → %s", m.start, m.end))

	var tabs []string
	for i, name := range tabNames {
		if tab(i) == m.activeTab {
			tabs = append(tabs, styleTabActive.Render(name))
		} else {
			tabs = append(tabs, styleTabInactive.Render(name))
		}
	}
	tabBar := strings.Join(tabs, styleDim.Render("│"))

	left := lipgloss.JoinHorizontal(lipgloss.Top, logo, title)
	right := styleHeader.Render(tabBar)
	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(left)-lipgloss.Width(right)))
	return styleHeader.Render(left + gap + right)
}

func (m Model) statusView() string {
	keys := []string{
		styleStatusKey.Render("↑/↓") + " navigate",
		styleStatusKey.Render("/") + " filter",
		styleStatusKey.Render("^E") + " export",
		styleStatusKey.Render("tab") + " switch",
		styleStatusKey.Render("q") + " quit",
	}
	return styleStatusBar.Width(m.width).Render(strings.Join(keys, "  "))
}

func (m Model) contentView() string {
	switch m.activeTab {
	case tabLog:
		return m.logModel.view()
	case tabNutrition:
		return m.nutriModel.view()
	}
	return ""
}

func (m Model) contentHeight() int {
	return m.height - 2 // header + status bar
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
