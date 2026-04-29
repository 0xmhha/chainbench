# Sprint 5a — Capability Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Wire up the capability gate end-to-end. Add the `network.capabilities`
Go handler (schema enum has existed since Sprint 2 but no handler yet — same
pattern as `node.rpc` was). Add provider-based capability inference. Expose
via MCP `chainbench_network_capabilities` tool. Wire bash test runner to
check `requires_capabilities` frontmatter against active network's caps and
skip mismatched tests with a clear reason. Add a couple of fault tests with
frontmatter as proof of concept.

**Architecture:** Capability set is provider-derived (no runtime probe in
this pass). `local` provider declares `[rpc ws process fs admin
network-topology]`, `remote` declares `[rpc ws]`, `ssh-remote` (future)
declares `[rpc ws process fs]`. Network's effective capabilities = set
intersection across all nodes. bash test runner pre-flights every test against
this set; tests without frontmatter run unconditionally (back-compat).

**Tech Stack:** Go (schema-extending handler), TypeScript (1 new MCP tool +
tests), bash (test runner integration + fixture). No new dependencies.

Spec: `docs/superpowers/specs/2026-04-29-sprint-5a-capability-gate.md`

---

## Commit Discipline

- English commit messages.
- NO `Co-Authored-By` trailer; NO "Generated with Claude Code"; NO emoji.
- Conventional Commits: `feat(network-net):`, `feat(mcp):`, `feat(test-runner):`,
  `test(bash):`, `docs+chore(sprint-5a):`.

## File Structure

**Create:**
- `mcp-server/test/network.test.ts` — vitest tests for the new MCP tool
- `tests/unit/tests/cmd-test-capabilities.sh` — bash unit test for capability gating

**Modify:**
- `network/cmd/chainbench-net/handlers_network.go` — add `newHandleNetworkCapabilities`
- `network/cmd/chainbench-net/handlers.go` — register `"network.capabilities"`
- `network/cmd/chainbench-net/handlers_test.go` — 5 unit tests
- `mcp-server/src/tools/network.ts` — add `NetworkCapabilitiesArgs` + `_networkCapabilitiesHandler` + register
- `lib/cmd_test.sh` — add `_cb_test_check_capabilities` + main-loop integration
- `tests/fault/<one-or-two-files>.sh` — add frontmatter (`requires_capabilities: [process]`)
- `mcp-server/package.json` — 0.6.0 → 0.7.0
- `mcp-server/src/index.ts` — McpServer version 0.7.0
- `docs/EVALUATION_CAPABILITY.md` — Sprint 5a row + cell flips (capability column or §5)
- `docs/VISION_AND_ROADMAP.md` — Sprint 5a 박스
- `docs/NEXT_WORK.md` — §1/§2/§3/§4.6

---

## Task 1 — Go: `network.capabilities` handler

**Files:**
- Modify: `network/cmd/chainbench-net/handlers_network.go` — add `newHandleNetworkCapabilities`
- Modify: `network/cmd/chainbench-net/handlers.go` — register
- Modify: `network/cmd/chainbench-net/handlers_test.go` — 5 unit tests

### Step 1.1: Inspect existing state surface

`network/internal/state/` has `LoadActive` (returns `Profile`) and `LoadRemote`
or similar. The handler needs to enumerate the network's nodes and their
provider type. Each `state.Node` has a `Provider` field (`"local"` /
`"remote"`).

If `state` doesn't expose a single helper that returns nodes for a given
network name, add one (or compose existing helpers). Read the state package
header before writing the handler to confirm.

### Step 1.2: Capability inference

```go
// providerCaps maps a provider type to its declared capability set.
// Sets are sorted alphabetically for deterministic JSON output.
var providerCaps = map[string][]string{
    "local":      {"admin", "fs", "network-topology", "process", "rpc", "ws"},
    "remote":     {"rpc", "ws"},
    "ssh-remote": {"fs", "process", "rpc", "ws"}, // future; declared for forward-compat
}

// inferNetworkCapabilities returns the set intersection across all nodes.
func inferNetworkCapabilities(nodes []state.Node) []string {
    if len(nodes) == 0 {
        return []string{}
    }
    common := make(map[string]struct{})
    for _, c := range providerCaps[nodes[0].Provider] {
        common[c] = struct{}{}
    }
    for _, n := range nodes[1:] {
        next := providerCaps[n.Provider]
        for c := range common {
            if !contains(next, c) {
                delete(common, c)
            }
        }
    }
    out := make([]string, 0, len(common))
    for c := range common {
        out = append(out, c)
    }
    sort.Strings(out)
    return out
}
```

