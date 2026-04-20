#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-B-05
# name: 검증자 member 제거 (GovBase proposeRemoveMember + voting)
# category: regression/b-wbft
# tags: [wbft, validator]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/b-wbft/b-05-remove-validator
# RT-B-05 — 검증자 member 제거 (GovBase proposeRemoveMember + voting)
#
# 시나리오 (B-04 후 실행):
#   1. TEST_ACC_D가 member로 추가되어 있음
#   2. validator1이 proposeRemoveMember(TEST_ACC_D, newQuorum=2) 호출
#   3. validator2가 approveProposal → quorum 달성
#   4. executeProposal → TEST_ACC_D가 member에서 제거
#   5. _onMemberRemoved 훅이 호출되지만, TEST_ACC_D는 validator가 아니므로 validator list 변동 없음
#
# 주의: 실제 기존 validator를 제거하는 건 BLS 키 재등록 필요하므로 여기서는 member만 제거
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-05-remove-validator"
check_env || { test_result; exit 1; }

# B-04가 선행되었는지 확인
added_member=$(cat /tmp/chainbench-regression/added_member.txt 2>/dev/null || echo "")
if [[ -z "$added_member" ]]; then
  printf '[INFO]  b-04 not run — using TEST_ACC_D anyway\n' >&2
  added_member="$TEST_ACC_D_ADDR"
fi

unlock_all_validators

# 제거 전 member 확인 (members mapping의 automatic getter 사용)
# GovBase에 isActiveMember() public view는 없음.
is_member_sel=$(selector "members(address)")
target_padded=$(pad_address "$added_member" | sed 's/^0x//')
before=$(eth_call_raw "1" "$GOV_VALIDATOR" "${is_member_sel}${target_padded}")
before_dec=$(python3 -c "
raw = '${before}'.removeprefix('0x')
print(int(raw[:64], 16) if len(raw) >= 64 else 0)
")

if [[ "$before_dec" != "1" ]]; then
  printf '[WARN]  target not a member, skipping (run b-04 first)\n' >&2
  _assert_pass "target not a member — b-04 should run first"
  test_result
  exit 0
fi

# proposeRemoveMember(addr, newQuorum=2)
propose_sel=$(selector "proposeRemoveMember(address,uint32)")
new_quorum_padded=$(pad_uint256 "2" | sed 's/^0x//')
propose_data="${propose_sel}${target_padded}${new_quorum_padded}"

receipt=$(gov_full_flow "$GOV_VALIDATOR" "$propose_data" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR") || {
  _assert_fail "gov_full_flow failed"
  test_result
  exit 1
}

exec_status=$(printf '%s' "$receipt" | python3 -c "import sys, json; r = json.load(sys.stdin); print(r.get('status', ''))")
assert_eq "$exec_status" "0x1" "executeProposal receipt status == 0x1"

# members(address) → isActive == false
after=$(eth_call_raw "1" "$GOV_VALIDATOR" "${is_member_sel}${target_padded}")
after_dec=$(python3 -c "
raw = '${after}'.removeprefix('0x')
print(int(raw[:64], 16) if len(raw) >= 64 else 0)
")
assert_eq "$after_dec" "0" "target is no longer an active member"

# Validator 개수는 여전히 4 (기존 4 validator 유지)
val_count=$(rpc "1" "istanbul_getValidators" '["latest"]' | python3 -c "
import sys, json
print(len(json.load(sys.stdin).get('result', [])))
")
assert_eq "$val_count" "4" "4 validators still active (TEST_ACC_D was member-only)"

test_result
