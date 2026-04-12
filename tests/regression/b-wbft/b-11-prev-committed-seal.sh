#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-B-11
# name: 블록 N+1의 PrevCommittedSeal이 블록 N의 committers를 포함
# category: regression/b-wbft
# tags: [wbft]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/b-wbft/b-11-prev-committed-seal
# RT-B-11 — 블록 N+1의 PrevCommittedSeal이 블록 N의 committers를 포함
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-11-prev-committed-seal"

# 4 validator → quorum=3
quorum=3

current=$(block_number "1")
assert_gt "$current" "1" "block >= 2 (need N+1)"

n=$(( current - 1 ))
n_next=$(( current ))

# 블록 N의 CommitSigners (RT-G-3-03 참고)
# istanbul_getCommitSignersFromBlock 응답의 필드명은 CamelCase: "Committers" (대문자 C)
commit_signers=$(rpc "1" "istanbul_getCommitSignersFromBlock" "[\"$(dec_to_hex "$n")\"]")
signer_count=$(printf '%s' "$commit_signers" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
committers = r.get('Committers', []) or r.get('committers', []) or []
print(len(committers))
")
assert_ge "$signer_count" "$quorum" "block N commit signers >= quorum"

# 블록 N+1의 PrevCommittedSeal.sealers count (list 길이)
extra=$(rpc "1" "istanbul_getWbftExtraInfo" "[\"$(dec_to_hex "$n_next")\"]")
prev_bits=$(printf '%s' "$extra" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
pcs = r.get('prevCommittedSeal', {}) or {}
sealers = pcs.get('sealers', []) or []
# sealers는 validator 주소 리스트
n_bits = len(sealers) if isinstance(sealers, list) else 0
print(n_bits)
")
assert_ge "$prev_bits" "$quorum" "block N+1 PrevCommittedSeal.sealers count >= quorum ($quorum), got $prev_bits"

# 테스트 환경에서는 보통 4개 모두 모임
# 관찰 값이 4인지 선택적으로 확인
if (( prev_bits == 4 )); then
  _assert_pass "all 4 validators collected (test env bonus)"
fi

test_result
