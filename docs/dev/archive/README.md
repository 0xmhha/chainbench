# archive/ — 대체·완료된 문서

> **새 작업의 근거로 쓰지 말 것.** 여기 있는 문서는 그때 무엇을 제안·측정했는지의
> 기록이다. 현재 무엇을 만들 것인지는 [`../chainbench-worklist.md`](../chainbench-worklist.md) 와
> `dev/` 의 **[현행 설계]** 문서가 정한다.

**지우지 않고 옮기는 이유**: 제안이 왜 그렇게 결정됐는지는 결정 자체와 별개의 정보다.
문서를 지우면 근거가 사라지고, 같은 논의를 다시 하게 된다. 위험한 것은 오래된 문서가
아니라 **오래됐다고 표시되지 않은 문서**다.

> 몇몇 문서는 정본(`chainbench-worklist`·`chainbench-design`)이 자기 **근거**로 인용한다
> (`chainbench-refactoring` WP1~6 · `chainbench-component-architecture` §2b 등).
> 그 인용은 유효하다 — 도출 과정을 가리키는 것이기 때문이다. 유효하지 않은 것은
> 여기 문서를 **현재 상태의 근거**로 쓰는 것이다.

## 설계 제안 — 구현되어 대체됨

| 문서 | 언제 | 무엇으로 대체됐나 |
|---|---|---|
| [`structure-and-atomic-cli-proposal.md`](structure-and-atomic-cli-proposal.md) | 2026-08-11 | 제안한 `internal/app` · `internal/core/launchopt` 는 **구현 완료**(launchopt 는 이후 `core/nodeconfig` 로 흡수). 남은 표면 논의는 [`../surface-unification-design.md`](../surface-unification-design.md). |
| [`chain-cli-execution-plan.md`](chain-cli-execution-plan.md) | 2026-08-10 (`2424ccc`) | 자체 헤더가 "진행 정본은 worklist" 라고 선언한다. 순서는 [`../chainbench-worklist.md`](../chainbench-worklist.md) §1g. |
| [`remote-wemix-deploy-design.md`](remote-wemix-deploy-design.md) | 2026-08 | 자기 헤더가 **[대체됨] 2026-09-05** 를 선언한다. `chainbench remote` 명령군과 `internal/chains/wemix/deploy` 는 없다 — `chain up --server` · poa 패밀리 phase 액션 · `upgrade`/`hardfork` · `keyring import --from srv://` 가 같은 일을 한다. |
| [`v2-move-map.md`](v2-move-map.md) | 2026-08-25 | **이동 완료.** 이동 대상 8패키지(`netcompose`·`launchopt`·`engine`·`testspec`·`keyreg` 등, 541 심볼)가 **코드에 전부 없다**(2026-09-11 확인). 모듈 경계는 [`../architecture/architecture-v2.md`](../architecture/architecture-v2.md). |
| [`chain-setup-next-automation.md`](chain-setup-next-automation.md) | 2026-08-10 (`2082f94`) | **완료.** "절차는 확정됐고 자동화하는 코드가 없다"가 이 문서의 한 줄 요약이었다. 그 코드가 `chainbench chain up`(20 스텝)이다. |
| [`wemix4-migration-plan.md`](wemix4-migration-plan.md) | 2026-08-10 | 절차 검토. 진행·판정은 [`../wemix4-port-tracker.md`](../wemix4-port-tracker.md) 가 승계했다. wemix4 실행 모델(단일 연속 체인·stateful phase) 서술은 여기가 원본이다. |

## 상태 추적 — 완료되어 닫힘

