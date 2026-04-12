#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-3-01
# name: istanbul_nodeAddress
# category: regression/g-api
# tags: [rpc]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# RT-G-3-01 — istanbul_nodeAddress
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g3-01-node-address"

for node in 1 2 3 4; do
  addr=$(rpc "$node" istanbul_nodeAddress '[]' | json_get - result)
  assert_contains "$addr" "0x" "node${node} istanbul_nodeAddress returns addr: $addr"
done

test_result
