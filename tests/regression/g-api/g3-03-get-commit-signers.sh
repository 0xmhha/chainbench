#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-3-03
# name: istanbul_getCommitSignersFromBlock
# category: regression/g-api
# tags: [rpc]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# RT-G-3-03 — istanbul_getCommitSignersFromBlock
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g3-03-get-commit-signers"

current=$(block_number "1")
resp=$(rpc 1 istanbul_getCommitSignersFromBlock "[\"$(dec_to_hex "$current")\"]")

author=$(printf '%s' "$resp" | json_get - "result.Author")
assert_contains "$author" "0x" "Author address returned"

committer_count=$(printf '%s' "$resp" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
c = r.get('Committers', []) or r.get('committers', [])
print(len(c))
")
assert_ge "$committer_count" "3" "committers >= quorum (3)"

test_result
