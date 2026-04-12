# Design: LLM Test Automation Integration

**Date:** 2026-04-12
**Status:** Draft
**Scope:** chainbench (bash CLI + MCP server) — LLM Integration Analysis Tier 1~2 구현

---

## 1. Background

### 1.1 현재 상태

chainbench에는 113개의 regression 테스트가 7개 카테고리(a~g)로 존재하며, go-stablenet의 `build/bin/gstable` 바이너리를 대상으로 실행된다.

**현재 동작하는 것:**
- CLI: `chainbench test run regression/a-ethereum/a2-01-legacy-tx`
- MCP: `chainbench_test_run`, `chainbench_test_list`, `chainbench_report`
- Assertion: `assert_eq`, `assert_gt`, `assert_ge`, `assert_true`, `assert_contains`, `assert_not_empty`
- RPC: `rpc <node> <method> [params]`, domain helpers (`get_base_fee`, `send_raw_tx`, `wait_tx_receipt_full` 등)
- 결과 JSON: `state/results/<name>_<ts>.json` (test, status, pass, fail, total, duration, failures[])

**현재 동작하지 않는 것 (LLM Integration Analysis A~L 전부 미구현):**
- 테스트 메타데이터 파싱 (A)
- 관찰값 수집 (B)
- 실패 진단 자동 캡처 (C)
- 기계 가독 출력 (D)
- dry-run (G)
- 압축 상태 (H)

### 1.2 go-stablenet 프로젝트 현황 (fact)

| 항목 | 상태 |
|------|------|
| `build/bin/gstable` | ✅ 존재 (42MB, 2026-04-12 빌드) |
| `cmd/logrot/main.go` | ❌ 없음 (go-stablenet에는 logrot cmd 없음) |
| 회귀 테스트 스펙 | ✅ `stablenet-test-case/regression-test-spec.md` (116 TC, Gherkin) |
| 하드포크 테스트 스펙 | 🔄 `stablenet-test-case/hardfork-test-spec.md` (별도 세션 검토 중) |
| `.mcp.json` | ❌ 없음 (chainbench mcp enable 미실행) |
| 빌드 명령 | `make gstable` → `build/bin/gstable` |
| Go 모듈 | `github.com/ethereum/go-ethereum` (go 1.23.0+) |
| 테스트 실행 | `make test` (160+ Go 패키지) |

### 1.3 현재 테스트 스크립트 헤더 패턴

```bash
#!/usr/bin/env bash
# Test: regression/a-ethereum/a2-01-legacy-tx
# RT-A-2-01 — Legacy Tx (type 0x0) 발행
# [v2 보강] effectiveGasPrice + gasLimit valid check (21000)
set -euo pipefail

source "$(dirname "$0")/../lib/common.sh"
test_start "regression/a-ethereum/a2-01-legacy-tx"
check_env || { test_result; exit 1; }
```

이미 `# RT-<ID>` 형식의 ID가 주석으로 존재하지만, 파싱 불가능한 자유형 텍스트.

---

## 2. Goals

1. **Tier 1-A**: 기존 113개 테스트의 메타데이터를 파싱 가능한 YAML-in-comment 프론트매터로 승격
2. **Tier 1-B**: `observe()` API로 테스트 실행 중 관찰값을 수집하고 결과 JSON에 포함
3. **Tier 1-C**: 실패 시 전 노드 상태를 자동 캡처하여 `state/failures/`에 저장
4. **Tier 1-D**: `--format jsonl` 옵션으로 NDJSON 이벤트 스트림 출력
5. **Tier 2-G**: `--dry-run` 옵션으로 실행 계획 미리보기
6. **Tier 2-H**: `chainbench_state_compact` MCP tool로 최소 context 상태 조회

## 3. Non-Goals

