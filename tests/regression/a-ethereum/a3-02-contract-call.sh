#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-3-02
# name: 컨트랙트 상태 변경 함수 호출 (set) + eth_call로 확인
# category: regression/a-ethereum
# tags: [contract]
# estimated_seconds: 35
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: [RT-A-3-01]
# ---end-meta---
# Test: regression/a-ethereum/a3-02-contract-call
# RT-A-3-02 — 컨트랙트 상태 변경 함수 호출 (set) + eth_call로 확인
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a3-02-contract-call"
check_env || { test_result; exit 1; }

# A-3-01에서 배포된 컨트랙트 주소 로드
contract_addr=$(cat /tmp/chainbench-regression/simple_storage.addr 2>/dev/null || echo "")
if [[ -z "$contract_addr" ]]; then
  _assert_fail "simple storage contract not deployed (run a3-01 first)"
  test_result
  exit 1
fi

# set(42) 호출
set_selector=$(selector "set(uint256)")
val_padded=$(pad_uint256 "42")
data="${set_selector}${val_padded:2}"  # 0x 제거 후 결합

tx_hash=$(python3 <<PYEOF
import json, requests
from eth_account import Account

pk = "${TEST_ACC_A_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)

tx = {"nonce": nonce, "to": "${contract_addr}", "value": 0, "gas": 100000, "chainId": chain_id,
      "data": "${data}",
      "maxFeePerGas": base_fee + 50_000_000_000_000,
      "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json()
print(resp.get("result", ""))
PYEOF
)

assert_contains "$tx_hash" "0x" "set(42) tx submitted"

receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status" "0x1" "set(42) receipt.status == 0x1"

# x()로 값 확인
get_selector=$(selector "x()")
result=$(eth_call_raw "1" "$contract_addr" "$get_selector")
val_dec=$(hex_to_dec "$result")
assert_eq "$val_dec" "42" "contract state x == 42 (set via tx, read via eth_call)"

test_result
