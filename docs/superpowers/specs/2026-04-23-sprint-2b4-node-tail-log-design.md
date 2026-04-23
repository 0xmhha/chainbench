# Sprint 2b.4 — node.tail_log (Finite) Design

> **작성일**: 2026-04-23
> **목적**: `node.tail_log` 커맨드 구현. 지정된 노드의 로그 파일에서 최근 N 라인을 읽어 반환한다. **Finite** 모드 (streaming 없음). 구조상 LocalDriver를 사용하지 않고, pids.json의 `log_file` 경로를 직접 Go로 읽는다.

---

## 1. Goal & Scope

### 1.1 Goal
- `{"command":"node.tail_log","args":{"node_id":"node1","lines":50}}` → `{node_id, log_file, lines:[...]}` 결과
- 기본 lines=50, 최대 1000 제한
- 구독자 없음 (스트리밍 안 함), 이벤트 emit 없음, 단일 result terminator

### 1.2 Out of Scope
- Streaming / follow 모드 (별도 sprint + schema 확장 필요)
- 로그 레벨 필터링 (stdout/stderr 구분, JSON 파싱 등)
- 원격 로그 tail (remote driver 도입 후)
- `chainbench.sh node log` bash 재사용 (path 계산 버그로 패스)

### 1.3 Approach
**LocalDriver 미사용**. 다음이 이유:
- `tail -n 50 <file>` 는 subprocess 없이 Go에서 구현 가능 (bufio + ring buffer, O(n) 메모리)
- `chainbench.sh node log` 는 `${CHAINBENCH_DIR}/data/logs/<label>.log` 경로를 계산하는 **버그** 보유 — pids.json의 `log_file` 필드를 무시함
- 직접 파일 읽기가 정확성·단순성·테스트 용이성 모두 우수

### 1.4 Success Criteria
- `cd network && go test -race ./...` green
- Coverage 유지 (`state` ≥85%, `cmd/chainbench-net` ≥80%)
- E2E: 임시 로그 파일 + fixture로 전체 파이프라인 검증

---

## 2. Package Structure

```
network/
├── internal/
│   └── state/
│       ├── tail.go                  # 신규 — TailFile(path, n) helper
│       ├── tail_test.go
│       └── network.go               # 수정: ProviderMeta 에 "log_file" 추가
└── cmd/chainbench-net/
    ├── handlers.go                  # + handleNodeTailLog + 등록
    ├── handlers_test.go             # + 6 tests
    └── e2e_test.go                  # + 1 E2E
```

LocalDriver 패키지 변경 없음.

---

## 3. Type Contracts

### 3.1 `state/tail.go`
```go
// TailFile returns the last n lines of the file at path (n >= 1).
// Lines preserve their original content (no trailing newline).
// If the file has fewer than n lines, all lines are returned.
// Uses a ring buffer — O(file-size) read time, O(n * avg_line_len) memory.
func TailFile(path string, n int) ([]string, error)
```

### 3.2 `state/network.go` 수정
```go
// In buildNodes, ProviderMeta gets an extra "log_file" key:
node := types.Node{
    // ... existing fields ...
    ProviderMeta: types.NodeProviderMeta{
        "pid_key":  id,
        "log_file": info.LogFile,   // NEW
    },
}
```

Schema 변경 없음 (`NodeProviderMeta`는 `map[string]interface{}`).

### 3.3 `cmd/chainbench-net/handlers.go` 추가
```go
// newHandleNodeTailLog returns a Handler for node.tail_log.
// Args: { "node_id": "nodeN", "lines": 50 (optional, default 50, max 1000) }.
// Returns: { node_id, log_file, lines: ["...", "..."] }.
func newHandleNodeTailLog(stateDir string) Handler
```

`allHandlers` 확장:
```go
"node.tail_log": newHandleNodeTailLog(stateDir),
```

(chainbenchDir 불필요 — subprocess 안 씀)

---

## 4. Data Flow

```
{"command":"node.tail_log","args":{"node_id":"node1","lines":50}}
  → handleNodeTailLog:
    1. Parse args → (node_id, lines)
    2. Validate: node_id exists, lines in [1, 1000] (default 50)
    3. state.LoadActive(stateDir) — find node
    4. Read log_file path from node.ProviderMeta["log_file"]
       └ missing → UPSTREAM_ERROR "log_file unknown for node ..."
    5. state.TailFile(logPath, lines)
       └ file not found → UPSTREAM_ERROR "log file not accessible: ..."
       └ read error → UPSTREAM_ERROR
    6. Return {node_id, log_file, lines: [...]}
       (단일 result terminator, 이벤트 emit 없음)
```