### Step 1.3: Handler

```go
func newHandleNetworkCapabilities(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network string `json:"network"`
        }
        if len(args) > 0 {
            if err := json.Unmarshal(args, &req); err != nil {
                return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
            }
        }
        if req.Network == "" {
            req.Network = "local"
        }
        nodes, err := loadNetworkNodes(stateDir, req.Network)
        if err != nil {
            return nil, err
        }
        caps := inferNetworkCapabilities(nodes)
        return map[string]any{
            "network":      req.Network,
            "capabilities": caps,
        }, nil
    }
}
```

`loadNetworkNodes` is a thin wrapper around the existing state reader (or use
the same path as `resolveNode` but iterate all nodes). Check what's already
exported and prefer reuse.

### Step 1.4: Register + tests

Register `"network.capabilities"` in `allHandlers` (alphabetical, between
`network.attach` and `network.load`).

Tests in `handlers_test.go`:
- `TestHandleNetworkCapabilities_Local` — seeded local network → all 6 caps
- `TestHandleNetworkCapabilities_Remote` — seeded remote-only network → 2 caps
- `TestHandleNetworkCapabilities_Hybrid` — 1 local + 1 remote → intersection
- `TestHandleNetworkCapabilities_DefaultsToLocal` — args 미지정 → "local"
- `TestAllHandlers_IncludesNetworkCapabilities`

Reuse `saveRemoteFixture` / state seeding helpers.

### Step 1.5: Verify + commit

```bash
go -C network test ./cmd/chainbench-net/... -count=1 -v -run 'TestHandleNetworkCapabilities|IncludesNetworkCapabilities'
go -C network test ./... -count=1 -timeout=60s
go -C network vet ./... && gofmt -l network/
```

```bash
git add network/cmd/chainbench-net/handlers.go \
        network/cmd/chainbench-net/handlers_network.go \
        network/cmd/chainbench-net/handlers_test.go
git commit -m "feat(network-net): network.capabilities handler with provider-based inference

Schema enum had 'network.capabilities' since Sprint 2 but no Go
handler. Sprint 5a wires the capability gate end-to-end, so the
handler now needs to exist on the Go side.

inferNetworkCapabilities returns the set intersection across the
network's nodes, where each node contributes the capability set
declared by its provider:
  - local: rpc, ws, process, fs, admin, network-topology
  - remote: rpc, ws
  - ssh-remote (future): rpc, ws, process, fs

Hybrid networks (mix of local + remote) get only the intersection,
ensuring the capability set is the conservative lower bound that
all nodes can satisfy.

Tests cover happy paths for local-only, remote-only, hybrid, the
default-to-local behavior when args.network is omitted, and the
dispatcher registration check."
```

---

## Task 2 — MCP: `chainbench_network_capabilities`

**Files:**
- Modify: `mcp-server/src/tools/network.ts`
- Create: `mcp-server/test/network.test.ts`

### Step 2.1: Schema + handler

```typescript
import { callWire } from "../utils/wire.js";
import { formatWireResult } from "../utils/wireResult.js";
import { type FormattedToolResponse } from "../utils/mcpResp.js";

export const NetworkCapabilitiesArgs = z.object({
  network: z.string().min(1).optional()
    .describe("Network alias. Defaults to 'local'."),
}).strict();

type NetworkCapabilitiesArgsT = z.infer<typeof NetworkCapabilitiesArgs>;

export async function _networkCapabilitiesHandler(
  args: NetworkCapabilitiesArgsT,
): Promise<FormattedToolResponse> {
  const wireArgs: Record<string, unknown> = {};
  if (args.network !== undefined) wireArgs.network = args.network;
  const result = await callWire("network.capabilities", wireArgs);
  return formatWireResult(result);
}
```

