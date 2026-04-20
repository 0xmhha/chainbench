#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-B-10
# name: 라운드 체인지 후 블록 정상 연결 (parentHash 체인 검증)
# category: regression/b-wbft
# tags: [wbft, consensus]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
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
# 라운드 체인지 이후 proposer rotation이 완료되기까지 수 블록 소요될 수 있으므로
# rc_block+5 ~ rc_block+15 범위에서 round=0인 블록을 찾는다.
found_round_zero=false
for offset in $(seq 5 15); do
  check_block=$(( rc_block + offset ))
  wait_for_block "1" "$check_block" 10 >/dev/null
  extra=$(rpc "1" "istanbul_getWbftExtraInfo" "[\"$(dec_to_hex "$check_block")\"]")
  round=$(printf '%s' "$extra" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
print(int(r.get('round', '0x0'), 16))
")
  if [[ "$round" == "0" ]]; then
    found_round_zero=true
    printf '[INFO]  block %s has round=0 (recovery confirmed)\n' "$check_block" >&2
    break
  fi
  printf '[INFO]  block %s still at round=%s\n' "$check_block" "$round" >&2
done
assert_true "$found_round_zero" "later blocks return to round=0"

test_result
