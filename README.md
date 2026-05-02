# wwlog

`wwlog` unlocks the WeightWatchers data you enter in the mobile app.

It provides a CLI for exporting text, Markdown, JSON, and CSV so you can analyze your data with your own tools across a selected date range.

It also includes a TUI for browsing daily logs, comparing nutrition, and viewing insights across a selected date range.

![wwlog demo](examples/wwlog_demo.gif)

## Features

- **Log tab** — day-by-day food log with a points bar, meal breakdown, and per-entry kcal;
  filter by date and sort entries by points or calories
- **Nutrition tab** — nutrient bars vs. recommended daily values, per-day averages,
  and asciigraph trend charts for calories, protein, carbs, and fat across the selected range
- **Insights tab** — a calendar heatmap of daily points budget, range summary,
  points by meal, macro distribution, top foods by points, and a zero-point food log
- **Pipeline mode** — `--json`, `--report`, and `--export` flags for scripting and file output
- **No DevTools required** — one `--login` step stores credentials in your system keychain

## Installation

**Homebrew** (macOS):

```bash
brew install ali5ter/tap/wwlog
```

**Binary** (macOS and Linux, no Go required):

```bash
# macOS Apple Silicon
curl -sL https://github.com/ali5ter/wwlog/releases/latest/download/wwlog_darwin_arm64.tar.gz | tar -xz
sudo mv wwlog /usr/local/bin/

# macOS Intel
curl -sL https://github.com/ali5ter/wwlog/releases/latest/download/wwlog_darwin_amd64.tar.gz | tar -xz
sudo mv wwlog /usr/local/bin/

# Linux arm64
curl -sL https://github.com/ali5ter/wwlog/releases/latest/download/wwlog_linux_arm64.tar.gz | tar -xz
sudo mv wwlog /usr/local/bin/

# Linux amd64
curl -sL https://github.com/ali5ter/wwlog/releases/latest/download/wwlog_linux_amd64.tar.gz | tar -xz
sudo mv wwlog /usr/local/bin/
```

**Go**:

```bash
go install github.com/ali5ter/wwlog@latest
```

## First-time setup

Authenticate once — credentials are stored securely in your system keychain:

```bash
wwlog --login
```

## Usage

```bash
# Open the TUI (defaults to the last 7 days)
wwlog

# Browse a specific date range
wwlog --start 2026-04-20 --end 2026-04-26

# Print insights report to stdout (no TUI)
wwlog --start 2026-04-20 --end 2026-04-26 --report

# Output log as JSON to stdout
wwlog --start 2026-04-20 --end 2026-04-26 --json

# Export to a file (JSON | CSV | Markdown | report)
wwlog --start 2026-04-20 --end 2026-04-26 --export markdown
wwlog --start 2026-04-20 --end 2026-04-26 --export json --output ~/Downloads/

# Clear stored credentials
wwlog --logout
```

## Key bindings

| Key | Action |
|-----|--------|
| `↑` / `↓` or `k` / `j` | Navigate dates |
| `⇧↑` / `⇧↓` | Scroll detail pane |
| `/` | Filter dates (Log and Nutrition tabs) |
| `s` | Cycle sort order (logged → by points → by kcal) |
| `r` | Change date range |
| `e` | Export (opens format picker) |
| `tab` / `⇧tab` | Switch tabs |
| `q` or `ctrl+c` | Quit |

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `-s`, `--start` | 7 days ago | Start date (YYYY-MM-DD) |
| `-e`, `--end` | today | End date (YYYY-MM-DD) |
| `--json` | — | Output log as JSON to stdout (no TUI) |
| `-r`, `--report` | — | Output insights report as text to stdout (no TUI) |
| `--export` | — | Export to file: `json`, `csv`, `markdown`, or `report` |
| `-o`, `--output` | `reports/` | Output file or directory for `--export` |
| `--no-tty` | — | Force pipeline mode even in a terminal |
| `--login` | — | Authenticate and store credentials |
| `--logout` | — | Clear stored credentials |
| `-l`, `--tld` | `com` | WW top-level domain (`com`, `co.uk`, etc.) |

## Configuration

Optional config at `~/.config/wwlog/config.toml`:

```toml
tld       = "com"   # WW top-level domain
```

## Credits

`wwlog` was inspired by [wwtracked](https://github.com/joswr1ght/wwtracked) by
[Joshua Wright](https://github.com/joswr1ght).
