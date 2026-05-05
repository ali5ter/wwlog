// Package tui implements the Bubble Tea TUI for wwlog.
package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/ali5ter/wwlog/internal/api"
	"github.com/ali5ter/wwlog/internal/auth"
)

type appScreen int

const (
	screenSplash appScreen = iota
	screenLog
)

// dialogKind identifies which (if any) modal dialog is currently open
// over the main Log screen.
type dialogKind int

const (
	dialogNone dialogKind = iota
	dialogDateRange
	dialogExport
)

type tab int

const (
	tabLog tab = iota
	tabNutrition
	tabInsights
)

var tabNames = []string{"Log", "Nutrition", "Insights"}

type dataMsg struct {
	logs   []*api.DayLog
	client *api.Client
	err    error
}

type versionMsg struct{ latest string }

// Tab-contextual footer animations.
var (
	animLogSpinner = spinner.Spinner{
		Frames: []string{"∘───", "─∘──", "──∘─", "───∘", "──∘─", "─∘──"},
		FPS:    time.Second / 8,
	}
	animNutriSpinner = spinner.Spinner{
		Frames: []string{"    ", "·   ", "●   ", "·   ", "    ", "    "},
		FPS:    time.Second / 3,
	}
)

// Model is the top-level Bubble Tea model.
type Model struct {
	width  int
	height int
	screen appScreen
	dialog dialogKind

	spinner       spinner.Model
	animLog       spinner.Model
	animNutrition spinner.Model
	loading       bool
	err           error
	activeTab     tab
	logs          []*api.DayLog
	logModel      logModel
	nutriModel    nutriModel
	insightsModel insightsModel

	splashModel    splashModel
	exportModel    exportModel
	dateRangeModel dateRangeModel
	authObj        *auth.Auth
	tld            string
	start          string
	end            string
	version        string
	latestVersion  string
	client         *api.Client
	statusMsg      string
}

