#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-1-09,TC-1-1-10
# name: BurnRefundClaimed and BurnDepositRefunded events
# category: regression/h-hardfork
# tags: [hardfork, boho, govminter, burn, events]
# estimated_seconds: 30
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-1-1-09 + TC-1-1-10 — Verify burn refund events in logs
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-13-burn-events"
check_env || { test_result; exit 1; }

BURN_VALUE="1000000000000000000"

# 1) proposeBurn → cancel (creates refundable balance)
tx_hash=$(propose_burn "$VALIDATOR_1_ADDR" "deadbeef" "$BURN_VALUE")
receipt=$(wait_tx_receipt_full 1 "$tx_hash" 30)
assert_eq "$(json_get "$receipt" "status")" "0x1" "proposeBurn succeeded"

proposal_id=$(extract_proposal_id_from_receipt 1 "$tx_hash")
cancel_tx=$(cancel_proposal "$GOV_MINTER" "$proposal_id" "$VALIDATOR_1_ADDR")
sleep 3

# 2) Check BurnDepositRefunded event in cancel flow
# Query logs from GOV_MINTER for BurnDepositRefunded topic
cancel_receipt=$(wait_tx_receipt_full 1 "$cancel_tx" 15)
if [[ -n "$cancel_receipt" ]]; then
  deposit_event=$(find_log_by_topic "$cancel_receipt" "$GOV_MINTER" "$BURN_DEPOSIT_REFUNDED_SIG")
  if [[ -n "$deposit_event" && "$deposit_event" != "null" ]]; then
    _assert_pass "BurnDepositRefunded event found in cancel receipt"
  else
    observe "cancel_receipt_logs" "$(json_get "$cancel_receipt" "logs")"
    _assert_fail "BurnDepositRefunded event NOT found in cancel receipt"
  fi
fi

# 3) claimBurnRefund → check BurnRefundClaimed event
claim_tx=$(claim_burn_refund "$VALIDATOR_1_ADDR")
assert_not_empty "$claim_tx" "claimBurnRefund tx sent"
sleep 3

claim_receipt=$(wait_tx_receipt_full 1 "$claim_tx" 15)
assert_not_empty "$claim_receipt" "claim receipt received"

claimed_event=$(find_log_by_topic "$claim_receipt" "$GOV_MINTER" "$BURN_REFUND_CLAIMED_SIG")
if [[ -n "$claimed_event" && "$claimed_event" != "null" ]]; then
  _assert_pass "BurnRefundClaimed event found in claim receipt"
else
  _assert_fail "BurnRefundClaimed event NOT found in claim receipt"
fi

test_result
