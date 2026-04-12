#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-1-01
# name: eth_getBlockByNumber(latest)
# category: regression/g-api
# tags: [rpc, block]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# RT-G-1-01 — eth_getBlockByNumber(latest)
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g1-01-get-block-by-number"

resp=$(rpc 1 eth_getBlockByNumber '["latest", false]')
num=$(printf '%s' "$resp" | json_get - "result.number")
hash=$(printf '%s' "$resp" | json_get - "result.hash")

assert_contains "$num" "0x" "number field is hex"
assert_contains "$hash" "0x" "hash field is hex"

# transactions field 존재
has_tx=$(printf '%s' "$resp" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
print('yes' if 'transactions' in r else 'no')
")
assert_eq "$has_tx" "yes" "block has transactions field"

test_result
