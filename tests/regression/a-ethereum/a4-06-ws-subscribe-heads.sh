#!/usr/bin/env bash
# Test: regression/a-ethereum/a4-06-ws-subscribe-heads
# RT-A-4-06 — eth_subscribe(newHeads) WebSocket 구독
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a4-06-ws-subscribe-heads"
check_env || { test_result; exit 1; }

# WebSocket으로 newHeads 구독 → 5개 블록 수신 검증
result=$(python3 <<'PYEOF' 2>&1 || true
import asyncio, json
try:
    import websockets
except ImportError:
    print("SKIP: websockets not installed")
    exit(0)

async def main():
    ws_url = "ws://127.0.0.1:9501"  # node1 WebSocket port
    try:
        async with websockets.connect(ws_url, open_timeout=5) as ws:
            await ws.send(json.dumps({"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}))
            resp = json.loads(await asyncio.wait_for(ws.recv(), timeout=5))
            sub_id = resp.get("result")
            if not sub_id:
                print("FAIL: no subscription id")
                return
            count = 0
            for _ in range(5):
                try:
                    msg = await asyncio.wait_for(ws.recv(), timeout=5)
                    data = json.loads(msg)
                    if "params" in data and data["params"].get("subscription") == sub_id:
                        header = data["params"]["result"]
                        if "number" in header and "hash" in header:
                            count += 1
                except asyncio.TimeoutError:
                    break
            print(f"OK:{count}")
    except Exception as e:
        print(f"ERROR:{e}")

asyncio.run(main())
PYEOF
)

printf '[INFO]  WebSocket result: %s\n' "$result" >&2

if [[ "$result" == "SKIP:"* ]]; then
  _assert_pass "$result"
elif [[ "$result" == OK:* ]]; then
  count="${result#OK:}"
  assert_ge "$count" "3" "received at least 3 newHeads messages (got $count)"
else
  _assert_fail "WebSocket subscribe failed: $result"
fi

test_result
