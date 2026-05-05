package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/ali5ter/wwlog/internal/api"
	"github.com/ali5ter/wwlog/internal/pipeline"
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
	w := dialogContentWidth(width)

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
			huh.NewConfirm().Key("confirmed").Affirmative("Save").Negative("Cancel"),
		),
	).WithTheme(wwHuhTheme{}).WithWidth(w).WithShowHelp(false)

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
	return renderDialog("Export your log", m.form.View(), "esc cancel · enter submit · tab next field")
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
