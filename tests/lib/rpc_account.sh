#!/usr/bin/env bash
# tests/lib/rpc_account.sh - Account and wallet query functions
#
# Functions: get_coinbase, unlock_account, get_balance
#
# Usage: source tests/lib/rpc_account.sh

# Guard against double-sourcing
[[ -n "${_CB_RPC_ACCOUNT_LOADED:-}" ]] && return 0
readonly _CB_RPC_ACCOUNT_LOADED=1

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

source "${CHAINBENCH_DIR}/lib/json_helpers.sh"
source "${CHAINBENCH_DIR}/tests/lib/rpc.sh"

# ---------------------------------------------------------------------------
# Account / wallet
# ---------------------------------------------------------------------------

# get_coinbase [target]
# Prints the coinbase (etherbase) address of the node.
get_coinbase() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local response
  response=$(rpc "$target" "eth_coinbase" "[]") || return 1
  cb_json_get_result "$response"
}

# unlock_account <target> <address> [password] [duration_secs]
# Unlocks the given account. Returns 1 and prints a warning if unlock fails.
# password 기본값은 빈 문자열(폐쇄망 keystore). 호출자가 환경에 맞는 비번을 넘긴다
# (common.sh unlock_validator → $CB_VALIDATOR_KEYSTORE_PASSWORD). 라이브러리에 비번을 하드코딩하지 않는다.
unlock_account() {
  local target="${1:?unlock_account: target required}"
  local address="${2:?unlock_account: address required}"
  local password="${3-}"
  local duration="${4:-600}"
  local response result
  response=$(rpc "$target" "personal_unlockAccount" \
    "[\"${address}\",\"${password}\",${duration}]") || return 1
  result=$(printf '%s' "$response" | jq -r '.result // empty' 2>/dev/null)
  if [[ "$result" != "true" ]]; then
    local errmsg
    errmsg=$(printf '%s' "$response" | jq -r '.error.message // "unknown error"' 2>/dev/null)
    printf '[WARN]  unlock_account failed (target=%s addr=%s): %s\n' "$target" "$address" "$errmsg" >&2
    return 1
  fi
}

# get_balance <target> <address>
# Prints the balance of address in wei (decimal integer).
get_balance() {
  local target="${1:?get_balance: target required}"
  local address="${2:?get_balance: address required}"
  local response result
  response=$(rpc "$target" "eth_getBalance" "[\"${address}\",\"latest\"]") || return 1
  result=$(cb_json_get_result "$response") || return 1
  cb_hex_to_dec "$result"
}
