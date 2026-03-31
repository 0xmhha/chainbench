#!/usr/bin/env bash
# Test: basic/wbft-consensus
# Description: Verify WBFT protocol properties - validator participation, round stability, commit seals
set -euo pipefail

source "$(dirname "$0")/../lib/rpc.sh"
source "$(dirname "$0")/../lib/assert.sh"
source "$(dirname "$0")/../lib/wait.sh"

test_start "basic/wbft-consensus"

# Ensure enough blocks exist for meaningful analysis
current_block=$(block_number "1")
if [[ "$current_block" -lt 30 ]]; then
  target_block=30
  printf '[INFO]  Waiting for block %d (current: %d)...\n' "$target_block" "$current_block" >&2
  reached=$(wait_for_block "1" "$target_block" 120)
  if [[ "$reached" == "timeout" ]]; then
    _assert_fail "timed out waiting for 30 blocks"
    test_result
    exit 1
  fi
  current_block="$reached"
fi

# ---------------------------------------------------------------------------
# 1. Validator set verification
# ---------------------------------------------------------------------------

validators_json=$(istanbul_get_validators "1")
validator_count=$(python3 -c "
import json, sys
vals = json.loads(sys.argv[1])
print(len(vals))
" "$validators_json")

printf '[INFO]  Active validators: %s\n' "$validator_count" >&2
assert_ge "$validator_count" "3" "at least 3 validators active (found $validator_count)"

# Verify each running node agrees on validator set
running_nodes=$(get_running_node_ids)
for nid in $running_nodes; do
  [[ "$nid" == "1" ]] && continue
  other_vals=$(istanbul_get_validators "$nid" 2>/dev/null || echo "[]")
  vals_match=$(python3 -c "
import json, sys
a = set(v.lower() for v in json.loads(sys.argv[1]))
b = set(v.lower() for v in json.loads(sys.argv[2]))
print('true' if a == b else 'false')
" "$validators_json" "$other_vals")
  assert_true "$vals_match" "node $nid agrees on validator set"
done

# ---------------------------------------------------------------------------
# 2. WBFT extra info: round stability and committed seals
# ---------------------------------------------------------------------------

sample_start=$(( current_block - 19 ))
[[ "$sample_start" -lt 1 ]] && sample_start=1
sample_count=$(( current_block - sample_start + 1 ))

round_zero_count=0
has_committed_seal=0
total_sampled=0

for blk in $(seq "$sample_start" "$current_block"); do
  extra_info=$(istanbul_get_wbft_extra_info "1" "$blk" 2>/dev/null || echo '{}')

  round=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
r = d.get('round', '0x0')
print(int(r, 16) if isinstance(r, str) and r.startswith('0x') else int(r))
" "$extra_info" 2>/dev/null || echo "0")

  committed=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
cs = d.get('committedSeal') or {}
sealers = cs.get('sealers', []) if isinstance(cs, dict) else []
print(len(sealers))
" "$extra_info" 2>/dev/null || echo "0")

  [[ "$round" -eq 0 ]] && round_zero_count=$(( round_zero_count + 1 ))
  [[ "$committed" -gt 0 ]] && has_committed_seal=$(( has_committed_seal + 1 ))
  total_sampled=$(( total_sampled + 1 ))
done

printf '[INFO]  Sampled %d blocks: round-0=%d, has-committed-seal=%d\n' \
  "$total_sampled" "$round_zero_count" "$has_committed_seal" >&2

# Healthy consensus: >= 80% blocks finalize in round 0
min_round_zero=$(( total_sampled * 8 / 10 ))
assert_ge "$round_zero_count" "$min_round_zero" \
  ">=80%% blocks finalized in round 0 ($round_zero_count/$total_sampled)"

# All blocks must have committed seals
assert_eq "$has_committed_seal" "$total_sampled" \
  "all sampled blocks have committed seals ($has_committed_seal/$total_sampled)"

# ---------------------------------------------------------------------------
# 3. Validator participation via istanbul_status
# ---------------------------------------------------------------------------

status_json=$(istanbul_status "1" "$sample_start" "$current_block" 2>/dev/null || echo '{}')

author_count=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
authors = d.get('authorCounts', {})
active = sum(1 for v in authors.values() if v > 0)
print(active)
" "$status_json" 2>/dev/null || echo "0")

printf '[INFO]  Validators that authored blocks: %s/%s\n' "$author_count" "$validator_count" >&2
assert_ge "$author_count" "2" \
  "at least 2 validators authored blocks in range ($author_count)"

# Check committed seal participation: all validators should have signed
committed_signers=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
sa = d.get('sealerActivity', {})
committed = sa.get('committed', {})
active = sum(1 for v in committed.values() if v > 0)
print(active)
" "$status_json" 2>/dev/null || echo "0")

printf '[INFO]  Validators with commit signatures: %s/%s\n' \
  "$committed_signers" "$validator_count" >&2
assert_ge "$committed_signers" "$validator_count" \
  "all validators signed committed seals ($committed_signers/$validator_count)"

# Round distribution check
round_info=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
rs = d.get('roundStats', {})
dist = rs.get('roundDistribution', {})
total = sum(int(v) for v in dist.values()) if dist else 0
round0 = int(dist.get('0', 0))
print(f'{round0}/{total}')
" "$status_json" 2>/dev/null || echo "0/0")

printf '[INFO]  Round distribution (round-0/total): %s\n' "$round_info" >&2

# ---------------------------------------------------------------------------
# 4. Commit signers meet BFT threshold on latest block
# ---------------------------------------------------------------------------

signers_json=$(istanbul_get_commit_signers "1" "$current_block" 2>/dev/null || echo '{}')
signer_count=$(python3 -c "
import json, sys
d = json.loads(sys.argv[1])
committers = d.get('committers', [])
print(len(committers))
" "$signers_json" 2>/dev/null || echo "0")

min_signers=$(python3 -c "
import math, sys
n = int(sys.argv[1])
print(math.ceil(2 * n / 3))
" "$validator_count")

printf '[INFO]  Block %d: %d commit signers (BFT minimum: %s)\n' \
  "$current_block" "$signer_count" "$min_signers" >&2

assert_ge "$signer_count" "$min_signers" \
  "commit signers >= BFT threshold ($signer_count >= $min_signers)"

test_result
