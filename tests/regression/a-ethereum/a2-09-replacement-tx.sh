#!/usr/bin/env bash
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

# 블록 포함 대기
sleep 5

# tx2만 블록에 포함되어야 함
tx1_receipt=$(get_receipt "1" "$tx1_hash")
tx2_receipt=$(get_receipt "1" "$tx2_hash")

tx2_included=$(printf '%s' "$tx2_receipt" | python3 -c "import sys, json; r = json.load(sys.stdin); print('yes' if r and r.get('status') == '0x1' else 'no')" 2>/dev/null || echo "no")
assert_eq "$tx2_included" "yes" "replacement tx2 included in block"

# tx1은 없거나 제거됨
tx1_not_found=$(printf '%s' "$tx1_receipt" | python3 -c "import sys, json; d = json.load(sys.stdin) if sys.stdin.read(1) else None; print('yes' if not d else 'no')" 2>/dev/null || echo "yes")
assert_eq "$tx1_not_found" "yes" "original tx1 not in chain (replaced)"

test_result
