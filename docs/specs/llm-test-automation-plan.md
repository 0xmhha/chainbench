# Implementation Plan: LLM Test Automation Integration

**Date:** 2026-04-12
**Design Spec:** `docs/specs/llm-test-automation-design.md`
**Prerequisite:** Logging & Binary Path Spec (Phase 1-8) ✅ 완료

---

## 1. 미구현 작업 목록

### 구현 대상 (7 phases)

| # | LLM Analysis ID | 작업명 | 구현 비용 | LLM 자동화 영향 |
|---|-----------------|-------|----------|---------------|
| 1 | B | Observables API | 낮음 | 실패 분석 왕복 3~5턴 → 1턴 |
| 2 | C | 실패 컨텍스트 자동 캡처 | 중간 | 진단 완전성, 1회 호출로 전체 정보 |
| 3 | D | JSONL 이벤트 스트림 | 낮음 | Context 사용량 50%+ 절감 |
| 4 | A | 테스트 프론트매터 파서 | 낮음 | 메타 탐색 비용 제거 |
| 5 | G | Dry-run 모드 | 낮음 | 사전 계획 수립 |
| 6 | H | Compact State MCP tool | 낮음 | 매 턴 상태 체크 비용 절감 |
| 7 | - | 프론트매터 일괄 추가 | 중간 (수량) | Phase 4 이후 데이터 입력 |

### 구현 보류 (사유 명시)

| LLM Analysis ID | 작업명 | 보류 사유 |
|-----------------|-------|----------|
| E | 스펙 연결 MCP 리소스 | go-stablenet에 `REGRESSION_TEST_CASES*.md` 없음 — 연결 대상 부재 |
| F | 고수준 assertion helper | `tests/regression/lib/common.sh`에 이미 `assert_receipt_status`, `gov_full_flow` 등 도메인 헬퍼 존재 |
| I | rerun-failed + snapshot | Tier 3 — 실사용 피드백 후 결정 |
| J | 스펙 기반 스캐폴딩 | 스펙 문서 부재 (E와 동일 사유) |
| K | 의존성 그래프 | Tier 3 — 프론트매터 `depends_on` 데이터 축적 후 결정 |
| L | daemon 모드 | Tier 3 — 구현 난이도 높음, ROI 불확실 |

---

## 2. Phase별 구현 상세

### Phase 1: Observables API

**수정 파일:**
- `tests/lib/assert.sh` — `observe()` 함수 추가, `test_start`에 배열 초기화, `test_result`에 직렬화

**단위 테스트:**
- `tests/unit/tests/assert-observe.sh`
  - `observe` 호출 후 결과 JSON에 `observed` 필드 존재
  - 다수 observe → 올바른 key-value 매핑
  - observe 미호출 시 `observed: {}`
  - value에 특수문자 포함 시 올바른 JSON escape

**TDD 흐름:**
1. RED: `assert-observe.sh` 작성 → `observe: command not found`
2. GREEN: `observe()` + `_OBSERVED_*` 배열 + `test_result` Python 직렬화 수정

**커밋:** `feat(test): add observe() API and observables to result JSON`

---

### Phase 2: 실패 컨텍스트 자동 캡처

**신규 파일:**
- `tests/lib/failure_context.sh` — `_cb_capture_failure_context()` 함수

**수정 파일:**
- `tests/lib/assert.sh` — `test_result`에서 `fail > 0` 시 `_cb_capture_failure_context` 호출
- `.gitignore` — `state/failures/` 추가

**단위 테스트:**
- `tests/unit/tests/failure-context.sh`
  - mock pids.json 생성 → 캡처 함수 호출 → `context.json` 존재 확인
  - pids.json 없을 때 graceful skip
  - 디렉토리 명명 규칙 검증 (`<safe_name>_<ts>`)

**커밋:** `feat(test): add failure context auto-capture on test failure`

---

### Phase 3: JSONL 이벤트 스트림

**수정 파일:**
- `tests/lib/assert.sh` — `CB_FORMAT` 분기 추가:
  - `test_start`: JSONL 이벤트 stdout
  - `_assert_pass`/`_assert_fail`: JSONL 이벤트 stdout (기존 stderr 유지)
  - `observe`: JSONL 이벤트 stdout
  - `test_result`: `test_end` 이벤트 stdout
- `lib/cmd_test.sh` — `--format jsonl` 옵션 파싱, `CB_FORMAT=jsonl` export

**단위 테스트:**
- `tests/unit/tests/assert-jsonl.sh`
  - `CB_FORMAT=jsonl` 설정 후 `test_start` → stdout에 JSON 이벤트 출력
  - `assert_eq` → stdout에 `assert_pass` 또는 `assert_fail` 이벤트
  - `observe` → stdout에 `observe` 이벤트
  - `test_result` → stdout에 `test_end` 이벤트
  - `CB_FORMAT=text` (기본) → stdout 비어있음 (stderr만 사용)

**커밋:** `feat(test): add JSONL event stream output mode`

---

### Phase 4: 테스트 프론트매터 파서

