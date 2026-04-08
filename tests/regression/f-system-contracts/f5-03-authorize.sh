#!/usr/bin/env bash
# Test: regression/f-system-contracts/f5-03-authorize
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/f-system-contracts/f5-03-authorize"
check_env || { test_result; exit 1; }

unlock_all_validators
target="$TEST_ACC_D_ADDR"
target_padded=$(pad_address "$target" | sed 's/^0x//')

is_sel=$(selector "isAuthorized(address)")
before=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_sel}${target_padded}")")
if [[ "$before" == "1" ]]; then
  # unAuthorize 먼저
  unp=$(selector "proposeRemoveAuthorizedAccount(address)")
  gov_full_flow "$GOV_COUNCIL" "${unp}${target_padded}" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR" >/dev/null
fi

propose_sel=$(selector "proposeAddAuthorizedAccount(address)")
propose_data="${propose_sel}${target_padded}"
receipt=$(gov_full_flow "$GOV_COUNCIL" "$propose_data" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR") || {
  _assert_fail "flow failed"; test_result; exit 1
}
status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status" "0x1" "authorize executed"

after=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_sel}${target_padded}")")
assert_eq "$after" "1" "target is now authorized"

echo "$receipt" > /tmp/chainbench-regression/f5_03_receipt.json
test_result
