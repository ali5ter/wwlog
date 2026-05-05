package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/ali5ter/wwlog/internal/auth"
)

const asciiLogo = `
 ╦ ╦ ╦ ╦ ╦   ╔═╗ ╔═╗
 ║║║ ║║║ ║   ║ ║ ║ ╦
 ╚╩╝ ╚╩╝ ╩═╝ ╚═╝ ╚═╝`

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
		WithTheme(wwHuhTheme{}).
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
		WithTheme(wwHuhTheme{}).
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

// renderGradientLogo renders the splash ASCII logo with a smooth RGB gradient
// interpolated line-by-line from teal (#00B388) at the top to purple
// (#6B4C9A) at the bottom.
func renderGradientLogo() string {
	lines := strings.Split(strings.TrimLeft(asciiLogo, "\n"), "\n")
	n := len(lines) - 1
	if n < 1 {
		n = 1
	}
	rendered := make([]string, len(lines))
	for i, line := range lines {
		t := float64(i) / float64(n)
		c := lerpColor(colorTeal, colorPurple, t)
		rendered[i] = lipgloss.NewStyle().Foreground(c).Bold(true).Render(line)
	}
	return strings.Join(rendered, "\n")
}

// splashBodyH is the reserved height for the body area (form or spinner).
// Keeping this constant ensures the logo never shifts position between phases.
const splashBodyH = 16

func (m splashModel) view() string {
	var body string
	switch m.phase {
	case splashChecking:
		body = styleSplashSub.Render(m.spinner.View() + "  Checking credentials…")
	default:
		if m.form != nil {
			body = m.form.View()
		}
	}
	if m.err != "" {
		body = lipgloss.JoinVertical(lipgloss.Center, styleError.Render("  "+m.err), "", body)
	}

	// Pad body to a fixed height so the logo's vertical position is identical
	// across all phases (spinner → form transitions don't shift the logo).
	paddedBody := lipgloss.Place(m.width, splashBodyH, lipgloss.Center, lipgloss.Top, body)

	content := lipgloss.JoinVertical(lipgloss.Center,
		renderGradientLogo(), "",
		styleSplashSub.Render("Browse and export your food log"),
		"",
		paddedBody,
		styleSplashHint.Render("ctrl+c to quit"),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// dateRangeModel is the in-TUI date range picker — same form as the splash
// date step, but presented as an overlay from the main log screen.
type dateRangeModel struct {
	form   *huh.Form
	width  int
	height int
}

func newDateRangeModel(start, end string, width, height int) dateRangeModel {
	w := dialogContentWidth(width)
	// huh forms capture the *string values at form-init time, so we need local
	// copies that the form fields can write into.
	s, e := start, end
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("start").Title("From").Placeholder("YYYY-MM-DD").Value(&s).Validate(validateDate),
			huh.NewInput().Key("end").Title("To").Placeholder("YYYY-MM-DD").Value(&e).Validate(validateDate),
		),
	).WithTheme(wwHuhTheme{}).WithWidth(w).WithShowHelp(false)
	return dateRangeModel{form: form, width: width, height: height}
}

func (m *dateRangeModel) resize(width, height int) {
	m.width = width
	m.height = height
}

func (m dateRangeModel) update(msg tea.Msg) (dateRangeModel, tea.Cmd) {
	model, cmd := m.form.Update(msg)
	m.form = model.(*huh.Form)
	return m, cmd
}

func (m dateRangeModel) view() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	return renderDialog("Change date range", m.form.View(), "esc cancel · enter submit · tab next field")
}

// wwHuhTheme is a huh.Theme styled with the WW colour palette.
type wwHuhTheme struct{}

func (wwHuhTheme) Theme(isDark bool) *huh.Styles {
	t := huh.ThemeCharm(isDark)

	teal := lipgloss.Color("#00B388")
	purple := lipgloss.Color("#6B4C9A")
	steel := lipgloss.Color("#7f93a6")
	muted := lipgloss.Color("#a8b6c0")
	panel := lipgloss.Color("#161d24")
	text := lipgloss.Color("#e9eff3")

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
