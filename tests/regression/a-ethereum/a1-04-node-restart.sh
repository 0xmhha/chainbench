#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-1-04
# name: 노드 재시작 후 블록 생산 재개 (1초 주기)
# category: regression/a-ethereum
# tags: [sync, restart]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/a-ethereum/a1-04-node-restart
# RT-A-1-04 — 노드 재시작 후 블록 생산 재개 (1초 주기)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a1-04-node-restart"

# 현재 블록과 시간 기록
before_block=$(block_number "1")
before_ts=$(rpc "1" "eth_getBlockByNumber" "[\"latest\", false]" | json_get - "result.timestamp")
before_ts_dec=$(hex_to_dec "$before_ts")
printf '[INFO]  Before restart: block=%s, ts=%s\n' "$before_block" "$before_ts_dec" >&2

# node2를 stop/start
"${CHAINBENCH_DIR}/chainbench.sh" node stop 2 --quiet 2>/dev/null || true
sleep 2
"${CHAINBENCH_DIR}/chainbench.sh" node start 2 --quiet 2>/dev/null || true

# 재시작 후 블록이 계속 생산되는지 확인 (1초 간격)
sleep 3
after_block=$(block_number "1")
assert_gt "$after_block" "$before_block" "blocks continue after restart"

# 5개 연속 블록 타임스탬프 차이가 모두 1초인지 확인
declare -a timestamps=()
for i in 0 1 2 3 4; do
  n=$(( after_block + i ))
  wait_for_block "1" "$n" 10 >/dev/null
  ts=$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$n")\", false]" | json_get - "result.timestamp")
  timestamps+=("$(hex_to_dec "$ts")")
done

# 인접 차이가 1초인지 검증
all_one_sec=true
for i in 1 2 3 4; do
  diff=$(( timestamps[i] - timestamps[i-1] ))
  printf '[INFO]  block diff[%d]: %d seconds\n' "$i" "$diff" >&2
  (( diff != 1 )) && all_one_sec=false
done
assert_true "$all_one_sec" "block period is 1 second for 5 consecutive blocks"

test_result
