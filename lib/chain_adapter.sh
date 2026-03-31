#!/usr/bin/env bash
# lib/chain_adapter.sh - Chain type adapter loader (Strategy pattern)
# Loads the appropriate adapter based on chain.type profile field.
#
# Usage: source "${CHAINBENCH_DIR}/lib/chain_adapter.sh"
# Depends: lib/json_helpers.sh

# Guard against double-sourcing
[[ -n "${_CB_CHAIN_ADAPTER_LOADED:-}" ]] && return 0
readonly _CB_CHAIN_ADAPTER_LOADED=1

_CB_ADAPTER_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_ADAPTER_LIB_DIR}/json_helpers.sh"

readonly _CB_ADAPTERS_DIR="${_CB_ADAPTER_LIB_DIR}/adapters"

# cb_adapter_load <chain_type>
# Load the adapter for the given chain type.
# chain_type: "stablenet" (default) | "wbft" | "wemix"
# Exit code: 0 (success), 1 (adapter not found)
cb_adapter_load() {
  local chain_type="${1:-stablenet}"
  local adapter_file="${_CB_ADAPTERS_DIR}/${chain_type}.sh"

  if [[ ! -f "$adapter_file" ]]; then
    echo "ERROR: No adapter found for chain type '${chain_type}'" >&2
    echo "Available adapters:" >&2
    for f in "${_CB_ADAPTERS_DIR}"/*.sh; do
      [[ -f "$f" ]] && echo "  - $(basename "${f%.sh}")" >&2
    done
    return 1
  fi

  source "$adapter_file"
  return 0
}

# cb_adapter_list
# List available adapter names.
cb_adapter_list() {
  for f in "${_CB_ADAPTERS_DIR}"/*.sh; do
    [[ -f "$f" ]] && basename "${f%.sh}"
  done
}
