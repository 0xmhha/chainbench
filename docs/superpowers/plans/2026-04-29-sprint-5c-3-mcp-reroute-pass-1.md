# Sprint 5c.3 — MCP Reroute (Pass 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Reroute the 3 `chainbench_node_*` MCP tools (stop/start/rpc) from
`runChainbench()` (bash CLI shell-out) to `callWire()` (chainbench-net wire).
Pre-work: extract `errorResp` and hex regex constants to shared `utils/`.
Add `node.rpc` Go handler (the only missing wire piece). Add a real-binary
integration test for one rerouted tool.

**Architecture:** No new high-level tools. Pre-work hoists 5c.2 P3 tech debt
(`errorResp` duplication, hex regex duplication) to `utils/mcpResp.ts` +
`utils/hex.ts`. New Go handler `node.rpc` is a thin generic JSON-RPC
passthrough using ethclient's underlying `rpc.Client.CallContext`. MCP
`node.ts` rewrites all three tool handlers to call `callWire(...)` and
formatWireResult. Real-binary integration test uses Python JSON-RPC mock
(Sprint 4 pattern) to spawn the actual chainbench-net binary.

**Tech Stack:** TypeScript + zod + vitest, Go (go-ethereum's rpc.Client). No
new dependencies.

Spec: `docs/superpowers/specs/2026-04-29-sprint-5c-3-mcp-reroute-pass-1.md`

---

## Commit Discipline

- English commit messages.
- NO `Co-Authored-By` trailer; NO "Generated with Claude Code"; NO emoji.
- Conventional Commits: `refactor(mcp):`, `feat(network-net):`, `feat(mcp):`,
  `test(mcp):`, `docs+chore(sprint-5c-3):`.

## File Structure

**Create:**
- `mcp-server/src/utils/mcpResp.ts` — FormattedToolResponse type + errorResp helper
- `mcp-server/src/utils/hex.ts` — hex regex + SIGNER_ALIAS + RPC_METHOD constants
- `mcp-server/test/mcpResp.test.ts` — errorResp test
- `mcp-server/test/node.test.ts` — wire-call unit tests for the 3 rerouted tools
- `mcp-server/test/integration/node_rpc.integration.test.ts` — real-binary test

**Modify:**
- `mcp-server/src/utils/wireResult.ts` — FormattedToolResponse moved to mcpResp; re-export for back-compat
- `mcp-server/src/tools/chain_read.ts` — drop local errorResp + hex consts; import from utils
- `mcp-server/src/tools/chain_tx.ts` — same
- `mcp-server/src/tools/node.ts` — rewrite 3 tools to use callWire
- `mcp-server/test/chain_read.test.ts` — import path adjustments only
- `mcp-server/test/chain_tx.test.ts` — same
- `mcp-server/vitest.config.ts` — include integration test path
- `network/cmd/chainbench-net/handlers_node_read.go` — newHandleNodeRpc
- `network/cmd/chainbench-net/handlers.go` — register node.rpc
- `network/cmd/chainbench-net/handlers_test.go` — node.rpc unit tests
- `mcp-server/package.json` — 0.5.0 → 0.6.0
- `mcp-server/src/index.ts` — McpServer version 0.6.0
- `docs/EVALUATION_CAPABILITY.md`
- `docs/VISION_AND_ROADMAP.md`
- `docs/NEXT_WORK.md`

---

## Task 0 — utils extraction

**Files:**
- Create: `mcp-server/src/utils/mcpResp.ts`
- Create: `mcp-server/src/utils/hex.ts`
- Create: `mcp-server/test/mcpResp.test.ts`
- Modify: `mcp-server/src/utils/wireResult.ts`
- Modify: `mcp-server/src/tools/chain_read.ts`
- Modify: `mcp-server/src/tools/chain_tx.ts`

### Step 0.1: `utils/mcpResp.ts`

```typescript
export interface FormattedToolResponse {
  content: Array<{ type: "text"; text: string; [k: string]: unknown }>;
  isError?: boolean;
  [k: string]: unknown;
}

export function errorResp(msg: string): FormattedToolResponse {
  return {
    content: [{ type: "text", text: `Error (INVALID_ARGS): ${msg}` }],
    isError: true,
  };
}
```

### Step 0.2: `utils/hex.ts`

Move all hex regexes + signer alias + rpc method regex from chain_read.ts /
chain_tx.ts to a single file with `export const` declarations. Same regex
literals — character-identical to current.

### Step 0.3: `wireResult.ts` re-export

`wireResult.ts` currently declares `FormattedToolResponse` inline. Replace with:

```typescript
import type { FormattedToolResponse } from "./mcpResp.js";
export type { FormattedToolResponse };
```

This keeps existing `import { FormattedToolResponse } from "./wireResult.js"`
working (back-compat re-export). Future cleanup can drop the re-export when
no callers reference it.

### Step 0.4: chain_read.ts and chain_tx.ts cleanups

- Remove local `errorResp` declarations.
- Remove local hex/regex const declarations.
- Add imports: `import { errorResp } from "../utils/mcpResp.js";` and `import { HEX_ADDRESS, HEX_DATA, ... } from "../utils/hex.js";`.
- Verify `_txSendHandler` and `_contractDeployHandler` still call the imported `errorResp`.

### Step 0.5: Test + commit

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/mcp-server
npm test           # 73 + 1 new mcpResp test = 74
npx tsc --noEmit
npm run build
```

```bash
git add mcp-server/src/utils/ mcp-server/src/tools/chain_read.ts \
        mcp-server/src/tools/chain_tx.ts mcp-server/test/mcpResp.test.ts
git commit -m "refactor(mcp): hoist errorResp + hex regex constants to utils/

Sprint 5c.3 prep. Sprint 5c.2 left the errorResp helper duplicated
between chain_read.ts and chain_tx.ts and the hex/regex constants
declared independently in three files (chain_read.ts, chain_tx.ts,
node.ts via implicit shapes). With Sprint 5c.3's reroute touching
node.ts and adding more tools that need the same shape, the rule
of three is met.

utils/mcpResp.ts hosts FormattedToolResponse and errorResp. The
former remains importable from utils/wireResult.ts via a type
re-export so existing call sites work unchanged. utils/hex.ts
hosts HEX_ADDRESS, HEX_DATA, HEX_TX_HASH, HEX_TOPIC, HEX_HEX,
HEX_STORAGE_KEY, SIGNER_ALIAS, and RPC_METHOD as module-level
const exports.

chain_read.ts and chain_tx.ts now import these instead of
declaring their own copies. Tests pass unchanged; tsc and build
clean."
```

---

## Task 1 — Go: `node.rpc` handler

**Files:**
- Modify: `network/cmd/chainbench-net/handlers_node_read.go` — add `newHandleNodeRpc`
- Modify: `network/cmd/chainbench-net/handlers.go` — register `"node.rpc"`
- Modify: `network/cmd/chainbench-net/handlers_test.go` — unit tests

### Step 1.1: Implement

```go
func newHandleNodeRpc(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network string          `json:"network"`
            NodeID  string          `json:"node_id"`
            Method  string          `json:"method"`
            Params  json.RawMessage `json:"params"`
        }
        if len(args) > 0 {
            if err := json.Unmarshal(args, &req); err != nil {
                return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
            }
        }
        if req.Method == "" {
            return nil, NewInvalidArgs("args.method is required")
        }
        if !rpcMethodRe.MatchString(req.Method) {
            return nil, NewInvalidArgs(fmt.Sprintf("args.method invalid: %q", req.Method))
        }

        // Decode params: must be a JSON array OR null/missing.
        var paramsList []any
        if len(req.Params) > 0 && string(req.Params) != "null" {
            if err := json.Unmarshal(req.Params, &paramsList); err != nil {
                return nil, NewInvalidArgs(fmt.Sprintf("args.params must be a JSON array or null: %v", err))
            }
        }

        _, node, err := resolveNode(stateDir, req.Network, req.NodeID)
        if err != nil {
            return nil, err
        }
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        client, err := dialNode(ctx, &node)
        if err != nil {
            return nil, err
        }
        defer client.Close()

        var raw json.RawMessage
        if err := client.Client().CallContext(ctx, &raw, req.Method, paramsList...); err != nil {
            return nil, NewUpstream(req.Method, err)
        }
        return map[string]any{"result": raw}, nil
    }
}

var rpcMethodRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
```

(Adjust imports: `regexp`, `time`, `context`, `encoding/json` — most already
present.)

### Step 1.2: Register

In `handlers.go`'s `allHandlers`, add:
```go
"node.rpc": newHandleNodeRpc(stateDir),
```

### Step 1.3: Tests (`handlers_test.go`)

5 tests, mocking with httptest:
- `TestHandleNodeRpc_Happy_BlockNumber` — mock returns `{"jsonrpc":"2.0","result":"0x10","id":1}`, handler returns `result: "0x10"`
- `TestHandleNodeRpc_BadMethod` (e.g., `"eth blockNumber"` with space) → INVALID_ARGS
- `TestHandleNodeRpc_BadParams` (params is object not array) → INVALID_ARGS
- `TestHandleNodeRpc_RpcFailure` (mock returns JSON-RPC error) → UPSTREAM_ERROR
- `TestAllHandlers_IncludesNodeRpc` — registration check

### Step 1.4: Verify + commit

```bash
go -C network test ./... -count=1 -timeout=60s
go -C network vet ./... && gofmt -l network/
```

```bash
git add network/cmd/chainbench-net/handlers.go \
        network/cmd/chainbench-net/handlers_node_read.go \
        network/cmd/chainbench-net/handlers_test.go
