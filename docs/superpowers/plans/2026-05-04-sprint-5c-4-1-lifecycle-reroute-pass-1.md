# Sprint 5c.4.1 — Lifecycle Reroute (Pass 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Reroute the simplest 2 lifecycle MCP tools (`chainbench_stop`,
`chainbench_status`) from `runChainbench` (bash CLI shell-out) to `callWire`
(chainbench-net wire) via thin Go wrapper handlers (`network.stop_all` and
`network.status`) that spawn the existing bash via `os/exec`. Pre-work:
extract `_harness.ts` for reusable real-binary integration test setup
(Sprint 5c.3 P3 prerequisite). Side-task: extend Go `node.start` to accept
`binary_path` arg and remove the Sprint 5c.3 runChainbench fallback.

**Architecture:** Thin wrapper pattern (VISION §5.12 M2). Go handler does
not replicate bash logic — just spawns `chainbench.sh stop|status` via
`exec.CommandContext`, captures stdout/stderr, maps exit code to
INVALID_ARGS / UPSTREAM_ERROR. Status's `--json` output is parsed and
passed through; stop's stdout is forwarded as-is. Local network only —
non-local rejects with NOT_SUPPORTED.

**Tech Stack:** Go (`os/exec`, `context.WithTimeout`), TypeScript + zod +
vitest. No new dependencies.

Spec: `docs/superpowers/specs/2026-05-04-sprint-5c-4-1-lifecycle-reroute-pass-1.md`

---

## Commit Discipline

- English commit messages.
- NO `Co-Authored-By` trailer; NO "Generated with Claude Code"; NO emoji.
- Conventional Commits: `refactor(test):`, `feat(network-net):`, `feat(mcp):`,
  `test(mcp):`, `docs+chore(sprint-5c-4-1):`.

## File Structure

**Create:**
- `mcp-server/test/integration/_harness.ts` — reusable real-binary test setup
- `mcp-server/test/lifecycle.test.ts` — vitest unit tests for the 2 rerouted tools
- `mcp-server/test/integration/lifecycle.integration.test.ts` — real-binary integration test for chainbench_status

**Modify:**
- `network/cmd/chainbench-net/handlers_network.go` — add 2 handlers
- `network/cmd/chainbench-net/handlers.go` — register 2 wire commands
- `network/cmd/chainbench-net/handlers_test.go` — 9 unit tests
- `network/cmd/chainbench-net/handlers_node_lifecycle.go` — extend `node.start` for binary_path
- `network/schema/command.json` — add 2 enum entries (regen)
- `network/internal/types/command_gen.go` — regenerated
- `mcp-server/src/tools/lifecycle.ts` — reroute 2 tools (stop + status)
- `mcp-server/src/tools/node.ts` — pass binary_path to wire (remove fallback branch)
- `mcp-server/test/node.test.ts` — adjust binary_path test for wire path
- `mcp-server/test/integration/node_rpc.integration.test.ts` — refactor to use _harness.ts
- `mcp-server/test/node_start_fallback.test.ts` — DELETE
- `mcp-server/package.json` — 0.7.0 → 0.7.1
- `mcp-server/src/index.ts` — McpServer version
- `docs/EVALUATION_CAPABILITY.md`
- `docs/VISION_AND_ROADMAP.md`
- `docs/NEXT_WORK.md`
- `docs/REMAINING_WORK.md`

---

## Task 0 — Integration test harness extract

**Files:**
- Create: `mcp-server/test/integration/_harness.ts`
- Modify: `mcp-server/test/integration/node_rpc.integration.test.ts`

### Step 0.1: Read existing setup

Read `mcp-server/test/integration/node_rpc.integration.test.ts` carefully —
identify: (a) port allocation, (b) Python mock spawn, (c) state seeding, (d)
env snapshot/restore, (e) teardown. These are the candidates for extraction.

### Step 0.2: Define harness API

