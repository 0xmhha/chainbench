#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-1-04
# name: eth_getTransactionReceipt (PR #70 fix 확인)
# category: regression/g-api
# tags: [rpc]
# estimated_seconds: 38
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# RT-G-1-04 — eth_getTransactionReceipt (PR #70 fix 확인)
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g1-04-get-tx-receipt"
check_env || { test_result; exit 1; }

tx_hash=$(send_raw_tx "1" "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1" "" "21000" "dynamic")
receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)

# status, effectiveGasPrice, logs 필드 확인
for field in status effectiveGasPrice logs; do
  val=$(printf '%s' "$receipt" | python3 -c "
import sys, json
r = json.load(sys.stdin)
v = r.get('${field}')
print('yes' if v is not None else 'no')
")
  assert_eq "$val" "yes" "receipt has $field field"
done

# BP/EN 양쪽 동일 확인 (PR #70 회귀)
sleep 3
en_receipt=$(get_receipt "5" "$tx_hash")
if [[ -n "$en_receipt" && "$en_receipt" != "null" ]]; then
  bp_egp=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('effectiveGasPrice', ''))")
  en_egp=$(printf '%s' "$en_receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('effectiveGasPrice', ''))")
  assert_eq "$en_egp" "$bp_egp" "BP and EN return identical effectiveGasPrice"
fi

test_result
