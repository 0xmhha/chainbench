# Sprint 2b.1 — network.load Command Design

> **작성일**: 2026-04-22
> **목적**: `chainbench-net`의 첫 wire-compliant 커맨드 `network.load`를 end-to-end 구현한다. Sprint 2a의 wire/events 라이브러리를 실제 사용하는 첫 사례이며, 이후 모든 커맨드의 템플릿이 된다. LocalDriver(subprocess wrapping)는 Sprint 2b.2의 범위.

---

## 1. Goal & Scope

### 1.1 Goal
- 파이프 입력 `echo '{"command":"network.load","args":{"name":"local"}}' | chainbench-net run` → stdout NDJSON으로 `Network` 객체를 `type=result, ok=true, data={name,chain_type,chain_id,nodes[]}` 형태로 emit
- 기존 `state/pids.json` + `state/current-profile.yaml`을 읽어 `types.Network`로 변환
- Exit code는 §5 에러 매핑 표에 따라 0/1/2/3 설정
- panic은 `INTERNAL` 결과로 변환 후 exit 1

### 1.2 Out of Scope (2b.2 이후)
- LocalDriver — subprocess wrapping of `lib/cmd_*.sh`
- `node.rpc`, `node.start`, `node.stop`, `node.tail_log` 등 노드 제어 커맨드
- `network.probe` (chain_type auto-detection)
- bash `network_client.sh` (Sprint 2c)
- 실제 기동 중이지 않은 상태(pids.json 없을 때)의 "idle chain" 표현

### 1.3 MVP 제약
- `args.name`은 **"local"만** 지원. 다른 값은 `INVALID_ARGS`.
- `state/pids.json`이 없거나 파싱 실패 시 `UPSTREAM_ERROR`.
- `state/current-profile.yaml`이 없으면 `UPSTREAM_ERROR`.
- chain_type은 profile에 명시 필드가 없으므로 **"stablenet" 기본값** (향후 profile 스키마 확장 시 `chain.type` 필드 추가 예정).

---

## 2. Package Structure (Minimalist)

```
network/
├── internal/
│   └── state/                       # 신규 — 파일 시스템 상태 → types.Network 변환
│       ├── doc.go
│       ├── profile.go               # profile YAML 파서 (필요 필드만)
│       ├── profile_test.go
│       ├── pids.go                  # pids.json 파서
│       ├── pids_test.go
│       ├── network.go               # profile + pids → types.Network
│       ├── network_test.go
│       └── testdata/                # 테스트 fixture
│           ├── profile-default.yaml
│           └── pids-default.json
└── cmd/chainbench-net/
    ├── main.go                      # (기존) cobra root + version + main — 수정
    ├── main_test.go                 # (기존) version 테스트 — 유지
    ├── run.go                       # 신규 — `run` 서브커맨드, stdin 디스패치
    ├── run_test.go
    ├── handlers.go                  # 신규 — command별 handler (network.load)
    ├── handlers_test.go
    ├── errors.go                    # 신규 — APIError + exitCode 매퍼
    ├── errors_test.go
    └── testdata/                    # E2E 테스트 fixture (state 디렉토리 시뮬)
        ├── pids.json
        └── current-profile.yaml
```

**Controller / NodeHandle는 별도 패키지 없음** — main.go 내부 map이 dispatch table 역할. 향후 2b.2에서 필요 시 추출.

### 2.1 Dependency graph

```
cmd/chainbench-net ──► internal/state ──► internal/types (generated)
        │
        └─► internal/wire, internal/events  (Sprint 2a 산출물)
```

단방향. state 패키지는 wire/events를 의존하지 않음.

---

## 3. Type Contracts / API

### 3.1 `internal/state/profile.go`