```typescript
// mcp-server/test/integration/_harness.ts

export interface RealBinaryHarnessOptions {
  /**
   * Methods returning JSON-RPC responses keyed by method name.
   * Default: eth_chainId → 0x539, eth_blockNumber → 0xdeadbeef.
   */
  rpcHandlers?: Record<string, (req: { id: number | string; params?: unknown[] }) => unknown>;
  /** Override pids.json content for the seeded local network. */
  pidsOverride?: object;
  /** Override current-profile.yaml content. */
  profileOverride?: object;
  /** Path to a fake chainbench.sh script (for lifecycle tests). */
  fakeChainbenchScript?: string;
}

export interface RealBinaryHarness {
  binaryPath: string;
  stateDir: string;
  mockPort: number;
  /** Cleanup handle — must be awaited in afterAll. */
  teardown: () => Promise<void>;
}

export function hasBinary(): boolean;

export async function setupRealBinaryHarness(
  opts?: RealBinaryHarnessOptions,
): Promise<RealBinaryHarness>;
```

### Step 0.3: Implementation

Move the inline code from `node_rpc.integration.test.ts` into `_harness.ts`,
parameterizing by `RealBinaryHarnessOptions`. Add the fixes from Sprint
5c.3 review:
- Cleanup-await: `mockProc.kill('SIGTERM')` followed by Promise wait + 1s
  timeout fallback to SIGKILL
- Port-race diagnostics: capture mock stderr to a buffer (not blackholed)
  and include the buffer in `waitForPort` timeout error message
- Per-test isolation: env snapshot taken inside `setupRealBinaryHarness`,
  restored in `teardown()`

### Step 0.4: Refactor `node_rpc.integration.test.ts` to use harness

Replace the inline setup with:
```typescript
let harness: RealBinaryHarness;
beforeAll(async () => {
  harness = await setupRealBinaryHarness({
    rpcHandlers: {
      eth_chainId: () => "0x539",
      eth_blockNumber: () => "0xdeadbeef",
    },
  });
});
afterAll(() => harness?.teardown());
```

Test body unchanged. Same 1 test must still pass.

### Step 0.5: Verify + commit

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/mcp-server
npm test           # 94/94 unchanged
npx tsc --noEmit
npm run build
```

```bash
git add mcp-server/test/integration/_harness.ts \
        mcp-server/test/integration/node_rpc.integration.test.ts
git commit -m "refactor(test): extract real-binary integration harness for reuse

Sprint 5c.4 prep. Sprint 5c.3 left the integration scaffolding
(port allocation + Python JSON-RPC mock + state-dir seeding +
env snapshot/restore + cleanup) inline in node_rpc.integration.test.ts.
With Sprint 5c.4 about to add a second integration test for
chainbench_status — and 5c.4.2+ adding more — the rule of two is
already met for the harness extraction.

mcp-server/test/integration/_harness.ts hosts setupRealBinaryHarness
with a customizable rpcHandlers map (per-method JSON-RPC response
builders), optional pids/profile overrides, and a teardown handle
that waits for SIGTERM + falls back to SIGKILL after 1s (closes
the 5c.3 review cleanup-await gap). The mock's stderr is captured
to a buffer instead of being blackholed so waitForPort timeout
errors include diagnostic context (closes 5c.3 review port-race
diagnostics gap).

node_rpc.integration.test.ts is refactored to consume the harness
— same 1 test still passes (regression check on the refactor).
The full vitest suite stays at 94/94."
```

---

## Task 1 — Go: `network.stop_all` handler

**Files:**
- Modify: `network/cmd/chainbench-net/handlers_network.go` — add `newHandleNetworkStopAll`
- Modify: `network/cmd/chainbench-net/handlers.go` — register
- Modify: `network/cmd/chainbench-net/handlers_test.go` — 4 unit tests
- Modify: `network/schema/command.json` — add `"network.stop_all"` enum + regen

### Step 1.1: Schema + regen

Add `"network.stop_all"` to the `command` enum in `command.json` (alphabetical,
between `network.probe` and existing entries). Run `cd network && go generate ./...`
to regenerate `command_gen.go`.

### Step 1.2: Handler implementation

```go
// In handlers_network.go

