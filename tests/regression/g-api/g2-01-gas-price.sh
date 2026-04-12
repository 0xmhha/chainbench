#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-2-01
# name: eth_gasPrice == baseFee + GasTip
# category: regression/g-api
# tags: [rpc, gas]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# RT-G-2-01 — eth_gasPrice == baseFee + GasTip
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g2-01-gas-price"

gas_price=$(hex_to_dec "$(rpc 1 eth_gasPrice "[]" | json_get - result)")
base_fee=$(get_base_fee "1")
header_tip=$(get_header_gas_tip "1")

expected=$(( base_fee + header_tip ))
assert_eq "$gas_price" "$expected" "eth_gasPrice == baseFee($base_fee) + GasTip($header_tip)"

test_result