Register in `registerNetworkTools`:
```typescript
server.tool(
  "chainbench_network_capabilities",
  "Read the capability set of a network (defaults to 'local'). " +
    "Capabilities indicate which operations are supported. " +
    "Local networks support: rpc, ws, process, fs, admin, network-topology. " +
    "Remote-attached networks support: rpc, ws (no process control or filesystem access). " +
    "Use this to gate features that depend on process control (e.g., crash tests need 'process').",
  NetworkCapabilitiesArgs.shape,
  _networkCapabilitiesHandler,
);
```

### Step 2.2: Tests

`network.test.ts` (NEW), 4 tests:
- `_Happy_Local` — mock returns ok:true with `data: {network: "local", capabilities: [...]}`; assert text contains all 6 caps
- `_Happy_Remote` — mock returns reduced cap set; assert text reflects
- `_StrictRejectsUnknownKeys`
- `_WireFailure_PassedThrough`

### Step 2.3: Verify + commit

```bash
cd mcp-server && npm test                # 90 + 4 = 94
cd mcp-server && npx tsc --noEmit
cd mcp-server && npm run build
```

```bash
git add mcp-server/src/tools/network.ts mcp-server/test/network.test.ts
git commit -m "feat(mcp): chainbench_network_capabilities tool

Wraps the chainbench-net network.capabilities wire command added
in the previous commit. Read-only — accepts an optional network
alias (defaults to 'local') and returns the network plus its
capability set.

LLM callers use this to gate calls that depend on process control
(node start/stop) or filesystem access (log tail), since
remote-attached networks do not provide those capabilities. The
tool description spells out the per-provider declarations so the
LLM can reason about which scenarios apply on the active network.
"
```

---

## Task 3 — bash test runner: `requires_capabilities` gating

**Files:**
- Modify: `lib/cmd_test.sh` — gating helper + main-loop integration
- Create: `tests/unit/tests/cmd-test-capabilities.sh`

### Step 3.1: Helper functions

In `lib/cmd_test.sh`:

```bash
# _cb_test_active_capabilities
# Calls chainbench-net network.capabilities and prints the cap list (space-separated).
# Returns 1 if chainbench-net is unreachable.
_cb_test_active_capabilities() {
  local data
  if ! data=$(cb_net_call "network.capabilities" '{}' 2>/dev/null); then
    return 1
  fi
  echo "$data" | jq -r '.capabilities[]' 2>/dev/null | tr '\n' ' ' | sed 's/ $//'
}

# _cb_test_check_capabilities <test_path>
# Returns 0 if test can run (no frontmatter or all caps satisfied).
# Returns 1 with a SKIP diagnostic on stderr if caps are missing.
# Returns 0 (best-effort allow) if chainbench-net is unreachable + frontmatter absent.
_cb_test_check_capabilities() {
  local test_path="$1"
  local meta_json
  meta_json=$(cb_parse_meta "$test_path" 2>/dev/null || echo "{}")
  local required
  required=$(echo "$meta_json" | jq -r '.requires_capabilities[]?' 2>/dev/null | tr '\n' ' ')
  required="${required% }"
  [[ -z "$required" ]] && return 0  # no requirements → can run

  local active_caps
  if ! active_caps=$(_cb_test_active_capabilities); then
    echo "WARN: cannot resolve network capabilities; running '$test_path' without gating" >&2
    return 0
  fi

  local missing=()
  for r in $required; do
    if ! echo " $active_caps " | grep -q " $r "; then
      missing+=("$r")
    fi
  done

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "SKIP: $test_path — requires capability: ${missing[*]}; active network provides: $active_caps" >&2
    return 1
  fi
  return 0
}
```

Source `lib/network_client.sh` at the top of `cmd_test.sh` (already sourced
indirectly? — verify).

### Step 3.2: Main-loop integration

In the test execution loop, add a precheck before invoking each test:

```bash
if ! _cb_test_check_capabilities "$test_path"; then
  ((skipped_count++))
  continue
fi
# ... existing test invocation ...
```