- ~~Tier 2-E (스펙 연결): go-stablenet에 스펙 문서 부재~~ → **해제**: 스펙 문서가 `stablenet-test-case/regression-test-spec.md` (116 TC, Gherkin 형식)로 존재 확인. `hardfork-test-spec.md`는 별도 세션에서 검토 중.
- Tier 2-F (고수준 assertion helper): `tests/regression/lib/common.sh`에 이미 `assert_receipt_status`, `assert_error_contains`, `gov_full_flow` 등 도메인 헬퍼 존재. 별도 `assert_chain.sh` 불필요
- Tier 3-I~L: 실사용 피드백 후 결정
- 기존 113개 테스트 스크립트의 로직 변경 (메타데이터 추가만 수행)
- go-stablenet 코드 변경

---

## 4. Architecture

### 4.1 Observables 데이터 흐름

```
테스트 스크립트 (.sh)
  │
  ├─ observe "bp_head" "$block_num"       # key-value 수집
  ├─ assert_eq "$actual" "$expected" ...   # 기존 assertion
  └─ test_result                           # 결과 직렬화
         │
         ▼
  state/results/<name>_<ts>.json
  {
    "test": "regression/a-ethereum/a2-01-legacy-tx",
    "status": "passed",
    "pass": 5, "fail": 0, "total": 5,
    "duration": 12,
    "observed": {                           ← NEW
      "base_fee": "20000000000000",
      "tx_hash": "0xabc...",
      "gas_used": "21000"
    },
    "failures": []
  }
```

### 4.2 실패 컨텍스트 캡처 흐름

```
test_result (fail > 0 감지)
  │
  └─ _cb_capture_failure_context()
       │
       ├─ rpc <node> eth_blockNumber     (전 노드)
       ├─ rpc <node> net_peerCount       (전 노드)
       ├─ rpc <node> eth_syncing         (전 노드)
       ├─ 최근 5 블록 hash/stateRoot     (node 1)
       └─ tail -200 <node_log>           (전 노드)
            │
            ▼
       state/failures/<test_safe_name>_<ts>/
       ├── context.json
       └── node<N>.log.tail  (노드별)
```

### 4.3 JSONL 이벤트 스트림

`CB_FORMAT=jsonl` 환경변수가 설정된 상태에서 `_assert_pass`/`_assert_fail`/`test_start`/`test_result`/`observe`가 각각 한 줄씩 JSONL을 stdout으로 출력.

```jsonl
{"event":"test_start","name":"regression/a-ethereum/a2-01-legacy-tx","ts":"2026-04-12T12:30:00Z"}
{"event":"observe","key":"base_fee","value":"20000000000000"}
{"event":"assert_pass","msg":"legacy tx hash returned"}
{"event":"observe","key":"tx_hash","value":"0xabc..."}
{"event":"assert_pass","msg":"receipt.status == 0x1"}
{"event":"test_end","name":"regression/a-ethereum/a2-01-legacy-tx","status":"passed","duration":12,"pass":5,"fail":0}
```

기존 text 출력 (stderr)은 변경 없음. JSONL은 stdout으로 분리.

### 4.4 프론트매터 스키마

```bash
#!/usr/bin/env bash
# ---chainbench-meta---
# id: RT-A-2-01
# name: Legacy Tx (type 0x0)
# category: regression/a-ethereum
# tags: [tx, legacy, type0]
# estimated_seconds: 15
# preconditions:
#   chain_running: true
#   python_packages: [eth-account, requests, eth-utils]
# depends_on: []
# ---end-meta---
set -euo pipefail
```

**파서**: `lib/test_meta.sh` — `awk`/`sed`로 `---chainbench-meta---` ~ `---end-meta---` 구간 추출 후 `# ` prefix 제거 → Python `yaml.safe_load`.

**호환성**: 프론트매터가 없는 스크립트는 빈 메타데이터로 동작 (기존 기능 유지).

### 4.5 Dry-run 모드

```bash
chainbench test run regression/a-ethereum --dry-run --format json
```

