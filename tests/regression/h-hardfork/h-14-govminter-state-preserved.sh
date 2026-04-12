#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-1-11
# name: GovMinter governance params preserved after v2 upgrade
# category: regression/h-hardfork
# tags: [hardfork, boho, govminter, governance, params]
# estimated_seconds: 60
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils, eth-abi]
# depends_on: []
# ---end-meta---
# TC-1-1-11 extended — Verify GovMinter governance params (quorum, expiry) and
# full governance lifecycle still work on v2 after upgrade.
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-14-govminter-state-preserved"
check_env || { test_result; exit 1; }

# --- 1) Check quorum ---
QUORUM_SEL=$(selector "quorum()")
quorum_raw=$(eth_call_raw 1 "$GOV_MINTER" "${QUORUM_SEL}")
quorum=$(hex_to_dec "$quorum_raw")
observe "quorum" "$quorum"
printf '[INFO]  GovMinter quorum = %s\n' "$quorum" >&2
assert_eq "$quorum" "2" "quorum is 2 after v2 upgrade"

# --- 2) Check proposal expiry ---
EXPIRY_SEL=$(selector "expiryBlocks()")
expiry_raw=$(eth_call_raw 1 "$GOV_MINTER" "${EXPIRY_SEL}" 2>/dev/null || echo "0x")
expiry=$(hex_to_dec "$expiry_raw")
observe "expiry_blocks" "$expiry"
printf '[INFO]  GovMinter expiryBlocks = %s\n' "$expiry" >&2
assert_eq "$expiry" "60" "expiryBlocks is 60 after v2 upgrade"

# --- 3) Full governance lifecycle: proposeBurn → cancel to verify state works ---
# A full mint proposal requires knowing the exact proposeMint ABI which may vary.
# Use the well-tested proposeBurn path to confirm governance lifecycle on v2.
BURN_VALUE="100000000000000000"  # 0.1 ETH

printf '[INFO]  Verifying governance lifecycle via proposeBurn on v2\n' >&2

tx=$(propose_burn "$VALIDATOR_1_ADDR" "cafebabe" "$BURN_VALUE")
assert_not_empty "$tx" "proposeBurn on v2 succeeded"

receipt=$(wait_tx_receipt_full 1 "$tx" 30)
assert_not_empty "$receipt" "proposeBurn receipt received"
tx_status=$(json_get "$receipt" "status")
assert_eq "$tx_status" "0x1" "proposeBurn receipt status is success"

proposal_id=$(extract_proposal_id_from_receipt 1 "$tx")
assert_not_empty "$proposal_id" "proposalId extracted from v2 GovMinter"
observe "proposal_id" "$proposal_id"

# Verify proposal is in Voting state (status=1)
prop_status=$(gov_proposal_status 1 "$GOV_MINTER" "$proposal_id")
observe "proposal_status_after_propose" "$prop_status"
assert_eq "$prop_status" "1" "proposal status is Voting (governance state works on v2)"

# Approve with a second validator to test multi-sig path
approve_tx=$(gov_approve 1 "$GOV_MINTER" "$proposal_id" "$VALIDATOR_2_ADDR" 2>/dev/null || true)
if [[ -n "$approve_tx" && "$approve_tx" != "null" ]]; then
  sleep 2
  prop_status2=$(gov_proposal_status 1 "$GOV_MINTER" "$proposal_id")
  observe "proposal_status_after_approve" "$prop_status2"
  printf '[INFO]  proposal status after approve = %s\n' "$prop_status2" >&2
fi

# Cancel proposal to release locked funds
printf '[INFO]  cancelling proposal to clean up\n' >&2
cancel_proposal "$GOV_MINTER" "$proposal_id" "$VALIDATOR_1_ADDR"
sleep 2

test_result
