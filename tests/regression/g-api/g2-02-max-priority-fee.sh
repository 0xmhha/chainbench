#!/usr/bin/env bash
# RT-G-2-02 — eth_maxPriorityFeePerGas == WBFTExtra.GasTip
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g2-02-max-priority-fee"

mpfpg=$(hex_to_dec "$(rpc 1 eth_maxPriorityFeePerGas "[]" | json_get - result)")
header_tip=$(get_header_gas_tip "1")

assert_eq "$mpfpg" "$header_tip" "eth_maxPriorityFeePerGas == WBFTExtra.GasTip (backend Anzeon branch)"

test_result
