#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-1-01
# name: 제네시스 블록으로 노드 초기화
# category: regression/a-ethereum
# tags: [sync, genesis]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/a-ethereum/a1-01-genesis-init
# RT-A-1-01 — 제네시스 블록으로 노드 초기화
# 모든 노드에서 블록 0 해시가 동일해야 함
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a1-01-genesis-init"

check_env || { test_result; exit 1; }

# 모든 노드에서 블록 0 조회
declare -a block0_hashes=()
for node in 1 2 3 4 5; do
  resp=$(rpc "$node" "eth_getBlockByNumber" '["0x0", false]' 2>/dev/null || echo "")
  hash=$(json_get "$resp" "result.hash")
  if [[ -n "$hash" && "$hash" != "null" ]]; then
    block0_hashes+=("$hash")
    printf '[INFO]  node%s block 0 hash = %s\n' "$node" "$hash" >&2
  fi
done

assert_gt "${#block0_hashes[@]}" "0" "at least one node returned block 0"

# 모든 해시가 동일한지 확인
first_hash="${block0_hashes[0]}"
all_same=true
for h in "${block0_hashes[@]}"; do
  [[ "$h" != "$first_hash" ]] && all_same=false
done
assert_true "$all_same" "all nodes have identical genesis block 0 hash"

# chainId가 8283(0x205B)인지 확인 (regression profile 기준)
chain_id_hex=$(rpc "1" "eth_chainId" "[]" | json_get - result)
chain_id_dec=$(hex_to_dec "$chain_id_hex")
assert_eq "$chain_id_dec" "8283" "chainId matches regression profile"

test_result
