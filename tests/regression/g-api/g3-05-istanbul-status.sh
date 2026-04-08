#!/usr/bin/env bash
# RT-G-3-05 — istanbul_status
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g3-05-istanbul-status"

current=$(block_number "1")
start=$(( current - 5 ))
(( start < 1 )) && start=1

resp=$(rpc 1 istanbul_status "[\"$(dec_to_hex "$start")\", \"$(dec_to_hex "$current")\"]")
for field in sealerActivity authorCounts blockRange roundStats; do
  has=$(printf '%s' "$resp" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
keys = [k.lower() for k in r.keys()]
print('yes' if '${field}'.lower() in keys else 'no')
")
  [[ "$has" == "yes" ]] && _assert_pass "status has $field field" || _assert_fail "status missing $field field"
done

test_result
