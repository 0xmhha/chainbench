#!/usr/bin/env bash
# tests/lib/rpc_block.sh - Block and chain query functions
#
# Functions: block_number, get_block, get_block_miner, get_chain_id,
#            wait_for_block, get_client_version, get_syncing
#
# Usage: source tests/lib/rpc_block.sh

# Guard against double-sourcing
[[ -n "${_CB_RPC_BLOCK_LOADED:-}" ]] && return 0
readonly _CB_RPC_BLOCK_LOADED=1

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

source "${CHAINBENCH_DIR}/lib/json_helpers.sh"
source "${CHAINBENCH_DIR}/tests/lib/rpc.sh"

# ---------------------------------------------------------------------------
# Block / chain queries
# ---------------------------------------------------------------------------

# block_number [target]
# Prints current block number as a decimal integer.
block_number() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local response result
  response=$(rpc "$target" "eth_blockNumber" "[]") || return 1
  result=$(cb_json_get_result "$response") || return 1
  cb_hex_to_dec "$result"
}

# get_block <target> <number>
# Prints the full block object JSON for the given decimal block number.
get_block() {
  local target="${1:?get_block: target required}"
  local number="${2:?get_block: number required}"
  local hex_num
  hex_num=$(printf '0x%x' "$number")
  rpc "$target" "eth_getBlockByNumber" "[\"${hex_num}\", false]"
}

# get_block_miner <target> <number>
# Prints the miner address of the block at the given decimal number.
get_block_miner() {
  local target="${1:?get_block_miner: target required}"
  local number="${2:?get_block_miner: number required}"
  local hex_num response result
  hex_num=$(printf '0x%x' "$number")
  response=$(rpc "$target" "eth_getBlockByNumber" "[\"${hex_num}\", false]") || return 1
  result=$(cb_json_get_result "$response") || return 1
  printf '%s\n' "$result" | cb_json_read_stdin "miner"
}

# get_chain_id [target]
# Prints the chain ID as a decimal integer.
get_chain_id() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local response result
  response=$(rpc "$target" "eth_chainId" "[]") || return 1
  result=$(cb_json_get_result "$response") || return 1
  cb_hex_to_dec "$result"
}

# wait_for_block <target> <target_block> [timeout_secs]
# Polls until the node's block number >= target. Prints the reached block number.
# Returns 0 on success, 1 on timeout.
wait_for_block() {
  local target="${1:?wait_for_block: target required}"
  local target_block="${2:?wait_for_block: target_block required}"
  local timeout="${3:-60}"

  local i current
  for i in $(seq 1 "$timeout"); do
    current=$(block_number "$target" 2>/dev/null || echo "0")
    if [[ "$current" -ge "$target_block" ]]; then
      echo "$current"
      return 0
    fi
    sleep 1
  done

  echo "timeout"
  return 1
}

# get_client_version [target]
# Prints the client version string.
get_client_version() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local response
  response=$(rpc "$target" "web3_clientVersion" "[]") || return 1
  cb_json_get_result "$response"
}

# get_syncing [target]
# Prints "false" if not syncing, or JSON sync object if syncing.
get_syncing() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local response result
  response=$(rpc "$target" "eth_syncing" "[]") || return 1
  result=$(cb_json_get_result "$response") || return 1
  if [[ -z "$result" || "$result" == "false" ]]; then
    printf 'false\n'
  else
    printf '%s\n' "$result"
  fi
}
