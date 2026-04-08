#!/usr/bin/env bash
# Test: regression/c-anzeon/c-06-min-basefee
# RT-C-06 — baseFee가 MinBaseFee(20 Gwei) 아래로 내려가지 않음
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/c-anzeon/c-06-min-basefee"

# 10 블록 관찰
current=$(block_number "1")
wait_for_block "1" $(( current + 10 )) 30 >/dev/null

below_min=false
MIN_BASE_FEE=20000000000000  # 20 Gwei

for i in 0 1 2 3 4 5 6 7 8 9; do
  n=$(( current + i ))
  bf=$(hex_to_dec "$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$n")\", false]" | json_get - 'result.baseFeePerGas')")
  if (( bf < MIN_BASE_FEE )); then
    below_min=true
    printf '[FAIL]  block %s baseFee=%s < MinBaseFee\n' "$n" "$bf" >&2
  fi
done

assert_true "$( ! $below_min && echo true || echo false )" "all blocks have baseFee >= MinBaseFee (20 Gwei)"

test_result
