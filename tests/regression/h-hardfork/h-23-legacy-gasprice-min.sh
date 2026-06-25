#!/usr/bin/env bash
# ---chainbench-meta---
# id: TC-1-3-04
# name: Legacy TX with gasPrice below minimum is rejected
# category: regression/h-hardfork
# tags: [hardfork, boho, gas, legacy, rejection]
# estimated_seconds: 20
# preconditions:
#   chain_running: true
# depends_on: []
# ---end-meta---
# TC-1-3-04 — Legacy TX gasPrice < minBaseFee + minTip → rejected
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/h-hardfork/h-23-legacy-gasprice-min"
check_env || { test_result; exit 1; }
ensure_nodes_running

base_fee=$(get_base_fee "$(node 1)")
gas_tip=$(get_header_gas_tip "$(node 1)")
min_required=$(( base_fee + gas_tip ))
too_low=$(( min_required - 1 ))

observe "minRequired" "$min_required"
observe "gasPrice_used" "$too_low"

nonce=$(get_nonce "$(node 1)" "$(acct_addr 1)")
result=$(python3 -c "
from eth_account import Account
from eth_utils import to_hex
import requests, json

key = '${TEST_ACC_A_PK}'
acct = Account.from_key(key)
tx = {
    'type': 0,
    'chainId': 8283,
    'nonce': ${nonce},
    'to': '$(acct_addr 2)',
    'value': 1000000000000000,
    'gas': 21000,
    'gasPrice': ${too_low},
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
  _assert_pass "Legacy TX with low gasPrice rejected: ${result#ERROR:}"
else
  _assert_fail "Legacy TX with low gasPrice was NOT rejected"
fi

test_result
