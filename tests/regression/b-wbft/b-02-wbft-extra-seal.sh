#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-B-02
# name: WBFTExtra에 Committed Seal + Prepared Seal 모두 존재하고 quorum 이상
# category: regression/b-wbft
# tags: [wbft]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/b-wbft/b-02-wbft-extra-seal
# RT-B-02 — WBFTExtra에 Committed Seal + Prepared Seal 모두 존재하고 quorum 이상
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/b-wbft/b-02-wbft-extra-seal"

# 4 validator 환경 → quorum = ceil(2*4/3) = 3
quorum=3

target=$(block_number "1")
# 최소 1블록은 있어야
assert_gt "$target" "0" "block exists"

# istanbul_getWbftExtraInfo 조회
resp=$(rpc "1" "istanbul_getWbftExtraInfo" "[\"$(dec_to_hex "$target")\"]")
committed=$(printf '%s' "$resp" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
cs = r.get('committedSeal', {}) or {}
sealers = cs.get('sealers', []) or []
sig = cs.get('signature', '')
# sealers는 validator 주소 리스트 (string array). 길이가 서명자 수.
n_bits = len(sealers) if isinstance(sealers, list) else 0
print(f'{n_bits}|{len(sig)}')
")
cs_bits=$(echo "$committed" | cut -d'|' -f1)
cs_sig_len=$(echo "$committed" | cut -d'|' -f2)

assert_ge "$cs_bits" "$quorum" "committedSeal.sealers bit count >= quorum ($quorum), got $cs_bits"
assert_gt "$cs_sig_len" "2" "committedSeal.signature is non-empty"

# Prepared seal
prepared=$(printf '%s' "$resp" | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
ps = r.get('preparedSeal', {}) or {}
sealers = ps.get('sealers', []) or []
sig = ps.get('signature', '')
# sealers는 validator 주소 리스트
n_bits = len(sealers) if isinstance(sealers, list) else 0
print(f'{n_bits}|{len(sig)}')
")
ps_bits=$(echo "$prepared" | cut -d'|' -f1)
ps_sig_len=$(echo "$prepared" | cut -d'|' -f2)

assert_ge "$ps_bits" "$quorum" "preparedSeal.sealers bit count >= quorum ($quorum), got $ps_bits"
assert_gt "$ps_sig_len" "2" "preparedSeal.signature is non-empty"

test_result
