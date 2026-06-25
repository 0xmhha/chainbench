#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-1-05
# name: claimBurnRefund increases native balance
# category: regression/h-hardfork
# tags: [hardfork, boho, govminter, burn, claim, refund]
# estimated_seconds: 30
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# TC-1-1-05 — cancel → claimBurnRefund → native balance increases
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-10-claim-burn-refund"
check_env || { test_result; exit 1; }
ensure_nodes_running

BURN_VALUE="1000000000000000000"

# 1) Create refundable balance (proposeBurn → cancel)
tx_hash=$(propose_burn "$(validator_addr 1)" "deadbeef" "$BURN_VALUE")
assert_not_empty "$tx_hash" "proposeBurn tx sent"
receipt=$(wait_tx_receipt_full "$(node 1)" "$tx_hash" 30)
assert_eq "$(json_get "$receipt" "status")" "0x1" "proposeBurn succeeded"

proposal_id=$(extract_proposal_id_from_receipt 1 "$tx_hash")
cancel_proposal "$GOV_MINTER" "$proposal_id" "$(validator_addr 1)"
sleep 3

# 2) Record balance before claim
balance_before=$(get_balance "$(node 1)" "$(validator_addr 1)")
observe "balance_before" "$balance_before"

# 3) Claim refund
claim_tx=$(claim_burn_refund "$(validator_addr 1)")
assert_not_empty "$claim_tx" "claimBurnRefund tx sent"
sleep 3

# 4) Verify balance increased (minus gas)
balance_after=$(get_balance "$(node 1)" "$(validator_addr 1)")
observe "balance_after" "$balance_after"

assert_gt "$balance_after" "$balance_before" "balance increased after claimBurnRefund"

# 5) Verify refundableBalance is now 0
refundable=$(get_refundable_balance 1 "$(validator_addr 1)")
assert_eq "$refundable" "0" "refundableBalance == 0 after claim"

test_result
