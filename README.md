# wwlog

`wwlog` unlocks the WeightWatchers data you enter in the mobile app.

It provides a CLI for exporting text, Markdown, JSON, and CSV so you can analyze your data with your own
tools across a selected date range.

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

## Data window

The WW `my-day` endpoint enforces a hard ~90-day backwards retention window — any date more
than 89 days before today returns HTTP 400, regardless of how long you've held your WW account.
This is a server-side policy, not a wwlog limitation.

If `--start` is older than the window, wwlog clamps it forward and prints a one-line notice:

```bash
$ wwlog --start 2024-01-01 --end 2026-05-05 --report
note: --start clamped to 2026-02-05 (WW retains ~90 days)
```

If `--end` is also older than the window, wwlog errors out before making any API calls. There's
no workaround — long-running history exports aren't possible with this API. Plan ahead if you
want to keep more than ~3 months of records: pull `--export json` regularly and archive the files.

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
