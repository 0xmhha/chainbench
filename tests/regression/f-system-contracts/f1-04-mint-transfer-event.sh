#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-F-1-04
# name: Mint 실행 시 Transfer(0x0 → beneficiary) 이벤트 발생
# category: regression/f-system-contracts
# tags: [governance, mint, transfer, event]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/f-system-contracts/f1-04-mint-transfer-event
# RT-F-1-04 (v2) — Mint 실행 시 Transfer(0x0 → beneficiary) 이벤트 발생
#
# GovMinter proposeMint → 승인 → execute → NativeCoinAdapter Transfer(from=0x0) 발생
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f1-04-mint-transfer-event"
check_env || { test_result; exit 1; }

unlock_all_validators

# F-2-01에서 저장한 receipt 사용 (없으면 직접 mint 수행)
mint_receipt=$(cat /tmp/chainbench-regression/last_mint_receipt.json 2>/dev/null || echo "")
if [[ -z "$mint_receipt" ]]; then
  printf '[INFO]  No mint receipt cached — running inline mint flow\n' >&2
  beneficiary="$TEST_ACC_B_ADDR"
  amount="1000000000000000000"
  timestamp=$(date +%s)
  deposit_id="F104-DEP-$(date +%s%N)"
  bank_ref="F104-BANK-${timestamp}"
  memo="f1-04 inline mint"

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
  ) || { _assert_fail "eth_abi encoding failed"; test_result; exit 1; }

  tx_hash=$(gov_call "1" "$GOV_MINTER" "$tx_data" "$VALIDATOR_1_ADDR" 1500000)
  propose_receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
  proposal_id=$(extract_proposal_id_from_receipt "1" "$tx_hash")

  approve_tx=$(gov_approve "1" "$GOV_MINTER" "$proposal_id" "$VALIDATOR_2_ADDR")
  sleep 2
  prop_status=$(gov_proposal_status "1" "$GOV_MINTER" "$proposal_id")
  if [[ "$prop_status" == "3" ]]; then
    approve_node=$(addr_to_node "$VALIDATOR_2_ADDR")
    mint_receipt=$(wait_tx_receipt_full "$approve_node" "$approve_tx" 30)
  else
    exec_tx=$(gov_execute "1" "$GOV_MINTER" "$proposal_id" "$VALIDATOR_1_ADDR")
    mint_receipt=$(wait_tx_receipt_full "1" "$exec_tx" 30)
  fi
  echo "$mint_receipt" > /tmp/chainbench-regression/last_mint_receipt.json
fi

# NativeCoinAdapter Transfer 이벤트 검색
log=$(find_log_by_topic "$mint_receipt" "$NATIVE_COIN_ADAPTER" "$TRANSFER_EVENT_SIG")
assert_not_empty "$log" "Transfer event present in mint receipt"

# topics[1] (from)이 0x0인지 확인
from_topic=$(printf '%s' "$log" | python3 -c "
import sys, json
l = json.load(sys.stdin)
topics = l.get('topics', [])
if len(topics) >= 2:
    print(int(topics[1], 16))
else:
    print(-1)
")
assert_eq "$from_topic" "0" "Transfer.from == 0x0 (mint)"

test_result
