# 구현 검증 보고서

> **검증일**: 2026-04-12
> **검증 대상**: `docs/superpowers/specs/2026-04-09-logging-and-binary-path-design.md` (Logging & Binary Path Spec)
> **검증 대상**: `docs/LLM_INTEGRATION_ANALYSIS.md` (LLM Integration Roadmap)
> **검증 기준**: 코드 구현체 (main 브랜치, commit `c2ff06e`)

---

## 1. Logging & Binary Path Design Spec 검증

### 1.1 구현 현황 요약

| Phase | 커밋 | 상태 | 단위 테스트 |
|-------|------|------|-----------|
| 0. stale .bak 제거 | N/A | ✅ 완료 (해당 파일 없음) | - |
| 1. Unit test runner | `ed9c884` | ✅ 완료 | 12 assertions |
| 2. resolve_binary 테스트 | `d50924f` | ✅ 완료 | 6 assertions |
| 3. 런타임 오버라이드 | `f1dfc9a`, `c2ff06e` | ✅ 완료 | 11 assertions |
| 4. env-first + overlay | `6a9a06f` | ✅ 완료 | 9 assertions |
| 5. config 서브커맨드 | `1a79d34` | ✅ 완료 | 9 assertions |
| 6. logrot 통합 | `863705b` | ✅ 완료 | 7 assertions |
| 7. MCP 도구 확장 | `fd4a7d1` | ✅ 완료 | TypeScript 빌드 통과 |
| 8. 문서 업데이트 | `f51a0f6` | ✅ 완료 | - |

**합계: 9 phase 중 9개 완료, 54개 단위 테스트 assertion 전체 통과**

### 1.2 컴포넌트별 상세 검증

#### `lib/common.sh` — 설계 §5.1

| 함수 | 설계 위치 | 구현 위치 | 검증 |
|------|----------|----------|------|
| `resolve_logrot()` | §4.3, §5.1 | `common.sh:222-275` | ✅ 6단계 discovery chain 구현 |
| `_cb_build_logrot_from_source()` | §5.1 | `common.sh:280-304` | ✅ main.go 검출 → `go build` → logrot-build.log |
| `_cb_parse_runtime_overrides()` | §5.1 | `common.sh:147-209` | ✅ nameref 배열, `--binary-path`, `--logrot-path`, `=` 구문, 검증 |

#### `lib/profile.sh` — 설계 §5.2

| 변경사항 | 설계 위치 | 구현 위치 | 검증 |
|---------|----------|----------|------|
| `_cb_set_var` env-first guard | §5.2 | `profile.sh:385-399` | ✅ `CHAINBENCH_PROFILE_ENV_OVERRIDE` 확인, `${!var_name+x}` 테스트 |
| overlay 머지 (Python block) | §5.2 | `profile.sh:261-269` | ✅ `state/local-config.yaml` deep_merge, `inherits` 제거 |
| `CHAINBENCH_LOGROT_PATH` export | §5.2 | `profile.sh:411` | ✅ `.chain.logrot_path` 필드 export |

#### CLI 커맨드 파서 연결 — 설계 §5.3

| 파일 | 설계 위치 | 구현 위치 | 검증 |
|------|----------|----------|------|
| `cmd_init.sh` | §5.3 | `cmd_init.sh:8-11` | ✅ `_cb_parse_runtime_overrides` 호출 후 `set --` |
| `cmd_start.sh` | §5.3 | `cmd_start.sh:7-9` | ✅ profile 로딩 전 파서 실행 |
| `cmd_restart.sh` | §5.3 | `cmd_restart.sh:84-86` | ✅ `cmd_restart_main` 진입부에서 파싱 |
| `cmd_node.sh` | §5.3 | `cmd_node.sh:307-309` | ✅ `_cb_node_cmd_start` 내부, node_num 소비 후 파싱 |

#### `cmd_start.sh` logrot 파이프라인 — 설계 §5.4

| 변경사항 | 설계 위치 | 구현 위치 | 검증 |
|---------|----------|----------|------|
| logrot 해석 | §4.4, §5.4 | `cmd_start.sh:105-113` | ✅ `resolve_logrot` 호출, `is_truthy` 체크 |
| process substitution launch | §4.4 | `cmd_start.sh:216-222` | ✅ `> >("${LOGROT_BIN}" ...)` 패턴, `$!`로 gstable PID 캡처 |
| dead code 제거 | §5.4 | - | ✅ `_logrot_bin` 변수 선언 제거됨 |

#### `cmd_config.sh` — 설계 §5.5

