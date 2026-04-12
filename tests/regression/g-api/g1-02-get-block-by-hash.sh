#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-1-02
# name: eth_getBlockByHash
# category: regression/g-api
# tags: [rpc, block]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# RT-G-1-02 — eth_getBlockByHash
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g1-02-get-block-by-hash"

latest_hash=$(rpc 1 eth_getBlockByNumber '["latest", false]' | json_get - "result.hash")
latest_num=$(rpc 1 eth_getBlockByNumber '["latest", false]' | json_get - "result.number")

resp=$(rpc 1 eth_getBlockByHash "[\"${latest_hash}\", false]")
num=$(printf '%s' "$resp" | json_get - "result.number")

assert_eq "$num" "$latest_num" "block number matches between hash and number lookup"

test_result
