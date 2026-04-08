#!/usr/bin/env bash
# RT-G-3-02 — istanbul_getValidators
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g3-02-get-validators"

vals=$(rpc 1 istanbul_getValidators '["latest"]' | python3 -c "
import sys, json
print(','.join(json.load(sys.stdin).get('result', [])))
")
assert_contains "$vals" "0x" "validators returned"

count=$(echo "$vals" | python3 -c "import sys; print(len(sys.stdin.read().strip().split(',')))")
assert_ge "$count" "4" "at least 4 validators"

test_result
