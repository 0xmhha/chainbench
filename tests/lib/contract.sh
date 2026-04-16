#!/usr/bin/env bash
# tests/lib/contract.sh — Contract interaction utilities (cast-based)
# Source this: source tests/lib/contract.sh
#
# Provides ABI encoding/decoding, eth_call, contract deployment,
# and state-changing calls using Foundry's `cast` CLI.

[[ -n "${_CB_CONTRACT_SH_LOADED:-}" ]] && return 0
readonly _CB_CONTRACT_SH_LOADED=1

_CB_CONTRACT_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${_CB_CONTRACT_LIB_DIR}/system_contracts.sh"
source "${_CB_CONTRACT_LIB_DIR}/rpc.sh"

# ---------------------------------------------------------------------------
# cast binary resolution
# ---------------------------------------------------------------------------

_CB_CAST_BIN=""

_cb_find_cast() {
  if command -v cast &>/dev/null; then
    _CB_CAST_BIN="$(command -v cast)"
    return 0
  fi

  local foundry_bin="${HOME}/.foundry/bin/cast"
  if [[ -x "$foundry_bin" ]]; then
    _CB_CAST_BIN="$foundry_bin"
    return 0
  fi

  echo "ERROR: 'cast' (foundry) is required but not found. Install: curl -L https://foundry.paradigm.xyz | bash" >&2
  return 1
}

_cb_find_cast || return 1

# ---------------------------------------------------------------------------
# Internal: RPC URL resolution
# ---------------------------------------------------------------------------

