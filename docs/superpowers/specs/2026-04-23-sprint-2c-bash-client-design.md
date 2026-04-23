# Sprint 2c — bash Client (`lib/network_client.sh`) Design

> **작성일**: 2026-04-23
> **목적**: bash에서 `chainbench-net`을 호출·파싱할 수 있는 라이브러리를 제공. bash CLI·test harness·MCP server 모두 동일 진입점 사용. Go 구현과 소비자 사이의 안정 인터페이스.

---

## 1. Goal & Scope

### 1.1 Goal
bash 함수 `cb_net_call <command> <args_json>` 하나로 `chainbench-net run` 의 전체 파이프라인을 쉽게 사용:
- stdin envelope 생성
- subprocess spawn
- stdout NDJSON 파싱 (result terminator 탐지)
- 성공 시 data JSON을 stdout으로 echo + exit 0
- 실패 시 stderr에 error code/message + exit 1

### 1.2 Out of Scope
- 기존 `chainbench node stop|start|restart` bash 명령 대체 (Sprint 3+에서 결정)
- 이벤트 스트림 callback (현 MVP는 result만; event는 stderr로 흘림)
- WebSocket / long-running subscription
- binary 자동 설치 (`install.sh` 확장은 별도)
- MCP server 연동 (후속 sprint)

### 1.3 Success Criteria
- `lib/network_client.sh` 소스 가능, `cb_net_call "network.load" '{"name":"local"}'` 동작
- `tests/unit/tests/network-wire-protocol.sh` 통과
- `tests/unit/run.sh` 전체 테스트 suite green

---

## 2. Package Structure

```
chainbench/
├── lib/
│   └── network_client.sh         # 신규 — bash client library
└── tests/unit/
    └── tests/
        └── network-wire-protocol.sh  # 신규 — E2E bash test
```

**기존 코드 수정 없음**. 순수 additive.

---

## 3. Library API (`lib/network_client.sh`)

### 3.1 Public functions

```bash
# cb_net_call <command> [args_json]
# Invokes chainbench-net with the given wire command.
#
# Returns:
#   0 — success. data JSON printed to stdout.
#   1 — APIError. "<code>: <message>" printed to stderr.
#   2 — spawn failure (binary not found, etc.). error printed to stderr.
#
# Examples:
#   data=$(cb_net_call "network.load" '{"name":"local"}') || exit $?
#   cb_net_call "node.stop" '{"node_id":"node1"}'
cb_net_call()
```

### 3.2 Internal helpers

```bash
# _cb_net_binary
# Resolves the chainbench-net binary path. Tries, in order:
#   1. $CHAINBENCH_NET_BIN env var
#   2. $CHAINBENCH_DIR/bin/chainbench-net
#   3. $CHAINBENCH_DIR/network/bin/chainbench-net
#   4. `command -v chainbench-net` (PATH)
# Prints the resolved path or returns 1 if not found.
_cb_net_binary()

# _cb_net_spawn <envelope_json>
# Pipes envelope to `chainbench-net run` stdin and captures stdout.
# Prints raw NDJSON stdout to the caller's stdout.
# Returns the subprocess exit code (0 / 1 / 2 / 3 per Go side).
_cb_net_spawn()

# _cb_net_parse_result <ndjson_stream>
# Reads NDJSON from stdin, finds the `type=result` terminator, and emits:
#   On ok=true:  prints data JSON to stdout, returns 0
#   On ok=false: prints "<code>: <message>" to stderr, returns 1
#   Missing terminator: prints diagnostic to stderr, returns 2
_cb_net_parse_result()
```

### 3.3 Dependencies

- `bash` 4.0+ (existing repo convention)
- `jq` (JSON parsing)
- `chainbench-net` binary (built via `cd network && go build -o bin/chainbench-net ./cmd/chainbench-net`)

Guard: library 소스 시점에 `command -v jq` 검증, 없으면 source 자체가 경고 출력.

---

## 4. Data Flow

