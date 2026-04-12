#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-3-04
# name: istanbul_getWbftExtraInfo
# category: regression/g-api
# tags: [rpc]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# RT-G-3-04 — istanbul_getWbftExtraInfo
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g3-04-get-wbft-extra"

resp=$(get_wbft_extra_json "1")
for field in gasTip committedSeal preparedSeal; do
  has=$(printf '%s' "$resp" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
print('yes' if '${field}' in r else 'no')
")
  assert_eq "$has" "yes" "WBFTExtra has $field field"
done

test_result
