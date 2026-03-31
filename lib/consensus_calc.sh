#!/usr/bin/env bash
# lib/consensus_calc.sh - BFT consensus calculation utilities
# Pure computation module with no external dependencies.
#
# Usage: source "${CHAINBENCH_DIR}/lib/consensus_calc.sh"

# Guard against double-sourcing
[[ -n "${_CB_CONSENSUS_CALC_LOADED:-}" ]] && return 0
readonly _CB_CONSENSUS_CALC_LOADED=1

# cb_bft_threshold <total_validators>
# Calculate BFT quorum threshold: floor(total * 2/3) + 1
# Example: cb_bft_threshold 4 → "3"
cb_bft_threshold() {
  local total="${1:?cb_bft_threshold: total_validators required}"
  if [[ "$total" -le 0 ]]; then
    printf '0\n'
    return 0
  fi
  printf '%d\n' $(( (total * 2 / 3) + 1 ))
}

# cb_consensus_status <active_count> <total_count>
# Determine consensus status: OK / DEGRADED / DOWN
# OK: active >= threshold
# DEGRADED: 0 < active < threshold
# DOWN: active == 0 or total == 0
cb_consensus_status() {
  local active="${1:?cb_consensus_status: active_count required}"
  local total="${2:?cb_consensus_status: total_count required}"

  if [[ "$total" -eq 0 || "$active" -eq 0 ]]; then
    printf 'DOWN\n'
    return 0
  fi

  local threshold
  threshold=$(cb_bft_threshold "$total")

  if [[ "$active" -ge "$threshold" ]]; then
    printf 'OK\n'
  else
    printf 'DEGRADED\n'
  fi
}

# cb_consensus_json <active_count> <total_count>
# Return consensus status as a JSON object.
cb_consensus_json() {
  local active="${1:?cb_consensus_json: active_count required}"
  local total="${2:?cb_consensus_json: total_count required}"
  local threshold
  threshold=$(cb_bft_threshold "$total")
  local status
  status=$(cb_consensus_status "$active" "$total")

  printf '{"active":%d,"total":%d,"threshold":%d,"status":"%s"}\n' \
    "$active" "$total" "$threshold" "$status"
}
