# wwlog

Browse and export your Weight Watchers food log from the terminal.

`wwlog` is a TUI application built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) that lets you
interactively explore your tracked food log by date, view nutritional summaries, and export reports — without
touching the WW website or app.

## Installation

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
# Open the TUI for today
wwlog

# Browse a specific date range
wwlog --start 2026-04-20 --end 2026-04-26

# Export last week's log to JSON (pipeline mode)
wwlog --start 2026-04-20 --end 2026-04-26 --json

# Clear stored credentials
wwlog --logout
```

## Key bindings

| Key | Action |
|-----|--------|
| `↑` / `↓` or `k` / `j` | Navigate dates |
| `/` | Filter dates |
| `tab` / `⇧tab` | Switch tabs |
| `^E` | Export current view |
| `q` | Quit |

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `-s`, `--start` | today | Start date (YYYY-MM-DD) |
| `-e`, `--end` | today | End date (YYYY-MM-DD) |
| `-n`, `--nutrition` | false | Include nutritional data |
| `--json` | false | Output as JSON (no TUI) |
| `--no-tty` | false | Force pipeline mode |
| `--login` | — | Authenticate and store credentials |
| `--logout` | — | Clear stored credentials |
| `-l`, `--tld` | `com` | WW domain (com, co.uk, etc.) |

## Configuration

Optional config at `~/.config/wwlog/config.toml`:

```toml
tld       = "com"   # WW top-level domain
theme     = "auto"  # colour theme
cache_ttl = 3600    # cache TTL in seconds
```
