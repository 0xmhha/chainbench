#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-4-04
# name: GovMinter bytecode changes at BohoBlock (delayed activation)
# category: regression/h-hardfork
# tags: [hardfork, boho, delayed, injection]
# estimated_seconds: 30
# preconditions:
#   chain_running: true
#   profile: hardfork-boho-delayed
# depends_on: []
# ---end-meta---
# TC-4-4-04 — GovMinter v1 before BohoBlock, v2 after
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-29-inject-contracts-delayed"
check_env || { test_result; exit 1; }

BOHO_BLOCK=10

# --- 1) Record v1 bytecode before BohoBlock ---
current=$(block_number 1)
if (( current < BOHO_BLOCK )); then
  block_hex=$(printf '0x%x' "$current")
  resp=$(rpc 1 "eth_getCode" "[\"${GOV_MINTER}\",\"${block_hex}\"]")
  code_v1=$(json_get "$resp" "result")
  code_v1_len=${#code_v1}
  printf '[INFO]  GovMinter code at block %d (pre-Boho): %d hex chars\n' "$current" "$code_v1_len" >&2
  observe "govminter_v1_len" "$code_v1_len"
  assert_gt "$code_v1_len" "2" "GovMinter v1 bytecode exists before BohoBlock"
else
  printf '[WARN]  Already past BohoBlock %d, skipping pre-Boho check\n' "$BOHO_BLOCK" >&2
  code_v1=""
  code_v1_len=0
fi

# --- 2) Wait for Boho activation ---
wait_for_block 1 "$BOHO_BLOCK" 60

# --- 3) Check v2 bytecode after BohoBlock ---
sleep 2  # allow 1-2 blocks for state to settle
resp=$(rpc 1 "eth_getCode" "[\"${GOV_MINTER}\",\"latest\"]")
code_v2=$(json_get "$resp" "result")
code_v2_len=${#code_v2}
printf '[INFO]  GovMinter code at latest (post-Boho): %d hex chars\n' "$code_v2_len" >&2
observe "govminter_v2_len" "$code_v2_len"

assert_gt "$code_v2_len" "100" "GovMinter v2 bytecode deployed after BohoBlock"

# --- 4) Verify code changed ---
if [[ -n "$code_v1" && "$code_v1_len" -gt 2 ]]; then
  if [[ "$code_v1" != "$code_v2" ]]; then
    _assert_pass "GovMinter bytecode changed at BohoBlock (v1 -> v2)"
  else
    _assert_fail "GovMinter bytecode did NOT change at BohoBlock"
  fi
fi

test_result
