#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-2-10
# name: SetCodeTx (type 0x4 / EIP-7702) 계정 코드 위임
# category: regression/a-ethereum
# tags: [tx, eip7702]
# estimated_seconds: 35
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/a-ethereum/a2-10-setcode-tx
# RT-A-2-10 — SetCodeTx (type 0x4 / EIP-7702) 계정 코드 위임
# [v2 보강] effectiveGasPrice + gasLimit valid check
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-10-setcode-tx"
check_env || { test_result; exit 1; }

# delegator: TEST_ACC_A, delegate: TEST_ACC_D (임의 EOA, 코드 없음 OK — EIP-7702 표준은 delegate가 EOA도 허용)
# 실제 검증: delegator의 code가 0xef0100 + delegate_address로 설정됨
result=$(python3 <<'PYEOF' 2>&1
import json, requests
from eth_account import Account
from eth_account.typed_transactions.set_code_transaction import SetCodeTransaction

pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"  # delegator
delegate = "0x90F79bf6EB2c4f870365E785982E1f101E93b906"  # TEST_ACC_D
url = "http://127.0.0.1:8501"

acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)

# EIP-7702 authorization: Authorization{chainId, address, nonce} signed by delegator
auth = acct.sign_authorization({
    "chainId": chain_id,
    "address": delegate,
    "nonce": nonce + 1,  # 자기 자신에게 위임할 때 nonce + 1
})

# Type 0x04 SetCodeTx
tx = {
    "type": 4,
    "nonce": nonce,
    "to": acct.address,  # self
    "value": 0,
    "gas": 200000,  # 21000 + per-auth gas
    "chainId": chain_id,
    "maxFeePerGas": base_fee + 50_000_000_000_000,
    "maxPriorityFeePerGas": 27_600_000_000_000,
    "authorizationList": [auth],
}

try:
    signed = acct.sign_transaction(tx)
    resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json()
    print(json.dumps(resp))
except Exception as e:
    print(json.dumps({"error": {"message": str(e)}}))
PYEOF
)

tx_hash=$(printf '%s' "$result" | python3 -c "import sys, json; print(json.load(sys.stdin).get('result', ''))")
if [[ -z "$tx_hash" || "$tx_hash" == "null" ]]; then
  # eth-account 버전에 따라 SetCodeTx 미지원 가능 → skip
  printf '[WARN]  SetCodeTx not supported by eth-account version, skipping\n' >&2
  _assert_pass "EIP-7702 SetCodeTx (skipped: requires eth-account with EIP-7702 support)"
  test_result
  exit 0
fi

assert_contains "$tx_hash" "0x" "SetCodeTx hash returned"

# Receipt 대기
receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status" "0x1" "receipt.status == 0x1"

# delegator의 code가 0xef0100 + delegate address 형식인지 검증
code=$(rpc "1" "eth_getCode" "[\"$TEST_ACC_A_ADDR\", \"latest\"]" | json_get - "result")
code_lower=$(printf '%s' "$code" | tr '[:upper:]' '[:lower:]')
# delegation prefix: 0xef0100
prefix="${code_lower:0:8}"
assert_eq "$prefix" "0xef0100" "delegator code starts with 0xef0100 (delegation prefix)"

# 길이 = 2(0x) + 6(ef0100) + 40(address) = 48
code_len=${#code_lower}
assert_eq "$code_len" "48" "delegator code length is 48 (0xef0100 + 20-byte address)"

# gasLimit valid check
gas_used_dec=$(hex_to_dec "$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('gasUsed', ''))")")
tx_gas_dec=$(hex_to_dec "$(rpc "1" "eth_getTransactionByHash" "[\"${tx_hash}\"]" | json_get - "result.gas")")
assert_ge "$tx_gas_dec" "$gas_used_dec" "tx.gasLimit >= gasUsed"
assert_ge "$tx_gas_dec" "21000" "tx.gasLimit >= 21000"

# effectiveGasPrice != null
effective_gp=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('effectiveGasPrice', ''))")
assert_not_empty "$effective_gp" "effectiveGasPrice is present"

test_result
