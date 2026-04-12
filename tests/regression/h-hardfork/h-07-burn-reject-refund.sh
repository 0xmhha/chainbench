#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-1-02
# name: proposeBurn rejection creates refundable balance
# category: regression/h-hardfork
# tags: [hardfork, boho, govminter, burn, reject, refund]
# estimated_seconds: 30
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-1-1-02 — proposeBurn → reject (quorum disapprove) → refundableBalance > 0
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-07-burn-reject-refund"
check_env || { test_result; exit 1; }

BURN_VALUE="1000000000000000000"

# 1) proposeBurn
tx_hash=$(propose_burn "$VALIDATOR_1_ADDR" "deadbeef" "$BURN_VALUE")
assert_not_empty "$tx_hash" "proposeBurn tx sent"
receipt=$(wait_tx_receipt_full 1 "$tx_hash" 30)
status=$(json_get "$receipt" "status")
assert_eq "$status" "0x1" "proposeBurn succeeded"

proposal_id=$(extract_proposal_id_from_receipt 1 "$tx_hash")
assert_not_empty "$proposal_id" "proposalId extracted"

# 2) Disapprove from 2 validators (quorum=2 → rejected)
printf '[INFO]  disapproving proposal %s\n' "$proposal_id" >&2
disapprove_proposal "$GOV_MINTER" "$proposal_id" "$VALIDATOR_2_ADDR"
sleep 2
disapprove_proposal "$GOV_MINTER" "$proposal_id" "$VALIDATOR_3_ADDR"
sleep 3

# 3) Verify refundableBalance
refundable=$(get_refundable_balance 1 "$VALIDATOR_1_ADDR")
observe "refundable_balance" "$refundable"
assert_gt "$refundable" "0" "refundableBalance > 0 after rejection"

test_result
