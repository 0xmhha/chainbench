# chainbench 남은 작업 리스트

> **최초 작성**: 2026-04-12
> **최종 업데이트**: 2026-04-12 (Phase 1-8 구현 완료)
> **기준 문서**: `docs/LLM_INTEGRATION_ANALYSIS.md`, `docs/superpowers/specs/2026-04-09-logging-and-binary-path-design.md`

---

## 1. Logging & Binary Path Design Spec — 구현 현황

> 문서: `docs/superpowers/specs/2026-04-09-logging-and-binary-path-design.md`

### Phase 0: stale .bak 파일 제거
- **상태**: ✅ 완료 (`.bak` 파일 없음)

### Phase 1: Unit test runner scaffolding
- **상태**: ✅ 완료 (`ed9c884`)
- `tests/unit/run.sh`, `tests/unit/lib/assert.sh`, `tests/unit/tests/smoke-meta.sh`, `tests/unit/fixtures/mock-gstable`

### Phase 2: resolve_binary 단위 테스트
- **상태**: ✅ 완료 (`d50924f`)
- `tests/unit/tests/common-resolve-binary.sh` — 6개 assertion

### Phase 3: `--binary-path` / `--logrot-path` 런타임 오버라이드
- **상태**: ✅ 완료 (`f1dfc9a`)
- `_cb_parse_runtime_overrides()`, `resolve_logrot()`, `_cb_build_logrot_from_source()` 구현
- `tests/unit/tests/common-parse-overrides.sh` — 11개 assertion

### Phase 4: env-first `_cb_set_var` + local overlay 머지
- **상태**: ✅ 완료 (`6a9a06f`)
- `_cb_set_var` env-first guard 적용, overlay 머지 로직, `.gitignore` 업데이트
- `tests/unit/tests/profile-env-override.sh` — 4개 assertion
- `tests/unit/tests/profile-overlay-merge.sh` — 5개 assertion

### Phase 5: `chainbench config` CLI 서브커맨드
- **상태**: ✅ 완료 (`1a79d34`)
- `lib/cmd_config.sh` — get/set/unset/list, atomic write, field validation
- `tests/unit/tests/cmd-config.sh` — 9개 assertion

### Phase 6: logrot 통합
- **상태**: ✅ 완료 (`863705b`)
- `cmd_start.sh` — process substitution 기반 logrot 파이프라인
- `profiles/default.yaml` — `chain.logrot_path` 필드 추가
- `cmd_stop.sh` — logrot cleanup 메시지 개선
- `tests/unit/tests/common-resolve-logrot.sh` — 7개 assertion

### Phase 7: MCP 도구 확장
- **상태**: ✅ 완료 (`fd4a7d1`)
- `shellEscapeArg` 유틸, `binary_path` 파라미터 (init/start/restart/node_start)
- `mcp-server/src/tools/config.ts` — `chainbench_config_set/get/list`
- `schema.ts` — `chain.logrot_path` 문서화

### Phase 8: 문서 업데이트
- **상태**: ✅ 완료 (`f51a0f6`)
- README.md, setup.sh 업데이트

### 요약: 9 phases 중 **9개 완료**

---

## 2. LLM Integration Analysis — 미구현 작업

> 문서: `docs/LLM_INTEGRATION_ANALYSIS.md`
> 성격: 로드맵/분석 문서 (A~L 총 12개 제안)
> 상태: **Tier 1~3 모두 미구현** (Logging spec 완료 후 착수 권장)

### Tier 1 — Quick Win

| ID | 제안 | 구현 상태 |
|----|------|----------|
| A | 스크립트 프론트매터 (YAML-in-comment) | ❌ 미구현 |
| B | 관찰값(observables) 캡처 API | ❌ 미구현 |
| C | 실패 시 자동 컨텍스트 캡처 | ❌ 미구현 |
| D | JSON-first 출력 모드 (NDJSON) | ❌ 미구현 |

### Tier 2 — 구조 개선

| ID | 제안 | 구현 상태 |
|----|------|----------|
| E | 스펙 연결 MCP 리소스 | ❌ 미구현 |
| F | 고수준 assertion helper | ❌ 미구현 |
| G | Dry-run / plan 모드 | ❌ 미구현 |
| H | 압축 상태 MCP tool | ❌ 미구현 |

### Tier 3 — 대화형 워크플로우

| ID | 제안 | 구현 상태 |
|----|------|----------|
| I | rerun-failed + snapshot/restore | ❌ 미구현 |
| J | 스펙 기반 테스트 스캐폴딩 | ❌ 미구현 |
| K | 의존성 그래프 | ❌ 미구현 |
| L | MCP 대화형 세션 (daemon) | ❌ 미구현 |

---

## 3. 불필요한 문서

| 문서 | 판정 | 사유 |
|------|------|------|
| `docs/superpowers/` 디렉토리 | ⚠️ 구조 변경 권장 | `docs/specs/`로 리네임 권장 |
| `docs/LLM_INTEGRATION_ANALYSIS.md` | ✅ 유지 | Tier 1~3 로드맵 역할 |
| `docs/superpowers/specs/2026-04-09-logging-and-binary-path-design.md` | ⚠️ 아카이브 가능 | 구현 완료. `docs/archive/`로 이동 권장 |
