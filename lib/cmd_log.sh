#!/usr/bin/env bash
# lib/cmd_log.sh - Dispatch handler for `chainbench log <subcommand>`
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.
#
# Subcommands:
#   log timeline [--json] [--last <N>]   Show consensus timeline (default last 30s)
#   log anomaly  [--json]                Detect log anomalies
#   log search   <pattern> [--node <N>]  Search across node log files

# Guard against double-sourcing
[[ -n "${_CB_CMD_LOG_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_LOG_SH_LOADED=1

_CB_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_LIB_DIR}/common.sh"

# ---- Constants ---------------------------------------------------------------

readonly _CB_LOG_LOGS_DIR="${_CB_LIB_DIR}/../logs"
readonly _CB_LOG_DATA_LOGS_DIR="${CHAINBENCH_DIR}/data/logs"
readonly _CB_LOG_PIDS_FILE="${CHAINBENCH_DIR}/state/pids.json"

# ---- Usage -------------------------------------------------------------------

_cb_log_usage() {
  cat >&2 <<'EOF'
Usage: chainbench log <subcommand> [options]

Subcommands:
  timeline [--json] [--last <N>]       Consensus event timeline (default last 30s)
  anomaly  [--json]                    Detect anomalous patterns in node logs
  search   <pattern> [--node <N>]      Search all node logs (or a specific node)

Options for timeline:
  --json         Output JSON array instead of human-readable table
  --last <N>     Show events from the last N seconds (default: 30)

Options for search:
  --node <N>     Limit search to node number N (1-based)
EOF
}

# ---- Helper: collect log file paths ------------------------------------------

# _cb_log_all_log_files
# Prints the path to each discovered node log file, one per line.
_cb_log_all_log_files() {
  if [[ -f "$_CB_LOG_PIDS_FILE" ]]; then
    python3 - "$_CB_LOG_PIDS_FILE" "$_CB_LOG_DATA_LOGS_DIR" <<'PYEOF'
import sys, json, os

pids_file = sys.argv[1]
logs_dir  = sys.argv[2]

with open(pids_file) as fh:
    data = json.load(fh)

for node in data.get('nodes', []):
    label = node.get('label', '')
    if not label:
        continue
    candidate = os.path.join(logs_dir, f'{label}.log')
    if os.path.isfile(candidate):
        print(candidate)
PYEOF
  fi

  # Fallback: glob data/logs/*.log
  if [[ -d "$_CB_LOG_DATA_LOGS_DIR" ]]; then
    local f
    for f in "${_CB_LOG_DATA_LOGS_DIR}"/*.log; do
      [[ -f "$f" ]] && printf '%s\n' "$f"
    done
  fi
}

# _cb_log_file_for_node <node_number>
# Resolves the log file path for node N (1-based).  Prints the path or returns 1.
_cb_log_file_for_node() {
  local node_num="${1:?_cb_log_file_for_node: node number required}"

  if ! [[ "$node_num" =~ ^[1-9][0-9]*$ ]]; then
    log_error "Node number must be a positive integer, got: '$node_num'"
    return 1
  fi

  if [[ ! -f "$_CB_LOG_PIDS_FILE" ]]; then
    log_error "pids.json not found — has the chain been started?"
    return 1
  fi

  python3 - "$_CB_LOG_PIDS_FILE" "$_CB_LOG_DATA_LOGS_DIR" "$node_num" <<'PYEOF'
import sys, json, os

pids_file  = sys.argv[1]
logs_dir   = sys.argv[2]
node_num   = int(sys.argv[3])

with open(pids_file) as fh:
    data = json.load(fh)

nodes = data.get('nodes', [])
idx   = node_num - 1

if idx < 0 or idx >= len(nodes):
    print(f'Node {node_num} does not exist (only {len(nodes)} configured)',
          file=sys.stderr)
    sys.exit(1)

label     = nodes[idx].get('label', f'node{node_num}')
candidate = os.path.join(logs_dir, f'{label}.log')

if not os.path.isfile(candidate):
    print(f'Log file not found: {candidate}', file=sys.stderr)
    sys.exit(1)

print(candidate)
PYEOF
}

# ---- Subcommand: timeline ----------------------------------------------------

_cb_log_cmd_timeline() {
  # Source the timeline library
  local timeline_sh="${_CB_LOG_LOGS_DIR}/timeline.sh"
  if [[ ! -f "$timeline_sh" ]]; then
    log_error "Timeline library not found: $timeline_sh"
    return 1
  fi
  # shellcheck source=logs/timeline.sh
  source "$timeline_sh"

  generate_timeline "$@"
}

# ---- Subcommand: anomaly -----------------------------------------------------

_cb_log_cmd_anomaly() {
  local anomaly_sh="${_CB_LOG_LOGS_DIR}/anomaly.sh"
  if [[ ! -f "$anomaly_sh" ]]; then
    log_error "Anomaly library not found: $anomaly_sh"
    return 1
  fi
  # shellcheck source=logs/anomaly.sh
  source "$anomaly_sh"

  detect_anomalies "$@"
}

# ---- Subcommand: search ------------------------------------------------------

_cb_log_cmd_search() {
  if [[ $# -lt 1 ]]; then
    log_error "Usage: chainbench log search <pattern> [--node <N>]"
    return 1
  fi

  local pattern="$1"
  shift

  local node_num=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --node)
        if [[ -z "${2:-}" ]]; then
          log_error "--node requires a node number"
          return 1
        fi
        node_num="$2"
        shift 2
        ;;
      --node=*)
        node_num="${1#--node=}"
        shift
        ;;
      *)
        log_warn "Unknown search option: $1 (ignoring)"
        shift
        ;;
    esac
  done

  local -a target_files=()

  if [[ -n "$node_num" ]]; then
    local node_log
    node_log="$(_cb_log_file_for_node "$node_num")" || return 1
    target_files=("$node_log")
  else
    while IFS= read -r lf; do
      [[ -n "$lf" ]] && target_files+=("$lf")
    done < <(_cb_log_all_log_files)
  fi

  if [[ ${#target_files[@]} -eq 0 ]]; then
    log_warn "No log files found to search."
    return 0
  fi

  local found=0
  local lf
  for lf in "${target_files[@]}"; do
    local node_label
    node_label="$(basename "${lf%.log}")"

    # Read matched lines and prefix with node label
    local matched
    matched="$(grep -n -- "$pattern" "$lf" 2>/dev/null)" || true

    if [[ -n "$matched" ]]; then
      found=1
      while IFS= read -r match_line; do
        printf '[%s] %s\n' "$node_label" "$match_line"
      done <<< "$matched"
    fi
  done

  if [[ "$found" -eq 0 ]]; then
    log_info "No matches found for pattern: $pattern"
  fi

  return 0
}

# ---- Main dispatcher ---------------------------------------------------------

cmd_log_main() {
  if [[ $# -lt 1 ]]; then
    _cb_log_usage
    return 1
  fi

  local subcmd="$1"
  shift

  case "$subcmd" in
    timeline)
      _cb_log_cmd_timeline "$@"
      ;;
    anomaly)
      _cb_log_cmd_anomaly "$@"
      ;;
    search)
      _cb_log_cmd_search "$@"
      ;;
    --help|-h|help)
      _cb_log_usage
      return 0
      ;;
    *)
      log_error "Unknown log subcommand: '$subcmd'"
      _cb_log_usage
      return 1
      ;;
  esac
}

# ---- Entry point -------------------------------------------------------------

cmd_log_main "$@"
