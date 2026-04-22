# Sprint 2a — Wire Protocol + Events Foundation Design

> **작성일**: 2026-04-22
> **목적**: `chainbench-net` 바이너리의 transport primitives(`wire`) 및 in-process pub/sub 레이어(`events`)를 순수 라이브러리로 구현한다. CLI entry나 외부 가시 변화는 Sprint 2b의 범위.

---

## 1. Goal & Scope

### 1.1 Goal
`network/internal/wire/` + `network/internal/events/` 두 Go 패키지를 구현. 각각 단위 테스트로 완전 격리 검증. 모든 출력은 `network/schema/event.json` JSON Schema에 byte-for-byte 정합.

### 1.2 Deliverable
- Go 패키지 2개 + 단위 테스트 5 파일 + 통합 테스트 1 파일
- `go test ./internal/wire/... ./internal/events/...` PASS
- race detector 통과 (`go test -race`)
- Coverage: wire ≥ 85%, events ≥ 80%

### 1.3 Out of Scope
- `cmd/chainbench-net/main.go` 수정 (2b)
- Subprocess spawn, cmd_*.sh 래핑 (2b)
- bash 클라이언트 `lib/network_client.sh` (2c)
- Signer 경계 (Sprint 4)
- Schema 변경

---

## 2. Package Structure

```
network/internal/
├── wire/                  # Transport primitives
│   ├── doc.go
│   ├── command.go         # Stdin envelope decoder
│   ├── emitter.go         # Stdout NDJSON emitter (mutex, terminator guard)
│   ├── decoder.go         # StreamMessage tagged-union decoder
│   ├── logger.go          # Stderr slog JSON handler setup
│   ├── command_test.go
│   ├── emitter_test.go
│   ├── decoder_test.go
│   ├── logger_test.go
│   └── wire_schema_test.go  # Integration: output ↔ schema.ValidateBytes
└── events/                # Pub/sub built on wire
    ├── doc.go
    ├── bus.go             # EventBus: fan-out + wire.Emitter delegate
    └── bus_test.go
```

### 2.1 Dependency graph

```
events ──depends on──▶ wire ──depends on──▶ types (generated) / schema (embed validator)
```

단방향. 순환 없음.

### 2.2 File size targets

| 파일 | 권장 lines | 상한 |
|---|---|---|
| command.go | 30~50 | 100 |
| emitter.go | 80~120 | 200 |
| decoder.go | 100~150 | 200 |
| logger.go | 30~50 | 80 |
| events/bus.go | 80~120 | 200 |

전부 권장 상한(800) 내.

---

## 3. Type Contracts / Public API

### 3.1 `wire/command.go`
```go
package wire

import (
    "encoding/json"
    "fmt"
    "io"

    "github.com/0xmhha/chainbench/network/internal/types"
)

// DecodeCommand reads one JSON command envelope from r.
// Unknown fields are rejected (DisallowUnknownFields).
func DecodeCommand(r io.Reader) (*types.Command, error)
```

### 3.2 `wire/emitter.go`
```go
package wire

import (
    "encoding/json"
    "errors"
    "io"
    "sync"
    "time"

    "github.com/0xmhha/chainbench/network/internal/types"
)

// ErrStreamClosed is returned by Emitter methods after EmitResult/
// EmitResultError terminates the stream.
var ErrStreamClosed = errors.New("wire: stream already closed by result")

type Emitter struct {
    mu     sync.Mutex
    enc    *json.Encoder
    clock  func() time.Time  // injectable for tests
    closed bool
}

func NewEmitter(w io.Writer) *Emitter

// Thread-safe. ts field auto-populated from e.clock().UTC().
func (e *Emitter) EmitEvent(name types.EventName, data map[string]any) error
func (e *Emitter) EmitProgress(step string, done, total int) error

// Terminator. After either of these, further emit returns ErrStreamClosed.
func (e *Emitter) EmitResult(ok bool, data map[string]any) error
func (e *Emitter) EmitResultError(code types.ResultErrorCode, message string) error
```

