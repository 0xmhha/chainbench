#!/usr/bin/env bash
# tests/unit/tests/adapter-mapping.sh - Pin go-stablenet -> stablenet mapping (M1)
#
# go-stablenet runs on the `stablenet` adapter via the `default` profile (no
# separate go-stablenet profile exists). The adapter is named after the chain,
# while the consensus engine is WBFT — whose RPC namespace is istanbul. These
# tests pin that mapping so a future author cannot silently repoint the default
# profile at the unimplemented `wbft` stub adapter.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR
export CHAINBENCH_QUIET=1
source "${CHAINBENCH_DIR}/lib/common.sh"
source "${CHAINBENCH_DIR}/lib/profile.sh"
source "${CHAINBENCH_DIR}/lib/chain_adapter.sh"

# ---- Test 1: default profile resolves chain type stablenet ----------------
describe "default profile -> CHAINBENCH_CHAIN_TYPE=stablenet"
chain_type="$( load_profile default >/dev/null 2>&1; printf '%s' "${CHAINBENCH_CHAIN_TYPE:-}" )"
assert_eq "${chain_type}" "stablenet" "default.yaml resolves to the stablenet adapter"

# ---- Test 2: stablenet adapter consensus namespace is istanbul ------------
describe "stablenet adapter consensus RPC namespace is istanbul"
ns="$( cb_adapter_load stablenet >/dev/null 2>&1; adapter_consensus_rpc_namespace )"
assert_eq "${ns}" "istanbul" "stablenet (WBFT consensus) uses the istanbul namespace"

# ---- Test 3: wbft stub adapter genesis is not implemented ------------------
describe "wbft stub adapter genesis fails (not implemented)"
rc=0
( cb_adapter_load wbft >/dev/null 2>&1; adapter_generate_genesis >/dev/null 2>&1 ) || rc=$?
assert_neq "$rc" "0" "wbft genesis returns non-zero (mis-selection guard)"

unit_summary
