# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

`wwlog` is a Go TUI application for browsing and exporting a Weight Watchers food log. It is a spiritual
successor to [wwtracked](https://github.com/ali5ter/wwtracked) (Python), redesigned for mass usability:
credentials are stored in the system keychain (no DevTools required), and the primary interface is an
interactive Bubble Tea TUI rather than a CLI report generator.

## Commands

```bash
# Dependencies (run once after cloning)
go mod tidy

# Build
go build -o wwlog .

# Run
go run . --start 2026-04-20 --end 2026-04-26

# Authenticate (first-time setup)
go run . --login

# Lint
golangci-lint run ./...

# Vet
go vet ./...
```

## Architecture

```
wwlog/
├── main.go                       # Entry point — calls cmd.Execute()
├── cmd/
│   ├── root.go                   # Cobra root command, flag definitions, dispatch
│   └── tty.go                    # TTY detection (isTTY)
├── config/
│   └── config.go                 # Viper config loader (~/.config/wwlog/config.toml)
└── internal/
    ├── auth/
    │   └── auth.go               # WW auth flow + system keychain storage
    ├── api/
    │   ├── types.go              # FoodEntry, Meals, DayLog, NutritionData
    │   └── client.go             # WW API client (FetchDay, FetchNutrition, DateRange)
    ├── pipeline/
    │   └── export.go             # Non-TUI output: EmitJSON, EmitMarkdown, EmitCSV
    └── tui/
        ├── model.go              # Top-level Bubble Tea model, tab routing, header/status
        ├── styles.go             # Lip Gloss colour palette + all style constants
        ├── keys.go               # Centralised key bindings
        ├── log.go                # Log tab: date list (left) + meal detail viewport (right)
        └── nutrition.go          # Nutrition tab: summary viewport (WIP)
```

## Execution modes

`cmd/root.go` dispatches to one of two modes based on flags and TTY detection:

- **TUI mode** (default when stdout is a terminal): launches Bubble Tea with async data loading
- **Pipeline mode** (`--json`, `--no-tty`, or piped stdout): loads data synchronously, emits JSON

## Authentication flow

`internal/auth/auth.go` implements a two-step WW login (mirroring the browser flow):

1. **Step 1** — POST `email` + `password` to
   `auth.weightwatchers.{tld}/login-apis/v1/authenticate` → returns `tokenId`
2. **Step 2** — GET `auth.weightwatchers.{tld}/openam/oauth2/authorize` with `wwAuth2={tokenId}`
   cookie → 302 redirect with `id_token` JWT in the URL fragment

The JWT is stored in the **system keychain** (via `go-keyring`) under service `wwlog`. On next run,
`Auth.Token()` reads the cached JWT, checks its `exp` claim, and only re-authenticates if it has
less than 5 minutes of validity remaining. Credentials (email + password) are also stored so
re-authentication is fully automatic and silent.

## WW API endpoints

All endpoints require `Authorization: Bearer {jwt}`.

| Purpose | Endpoint |
|---------|----------|
| Daily food log | `GET cmx.weightwatchers.{tld}/api/v3/cmx/operations/composed/members/~/my-day/{YYYY-MM-DD}` |
| Food nutrition | `GET cmx.weightwatchers.{tld}/api/v3/public/foods/{id}?fullDetails=true` |
| Member food | `GET cmx.weightwatchers.{tld}/api/v3/cmx/members/~/custom-foods/foods/{id}?fullDetails=true` |
| Recipe nutrition | `GET cmx.weightwatchers.{tld}/api/v3/public/recipes/{id}?fullDetails=true` |
| Member recipe | `GET cmx.weightwatchers.{tld}/api/v3/cmx/members/~/custom-foods/recipes/{id}?fullDetails=true` |

`SourceType` on each `FoodEntry` determines which endpoint to use. `MEMBERFOODQUICK` (quick-add)
entries have no nutrition endpoint and are skipped.

## TUI structure

The TUI uses the Elm architecture via Bubble Tea. `model.go` is the top-level orchestrator:

- **Init**: fires the spinner tick and an async data fetch command
- **Update**: routes `tea.KeyMsg` to tab switching or the active tab's `update()` method
- **View**: renders header + active tab content + status bar

Each tab (`logModel`, `nutriModel`) implements `update(tea.Msg) (T, tea.Cmd)` and `view() string`,
and a `resize(width, height int)` method called on `tea.WindowSizeMsg`.

### Log tab layout

```
┌─────────────────┬────────────────────────────────────┐
│  Date list      │  Meal detail viewport               │
│  (bubbles/list) │  Breakfast / Lunch / Dinner / Snacks│
│  width/3        │  remaining width                    │
└─────────────────┴────────────────────────────────────┘
```

## Charm stack

| Library | Role |
|---------|------|
| `bubbletea` | Elm-architecture TUI runtime |
| `bubbles` | `list.Model`, `viewport.Model`, `spinner.Model` |
| `lipgloss` | Styling, layout, colour palette |
| `glamour` | Markdown rendering (planned for export preview) |

## Colour palette

WW-inspired, defined in `internal/tui/styles.go`:

| Variable | Hex | Use |
|----------|-----|-----|
| `colorTeal` | `#00B388` | WW signature green — active, selected |
| `colorPurple` | `#6B4C9A` | WW app purple — prompts, pointers, errors |
| `colorSteel` | `#7f93a6` | Labels, secondary info |
| `colorMuted` | `#a8b6c0` | Inactive items, portion text, borders |
| `colorText` | `#e9eff3` | Primary food item text |
| `colorPanel` | `#161d24` | Header/status bar background |
| `colorLine` | `#2b3742` | Borders, separators |

## Key design decisions

- **Keychain over DevTools**: the biggest UX improvement over `wwtracked`. Users authenticate once;
  token refresh is silent and automatic.
- **Same WW API**: the backend API endpoints are unchanged from `wwtracked`. The TLD flag (`--tld`)
  supports non-US regions (e.g. `co.uk`).
- **Pipeline mode**: piping `wwlog | jq` works naturally — TTY detection switches to JSON output.
- **Nutrition is expensive**: each food entry requires a separate API call. It's opt-in (`--nutrition`)
  and not fetched during TUI browsing to keep load times fast.
- **No external config required for auth**: `--login` is the only setup step.

## Sibling project

`wwtracked` (Python, `../wwtracked/`) is the original script this project supersedes. The WW API
knowledge there remains accurate and was the primary reference for `internal/api/`.