```json
{
  "target": "regression/a-ethereum",
  "scripts": [
    {
      "script": "a1-01-genesis-init.sh",
      "meta": { "id": "RT-A-1-01", "tags": ["genesis"], "estimated_seconds": 5, "depends_on": [] }
    },
    {
      "script": "a2-01-legacy-tx.sh",
      "meta": { "id": "RT-A-2-01", "tags": ["tx", "legacy"], "estimated_seconds": 15, "depends_on": [] }
    }
  ],
  "total_scripts": 30,
  "total_estimated_seconds": 450
}
```

### 4.6 Compact State MCP Tool

```json
// chainbench_state_compact 응답 (< 300 bytes)
{
  "running": true,
  "profile": "regression",
  "nodes": {
    "1": {"block": 1234, "peers": 4, "role": "bp"},
    "5": {"block": 1234, "peers": 4, "role": "en"}
  },
  "consensus": "ok",
  "last_test": {"name": "a2-01-legacy-tx", "status": "passed", "ts": "2026-04-12T12:30:00Z"}
}
```

---

## 5. Detailed Component Changes

### 5.1 `tests/lib/assert.sh` 확장

**새로운 함수:**

```bash
# observe <key> <value>
# 테스트 실행 중 관찰값을 수집. test_result에서 JSON으로 직렬화.
observe() {
  local key="$1" value="$2"
  _OBSERVED_KEYS+=("$key")
  _OBSERVED_VALUES+=("$value")
  if [[ "${CB_FORMAT:-text}" == "jsonl" ]]; then
    printf '{"event":"observe","key":"%s","value":"%s"}\n' "$key" "$value"
  fi
}
```

**변경되는 함수:**

- `test_start`: `_OBSERVED_KEYS=()`, `_OBSERVED_VALUES=()` 초기화 추가. JSONL 모드 시 `test_start` 이벤트 출력.
- `_assert_pass`/`_assert_fail`: JSONL 모드 시 이벤트 출력 추가 (기존 stderr 출력은 유지).
- `test_result`:
  - `observed` 필드를 결과 JSON에 포함
  - fail > 0 이면 `_cb_capture_failure_context` 호출
  - JSONL 모드 시 `test_end` 이벤트 출력

**새로운 내부 상태:**

```bash
_OBSERVED_KEYS=()
_OBSERVED_VALUES=()
```

### 5.2 `lib/test_meta.sh` (신규)

**함수:**

```bash
# cb_parse_meta <script_path>
# 스크립트의 YAML-in-comment 프론트매터를 파싱하여 JSON stdout 출력.
# 프론트매터가 없으면 빈 JSON 객체 {} 출력.
cb_parse_meta() {
  local script="$1"
  local yaml_block
  yaml_block=$(awk '/^# ---chainbench-meta---$/,/^# ---end-meta---$/' "$script" \
    | grep -v '# ---.*meta---' \
    | sed 's/^# //')

  if [[ -z "$yaml_block" ]]; then
    echo "{}"
    return 0
  fi

  printf '%s' "$yaml_block" | python3 -c "
import sys, json
try:
    import yaml
    data = yaml.safe_load(sys.stdin.read()) or {}
except ImportError:
    data = {}
print(json.dumps(data))
"
}
```

### 5.3 `tests/lib/failure_context.sh` (신규)

