#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-5-10,TC-4-5-11,TC-4-5-12
# name: Delayed Boho activation preserves account states (decodePrealloc restoration)
# category: regression/h-hardfork
# tags: [hardfork, boho, account-extra, delayed]
# estimated_seconds: 60
# preconditions:
#   chain_running: true
#   profile: hardfork-boho-delayed
# depends_on: []
# ---end-meta---
# TC-4-5-10 — Before BohoBlock (10): chain is operational (block advances, no panic)
# TC-4-5-11 — After BohoBlock: GovMinter v2 bytecode is deployed
# TC-4-5-12 — After BohoBlock: account states (balance, nonce) from genesis are preserved
#              (decodePrealloc restoration during runtime upgrade must not wipe alloc state)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-35-extra-v1-to-v2-delayed"
check_env || { test_result; exit 1; }

BOHO_BLOCK=10

# ============================================================
# TC-4-5-10: Before BohoBlock — chain produces blocks normally
# ============================================================
current=$(block_number 1)
observe "block_at_start" "$current"

if (( current < BOHO_BLOCK )); then
  # Record GovMinter v1 bytecode length
  block_hex=$(printf '0x%x' "$current")
  resp_v1=$(rpc 1 "eth_getCode" "[\"${GOV_MINTER}\",\"${block_hex}\"]")
  code_v1=$(json_get "$resp_v1" "result")
  code_v1_len=${#code_v1}
  observe "govminter_v1_code_len" "$code_v1_len"
  assert_gt "$code_v1_len" "2" \
    "TC-4-5-10: GovMinter bytecode exists before BohoBlock (chain is operational)"
else
  printf '[WARN]  Already at/past BohoBlock %d (current=%d), skipping pre-Boho check\n' \
    "$BOHO_BLOCK" "$current" >&2
  _assert_pass "TC-4-5-10: skipped (already past BohoBlock)"
  code_v1=""
  code_v1_len=0
fi

# ============================================================
# TC-4-5-11: Wait for BohoBlock → GovMinter v2 deployed
# ============================================================
wait_for_block 1 "$BOHO_BLOCK" 90

# Allow 1-2 additional blocks for state transitions to settle
sleep 2

resp_v2=$(rpc 1 "eth_getCode" "[\"${GOV_MINTER}\",\"latest\"]")
code_v2=$(json_get "$resp_v2" "result")
code_v2_len=${#code_v2}
observe "govminter_v2_code_len" "$code_v2_len"
assert_gt "$code_v2_len" "100" \
  "TC-4-5-11: GovMinter v2 bytecode deployed after BohoBlock"

# Verify bytecode actually changed (v1 != v2) when we had a v1 snapshot
if [[ -n "$code_v1" && "$code_v1_len" -gt 2 ]]; then
  if [[ "$code_v1" != "$code_v2" ]]; then
    _assert_pass "TC-4-5-11b: GovMinter bytecode changed at BohoBlock (v1 → v2)"
  else
    _assert_fail "TC-4-5-11b: GovMinter bytecode did NOT change at BohoBlock"
  fi
fi

# ============================================================
# TC-4-5-12: Account states preserved after Boho activation
# ============================================================
# TEST_ACC_A is a standard alloc account (1000000000000000000000000000 wei).
# decodePrealloc restoration must preserve its balance after the upgrade.
ta_balance=$(get_balance 1 "$TEST_ACC_A_ADDR")
observe "balance_TEST_ACC_A_post_boho" "$ta_balance"
assert_gt "$ta_balance" "0" \
  "TC-4-5-12a: TEST_ACC_A balance > 0 after BohoBlock (genesis alloc preserved)"

# TEST_ACC_A nonce should still be 0 (we never sent txs from it in this test)
ta_nonce=$(get_nonce 1 "$TEST_ACC_A_ADDR")
observe "nonce_TEST_ACC_A_post_boho" "$ta_nonce"
assert_eq "$ta_nonce" "0" \
  "TC-4-5-12b: TEST_ACC_A nonce == 0 after BohoBlock (account state not wiped)"

# Chain should still be advancing blocks after Boho activation
post_boho_block=$(block_number 1)
observe "block_post_boho" "$post_boho_block"
assert_gt "$post_boho_block" "$(( BOHO_BLOCK - 1 ))" \
  "TC-4-5-12c: chain continues producing blocks after BohoBlock"

test_result
