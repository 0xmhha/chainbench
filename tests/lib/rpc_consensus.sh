#!/usr/bin/env bash
# tests/lib/rpc_consensus.sh - Istanbul/WBFT consensus query functions
#
# Functions: istanbul_get_validators, istanbul_get_wbft_extra_info,
#            istanbul_status, istanbul_get_commit_signers
#
# Usage: source tests/lib/rpc_consensus.sh

# Guard against double-sourcing
[[ -n "${_CB_RPC_CONSENSUS_LOADED:-}" ]] && return 0
readonly _CB_RPC_CONSENSUS_LOADED=1

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

source "${CHAINBENCH_DIR}/lib/json_helpers.sh"
source "${CHAINBENCH_DIR}/tests/lib/rpc.sh"

# ---------------------------------------------------------------------------
# Istanbul / WBFT consensus queries
# ---------------------------------------------------------------------------

# istanbul_get_validators [target] [block_number_decimal]
# Prints JSON array of validator addresses at given block or latest.
istanbul_get_validators() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local block_num="${2:-}"
  local params="[]"

  if [[ -n "$block_num" ]]; then
    local hex_num
    hex_num=$(printf '0x%x' "$block_num")
    params="[\"${hex_num}\"]"
  fi

  local response
  response=$(rpc "$target" "istanbul_getValidators" "$params") || return 1
  cb_json_get_result "$response"
}

# istanbul_get_wbft_extra_info <target> <block_number_decimal>
# Prints decoded WBFT extra data JSON for the given block.
istanbul_get_wbft_extra_info() {
  local target="${1:?istanbul_get_wbft_extra_info: target required}"
  local block_num="${2:?istanbul_get_wbft_extra_info: block_number required}"
  local hex_num response

  hex_num=$(printf '0x%x' "$block_num")
  response=$(rpc "$target" "istanbul_getWbftExtraInfo" "[\"${hex_num}\"]") || return 1
  cb_json_get_result "$response"
}

# istanbul_status <target> <start_block_decimal> <end_block_decimal>
# Prints comprehensive validator activity JSON for the block range.
istanbul_status() {
  local target="${1:?istanbul_status: target required}"
  local start_block="${2:?istanbul_status: start_block required}"
  local end_block="${3:?istanbul_status: end_block required}"
  local hex_start hex_end response

  hex_start=$(printf '0x%x' "$start_block")
  hex_end=$(printf '0x%x' "$end_block")

  response=$(rpc "$target" "istanbul_status" "[\"${hex_start}\",\"${hex_end}\"]") || return 1
  cb_json_get_result "$response"
}

# istanbul_get_commit_signers <target> <block_number_decimal>
# Prints JSON with block author and commit signers.
istanbul_get_commit_signers() {
  local target="${1:?istanbul_get_commit_signers: target required}"
  local block_num="${2:?istanbul_get_commit_signers: block_number required}"
  local hex_num response

  hex_num=$(printf '0x%x' "$block_num")
  response=$(rpc "$target" "istanbul_getCommitSignersFromBlock" "[\"${hex_num}\"]") || return 1
  cb_json_get_result "$response"
}
