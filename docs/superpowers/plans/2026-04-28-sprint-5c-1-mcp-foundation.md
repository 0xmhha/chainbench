# Sprint 5c.1 — MCP Foundation + 2 High-Level Tools Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Build the TypeScript wire-spawn helper, NDJSON result transformer,
and two representative high-level MCP tools (`chainbench_account_state`,
`chainbench_tx_send` with `mode: legacy|1559`) on top of the Sprint 4 series
Go capability. Sprint 5 시리즈의 첫 패스 — 5c.2 (남은 4 tool + 추가 mode) /
5c.3 (기존 tool reroute) 가 후속.

**Architecture:** `mcp-server/src/utils/wire.ts` 가 `chainbench-net run` 을
spawn (binary resolution 은 bash `lib/network_client.sh` 와 동일 순서: env →
$CHAINBENCH_DIR/bin → $CHAINBENCH_DIR/network/bin → PATH 분해 후 existsSync).
NDJSON envelope stdin, NDJSON stream stdout. `wireResult.ts` 가 events/progress
/result 를 MCP text content + isError 응답으로 집계. 두 신규 tool 은 zod 로
boundary 검증 후 wire 호출. signer alias 만 받고 key material 은 host env 에
의존 (`CHAINBENCH_SIGNER_<ALIAS>_KEY` / `_KEYSTORE` / `_KEYSTORE_PASSWORD`).

**Tech Stack:** TypeScript + `@modelcontextprotocol/sdk` (>=1.0), `zod`, Node
`child_process.spawn` / `readline`. 테스트는 `vitest` (devDep 추가). Mock 바이너리는
`fixtures/mock-chainbench-net.mjs` (Node ESM 단독 실행).

Spec: `docs/superpowers/specs/2026-04-28-sprint-5c-1-mcp-foundation.md`

---

## Commit Discipline

- English commit messages.
- NO `Co-Authored-By` trailer; NO "Generated with Claude Code"; NO emoji.
- Conventional Commits prefix per sprint convention (`feat(mcp):`, `test(mcp):`,
  `docs(...)`, `chore(mcp):`).

## File Structure

**Create:**
- `mcp-server/src/utils/wire.ts` — spawn + NDJSON parser
- `mcp-server/src/utils/wireResult.ts` — wire result → MCP response formatter
- `mcp-server/src/tools/chain.ts` — `chainbench_account_state` +
  `chainbench_tx_send` registration
- `mcp-server/test/wire.test.ts`
- `mcp-server/test/wireResult.test.ts`
- `mcp-server/test/chain.test.ts`
- `mcp-server/test/fixtures/mock-chainbench-net.mjs` — programmable NDJSON mock
- `mcp-server/vitest.config.ts`

**Modify:**
- `mcp-server/src/index.ts` — `registerChainTools(server)` 추가
- `mcp-server/package.json` — vitest devDep + test script + version bump 0.3.0 → 0.4.0
- `mcp-server/tsconfig.json` — test 디렉토리 컴파일 제외 / include 조정 (필요 시)
- `docs/EVALUATION_CAPABILITY.md` — MCP column 3 cell ✅ + §6 coverage 갱신
- `docs/VISION_AND_ROADMAP.md` — Sprint 5c.1 row + §6 Sprint 5 분할
- `docs/NEXT_WORK.md` — §2.1 / §3 / §4.6 갱신

---

## Task 1 — Wire helper (`utils/wire.ts`) + mock binary + tests

**Files:**
- Create: `mcp-server/src/utils/wire.ts`
- Create: `mcp-server/test/wire.test.ts`
- Create: `mcp-server/test/fixtures/mock-chainbench-net.mjs`
- Create: `mcp-server/vitest.config.ts`
- Modify: `mcp-server/package.json` (vitest devDep + script)

### Step 1.1: vitest 셋업

```jsonc
// package.json devDependencies
"vitest": "^1.6.0"

// scripts
"test": "vitest run",
"test:watch": "vitest"
```

`mcp-server/vitest.config.ts`:
```typescript
import { defineConfig } from "vitest/config";
export default defineConfig({
  test: {
    include: ["test/**/*.test.ts"],
    environment: "node",
    testTimeout: 20000,
  },
});
```

