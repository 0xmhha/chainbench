#!/usr/bin/env bash
# lib/pids_state.sh - Centralized pids.json state management
# All reads/writes to pids.json go through this module.
#
# Usage: source "${CHAINBENCH_DIR}/lib/pids_state.sh"
# Depends: lib/json_helpers.sh

# Guard against double-sourcing
[[ -n "${_CB_PIDS_STATE_LOADED:-}" ]] && return 0
readonly _CB_PIDS_STATE_LOADED=1

# ---- Setup -------------------------------------------------------------------

_CB_PIDS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_PIDS_LIB_DIR}/json_helpers.sh"

# Internal constant — not exposed to consumers
readonly _CB_PIDS_FILE="${CHAINBENCH_DIR}/state/pids.json"

# ---- Existence check ---------------------------------------------------------

# pids_exists
# Check if pids.json exists.
# Exit code: 0 (exists), 1 (does not exist)
pids_exists() {
  [[ -f "$_CB_PIDS_FILE" ]]
}

# ---- Meta field access -------------------------------------------------------

# pids_read_meta <field>
# Read a top-level meta field: chain_id, profile, started_at, etc.
# Example: pids_read_meta "profile" → "default"
pids_read_meta() {
  local field="${1:?pids_read_meta: field required}"
  pids_exists || return 1
  cb_json_read "$_CB_PIDS_FILE" "$field"
}

# ---- Node queries ------------------------------------------------------------

# pids_list_nodes [--running-only] [--format=id|json]
# List node IDs or full node data.
# Default: space-separated IDs sorted numerically.
pids_list_nodes() {
  pids_exists || return 1

  local running_only=false
  local format="id"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --running-only) running_only=true; shift ;;
      --format=*)     format="${1#--format=}"; shift ;;
      *)              shift ;;
    esac
  done

  python3 - "$_CB_PIDS_FILE" "$running_only" "$format" <<'PYEOF'
import sys, json

pids_file    = sys.argv[1]
running_only = sys.argv[2] == "true"
fmt          = sys.argv[3]

with open(pids_file) as fh:
    data = json.load(fh)

nodes = data.get("nodes", {})
if isinstance(nodes, list):
    nodes = {str(i + 1): n for i, n in enumerate(nodes)}

items = sorted(nodes.items(), key=lambda x: int(x[0]))

if running_only:
    items = [(k, v) for k, v in items if v.get("status") == "running"]

if fmt == "json":
    result = []
    for k, v in items:
        entry = dict(v)
        entry["id"] = k
        result.append(entry)
    print(json.dumps(result))
else:
    print(" ".join(k for k, _ in items))
PYEOF
}

# pids_get_field <node_id> <field>
# Read a single field from a node entry.
# Example: pids_get_field "3" "http_port" → "8503"
# Exit code: 0 (success), 1 (node not found)
pids_get_field() {
  local node_id="${1:?pids_get_field: node_id required}"
  local field="${2:?pids_get_field: field required}"
  pids_exists || return 1
  cb_json_read "$_CB_PIDS_FILE" "nodes.${node_id}.${field}"
}

# pids_get_all_pids
# Print space-separated PID list for all nodes (regardless of status).
pids_get_all_pids() {
  pids_exists || return 1
  python3 -c "
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
pids = [str(n.get('pid','')) for n in d.get('nodes',{}).values() if n.get('pid')]
print(' '.join(pids))
" "$_CB_PIDS_FILE"
}

# pids_node_count
# Print total number of nodes.
pids_node_count() {
  pids_exists || { printf '0\n'; return 1; }
  cb_json_array_len "$_CB_PIDS_FILE" "nodes"
}

# ---- Node mutations ----------------------------------------------------------

# pids_create <json_content>
# Create pids.json from a full JSON string. Overwrites any existing file.
pids_create() {
  local content="${1:?pids_create: json content required}"
  local state_dir
  state_dir="$(dirname "$_CB_PIDS_FILE")"
  mkdir -p "$state_dir"
  printf '%s\n' "$content" > "$_CB_PIDS_FILE"
}

# pids_update_node <node_id> <field> <value>
# Update a single field on a node entry.
# Example: pids_update_node "3" "status" "stopped"
pids_update_node() {
  local node_id="${1:?pids_update_node: node_id required}"
  local field="${2:?pids_update_node: field required}"
  local value="$3"
  pids_exists || return 1
  cb_json_write "$_CB_PIDS_FILE" "nodes.${node_id}.${field}" "$value"
}

# pids_mark_all_stopped
# Set all nodes to status=stopped with stopped_at timestamp.
pids_mark_all_stopped() {
  pids_exists || return 0

  local stopped_at
  stopped_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  python3 - "$_CB_PIDS_FILE" "$stopped_at" <<'PYEOF'
import sys, json

pids_file  = sys.argv[1]
stopped_at = sys.argv[2]

with open(pids_file) as fh:
    data = json.load(fh)

nodes = data.get("nodes", {})
for nid in nodes:
    nodes[nid]["status"]     = "stopped"
    nodes[nid]["stopped_at"] = stopped_at

with open(pids_file, "w") as fh:
    json.dump(data, fh, indent=2)
    fh.write("\n")
PYEOF
}

# pids_mark_node_dead <node_id>
# Mark a specific node as dead (unexpected termination).
# Differs from "stopped" (graceful shutdown).
pids_mark_node_dead() {
  local node_id="${1:?pids_mark_node_dead: node_id required}"
  pids_exists || return 0

  local stopped_at
  stopped_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  pids_update_node "$node_id" "status" "dead"
  pids_update_node "$node_id" "stopped_at" "$stopped_at"
}

# ---- Process liveness --------------------------------------------------------

# pids_pid_alive <node_id>
# Check if a node's PID is still running.
# Exit code: 0 (alive), 1 (dead or no PID)
pids_pid_alive() {
  local node_id="${1:?pids_pid_alive: node_id required}"
  local pid
  pid=$(pids_get_field "$node_id" "pid") || return 1
  [[ -n "$pid" && "$pid" != "0" ]] && kill -0 "$pid" 2>/dev/null
}

# ---- Cleanup -----------------------------------------------------------------

# pids_remove
# Delete the pids.json file.
pids_remove() {
  rm -f "$_CB_PIDS_FILE"
}
