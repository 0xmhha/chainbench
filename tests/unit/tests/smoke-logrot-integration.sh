#!/usr/bin/env bash
# tests/unit/tests/smoke-logrot-integration.sh
# Smoke test: verify logrot rotates files when size limit is exceeded.
# Requires the logrot binary to be available (skips if not found).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

REAL_CB_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
source "${REAL_CB_DIR}/lib/common.sh"

# Resolve logrot — skip test if not available
LOGROT_BIN=""
LOGROT_BIN="$(resolve_logrot "" "")" 2>/dev/null || true

if [[ -z "${LOGROT_BIN}" ]]; then
  echo "  [SKIP] logrot binary not available, skipping smoke test"
  unit_summary
  exit 0
fi

TMPDIR_ROOT="$(mktemp -d)"
trap 'kill %1 %2 2>/dev/null; wait 2>/dev/null; rm -rf "$TMPDIR_ROOT"' EXIT

LOGFILE="${TMPDIR_ROOT}/test-node.log"

# Start a writer that appends ~100 bytes per iteration
(
  i=0
  while true; do
    printf 'block %06d: %s\n' "$i" "$(head -c 60 /dev/urandom | base64 | tr -d '\n')" >> "${LOGFILE}"
    i=$((i + 1))
    sleep 0.01
  done
) &
WRITER_PID=$!

# Start logrot watching the file — rotate at 1K, keep 3 backups
"${LOGROT_BIN}" "${LOGFILE}" 1K 3 &
LOGROT_PID=$!

# Wait up to 5 seconds for rotation to occur
describe "logrot: rotates file when size exceeded"
ROTATED=0
for _t in $(seq 1 50); do
  if [[ -f "${LOGFILE}.1" ]]; then
    ROTATED=1
    break
  fi
  sleep 0.1
done

# Stop both processes
kill "$WRITER_PID" 2>/dev/null || true
kill "$LOGROT_PID" 2>/dev/null || true
wait "$WRITER_PID" 2>/dev/null || true
wait "$LOGROT_PID" 2>/dev/null || true

assert_eq "$ROTATED" "1" "rotated file ${LOGFILE}.1 appeared"

unit_summary
