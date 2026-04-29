# Sprint 5a — Capability Gate Design Spec

> 2026-04-29 · VISION §5.5 + §5.7 — Sprint 5a (capability gate)
> Scope: 기존 network.capabilities schema enum + test_meta.sh frontmatter parser
> 두 기존 자산을 wire 로 연결. Go 측 `network.capabilities` 핸들러 + provider
> 별 capability 추론 + MCP `chainbench_network_capabilities` tool + bash test
> runner 의 `requires_capabilities` skip gating + 예제 test frontmatter.

## 1. Goal

VISION §5.5 가 정의한 capability negotiation 의 첫 패스. 본 sprint 종료 시
다음 시나리오가 동작:

1. 사용자가 remote 네트워크를 `network.attach` 후 `chainbench test run all` 실행
2. `requires_capabilities: [process]` 이 표기된 fault test 들이 **자동 skip**
   되며 명확한 사유 출력 (`SKIP: tests/fault/node-crash.sh — requires
   capability: process; remote network 'devnet' provides: rpc ws`)
3. local 네트워크에서는 동일 test 가 정상 실행 (process capability 보유)
4. coding agent (MCP) 가 `chainbench_network_capabilities` tool 로 active
   network 의 capability set 을 읽고 어떤 시나리오가 자동화 가능한지 판단

## 2. Non-Goals (Deferred)

- **모든 기존 test 에 frontmatter 부여** — 본 sprint 는 fault/ 카테고리 1-2
  파일만 frontmatter 추가 (proof-of-concept). 나머지는 점진적으로 (트리거:
  remote 환경에서 test 실행 시 발견되는 부적합 케이스).
- **Capability 자동 발견 (probe-based)** — 현재 capability 는 provider
  declaration 에서만. Remote 노드의 실제 admin RPC 가능 여부 등 runtime
  probe 는 후속.
- **Per-node capability 차이** — Hybrid 네트워크의 노드별 capability 교집합
  로직은 본 sprint 에서 단순 처리 (provider type 만 보고). 본격 처리는 5d
  (hybrid 예제) 에서.
- **`network-topology` capability 의 실제 활용** — 본 sprint 는 capability
  표명만. partition/heal 시뮬레이션 등 활용은 후속.
- **MCP 측 capability check** — coding agent 가 capability 부재로 tool 호출
  실패 시 자동 retry/skip 하는 layer 는 후속. 본 sprint 는 capability set
  조회만 노출.

## 3. Security Contract

Capability 정보는 sensitive 하지 않음. wire 응답에 alias / key material 없음.
Sprint 4 redaction boundary 그대로 유지.

## 4. User-Facing Surface

### 4.1 Go: `network.capabilities` 핸들러 (Task 1)

```go
func newHandleNetworkCapabilities(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network string `json:"network"`
        }
        if len(args) > 0 {
            if err := json.Unmarshal(args, &req); err != nil {
                return nil, NewInvalidArgs(...)
            }
        }
        if req.Network == "" {
            req.Network = "local"
        }
        nodes, err := state.LoadNodes(stateDir, req.Network)
        if err != nil {
            return nil, err
        }
        // Infer capabilities: intersect across all nodes
        caps := inferCapabilities(nodes)
        return map[string]any{
            "network": req.Network,
            "capabilities": caps,
        }, nil
    }
}
```

**Capability inference rules** (per node provider):
- `local` provider → `["rpc", "ws", "process", "fs", "admin", "network-topology"]`
- `remote` provider → `["rpc", "ws"]`
- `ssh-remote` provider (future) → `["rpc", "ws", "process", "fs"]`
- 노드 집합의 capability = 모든 노드의 set 교집합 (가장 보수적)

**Result shape**:
```json
{
  "network": "local",
  "capabilities": ["rpc", "ws", "process", "fs", "admin", "network-topology"]
}
```

### 4.2 MCP: `chainbench_network_capabilities` (Task 3)

```typescript
const NetworkCapabilitiesArgs = z.object({
  network: z.string().min(1).optional()
    .describe("Network alias. Defaults to 'local'."),
}).strict();
```

Wire command: `network.capabilities`. Result: 위 shape 그대로.

Tool description: "Read the capability set of the active or named network. Capabilities indicate which operations are supported (rpc/ws always, process/fs/admin/network-topology only on local). Use this to gate features that depend on process control or filesystem access (e.g., node crash tests require 'process'). Local networks support all capabilities; remote-attached networks support only rpc/ws."

### 4.3 Bash: `requires_capabilities` skip gating (Task 4)

`lib/cmd_test.sh` 의 test runner 에 새 함수 추가:

```bash
# Returns 0 if test can run, 1 if it should be skipped (with reason).
_cb_test_check_capabilities() {
  local test_path="$1"
  local meta_json
  meta_json=$(cb_parse_meta "$test_path")
  local required
  required=$(echo "$meta_json" | jq -r '.requires_capabilities[]?' 2>/dev/null)
  [[ -z "$required" ]] && return 0  # no requirements → can run

  local active_caps
  active_caps=$(_cb_test_active_capabilities) || return 1

  local missing=()
  for r in $required; do
    if ! echo "$active_caps" | tr ' ' '\n' | grep -qx "$r"; then
      missing+=("$r")
    fi
  done

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "SKIP: $test_path — requires capability: ${missing[*]}; active network provides: $active_caps" >&2
    return 1
  fi
  return 0
}

_cb_test_active_capabilities() {
  # Use cb_net_call (lib/network_client.sh) → wire network.capabilities
  local data
  data=$(cb_net_call "network.capabilities" '{}')
  echo "$data" | jq -r '.capabilities[]' 2>/dev/null | tr '\n' ' '
}
```

