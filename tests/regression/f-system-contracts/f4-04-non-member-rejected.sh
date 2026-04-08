#!/usr/bin/env bash
# Test: regression/f-system-contracts/f4-04-non-member-rejected
# RT-F-4-04 — 비멤버 계정의 GovMasterMinter.proposeConfigureMinter 호출 거부
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f4-04-non-member-rejected"
check_env || { test_result; exit 1; }

# TEST_ACC_A (비멤버)로 proposeConfigureMinter 호출
propose_sel=$(selector "proposeConfigureMinter(address,uint256)")
m_padded=$(pad_address "$TEST_ACC_B_ADDR" | sed 's/^0x//')
allow_padded=$(pad_uint256 "1000000000000000000" | sed 's/^0x//')
propose_data="${propose_sel}${m_padded}${allow_padded}"

tx_hash=$(python3 <<PYEOF
import json, requests
from eth_account import Account
pk = "${TEST_ACC_A_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "to": "${GOV_MASTER_MINTER}", "value": 0, "gas": 500000, "chainId": chain_id,
      "data": "${propose_data}",
      "maxFeePerGas": base_fee + 50_000_000_000_000, "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
print(requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1}).json().get("result", ""))
PYEOF
)

receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status" "0x0" "non-member proposeConfigureMinter reverted (status==0x0)"

test_result
