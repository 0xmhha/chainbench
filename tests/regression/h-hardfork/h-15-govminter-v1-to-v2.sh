#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-1-12
# name: GovMinter bytecode transitions from v1 to v2 at BohoBlock
# category: regression/h-hardfork
# tags: [hardfork, boho, govminter, upgrade, bytecode, delayed]
# estimated_seconds: 60
# preconditions:
#   chain_running: true
#   profile: hardfork-boho-delayed
# depends_on: []
# ---end-meta---
# TC-1-1-12 — Before BohoBlock=10: record v1 code hash/length.
#             After BohoBlock=10: verify code changed (different hash or length).
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-15-govminter-v1-to-v2"
check_env || { test_result; exit 1; }

BOHO_BLOCK=10

# --- 1) Record v1 bytecode before BohoBlock ---
current=$(block_number 1)
observe "current_block" "$current"

if (( current < BOHO_BLOCK )); then
  block_hex=$(printf '0x%x' "$current")
  resp=$(rpc 1 "eth_getCode" "[\"${GOV_MINTER}\",\"${block_hex}\"]")
  code_v1=$(json_get "$resp" "result")
  code_v1_len=${#code_v1}
  printf '[INFO]  GovMinter code at block %d (pre-Boho): %d hex chars\n' "$current" "$code_v1_len" >&2
  observe "govminter_v1_len" "$code_v1_len"
  assert_gt "$code_v1_len" "2" "GovMinter v1 bytecode exists before BohoBlock"
else
  printf '[WARN]  Already past BohoBlock %d (current=%d), recording current code as v1 baseline\n' \
    "$BOHO_BLOCK" "$current" >&2
  resp=$(rpc 1 "eth_getCode" "[\"${GOV_MINTER}\",\"latest\"]")
  code_v1=$(json_get "$resp" "result")
  code_v1_len=${#code_v1}
  observe "govminter_current_len" "$code_v1_len"
  assert_gt "$code_v1_len" "100" "GovMinter has substantial bytecode (already past Boho)"
  _assert_pass "Skipping v1->v2 transition check (chain already past BohoBlock $BOHO_BLOCK)"
  test_result
  exit 0
fi

# --- 2) Wait for BohoBlock ---
printf '[INFO]  Waiting for BohoBlock %d\n' "$BOHO_BLOCK" >&2
wait_for_block 1 "$BOHO_BLOCK" 120

# Allow 1-2 blocks for state injection to settle
sleep 2

# --- 3) Check v2 bytecode after BohoBlock ---
resp=$(rpc 1 "eth_getCode" "[\"${GOV_MINTER}\",\"latest\"]")
code_v2=$(json_get "$resp" "result")
code_v2_len=${#code_v2}
printf '[INFO]  GovMinter code at latest (post-Boho): %d hex chars\n' "$code_v2_len" >&2
observe "govminter_v2_len" "$code_v2_len"

assert_gt "$code_v2_len" "100" "GovMinter v2 bytecode deployed after BohoBlock"

# --- 4) Verify code changed ---
if [[ "$code_v1" == "$code_v2" ]]; then
  _assert_fail "GovMinter bytecode did NOT change at BohoBlock (v1 == v2)"
else
  _assert_pass "GovMinter bytecode changed at BohoBlock (v1 -> v2 upgrade confirmed)"
fi

# --- 5) Verify length also differs (belt-and-suspenders) ---
if (( code_v1_len != code_v2_len )); then
  _assert_pass "GovMinter bytecode length changed: ${code_v1_len} -> ${code_v2_len} hex chars"
else
  printf '[INFO]  Bytecode length unchanged (%d) but content differed\n' "$code_v1_len" >&2
fi

test_result
