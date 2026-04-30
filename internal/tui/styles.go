package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// lerpColor linearly interpolates between two hex colours at position t ∈ [0,1].
func lerpColor(a, b lipgloss.Color, t float64) lipgloss.Color {
	ar, ag, ab := hexToRGB(string(a))
	br, bg, bb := hexToRGB(string(b))
	lerp := func(x, y uint8) uint8 {
		return uint8(float64(x) + (float64(y)-float64(x))*t)
	}
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", lerp(ar, br), lerp(ag, bg), lerp(ab, bb)))
}

func hexToRGB(hex string) (uint8, uint8, uint8) {
	hex = strings.TrimPrefix(hex, "#")
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return uint8(r), uint8(g), uint8(b)
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
			Foreground(colorTeal)

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

	styleFilterText = lipgloss.NewStyle().
			Foreground(colorText)

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
