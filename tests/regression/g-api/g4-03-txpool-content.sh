#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-4-03
# name: txpool_content: pending/queued 분리 내용 확인
# category: regression/g-api
# tags: [rpc]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# RT-G-4-03 (v2) — txpool_content: pending/queued 분리 내용 확인
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g4-03-txpool-content"

content=$(rpc 1 txpool_content '[]')
has_pending=$(printf '%s' "$content" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
print('yes' if 'pending' in r else 'no')
")
has_queued=$(printf '%s' "$content" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
print('yes' if 'queued' in r else 'no')
")

assert_eq "$has_pending" "yes" "content has pending"
assert_eq "$has_queued" "yes" "content has queued"

test_result
