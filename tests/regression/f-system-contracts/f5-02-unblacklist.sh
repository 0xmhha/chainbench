#!/usr/bin/env bash
# ---chainbench-meta---
# id: 
# name: f5-02-unblacklist
# category: regression/f-system-contracts
# tags: [governance, access-control]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/f-system-contracts/f5-02-unblacklist
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/f-system-contracts/f5-02-unblacklist"
check_env || { test_result; exit 1; }

unlock_all_validators
target="$TEST_ACC_E_ADDR"
target_padded=$(pad_address "$target" | sed 's/^0x//')

# blacklisted 확인
is_sel=$(selector "isBlacklisted(address)")
before=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_sel}${target_padded}")")
if [[ "$before" != "1" ]]; then
  # blacklist 먼저
  propose_add=$(selector "proposeAddBlacklist(address)")
  gov_full_flow "$GOV_COUNCIL" "${propose_add}${target_padded}" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR" >/dev/null
fi

# unBlacklist
propose_sel=$(selector "proposeRemoveBlacklist(address)")
propose_data="${propose_sel}${target_padded}"
receipt=$(gov_full_flow "$GOV_COUNCIL" "$propose_data" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR") || {
  _assert_fail "flow failed"; test_result; exit 1
}
status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status" "0x1" "unBlacklist executed"

after=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_sel}${target_padded}")")
assert_eq "$after" "0" "target no longer blacklisted"

echo "$receipt" > /tmp/chainbench-regression/f5_02_receipt.json
test_result
