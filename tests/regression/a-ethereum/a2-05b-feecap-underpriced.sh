#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-2-05b
# name: GasFeeCap < MinBaseFee+MinTip tx 거부 (Anzeon 활성)
# category: regression/a-ethereum
# tags: [tx, anzeon]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/a-ethereum/a2-05b-feecap-underpriced
# RT-A-2-05b — GasFeeCap < MinBaseFee+MinTip tx 거부 (Anzeon 활성)
# Expected: ErrUnderpriced with message prefix "transaction underpriced: gas fee cap"
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-05b-feecap-underpriced"
check_env || { test_result; exit 1; }

# gasFeeCap = MinBaseFee - 1 (하한 미만) → Anzeon 활성 시 ErrUnderpriced
err_output=$(python3 <<'PYEOF' 2>&1 || true
import json, requests
from eth_account import Account

pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
to = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
url = "http://127.0.0.1:8501"

acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)

MIN_BASE_FEE = 20_000_000_000_000  # 20 Gwei (regression profile)
tx = {
    "nonce": nonce, "to": to, "value": 1, "gas": 21000, "chainId": chain_id,
    "maxFeePerGas": MIN_BASE_FEE - 1,   # 하한 미만 → ErrUnderpriced 기대
    # NOTE: maxPriorityFeePerGas must be <= maxFeePerGas (EIP-1559 invariant).
    # Previously set to 27.6e12 which is GREATER than maxFeePerGas(19.99e12),
    # causing the node to reject with "max priority fee higher than max fee"
    # before reaching the fee cap validation. Lower to 1 wei to trigger the
    # intended gas fee cap check path.
    "maxPriorityFeePerGas": 1,
    "type": 2,
}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json()
print(json.dumps(resp))
PYEOF
)

assert_contains "$err_output" "underpriced" "error contains 'underpriced'"
if [[ "$err_output" == *"gas fee cap"* ]]; then
  _assert_pass "error message prefix: 'transaction underpriced: gas fee cap'"
else
  _assert_fail "error message missing 'gas fee cap' prefix: $err_output"
fi

test_result
