#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-1-06
# name: eth_getCode on NativeCoinAdapter (0x1000)
# category: regression/g-api
# tags: [rpc]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# RT-G-1-06 — eth_getCode on NativeCoinAdapter (0x1000)
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g1-06-get-code-system"

code=$(rpc 1 eth_getCode "[\"${NATIVE_COIN_ADAPTER}\", \"latest\"]" | json_get - result)
assert_contains "$code" "0x" "code is hex"
code_len=${#code}
assert_gt "$code_len" "10" "NativeCoinAdapter has non-empty bytecode (len=$code_len)"

test_result
