#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-3-01
# name: DynamicFeeTx with gasFeeCap below minimum is rejected
# category: regression/h-hardfork
# tags: [hardfork, boho, gas, feecap, rejection]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-1-3-01 — gasFeeCap < minBaseFee + minTip → rejected
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-20-gas-feecap-below-min"
check_env || { test_result; exit 1; }

base_fee=$(get_base_fee 1)
gas_tip=$(get_header_gas_tip 1)
min_required=$(( base_fee + gas_tip ))
too_low=$(( min_required - 1 ))

observe "baseFee" "$base_fee"
observe "gasTip" "$gas_tip"
observe "minRequired" "$min_required"
observe "gasFeeCap_used" "$too_low"

# Try to send with gasFeeCap below minimum
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
    'value': 1000000000000000000,
    'gas': 21000,
    'maxFeePerGas': ${too_low},
    'maxPriorityFeePerGas': 0,
}
signed = acct.sign_transaction(tx)
raw = to_hex(signed.raw_transaction)
payload = json.dumps({'jsonrpc':'2.0','method':'eth_sendRawTransaction','params':[raw],'id':1})
r = requests.post('http://127.0.0.1:8501', data=payload, headers={'Content-Type':'application/json'}, timeout=5)
resp = r.json()
if 'error' in resp:
    print('ERROR:' + resp['error'].get('message',''))
else:
    print('OK:' + resp.get('result',''))
" 2>&1)

observe "send_result" "$result"

if [[ "$result" == ERROR:* ]]; then
  _assert_pass "TX with low gasFeeCap rejected: ${result#ERROR:}"
else
  _assert_fail "TX with low gasFeeCap was NOT rejected: $result"
fi

test_result
