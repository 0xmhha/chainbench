#!/usr/bin/env bash
# Test: regression/e-blacklist-authorized/e-05-zero-address
# RT-E-05 — Zero Address로 코인 전송 차단 (ErrZeroAddressTransfer)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/e-blacklist-authorized/e-05-zero-address"
check_env || { test_result; exit 1; }

# TEST_ACC_A → 0x000...000 value=1 → ErrZeroAddressTransfer
err_output=$(python3 <<'PYEOF' 2>&1 || true
import json, requests
from eth_account import Account
pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "to": "0x0000000000000000000000000000000000000000", "value": 1, "gas": 100000, "chainId": chain_id,
      "maxFeePerGas": base_fee + 50_000_000_000_000, "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json()
print(json.dumps(resp))
PYEOF
)

# 에러 또는 receipt.status == 0 (EVM 단계 거부)
if [[ "$err_output" == *"zero address"* ]] || [[ "$err_output" == *"ZeroAddress"* ]]; then
  _assert_pass "ErrZeroAddressTransfer at submission"
else
  # tx가 들어갔다면 receipt.status == 0 확인
  tx_hash=$(printf '%s' "$err_output" | python3 -c "
import sys, json
try:
    d = json.loads(sys.stdin.read())
    print(d.get('result', ''))
except Exception:
    print('')
")
  if [[ -n "$tx_hash" && "$tx_hash" != "null" ]]; then
    status=$(wait_receipt "1" "$tx_hash" 20)
    assert_eq "$status" "failed" "zero address tx fails at EVM level"
  else
    _assert_fail "unexpected: $err_output"
  fi
fi

test_result
