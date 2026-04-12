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

# proposeMint 시그니처는 GovMinter 구현에 따라 다름. 일반적으로:
#   proposeMint(address beneficiary, uint256 amount) 또는 proposeMint(bytes proofData)
# v1 GovMinter는 proposeMint(bytes proofData) 형식 — proofData 구조는 구현에 따라

# 단순화: 기본 구조 ABI로 proposeMintWithDeposit 또는 유사 함수 시도
# 정확한 시그니처를 모를 경우 F-2-01 실행을 먼저 하고 그 결과를 관찰

printf '[INFO]  This test requires F-2-01 (GovMinter proposeMint full flow) to have run\n' >&2
printf '[INFO]  Checking beneficiary Transfer event from the most recent mint receipt\n' >&2

# F-2-01에서 저장한 receipt 사용
mint_receipt=$(cat /tmp/chainbench-regression/last_mint_receipt.json 2>/dev/null || echo "")
if [[ -z "$mint_receipt" ]]; then
  _assert_fail "no mint receipt available (run f2-01 first)"
  test_result
  exit 1
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
