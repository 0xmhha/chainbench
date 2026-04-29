# Sprint 5c.3 — MCP Reroute (Pass 1) Design Spec

> 2026-04-29 · VISION §6 Sprint 5 — 5c third pass (reroute initiation)
> Scope: Sprint 5c.1/5c.2 가 깐 wire helper + 6 high-level tool 위에서 기존 38
> tool 의 wire 경유 reroute 를 시작. 첫 패스는 utils 추출 + `node.rpc` wire
> 핸들러 추가 + 3 node tool reroute (`chainbench_node_rpc` / `_node_stop` /
> `_node_start`) + 실 바이너리 integration 테스트 layer 구축.

## 1. Goal

`chainbench_node_*` 3 tool 을 첫 reroute 대상으로 잡는 이유:
- `node.start` / `node.stop` 의 wire 핸들러는 이미 존재 (Sprint 2b)
- `node.rpc` 는 schema enum 에 등록만 됐고 Go 핸들러는 없음 — generic
  JSON-RPC passthrough 라 작은 추가
- 3 tool 모두 `lib/cmd_node.sh` 경유 → 직접 wire 로 전환하면 spawn 1 hop
  (bash → wire) 만큼 latency 감소 + 일관된 NDJSON 응답
- 향후 reroute (5c.4 부터) 의 검증된 패턴 확립 + integration 테스트 hook

또한 5c.2 의 P3 tech debt 두 건을 흡수:
- `errorResp` helper duplication (chain_read.ts + chain_tx.ts) → `utils/mcpResp.ts`
- hex regex / SIGNER_ALIAS 등 상수 중복 → `utils/hex.ts`

본 sprint 종료 시 reroute coverage = 3/38 ≈ 8%. 다음 패스 (5c.4) 가
lifecycle 또는 remote tool 흡수.

## 2. Non-Goals (Deferred)

- **나머지 35 tool reroute** — lifecycle / test / log / remote / consensus /
  network / config / spec / schema 는 5c.4+ 에서. 각 패스는 도메인별 묶음.
- **Bash CLI deprecation** — `chainbench.sh` 진입점 / `lib/cmd_*.sh` 모두
  유지. MCP layer 만 wire 로 전환 — bash 사용자는 영향 없음.
- **새 high-level tool 추가** — Sprint 5c.1/5c.2 가 끝낸 surface 그대로.
- **`chain_tx.ts` 파일 분할** — 5c.2 P3 P3 watch. 단일 패스 reroute 가 다른
  파일에 추가하는 동안 이 파일은 손대지 않음.
- **WS subscription** — 후속.
- **Capability gate (5a) / SSH driver (5b) / Hybrid 예제 (5d)** — 별 sprint.

## 3. Security Contract

기존 redaction boundary 그대로. 본 sprint 가 추가하는 surface:

- **`node.rpc` Go 핸들러** — 임의 method + params 의 JSON-RPC passthrough.
  보안 고려:
  - `eth_sendRawTransaction` 등 raw signed tx 받기 가능 (이미 외부에서 서명한
    tx) — 키 자체는 wire 로 들어오지 않음. 4d 의 contract_call/events 와 동일
    수준 surface
  - `eth_sign` / `personal_sign` 등 노드의 unlock 된 계정 사용은 가능하나
    chainbench-net 의 signer alias 와 무관 — chain 노드의 책임
  - Raw method/params 를 그대로 통과; 입력 검증은 method 이름 정규식만 (TS
    layer 가 이미 zod 로 거절)
- **`chainbench_node_rpc` reroute** — 인터페이스 변경 없음. 기존 zod schema
  유지 (`method` 정규식 + `params` JSON array). signer alias 안 받음 (raw
  passthrough 이므로 chainbench-net 의 signer 체계 미사용).

## 4. User-Facing Surface

### 4.1 `network/cmd/chainbench-net/handlers_node_read.go` 확장 (Task 1)

새 핸들러 `newHandleNodeRpc(stateDir)`:

```go
func newHandleNodeRpc(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network string          `json:"network"`
            NodeID  string          `json:"node_id"`
            Method  string          `json:"method"`
            Params  json.RawMessage `json:"params"`
        }
        // parse, validate method (alphanumeric_underscore), params is JSON array or null
        // resolveNode + dialNode (RPC client)
        // client.Client().CallContext(ctx, &raw, req.Method, paramsList...)
        // return {"result": raw}
    }
}
```

**Args**:
- `network` (string, required) — 기존 resolveNode 패턴
- `node_id` (string, optional) — 기본 첫 노드
- `method` (string, required) — JSON-RPC method (`^[a-zA-Z][a-zA-Z0-9_]*$`)
- `params` (JSON, optional) — JSON array or null. 미제공 시 빈 배열

