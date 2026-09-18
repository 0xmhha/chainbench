#!/usr/bin/env bash
# stablenet-delayed-fork.sh — live smoke test of the delayed-fork harness and the
# delayed-Boho fork-transition cases, driving the built chainbench CLI against a
# real gstable binary.
#
# It boots a local stablenet with Boho delayed to a small block
# (genesis.overrides.bohoBlock), so the network crosses the fork within seconds,
# then runs the delayed-boho-gated fork-transition cases and asserts they PASS
# (skip=0 — a silent skip would mean the capability never propagated). It can
# also run the live governance-write cases (GOV=1, the default).
#
# This is the B-track verification path from docs/dev/HandOff.md §4: the delayed
# fork and governance-write cases are otherwise only registration+gating
# verified. It needs a real gstable binary and cannot run in CI (no chain
# binary), so it lives in tests/repro and is run manually in a normal
# environment.
#
# Requirements (overridable via env): GSTABLE_BIN (or gstable on PATH), python3.
#
#   GSTABLE_BIN=/path/to/gstable tests/repro/stablenet-delayed-fork.sh
#   GOV=0 BOHO_BLOCK=5 tests/repro/stablenet-delayed-fork.sh   # fork cases only
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GSTABLE_BIN="${GSTABLE_BIN:-$(command -v gstable || true)}"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-delayed-fork-bin}"
WORK="${WORK:-/tmp/stablenet-delayed-fork}"
BOHO_BLOCK="${BOHO_BLOCK:-5}"
VALIDATORS="${VALIDATORS:-4}"
ENDPOINTS="${ENDPOINTS:-1}"
SETTLE="${SETTLE:-15}" # seconds to let the network boot + peer before running cases

# delayed-boho-gated fork-transition cases (tests/anzeon/fork_transition.go).
FORK_CASES=(
  govminter-code-changes-at-boho
  p256-inactive-before-boho
  anzeon-active-before-boho
  prealloc-preserved-across-boho
)
# Live governance-write cases (node-side signing to quorum). Set GOV=0 to skip.
GOV_CASES=(
  mint-proposal-executes
  burn-proposal-executes
  validator-add-member-executes
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
[ -x "$CHAINBENCH" ] || {
  echo "building chainbench"
  (cd "$REPO" && go build -o "$CHAINBENCH" ./cmd/chainbench) || exit 2
}

cleanup
sleep 1
rm -rf "$WORK"
mkdir -p "$WORK"

log "boot stablenet with Boho delayed to block $BOHO_BLOCK"
"$CHAINBENCH" net up \
  --chain stablenet \
  --binary "$GSTABLE_BIN" \
  --workspace-dir "$WORK" \
  --keys "$REPO/keys/preset" \
  --validators "$VALIDATORS" --endpoints "$ENDPOINTS" \
  --set "bohoBlock=$BOHO_BLOCK" || {
  echo "net up failed"
  exit 1
}

# The workspace must advertise the delayed-boho capability, else the
# fork-transition cases would silently skip and this smoke test would be moot.
if ! python3 -c "import json,sys; caps=json.load(open('$WORK/workspace.json')).get('capabilities',[]); sys.exit(0 if 'delayed-boho' in caps else 1)"; then
  echo "workspace.json missing delayed-boho capability — override not applied"
  exit 1
fi

log "settle ${SETTLE}s for boot + peering"
sleep "$SETTLE"

run_cases() {
  local label="$1"
  shift
  local args=()
  for c in "$@"; do args+=(--name "$c"); done
  log "run $label cases: $*"
  local out
  if ! out="$("$CHAINBENCH" test --workspace-dir "$WORK" "${args[@]}" 2>&1)"; then
    echo "$out"
    echo "FAIL: $label cases reported failures"
    return 1
  fi
  echo "$out"
  # Guard against silent skips: every selected case must have run.
  if echo "$out" | grep -qE 'skip=[1-9]'; then
    echo "FAIL: $label had skipped cases (capability/gating problem)"
    return 1
  fi
  return 0
}

rc=0
run_cases "fork-transition" "${FORK_CASES[@]}" || rc=1
if [ "${GOV:-1}" = "1" ]; then
  run_cases "governance-write" "${GOV_CASES[@]}" || rc=1
fi

"$CHAINBENCH" stop --workspace-dir "$WORK" >/dev/null 2>&1 || true

if [ "$rc" = 0 ]; then log "PASS: delayed-fork + governance smoke green"; else log "FAIL"; fi
exit "$rc"
