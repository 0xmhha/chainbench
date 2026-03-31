#!/usr/bin/env bash
# tests/lib/rpc.sh - RPC utility library for chainbench tests
# Usage: source tests/lib/rpc.sh
#
# Supports two modes:
#   - Local:  rpc <node_number> <method> [params]   (reads port from pids.json)
#   - Remote: rpc @<alias> <method> [params]         (resolves URL from remotes.json)
#
# When CHAINBENCH_REMOTE is set, functions default to '@<CHAINBENCH_REMOTE>'
# instead of local node '1'.
#
# Domain functions are split into separate modules (auto-sourced at end):
#   rpc_block.sh, rpc_account.sh, rpc_tx.sh, rpc_txpool.sh,
#   rpc_consensus.sh, rpc_admin.sh

# Guard against double-sourcing
[[ -n "${_CB_RPC_LOADED:-}" ]] && return 0
readonly _CB_RPC_LOADED=1

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

source "${CHAINBENCH_DIR}/lib/json_helpers.sh"
source "${CHAINBENCH_DIR}/lib/pids_state.sh"

# RPC timeout: longer for remote (network latency), shorter for local
_CB_RPC_TIMEOUT_LOCAL="${CHAINBENCH_RPC_TIMEOUT:-3}"
_CB_RPC_TIMEOUT_REMOTE="${CHAINBENCH_RPC_TIMEOUT:-10}"

# ---------------------------------------------------------------------------
# Remote state helpers (loaded lazily)
# ---------------------------------------------------------------------------

_cb_rpc_remote_loaded=0

_cb_rpc_ensure_remote_state() {
  if [[ "$_cb_rpc_remote_loaded" == "0" ]]; then
    if [[ -f "${CHAINBENCH_DIR}/lib/remote_state.sh" ]]; then
      source "${CHAINBENCH_DIR}/lib/common.sh" 2>/dev/null || true
      source "${CHAINBENCH_DIR}/lib/remote_state.sh" 2>/dev/null || true
      _cb_rpc_remote_loaded=1
    fi
  fi
}

# ---------------------------------------------------------------------------
# Internal routing helpers
# ---------------------------------------------------------------------------

# _cb_rpc_default_target
# Returns the default RPC target: '@<CHAINBENCH_REMOTE>' if set, else '1'.
_cb_rpc_default_target() {
  if [[ -n "${CHAINBENCH_REMOTE:-}" ]]; then
    printf '@%s' "$CHAINBENCH_REMOTE"
  else
    printf '1'
  fi
}

# _cb_rpc_is_remote <target>
# Returns 0 if target starts with '@', 1 otherwise.
_cb_rpc_is_remote() {
  [[ "$1" == @* ]]
}

# _cb_rpc_resolve_remote_url <alias>
# Resolves remote alias to RPC URL. Prints URL on stdout.
_cb_rpc_resolve_remote_url() {
  local alias="$1"
  _cb_rpc_ensure_remote_state
  _cb_remote_get_url "$alias"
}

# _cb_rpc_resolve_remote_auth <alias>
# Resolves remote alias to auth header. Prints header on stdout (empty if none).
_cb_rpc_resolve_remote_auth() {
  local alias="$1"
  _cb_rpc_ensure_remote_state
  _cb_remote_get_auth_header "$alias" 2>/dev/null || echo ""
}

# ---------------------------------------------------------------------------
# Core RPC
# ---------------------------------------------------------------------------

# rpc_url <url> <method> [params] [auth_header]
# Makes a raw JSON-RPC call to an arbitrary URL. Prints the full response JSON.
rpc_url() {
  local url="${1:?rpc_url: url required}"
  local method="${2:?rpc_url: method required}"
  local params="${3:-[]}"
  local auth_header="${4:-}"

  local -a curl_args=(
    -s --max-time "$_CB_RPC_TIMEOUT_REMOTE"
    -X POST
    -H "Content-Type: application/json"
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":${params},\"id\":1}"
  )

  if [[ -n "$auth_header" ]]; then
    curl_args+=(-H "Authorization: ${auth_header}")
  fi

  curl_args+=("$url")
  curl "${curl_args[@]}"
}