| 서브커맨드 | 설계 위치 | 구현 위치 | 검증 |
|-----------|----------|----------|------|
| `config list` | §5.5 | `cmd_config.sh` `_cb_config_list` | ✅ 파일 내용 출력 또는 `(empty)` |
| `config get <field>` | §5.5 | `cmd_config.sh` `_cb_config_get` | ✅ dot-notation, exit 1 on miss |
| `config set <field> <value>` | §5.5 | `cmd_config.sh` `_cb_config_set` | ✅ JSON 파싱, atomic write (temp+rename) |
| `config unset <field>` | §5.5 | `cmd_config.sh` `_cb_config_unset` | ✅ cascade-clean empty dicts |
| field validation | §5.5 | `cmd_config.sh:15` | ✅ `^[a-zA-Z0-9_][a-zA-Z0-9_.]*$`, `..` 거부 |

#### MCP 서버 — 설계 §5.6, §5.7, §5.8

| 컴포넌트 | 설계 위치 | 구현 위치 | 검증 |
|---------|----------|----------|------|
| `shellEscapeArg` | §5.6 | `utils/exec.ts:38-40` | ✅ POSIX single-quote escape |
| `binary_path` on `chainbench_init` | §5.7 | `lifecycle.ts:59` | ✅ optional param, validation, `buildBinaryPathArg` |
| `binary_path` on `chainbench_start` | §5.7 | `lifecycle.ts:93` | ✅ |
| `binary_path` on `chainbench_restart` | §5.7 | `lifecycle.ts:124` | ✅ |
| `binary_path` on `chainbench_node_start` | §5.7 | `node.ts:46-49` | ✅ validation + shellEscapeArg |
| `chainbench_config_set` | §5.8 | `config.ts:14-32` | ✅ field regex validation, shellEscapeArg |
| `chainbench_config_get` | (추가) | `config.ts:34-47` | ✅ |
| `chainbench_config_list` | (추가) | `config.ts:49+` | ✅ |
| `registerConfigTools` | §5.8 | `index.ts:11,27` | ✅ |

#### Profile & 기타 — 설계 §5.9

| 변경사항 | 구현 위치 | 검증 |
|---------|----------|------|
| `profiles/default.yaml` — `chain.logrot_path` | `default.yaml:10` | ✅ |
| `.gitignore` — `state/local-config.yaml` | `.gitignore:8` | ✅ |
| `.gitignore` — `state/logrot-build.log` | `.gitignore:9` | ✅ |
| `schema.ts` — `chain.logrot_path` 문서 | `schema.ts:21` | ✅ |
| `README.md` — config 섹션, logrot 트러블슈팅 | README.md | ✅ |
| `setup.sh` — next-steps 변경 | `setup.sh:105-109` | ✅ |

### 1.3 설계 대비 차이점

| 항목 | 설계 | 실제 구현 | 영향 |
|------|------|----------|------|
| `cmd_stop.sh` logrot cleanup | §5.4 "message" | `pkill -f` + `log_info` 추가 | ✅ 설계 초과 충족 |
| `chainbench_config_get` MCP tool | 설계에 없음 (Non-Goal) | 구현됨 | ✅ 추가 기능 (유용) |
| `chainbench_config_list` MCP tool | 설계에 없음 | 구현됨 | ✅ 추가 기능 (유용) |
| Unit test directory layout | §8.1과 동일 | `tests/unit/{run.sh,lib/,tests/,fixtures/}` | ✅ 일치 |

### 1.4 사이드 이펙트 분석

| 영향 범위 | 분석 결과 |
|----------|----------|
| 기존 regression 테스트 | ⚠️ 없음 — `tests/regression/` 미변경 |
| 기존 basic/fault/stress 테스트 | ⚠️ 없음 — 미변경 |
| `_cb_set_var` 호환성 | ✅ env 미설정 시 기존과 동일 (profile 값 사용) |
| `cmd_start.sh` logrot 없는 환경 | ✅ `LOGROT_BIN=""` → plain `>>` append (기존 동작) |
| MCP tool optional params | ✅ `binary_path` 생략 시 기존과 동일 |
| `cmd_init.sh` parser 추가 | ✅ 인자 없으면 파서가 no-op, 기존 동작 유지 |

---

## 2. LLM Integration Analysis 검증

### 2.1 구현 현황

