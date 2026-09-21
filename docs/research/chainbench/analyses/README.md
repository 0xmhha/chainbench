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

2026-09-21 에 지운 것: 한 세션이 코드를 파악하려고 만든 분석 여섯(`01`~`05`), 끝난 일의
인수인계서 셋(`06`·`07`·`09`), 저장소 소스 스냅샷을 품고 있던
`reuse-audit-20260909-131030/`(6.7MB). 인용하는 곳이 없었다.
