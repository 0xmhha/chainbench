#!/usr/bin/env bash
# Test: regression/a-ethereum/a1-07-block-fetcher-path
# RT-A-1-07 (v2 신규) — Block Fetcher 경로: 작은 gap(= 1) 또는 정상 운영 중 NewBlock 전파
#
# 두 서브 시나리오:
#   Part 1 — 정상 운영 중 NewBlock 전파 (Downloader 개입 없이 Fetcher 경로 확인)
#   Part 2 — peer 제거 → 매우 짧게 대기 (gap = 1) → 재연결 → Fetcher 경로로 단일 블록 수신
#
# 근거 코드:
#   - eth/sync.go:214 (Anzeon): peer.TD <= ourTD + 1이면 nextSyncOp == nil → Downloader 미동작
#   - eth/fetcher/block_fetcher.go: NewBlock/NewBlockHashes 메시지 수신 시 단일 블록 import
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a1-07-block-fetcher-path"

# ============================================================================
# Part 1: 정상 운영 중 NewBlock 전파 (Fetcher 경로)
# ============================================================================

# 사전 동기화 확인
sync_info=$(check_sync)
diff=$(printf '%s' "$sync_info" | python3 -c "
import sys, json
print(json.load(sys.stdin).get('diff', 999))
")
assert_true "$( [[ $diff -le 2 ]] && echo true || echo false )" "Part 1 start: all nodes in sync (diff=$diff)"

base_block=$(block_number "1")
printf '[INFO]  Part 1: observing 5 blocks for Fetcher delivery\n' >&2

# 5개 블록 생산 동안 lag 관찰
max_lag=0
for i in 1 2 3 4 5; do
  next=$(( base_block + i ))
  wait_for_block "1" "$next" 5 >/dev/null
  sleep 1.5  # Fetcher가 전파할 시간
  bp_head=$(block_number "1")
  en_head=$(block_number "5")
  lag=$(( bp_head - en_head ))
  (( lag < 0 )) && lag=$((-lag))
  (( lag > max_lag )) && max_lag=$lag
  printf '[INFO]  Part1 iter %s: BP=%s EN=%s lag=%s\n' "$i" "$bp_head" "$en_head" "$lag" >&2
done

assert_true "$( [[ $max_lag -le 2 ]] && echo true || echo false )" "Part 1: max_lag <= 2 (Fetcher delivered promptly, lag=$max_lag)"

# ============================================================================
# Part 2: peer 제거 → 짧게 대기 (gap 1) → 재연결 → Fetcher 수신
# ============================================================================

printf '[INFO]  Part 2: removing EN5 peers briefly for gap=1 scenario\n' >&2

# EN5의 모든 peer 제거
peer_enodes=$(rpc "5" "admin_peers" "[]" | python3 -c "
import sys, json
peers = json.load(sys.stdin).get('result', [])
for p in peers:
    remote = p.get('network', {}).get('remoteAddress', '')
    pid = p.get('id', '')
    if pid and remote:
        print(f'enode://{pid}@{remote}')
")

declare -a removed_enodes=()
while IFS= read -r enode; do
  [[ -z "$enode" ]] && continue
  admin_remove_peer "5" "$enode" >/dev/null 2>&1 || true
  removed_enodes+=("$enode")
done <<< "$peer_enodes"

printf '[INFO]  removed %s peers from EN5\n' "${#removed_enodes[@]}" >&2

# gap=1 만들기 위한 짧은 대기 — BP가 1블록만 생산
en_before_gap=$(block_number "5" 2>/dev/null || echo "0")
bp_before_gap=$(block_number "1")

# BP가 1블록 더 생산할 때까지 대기
wait_for_block "1" $(( bp_before_gap + 1 )) 5 >/dev/null

sleep 0.5
bp_after_wait=$(block_number "1")
en_after_wait=$(block_number "5" 2>/dev/null || echo "$en_before_gap")
small_gap=$(( bp_after_wait - en_after_wait ))
printf '[INFO]  Part 2 gap: BP=%s EN=%s gap=%s\n' "$bp_after_wait" "$en_after_wait" "$small_gap" >&2

# BP1 enode로 재연결
bp1_enode=$(admin_enode "1")
admin_add_peer "5" "$bp1_enode" >/dev/null

# 재연결 후 EN5가 head에 도달하는지 확인 (Fetcher 또는 Downloader 어느 경로든 수신 확인)
# gap 1인 경우 대부분 Fetcher 경로, tdCheckTimer 경과 시 Downloader 경로 가능
sync_success=false
for i in $(seq 1 30); do
  bp_h=$(block_number "1")
  en_h=$(block_number "5" 2>/dev/null || echo "0")
  d=$(( bp_h - en_h ))
  (( d < 0 )) && d=$((-d))
  if (( d <= 2 )); then
    sync_success=true
    printf '[INFO]  Part 2 synced at t=%ss: BP=%s EN=%s (block fetcher/downloader hybrid)\n' "$i" "$bp_h" "$en_h" >&2
    break
  fi
  sleep 1
done
assert_true "$sync_success" "Part 2: EN5 catchup within 30s (Fetcher or forced Downloader)"

# 최종 블록 해시 일치 확인
final_block=$(block_number "1")
(( final_block -= 1 ))  # 안정적인 직전 블록
bp_hash=$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$final_block")\", false]" | json_get - "result.hash")
en_hash=$(rpc "5" "eth_getBlockByNumber" "[\"$(dec_to_hex "$final_block")\", false]" | json_get - "result.hash")
assert_eq "$en_hash" "$bp_hash" "Part 2: final block $final_block hash matches"

test_result
