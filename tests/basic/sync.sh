#!/usr/bin/env bash
# Test: basic/sync
# Description: Verify all running nodes have synchronized block heights
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"

test_start "basic/sync"

# Call check_sync and parse results
sync_json=$(check_sync)
printf '[INFO]  Sync result: %s\n' "$sync_json" >&2

synced=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
print('true' if d.get('synced') else 'false')
" "$sync_json")

diff_val=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
print(d.get('diff', 999))
" "$sync_json")

min_block=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
print(d.get('min', 0))
" "$sync_json")

max_block=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
print(d.get('max', 0))
" "$sync_json")

printf '[INFO]  Block range: min=%s max=%s diff=%s\n' \
  "$min_block" "$max_block" "$diff_val" >&2

# Assert synced is true
assert_true "$synced" "nodes are synced"

# Assert diff <= 2
assert_ge "2" "$diff_val" "block height diff <= 2 (got $diff_val)"

test_result
