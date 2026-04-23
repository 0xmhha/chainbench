# Sprint 2b.3 — node.start + node.restart Design

> **작성일**: 2026-04-23
> **목적**: Sprint 2b.2에서 확립한 LocalDriver 패턴을 확장하여 `node.start` 와 `node.restart` 커맨드를 구현한다. restart는 Go 핸들러에서 stop+start를 합성 (bash CLI에 restart 없음).

---

## 1. Goal & Scope

### 1.1 Goal
- `{"command":"node.start","args":{"node_id":"node1"}}` → `chainbench.sh node start 1` subprocess → `node.started` 이벤트 + 결과
- `{"command":"node.restart","args":{"node_id":"node1"}}` → 핸들러가 stop+start 순차 실행 → `node.stopped` + `node.started` 2개 이벤트 + 결과

### 1.2 Out of Scope
- `node.tail_log` (streaming, 별도 sprint)
- `node.rpc` (HTTP 경로, 별도 sprint)
- `restart` 중간 지연 시간 튜닝 (stop이 이미 동기적으로 프로세스 종료 대기)
- Retry/rollback (start 실패 시 stop 상태 유지 — 운영자가 재시도)

### 1.3 Success Criteria
- `cd network && go test -race ./...` green
- Coverage: `drivers/local` ≥ 85%, `cmd/chainbench-net` ≥ 80% 유지
- E2E: stub script로 start + restart 전체 pipeline 검증 (포함 이벤트 순서)

---

## 2. Package Structure

```
network/
├── internal/drivers/local/
│   ├── start.go                # 신규 — StartNode wrapper (StopNode와 대칭)
│   └── start_test.go
└── cmd/chainbench-net/
    ├── handlers.go             # + handleNodeStart, handleNodeRestart
    ├── handlers_test.go        # + 9 tests
    ├── e2e_test.go             # + 2 E2E (start, restart)
    └── testdata/
        └── chainbench-stub.sh  # 수정: node start 케이스 추가
```

이전 Sprint의 Driver 구조/인터페이스는 그대로 재사용. `Run(ctx, args...)`가 모든 것을 담당.

---

## 3. Type Contracts

### 3.1 `drivers/local/start.go`
```go
// StartNode invokes `chainbench.sh node start <nodeNum>`. Mirrors StopNode.
// Performs no validation — caller responsibility.
func (d *Driver) StartNode(ctx context.Context, nodeNum string) (*RunResult, error)
```

### 3.2 `cmd/chainbench-net/handlers.go` 추가
```go
// newHandleNodeStart returns a Handler for node.start.
// Args: {node_id: "nodeN"}. On success: emits node.started event + returns
// {node_id, started:true}.
func newHandleNodeStart(stateDir, chainbenchDir string) Handler

// newHandleNodeRestart returns a Handler for node.restart.
// Composition: stop → start. Emits node.stopped + node.started events in
// order. On stop failure: UPSTREAM_ERROR (no start attempted).
// On start failure after successful stop: UPSTREAM_ERROR (node left stopped).
// Returns {node_id, restarted:true}.
func newHandleNodeRestart(stateDir, chainbenchDir string) Handler
```

### 3.3 `allHandlers` update
```go
func allHandlers(stateDir, chainbenchDir string) map[string]Handler {
    return map[string]Handler{
        "network.load":  newHandleNetworkLoad(stateDir),
        "node.stop":     newHandleNodeStop(stateDir, chainbenchDir),
        "node.start":    newHandleNodeStart(stateDir, chainbenchDir),
        "node.restart":  newHandleNodeRestart(stateDir, chainbenchDir),
    }
}
```

### 3.4 Shared helper (refactor in handlers.go)
`handleNodeStop` 와 새 핸들러들이 중복하는 로직:
- args 파싱 + `node_id` prefix 검증 + 숫자 추출
- `state.LoadActive` 로 노드 존재 확인

→ Sprint 2b.2의 `handleNodeStop` 구현에서 추출된 helper로 `resolveNodeID(stateDir, args)` 를 만들어 재사용. 이는 Sprint 2b.2 코드의 작은 리팩토링 (기능 변경 없음).

```go
// resolveNodeID parses {node_id} from args, validates prefix, extracts numeric
// suffix, and verifies existence in state/pids.json. Returns (nodeID, nodeNum).
func resolveNodeID(stateDir string, args json.RawMessage) (nodeID, nodeNum string, err error)
```

---

## 4. Data Flow

### 4.1 node.start
```
{"command":"node.start","args":{"node_id":"node1"}}
  → handleNodeStart:
    1. resolveNodeID → ("node1", "1", nil)
    2. driver.StartNode(ctx, "1")
       ├ err → UPSTREAM_ERROR
       └ exitCode != 0 → UPSTREAM_ERROR (stderr tail)
    3. bus.Publish(Event{Name:"node.started", Data:{node_id:"node1"}})
    4. Return {node_id:"node1", started:true}
```

