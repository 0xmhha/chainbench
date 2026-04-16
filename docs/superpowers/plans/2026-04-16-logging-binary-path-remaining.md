# Logging & Binary Path — Remaining Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the 3 remaining issues from the approved spec at `docs/superpowers/specs/2026-04-09-logging-and-binary-path-design.md`

**Architecture:** The spec's 9-phase implementation is 95% complete. Three items remain: (1) cmd_start.sh launch pattern must change from process substitution to the spec's corrected separate-process model, (2) cmd_config.sh has a data-loss bug when PyYAML is unavailable, (3) a smoke test for logrot integration is missing.

**Tech Stack:** Bash 4+, Python 3, existing chainbench unit test framework (`tests/unit/`)

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `lib/cmd_start.sh` | Modify lines 220-230 | Change logrot launch from process substitution to separate companion process |
| `lib/cmd_config.sh` | Modify lines 128-135 and 180-185 | Add JSON fallback when PyYAML is unavailable for `set` and `unset` |
| `tests/unit/tests/smoke-logrot-integration.sh` | Create | Verify logrot file-watching rotation works with mock data |

---

### Task 1: Fix cmd_start.sh launch pattern (process substitution → separate process)

**Files:**
- Modify: `lib/cmd_start.sh:220-230`

**Context:** The spec (§4.4) was corrected to use a separate-process model instead of process substitution. `logrot` is a file-watching rotator that monitors a file on disk — it does not read from stdin. The current code uses `> >(logrot ...)` which routes stdout through a process substitution, but this is incorrect for logrot's actual behavior. The separate-process model launches logrot as an independent companion watcher, preserving gstable's PID in `$!` for health checks.

- [ ] **Step 1: Read the current launch block to confirm exact code to replace**

Run: `sed -n '220,230p' lib/cmd_start.sh`

Expected output:
```bash
  # Launch node, routing output through logrot if available
  if [[ -n "${LOGROT_BIN:-}" ]]; then
    nohup "${launch_cmd[@]}" \
      > >("${LOGROT_BIN}" "${node_log}" "${CHAINBENCH_LOG_MAX_SIZE:-10M}" "${CHAINBENCH_LOG_MAX_FILES:-5}") \
      2>&1 &
    local node_pid=$!
  else
    nohup "${launch_cmd[@]}" >> "${node_log}" 2>&1 &
    local node_pid=$!
  fi
  disown "${node_pid}" 2>/dev/null || true
```

- [ ] **Step 2: Replace the launch block with separate-process model**

Replace lines 220-230 of `lib/cmd_start.sh` with:

```bash
  # Launch node — gstable writes directly to file, PID captured for health checks
  nohup "${launch_cmd[@]}" >> "${node_log}" 2>&1 &
  local node_pid=$!
  disown "${node_pid}" 2>/dev/null || true

  # Launch logrot as a companion file watcher (if available)
  # logrot monitors the file on disk and rotates it when it exceeds max_size.
  # It runs as a separate process — cleanup is handled by cmd_stop.sh pkill.
  if [[ -n "${LOGROT_BIN:-}" ]]; then
    nohup "${LOGROT_BIN}" "${node_log}" \
      "${CHAINBENCH_LOG_MAX_SIZE:-10M}" "${CHAINBENCH_LOG_MAX_FILES:-5}" \
      >/dev/null 2>&1 &
    disown $! 2>/dev/null || true
  fi
```

- [ ] **Step 3: Verify no other process-substitution references remain**

Run: `grep -n '> >(' lib/cmd_start.sh`

Expected: No output (no matches).

- [ ] **Step 4: Run existing unit tests to verify no regressions**

Run: `bash tests/unit/run.sh 2>&1 | tail -5`

Expected: `common-resolve-binary.sh`, `common-resolve-logrot.sh`, `common-parse-overrides.sh` all PASS. Total failures should not increase.

- [ ] **Step 5: Commit**

```bash
git add lib/cmd_start.sh
git commit -m "fix(start): use separate-process model for logrot companion

logrot is a file-watching rotator, not a stdin processor.
Launch it as an independent companion process instead of via
process substitution. This matches the spec's corrected §4.4
and preserves gstable PID tracking for health checks."
```

---

### Task 2: Fix cmd_config.sh data-loss bug (PyYAML fallback)

**Files:**
- Modify: `lib/cmd_config.sh:128-135` (`_cb_config_set`) and `lib/cmd_config.sh:180-185` (`_cb_config_unset`)

**Context:** When PyYAML is not installed, `_cb_config_set()` at line 131-135 catches `ImportError` and does `pass`, leaving `data = {}`. This means the second `set` call loses all previously stored fields. The `_cb_config_get()` function already has a correct fallback (tries JSON parsing). The `set` and `unset` functions need the same fallback.

The dump function at line 147-151 already handles the no-PyYAML case by writing JSON. So the file format without PyYAML is JSON, and the load fallback must try JSON parsing.

- [ ] **Step 1: Verify the bug exists**

