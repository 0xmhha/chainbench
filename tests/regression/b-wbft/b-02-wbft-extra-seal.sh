#!/usr/bin/env bash
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
sealers = cs.get('sealers', '')
sig = cs.get('signature', '')
# sealers bitmap은 hex 문자열. 1인 bit 수를 계산
n_bits = bin(int(sealers, 16)).count('1') if sealers and sealers != '0x0' else 0
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
sealers = ps.get('sealers', '')
sig = ps.get('signature', '')
n_bits = bin(int(sealers, 16)).count('1') if sealers and sealers != '0x0' else 0
print(f'{n_bits}|{len(sig)}')
")
ps_bits=$(echo "$prepared" | cut -d'|' -f1)
ps_sig_len=$(echo "$prepared" | cut -d'|' -f2)

assert_ge "$ps_bits" "$quorum" "preparedSeal.sealers bit count >= quorum ($quorum), got $ps_bits"
assert_gt "$ps_sig_len" "2" "preparedSeal.signature is non-empty"

test_result
