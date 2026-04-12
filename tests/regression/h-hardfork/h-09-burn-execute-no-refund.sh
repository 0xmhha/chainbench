#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-1-04
# name: Executed burn has no refundable balance
# category: regression/h-hardfork
# tags: [hardfork, boho, govminter, burn, execute]
# estimated_seconds: 30
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-1-1-04 — proposeBurn → approve → execute → refundableBalance == 0
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-09-burn-execute-no-refund"
check_env || { test_result; exit 1; }

BURN_VALUE="1000000000000000000"

# 1) proposeBurn
tx_hash=$(propose_burn "$VALIDATOR_1_ADDR" "deadbeef" "$BURN_VALUE")
assert_not_empty "$tx_hash" "proposeBurn tx sent"
receipt=$(wait_tx_receipt_full 1 "$tx_hash" 30)
assert_eq "$(json_get "$receipt" "status")" "0x1" "proposeBurn succeeded"

proposal_id=$(extract_proposal_id_from_receipt 1 "$tx_hash")
assert_not_empty "$proposal_id" "proposalId extracted"

# 2) Approve + execute via governance
printf '[INFO]  approving proposal %s\n' "$proposal_id" >&2
gov_approve 1 "$GOV_MINTER" "$proposal_id" "$VALIDATOR_2_ADDR"
sleep 2
gov_approve 1 "$GOV_MINTER" "$proposal_id" "$VALIDATOR_3_ADDR"
sleep 3

# Check if auto-executed
prop_status=$(gov_proposal_status 1 "$GOV_MINTER" "$proposal_id")
if [[ "$prop_status" != "3" ]]; then
  gov_execute 1 "$GOV_MINTER" "$proposal_id" "$VALIDATOR_1_ADDR"
  sleep 3
fi

# 3) Verify refundableBalance == 0 (burn was executed, funds consumed)
refundable=$(get_refundable_balance 1 "$VALIDATOR_1_ADDR")
observe "refundable_balance" "$refundable"
assert_eq "$refundable" "0" "refundableBalance == 0 after successful burn execution"

test_result
