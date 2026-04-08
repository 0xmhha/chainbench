#!/usr/bin/env bash
# Test: regression/a-ethereum/a1-05-p2p-peers
# RT-A-1-05 — P2P 피어 연결 확인
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"

test_start "regression/a-ethereum/a1-05-p2p-peers"

# 각 노드의 peerCount ≥ 1 확인
for node in 1 2 3 4 5; do
  count=$(peer_count "$node" 2>/dev/null || echo "0")
  printf '[INFO]  node%s peerCount=%s\n' "$node" "$count" >&2
  assert_ge "$count" "1" "node${node} has at least 1 peer"
done

# admin_peers 응답에서 상대 노드 enode가 표시되는지 확인 (node1 기준)
peers_json=$(rpc "1" "admin_peers" "[]")
peer_count_in_response=$(printf '%s' "$peers_json" | python3 -c "
import sys, json
data = json.load(sys.stdin)
result = data.get('result', [])
print(len(result) if isinstance(result, list) else 0)
")
assert_ge "$peer_count_in_response" "1" "admin_peers returns at least 1 peer for node1"

# 각 peer 객체가 id/network 필드를 포함하는지 확인
has_id=$(printf '%s' "$peers_json" | python3 -c "
import sys, json
data = json.load(sys.stdin)
result = data.get('result', [])
if result and isinstance(result, list):
    p = result[0]
    print('yes' if p.get('id') else 'no')
else:
    print('no')
")
assert_eq "$has_id" "yes" "peer object contains id field"

test_result
