# 관련 이력

전체 저장소는 분석 시점까지 745개 커밋을 갖는다. 월별 수치와 디렉터리 최초 등장 기록은 스크립트가 만든 [git-timeline.md](graph/git-timeline.md)에 보존했다. 키워드 `auth`가 commit message에 나온 것만으로 Web 계정 구현의 시작이라고 판단하지 않았다.

| 날짜 | 커밋 | 이번 분석과 관련된 변화 |
|---|---|---|
| 2026-07-27 | `002655c3` | Svelte SPA를 Go 서버에서 제공하는 변경이 들어왔다. |
| 2026-08-24 | `f619718c` | 데몬 이름이 chainbench-dashboard로 바뀌었다. |
| 2026-10-12 | `b9e5be39` | 보안과 영속 작업 기능이 현재 Web 진입점에 결합됐다. |
| 2026-10-13 | `662fff77` | Web 작업이 chainbench CLI를 사용하도록 바뀌었다. |

위 표는 해당 파일의 `git log`와 커밋 제목을 기준으로 요약했다. 커밋의 전체 기능을 제목만으로 보증하는 표는 아니다.

현재 코드에서 실행 재사용 경계는 CLI runner에 있고, 인증 경계는 함수로 주입된다. 반면 감사 API는 concrete WebAuth를 참조한다. 계정 선택을 구현할 때 이 불균형을 정리하는 것이 적합하다. (`cmd/chainbench-dashboard/main.go:135-142`, `internal/dashboard/deployment.go:17`, `internal/dashboard/audit.go:60-78`)

과거 그래프를 현재 코드의 증거로 재사용하지 않았다. 이번 그래프는 현재 커밋에서 다시 측정했고 기존 `analyses/graph/` 파일은 보존했다.
