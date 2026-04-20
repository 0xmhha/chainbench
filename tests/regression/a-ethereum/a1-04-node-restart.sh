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
# BFT 합의는 노드 재시작 후 round-change로 인해 1-2블록 동안 2초 간격이 발생할 수 있으므로
# 충분히 기다린 후 안정화된 블록 구간에서 검증한다.
sleep 5
after_block=$(block_number "1")
assert_gt "$after_block" "$before_block" "blocks continue after restart"

# 안정화 이후 5개 연속 블록 타임스탬프 차이 확인
# 몇 블록 더 기다려서 round-change 복구 구간을 지난 블록부터 측정
start_block=$(( after_block + 3 ))
wait_for_block "1" "$(( start_block + 4 ))" 15 >/dev/null

declare -a timestamps=()
for i in 0 1 2 3 4; do
  n=$(( start_block + i ))
  ts=$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$n")\", false]" | json_get - "result.timestamp")
  timestamps+=("$(hex_to_dec "$ts")")
done

# 인접 차이가 1초인지 검증 (BFT round-change로 인해 최대 1개 블록이 2초일 수 있음)
outlier_count=0
for i in 1 2 3 4; do
  diff=$(( timestamps[i] - timestamps[i-1] ))
  printf '[INFO]  block diff[%d]: %d seconds\n' "$i" "$diff" >&2
  if (( diff < 1 || diff > 2 )); then
    # diff가 0이거나 3초 이상이면 확실히 비정상
    outlier_count=$(( outlier_count + 99 ))
  elif (( diff != 1 )); then
    outlier_count=$(( outlier_count + 1 ))
  fi
done
# 최대 1개 블록까지 2초 허용 (나머지는 1초여야 함)
assert_true "$( [[ $outlier_count -le 1 ]] && echo true || echo false )" "block period ~1 second for 5 consecutive blocks (outliers=$outlier_count)"

test_result