`tsconfig.json` 의 `include` 가 `test/` 를 포함하지 않게 — test 는 vitest 가
ts-on-the-fly 로 실행. (`include: ["src/**/*"]` 만 유지.)

### Step 1.2: Mock chainbench-net

`mcp-server/test/fixtures/mock-chainbench-net.mjs` — 실행되면 stdin 에서 한 줄
JSON envelope 을 읽고 `MOCK_SCRIPT` 환경변수의 시나리오대로 NDJSON 라인을
emit. 시나리오는 base64 JSON encoded array of `{kind, line, delayMs}`.

```javascript
#!/usr/bin/env node
import { stdin, stdout, stderr } from "node:process";

const script = JSON.parse(
  Buffer.from(process.env.MOCK_SCRIPT ?? "W10=", "base64").toString("utf-8"),
);

const envelopeChunks = [];
stdin.on("data", (b) => envelopeChunks.push(b));
stdin.on("end", async () => {
  const envelope = Buffer.concat(envelopeChunks).toString("utf-8").trim();
  // Echo envelope to stderr as a debug line (slog-shape JSON).
  stderr.write(JSON.stringify({ level: "DEBUG", msg: "envelope", envelope }) + "\n");

  for (const step of script) {
    if (step.delayMs) await new Promise((r) => setTimeout(r, step.delayMs));
    if (step.kind === "stdout") stdout.write(step.line + "\n");
    else if (step.kind === "stderr") stderr.write(step.line + "\n");
    else if (step.kind === "exit") process.exit(step.code ?? 0);
  }
  process.exit(process.env.MOCK_EXIT ? Number(process.env.MOCK_EXIT) : 0);
});
```

테스트가 시나리오를 base64 인코딩 후 환경변수로 넘김.

### Step 1.3: `wire.ts` 본체

타입 + `resolveBinary()` + `callWire()`. Spec §4.1 그대로 시그니처. 핵심 로직:

```typescript
import { existsSync } from "node:fs";
import { spawn } from "node:child_process";
import { resolve as resolvePath, dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import readline from "node:readline";

const __dirname = dirname(fileURLToPath(import.meta.url));

export type WireEventLine = {
  type: "event";
  name: string;
  data?: Record<string, unknown>;
  ts?: string;
};
export type WireProgressLine = {
  type: "progress";
  step: string;
  done?: number;
  total?: number;
};
export type WireResultLine =
  | { type: "result"; ok: true; data: Record<string, unknown> }
  | { type: "result"; ok: false; error: { code: string; message: string; details?: unknown } };

export interface WireCallResult {
  result: WireResultLine;
  events: WireEventLine[];
  progress: WireProgressLine[];
  stderr: string;
  exitCode: number;
}

export interface WireCallOptions {
  envOverrides?: Record<string, string>;
  timeoutMs?: number;
  binaryPath?: string;
}

const PATH_SEP = process.platform === "win32" ? ";" : ":";
const BINARY_NAME = "chainbench-net";

export function resolveBinary(): string {
  const fromEnv = process.env.CHAINBENCH_NET_BIN;
  if (fromEnv && existsSync(fromEnv)) return fromEnv;

  const dir = process.env.CHAINBENCH_DIR;
  if (dir) {
    const a = resolvePath(dir, "bin", BINARY_NAME);
    if (existsSync(a)) return a;
    const b = resolvePath(dir, "network", "bin", BINARY_NAME);
    if (existsSync(b)) return b;
  }

  // PATH lookup — split process.env.PATH and check each entry; no shell exec.
  const pathEntries = (process.env.PATH ?? "").split(PATH_SEP).filter(Boolean);
  for (const entry of pathEntries) {
    const candidate = join(entry, BINARY_NAME);
    if (existsSync(candidate)) return candidate;
  }

  throw new Error("chainbench-net binary not found");
}

const DEFAULT_TIMEOUT_MS = 120000;

export async function callWire(
  command: string,
  args: Record<string, unknown>,
  options: WireCallOptions = {},
): Promise<WireCallResult> {
  const bin = options.binaryPath ?? resolveBinary();
  const env = { ...process.env, ...(options.envOverrides ?? {}) };
  const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;

  const child = spawn(bin, ["run"], { env, stdio: ["pipe", "pipe", "pipe"] });

  const events: WireEventLine[] = [];
  const progress: WireProgressLine[] = [];
  let result: WireResultLine | null = null;
  let stderrBuf = "";

  const rl = readline.createInterface({ input: child.stdout });
  rl.on("line", (line) => {
    const trimmed = line.trim();
    if (!trimmed.startsWith("{")) return;
    let parsed: unknown;
    try {
      parsed = JSON.parse(trimmed);
    } catch {
      return; // 방어적: malformed line silent skip
    }
    const obj = parsed as { type?: string };
    if (obj.type === "event") events.push(obj as WireEventLine);
    else if (obj.type === "progress") progress.push(obj as WireProgressLine);
    else if (obj.type === "result" && result === null) result = obj as WireResultLine;
  });

  child.stderr.on("data", (b) => { stderrBuf += b.toString("utf-8"); });

  // envelope write
  const envelope = JSON.stringify({ command, args }) + "\n";
  child.stdin.write(envelope);
  child.stdin.end();

  const exitCode = await new Promise<number>((resolveExit, rejectExit) => {
    let timer: NodeJS.Timeout | null = null;
    const onExit = (code: number | null, signal: NodeJS.Signals | null) => {
      if (timer) clearTimeout(timer);
      resolveExit(code ?? (signal ? 128 : 0));
    };
    child.once("error", rejectExit);
    child.once("close", onExit);
    timer = setTimeout(() => {
      child.kill("SIGTERM");
      rejectExit(new Error(`chainbench-net timeout after ${timeoutMs}ms`));
    }, timeoutMs);
  });

  // readline 이 stdout close 후에도 buffer 잔여를 처리하므로 다음 microtask 까지 대기
  await new Promise((r) => setImmediate(r));

  if (result === null) {
    throw new Error(
      `chainbench-net produced no result terminator (exit ${exitCode}). stderr: ${stderrBuf.slice(0, 500)}`,
    );
  }

  return { result, events, progress, stderr: stderrBuf, exitCode };
}
```

