# Sprint 5c.1 — MCP Foundation + 2 High-Level Tools Design Spec

> 2026-04-28 · VISION §6 Sprint 5 — 5c first pass (foundation + two-tool vertical slice)
> Scope: Sprint 4d 가 채운 Go `network/` 능력을 MCP 로 노출하는 첫 단계.
> wire-spawn 헬퍼 + NDJSON → MCP response transformer + 대표 read tool (`chainbench_account_state`)
> + 대표 write tool (`chainbench_tx_send` legacy/1559 통합 모드).
> 본 sprint 종료 시 `EVALUATION_CAPABILITY.md` §2/§4 의 MCP column 2 cell 이 ✅ 로 전환되고
> coding agent 가 chainbench-net wire 를 직접 알지 않고도 합리적 evaluation 을 수행 가능.

## 1. Goal

Sprint 5 시리즈의 1차 패스 (5c.1). 목표는 다음 3가지 인프라 + 2 tool:

1. **TS wire helper** (`mcp-server/src/utils/wire.ts`) — `chainbench-net run` 을
   spawn 하여 NDJSON envelope 입력 + NDJSON stream 출력을 처리. 바이너리 경로
   resolution 은 bash `lib/network_client.sh` 와 동일 (env `CHAINBENCH_NET_BIN`
   → `$CHAINBENCH_DIR/bin/` → `$CHAINBENCH_DIR/network/bin/` → `PATH`).
2. **Result transformer** (`mcp-server/src/utils/wireResult.ts`) — `event` /
   `progress` / `result` 라인을 LLM 친화 응답으로 변환. 실패 시 phase 단위
   힌트 제공. 성공 시 `data` 필드 + 선택적 `phases` 요약.
3. **2 high-level tools**:
   - `chainbench_account_state` — `node.account_state` wire 호출 + 결과 그대로 노출
   - `chainbench_tx_send` — `mode: "legacy" | "1559"` discriminator. `node.tx_send`
     호출 + signer alias 환경변수 forwarding. 0x4 (set_code) / 0x16 (fee_delegation)
     모드는 본 sprint 미노출 (5c.2 에서 확장).

향후 5c.2/5c.3 가 나머지 4 high-level tool (`chainbench_contract_deploy`,
`chainbench_contract_call`, `chainbench_events_get`, `chainbench_tx_wait`) +
기존 38 tool reroute 를 흡수.

## 2. Non-Goals (Deferred)

- **기존 tool reroute** — `chainbench_init`, `chainbench_start`, `chainbench_node_*`
  등은 본 sprint 에서 손대지 않음. 기존 `runChainbench()` 그대로. 5c.3 에서 점진 이관.
- **Set-Code / fee-delegation mode** — `chainbench_tx_send` 의 `mode` enum 에서
  legacy/1559 만 정의. set_code / fee_delegation 은 5c.2 sprint 에서 추가.
- **Contract deploy/call/events/wait tools** — 5c.2 sprint.
- **WS subscription tool / chain log streaming** — VISION §5.10. 5+ 후속.
- **NDJSON event 의 LLM 실시간 스트리밍** — MCP tool 응답은 단일 reply (text content
  list). 진행 이벤트는 `phases` 요약으로 집계되어 응답에 포함. 진짜 stream UX 는 후속.
- **Capability gate** — Sprint 5a.
- **SSH driver / hybrid 예제** — Sprint 5b/5d.

## 3. Security Contract

Sprint 4 / 4b / 4c 의 redaction boundary 그대로 상속:

- MCP 서버는 **chainbench-net 의 stdout NDJSON 만 사용자에게 노출**. stderr (slog)
  는 MCP 서버 자신의 stderr 로 흘려보내거나 (개발 환경) drop. 현 키 redaction
  계약 (`signer.Sealed.LogValue() = "***"`) 은 chainbench-net 내부에서 이미 보장됨.
- `chainbench_tx_send` 가 받는 `signer: <alias>` 는 alias **이름만**. 실제
  `CHAINBENCH_SIGNER_<ALIAS>_KEY` / `_KEYSTORE` / `_KEYSTORE_PASSWORD` 환경변수는
  MCP 서버 spawn 시 호스트 환경에서 주어진다고 가정. wire helper 는 `process.env`
  를 그대로 forwarding.
