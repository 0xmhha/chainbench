#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-B-08
# name: 쿼럼 미달 시 블록 수락 거부
# category: regression/b-wbft
# tags: [wbft, quorum]
# estimated_seconds: 22
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/b-wbft/b-08-quorum-deficient
# RT-B-08 — 쿼럼 미달 시 블록 수락 거부
#
# 직접 quorum 미달 블록을 주입하려면 P2P 레벨 조작이 필요하므로,
# 여기서는 "2개 validator 중단 시 체인이 멈추는지"로 간접 검증
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-08-quorum-deficient"

# 초기 블록 진행 확인
b0=$(block_number "1")
sleep 2
b1=$(block_number "1")
assert_gt "$b1" "$b0" "chain is producing blocks initially"

# 2 validator 중단 (4 → 2, quorum 3 미달)
printf '[INFO]  stopping validator3 and validator4 (2/4 remaining, quorum 3 not met)\n' >&2
"${CHAINBENCH_DIR}/chainbench.sh" node stop 3 --quiet 2>/dev/null || true
"${CHAINBENCH_DIR}/chainbench.sh" node stop 4 --quiet 2>/dev/null || true

sleep 5
b2=$(block_number "1")
sleep 5
b3=$(block_number "1")

printf '[INFO]  blocks: before=%s, after_stop_+5s=%s, after_stop_+10s=%s\n' "$b1" "$b2" "$b3" >&2

# 블록이 정지했거나 거의 정지 (quorum 미달로 commit 불가)
diff=$(( b3 - b2 ))
assert_true "$( [[ $diff -le 2 ]] && echo true || echo false )" "chain halted or near-halted (<=2 blocks in 5s) due to quorum deficiency"

# 복구: validator3, 4 재시작
"${CHAINBENCH_DIR}/chainbench.sh" node start 3 --quiet 2>/dev/null || true
"${CHAINBENCH_DIR}/chainbench.sh" node start 4 --quiet 2>/dev/null || true
sleep 5

b4=$(block_number "1")
assert_gt "$b4" "$b3" "chain resumed after quorum restored"

test_result