```bash
# _cb_capture_failure_context <test_name>
# 실패 시 전 노드의 체인 상태와 로그를 자동 수집.
_cb_capture_failure_context() {
  local test_name="$1"
  local safe_name ts ctx_dir
  safe_name=$(printf '%s' "$test_name" | tr -cs '[:alnum:]-_' '_')
  ts=$(date +%Y%m%d_%H%M%S)
  ctx_dir="${CHAINBENCH_DIR}/state/failures/${safe_name}_${ts}"
  mkdir -p "$ctx_dir"

  # 노드 수 확인 (pids.json 기반)
  local pids_file="${CHAINBENCH_DIR}/state/pids.json"
  [[ ! -f "$pids_file" ]] && return 0

  python3 - "$pids_file" "$ctx_dir" <<'PYEOF'
import sys, json, subprocess, os

pids_file = sys.argv[1]
ctx_dir = sys.argv[2]

with open(pids_file) as f:
    data = json.load(f)

nodes = data.get("nodes", {})
context = {"nodes": {}, "recent_blocks": []}

for key, node in nodes.items():
    http_port = node.get("http_port")
    log_file = node.get("log_file", "")
    node_ctx = {"port": http_port, "type": node.get("type", "")}

    if http_port:
        url = f"http://127.0.0.1:{http_port}"
        for method in ["eth_blockNumber", "net_peerCount", "eth_syncing"]:
            try:
                r = subprocess.run(
                    ["curl", "-s", "--max-time", "3", "-X", "POST",
                     "-H", "Content-Type: application/json",
                     "--data", json.dumps({"jsonrpc":"2.0","method":method,"params":[],"id":1}),
                     url],
                    capture_output=True, text=True, timeout=5
                )
                resp = json.loads(r.stdout) if r.stdout else {}
                node_ctx[method] = resp.get("result", "error")
            except Exception:
                node_ctx[method] = "unreachable"

    # 로그 tail
    if log_file and os.path.isfile(log_file):
        tail_path = os.path.join(ctx_dir, f"node{key}.log.tail")
        try:
            subprocess.run(["tail", "-200", log_file], stdout=open(tail_path, "w"), timeout=5)
        except Exception:
            pass

    context["nodes"][key] = node_ctx

# 최근 5 블록 (node 1 기준)
first_node = next(iter(nodes.values()), {})
first_port = first_node.get("http_port")
if first_port:
    try:
        url = f"http://127.0.0.1:{first_port}"
        r = subprocess.run(
            ["curl", "-s", "--max-time", "3", "-X", "POST",
             "-H", "Content-Type: application/json",
             "--data", json.dumps({"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}),
             url],
            capture_output=True, text=True, timeout=5
        )
        head = int(json.loads(r.stdout).get("result", "0x0"), 16)
        for i in range(max(0, head - 4), head + 1):
            hex_num = hex(i)
            r2 = subprocess.run(
                ["curl", "-s", "--max-time", "3", "-X", "POST",
                 "-H", "Content-Type: application/json",
                 "--data", json.dumps({"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":[hex_num, False],"id":1}),
                 url],
                capture_output=True, text=True, timeout=5
            )
            blk = json.loads(r2.stdout).get("result", {})
            context["recent_blocks"].append({
                "number": i,
                "hash": blk.get("hash", ""),
                "stateRoot": blk.get("stateRoot", ""),
                "miner": blk.get("miner", "")
            })
    except Exception:
        pass

with open(os.path.join(ctx_dir, "context.json"), "w") as f:
    json.dump(context, f, indent=2)
PYEOF

  printf '[FAIL-CTX] Failure context saved to %s\n' "$ctx_dir" >&2
}
```

### 5.4 `lib/cmd_test.sh` 확장

**변경사항:**

1. `_cb_test_run_single`: `CB_FORMAT` 환경변수를 자식 프로세스에 전달
2. `cmd_test_main`: `--format jsonl`, `--dry-run`, `--tag`, `--match` 옵션 파싱 추가
3. `_cb_test_dry_run`: 신규 함수, 메타데이터 파서로 실행 계획 JSON 생성
4. `_cb_test_list` 확장: `--format json` 시 각 테스트에 메타 정보 포함

### 5.5 MCP 확장

**`mcp-server/src/tools/test.ts` 변경:**
- `chainbench_test_run`: `format` 파라미터 추가 (`"text"` | `"jsonl"`)
- `chainbench_test_list`: 응답에 메타데이터 포함

**`mcp-server/src/tools/status.ts` 변경 (또는 lifecycle.ts 내):**
- `chainbench_state_compact`: 신규 tool, `chainbench status --compact --json` 호출

**`mcp-server/src/tools/test.ts` 신규 tool:**
- `chainbench_failure_context`: `state/failures/` 최근 디렉토리의 `context.json` 반환

