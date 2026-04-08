#!/usr/bin/env bash
# Test: regression/c-anzeon/c-02-authorized-account-gastip-free
# RT-C-02 — Authorized 계정은 tipCap을 자유롭게 설정할 수 있음
#
# Pre-requisite: GovCouncil로 TEST_ACC_D를 authorize 해야 함 (E/F 카테고리에서 처리)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/c-anzeon/c-02-authorized-account-gastip-free"
check_env || { test_result; exit 1; }

# TEST_ACC_D가 Authorized 인지 확인
is_auth_sel=$(selector "isAuthorized(address)")
addr_padded=$(pad_address "$TEST_ACC_D_ADDR" | sed 's/^0x//')
call_data="${is_auth_sel}${addr_padded}"

is_auth_result=$(eth_call_raw "1" "$ACCOUNT_MANAGER" "$call_data")
is_auth=$(hex_to_dec "$is_auth_result")

if [[ "$is_auth" != "1" ]]; then
  printf '[SKIP] TEST_ACC_D is not authorized yet (run F-5-08 first)\n' >&2
  _assert_pass "authorized account test skipped (prerequisite: F-5-08)"
  test_result
  exit 0
fi

# TEST_ACC_D로 임의의 tip 사용
header_tip=$(rpc "1" "istanbul_getWbftExtraInfo" '["latest"]' | python3 -c "
import sys, json
r = json.load(sys.stdin).get('result', {})
print(int(r.get('gasTip', '0x0'), 16))
")
base_fee=$(get_base_fee "1")
custom_tip=$(( header_tip * 3 ))  # header보다 3배 큰 tip

tx_hash=$(python3 <<PYEOF
import json, requests
from eth_account import Account
pk = "${TEST_ACC_D_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "to": "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", "value": 1, "gas": 21000, "chainId": chain_id,
      "maxFeePerGas": base_fee * 2 + ${custom_tip},
      "maxPriorityFeePerGas": ${custom_tip},
      "type": 2}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1}).json()
print(resp.get("result", ""))
PYEOF
)

receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
effective_gp=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(int(json.load(sys.stdin).get('effectiveGasPrice', '0x0'), 16))")

# Authorized 계정: effectiveGasPrice = baseFee + custom_tip (강제 변환 없음)
tip_used=$(( effective_gp - base_fee ))
printf '[INFO]  effective tip=%s, custom_tip=%s, header_tip=%s\n' "$tip_used" "$custom_tip" "$header_tip" >&2

# 오차 허용 10%
tip_lower=$(( custom_tip * 90 / 100 ))
tip_upper=$(( custom_tip * 110 / 100 ))
in_range=$( [[ $tip_used -ge $tip_lower && $tip_used -le $tip_upper ]] && echo true || echo false )
assert_true "$in_range" "Authorized account tip ($tip_used) ≈ custom_tip ($custom_tip), not forced to header_tip"

# AuthorizedTxExecuted 이벤트 확인 (RT-E-09와 중첩)
log_found=$(find_log_by_topic "$receipt" "$ACCOUNT_MANAGER" "$AUTHORIZED_TX_EXECUTED_SIG")
assert_not_empty "$log_found" "AuthorizedTxExecuted event present in receipt.logs"

test_result
