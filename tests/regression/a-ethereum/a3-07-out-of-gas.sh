#!/usr/bin/env bash
# Test: regression/a-ethereum/a3-07-out-of-gas
# RT-A-3-07 — out-of-gas tx: gasUsed == gasLimit, 전량 소비, 환불 없음
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a3-07-out-of-gas"
check_env || { test_result; exit 1; }

contract_addr=$(cat /tmp/chainbench-regression/simple_storage.addr 2>/dev/null || echo "")
if [[ -z "$contract_addr" ]]; then
  _assert_fail "simple storage not deployed (run a3-01 first)"
  test_result
  exit 1
fi

# set(uint256) 실제 가스 비용보다 부족한 gasLimit 설정
# set() 은 보통 ~30000 가스. gasLimit을 22000으로 설정 → OOG
set_sel=$(selector "set(uint256)")
data="${set_sel}$(pad_uint256 999 | sed 's/^0x//')"
gas_limit=22000

tx_hash=$(python3 <<PYEOF
import json, requests
from eth_account import Account
pk = "${TEST_ACC_A_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)
nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
tx = {"nonce": nonce, "to": "${contract_addr}", "value": 0, "gas": ${gas_limit}, "chainId": chain_id,
      "data": "${data}",
      "maxFeePerGas": base_fee + 50_000_000_000_000,
      "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1}).json()
print(resp.get("result", ""))
PYEOF
)

assert_contains "$tx_hash" "0x" "OOG tx submitted"

receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
status=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', ''))")
assert_eq "$status" "0x0" "receipt.status == 0x0 (failed)"

# gasUsed == gasLimit (전량 소비)
gas_used=$(hex_to_dec "$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('gasUsed', ''))")")
printf '[INFO]  gasUsed=%s, gasLimit=%s\n' "$gas_used" "$gas_limit" >&2
assert_eq "$gas_used" "$gas_limit" "gasUsed == gasLimit (전량 소비, 환불 없음)"

test_result
