#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-F-5-01
# name: GovCouncil blacklist proposal → execute → isBlacklisted == true
# category: regression/f-system-contracts
# tags: [governance, access-control]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# Test: regression/f-system-contracts/f5-01-blacklist
# RT-F-5-01 — GovCouncil blacklist proposal → execute → isBlacklisted == true
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f5-01-blacklist"
check_env || { test_result; exit 1; }
ensure_nodes_running

unlock_all_validators

target="$(acct_addr 5)"
target_padded=$(pad_address "$target" | sed 's/^0x//')

# 이미 blacklisted면 먼저 unBlacklist
is_bl_sel=$(selector "isBlacklisted(address)")
before=$(hex_to_dec "$(eth_call_raw "$(node 1)" "$ACCOUNT_MANAGER" "${is_bl_sel}${target_padded}")")
if [[ "$before" == "1" ]]; then
  unbl_sel=$(selector "proposeRemoveBlacklist(address)")
  unbl_data="${unbl_sel}${target_padded}"
  gov_full_flow "$GOV_COUNCIL" "$unbl_data" "$(validator_addr 1)" "$(validator_addr 2)" >/dev/null || true
fi

# proposeAddBlacklist
propose_sel=$(selector "proposeAddBlacklist(address)")
propose_data="${propose_sel}${target_padded}"

receipt=$(gov_full_flow "$GOV_COUNCIL" "$propose_data" "$(validator_addr 1)" "$(validator_addr 2)") || {
  _assert_fail "flow failed"; test_result; exit 1
}
status=$(printf '%s' "$receipt" | jq -r '.status // empty')
assert_eq "$status" "0x1" "blacklist proposal executed"

after=$(hex_to_dec "$(eth_call_raw "$(node 1)" "$ACCOUNT_MANAGER" "${is_bl_sel}${target_padded}")")
assert_eq "$after" "1" "target blacklisted"

echo "$receipt" > /tmp/chainbench-regression/f5_01_receipt.json

test_result
