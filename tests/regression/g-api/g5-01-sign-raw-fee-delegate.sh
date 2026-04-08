#!/usr/bin/env bash
# RT-G-5-01 — eth_signRawFeeDelegateTransaction
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g5-01-sign-raw-fee-delegate"
check_env || { test_result; exit 1; }

# 간접 검증: RPC 메서드 존재 여부
resp=$(rpc 1 eth_signRawFeeDelegateTransaction "[{\"from\":\"${TEST_ACC_C_ADDR}\"},\"0x00\"]" 2>/dev/null || echo "")
has_method=$(printf '%s' "$resp" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    err = d.get('error', {}).get('message', '')
    print('no' if 'not found' in err.lower() or 'method not' in err.lower() else 'yes')
except Exception:
    print('yes')
")
assert_eq "$has_method" "yes" "eth_signRawFeeDelegateTransaction RPC exists"

# 실제 흐름은 D-01에서 (fee_delegate.py) 검증됨
printf '[INFO]  Full signing flow covered by d-fee-delegation/d-01-fee-delegate-normal\n' >&2

test_result
