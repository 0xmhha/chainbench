#!/usr/bin/env bash
# Test: regression/a-ethereum/a1-06-downloader-path
# RT-A-1-06 (v2 신규) — Downloader 경로: 큰 gap(≥ 10) 생성 후 동기화
#
# 전략:
#   1. EN5를 stop (admin_removePeer는 chainbench static peer가 즉시 재연결하므로 무효)
#   2. 충분히 대기해서 gap ≥ 10 블록 생성 → Block Fetcher(<=2블록)가 아닌 Downloader 경로
#   3. EN5 restart
#   4. Downloader가 헤더·바디를 순차 다운로드하여 catchup 확인
#
# 근거 코드:
#   - eth/sync.go: Downloader는 큰 gap에서, Block Fetcher는 near-head에서 동작
#   - 큰 gap(10+) → 반드시 Downloader 경로
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a1-06-downloader-path"

# ---- Phase 0: 사전 확인 ----
en_initial=$(block_number "5" 2>/dev/null || echo "0")
bp_initial=$(block_number "1")
initial_diff=$(( bp_initial - en_initial ))
(( initial_diff < 0 )) && initial_diff=$((-initial_diff))
assert_true "$( [[ $initial_diff -le 2 ]] && echo true || echo false )" "EN5 is initially in sync with BP1 (diff=$initial_diff)"

# ---- Phase 1: EN5 stop ----
printf '[INFO]  Phase 1: stopping EN5 to create large gap\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node stop 5 --quiet 2>/dev/null || true
sleep 1

en_stop_block=$(block_number "1")
printf '[INFO]  BP1 at block %s when EN5 stopped\n' "$en_stop_block" >&2

# ---- Phase 2: gap 생성 (최소 10 블록) ----
# 1초 블록 주기 기준 15초 → 약 15 블록 gap
printf '[INFO]  Phase 2: waiting 15s for BP to produce blocks\n' >&2
sleep 15

bp_before_restart=$(block_number "1")
gap=$(( bp_before_restart - en_stop_block ))
printf '[INFO]  gap created: %s blocks (BP1 now at %s)\n' "$gap" "$bp_before_restart" >&2
assert_ge "$gap" "10" "large gap (>= 10 blocks) created for Downloader path"

# ---- Phase 3: EN5 restart ----
printf '[INFO]  Phase 3: restarting EN5\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node start 5 --quiet 2>/dev/null || true

# ---- Phase 4: Downloader로 동기화 완료 대기 ----
printf '[INFO]  Phase 4: waiting for Downloader catchup (max 60s)\n' >&2
sync_success=false
for i in $(seq 1 60); do
  en_block=$(block_number "5" 2>/dev/null || echo "0")
  bp_block=$(block_number "1")
  diff=$(( bp_block - en_block ))
  (( diff < 0 )) && diff=$((-diff))
  if (( diff <= 2 )); then
    sync_success=true
    printf '[INFO]  synced via Downloader at t=%ss: BP=%s EN=%s diff=%s\n' "$i" "$bp_block" "$en_block" "$diff" >&2
    break
  fi
  sleep 1
done
assert_true "$sync_success" "EN5 catchup via Downloader (final diff <= 2)"

# ---- Verify: gap 중간의 블록 해시 일치 ----
# Downloader가 중간 블록을 올바르게 삽입했는지 검증
sample_block=$(( en_stop_block + gap / 2 ))
bp_hash=$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$sample_block")\", false]" | json_get - "result.hash")
en_hash=$(rpc "5" "eth_getBlockByNumber" "[\"$(dec_to_hex "$sample_block")\", false]" | json_get - "result.hash")
assert_eq "$en_hash" "$bp_hash" "Downloader inserted block $sample_block with correct hash"

test_result
