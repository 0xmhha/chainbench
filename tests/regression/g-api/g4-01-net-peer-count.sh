#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-4-01
# name: net_peerCount
# category: regression/g-api
# tags: [rpc, p2p]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# RT-G-4-01 — net_peerCount
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g4-01-net-peer-count"

for node in 1 2 3 4 5; do
  count=$(peer_count "$node")
  assert_ge "$count" "1" "node${node} has >=1 peer"
done

test_result
