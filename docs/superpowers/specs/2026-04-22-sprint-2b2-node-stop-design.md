# Sprint 2b.2 — node.stop Command via LocalDriver Design

> **작성일**: 2026-04-22
> **목적**: 첫 subprocess-backed 커맨드 `node.stop`을 구현한다. `chainbench.sh node stop <N>` 를 Go subprocess로 감싸는 LocalDriver 기반을 확립하고, 이를 통해 향후 `node.start/restart/tail_log` 확장이 모두 동일 패턴을 따르도록 한다.

---

## 1. Goal & Scope

### 1.1 Goal
- `{"command":"node.stop","args":{"node_id":"node1"}}` 입력 시:
  1. pids.json에서 node_id 존재 검증
  2. `chainbench.sh node stop <N>` subprocess 실행 (N은 node_id의 숫자 suffix)
  3. subprocess stdout/stderr를 slog로 스트리밍
  4. 성공 시 `node.stopped` 이벤트 emit + `{node_id, stopped:true}` 결과
  5. 실패 시 적절한 APIError + exit code

### 1.2 Out of Scope
- `node.start`, `node.restart`, `node.tail_log`, `node.rpc` — 후속 sprint
- Remote/SSH driver
- 동시 command 처리
- 프로세스 grace 타임 튜닝
- tail_log 류 streaming 커맨드 (이벤트 emitter 장기 보유 모델)

### 1.3 Success Criteria
- `cd network && go test -race ./...` 모두 green
- Coverage: `internal/drivers/local` ≥ 85%, `cmd/chainbench-net` ≥ 80% 유지
- E2E: testdata의 stub script로 full pipeline 검증 (stdin → wire → handler → subprocess → bus event → result terminator)

---

## 2. Package Structure

```
network/
├── internal/
│   └── drivers/
│       └── local/                    # 신규 패키지
│           ├── doc.go
│           ├── driver.go             # Driver + Run(ctx, args...)
│           ├── driver_test.go
│           ├── stop.go               # StopNode(ctx, nodeNum)
│           └── stop_test.go
└── cmd/chainbench-net/
    ├── handlers.go                   # + handleNodeStop (allHandlers 시그니처 확장)
    ├── handlers_test.go              # + 3 tests for node.stop
    ├── run.go                        # CHAINBENCH_DIR env 읽어 allHandlers에 전달
    ├── e2e_test.go                   # + E2E for node.stop
    └── testdata/
        └── chainbench-stub.sh        # 신규: fake chainbench.sh
```

### 2.1 Dependency graph

```
cmd/chainbench-net ──► drivers/local   (NEW)
                 └──► state, wire, events, types

drivers/local     ──► stdlib only (context, os/exec, io, bufio, log/slog, time)
```

단방향. drivers/local은 wire/events/state/types 모두 의존하지 않음 → 순수 process runner.

---

## 3. Type Contracts

### 3.1 `drivers/local/driver.go`
```go
// Driver executes chainbench CLI subcommands. Subprocess stdout/stderr
// stream into slog.Info / slog.Warn per line. Result struct captures full
// output for handler-side inspection (error tail, status parsing).
type Driver struct {
    chainbenchDir string
    // exec is a seam for tests; defaults to exec.CommandContext.
    exec func(ctx context.Context, name string, args ...string) *exec.Cmd
}

func NewDriver(chainbenchDir string) *Driver

// NewDriverWithExec is the testable constructor; prod code uses NewDriver.
func NewDriverWithExec(chainbenchDir string,
    exec func(ctx context.Context, name string, args ...string) *exec.Cmd) *Driver

type RunResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
    Duration time.Duration
}

// Run executes `<chainbenchDir>/chainbench.sh <args...>` as a subprocess.
// Streams stdout/stderr line-by-line to slog during execution; returns
// the full captured buffers + exit code + wall clock duration.
//
// Non-zero exit code does NOT return an error — the caller inspects
// RunResult.ExitCode. Only start errors / IO errors / context cancellation
// return a non-nil error.
func (d *Driver) Run(ctx context.Context, args ...string) (*RunResult, error)
```

