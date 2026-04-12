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

burn_receipt=$(cat /tmp/chainbench-regression/last_burn_receipt.json 2>/dev/null || echo "")
if [[ -z "$burn_receipt" ]]; then
  _assert_fail "no burn receipt available (run f2-02 first)"
  test_result
  exit 1
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
