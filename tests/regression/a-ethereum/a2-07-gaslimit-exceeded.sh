#!/usr/bin/env bash
# Test: regression/a-ethereum/a2-07-gaslimit-exceeded
# RT-A-2-07 — Gas Limit 초과 tx 거부 (블록 gas limit 초과)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-07-gaslimit-exceeded"
check_env || { test_result; exit 1; }

# 블록 gasLimit 조회
block_gas_limit_hex=$(rpc "1" "eth_getBlockByNumber" "[\"latest\", false]" | json_get - "result.gasLimit")
block_gas_limit=$(hex_to_dec "$block_gas_limit_hex")
printf '[INFO]  block gasLimit = %s\n' "$block_gas_limit" >&2

# tx.gasLimit = block.gasLimit + 1 → 거부
excess_gas=$(( block_gas_limit + 1 ))

err_output=$(python3 <<PYEOF 2>&1 || true
import json, requests
from eth_account import Account

pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
to = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
url = "http://127.0.0.1:8501"

acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)

tx = {
    "nonce": nonce, "to": to, "value": 1,
    "gas": ${excess_gas},  # 블록 한도 초과
    "chainId": chain_id,
    "maxFeePerGas": base_fee + 50_000_000_000_000,
    "maxPriorityFeePerGas": 27_600_000_000_000,
    "type": 2,
}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json()
print(json.dumps(resp))
PYEOF
)

# "exceeds block gas limit" 에러 확인
if [[ "$err_output" == *"exceeds block gas limit"* ]] || [[ "$err_output" == *"gas limit"* ]]; then
  _assert_pass "error contains 'gas limit' violation"
else
  _assert_fail "error does not contain gas limit violation: $err_output"
fi

test_result
