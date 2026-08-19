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

# Launch the TUI with a month of data
Type "./wwlog --start 2026-05-23 --end 2026-06-22"
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
Sleep 800ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 800ms

# Focus the detail pane (border highlights, footer hint flips to
# "scroll detail") to read a longer day's meals, then return to the list
Right
Sleep 700ms
Down
Sleep 350ms
Down
Sleep 350ms
Left
Sleep 600ms

# Sort food entries: logged order → by points → by kcal
Type "s"
Sleep 900ms
Type "s"
Sleep 900ms
Type "s"
Sleep 800ms

# Filter dates
Type "/"
Sleep 300ms
Type "Jun"
Sleep 900ms
Enter
Sleep 900ms
Down
Sleep 500ms
Down
Sleep 800ms
Type "/"
Sleep 200ms
Backspace
Sleep 100ms
Backspace
Sleep 100ms
Backspace
Sleep 100ms
Enter
Sleep 500ms

# ── Nutrition tab ─────────────────────────────────────────────
Tab
Sleep 800ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 800ms

# Focus the detail pane and scroll past the nutrient bars into the
# Carbs/Fat trend charts below the fold
Right
Sleep 900ms
Down
Sleep 350ms
Down
Sleep 350ms
Down
Sleep 350ms
Down
Sleep 350ms
Down
Sleep 350ms
Down
Sleep 1000ms
Left
Sleep 600ms

# ── Insights tab ─────────────────────────────────────────────
Tab
Sleep 1300ms
PageDown
Sleep 1800ms
PageUp
Sleep 1300ms

# ── Export dialog ─────────────────────────────────────────────
Type "e"
Sleep 1500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Up
Sleep 400ms
Up
Sleep 400ms
Up
Sleep 400ms
Escape
Sleep 800ms

# ── Range dialog ─────────────────────────────────────────────
Type "r"
Sleep 1500ms
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
Type "06-10" Enter
Sleep 800ms
Enter

# Loading 6 days from the API
Sleep 7s

# Review & quit
Tab
Sleep 800ms
Tab
Sleep 800ms
Type "q"
Sleep 1s