func newHandleNetworkStopAll(stateDir, chainbenchDir string) Handler {
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
        if req.Network != "local" {
            return nil, NewNotSupported(
                fmt.Sprintf("network.stop_all only operates on the local network; got %q", req.Network),
            )
        }

        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        cmd := exec.CommandContext(ctx, filepath.Join(chainbenchDir, "chainbench.sh"), "stop", "--quiet")
        cmd.Env = append(os.Environ(), "CHAINBENCH_DIR="+chainbenchDir)
        out, err := cmd.CombinedOutput()
        if err != nil {
            return nil, NewUpstream("chainbench stop", fmt.Errorf("%w: %s", err, string(out)))
        }
        return map[string]any{
            "network": "local",
            "stdout":  string(out),
        }, nil
    }
}
```

Imports needed: `os/exec`, `path/filepath` (already imported), `context`,
`time` (already imported), `os` (already imported).

### Step 1.3: Register

In `handlers.go`'s `allHandlers` map (alphabetical position):
```go
"network.stop_all": newHandleNetworkStopAll(stateDir, chainbenchDir),
```

The `chainbenchDir` is already passed to `allHandlers(stateDir, chainbenchDir)`
per Sprint 4 follow-up — confirm by reading the function signature.

### Step 1.4: Tests

`handlers_test.go` 4 new tests:
- `TestHandleNetworkStopAll_Happy` — write a fake `chainbench.sh` to t.TempDir() that exits 0, set CHAINBENCH_DIR, call handler, assert ok + stdout
- `TestHandleNetworkStopAll_RemoteRejected` — args `{"network": "sepolia"}` → NOT_SUPPORTED with descriptive message
- `TestHandleNetworkStopAll_BashFailure` — fake script exits 1 with stderr → UPSTREAM_ERROR
- `TestAllHandlers_IncludesNetworkStopAll`

The fake chainbench.sh template:
```bash
#!/bin/bash
echo "fake chainbench: $@"
exit ${FAKE_EXIT:-0}
```

### Step 1.5: Verify + commit

```bash
go -C network test ./cmd/chainbench-net/... -count=1 -v -run 'TestHandleNetworkStopAll|TestAllHandlers_IncludesNetworkStopAll'
go -C network test ./... -count=1 -timeout=60s
go -C network vet ./... && gofmt -l network/
```

```bash
git add network/cmd/chainbench-net/handlers.go \
        network/cmd/chainbench-net/handlers_network.go \
        network/cmd/chainbench-net/handlers_test.go \
        network/schema/command.json \
        network/internal/types/command_gen.go
git commit -m "feat(network-net): network.stop_all thin wrapper of chainbench stop

First lifecycle reroute handler (Sprint 5c.4.1). Per VISION §5.12
M2, the LocalDriver-equivalent for lifecycle commands wraps the
existing bash chainbench.sh via os/exec rather than reimplementing
the 70-line cmd_stop.sh natively in Go. This keeps the bash CLI
authoritative for the actual node-killing logic while giving the
MCP layer a uniform NDJSON wire surface.

The handler accepts an optional 'network' arg (defaults 'local')
and rejects non-local with NOT_SUPPORTED — remote networks have
no node lifecycle to stop. Bash exit codes map to UPSTREAM_ERROR;
args parsing failures map to INVALID_ARGS. Stdout is forwarded
as-is in the result envelope so the LLM sees the per-PID SIGTERM
trace.

Tests cover the happy path, remote-rejection, bash-failure
classification, and dispatcher registration. The fake chainbench.sh
is written to t.TempDir() so tests don't depend on the real
project tree.
"
```

---

## Task 2 — MCP: reroute `chainbench_stop`

**Files:**
- Modify: `mcp-server/src/tools/lifecycle.ts`
- Create: `mcp-server/test/lifecycle.test.ts`

### Step 2.1: Add wire-call rewrite

In `lifecycle.ts`, replace the existing `chainbench_stop` registration with
the wire-routed version. Keep `chainbench_init` / `_start` / `_restart` /
`_status` (status is rewritten in Task 4) untouched here.

```typescript
import { callWire } from "../utils/wire.js";
import { formatWireResult } from "../utils/wireResult.js";
import { type FormattedToolResponse } from "../utils/mcpResp.js";

export const StopArgs = z.object({
  network: z.string().min(1).optional()
    .describe("Network alias. Defaults to 'local'. Remote networks reject."),
}).strict();

export async function _stopHandler(
  args: z.infer<typeof StopArgs>,
): Promise<FormattedToolResponse> {
  const wireArgs: Record<string, unknown> = {};
  if (args.network !== undefined) wireArgs.network = args.network;
  const result = await callWire("network.stop_all", wireArgs);
  return formatWireResult(result);
}