**신규 파일:**
- `lib/test_meta.sh` — `cb_parse_meta()` 함수

**수정 파일:**
- `lib/cmd_test.sh` — `test list --format json` 응답에 메타 포함, `chainbench_test_meta` 기능

**단위 테스트:**
- `tests/unit/tests/test-meta-parse.sh`
  - 프론트매터 있는 fixture script → 정확한 JSON 파싱
  - 프론트매터 없는 fixture script → `{}`
  - 잘못된 YAML → `{}`
  - `id`, `tags`, `estimated_seconds`, `depends_on` 필드 추출 검증

**Fixture:**
- `tests/unit/fixtures/test-with-meta.sh` (프론트매터 있는 더미)
- `tests/unit/fixtures/test-without-meta.sh` (프론트매터 없는 더미)

**커밋:** `feat(test): add YAML frontmatter parser for test metadata`

---

### Phase 5: Dry-run 모드

**수정 파일:**
- `lib/cmd_test.sh` — `--dry-run` 플래그 파싱, `_cb_test_dry_run()` 함수 추가

**동작:**
1. 대상 스크립트 목록 수집 (`_cb_test_collect_scripts`)
2. 각 스크립트에서 `cb_parse_meta` 호출
3. JSON 또는 text로 실행 계획 출력 (스크립트 실행하지 않음)

**단위 테스트:**
- `tests/unit/tests/cmd-test-dryrun.sh`
  - fixture 디렉토리에 2개 스크립트 배치
  - `--dry-run --format json` → JSON에 scripts 배열, total 카운트
  - 실제 스크립트 미실행 확인

**커밋:** `feat(test): add dry-run mode for test execution planning`

---

### Phase 6: MCP 도구 확장

**신규 파일:**
- (없음 — 기존 파일 수정)

**수정 파일:**
- `mcp-server/src/tools/lifecycle.ts` (또는 별도 status tool):
  - `chainbench_state_compact` tool 추가
  - `chainbench status --compact --json` 호출
- `mcp-server/src/tools/test.ts`:
  - `chainbench_failure_context` tool 추가
  - `state/failures/` 최근 디렉토리 `context.json` 읽기
  - `chainbench_test_run` `format` 파라미터 추가
- `lib/cmd_status.sh`:
  - `--compact --json` 옵션 지원 (JSON으로 < 300 bytes 출력)
- `mcp-server/src/index.ts`: 등록

**검증:** `npx tsc --noEmit` 통과

**커밋:** `feat(mcp): add state_compact and failure_context tools`

---

### Phase 7: 프론트매터 일괄 추가

**수정 파일:**
- `tests/regression/a-ethereum/*.sh` (30개) — 우선 적용
- 나머지 카테고리 (b~g, 83개) — 피드백 후 적용

**프론트매터 생성 규칙:**
- `id`: 기존 `# RT-<ID>` 주석에서 추출
- `name`: 기존 `# RT-<ID> — <설명>` 주석에서 추출
- `category`: 디렉토리 경로에서 추출
- `tags`: 테스트 내용 기반 수동 태깅 (tx, contract, rpc, governance, sync 등)
- `estimated_seconds`: 테스트 body의 `sleep`, `wait_*` timeout 기반 추정
- `depends_on`: 테스트 body에서 다른 테스트의 산출물 참조 여부로 판단

**예시 변환:**

Before:
```bash
#!/usr/bin/env bash
# Test: regression/a-ethereum/a2-01-legacy-tx
# RT-A-2-01 — Legacy Tx (type 0x0) 발행
# [v2 보강] effectiveGasPrice + gasLimit valid check (21000)
set -euo pipefail
```

After:
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
# [v2 보강] effectiveGasPrice + gasLimit valid check (21000)
set -euo pipefail
```

**커밋 단위:** 10개 파일씩 배치 (3 커밋으로 a 카테고리 완료)

---

## 3. 의존성 그래프

```
Phase 1 (observe)
    │
    ├──→ Phase 2 (failure context)  ──→ Phase 6 (MCP tools)
    │
    └──→ Phase 3 (JSONL)            ──→ Phase 6 (MCP tools)

Phase 4 (frontmatter parser)
    │
    ├──→ Phase 5 (dry-run)
    │
    └──→ Phase 7 (frontmatter 일괄 추가)
```

**병렬 가능:** Phase 1과 Phase 4는 독립적. 동시 시작 가능.

---

## 4. 검증 체크리스트

각 Phase 완료 시:

- [ ] `bash tests/unit/run.sh` — 전체 통과
- [ ] `cd mcp-server && npx tsc --noEmit` — TypeScript 에러 없음
- [ ] 기존 regression 테스트 대표 1개 실행 (`a1-01`) — 기존 동작 유지 확인
- [ ] git commit (conventional commit 형식)

Phase 7 완료 시 추가:
- [ ] 프론트매터 있는 테스트 `chainbench test list --format json` → 메타 포함 확인
- [ ] `chainbench test run <target> --dry-run` → 계획 JSON 출력 확인
