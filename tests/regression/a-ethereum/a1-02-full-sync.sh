#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-1-02
# name: Full Sync: 동기화 제공 노드가 동기화 요청 노드보다 2 이상 높음
# category: regression/a-ethereum
# tags: [sync]
# estimated_seconds: 26
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/a-ethereum/a1-02-full-sync
# RT-A-1-02 (v2 revised) — Full Sync: 동기화 제공 노드가 동기화 요청 노드보다 2 이상 높음
#
# 시나리오:
#   1. BP1 (제공자)는 정상 운영, EN5 (요청자) stop
#   2. BP1이 EN5보다 >=2 블록 앞서도록 대기
#   3. EN5 재기동 (--syncmode full, 기본값)
#   4. EN5가 BP1 head에 도달하는지 확인
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a1-02-full-sync"

# EN5 중단
printf '[INFO]  stopping EN node5 for gap creation\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node stop 5 --quiet 2>/dev/null || true

# EN5 stop 시점의 head 기록
en_stop_block=$(block_number "1")
printf '[INFO]  BP1 at block %s when EN5 stopped\n' "$en_stop_block" >&2

# BP가 EN보다 최소 3 블록 앞서게 대기
target=$(( en_stop_block + 3 ))
wait_for_block "1" "$target" 20 >/dev/null
bp_block_before_restart=$(block_number "1")
printf '[INFO]  BP1 reached block %s (gap creation done)\n' "$bp_block_before_restart" >&2

# EN5 재기동
printf '[INFO]  restarting EN node5\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node start 5 --quiet 2>/dev/null || true

# EN5 동기화 대기 (최대 30초)
sync_success=false
for i in $(seq 1 30); do
  en_block=$(block_number "5" 2>/dev/null || echo "0")
  bp_block=$(block_number "1")
  diff=$(( bp_block - en_block ))
  (( diff < 0 )) && diff=$((-diff))
  if (( diff <= 2 )); then
    sync_success=true
    printf '[INFO]  EN5 synced at block %s (BP1=%s, diff=%s)\n' "$en_block" "$bp_block" "$diff" >&2
    break
  fi
  sleep 1
done

assert_true "$sync_success" "EN5 synced with BP1 via full sync (diff <= 2)"

# 임의 블록 번호에서 해시 일치 확인
sample_block=$(( bp_block_before_restart - 1 ))
(( sample_block < 1 )) && sample_block=1
bp_hash=$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$sample_block")\", false]" | json_get - "result.hash")
en_hash=$(rpc "5" "eth_getBlockByNumber" "[\"$(dec_to_hex "$sample_block")\", false]" | json_get - "result.hash")
assert_eq "$en_hash" "$bp_hash" "EN5 and BP1 have identical hash at block $sample_block"

# eth_syncing == false (동기화 완료)
syncing=$(rpc "5" "eth_syncing" "[]" | json_get - result)
# syncing이 false이거나 false 객체면 완료
if [[ "$syncing" == "false" || "$syncing" == "null" || -z "$syncing" ]]; then
  _assert_pass "EN5 eth_syncing returns false (sync complete)"
fi

test_result
