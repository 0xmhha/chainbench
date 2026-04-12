#!/usr/bin/env bash
# tests/unit/tests/cmd-spec-lookup.sh - Test spec lookup command
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

# Create a minimal spec fixture
SPEC_DOC="${TMPDIR_ROOT}/spec.md"
cat > "$SPEC_DOC" <<'SPEC'
# Test Spec

## Section A

#### RT-A-2-01 — Legacy Tx (type 0x0)

| 필드 | 값 |
|---|---|
| **ID** | RT-A-2-01 |
| **우선순위** | P0 |
| **유형** | Positive / Regression |
| **선행 TC** | RT-F-5-03 |
| **연관 TC** | RT-A-2-02, RT-A-2-08 |
| **코드 근거** | core/types/tx_legacy.go |

##### 검증 의도 (Why)
Legacy Tx effectiveGasPrice 검증

##### 시나리오 (Gherkin-style)
- **Given** 잔액이 충분한 계정
- **When** legacy tx 전송
- **Then** receipt.status == 0x1

---

#### RT-B-01 — 1초 블록 주기

| 필드 | 값 |
|---|---|
| **ID** | RT-B-01 |
| **우선순위** | P0 |

##### 시나리오
- **Given** WBFT 체인 실행
- **Then** 블록 간격 1초

---
SPEC

source "${CHAINBENCH_DIR}/lib/cmd_spec.sh" 2>/dev/null || true
# Re-source to get functions (guard prevents double-load)
unset _CB_CMD_SPEC_SH_LOADED 2>/dev/null || true

# ---- Test 1: lookup finds existing TC ----------------------------------------
describe "spec lookup: finds RT-A-2-01"
result="$(CHAINBENCH_SPEC_DOC="$SPEC_DOC" bash -c "
  source '${CHAINBENCH_DIR}/lib/common.sh'
  unset _CB_CMD_SPEC_SH_LOADED
  source '${CHAINBENCH_DIR}/lib/cmd_spec.sh'
" _ lookup RT-A-2-01 "$SPEC_DOC" 2>/dev/null)"
has_id="$(echo "$result" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)"
assert_eq "$has_id" "RT-A-2-01" "found correct ID"

has_title="$(echo "$result" | python3 -c "import sys,json; print(json.load(sys.stdin).get('title',''))" 2>/dev/null)"
assert_contains "$has_title" "Legacy Tx" "title contains Legacy Tx"

has_priority="$(echo "$result" | python3 -c "import sys,json; print(json.load(sys.stdin).get('priority',''))" 2>/dev/null)"
assert_eq "$has_priority" "P0" "priority is P0"

# ---- Test 2: lookup finds TC with different ID format -------------------------
describe "spec lookup: finds RT-B-01"
result2="$(CHAINBENCH_SPEC_DOC="$SPEC_DOC" bash -c "
  source '${CHAINBENCH_DIR}/lib/common.sh'
  unset _CB_CMD_SPEC_SH_LOADED
  source '${CHAINBENCH_DIR}/lib/cmd_spec.sh'
" _ lookup RT-B-01 "$SPEC_DOC" 2>/dev/null)"
has_id2="$(echo "$result2" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)"
assert_eq "$has_id2" "RT-B-01" "found RT-B-01"

# ---- Test 3: lookup returns error for nonexistent TC -------------------------
describe "spec lookup: returns error for missing TC"
rc=0
CHAINBENCH_SPEC_DOC="$SPEC_DOC" bash -c "
  source '${CHAINBENCH_DIR}/lib/common.sh'
  unset _CB_CMD_SPEC_SH_LOADED
  source '${CHAINBENCH_DIR}/lib/cmd_spec.sh'
" _ lookup RT-Z-99-99 "$SPEC_DOC" 2>/dev/null || rc=$?
assert_eq "$rc" "1" "exits 1 for missing TC"

# ---- Test 4: excerpt contains scenario content --------------------------------
describe "spec lookup: excerpt contains scenario"
excerpt="$(echo "$result" | python3 -c "import sys,json; print(json.load(sys.stdin).get('excerpt',''))" 2>/dev/null)"
assert_contains "$excerpt" "Given" "excerpt has Given keyword"
assert_contains "$excerpt" "Then" "excerpt has Then keyword"

unit_summary