**경계 사항**:
- `result === null` 보호로 mid-stream parse 실수 방어
- timeout 은 `child.kill('SIGTERM')` + reject — kill 후 process exit 까지의
  잔여 데이터는 무시
- `setImmediate` await 로 readline buffer drain (close 이벤트 후 line 이벤트가
  남을 수 있음)
- PATH lookup 은 shell 없이 `process.env.PATH` 분해 + `existsSync` — 인젝션
  공격 표면 없음
- `existsSync` / `process.env.CHAINBENCH_NET_BIN` 우선순위는 spec §4.1 그대로

### Step 1.4: 테스트 (RED → GREEN)

`mcp-server/test/wire.test.ts` — 9 tests (spec §6.1):

```typescript
import { describe, it, expect } from "vitest";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { callWire } from "../src/utils/wire.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const MOCK_BIN = resolve(__dirname, "fixtures/mock-chainbench-net.mjs");

function script(steps: Array<{kind:"stdout"|"stderr"|"exit", line?:string, code?:number, delayMs?:number}>): string {
  return Buffer.from(JSON.stringify(steps), "utf-8").toString("base64");
}

// 9 tests, each spawning Node with the mock fixture as the binary.
// resolveBinary 테스트 3개는 process.env / fs 의 임시 fixture 로 검증.
```

각 테스트의 mock 시나리오:
- Happy: `progress` 2 + `event` 1 + `result ok:true`
- Failure: `result ok:false` + `error.code`
- NoTerminator: stdout 닫고 exit 0
- Timeout: 영구 sleep (delayMs: 60000) + timeoutMs:200
- EnvOverrides: result.data 에 `process.env.FOO` echo (mock 의 stdout 라인을
  envOverrides 로 control — `MOCK_SCRIPT` 와 `FOO` 둘 다 envOverrides 로 전달)
- NonZeroExit: ok:true result + exit 1

`binaryPath: 'node ' + MOCK_BIN` 로는 spawn 못함 — 대신 `binaryPath: '/usr/bin/env'`
+ args 를 변경할 수 없음. 해결: mock 파일 첫 줄 `#!/usr/bin/env node` + 실행권
한 부여 후 `binaryPath: MOCK_BIN` 직접 spawn.