### 5.6 `.gitignore` 추가

```
state/failures/
```

---

## 6. go-stablenet 호환성

| 항목 | chainbench 측 | go-stablenet 측 | 호환성 |
|------|--------------|----------------|--------|
| 바이너리 위치 | `resolve_binary` → `build/bin/gstable` | `make gstable` → `build/bin/gstable` | ✅ 일치 |
| logrot | `resolve_logrot` → fallback to plain `>>` | `cmd/logrot/` 없음 | ✅ graceful fallback |
| 테스트 계정 | `tests/regression/lib/common.sh` 하드코딩 | N/A (chainbench 자체 관리) | ✅ 독립 |
| RPC 인터페이스 | `eth_*`, `net_*`, `istanbul_*` | gstable JSON-RPC | ✅ 호환 |
| 시스템 컨트랙트 | `common.sh`에 주소 하드코딩 | 제네시스에 포함 | ✅ 일치 |
| Python 의존성 | `eth-account`, `eth-utils`, `eth-abi` | N/A | ✅ chainbench만 필요 |

**스펙 연결 (Tier 2-E)**: 스펙 문서가 `stablenet-test-case/regression-test-spec.md`에 존재함을 확인 (116 TC, Gherkin 형식, ID 체계 호환). 구현 가능 상태로 전환됨.

---

## 7. Error Handling

### 7.1 Observables

| 상황 | 동작 |
|------|------|
| `observe` 호출 시 value 빈 문자열 | 그대로 저장 (`""`) |
| 결과 JSON 직렬화 실패 | `test_result`의 기존 Python 블록에 `observed` 필드만 추가하므로 Python 예외 시 fallback (빈 dict) |

### 7.2 실패 컨텍스트 캡처

| 상황 | 동작 |
|------|------|
| 체인 미실행 (pids.json 없음) | 캡처 skip, 경고 없음 |
| 특정 노드 unreachable | 해당 노드 RPC 결과를 `"unreachable"`로 기록, 나머지 계속 |
| 로그 파일 없음 | tail skip |
| `state/failures/` 쓰기 실패 | stderr 경고, 테스트 결과에는 영향 없음 |

### 7.3 JSONL 모드

| 상황 | 동작 |
|------|------|
| `CB_FORMAT` 미설정 | 기존 text 모드 (변경 없음) |
| JSONL stdout + 기존 stderr | 분리 출력 — LLM은 stdout만 파싱 |

### 7.4 프론트매터 파서

| 상황 | 동작 |
|------|------|
| 프론트매터 없는 스크립트 | `{}` 반환 (기존 기능 유지) |
| 잘못된 YAML | Python 예외 → `{}` fallback |
| PyYAML 미설치 | `{}` fallback |

---

## 8. Testing

### 8.1 추가 단위 테스트

| 파일 | 내용 |
|------|------|
| `tests/unit/tests/assert-observe.sh` | `observe` 함수 + 결과 JSON `observed` 필드 검증 |
| `tests/unit/tests/assert-jsonl.sh` | JSONL 이벤트 포맷 검증 |
| `tests/unit/tests/test-meta-parse.sh` | 프론트매터 파서 검증 (있는 경우, 없는 경우, 잘못된 YAML) |
| `tests/unit/tests/cmd-test-dryrun.sh` | dry-run JSON 출력 검증 |

### 8.2 통합 검증

기존 `bash tests/unit/run.sh` 실행 시 모든 테스트 통과 확인.
기존 113개 regression 테스트 중 대표 3개(`a1-01`, `a2-01`, `f1-01`) 실행하여 `observed` 필드가 결과에 포함되는지 확인.

---

## 9. Implementation Phases

