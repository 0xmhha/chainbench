#!/usr/bin/env bash
# Test: regression/f-system-contracts/f4-01-add-minter
# RT-F-4-01 — GovMasterMinter.proposeConfigureMinter(address, uint256) → 승인 → execute
#
# ABI (v1 GovMasterMinter.sol:118):
#   function proposeConfigureMinter(address minter, uint256 minterAllowedAmount)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f4-01-add-minter"
check_env || { test_result; exit 1; }

unlock_all_validators

minter_addr="$TEST_ACC_D_ADDR"
allowance="10000000000000000000"  # 10 ether

propose_sel=$(selector "proposeConfigureMinter(address,uint256)")
m_padded=$(pad_address "$minter_addr" | sed 's/^0x//')
allow_padded=$(pad_uint256 "$allowance" | sed 's/^0x//')
propose_data="${propose_sel}${m_padded}${allow_padded}"

receipt=$(gov_full_flow "$GOV_MASTER_MINTER" "$propose_data" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR") || {
  _assert_fail "gov flow failed"; test_result; exit 1
}

exec_status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$exec_status" "0x1" "configureMinter execute status == 0x1"

# NativeCoinAdapter.isMinter 확인 (GovMinter가 아니라 NativeCoinAdapter 자체 Minter role)
# GovMasterMinter는 NativeCoinAdapter의 masterMinter로 설정되어 있음 → NativeCoinAdapter 상에 Minter 등록
is_minter_sel=$(selector "isMinter(address)")
is_minter_result=$(eth_call_raw 1 "$NATIVE_COIN_ADAPTER" "${is_minter_sel}${m_padded}" 2>/dev/null || echo "0x0")
is_minter=$(hex_to_dec "$is_minter_result")

if [[ "$is_minter" == "1" ]]; then
  _assert_pass "TEST_ACC_D registered as NativeCoinAdapter Minter"
else
  # minterAllowance로도 확인
  allow_sel=$(selector "minterAllowance(address)")
  allow_result=$(hex_to_dec "$(eth_call_raw 1 "$NATIVE_COIN_ADAPTER" "${allow_sel}${m_padded}" 2>/dev/null || echo "0x0")")
  assert_gt "$allow_result" "0" "TEST_ACC_D has non-zero minterAllowance ($allow_result)"
fi

test_result