테스트 setup 에서 `fs.chmodSync(MOCK_BIN, 0o755)` 1회 보장.

### Step 1.5: 검증 + 커밋

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/mcp-server && npm install
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/mcp-server && npm test
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/mcp-server && npx tsc --noEmit
```

```bash
git add mcp-server/src/utils/wire.ts mcp-server/test/wire.test.ts \
        mcp-server/test/fixtures/mock-chainbench-net.mjs \
        mcp-server/vitest.config.ts mcp-server/package.json mcp-server/package-lock.json
git commit -m "feat(mcp): wire helper for spawning chainbench-net with NDJSON streams

mcp-server/src/utils/wire.ts spawns the chainbench-net binary, feeds an
NDJSON envelope on stdin, and accumulates events / progress / result
lines from stdout. Binary resolution mirrors lib/network_client.sh:
CHAINBENCH_NET_BIN env first, then \$CHAINBENCH_DIR/bin, then
\$CHAINBENCH_DIR/network/bin, finally a shell-free PATH split that
checks each entry with existsSync.

callWire() returns the captured streams plus exit code; missing result
terminator throws (the binary contracts to always emit one). Caller
checks result.ok to distinguish success from API error — non-zero exit
codes alone are not treated as failure when a result line was seen.

Test fixture mock-chainbench-net.mjs is a Node ESM script that emits
caller-defined NDJSON sequences via the MOCK_SCRIPT env var, so the
nine wire.test.ts cases run without building the Go binary."
```

---

## Task 2 — Result transformer (`wireResult.ts`) + tests

**Files:**
- Create: `mcp-server/src/utils/wireResult.ts`
- Create: `mcp-server/test/wireResult.test.ts`

### Step 2.1: 테스트 먼저

`wireResult.test.ts` — 5 tests (spec §6.2). 입력은 `WireCallResult` 객체 직접
구성 — wire helper 의존성 없음.

```typescript
import { describe, it, expect } from "vitest";
import { formatWireResult } from "../src/utils/wireResult.js";
import type { WireCallResult } from "../src/utils/wire.js";

const baseSuccess = (data: Record<string, unknown>): WireCallResult => ({
  result: { type: "result", ok: true, data },
  events: [],
  progress: [],
  stderr: "",
  exitCode: 0,
});
```

테스트 5개:
- `_Success_RendersDataPretty` — data `{balance:"0x10"}` → text 가 JSON pretty.
  `isError` 누락 (or false)
- `_Success_EmptyData_RendersDoneText` — data `{}` → text === "Done."
- `_Success_WithPhasesSummary` — progress 3개 + event 2개 → text 에 "Phases: ..."
  + "Events: ..." 포함
- `_Failure_RendersErrorWithCode` — ok:false code "INVALID_ARGS" → text 가
  "Error (INVALID_ARGS): <message>"; isError true
- `_Failure_WithLastPhaseHint` — failure + progress N개 → 마지막 step 이 "Last
  phase: <step>" 로 노출

### Step 2.2: 구현

```typescript
import type { WireCallResult } from "./wire.js";

export interface FormattedToolResponse {
  content: Array<{ type: "text"; text: string }>;
  isError?: boolean;
}

const MAX_NAMES_IN_SUMMARY = 8;

export function formatWireResult(result: WireCallResult): FormattedToolResponse {
  const { result: r, events, progress } = result;

  if (r.ok === false) {
    const lines: string[] = [];
    lines.push(`Error (${r.error.code}): ${r.error.message || "(no message)"}`);
    if (progress.length > 0) {
      const last = progress[progress.length - 1];
      lines.push(`Last phase: ${last.step}`);
    }
    return {
      content: [{ type: "text", text: lines.join("\n") }],
      isError: true,
    };
  }

  // success
  const dataKeys = Object.keys(r.data);
  const lines: string[] = [];
  if (dataKeys.length === 0) {
    lines.push("Done.");
  } else {
    lines.push(JSON.stringify(r.data, null, 2));
  }
  if (progress.length > 0) {
    const names = progress.slice(0, MAX_NAMES_IN_SUMMARY).map((p) => p.step);
    const overflow = progress.length > MAX_NAMES_IN_SUMMARY ? ` ...+${progress.length - MAX_NAMES_IN_SUMMARY}` : "";
    lines.push(`Phases: [${progress.length}] ${names.join(", ")}${overflow}`);
  }
  if (events.length > 0) {
    const names = events.slice(0, MAX_NAMES_IN_SUMMARY).map((e) => e.name);
    const overflow = events.length > MAX_NAMES_IN_SUMMARY ? ` ...+${events.length - MAX_NAMES_IN_SUMMARY}` : "";
    lines.push(`Events: [${events.length}] ${names.join(", ")}${overflow}`);
  }
  return { content: [{ type: "text", text: lines.join("\n") }] };
}
```

### Step 2.3: 검증 + 커밋

```bash
cd mcp-server && npm test
cd mcp-server && npx tsc --noEmit
```

```bash
git add mcp-server/src/utils/wireResult.ts mcp-server/test/wireResult.test.ts
git commit -m "feat(mcp): wireResult formatter from NDJSON streams to MCP response