server.tool(
  "chainbench_stop",
  "Stop all running chain nodes gracefully. Local network only — remote " +
  "networks reject (no process control). Returns the bash CLI's stdout " +
  "showing per-PID SIGTERM and graceful-shutdown wait status.",
  StopArgs.shape,
  _stopHandler,
);
```

### Step 2.2: Tests

`mcp-server/test/lifecycle.test.ts` (NEW), 3 tests for stop:
- `_Happy` — mock binary returns ok:true with stdout; assert text contains stdout
- `_StrictRejectsUnknownKeys` — `expect(() => StopArgs.parse({network:"local", extra:"bar"})).toThrow()`
- `_WireFailure_PassedThrough` — mock returns ok:false UPSTREAM_ERROR; isError:true

Use the env-save/restore pattern from `chain_read.test.ts`.

### Step 2.3: Verify + commit

```bash
cd mcp-server && npm test                # 94 + 3 = 97 (status reroute pending)
cd mcp-server && npx tsc --noEmit
cd mcp-server && npm run build
```

```bash
git add mcp-server/src/tools/lifecycle.ts mcp-server/test/lifecycle.test.ts
git commit -m "feat(mcp): reroute chainbench_stop through callWire

First lifecycle MCP reroute (Sprint 5c.4.1). chainbench_stop now
invokes the chainbench-net network.stop_all wire handler (added
in the previous commit) instead of shelling out to chainbench.sh.

The schema gains an optional 'network' field (defaults 'local')
so future remote-network workflows can opt in — current callers
that omit the arg see no surface change.

Three tests in the new lifecycle.test.ts cover the happy path,
strict-mode rejection of unknown keys (matches the 5c.3 NodeRpcArgs
precedent), and wire-failure passthrough.
"
```

---

## Task 3 — Go: `network.status` handler

**Files:**
- Modify: `network/cmd/chainbench-net/handlers_network.go` — add `newHandleNetworkStatus`
- Modify: `network/cmd/chainbench-net/handlers.go` — register
- Modify: `network/cmd/chainbench-net/handlers_test.go` — 4 unit tests
- Modify: `network/schema/command.json` — add `"network.status"` enum + regen

### Step 3.1: Schema + regen

Add `"network.status"` (alphabetical, after `network.probe`).

### Step 3.2: Handler

Same shape as Task 1 but:
- Spawns `chainbench.sh status --json` (always JSON output)
- Parses stdout as JSON via `json.Unmarshal` into `map[string]any`
- Returns the parsed map directly as the result envelope's data field
- Bash JSON parse failure → INTERNAL (invariant — bash is contracted to emit JSON)

### Step 3.3: Tests

4 tests in `handlers_test.go`:
- `TestHandleNetworkStatus_Happy` — fake script outputs `{"nodes":[...],"healthy":true}`, handler parses + returns map
- `TestHandleNetworkStatus_RemoteRejected`
- `TestHandleNetworkStatus_BadJsonOutput` → INTERNAL
- `TestAllHandlers_IncludesNetworkStatus`

### Step 3.4: Verify + commit

```bash
go -C network test ./cmd/chainbench-net/... -count=1 -v -run 'TestHandleNetworkStatus|TestAllHandlers_IncludesNetworkStatus'
go -C network test ./... -count=1 -timeout=60s
```

```bash
git add network/cmd/chainbench-net/handlers.go \
        network/cmd/chainbench-net/handlers_network.go \
        network/cmd/chainbench-net/handlers_test.go \
        network/schema/command.json \
        network/internal/types/command_gen.go
git commit -m "feat(network-net): network.status thin wrapper of chainbench status --json

Second lifecycle reroute handler. Mirrors network.stop_all's pattern
— spawn chainbench.sh status --json via os/exec, parse the JSON
output, return the parsed map as the wire result envelope's data
field. The bash CLI does the heavy lifting (387 lines of per-node
RPC queries + composite formatting); the Go handler is a thin
adapter into the wire protocol.

JSON parse failure on bash output is INTERNAL because bash is
contracted to emit JSON in this mode — a parse failure indicates
a regression in the bash side, not a caller bug.
"
```

---

## Task 4 — MCP: reroute `chainbench_status` + integration test

**Files:**
- Modify: `mcp-server/src/tools/lifecycle.ts`
- Modify: `mcp-server/test/lifecycle.test.ts` — add 3 tests
- Create: `mcp-server/test/integration/lifecycle.integration.test.ts`

### Step 4.1: Reroute

Replace `chainbench_status` registration in `lifecycle.ts`. Same shape as
stop but wire command `"network.status"`.

```typescript
export const StatusArgs = z.object({
  network: z.string().min(1).optional()
    .describe("Network alias. Defaults to 'local'."),
}).strict();

