#!/usr/bin/env bash
# tests/lib/rpc_tx.sh - Transaction submission and receipt functions
#
# Functions: send_tx, wait_receipt, get_gas_price
#
# Usage: source tests/lib/rpc_tx.sh

# Guard against double-sourcing
[[ -n "${_CB_RPC_TX_LOADED:-}" ]] && return 0
readonly _CB_RPC_TX_LOADED=1

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

source "${CHAINBENCH_DIR}/lib/json_helpers.sh"
source "${CHAINBENCH_DIR}/tests/lib/rpc.sh"

# ---------------------------------------------------------------------------
# Transactions
# ---------------------------------------------------------------------------

# send_tx <target> <from> <to> [value]
# Prints the tx hash on success, or TX_ERROR:<message> on failure.
send_tx() {
  local target="${1:?send_tx: target required}"
  local from="${2:?send_tx: from required}"
  local to="${3:?send_tx: to required}"
  local value="${4:-0x1}"

  local response
  response=$(rpc "$target" "eth_sendTransaction" \
    "[{\"from\":\"${from}\",\"to\":\"${to}\",\"value\":\"${value}\"}]") || {
    printf 'TX_ERROR:rpc_call_failed\n'
    return 1
  }

  if cb_json_has_error "$response"; then
    local err_msg
    err_msg=$(printf '%s' "$response" | cb_json_read_stdin "error.message" "unknown")
    printf 'TX_ERROR:%s\n' "$err_msg"
    return 1
  fi

  cb_json_get_result "$response"
}

# wait_receipt <target> <tx_hash> [timeout_secs]
# Polls until a receipt is available. Prints: success | failed | timeout.
# Returns 0 on success/failed, 1 on timeout.
wait_receipt() {
  local target="${1:?wait_receipt: target required}"
  local tx_hash="${2:?wait_receipt: tx_hash required}"
  local timeout="${3:-30}"

  local i response result status
  for i in $(seq 1 "$timeout"); do
    response=$(rpc "$target" "eth_getTransactionReceipt" "[\"${tx_hash}\"]") || {
      sleep 1
      continue
    }

    result=$(cb_json_get_result "$response" 2>/dev/null || echo "")

    if [[ -z "$result" || "$result" == "null" ]]; then
      sleep 1
      continue
    fi

    status=$(printf '%s' "$result" | cb_json_read_stdin "status" "")
    if [[ "$status" == "0x1" ]]; then
      echo "success"
    else
      echo "failed"
    fi
    return 0
  done

  echo "timeout"
  return 1
}

# get_gas_price [target]
# Prints the gas price in wei (decimal).
get_gas_price() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local response result
  response=$(rpc "$target" "eth_gasPrice" "[]") || return 1
  result=$(cb_json_get_result "$response") || return 1
  cb_hex_to_dec "$result"
}
