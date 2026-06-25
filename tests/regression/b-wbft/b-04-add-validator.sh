#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-B-04
# name: 검증자 member 추가 (GovBase proposeAddMember + voting)
# category: regression/b-wbft
# tags: [wbft, validator]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# Test: regression/b-wbft/b-04-add-validator
# RT-B-04 — 검증자 member 추가 (GovBase proposeAddMember + voting)
#
# 시나리오:
#   1. 현재 4 validator (member), quorum=2
#   2. validator1이 proposeAddMember(TEST_ACC_D, newQuorum=3) 호출
#   3. validator2가 approveProposal → quorum(2) 달성 → 상태 Approved
#   4. validator1이 executeProposal 호출 → TEST_ACC_D가 GovValidator member로 추가
#   5. validator 목록 자체는 변하지 않음 (configureValidator 미호출)
#      실제 validator로 활동하려면 새 member가 configureValidator + BLS key 등록 필요
#
# 참고: 이 TC는 "member 추가"까지만 검증. 실제 validator 추가는 BLS 키 프로비저닝이 필요
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-04-add-validator"
check_env || { test_result; exit 1; }
ensure_nodes_running

# 모든 validator keystore unlock
unlock_all_validators

# 초기 member 수 확인 (memberList 조회) — GovValidator 주소
# versionedMemberList(uint32) — memberVersion=1
sel=$(selector "memberList()")
before_data=$(eth_call_raw "$(node 1)" "$GOV_VALIDATOR" "$sel" 2>/dev/null || echo "0x")
printf '[INFO]  memberList result (hex): %s\n' "${before_data:0:100}" >&2

# proposeAddMember(TEST_ACC_D, newQuorum=2)
# NOTE: quorum을 2(현재값)로 유지하여 후속 b-05-remove-validator에서도
# 2명 approve로 충분하도록 함 (이전 newQuorum=3은 b-05의 approve 수 부족 야기)
propose_sel=$(selector "proposeAddMember(address,uint32)")
new_member_padded=$(pad_address "$(acct_addr 4)" | sed 's/^0x//')
new_quorum_padded=$(pad_uint256 "2" | sed 's/^0x//')
propose_data="${propose_sel}${new_member_padded}${new_quorum_padded}"

receipt=$(gov_full_flow "$GOV_VALIDATOR" "$propose_data" "$(validator_addr 1)" "$(validator_addr 2)") || {
  _assert_fail "gov_full_flow failed"
  test_result
  exit 1
}

# execute tx status 확인
exec_status=$(printf '%s' "$receipt" | python3 -c "import sys, json; r = json.load(sys.stdin); print(r.get('status', ''))")
assert_eq "$exec_status" "0x1" "executeProposal receipt status == 0x1"

# members(address) → (bool isActive, uint32 joinedAt) automatic getter
# GovBase에 isActiveMember() public view는 없음. members mapping의 automatic getter 사용.
members_sel=$(selector "members(address)")
target_padded=$(pad_address "$(acct_addr 4)" | sed 's/^0x//')
members_result=$(eth_call_raw "$(node 1)" "$GOV_VALIDATOR" "${members_sel}${target_padded}")
is_member=$(python3 -c "
raw = '${members_result}'.removeprefix('0x')
print(int(raw[:64], 16) if len(raw) >= 64 else 0)
")
assert_eq "$is_member" "1" "TEST_ACC_D is now an active member of GovValidator"

# validator list는 변하지 않음 (configureValidator 미호출)
val_list=$(rpc "$(node 1)" "istanbul_getValidators" '["latest"]' | python3 -c "
import sys, json
print(len(json.load(sys.stdin).get('result', [])))
")
assert_eq "$val_list" "4" "validator count unchanged (4) — configureValidator not called"

# 상태 저장 (b-05가 이어서 제거 테스트)
echo "$(acct_addr 4)" > /tmp/chainbench-regression/added_member.txt

test_result
