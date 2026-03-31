#!/usr/bin/env bash
# lib/rpc_client.sh - Unified JSON-RPC client for chainbench
# Single source of truth for all RPC calls (local and remote).
#
# Usage: source "${CHAINBENCH_DIR}/lib/rpc_client.sh"
# Depends: lib/json_helpers.sh, lib/pids_state.sh (local), lib/remote_state.sh (remote)

# Guard against double-sourcing
[[ -n "${_CB_RPC_CLIENT_LOADED:-}" ]] && return 0
readonly _CB_RPC_CLIENT_LOADED=1

# ---- Setup -------------------------------------------------------------------

_CB_RPC_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_RPC_LIB_DIR}/json_helpers.sh"
source "${_CB_RPC_LIB_DIR}/pids_state.sh"

# ---- Constants ---------------------------------------------------------------

readonly CB_RPC_TIMEOUT_LOCAL="${CHAINBENCH_RPC_TIMEOUT_LOCAL:-3}"
readonly CB_RPC_TIMEOUT_REMOTE="${CHAINBENCH_RPC_TIMEOUT_REMOTE:-10}"

# ---- Remote state lazy loader ------------------------------------------------

_cb_rpc_remote_loaded=false

_cb_rpc_ensure_remote_state() {
  if [[ "$_cb_rpc_remote_loaded" == "false" ]]; then
    if [[ -f "${_CB_RPC_LIB_DIR}/remote_state.sh" ]]; then
      source "${_CB_RPC_LIB_DIR}/common.sh" 2>/dev/null || true
      source "${_CB_RPC_LIB_DIR}/remote_state.sh" 2>/dev/null || true
      _cb_rpc_remote_loaded=true
    fi
  fi
}

# ---- Low-level RPC -----------------------------------------------------------

# cb_rpc_raw <url> <method> [params] [auth_header] [timeout]
# Execute a JSON-RPC call via curl. Returns the full response JSON.
# Exit code: 0 (HTTP success), 1 (curl failure)
cb_rpc_raw() {
  local url="${1:?cb_rpc_raw: url required}"
  local method="${2:?cb_rpc_raw: method required}"
  local params="${3:-[]}"
  local auth_header="${4:-}"
  local timeout="${5:-$CB_RPC_TIMEOUT_REMOTE}"

  local -a curl_args=(
    -s --max-time "$timeout"
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

# cb_rpc_result <url> <method> [params] [auth_header] [timeout]
# Execute RPC and return only the .result field.
# Exit code: 0 (success), 1 (RPC error or network failure)
cb_rpc_result() {
  local response
  response=$(cb_rpc_raw "$@") || return 1
  cb_json_get_result "$response"
}

# ---- Target-resolving wrappers -----------------------------------------------

# cb_rpc_local <node_id> <method> [params]
# RPC call to a local node. Resolves HTTP port from pids.json.
cb_rpc_local() {
  local node_id="${1:?cb_rpc_local: node_id required}"
  local method="${2:?cb_rpc_local: method required}"
  local params="${3:-[]}"

  local port
  port=$(pids_get_field "$node_id" "http_port") || {
    printf 'ERROR: node %s not found in pids.json\n' "$node_id" >&2
    return 1
  }

  if [[ -z "$port" ]]; then
    printf 'ERROR: node %s has no http_port\n' "$node_id" >&2
    return 1
  fi

  cb_rpc_raw "http://127.0.0.1:${port}" "$method" "$params" "" "$CB_RPC_TIMEOUT_LOCAL"
}

# cb_rpc_local_result <node_id> <method> [params]
# RPC call to local node, returns only .result.
cb_rpc_local_result() {
  local response
  response=$(cb_rpc_local "$@") || return 1
  cb_json_get_result "$response"
}

# cb_rpc_remote <alias> <method> [params]
# RPC call to a remote chain. Resolves URL and auth from remotes.json.
cb_rpc_remote() {
  local alias="${1:?cb_rpc_remote: alias required}"
  local method="${2:?cb_rpc_remote: method required}"
  local params="${3:-[]}"

  _cb_rpc_ensure_remote_state

  local url auth_header
  url=$(_cb_remote_get_url "$alias" 2>/dev/null) || {
    printf 'ERROR: remote alias "%s" not found\n' "$alias" >&2
    return 1
  }
  auth_header=$(_cb_remote_get_auth_header "$alias" 2>/dev/null || echo "")

  cb_rpc_raw "$url" "$method" "$params" "$auth_header" "$CB_RPC_TIMEOUT_REMOTE"
}

# cb_rpc_remote_result <alias> <method> [params]
# RPC call to remote chain, returns only .result.
cb_rpc_remote_result() {
  local response
  response=$(cb_rpc_remote "$@") || return 1
  cb_json_get_result "$response"
}

# ---- Unified target wrapper --------------------------------------------------

# cb_rpc <target> <method> [params]
# Unified RPC wrapper. Target can be:
#   - Numeric (e.g. "1", "3"): local node → resolves from pids.json
#   - "@alias" (e.g. "@eth-main"): remote → resolves from remotes.json
cb_rpc() {
  local target="${1:?cb_rpc: target required}"
  shift

  if [[ "$target" == @* ]]; then
    cb_rpc_remote "${target#@}" "$@"
  else
    cb_rpc_local "$target" "$@"
  fi
}

# cb_rpc_get_result <target> <method> [params]
# Unified RPC wrapper returning only .result.
cb_rpc_get_result() {
  local response
  response=$(cb_rpc "$@") || return 1
  cb_json_get_result "$response"
}
