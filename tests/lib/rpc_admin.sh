#!/usr/bin/env bash
# tests/lib/rpc_admin.sh - Admin and P2P peer management functions
#
# Functions: admin_peers, admin_node_info, admin_enode,
#            admin_remove_peer, admin_add_peer
#
# Usage: source tests/lib/rpc_admin.sh

# Guard against double-sourcing
[[ -n "${_CB_RPC_ADMIN_LOADED:-}" ]] && return 0
readonly _CB_RPC_ADMIN_LOADED=1

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

source "${CHAINBENCH_DIR}/lib/json_helpers.sh"
source "${CHAINBENCH_DIR}/tests/lib/rpc.sh"

# ---------------------------------------------------------------------------
# Admin / P2P management
# ---------------------------------------------------------------------------

# admin_peers [target]
# Prints JSON array of connected peer info objects.
admin_peers() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local response
  response=$(rpc "$target" "admin_peers" "[]") || return 1
  cb_json_get_result "$response"
}

# admin_node_info [target]
# Prints JSON node info including enode URL.
admin_node_info() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local response
  response=$(rpc "$target" "admin_nodeInfo" "[]") || return 1
  cb_json_get_result "$response"
}

# admin_enode [target]
# Prints the enode URL of the node.
admin_enode() {
  local target="${1:-$(_cb_rpc_default_target)}"
  local info
  info=$(admin_node_info "$target") || return 1
  printf '%s' "$info" | cb_json_read_stdin "enode" ""
}

# admin_remove_peer <target> <enode_url>
# Removes a peer and disconnects. Prints "true" or "false".
admin_remove_peer() {
  local target="${1:?admin_remove_peer: target required}"
  local enode="${2:?admin_remove_peer: enode_url required}"
  local response
  response=$(rpc "$target" "admin_removePeer" "[\"${enode}\"]") || return 1
  cb_json_get_result "$response"
}

# admin_add_peer <target> <enode_url>
# Adds a peer to the static list. Prints "true" or "false".
admin_add_peer() {
  local target="${1:?admin_add_peer: target required}"
  local enode="${2:?admin_add_peer: enode_url required}"
  local response
  response=$(rpc "$target" "admin_addPeer" "[\"${enode}\"]") || return 1
  cb_json_get_result "$response"
}
