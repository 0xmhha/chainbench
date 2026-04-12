#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-F-2-03
# name: quorum 미달 → proposal 상태 Voting 유지, 발행 미실행
# category: regression/f-system-contracts
# tags: [governance, quorum]
# estimated_seconds: 65
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/f-system-contracts/f2-03-quorum-deficient
# RT-F-2-03 (v2) — quorum 미달 → proposal 상태 Voting 유지, 발행 미실행
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f2-03-quorum-deficient"
check_env || { test_result; exit 1; }

unlock_all_validators

beneficiary="$TEST_ACC_B_ADDR"
amount="100000000000000000"  # 0.1 ether
timestamp=$(date +%s)
deposit_id="REG-Q-$(date +%s%N)"
bank_ref="REG-BANK-Q-${timestamp}"
memo="quorum test"

tx_data=$(python3 <<PYEOF
from eth_abi import encode
from eth_utils import keccak
proof = encode(
    ['address', 'uint256', 'uint256', 'string', 'string', 'string'],
    ['${beneficiary}', ${amount}, ${timestamp}, '${deposit_id}', '${bank_ref}', '${memo}']
)
selector = keccak(text='proposeMint(bytes)')[:4]
call_data = encode(['bytes'], [proof])
print('0x' + (selector + call_data).hex())
PYEOF
)

# validator1이 propose (자동 1표)
tx_hash=$(gov_call "1" "$GOV_MINTER" "$tx_data" "$VALIDATOR_1_ADDR" 1500000)
wait_receipt "1" "$tx_hash" 30 >/dev/null
proposal_id=$(extract_proposal_id_from_receipt "1" "$tx_hash")
assert_not_empty "$proposal_id" "proposalId extracted"

# validator2 approve하지 않고 바로 execute 시도 → quorum 미달로 revert 예상
# GovMinter quorum = 2 → 1명만 투표한 상태
exec_tx=$(gov_execute "1" "$GOV_MINTER" "$proposal_id" "$VALIDATOR_1_ADDR" 2>/dev/null || echo "")
if [[ -n "$exec_tx" && "$exec_tx" != "null" ]]; then
  exec_receipt=$(wait_tx_receipt_full "1" "$exec_tx" 30)
  exec_status=$(printf '%s' "$exec_receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
  assert_eq "$exec_status" "0x0" "execute reverted (quorum deficient)"
fi

# proposalStatus == Voting (enum 1)
status=$(gov_proposal_status "1" "$GOV_MINTER" "$proposal_id" 2>/dev/null || echo "")
if [[ -n "$status" ]]; then
  assert_eq "$status" "1" "proposal state == Voting (1), not executed"
fi

test_result