**Result**: `{"result": <raw RPC response>}` — 그대로 통과.

**Error matrix**:
- INVALID_ARGS — bad method name, params not array/null, network/node 없음
- UPSTREAM_ERROR — RPC call 실패 (timeout, JSON-RPC error)
- INTERNAL — invariant 깨짐

### 4.2 `mcp-server/src/utils/mcpResp.ts` (Task 0, NEW)

```typescript
export interface FormattedToolResponse { ... }  // wireResult.ts 에서 이전

export function errorResp(msg: string): FormattedToolResponse {
  return {
    content: [{ type: "text", text: `Error (INVALID_ARGS): ${msg}` }],
    isError: true,
  };
}
```

`FormattedToolResponse` 인터페이스도 본 파일로 이전 (현재 wireResult.ts 가
declare). wireResult.ts 는 이를 `import type` 로 사용 + re-export 유지
(backward compat).

### 4.3 `mcp-server/src/utils/hex.ts` (Task 0, NEW)

```typescript
export const HEX_ADDRESS = /^0x[a-fA-F0-9]{40}$/;
export const HEX_DATA = /^0x([a-fA-F0-9]{2})*$/;
export const HEX_TX_HASH = /^0x[a-fA-F0-9]{64}$/;
export const HEX_TOPIC = /^0x[a-fA-F0-9]{64}$/;
export const HEX_HEX = /^0x[a-fA-F0-9]+$/;
export const HEX_STORAGE_KEY = /^0x[a-fA-F0-9]{1,64}$/;
export const SIGNER_ALIAS = /^[A-Za-z][A-Za-z0-9_]*$/;
export const RPC_METHOD = /^[a-zA-Z][a-zA-Z0-9_]*$/;
```

`chain_read.ts` / `chain_tx.ts` / `node.ts` 모두 이 파일에서 import.

### 4.4 `mcp-server/src/tools/node.ts` reroute (Task 3)

기존 `runChainbench(...)` → `callWire(...)`. 핵심 변환:

**`chainbench_node_stop` (1-based int → wire)**:
```typescript
async ({ node }) => {
  const result = await callWire("node.stop", {
    network: "local",
    node_id: `node${node}`,
  });
  return formatWireResult(result);
}
```

**`chainbench_node_start`**: 동일 패턴. `binary_path` arg 는 wire 가 미지원
이므로 envOverrides 로 처리:
```typescript
async ({ node, binary_path }) => {
  const envOverrides = binary_path ? { CHAINBENCH_NODE_BINARY: binary_path } : undefined;
  const result = await callWire("node.start", {...}, { envOverrides });
  ...
}
```

(우선 wire 의 `node.start` 가 binary_path 인자를 받지 않으면 envOverrides 도
미사용 — 기존 bash 가 어떻게 처리하는지 확인 필요. chainbench-net 의
LocalDriver 는 `lib/cmd_node.sh` 를 spawn 하므로 env 전파 가능성 검토.)

**중요**: 기존 bash `node start` 는 `--binary-path` 옵션 처리. wire reroute
시 이 시그니처를 어떻게 보존할지. 본 sprint 의 가장 까다로운 부분. 두 옵션:
- (a) `binary_path` 미지정 호출은 wire 로, 지정 시 fallback to `runChainbench`
- (b) `node.start` wire 핸들러를 확장하여 `binary_path` arg 추가 (Go work)
- 본 sprint: (a) 채택. binary_path 처리 wire 확장은 5c.4 에서.

**`chainbench_node_rpc` (Task 2)**:
```typescript
async ({ node, method, params }) => {
  const wireArgs = {
    network: "local",
    node_id: `node${node}`,
    method,
    params: params ? JSON.parse(params) : [],
  };
  const result = await callWire("node.rpc", wireArgs);
  return formatWireResult(result);
}
```

기존 zod schema 유지 (method regex + params optional JSON string).

### 4.5 Real-binary integration test (Task 4, NEW)

`mcp-server/test/integration/node_rpc.integration.test.ts` (NEW):
- 실제 `chainbench-net` 바이너리 (build first, `network/bin/chainbench-net`)
  spawn
- Python JSON-RPC mock (Sprint 4 패턴 — `tests/unit/tests/security-key-boundary.sh`
  와 동형) 또는 nock-based fake server
- chainbench_node_rpc 를 호출하여 단일 happy-path 검증

vitest 설정:
- 별도 test suite (`integration` 디렉토리) — `vitest.config.ts` 에 두 번째
  test config 또는 단일 config 의 include 확장