// Run initialises and starts the TUI, blocking until the user quits.
func Run(authObj *auth.Auth, tld, preStart, preEnd string, version string) error {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(colorTeal)

	al := spinner.New()
	al.Spinner = animLogSpinner
	al.Style = lipgloss.NewStyle().Foreground(colorSteel)

	an := spinner.New()
	an.Spinner = animNutriSpinner
	an.Style = lipgloss.NewStyle().Foreground(colorSteel)

	m := Model{
		spinner:       s,
		animLog:       al,
		animNutrition: an,
		screen:        screenSplash,
		splashModel:   newSplashModel(authObj, version, preStart, preEnd),
		authObj:       authObj,
		tld:           tld,
		version:       version,
	}

	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.animLog.Tick,
		m.animNutrition.Tick,
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
		m.dateRangeModel.resize(m.width, m.height)
		m.logModel.resize(m.width, m.contentHeight())
		m.nutriModel.resize(m.width, m.contentHeight())
		m.insightsModel.resize(m.width, m.contentHeight())
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
		return m, tea.Batch(
			func() tea.Msg {
				token, err := authObj.Token()
				if err != nil {
					return dataMsg{err: err}
				}
				client := api.New(token, tld)
				logs, err := fetchLogs(client, start, end)
				return dataMsg{logs: logs, client: client, err: err}
			},
			func() tea.Msg { return versionMsg{latest: api.FetchLatestVersion()} },
		)

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
		// Nutrition is embedded in each food entry — compute synchronously, no extra API calls.
		nutrition := api.ComputeAllNutrition(m.logs)
		loc := newLocale(m.tld)
		m.logModel = newLogModel(m.logs, m.width, m.contentHeight(), loc)
		m.nutriModel = newNutriModel(m.logs, nutrition, m.width, m.contentHeight(), loc)
		m.insightsModel = newInsightsModel(m.logs, m.width, m.contentHeight())
		return m, nil

	case versionMsg:
		m.latestVersion = msg.latest
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
		// Always tick every spinner — each Update is a no-op when the ID doesn't match.
		var c1, c2, c3, c4 tea.Cmd
		m.splashModel, c1 = m.splashModel.update(msg)
		m.spinner, c2 = m.spinner.Update(msg)
		m.animLog, c3 = m.animLog.Update(msg)
		m.animNutrition, c4 = m.animNutrition.Update(msg)
		return m, tea.Batch(c1, c2, c3, c4)

	case tea.KeyPressMsg:
		// Splash: only ctrl+c quits — q is a valid character in huh text fields.
		if m.screen == screenSplash {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.splashModel, cmd = m.splashModel.update(msg)
			return m, cmd
		}
		// Export dialog: esc cancels, ctrl+c quits, everything else goes to huh.
		if m.dialog == dialogExport {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			if msg.String() == "esc" {
				m.dialog = dialogNone
				return m, nil
			}
			var cmd tea.Cmd
			m.exportModel, cmd = m.exportModel.update(msg)
			if m.exportModel.form.State == huh.StateCompleted {
				format := m.exportModel.form.GetString("format")
				dir := m.exportModel.form.GetString("dir")
				m.dialog = dialogNone
				m.statusMsg = styleFoodPortion.Render("  Saving…")
				return m, runExport(format, dir, m.start, m.end, m.logs)
			}
			if m.exportModel.form.State == huh.StateAborted {
				m.dialog = dialogNone
			}
			return m, cmd
		}
		// Date range dialog: esc cancels, ctrl+c quits, enter submits.
		if m.dialog == dialogDateRange {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			if msg.String() == "esc" {
				m.dialog = dialogNone
				return m, nil
			}
			var cmd tea.Cmd
			m.dateRangeModel, cmd = m.dateRangeModel.update(msg)
			if m.dateRangeModel.form.State == huh.StateCompleted {
				m.start = m.dateRangeModel.form.GetString("start")
				m.end = m.dateRangeModel.form.GetString("end")
				m.dialog = dialogNone
				m.loading = true
				authObj := m.authObj
				tld := m.tld
				start := m.start
				end := m.end
				return m, func() tea.Msg {
					token, err := authObj.Token()
					if err != nil {
						return dataMsg{err: err}
					}
					client := api.New(token, tld)
					logs, err := fetchLogs(client, start, end)
					return dataMsg{logs: logs, client: client, err: err}
				}
			}
			if m.dateRangeModel.form.State == huh.StateAborted {
				m.dialog = dialogNone
			}
			return m, cmd
		}
		// Log screen.
		if key.Matches(msg, keys.Quit) {
			return m, tea.Quit
		}
		m.statusMsg = "" // any keypress clears the status message
		tabFiltering := (m.activeTab == tabLog && m.logModel.filtering) ||
			(m.activeTab == tabNutrition && m.nutriModel.filtering)
		if !m.loading && m.err == nil && !tabFiltering {
			switch {
			case key.Matches(msg, keys.Export):
				m.exportModel = newExportModel(m.width, m.height)
				m.dialog = dialogExport
				return m, m.exportModel.form.Init()
			case key.Matches(msg, keys.DateRange):
				m.dateRangeModel = newDateRangeModel(m.start, m.end, m.width, m.height)
				m.dialog = dialogDateRange
				return m, m.dateRangeModel.form.Init()
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

	// Route non-key messages to splash or active dialog.
	if m.screen == screenSplash {
		var cmd tea.Cmd
		m.splashModel, cmd = m.splashModel.update(msg)
		return m, cmd
	}
	if m.dialog == dialogExport {
		var cmd tea.Cmd
		m.exportModel, cmd = m.exportModel.update(msg)
		// huh may complete via an internal message rather than a KeyMsg.
		if m.exportModel.form.State == huh.StateCompleted {
			format := m.exportModel.form.GetString("format")
			dir := m.exportModel.form.GetString("dir")
			m.dialog = dialogNone
			m.statusMsg = styleFoodPortion.Render("  Saving…")
			return m, runExport(format, dir, m.start, m.end, m.logs)
		}
		return m, cmd
	}
	if m.dialog == dialogDateRange {
		var cmd tea.Cmd
		m.dateRangeModel, cmd = m.dateRangeModel.update(msg)
		// huh may complete via an internal message rather than a KeyMsg.
		if m.dateRangeModel.form.State == huh.StateCompleted {
			m.start = m.dateRangeModel.form.GetString("start")
			m.end = m.dateRangeModel.form.GetString("end")
			m.dialog = dialogNone
			m.loading = true
			authObj := m.authObj
			tld := m.tld
			start, end := m.start, m.end
			return m, func() tea.Msg {
				token, err := authObj.Token()
				if err != nil {
					return dataMsg{err: err}
				}
				client := api.New(token, tld)
				logs, err := fetchLogs(client, start, end)
				return dataMsg{logs: logs, client: client, err: err}
			}
		}
		return m, cmd
	}

	// Delegate to the active tab model (only once data is loaded).
	if m.loading || m.err != nil {
		return m, nil
	}
	var cmd tea.Cmd
	switch m.activeTab {
	case tabLog:
		m.logModel, cmd = m.logModel.update(msg)
	case tabNutrition:
		m.nutriModel, cmd = m.nutriModel.update(msg)
	case tabInsights:
		m.insightsModel, cmd = m.insightsModel.update(msg)
	}
	return m, cmd
}

func (m Model) View() tea.View {
	v := tea.NewView(m.viewContent())
	v.AltScreen = true
	return v
}

func (m Model) viewContent() string {
	if m.screen == screenSplash {
		return m.splashModel.view()
	}
	if m.loading {
		return m.loadingView()
	}
	if m.err != nil {
		return styleError.Render(fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err))
	}

	sep := styleDim.Render(strings.Repeat("─", m.width))
	main := lipgloss.JoinVertical(lipgloss.Left,
		m.headerView(),
		sep,
		m.contentView(),
		sep,
		m.statusView(),
	)

	// If a dialog is active, composite it on top of the main TUI. The Lipgloss
	// v2 compositor draws layers in z-order at cell coordinates, so the main
	// content stays visible behind/around the dialog box.
	switch m.dialog {
	case dialogDateRange:
		return overlayDialog(main, m.dateRangeModel.view(), m.width, m.height)
	case dialogExport:
		return overlayDialog(main, m.exportModel.view(), m.width, m.height)
	}
	return main
}

// overlayDialog composites a dialog box on top of the main TUI background using
// the Lipgloss v2 compositor. The dialog is centred horizontally and vertically.
func overlayDialog(bg, dialog string, width, height int) string {
	dw, dh := lipgloss.Size(dialog)
	x := (width - dw) / 2
	if x < 0 {
		x = 0
	}
	y := (height - dh) / 2
	if y < 0 {
		y = 0
	}
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(bg),
		lipgloss.NewLayer(dialog).X(x).Y(y).Z(1),
	).Render()
}

