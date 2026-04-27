package tui

import "github.com/charmbracelet/lipgloss"

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// WW-inspired colour palette.
var (
	colorTeal   = lipgloss.Color("#00B388") // WW signature green — active, selected, accents
	colorPurple = lipgloss.Color("#6B4C9A") // WW app purple — prompts, pointers
	colorSteel  = lipgloss.Color("#7f93a6") // Secondary text, metadata, labels
	colorMuted  = lipgloss.Color("#a8b6c0") // Inactive items, borders, dim elements
	colorText   = lipgloss.Color("#e9eff3") // Primary text
	colorPanel  = lipgloss.Color("#161d24") // Header/status backgrounds
	colorLine   = lipgloss.Color("#2b3742") // Borders, separators
)

var (
	styleHeader = lipgloss.NewStyle().
			Background(colorPanel).
			Foreground(colorSteel).
			Padding(0, 1).
			Bold(true)

	styleHeaderAccent = lipgloss.NewStyle().
				Background(colorPanel).
				Foreground(colorTeal).
				Bold(true)

	styleTabActive = lipgloss.NewStyle().
			Foreground(colorTeal).
			Bold(true).
			Padding(0, 1)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(colorMuted).
				Padding(0, 1)

	styleStatusBar = lipgloss.NewStyle().
			Background(colorPanel).
			Foreground(colorMuted).
			Padding(0, 1)

	styleStatusKey = lipgloss.NewStyle().
			Background(colorPanel).
			Foreground(colorPurple)

	stylePanelBorder = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, true, false, false).
				BorderForeground(colorLine)

	styleMealHeading = lipgloss.NewStyle().
				Foreground(colorTeal).
				Bold(true)

	styleFoodItem = lipgloss.NewStyle().
			Foreground(colorText)

	styleFoodPortion = lipgloss.NewStyle().
				Foreground(colorMuted)

	styleSelected = lipgloss.NewStyle().
			Foreground(colorTeal).
			Bold(true)

	styleNormal = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleDetailLabel = lipgloss.NewStyle().
				Foreground(colorSteel).
				Bold(true)

	styleDetailValue = lipgloss.NewStyle().
				Foreground(colorText)

	styleFilterPrompt = lipgloss.NewStyle().
				Foreground(colorPurple)

	styleError = lipgloss.NewStyle().
			Foreground(colorPurple).
			Padding(1, 2)

	styleDim = lipgloss.NewStyle().
			Foreground(colorLine)

	// Splash screen styles
	styleSplashLogo = lipgloss.NewStyle().
			Foreground(colorTeal).
			Bold(true)

	styleSplashTitle = lipgloss.NewStyle().
				Foreground(colorText).
				Bold(true)

	styleSplashSub = lipgloss.NewStyle().
			Foreground(colorSteel)

	styleSplashFormTitle = lipgloss.NewStyle().
				Foreground(colorTeal).
				Bold(true)

	styleSplashHint = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleSplashInputPrompt = lipgloss.NewStyle().
				Foreground(colorPurple)
)
