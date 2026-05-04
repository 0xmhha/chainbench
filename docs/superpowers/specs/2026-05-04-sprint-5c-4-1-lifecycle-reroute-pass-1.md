# Sprint 5c.4.1 — Lifecycle Reroute (Pass 1) Design Spec

> 2026-05-04 · VISION §5.12 M2 + REMAINING_WORK §4 P1 — Sprint 5c.4 first pass
> Scope: Sprint 5c.3 가 깐 reroute 패턴 + integration test layer 위에서, 가장
> 단순한 2 lifecycle tool (`chainbench_stop`, `chainbench_status`) 을 wire 경유
> 로 전환. 동시에 5c.3 P3 prerequisite 인 integration test harness 추출 +
> `chainbench_node_start binary_path` Go-side 확장으로 5c.3 fallback 제거.
> 나머지 4 lifecycle tool (`init`, `start`, `restart`, `clean`) 은 5c.4.2+.

## 1. Goal

Sprint 5c.3 가 검증한 reroute 패턴 (`XxxArgs.strict()` + `_xxxHandler` +
`callWire` + `formatWireResult`) 를 lifecycle 도메인의 가장 단순한 2 tool 에
적용. 동시에 향후 reroute 가 6+ integration test 추가하기 전에 harness 를
재사용 가능한 형태로 추출.

**왜 thin wrapper 인가**: bash lifecycle 코드는 1,367 lines (cmd_status 만
387 lines). 네이티브 Go 포팅은 multi-week 작업. VISION §5.12 M2 가 명시한
"LocalDriver 는 초기엔 `os/exec` 로 기존 `cmd_*.sh` 호출" 패턴 채택. Go
핸들러는 (a) bash spawn (b) stdout 캡처 (c) 종료 코드를 NDJSON envelope 의
INVALID_ARGS / UPSTREAM_ERROR 로 매핑.

**왜 stop + status 만**: stop 은 70 lines (가장 단순), status 는 이미
`--json` mode 가 있어 wrap 시 추가 파싱 불필요. init / start / restart /
clean 은 더 복잡 — 5c.4.2 에서.

본 sprint 종료 시 reroute coverage = 5/38 ≈ 13% (5c.3 의 3 + 본 패스의 2).

## 2. Non-Goals (Deferred)

- **나머지 4 lifecycle tool reroute** — `chainbench_init/start/restart` + 신규
  `chainbench_clean` 노출 (현재 MCP 미노출). 5c.4.2 에서 도메인별 묶음으로.
- **Native Go 포팅** — 본 sprint 의 Go 핸들러는 모두 bash spawn wrap. 진정한
  Go 포팅은 후속 sprint 에서 capability 별로 점진 (예: `network.stop_all`
  은 PID kill 만 하는 단순 로직이라 native 포팅 가능 — 5c.4.3+).
- **Capability-aware dispatch** — Go 핸들러가 capability 미지원 시 NOT_SUPPORTED
  반환하는 layer 는 후속 (현재는 bash 가 자체 처리).
- **`chainbench_status --remote` 옵션** — wire 경유 status 는 active local
  network 만. remote 환경 status 는 5c.4.2+.
- **Bash CLI deprecation** — `chainbench stop` / `status` 등 bash CLI 진입점은
  무손상.

## 3. Security Contract

기존 redaction boundary 그대로. 본 sprint 의 신규 surface:

- **`network.stop_all` Go 핸들러** — `chainbench stop --quiet` spawn. bash 가
  pids.json 을 읽고 pkill 로 노드 프로세스 종료. signer/key material 무관.
  보안 영향 없음.
- **`network.status` Go 핸들러** — `chainbench status --json` spawn. bash 가
  RPC 쿼리 후 JSON 출력. 응답에 RPC URL / port 정보가 포함될 수 있음 — 그러나
  이는 이미 `state/networks/<name>.json` 의 공개 정보. signer 와 무관.
- **bash spawn 자체** — args 는 caller 가 zod 로 사전 검증된 값만 (현재는
  args 자체가 없음 — stop/status 는 input 없는 명령). 향후 `init/start` 가
  reroute 될 때는 `profile` / `binary_path` 등 args 를 shell-escape 필수.

