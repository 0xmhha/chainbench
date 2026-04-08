#!/usr/bin/env bash
# RT-F-5-09 — AuthorizedAccountRemoved 이벤트
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/f-system-contracts/f5-09-authorized-account-removed-event"

receipt=$(cat /tmp/chainbench-regression/f5_04_receipt.json 2>/dev/null || echo "")
if [[ -z "$receipt" ]]; then
  _assert_fail "run f5-04 first"; test_result; exit 1
fi

sig=$(python3 -c "from eth_utils import keccak; print('0x' + keccak(text='AuthorizedAccountRemoved(address,uint256)').hex())")
log=$(find_log_by_topic "$receipt" "$GOV_COUNCIL" "$sig")
assert_not_empty "$log" "AuthorizedAccountRemoved event in f5-04 receipt"

test_result