**Invariants:**
- `EmitResult`는 `ok=true` 필수. `data`는 optional.
- `EmitResultError`는 내부적으로 `ok=false`, `error={code,message}` 필드 세팅.
- `event.json` schema의 if/then 불변식(ok:true → no error; ok:false → error required)을 타입 수준에서 강제.

### 3.3 `wire/decoder.go`
```go
package wire

import (
    "encoding/json"
    "fmt"

    "github.com/0xmhha/chainbench/network/internal/types"
)

// Message is a sealed interface. Implementations: EventMessage, ProgressMessage, ResultMessage.
type Message interface {
    isMessage()
}

type EventMessage    struct{ types.Event }
type ProgressMessage struct{ types.Progress }
type ResultMessage   struct{ types.Result }

func (EventMessage) isMessage()    {}
func (ProgressMessage) isMessage() {}
func (ResultMessage) isMessage()   {}

// DecodeMessage parses one NDJSON line. Dispatches on "type" field.
// Returns error for unknown type or malformed JSON.
func DecodeMessage(line []byte) (Message, error)
```

### 3.4 `wire/logger.go`
```go
package wire

import (
    "io"
    "log/slog"
    "os"
)

// SetupLogger configures slog JSON handler on stderr by default, or on a file
// path set via CHAINBENCH_NET_LOG env var. Level from CHAINBENCH_NET_LOG_LEVEL
// ({debug|info|warn|error}, default info).
func SetupLogger() *slog.Logger

// SetupLoggerTo is the testable variant with explicit writer + level.
func SetupLoggerTo(w io.Writer, level slog.Level) *slog.Logger
```

### 3.5 `events/bus.go`
```go
package events

import (
    "errors"
    "sync"
    "time"

    "github.com/0xmhha/chainbench/network/internal/types"
    "github.com/0xmhha/chainbench/network/internal/wire"
)

var ErrBusClosed = errors.New("events: bus closed")

type Event struct {
    Name types.EventName
    Data map[string]any
    TS   time.Time
}

type Bus struct {
    mu      sync.RWMutex
    subs    []chan Event
    emitter *wire.Emitter
    clock   func() time.Time
    closed  bool
}

const DefaultSubscriberBuffer = 16

func NewBus(emitter *wire.Emitter) *Bus

// Publish fans out to in-process subscribers (non-blocking; drops if channel full)
// and delegates to emitter.EmitEvent for wire output.
// Returns ErrBusClosed if Close was called.
func (b *Bus) Publish(ev Event) error

// Subscribe returns a buffered channel. Bus retains reference; Close() closes all.
func (b *Bus) Subscribe() <-chan Event

// Close closes all subscriber channels and rejects subsequent Publish.
func (b *Bus) Close() error
```

---

## 4. Data Flow

### 4.1 Producer (바이너리 내부, Sprint 2b에서 활용)
```
main() ─► wire.SetupLogger()
     ─► wire.NewEmitter(os.Stdout)
     ─► wire.DecodeCommand(os.Stdin)  (err → EmitResultError PROTOCOL_ERROR; exit 3)
     ─► events.NewBus(emitter)
     ─► handler: bus.Publish(ev) / bus.Publish(ev) ...
     ─► emitter.EmitResult(...) | emitter.EmitResultError(...)
     ─► exit {0|1|2|3}
slog → stderr (throughout)
```

### 4.2 Consumer (bash/Go/MCP, Sprint 2c에서 구현)
```
scanner := bufio.NewScanner(subprocess.Stdout)
for scanner.Scan() {
    msg, _ := wire.DecodeMessage(scanner.Bytes())
    switch m := msg.(type) {
    case wire.EventMessage:    onEvent(m.Event)
    case wire.ProgressMessage: onProgress(m.Progress)
    case wire.ResultMessage:   return m.Result          // terminator
    }
}
```