| Phase | 커밋 메시지 | 범위 | 의존성 |
|-------|-----------|------|--------|
| 1 | `feat(test): add observe() API and observables to result JSON` | `tests/lib/assert.sh` 확장 + 단위 테스트 | 없음 |
| 2 | `feat(test): add failure context auto-capture on test failure` | `tests/lib/failure_context.sh` + `test_result` 훅 + `.gitignore` | Phase 1 |
| 3 | `feat(test): add JSONL event stream output mode` | `assert.sh` JSONL 분기 + `cmd_test.sh` `--format` 옵션 + 단위 테스트 | Phase 1 |
| 4 | `feat(test): add YAML frontmatter parser for test metadata` | `lib/test_meta.sh` + `cmd_test.sh` list 확장 + 단위 테스트 | 없음 |
| 5 | `feat(test): add dry-run mode for test execution planning` | `cmd_test.sh` `--dry-run` + 단위 테스트 | Phase 4 |
| 6 | `feat(mcp): add state_compact and failure_context tools` | MCP tool 2개 + TypeScript 빌드 | Phase 2 |
| 7 | `chore(test): add frontmatter to regression test scripts` | 113개 스크립트에 `---chainbench-meta---` 블록 추가 | Phase 4 |

Phase 1~6은 순차 구현. Phase 7은 Phase 4 완료 후 일괄 작업.

---

## 10. LLM 테스트 자동화 워크플로우

### 10.1 LLM이 go-stablenet 코드를 변경한 후 검증하는 흐름

```
LLM이 go-stablenet 코드 수정
    │
    ▼
make gstable                                    # 바이너리 빌드
    │
    ▼
chainbench restart --binary-path $(pwd)/build/bin/gstable
    │
    ▼
chainbench test run regression/a-ethereum --format jsonl
    │
    ├─ stdout: JSONL 이벤트 스트림 (LLM이 파싱)
    ├─ state/results/*.json (observed 포함)
    └─ [fail 시] state/failures/*/context.json
    │
    ▼
LLM이 결과 분석
    ├─ PASS → 완료
    └─ FAIL → chainbench_failure_context 호출
              → 노드 블록높이, peer 수, 로그 tail 확인
              → 코드 수정 → 반복
```

### 10.2 MCP 기반 대화형 흐름

```
사용자: "consensus 로직 변경 후 regression 돌려줘"

LLM:
  1. chainbench_state_compact         → 체인 상태 확인 (< 300 bytes)
  2. chainbench_test_run              → regression/b-wbft --format jsonl
  3. [결과 분석]
  4. chainbench_failure_context       → 실패 시 진단 정보 1회 호출
  5. 사용자에게 분석 결과 보고
```

### 10.3 Playwright 패턴과의 대응

| Frontend (Playwright) | Blockchain (chainbench) |
|----------------------|------------------------|
| `page.goto(url)` | `chainbench init && start` |
| `page.click(selector)` | `send_raw_tx(...)` |
| `expect(locator).toHaveText(...)` | `assert_eq "$status" "0x1"` |
| `page.screenshot()` on failure | `_cb_capture_failure_context` on failure |
| `console.log` capture | `observe "key" "value"` |
| JSON test report | `state/results/*.json` + `observed` |
| `--reporter=json` | `--format jsonl` |

---

## 11. Open Questions

1. **프론트매터 일괄 추가 범위**: 113개 전부 vs 카테고리 a만 우선?
   - 권장: a 카테고리(30개) 우선 추가, 나머지는 피드백 후 일괄 적용
2. **observed 키 표준화**: 자유 key-value vs 권장 키 리스트?
   - 권장: 자유 key-value. 다만 가이드 문서에 일반적인 키 (`block_number`, `tx_hash`, `gas_used`, `base_fee`, `peer_count`) 제시
3. **failure context 수집 범위**: 전 노드 vs 테스트에서 사용한 노드만?
   - 권장: 전 노드 (노드 수 ≤ 10이므로 비용 낮음)
4. **JSONL과 text 출력 동시 사용**: stdout (JSONL) + stderr (text) 분리가 모든 환경에서 동작하는가?
   - `bash -c 'script' 2>/dev/null` 패턴으로 MCP에서 JSONL만 캡처 가능