## 4. User-Facing Surface

### 4.1 Go: `network.stop_all` 핸들러 (Task 1)

```go
func newHandleNetworkStopAll(stateDir, chainbenchDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network string `json:"network"`
        }
        if len(args) > 0 {
            if err := json.Unmarshal(args, &req); err != nil {
                return nil, NewInvalidArgs(...)
            }
        }
        if req.Network != "" && req.Network != "local" {
            return nil, NewNotSupported("network.stop_all only operates on the local network")
        }
        // Spawn `chainbench stop --quiet` via os/exec
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        cmd := exec.CommandContext(ctx, filepath.Join(chainbenchDir, "chainbench.sh"), "stop", "--quiet")
        cmd.Env = append(os.Environ(), "CHAINBENCH_DIR="+chainbenchDir)
        out, err := cmd.CombinedOutput()
        if err != nil {
            return nil, NewUpstream("chainbench stop", fmt.Errorf("exit: %v, output: %s", err, string(out)))
        }
        return map[string]any{
            "network": "local",
            "stdout":  string(out),
        }, nil
    }
}
```

**Args**:
- `network` (string, optional) — 기본 `"local"`. non-local → NOT_SUPPORTED
  (remote 네트워크는 stop 의미 없음 — 노드 lifecycle 미보유)

**Result**: `{network: "local", stdout: "..."}` — bash 출력 그대로 통과.

**Error matrix**:
- INVALID_ARGS — args 파싱 실패
- NOT_SUPPORTED — network != local
- UPSTREAM_ERROR — bash spawn 실패 또는 non-zero 종료

### 4.2 Go: `network.status` 핸들러 (Task 3)

```go
func newHandleNetworkStatus(stateDir, chainbenchDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network string `json:"network"`
        }
        // ... parse + validate
        // Spawn `chainbench status --json` via os/exec
        cmd := exec.CommandContext(ctx, filepath.Join(chainbenchDir, "chainbench.sh"), "status", "--json")
        cmd.Env = append(os.Environ(), "CHAINBENCH_DIR="+chainbenchDir)
        out, err := cmd.Output()  // status output parsed as JSON
        if err != nil {
            return nil, NewUpstream("chainbench status", err)
        }
        var statusJson map[string]any
        if err := json.Unmarshal(out, &statusJson); err != nil {
            return nil, NewInternal("status output is not valid JSON: " + err.Error())
        }
        return statusJson, nil
    }
}
```

**Args**: 동일 (network optional, defaults local).

**Result**: bash `chainbench status --json` 의 JSON 그대로 파싱하여 통과
(per-node 정보 + 합의 health 등).

**Error matrix**: INVALID_ARGS / NOT_SUPPORTED / UPSTREAM_ERROR (bash spawn) /
INTERNAL (JSON parse 실패 — invariant).

### 4.3 MCP: `chainbench_stop` reroute (Task 2)

```typescript
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

기존 `chainbench_stop` 은 args 없는 호출. 신규 schema 는 optional `network`
추가 (기본 local). LLM caller 변화 없음.

### 4.4 MCP: `chainbench_status` reroute (Task 4)

```typescript
export const StatusArgs = z.object({
  network: z.string().min(1).optional(),
}).strict();
```

Wire command: `network.status`. Result 그대로 통과 (bash 의 JSON 응답).

기존 `chainbench_status` 도 args 없음. 신규 schema 는 optional `network` 추가.

### 4.5 Go: `node.start` 의 `binary_path` arg 확장 (Task 5)

기존 `node.start` 핸들러 (Sprint 2b) 의 args:
```
{"network": "...", "node_id": "..."}
```

새 args (optional):
```
{"network": "...", "node_id": "...", "binary_path": "/abs/path"}
```

Handler 가 `binary_path` 가 있으면 spawn 시 `--binary-path` 인자 전달.
없으면 기존 동작.

이로써 MCP `chainbench_node_start` 의 `runChainbench` fallback 경로 (5c.3
Task 3) 제거 가능 — 항상 `callWire` 경유.

### 4.6 Integration test harness (Task 0)

`mcp-server/test/integration/_harness.ts` (NEW):

```typescript
export interface RealBinaryHarnessOptions {
  /** Methods returning JSON-RPC responses keyed by method name. */
  rpcHandlers?: Record<string, (req: any) => any>;
  /** Override pids.json content for the seeded local network. */
  pidsOverride?: object;
  /** Override current-profile.yaml content. */
  profileOverride?: object;
}

