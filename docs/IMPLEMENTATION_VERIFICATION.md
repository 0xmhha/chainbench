# 구현 검증 보고서

> **최초 검증일**: 2026-04-12
> **최종 갱신일**: 2026-04-12
> **기준 커밋**: `b72c973` (main 브랜치)

---

## 1. Logging & Binary Path Design Spec — 100% 구현 완료

### 1.1 구현 현황

| Phase | 커밋 | 상태 | 단위 테스트 |
|-------|------|------|-----------|
| 0. stale .bak 제거 | N/A | ✅ 완료 | - |
| 1. Unit test runner | `ed9c884` | ✅ 완료 | 12 assertions |
| 2. resolve_binary 테스트 | `d50924f` | ✅ 완료 | 6 assertions |
| 3. 런타임 오버라이드 | `f1dfc9a`, `c2ff06e` | ✅ 완료 | 11 assertions |
| 4. env-first + overlay | `6a9a06f` | ✅ 완료 | 9 assertions |
| 5. config 서브커맨드 | `1a79d34` | ✅ 완료 | 9 assertions |
| 6. logrot 통합 | `863705b` | ✅ 완료 | 7 assertions |
| 7. MCP 도구 확장 | `fd4a7d1` | ✅ 완료 | TypeScript 빌드 통과 |
| 8. 문서 업데이트 | `f51a0f6` | ✅ 완료 | - |

### 1.2 컴포넌트별 코드 위치

| 컴포넌트 | 파일:라인 | 검증 |
|---------|----------|------|
| `resolve_logrot()` | `lib/common.sh:222-275` | ✅ |
| `_cb_build_logrot_from_source()` | `lib/common.sh:280-304` | ✅ |
| `_cb_parse_runtime_overrides()` | `lib/common.sh:147-209` | ✅ |
| 파서 호출: cmd_init | `lib/cmd_init.sh:10` | ✅ |
| 파서 호출: cmd_start | `lib/cmd_start.sh:9` | ✅ |
| 파서 호출: cmd_restart | `lib/cmd_restart.sh:85` | ✅ |
| 파서 호출: cmd_node | `lib/cmd_node.sh:309` | ✅ |
| `_cb_set_var` env-first guard | `lib/profile.sh:385-399` | ✅ |
| overlay 머지 (Python) | `lib/profile.sh:261-269` | ✅ |
| `CHAINBENCH_LOGROT_PATH` export | `lib/profile.sh:411` | ✅ |
| logrot launch pipeline | `lib/cmd_start.sh:216-222` | ✅ |
| `config` 서브커맨드 | `lib/cmd_config.sh` | ✅ |
| `shellEscapeArg` | `mcp-server/src/utils/exec.ts:38-40` | ✅ |
| `binary_path` MCP param | `lifecycle.ts:59,93,124`, `node.ts:46-49` | ✅ |
| `chainbench_config_set/get/list` | `mcp-server/src/tools/config.ts` | ✅ |
| `chain.logrot_path` profile | `profiles/default.yaml:10` | ✅ |
| `.gitignore` overlay/logrot | `.gitignore:8-9` | ✅ |

---

## 2. LLM Integration Analysis — 6/12 구현 완료

### 2.1 구현 완료 항목 (6개)

#### A. 스크립트 프론트매터 (YAML-in-comment)

| 검증 항목 | 결과 |
|----------|------|
| `lib/test_meta.sh` 존재 | ✅ `cb_parse_meta()` 함수 |
| 전체 regression 스크립트 적용 | ✅ 114/114 파일에 `# ---chainbench-meta---` 블록 (100%) |
| `test list --format json` | ✅ `cmd_test.sh` `_cb_test_cmd_list_json` 함수 |
| 파싱 정확성 | ✅ id, name, category, tags, estimated_seconds, depends_on 추출 |
| 단위 테스트 | ✅ `tests/unit/tests/test-meta-parse.sh` (6 assertions) |

#### B. 관찰값(observables) 캡처 API

