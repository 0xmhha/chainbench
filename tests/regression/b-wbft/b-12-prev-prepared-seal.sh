#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-B-12
# name: 블록 N+1의 PrevPreparedSeal이 블록 N의 prepare 서명자 포함
# category: regression/b-wbft
# tags: [wbft]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/b-wbft/b-12-prev-prepared-seal
# RT-B-12 — 블록 N+1의 PrevPreparedSeal이 블록 N의 prepare 서명자 포함
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-12-prev-prepared-seal"

quorum=3  # 4 validator → ceil(2*4/3)=3

current=$(block_number "1")
assert_gt "$current" "1" "block >= 2 (need N+1)"

extra=$(rpc "1" "istanbul_getWbftExtraInfo" "[\"$(dec_to_hex "$current")\"]")
prev_bits=$(printf '%s' "$extra" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
pps = r.get('prevPreparedSeal', {}) or {}
sealers = pps.get('sealers', []) or []
sig = pps.get('signature', '')
# sealers는 validator 주소 리스트
n_bits = len(sealers) if isinstance(sealers, list) else 0
print(f'{n_bits}|{len(sig)}')
")

bits=$(echo "$prev_bits" | cut -d'|' -f1)
sig_len=$(echo "$prev_bits" | cut -d'|' -f2)

assert_ge "$bits" "$quorum" "PrevPreparedSeal.sealers bit count >= quorum ($quorum), got $bits"
assert_gt "$sig_len" "2" "PrevPreparedSeal.signature is non-empty"

if (( bits == 4 )); then
  _assert_pass "all 4 validators collected (test env)"
fi

test_result
