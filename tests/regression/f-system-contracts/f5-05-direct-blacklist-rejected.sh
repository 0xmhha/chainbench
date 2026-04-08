#!/usr/bin/env bash
# Test: regression/f-system-contracts/f5-05-direct-blacklist-rejected
# RT-F-5-05 — 비멤버가 AccountManager.blacklist 직접 호출 → revert
set -euo pipefail
source "$(dirname "$0")/../lib/common.sh"
test_start "regression/f-system-contracts/f5-05-direct-blacklist-rejected"
check_env || { test_result; exit 1; }

# AccountManager.blacklist(address) selector
bl_sel=$(selector "blacklist(address)")
target_padded=$(pad_address "$TEST_ACC_B_ADDR" | sed 's/^0x//')
data="${bl_sel}${target_padded}"

# TEST_ACC_A (비멤버)가 AccountManager.blacklist 직접 호출 시도 → revert
tx_hash=$(python3 <<PYEOF
import json, requests
from eth_account import Account
pk = "${TEST_ACC_A_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "to": "${ACCOUNT_MANAGER}", "value": 0, "gas": 200000, "chainId": chain_id,
      "data": "${data}",
      "maxFeePerGas": base_fee + 50_000_000_000_000, "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
print(requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1}).json().get("result", ""))
PYEOF
)

if [[ -z "$tx_hash" || "$tx_hash" == "null" ]]; then
  _assert_pass "tx rejected at submission (access control)"
else
  receipt=$(wait_tx_receipt_full "1" "$tx_hash" 20)
  status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
  assert_eq "$status" "0x0" "direct blacklist call reverted"
fi

test_result
