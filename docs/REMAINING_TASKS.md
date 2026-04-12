# chainbench 작업 현황

> **최초 작성**: 2026-04-12
> **최종 업데이트**: 2026-04-12 (LLM Integration Tier 1 + Tier 2 일부 구현 완료)
> **기준 커밋**: `b72c973`

---

## 1. Logging & Binary Path Design Spec — 전체 완료

> 문서: `docs/superpowers/specs/2026-04-09-logging-and-binary-path-design.md`

| Phase | 내용 | 커밋 | 상태 |
|-------|------|------|------|
| 0 | stale .bak 제거 | N/A | ✅ 완료 |
| 1 | Unit test runner scaffolding | `ed9c884` | ✅ 완료 |
| 2 | resolve_binary 단위 테스트 | `d50924f` | ✅ 완료 |
| 3 | `--binary-path` / `--logrot-path` 런타임 오버라이드 | `f1dfc9a`, `c2ff06e` | ✅ 완료 |
| 4 | env-first `_cb_set_var` + local overlay 머지 | `6a9a06f` | ✅ 완료 |
| 5 | `chainbench config` CLI 서브커맨드 | `1a79d34` | ✅ 완료 |
| 6 | logrot 통합 | `863705b` | ✅ 완료 |
| 7 | MCP 도구 확장 (binary_path, config_set) | `fd4a7d1` | ✅ 완료 |
| 8 | 문서 업데이트 | `f51a0f6` | ✅ 완료 |

**9/9 phases 완료**

---

## 2. LLM Integration Analysis — 구현 현황

> 문서: `docs/LLM_INTEGRATION_ANALYSIS.md`
> 설계: `docs/specs/llm-test-automation-design.md`

### Tier 1 — Quick Win (4/4 완료)

| ID | 제안 | 커밋 | 구현 위치 | 상태 |
|----|------|------|----------|------|
| A | 프론트매터 (YAML-in-comment) | `3e3ed2c`, `b72c973`, `f055925`, `4e15f44`, `0c125ee` | `lib/test_meta.sh`, 전체 114개 스크립트 (100%), `test list --format json` | ✅ 완료 |
| B | 관찰값(observables) 캡처 | `069ce95` | `tests/lib/assert.sh:82` `observe()`, 결과 JSON `observed` 필드 | ✅ 완료 |
| C | 실패 시 자동 컨텍스트 캡처 | `46df850` | `tests/lib/failure_context.sh`, `assert.sh:266` 자동 호출 | ✅ 완료 |
| D | JSONL 이벤트 스트림 | `e2c0a4e` | `assert.sh` CB_FORMAT 분기 5개소, `cmd_test.sh` `--format jsonl` | ✅ 완료 |

### Tier 2 — 구조 개선 (2/4 완료)

| ID | 제안 | 커밋 | 구현 위치 | 상태 |
|----|------|------|----------|------|
| E | 스펙 연결 MCP 리소스 | - | - | ⏸️ 보류 (go-stablenet에 spec 문서 부재) |
| F | 고수준 assertion helper | - | - | ⏸️ 보류 (`common.sh`에 `assert_receipt_status`, `gov_full_flow` 이미 존재) |
| G | Dry-run / plan 모드 | `33381b5` | `cmd_test.sh:230` `_cb_test_dry_run`, `--dry-run` 옵션 | ✅ 완료 |
| H | 압축 상태 MCP tool | `aef22e6` | `test.ts:208` `chainbench_state_compact`, `cmd_status.sh` `--compact` | ✅ 완료 |

### Tier 3 — 대화형 워크플로우 (0/4)

| ID | 제안 | 상태 | 보류 사유 |
|----|------|------|----------|
| I | rerun-failed + snapshot/restore | ⏸️ 보류 | Tier 3 — 실사용 피드백 후 결정 |
| J | 스펙 기반 테스트 스캐폴딩 | ⏸️ 보류 | spec 문서 부재 (E와 동일) |
| K | 의존성 그래프 | ⏸️ 보류 | 프론트매터 `depends_on` 데이터 축적 후 결정 |
| L | MCP 대화형 세션 (daemon) | ⏸️ 보류 | 구현 난이도 높음, ROI 불확실 |

### 요약

```
전체 12개 제안:  6개 구현 ✅  |  6개 보류 ⏸️  |  0개 미착수
Tier 1 (A-D):   4/4 완료
Tier 2 (E-H):   2/4 완료 (E,F 사유 있는 보류)
Tier 3 (I-L):   0/4 보류
```

---

## 3. 단위 테스트 현황

| # | 파일 | Assertions | 커버리지 |
|---|------|-----------|---------|
| 1 | `smoke-meta.sh` | 12 | 테스트 프레임워크 자체 검증 |
| 2 | `common-resolve-binary.sh` | 6 | resolve_binary 6단계 우선순위 |
| 3 | `common-parse-overrides.sh` | 11 | _cb_parse_runtime_overrides 파싱 |
| 4 | `common-resolve-logrot.sh` | 7 | resolve_logrot 7단계 discovery |
| 5 | `profile-env-override.sh` | 4 | _cb_set_var env-first guard |
| 6 | `profile-overlay-merge.sh` | 5 | overlay 머지 + deep merge |
| 7 | `cmd-config.sh` | 9 | config get/set/unset/list |
| 8 | `assert-observe.sh` | 6 | observe() API + 결과 JSON |
| 9 | `test-meta-parse.sh` | 6 | 프론트매터 파서 |
| 10 | `assert-jsonl.sh` | 5 | JSONL 이벤트 스트림 |
| 11 | `failure-context.sh` | 7 | 실패 컨텍스트 캡처 |
| 12 | `cmd-test-dryrun.sh` | 4 | dry-run 모드 |
| | **합계** | **82** | |

---

## 4. 문서 상태

| 문서 | 상태 | 조치 |
|------|------|------|
| `docs/REMAINING_TASKS.md` | ✅ 최신 | 이 문서 |
| `docs/IMPLEMENTATION_VERIFICATION.md` | ✅ 최신 | 동시 갱신 |
| `docs/LLM_INTEGRATION_ANALYSIS.md` | ℹ️ 원본 유지 | 로드맵 원본으로 보존 (구현 현황은 이 문서에서 추적) |
| `docs/specs/llm-test-automation-design.md` | ✅ 유효 | Phase 1-7 설계 문서 |
| `docs/specs/llm-test-automation-plan.md` | ✅ 유효 | Phase 1-7 구현 계획 |
| `docs/superpowers/specs/2026-04-09-logging-and-binary-path-design.md` | 📦 아카이브 대상 | 구현 완료 |
