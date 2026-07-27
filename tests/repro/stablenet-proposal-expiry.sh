#!/usr/bin/env bash
# stablenet-proposal-expiry.sh — live smoke of the proposal-expiry case
# (regression f-system-contracts f3-06), driving the built chainbench CLI with
# the short-expiry genesis overlay against a real gstable binary.
#
# It boots a stablenet with the short-expiry overlay
# (manifests/overlays/stablenet-short-expiry.json), which sets GovValidator
# expiry to 30s and advertises the short-expiry capability, then runs the gated
# case (which proposes, waits past the expiry, and asserts the proposal
# transitions to Expired). The case waits ~35s in real time, so this run takes a
# little over half a minute plus boot.
#
# Needs a real gstable binary (not runnable in CI), like the other tests/repro
# scripts.
#
# Requirements (overridable via env): GSTABLE_BIN (or gstable on PATH), python3.
#
#   GSTABLE_BIN=/path/to/gstable tests/repro/stablenet-proposal-expiry.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GSTABLE_BIN="${GSTABLE_BIN:-$(command -v gstable || true)}"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-proposal-expiry-bin}"
WORK="${WORK:-/tmp/stablenet-proposal-expiry}"
OVERLAY="${OVERLAY:-$REPO/manifests/overlays/stablenet-short-expiry.json}"
VALIDATORS="${VALIDATORS:-4}"
ENDPOINTS="${ENDPOINTS:-1}"
SETTLE="${SETTLE:-15}"

CASE="proposal-expiry-transitions"

log() { printf '\n=== %s ===\n' "$*"; }
cleanup() { pkill -9 -f "$WORK" 2>/dev/null || true; }
trap cleanup EXIT

command -v python3 >/dev/null || {
  echo "missing: python3"
  exit 2
}
[ -n "$GSTABLE_BIN" ] && [ -x "$GSTABLE_BIN" ] || {
  echo "no gstable binary (set GSTABLE_BIN=/path/to/gstable)"
  exit 2
}
[ -f "$OVERLAY" ] || {
  echo "missing overlay: $OVERLAY"
  exit 2
}
[ -x "$CHAINBENCH" ] || {
  echo "building chainbench"
  (cd "$REPO" && go build -o "$CHAINBENCH" ./cmd/chainbench) || exit 2
}

cleanup
sleep 1
rm -rf "$WORK"
mkdir -p "$WORK"

log "boot stablenet with the short-expiry overlay"
"$CHAINBENCH" setup --launch \
  --chain stablenet \
  --binary "$GSTABLE_BIN" \
  --data-dir "$WORK" \
  --keys-dir "$REPO/keys/preset" \
  --validators "$VALIDATORS" --endpoints "$ENDPOINTS" \
  --genesis-overlay "$OVERLAY" || {
  echo "setup --launch failed"
  exit 1
}

if ! python3 -c "import json,sys; caps=json.load(open('$WORK/nodeset.json')).get('capabilities',[]); sys.exit(0 if 'short-expiry' in caps else 1)"; then
  echo "nodeset.json missing short-expiry capability — overlay not applied"
  exit 1
fi

log "settle ${SETTLE}s for boot + peering"
sleep "$SETTLE"

log "run $CASE (waits ~35s for the proposal to expire)"
if ! out="$("$CHAINBENCH" test --data-dir "$WORK" --name "$CASE" 2>&1)"; then
  echo "$out"
  echo "FAIL: proposal-expiry case reported failures"
  exit 1
fi
echo "$out"
if echo "$out" | grep -qE 'skip=[1-9]'; then
  echo "FAIL: case skipped (capability/gating problem)"
  exit 1
fi

"$CHAINBENCH" stop --data-dir "$WORK" >/dev/null 2>&1 || true
log "PASS: proposal-expiry transitions to Expired"
