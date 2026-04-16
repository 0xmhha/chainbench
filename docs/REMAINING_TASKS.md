# chainbench 작업 현황

> **최초 작성**: 2026-04-12
> **최종 업데이트**: 2026-04-16
> **기준 커밋**: `89a1c99`

---

## 완료 항목 요약

| 영역 | 상태 | 비고 |
|------|------|------|
| Logging & Binary Path (9 phases) | ✅ 전체 완료 | logrot 통합, config overlay, MCP 확장 |
| LLM Integration Tier 1 (A-D) | ✅ 4/4 완료 | 프론트매터, observables, JSONL |
| LLM Integration Tier 2 (E-H) | ✅ 3/4 완료 | F 의도적 보류 |
| LLM Integration Tier 3 (I-L) | ✅ 1/4 완료 | J 완료, I·K·L 의도적 보류 |
| Regression 테스트 스크립트 | ✅ 114개 | 7개 섹션 (A~G) |
| Hardfork 테스트 스크립트 | ✅ 40개 | h-hardfork 디렉토리 |

---

## 남은 작업

### Phase A — 즉시 가능 ✅ 완료 (2026-04-16, `0d2f1fd`)

| # | 작업 | 상태 |
|---|------|------|
| A-1 | `test-meta-parse.sh` 단위 테스트 실패 수정 | ✅ bash 따옴표 충돌 해소 |
| A-2 | `hardfork-boho-pre.yaml` 프로파일 생성 | ✅ BohoBlock=999999999 |
| A-3 | `hardfork-boho-post.yaml` 프로파일 생성 | ✅ BohoBlock=0 |

### Phase B — Layer 2 테스트 유틸리티

> **의사결정 완료 (2026-04-16)**: 도구 선택 — 3-tier 구조
>
> | 역할 | 도구 | 적용 범위 |
> |------|------|----------|
> | 표준 tx/ABI/이벤트 | **cast** (foundry) | 대부분 TC (A, B, C, E, G 섹션) |
> | 체인 고유 tx 타입 (0x16, EIP-7702) | **Go 헬퍼** (`chainutil`, go-stablenet import) | D, F 섹션 + 하드포크 TC |
> | 의도적 malformed/invalid tx | **Python** (rlp + inline `python3 -c`) | 거부 경로 테스트 (소수) |
>
> Go 헬퍼는 go-stablenet의 `cmd/chainutil/main.go`로 추가, `resolve_binary` 패턴으로 자동 탐색/빌드.

| # | 파일 | 줄 수 | 커밋 | 상태 |
|---|------|------|------|------|
| B-1 | `tests/lib/system_contracts.sh` | 107 | `c9a8312` | ✅ 완료 |
| B-2 | `tests/lib/contract.sh` | 223 | `7cac347` | ✅ 완료 |
| B-3 | `tests/lib/event.sh` | 246 | `33f1f36` | ✅ 완료 |
| B-4 | `tests/lib/chain_state.sh` | 203 | `66c112f` | ✅ 완료 |
| B-5 | `tests/lib/tx_builder.sh` | 292 | `a13ebab` | ✅ 완료 |
| B-T | 단위 테스트 5개 (`lib-*.sh`) | — | `857add9` | ✅ 20/20 pass |
| B-E | E2E 테스트 5개 (`z-layer2-e2e/z-0*.sh`) | — | `7e2fefa` | ✅ 생성 (체인 실행 시 검증) |

**총 1,071줄** 신규 라이브러리 코드 + 단위 테스트 20개 통과.

남은 TODO: `tests/regression/lib/common.sh`의 인라인 함수(`send_raw_tx`, `gov_full_flow`, `assert_receipt_status`)를 Layer 2로 점진적 마이그레이션 (기존 테스트 호환 유지).

### Phase C — CI/커맨드 통합 (Layer 2 완료 후)

| # | 작업 | 상태 |
|---|------|------|
| C-1 | Claude Code 커맨드 2종 (`stablenet-test-hardfork.md`, `stablenet-test-regression.md`) | ❌ 미생성 |
| C-2 | MCP 도구 확장 (hardfork/regression 테스트 실행) | ❌ 미생성 |
| C-3 | GitHub Actions 워크플로우 (선택) | ❌ 미생성 |

### 보류 항목 (의도적 — 해제 조건 충족 시 진행)

| ID | 항목 | 보류 사유 | 해제 조건 |
|----|------|----------|----------|
| F | 고수준 assertion helper | `common.sh`에 4개 이미 존재 | 테스트 작성 중 반복 패턴 발견 시 |
| I | rerun-failed + snapshot/restore | ROI 불확실 | 실사용 피드백 후 |
| K | 의존성 그래프 | `depends_on` 데이터 부족 | 프론트매터 축적 후 |
| L | MCP 대화형 세션 (daemon) | 구현 난이도 높음 | 명확한 유즈케이스 확보 시 |

---

## 단위 테스트 현황

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
| 9 | `test-meta-parse.sh` | 6 | 프론트매터 파서 (**1 실패**) |
| 10 | `assert-jsonl.sh` | 5 | JSONL 이벤트 스트림 |
| 11 | `failure-context.sh` | 7 | 실패 컨텍스트 캡처 |
| 12 | `cmd-test-dryrun.sh` | 4 | dry-run 모드 |
| 13 | `smoke-logrot-integration.sh` | 1 | logrot 파일 로테이션 (SKIP if unavailable) |
| | **합계** | **83** | |

---

## 문서 상태

| 문서 | 상태 | 비고 |
|------|------|------|
| `docs/REMAINING_TASKS.md` | ✅ 현행 | 이 문서 (마스터 추적) |
| `docs/chainbench-test-system-design.md` | 🔄 진행 중 | hardfork/regression 테스트 설계 (Phase B 참조) |

### 정리 이력

**2026-04-16**: 구현 완료된 문서 7개 삭제 (git 이력에 보존):
- `IMPLEMENTATION_VERIFICATION.md`, `LLM_INTEGRATION_ANALYSIS.md`
- `specs/llm-test-automation-{design,plan}.md`
- `superpowers/specs/2026-04-09-logging-and-binary-path-design.{md,ko.md}`
- `superpowers/plans/2026-04-16-logging-binary-path-remaining.md`
