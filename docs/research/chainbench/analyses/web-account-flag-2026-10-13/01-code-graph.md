# 전체 AST 코드 그래프

## 방법과 범위

`git ls-files`가 반환한 Go, Python, JavaScript, Svelte, 셸, HTML, CSS 파일을 모두 대상으로 했다. 테스트와 저장된 SPA 번들도 포함했다. JSON/YAML은 데이터와 설정이므로 소스 AST 범위에서 제외했다. 미추적 파일, 의존성 설치 폴더, Git 내부 데이터는 포함하지 않았다.

| 대상 | 파서 | 추출 정보 |
|---|---|---|
| Go 전체 | tree-sitter-go 0.25.0 | 선언, import, 구문상 호출, 오류 위치 |
| Go 주요 코드 트리 | 저장소의 go/parser 도구 | 패키지, 공개 심볼을 참조하는 import 간선, 선언 위치 |
| Python | tree-sitter-python 0.25.0 | 선언, import, 구문상 호출 |
| JS/MJS | tree-sitter-javascript 0.25.0 | 선언, import, 구문상 호출 |
| 셸 | tree-sitter-bash 0.25.1 | 함수, 명령 노드, 오류 위치 |
| Svelte/HTML/CSS | 잠긴 Svelte 5.56.8 컴파일러 | 스크립트, 템플릿, 스타일의 AST와 컴포넌트 참조 |

codemine의 추출·통계 스크립트를 먼저 실행했다. 기본 스크립트가 지원하지 않는 웹과 셸은 `tools/`의 보완 도구로 추가했다. Go 도구는 import의 전체 경로를 사용하므로 basename만으로 연결하는 기본 모듈 그래프보다 정확하다. (`scripts/inventory/code-graph/main.go:1-8`, `scripts/inventory/code-graph/main.go:133-153`)

## 측정값

수치는 `graph/summary.json`과 `graph/code-graph.json`에서 가져왔다.

| 파일 종류 | 파싱한 파일 수 |
|---|---:|
| Go | 1,141 |
| Python | 100 |
| JavaScript | 8 |
| MJS | 79 |
| Svelte | 25 |
| 셸 | 62 |
| HTML | 3 |
| CSS | 1 |
| 합계 | 1,419 |

| 그래프 | 측정값 |
|---|---:|
| 전체 AST 노드 | 2,292,588 |
| 구문상 호출 기록 | 75,404 |
| 파일 import 간선 | 372 |
| Go 패키지 import 출현 | 1,759 |
| 주요 Go 코드 트리의 패키지 | 78 |
| 주요 Go 코드 트리의 패키지 간선 | 263 |
| 주요 Go 코드 트리의 선언 | 5,054 |
| codemine depth 3 모듈 | 77 |
| codemine 타입 항목 | 1,085 |

구문상 호출 기록은 함수의 실행 횟수나 완전히 해석된 호출 그래프가 아니다. 테스트와 생성된 번들에 같은 기능의 코드가 반복되므로 AST 노드 수가 제품 복잡도를 직접 뜻하지도 않는다.

저장소의 Go 도구는 `cmd`, `internal`, `scripts`, `tests`만 순회한다. 그래서 Go 파일 1,140개를 대상으로 한다. 나머지 `docs/dev/codegraph/main.go`도 전체 소스 그래프에서는 파싱했다. (`scripts/inventory/code-graph/main.go:140`, `docs/dev/codegraph/main.go:1`)

## 연결이 많은 패키지

fan-in은 해당 패키지를 import하는 제품 코드 패키지 수다. 아래 수치는 저장소의 Go 도구가 측정한 것이다. 테스트 import도 섞는 codemine 모듈 통계와 직접 비교하지 않는다.

| 패키지 | fan-in | 이번 변경에서의 의미 |
|---|---:|---|
| `internal/app` | 19 | 웹, CLI와 MCP가 사용하는 기능과 실행 주체가 모인다. |
| `internal/core/node` | 19 | 노드 모델을 공유한다. 로그인 선택을 위해 바꿀 필요는 없다. |
| `internal/core/registry` | 18 | 체인 등록 경로다. 인증 선택과 직접 연결되지 않는다. |
| `internal/preset` | 15 | 구성 자료를 읽는다. 계정 유무와 별개다. |
| `cmd/chainbench/surface` | 13 | CLI 실행 의존성을 조립한다. |
| `internal/core/filestore` | 13 | 영속 파일 저장을 공유한다. |

대시보드 진입점은 `app`, `dashboard`, `webcli`, 체인 등록과 관측 버스를 연결한다. 로그인 선택은 여기서 주입하는 인증 함수와 실행 권한 함수의 차이로 표현할 수 있다. (`cmd/chainbench-dashboard/main.go:16-23`, `cmd/chainbench-dashboard/main.go:93-112`, `cmd/chainbench-dashboard/main.go:131-151`)

## 계층과 타입 그래프의 한계

Go 도구는 core에서 preset으로 가는 간선 6개를 계층 위반으로 보고했다. 이는 도구의 경로 분류 결과이며, 이번 계정 기능을 구현하면서 새로 생긴 간선이 아니다. 원시 목록은 `graph/summary.json`에 보존했다. 과거 그래프 문서에도 같은 관계가 기록되어 있다. (`docs/research/chainbench/analyses/graph/README.md:68-75`)

기본 타입 그래프의 상속 간선은 주로 Python 테스트 클래스에서 나온다. Go interface 구현 여부는 타입 검사로 확인하지 않았으므로 이 그래프에서 구현 관계를 추론하지 않는다. 실행 중 함수 값, reflection, JSON 요청 경로, Svelte 반응형 실행 순서도 import 그래프만으로 결정되지 않는다. 인증 경로의 연결은 실제 호출 지점으로 별도 확인했다. (`internal/dashboard/deployment.go:17`, `internal/app/web_job_execution.go:49-80`)

## 파싱 오류

`env/docker/gen-env.sh:48`의 중첩된 Bash 산술식에서 tree-sitter가 오류 노드를 만들었다. 전체 파일은 오류를 복구하면서 파싱했고, 위치를 보존했다. `bash -n env/docker/gen-env.sh`는 통과했다. 따라서 이 결과만으로 셸 코드의 문법 오류라고 판단하지 않는다. 나머지 1,418개 파일에는 파서 오류가 없었다.
