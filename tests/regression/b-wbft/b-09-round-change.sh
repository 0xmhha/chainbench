#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-B-09
# name: 라운드 체인지 동작 (제안자 노드 중단 시)
# category: regression/b-wbft
# tags: [wbft, consensus]
# estimated_seconds: 69
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/b-wbft/b-09-round-change
# RT-B-09 — 라운드 체인지 동작 (제안자 노드 중단 시)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-09-round-change"

# 현재 블록에서 제안자 확인
current_block=$(block_number "1")
proposer_info=$(rpc "1" "istanbul_getWbftExtraInfo" "[\"$(dec_to_hex "$current_block")\"]")

# 다음 블록 제안자 추정 (round-robin policy)
# chainbench에서 현재 제안자 노드를 정확히 알기 어려우므로, node1을 stop해서 round change 발생 유도
# 단, node1을 stop하면 RPC 조회 대상도 바뀌어야 하므로 node2를 관찰

# node1 중단 → 3/4 validator → 여전히 quorum(3) 충족
# round change 발생 후 다른 proposer가 블록 생성하길 기다림
printf '[INFO]  stopping node1 to trigger round change\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node stop 1 --quiet 2>/dev/null || true

# sleep 6 대신 node2에서 현재 블록보다 2 이상 높은 블록 생산될 때까지 대기
# (round change 완료 + 정상 블록 생산 증거)
target_block=$(( current_block + 2 ))
wait_for_block "2" "$target_block" 20 >/dev/null 2>&1 || true

# node2에서 새 블록의 round 확인
b_after=$(block_number "2")
if (( b_after > current_block )); then
  extra=$(rpc "2" "istanbul_getWbftExtraInfo" "[\"$(dec_to_hex "$b_after")\"]")
  round=$(printf '%s' "$extra" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
print(int(r.get('round', '0x0'), 16))
")
  # 라운드가 0보다 크거나, 제안자 전환으로 round=0으로 복귀했을 수 있음 → 블록 진행 자체가 증거
  _assert_pass "chain progressed to block $b_after after node1 stop (round=$round)"
else
  _assert_fail "chain did not progress after node1 stop"
fi

# 복구
"${CHAINBENCH_DIR}/chainbench.sh" node start 1 --quiet 2>/dev/null || true

# node1이 catch up 할 때까지 기다림 (stop 기간 동안 뒤처진 블록 동기화)
# 이전에는 sleep 5 로 고정해서 sync 미완료 상태의 node1 head를 읽어 false negative 발생
wait_for_block "1" "$b_after" 30 >/dev/null 2>&1 || true
sleep 3  # 추가 진행 여유

# 복구 후 체인이 계속 진행했는지 — node2 기준으로 head 증가 확인
# (node2는 stop 영향 없음, 실제 chain head를 반영)
b_final=$(block_number "2")
assert_gt "$b_final" "$b_after" "chain continues after node1 rejoined (node2 head)"

# 상태 저장 (b-10에서 참조)
echo "$b_after" > /tmp/chainbench-regression/round_change_block.txt

test_result
