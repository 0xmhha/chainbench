#!/usr/bin/env bash
# Test: regression/a-ethereum/a2-06-insufficient-funds
# RT-A-2-06 — 잔액 부족 tx 거부
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-06-insufficient-funds"
check_env || { test_result; exit 1; }

# TEST_ACC_B는 100 ether만 있음. 1000 ether 송금 시도 → 거부
err_output=$(python3 <<'PYEOF' 2>&1 || true
import json, requests
from eth_account import Account

pk = "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"  # TEST_ACC_B
to = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"  # TEST_ACC_A
url = "http://127.0.0.1:8501"

acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)

tx = {
    "nonce": nonce, "to": to, "value": 1000 * 10**18,  # 1000 ether (잔액 초과)
    "gas": 21000, "chainId": chain_id,
    "maxFeePerGas": base_fee + 50_000_000_000_000,
    "maxPriorityFeePerGas": 27_600_000_000_000,
    "type": 2,
}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json()
print(json.dumps(resp))
PYEOF
)

# "insufficient funds" 에러 확인
assert_contains "$err_output" "insufficient funds" "error contains 'insufficient funds'"

test_result
