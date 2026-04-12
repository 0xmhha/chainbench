#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-E-09
# name: Authorized 계정 tx 실행 시 AuthorizedTxExecuted 이벤트 발생
# category: regression/e-blacklist-authorized
# tags: [blacklist, access-control]
# estimated_seconds: 35
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/e-blacklist-authorized/e-09-authorized-tx-executed
# RT-E-09 — Authorized 계정 tx 실행 시 AuthorizedTxExecuted 이벤트 발생
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/e-blacklist-authorized/e-09-authorized-tx-executed"
check_env || { test_result; exit 1; }

# TEST_ACC_D가 authorized인지 확인 (e-08 선행)
is_auth_sel=$(selector "isAuthorized(address)")
td_padded=$(pad_address "$TEST_ACC_D_ADDR" | sed 's/^0x//')
is_auth=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_auth_sel}${td_padded}")")

if [[ "$is_auth" != "1" ]]; then
  _assert_fail "TEST_ACC_D should be authorized (run e-08 first)"
  test_result
  exit 1
fi

# TEST_ACC_D로 tx 발행
tx_hash=$(send_raw_tx "1" "$TEST_ACC_D_PK" "$TEST_ACC_B_ADDR" "1" "" "21000" "dynamic")
receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
assert_not_empty "$receipt" "receipt retrieved"

# AccountManager(0xb00003) 주소의 AuthorizedTxExecuted 이벤트 검색
log=$(find_log_by_topic "$receipt" "$ACCOUNT_MANAGER" "$AUTHORIZED_TX_EXECUTED_SIG")
assert_not_empty "$log" "AuthorizedTxExecuted event found in receipt.logs"

# 마지막 log인지 확인 (code 주석: "Must be the last log added")
last_log_sig=$(printf '%s' "$receipt" | python3 -c "
import sys, json
r = json.load(sys.stdin)
logs = r.get('logs', [])
if logs:
    print(logs[-1].get('topics', [''])[0].lower())
else:
    print('')
")
expected_sig=$(echo "$AUTHORIZED_TX_EXECUTED_SIG" | tr '[:upper:]' '[:lower:]')
assert_eq "$last_log_sig" "$expected_sig" "AuthorizedTxExecuted is the last log (PR #70 derive dependency)"

test_result
