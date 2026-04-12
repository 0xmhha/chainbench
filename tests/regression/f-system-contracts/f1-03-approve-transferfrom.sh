#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-F-1-03
# name: approve / transferFrom 정상 동작
# category: regression/f-system-contracts
# tags: [governance, transfer]
# estimated_seconds: 65
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/f-system-contracts/f1-03-approve-transferfrom
# RT-F-1-03 (v2) — approve / transferFrom 정상 동작
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f1-03-approve-transferfrom"
check_env || { test_result; exit 1; }

# TEST_ACC_A가 TEST_ACC_D에 5 ether approve
approve_sel=$(selector "approve(address,uint256)")
spender_padded=$(pad_address "$TEST_ACC_D_ADDR" | sed 's/^0x//')
allowance_padded=$(pad_uint256 "5000000000000000000" | sed 's/^0x//')  # 5 ether
approve_data="${approve_sel}${spender_padded}${allowance_padded}"

tx1=$(python3 <<PYEOF
import json, requests
from eth_account import Account
pk = "${TEST_ACC_A_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "to": "${NATIVE_COIN_ADAPTER}", "value": 0, "gas": 150000, "chainId": chain_id,
      "data": "${approve_data}",
      "maxFeePerGas": base_fee + 50_000_000_000_000, "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
print(requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json().get("result", ""))
PYEOF
)
wait_receipt "1" "$tx1" 30 >/dev/null
assert_contains "$tx1" "0x" "approve tx submitted"

# allowance 확인
allow_sel=$(selector "allowance(address,address)")
owner_padded=$(pad_address "$TEST_ACC_A_ADDR" | sed 's/^0x//')
allow_result=$(hex_to_dec "$(eth_call_raw 1 "$NATIVE_COIN_ADAPTER" "${allow_sel}${owner_padded}${spender_padded}")")
assert_eq "$allow_result" "5000000000000000000" "allowance == 5 ether"

# TEST_ACC_D가 transferFrom 호출 (A → B, 3 ether)
from_sel=$(selector "transferFrom(address,address,uint256)")
to_padded=$(pad_address "$TEST_ACC_B_ADDR" | sed 's/^0x//')
amount_padded=$(pad_uint256 "3000000000000000000" | sed 's/^0x//')  # 3 ether
tf_data="${from_sel}${owner_padded}${to_padded}${amount_padded}"

bal_b_before=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${TEST_ACC_B_ADDR}\", \"latest\"]" | json_get - result)")

tx2=$(python3 <<PYEOF
import json, requests
from eth_account import Account
pk = "${TEST_ACC_D_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "to": "${NATIVE_COIN_ADAPTER}", "value": 0, "gas": 200000, "chainId": chain_id,
      "data": "${tf_data}",
      "maxFeePerGas": base_fee + 50_000_000_000_000, "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
print(requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json().get("result", ""))
PYEOF
)
wait_receipt "1" "$tx2" 30 >/dev/null

# B 잔액 +3 ether
bal_b_after=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${TEST_ACC_B_ADDR}\", \"latest\"]" | json_get - result)")
diff=$(( bal_b_after - bal_b_before ))
assert_eq "$diff" "3000000000000000000" "TEST_ACC_B gained 3 ether via transferFrom"

# allowance == 2 ether
allow_after=$(hex_to_dec "$(eth_call_raw 1 "$NATIVE_COIN_ADAPTER" "${allow_sel}${owner_padded}${spender_padded}")")
assert_eq "$allow_after" "2000000000000000000" "allowance decremented to 2 ether"

test_result
