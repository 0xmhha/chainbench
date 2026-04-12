#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-G-2-04
# name: eth_estimateGas (NativeCoinAdapter.transfer)
# category: regression/g-api
# tags: [rpc]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# RT-G-2-04 — eth_estimateGas (NativeCoinAdapter.transfer)
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g2-04-estimate-system-call"

sel=$(selector "transfer(address,uint256)")
to_padded=$(pad_address "$TEST_ACC_B_ADDR" | sed 's/^0x//')
amount_padded=$(pad_uint256 "1000" | sed 's/^0x//')
data="${sel}${to_padded}${amount_padded}"

estimate=$(hex_to_dec "$(rpc 1 eth_estimateGas "[{\"from\":\"${TEST_ACC_A_ADDR}\",\"to\":\"${NATIVE_COIN_ADAPTER}\",\"data\":\"${data}\"}]" | json_get - result)")
assert_gt "$estimate" "21000" "estimate > 21000 for NativeCoinAdapter.transfer"
assert_gt "$estimate" "0" "estimate != 0"

test_result