# rpc <target> <method> [params]
# Makes a raw JSON-RPC call and prints the full response JSON.
#
# target:
#   - Numeric (e.g. "1", "3"): local node — resolves port from pids.json
#   - "@alias" (e.g. "@eth-main"): remote — resolves URL from remotes.json
rpc() {
  local target="${1:?rpc: target required}" method="${2:?rpc: method required}" params="${3:-[]}"

  if _cb_rpc_is_remote "$target"; then
    local alias="${target#@}"
    local url auth_header
    url="$(_cb_rpc_resolve_remote_url "$alias")" || {
      echo '{"jsonrpc":"2.0","error":{"code":-1,"message":"Failed to resolve remote alias"},"id":1}'
      return 1
    }
    auth_header="$(_cb_rpc_resolve_remote_auth "$alias")"
    rpc_url "$url" "$method" "$params" "$auth_header"
  else
    local port
    port=$(pids_get_field "$target" "http_port") || {
      printf 'ERROR: node %s not found in pids.json\n' "$target" >&2
      return 1
    }
    curl -s --max-time "$_CB_RPC_TIMEOUT_LOCAL" -X POST "http://127.0.0.1:${port}" \
      -H "Content-Type: application/json" \
      -d "{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":${params},\"id\":1}"
  fi
}

# ---------------------------------------------------------------------------
# Retained high-use functions (most commonly called from tests)
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

# peer_count [target]
# Prints connected peer count as a decimal integer.
peer_count() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local response result
  response=$(rpc "$target" "net_peerCount" "[]") || return 1
  result=$(cb_json_get_result "$response") || return 1
  cb_hex_to_dec "$result"
}

# check_sync
# Prints a JSON object: {min, max, diff, synced}.
# For remote targets (CHAINBENCH_REMOTE set), queries single endpoint.
check_sync() {
  if [[ -n "${CHAINBENCH_REMOTE:-}" ]]; then
    local block
    block=$(block_number "@${CHAINBENCH_REMOTE}" 2>/dev/null || echo "0")
    printf '{"min":%d,"max":%d,"diff":0,"synced":true}\n' "$block" "$block"
    return 0
  fi

  # Local mode: query all running nodes via pids_state
  local node_ids
  node_ids=$(pids_list_nodes --running-only) || {
    printf '{"min":0,"max":0,"diff":0,"synced":false,"error":"no_running_nodes"}\n'
    return 1
  }

  local blocks=()
  local nid
  for nid in $node_ids; do
    local b
    b=$(block_number "$nid" 2>/dev/null) || continue
    blocks+=("$b")
  done

  if [[ "${#blocks[@]}" -eq 0 ]]; then
    printf '{"min":0,"max":0,"diff":0,"synced":false,"error":"no_running_nodes"}\n'
    return 1
  fi

  local mn mx
  mn="${blocks[0]}"
  mx="${blocks[0]}"
  for b in "${blocks[@]}"; do
    (( b < mn )) && mn="$b"
    (( b > mx )) && mx="$b"
  done

  local diff synced
  diff=$(( mx - mn ))
  if (( diff <= 2 )); then synced="true"; else synced="false"; fi

  printf '{"min":%d,"max":%d,"diff":%d,"synced":%s}\n' "$mn" "$mx" "$diff" "$synced"
}

# ---------------------------------------------------------------------------
# Backward compatibility: auto-source all domain modules
# ---------------------------------------------------------------------------

_CB_RPC_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
for _rpc_mod in rpc_block rpc_account rpc_tx rpc_txpool rpc_consensus rpc_admin; do
  [[ -f "${_CB_RPC_LIB_DIR}/${_rpc_mod}.sh" ]] && source "${_CB_RPC_LIB_DIR}/${_rpc_mod}.sh"
done
unset _rpc_mod