- TS 측에서 절대 key material 을 받거나 저장하지 않음. tool argument schema 에
  raw key 필드 추가 금지 (Zod 가 거절). 신규 tool args 는 alias 만 받는다.
- 에러 응답 포맷터는 chainbench-net 의 에러 message 를 그대로 통과 — TS 측에서
  추가 redaction 은 불필요 (Go signer boundary 가 이미 검증됨).

## 4. User-Facing Surface

### 4.1 `mcp-server/src/utils/wire.ts`

```typescript
import { spawn } from "node:child_process";
import { resolve } from "node:path";
import readline from "node:readline";

export type WireResultLine =
  | { type: "result"; ok: true; data: Record<string, unknown> }
  | {
      type: "result";
      ok: false;
      error: { code: string; message: string; details?: unknown };
    };

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

export type WireStreamLine = WireResultLine | WireEventLine | WireProgressLine;

export interface WireCallResult {
  result: WireResultLine;          // 마지막 result 라인 (성공/실패 무관)
  events: WireEventLine[];          // 누적된 event 라인
  progress: WireProgressLine[];     // 누적된 progress 라인
  stderr: string;                   // chainbench-net stderr (slog) — 디버그용
  exitCode: number;
}

export interface WireCallOptions {
  /** chainbench-net 환경변수 추가/오버라이드 */
  envOverrides?: Record<string, string>;
  /** 호출 timeout (ms). 기본 120000 */
  timeoutMs?: number;
  /** chainbench-net 바이너리 절대경로. 미지정 시 resolveBinary() */
  binaryPath?: string;
}

export function resolveBinary(): string;

export async function callWire(
  command: string,
  args: Record<string, unknown>,
  options?: WireCallOptions
): Promise<WireCallResult>;
```

**Binary resolution (bash 와 동일 순서)**:
1. `process.env.CHAINBENCH_NET_BIN` (실행 가능)
2. `${CHAINBENCH_DIR}/bin/chainbench-net`
3. `${CHAINBENCH_DIR}/network/bin/chainbench-net`
4. `which chainbench-net` (PATH)

미해결 시 `Error('chainbench-net binary not found')` throw.

**Spawn 동작**:
- `spawn(bin, ['run'], { env: { ...process.env, ...envOverrides }, stdio: ['pipe','pipe','pipe'] })`
- stdin 에 `JSON.stringify({command, args}) + '\n'` write 후 close
- stdout 을 readline 으로 라인 단위 파싱:
  - `JSON.parse(line)` 시도. 실패 라인은 silent skip (chainbench-net 은 stdout 에
    NDJSON 만 보내지만 방어적으로)
  - `type` 별로 events / progress / result 분류
  - `result` 라인 만나면 추가 라인 무시 (terminator)
- stderr 누적 (string, debug 용)
- timeout 도달 시 `child.kill('SIGTERM')` + `Error('chainbench-net timeout after Nms')`
- exit code 가 0 이 아니어도 result 라인이 있으면 정상 반환 (caller 가 ok 확인)

**Out-of-scope**:
- Subscription / 장기 실행 호출 (chainbench-net spawn 이 SIGTERM 까지 살아있는 경우).
  본 sprint 의 두 tool 은 둘 다 단일-result 호출이므로 단순 spawn-and-collect 로 충분.

### 4.2 `mcp-server/src/utils/wireResult.ts`

```typescript
import type { WireCallResult } from "./wire.js";

export interface FormattedToolResponse {
  content: Array<{ type: "text"; text: string }>;
  isError?: boolean;
}

/**
 * Chainbench-net wire result 를 MCP tool response 로 변환.
 *
 * 성공:
 *   - data 가 비어있지 않으면 JSON.stringify(data, null, 2) 본문
 *   - phases 가 1+ 이면 "Phases: [n] step1, step2, ..." 한 줄 추가
 *   - events 가 1+ 이면 "Events: [n] name1, name2, ..." 한 줄 추가
 *
 * 실패:
 *   - "Error (CODE): message" 본문
 *   - 마지막 1~3개 progress step 이 있으면 "Last phase: stepN" 추가
 *   - isError: true
 */
export function formatWireResult(result: WireCallResult): FormattedToolResponse;
```

