#!/usr/bin/env bash
# tests/lib/wait.sh - Polling/wait utilities for chainbench tests
# Usage: source tests/lib/wait.sh
# Requires: rpc.sh (for wait_for_block and wait_for_sync)

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

# Color codes (consistent with lib/common.sh)
_WAIT_YELLOW='\033[0;33m'
_WAIT_GREEN='\033[0;32m'
_WAIT_RED='\033[0;31m'
_WAIT_RESET='\033[0m'

# ---------------------------------------------------------------------------
# Generic condition polling
# ---------------------------------------------------------------------------

# wait_for_condition <check_cmd> [timeout_secs] [interval_secs]
#
# Repeatedly evaluates check_cmd (via eval) until it exits with code 0.
# Prints "ok" and returns 0 on success.
# Prints "timeout" and returns 1 after timeout_secs is exhausted.
#
# Example:
#   wait_for_condition "peer_count 1 | grep -q '[^0]'" 60 2
wait_for_condition() {
  local check_cmd="${1:?wait_for_condition: check_cmd required}"
  local timeout="${2:-60}"
  local interval="${3:-1}"

  local elapsed=0
  while [[ "$elapsed" -lt "$timeout" ]]; do
    if eval "$check_cmd" > /dev/null 2>&1; then
      echo "ok"
      return 0
    fi
    sleep "$interval"
    elapsed=$(( elapsed + interval ))
  done

  echo "timeout"
  return 1
}

# ---------------------------------------------------------------------------
# Block-level wait (delegates to rpc.sh wait_for_block if available,
# otherwise provides a self-contained fallback)
# ---------------------------------------------------------------------------

# wait_for_block <node> <target_block> [timeout_secs]
#
# Polls until the node's block number >= target_block.
# Prints the reached block number on success, "timeout" on failure.
# Returns 0 on success, 1 on timeout.
#
# NOTE: If rpc.sh has already been sourced its wait_for_block is preferred.
# This function is defined here so wait.sh is usable on its own.
if ! declare -f wait_for_block > /dev/null 2>&1; then
  wait_for_block() {
    local node="${1:?wait_for_block: node required}"
    local target="${2:?wait_for_block: target required}"
    local timeout="${3:-60}"

    if ! declare -f block_number > /dev/null 2>&1; then
      echo "ERROR: wait_for_block requires rpc.sh to be sourced first." >&2
      return 1
    fi

    local i
    for i in $(seq 1 "$timeout"); do
      local current
      current=$(block_number "$node" 2>/dev/null || echo "0")
      if [[ "$current" -ge "$target" ]]; then
        echo "$current"
        return 0
      fi
      sleep 1
    done

    echo "timeout"
    return 1
  }
fi

# ---------------------------------------------------------------------------
# Sync wait
# ---------------------------------------------------------------------------

# wait_for_sync [timeout_secs]
#
# Polls check_sync (from rpc.sh) until all running nodes are within 2 blocks
# of each other (i.e. synced=true in the JSON output).
# Prints "synced" and returns 0 on success, "timeout" and returns 1 on failure.
wait_for_sync() {
  local timeout="${1:-120}"

  if ! declare -f check_sync > /dev/null 2>&1; then
    echo "ERROR: wait_for_sync requires rpc.sh to be sourced first." >&2
    return 1
  fi

  printf "${_WAIT_YELLOW}[WAIT]${_WAIT_RESET}  Waiting up to %ds for nodes to sync...\n" "$timeout" >&2

  local elapsed=0
  while [[ "$elapsed" -lt "$timeout" ]]; do
    local sync_json synced
    sync_json=$(check_sync 2>/dev/null || echo '{"synced":false}')
    synced=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
print('true' if d.get('synced') else 'false')
" "$sync_json")

    if [[ "$synced" == "true" ]]; then
      local max_block
      max_block=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
print(d.get('max', 0))
" "$sync_json")
      printf "${_WAIT_GREEN}[WAIT]${_WAIT_RESET}  Nodes synced at block %s\n" "$max_block" >&2
      echo "synced"
      return 0
    fi

    sleep 2
    elapsed=$(( elapsed + 2 ))
  done

  printf "${_WAIT_RED}[WAIT]${_WAIT_RESET}  Sync timeout after %ds\n" "$timeout" >&2
  echo "timeout"
  return 1
}

# ---------------------------------------------------------------------------
# Sleep with progress message
# ---------------------------------------------------------------------------

# wait_seconds <n> [msg]
# Sleeps for n seconds, printing a status message before and after.
wait_seconds() {
  local n="${1:?wait_seconds: duration required}"
  local msg="${2:-Waiting ${n}s...}"

  printf "${_WAIT_YELLOW}[WAIT]${_WAIT_RESET}  %s\n" "$msg" >&2
  sleep "$n"
  printf "${_WAIT_GREEN}[WAIT]${_WAIT_RESET}  Done waiting %ds\n" "$n" >&2
}