| Tier | ID | 제안명 | 코드 존재 여부 | 상태 |
|------|----|-------|--------------|------|
| 1 | A | 스크립트 프론트매터 (YAML-in-comment) | `# ---chainbench-meta---` 블록 없음 | ❌ 미구현 |
| 1 | B | 관찰값(observables) 캡처 API | `observe()` 함수 없음, `_OBSERVED_*` 배열 없음 | ❌ 미구현 |
| 1 | C | 실패 시 자동 컨텍스트 캡처 | `state/failures/` 디렉토리 없음, MCP tool 없음 | ❌ 미구현 |
| 1 | D | JSON-first 출력 모드 (NDJSON) | `--format jsonl` 옵션 없음, `CB_FORMAT` 없음 | ❌ 미구현 |
| 2 | E | 스펙 연결 MCP 리소스 | `mcp-server/src/tools/spec.ts` 없음 | ❌ 미구현 |
| 2 | F | 고수준 assertion helper | `tests/lib/assert_chain.sh` 없음 | ❌ 미구현 |
| 2 | G | Dry-run / plan 모드 | `--dry-run` 옵션 없음 | ❌ 미구현 |
| 2 | H | 압축 상태 MCP tool | `chainbench_state_compact` 없음 | ❌ 미구현 |
| 3 | I | rerun-failed + snapshot | `rerun-failed`, `snapshot` 커맨드 없음 | ❌ 미구현 |
| 3 | J | 스캐폴딩 | `test scaffold` 없음 | ❌ 미구현 |
| 3 | K | 의존성 그래프 | `test graph` 없음 | ❌ 미구현 |
| 3 | L | daemon 모드 | `daemon` 서브커맨드 없음 | ❌ 미구현 |

**검증 방법**: `grep -r` 로 각 제안별 키워드 (`observe`, `chainbench-meta`, `--format jsonl`, `chainbench_failure_context`, `spec_lookup`, `assert_tx_success`, `--dry-run`, `state_compact`, `rerun-failed`, `scaffold`, `test graph`, `daemon`) 검색 → `docs/` 하위 문서에만 존재, 코드 구현체에는 없음.

**결론: 12개 제안 중 0개 구현됨**

---

## 3. 테스트 검증

### 3.1 테스트 파일 목록

| # | 파일 | Assertion 수 | 커버리지 |
|---|------|-------------|---------|
| 1 | `smoke-meta.sh` | 12 | 테스트 프레임워크 자체 검증 |
| 2 | `common-resolve-binary.sh` | 6 | `resolve_binary` 6단계 우선순위 |
| 3 | `common-parse-overrides.sh` | 11 | `_cb_parse_runtime_overrides` 파싱 |
| 4 | `common-resolve-logrot.sh` | 7 | `resolve_logrot` 7단계 discovery |
| 5 | `profile-env-override.sh` | 4 | `_cb_set_var` env-first guard |
| 6 | `profile-overlay-merge.sh` | 5 | overlay 머지 + deep merge + inherits 무시 |
| 7 | `cmd-config.sh` | 9 | config get/set/unset/list/validation |
| | **합계** | **54** | |

### 3.2 빌드 검증

| 대상 | 결과 |
|------|------|
| `bash tests/unit/run.sh` | ✅ 7/7 passed, 54/54 assertions |
| `npx tsc --noEmit` (MCP server) | ✅ 0 errors |
| `shellcheck` (해당 시) | 미실행 (shellcheck 미설치) |

---

## 4. 문서 상태 정리

| 문서 | 현재 상태 | 권장 조치 |
|------|----------|----------|
| `docs/superpowers/specs/2026-04-09-logging-and-binary-path-design.md` | 구현 완료 | `docs/archive/`로 이동 또는 상단에 "Implemented" 표기 |
| `docs/LLM_INTEGRATION_ANALYSIS.md` | 12개 제안 모두 미구현 | 유지 — 향후 로드맵 역할 |
| `docs/REMAINING_TASKS.md` | 현황 반영 완료 | 유지 |
| `docs/superpowers/` 디렉토리명 | 프로젝트와 무관한 명칭 | `docs/specs/`로 리네임 권장 |

---

## 5. 결론

### Logging & Binary Path Spec
- **설계 문서 대비 100% 구현 완료** (9/9 phases)
- 설계에 명시된 모든 함수, 변수, 파일, MCP tool이 정확한 위치에 구현됨
- CLI 연결 (`_cb_parse_runtime_overrides` call sites)이 초기 누락되었으나 `c2ff06e`에서 수정 완료
- 설계에 없는 추가 기능 (`config_get`, `config_list` MCP tools)도 구현

### LLM Integration Analysis
- **12개 제안 중 0개 구현** — 이 문서는 로드맵/분석 문서이며, 구현 착수를 위해서는 별도 설계 문서(approved spec) 작성이 필요
- 권장 착수 순서: B (observables) → C (failure context) → D (JSONL) → A (frontmatter) → G (dry-run)
