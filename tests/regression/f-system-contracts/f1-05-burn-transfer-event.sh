#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-F-1-05
# name: Burn 실행 시 Transfer(account → 0x0) 이벤트 발생
# category: regression/f-system-contracts
# tags: [governance, mint, transfer, event]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/f-system-contracts/f1-05-burn-transfer-event
# RT-F-1-05 (v2) — Burn 실행 시 Transfer(account → 0x0) 이벤트 발생
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f1-05-burn-transfer-event"
check_env || { test_result; exit 1; }

unlock_all_validators

burn_receipt=$(cat /tmp/chainbench-regression/last_burn_receipt.json 2>/dev/null || echo "")
if [[ -z "$burn_receipt" ]]; then
  printf '[INFO]  No burn receipt cached — running inline burn flow\n' >&2
  from_addr="$VALIDATOR_1_ADDR"
  amount="500000000000000000"
  timestamp=$(date +%s)
  withdrawal_id="F105-WD-$(date +%s%N)"
  ref_id="F105-REF-${timestamp}"
  memo="f1-05 inline burn"

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

  amount_hex=$(dec_to_hex "$amount")
  tx_hash=$(rpc "1" "eth_sendTransaction" \
    "[{\"from\":\"${from_addr}\",\"to\":\"${GOV_MINTER}\",\"data\":\"${tx_data}\",\"gas\":\"0x16e360\",\"value\":\"${amount_hex}\"}]" | json_get - result)
  propose_receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
  proposal_id=$(extract_proposal_id_from_receipt "1" "$tx_hash")

  approve_tx=$(gov_approve "1" "$GOV_MINTER" "$proposal_id" "$VALIDATOR_2_ADDR")
  sleep 2
  prop_status=$(gov_proposal_status "1" "$GOV_MINTER" "$proposal_id")
  if [[ "$prop_status" == "3" ]]; then
    approve_node=$(addr_to_node "$VALIDATOR_2_ADDR")
    burn_receipt=$(wait_tx_receipt_full "$approve_node" "$approve_tx" 30)
  else
    exec_tx=$(gov_execute "1" "$GOV_MINTER" "$proposal_id" "$VALIDATOR_1_ADDR")
    burn_receipt=$(wait_tx_receipt_full "1" "$exec_tx" 30)
  fi
  echo "$burn_receipt" > /tmp/chainbench-regression/last_burn_receipt.json
fi

log=$(find_log_by_topic "$burn_receipt" "$NATIVE_COIN_ADAPTER" "$TRANSFER_EVENT_SIG")
assert_not_empty "$log" "Transfer event present in burn receipt"

to_topic=$(printf '%s' "$log" | python3 -c "
import sys, json
l = json.load(sys.stdin)
topics = l.get('topics', [])
if len(topics) >= 3:
    print(int(topics[2], 16))
else:
    print(-1)
")
assert_eq "$to_topic" "0" "Transfer.to == 0x0 (burn)"

test_result
