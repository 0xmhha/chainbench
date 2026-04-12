#!/usr/bin/env bash
# tests/unit/tests/cmd-test-scaffold.sh - Test scaffold command
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_ROOT"' EXIT

# Create minimal spec
SPEC_DOC="${TMPDIR_ROOT}/spec.md"
cat > "$SPEC_DOC" <<'SPEC'
#### RT-X-1-01 — Sample scaffold test

| 필드 | 값 |
|---|---|
| **ID** | RT-X-1-01 |
| **우선순위** | P1 |
| **선행 TC** | none |

##### 검증 의도 (Why)
Verify that sample works correctly.

##### 시나리오 (Gherkin-style)
- **Given** a running chain with 4 validators
- **When** the user sends a transaction
- **Then** receipt.status == 0x1
- **And** gasUsed > 0

---
SPEC

OUTPUT_DIR="${TMPDIR_ROOT}/output"
mkdir -p "$OUTPUT_DIR"

# Source scaffold function
source "${CHAINBENCH_DIR}/lib/test_scaffold.sh" 2>/dev/null || true

# ---- Test 1: scaffold generates a file ---------------------------------------
describe "scaffold: generates script file"
_cb_test_scaffold "$SPEC_DOC" "RT-X-1-01" "$OUTPUT_DIR" 2>/dev/null
generated=$(ls "$OUTPUT_DIR"/*.sh 2>/dev/null | head -1)
assert_nonempty "$generated" "script file created"

# ---- Test 2: generated file has frontmatter ----------------------------------
describe "scaffold: output has frontmatter"
assert_contains "$(cat "$generated")" "---chainbench-meta---" "has meta start"
assert_contains "$(cat "$generated")" "id: RT-X-1-01" "has correct ID"

# ---- Test 3: generated file has scenario as comments -------------------------
describe "scaffold: output has scenario steps"
assert_contains "$(cat "$generated")" "Given" "has Given step"
assert_contains "$(cat "$generated")" "Then" "has Then step"

# ---- Test 4: generated file is valid bash ------------------------------------
describe "scaffold: output is valid bash"
rc=0
bash -n "$generated" 2>/dev/null || rc=$?
assert_eq "$rc" "0" "passes bash syntax check"

unit_summary
