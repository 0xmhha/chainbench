#!/usr/bin/env bash
# RT-F-5-07 — AddressUnblacklisted 이벤트
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/f-system-contracts/f5-07-address-unblacklisted-event"
check_env || { test_result; exit 1; }

receipt=$(cat /tmp/chainbench-regression/f5_02_receipt.json 2>/dev/null || echo "")
if [[ -z "$receipt" ]]; then
  _assert_fail "run f5-02 first"; test_result; exit 1
fi

sig=$(python3 -c "from eth_utils import keccak; print('0x' + keccak(text='AddressUnblacklisted(address,uint256)').hex())")
log=$(find_log_by_topic "$receipt" "$GOV_COUNCIL" "$sig")
assert_not_empty "$log" "AddressUnblacklisted event in f5-02 receipt"

test_result
