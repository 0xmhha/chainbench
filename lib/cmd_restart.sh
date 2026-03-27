#!/usr/bin/env bash
# lib/cmd_restart.sh - Stop, clean, re-init, and start the chain
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.

# Guard against double-sourcing
[[ -n "${_CB_CMD_RESTART_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_RESTART_SH_LOADED=1

_CB_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_LIB_DIR}/common.sh"

# ---- Constants ---------------------------------------------------------------

readonly _CB_CURRENT_PROFILE_FILE="${CHAINBENCH_DIR}/state/current-profile.yaml"

# ---- Helpers -----------------------------------------------------------------

# _cb_restart_resolve_profile
# Determines the profile name to use for re-init.
# Priority: 1) --profile flag already in CHAINBENCH_PROFILE
#           2) state/current-profile.yaml (saved by a previous init)
#           3) "default"
_cb_restart_resolve_profile() {
  if [[ -n "${CHAINBENCH_PROFILE:-}" && "${CHAINBENCH_PROFILE}" != "default" ]]; then
    printf '%s\n' "$CHAINBENCH_PROFILE"
    return 0
  fi

  if [[ -f "$_CB_CURRENT_PROFILE_FILE" ]]; then
    local saved
    saved="$(python3 - "$_CB_CURRENT_PROFILE_FILE" <<'PYEOF'
import sys, re

path = sys.argv[1]
with open(path) as fh:
    for line in fh:
        m = re.match(r'^\s*profile\s*:\s*(\S+)', line)
        if m:
            print(m.group(1))
            sys.exit(0)
print("")
PYEOF
)"
    if [[ -n "$saved" ]]; then
      printf '%s\n' "$saved"
      return 0
    fi
  fi

  printf 'default\n'
}

# _cb_restart_run_subcommand <subcmd> [args...]
# Sources a cmd_<subcmd>.sh handler in a sub-shell-like manner by unsetting
# the guard variable so the file can be re-sourced cleanly if needed.
_cb_restart_run_subcommand() {
  local subcmd="$1"
  shift
  local handler="${_CB_LIB_DIR}/cmd_${subcmd}.sh"

  if [[ ! -f "$handler" ]]; then
    log_error "Subcommand handler not found: $handler"
    return 1
  fi

  # Unset the guard so the file can be sourced fresh in this shell
  local guard_var="_CB_CMD_${subcmd^^}_SH_LOADED"
  unset "$guard_var" 2>/dev/null || true

  log_info "Running: chainbench $subcmd $*"
  # Run in a subshell so each step's set/unset side-effects are isolated,
  # but inherit all exported variables.
  (
    set -- "$@"
    # shellcheck disable=SC1090
    source "$handler"
  )
}

# ---- Main restart logic ------------------------------------------------------

cmd_restart_main() {
  # Parse restart-specific flags (none currently; reserved for future use)
  local profile=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --profile)
        profile="${2:?--profile requires a name}"
        shift 2
        ;;
      --profile=*)
        profile="${1#--profile=}"
        shift
        ;;
      *)
        log_warn "Unknown restart option: $1 (ignoring)"
        shift
        ;;
    esac
  done

  # Allow explicit --profile on the restart command to override stored profile
  if [[ -n "$profile" ]]; then
    export CHAINBENCH_PROFILE="$profile"
  fi

  local resolved_profile
  resolved_profile="$(_cb_restart_resolve_profile)"

  log_info "Restarting chain with profile: $resolved_profile"

  # Step 1: stop
  if ! _cb_restart_run_subcommand stop; then
    log_error "Stop phase failed during restart"
    return 1
  fi

  # Step 2: clean
  if ! _cb_restart_run_subcommand clean; then
    log_error "Clean phase failed during restart"
    return 1
  fi

  # Step 3: init with resolved profile
  export CHAINBENCH_PROFILE="$resolved_profile"
  if ! _cb_restart_run_subcommand init; then
    log_error "Init phase failed during restart"
    return 1
  fi

  # Step 4: start
  if ! _cb_restart_run_subcommand start; then
    log_error "Start phase failed during restart"
    return 1
  fi

  log_success "Chain restarted with profile '$resolved_profile'"
  return 0
}

# ---- Entry point -------------------------------------------------------------

cmd_restart_main "$@"
