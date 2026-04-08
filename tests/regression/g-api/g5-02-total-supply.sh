#!/usr/bin/env bash
# RT-G-5-02 — eth_call NativeCoinAdapter.totalSupply
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g5-02-total-supply"

sel=$(selector "totalSupply()")
result=$(eth_call_raw 1 "$NATIVE_COIN_ADAPTER" "$sel")
total=$(hex_to_dec "$result")
assert_gt "$total" "0" "totalSupply > 0"
printf '[INFO]  totalSupply = %s wei\n' "$total" >&2

test_result
