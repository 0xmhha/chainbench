#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-F-3-03
# name: 검증자 제거 proposal → 승인 → execute
# category: regression/f-system-contracts
# tags: [governance, validator]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/f-system-contracts/f3-03-remove-validator
# RT-F-3-03 — 검증자 제거 proposal → 승인 → execute
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f3-03-remove-validator"
check_env || { test_result; exit 1; }

unlock_all_validators

# F-3-02에서 추가된 TEST_ACC_E를 제거
target="$TEST_ACC_E_ADDR"
target_padded=$(pad_address "$target" | sed 's/^0x//')

# GovBase에 isActiveMember() public view는 없음. members mapping의 automatic getter 사용.
members_sel=$(selector "members(address)")
members_result=$(eth_call_raw 1 "$GOV_VALIDATOR" "${members_sel}${target_padded}")
is_member=$(python3 -c "
raw = '${members_result}'.removeprefix('0x')
print(int(raw[:64], 16) if len(raw) >= 64 else 0)
")
if [[ "$is_member" != "1" ]]; then
  _assert_pass "target not a member — skip"
  test_result
  exit 0
fi

# proposeRemoveMember(addr, newQuorum=2)
propose_sel=$(selector "proposeRemoveMember(address,uint32)")
quorum_padded=$(pad_uint256 "2" | sed 's/^0x//')
propose_data="${propose_sel}${target_padded}${quorum_padded}"

# f3-01에서 quorum=3으로 변경했으므로, approve 3명 필요 (proposer + 2 approvers)
receipt=$(gov_full_flow "$GOV_VALIDATOR" "$propose_data" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR" "$VALIDATOR_3_ADDR") || {
  _assert_fail "flow failed"; test_result; exit 1
}

exec_status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$exec_status" "0x1" "execute succeeded"

members_result_after=$(eth_call_raw 1 "$GOV_VALIDATOR" "${members_sel}${target_padded}")
after=$(python3 -c "
raw = '${members_result_after}'.removeprefix('0x')
print(int(raw[:64], 16) if len(raw) >= 64 else 0)
")
assert_eq "$after" "0" "TEST_ACC_E removed from GovValidator members"

test_result
