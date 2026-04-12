#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-4-05
# name: eth_chainId genesis 설정과 일치
# category: regression/a-ethereum
# tags: [rpc, chainId]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/a-ethereum/a4-05-eth-chain-id
# RT-A-4-05 — eth_chainId genesis 설정과 일치
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a4-05-eth-chain-id"

chain_id_hex=$(rpc "1" "eth_chainId" "[]" | json_get - result)
chain_id_dec=$(hex_to_dec "$chain_id_hex")

printf '[INFO]  eth_chainId returned: %s (dec: %s)\n' "$chain_id_hex" "$chain_id_dec" >&2
# regression profile의 chainId = 8283 (0x205B)
assert_eq "$chain_id_dec" "8283" "chainId == 8283 (regression profile)"

# 모든 노드가 동일한 chainId 반환
for node in 2 3 4 5; do
  id=$(rpc "$node" "eth_chainId" "[]" | json_get - result)
  id_dec=$(hex_to_dec "$id")
  assert_eq "$id_dec" "8283" "node${node} chainId == 8283"
done

test_result
