#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-3-05
# name: eth_call로 revert하는 함수 호출 시 에러 반환
# category: regression/a-ethereum
# tags: [contract, revert, eth_call]
# estimated_seconds: 35
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: [RT-A-3-05]
# ---end-meta---
# Test: regression/a-ethereum/a3-05-eth-call-revert
# RT-A-3-05 — eth_call로 revert하는 함수 호출 시 에러 반환
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a3-05-eth-call-revert"
check_env || { test_result; exit 1; }

# Reverter 컨트랙트: 항상 revert "BAD_INPUT"
# contract Reverter { function fail() public pure { revert("BAD_INPUT"); } }
# solc 0.8.30, --optimize --no-cbor-metadata (이전 placeholder bytecode 교체)
REVERTER_BYTECODE="0x6080604052348015600e575f5ffd5b50606a80601a5f395ff3fe6080604052348015600e575f5ffd5b50600436106026575f3560e01c8063a9cc471814602a575b5f5ffd5b60306032565b005b60405162461bcd60e51b815260206004820152600960248201526810905117d25394155560ba1b604482015260640160405180910390fd"

# Deploy
deploy_tx=$(python3 <<PYEOF
import json, requests
from eth_account import Account
pk = "${TEST_ACC_A_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "value": 0, "gas": 300000, "chainId": chain_id,
      "data": "${REVERTER_BYTECODE}",
      "maxFeePerGas": base_fee + 50_000_000_000_000,
      "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json()
print(resp.get("result", ""))
PYEOF
)

receipt=$(wait_tx_receipt_full "1" "$deploy_tx" 30)
reverter_addr=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('contractAddress', ''))")
assert_not_empty "$reverter_addr" "Reverter contract deployed"
# checksum 형식으로 정규화 (eth-account >=0.13 요구)
reverter_addr=$(to_checksum "$reverter_addr")
echo "$reverter_addr" > /tmp/chainbench-regression/reverter.addr

# fail() 호출 → revert
fail_selector=$(selector "fail()")
call_resp=$(rpc "1" "eth_call" "[{\"to\":\"${reverter_addr}\",\"data\":\"${fail_selector}\"}, \"latest\"]")

# 에러 응답 확인
has_error=$(printf '%s' "$call_resp" | python3 -c "import sys, json; print('yes' if 'error' in json.load(sys.stdin) else 'no')")
assert_eq "$has_error" "yes" "eth_call returns error for revert"

error_msg=$(printf '%s' "$call_resp" | python3 -c "import sys, json; print(json.load(sys.stdin).get('error', {}).get('message', ''))")
assert_contains "$error_msg" "execution reverted" "error message contains 'execution reverted'"

test_result
