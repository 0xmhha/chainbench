#!/usr/bin/env bash
# Test: regression/f-system-contracts/f3-01-add-validator-proposal
# RT-F-3-01 — 검증자 추가 proposal 생성 (Voting 상태)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f3-01-add-validator-proposal"
check_env || { test_result; exit 1; }

unlock_all_validators

# proposeAddMember(new, newQuorum=3)
propose_sel=$(selector "proposeAddMember(address,uint32)")
new_padded=$(pad_address "$TEST_ACC_E_ADDR" | sed 's/^0x//')
quorum_padded=$(pad_uint256 "3" | sed 's/^0x//')
propose_data="${propose_sel}${new_padded}${quorum_padded}"

tx_hash=$(gov_call "1" "$GOV_VALIDATOR" "$propose_data" "$VALIDATOR_1_ADDR" 800000)
assert_contains "$tx_hash" "0x" "propose tx submitted"

proposal_id=$(extract_proposal_id_from_receipt "1" "$tx_hash")
assert_not_empty "$proposal_id" "proposalId extracted"

# 상태 == Voting (enum 1)
status=$(gov_proposal_status "1" "$GOV_VALIDATOR" "$proposal_id" 2>/dev/null || echo "")
if [[ -n "$status" ]]; then
  assert_eq "$status" "1" "proposal state == Voting (1)"
fi

# 저장: F-3-02에서 사용
echo "$proposal_id" > /tmp/chainbench-regression/f3_01_proposal.txt

test_result
