#!/usr/bin/env bash
# tests/unit/tests/network-id-resolve.sh - _cb_resolve_network_id precedence
#
# P1-4b (SSOT-X1): network_id was a dead knob — CHAINBENCH_NETWORK_ID was parsed
# but never reached gstable. This helper is the re-wiring point; it resolves the
# effective P2P network id that cmd_start.sh now passes via --networkid.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR
export CHAINBENCH_QUIET=1
source "${CHAINBENCH_DIR}/lib/common.sh"

describe "_cb_resolve_network_id precedence"

# Explicit network_id wins over chain_id.
CHAINBENCH_NETWORK_ID=7777
CHAINBENCH_CHAIN_ID=8283
assert_eq "$(_cb_resolve_network_id)" "7777" "explicit network_id wins over chain_id"

# Falls back to chain_id when network_id is empty (devnets match the two).
CHAINBENCH_NETWORK_ID=""
CHAINBENCH_CHAIN_ID=8283
assert_eq "$(_cb_resolve_network_id)" "8283" "falls back to chain_id when network_id empty"

# Empty when neither is set — caller omits --networkid, gstable uses its default.
CHAINBENCH_NETWORK_ID=""
CHAINBENCH_CHAIN_ID=""
assert_eq "$(_cb_resolve_network_id)" "" "empty when neither network_id nor chain_id set"

unit_summary
