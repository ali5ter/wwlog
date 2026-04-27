# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

`wwlog` is a Go TUI application for browsing and exporting a Weight Watchers food log. It is a
successor to [wwtracked](https://github.com/ali5ter/wwtracked) (a Python script in the sibling
directory `../wwtracked/`), redesigned for mass usability:

- **No DevTools required** — credentials are stored in the system keychain via one `--login` step
- **Interactive TUI** — Bubble Tea date browser with meal detail view, replacing stdout Markdown
- **Pipeline mode** — piping or `--json` flag emits JSON for scripting
- **Same WW API** — all endpoint knowledge was ported directly from `wwtracked.py`

## Current state

This is a fresh scaffold — the architecture and all packages are in place but the project has
**not yet been compiled or tested**. The immediate next steps are:

1. Run `go mod tidy` to fetch dependencies and generate `go.sum`
2. Run `go build -o wwlog .` and fix any compilation errors
3. Test `./wwlog --login` to verify the keychain auth flow works
4. Test `./wwlog --start 2026-04-20 --end 2026-04-26` to verify the TUI loads
5. Push to a new GitHub repo (`github.com/ali5ter/wwlog`)

## Commands

```bash
go mod tidy                            # fetch dependencies (required first)
go build -o wwlog .                    # build binary
go run . --login                       # authenticate (stores creds in keychain)
go run . --start 2026-04-20 --end 2026-04-26   # run TUI for a date range
go run . --start 2026-04-20 --end 2026-04-26 --json  # pipeline/JSON mode
go vet ./...                           # vet
golangci-lint run ./...                # lint (if installed)
```

## Architecture

```
wwlog/
├── main.go                       # Entry point — calls cmd.Execute()
├── cmd/
│   ├── root.go                   # Cobra root command, all flags, dispatch logic
│   └── tty.go                    # TTY detection (isTTY)
├── config/
│   └── config.go                 # Viper config loader (~/.config/wwlog/config.toml)
└── internal/
    ├── auth/
    │   └── auth.go               # WW auth + system keychain storage
    ├── api/
    │   ├── types.go              # FoodEntry, Meals, DayLog, NutritionData, SourceType constants
    │   └── client.go             # WW API client: FetchDay, FetchNutrition, DateRange
    ├── pipeline/
    │   └── export.go             # EmitJSON, EmitMarkdown, EmitCSV
    └── tui/
        ├── model.go              # Top-level Bubble Tea model, tab routing, header/status bar
        ├── styles.go             # Lip Gloss colour palette + all style variables
        ├── keys.go               # Centralised key bindings
        ├── log.go                # Log tab: bubbles/list (dates) + viewport (meal detail)
        └── nutrition.go          # Nutrition tab: placeholder viewport, needs wiring up
```

## Execution flow

`cmd/root.go` handles two modes:

- **TUI mode** — default when stdout is a terminal; async data fetch with spinner
- **Pipeline mode** — `--json`, `--no-tty`, or piped stdout; sync fetch, JSON to stdout

## Authentication (the key improvement over wwtracked)

`internal/auth/auth.go` — two-step WW login matching the browser flow:

1. POST `email` + `password` → `auth.weightwatchers.{tld}/login-apis/v1/authenticate` → `tokenId`
2. GET `auth.weightwatchers.{tld}/openam/oauth2/authorize` with `wwAuth2={tokenId}` cookie
   → 302 redirect with `id_token` JWT in the URL fragment

JWT and credentials are stored in the **system keychain** (via `go-keyring`, service name `wwlog`).
`Auth.Token()` checks the JWT `exp` claim and silently re-authenticates if within 5 minutes of
expiry. Users never need to touch DevTools.

## WW API endpoints

All require `Authorization: Bearer {jwt}`.

| Purpose | Endpoint |
|---------|----------|
| Daily food log | `GET cmx.weightwatchers.{tld}/api/v3/cmx/operations/composed/members/~/my-day/{YYYY-MM-DD}` |
| Food nutrition | `GET cmx.weightwatchers.{tld}/api/v3/public/foods/{id}?fullDetails=true` |
| Member food | `GET cmx.weightwatchers.{tld}/api/v3/cmx/members/~/custom-foods/foods/{id}?fullDetails=true` |
| Recipe | `GET cmx.weightwatchers.{tld}/api/v3/public/recipes/{id}?fullDetails=true` |
| Member recipe | `GET cmx.weightwatchers.{tld}/api/v3/cmx/members/~/custom-foods/recipes/{id}?fullDetails=true` |

`SourceType` on each `FoodEntry` determines which endpoint to use. `MEMBERFOODQUICK` entries are
skipped — they have no nutrition endpoint.

## TUI structure

Built with the Charm stack, following the same conventions as the sibling project `../clu/`:

- Top-level `Model` in `model.go` owns all state and routes to tab sub-models
- Each tab implements `update(tea.Msg) (T, tea.Cmd)`, `view() string`, `resize(w, h int)`
- Styles are centralised in `styles.go` — never inline Lip Gloss in logic files
- Key bindings are centralised in `keys.go`

### Log tab layout

```
┌──────────────────┬───────────────────────────────────┐
│  Date list       │  Meal detail viewport              │
│  bubbles/list    │  Breakfast / Lunch / Dinner / Snacks
│  width/3         │  remaining width                   │
└──────────────────┴───────────────────────────────────┘
```

## Colour palette (styles.go)

| Variable | Hex | Use |
|----------|-----|-----|
| `colorTeal` | `#00B388` | WW signature green — active, selected |
| `colorPurple` | `#6B4C9A` | WW app purple — prompts, errors |
| `colorSteel` | `#7f93a6` | Labels, secondary info |
| `colorMuted` | `#a8b6c0` | Inactive items, portion text |
| `colorText` | `#e9eff3` | Primary food item text |
| `colorPanel` | `#161d24` | Header/status bar background |
| `colorLine` | `#2b3742` | Borders, separators |

## Known gaps to address

- `internal/tui/nutrition.go` — placeholder only; needs nutrition data fetched and rendered
- `cmd/root.go` — `--nutrition` flag is plumbed but nutrition fetch not wired into the TUI path
- No `go.sum` yet — run `go mod tidy` first
- No GitHub repo yet — needs creating at `github.com/ali5ter/wwlog` and pushing
- No GoReleaser config yet — see `../clu/.goreleaser.yml` for the pattern to follow
- Export (`^E` key binding) is defined but not yet implemented in the TUI

## Sibling projects

- `../wwtracked/` — original Python script; source of WW API knowledge
- `../clu/` — Go + Charm project this one follows for conventions, structure, and style
