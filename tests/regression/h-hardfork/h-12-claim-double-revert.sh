#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-1-07
# name: Second claimBurnRefund reverts (double claim prevention)
# category: regression/h-hardfork
# tags: [hardfork, boho, govminter, burn, revert]
# estimated_seconds: 40
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-1-1-07 — First claim succeeds, second claim reverts
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-12-claim-double-revert"
check_env || { test_result; exit 1; }

BURN_VALUE="1000000000000000000"

# 1) Create refundable balance
tx=$(propose_burn "$VALIDATOR_1_ADDR" "deadbeef" "$BURN_VALUE")
assert_not_empty "$tx" "proposeBurn sent"
receipt=$(wait_tx_receipt_full 1 "$tx" 30)
assert_eq "$(json_get "$receipt" "status")" "0x1" "proposeBurn succeeded"
proposal_id=$(extract_proposal_id_from_receipt 1 "$tx")
cancel_proposal "$GOV_MINTER" "$proposal_id" "$VALIDATOR_1_ADDR"
sleep 3

# 2) First claim — should succeed
refundable=$(get_refundable_balance 1 "$VALIDATOR_1_ADDR")
assert_gt "$refundable" "0" "refundableBalance > 0 before first claim"

claim1=$(claim_burn_refund "$VALIDATOR_1_ADDR")
assert_not_empty "$claim1" "first claimBurnRefund sent"
sleep 3

refundable_after=$(get_refundable_balance 1 "$VALIDATOR_1_ADDR")
assert_eq "$refundable_after" "0" "refundableBalance == 0 after first claim"

# 3) Second claim — should revert
claim2=$(claim_burn_refund "$VALIDATOR_1_ADDR" 2>/dev/null || true)
if [[ -z "$claim2" || "$claim2" == "null" ]]; then
  _assert_pass "second claimBurnRefund rejected (no tx hash)"
else
  sleep 3
  receipt2=$(wait_tx_receipt_full 1 "$claim2" 15)
  if [[ -n "$receipt2" ]]; then
    status2=$(json_get "$receipt2" "status")
    assert_eq "$status2" "0x0" "second claimBurnRefund reverted"
  else
    _assert_pass "second claimBurnRefund failed (no receipt)"
  fi
fi

test_result
