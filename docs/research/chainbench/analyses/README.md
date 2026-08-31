# chainbench 프로젝트 구조 분석

> 생성일: 2026-08-31 · 최초 분석 커밋: `0bcf105` · 계획 보정 커밋: `0c57644` · tree-sitter AST 590개 소스 파일 · 심층 조사 7개 주제

| 문서 | 내용 |
|---|---|
| [01-code-graph.md](01-code-graph.md) | AST 코드 그래프와 의존 구조 |
| [02-diagrams.md](02-diagrams.md) | 현재 실행 흐름과 목표 경계 |
| [03-feature-list.md](03-feature-list.md) | 기능과 구현 모듈의 대응 |
| [04-history-evolution.md](04-history-evolution.md) | 구조가 형성된 과정 |
| [05-design-insights.md](05-design-insights.md) | 리팩토링 우선순위와 검증 전략 |
| [06-refactoring-plan.md](06-refactoring-plan.md) | 합의한 목표 모듈과 bottom-up 실행 계획 |

## 빠른 요약

- 저장소는 61개 Go 패키지로 나뉘어 있고 레이어 역방향 의존과 파일 쓰기 소유권을 테스트로 감시한다 (`internal/arch/layers_test.go:89`, `internal/arch/state_test.go:57`).
- 디렉터리 깊이 2의 AST 집계에서는 `internal/core`가 22,967 LOC, `cmd/chainbench`가 10,333 LOC다. 이는 여러 실제 Go 패키지를 합친 수치이므로 크기 자체보다 경계 간 실행 중복을 우선 봐야 한다.
- `keyring`은 모델, 파생, 저장, 동작, CLI 순서가 실제 경계로 드러난 기준 구현이다 (`internal/core/keyring/operation/operation.go:1`).
- `keyring` 뒤의 리팩토링은 작은 책임을 나누기보다 큰 소유자 패키지로 옮기거나 합친 경우가 많다. `resource`는 pool, port, inventory, 서버 설정, SSH 접근을 함께 담는다 (`internal/resource/pool.go:17`, `internal/resource/serverset.go:64`).
- 네트워크 구성도 `chainsetup.NetUp`과 `testengine.NewBuildEnv`에 각각 존재한다 (`internal/chainsetup/verbs_up.go:85`, `internal/testengine/buildenv.go:71`). 다만 상위 경로를 먼저 합치지 않고, 하위 모듈의 계약부터 정리한다.
- `testspec`은 문법·마이그레이션·정적 검증·런타임 인터프리터를 한 패키지에 담아 인프라 의존이 넓다 (`internal/testspec/interpreter.go:6`).
- CLI의 workspace 실행은 `app.RunSuite`를 거치지만 attach/local 실행은 CLI에서 엔진을 직접 조립한다 (`cmd/chainbench/run.go:60`, `cmd/chainbench/run.go:192`).
- 우선순위는 `resource → node → genesis → nodeconfig → process → chainsetup → testengine → surface`다. 각 단계는 새 소유자, 옛 판단 지점 삭제, 다음 계층의 산출물 재사용을 함께 증명해야 한다.