formatWireResult turns a WireCallResult into the {content,isError}
shape the MCP SDK expects. Success path pretty-prints the data field
and (when present) appends compact 'Phases: [n] step1, step2, ...'
and 'Events: [n] name1, name2, ...' summary lines so the LLM sees
the structured outcome plus a coarse trace of what the binary did.
Failure path renders 'Error (CODE): message' and, when a phase trace
exists, appends 'Last phase: <step>' to point the LLM at where the
work stopped. Phase / event lists cap at the first 8 entries with a
'...+N' suffix so long-running calls don't bloat responses."
```

---

## Task 3 — `chainbench_account_state` tool

**Files:**
- Create: `mcp-server/src/tools/chain.ts` (이번 task 에서 read tool 만 담음)
- Create: `mcp-server/test/chain.test.ts` (read 테스트만 — write 는 Task 4 에서 추가)
- Modify: `mcp-server/src/index.ts` — `registerChainTools` import + 호출

### Step 3.1: 테스트 먼저 (RED)

`mcp-server/test/chain.test.ts` 의 첫 4 테스트 (spec §6.3).

McpServer 의 `tool()` 은 등록만 하므로, 테스트에서는 핸들러를 직접 호출. 패턴:
`chain.ts` 가 핸들러 함수를 named export 로도 노출 (`_accountStateHandler`).

```typescript
import { describe, it, expect } from "vitest";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { _accountStateHandler } from "../src/tools/chain.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const MOCK_BIN = resolve(__dirname, "fixtures/mock-chainbench-net.mjs");

// 테스트는 binaryPath option 으로 직접 mock 을 주입할 수 없음 —
// chain.ts 핸들러는 callWire 를 직접 호출하므로 process.env.CHAINBENCH_NET_BIN
// 을 setting 하여 resolveBinary 를 우회.
```

테스트 4개 (account_state):
- `_Happy_DefaultFields` — fields 미지정. mock 응답 ok:true → text 에 JSON 포함
- `_Happy_StorageRequiresKey_RejectedAtBoundary` — `fields: ['storage']` 만 있고
  `storage_key` 없음 → handler 가 isError + INVALID_ARGS (cross-field 검증, zod
  schema 가 아닌 handler 내부)
- `_BadAddress_RejectedAtBoundary` — schema.parse 가 throw — `expect(() => AccountStateArgs.parse({...})).toThrow()`
- `_WireFailure_PassedThrough` — mock 이 INVALID_ARGS 응답 → tool isError:true

### Step 3.2: 구현

`chain.ts`:

```typescript
import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { callWire } from "../utils/wire.js";
import { formatWireResult, type FormattedToolResponse } from "../utils/wireResult.js";

const HEX_ADDRESS = /^0x[a-fA-F0-9]{40}$/;
const HEX_STORAGE_KEY = /^0x[a-fA-F0-9]{1,64}$/;

const FIELD = z.enum(["balance", "nonce", "code", "storage"]);

export const AccountStateArgs = z.object({
  network: z.string().min(1),
  node_id: z.string().optional(),
  address: z.string().regex(HEX_ADDRESS),
  fields: z.array(FIELD).optional(),
  storage_key: z.string().regex(HEX_STORAGE_KEY).optional(),
  block_number: z.union([z.string(), z.number()]).optional(),
});

