#!/usr/bin/env bash
# Test: regression/b-wbft/b-01-block-period
# RT-B-01 — 블록 생산 주기 1초 간격
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-01-block-period"

current=$(block_number "1")
wait_for_block "1" $(( current + 5 )) 15 >/dev/null

# 10개 연속 블록의 timestamp 수집
declare -a timestamps=()
for i in 0 1 2 3 4 5 6 7 8 9; do
  n=$(( current + i ))
  ts_hex=$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$n")\", false]" | json_get - "result.timestamp")
  timestamps+=("$(hex_to_dec "$ts_hex")")
done

# 인접 차이 모두 1초인지
all_ok=true
for i in 1 2 3 4 5 6 7 8 9; do
  diff=$(( timestamps[i] - timestamps[i-1] ))
  if (( diff != 1 )); then
    all_ok=false
    printf '[FAIL] block %d - %d: diff=%d\n' "$((current + i - 1))" "$((current + i))" "$diff" >&2
  fi
done

assert_true "$all_ok" "all 10 consecutive blocks have 1s period"

test_result
