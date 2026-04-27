// Package tui implements the Bubble Tea TUI for wwlog.
package tui

import (
	"fmt"
	"strings"

	"github.com/ali5ter/wwlog/internal/api"
	"github.com/ali5ter/wwlog/internal/auth"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type appScreen int

const (
	screenSplash appScreen = iota
	screenLog
	screenExport
)

type tab int

const (
	tabLog tab = iota
	tabNutrition
)

var tabNames = []string{"Log", "Nutrition"}

type dataMsg struct {
	logs   []*api.DayLog
	client *api.Client
	err    error
}

// Model is the top-level Bubble Tea model.
type Model struct {
	width  int
	height int
	screen appScreen

	spinner    spinner.Model
	loading    bool
	err        error
	activeTab  tab
	logs       []*api.DayLog
	logModel   logModel
	nutriModel nutriModel

	splashModel splashModel
	exportModel exportModel
	authObj     *auth.Auth
	tld         string
	start       string
	end         string
	version     string
	nutrition   bool
	client      *api.Client
	statusMsg   string
}

// Run initialises and starts the TUI, blocking until the user quits.
func Run(authObj *auth.Auth, tld, preStart, preEnd string, nutrition bool, version string) error {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(colorTeal)

	m := Model{
		spinner:     s,
		screen:      screenSplash,
		splashModel: newSplashModel(authObj, version, preStart, preEnd),
		authObj:     authObj,
		tld:         tld,
		nutrition:   nutrition,
		version:     version,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.splashModel.init(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.splashModel.resize(m.width, m.height)
		m.exportModel.resize(m.width, m.height)
		m.logModel.resize(m.width, m.contentHeight())
		m.nutriModel.resize(m.width, m.contentHeight())
		return m, nil

	case splashDoneMsg:
		m.start = msg.start
		m.end = msg.end
		m.screen = screenLog
		m.loading = true
		authObj := m.authObj
		tld := m.tld
		start := msg.start
		end := msg.end
		return m, func() tea.Msg {
			token, err := authObj.Token()
			if err != nil {
				return dataMsg{err: err}
			}
			client := api.New(token, tld)
			logs, err := fetchLogs(client, start, end)
			return dataMsg{logs: logs, client: client, err: err}
		}

	case dataMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.logs = msg.logs
		if msg.client != nil {
			m.client = msg.client
		}
		m.logModel = newLogModel(m.logs, m.width, m.contentHeight())
		m.nutriModel = newNutriModel(m.logs, m.width, m.contentHeight())
		return m, nil

	case exportDoneMsg:
		m.screen = screenLog
		if msg.err != nil {
			m.statusMsg = styleError.Render("  Export failed: " + msg.err.Error())
		} else {
			m.statusMsg = styleMealHeading.Render("  ✓ Exported → " + msg.filename)
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Splash: only ctrl+c quits — q is a valid character in huh text fields.
		if m.screen == screenSplash {
			if msg.Type == tea.KeyCtrlC {
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.splashModel, cmd = m.splashModel.update(msg)
			return m, cmd
		}
		// Export: esc cancels, ctrl+c quits, everything else goes to huh.
		if m.screen == screenExport {
			if msg.Type == tea.KeyCtrlC {
				return m, tea.Quit
			}
			if msg.Type == tea.KeyEsc {
				m.screen = screenLog
				return m, nil
			}
			var cmd tea.Cmd
			m.exportModel, cmd = m.exportModel.update(msg)
			if m.exportModel.form.State == huh.StateCompleted {
				format := m.exportModel.form.GetString("format")
				return m, runExport(format, m.start, m.end, m.logs)
			}
			if m.exportModel.form.State == huh.StateAborted {
				m.screen = screenLog
			}
			return m, cmd
		}
		// Log screen.
		if key.Matches(msg, keys.Quit) {
			return m, tea.Quit
		}
		m.statusMsg = "" // any keypress clears the status message
		if !m.loading && m.err == nil {
			switch {
			case key.Matches(msg, keys.Export):
				m.exportModel = newExportModel(m.width, m.height)
				m.screen = screenExport
				return m, m.exportModel.form.Init()
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
	}

	// Route non-key messages to splash or export screens.
	if m.screen == screenSplash {
		var cmd tea.Cmd
		m.splashModel, cmd = m.splashModel.update(msg)
		return m, cmd
	}
	if m.screen == screenExport {
		var cmd tea.Cmd
		m.exportModel, cmd = m.exportModel.update(msg)
		return m, cmd
	}

	// Delegate to the active tab model.
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
	if m.screen == screenSplash {
		return m.splashModel.view()
	}
	if m.screen == screenExport {
		return m.exportModel.view()
	}
	if m.loading {
		return m.loadingView()
	}
	if m.err != nil {
		return styleError.Render(fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err))
	}

	header := m.headerView()
	status := m.statusView()
	content := m.contentView()
	return lipgloss.JoinVertical(lipgloss.Left, header, content, status)
}

func (m Model) loadingView() string {
	spinStr := styleSplashSub.Render(fmt.Sprintf("%s  Loading your food log…", m.spinner.View()))
	content := lipgloss.JoinVertical(lipgloss.Center,
		styleSplashLogo.Render(asciiLogo), "",
		spinStr, "",
		styleSplashHint.Render("q to quit"),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
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
	if m.statusMsg != "" {
		return styleStatusBar.Width(m.width).Render(m.statusMsg)
	}
	hints := []string{
		styleStatusKey.Render("↑/↓") + " navigate",
		styleStatusKey.Render("/") + " filter",
		styleStatusKey.Render("^E") + " export",
		styleStatusKey.Render("tab") + " switch",
		styleStatusKey.Render("q/^C") + " quit",
	}
	return styleStatusBar.Width(m.width).Render(strings.Join(hints, "  "))
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

func fetchLogs(client *api.Client, start, end string) ([]*api.DayLog, error) {
	dates, err := api.DateRange(start, end)
	if err != nil {
		return nil, err
	}
	var logs []*api.DayLog
	for _, date := range dates {
		day, err := client.FetchDay(date)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", date, err)
		}
		logs = append(logs, day)
	}
	return logs, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
