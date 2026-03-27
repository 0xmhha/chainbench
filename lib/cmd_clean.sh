#!/usr/bin/env bash
# lib/cmd_clean.sh - Remove node data, state files; preserve test results and config
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.

# Guard against double-sourcing
[[ -n "${_CB_CMD_CLEAN_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_CLEAN_SH_LOADED=1

_CB_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_LIB_DIR}/common.sh"

# ---- Constants ---------------------------------------------------------------

readonly _CB_CLEAN_PIDS_FILE="${CHAINBENCH_DIR}/state/pids.json"
readonly _CB_CLEAN_DATA_DIR="${CHAINBENCH_DIR}/data"
readonly _CB_CLEAN_STATE_DIR="${CHAINBENCH_DIR}/state"

# ---- Helpers -----------------------------------------------------------------

# _cb_clean_has_running_nodes
# Returns 0 if pids.json exists and contains at least one node with
# status "running".
_cb_clean_has_running_nodes() {
  [[ -f "$_CB_CLEAN_PIDS_FILE" ]] || return 1

  python3 - "$_CB_CLEAN_PIDS_FILE" <<'PYEOF'
import sys, json

with open(sys.argv[1]) as fh:
    data = json.load(fh)

running = [n for n in data.get("nodes", []) if n.get("status") == "running"]
sys.exit(0 if running else 1)
PYEOF
}

# _cb_clean_stop_chain
# Sources cmd_stop.sh to gracefully stop all running nodes.
_cb_clean_stop_chain() {
  local handler="${_CB_LIB_DIR}/cmd_stop.sh"

  if [[ ! -f "$handler" ]]; then
    log_error "cmd_stop.sh not found; cannot stop running nodes before clean"
    return 1
  fi

  # Unset stop guard so it can be sourced fresh
  unset _CB_CMD_STOP_SH_LOADED 2>/dev/null || true

  log_info "Stopping running nodes before clean"
  (
    # shellcheck disable=SC1090
    source "$handler"
  )
}

# _cb_clean_remove_path <path> <description>
# Removes a file or directory, logging the action.
_cb_clean_remove_path() {
  local path="$1"
  local description="$2"

  if [[ -e "$path" || -L "$path" ]]; then
    rm -rf -- "$path"
    log_info "Removed $description: $path"
  fi
}

# ---- Main clean logic --------------------------------------------------------

cmd_clean_main() {
  # Parse optional flags
  local force=0
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --force|-f)
        force=1
        shift
        ;;
      --help|-h)
        printf 'Usage: chainbench clean [--force]\n' >&2
        return 0
        ;;
      *)
        log_warn "Unknown clean option: $1 (ignoring)"
        shift
        ;;
    esac
  done

  # Stop any running nodes first
  if _cb_clean_has_running_nodes; then
    if ! _cb_clean_stop_chain; then
      if [[ "$force" == "1" ]]; then
        log_warn "Stop phase failed; continuing with --force"
      else
        log_error "Failed to stop running nodes. Use --force to clean anyway."
        return 1
      fi
    fi
  fi

  log_info "Cleaning chain data..."

  # --- data/node*/ (node datadirs) -------------------------------------------
  local node_dir
  for node_dir in "${_CB_CLEAN_DATA_DIR}"/node*/; do
    [[ -d "$node_dir" ]] || continue
    _cb_clean_remove_path "$node_dir" "node datadir"
  done

  # --- data/logs/*.log --------------------------------------------------------
  local log_file
  for log_file in "${_CB_CLEAN_DATA_DIR}"/logs/*.log; do
    [[ -f "$log_file" ]] || continue
    _cb_clean_remove_path "$log_file" "log file"
  done

  # --- data/genesis.json ------------------------------------------------------
  _cb_clean_remove_path "${_CB_CLEAN_DATA_DIR}/genesis.json" "genesis.json"

  # --- data/config_*.toml -----------------------------------------------------
  local cfg_file
  for cfg_file in "${_CB_CLEAN_DATA_DIR}"/config_*.toml; do
    [[ -f "$cfg_file" ]] || continue
    _cb_clean_remove_path "$cfg_file" "config TOML"
  done

  # --- state/pids.json --------------------------------------------------------
  _cb_clean_remove_path "${_CB_CLEAN_STATE_DIR}/pids.json" "pids.json"

  # --- state/current-profile.yaml ---------------------------------------------
  _cb_clean_remove_path "${_CB_CLEAN_STATE_DIR}/current-profile.yaml" "current-profile.yaml"

  # Intentionally preserved:
  #   state/results/*    - test results
  #   profiles/          - profile definitions
  #   templates/         - node/genesis templates
  #   tests/             - test suite
  #   lib/               - library code

  log_success "Clean complete"
  return 0
}

# ---- Entry point -------------------------------------------------------------

cmd_clean_main "$@"