### 4.3 Internal event (in-process)
```
handler goroutine                          subscribers
    bus.Publish(ev)
     ├─ fan-out: subs[i] ← ev (non-blocking; drop if buffer full)
     └─ emitter.EmitEvent(ev.Name, ev.Data) ──► stdout
```

---

## 5. Error Mapping

| 발생 시점 | 처리 | Result error code | Exit code (2b) |
|---|---|---|---|
| `DecodeCommand` 실패 | `EmitResultError(PROTOCOL_ERROR, ...)` | `PROTOCOL_ERROR` | 3 |
| 알 수 없는 `command.command` | `EmitResultError(INVALID_ARGS, ...)` | `INVALID_ARGS` | 1 |
| Capability 없는 op | `EmitResultError(NOT_SUPPORTED, ...)` | `NOT_SUPPORTED` | 2 |
| 외부 RPC/subprocess 실패 | `EmitResultError(UPSTREAM_ERROR, ...)` | `UPSTREAM_ERROR` | 1 |
| Go panic → recover | `EmitResultError(INTERNAL, ...)` | `INTERNAL` | 1 |
| 정상 완료 | `EmitResult(true, data)` | — | 0 |

Exit code 정책은 Sprint 2b main()에서 구현. 2a는 error code 상수만 노출.

---

## 6. Testing Strategy

### 6.1 Unit tests per file
- `command_test.go`: valid/malformed envelope, unknown fields
- `emitter_test.go`: 각 Emit 메서드, terminator guard, **100 goroutine concurrent emit**, clock 주입
- `decoder_test.go`: event/progress/result/error 각 분기, unknown type, malformed, roundtrip(Emitter ↔ Decoder)
- `logger_test.go`: 레벨, env override, JSON 형식

### 6.2 Integration test
`wire_schema_test.go`: Emitter 출력 각 라인이 `network/schema.ValidateBytes("event", line)` 통과. 스키마 변경 시 wire 깨짐 → 재현성 보장.

### 6.3 events/bus_test.go
Publish 전달, wire.Emitter 호출, non-blocking drop, close 동작, concurrent publish (race detector).

### 6.4 Race detector
```
cd network && go test -race ./internal/wire/... ./internal/events/...
```
Emitter mutex + Bus RWMutex regression 방어.

### 6.5 Coverage
- `wire`: ≥ 85% line
- `events`: ≥ 80% line
- `go test -coverprofile`로 측정. 미달 시 테스트 추가.

---

## 7. Concurrency & Invariants

- `Emitter` mutex로 동시 emit 직렬화 — 라인 단위 atomic 출력 보장
- `Emitter.closed` 플래그 mutex 내부 체크 — race 없음
- `Bus.Publish`는 fan-out 시 RLock, subscriber 추가/제거는 Lock
- `Bus` subscriber 채널 buffer = 16 (DefaultSubscriberBuffer 상수)
- `clock` 주입은 모든 mutable-time 타입 (Emitter, Bus)에 적용 → 테스트 결정성

---

## 8. 보안 경계 주의 (Sprint 4 예고)

본 Sprint 2a는 서명 키를 다루지 않음. 단 향후 Sprint 4에서 signer 도입 시 `wire.Emitter`가 **key material을 절대 NDJSON에 포함하지 않음**을 Sprint 4 negative test로 검증 예정. 2a 단계에서는 `Emitter.EmitEvent(data map[string]any)` 시그니처가 `any`를 받으므로 이론상 키 주입 가능 → Sprint 4에서 redactor 훅 추가 설계.

---

## 9. 완료 기준 (Definition of Done)

1. `network/internal/wire/` 4 파일 + 5 테스트 파일
2. `network/internal/events/` 2 파일 + 1 테스트 파일
3. `cd network && go build ./... && go test -race ./... && go vet ./... && gofmt -l .` 전부 green
4. Coverage 목표 달성
5. `wire_schema_test.go` 통합 테스트 통과
6. 각 파일 독립 커밋 + 최종 push

다음 단계: `writing-plans`로 implementation plan 작성 → `subagent-driven-development`로 실행.
