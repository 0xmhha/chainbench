#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-4-5-08
# name: alloc.Extra sync must not alter account balances
# category: regression/h-hardfork
# tags: [hardfork, boho, account-extra]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   profile: hardfork-account-extra
# depends_on: []
# ---end-meta---
# TC-4-5-08 — After genesis init syncs Extra bits into AccountManager/GovCouncil,
#              TEST_ACC_D and TEST_ACC_E must still hold their allocated balances.
#              alloc.balance = 0xDE0B6B3A7640000 = 1 ETH = 1000000000000000000 wei
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-34-extra-balance-preserved"
check_env || { test_result; exit 1; }

# Expected balance: 1 ETH in wei
readonly EXPECTED_BALANCE_WEI="1000000000000000000"

# --- TEST_ACC_D balance ---
td_balance=$(get_balance 1 "$TEST_ACC_D_ADDR")
observe "balance_TEST_ACC_D_wei" "$td_balance"
assert_eq "$td_balance" "$EXPECTED_BALANCE_WEI" \
  "TC-4-5-08a: TEST_ACC_D balance == 1 ETH (Extra sync did not alter balance)"

# --- TEST_ACC_E balance ---
te_balance=$(get_balance 1 "$TEST_ACC_E_ADDR")
observe "balance_TEST_ACC_E_wei" "$te_balance"
assert_eq "$te_balance" "$EXPECTED_BALANCE_WEI" \
  "TC-4-5-08b: TEST_ACC_E balance == 1 ETH (Extra sync did not alter balance)"

test_result