git commit -m "feat(network-net): node.rpc generic JSON-RPC passthrough handler

Schema enum had 'node.rpc' since Sprint 2 but no Go handler — the
old YAGNI shelf since the bash chainbench_node_rpc tool covered
the use case via runChainbench. Sprint 5c.3 reroutes the MCP
node.rpc tool through the wire helper, so the handler now needs
to exist on the Go side.

newHandleNodeRpc validates the method against an alphanumeric_+
regex (matches the existing TS-side check), decodes params as a
JSON array or null, resolves the node via resolveNode, and
delegates to ethclient's underlying rpc.Client.CallContext.
Result is the raw JSON-RPC response wrapped in a {result: ...}
envelope so the MCP layer can pretty-print or pass through.

Tests cover happy-path block_number, bad method names, malformed
params, RPC-side errors, and dispatcher registration."
```

---

## Task 2 — MCP: reroute `chainbench_node_rpc`

**Files:**
- Modify: `mcp-server/src/tools/node.ts`
- Create: `mcp-server/test/node.test.ts`

### Step 2.1: Rewrite the rpc tool

In `node.ts`:
```typescript
import { callWire } from "../utils/wire.js";
import { formatWireResult } from "../utils/wireResult.js";
import { RPC_METHOD } from "../utils/hex.js";

