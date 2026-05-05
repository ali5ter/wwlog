Output wwlog_demo.gif

Set FontSize 13
Set Width 1280
Set Height 720
Set Padding 10
Set Shell bash

Hide
Type "cd .. && go build -o wwlog . 2>/dev/null" Enter
Sleep 4s
Type `PS1="\[\e[38;5;63m\]❯\[\e[0m\] "` Enter
Type "clear" Enter
Sleep 200ms
Show

# Launch the TUI with a week of data
Type "./wwlog --start 2026-03-30 --end 2026-04-30"
Sleep 600ms
Enter

# Splash: credential check spinner
Sleep 1200ms

# Date form appears with dates pre-filled
# Enter advances From → To, Enter again submits
Enter
Sleep 300ms
Enter
Sleep 300ms
Enter

# Loading 30 days from the API
Sleep 12s

# ── Log tab ──────────────────────────────────────────────────
Sleep 1s
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
Sleep 1s
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
Sleep 1s
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Up
Sleep 200ms
Up
Sleep 200ms
Up
Sleep 200ms
Up
Sleep 200ms
Up
Sleep 800ms

# ── Export dialog ─────────────────────────────────────────────
Type "e"
Sleep 2s
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 600ms
Up
Sleep 500ms
Up
Sleep 500ms
Up
Sleep 500ms
Escape
Sleep 1s

# ── Range dialog ─────────────────────────────────────────────
Type "r"
Sleep 2s
Backspace
Sleep 100ms
Backspace
Sleep 100ms
Backspace
Sleep 100ms
Backspace
Sleep 100ms
Backspace
Sleep 100ms
Type "04-23" Enter
Sleep 1s
Enter

# Loading 6 days from the API
Sleep 7s

# Review & quit
Tab
Sleep 1s
Tab
Sleep 1s
Type "q"
Sleep 1s