export async function _statusHandler(
  args: z.infer<typeof StatusArgs>,
): Promise<FormattedToolResponse> {
  const wireArgs: Record<string, unknown> = {};
  if (args.network !== undefined) wireArgs.network = args.network;
  const result = await callWire("network.status", wireArgs);
  return formatWireResult(result);
}

server.tool(
  "chainbench_status",
  "Return current node status as JSON. Includes per-node block height, " +
  "peer count, running state, and overall consensus health. Local network " +
  "only — remote networks reject (use chainbench_node_rpc for remote).",
  StatusArgs.shape,
  _statusHandler,
);
```

### Step 4.2: Unit tests (3 more in lifecycle.test.ts)

- `_StatusHappy` — mock binary returns ok:true with status JSON; text contains the JSON
- `_StatusStrictRejectsUnknownKeys`
- `_StatusWireFailure_PassedThrough`

### Step 4.3: Integration test

`mcp-server/test/integration/lifecycle.integration.test.ts` (NEW). Uses
`_harness.ts` from Task 0:

```typescript
import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { setupRealBinaryHarness, hasBinary, type RealBinaryHarness } from "./_harness.js";
import { writeFileSync, chmodSync } from "node:fs";
import { join } from "node:path";

describe.skipIf(!hasBinary())("integration: chainbench_status", () => {
  let harness: RealBinaryHarness;

  beforeAll(async () => {
    // Write fake chainbench.sh that emits a valid JSON status response.
    const fakeScript = `#!/bin/bash
if [[ "$1" == "status" && "$2" == "--json" ]]; then
  echo '{"nodes":[{"id":"node1","running":true,"block_height":42}],"healthy":true}'
  exit 0
fi
exit 1
`;
    harness = await setupRealBinaryHarness({
      // No RPC mock needed — status doesn't dial RPC; it spawns bash
      fakeChainbenchScript: fakeScript,
    });
  });

  afterAll(() => harness?.teardown());

  it("end-to-end status through chainbench-net", async () => {
    const { _statusHandler } = await import("../../src/tools/lifecycle.js");
    const out = await _statusHandler({});
    const text = out.content[0].text;
    expect(text).toContain("healthy");
    expect(text).toContain("block_height");
    expect(out.isError).toBeFalsy();
  });
});
```

This requires `_harness.ts` to support `fakeChainbenchScript` — Task 0
should have added that option. If not, add it now (write the script to
`<stateDir>/chainbench.sh` + chmod 0755 + set `CHAINBENCH_DIR=<stateDir>`).

### Step 4.4: Verify + commit

```bash
cd mcp-server && npm test                # 97 + 3 + 1 = 101
cd mcp-server && npx tsc --noEmit
cd mcp-server && npm run build
```

```bash
git add mcp-server/src/tools/lifecycle.ts \
        mcp-server/test/lifecycle.test.ts \
        mcp-server/test/integration/lifecycle.integration.test.ts
git commit -m "feat(mcp): reroute chainbench_status + integration test

Final reroute of Sprint 5c.4.1's two-tool batch. chainbench_status
now invokes network.status via callWire; the bash JSON output is
parsed Go-side and forwarded as a structured object so the LLM
sees the same JSON shape it always has.

Three new lifecycle.test.ts cases mirror chainbench_stop's
coverage — happy / strict-mode / wire-failure-passthrough.

