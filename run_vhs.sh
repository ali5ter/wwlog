#!/usr/bin/env bash

# @file run_vhs.sh
# @brief Generate git animations using vhs
# @ref https://github.com/charmbracelet/vhs

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