server.tool(
  "chainbench_node_rpc",
  "Send a JSON-RPC call directly to a specific node and return the response. " +
  "Useful for querying state or sending transactions via raw RPC. The 1-based " +
  "node index is mapped to the wire layer's node_id ('node1', 'node2', ...).",
  {
    node: NODE_INDEX_SCHEMA,
    method: z.string().regex(RPC_METHOD).describe("JSON-RPC method name"),
    params: z.string().optional().describe("JSON array of parameters; defaults to '[]'."),
  },
  async ({ node, method, params }) => {
    let parsedParams: unknown[] = [];
    if (params !== undefined) {
      try {
        const parsed = JSON.parse(params);
        if (!Array.isArray(parsed)) {
          return errorResp("params must be a JSON array");
        }
        parsedParams = parsed;
      } catch {
        return errorResp("params is not valid JSON");
      }
    }
    const result = await callWire("node.rpc", {
      network: "local",
      node_id: `node${node}`,
      method,
      params: parsedParams,
    });
    return formatWireResult(result);
  },
);
```

Notes:
- `RPC_METHOD` regex now lives in `utils/hex.ts` (Task 0)
- `errorResp` from `utils/mcpResp.ts`
- 1-based int → `nodeN` string conversion (matches state file format)

### Step 2.2: Tests in `node.test.ts`

Mirror chain_read.test.ts patterns. Use the existing mock fixture
(`mock-chainbench-net.mjs`) via `CHAINBENCH_NET_BIN` env override.

- `_Happy_NodeRpc` — `eth_blockNumber` happy path; mock emits `{result: "0x10"}` data; assert text contains `0x10`
- `_BadMethod_RejectedAtBoundary` — zod regex rejects `"eth blockNumber"`
- `_BadParams_NotArray_ReturnsIsError` — `params: '{"x":1}'` → handler returns errorResp
- `_BadParams_NotJSON_ReturnsIsError` — `params: 'not-json'` → errorResp
- `_WireFailure_PassedThrough` — mock ok:false, handler returns isError:true

### Step 2.3: Verify + commit

```bash
cd mcp-server && npm test           # 74 + 5 = 79 (excluding integration)
cd mcp-server && npx tsc --noEmit
cd mcp-server && npm run build
```

```bash
git add mcp-server/src/tools/node.ts mcp-server/test/node.test.ts
git commit -m "feat(mcp): reroute chainbench_node_rpc through callWire

First reroute of Sprint 5c.3. chainbench_node_rpc now invokes the
chainbench-net node.rpc wire handler (added in the previous
commit) instead of shelling out to chainbench.sh. The 1-based
node index argument is preserved; the wire layer's node_id
('node1', 'node2', ...) is constructed by string concatenation.

The zod schema is unchanged (method regex + params JSON array
string) — LLM callers see no surface change. The handler now
parses params eagerly to a JS array before passing to callWire,
catching malformed JSON at the boundary with errorResp instead
of letting the bash layer return a shell error string."
```

---

## Task 3 — MCP: reroute `chainbench_node_stop` and `chainbench_node_start`

**Files:**
- Modify: `mcp-server/src/tools/node.ts`
- Modify: `mcp-server/test/node.test.ts`

### Step 3.1: Rewrite handlers

`chainbench_node_stop`:
```typescript
async ({ node }) => {
  const result = await callWire("node.stop", {
    network: "local",
    node_id: `node${node}`,
  });
  return formatWireResult(result);
}
```

`chainbench_node_start`: tricky because of `binary_path` arg.
- Wire `node.start` does NOT accept `binary_path` (per Sprint 2b state).
- For Sprint 5c.3 Pass 1: when `binary_path` is provided, fall back to
  `runChainbench(...)` to preserve behavior; when absent, use `callWire`.
- Document this asymmetry in a code comment + add a P3 row in NEXT_WORK to
  drop the fallback once Go-side `node.start` accepts `binary_path`.

```typescript
async ({ node, binary_path }) => {
  if (binary_path !== undefined) {
    // Validate per existing rules
    if (binary_path.length === 0) return errorResp("binary_path must not be empty");
    if (!binary_path.startsWith("/")) return errorResp("binary_path must be an absolute path");
    // Wire layer doesn't yet accept binary_path; fall back to bash.
    // TODO Sprint 5c.4: extend node.start wire handler to take binary_path.
    const result = runChainbench(`node start ${node} --binary-path ${shellEscapeArg(binary_path)}`);
    return { content: [{ type: "text" as const, text: result.stdout || result.stderr || "Done." }],
             isError: result.exitCode !== 0 };
  }
  const result = await callWire("node.start", {
    network: "local",
    node_id: `node${node}`,
  });
  return formatWireResult(result);
}
```

### Step 3.2: Tests

Add 5 tests to `node.test.ts`:
- `_NodeStop_Happy` — wire success
- `_NodeStop_WireFailure_PassedThrough`
- `_NodeStart_NoBinaryPath_UsesWire` — no binary_path → wire path; verify
  envelope shape via callWire spy if practical, or via mock fixture response
- `_NodeStart_WithBinaryPath_FallsBackToBash` — runs runChainbench; verify
  with vi.mock on `../utils/exec.js` (similar to chain_read_timeout.test.ts
  pattern)
- `_NodeStart_BadBinaryPath_ReturnsError` — empty / relative path → errorResp

### Step 3.3: Verify + commit

```bash
cd mcp-server && npm test           # 79 + 5 = 84
```

```bash
git commit -m "feat(mcp): reroute chainbench_node_stop and node_start through callWire

