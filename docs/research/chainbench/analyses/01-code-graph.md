# AST 코드 그래프 분석

## 1. 어떻게 분석했는가

codemine의 tree-sitter 추출기로 Go·Python·JavaScript 소스를 파싱하고, 모듈 깊이 2에서 선언과 import edge를 추출했다. `vendor`, `node_modules`, `dist`는 제외했다. Go 패키지의 실제 경계는 `go list ./internal/... ./cmd/...` 및 저장소의 아키텍처 테스트로 보정했다.

| AST 정보 | 용도 |
|---|---|
| 파일·LOC·타입·함수 | 집중된 변경 지점 탐색 |
| 모듈 import edge | fan-in과 결합 방향 확인 |
| Go 패키지 목록 | 깊이 2 집계가 합친 실제 경계 복원 |
| git timeline | 구조 변화의 시점 확인 |

## 2. 그래프 규모

| 항목 | 수치 |
|---|---:|
| 소스 파일 | 590 |
| LOC | 75,498 |
| 타입 | 502 |
| 깊이 2 모듈 | 25 |
| 모듈 import edge | 68 |
| 실제 Go 패키지 (`internal`+`cmd`) | 61 |

상위 집계 모듈은 `internal/core` 22,967 LOC, `cmd/chainbench` 10,333 LOC, `internal/chainsetup` 5,790 LOC, `internal/consensus` 4,462 LOC다. `internal/core` 아래에는 driver, launcher, session, registry 등 독립 패키지가 있으므로 이 수치는 폴더 재배치의 직접 근거가 아니라 조사 우선순위다.

## 3. 모듈 그래프

| 모듈 | fan-in | 해석 |
|---|---:|---|
| `internal/accounts` | 12 | 계정·서명 어휘를 여러 테스트/체인 기능이 공유 |
| `internal/testkit` | 8 | 기존 Go 함수 테스트 모델의 공용 결과/등록 경계 |
| `internal/resource` | 6 | 서버 세트·배치·포트·접근을 묶은 자원 facade |
| `internal/consensus` | 5 | 합의 패밀리별 genesis/upgrade 기능 |
| `internal/chainsetup` | 4 | workspace 기반 네트워크 구성 오케스트레이션 |
| `internal/testspec` | 4 | DSL 파싱부터 실행 계약까지 포함 |

레이어 규칙은 문서의 배치 표를 읽어 모든 패키지의 소속과 역방향 import를 검사한다 (`internal/arch/layers_test.go:24`, `internal/arch/layers_test.go:124`). 이 방식은 규칙의 복제본을 코드에 만들지 않는 장점이 있다. 다만 MCP가 core를 직접 import하는 예외는 별도 shrink-only 목록으로 남아 있어 표면 경로는 아직 이행 중이다 (`internal/arch/mcp_imports_test.go:17`).

## 4. 타입 그래프

Go 트리에서는 상속 edge가 추출되지 않았다. 대신 중요한 계약은 인터페이스에 있다.

- `registry.ChainPlugin`과 `ConsensusFamily`가 체인 확장 계약을 컴파일 단계에서 강제한다 (`internal/core/registry/registry.go:29`, `internal/core/registry/registry.go:97`).
- `testengine.Deps`가 세션 생성, 환경 구성, 스펙 실행을 주입한다 (`internal/testengine/engine_impl.go:16`).
- `testspec.Registry`는 액션·어설션·리더를 이름으로 연결하지만 등록 누락은 컴파일러가 잡지 못한다 (`internal/testspec/interpreter.go:29`).
- `session.Session`이 환경과 테스트 레코드의 생성·저장 경계를 소유한다 (`internal/core/session/session.go:27`).

## 5. 그래프에서 읽히는 설계 원칙

1. 레이어 방향은 이미 기계적으로 보호된다. 다음 단계는 같은 방향 안에서 중복된 실행 흐름을 하나로 합치는 일이다.
2. `chainsetup`을 네트워크 구성의 단일 소유자로 만들고 `testengine`은 구성된 환경의 테스트 실행만 맡아야 한다.
3. `testspec`의 순수 문법 모델과 I/O 런타임 인터프리터는 패키지 경계로 분리해야 한다.
4. `app`에는 `RunSuite` 같은 유스케이스만 남기고 타입 별칭·전달 래퍼는 이행 완료 후 제거해야 한다.
5. CLI가 core 계약을 원자적으로 호출할 수 있게 하되, 조립 규칙은 공용 application service에 두어 MCP와 동작이 갈라지지 않게 해야 한다.
