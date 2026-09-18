# 실제 빌드 선택 파일의 AST 그래프

세 클라이언트의 Makefile 진입점과 build/ci.go 태그를 확인한 뒤 `go list -mod=readonly -deps -json`으로 빌드 의존성을 선택했다. 그중 해당 프로젝트 내부의 GoFiles와 CgoFiles만 별도 디렉터리에 보존하고 codemine의 tree-sitter 추출기를 실행했다. 전체 저장소의 모든 .go 파일을 빌드 파일로 간주하지 않았다.

## 빌드 기준

| 프로젝트 | Makefile 대상 / 바이너리 | darwin/arm64 태그 | linux/amd64 태그 |
|---|---|---|---|
| go-wemix | make gwemix / gwemix | 없음, USE_ROCKSDB=NO | rocksdb, Linux 기본 USE_ROCKSDB=YES 조건 |
| go-wbft | make gwemix / gwemix | urfave_cli_no_docs,ckzg | urfave_cli_no_docs,ckzg |
| go-stablenet | make gstable / gstable | urfave_cli_no_docs,ckzg | urfave_cli_no_docs,ckzg |

모든 프로필은 CGO_ENABLED=1, 분석에 사용한 Go는 1.25.12다. WBFT/Stable의 Make 빌드는 일반 `go list`와 달리 urfave_cli_no_docs와 ckzg 태그를 추가한다. Ubuntu trusty 예외·static 옵션·다른 OS/arch는 별도 프로필이다. 이 분석의 Linux 프로필은 Linux make 기본 선택 파일을 비교한 것이며 운영 서버의 실제 빌드 설정을 수집한 결과는 아니다. 근거: `sources/go-wemix/Makefile:12`, `sources/go-wemix/Makefile:45`, `sources/go-wbft/Makefile:17`, `sources/go-wbft/build/ci.go:209`, `sources/go-stablenet/Makefile:17`, `sources/go-stablenet/build/ci.go:209`.

WEMIX3의 cmd/gwemix는 cmd/geth를 가리키는 링크이므로 보존 파일과 AST에서는 cmd/geth로 정규화했다. 실제 go list의 진입 패키지 이름은 그대로 보존했다. 과거 BUILD_SOURCE_FILES.md의 다른 커밋 목록을 현재 빌드 목록으로 재사용하지 않았다.

## 생성 결과

| 프로젝트 | 프로필 | Go 파일 | 내부 패키지 | AST 타입 선언 | AST 함수·메서드 집계 | 정확한 내부 import 간선 |
|---|---|---:|---:|---:|---:|---:|
| go-wemix | darwin/arm64 | 630 | 120 | 1936 | 10195 | 890 |
| go-wemix | linux/amd64 | 630 | 120 | 1941 | 10236 | 891 |
| go-wbft | darwin/arm64 | 685 | 135 | 1824 | 9899 | 964 |
| go-wbft | linux/amd64 | 685 | 135 | 1824 | 9903 | 964 |
| go-stablenet | darwin/arm64 | 668 | 129 | 1528 | 8305 | 901 |
| go-stablenet | linux/amd64 | 668 | 129 | 1528 | 8309 | 901 |

Go 파일 수가 OS 사이에 같아도 파일 집합은 다르다. go-wemix는 Linux에서 rocksdb.go를 선택하고 darwin에서는 norocksdb.go를 선택한다. 세 프로젝트 모두 fdlimit, metrics disk, blake2b 구현 등에 OS/arch 차이가 있다. 전체 차집합은 `../build/summary.json`에 있다.

## 그래프를 읽는 방법

- `code-graph.json`: tree-sitter로 파싱한 모듈·타입 선언과 함수 집계.
- `module-imports.tsv`: AST 추출기가 이름을 해석한 모듈 참조와 횟수. 이름 해석에는 한계가 있다.
- `build-package-graph.json` / `.dot`: go list의 전체 import path로 구성한 정확한 패키지 의존 그래프. 내부 간선을 판정할 때 이 자료를 우선한다.
- `../build/*-selected.json`: 프로필, 선택 Go 파일, 제외된 Go 파일, 패키지 imports, C/assembly/embed 등 다른 빌드 입력 목록, 파일 해시.
- `../build/*release.json`: 외부 패키지와 표준 라이브러리까지 포함한 원래 go list 출력. WEMIX darwin 출력은 `go-wemix-darwin-arm64.json`이다.

Go 타입 간선은 추출기에서 0개로 나왔다. 인터페이스 구현체, 동적 디스패치 또는 함수 단위의 완전한 호출 그래프가 없다는 뜻이며, 프로그램에 타입 관계가 없다는 결론이 아니다. 런타임 경로와 정책 차이는 별도의 소스 추적으로 확인했다. C/assembly/embed 입력과 외부 모듈은 목록으로 남겼지만 해당 파일의 AST를 이 Go 그래프에 포함하지 않았다. 바이너리 링크나 테스트 실행은 하지 않았으므로 실제 배포 바이너리의 정상 빌드를 보증하지 않는다.

## 파일별 링크

| 프로젝트 | darwin/arm64 AST | Linux/amd64 AST | 정확한 패키지 그래프 |
|---|---|---|---|
| go-wemix | [AST](../graph/go-wemix/code-graph.json) | [AST](../graph/go-wemix-linux-amd64/code-graph.json) | [package graph](../graph/go-wemix/build-package-graph.json) |
| go-wbft | [AST](../graph/go-wbft/code-graph.json) | [AST](../graph/go-wbft-linux-amd64/code-graph.json) | [package graph](../graph/go-wbft/build-package-graph.json) |
| go-stablenet | [AST](../graph/go-stablenet/code-graph.json) | [AST](../graph/go-stablenet-linux-amd64/code-graph.json) | [package graph](../graph/go-stablenet/build-package-graph.json) |

[선택 모듈 다이어그램](build-diagrams.md)은 정확한 패키지 그래프의 일부를 보여 준다. 전체 간선과 외부 의존은 JSON에 있다.