Second reroute of Sprint 5c.3. node.stop has no extra args so the
migration is mechanical: 1-based int -> 'nodeN' string ID, then
callWire('node.stop', ...). node.start splits two ways:
  - If binary_path is omitted (the common case), the wire path
    fires.
  - If binary_path is supplied, the handler falls back to
    runChainbench because the chainbench-net node.start wire
    handler does not yet accept that arg. Sprint 5c.4 will extend
    the Go side and remove this fallback.

The tx-level fallback is documented inline and tracked in NEXT_WORK
as a P3 row so the next reroute pass closes it."
```

---

## Task 4 — Real-binary integration test (or fallback)

**Files:**
- Create: `mcp-server/test/integration/node_rpc.integration.test.ts`
- Modify: `mcp-server/vitest.config.ts` (include integration path) or add separate config

### Step 4.1: Build chainbench-net binary

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network
go build -o bin/chainbench-net ./cmd/chainbench-net
```

This must succeed before the integration test can run. Test setup checks for
the binary's existence and `it.skip`s the suite if missing (so CI without Go
toolchain still passes).

### Step 4.2: Test scenario

The test:
1. Spawns a Python JSON-RPC mock server (port-by-fixture) that responds to
   `eth_blockNumber` with `0xdeadbeef`
2. Creates a temp `state/` dir with `current-profile.yaml` + `pids.json`
   pointing to the mock URL (or uses `network.attach` first to register a
   "remote" network pointing at the mock)
3. Sets `process.env.CHAINBENCH_NET_BIN` to the built binary
4. Calls `_nodeRpcHandler({node: 1, method: "eth_blockNumber"})` (or invokes
   via callWire directly)
5. Asserts the response contains `0xdeadbeef`

### Step 4.3: Decide

If the integration test scaffolding becomes too brittle within a single task,
fallback: keep just the unit-level mock fixture coverage (already done in
Tasks 0-3), and defer real-binary integration to Sprint 5c.4 as a separate
task. Update NEXT_WORK §3 P3 to keep the integration-test row open.

### Step 4.4: Commit (if integration test lands)

```bash
git commit -m "test(mcp): real-binary integration test for chainbench_node_rpc

First end-to-end test that builds the chainbench-net binary,
spawns a Python JSON-RPC mock, and verifies callWire's contract
holds against the actual Go handler — not just a Node.js mock
fixture. The suite skips itself if the binary is missing
(GO toolchain absent in CI) so unit tests still run.

Closes the 5c.2 P3 row about absent vitest integration tests
for the rerouted tool path."
```

If fallback chosen, no commit; just note in Task 5.

---

## Task 5 — Docs + version bump 0.6.0