**설계 의도**: LLM 이 한 응답으로 (1) 결과 데이터 (2) 거친 phase 요약 둘 다
받도록. 진짜 시간순 stream UX 는 미루되, 실패 시 어느 phase 에서 깨졌는지
힌트는 즉시 제공. JSON.stringify pretty-print 는 LLM 이 필드 식별하기 쉬움.

### 4.3 `chainbench_account_state` tool

`mcp-server/src/tools/chain.ts` (NEW):

```typescript
server.tool(
  "chainbench_account_state",
  "Read account balance/nonce/code/storage from a network. " +
  "Network can be local or remote (attached). Returns hex-encoded values.",
  {
    network: z.string().min(1).describe("Network name (e.g. 'local', 'sepolia')"),
    node_id: z.string().optional().describe("Node ID, default first node"),
    address: z.string().regex(/^0x[a-fA-F0-9]{40}$/).describe("Account address"),
    fields: z.array(z.enum(["balance", "nonce", "code", "storage"])).optional()
      .describe("Default ['balance','nonce','code']. 'storage' requires storage_key."),
    storage_key: z.string().regex(/^0x[a-fA-F0-9]{1,64}$/).optional()
      .describe("Storage slot, required if fields includes 'storage'"),
    block_number: z.union([z.string(), z.number()]).optional()
      .describe("'latest', 'earliest', '0x10', or integer block number"),
  },
  async (args) => {
    const wireArgs: Record<string, unknown> = { ... };
    const result = await callWire("node.account_state", wireArgs);
    return formatWireResult(result);
  }
);
```

Zod 가 boundary 검증 → wire 호출은 항상 well-formed args 만 통과. chainbench-net
의 추가 검증 (예: storage 요청 시 storage_key 필수) 은 second line of defense
역할로 그대로 작동.

### 4.4 `chainbench_tx_send` tool

```typescript
server.tool(
  "chainbench_tx_send",
  "Send a signed transaction. Mode 'legacy' uses pre-EIP-1559 gas pricing. " +
  "Mode '1559' uses EIP-1559 dynamic-fee fields. Other modes (set_code, " +
  "fee_delegation) will be added in future MCP releases — use the wire " +
  "protocol directly for now.",
  {
    network: z.string().min(1),
    node_id: z.string().optional(),
    signer: z.string().regex(/^[A-Za-z][A-Za-z0-9_]*$/)
      .describe("Signer alias. CHAINBENCH_SIGNER_<ALIAS>_KEY (or _KEYSTORE+_KEYSTORE_PASSWORD) must be set in the host environment."),
    mode: z.enum(["legacy", "1559"]).describe("Transaction fee mode."),
    to: z.string().regex(/^0x[a-fA-F0-9]{40}$/).optional()
      .describe("Recipient. Omit for contract creation (use chainbench_contract_deploy in future MCP releases)."),
    value: z.string().optional().describe("Value in wei (decimal or 0x-hex)"),
    data: z.string().regex(/^0x[a-fA-F0-9]*$/).optional(),
    gas: z.union([z.string(), z.number()]).optional(),
    nonce: z.union([z.string(), z.number()]).optional(),
    // legacy-only:
    gas_price: z.string().optional().describe("Required for mode 'legacy'. Decimal wei or 0x-hex."),
    // 1559-only:
    max_fee_per_gas: z.string().optional().describe("Required for mode '1559'."),
    max_priority_fee_per_gas: z.string().optional().describe("Required for mode '1559'."),
  },
  async (args) => {
    // 1. mode-specific cross-field validation
    //    mode=legacy → gas_price 필수, max_fee_* 거절
    //    mode=1559   → max_fee_per_gas + max_priority_fee_per_gas 필수, gas_price 거절
    // 2. wire args 매핑 (chainbench-net 의 node.tx_send 가 fee mode 자동 감지:
    //    max_fee_per_gas 있으면 1559, 없으면 legacy)
    // 3. callWire("node.tx_send", wireArgs)
    // 4. formatWireResult
  }
);
```

