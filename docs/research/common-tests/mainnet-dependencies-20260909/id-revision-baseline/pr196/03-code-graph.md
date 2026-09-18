# AST 그래프와 영향 범위

## 분석 기준

로컬 `dev` HEAD `902f9fce85c108cf24cdeb77bc70a622049748ce`와 PR head `5a93553cb25d41770c7419cd325229ac81bb3042`를 각각 `git archive`로 보존했다. 작업 트리와 Git 인덱스를 수정하지 않았다. PR base는 `master`의 `c8321fc693bef619ec8abd80c82f5a224e89802a`다. 양쪽은 분기된 상태이며 로컬을 PR 소스로 덮는 방식으로 통합하면 안 된다.

AST는 codemine의 기존 `extract_graph.py`(tree-sitter)와 `go_inventory.go`(Go 표준 go/parser)로 생성했다. 새 AST 파서를 만들지 않았다. Go inventory의 전체 import 경로를 집계한 그래프가 `../graph/exact-import-graph.json`이다. 기본 tree-sitter의 basename 기반 import 연결은 동명 패키지를 혼동할 수 있어 참고 자료로 구분했다.

## 규모와 원시 데이터

| 항목 | PR head 수치 | 근거 |
|---|---:|---|
| Go 패키지 디렉터리 | 206개 | full-go-inventory.json |
| `_test.go`가 아닌 Go 파일 | 882개 | exact-import-graph.json |
| `_test.go` 파일 | 352개 | 동일 |
| non-test 파일의 내부 import 연결 | 1,344개 | 동일 |
| test 파일의 내부 import 연결 | 620개 | 동일 |
| tree-sitter 대상 | Go 1,185개 + Python 2개 | pr-head/code-graph.json |
| tree-sitter 모듈 | 181개 | 동일 |
| tree-sitter 선언 타입 | 2,522개 | 동일 |

전체 Go inventory는 테스트 보조 프로그램·build 코드·생성 코드도 포함한다. `non-test`는 파일명 분류이며 전부 gwemix 바이너리에 포함된다는 뜻이 아니다. tree-sitter 결과는 testdata/tests/tools와 기본 제외 디렉터리를 뺐기 때문에 수치가 다르다. Solidity와 JavaScript는 이번 AST 도구의 지원 범위 밖이며 Go 바인딩과 실제 호출 경로를 중심으로 검토했다.

`cmd/gwemix`는 `geth`를 가리키는 심볼릭 링크다. 중복 집계를 피한 그래프에서는 `cmd/geth`가 해당 구현으로 나타난다. Makefile의 gwemix 대상은 이 링크 경로를 빌드한다(`sources/pr-head/Makefile:41`). `.new` 파일은 Go 소스가 아니다.

`.claude/docs/BUILD_SOURCE_FILES.md`도 참고용으로 보존했다. 이 자료는 2026-05-13의 다른 커밋과 darwin/arm64 기준이므로 현재 바이너리의 정확한 빌드 참여 목록으로 사용하지 않았다. Linux RocksDB 등 build tag별 실제 빌드 결과는 이번에 검증하지 않았다.

| 파일 | 용도 |
|---|---|
| [전체 Go inventory](../graph/full-go-inventory.json) | 파일·패키지·import·선언 목록 |
| [전체 경로 import 그래프](../graph/exact-import-graph.json) | 테스트와 non-test 연결, 외부 의존 |
| [Graphviz DOT](../graph/package-imports.dot) | 전체 그래프를 다른 도구에서 탐색할 입력. 변경 패키지는 붉은 배경 |
| [PR tree-sitter 그래프](../graph/pr-head/code-graph.json) | 타입 위치와 기본 모듈 통계 |
| [로컬 tree-sitter 그래프](../graph/local-head/code-graph.json) | 로컬 dev 기준 비교 자료 |

## 변경 패키지와 소비자

PR 변경은 `params`, `miner`, `core`, `eth/protocols/eth`에 모인다. miner는 core와 types를 사용하고, eth는 miner와 downloader를 조립한다. downloader는 인터페이스로 체인 삽입을 요청하므로 `downloader → core`가 import 그래프에 직접 나타나지 않아도 런타임으로는 영향을 준다. import 그래프를 완전한 호출 그래프로 해석하면 안 된다.

수동 호출 추적을 결합한 결과는 [pr-impact.md](pr-impact.md)에 있다. 특히 생산한 블록의 직접 저장, 일반 InsertChain, snap의 receipt 삽입은 같은 검증 함수를 항상 거치지 않는다. 이 차이가 N-003, N-004, N-014의 근거다.

빌드 태그별 실행 가능성, 모든 interface dispatch, 외부 SDK 내부 경로를 AST 그래프만으로 검증하지 않았다. 파싱 성공은 빌드·테스트 성공이 아니다.