### 3.2 `drivers/local/stop.go`
```go
// StopNode invokes `chainbench.sh node stop <nodeNum>`. nodeNum is the
// numeric pids.json key ("1", "2", ...). Thin wrapper over Run with a
// fixed argv shape.
func (d *Driver) StopNode(ctx context.Context, nodeNum string) (*RunResult, error)
```

### 3.3 `cmd/chainbench-net/handlers.go` 추가
```go
// newHandleNodeStop returns a Handler that stops a node by id.
// Args: { "node_id": "nodeN" }.
//
// Validates the node exists in pids.json via state.LoadActive (reusing
// LoadActive so we don't duplicate file parsing). Emits a node.stopped
// bus event on success. Returns {node_id, stopped:true}.
func newHandleNodeStop(stateDir, chainbenchDir string) Handler

// Updated signature:
func allHandlers(stateDir, chainbenchDir string) map[string]Handler
```

### 3.4 `cmd/chainbench-net/run.go` 수정
```go
// Pass chainbenchDir from env to handler registry.
func newRunCmd() *cobra.Command {
    return &cobra.Command{
        // ...
        RunE: func(cmd *cobra.Command, _ []string) error {
            stateDir := envOrDefault("CHAINBENCH_STATE_DIR", "state")
            chainbenchDir := envOrDefault("CHAINBENCH_DIR", ".")
            return runOnce(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(),
                allHandlers(stateDir, chainbenchDir))
        },
    }
}
```

---

## 4. Data Flow

```
{"command":"node.stop","args":{"node_id":"node1"}}
            │
            ▼
  handleNodeStop(args, bus):
    1. Parse args → req.NodeID = "node1"
    2. Strip "node" prefix → num = "1"
    3. state.LoadActive(stateDir) — verify pids.json has "1" key
        └─ missing → INVALID_ARGS "node_id not found"
    4. driver := local.NewDriver(chainbenchDir)
    5. result, err := driver.StopNode(ctx, "1")
         stdout/stderr lines stream to slog during call
        └─ err != nil → UPSTREAM_ERROR "subprocess exec failed"
        └─ result.ExitCode != 0 → UPSTREAM_ERROR "exit N: <stderr tail>"
    6. bus.Publish(Event{
         Name: "node.stopped",
         Data: {"node_id":"node1","reason":"manual"},
       })
    7. Return {"node_id":"node1","stopped":true}
            │
            ▼
  runOnce emits:
      NDJSON line 1 (event): {"type":"event","name":"node.stopped",...}
      NDJSON line 2 (result): {"type":"result","ok":true,"data":{...}}
```

### 4.1 Context handling
- `runOnce` 내부에서 `context.Background()` 시작 (cancel 없음 — 첫 MVP)
- `StopNode(ctx, ...)` 에 전달
- 향후 signal handler 추가 시 같은 ctx로 cancel 전파

### 4.2 slog stream 형식
```
{"time":"2026-04-22T...","level":"INFO","msg":"subprocess stdout","source":"local.Run","line":"stub: node 1 stopped"}
{"time":"2026-04-22T...","level":"WARN","msg":"subprocess stderr","source":"local.Run","line":"..."}
```

---

## 5. Error Mapping

| 상황 | APIError code | Exit code |
|---|---|---|
| `args` 파싱 실패 | INVALID_ARGS | 1 |
| `args.node_id` 비어있음 | INVALID_ARGS | 1 |
| `node_id` prefix가 "node"가 아님 | INVALID_ARGS | 1 |
| `state.LoadActive` 실패 (pids.json 없음) | UPSTREAM_ERROR | 1 |
| node_id가 pids.json에 없음 | INVALID_ARGS | 1 |
| `chainbench.sh` 실행 실패 (파일 없음, 권한 등) | UPSTREAM_ERROR | 1 |
| subprocess exit != 0 | UPSTREAM_ERROR (stderr tail 포함) | 1 |
| `ctx.Err() != nil` | UPSTREAM_ERROR | 1 |
| 정상 완료 | — | 0 |

