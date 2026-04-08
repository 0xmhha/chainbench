#!/usr/bin/env bash
# RT-G-4-04 — admin_peers
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g4-04-admin-peers"

resp=$(rpc 1 admin_peers '[]')
count=$(printf '%s' "$resp" | python3 -c "
import sys, json
print(len(json.load(sys.stdin).get('result', [])))
")
assert_ge "$count" "1" "at least 1 peer"

# 각 peer에 id, network, protocols 필드
first_has=$(printf '%s' "$resp" | python3 -c "
import sys, json
peers = json.load(sys.stdin).get('result', [])
if peers:
    p = peers[0]
    print('yes' if p.get('id') else 'no')
else:
    print('no')
")
assert_eq "$first_has" "yes" "peer has id field"

test_result
