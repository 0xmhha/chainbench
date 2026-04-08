#!/usr/bin/env bash
# RT-G-3-04 — istanbul_getWbftExtraInfo
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g3-04-get-wbft-extra"

resp=$(rpc 1 istanbul_getWbftExtraInfo '["latest"]')
for field in gasTip committedSeal preparedSeal; do
  has=$(printf '%s' "$resp" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
print('yes' if '${field}' in r else 'no')
")
  assert_eq "$has" "yes" "WBFTExtra has $field field"
done

test_result