func (m Model) loadingView() string {
	spinStr := styleSplashSub.Render(fmt.Sprintf("%s  Loading your food log…", m.spinner.View()))
	// Mirror the splash view structure exactly: paddedBody + hint appended outside,
	// giving the same total content height so the logo sits at the same row.
	paddedBody := lipgloss.Place(m.width, splashBodyH, lipgloss.Center, lipgloss.Top, spinStr)
	content := lipgloss.JoinVertical(lipgloss.Center,
		renderGradientLogo(), "",
		styleSplashSub.Render("Browse and export your food log"),
		"",
		paddedBody,
		styleSplashHint.Render("q to quit"),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) headerView() string {
	title := styleHeaderAccent.Render("wwlog")

	var tabParts strings.Builder
	for i, name := range tabNames {
		if tab(i) == m.activeTab {
			tabParts.WriteString(styleTabActive.Render(name))
		} else {
			tabParts.WriteString(styleTabInactive.Render(name))
		}
	}

	dateRange := styleHeader.Render(m.start + " → " + m.end)
	left := lipgloss.JoinHorizontal(lipgloss.Center, title, styleHeader.Render(" · "), tabParts.String())
	gap := max(0, m.width-lipgloss.Width(left)-lipgloss.Width(dateRange))
	return lipgloss.NewStyle().
		Background(colorPanel).
		Width(m.width).
		Render(left + strings.Repeat(" ", gap) + dateRange)
}

func (m Model) statusView() string {
	if m.statusMsg != "" {
		return styleStatusBar.Width(m.width).Render(m.statusMsg)
	}
	hints := []string{
		styleStatusKey.Render("↑/↓") + " navigate",
		styleStatusKey.Render("⇧↑/↓") + " scroll",
	}
	if m.activeTab == tabLog || m.activeTab == tabNutrition {
		hints = append(hints, styleStatusKey.Render("/")+" filter")
	}
	hints = append(hints,
		styleStatusKey.Render("r")+" range",
		styleStatusKey.Render("s")+" sort",
		styleStatusKey.Render("e")+" export",
		styleStatusKey.Render("tab")+" switch",
		styleStatusKey.Render("q")+" quit",
	)
	left := strings.Join(hints, "  ")

	var right string
	currentNorm := strings.TrimPrefix(m.version, "v")
	if m.latestVersion != "" && m.latestVersion != currentNorm {
		right = lipgloss.NewStyle().
			Background(colorTeal).
			Foreground(colorPanel).
			Padding(0, 1).
			Render("↑ v" + m.latestVersion + " available")
	} else {
		var anim string
		switch m.activeTab {
		case tabLog:
			anim = m.animLog.View()
		case tabNutrition:
			anim = m.animNutrition.View()
		}
		right = lipgloss.NewStyle().Foreground(colorSteel).Render(anim)
	}

	contentWidth := m.width - 2
	gap := contentWidth - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return styleStatusBar.Width(m.width).Render(left + strings.Repeat(" ", gap) + right)
}

func (m Model) contentView() string {
	switch m.activeTab {
	case tabLog:
		return m.logModel.view()
	case tabNutrition:
		return m.nutriModel.view()
	case tabInsights:
		return m.insightsModel.view()
	}
	return ""
}

func (m Model) contentHeight() int {
	return m.height - 4 // header + sep + sep + status
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