type AccountStateArgsT = z.infer<typeof AccountStateArgs>;

export async function _accountStateHandler(args: AccountStateArgsT): Promise<FormattedToolResponse> {
  if (args.fields?.includes("storage") && !args.storage_key) {
    return {
      content: [{ type: "text", text: "Error (INVALID_ARGS): fields includes 'storage' but storage_key is missing" }],
      isError: true,
    };
  }
  const wireArgs: Record<string, unknown> = {
    network: args.network,
    address: args.address,
  };
  if (args.node_id !== undefined) wireArgs.node_id = args.node_id;
  if (args.fields !== undefined) wireArgs.fields = args.fields;
  if (args.storage_key !== undefined) wireArgs.storage_key = args.storage_key;
  if (args.block_number !== undefined) wireArgs.block_number = args.block_number;
  const result = await callWire("node.account_state", wireArgs);
  return formatWireResult(result);
}

export function registerChainTools(server: McpServer): void {
  server.tool(
    "chainbench_account_state",
    "Read account balance/nonce/code/storage from a network. " +
    "Network can be local or remote (attached). Returns hex-encoded values.",
    AccountStateArgs.shape,
    _accountStateHandler,
  );
  // tx_send 는 Task 4
}
```

`index.ts` 에 `registerChainTools(server)` 한 줄 추가.

### Step 3.3: 검증 + 커밋

```bash
cd mcp-server && npm test
cd mcp-server && npx tsc --noEmit
```

```bash
git add mcp-server/src/tools/chain.ts mcp-server/test/chain.test.ts mcp-server/src/index.ts
git commit -m "feat(mcp): chainbench_account_state high-level tool

Wraps the chainbench-net node.account_state command with a Zod schema
that rejects bad addresses and storage requests without keys at the
MCP boundary, before any wire spawn. Successful responses surface the
balance/nonce/code/storage hex values via the wireResult formatter;
upstream INVALID_ARGS / UPSTREAM_ERROR pass through with isError.

Tool name follows the existing chainbench_<verb> convention. Network
parameter accepts both local profiles and remote-attached names — the
binary already routes those via state/networks/<name>.json so the MCP
layer stays oblivious to provider type."
```

---

## Task 4 — `chainbench_tx_send` tool (mode: legacy|1559)

**Files:**
- Modify: `mcp-server/src/tools/chain.ts` — add tx_send registration
- Modify: `mcp-server/test/chain.test.ts` — add 6 tx_send tests

### Step 4.1: 테스트 먼저 (RED)

추가 6 tests (spec §6.3):
- `_Happy_Legacy` — `_buildTxSendWireArgs({mode:"legacy", gas_price:"0x..."})` →
  wireArgs.gas_price 포함, max_fee_* 없음
- `_Happy_1559` — `_buildTxSendWireArgs({mode:"1559", max_fee_per_gas, max_priority_fee_per_gas})`
  → wireArgs 에 두 max_fee 필드, gas_price 없음
- `_LegacyWithMaxFee_Rejected` — `{mode:"legacy", max_fee_per_gas:..}` →
  `{error: "..."}`
- `_1559WithoutMaxFee_Rejected` — `{mode:"1559", max_fee_per_gas:.., (no priority)}`
  → `{error: "..."}`
- `_BadSignerAlias_RejectedAtBoundary` — `TxSendArgs.parse({signer:"1bad",...})`
  → throw
- `_WireFailure_PassedThrough` — handler black-box, mock 이 ok:false → isError

### Step 4.2: 구현

```typescript
const HEX_DATA = /^0x[a-fA-F0-9]*$/;
const SIGNER_ALIAS = /^[A-Za-z][A-Za-z0-9_]*$/;
const MODE = z.enum(["legacy", "1559"]);

export const TxSendArgs = z.object({
  network: z.string().min(1),
  node_id: z.string().optional(),
  signer: z.string().regex(SIGNER_ALIAS),
  mode: MODE,
  to: z.string().regex(HEX_ADDRESS).optional(),
  value: z.string().optional(),
  data: z.string().regex(HEX_DATA).optional(),
  gas: z.union([z.string(), z.number()]).optional(),
  nonce: z.union([z.string(), z.number()]).optional(),
  gas_price: z.string().optional(),
  max_fee_per_gas: z.string().optional(),
  max_priority_fee_per_gas: z.string().optional(),
});