A new integration test file under test/integration/ proves the
wire path works against the real chainbench-net binary (skipped
in CI without the Go toolchain), using the harness extracted in
Task 0 plus a fakeChainbenchScript option that lets the test seed
a stub chainbench.sh emitting a known JSON status response.
"
```

---

## Task 5 — Go: `node.start` accepts `binary_path` + remove MCP fallback

**Files:**
- Modify: `network/cmd/chainbench-net/handlers_node_lifecycle.go` — extend node.start
- Modify: `network/cmd/chainbench-net/handlers_test.go` — 1 new test
- Modify: `mcp-server/src/tools/node.ts` — remove fallback branch
- Modify: `mcp-server/test/node.test.ts` — adjust binary_path test
- Delete: `mcp-server/test/node_start_fallback.test.ts`

### Step 5.1: Go extension

Read `handlers_node_lifecycle.go`'s `newHandleNodeStart`. Add `BinaryPath`
field to the args struct. If `BinaryPath != ""`, validate it (`startsWith("/")`)
and pass it to whatever spawn happens internally (likely `chainbench.sh node
start ${node} --binary-path ${path}`).

### Step 5.2: Go test

Add `TestHandleNodeStart_BinaryPath` — pass `binary_path: "/abs/path"` in
args, assert the spawn command line includes `--binary-path /abs/path`. Use
the same fake-chainbench.sh + capture-args pattern as Task 1.

### Step 5.3: TS rewrite

In `mcp-server/src/tools/node.ts`, find `_nodeStartHandler`. Remove the
`if (args.binary_path !== undefined) { ... runChainbench(...) }` fallback
branch and instead pass `binary_path` to `wireArgs`:

```typescript
export async function _nodeStartHandler(args: z.infer<typeof NodeStartArgs>): Promise<FormattedToolResponse> {
  if (args.binary_path !== undefined) {
    if (args.binary_path.length === 0) return errorResp("binary_path must not be empty");
    if (!args.binary_path.startsWith("/")) return errorResp("binary_path must be an absolute path");
  }
  const wireArgs: Record<string, unknown> = {
    network: "local",
    node_id: `node${args.node}`,
  };
  if (args.binary_path !== undefined) wireArgs.binary_path = args.binary_path;
  const result = await callWire("node.start", wireArgs);
  return formatWireResult(result);
}
```

### Step 5.4: Test cleanup

Delete `mcp-server/test/node_start_fallback.test.ts` (vi.mock-based fallback
test no longer applies). Add to `mcp-server/test/node.test.ts`:

- `_NodeStart_BinaryPath_PassesToWire` — mock returns ok:true; verify
  binary_path was forwarded (via mock script that echoes envelope to stderr,
  or just trust formatWireResult shape since wire mock fixture doesn't
  capture envelope keys directly — pin via the bad-path validation tests
  instead which we keep)

### Step 5.5: Verify + commit

```bash
go -C network test ./... -count=1 -timeout=60s
cd mcp-server && npm test                # 101 - 2 (fallback) + 1 (new) = 100
```

```bash
git add network/cmd/chainbench-net/handlers_node_lifecycle.go \
        network/cmd/chainbench-net/handlers_test.go \
        mcp-server/src/tools/node.ts \
        mcp-server/test/node.test.ts
git rm mcp-server/test/node_start_fallback.test.ts
git commit -m "feat(network-net+mcp): node.start accepts binary_path; remove MCP fallback

Sprint 5c.3 left chainbench_node_start with a runChainbench
fallback branch when the caller supplied binary_path, because
the wire layer's node.start did not yet accept that arg. This
commit closes that asymmetry: the Go handler now reads binary_path
from args and threads it through to the internal node-spawn
command, and the MCP handler unconditionally calls callWire.

