#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-F-2-01
# name: 코인 발행: proposeMint(proofData) → 승인 → execute
# category: regression/f-system-contracts
# tags: [governance, proposal, mint]
# estimated_seconds: 67
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# Test: regression/f-system-contracts/f2-01-mint-proposal
# RT-F-2-01 — 코인 발행: proposeMint(proofData) → 승인 → execute
#
# ABI (v1 GovMinter.sol:202):
#   function proposeMint(bytes memory proofData)
#   proofData = abi.encode(beneficiary, amount, timestamp, depositId, bankReference, memo)
#   (MintProof 구조체의 각 필드를 순서대로 encode)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f2-01-mint-proposal"
check_env || { test_result; exit 1; }
ensure_nodes_running

unlock_all_validators

beneficiary="$(acct_addr 2)"
amount="1000000000000000000"  # 1 ether
timestamp=$(date +%s)
deposit_id="REG-DEP-$(date +%s%N)"
bank_ref="REG-BANK-${timestamp}"
memo="regression mint"

bal_before=$(hex_to_dec "$(rpc "$(node 1)" eth_getBalance "[\"${beneficiary}\", \"latest\"]" | json_get - result)")

# Python으로 proposeMint tx data 생성 (eth_abi로 bytes 인코딩)
tx_data=$(python3 <<PYEOF
from eth_abi import encode
from eth_utils import keccak

# MintProof fields
proof = encode(
    ['address', 'uint256', 'uint256', 'string', 'string', 'string'],
    ['${beneficiary}', ${amount}, ${timestamp}, '${deposit_id}', '${bank_ref}', '${memo}']
)

# proposeMint(bytes) 함수 호출 인코딩
selector = keccak(text='proposeMint(bytes)')[:4]
# bytes 파라미터는 동적 타입 → ABI는 [offset, length, payload]
call_data = encode(['bytes'], [proof])
print('0x' + (selector + call_data).hex())
PYEOF
) || { _assert_fail "eth_abi encoding failed"; test_result; exit 1; }

tx_hash=$(gov_call "1" "$GOV_MINTER" "$tx_data" "$(validator_addr 1)" 1500000)
if [[ -z "$tx_hash" || "$tx_hash" == "null" ]]; then
  _assert_fail "proposeMint tx submission failed"
  test_result
  exit 1
fi

propose_receipt=$(wait_tx_receipt_full "$(node 1)" "$tx_hash" 30)
propose_status=$(printf '%s' "$propose_receipt" | jq -r '.status // empty')
assert_eq "$propose_status" "0x1" "proposeMint receipt status == 0x1"

proposal_id=$(extract_proposal_id_from_receipt "1" "$tx_hash")
assert_not_empty "$proposal_id" "proposalId extracted"
printf '[INFO]  mint proposal_id=%s\n' "$proposal_id" >&2

# validator2 approve → quorum 달성 (GovBase는 quorum 도달 시 approve 내에서 자동 실행)
approve_tx=$(gov_approve "$(node 1)" "$GOV_MINTER" "$proposal_id" "$(validator_addr 2)")
sleep 2

# approve 후 proposal 상태 확인 — auto-execute 되었으면 별도 execute 불필요
prop_status=$(gov_proposal_status "$(node 1)" "$GOV_MINTER" "$proposal_id")
if [[ "$prop_status" == "3" ]]; then
  # Auto-executed during approve
  approve_node=$(addr_to_node "$(validator_addr 2)")
  exec_receipt=$(wait_tx_receipt_full "$approve_node" "$approve_tx" 30)
  echo "$exec_receipt" > /tmp/chainbench-regression/last_mint_receipt.json
else
  # Manual execute required
  exec_tx=$(gov_execute "$(node 1)" "$GOV_MINTER" "$proposal_id" "$(validator_addr 1)")
  exec_receipt=$(wait_tx_receipt_full "$(node 1)" "$exec_tx" 30)
  echo "$exec_receipt" > /tmp/chainbench-regression/last_mint_receipt.json
fi

exec_status=$(printf '%s' "$exec_receipt" | jq -r '.status // empty')
assert_eq "$exec_status" "0x1" "mint execute receipt.status == 0x1"

# beneficiary 잔액 증가
bal_after=$(hex_to_dec "$(rpc "$(node 1)" eth_getBalance "[\"${beneficiary}\", \"latest\"]" | json_get - result)")
diff=$(( bal_after - bal_before ))
assert_eq "$diff" "$amount" "beneficiary balance increased by $amount"

test_result
