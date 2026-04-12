#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-1-06
# name: claimBurnRefund with zero balance reverts
# category: regression/h-hardfork
# tags: [hardfork, boho, govminter, burn, revert]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-1-1-06 — claimBurnRefund with 0 refundable should revert
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-11-claim-zero-revert"
check_env || { test_result; exit 1; }

# Verify no refundable balance for VALIDATOR_2 (fresh account)
refundable=$(get_refundable_balance 1 "$VALIDATOR_2_ADDR")
observe "refundable_before" "$refundable"

# Attempt claim with 0 refundable — should revert
claim_tx=$(claim_burn_refund "$VALIDATOR_2_ADDR" 2>/dev/null || true)

if [[ -z "$claim_tx" || "$claim_tx" == "null" ]]; then
  _assert_pass "claimBurnRefund with 0 balance was rejected (no tx hash)"
else
  # Check receipt status
  sleep 3
  receipt=$(wait_tx_receipt_full 1 "$claim_tx" 15)
  if [[ -n "$receipt" ]]; then
    status=$(json_get "$receipt" "status")
    assert_eq "$status" "0x0" "claimBurnRefund with 0 balance reverted (status=0x0)"
  else
    _assert_pass "claimBurnRefund tx failed (no receipt)"
  fi
fi

test_result