# _cb_resolve_rpc_url <target>
# Resolves a node number or URL to an RPC URL.
#   target "1"          → "http://127.0.0.1:<port_from_pids_json>"
#   target "http://..." → returned as-is
#   target "@alias"     → resolved via _cb_rpc_resolve_remote_url (from rpc.sh)
_cb_resolve_rpc_url() {
  local target="${1:?_cb_resolve_rpc_url: target required}"

  # Raw URL passthrough
  if [[ "$target" == http://* || "$target" == https://* ]]; then
    printf '%s' "$target"
    return 0
  fi

  # Remote alias (e.g. @eth-main) — delegate to rpc.sh resolver
  if [[ "$target" == @* ]]; then
    local alias="${target#@}"
    _cb_rpc_resolve_remote_url "$alias"
    return $?
  fi

  # Numeric: resolve from pids.json, fallback to default port scheme
  local port
  if pids_exists 2>/dev/null && port=$(pids_get_field "$target" "http_port" 2>/dev/null); then
    printf 'http://127.0.0.1:%s' "$port"
    return 0
  fi

  # Fallback: base_port + (node - 1). Default base is 8500.
  local base_port="${CHAINBENCH_BASE_PORT:-8500}"
  port=$(( base_port + target ))
  printf 'http://127.0.0.1:%d' "$port"
}

# ---------------------------------------------------------------------------
# ABI helpers
# ---------------------------------------------------------------------------

# cb_selector <function_sig>
# Compute the 4-byte function selector.
# Example: cb_selector "transfer(address,uint256)" → 0xa9059cbb
cb_selector() {
  local sig="${1:?cb_selector: function signature required}"
  "$_CB_CAST_BIN" sig "$sig"
}

# cb_abi_encode <function_sig> [args...]
# ABI-encode a function call. Returns full calldata (0x...).
# Example: cb_abi_encode "transfer(address,uint256)" "0xABC..." "1000"
cb_abi_encode() {
  local sig="${1:?cb_abi_encode: function signature required}"
  shift
  "$_CB_CAST_BIN" calldata "$sig" "$@"
}

# cb_abi_decode <types> <data>
# Decode ABI-encoded return data.
# Example: cb_abi_decode "uint256" "0x000...3e8" → 1000
cb_abi_decode() {
  local types="${1:?cb_abi_decode: types required}"
  local data="${2:?cb_abi_decode: data required}"
  "$_CB_CAST_BIN" abi-decode --input "$types" "$data" 2>/dev/null \
    || "$_CB_CAST_BIN" decode-abi-encoded "$types" "$data"
}

# ---------------------------------------------------------------------------
# eth_call
# ---------------------------------------------------------------------------

# cb_eth_call_raw <target> <to> <data_hex>
# Low-level eth_call with raw hex calldata. Returns raw hex response.
# Compatibility wrapper for the manual pattern in existing tests.
cb_eth_call_raw() {
  local target="${1:?cb_eth_call_raw: target required}"
  local to="${2:?cb_eth_call_raw: to address required}"
  local data="${3:?cb_eth_call_raw: data required}"

  local url
  url="$(_cb_resolve_rpc_url "$target")" || return 1

  [[ "$data" != 0x* ]] && data="0x${data}"

  "$_CB_CAST_BIN" call --rpc-url "$url" "$to" "$data" 2>/dev/null \
    || { echo "ERROR: cb_eth_call_raw failed (target=$target, to=$to)" >&2; return 1; }
}

# cb_eth_call <target> <to> <function_sig> [args...]
# High-level eth_call using cast ABI encoding. Returns decoded result.
# Example: cb_eth_call "1" "$SC_NATIVE_COIN_ADAPTER" "balanceOf(address)" "$addr"
cb_eth_call() {
  local target="${1:?cb_eth_call: target required}"
  local to="${2:?cb_eth_call: to address required}"
  local sig="${3:?cb_eth_call: function signature required}"
  shift 3

  local url
  url="$(_cb_resolve_rpc_url "$target")" || return 1

  "$_CB_CAST_BIN" call --rpc-url "$url" "$to" "$sig" "$@" 2>/dev/null \
    || { echo "ERROR: cb_eth_call failed (target=$target, to=$to, sig=$sig)" >&2; return 1; }
}

# ---------------------------------------------------------------------------
# State-changing transactions
# ---------------------------------------------------------------------------

# cb_send_call <target> <private_key> <to> <function_sig> [args...]
# Send a state-changing transaction. Returns tx hash.
# Example: cb_send_call "1" "$PK" "$contract" "transfer(address,uint256)" "$addr" "1000"
cb_send_call() {
  local target="${1:?cb_send_call: target required}"
  local private_key="${2:?cb_send_call: private_key required}"
  local to="${3:?cb_send_call: to address required}"
  local sig="${4:?cb_send_call: function signature required}"
  shift 4

  local url
  url="$(_cb_resolve_rpc_url "$target")" || return 1

  local output
  output=$("$_CB_CAST_BIN" send \
    --rpc-url "$url" \
    --private-key "$private_key" \
    "$to" "$sig" "$@" \
    --json 2>/dev/null) \
    || { echo "ERROR: cb_send_call failed (target=$target, to=$to, sig=$sig)" >&2; return 1; }

  # Extract tx hash from JSON receipt
  local tx_hash
  tx_hash=$(printf '%s' "$output" | python3 -c "import sys,json; print(json.load(sys.stdin).get('transactionHash',''))" 2>/dev/null) \
    || tx_hash=$(printf '%s' "$output" | grep -o '"transactionHash":"0x[0-9a-fA-F]*"' | head -1 | sed 's/.*:"//;s/"//')

  if [[ -z "$tx_hash" ]]; then
    echo "ERROR: cb_send_call: could not extract tx hash from receipt" >&2
    return 1
  fi

  printf '%s' "$tx_hash"
}

# cb_deploy_contract <target> <private_key> <bytecode_hex> [constructor_args_hex]
# Deploy a contract and return the deployed contract address.
# Example: cb_deploy_contract "1" "$PK" "0x608060..." "0x000...abc"
cb_deploy_contract() {
  local target="${1:?cb_deploy_contract: target required}"
  local private_key="${2:?cb_deploy_contract: private_key required}"
  local bytecode="${3:?cb_deploy_contract: bytecode required}"
  local constructor_args="${4:-}"

  local url
  url="$(_cb_resolve_rpc_url "$target")" || return 1

  [[ "$bytecode" != 0x* ]] && bytecode="0x${bytecode}"

  local create_data="$bytecode"
  if [[ -n "$constructor_args" ]]; then
    local args_stripped="${constructor_args#0x}"
    create_data="${bytecode}${args_stripped}"
  fi

  local output
  output=$("$_CB_CAST_BIN" send \
    --rpc-url "$url" \
    --private-key "$private_key" \
    --create "$create_data" \
    --json 2>/dev/null) \
    || { echo "ERROR: cb_deploy_contract failed (target=$target)" >&2; return 1; }

  # Extract deployed contract address
  local contract_addr
  contract_addr=$(printf '%s' "$output" | python3 -c "import sys,json; print(json.load(sys.stdin).get('contractAddress',''))" 2>/dev/null) \
    || contract_addr=$(printf '%s' "$output" | grep -o '"contractAddress":"0x[0-9a-fA-F]*"' | head -1 | sed 's/.*:"//;s/"//')

  if [[ -z "$contract_addr" || "$contract_addr" == "null" ]]; then
    echo "ERROR: cb_deploy_contract: could not extract deployed address from receipt" >&2
    return 1
  fi

  printf '%s' "$contract_addr"
}
