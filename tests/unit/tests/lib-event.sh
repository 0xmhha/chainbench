#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"
export CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
source "${CHAINBENCH_DIR}/tests/lib/event.sh"

describe "event: cb_topic_hash computes Transfer topic correctly"
topic=$(cb_topic_hash "Transfer(address,address,uint256)")
assert_eq "$topic" \
  "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" \
  "Transfer topic"

describe "event: cb_find_log finds matching log"
mock_receipt='{"logs":[{"address":"0x1000","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"],"data":"0x01"}]}'
result=$(cb_find_log "$mock_receipt" "0x1000" \
  "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
assert_nonempty "$result" "found matching log"

describe "event: cb_find_log returns empty for non-matching address"
result=$(cb_find_log "$mock_receipt" "0x2000" \
  "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
assert_eq "$result" "" "no match returns empty"

describe "event: cb_count_logs counts correctly"
mock_receipt2='{"logs":[{"address":"0x1000","topics":["0xaaa"],"data":"0x"},{"address":"0x1000","topics":["0xbbb"],"data":"0x"},{"address":"0x2000","topics":["0xccc"],"data":"0x"}]}'
count=$(cb_count_logs "$mock_receipt2" "0x1000")
assert_eq "$count" "2" "two logs from 0x1000"

describe "event: cb_count_logs returns 0 for unknown address"
count=$(cb_count_logs "$mock_receipt2" "0x9999")
assert_eq "$count" "0" "zero logs for unknown address"

unit_summary