---

## 5. Error Mapping

| 상황 | code | exit |
|---|---|---|
| args 파싱/missing node_id/bad prefix | INVALID_ARGS | 1 |
| node_id not found | INVALID_ARGS | 1 |
| lines < 1 | INVALID_ARGS | 1 |
| lines > 1000 | INVALID_ARGS | 1 |
| state.LoadActive 실패 | UPSTREAM_ERROR | 1 |
| provider_meta에 log_file 키 없음 | UPSTREAM_ERROR | 1 |
| log file open 실패 | UPSTREAM_ERROR | 1 |
| read error 중간 | UPSTREAM_ERROR | 1 |
| 정상 | — | 0 |

---

## 6. Testing Strategy

### 6.1 `state/tail_test.go`
- `TestTailFile_ReturnsLastNLines` — 5 line file, tail 3 → last 3
- `TestTailFile_FewerLinesThanN` — 3 line file, tail 10 → all 3
- `TestTailFile_EmptyFile` → `[]` (nil or len 0)
- `TestTailFile_MissingFile` → error
- `TestTailFile_NEqualOne` → last line only
- `TestTailFile_LargeLines` — 1MB single line이 scanner 버퍼 안 넘음 (이미 Sprint 2a wire에서 1MB 버퍼 설정 있음, 패턴 재사용)

### 6.2 `state/network_test.go` 추가/수정
기존 `TestLoadActive_HappyPath` 에 ProviderMeta의 `log_file` 키 확인 추가:
```go
if meta := net.Nodes[0].ProviderMeta; meta["log_file"] != "/tmp/node-data/logs/node1.log" {
    t.Errorf("log_file: got %v", meta["log_file"])
}
```

### 6.3 `cmd/chainbench-net/handlers_test.go` 추가
- `TestHandleNodeTailLog_HappyPath` — tempfile에 10 라인 쓰고 lines=3 → 마지막 3 라인 반환
- `TestHandleNodeTailLog_MissingNodeID` → INVALID_ARGS
- `TestHandleNodeTailLog_UnknownNodeID` → INVALID_ARGS
- `TestHandleNodeTailLog_InvalidLines_Negative` (-1) → INVALID_ARGS
- `TestHandleNodeTailLog_InvalidLines_OverMax` (1001) → INVALID_ARGS
- `TestHandleNodeTailLog_DefaultLines_50` (lines 생략) → 최대 50 라인
- `TestHandleNodeTailLog_LogFileMissing` (log_file 경로에 실제 파일 없음) → UPSTREAM_ERROR
- `TestAllHandlers_IncludesNodeTailLog`

### 6.4 E2E
- `TestE2E_NodeTailLog_ViaRootCommand`:
  - tempdir에 pids.json (log_file 경로를 tempdir 내부로 rewrite), profile, 실제 로그 파일 셋업
  - stdin: `{"command":"node.tail_log","args":{"node_id":"node1","lines":5}}`
  - stdout 파싱, `data.lines` 배열 확인
  - schema.ValidateBytes("event", line)

### 6.5 Race detector
```
go test -race ./internal/state/... ./cmd/chainbench-net/...
```

---

## 7. Security Boundary

- `log_file` 경로는 pids.json에서 읽음 — 사용자 직접 입력 아님, directory traversal 여지 제한적
- 핸들러는 경로를 그대로 `os.Open` — 심볼릭 링크 체크 등 추가 방어 안 함 (MVP)
- 반환하는 `lines` 배열은 로그 raw 내용 — 노드가 잘못 로그에 기록한 시크릿이 있다면 여기서 노출될 수 있음 (Sprint 4 signer 도입 시 재검토)

---

## 8. 완료 기준 (DoD)

1. `internal/state/tail.go` + tests (TailFile helper)
2. `internal/state/network.go` 수정: `ProviderMeta["log_file"]` 포함
3. `cmd/chainbench-net/handlers.go` — `handleNodeTailLog` + `allHandlers` 등록
4. `cmd/chainbench-net/handlers_test.go` — 8 new tests
5. `cmd/chainbench-net/e2e_test.go` — 1 E2E
6. 모든 gate green, coverage 유지
7. 커밋 메시지 `network: ...` 프리픽스, 영어, co-author 없음
