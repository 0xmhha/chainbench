#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-D-05
# name: FeePayer 잔액 부족 시 거부
# category: regression/d-fee-delegation
# tags: [fee-delegation, gas]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/d-fee-delegation/d-05-feepayer-insufficient
# RT-D-05 — FeePayer 잔액 부족 시 거부
#
# 새 랜덤 keypair 생성 → 잔액 0인 주소를 FeePayer로 사용 → 블록 실행 중 ErrInsufficientFunds
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/d-fee-delegation/d-05-feepayer-insufficient"
check_env || { test_result; exit 1; }

python3 -c "import rlp, eth_keys" 2>/dev/null || { _assert_fail "missing python deps"; test_result; exit 1; }

HELPER="${CHAINBENCH_DIR}/tests/regression/lib/fee_delegate.py"

# 잔액 0인 새 keypair 생성
empty_pk=$(python3 -c "
from eth_account import Account
import secrets
acct = Account.create()
print(acct.key.hex())
")
empty_addr=$(python3 -c "
from eth_account import Account
acct = Account.from_key('$empty_pk')
print(acct.address)
")

# 확인: 잔액 0
empty_bal=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${empty_addr}\", \"latest\"]" | json_get - result)")
assert_eq "$empty_bal" "0" "generated empty feePayer has 0 balance ($empty_addr)"

# Sender가 서명, FeePayer는 잔액 0인 주소로 지정
result=$(python3 "$HELPER" send \
  --rpc http://127.0.0.1:8501 \
  --sender-pk "$TEST_ACC_A_PK" \
  --fee-payer-pk "$empty_pk" \
  --to "$TEST_ACC_B_ADDR" \
  --value 1 \
  --gas 21000 2>&1)

# txpool 진입 시점 또는 블록 실행 시점에 거부
has_error=$(printf '%s' "$result" | python3 -c "
import sys, json
try:
    d = json.loads(sys.stdin.read())
    print('yes' if 'error' in d.get('rpcResponse', {}) else 'no')
except Exception:
    print('yes')
")
assert_eq "$has_error" "yes" "tx with empty feePayer rejected"

err_msg=$(printf '%s' "$result" | python3 -c "
import sys, json
try:
    d = json.loads(sys.stdin.read())
    print(d.get('rpcResponse', {}).get('error', {}).get('message', ''))
except Exception:
    print('')
")
printf '[INFO]  error: %s\n' "$err_msg" >&2

if [[ "$err_msg" == *"insufficient funds"* ]] || [[ "$err_msg" == *"feePayer"* ]]; then
  _assert_pass "error indicates feePayer insufficient funds"
else
  _assert_fail "unexpected error: $err_msg"
fi

test_result
