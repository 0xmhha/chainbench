#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-1-01
# name: proposeBurn cancel creates refundable balance
# category: regression/h-hardfork
# tags: [hardfork, boho, govminter, burn, refund]
# estimated_seconds: 30
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-1-1-01 — proposeBurn → cancel → refundableBalance > 0
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-06-burn-cancel-refund"
check_env || { test_result; exit 1; }

BURN_VALUE="1000000000000000000"  # 1 ETH

# 1) proposeBurn with 1 ETH from VALIDATOR_1
printf '[INFO]  proposeBurn from %s with %s wei\n' "$VALIDATOR_1_ADDR" "$BURN_VALUE" >&2
tx_hash=$(propose_burn "$VALIDATOR_1_ADDR" "deadbeef" "$BURN_VALUE")
assert_not_empty "$tx_hash" "proposeBurn tx sent"

receipt=$(wait_tx_receipt_full 1 "$tx_hash" 30)
assert_not_empty "$receipt" "proposeBurn receipt received"
status=$(json_get "$receipt" "status")
assert_eq "$status" "0x1" "proposeBurn succeeded"

# Extract proposal ID
proposal_id=$(extract_proposal_id_from_receipt 1 "$tx_hash")
assert_not_empty "$proposal_id" "proposalId extracted"
observe "proposal_id" "$proposal_id"

# 2) Cancel proposal (proposer can cancel)
printf '[INFO]  cancelling proposal %s\n' "$proposal_id" >&2
cancel_tx=$(cancel_proposal "$GOV_MINTER" "$proposal_id" "$VALIDATOR_1_ADDR")
assert_not_empty "$cancel_tx" "cancel tx sent"
sleep 3

# 3) Verify refundableBalance > 0
refundable=$(get_refundable_balance 1 "$VALIDATOR_1_ADDR")
observe "refundable_balance" "$refundable"
assert_gt "$refundable" "0" "refundableBalance > 0 after cancel"

# 4) Verify burnBalance == 0 (no active burn)
burn_bal=$(get_burn_balance 1 "$VALIDATOR_1_ADDR")
observe "burn_balance" "$burn_bal"
assert_eq "$burn_bal" "0" "burnBalance == 0 after cancel"

test_result
