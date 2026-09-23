# chainbench 분석 자료

이 디렉터리는 **인용되는 분석**만 둔다. 어떤 문서가 "배경 분석은 여기" 라고 적으면
그 파일은 여기 남고 저장소에 커밋된다. 인용하는 곳이 없어지면 파일도 지운다.

읽는 순서를 정하는 색인이 아니라, **끊어진 인용을 만들지 않기 위한 목록**이다.

| 파일 | 누가 인용하나 |
|---|---|
| `08-dsl-15-node-analysis-response.md` | (이전부터 커밋돼 있던 것) |
| `10-prepared-inputs-server-ref-handoff.md` | `docs/dev/chainbench-worklist.md:1753` |
| `11-monitoring-open-issues.md` | `docs/dev/chainbench-worklist.md:1783`, `docs/dev/monitoring-issue-review-2026-09-10.md:4` |
| `12-mainnet-profile-plan.md` | `docs/dev/architecture/mainnet-config-worklist.md:11` |

## 아직 정리되지 않은 것

`13-state-driven-refactoring-review-2026-09-21.md` 와
`14-hsm-pattern-review-2026-09-21.md` 는 2026-09-21 의 상태 주도 리팩토링을 검토한 문서다.
결론을 `docs/dev/architecture/design-v3/` 로 옮긴 뒤 여기서 지운다. 두 문서가
`graph/` 와 `graph-snapshots/` 의 그래프 자료를 인용하므로 그것들도 같이 남아 있다.
어느 파일이 어느 커밋을 잰 것인지는 [`graph/README.md`](graph/README.md) 가 적는다.

`15-hsm-refactoring-handoff-2026-09-21.md` 는 그 리팩토링의 작업 prompt 이고,
`16-hsm-implementation-review-2026-09-22.md` 는 **들어온 구현을 검토한 문서**다. 16번은 14번(설계)과
15번(실행)을 인용하고, `Comparing` 가지의 성공 경로가 끊긴 것을 포함해 고쳐야 할 것 다섯을 우선순위로
적어뒀다.

`17-hsm-implementation-fix-prompt-2026-09-22.md` 는 16번의 지적을 **확인하고 고치는 작업 prompt** 다.
16번을 인용하며, 항목마다 확인 방법과 판정(`확인됨`/`문제 아님`/`일부만`)을 요구한다. 2.1 장에는
네트워크 없이 도는 RED 테스트가 그대로 들어 있다. 여섯 항목이 전부 닫히면 16번과 17번을 함께 지운다.

`18-hsm-state-diagrams-2026-09-22.md` 는 두 machine(`composition`, `run`)의 **state diagram** 이다.
state tree, Cmd 와 Event 목록, 전이와 그 조건, error 가 어디로 가는지를 코드에서 뽑아 mermaid 로
그렸다. `graph/lifecycle-transitions.mmd` 는 제거된 옛 lifecycle 을 그린 것이라 이것과 다른 그림이며,
18번이 그 자리를 대신한다. 16번·17번이 지워져도 **이 문서는 남긴다.** 코드가 바뀌면 같이 고친다.

2026-09-21 에 지운 것: 한 세션이 코드를 파악하려고 만든 분석 여섯(`01`~`05`), 끝난 일의
인수인계서 셋(`06`·`07`·`09`), 저장소 소스 스냅샷을 품고 있던
`reuse-audit-20260909-131030/`(6.7MB). 인용하는 곳이 없었다.
