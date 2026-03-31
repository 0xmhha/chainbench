#!/usr/bin/env bash
# lib/log_utils.sh - Shared log file collection utilities
# Single source of truth for discovering node log file paths.
#
# Usage: source "${CHAINBENCH_DIR}/lib/log_utils.sh"
# Depends: lib/pids_state.sh

# Guard against double-sourcing
[[ -n "${_CB_LOG_UTILS_LOADED:-}" ]] && return 0
readonly _CB_LOG_UTILS_LOADED=1

_CB_LOG_UTILS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_LOG_UTILS_DIR}/pids_state.sh"

# cb_collect_log_files [node_id]
# Collect log file paths for all nodes (or a specific node).
# Prints one path per line.
# Strategy: 1) pids.json log_file field  2) fallback glob on data/logs/
cb_collect_log_files() {
  local target_node="${1:-}"
  local logs_dir="${CHAINBENCH_DIR}/data/logs"

  if pids_exists; then
    if [[ -n "$target_node" ]]; then
      local log_file
      log_file=$(pids_get_field "$target_node" "log_file" 2>/dev/null)
      if [[ -n "$log_file" && -f "$log_file" ]]; then
        printf '%s\n' "$log_file"
        return 0
      fi
      # Fallback for specific node
      local candidate="${logs_dir}/node${target_node}.log"
      [[ -f "$candidate" ]] && printf '%s\n' "$candidate"
      return 0
    fi

    # All nodes
    local node_ids
    node_ids=$(pids_list_nodes 2>/dev/null)
    local id log_file
    for id in $node_ids; do
      log_file=$(pids_get_field "$id" "log_file" 2>/dev/null)
      [[ -n "$log_file" && -f "$log_file" ]] && printf '%s\n' "$log_file"
    done
    return 0
  fi

  # Fallback: glob
  if [[ -d "$logs_dir" ]]; then
    local f
    for f in "${logs_dir}"/*.log; do
      [[ -f "$f" ]] && printf '%s\n' "$f"
    done
  fi
}
