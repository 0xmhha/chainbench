# chainbench 분석 자료

이 디렉터리는 **인용되는 분석**만 둔다. 어떤 문서가 "배경 분석은 여기" 라고 적으면
그 파일은 여기 남고 저장소에 커밋된다. 인용하는 곳이 없어지면 파일도 지운다.

읽는 순서를 정하는 색인이 아니라, **끊어진 인용을 만들지 않기 위한 목록**이다.

| 파일 | 누가 인용하나 |
|---|---|
| `10-prepared-inputs-server-ref-handoff.md` | `docs/dev/chainbench-worklist.md:1781`, `docs/guide/config-files.md:141` |
| `12-mainnet-profile-plan.md` | `docs/dev/architecture/mainnet-config-worklist.md:11` |
| `18-hsm-state-diagrams-2026-09-22.md` | `graph/README.md` |
| `19-state-machine-gaps-2026-09-24.md` | 상태 머신 빈 경로 수정안의 검토 근거(수정이 들어가면 그 커밋이 인용한다) |
| `20-state-naming-review-2026-09-24.md` | 상태 이름을 02 규약으로 되돌리는 검토(결정 D1~D5 뒤 구현 커밋이 인용한다) |

`18-hsm-state-diagrams-2026-09-22.md` 는 두 machine(`composition`, `run`)의 **state diagram** 이다.
state tree, Cmd 와 Event 목록, 전이와 그 조건, error 가 어디로 가는지를 코드에서 뽑아 mermaid 로
그렸다. `graph/lifecycle-transitions.mmd` 는 제거된 옛 lifecycle 을 그린 것이라 이것과 다른 그림이며,
18번이 그 자리를 대신한다. 코드가 바뀌면 같이 고친다.

`graph/` 는 어느 파일이 어느 커밋을 잰 것인지 [`graph/README.md`](graph/README.md) 에 적는다.

## 지운 것

2026-09-24 에 지운 것: 상태 주도 리팩토링의 검토·작업 prompt 다섯(`13`~`17`) — 리팩토링이
PR #425(`cde3a08f`)로 머지됐고 결론은 `docs/dev/architecture/design-v3/` 에 있다. 모니터링
이슈 분석 `11` — 모니터링 트랙이 머지돼 끝났다. 인용이 없던 `08`, 옛 그래프 측정 `graph-snapshots/`.
본문은 `git show cde3a08f:docs/research/chainbench/analyses/<파일>` 로 본다.

2026-09-21 에 지운 것: 한 세션이 코드를 파악하려고 만든 분석 여섯(`01`~`05`), 끝난 일의
인수인계서 셋(`06`·`07`·`09`), 저장소 소스 스냅샷을 품고 있던
`reuse-audit-20260909-131030/`(6.7MB). 인용하는 곳이 없었다.
