#!/usr/bin/env bash
# RT-G-3-06 — istanbul_isValidator
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g3-06-is-validator"

# node1~4 = BP validator, node5 = EN
for node in 1 2 3 4; do
  is_val=$(rpc "$node" istanbul_isValidator '["latest"]' | json_get - result)
  assert_eq "$is_val" "true" "node${node} is validator"
done

is_val5=$(rpc 5 istanbul_isValidator '["latest"]' | json_get - result)
assert_eq "$is_val5" "false" "node5 (EN) is NOT validator"

test_result
