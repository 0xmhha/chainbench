#!/usr/bin/env bash
# Test: regression/a-ethereum/a3-01-contract-deploy
# RT-A-3-01 — 컨트랙트 배포
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a3-01-contract-deploy"
check_env || { test_result; exit 1; }

# 단순 storage 컨트랙트 bytecode (SimpleStorage.sol compiled)
# contract SimpleStorage { uint256 public x; function set(uint256 _x) public { x = _x; } }
BYTECODE="0x608060405234801561001057600080fd5b50610150806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80630c55699c1461003b57806360fe47b114610059575b600080fd5b610043610075565b60405161005091906100a1565b60405180910390f35b610073600480360381019061006e91906100ed565b61007b565b005b60005481565b8060008190555050565b6000819050919050565b61009b81610088565b82525050565b60006020820190506100b66000830184610092565b92915050565b600080fd5b6100ca81610088565b81146100d557600080fd5b50565b6000813590506100e7816100c1565b92915050565b600060208284031215610103576101026100bc565b5b6000610111848285016100d8565b9150509291505056fea2646970667358221220000000000000000000000000000000000000000000000000000000000000000064736f6c63430008120033"

# TEST_ACC_A로 배포 (to=null)
tx_hash=$(python3 <<PYEOF
import json, requests
from eth_account import Account

pk = "${TEST_ACC_A_PK}"
url = "http://127.0.0.1:8501"
acct = Account.from_key(pk)

nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)

tx = {
    "nonce": nonce, "value": 0, "gas": 500000, "chainId": chain_id,
    "data": "${BYTECODE}",
    "maxFeePerGas": base_fee + 50_000_000_000_000,
    "maxPriorityFeePerGas": 27_600_000_000_000,
    "type": 2,
}
signed = acct.sign_transaction(tx)
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.rawTransaction.hex()],"id":1}).json()
print(resp.get("result", ""))
PYEOF
)

assert_contains "$tx_hash" "0x" "deploy tx hash returned"

receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
assert_not_empty "$receipt" "receipt retrieved"

# contractAddress 확인
contract_addr=$(printf '%s' "$receipt" | python3 -c "import sys, json; print(json.load(sys.stdin).get('contractAddress', ''))")
assert_not_empty "$contract_addr" "receipt.contractAddress is set"
assert_contains "$contract_addr" "0x" "contractAddress is hex"
printf '[INFO]  deployed at %s\n' "$contract_addr" >&2

# eth_getCode로 배포 확인 (runtime code, creation code와 다름)
code=$(rpc "1" "eth_getCode" "[\"$contract_addr\", \"latest\"]" | json_get - "result")
code_len=${#code}
assert_gt "$code_len" "10" "eth_getCode returns non-empty bytecode (len=$code_len)"

# 저장 (후속 테스트에서 참조용)
mkdir -p /tmp/chainbench-regression
echo "$contract_addr" > /tmp/chainbench-regression/simple_storage.addr

test_result