| 문서 | 언제 | 왜 닫혔나 |
|---|---|---|
| [`legacy-retirement-plan.md`](legacy-retirement-plan.md) | 2026-09-01 | 자기 헤더가 **실행 완료(R5)** 를 선언한다. `internal/testkit`·`internal/core/pipeline/testrun`·`cmd/chainbench test` 레거시·MCP `chainbench_test*`·레거시 케이스 패키지가 삭제됐다. |
| [`remaining-work.md`](remaining-work.md) | 2026-08-14 | 근거 ledger `tests/specs/README.md` 가 없다(→ `tests/tc/`). 남은 항목은 worklist 로 이관됐고, 본문이 인용하는 `internal/testkit`·`internal/engine`·`setup --launch` 는 전부 존재하지 않는다. |
| [`repro-migration-remaining.md`](repro-migration-remaining.md) | 2026-07-29 | 자기 §"Remaining — nothing to port". 유일한 비대상 `attach-external` 은 포팅 대상이 아니라고 결론냈다. e2e 실행법은 [`../../../tests/e2e/README.md`](../../../tests/e2e/README.md). |
| [`stablenet-post-v1.0.0-change-test-catalog.md`](stablenet-post-v1.0.0-change-test-catalog.md) | 2026-08-05 | 포팅 참고용 카탈로그 51건. [`../legacy-test-migration.md`](../legacy-test-migration.md) §7 이 이관 완료를 기록한다. 케이스 목록으로서의 값은 남는다. |
| [`handoff-2026-08-22.md`](handoff-2026-08-22.md) | 2026-08-22 | 세션 인수인계 스냅샷. 그날의 컨텍스트다. |

## 측정 스냅샷 — 재측정으로 대체됨

| 문서 | 언제 | 대체 |
|---|---|---|
| [`chainbench-audit-2026-08-09.md`](chainbench-audit-2026-08-09.md) | 2026-08-09 (#204) | 감사 절차와 근거의 보존본. 현재 상태는 worklist 와 코드. |
| [`chainbench-refactoring.md`](chainbench-refactoring.md) | 2026-08 | pkg→internal 시기의 WP1~6 감사. 대부분 완료. worklist 가 자기 근거로 인용한다. |
| [`chainbench-component-architecture.md`](chainbench-component-architecture.md) | 2026-08-09 | High/Middle/Low 계층 + TDD 구현 플랜. 구현 착수 **전** 문서. worklist 가 §2b·§3·§5·§1b 를 근거로 인용한다. |
| [`chain-binary-flag-graph.md`](chain-binary-flag-graph.md) | 2026-08-11 | 3체인 바이너리 플래그 AST 실측 + 실행옵션 모듈 설계 비판. 체인 바이너리 표면의 정본은 이제 [`../../chain-analysis/`](../../chain-analysis/README.md) 이고, 설계분은 `core/nodeconfig` 로 구현됐다. |
| [`software-architecture-2026-08-11.md`](software-architecture-2026-08-11.md) | 2026-08-11 (`2181191`) | 계층·컨텍스트·실행모델·환경 5요소·검증원·동시성. 현재 계층은 [`../architecture/layers.md`](../architecture/layers.md), 실측은 [`../architecture/code-graph.md`](../architecture/code-graph.md). |
| [`component-diagram-2026-08-11.md`](component-diagram-2026-08-11.md) | 2026-08-11 | 컴포넌트 맵 · C4 컨테이너 뷰. 목표 그림은 [`../architecture/target-architecture.md`](../architecture/target-architecture.md). |
| [`sequence-diagrams-2026-08-11.md`](sequence-diagrams-2026-08-11.md) | 2026-08-11 | 전체 run · BuildEnv · Interpreter · 원자 스텝 CLI · 원격 SSH · 실패 경로. |
| [`state-diagrams-2026-08-11.md`](state-diagrams-2026-08-11.md) | 2026-08-11 | TestRun · Environment · NodeProcess · Session/키 · 워크스페이스 스텝. |

> 이 네 다이어그램 묶음(software-architecture · component · sequence · state)은 서로를
> 인용하는 한 세트다. 2026-08-11 커밋 `2181191` 기준이며, 그 뒤 R·N·U 트랙이 지나갔으므로
> 패키지 경로는 대부분 현재와 다르다.
