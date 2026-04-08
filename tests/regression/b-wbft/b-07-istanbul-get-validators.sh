#!/usr/bin/env bash
# Test: regression/b-wbft/b-07-istanbul-get-validators
# RT-B-07 — istanbul_getValidators RPC (wbft_* 아님)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-07-istanbul-get-validators"

# istanbul_getValidators 호출
resp=$(rpc "1" "istanbul_getValidators" '["latest"]')
result=$(printf '%s' "$resp" | python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'error' in data:
    print('ERR')
else:
    r = data.get('result', [])
    print(','.join(r).lower())
")

assert_contains "$result" "0x" "istanbul_getValidators returned address array"

# 4개 밸리데이터 확인
count=$(python3 -c "print(len('$result'.split(',')))")
assert_eq "$count" "4" "4 validators"

# 각 validator 주소가 genesis의 4개 중 하나인지
for v in "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR" "$VALIDATOR_3_ADDR" "$VALIDATOR_4_ADDR"; do
  v_lower=$(echo "$v" | tr '[:upper:]' '[:lower:]')
  assert_contains "$result" "$v_lower" "genesis validator $v_lower present"
done

# wbft_getValidators는 존재하지 않음 (v2에서 istanbul_*로 정정)
wbft_resp=$(rpc "1" "wbft_getValidators" '["latest"]' 2>/dev/null || echo "")
wbft_err=$(printf '%s' "$wbft_resp" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    print('yes' if 'error' in d else 'no')
except Exception:
    print('yes')
" 2>/dev/null || echo "yes")
assert_eq "$wbft_err" "yes" "wbft_getValidators does not exist (only istanbul_getValidators)"

test_result