type TxSendArgsT = z.infer<typeof TxSendArgs>;

export function _buildTxSendWireArgs(args: TxSendArgsT):
  | { wireArgs: Record<string, unknown> }
  | { error: string } {
  if (args.mode === "legacy") {
    if (!args.gas_price) return { error: "mode 'legacy' requires gas_price" };
    if (args.max_fee_per_gas || args.max_priority_fee_per_gas) {
      return { error: "mode 'legacy' rejects max_fee_per_gas / max_priority_fee_per_gas" };
    }
  } else {
    // 1559
    if (!args.max_fee_per_gas || !args.max_priority_fee_per_gas) {
      return { error: "mode '1559' requires both max_fee_per_gas and max_priority_fee_per_gas" };
    }
    if (args.gas_price) {
      return { error: "mode '1559' rejects gas_price" };
    }
  }

  const wireArgs: Record<string, unknown> = {
    network: args.network,
    signer: args.signer,
  };
  for (const [k, v] of Object.entries(args)) {
    if (k === "mode" || k === "network" || k === "signer") continue;
    if (v === undefined) continue;
    wireArgs[k] = v;
  }
  return { wireArgs };
}

export async function _txSendHandler(args: TxSendArgsT): Promise<FormattedToolResponse> {
  const built = _buildTxSendWireArgs(args);
  if ("error" in built) {
    return {
      content: [{ type: "text", text: `Error (INVALID_ARGS): ${built.error}` }],
      isError: true,
    };
  }
  const result = await callWire("node.tx_send", built.wireArgs);
  return formatWireResult(result);
}

// register inside registerChainTools
server.tool(
  "chainbench_tx_send",
  "Send a signed transaction. Mode 'legacy' uses pre-EIP-1559 gas pricing " +
  "(gas_price required). Mode '1559' uses EIP-1559 dynamic fee fields " +
  "(max_fee_per_gas + max_priority_fee_per_gas required). The signer alias " +
  "must have CHAINBENCH_SIGNER_<ALIAS>_KEY (or _KEYSTORE + _KEYSTORE_PASSWORD) " +
  "set in the host environment. Future MCP releases will add modes for " +
  "set_code (EIP-7702) and fee_delegation (go-stablenet 0x16).",
  TxSendArgs.shape,
  _txSendHandler,
);
```

### Step 4.3: 검증 + 커밋

```bash
cd mcp-server && npm test
cd mcp-server && npx tsc --noEmit
```

```bash
git add mcp-server/src/tools/chain.ts mcp-server/test/chain.test.ts
git commit -m "feat(mcp): chainbench_tx_send with explicit legacy/1559 mode

Adds the second high-level tool of Sprint 5c.1. The 'mode' enum
('legacy' | '1559') makes the LLM's intent explicit at the boundary
so mismatched fee fields are rejected before chainbench-net is
spawned: legacy must carry gas_price and reject max_fee_*,
1559 must carry both max_fee_per_gas and max_priority_fee_per_gas
and reject gas_price.

The signer parameter is an alias only; the tool never receives raw
key material. The actual CHAINBENCH_SIGNER_<ALIAS>_KEY (or
_KEYSTORE + _KEYSTORE_PASSWORD) lives in the host environment that
spawned the MCP server, and chainbench-net inherits that env when
the wire helper spawns it. The Sprint 4 redaction boundary in the
Go signer package therefore continues to enforce the no-leak
contract end-to-end.

