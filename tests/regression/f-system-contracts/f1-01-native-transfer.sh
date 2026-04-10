#!/usr/bin/env bash
# Test: regression/f-system-contracts/f1-01-native-transfer
# RT-F-1-01 — NativeCoinAdapter.transfer → 기본 코인 전송과 동일
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/f-system-contracts/f1-01-native-transfer"
check_env || { test_result; exit 1; }

# transfer(address,uint256)
sel=$(selector "transfer(address,uint256)")
to_padded=$(pad_address "$TEST_ACC_B_ADDR" | sed 's/^0x//')
amount_padded=$(pad_uint256 "1000000000000000" | sed 's/^0x//')  # 0.001 ether
data="${sel}${to_padded}${amount_padded}"

before=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${TEST_ACC_B_ADDR}\", \"latest\"]" | json_get - result)")

# NativeCoinAdapter.transfer 호출 (TEST_ACC_A → TEST_ACC_B)
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
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json()
print(resp.get("result", ""))
PYEOF
)

receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status" "0x1" "transfer receipt.status == 0x1"

# 수신자 잔액 증가
after=$(hex_to_dec "$(rpc 1 eth_getBalance "[\"${TEST_ACC_B_ADDR}\", \"latest\"]" | json_get - result)")
diff=$(( after - before ))
assert_eq "$diff" "1000000000000000" "recipient balance increased by 0.001 ether"

# Transfer 이벤트 확인
log=$(find_log_by_topic "$receipt" "$NATIVE_COIN_ADAPTER" "$TRANSFER_EVENT_SIG")
assert_not_empty "$log" "Transfer event emitted by NativeCoinAdapter"

test_result
