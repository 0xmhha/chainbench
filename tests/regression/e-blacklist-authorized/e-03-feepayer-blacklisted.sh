#!/usr/bin/env bash
# Test: regression/e-blacklist-authorized/e-03-feepayer-blacklisted
# RT-E-03 — FeePayer가 블랙리스트 계정이면 거부
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/e-blacklist-authorized/e-03-feepayer-blacklisted"
check_env || { test_result; exit 1; }

python3 -c "import rlp, eth_keys" 2>/dev/null || { _assert_fail "missing python deps"; test_result; exit 1; }

# TEST_ACC_E가 여전히 blacklist 상태여야 함 (e-01 후)
target_padded=$(pad_address "$TEST_ACC_E_ADDR" | sed 's/^0x//')
is_bl_sel=$(selector "isBlacklisted(address)")
is_bl=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_bl_sel}${target_padded}")")

if [[ "$is_bl" != "1" ]]; then
  _assert_fail "TEST_ACC_E should be blacklisted (run e-01 first)"
  test_result
  exit 1
fi

# TEST_ACC_E를 FeePayer로 지정한 FeeDelegateDynamicFeeTx 전송 → 거부
HELPER="${CHAINBENCH_DIR}/tests/regression/lib/fee_delegate.py"

result=$(python3 "$HELPER" send \
  --rpc http://127.0.0.1:8501 \
  --sender-pk "$TEST_ACC_A_PK" \
  --fee-payer-pk "$TEST_ACC_E_PK" \
  --to "$TEST_ACC_B_ADDR" \
  --value 1 \
  --gas 21000 2>&1)

err_msg=$(printf '%s' "$result" | python3 -c "
import sys, json
try:
    d = json.loads(sys.stdin.read())
    print(d.get('rpcResponse', {}).get('error', {}).get('message', ''))
except Exception:
    print('')
")

assert_contains "$err_msg" "blacklist" "FeePayer blacklist error"

test_result
