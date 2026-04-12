#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-3-02
# name: DynamicFeeTx with exact minimum gasFeeCap is accepted
# category: regression/h-hardfork
# tags: [hardfork, boho, gas, feecap, boundary]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-1-3-02 — gasFeeCap == minBaseFee + minTip → accepted
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-21-gas-feecap-exact-min"
check_env || { test_result; exit 1; }

base_fee=$(get_base_fee 1)
gas_tip=$(get_header_gas_tip 1)
exact_min=$(( base_fee + gas_tip ))

observe "baseFee" "$base_fee"
observe "gasTip" "$gas_tip"
observe "gasFeeCap_exact" "$exact_min"

nonce=$(get_nonce 1 "$TEST_ACC_A_ADDR")
result=$(python3 -c "
from eth_account import Account
from eth_utils import to_hex
import requests, json

key = '${TEST_ACC_A_PK}'
acct = Account.from_key(key)
tx = {
    'type': 2,
    'chainId': 8283,
    'nonce': ${nonce},
    'to': '${TEST_ACC_B_ADDR}',
    'value': 1000000000000000,
    'gas': 21000,
    'maxFeePerGas': ${exact_min},
    'maxPriorityFeePerGas': ${gas_tip},
}
signed = acct.sign_transaction(tx)
raw = to_hex(signed.raw_transaction)
payload = json.dumps({'jsonrpc':'2.0','method':'eth_sendRawTransaction','params':[raw],'id':1})
r = requests.post('http://127.0.0.1:8501', data=payload, headers={'Content-Type':'application/json'}, timeout=5)
resp = r.json()
if 'error' in resp:
    print('ERROR:' + resp['error'].get('message',''))
else:
    print(resp.get('result',''))
" 2>&1)

observe "send_result" "$result"

if [[ "$result" == ERROR:* ]]; then
  _assert_fail "TX with exact min gasFeeCap was rejected: ${result#ERROR:}"
else
  assert_not_empty "$result" "TX hash returned"
  # Verify receipt
  receipt=$(wait_tx_receipt_full 1 "$result" 30)
  status=$(json_get "$receipt" "status")
  assert_eq "$status" "0x1" "TX with exact min gasFeeCap succeeded"
fi

test_result