**Files:**
- Modify: `mcp-server/package.json` — 0.5.0 → 0.6.0
- Modify: `mcp-server/src/index.ts`
- Modify: `docs/EVALUATION_CAPABILITY.md`
- Modify: `docs/VISION_AND_ROADMAP.md`
- Modify: `docs/NEXT_WORK.md`

### Step 5.1: Version bump

`package.json` 0.5.0 → 0.6.0. `index.ts` `version: "0.6.0"`.

### Step 5.2: EVALUATION_CAPABILITY.md

§5 — Sprint 5c.3 row: "MCP reroute pass 1 — utils extraction + node.rpc Go
handler + 3 chainbench_node_* reroute (~8% of 38) + integration test layer".
✅ 완료 (2026-04-29).

§6 — coverage tally line: "Reroute progress: 3/38 (~8%) — 5c.4+ continues".
MCP high-level tool count (~60%) doesn't change since this sprint adds 0
high-level surface.

### Step 5.3: VISION_AND_ROADMAP.md

§6 Sprint 5 갱신:
- `Sprint 5c.3 — MCP reroute (pass 1)` 완료 박스
- `Sprint 5c.4 — MCP reroute (pass 2: lifecycle)` 미완 (다음 P1)
- 5a / 5b / 5d 그대로

§5.17.7 — 짧은 갱신: "Sprint 5c.3 (2026-04-29) 에서 utils 추출 + node.rpc
Go 핸들러 + 3 chainbench_node_* reroute. 나머지 35 tool 은 5c.4+ 패스."

### Step 5.4: NEXT_WORK.md

- 헤더 `최종 업데이트` → `2026-04-29 (Sprint 5c.3 완료 — Sprint 5 series 진행 중)`
- §2.1 timeline 표 — 5c.3 row 추가 (commit count: spec/plan + Task 0 utils +
  Task 1 Go + Task 2 reroute + Task 3 reroute + Task 4 integration (optional)
  + Task 5 docs = 6-7)
- §2.3 — vitest 73 → 84 (or so depending on integration count); Go test
  coverage tick (1 new handler, 5 tests)
- §3 P1 — 다음 P1 = Sprint 5c.4 (lifecycle reroute)
- §3 P3 — 다음 row 추가:
  - `5c.3: chainbench_node_start binary_path fallback to runChainbench` —
    wire 측 node.start 가 binary_path 미지원 — 5c.4 가 Go 확장 시 제거
  - 만약 integration test fallback 했다면: `5c.3: real-binary integration
    test deferred` row 유지
- 최근 완료 entry 추가

### Step 5.5: Verify + commit

```bash
cd mcp-server && npm test
cd mcp-server && npx tsc --noEmit && npm run build
go -C network test ./... -count=1 -timeout=60s
bash tests/unit/run.sh
```

```bash
git commit -m "docs+chore(sprint-5c-3): roadmap + capability matrix + version 0.6.0

Sprint 5c.3 first reroute pass completes. EVALUATION_CAPABILITY
gains a §5 row noting reroute progress 3/38 (~8%); the §6 MCP
coverage figure is unchanged since this sprint adds zero
high-level surface (its job was the lift-and-shift of three
existing tools).

VISION_AND_ROADMAP Sprint 5c.3 marked complete; next P1 is
5c.4 (lifecycle: chainbench_init/start/stop/restart/status/clean
reroute, requires Go-side wire handlers for the missing
commands).

NEXT_WORK timeline gains the 5c.3 row; tech debt picks up the
chainbench_node_start binary_path fallback (5c.4 closes it via
Go extension).

mcp-server version 0.5.0 -> 0.6.0 (internal reroute milestone;
no LLM-facing surface changes)."
```

---

## Final report (after all tasks)

Commit chain (~6-7 expected). vitest count delta. Go test count delta.
Reroute coverage 3/38. Confirmed deferrals: 5c.4 (35 remaining tools, in
domain-grouped passes), 5a/5b/5d (capability/SSH/hybrid). Possible follow-up
P3 if integration test fell back.
