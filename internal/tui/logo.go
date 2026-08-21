package tui

import (
	_ "embed"
	"strings"
)

// cfontsLogoRaw is the wwlog wordmark, generated once via
// `cfonts wwlog -f chrome -g cyan,magenta -t -s` (see
// scripts/generate_logo.sh) and embedded at build time. Shared by the splash
// screen and the exit banner so neither has to shell out to cfonts — a
// Node/npm tool — on every render or process exit.
//
//go:embed assets/logo.ans
var cfontsLogoRaw string

var cfontsLogo = strings.TrimRight(cfontsLogoRaw, "\n")
