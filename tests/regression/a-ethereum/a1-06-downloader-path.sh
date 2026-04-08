#!/usr/bin/env bash
# Test: regression/a-ethereum/a1-06-downloader-path
# RT-A-1-06 (v2 신규) — Downloader 경로: peer 재연결 시 큰 gap(≥ 2) 따라잡기
#
# 전략:
#   1. EN5의 모든 peer를 admin_removePeer로 제거 (노드는 계속 실행)
#   2. 충분히 대기해서 gap ≥ 2 생성 (실제로는 ≥ 10 블록)
#   3. 단일 BP peer를 admin_addPeer로 재연결
#   4. ForceSyncCycle(10s) 경과 후 chainSyncer.loop가 Downloader 트리거
#   5. EN5가 BP head까지 수렴하는지 확인
#
# 근거 코드:
#   - eth/sync.go:214 (Anzeon): peer.TD > ourTD + tdAdjustment → Downloader
#   - eth/sync.go:33: defaultMinSyncPeers=5, 5 노드 환경(peer 4)은 forced 경로
#   - eth/ethconfig/config.go:61-62: ForceSyncCycle=10s, TdSyncInterval=10s
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a1-06-downloader-path"

# ---- Phase 0: 사전 확인 ----
en_initial=$(block_number "5" 2>/dev/null || echo "0")
bp_initial=$(block_number "1")
initial_diff=$(( bp_initial - en_initial ))
(( initial_diff < 0 )) && initial_diff=$((-initial_diff))
assert_true "$( [[ $initial_diff -le 2 ]] && echo true || echo false )" "EN5 is initially in sync with BP1 (diff=$initial_diff)"

# ---- Phase 1: EN5의 모든 peer를 제거 ----
printf '[INFO]  Phase 1: removing all peers from EN5 via admin_removePeer\n' >&2
peer_enodes=$(rpc "5" "admin_peers" "[]" | python3 -c "
import sys, json
peers = json.load(sys.stdin).get('result', [])
for p in peers:
    # enode URL 구성: enode://<id>@<ip>:<port>
    info = p.get('network', {})
    remote = info.get('remoteAddress', '')
    if not remote:
        continue
    pid = p.get('id', '')
    if not pid:
        continue
    print(f'enode://{pid}@{remote}')
")

if [[ -z "$peer_enodes" ]]; then
  _assert_fail "could not retrieve peer enodes from EN5"
  test_result
  exit 1
fi

# 모든 peer 제거
removed_count=0
while IFS= read -r enode; do
  [[ -z "$enode" ]] && continue
  result=$(admin_remove_peer "5" "$enode" 2>/dev/null || echo "")
  printf '[INFO]  admin_removePeer(%s...) → %s\n' "${enode:0:40}" "$result" >&2
  removed_count=$(( removed_count + 1 ))
done <<< "$peer_enodes"

assert_ge "$removed_count" "1" "at least 1 peer removed from EN5"

# peer 제거 반영 대기
sleep 2

# peer 0 확인
peer_count_after_remove=$(peer_count "5")
printf '[INFO]  EN5 peer_count after remove = %s\n' "$peer_count_after_remove" >&2
# static peer는 재접속 시도할 수 있으므로 0이 아닐 수 있지만, 급격히 감소했어야 함
assert_true "$( [[ $peer_count_after_remove -lt 4 ]] && echo true || echo false )" "EN5 peer count decreased (now $peer_count_after_remove)"

# ---- Phase 2: gap 생성 (BP는 계속 생산, EN5는 peer 없어 수신 불가) ----
printf '[INFO]  Phase 2: waiting 15s for BP to produce blocks while EN5 is isolated\n' >&2
sleep 15

bp_after=$(block_number "1")
en_after=$(block_number "5" 2>/dev/null || echo "$en_initial")
gap=$(( bp_after - en_after ))
printf '[INFO]  after wait: BP1=%s EN5=%s gap=%s\n' "$bp_after" "$en_after" "$gap" >&2

# gap이 충분히 커야 Downloader 경로로 확실히 진입
# 단, chainbench static peer dialer가 자동 재연결했을 수도 있음 — 그 경우 gap이 작을 수 있음
if (( gap < 5 )); then
  printf '[WARN]  gap (%s) small — chainbench staticDialer may have reconnected. Test still valid if gap >= 2\n' "$gap" >&2
fi
assert_ge "$gap" "2" "block gap >= 2 for Downloader trigger (tdAdjustment=1)"

# ---- Phase 3: 단일 BP peer로 재연결 ----
# BP1의 enode 조회
bp1_enode=$(admin_enode "1")
assert_not_empty "$bp1_enode" "BP1 enode URL retrieved"
printf '[INFO]  Phase 3: reconnecting EN5 to BP1 via admin_addPeer\n' >&2

add_result=$(admin_add_peer "5" "$bp1_enode")
printf '[INFO]  admin_addPeer result = %s\n' "$add_result" >&2

# ---- Phase 4: Downloader로 동기화 완료 대기 ----
# ForceSyncCycle(10s) + 실제 다운로드 시간 고려하여 최대 90초 대기
printf '[INFO]  Phase 4: waiting for Downloader to catch up (max 90s)\n' >&2
sync_success=false
for i in $(seq 1 90); do
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

# ---- Verify: 중간 샘플 블록 해시 일치 ----
# gap 중간 지점의 블록이 EN5에 올바르게 삽입되었는지
sample_block=$(( en_after + gap / 2 ))
bp_hash=$(rpc "1" "eth_getBlockByNumber" "[\"$(dec_to_hex "$sample_block")\", false]" | json_get - "result.hash")
en_hash=$(rpc "5" "eth_getBlockByNumber" "[\"$(dec_to_hex "$sample_block")\", false]" | json_get - "result.hash")
assert_eq "$en_hash" "$bp_hash" "Downloader inserted block $sample_block with correct hash"

test_result