---

## 6. Testing Strategy

### 6.1 Driver unit tests (injected fake exec)

**Fake exec pattern:**
```go
func fakeExec(stdout, stderr string, exitCode int) func(ctx context.Context, name string, args ...string) *exec.Cmd {
    return func(ctx context.Context, name string, args ...string) *exec.Cmd {
        // Use `exec.CommandContext(ctx, "sh", "-c", "<prints>")` to simulate.
        // Simpler: spawn `sh -c` with a one-liner that echoes + exits.
    }
}
```

Tests:
- `TestDriver_Run_Success` — fake prints "hello", exit 0 → RunResult{ExitCode:0, Stdout:"hello\n"}
- `TestDriver_Run_NonZeroExit` — fake exit 1 → RunResult.ExitCode=1, err=nil
- `TestDriver_Run_StderrCaptured` — stderr line → captured in RunResult.Stderr
- `TestDriver_Run_ContextCanceled` — cancel ctx before exec → err != nil
- `TestStopNode_CallsChainbenchCorrectArgs` — fake records args, assert `["node","stop","1"]`

### 6.2 Handler tests

Use real `exec.Command` with `testdata/chainbench-stub.sh` (executable) + copy state fixtures.

- `TestHandleNodeStop_HappyPath` — stub success, state exists → returns `{node_id:"node1",stopped:true}`, bus receives `node.stopped` event
- `TestHandleNodeStop_MissingArgs` — no args.node_id → INVALID_ARGS
- `TestHandleNodeStop_UnknownNode` — node_id "node99" not in pids.json → INVALID_ARGS
- `TestHandleNodeStop_StubFails` — stub with "fail" sentinel → UPSTREAM_ERROR with stderr content
- `TestAllHandlers_IncludesNodeStop` — registration check

### 6.3 E2E test

`TestE2E_NodeStop_ViaRootCommand` — full cobra Execute():
- testdata에 chainbench-stub.sh + pids.json + current-profile.yaml 세팅
- `CHAINBENCH_STATE_DIR` + `CHAINBENCH_DIR` env 설정
- stdin: `{"command":"node.stop","args":{"node_id":"node1"}}`
- stdout 파싱: event 1개(`node.stopped`) + result terminator
- schema.ValidateBytes("event", line) 각 라인 검증

### 6.4 Race detector
`go test -race ./internal/drivers/local/... ./cmd/chainbench-net/...`

---

## 7. Security Boundary

- `chainbench.sh` 경로는 `CHAINBENCH_DIR` env에서 읽음. 디렉토리 traversal 방지: Driver는 `chainbenchDir`을 그대로 사용하며 arbitrary path 조립 안 함 (argv[0] = `filepath.Join(dir, "chainbench.sh")`)
- subprocess env: 상속만 — signer env var는 Sprint 4에서 격리 설계 예정
- stdout/stderr는 slog로만 흘림 — **wire NDJSON stdout에는 subprocess raw 출력이 섞이지 않음** (중요한 wire 규약 유지)

---

## 8. 완료 기준 (DoD)

1. `internal/drivers/local/` 4 파일 (doc, driver, stop + 각 test) + stop_test
2. `cmd/chainbench-net/` 수정: handlers.go (+ handleNodeStop, allHandlers 시그니처), run.go (CHAINBENCH_DIR), handlers_test.go (+ 5 tests), e2e_test.go (+ 1 test), testdata/chainbench-stub.sh
3. `cd network && go build ./... && go test -race ./... && go vet ./... && gofmt -l .` 전부 green
4. Coverage 목표 달성
5. 각 task 단위 commit, `network: ...` 프리픽스, co-author 없음
6. 수동 smoke: 실제 `chainbench.sh` 와 연동해 `echo ... | chainbench-net run` 확인 (선택)

다음 단계: `writing-plans` → `subagent-driven-development`.