Run:
```bash
TMPDIR=$(mktemp -d) && FAKE="${TMPDIR}/cb" && mkdir -p "${FAKE}/state" "${FAKE}/lib" && cp lib/cmd_config.sh lib/common.sh "${FAKE}/lib/" && CHAINBENCH_DIR="${FAKE}" CHAINBENCH_QUIET=1 bash -c 'export CHAINBENCH_DIR CHAINBENCH_QUIET; source "$CHAINBENCH_DIR/lib/common.sh"; unset _CB_CMD_CONFIG_SH_LOADED; source "$CHAINBENCH_DIR/lib/cmd_config.sh"' _ set chain.binary_path /opt/a 2>/dev/null && CHAINBENCH_DIR="${FAKE}" CHAINBENCH_QUIET=1 bash -c 'export CHAINBENCH_DIR CHAINBENCH_QUIET; source "$CHAINBENCH_DIR/lib/common.sh"; unset _CB_CMD_CONFIG_SH_LOADED; source "$CHAINBENCH_DIR/lib/cmd_config.sh"' _ set chain.logrot_path /opt/b 2>/dev/null && CHAINBENCH_DIR="${FAKE}" CHAINBENCH_QUIET=1 bash -c 'export CHAINBENCH_DIR CHAINBENCH_QUIET; source "$CHAINBENCH_DIR/lib/common.sh"; unset _CB_CMD_CONFIG_SH_LOADED; source "$CHAINBENCH_DIR/lib/cmd_config.sh"' _ get chain.binary_path 2>/dev/null; rm -rf "$TMPDIR"
```

Expected (bug present): `(not found)` — the first field was lost after the second `set`.

- [ ] **Step 2: Fix the load fallback in `_cb_config_set`**

In `lib/cmd_config.sh`, replace the `_cb_config_set` Python block's load section (lines 128-135):

**Before:**
```python
# Load existing overlay
data = {}
if os.path.isfile(overlay_path):
    try:
        import yaml
        with open(overlay_path) as fh:
            data = yaml.safe_load(fh) or {}
    except ImportError:
        pass
```

**After:**
```python
# Load existing overlay
data = {}
if os.path.isfile(overlay_path):
    with open(overlay_path) as fh:
        content = fh.read()
    if content.strip():
        try:
            import yaml
            data = yaml.safe_load(content) or {}
        except ImportError:
            try:
                data = json.loads(content)
            except (json.JSONDecodeError, ValueError):
                data = {}
```

- [ ] **Step 3: Fix the same pattern in `_cb_config_unset`**

In `lib/cmd_config.sh`, replace the `_cb_config_unset` Python block's load section (lines 180-185):

**Before:**
```python
# Load existing overlay
try:
    import yaml
    with open(overlay_path) as fh:
        data = yaml.safe_load(fh) or {}
except ImportError:
    data = {}
```

**After:**
```python
# Load existing overlay
data = {}
with open(overlay_path) as fh:
    content = fh.read()
if content.strip():
    try:
        import yaml
        data = yaml.safe_load(content) or {}
    except ImportError:
        try:
            data = json.loads(content)
        except (json.JSONDecodeError, ValueError):
            data = {}
```

- [ ] **Step 4: Verify the bug is fixed**

Run the same command as Step 1.

Expected (bug fixed): `/opt/a` — the first field is preserved after the second `set`.

- [ ] **Step 5: Run the cmd-config unit test**

Run: `bash tests/unit/tests/cmd-config.sh 2>&1`

Expected: All 8 assertions PASS, exit code 0.

- [ ] **Step 6: Run all unit tests**

Run: `bash tests/unit/run.sh 2>&1 | tail -5`

Expected: `cmd-config.sh` now PASSES. Total failures decrease by 1.

- [ ] **Step 7: Commit**

```bash
git add lib/cmd_config.sh
git commit -m "fix(config): add JSON fallback when PyYAML unavailable

The set and unset commands previously lost existing overlay data
on systems without PyYAML because the ImportError handler silently
fell through to an empty dict. Add a JSON parse fallback matching
the pattern already used by the get command."
```

---

### Task 3: Add smoke-logrot-integration test

**Files:**
- Create: `tests/unit/tests/smoke-logrot-integration.sh`

**Context:** Spec §8.3 calls for a mock-based smoke test that verifies logrot rotates files. Instead of running gstable, a shell loop continuously appends data to a file while logrot watches it. The test asserts that a rotated backup file (`.1`) appears within a timeout.

This test requires the `logrot` binary to be available. If it isn't, the test should skip gracefully (not fail).

- [ ] **Step 1: Create the smoke test file**

Create `tests/unit/tests/smoke-logrot-integration.sh`:

```bash
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
```

- [ ] **Step 2: Make it executable**

Run: `chmod +x tests/unit/tests/smoke-logrot-integration.sh`

- [ ] **Step 3: Run the smoke test in isolation**

Run: `bash tests/unit/tests/smoke-logrot-integration.sh 2>&1`

Expected outcomes:
- If logrot is available: `✓ rotated file ... appeared` + PASS
- If logrot is NOT available: `[SKIP] logrot binary not available` + exit 0

- [ ] **Step 4: Run all unit tests**

Run: `bash tests/unit/run.sh 2>&1 | tail -5`

Expected: New test appears in the list. No new failures.

- [ ] **Step 5: Commit**

```bash
git add tests/unit/tests/smoke-logrot-integration.sh
git commit -m "test(logrot): add smoke test for file rotation

Mock-based test that verifies logrot rotates files when a size
threshold is exceeded. Skips gracefully if logrot binary is
not available. Covers spec §8.3 smoke-logrot-integration."
```

---

## Verification Checklist

After all 3 tasks are complete:

- [ ] `bash tests/unit/run.sh` — all tests PASS (only `test-meta-parse.sh` may still fail — it's unrelated to this spec)
- [ ] `grep -n '> >(' lib/cmd_start.sh` — no process substitution remaining
- [ ] `grep -c 'except ImportError' lib/cmd_config.sh` — zero occurrences (the bare `pass` pattern is eliminated)
- [ ] `ls tests/unit/tests/smoke-logrot-integration.sh` — file exists
