package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ali5ter/wwlog/internal/api"
	"github.com/ali5ter/wwlog/internal/pipeline"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type exportModel struct {
	form   *huh.Form
	width  int
	height int
}

type exportDoneMsg struct {
	filename string
	err      error
}

func validateDir(s string) error {
	expanded := expandHome(s)
	info, err := os.Stat(expanded)
	if err != nil {
		return fmt.Errorf("directory not found")
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}

func expandHome(s string) string {
	if strings.HasPrefix(s, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, s[2:])
		}
	}
	return s
}

func newExportModel(width, height int) exportModel {
	w := width - 8
	if w > 60 {
		w = 60
	}
	if w < 44 {
		w = 44
	}

	home, _ := os.UserHomeDir()
	dir := home
	if cwd, err := os.Getwd(); err == nil {
		dir = cwd
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("format").
				Title("Export format").
				Description("Choose how to save your food log").
				Options(
					huh.NewOption("Text      insights summary", "report"),
					huh.NewOption("JSON      full structured data", "json"),
					huh.NewOption("Markdown  readable daily report", "md"),
					huh.NewOption("CSV       food log entries", "csv"),
				),
			huh.NewInput().
				Key("dir").
				Title("Save to directory").
				Description("Where to save the file (~ for home)").
				Value(&dir).
				Validate(validateDir),
		),
	).WithTheme(wwHuhTheme()).WithWidth(w).WithShowHelp(true)

	return exportModel{form: form, width: width, height: height}
}

func (m *exportModel) resize(width, height int) {
	m.width = width
	m.height = height
}

func (m exportModel) update(msg tea.Msg) (exportModel, tea.Cmd) {
	model, cmd := m.form.Update(msg)
	m.form = model.(*huh.Form)
	return m, cmd
}

func (m exportModel) view() string {
	content := lipgloss.JoinVertical(lipgloss.Center,
		renderGradientLogo(), "",
		styleSplashTitle.Render("Export your log"),
		styleSplashSub.Render("Choose a format and directory, then press enter"), "",
		m.form.View(), "",
		styleSplashHint.Render("esc to cancel · ctrl+c to quit"),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func runExport(format, dir, start, end string, logs []*api.DayLog) tea.Cmd {
	return func() tea.Msg {
		ext := format
		if ext == "report" {
			ext = "txt"
		}
		dest := expandHome(dir)
		filename := filepath.Join(dest, fmt.Sprintf("wwlog-%s_%s.%s", start, end, ext))
		f, err := os.Create(filename)
		if err != nil {
			return exportDoneMsg{err: fmt.Errorf("create %s: %w", filename, err)}
		}
		defer f.Close()

		switch format {
		case "report":
			err = pipeline.EmitTextReport(f, logs)
		case "json":
			err = pipeline.WriteJSON(f, logs)
		case "md":
			err = pipeline.EmitMarkdown(f, logs)
		case "csv":
			err = pipeline.WriteLogCSV(f, logs)
		}
		if err != nil {
			return exportDoneMsg{err: fmt.Errorf("write %s: %w", filename, err)}
		}
		return exportDoneMsg{filename: filename}
	}
}
