#!/usr/bin/env bash
# Test: regression/a-ethereum/a3-01-contract-deploy
# RT-A-3-01 — 컨트랙트 배포
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a3-01-contract-deploy"
check_env || { test_result; exit 1; }

# SimpleStorage 컨트랙트 bytecode (solc 0.8.30, --optimize --no-cbor-metadata)
# contract SimpleStorage { uint256 public x; function set(uint256 _x) public { x = _x; } }
# NOTE: 이전 BYTECODE는 IPFS 해시가 모두 0인 placeholder였으며, 실제 체인에서 배포 시
# "invalid jump destination" 에러로 revert되는 corrupt 상태였음. 정상 compiled bytecode로 교체.
BYTECODE="0x6080604052348015600e575f5ffd5b50607480601a5f395ff3fe6080604052348015600e575f5ffd5b50600436106030575f3560e01c80630c55699c14603457806360fe47b114604d575b5f5ffd5b603b5f5481565b60405190815260200160405180910390f35b605c6058366004605e565b5f55565b005b5f60208284031215606d575f5ffd5b503591905056"

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
resp = requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1}).json()
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

# 저장 (후속 테스트에서 참조용) — checksum 형식으로 정규화
# eth-account >=0.13 는 소문자 주소를 tx["to"]로 받지 않음
contract_addr=$(to_checksum "$contract_addr")
mkdir -p /tmp/chainbench-regression
echo "$contract_addr" > /tmp/chainbench-regression/simple_storage.addr

test_result