Track `skipped_count` separately from pass/fail counts and surface in the
final report:

```
Test summary: passed=N failed=M skipped=K
```

### Step 3.3: bash unit test

`tests/unit/tests/cmd-test-capabilities.sh` (NEW). Pattern after existing
`tests/unit/tests/cmd-test-*.sh`. Uses Python mock for chainbench-net.

5 cases:
1. `requires_capabilities: [process]` + active caps include `process` → run (return 0)
2. `requires_capabilities: [process]` + active = `[rpc ws]` → skip (return 1, SKIP message contains "process")
3. No frontmatter → run unconditionally (return 0 even if active caps unknown)
4. Empty `requires_capabilities: []` → run (return 0)
5. chainbench-net unreachable + frontmatter present → WARN + run (return 0; best-effort allow)

### Step 3.4: Verify + commit

```bash
bash tests/unit/run.sh   # all 33 + 1 = 34 pass
```

```bash
git add lib/cmd_test.sh tests/unit/tests/cmd-test-capabilities.sh
git commit -m "feat(test-runner): capability gating via requires_capabilities frontmatter

When a test script declares 'requires_capabilities: [process, fs]'
in its chainbench-meta block, the runner now queries the active
network's capability set (via chainbench-net network.capabilities)
and skips the test with a clear diagnostic if any required
capability is missing.

Tests without frontmatter continue to run unconditionally
(back-compat — existing tests are unaffected). When chainbench-net
is unreachable (e.g., no active network), the runner falls back
to permissive mode and emits a WARN so the user can investigate
without blocking the run.

The skipped count is surfaced in the final report alongside
pass/fail. Sprint 5a adds frontmatter to selected fault tests in
a follow-up commit."
```

---

## Task 4 — Example test frontmatter

**Files:**
- Modify: 1-2 files in `tests/fault/` (e.g., `node-crash.sh`, `peer-disconnect.sh`)

### Step 4.1: Choose target tests

Inspect `tests/fault/` and pick 1-2 tests that genuinely require process
control (node stop/start) — these are the natural candidates for
`requires_capabilities: [process]`.

If `tests/fault/` has tests that touch the filesystem (log tail), add
`requires_capabilities: [process, fs]` instead.

### Step 4.2: Add frontmatter

```bash
#!/usr/bin/env bash
# ---chainbench-meta---
# description: <one-line>
# requires_capabilities: [process]
# chain_compat: [stablenet, wbft]
# ---end-meta---

# ... existing test body ...
```

Verify the frontmatter parses with `cb_parse_meta` standalone:
```bash
source lib/test_meta.sh
cb_parse_meta tests/fault/node-crash.sh | jq .
# Expected: { "description": "...", "requires_capabilities": ["process"], "chain_compat": [...] }
```

### Step 4.3: Smoke test

Manually run `chainbench test run fault` against:
- a local network (cap `process` present) → all tests run
- (simulated) remote-only network → frontmatter'd tests skip with the SKIP message

(Or just verify by reading the gating logic — full smoke test requires a
running chainbench environment.)

### Step 4.4: Commit

```bash
git add tests/fault/<file1>.sh [tests/fault/<file2>.sh]
git commit -m "test(fault): annotate process-bound fault tests with requires_capabilities

Sets up the first concrete users of Sprint 5a's capability gate.
The frontmatter declares the 'process' capability requirement so
the runner skips these tests cleanly when run against a
remote-attached network (which provides only rpc + ws).

Subsequent passes will sweep the rest of the fault category and
the regression categories that touch chain state via process
control or filesystem reads. This first pass is the
proof-of-concept that the gating actually fires end-to-end."
```

---

## Task 5 — Docs + version bump 0.7.0

**Files:**
- Modify: `mcp-server/package.json` — 0.6.0 → 0.7.0
- Modify: `mcp-server/src/index.ts` — McpServer version
- Modify: `docs/EVALUATION_CAPABILITY.md`
- Modify: `docs/VISION_AND_ROADMAP.md`
- Modify: `docs/NEXT_WORK.md`

### Step 5.1: Version bump

`package.json` 0.6.0 → 0.7.0. `index.ts` `version: "0.7.0"`.