| 검증 항목 | 결과 |
|----------|------|
| `observe()` 함수 | ✅ `tests/lib/assert.sh:82` |
| `_OBSERVED_KEYS[]` / `_OBSERVED_VALUES[]` | ✅ `assert.sh:21-22` |
| 결과 JSON `observed` 필드 | ✅ `assert.sh:223-234` 직렬화 |
| `test_start`에서 배열 초기화 | ✅ `assert.sh:39-40` |
| 단위 테스트 | ✅ `tests/unit/tests/assert-observe.sh` (6 assertions) |

#### C. 실패 시 자동 컨텍스트 캡처

| 검증 항목 | 결과 |
|----------|------|
| `tests/lib/failure_context.sh` | ✅ 107 lines, `_cb_capture_failure_context()` line 14 |
| `assert.sh`에서 자동 호출 | ✅ `assert.sh:266` (`fail > 0` 조건) |
| `assert.sh`에서 source | ✅ `assert.sh:8-10` |
| `state/failures/` 디렉토리 | ✅ `.gitignore`에 포함 |
| 수집 항목: eth_blockNumber, net_peerCount, eth_syncing | ✅ Python 블록 내 RPC 호출 |
| 수집 항목: 최근 5블록 hash/stateRoot | ✅ Python 블록 내 구현 |
| 수집 항목: node log tail -200 | ✅ subprocess.run(["tail"]) |
| 단위 테스트 | ✅ `tests/unit/tests/failure-context.sh` (7 assertions) |

#### D. JSON-first 출력 모드 (NDJSON)

| 검증 항목 | 결과 |
|----------|------|
| `CB_FORMAT` 환경변수 | ✅ `assert.sh` 5개소 참조 |
| `test_start` → JSONL 이벤트 | ✅ `assert.sh:46-49` |
| `_assert_pass` → JSONL 이벤트 | ✅ `assert.sh:58-60` |
| `_assert_fail` → JSONL 이벤트 | ✅ `assert.sh:68-70` |
| `observe` → JSONL 이벤트 | ✅ `assert.sh:87-89` |
| `test_result` → `test_end` 이벤트 | ✅ `assert.sh:271-274` |
| `--format jsonl` CLI 옵션 | ✅ `lib/cmd_test.sh:307` |
| `CB_FORMAT` export | ✅ `lib/cmd_test.sh:313` |
| MCP `chainbench_test_run` format param | ✅ `mcp-server/src/tools/test.ts` format enum |
| 단위 테스트 | ✅ `tests/unit/tests/assert-jsonl.sh` (5 assertions) |

#### G. Dry-run / plan 모드

| 검증 항목 | 결과 |
|----------|------|
| `_cb_test_dry_run()` 함수 | ✅ `lib/cmd_test.sh:230` |
| `--dry-run` 옵션 파싱 | ✅ `lib/cmd_test.sh:307` |
| JSON 출력 (target, scripts, meta, total) | ✅ Python 블록 내 구현 |
| text 출력 | ✅ bash printf 기반 |
| `test_meta.sh` 연동 | ✅ `cb_parse_meta` 호출 |
| 단위 테스트 | ✅ `tests/unit/tests/cmd-test-dryrun.sh` (4 assertions) |

#### H. 압축 상태 MCP tool

| 검증 항목 | 결과 |
|----------|------|
| `chainbench_state_compact` MCP tool | ✅ `mcp-server/src/tools/test.ts:208` |
| `--compact` CLI 옵션 | ✅ `lib/cmd_status.sh:26` |
| compact JSON 출력 (< 300 bytes) | ✅ Python 블록 내 구현 |
| pids.json fallback | ✅ TypeScript fallback 로직 |
| `chainbench_failure_context` MCP tool | ✅ `mcp-server/src/tools/test.ts:164-205` |

### 2.2 보류 항목 (6개)

| ID | 제안 | 보류 사유 | 재검토 조건 |
|----|------|----------|-----------|
| E | 스펙 연결 MCP 리소스 | go-stablenet에 `REGRESSION_TEST_CASES*.md` 부재 | 스펙 문서 생성 시 |
| F | 고수준 assertion helper | `common.sh`에 `assert_receipt_status`, `gov_full_flow` 등 이미 존재 | 부족 시 추가 |
| I | rerun-failed + snapshot | Tier 3, 실사용 피드백 필요 | 반복 디버깅 빈도 증가 시 |
| J | 스펙 기반 스캐폴딩 | spec 문서 부재 (E와 동일) | E 해결 후 |
| K | 의존성 그래프 | 프론트매터 `depends_on` 데이터 축적 필요 | 전체 스크립트 frontmatter 완료 후 |
| L | daemon 모드 | 구현 난이도 높음, ROI 불확실 | 성능 병목 측정 후 |

