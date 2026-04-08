#!/usr/bin/env bash
# RT-G-2-01 — eth_gasPrice == baseFee + GasTip
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g2-01-gas-price"

gas_price=$(hex_to_dec "$(rpc 1 eth_gasPrice "[]" | json_get - result)")
base_fee=$(get_base_fee "1")
header_tip=$(rpc 1 istanbul_getWbftExtraInfo '["latest"]' | python3 -c "
import sys, json
print(int(json.load(sys.stdin).get('result', {}).get('gasTip', '0x0'), 16))
")

expected=$(( base_fee + header_tip ))
assert_eq "$gas_price" "$expected" "eth_gasPrice == baseFee($base_fee) + GasTip($header_tip)"

test_result
