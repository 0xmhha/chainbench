#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-2-09
# name: 동일 nonce, 더 높은 GasFeeCap으로 tx 교체
# category: regression/a-ethereum
# tags: [tx, replacement]
# estimated_seconds: 10
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
# Test: regression/a-ethereum/a2-09-replacement-tx
# RT-A-2-09 — 동일 nonce, 더 높은 GasFeeCap으로 tx 교체
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a2-09-replacement-tx"
check_env || { test_result; exit 1; }

result=$(python3 <<'PYEOF' 2>&1 || true
import json, requests, time
from eth_account import Account

pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
to = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
url = "http://127.0.0.1:8501"

acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)

# tx1: 낮은 fee
tx1 = {"nonce": nonce, "to": to, "value": 1, "gas": 21000, "chainId": chain_id,
       "maxFeePerGas": base_fee + 27_600_000_000_000,
       "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed1 = acct.sign_transaction(tx1)
r1 = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed1.raw_transaction.to_0x_hex()],"id":1}).json()

# tx2: 동일 nonce, 110% 이상 높은 fee
tx2 = {"nonce": nonce, "to": to, "value": 2, "gas": 21000, "chainId": chain_id,
       "maxFeePerGas": int((base_fee + 27_600_000_000_000) * 1.2),
       "maxPriorityFeePerGas": int(27_600_000_000_000 * 1.2), "type": 2}
signed2 = acct.sign_transaction(tx2)
r2 = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed2.raw_transaction.to_0x_hex()],"id":1}).json()

print(json.dumps({"tx1": r1.get("result", r1.get("error")), "tx2": r2.get("result", r2.get("error"))}))
PYEOF
)

tx1_hash=$(printf '%s' "$result" | python3 -c "import sys, json; print(json.load(sys.stdin).get('tx1', ''))")
tx2_hash=$(printf '%s' "$result" | python3 -c "import sys, json; print(json.load(sys.stdin).get('tx2', ''))")

assert_contains "$tx1_hash" "0x" "tx1 submitted"
assert_contains "$tx2_hash" "0x" "tx2 (replacement) submitted"

# tx2가 mine될 때까지 대기 (sleep 5는 timing-dependent로 run-all에서 flaky)
tx2_receipt=$(wait_tx_receipt_full "1" "$tx2_hash" 30 2>/dev/null || echo "")

tx2_included="no"
if [[ -n "$tx2_receipt" ]]; then
  tx2_status=$(printf '%s' "$tx2_receipt" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))" 2>/dev/null || echo "")
  [[ "$tx2_status" == "0x1" ]] && tx2_included="yes"
fi
assert_eq "$tx2_included" "yes" "replacement tx2 included in block"

# tx1은 교체되어 mine 안 됨 — receipt이 null이어야
tx1_check=$(rpc "1" "eth_getTransactionReceipt" "[\"${tx1_hash}\"]" | json_get - result)
tx1_not_found="yes"
[[ -n "$tx1_check" && "$tx1_check" != "null" ]] && tx1_not_found="no"
assert_eq "$tx1_not_found" "yes" "original tx1 not in chain (replaced)"

test_result
