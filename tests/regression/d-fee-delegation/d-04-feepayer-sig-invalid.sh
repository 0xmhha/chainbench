#!/usr/bin/env bash
# Test: regression/d-fee-delegation/d-04-feepayer-sig-invalid
# RT-D-04 — FeePayer 서명 변조 시 거부
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/d-fee-delegation/d-04-feepayer-sig-invalid"
check_env || { test_result; exit 1; }

python3 -c "import rlp, eth_keys" 2>/dev/null || { _assert_fail "missing python deps"; test_result; exit 1; }

HELPER="${CHAINBENCH_DIR}/tests/regression/lib/fee_delegate.py"

result=$(python3 "$HELPER" send \
  --rpc http://127.0.0.1:8501 \
  --sender-pk "$TEST_ACC_A_PK" \
  --fee-payer-pk "$TEST_ACC_C_PK" \
  --to "$TEST_ACC_B_ADDR" \
  --value 1 \
  --gas 21000 \
  --tamper feepayer 2>&1)

has_error=$(printf '%s' "$result" | python3 -c "
import sys, json
try:
    d = json.loads(sys.stdin.read())
    print('yes' if 'error' in d.get('rpcResponse', {}) else 'no')
except Exception:
    print('yes')
")
assert_eq "$has_error" "yes" "tx with tampered feePayer signature rejected"

err_msg=$(printf '%s' "$result" | python3 -c "
import sys, json
try:
    d = json.loads(sys.stdin.read())
    print(d.get('rpcResponse', {}).get('error', {}).get('message', ''))
except Exception:
    print('')
")
printf '[INFO]  error: %s\n' "$err_msg" >&2

# 에러 메시지에 feePayer 또는 signature 관련 키워드
if [[ "$err_msg" == *"feePayer"* ]] || [[ "$err_msg" == *"fee payer"* ]] || [[ "$err_msg" == *"invalid"* ]] || [[ "$err_msg" == *"signature"* ]]; then
  _assert_pass "error indicates invalid feePayer signature"
else
  _assert_fail "unexpected error message: $err_msg"
fi

test_result
