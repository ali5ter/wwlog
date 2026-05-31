# wwlog

Your WeightWatchers food log data, permanently yours.

`wwlog` builds a local archive of your WW food log and gives you more insight into it
than the WW app provides. The archive grows every time you run it — and keeps working
long after any particular subscription or app version.

It includes a full-featured TUI for browsing daily logs, comparing nutrition, and
viewing insights, plus a pipeline mode for scripting and file export.

![wwlog demo](examples/wwlog_demo.gif)

## Features

- **Local archive** — a permanent, plain-JSON store of your food log that survives
  subscription changes, API windows, and app discontinuations
- **Offline mode** — browse, report, and export from your local archive with no
  network calls required
- **Log tab** — day-by-day food log with a points bar, meal breakdown, and per-entry kcal;
  filter by date and sort entries by points or calories
- **Nutrition tab** — nutrient bars vs. recommended daily values, per-day averages,
  and trend charts for calories, protein, carbs, and fat across the selected range
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

## Quick start

**1. Authenticate** — credentials are stored securely in your system keychain:

```bash
wwlog --login
```

**2. Archive your history** — pull the full available history into your local store:

```bash
wwlog --archive
```

Run this once to capture everything the WW API can return (~90 days). Run it again
any time to update. Your archive is plain JSON files — readable, portable, and yours.

**3. Browse or export:**

```bash
# Open the TUI (defaults to the last 7 days)
wwlog

# Browse a specific date range
wwlog --start 2026-04-20 --end 2026-04-26

# Print insights report to stdout
wwlog --start 2026-04-20 --end 2026-04-26 --report

# Output log as JSON
wwlog --start 2026-04-20 --end 2026-04-26 --json

# Export to a file (json | csv | markdown | report)
wwlog --start 2026-04-20 --end 2026-04-26 --export markdown
wwlog --start 2026-04-20 --end 2026-04-26 --export json --output ~/Downloads/
```

## Archive & offline

### Building your archive

```bash
# Pull the full API window into your local store (~90 days)
wwlog --archive

# Check what your archive covers
wwlog --status
```

`--status` output:

```text
Store     ~/Library/Application Support/wwlog/store
Coverage  2026-02-26 → 2026-05-31  (95 days)
```

`--archive` output:

```text
Archiving 90 days (2026-03-03 → 2026-05-31)
  fetched 90 / 90...
Archive complete.
  New       85 days
  Updated    5 days
  Stored    95 days total (2026-02-26 → 2026-05-31)
```

Run `--archive` regularly to keep your store current. It is safe to run repeatedly —
existing entries are updated, not duplicated.

### Browsing your archive offline

Once you have a local archive, wwlog works with no network connection:

```bash
# No API calls — serves entirely from your local archive
wwlog --offline --start 2026-01-01 --end 2026-03-31 --report
wwlog --offline --start 2026-01-01 --end 2026-03-31 --json
```

The TUI also falls back to your archive automatically when the WW API is unavailable,
surfacing a notice in the status bar rather than erroring out.

### Cloud sync

Set `store_dir` in `config.toml` to a Dropbox, iCloud Drive, or any synced folder to
share your archive across machines:

```toml
store_dir = "~/Dropbox/wwlog-store"
```

### The API retention window

The WW API enforces a hard ~90-day backwards retention limit — dates older than 89 days
return HTTP 400 regardless of account age. Your local archive is the solution: once a day
is stored locally, it's available indefinitely regardless of the API window.

If you request a range that starts before the window, wwlog serves older dates from your
archive automatically and fetches the in-window dates from the API:

```text
note: 45 day(s) loaded from local store
```

If the API is unavailable for any dates in the window, wwlog falls back to your archive
for those dates and continues:

```text
note: API unavailable for 3 day(s) — showing archived data where available
```

## JSON pipeline examples

The `--json` flag outputs a JSON array of day logs. Each element contains the date, meals
(morning / midday / evening / anytime arrays of food entries), and a points summary. Pipe it
to `jq` for quick ad-hoc analysis.

**Daily points summary:**

```bash
wwlog --start 2026-04-20 --end 2026-04-26 --json \
  | jq '.[] | {date: .Date, used: .Points.DailyUsed, target: .Points.DailyTarget}'
```

**All food names and points across the range (flat list):**

```bash
wwlog --start 2026-04-20 --end 2026-04-26 --json \
  | jq '[.[].Meals | to_entries[].value[]] | map({name: .name, pts: .pointsPrecise})'
```

**Days where you went over budget:**

```bash
wwlog --start 2026-04-20 --end 2026-04-26 --json \
  | jq '.[] | select(.Points.DailyUsed > .Points.DailyTarget)
        | {date: .Date, over: (.Points.DailyUsed - .Points.DailyTarget)}'
```

**Top foods by points for the week:**

```bash
wwlog --start 2026-04-20 --end 2026-04-26 --json \
  | jq '[.[].Meals | to_entries[].value[]]
        | group_by(.name)
        | map({name: .[0].name, total_pts: (map(.pointsPrecise) | add)})
        | sort_by(-.total_pts) | .[0:10]'
```

**Breakfast items sorted by calories:**

```bash
wwlog --start 2026-04-20 --end 2026-04-26 --json \
  | jq '[.[].Meals.morning[]
        | {name: .name,
           kcal: (.defaultPortion.nutrition.calories * (.portionSize / .defaultPortion.size))}]
        | sort_by(-.kcal)'
```

> The calorie calculation above mirrors what the app does internally — scale
> `defaultPortion.nutrition.calories` by `portionSize / defaultPortion.size`. Entries where WW
> omits calories (some protein/meat foods) will show `0`; use `--report` for accurate totals.

**Average macro distribution across the range (% of calories from protein, carbs, and fat):**

```bash
wwlog --start 2026-04-20 --end 2026-04-26 --json \
  | jq '
      [.[].Meals | to_entries[].value[]]
      | map(
          (if .defaultPortion.size > 0 then .portionSize / .defaultPortion.size else 1 end) as $scale
          | { p: (.defaultPortion.nutrition.protein * $scale),
              c: (.defaultPortion.nutrition.carbs   * $scale),
              f: (.defaultPortion.nutrition.fat     * $scale) }
        )
      | (map(.p) | add) * 4 as $pkcal
      | (map(.c) | add) * 4 as $ckcal
      | (map(.f) | add) * 9 as $fkcal
      | ($pkcal + $ckcal + $fkcal) as $total
      | { protein: "\($pkcal / $total * 100 | round)%",
          carbs:   "\($ckcal / $total * 100 | round)%",
          fat:     "\($fkcal / $total * 100 | round)%" }
    '
```

> Macros are scaled to the tracked portion size, then converted to kcal using Atwater factors
> (protein and carbs ×4, fat ×9) to match what the Insights tab shows.

All pipeline examples work with `--offline` — no WW API required:

```bash
wwlog --offline --start 2026-04-20 --end 2026-04-26 --json | jq '...'
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
| `--archive` | — | Pull the full API window into the local store |
| `--status` | — | Show local archive coverage and exit |
| `--offline` | — | Serve from local archive only (no API calls) |
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
tld         = "com"   # WW top-level domain
weight_unit = "lb"    # override weight unit: "lb" or "kg" (default: from API)
store_dir   = ""      # local archive path (default: alongside this config file)
```

## Credits

`wwlog` was inspired by [wwtracked](https://github.com/joswr1ght/wwtracked) by
[Joshua Wright](https://github.com/joswr1ght).
