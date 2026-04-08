#!/usr/bin/env bash
# Test: regression/a-ethereum/a3-05-eth-call-revert
# RT-A-3-05 — eth_call로 revert하는 함수 호출 시 에러 반환
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a3-05-eth-call-revert"
check_env || { test_result; exit 1; }

# Reverter 컨트랙트: 항상 revert "BAD_INPUT"
# contract Reverter { function fail() public pure { revert("BAD_INPUT"); } }
REVERTER_BYTECODE="0x6080604052348015600f57600080fd5b5060a58061001e6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063a9cc471814602d575b600080fd5b60336035565b005b6040517f08c379a00000000000000000000000000000000000000000000000000000000081526004810182815260096024830152684241445f494e50555460b81b604483015281519192909182906064019081905290819003820190fd"

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
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1}).json()
print(resp.get("result", ""))
PYEOF
)

receipt=$(wait_tx_receipt_full "1" "$deploy_tx" 30)
reverter_addr=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('contractAddress', ''))")
assert_not_empty "$reverter_addr" "Reverter contract deployed"
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