**Mode discriminator 의 의미**: chainbench-net 의 `node.tx_send` 는 fee 필드의
유무로 mode 를 자동 판별 (Sprint 4b 결정). MCP 측 `mode` 는 **LLM 의 의도를
명시화**하여 잘못된 필드 조합을 boundary 에서 거절하기 위함. 예: LLM 이
`mode=legacy` + `max_fee_per_gas` 를 같이 보내면 INVALID_ARGS 즉시 반환.

## 5. Package Layout

**Create:**
- `mcp-server/src/utils/wire.ts`
- `mcp-server/src/utils/wireResult.ts`
- `mcp-server/src/tools/chain.ts` — Sprint 5c 가 추가하는 high-level chain tool
  들의 공용 파일. 5c.1 은 `chainbench_account_state` + `chainbench_tx_send` 만.
- `mcp-server/test/wire.test.ts` — wire helper unit 테스트 (mock binary)
- `mcp-server/test/wireResult.test.ts` — formatter 테스트
- `mcp-server/test/chain.test.ts` — tool wrapper 테스트
- `mcp-server/test/fixtures/mock-chainbench-net.mjs` — 테스트용 mock 바이너리
  (NDJSON 출력 시뮬레이션)

**Modify:**
- `mcp-server/src/index.ts` — `registerChainTools(server)` 추가
- `mcp-server/package.json` — devDeps `vitest`, `@types/node` (이미 있음). test
  script 추가
- `mcp-server/tsconfig.json` — test 디렉토리 포함되도록 검토
- `docs/EVALUATION_CAPABILITY.md` — §2 (Account state assert), §4 (Account state
  /balance/nonce/code/storage) MCP column 을 ✅ 로. §2 (Legacy / 1559 tx) MCP
  column 도 ✅. §6 MCP coverage 10% → 25% 추정 갱신
- `docs/VISION_AND_ROADMAP.md` — Sprint 5c.1 체크박스 신설 + §6 Sprint 5 line 갱신
- `docs/NEXT_WORK.md` — §2.1 timeline 5c.1 row 추가, §3 P1 갱신, §4.6 file size
  표 (mcp-server 신규 파일들 — 권장치 초과 없음 확인)

## 6. Tests

### 6.1 `wire.test.ts` (Task 1)

테스트는 vitest. 모든 테스트가 fixture mock 바이너리 (`fixtures/mock-chainbench-net.mjs`)
를 spawn — 실제 Go 바이너리 빌드 의존성 없음.

- `resolveBinary_PrefersEnv` — `CHAINBENCH_NET_BIN` 우선
- `resolveBinary_FallsBackToChainbenchDir` — env 없을 때 `$CHAINBENCH_DIR/bin/`
- `resolveBinary_NotFound_Throws` — 어디에도 없으면 throw
- `callWire_Happy_ReturnsResultEventsProgress` — mock 이 progress 2개 + event 1개
  + result 1개 출력. wire helper 가 모두 분류하여 반환
- `callWire_OnError_StillReturnsResultLine` — mock 이 ok:false result 출력. helper
  가 throw 하지 않고 result 그대로 반환 (caller 가 ok 검사)
- `callWire_NoTerminator_Throws` — mock 이 result 없이 종료. helper 가 throw
- `callWire_Timeout_KillsAndThrows` — mock 이 영구 sleep. timeout 발동
- `callWire_PassesEnvOverrides` — mock 이 `process.env.FOO` 를 result.data 에
  echo. envOverrides: {FOO:'bar'} 가 전달되는지 확인
- `callWire_NonZeroExit_WithResult_Succeeds` — mock 이 ok:true result 후 exit 1.
  helper 가 result 반환

### 6.2 `wireResult.test.ts` (Task 2)

- `formatWireResult_Success_RendersDataPretty` — result.ok=true + data 비어있지
  않으면 JSON pretty 본문
- `formatWireResult_Success_EmptyData_RendersDoneText` — data 비어있으면 "Done."
- `formatWireResult_Success_WithPhasesSummary` — progress 라인 N개 + event 라인
  M개 → "Phases: ..." / "Events: ..." 라인 포함
- `formatWireResult_Failure_RendersErrorWithCode` — ok=false → "Error (CODE):
  message" 본문 + isError:true