### Step 5.2: EVALUATION_CAPABILITY.md

Add a new column (or row) for `network.capabilities` if not already present.
Add §5 row for Sprint 5a:
- `**5a**` | `Capability gate — network.capabilities Go handler + MCP tool + bash test runner integration + first proof-of-concept frontmatter` | `✅ 완료 (2026-04-29)`

§6 closing: "Sprint 5a 완료 (2026-04-29): capability gate 가 wire (Go) + MCP
+ bash test runner 세 layer 모두에서 동작. fault test 가 remote 환경에서
자동 skip — coding agent 가 시나리오 자동화 가능 여부 사전 판단 가능."

### Step 5.3: VISION_AND_ROADMAP.md

§6 Sprint 5 갱신:
- `Sprint 5c.1/5c.2/5c.3` 그대로
- `**Sprint 5a — Capability gate**` 완료 박스
- `Sprint 5b — SSHRemoteDriver` 다음 P1
- `Sprint 5c.4 — MCP reroute pass 2` 후순위
- `Sprint 5d — Hybrid 예제` 후순위

§5.5 (Capability Negotiation) 에 짧은 갱신: "Sprint 5a 가 본 절의 첫 패스.
provider-based 추론 + bash test runner gating + MCP exposure 완료. Runtime
probe / per-node 차이 / 모든 test frontmatter 부여는 후속."

### Step 5.4: NEXT_WORK.md

§1 헤더 → `2026-04-29 (Sprint 5a 완료)`

§2.1 timeline 표 — add 5a row:
| **5a** | **Capability gate — wire handler + MCP tool + bash runner gating** | **2026-04-29** | **6** |

§2.3 — bullet 갱신: vitest 90 → 94; bash test 33 → 34.

§3 P1 narrative — 다음 P1 = Sprint 5b (SSHRemoteDriver).

§3 P3 누적 tech debt 표 — 다음 row 추가:
- `5a: All fault/regression tests not yet frontmatter'd` — 5a Pass 1 은 1-2
  파일만 demo. 점진 부여 필요. 트리거: remote 환경에서 부적합 test 발견 시
- `5a: Runtime capability probe` — provider declaration 만 사용. admin RPC
  실제 동작 여부 등 runtime probe 후속
- `5a: MCP capability-aware tool gating` — coding agent 가 capability 부재
  시 자동 retry/skip 하는 layer 부재

§3 (또는 최근 완료) — Sprint 5a entry 추가.

§4.6 file size 표 — 신규 파일 사이즈 명시 (network.test.ts 등).

### Step 5.5: Verify + commit

```bash
cd mcp-server && npm test
cd mcp-server && npx tsc --noEmit && npm run build
go -C network test ./... -count=1 -timeout=60s
bash tests/unit/run.sh
```

```bash
git add mcp-server/package.json mcp-server/src/index.ts \
        docs/EVALUATION_CAPABILITY.md docs/VISION_AND_ROADMAP.md docs/NEXT_WORK.md
git commit -m "docs+chore(sprint-5a): roadmap + capability matrix + version 0.7.0

Sprint 5a (capability gate) completes. Per-provider capability
inference is wired through the network.capabilities Go handler,
the chainbench_network_capabilities MCP tool, and the bash test
runner's requires_capabilities frontmatter check.

EVALUATION_CAPABILITY §5 gains the 5a row. VISION_AND_ROADMAP
marks Sprint 5a complete; next P1 is Sprint 5b (SSHRemoteDriver).
NEXT_WORK timeline gains the 5a row; tech debt tracks the
incremental frontmatter sweep, runtime capability probing, and
MCP-side capability-aware tool gating as follow-ups.

mcp-server version 0.6.0 -> 0.7.0 (capability gate is a new
LLM-facing tool surface)."
```

---

## Final report (after all tasks)

Commit chain (~6-7 expected). vitest 90 → 94. bash test count delta. Go test
count delta (5 new in handlers_test.go). Confirmed deferrals: 5a Pass 2+
(frontmatter sweep), 5b (SSHRemoteDriver), 5d (hybrid), runtime capability
probe.
