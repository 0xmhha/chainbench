#!/usr/bin/env bash
# lib/formatter.sh - Output formatting utilities for chainbench
# Provides table, JSON envelope, and key-value formatting.
#
# Usage: source "${CHAINBENCH_DIR}/lib/formatter.sh"
# Depends: lib/json_helpers.sh

# Guard against double-sourcing
[[ -n "${_CB_FORMATTER_LOADED:-}" ]] && return 0
readonly _CB_FORMATTER_LOADED=1

_CB_FORMATTER_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_FORMATTER_LIB_DIR}/json_helpers.sh"

# ---- JSON envelope -----------------------------------------------------------

# cb_format_json_envelope <status> <command> <data_json>
# Produce a standard JSON output envelope for MCP and --json mode.
# status: "ok" or "error"
# data_json: valid JSON string (object or array)
cb_format_json_envelope() {
  local status="${1:?cb_format_json_envelope: status required}"
  local command="${2:?cb_format_json_envelope: command required}"
  local data_json="${3:-{}}"

  local timestamp
  timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  python3 -c "
import json, sys
envelope = {
    'status':    sys.argv[1],
    'command':   sys.argv[2],
    'timestamp': sys.argv[3],
    'data':      json.loads(sys.argv[4])
}
print(json.dumps(envelope, indent=2))
" "$status" "$command" "$timestamp" "$data_json"
}

# cb_format_json_error <command> <message> [code]
# Produce a standard JSON error envelope.
cb_format_json_error() {
  local command="${1:?cb_format_json_error: command required}"
  local message="${2:?cb_format_json_error: message required}"
  local code="${3:-GENERAL_ERROR}"

  local timestamp
  timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  python3 -c "
import json, sys
envelope = {
    'status':    'error',
    'command':   sys.argv[1],
    'timestamp': sys.argv[2],
    'error': {
        'message': sys.argv[3],
        'code':    sys.argv[4]
    }
}
print(json.dumps(envelope, indent=2))
" "$command" "$timestamp" "$message" "$code"
}

# ---- Uptime formatting -------------------------------------------------------

# cb_format_uptime <started_at_iso>
# Convert an ISO timestamp to a human-friendly uptime string like "5m 12s".
cb_format_uptime() {
  local started_at="$1"
  if [[ -z "$started_at" ]]; then
    printf 'unknown\n'
    return 0
  fi

  python3 -c "
import sys
from datetime import datetime, timezone

started_str = sys.argv[1]
try:
    started = datetime.fromisoformat(started_str.replace('Z', '+00:00'))
    now     = datetime.now(timezone.utc)
    delta   = max(0, int((now - started).total_seconds()))
    days    = delta // 86400
    hours   = (delta % 86400) // 3600
    minutes = (delta % 3600) // 60
    secs    = delta % 60
    parts = []
    if days:    parts.append(f'{days}d')
    if hours:   parts.append(f'{hours}h')
    if minutes: parts.append(f'{minutes}m')
    parts.append(f'{secs}s')
    print(' '.join(parts))
except Exception:
    print('unknown')
" "$started_at"
}

# ---- Key-value formatting ----------------------------------------------------

# cb_format_kv <key> <value>
# Print a formatted key-value pair for status displays.
cb_format_kv() {
  local key="$1" value="$2"
  printf '  %-20s  %s\n' "$key" "$value"
}

# ---- Table formatting --------------------------------------------------------

# cb_format_table_header <col_spec>
# Print table header and separator. col_spec is pipe-separated "Name:Width" pairs.
# Example: cb_format_table_header "Node:5|Type:10|PID:6|Status:8"
cb_format_table_header() {
  local spec="$1"
  local IFS='|'
  local -a cols=()
  local -a widths=()

  for pair in $spec; do
    local name="${pair%%:*}"
    local width="${pair#*:}"
    cols+=("$name")
    widths+=("$width")
  done

  # Print header
  local line=" "
  local sep=" "
  local i
  for i in "${!cols[@]}"; do
    local w="${widths[$i]}"
    line+="$(printf "%-${w}s  " "${cols[$i]}")"
    sep+="$(printf '%0.s─' $(seq 1 "$w"))  "
  done
  printf '%s\n' "$line"
  printf '%s\n' "$sep"
}

# cb_format_table_row <col_widths> <values...>
# Print a single table row. col_widths is comma-separated widths.
# Example: cb_format_table_row "5,10,6,8" "1" "validator" "12345" "running"
cb_format_table_row() {
  local widths_str="$1"
  shift

  local IFS=','
  local -a widths=($widths_str)

  local line=" "
  local i=0
  for val in "$@"; do
    local w="${widths[$i]:-10}"
    line+="$(printf "%-${w}s  " "$val")"
    (( i++ )) || true
  done
  printf '%s\n' "$line"
}
