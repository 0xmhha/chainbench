#!/usr/bin/env bash
# lib/cmd_remote.sh - Manage remote chain RPC connections (dispatcher).
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.
#
# Sub-subcommands: add / list / remove / select / info
# The handler implementations live in lib/remote_commands.sh (P2-2a split).

# Guard against double-sourcing
[[ -n "${_CB_CMD_REMOTE_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_REMOTE_SH_LOADED=1

_CB_REMOTE_CMD_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/remote_commands.sh
source "${_CB_REMOTE_CMD_LIB_DIR}/remote_commands.sh"

# ---- Subcommand dispatcher ---------------------------------------------------

cmd_remote_main() {
  if [[ $# -lt 1 ]]; then
    _cb_remote_usage
    return 1
  fi

  local subcmd="$1"
  shift

  case "$subcmd" in
    add)
      _cb_remote_cmd_add "$@"
      ;;
    list)
      _cb_remote_cmd_list "$@"
      ;;
    remove)
      if [[ $# -lt 1 ]]; then
        log_error "Usage: chainbench remote remove <alias>"
        return 1
      fi
      _cb_remote_cmd_remove "$1"
      ;;
    select)
      if [[ $# -lt 1 ]]; then
        log_error "Usage: chainbench remote select <alias>"
        return 1
      fi
      _cb_remote_cmd_select "$1"
      ;;
    info)
      _cb_remote_cmd_info "$@"
      ;;
    --help|-h|help)
      _cb_remote_usage
      return 0
      ;;
    *)
      log_error "Unknown remote subcommand: '$subcmd'"
      _cb_remote_usage
      return 1
      ;;
  esac
}

# ---- Entry point -------------------------------------------------------------

cmd_remote_main "$@"
