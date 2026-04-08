#!/usr/bin/env bash
# Test: regression/b-wbft/b-10-post-round-change
# RT-B-10 — 라운드 체인지 후 블록 정상 연결 (parentHash 체인 검증)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-10-post-round-change"

rc_block=$(cat /tmp/chainbench-regression/round_change_block.txt 2>/dev/null || echo "")
if [[ -z "$rc_block" ]]; then
  printf '[WARN]  b-09 not run previously, using latest block\n' >&2
  rc_block=$(block_number "1")
fi

# rc_block ~ rc_block+5 까지 parentHash 체인 검증
prev_hash=""
for i in 0 1 2 3 4 5; do
  n=$(( rc_block + i ))
  wait_for_block "1" "$n" 10 >/dev/null
  blk=$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$n")\", false]")
  h=$(printf '%s' "$blk" | json_get - "result.hash")
  ph=$(printf '%s' "$blk" | json_get - "result.parentHash")

  if [[ -n "$prev_hash" ]]; then
    assert_eq "$ph" "$prev_hash" "block $n parentHash matches block $((n-1)) hash"
  fi
  prev_hash="$h"
done

# round=0 복귀 확인 (RT-B-09 이후 일반 블록들)
last_block=$(( rc_block + 5 ))
extra=$(rpc "1" "istanbul_getWbftExtraInfo" "[\"$(dec_to_hex "$last_block")\"]")
round=$(printf '%s' "$extra" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
print(int(r.get('round', '0x0'), 16))
")
assert_eq "$round" "0" "later blocks return to round=0"

test_result