- `formatWireResult_Failure_WithLastPhaseHint` — progress 라인 있으면 마지막
  step 을 "Last phase: <step>" 로 표기

### 6.3 `chain.test.ts` (Task 3+4)

각 tool 에 대해 zod schema 만으로 충분히 거절되는 케이스 + 성공 케이스 (mock
바이너리가 정해진 응답 emit) 검증.

`chainbench_account_state`:
- `_Happy_DefaultFields` — fields 미지정 → wire args 에 fields 미포함, mock 응답
  pretty-print
- `_Happy_StorageRequiresKey_RejectedAtBoundary` — `fields: ['storage']` 만 있고
  `storage_key` 없으면 zod refine 거절 (boundary 에서)
- `_BadAddress_RejectedAtBoundary` — 잘못된 hex address → zod 거절
- `_WireFailure_PassedThrough` — mock 이 INVALID_ARGS 응답 → tool isError:true

`chainbench_tx_send`:
- `_Happy_Legacy` — mode=legacy + gas_price → wire args 변환
- `_Happy_1559` — mode=1559 + max_fee_* → wire args 변환
- `_LegacyWithMaxFee_Rejected` — mode=legacy + max_fee_per_gas 같이 → INVALID_ARGS
- `_1559WithoutMaxFee_Rejected` — mode=1559 + max_fee 누락 → INVALID_ARGS
- `_BadSignerAlias_RejectedAtBoundary` — alias regex 위반 → zod 거절
- `_WireFailure_PassedThrough` — mock 이 ok:false → isError

### 6.4 통합 테스트

별도 통합 테스트 파일 없음 — Sprint 4d 까지의 Go E2E 가 chainbench-net 동작을
검증, 5c.1 의 단위 테스트가 TS layer 를 검증. real-binary 통합은 future sprint
(5c.3 reroute 시) 에 chainbench-mcp-server 를 spawn 해서 elf-test 를 추가할 수
있음.

## 7. Schema

`network/schema/command.json` 변경 없음. Sprint 4d 가 추가한 4 enum 그대로 사용.

## 8. Error Classification

Wire helper 가 raw 그대로 통과. Tool wrapper 가 mode-specific 검증 (legacy 와
1559 fee 필드 충돌) 만 boundary 에서 추가:

| 단계 | 코드 | 사용처 |
|---|---|---|
| Zod schema | (무) | 잘못된 hex / 잘못된 enum / 잘못된 alias 모양 — return 전 거절 |
| Tool wrapper | INVALID_ARGS | mode + fee 필드 조합 위반 |
| chainbench-net | INVALID_ARGS / UPSTREAM_ERROR / NOT_SUPPORTED / INTERNAL | Go 측 boundary |

`isError: true` 는 wire result 가 ok:false 일 때만 + tool wrapper 의 cross-field
검증 실패 시. Zod 실패는 MCP SDK 가 자체 처리.

## 9. Out-of-Scope Reminders

**5c.2 로 이행**:
- `chainbench_contract_deploy`, `chainbench_contract_call`, `chainbench_events_get`,
  `chainbench_tx_wait`
- `chainbench_tx_send` 의 `mode: "set_code"` / `"fee_delegation"` 추가
- `EVALUATION_CAPABILITY` MCP column 추가 cell 들

**5c.3 로 이행**:
- 기존 38 tool 의 wire 경유 reroute (`runChainbench` → `callWire`)
- `lib/cmd_init/start/stop/...` 의 Go 포팅 (network.init wire handler 등)

**Sprint 5 series 외 항목**:
- WebSocket / chain log streaming
- Capability gate (5a)
- SSH driver (5b)
- Hybrid example (5d)
- Adapter.SupportedTxTypes() promotion

## 10. Migration / Backwards Compat

- 기존 `runChainbench()` + bash CLI 경로는 무손상. 신규 `callWire()` 는 별개 path.
- 신규 tool 2개는 알려지지 않은 이름이므로 기존 LLM 통합에 영향 없음.
- mcp-server 의 `name` (chainbench-mcp-server) 와 transport 동일.
- 패키지 버전 0.3.0 → 0.4.0 (high-level evaluation tool 첫 노출이므로 minor bump).
