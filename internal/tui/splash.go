package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ali5ter/wwlog/internal/auth"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/huh"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const asciiLogo = `
 ██╗    ██╗██╗    ██╗
 ██║    ██║██║    ██║
 ██║ █╗ ██║██║ █╗ ██║
 ██║███╗██║██║███╗██║
 ╚███╔███╔╝╚███╔███╔╝
  ╚══╝╚══╝  ╚══╝╚══╝`

type splashPhase int

const (
	splashChecking splashPhase = iota
	splashLogin
	splashDateRange
)

type authCheckedMsg struct{ authed bool }
type loginErrMsg struct{ msg string }
type splashDoneMsg struct{ start, end string }

type splashModel struct {
	phase    splashPhase
	authObj  *auth.Auth
	version  string
	preStart string
	preEnd   string
	width    int
	height   int
	err      string
	form     *huh.Form
	spinner  spinner.Model
}

func newSplashModel(a *auth.Auth, version, preStart, preEnd string) splashModel {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(colorTeal)
	return splashModel{
		phase:    splashChecking,
		authObj:  a,
		version:  version,
		preStart: preStart,
		preEnd:   preEnd,
		spinner:  s,
	}
}

func (m *splashModel) resize(width, height int) {
	m.width = width
	m.height = height
}

func (m splashModel) init() tea.Cmd {
	return tea.Batch(m.checkAuthCmd, m.spinner.Tick)
}

func (m splashModel) checkAuthCmd() tea.Msg {
	_, err := m.authObj.Token()
	return authCheckedMsg{authed: err == nil}
}

func (m splashModel) loginCmd() tea.Msg {
	email := m.form.GetString("email")
	password := m.form.GetString("password")
	_, err := m.authObj.Login(email, password)
	if err != nil {
		return loginErrMsg{msg: err.Error()}
	}
	return authCheckedMsg{authed: true}
}

func (m splashModel) buildLoginForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("email").
				Title("Email").
				Placeholder("you@example.com"),
			huh.NewInput().
				Key("password").
				Title("Password").
				EchoMode(huh.EchoModePassword).
				Placeholder("enter your password"),
		),
	).
		WithTheme(wwHuhTheme()).
		WithWidth(m.formWidth()).
		WithShowHelp(true)
}

func (m splashModel) buildDateForm() *huh.Form {
	today := time.Now().Format("2006-01-02")
	weekAgo := time.Now().AddDate(0, 0, -6).Format("2006-01-02")
	start := m.preStart
	if start == "" {
		start = weekAgo
	}
	end := m.preEnd
	if end == "" {
		end = today
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("start").
				Title("From").
				Placeholder("YYYY-MM-DD").
				Value(&start).
				Validate(validateDate),
			huh.NewInput().
				Key("end").
				Title("To").
				Placeholder("YYYY-MM-DD").
				Value(&end).
				Validate(validateDate),
		),
	).
		WithTheme(wwHuhTheme()).
		WithWidth(m.formWidth()).
		WithShowHelp(true)
}

func validateDate(s string) error {
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return fmt.Errorf("use YYYY-MM-DD")
	}
	return nil
}

func (m splashModel) formWidth() int {
	w := m.width / 2
	if w < 44 {
		w = 44
	}
	if w > 72 {
		w = 72
	}
	return w
}

func (m splashModel) update(msg tea.Msg) (splashModel, tea.Cmd) {
	switch msg := msg.(type) {

	case spinner.TickMsg:
		if m.phase == splashChecking {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case authCheckedMsg:
		m.err = ""
		if msg.authed {
			m.phase = splashDateRange
			m.form = m.buildDateForm()
		} else {
			m.phase = splashLogin
			m.form = m.buildLoginForm()
		}
		return m, m.form.Init()

	case loginErrMsg:
		m.err = msg.msg
		m.phase = splashLogin
		m.form = m.buildLoginForm()
		return m, m.form.Init()
	}

	if m.form == nil {
		return m, nil
	}

	model, cmd := m.form.Update(msg)
	m.form = model.(*huh.Form)

	if m.form.State == huh.StateCompleted {
		switch m.phase {
		case splashLogin:
			return m, m.loginCmd
		case splashDateRange:
			start := m.form.GetString("start")
			end := m.form.GetString("end")
			return m, func() tea.Msg { return splashDoneMsg{start: start, end: end} }
		}
	}

	return m, cmd
}

// renderGradientLogo renders the WW ASCII logo with a teal→purple→steel gradient.
func renderGradientLogo() string {
	lines := strings.Split(strings.TrimLeft(asciiLogo, "\n"), "\n")
	palette := []lipgloss.Color{colorTeal, colorTeal, colorPurple, colorPurple, colorSteel, colorMuted}
	rendered := make([]string, len(lines))
	for i, line := range lines {
		c := palette[i%len(palette)]
		rendered[i] = lipgloss.NewStyle().Foreground(c).Bold(true).Render(line)
	}
	return strings.Join(rendered, "\n")
}

func (m splashModel) view() string {
	logoStr := renderGradientLogo()
	titleStr := styleSplashTitle.Render("wwlog  " + m.version)
	subStr := styleSplashSub.Render("Weight Watchers food log browser")

	var middle string
	switch m.phase {
	case splashChecking:
		middle = styleSplashSub.Render(m.spinner.View() + "  Checking credentials…")
	default:
		if m.form != nil {
			middle = m.form.View()
		}
	}

	parts := []string{logoStr, "", titleStr, subStr, ""}
	if m.err != "" {
		parts = append(parts, styleError.Render("  "+m.err), "")
	}
	parts = append(parts, middle)
	parts = append(parts, "", styleSplashHint.Render("ctrl+c to quit"))

	content := lipgloss.JoinVertical(lipgloss.Center, parts...)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// wwHuhTheme returns a huh theme styled with the WW colour palette.
func wwHuhTheme() *huh.Theme {
	t := huh.ThemeCharm()

	teal   := lipgloss.Color("#00B388")
	purple := lipgloss.Color("#6B4C9A")
	steel  := lipgloss.Color("#7f93a6")
	muted  := lipgloss.Color("#a8b6c0")
	panel  := lipgloss.Color("#161d24")
	text   := lipgloss.Color("#e9eff3")

	t.Focused.Base = t.Focused.Base.BorderForeground(teal)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = lipgloss.NewStyle().Foreground(teal).Bold(true)
	t.Focused.Description = lipgloss.NewStyle().Foreground(steel)
	t.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(purple)
	t.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(purple)
	t.Focused.SelectSelector = lipgloss.NewStyle().Foreground(teal).SetString("> ")
	t.Focused.NextIndicator = lipgloss.NewStyle().Foreground(teal).SetString("→")
	t.Focused.FocusedButton = lipgloss.NewStyle().
		Foreground(panel).Background(teal).Bold(true).
		Padding(0, 3).MarginRight(1)
	t.Focused.BlurredButton = lipgloss.NewStyle().
		Foreground(muted).Background(panel).
		Padding(0, 3).MarginRight(1)
	t.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(teal)
	t.Focused.TextInput.CursorText = lipgloss.NewStyle().Foreground(panel).Background(teal)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(muted)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(purple)
	t.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(text)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.Title = lipgloss.NewStyle().Foreground(steel)
	t.Blurred.TextInput.Text = lipgloss.NewStyle().Foreground(muted)
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = lipgloss.NewStyle().Foreground(teal).Bold(true)
	t.Group.Description = lipgloss.NewStyle().Foreground(steel)

	return t
}