The fallback test file is deleted (vi.mock pattern no longer
needed); a new node.test.ts case verifies binary_path validation
fires on empty / relative paths before any wire call. Sprint
5c.3 P3 row 'chainbench_node_start binary_path falls back to
runChainbench' is closed.
"
```

---

## Task 6 — Docs + version bump 0.7.1

**Files:**
- Modify: `mcp-server/package.json` — 0.7.0 → 0.7.1
- Modify: `mcp-server/src/index.ts`
- Modify: `docs/EVALUATION_CAPABILITY.md`
- Modify: `docs/VISION_AND_ROADMAP.md`
- Modify: `docs/NEXT_WORK.md`
- Modify: `docs/REMAINING_WORK.md`

### Step 6.1: Version bump (patch — only 2/5 lifecycle rerouted)

`package.json` 0.7.0 → 0.7.1. `index.ts` `version: "0.7.1"`.

### Step 6.2: EVALUATION_CAPABILITY

§5 row:
- `**5c.4.1**` | MCP reroute pass 2 (lifecycle subset — chainbench_stop + _status) + integration harness extract + node.start binary_path Go support | ✅ 완료 (2026-05-04) |

§6 paragraph: "Sprint 5c.4.1 완료 (2026-05-04): lifecycle 6 tool 중 가장 단순한
2 tool (`chainbench_stop`, `chainbench_status`) reroute. Thin Go wrapper 패턴
(VISION §5.12 M2) — Go 핸들러는 `os/exec` 로 기존 bash `chainbench.sh stop|
status --json` spawn. Native Go 포팅 회피 (bash lifecycle 1,367 lines). 동시에
integration test harness 추출 (`_harness.ts`) 로 5c.3 P3 prerequisite 닫힘 +
`chainbench_node_start binary_path` Go 확장으로 5c.3 fallback 제거. Reroute
진행도 3/38 → 5/38 (~13%). 5c.4.2 가 나머지 4 lifecycle tool (init/start/
restart/clean) 흡수."

### Step 6.3: VISION_AND_ROADMAP

Header `최종 업데이트` → `2026-05-04 (Sprint 5c.4.1 완료)`

§6 갱신:
- `Sprint 5a` 그대로 (완료)
- `**Sprint 5c.4.1 — Lifecycle reroute pass 1 (stop + status + harness)**` 완료 박스
- `Sprint 5c.4.2 — Lifecycle reroute pass 2 (init/start/restart/clean)` (다음 P1)
- 5b / 5d 그대로

§5.17.7 갱신 짧게.

### Step 6.4: NEXT_WORK

§1 헤더 → `2026-05-04 (Sprint 5c.4.1 완료)`
§2.1 timeline row 추가
§3 P1 → Sprint 5c.4.2
§3 P3 갱신 — 닫힌 row 표시 (5c.3 binary_path / harness extract / cleanup-await),
새 row 없음 (또는 Sprint 5c.4.1 review minor 발견 시 추가)
§4.6 file size 표 — lifecycle.ts / _harness.ts / lifecycle.integration.test.ts
사이즈 추가

### Step 6.5: REMAINING_WORK

§2 빠른 상태 — Sprint 5c.4.1 추가, reroute 5/38 (~13%)
§4 P1 변경 — Sprint 5c.4.1 → Sprint 5c.4.2 (init/start/restart/clean 남음)
§4 5c.4.1 의 작업 완료 명시 (체크박스 또는 prose)
§9 권장 순서 — 다음은 5c.4.2 또는 5d

### Step 6.6: Verify + commit

```bash
cd mcp-server && npm test                                         # 100
cd mcp-server && npx tsc --noEmit && npm run build
go -C network test ./... -count=1 -timeout=60s                    # regression
bash tests/unit/run.sh                                             # regression
```

```bash
git add mcp-server/package.json mcp-server/src/index.ts \
        docs/EVALUATION_CAPABILITY.md docs/VISION_AND_ROADMAP.md \
        docs/NEXT_WORK.md docs/REMAINING_WORK.md
git commit -m "docs+chore(sprint-5c-4-1): roadmap + capability matrix + version 0.7.1

Sprint 5c.4.1 (lifecycle reroute pass 1) completes. The thin Go
wrapper pattern (VISION §5.12 M2) is validated end-to-end on the
two simplest lifecycle tools — chainbench_stop and chainbench_status
— before Sprint 5c.4.2 takes on the more complex init/start/restart
quartet.

EVALUATION_CAPABILITY §5 gains the 5c.4.1 row and §6 notes reroute
progress 3/38 → 5/38 (~13%). VISION_AND_ROADMAP marks 5c.4.1
complete and frames 5c.4.2 as the next P1. NEXT_WORK timeline
gains the row; the 5c.3 P3 entries for binary_path fallback,
harness extraction, and cleanup-await/port-race diagnostics are
all closed by this sprint.

mcp-server version 0.7.0 -> 0.7.1 (patch — 2 of 5 lifecycle tools
rerouted; the minor bump to 0.8.0 follows when 5c.4.2 closes the
remaining four).
"
```

---

## Final report (after all tasks)

Commit chain (~7-8 expected). vitest count delta (94 → 100). Go test count
delta (handlers_test.go +9, +1 for binary_path = 10). Reroute coverage
3/38 → 5/38 (~13%). Confirmed deferrals: 5c.4.2 (init/start/restart/clean),
5b/5d.
