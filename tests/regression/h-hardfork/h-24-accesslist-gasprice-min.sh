#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-3-05
# name: AccessListTx with gasPrice below minimum is rejected
# category: regression/h-hardfork
# tags: [hardfork, boho, gas, accesslist, rejection]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# TC-1-3-05 — AccessListTx gasPrice < minBaseFee + minTip → rejected
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-24-accesslist-gasprice-min"
check_env || { test_result; exit 1; }

base_fee=$(get_base_fee 1)
gas_tip=$(get_header_gas_tip 1)
min_required=$(( base_fee + gas_tip ))
too_low=$(( min_required - 1 ))

observe "baseFee" "$base_fee"
observe "gasTip" "$gas_tip"
observe "minRequired" "$min_required"
observe "gasPrice_used" "$too_low"

nonce=$(get_nonce 1 "$TEST_ACC_A_ADDR")
result=$(python3 -c "
from eth_account import Account
from eth_utils import to_hex
import requests, json

acct = Account.from_key('${TEST_ACC_A_PK}')
tx = {
    'type': 1, 'chainId': 8283, 'nonce': ${nonce},
    'to': '${TEST_ACC_B_ADDR}', 'value': 1000000000000000,
    'gas': 21000, 'gasPrice': ${too_low}, 'accessList': [],
}
signed = acct.sign_transaction(tx)
payload = json.dumps({'jsonrpc':'2.0','method':'eth_sendRawTransaction','params':[to_hex(signed.raw_transaction)],'id':1})
r = requests.post('http://127.0.0.1:8501', data=payload, headers={'Content-Type':'application/json'}, timeout=5)
resp = r.json()
if 'error' in resp:
    print('ERROR:' + resp['error'].get('message',''))
else:
    print('OK:' + resp.get('result',''))
" 2>&1)

if [[ "$result" == ERROR:* ]]; then
  _assert_pass "AccessListTx with low gasPrice rejected: ${result#ERROR:}"
else
  _assert_fail "AccessListTx with low gasPrice was NOT rejected"
fi

test_result
