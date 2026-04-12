#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-D-03
# name: Sender 서명 변조 시 거부 (--tamper sender)
# category: regression/d-fee-delegation
# tags: [fee-delegation]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/d-fee-delegation/d-03-sender-sig-invalid
# RT-D-03 — Sender 서명 변조 시 거부 (--tamper sender)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/d-fee-delegation/d-03-sender-sig-invalid"
check_env || { test_result; exit 1; }

python3 -c "import rlp, eth_keys" 2>/dev/null || {
  _assert_fail "missing python deps (rlp, eth-keys)"
  test_result
  exit 1
}

HELPER="${CHAINBENCH_DIR}/tests/regression/lib/fee_delegate.py"

# tamper=sender로 서명 변조 후 전송
result=$(python3 "$HELPER" send \
  --rpc http://127.0.0.1:8501 \
  --sender-pk "$TEST_ACC_A_PK" \
  --fee-payer-pk "$TEST_ACC_C_PK" \
  --to "$TEST_ACC_B_ADDR" \
  --value 1 \
  --gas 21000 \
  --tamper sender 2>/dev/null) || true

# 에러 여부 확인
has_error=$(printf '%s' "$result" | python3 -c "
import sys, json
try:
    d = json.loads(sys.stdin.read())
    rpc_resp = d.get('rpcResponse', {})
    print('yes' if 'error' in rpc_resp else 'no')
except Exception:
    print('yes')
")

assert_eq "$has_error" "yes" "tx with tampered sender signature rejected"

# 에러 메시지 검증
err_msg=$(printf '%s' "$result" | python3 -c "
import sys, json
try:
    d = json.loads(sys.stdin.read())
    print(d.get('rpcResponse', {}).get('error', {}).get('message', ''))
except Exception:
    print('')
")
printf '[INFO]  error: %s\n' "$err_msg" >&2

if [[ "$err_msg" == *"invalid sender"* ]] || [[ "$err_msg" == *"invalid transaction"* ]] || [[ "$err_msg" == *"invalid signature"* ]] || [[ "$err_msg" == *"invalid"* ]]; then
  _assert_pass "error indicates invalid sender signature"
else
  _assert_fail "unexpected error message: $err_msg"
fi

test_result
