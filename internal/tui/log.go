package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ali5ter/wwlog/internal/api"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sortMode int

const (
	sortLogged  sortMode = iota // logged order (default)
	sortByPts                   // highest WW points first
	sortByKcal                  // highest calories first
)

func (s sortMode) label() string {
	switch s {
	case sortByPts:
		return "sorted by points"
	case sortByKcal:
		return "sorted by kcal"
	default:
		return ""
	}
}

func (s sortMode) next() sortMode {
	return (s + 1) % 3
}

// logModel is the food log tab — a date list on the left, meal detail on the right.
type logModel struct {
	list        list.Model
	filter      textinput.Model
	filtering   bool
	detail      viewport.Model
	allLogs     []*api.DayLog
	logs        []*api.DayLog // filtered view
	width       int
	height      int
	selected    int
	sort        sortMode
	initialized bool
}

type dateItem struct {
	log *api.DayLog
}

func (d dateItem) Title() string       { return formatDateShort(d.log.Date) }
func (d dateItem) Description() string { return mealSummary(d.log) }
func (d dateItem) FilterValue() string { return d.log.Date }

func newLogModel(logs []*api.DayLog, width, height int) logModel {
	listWidth := width / 3
	detailWidth := width - listWidth
	listHeight := height - 2 // filter bar + separator

	items := make([]list.Item, len(logs))
	for i, l := range logs {
		items[i] = dateItem{log: l}
	}

	l := list.New(items, list.NewDefaultDelegate(), listWidth, listHeight)
	l.Title = "Dates"
	l.Styles.Title = styleMealHeading
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	fi := textinput.New()
	fi.Placeholder = "filter by date (e.g. Jan, 04)"
	fi.PromptStyle = styleFilterPrompt
	fi.TextStyle = styleFilterText
	fi.Prompt = "> "

	vp := viewport.New(detailWidth, height)

	m := logModel{
		list:        l,
		filter:      fi,
		detail:      vp,
		allLogs:     logs,
		logs:        logs,
		width:       width,
		height:      height,
		initialized: true,
	}
	if len(logs) > 0 {
		m.detail.SetContent(renderDay(logs[0], detailWidth-2, m.sort))
	}
	return m
}

