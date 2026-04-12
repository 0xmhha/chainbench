#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-4-03
# name: eth_sendRawTransaction 서명된 tx 전파
# category: regression/a-ethereum
# tags: [rpc, sendRawTransaction]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/a-ethereum/a4-03-send-raw-tx
# RT-A-4-03 — eth_sendRawTransaction 서명된 tx 전파
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a4-03-send-raw-tx"
check_env || { test_result; exit 1; }

# Raw tx 전송 (common.sh의 send_raw_tx 사용)
tx_hash=$(send_raw_tx "1" "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1" "" "21000" "dynamic")
assert_contains "$tx_hash" "0x" "txHash returned from eth_sendRawTransaction"

# 블록 포함되기 전에 txpool에서 조회 가능한지 확인 (짧은 시간 내)
# txpool_content로 pending 영역 확인
pending_info=$(rpc "1" "txpool_content" "[]")
in_pending=$(printf '%s' "$pending_info" | python3 -c "
import sys, json
data = json.load(sys.stdin)
pending = data.get('result', {}).get('pending', {})
target_hash = '${tx_hash}'.lower()
found = False
for addr, nonce_map in pending.items():
    for nonce, tx in nonce_map.items():
        if tx.get('hash', '').lower() == target_hash:
            found = True
            break
print('yes' if found else 'no')
")

# tx가 이미 블록에 포함되었을 수도 있으므로, pending 또는 mined 둘 중 하나로 확인
if [[ "$in_pending" == "yes" ]]; then
  _assert_pass "tx found in txpool.pending before mining"
else
  # 이미 블록에 포함된 경우 receipt 조회
  status=$(wait_receipt "1" "$tx_hash" 10 2>/dev/null || echo "")
  if [[ "$status" == "success" ]]; then
    _assert_pass "tx mined quickly (not in pending anymore)"
  else
    _assert_fail "tx neither in pending nor mined: $tx_hash"
  fi
fi

test_result
