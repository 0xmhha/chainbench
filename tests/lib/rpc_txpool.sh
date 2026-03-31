#!/usr/bin/env bash
# tests/lib/rpc_txpool.sh - Transaction pool query functions
#
# Functions: txpool_status, txpool_content, txpool_pending_count
#
# Usage: source tests/lib/rpc_txpool.sh

# Guard against double-sourcing
[[ -n "${_CB_RPC_TXPOOL_LOADED:-}" ]] && return 0
readonly _CB_RPC_TXPOOL_LOADED=1

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

source "${CHAINBENCH_DIR}/lib/json_helpers.sh"
source "${CHAINBENCH_DIR}/tests/lib/rpc.sh"

# ---------------------------------------------------------------------------
# Transaction pool
# ---------------------------------------------------------------------------

# txpool_status [target]
# Prints JSON: {pending: N, queued: N}.
txpool_status() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local response result pending queued

  response=$(rpc "$target" "txpool_status" "[]") || return 1
  result=$(cb_json_get_result "$response") || return 1

  pending=$(printf '%s' "$result" | cb_json_read_stdin "pending" "0x0")
  queued=$(printf '%s' "$result" | cb_json_read_stdin "queued" "0x0")

  pending=$(cb_hex_to_dec "$pending")
  queued=$(cb_hex_to_dec "$queued")

  printf '{"pending":%s,"queued":%s}\n' "$pending" "$queued"
}

# txpool_content [target]
# Prints full txpool content JSON: {pending: {...}, queued: {...}}.
txpool_content() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local response
  response=$(rpc "$target" "txpool_content" "[]") || return 1
  cb_json_get_result "$response"
}

# txpool_pending_count [target]
# Prints the number of pending transactions as decimal.
txpool_pending_count() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local status
  status=$(txpool_status "$target") || return 1
  printf '%s' "$status" | cb_json_read_stdin "pending" "0"
}
