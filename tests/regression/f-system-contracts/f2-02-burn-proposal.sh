#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-F-2-02
# name: 코인 소각: proposeBurn(proofData) payable → 승인 → execute
# category: regression/f-system-contracts
# tags: [governance, proposal, mint]
# estimated_seconds: 67
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/f-system-contracts/f2-02-burn-proposal
# RT-F-2-02 — 코인 소각: proposeBurn(proofData) payable → 승인 → execute
#
# ABI (v1 GovMinter.sol:272):
#   function proposeBurn(bytes calldata proofData) external payable
#   proofData = abi.encode(from, amount, timestamp, withdrawalId, referenceId, memo)
#   msg.value == proof.amount 필수 (proposer가 burn 대상이어야 함: BurnFromMustBeProposer)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f2-02-burn-proposal"
check_env || { test_result; exit 1; }

unlock_all_validators

# proposer 본인(validator1)이 burn from 대상이 되어야 함 (BurnFromMustBeProposer 조건)
from_addr="$VALIDATOR_1_ADDR"
amount="500000000000000000"  # 0.5 ether
timestamp=$(date +%s)
withdrawal_id="REG-WD-$(date +%s%N)"
ref_id="REG-REF-${timestamp}"
memo="regression burn"

bal_before=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${from_addr}\", \"latest\"]" | json_get - result)")

tx_data=$(python3 <<PYEOF
from eth_abi import encode
from eth_utils import keccak
proof = encode(
    ['address', 'uint256', 'uint256', 'string', 'string', 'string'],
    ['${from_addr}', ${amount}, ${timestamp}, '${withdrawal_id}', '${ref_id}', '${memo}']
)
selector = keccak(text='proposeBurn(bytes)')[:4]
call_data = encode(['bytes'], [proof])
print('0x' + (selector + call_data).hex())
PYEOF
) || { _assert_fail "eth_abi encoding failed"; test_result; exit 1; }

# proposeBurn은 payable — value == amount 필수
amount_hex=$(dec_to_hex "$amount")

tx_hash=$(rpc "1" "eth_sendTransaction" \
  "[{\"from\":\"${from_addr}\",\"to\":\"${GOV_MINTER}\",\"data\":\"${tx_data}\",\"gas\":\"0x16e360\",\"value\":\"${amount_hex}\"}]" | json_get - result)

if [[ -z "$tx_hash" || "$tx_hash" == "null" ]]; then
  _assert_fail "proposeBurn tx submission failed"
  test_result
  exit 1
fi

propose_receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
propose_status=$(printf '%s' "$propose_receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$propose_status" "0x1" "proposeBurn receipt status == 0x1"

proposal_id=$(extract_proposal_id_from_receipt "1" "$tx_hash")
printf '[INFO]  burn proposal_id=%s\n' "$proposal_id" >&2

# validator2 approve → quorum 달성 (GovBase는 quorum 도달 시 approve 내에서 자동 실행)
approve_tx=$(gov_approve "1" "$GOV_MINTER" "$proposal_id" "$VALIDATOR_2_ADDR")
sleep 2

# approve 후 proposal 상태 확인 — auto-execute 되었으면 별도 execute 불필요
prop_status=$(gov_proposal_status "1" "$GOV_MINTER" "$proposal_id")
if [[ "$prop_status" == "3" ]]; then
  approve_node=$(addr_to_node "$VALIDATOR_2_ADDR")
  exec_receipt=$(wait_tx_receipt_full "$approve_node" "$approve_tx" 30)
  echo "$exec_receipt" > /tmp/chainbench-regression/last_burn_receipt.json
else
  exec_tx=$(gov_execute "1" "$GOV_MINTER" "$proposal_id" "$VALIDATOR_1_ADDR")
  exec_receipt=$(wait_tx_receipt_full "1" "$exec_tx" 30)
  echo "$exec_receipt" > /tmp/chainbench-regression/last_burn_receipt.json
fi

exec_status=$(printf '%s' "$exec_receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$exec_status" "0x1" "burn execute receipt.status == 0x1"

test_result