### 4.2 node.restart (composition)
```
{"command":"node.restart","args":{"node_id":"node1"}}
  → handleNodeRestart:
    1. resolveNodeID → ("node1", "1")
    2. driver.StopNode(ctx, "1")
       ├ err or exitCode != 0 → UPSTREAM_ERROR "restart aborted: stop failed"
    3. bus.Publish(Event{Name:"node.stopped", Data:{node_id,reason:"restart"}})
    4. driver.StartNode(ctx, "1")
       ├ err or exitCode != 0 → UPSTREAM_ERROR "restart incomplete: stop ok, start failed"
    5. bus.Publish(Event{Name:"node.started", Data:{node_id}})
    6. Return {node_id, restarted:true}
```

**불변식**: 이벤트 순서는 반드시 `stopped` → `started`. Start 실패 시 `started` 이벤트는 emit 안 됨.

---

## 5. Error Mapping

| 상황 | code | exit |
|---|---|---|
| args 파싱/prefix 실패 | INVALID_ARGS | 1 |
| node_id 존재 안 함 | INVALID_ARGS | 1 |
| subprocess 실행 실패 | UPSTREAM_ERROR | 1 |
| start exit != 0 | UPSTREAM_ERROR | 1 |
| restart: stop 실패 | UPSTREAM_ERROR | 1 (stopped 이벤트 안 냄) |
| restart: stop 성공 + start 실패 | UPSTREAM_ERROR | 1 (stopped 이벤트는 냄, started는 안 냄) |

---

## 6. Testing Strategy

### 6.1 Driver unit tests
- `TestStartNode_CallsCorrectArgs` — 주입된 exec로 args 검증 `["chainbench.sh","node","start","3"]`

### 6.2 Handler tests (in handlers_test.go)

**start**:
- `TestHandleNodeStart_HappyPath`
- `TestHandleNodeStart_MissingNodeID`
- `TestHandleNodeStart_UnknownNodeID`
- `TestHandleNodeStart_SubprocessFails`

**restart**:
- `TestHandleNodeRestart_HappyPath_EmitsBothEvents` (순서 검증)
- `TestHandleNodeRestart_StopFails` (no started event, stop 실패만 전파)
- `TestHandleNodeRestart_StartFailsAfterStop` (stopped 이벤트는 나지만 started는 안 남)
- `TestHandleNodeRestart_MissingNodeID`

**registration**:
- `TestAllHandlers_IncludesNodeStartRestart`

### 6.3 E2E
- `TestE2E_NodeStart_ViaRootCommand`
- `TestE2E_NodeRestart_ViaRootCommand` — stdout 파싱해서 `node.stopped` 이벤트가 `node.started` 이벤트보다 먼저 오는지 확인

### 6.4 Stub script 확장
```bash
case "$subcmd $action" in
  "node stop")    # 기존 그대로
  "node start")   # 신규
    node="${3:-}"
    if [[ "$node" == "fail" ]]; then exit 1; fi
    echo "stub: node $node started"
    exit 0
    ;;
esac
```

---

## 7. Refactor: `resolveNodeID` helper

Sprint 2b.2의 `handleNodeStop`이 다음 블록을 가짐:
```go
// parse args.node_id, validate prefix "node", extract suffix, verify LoadActive.Nodes contains id
```

이것이 3개 핸들러에 복붙되면 DRY 위반. 2b.3 Task 0에서 추출:
- 새 파일/같은 파일 상관없이 `resolveNodeID(stateDir, args) (nodeID, nodeNum string, err error)`
- `handleNodeStop`, `handleNodeStart`, `handleNodeRestart` 가 공유

기능 변경 없음 — 기존 테스트 모두 그대로 통과.

---

## 8. 완료 기준 (DoD)

1. `drivers/local/start.go` + test (StartNode)
2. `testdata/chainbench-stub.sh` — `node start` 케이스 추가
3. `cmd/chainbench-net/handlers.go` — `resolveNodeID` helper 추출 + `handleNodeStart` + `handleNodeRestart` + `allHandlers` 업데이트
4. `cmd/chainbench-net/handlers_test.go` — 9 new tests (start 4, restart 4, registration 1)
5. `cmd/chainbench-net/e2e_test.go` — 2 E2E (start, restart with event order check)
6. `cd network && go build ./... && go test -race ./... && go vet ./... && gofmt -l .` green
7. Coverage 목표 유지
8. 커밋 메시지 `network: ...` 프리픽스, 영어, co-author 없음

다음 단계: `writing-plans`로 implementation plan 작성 → `subagent-driven-development`로 실행.
