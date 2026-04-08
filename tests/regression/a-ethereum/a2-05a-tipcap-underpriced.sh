#!/usr/bin/env bash
# Test: regression/a-ethereum/a2-05a-tipcap-underpriced
# RT-A-2-05a — GasTipCap < MinTip tx 거부 (txpool 하한 검증)
# Expected: ErrUnderpriced with message prefix "transaction underpriced: gas tip cap"
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-05a-tipcap-underpriced"
check_env || { test_result; exit 1; }

# 매우 낮은 tipCap (1 wei) 전송 → ErrUnderpriced 예상
err_output=$(python3 <<'PYEOF' 2>&1 || true
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
    "nonce": nonce, "to": to, "value": 1, "gas": 21000, "chainId": chain_id,
    "maxFeePerGas": base_fee * 2,
    "maxPriorityFeePerGas": 1,  # 매우 낮은 tip
    "type": 2,
}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1}).json()
print(json.dumps(resp))
PYEOF
)

# 에러 메시지에 "underpriced" 또는 "gas tip cap" 포함 확인
assert_contains "$err_output" "underpriced" "error contains 'underpriced'"
# 메시지 prefix: "transaction underpriced: gas tip cap"
if [[ "$err_output" == *"gas tip cap"* ]]; then
  _assert_pass "error message prefix: 'transaction underpriced: gas tip cap'"
else
  _assert_fail "error message missing 'gas tip cap' prefix: $err_output"
fi

test_result
