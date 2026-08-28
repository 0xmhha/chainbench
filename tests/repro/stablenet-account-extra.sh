#!/usr/bin/env bash
# stablenet-account-extra.sh — live smoke test of the account-Extra bitmap cases
# (regression h-hardfork h-30/h-33/h-34), driving the built chainbench CLI with
# the account-extra genesis overlay against a real gstable binary.
#
# It boots a local stablenet with the account-extra overlay
# (pkg/chains/stablenet/overlays/account-extra.json), which adds three alloc
# accounts whose Extra bits seed AccountManager status at genesis-init and
# advertises the account-extra capability, then runs the gated cases and asserts
# they PASS (skip=0 — a silent skip means the overlay capability never
# propagated).
#
# This is the B-track verification path for step 4 (docs/dev/HandOff.md §4). It
# needs a real gstable binary and so runs manually (like the other tests/repro
# scripts), not in CI.
#
# Requirements (overridable via env): GSTABLE_BIN (or gstable on PATH), python3.
#
#   GSTABLE_BIN=/path/to/gstable tests/repro/stablenet-account-extra.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GSTABLE_BIN="${GSTABLE_BIN:-$(command -v gstable || true)}"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-account-extra-bin}"
WORK="${WORK:-/tmp/stablenet-account-extra}"
OVERLAY="${OVERLAY:-$REPO/pkg/chains/stablenet/overlays/account-extra.json}"
VALIDATORS="${VALIDATORS:-4}"
ENDPOINTS="${ENDPOINTS:-1}"
SETTLE="${SETTLE:-15}"

CASES=(
  authorized-extra-bit-synced
  blacklisted-extra-bit-synced
  dual-status-extra
  extra-balance-preserved
)

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

log "boot stablenet with the account-extra overlay"
"$CHAINBENCH" net up \
  --chain stablenet \
  --binary "$GSTABLE_BIN" \
  --workspace-dir "$WORK" \
  --keys "$REPO/keys/preset" \
  --validators "$VALIDATORS" --endpoints "$ENDPOINTS" \
  --overlay "$OVERLAY" || {
  echo "net up failed"
  exit 1
}

# The overlay must advertise account-extra, else the cases would silently skip.
if ! python3 -c "import json,sys; caps=json.load(open('$WORK/workspace.json')).get('capabilities',[]); sys.exit(0 if 'account-extra' in caps else 1)"; then
  echo "workspace.json missing account-extra capability — overlay not applied"
  exit 1
fi

log "settle ${SETTLE}s for boot + peering"
sleep "$SETTLE"

args=()
for c in "${CASES[@]}"; do args+=(--name "$c"); done
log "run account-extra cases: ${CASES[*]}"
if ! out="$("$CHAINBENCH" test --workspace-dir "$WORK" "${args[@]}" 2>&1)"; then
  echo "$out"
  echo "FAIL: account-extra cases reported failures"
  exit 1
fi
echo "$out"
if echo "$out" | grep -qE 'skip=[1-9]'; then
  echo "FAIL: account-extra had skipped cases (capability/gating problem)"
  exit 1
fi

"$CHAINBENCH" stop --workspace-dir "$WORK" >/dev/null 2>&1 || true
log "PASS: account-extra bitmap smoke green"