Test runner 의 main loop 에서 각 test 실행 전 `_cb_test_check_capabilities`
호출. Skip 시 그 결과를 final report 에 별도 카운트.

### 4.4 Example test frontmatter (Task 5)

기존 `tests/fault/` 의 1-2 파일에 frontmatter 추가 (예: `node-crash.sh`,
`peer-disconnect.sh`):

```bash
#!/usr/bin/env bash
# ---chainbench-meta---
# description: Stop 1/4 validators and verify consensus continues
# requires_capabilities: [process]
# chain_compat: [stablenet, wbft]
# ---end-meta---
```

bash test runner 가 본 frontmatter 를 파싱하여 active network 의 capability
와 매칭. 이는 5a Pass 1 의 demo — 5a Pass 2+ 가 모든 fault / process-bound
test 에 frontmatter 점진 부여.

## 5. Package Layout

**Create:**
- (none — all changes are extensions to existing files or new test fixtures)

**Modify:**
- `network/cmd/chainbench-net/handlers_network.go` — add `newHandleNetworkCapabilities`
- `network/cmd/chainbench-net/handlers.go` — register `"network.capabilities"`
- `network/cmd/chainbench-net/handlers_test.go` — 4-5 tests
- `network/internal/state/network.go` (or wherever LoadNodes is) — possibly extend if a per-node provider type lookup helper is missing
- `mcp-server/src/tools/network.ts` — add `NetworkCapabilitiesArgs` + `_networkCapabilitiesHandler` + register
- `mcp-server/test/network.test.ts` (NEW) — 3-4 tests
- `lib/cmd_test.sh` — add `_cb_test_check_capabilities` + integration in main loop
- `tests/fault/node-crash.sh` (or similar) — add frontmatter
- `tests/unit/tests/cmd-test-capabilities.sh` (NEW) — bash unit test for the gating logic
- `docs/EVALUATION_CAPABILITY.md` — Sprint 5a row
- `docs/VISION_AND_ROADMAP.md` — Sprint 5a 박스
- `docs/NEXT_WORK.md` — §1/§2/§3/§4.6
- `mcp-server/package.json` + `index.ts` — version 0.6.0 → 0.7.0

## 6. Tests

### 6.1 Go (Task 1)

`handlers_test.go` neue:
- `TestHandleNetworkCapabilities_Local` — local network → all 6 caps
- `TestHandleNetworkCapabilities_Remote` — remote-attached → rpc + ws only
- `TestHandleNetworkCapabilities_Hybrid` — 1 local + 1 remote → intersection (rpc+ws)
- `TestHandleNetworkCapabilities_DefaultLocal` — args 미지정 → local
- `TestAllHandlers_IncludesNetworkCapabilities`

### 6.2 TS (Task 3)

`network.test.ts` neue:
- `_Happy_Local` — wire returns local caps; tool surfaces them
- `_Happy_Remote` — wire returns remote caps; tool surfaces
- `_StrictRejectsUnknownKeys`
- `_WireFailure_PassedThrough`

### 6.3 Bash (Task 4+5)

`tests/unit/tests/cmd-test-capabilities.sh` (NEW) — capability gate unit tests:
- `requires_capabilities: [process]` + active = `[rpc ws]` → skip
- `requires_capabilities: [process]` + active = `[rpc ws process fs]` → run
- `requires_capabilities` 누락 → run
- `requires_capabilities: []` → run
- chainbench-net unavailable → skip with diagnostic

본 unit test 는 chainbench-net mock binary 를 spawn 해서 capability response
를 흉내. 기존 bash test pattern (Sprint 4d 의 `node-tx-wait.sh` 등) 따라감.

## 7. Schema

`network/schema/command.json` 변경 없음 (`network.capabilities` 이미 enum).

## 8. Error Classification

기존 매트릭스. capability 조회 실패 → UPSTREAM_ERROR (state load 실패) 또는
INVALID_ARGS (network 이름 잘못됨).

## 9. Out-of-Scope Reminders

**5a Pass 2+ 으로 이행**:
- 모든 fault / process-bound test 에 frontmatter 부여
- `network-topology` capability 가 실제로 partition/heal 같은 시뮬레이션 시
  체크되도록
- MCP 측 capability-aware tool gating

**Sprint 5 series 외**:
- Per-node hybrid 시 노드별 capability 교집합 (5d)
- Runtime capability probe (admin RPC 응답 여부 등)

## 10. Migration / Backwards Compat

- `chainbench test run` 동작: frontmatter 가 없는 test 는 capability check
  없이 실행 (기존 동작 유지). frontmatter 가 있는 test 만 새 gating 적용.
- 따라서 기존 사용자 영향 없음. Remote 환경에서 `test run` 처음 시도 시
  fault test 가 자동 skip 되어 더 좋은 UX.
- 패키지 버전 0.6.0 → 0.7.0 (capability gate 첫 노출).
