#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-E-02
# name: 블랙리스트 계정이 Recipient인 tx 거부
# category: regression/e-blacklist-authorized
# tags: [blacklist, access-control]
# estimated_seconds: 5
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/e-blacklist-authorized/e-02-recipient-blacklisted
# RT-E-02 — 블랙리스트 계정이 Recipient인 tx 거부
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/e-blacklist-authorized/e-02-recipient-blacklisted"
check_env || { test_result; exit 1; }

# E-01이 선행되어 TEST_ACC_E가 blacklist 되어있다고 가정
target_padded=$(pad_address "$TEST_ACC_E_ADDR" | sed 's/^0x//')
is_bl_sel=$(selector "isBlacklisted(address)")
is_bl=$(hex_to_dec "$(eth_call_raw 1 "$ACCOUNT_MANAGER" "${is_bl_sel}${target_padded}")")

if [[ "$is_bl" != "1" ]]; then
  printf '[INFO]  TEST_ACC_E not blacklisted, running e-01 first is recommended\n' >&2
fi

# TEST_ACC_A가 TEST_ACC_E(blacklisted)로 송금 시도 → 거부
err_output=$(python3 <<'PYEOF' 2>&1 || true
import json, requests
from eth_account import Account
pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"  # TEST_ACC_A
to = "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"  # TEST_ACC_E (blacklisted)
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "to": to, "value": 1, "gas": 21000, "chainId": chain_id,
      "maxFeePerGas": base_fee + 50_000_000_000_000, "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json()
print(json.dumps(resp))
PYEOF
)

assert_contains "$err_output" "blacklist" "recipient blacklist error"

test_result
