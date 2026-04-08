#!/usr/bin/env bash
# Test: regression/a-ethereum/a4-04-eth-get-logs
# RT-A-4-04 — eth_getLogs 이벤트 로그 조회
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a4-04-eth-get-logs"
check_env || { test_result; exit 1; }

# NativeCoinAdapter Transfer 이벤트는 모든 value transfer에서 발생 (Anzeon)
# TEST_ACC_A → TEST_ACC_B 송금
tx_hash=$(send_raw_tx "1" "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1000000000000000" "" "21000" "dynamic")
receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
block_num=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('blockNumber', ''))")

# NativeCoinAdapter address 필터로 log 조회
logs=$(rpc "1" "eth_getLogs" "[{\"fromBlock\":\"${block_num}\",\"toBlock\":\"${block_num}\",\"address\":\"${NATIVE_COIN_ADAPTER}\"}]")
log_count=$(printf '%s' "$logs" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', [])
print(len(r))
")
assert_gt "$log_count" "0" "eth_getLogs returned at least 1 log"

# Transfer topic 필터
transfer_logs=$(rpc "1" "eth_getLogs" "[{\"fromBlock\":\"${block_num}\",\"toBlock\":\"${block_num}\",\"topics\":[\"${TRANSFER_EVENT_SIG}\"]}]")
transfer_count=$(printf '%s' "$transfer_logs" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', [])
print(len(r))
")
assert_gt "$transfer_count" "0" "eth_getLogs with Transfer topic filter returned logs"

test_result
