#!/usr/bin/env bash
# Test: regression/b-wbft/b-03-epoch-transition
# RT-B-03 — 에폭 전환 — 검증자 집합 갱신
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-03-epoch-transition"

# regression profile의 epoch = 30
EPOCH_LENGTH=30

# 현재 블록 확인
current=$(block_number "1")

# 첫 에폭 경계 블록 번호 계산 (epoch_length - 1 이 epoch block)
# 예: epochLength=30 → blocks 29, 59, 89, ... 이 epoch 전환 블록
first_epoch_block=$(( (current / EPOCH_LENGTH + 1) * EPOCH_LENGTH - 1 ))
printf '[INFO]  current=%s, waiting for epoch block %s\n' "$current" "$first_epoch_block" >&2

wait_for_block "1" "$first_epoch_block" 60 >/dev/null

# 에폭 블록의 WBFTExtra 조회
resp=$(rpc "1" "istanbul_getWbftExtraInfo" "[\"$(dec_to_hex "$first_epoch_block")\"]")
epoch_info_present=$(printf '%s' "$resp" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
ei = r.get('epochInfo')
print('yes' if ei else 'no')
")
assert_eq "$epoch_info_present" "yes" "epoch block $first_epoch_block contains epochInfo"

# 검증자 집합 길이 (4)
validator_count=$(printf '%s' "$resp" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
ei = r.get('epochInfo', {}) or {}
vals = ei.get('validators', []) or ei.get('candidates', [])
print(len(vals))
")
assert_ge "$validator_count" "4" "epoch candidates length >= 4"

test_result