```go
// Profile represents the subset of profile YAML fields needed by state.LoadActive.
// Extend lazily as more commands require more fields.
type Profile struct {
    Name        string     // top-level "name" (e.g., "default")
    Chain       ChainBlock // "chain:" section
    Nodes       NodesBlock // "nodes:" section
    Ports       PortsBlock // "ports:" section
}

type ChainBlock struct {
    Binary     string // "chain.binary" (e.g., "gstable")
    BinaryPath string // "chain.binary_path"
    ChainID    int64  // "chain.chain_id" (EVM chain ID, integer)
    NetworkID  int64  // "chain.network_id"
    Type       string // "chain.type" — may be empty (default: "stablenet")
}

type NodesBlock struct {
    Validators int    // "nodes.validators"
    Endpoints  int    // "nodes.endpoints"
}

type PortsBlock struct {
    BaseHTTP int // "ports.base_http"
    BaseWS   int // "ports.base_ws"
    BaseP2P  int // "ports.base_p2p"
}

// ParseProfile parses a profile YAML from r. Missing optional fields get zero
// values; consumers apply defaults.
func ParseProfile(r io.Reader) (*Profile, error)

// ReadProfileFile is a convenience wrapper around ParseProfile.
func ReadProfileFile(path string) (*Profile, error)
```

### 3.2 `internal/state/pids.go`

```go
// PIDsFile mirrors the shape of state/pids.json.
type PIDsFile struct {
    ChainID   string              `json:"chain_id"`   // generated instance ID (string)
    Profile   string              `json:"profile"`
    StartedAt string              `json:"started_at"`
    Nodes     map[string]NodeInfo `json:"nodes"`      // key = "1".."N"
}

type NodeInfo struct {
    PID         int    `json:"pid"`
    Type        string `json:"type"`          // "validator" | "endpoint"
    P2PPort     int    `json:"p2p_port"`
    HTTPPort    int    `json:"http_port"`
    WSPort      int    `json:"ws_port"`
    AuthPort    int    `json:"auth_port"`
    MetricsPort int    `json:"metrics_port"`
    Status      string `json:"status"`
    LogFile     string `json:"log_file"`
    Binary      string `json:"binary"`
    Datadir     string `json:"datadir"`
    // saved_args intentionally elided — not needed for Network JSON
}

// ParsePIDs parses state/pids.json from r.
func ParsePIDs(r io.Reader) (*PIDsFile, error)

// ReadPIDsFile is a convenience wrapper.
func ReadPIDsFile(path string) (*PIDsFile, error)
```

### 3.3 `internal/state/network.go`

```go
// LoadActiveOptions controls how state files map to a Network.
type LoadActiveOptions struct {
    StateDir string // default: "state/"
    Name     string // Network.name; default: "local"
}

// LoadActive reads pids.json + current-profile.yaml under opts.StateDir and
// builds a Network object suitable for wire emission.
//
// Missing state files return a wrapped sentinel that the command layer maps
// to UPSTREAM_ERROR. Malformed content returns wrapped parse errors.
func LoadActive(opts LoadActiveOptions) (*types.Network, error)

// ErrStateNotFound is returned when required state files are missing.
var ErrStateNotFound = errors.New("state: active chain state not found")
```

**Mapping rules** (pids + profile → Network):
| Network field | Source |
|---|---|
| `name` | `opts.Name` (default "local") |
| `chain_type` | `profile.Chain.Type` (default "stablenet" when empty) |
| `chain_id` | `profile.Chain.ChainID` (integer) |
| `nodes[i].id` | `"node" + pidsKey` (e.g., "node1") |
| `nodes[i].role` | `NodeInfo.Type` mapped: `"validator"→validator`, `"endpoint"→endpoint`, anything else → `observer` |
| `nodes[i].provider` | `"local"` (Sprint 2b.1 is local-only) |
| `nodes[i].http` | `"http://127.0.0.1:" + NodeInfo.HTTPPort` |
| `nodes[i].ws`   | `"ws://127.0.0.1:"   + NodeInfo.WSPort`   |
| `nodes[i].provider_meta` | `{"pid_key": "<N>"}` — future drivers can reach back to pids.json |

Node ordering: sorted by numeric pidsKey (ascending) for determinism.

### 3.4 `cmd/chainbench-net/errors.go`

