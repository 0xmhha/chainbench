#!/usr/bin/env bash
# Test: regression/f-system-contracts/f5-06-address-blacklisted-event
# RT-F-5-06 — blacklist 등록 시 AddressBlacklisted 이벤트 발생
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/f-system-contracts/f5-06-address-blacklisted-event"
check_env || { test_result; exit 1; }

receipt=$(cat /tmp/chainbench-regression/f5_01_receipt.json 2>/dev/null || echo "")
if [[ -z "$receipt" ]]; then
  _assert_fail "run f5-01 first"; test_result; exit 1
fi

# AddressBlacklisted(address indexed account, uint256 indexed proposalId)
sig=$(python3 -c "from eth_utils import keccak; print('0x' + keccak(text='AddressBlacklisted(address,uint256)').hex())")
log=$(find_log_by_topic "$receipt" "$GOV_COUNCIL" "$sig")
assert_not_empty "$log" "AddressBlacklisted event in f5-01 receipt"

test_result
