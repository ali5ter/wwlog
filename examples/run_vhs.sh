#!/usr/bin/env bash
#
# @name run_vhs.sh
# @brief Generate demo animations using vhs
# @author Alister Lewis-Bowen <alister@lewis-bowen.org>
# @version 0.1.0
# @date 2026-05-06
# @license MIT
#
# @usage run_vhs.sh [tape]
#   tape  Path to a .tape file, or omit to process all *.tape files in the current directory
#
# @dependencies
#   vhs  https://github.com/charmbracelet/vhs
#
# @exit_codes
#   0  Success
#   1  vhs not installed

tape=${1:-"all"}

type vhs &>/dev/null || {
  echo "vhs is not installed. Refer to https://github.com/charmbracelet/vhs for installation instructions."
  exit 1
}

# https://github.com/charmbracelet/vhs/issues/419
unset PROMPT_COMMAND

if [[ "$tape" != "all" ]]; then
  vhs "$tape"
else
  for tape in *.tape; do
    # Skipe sourced vhs configuration file
    [[ "$tape" == "config.tape" ]] && continue
    vhs "$tape"
  done
fi