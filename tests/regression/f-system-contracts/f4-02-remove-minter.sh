#!/usr/bin/env bash
# Test: regression/f-system-contracts/f4-02-remove-minter
# RT-F-4-02 — GovMasterMinter.proposeRemoveMinter(address) → 승인 → execute
#
# ABI (v1 GovMasterMinter.sol:130): function proposeRemoveMinter(address minter)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f4-02-remove-minter"
check_env || { test_result; exit 1; }

unlock_all_validators

# F-4-01에서 TEST_ACC_D를 Minter로 등록했다고 가정
minter_addr="$TEST_ACC_D_ADDR"
m_padded=$(pad_address "$minter_addr" | sed 's/^0x//')

# 현재 상태 확인
is_minter_sel=$(selector "isMinter(address)")
before=$(hex_to_dec "$(eth_call_raw 1 "$NATIVE_COIN_ADAPTER" "${is_minter_sel}${m_padded}" 2>/dev/null || echo "0x0")")
if [[ "$before" != "1" ]]; then
  printf '[WARN]  TEST_ACC_D is not a Minter, run f4-01 first\n' >&2
fi

propose_sel=$(selector "proposeRemoveMinter(address)")
propose_data="${propose_sel}${m_padded}"

receipt=$(gov_full_flow "$GOV_MASTER_MINTER" "$propose_data" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR") || {
  _assert_fail "flow failed"; test_result; exit 1
}

exec_status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$exec_status" "0x1" "removeMinter execute status == 0x1"

after=$(hex_to_dec "$(eth_call_raw 1 "$NATIVE_COIN_ADAPTER" "${is_minter_sel}${m_padded}" 2>/dev/null || echo "0x0")")
assert_eq "$after" "0" "TEST_ACC_D no longer a Minter"

test_result
