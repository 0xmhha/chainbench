#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-2-03
# name: eth_feeHistory
# category: regression/g-api
# tags: [rpc, gas]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# RT-G-2-03 — eth_feeHistory
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g2-03-fee-history"

resp=$(rpc 1 eth_feeHistory '["0x10", "latest", []]')
all_gte_min=$(printf '%s' "$resp" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
base_fees = r.get('baseFeePerGas', [])
MIN = 20_000_000_000_000
ok = all(int(b, 16) >= MIN for b in base_fees)
print('yes' if ok and base_fees else 'no')
")
assert_eq "$all_gte_min" "yes" "all baseFeePerGas entries >= MinBaseFee"

test_result
