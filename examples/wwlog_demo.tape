Output wwlog_demo.gif

Set FontSize 13
Set Width 1350
Set Height 780
Set Padding 10
Set Shell bash

Hide
Type "cd /Users/alister/Documents/Projects/wwlog" Enter
Type "go build -o wwlog . 2>/dev/null" Enter
Sleep 4s
Type `PS1="\[\e[38;5;63m\]❯\[\e[0m\] "` Enter
Type "clear" Enter
Sleep 200ms
Show

# Launch the TUI with a week of data
Type "./wwlog --start 2026-04-22 --end 2026-04-27"
Sleep 600ms
Enter

# Splash: credential check spinner
Sleep 1200ms

# Date form appears with dates pre-filled
# Enter advances From → To, Enter again submits
Enter
Sleep 300ms
Enter

# Loading 6 days from the API
Sleep 7s

# ── Log tab ──────────────────────────────────────────────────
Sleep 800ms
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 800ms
Up
Sleep 500ms
Up
Sleep 500ms
Up
Sleep 500ms
Up
Sleep 500ms
Up
Sleep 800ms

# ── Nutrition tab ─────────────────────────────────────────────
Tab
Sleep 1000ms
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 800ms

# ── Insights tab ─────────────────────────────────────────────
Tab
Sleep 1500ms

# ── Export dialog ─────────────────────────────────────────────
Ctrl+e
Sleep 1500ms
Escape
Sleep 800ms

# Quit
Type "q"
Sleep 1s