- 실 빌드 의존: `npm run build:network` 이 `network/bin/chainbench-net` 를
  생성해야 함. CI 에서는 prebuilt 가능
- 본 sprint 는 단일 시나리오 (`eth_blockNumber` happy-path) 만 — 후속 reroute
  task 가 다른 시나리오 추가

**Optional**: 통합 테스트가 까다로우면 본 sprint 에서 mock 버전만 강화하고
real-binary 는 5c.4 로 미루는 것도 가능. 결정: 우선 시도 → 막히면 mock 으로
후퇴.

## 5. Package Layout

**Create:**
- `mcp-server/src/utils/mcpResp.ts`
- `mcp-server/src/utils/hex.ts`
- `mcp-server/test/integration/node_rpc.integration.test.ts` (또는 mock 강화 fallback)

**Modify:**
- `mcp-server/src/utils/wireResult.ts` — `FormattedToolResponse` 를 `mcpResp.ts`
  에서 import + re-export
- `mcp-server/src/tools/chain_read.ts` — `errorResp` 제거 (mcpResp.ts import),
  hex regex 제거 (hex.ts import)
- `mcp-server/src/tools/chain_tx.ts` — 동일
- `mcp-server/src/tools/node.ts` — `runChainbench` → `callWire`. 3 tool 전부.
- `mcp-server/test/chain_read.test.ts` / `chain_tx.test.ts` — import 경로 수정
- `mcp-server/test/node.test.ts` (NEW or extend) — node.ts 의 wire 호출
  단위 테스트 (mock binary 패턴)
- `network/cmd/chainbench-net/handlers_node_read.go` — `newHandleNodeRpc` 추가
- `network/cmd/chainbench-net/handlers.go` — register `node.rpc`
- `network/cmd/chainbench-net/handlers_test.go` — node.rpc 단위 테스트
- `mcp-server/package.json` — 0.5.0 → 0.6.0
- `mcp-server/src/index.ts` — McpServer version 0.6.0
- `docs/EVALUATION_CAPABILITY.md` — 갱신 (필요 시)
- `docs/VISION_AND_ROADMAP.md` — Sprint 5c.3 박스
- `docs/NEXT_WORK.md` — §1/§2/§3/§4.6

## 6. Tests

### 6.1 Go 단위 테스트 (Task 1)

`handlers_test.go` 의 신규:
- `TestHandleNodeRpc_Happy` — mock RPC 서버, eth_blockNumber 통과
- `TestHandleNodeRpc_BadMethod` → INVALID_ARGS
- `TestHandleNodeRpc_BadParams` (params 가 array 아님) → INVALID_ARGS
- `TestHandleNodeRpc_RpcFailure` → UPSTREAM_ERROR
- `TestAllHandlers_IncludesNodeRpc`

### 6.2 TS 단위 테스트 (Task 0, 3)

`mcpResp.test.ts` (NEW) — `errorResp` 결과 형태 검증.

`node.test.ts` (NEW or extend) — 3 tool 의 wire 변환:
- `chainbench_node_stop` — 1-based int → `node_id: "node1"` 변환
- `chainbench_node_start` — 동일
- `chainbench_node_rpc` — params JSON 파싱 + wire args
- 각 tool 의 wire failure 통과 (mock 이 ok:false 응답)
- binary_path arg 미지원 → 적절한 처리 (현재 bash 로 fallback 또는 graceful
  rejection)

### 6.3 Real-binary integration (Task 4)

선택적 — fallback: mock 강화로 대체.

## 7. Schema

`network/schema/command.json` 변경 없음 — `node.rpc` 가 이미 enum 등록됨.

## 8. Error Classification

기존 매트릭스 그대로.

## 9. Out-of-Scope Reminders

**5c.4 으로 이행**:
- lifecycle tools (chainbench_init/start/stop/restart/status/clean) — 가장 자주
  쓰이는 tool. 단, `chainbench_init` 등은 wire 핸들러가 없으므로 Go 측 추가 필요
- node.start binary_path arg → wire 확장
- 다른 tool 도메인

**Sprint 5 series 외**:
- WebSocket / chain log streaming
- Capability gate (5a) / SSH driver (5b) / Hybrid 예제 (5d)

## 10. Migration / Backwards Compat

- `chainbench_node_*` MCP tool 이름 / 인자 변경 없음 — LLM caller 영향 없음
- bash CLI `chainbench node ...` 영향 없음
- 패키지 버전 0.5.0 → 0.6.0 (3 tool 의 internal reroute + utils refactor)