---

## 3. 테스트 검증

### 3.1 단위 테스트 (12 파일, 82 assertions)

| # | 파일 | Assertions | 대상 |
|---|------|-----------|------|
| 1 | `smoke-meta.sh` | 12 | 테스트 프레임워크 |
| 2 | `common-resolve-binary.sh` | 6 | `resolve_binary` |
| 3 | `common-parse-overrides.sh` | 11 | `_cb_parse_runtime_overrides` |
| 4 | `common-resolve-logrot.sh` | 7 | `resolve_logrot` |
| 5 | `profile-env-override.sh` | 4 | `_cb_set_var` env-first |
| 6 | `profile-overlay-merge.sh` | 5 | overlay merge |
| 7 | `cmd-config.sh` | 9 | config 서브커맨드 |
| 8 | `assert-observe.sh` | 6 | observe() API |
| 9 | `test-meta-parse.sh` | 6 | 프론트매터 파서 |
| 10 | `assert-jsonl.sh` | 5 | JSONL 이벤트 |
| 11 | `failure-context.sh` | 7 | 실패 컨텍스트 |
| 12 | `cmd-test-dryrun.sh` | 4 | dry-run 모드 |

### 3.2 빌드 검증

| 대상 | 결과 |
|------|------|
| `bash tests/unit/run.sh` | ✅ 12/12 passed, 82 assertions |
| `npx tsc --noEmit` (MCP server) | ✅ 0 errors |

### 3.3 프론트매터 적용 현황

| 카테고리 | 전체 스크립트 | 프론트매터 적용 | 비율 |
|---------|-------------|---------------|------|
| a-ethereum | 32 | 32 | 100% |
| b-wbft | 12 | 12 | 100% |
| c-anzeon | 7 | 7 | 100% |
| d-fee-delegation | 4 | 4 | 100% |
| e-blacklist-authorized | 9 | 9 | 100% |
| f-system-contracts | 27 | 27 | 100% |
| g-api | 21 | 21 | 100% |
| **합계** | **114** | **114** | **100%** |

---

## 4. 사이드 이펙트 분석

| 영향 범위 | 결과 |
|----------|------|
| 기존 regression 테스트 로직 | ⚠️ 없음 — 프론트매터는 주석이므로 실행에 영향 없음 |
| `test_result` JSON 스키마 | `observed` 필드 추가 — 기존 소비자는 무시 가능 (additive change) |
| `assert.sh` 호환성 | `observe()` 미호출 시 `observed: {}` — 기존 테스트 영향 없음 |
| `failure_context.sh` source | `assert.sh` 상단에서 조건부 source — 파일 없으면 skip |
| `CB_FORMAT` 기본값 | `text` — 기존 동작 변경 없음 |
| MCP tool 추가 | additive — 기존 tool 시그니처 변경 없음 (`format` param은 optional default) |

---

## 5. 전체 요약

### 구현 통계

```
Logging & Binary Path Spec:      9/9 phases  (100%)
LLM Integration Tier 1 (A-D):    4/4 items   (100%)
LLM Integration Tier 2 (E-H):    2/4 items   ( 50%)
LLM Integration Tier 3 (I-L):    0/4 items   (  0%)
────────────────────────────────────────────────────
전체:                             15/21 items  ( 71%)
단위 테스트:                      12 files, 82 assertions
TypeScript:                       0 errors
사이드 이펙트:                    없음
```

### 미완료 사항 (사유 있는 보류)

| 항목 | 핵심 차단 요인 |
|------|--------------|
| E (스펙 연결) | go-stablenet에 regression spec 문서 미존재 |
| F (고수준 assertion) | `common.sh`에 이미 충분한 도메인 헬퍼 존재 |
| I-L (Tier 3) | 실사용 피드백 수집 전 ROI 판단 불가 |
| 프론트매터 전체 카테고리 | ✅ 114/114 완료 (a~g 전체) |