export interface RealBinaryHarness {
  binaryPath: string;
  stateDir: string;
  mockPort: number;
  teardown: () => Promise<void>;
}

export async function setupRealBinaryHarness(
  opts?: RealBinaryHarnessOptions,
): Promise<RealBinaryHarness>;
```

기존 `node_rpc.integration.test.ts` 의 setup 코드를 추출:
- `pickFreePort` + `waitForPort` (재사용 가능 helper)
- Python JSON-RPC mock spawn (rpcHandlers 로 customizable)
- State dir seeding (pids.json + current-profile.yaml; overrides 가능)
- Env snapshot/restore in teardown
- Cleanup-await + port-race 진단 강화 (5c.3 review I1 + I2 흡수)

`node_rpc.integration.test.ts` 를 `_harness.ts` 사용하도록 refactor — 동일
test 가 통과해야 함 (refactor 검증).

## 5. Package Layout

**Create:**
- `mcp-server/test/integration/_harness.ts` — reusable real-binary test setup

**Modify:**
- `network/cmd/chainbench-net/handlers_network.go` — add `newHandleNetworkStopAll` + `newHandleNetworkStatus`
- `network/cmd/chainbench-net/handlers.go` — register `"network.stop_all"` + `"network.status"`
- `network/cmd/chainbench-net/handlers_test.go` — 8 unit tests (4 per handler)
- `network/cmd/chainbench-net/handlers_node_lifecycle.go` — extend `node.start` to accept `binary_path`
- `network/schema/command.json` — add `"network.status"` + `"network.stop_all"` to enum (regenerate)
- `mcp-server/src/tools/lifecycle.ts` — reroute `chainbench_stop` + `chainbench_status` (init/start/restart unchanged for now)
- `mcp-server/src/tools/node.ts` — remove `binary_path` fallback (Sprint 5c.3 P3 row 닫힘); pass `binary_path` to wire
- `mcp-server/test/lifecycle.test.ts` (NEW) — 6 vitest tests for the 2 rerouted tools
- `mcp-server/test/integration/node_rpc.integration.test.ts` — refactor to use `_harness.ts`
- `mcp-server/test/integration/lifecycle.integration.test.ts` (NEW) — 1 real-binary test for `chainbench_status`
- `mcp-server/test/node.test.ts` / `node_start_fallback.test.ts` — delete fallback test, add wire passthrough test for `binary_path`
- `mcp-server/package.json` — 0.7.0 → 0.7.1 (patch — 2/5 lifecycle tool rerouted, not minor)
- `mcp-server/src/index.ts` — McpServer version 0.7.1
- `docs/EVALUATION_CAPABILITY.md` — Sprint 5c.4.1 row + reroute progress 5/38
- `docs/VISION_AND_ROADMAP.md` — Sprint 5c.4.1 박스
- `docs/NEXT_WORK.md` + `docs/REMAINING_WORK.md` — timeline + P1 + P3 갱신

## 6. Tests

### 6.1 Go (Tasks 1, 3, 5)

`handlers_test.go` 신규 8 + 1 = 9 tests:

**network.stop_all** (4):
- `TestHandleNetworkStopAll_Happy` — local network, mock chainbench.sh exits 0
- `TestHandleNetworkStopAll_RemoteRejected` — `network: "sepolia"` → NOT_SUPPORTED
- `TestHandleNetworkStopAll_BashFailure` — mock chainbench.sh exits 1 → UPSTREAM_ERROR
- `TestAllHandlers_IncludesNetworkStopAll`

**network.status** (4):
- `TestHandleNetworkStatus_Happy` — mock chainbench.sh outputs JSON, handler parses + returns
- `TestHandleNetworkStatus_RemoteRejected`
- `TestHandleNetworkStatus_BadJsonOutput` → INTERNAL (invariant)
- `TestAllHandlers_IncludesNetworkStatus`

**node.start binary_path** (1):
- `TestHandleNodeStart_BinaryPath` — args 에 `binary_path` 포함 시 wire envelope 에 전달

Mock chainbench.sh: tests use `t.TempDir()` 에 fake `chainbench.sh` 작성
(echo + exit 종료 코드 제어). `CHAINBENCH_DIR` env 로 spawn target 지시.

### 6.2 TS unit (Tasks 2, 4)

`mcp-server/test/lifecycle.test.ts` (NEW), 6 tests:

**chainbench_stop** (3):
- `_Happy` — mock binary returns ok:true; assert text contains "Done." or stdout
- `_StrictRejectsUnknownKeys`
- `_WireFailure_PassedThrough` — mock returns ok:false UPSTREAM_ERROR

**chainbench_status** (3):
- `_Happy` — mock returns ok:true with JSON status; assert text contains JSON
- `_StrictRejectsUnknownKeys`
- `_WireFailure_PassedThrough`

### 6.3 TS integration (Task 4)

`mcp-server/test/integration/lifecycle.integration.test.ts` (NEW):
- 1 happy-path test using `_harness.ts` — spawn real `chainbench-net`, stub
  bash chainbench.sh with a fake script in temp dir, call `_statusHandler`,
  assert response

`mcp-server/test/integration/node_rpc.integration.test.ts` — refactor to
use `_harness.ts`. Same test passes (regression check on refactor).

### 6.4 TS removed

`mcp-server/test/node_start_fallback.test.ts` — DELETE. Sprint 5c.3 의
fallback 코드 제거됨. 새 `_NodeStart_BinaryPath_PassesToWire` test 가
node.test.ts 에 추가되어 binary_path 가 wire 로 전달됨을 검증.

### 6.5 Test count delta

- Go: 16 packages → 16 packages (handlers_test.go +9 tests)
- vitest: 94 → 94 - 2 (fallback delete) + 6 lifecycle + 1 integration + 1 node binary_path = **100**
- bash: 34/34 unchanged

## 7. Schema

`network/schema/command.json` enum 에 추가 (알파벳 순):
- `"network.status"`
- `"network.stop_all"`

`go generate ./...` 으로 `command_gen.go` 재생성. 같은 commit 에 포함.

## 8. Error Classification

| 단계 | 코드 | 사용처 |
|---|---|---|
| Zod schema | (zod throw) | unknown key, bad enum, malformed args |
| Tool handler | (zod handles cross-field via `.strict()`) | — |
| Wire (Go handler) | INVALID_ARGS | args 파싱 실패 |
| Wire (Go handler) | NOT_SUPPORTED | network != "local" (lifecycle 은 local 전용) |
| Wire (Go handler) | UPSTREAM_ERROR | bash spawn 실패, non-zero 종료 |
| Wire (Go handler) | INTERNAL | bash 출력이 JSON 아님 (status 만 — invariant) |

## 9. Out-of-Scope Reminders

**5c.4.2 으로 이행**:
- `chainbench_init` / `_start` / `_restart` reroute (bash spawn pattern repeat)
- `chainbench_clean` MCP 신규 노출 (현재 MCP 미노출)
- Bash spawn 패턴의 args shell-escape (init/start 가 profile/binary_path 받음)
- `network.status` 의 `--remote` 옵션 지원

**Sprint 5 series 외**:
- Native Go 포팅 (모든 lifecycle 명령) — 별도 sprint
- Capability-aware dispatch
- WS subscription / chain log streaming

## 10. Migration / Backwards Compat

- `chainbench_stop` / `chainbench_status` MCP tool 시그니처는 호환:
  - 기존: args 없음
  - 신규: optional `network` 추가 (defaults `"local"` — 기존 동작 보존)
- bash CLI `chainbench stop` / `status` 영향 없음
- `chainbench_node_start binary_path` 의 fallback 제거 — 시그니처 동일, 내부
  구현만 wire 경유로 통일 (LLM caller 영향 없음)
- 패키지 버전 0.7.0 → 0.7.1 (patch — 2/5 lifecycle tool rerouted; minor bump
  은 5c.4 series 전체 완료 시점에 0.8.0)