```go
// APIError carries a wire error code + message. Command handlers return this
// (or a generic error, treated as INTERNAL) to the run dispatcher, which emits
// a result=error and selects the exit code.
type APIError struct {
    Code    types.ResultErrorCode
    Message string
    Cause   error // optional underlying error for logging
}

func (e *APIError) Error() string { return ... }
func (e *APIError) Unwrap() error { return e.Cause }

// Helpers for common shapes.
func NewInvalidArgs(msg string) *APIError
func NewNotSupported(msg string) *APIError
func NewUpstream(msg string, cause error) *APIError
func NewProtocolError(msg string, cause error) *APIError
func NewInternal(msg string, cause error) *APIError

// exitCode maps an error (possibly nil) to an OS exit code per VISION §5.
//  nil                         → 0
//  APIError.Code=NOT_SUPPORTED → 2
//  APIError.Code=PROTOCOL_ERROR→ 3
//  APIError.Code=INVALID_ARGS  → 1
//  APIError.Code=UPSTREAM_ERROR→ 1
//  APIError.Code=INTERNAL      → 1
//  any other error             → 1 (treated as INTERNAL)
func exitCode(err error) int
```

### 3.5 `cmd/chainbench-net/run.go`

```go
// newRunCmd builds the "run" subcommand. It reads a wire command envelope
// from stdin, dispatches to a handler, and emits the result as NDJSON on
// stdout. Structured logs go to stderr. Exit code follows exitCode(err).
func newRunCmd() *cobra.Command

// runOnce is the testable entry: stdin reader, stdout writer, stderr writer,
// and a dispatch table. Separating the IO from cobra/os simplifies tests.
//
// Returns the error to be surfaced; main.go translates to exit code.
func runOnce(stdin io.Reader, stdout io.Writer, stderr io.Writer,
             handlers map[string]Handler) error
```

**Handler signature:**
```go
type Handler func(args json.RawMessage, bus *events.Bus) (result map[string]any, err error)
```