### 4.1 Happy path
```
caller:   cb_net_call "network.load" '{"name":"local"}'
            │
  envelope: {"command":"network.load","args":{"name":"local"}}   ← jq로 생성
            │
  _cb_net_spawn pipes envelope to stdin of chainbench-net run
            │
  stdout captured as NDJSON stream (may include events + terminator)
            │
  _cb_net_parse_result reads stream, finds last type=result
    ok=true → echo .data JSON → return 0
    ok=false → stderr error → return 1
```

### 4.2 Error paths

| 상황 | 결과 |
|---|---|
| `jq` 없음 | source 시점에 stderr 경고, cb_net_call 호출 시 return 2 |
| chainbench-net 바이너리 없음 | stderr "chainbench-net not found", return 2 |
| chainbench-net이 비정상 종료 (신호 등) | stderr + subprocess exit code 전달 |
| NDJSON 파싱 실패 / terminator 없음 | stderr "no result terminator in stream", return 2 |
| APIError result | stderr "<code>: <message>", return 1 |
| 정상 | data JSON stdout, return 0 |

---

## 5. Testing Strategy

### 5.1 Unit test — `tests/unit/tests/network-wire-protocol.sh`

기존 harness 패턴 (`set -euo pipefail`, `source tests/unit/lib/assert.sh`, subshell execution) 따름.

**Scenarios**:

1. **Missing binary** — `CHAINBENCH_NET_BIN=/nonexistent` → `cb_net_call` returns 2, stderr 메시지 확인
2. **Happy path — network.load** — 실제 빌드된 binary + fixture state dir 사용
   - `tempdir` 생성 → `pids.json` + `current-profile.yaml` fixture 복사
   - `CHAINBENCH_STATE_DIR=$tempdir`, `CHAINBENCH_NET_BIN=<built binary>`
   - `data=$(cb_net_call "network.load" '{"name":"local"}')`
   - `assert_contains "$data" '"name":"local"'`
   - `assert_contains "$data" '"chain_type":"stablenet"'`
3. **API error — wrong name** — `cb_net_call "network.load" '{"name":"bogus"}'` → return 1, stderr `INVALID_ARGS: ...`
4. **Missing jq guard** (optional) — PATH 조작으로 jq 숨기고 source 경고 확인

### 5.2 Binary build step

Test가 자체적으로 binary를 빌드해야 함 (CI 친화적):
```bash
build_binary() {
    local out="${TMPDIR}/chainbench-net-test"
    ( cd "${CHAINBENCH_DIR}/network" && go build -o "$out" ./cmd/chainbench-net )
    echo "$out"
}
```

Test 시작 시 한 번 빌드, `$CHAINBENCH_NET_BIN` 설정.

### 5.3 Test isolation

- `TMPDIR` 사용 (`mktemp -d`) — state fixture, binary 모두 여기에
- `trap 'rm -rf "$TMPDIR"' EXIT`
- PATH는 건드리지 않음 (binary는 env로 주입)

### 5.4 Coverage target
- Library: 4~6 핵심 시나리오 (각 return code path 포함)
- 코드 line coverage 측정은 bash에서 실용적이지 않으므로 시나리오 커버리지로 대체

---

## 6. Security / Boundary

- `args_json` 은 사용자 입력 그대로 subprocess stdin으로 들어감 — shell injection 불가 (stdin이지 argv가 아님)
- Envelope 생성은 `jq -cn` 사용 (shell interpolation 경유 X) — JSON escaping 자동
- Binary path resolution에서 untrusted PATH 환경변수 검증 안 함 (운영자 책임)
- stderr에 error message 출력 — 로그 시스템에 섞일 수 있음. 로그 파일 설계는 Sprint 4 signer 경계와 묶어서

---

## 7. 완료 기준 (DoD)

1. `lib/network_client.sh` — `cb_net_call` + 3 internal helpers
2. `tests/unit/tests/network-wire-protocol.sh` — 3~4 시나리오, `tests/unit/run.sh`에서 PASS
3. 빌드 자동화 (테스트가 binary를 자체 빌드)
4. `network/README.md` — "Bash client usage" 섹션 추가 (1~2 예시 코드)
5. 커밋 메시지 `network: ...` 또는 `lib: ...`, 영어, co-author 없음

다음 단계: `writing-plans`로 implementation plan → `subagent-driven-development`로 실행.
