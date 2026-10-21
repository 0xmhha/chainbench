# 측정과 재생성

측정일: 2026-10-13. 커밋: `662fff77382eb031df687d371a8bc8f63cad581f`.
전체 범위와 오류 1개는 [01-code-graph.md](../01-code-graph.md)에 설명했다.

| 파일 | 의미 |
|---|---|
| `code-graph.json`, `module-imports.tsv`, `graph-report.md` | codemine 0.5.2의 depth 3 모듈/타입 통계. Go와 Python만 해당한다. |
| `go-package-graph.json` | go/parser로 주요 Go 코드 트리의 패키지와 심볼을 측정했다. |
| `source-graph.json` | 모든 추적 소스와 화면의 파일별 AST 결과. 본문 리터럴은 저장하지 않는다. |
| `web-ast.json` | Svelte 전체 AST에서 추출한 선언, 컴포넌트, import, 호출과 권한 조건. |
| `summary.json` | 원시 그래프에서 계산한 측정값. |
| `go-packages.dot`, `go-packages.svg` | 주요 Go 코드 트리의 모든 패키지 import 관계. |
| `web-files.dot`, `web-files.svg` | 모든 web 소스 파일과 해석된 파일 import 관계. |
| `git-timeline.md` | git_timeline.py가 추출한 저장소 이력. |

도구가 반환하는 패키지 간선은 import 방향이다. 함수 값의 실제 대상과 HTTP 호출은 별도로 코드에서 검토했다. 구문상 호출 목록을 완전히 해석된 런타임 호출 그래프라고 읽지 않는다.

## 재생성

codemine 환경에는 tree-sitter 0.26.0, go 0.25.0, python 0.25.0을 사용했다. 보완 문법은 [공식 Bash 문법 0.25.1](https://pypi.org/project/tree-sitter-bash/0.25.1/)과 [공식 JavaScript 문법 0.25.0](https://pypi.org/project/tree-sitter-javascript/0.25.0/)이다. Svelte는 `web/package-lock.json`에 잠긴 5.56.8이다. 렌더러는 `@viz-js/viz` 3.25.0을 사용했다. 의존성은 임시 위치에서 사용했고 저장소의 의존성 설정은 바꾸지 않았다.

아래 환경 변수는 각 도구를 설치한 경로를 가리킨다. `STRUCTURE_ROOT`는 codemine 플러그인 루트, `AST_PYTHON`은 준비된 Python 가상환경, `GRAMMAR_DIR`은 Bash/JS 문법 위치, `SVELTE_COMPILER`와 `VIZ_MODULE`은 해당 모듈 파일이다. 모든 명령은 저장소 루트에서 실행한다.

```sh
OUT=docs/research/chainbench/analyses/web-account-flag-2026-10-13
"$AST_PYTHON" "$STRUCTURE_ROOT/scripts/extract_graph.py" --repo . --out "$OUT/graph" --module-depth 3
"$AST_PYTHON" "$STRUCTURE_ROOT/scripts/graph_report.py" --graph "$OUT/graph/code-graph.json" --top 20 > "$OUT/graph/graph-report.md"
go run ./scripts/inventory/code-graph -symbols . > "$OUT/graph/go-package-graph.json"
node "$OUT/tools/parse-web-ast.cjs" . "$SVELTE_COMPILER" "$OUT/graph/web-ast.json"
"$AST_PYTHON" "$OUT/tools/parse-source-ast.py" --repo . --grammar-dir "$GRAMMAR_DIR" --web-ast "$OUT/graph/web-ast.json" --out "$OUT/graph/source-graph.json"
node "$OUT/tools/render-graphs.cjs" "$OUT/graph" "$VIZ_MODULE"
python3 "$STRUCTURE_ROOT/scripts/git_timeline.py" --repo . --keywords web,auth,dashboard > "$OUT/graph/git-timeline.md"
python3 "$STRUCTURE_ROOT/scripts/verify_citations.py" --docs "$OUT" --repo .
python3 "$STRUCTURE_ROOT/scripts/lint_mermaid.py" --docs "$OUT"
```

기본 추출기는 repo를 절대 경로로 기록하므로 저장할 때 `repo`를 `.`으로 바꾼다. 파일별 SHA-256과 commit은 `source-graph.json`에 있다.

`verify_citations.py`는 Svelte 확장자를 지원하지 않는다. 이번 Svelte 인용은 파일의 실재와 줄 수를 별도로 검사했고, 주장을 뒷받침하는 내용도 코드에서 확인했다. Mermaid 검증은 구조 검사이며 브라우저에서 그려 보는 테스트는 아니다.
