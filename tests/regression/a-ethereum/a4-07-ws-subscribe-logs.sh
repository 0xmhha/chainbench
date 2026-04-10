#!/usr/bin/env bash
# Test: regression/a-ethereum/a4-07-ws-subscribe-logs
# RT-A-4-07 — eth_subscribe(logs) WebSocket 구독
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a4-07-ws-subscribe-logs"
check_env || { test_result; exit 1; }

# Background에서 WebSocket 구독 + 별도 tx로 Transfer 이벤트 발생 → 수신 확인
result=$(python3 <<'PYEOF' 2>&1 || true
import asyncio, json, time, threading, requests
try:
    import websockets
except ImportError:
    print("SKIP: websockets not installed")
    exit(0)

from eth_account import Account

NATIVE_COIN_ADAPTER = "0x0000000000000000000000000000000000001000"
TRANSFER_TOPIC = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

def trigger_tx():
    time.sleep(2)
    pk = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
    url = "http://127.0.0.1:8501"
    acct = Account.from_key(pk)
    nonce = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getTransactionCount","params":[acct.address, "pending"],"id":1}).json()["result"], 16)
    chain_id = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}).json()["result"], 16)
    base_fee = int(requests.post(url, json={"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).json()["result"]["baseFeePerGas"], 16)
    tx = {"nonce": nonce, "to": "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
          "value": 1000000000000000, "gas": 21000, "chainId": chain_id,
          "maxFeePerGas": base_fee + 50_000_000_000_000,
          "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
    signed = acct.sign_transaction(tx)
    requests.post(url, json={"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[signed.raw_transaction.to_0x_hex()],"id":1})

async def main():
    t = threading.Thread(target=trigger_tx)
    t.start()
    try:
        async with websockets.connect("ws://127.0.0.1:9501", open_timeout=5) as ws:
            sub_req = {"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["logs",{"address":NATIVE_COIN_ADAPTER,"topics":[TRANSFER_TOPIC]}]}
            await ws.send(json.dumps(sub_req))
            resp = json.loads(await asyncio.wait_for(ws.recv(), timeout=5))
            sub_id = resp.get("result")
            for _ in range(15):
                try:
                    msg = await asyncio.wait_for(ws.recv(), timeout=2)
                    data = json.loads(msg)
                    if "params" in data and data["params"].get("subscription") == sub_id:
                        log = data["params"]["result"]
                        if log.get("address", "").lower() == NATIVE_COIN_ADAPTER.lower():
                            print("OK:received_transfer_log")
                            return
                except asyncio.TimeoutError:
                    continue
            print("FAIL:no_log_received")
    except Exception as e:
        print(f"ERROR:{e}")
    t.join()

asyncio.run(main())
PYEOF
)

printf '[INFO]  result: %s\n' "$result" >&2

if [[ "$result" == "SKIP:"* ]]; then
  _assert_pass "$result"
elif [[ "$result" == OK:* ]]; then
  _assert_pass "received Transfer log via WebSocket subscription"
else
  _assert_fail "WebSocket logs subscribe failed: $result"
fi

test_result
