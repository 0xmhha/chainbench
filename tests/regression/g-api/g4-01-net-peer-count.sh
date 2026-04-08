#!/usr/bin/env bash
# RT-G-4-01 — net_peerCount
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g4-01-net-peer-count"

for node in 1 2 3 4 5; do
  count=$(peer_count "$node")
  assert_ge "$count" "1" "node${node} has >=1 peer"
done

test_result