func (m logModel) update(msg tea.Msg) (logModel, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok {
		if m.filtering {
			switch msg.String() {
			case "enter", "esc":
				m.filtering = false
				m.filter.Blur()
				m.applyFilter()
				return m, nil
			case "up", "k", "down", "j":
				m.filtering = false
				m.filter.Blur()
				m.applyFilter()
				// fall through to list navigation below
			default:
				m.filter, cmd = m.filter.Update(msg)
				cmds = append(cmds, cmd)
				m.applyFilter()
				return m, tea.Batch(cmds...)
			}
		} else {
			switch {
			case key.Matches(msg, keys.Filter):
				m.filtering = true
				m.filter.Focus()
				return m, textinput.Blink
			case key.Matches(msg, keys.ScrollUp):
				m.detail.LineUp(3)
				return m, nil
			case key.Matches(msg, keys.ScrollDown):
				m.detail.LineDown(3)
				return m, nil
			case key.Matches(msg, keys.Sort):
				m.sort = m.sort.next()
				if m.selected < len(m.logs) {
					m.detail.SetContent(renderDay(m.logs[m.selected], m.detail.Width-2, m.sort))
					m.detail.GotoTop()
				}
				return m, nil
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	selChanged := false
	if i := m.list.Index(); i != m.selected && i < len(m.logs) {
		m.selected = i
		m.detail.SetContent(renderDay(m.logs[i], m.detail.Width-2, m.sort))
		m.detail.GotoTop()
		selChanged = true
	}

	if !selChanged {
		// Only forward non-key messages to the viewport. Key messages are
		// handled above (ScrollUp/ScrollDown) or consumed by the list.
		// Forwarding keys here causes the viewport to scroll when the list
		// is at its first or last item and the navigation key has no effect.
		if _, isKey := msg.(tea.KeyMsg); !isKey {
			m.detail, cmd = m.detail.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *logModel) applyFilter() {
	q := strings.ToLower(m.filter.Value())
	var filtered []*api.DayLog
	for _, l := range m.allLogs {
		if q == "" ||
			strings.Contains(strings.ToLower(l.Date), q) ||
			strings.Contains(strings.ToLower(formatDateShort(l.Date)), q) {
			filtered = append(filtered, l)
		}
	}
	m.logs = filtered
	items := make([]list.Item, len(filtered))
	for i, l := range filtered {
		items[i] = dateItem{log: l}
	}
	m.list.SetItems(items)
	m.selected = 0
	if len(filtered) > 0 {
		m.detail.SetContent(renderDay(filtered[0], m.detail.Width-2, m.sort))
	} else {
		m.detail.SetContent(styleFoodPortion.Render("No matching dates."))
	}
	m.detail.GotoTop()
}

func (m logModel) view() string {
	listWidth := m.width / 3
	detailWidth := m.width - listWidth

	var filterBar string
	if m.filtering {
		filterBar = m.filter.View()
	} else if m.filter.Value() != "" {
		filterBar = styleFilterPrompt.Render("> ") + styleFilterText.Render(m.filter.Value())
	} else {
		filterBar = styleDim.Render("> filter by date…")
	}
	filterSep := styleDim.Render(strings.Repeat("─", listWidth-1))

	listPane := stylePanelBorder.Width(listWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, filterBar, filterSep, m.list.View()),
	)
	detailPane := lipgloss.NewStyle().Width(detailWidth).Padding(0, 1).Render(m.detail.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, detailPane)
}

func (m *logModel) resize(width, height int) {
	if !m.initialized {
		return
	}
	m.width = width
	m.height = height
	listWidth := width / 3
	listHeight := height - 2
	m.list.SetSize(listWidth, listHeight)
	detailWidth := width - listWidth
	m.detail.Width = detailWidth
	m.detail.Height = height
	if m.selected < len(m.logs) && len(m.logs) > 0 {
		m.detail.SetContent(renderDay(m.logs[m.selected], detailWidth-2, m.sort))
	}
}

func renderPointsSummary(b *strings.Builder, pts api.DayPoints, contentWidth int) {
	if pts.DailyTarget == 0 {
		return
	}
	barWidth := 20
	bar := makeBar(pts.DailyUsed, pts.DailyTarget, barWidth)

	usedStr := styleDetailValue.Render(fmt.Sprintf("%.0f", pts.DailyUsed))
	targetStr := styleFoodPortion.Render(fmt.Sprintf("/ %.0f", pts.DailyTarget))
	remStr := styleFoodPortion.Render(fmt.Sprintf("  %.0f left", pts.DailyRemaining))

	dailyLabel := lipgloss.NewStyle().Width(8).Render(styleDetailLabel.Render("Points"))
	fmt.Fprintf(b, "  %s  %s  %s %s%s\n", dailyLabel, bar, usedStr, targetStr, remStr)

	var meta []string
	if pts.WeeklyAllowanceRemaining != 0 {
		meta = append(meta, fmt.Sprintf("Weekly bank %+.0f", pts.WeeklyAllowanceRemaining))
	}
	if pts.ActivityEarned != 0 {
		meta = append(meta, fmt.Sprintf("Activity +%.0f earned", pts.ActivityEarned))
	}
	if pts.Weight > 0 {
		meta = append(meta, fmt.Sprintf("Weight %.1f %s", pts.Weight, pts.WeightUnit))
	}
	if len(meta) > 0 {
		fmt.Fprintf(b, "  %s\n", styleFoodPortion.Render(strings.Join(meta, "  ·  ")))
	}
	fmt.Fprintln(b)
}

func sortEntries(entries []api.FoodEntry, mode sortMode) []api.FoodEntry {
	if mode == sortLogged || len(entries) == 0 {
		return entries
	}
	cp := make([]api.FoodEntry, len(entries))
	copy(cp, entries)
	sort.Slice(cp, func(i, j int) bool {
		switch mode {
		case sortByPts:
			return entryPoints(cp[i]) > entryPoints(cp[j])
		case sortByKcal:
			return cp[i].Nutrition().Calories > cp[j].Nutrition().Calories
		}
		return false
	})
	return cp
}

func renderDay(day *api.DayLog, width int, mode sortMode) string {
	var b strings.Builder
	sepWidth := width - 2
	if sepWidth < 1 {
		sepWidth = 1
	}
	heading := formatDateLong(day.Date)
	if lbl := mode.label(); lbl != "" {
		heading += styleFoodPortion.Render("  (" + lbl + ")")
	}
	fmt.Fprintf(&b, "%s\n", styleMealHeading.Render(heading))
	fmt.Fprintf(&b, "%s\n\n", styleDim.Render(strings.Repeat("─", sepWidth)))
	renderPointsSummary(&b, day.Points, width)
	renderMeal(&b, "☀  Breakfast", sortEntries(day.Meals.Morning, mode))
	renderMeal(&b, "☁  Lunch", sortEntries(day.Meals.Midday, mode))
	renderMeal(&b, "☽  Dinner", sortEntries(day.Meals.Evening, mode))
	renderMeal(&b, "✦  Snacks", sortEntries(day.Meals.Anytime, mode))
	return b.String()
}

func renderMeal(b *strings.Builder, name string, entries []api.FoodEntry) {
	fmt.Fprintf(b, "%s\n", styleDetailLabel.Render(name))
	if len(entries) == 0 {
		fmt.Fprintf(b, "  %s\n", styleFoodPortion.Render("Nothing logged"))
	}
	for _, e := range entries {
		serving := e.ServingDesc
		if serving == "" && e.PortionName != "" {
			serving = fmt.Sprintf("%s %s", formatPortion(e.PortionSize), e.PortionName)
		}

		pts := entryPoints(e)
		ptsStr := styleStatusKey.Render(fmt.Sprintf("  %.0fpt", pts))

		cal := e.Nutrition().Calories
		calStr := ""
		if cal > 0 {
			calStr = styleFoodPortion.Render(fmt.Sprintf("  %.0f kcal", cal))
		}

		servingStr := ""
		if serving != "" {
			servingStr = styleFoodPortion.Render("  " + serving)
		}

		fmt.Fprintf(b, "  %s%s%s%s\n",
			styleFoodItem.Render(truncate(e.Name, 35)),
			servingStr, ptsStr, calStr)
	}
	fmt.Fprintln(b)
}

func entryPoints(e api.FoodEntry) float64 {
	return math.Round(e.PointsPrecise)
}

func mealSummary(day *api.DayLog) string {
	pts := day.Points
	if pts.DailyTarget == 0 {
		return ""
	}
	s := fmt.Sprintf("%.0fpt / %.0fpt", pts.DailyUsed, pts.DailyTarget)
	if pts.Weight > 0 {
		s += fmt.Sprintf("  ·  %.1f %s", pts.Weight, pts.WeightUnit)
	}
	return s
}

func formatDateShort(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.Format("Mon, 2 Jan")
}

func formatDateLong(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.Format("Monday 2 January 2006")
}

func formatPortion(size float64) string {
	if size == float64(int(size)) {
		return fmt.Sprintf("%g", size)
	}
	return fmt.Sprintf("%.1f", size)
}
