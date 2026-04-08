#!/usr/bin/env bash
# RT-G-5-03 — eth_call NativeCoinAdapter.allowance
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/g-api/g5-03-allowance"
check_env || { test_result; exit 1; }

# approve 먼저
approve_sel=$(selector "approve(address,uint256)")
spender_padded=$(pad_address "$TEST_ACC_D_ADDR" | sed 's/^0x//')
amount_padded=$(pad_uint256 "7000000000000000000" | sed 's/^0x//')
data="${approve_sel}${spender_padded}${amount_padded}"

tx_hash=$(python3 <<PYEOF
import json, requests
from eth_account import Account
pk = "${TEST_ACC_A_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "to": "${NATIVE_COIN_ADAPTER}", "value": 0, "gas": 150000, "chainId": chain_id,
      "data": "${data}",
      "maxFeePerGas": base_fee + 50_000_000_000_000, "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
print(requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1}).json().get("result", ""))
PYEOF
)
wait_receipt "1" "$tx_hash" 30 >/dev/null

# allowance 확인
allow_sel=$(selector "allowance(address,address)")
owner_padded=$(pad_address "$TEST_ACC_A_ADDR" | sed 's/^0x//')
result=$(hex_to_dec "$(eth_call_raw 1 "$NATIVE_COIN_ADAPTER" "${allow_sel}${owner_padded}${spender_padded}")")
assert_eq "$result" "7000000000000000000" "allowance == 7 ether"

test_result