- `args` passed as raw JSON so handlers can unmarshal into a command-specific struct
- `bus` available if handler needs to emit progress/events mid-flight (network.load doesn't, but future handlers will)
- Return `(data, nil)` on success → `EmitResult(true, data)`
- Return `(_, *APIError)` → `EmitResultError(code, msg)`
- Return `(_, other err)` → `EmitResultError(INTERNAL, err.Error())`

### 3.6 `cmd/chainbench-net/handlers.go`

```go
// handleNetworkLoad is the handler for command "network.load".
// args shape: { "name": string } — only "local" is currently supported.
func handleNetworkLoad(args json.RawMessage, bus *events.Bus) (map[string]any, error)

// allHandlers returns the full dispatch table. Registered in main/run wiring.
func allHandlers(stateDir string) map[string]Handler
```

Note: `stateDir` is passed via closure when building the dispatch table, not per-call. Keeps Handler signature clean.

### 3.7 `cmd/chainbench-net/main.go` (modifications)

- Add `root.AddCommand(newRunCmd())`
- Main function: on `Execute()` error, check for `*exitError` via `errors.As`, use its code; else default to 1. Keep existing stderr message.

---

## 4. Data Flow

### 4.1 Happy path (`network.load`)

```
$ echo '{"command":"network.load","args":{"name":"local"}}' | chainbench-net run
     │                                                           │
     └──stdin──────────────────────────────────────┐             │
                                                    ▼             │
                                    wire.DecodeCommand            │
                                    (returns *types.Command)      │
                                                    │             │
                                                    ▼             │
                                      dispatch by cmd.Command     │
                                                    │             │
                                                    ▼             │
                          handleNetworkLoad(args, bus):           │
                            - Unmarshal args → {Name}             │
                            - Validate Name == "local"            │
                            - state.LoadActive({StateDir,Name})   │
                                - state.ReadProfileFile           │
                                - state.ReadPIDsFile              │
                                - construct types.Network         │
                            - return map[string]any(network), nil │
                                                    │             │
                                                    ▼             │
                                emitter.EmitResult(true, data) ───┼──► stdout NDJSON
                                                                  │
                                                  slog.Info/Warn ─┘──► stderr (structured log)
                                                                  
                                                  exit 0
```

### 4.2 Error path — unknown command
```
{"command":"bogus","args":{}}  (but wire.DecodeCommand itself rejects unknown
  enum values via types.Command.UnmarshalJSON — so this is caught at decode)
→ APIError{PROTOCOL_ERROR, "unknown enum"} 
→ EmitResultError(PROTOCOL_ERROR, ...)
→ exit 3
```

### 4.3 Error path — pids.json missing
```
handleNetworkLoad → state.LoadActive → ErrStateNotFound
→ wrapped as APIError{UPSTREAM_ERROR, "state/pids.json not found", cause}
→ EmitResultError(UPSTREAM_ERROR, ...)
→ exit 1
```

### 4.4 Error path — panic
```
defer recover() in runOnce → EmitResultError(INTERNAL, "panic: ...")
→ exit 1
```

---

## 5. Error Mapping Table (§5 of VISION)

Implemented by `exitCode(err)`:

| 상황 | APIError Code | Exit |
|---|---|---|
| stdin decode 실패 (malformed JSON) | PROTOCOL_ERROR | 3 |
| 알 수 없는 `command.command` enum | PROTOCOL_ERROR (wire gen rejects) | 3 |
| `args.name` invalid | INVALID_ARGS | 1 |
| state file missing / parse fail | UPSTREAM_ERROR | 1 |
| Handler not registered | NOT_SUPPORTED | 2 |
| panic | INTERNAL | 1 |
| Success | — | 0 |

---

## 6. Testing Strategy

### 6.1 Unit tests per file

- `internal/state/profile_test.go` — YAML 파싱 (필드 누락·빈 파일·잘못된 구조)
- `internal/state/pids_test.go` — JSON 파싱 (malformed, 빈 nodes map)
- `internal/state/network_test.go` — LoadActive 통합: fixture 두 파일 → types.Network 생성, 스키마 검증 (기존 `network/schema.ValidateBytes`로 cross-check)
- `cmd/chainbench-net/errors_test.go` — APIError 생성자 + exitCode 분기
- `cmd/chainbench-net/run_test.go` — runOnce: 정상 dispatch, 알 수 없는 command, panic recovery
- `cmd/chainbench-net/handlers_test.go` — handleNetworkLoad: args 파싱, name 검증, state 에러 처리

### 6.2 End-to-end test

`cmd/chainbench-net/main_test.go` (또는 새 `e2e_test.go`):
- `testdata/pids.json` + `testdata/current-profile.yaml` fixture 생성
- `runOnce`를 직접 호출 (subprocess spawn 불필요 — 테스트는 binary 없이도 돈다)
- stdout buffer → 라인별 `wire.DecodeMessage` → ResultMessage의 data 필드가 Network schema 통과 확인

### 6.3 Race detector
```
cd network && go test -race ./internal/state/... ./cmd/chainbench-net/...
```

### 6.4 Coverage 목표
- `internal/state/`: ≥ 85%
- `cmd/chainbench-net/`: ≥ 80% (cobra boilerplate 포함하므로 약간 낮춤)

---

## 7. Concurrency & Invariants

- `runOnce`는 single-threaded dispatch. Handler 내부에서 goroutine 사용 시 `bus.Publish` 경유 (thread-safe).
- `state.*` 함수는 순수 read-only. 동시 호출 안전.
- `APIError`는 immutable (pointer 반환이지만 필드 수정 없음).

---

## 8. 보안 경계 주의

- `state/pids.json`의 `binary` 필드는 절대 경로. stderr log에 포함 가능 (OS 사용자 기밀 아님). 
- 환경 변수 로깅 금지 — `CHAINBENCH_SIGNER_*` 류 변수가 Sprint 4에서 도입될 예정이므로 미리 원칙 확립.
- `APIError.Cause`는 internal logging 용. 외부 result의 `error.message`에는 포함하지 않음 (information disclosure 방지).

---

## 9. 완료 기준 (Definition of Done)

1. `internal/state/{profile,pids,network}.go` + 각 `*_test.go` (fixture 포함)
2. `cmd/chainbench-net/{run,handlers,errors}.go` + 각 `*_test.go`
3. E2E 테스트: stdin envelope → stdout NDJSON, 기존 `network/schema` 로 cross-validate
4. `cd network && go build ./... && go test -race ./... && go vet ./... && gofmt -l .` 전부 green
5. Coverage 목표 달성 (state ≥ 85%, cmd ≥ 80%)
6. 커밋 메시지 `network: ...` 프리픽스, 영어, co-author 없음
7. `chainbench-net run` 수동 실행으로 happy path 확인 (`echo ... | chainbench-net run`)

다음 단계: `writing-plans`로 implementation plan 작성 → `subagent-driven-development`로 실행.
