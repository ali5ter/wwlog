# wwlog

Browse and export your Weight Watchers food log from the terminal.

`wwlog` is a TUI application built to let you interactively explore your tracked food log by date, view nutritional summaries, and *export data reports* — without touching the WW app.

![wwlog demo](wwlog_demo.gif)

## Installation

**Homebrew** (macOS and Linux):

```bash
brew tap ali5ter/wwlog
brew install wwlog
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

## WW API reference

[`WW_API_REFERENCE.md`](WW_API_REFERENCE.md) documents the Weight Watchers API
endpoints used by `wwlog`, reverse-engineered from live traffic. It covers
authentication, the member profile endpoint, and the `my-day` food log endpoint,
including all known fields and their meanings. Useful for developers building
tools against the WW API.

## Configuration

Optional config at `~/.config/wwlog/config.toml`:

```toml
tld       = "com"   # WW top-level domain
theme     = "auto"  # colour theme
cache_ttl = 3600    # cache TTL in seconds
```
