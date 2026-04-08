#!/usr/bin/env bash
# Test: regression/f-system-contracts/f4-03-masterminter-self-member
# RT-F-4-03 — GovMasterMinter 자체 멤버 추가/제거 (proposeAddMember, proposeRemoveMember)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f4-03-masterminter-self-member"
check_env || { test_result; exit 1; }

unlock_all_validators

# GovMasterMinter에 TEST_ACC_D를 member로 추가
propose_sel=$(selector "proposeAddMember(address,uint32)")
new_padded=$(pad_address "$TEST_ACC_D_ADDR" | sed 's/^0x//')
quorum_padded=$(pad_uint256 "3" | sed 's/^0x//')
propose_data="${propose_sel}${new_padded}${quorum_padded}"

receipt_add=$(gov_full_flow "$GOV_MASTER_MINTER" "$propose_data" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR") || {
  _assert_fail "add member flow failed"; test_result; exit 1
}
status_add=$(printf '%s' "$receipt_add" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status_add" "0x1" "addMember executed"

is_sel=$(selector "isActiveMember(address)")
is_member=$(hex_to_dec "$(eth_call_raw 1 "$GOV_MASTER_MINTER" "${is_sel}${new_padded}")")
assert_eq "$is_member" "1" "TEST_ACC_D added to GovMasterMinter members"

# 제거
unlock_all_validators
remove_sel=$(selector "proposeRemoveMember(address,uint32)")
q2_padded=$(pad_uint256 "2" | sed 's/^0x//')
remove_data="${remove_sel}${new_padded}${q2_padded}"

receipt_rm=$(gov_full_flow "$GOV_MASTER_MINTER" "$remove_data" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR") || {
  _assert_fail "remove member flow failed"; test_result; exit 1
}
status_rm=$(printf '%s' "$receipt_rm" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status_rm" "0x1" "removeMember executed"

after=$(hex_to_dec "$(eth_call_raw 1 "$GOV_MASTER_MINTER" "${is_sel}${new_padded}")")
assert_eq "$after" "0" "TEST_ACC_D removed from GovMasterMinter members"

test_result