set_code (0x4) and fee_delegation (0x16) modes are deliberately not
exposed yet — the tool description says so and points callers at
the wire protocol for those cases until Sprint 5c.2."
```

---

## Task 5 — Docs + version bump + final review prep

**Files:**
- Modify: `mcp-server/package.json` — version 0.3.0 → 0.4.0
- Modify: `mcp-server/src/index.ts` — server version 동기화
- Modify: `docs/EVALUATION_CAPABILITY.md`
- Modify: `docs/VISION_AND_ROADMAP.md`
- Modify: `docs/NEXT_WORK.md`

### Step 5.1: Version bump

`package.json` `"version": "0.4.0"` 와 `src/index.ts` `new McpServer({version: "0.4.0"})`.

### Step 5.2: EVALUATION_CAPABILITY.md

§2 (Tx 매트릭스):
- `Legacy tx (0x0) value transfer` MCP 열: ❌ → ✅ Sprint 5c.1
- `EIP-1559 (0x2) dynamic fee` MCP 열: ❌ → ✅ Sprint 5c.1
- `Account state assert` MCP 열: partial → ✅ Sprint 5c.1

§4 (검증 매트릭스):
- `Account state (balance/nonce/code/storage)` MCP 열: partial → ✅ Sprint 5c.1

§5 (Sprint 별 도달 목표):
- `5c.1` row 추가: "MCP wire helper + transformer + chainbench_account_state +
  chainbench_tx_send (legacy/1559)"

§6:
- "MCP high-level tool surface: 약 10% → 약 25%"

### Step 5.3: VISION_AND_ROADMAP.md

§6 Sprint 5 항목을 5c.1 / 5c.2 / 5c.3 로 분할:
- `Sprint 5c.1` — 완료 박스 (이번 sprint)
- `Sprint 5c.2` — 미완 (set_code + fee_delegation modes, contract_deploy/call,
  events_get, tx_wait)
- `Sprint 5c.3` — 미완 (기존 tool reroute)
- `Sprint 5a/5b/5d` — 기존 그대로

§5.17.7 (MCP 서버 연동) 에 짧은 갱신.

### Step 5.4: NEXT_WORK.md

§1 헤더의 "최종 업데이트" 라인 갱신.
§2.1 timeline 표에 5c.1 row 추가.
§3 Priority 1 갱신 — 다음 P1 = Sprint 5c.2.
§3 P3 누적 tech debt 표 — vitest 통합 테스트 row 추가.
§4.6 file size 표 — 신규 파일 사이즈 명시 (모두 권장치 이내 확인).

### Step 5.5: 검증 + 커밋

```bash
cd mcp-server && npm test
cd mcp-server && npx tsc --noEmit
cd mcp-server && npm run build
go -C network test ./... -count=1 -timeout=60s   # 회귀 없음 확인
bash tests/unit/run.sh                            # 회귀 없음 확인
```

```bash
git add mcp-server/package.json mcp-server/src/index.ts \
        docs/EVALUATION_CAPABILITY.md docs/VISION_AND_ROADMAP.md docs/NEXT_WORK.md
git commit -m "docs+chore(sprint-5c-1): roadmap + capability matrix + version 0.4.0

EVALUATION_CAPABILITY MCP column:
  - §2 Legacy tx, §2 EIP-1559, §2/§4 Account state assert: ✅ Sprint 5c.1
  - §6 MCP coverage 10% -> 25%

VISION_AND_ROADMAP §6 Sprint 5 split into 5c.1 (this sprint) /
5c.2 (remaining 4 high-level tools + set_code + fee_delegation
modes) / 5c.3 (existing 38 tool reroute). 5a / 5b / 5d unchanged.

NEXT_WORK timeline gains the 5c.1 row; P1 shifts to Sprint 5c.2.

mcp-server version 0.3.0 -> 0.4.0 (first high-level evaluation tool
release; new tools chainbench_account_state and chainbench_tx_send)."
```

---

## Final report (after all tasks)

Commit chain (5 expected: wire / transformer / account_state / tx_send / docs).
mcp-server file sizes (utils/wire.ts, utils/wireResult.ts, tools/chain.ts —
권장치 검증). Test counts (vitest: 9 wire + 5 wireResult + 10 chain = 24).
Coverage delta (MCP 10% → 25%, EVALUATION cells flipped). Confirmed deferrals:
5c.2 (4 tools + set_code/fee_delegation modes), 5c.3 (reroute), 5a/5b/5d
(capability/SSH/hybrid).

가능한 follow-up (P3 tech debt 후보):
- mcp-server vitest 통합 테스트 (실 chainbench-net spawn)
- mcp-server/src/tools/ 의 기존 큰 파일 (test.ts 335, schema.ts 308) 검토는
  5c.3 reroute 시 자연스럽게 흡수
- wire helper 의 subscription / 장기 실행 호출 지원 — chain log streaming sprint
  에서
