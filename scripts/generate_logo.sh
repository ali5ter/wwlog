#!/usr/bin/env bash
#
# generate_logo.sh - Regenerate the embedded wwlog cfonts logo
#
# Captures `cfonts wwlog -f chrome -g cyan,magenta -t -s` output to
# internal/tui/assets/logo.ans, which internal/tui/logo.go embeds at build
# time via go:embed. The Go build never shells out to cfonts itself — this
# script is the only place that does. Re-run it whenever the logo text,
# font, or gradient colors change.
#
# Author: Alister Lewis-Bowen <alister@lewis-bowen.org>
# Version: 1.0.0
# Date: 2026-08-21
# License: MIT
#
# Usage: ./scripts/generate_logo.sh
#
# Dependencies: bash 4.0+, cfonts (npm install -g cfonts)
#
# Exit codes:
#   0 - Success
#   1 - cfonts not installed

set -euo pipefail

readonly BOLD=$'\033[1m'
readonly GREEN=$'\033[32m'
readonly RED=$'\033[31m'
readonly RESET=$'\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
readonly REPO_ROOT
readonly LOGO_FILE="${REPO_ROOT}/internal/tui/assets/logo.ans"
readonly LOGO_TEXT="wwlog"
readonly LOGO_FONT="chrome"
readonly LOGO_GRADIENT="cyan,magenta"

echo "${BOLD}Generate wwlog logo${RESET}"

if ! command -v cfonts &>/dev/null; then
    echo "${RED}error:${RESET} cfonts not found on PATH" >&2
    echo "Install with: npm install -g cfonts" >&2
    exit 1
fi

cfonts "$LOGO_TEXT" -f "$LOGO_FONT" -g "$LOGO_GRADIENT" -t -s > "$LOGO_FILE"

echo "${GREEN}✓${RESET} wrote ${LOGO_FILE#"${REPO_ROOT}"/}"
