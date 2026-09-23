# 그래프 자료 — 무엇을 언제 잰 것인가

각 파일이 **어느 커밋의 트리를 잰 것인지** 적는다. 이 표가 없어서 한 번 틀렸다:
2026-09-22 에 커밋된 `code-graph.json` 은 `internal/chainsetup` 을 103 파일로 적고 있었는데,
그 그래프를 커밋한 시점(`47a27257`)의 트리에는 120 파일이 있었다. main 을 잰 그래프가
브랜치 끝의 그래프처럼 인용되고 있었다. 옛 측정은
[`../graph-snapshots/a5386b0e/`](../graph-snapshots/a5386b0e/) 로 옮겼다.

| 파일 | 잰 커밋 | 도구 |
|---|---|---|
| `code-graph.json` · `module-imports.tsv` · `module-graph.mmd` · `graph-report.md` | `45c6cb36` | tree-sitter, depth 2 |
| `code-graph-d3.json` · `module-imports-d3.tsv` · `module-graph-d3.mmd` · `core-graph-d3.mmd` · `graph-report-d3.md` | `45c6cb36` | tree-sitter, depth 3 |
| `callgraph-state.json` | `a5386b0e` (main) | go/types 호출 그래프 — **생성 도구가 저장소에 없다** |
| `lifecycle-transitions.json` · `lifecycle-transitions.mmd` | `a5386b0e` (main) | 전이표 추출 — **제거된 옛 lifecycle 을 그린 것** |

뒤의 둘은 갱신하지 않았다. `callgraph-state.json` 은 재생성 절차가 남아 있지 않고,
`lifecycle-transitions.*` 은 그리는 대상이 HSM 리팩토링에서 사라졌다 — 그 자리는
[`../18-hsm-state-diagrams-2026-09-22.md`](../18-hsm-state-diagrams-2026-09-22.md) 가 대신한다.

## 다시 뽑는 법

codemine 플러그인의 venv 에서 돌린다. grammar 버전이 핀으로 고정돼 있어서 같은 트리는
어느 기기에서든 같은 수를 낸다.

```sh
P=~/.claude/plugins/cache/0xmhha-devkit/codemine/<버전>
bash "$P/scripts/setup_env.sh"

# depth 2
"$P/.venv/bin/python" "$P/scripts/extract_graph.py" --repo . --out <out> --module-depth 2
"$P/.venv/bin/python" "$P/scripts/graph_report.py" --graph <out>/code-graph.json --top 30

# depth 3 — internal/core 가 depth 2 에서 247 파일 한 덩어리로 뭉치므로 필요하다
"$P/.venv/bin/python" "$P/scripts/extract_graph.py" --repo . --out <out> --module-depth 3
```

`extract_graph.py` 가 적는 `repo` 필드는 돌린 기기의 절대 경로다. 커밋할 때 `.` 로 바꾼다 —
남의 홈 디렉터리 경로가 저장소에 들어가는 것을 막고, 어느 기기에서 뽑았든 파일이 같아진다.

`.mmd` 는 앞에 mermaid frontmatter(`---`)를 두지 않는다. `lint_mermaid.py` 가 첫 줄에서
다이어그램 종류를 읽기 때문에 frontmatter 가 있으면 검사에 걸린다.

## 이번 측정에서 나온 것 (`45c6cb36`)

- 852 소스 파일(Go 830 · Python 22), 26 모듈(depth 2) / 75 모듈(depth 3), 774 타입,
  내부 import 간선 49개(depth 2) / 304개(depth 3).
- `internal/core/statemachine` 8 파일 · 1,361 줄. `internal/core/lifecycle` 은 4 파일 ·
  846 줄로 줄었고 `chainsetup` 에서 17 간선, `app` 에서 4 간선이 아직 남아 있다.
- 저장소 자체 도구(`go run ./scripts/inventory/code-graph .`)는 layer 위반 **6건**을 본다 —
  `internal/core` 의 여섯 패키지가 `internal/preset` 을 import 한다. **main 에도 그대로
  있다**(`git grep -l internal/preset main -- 'internal/core/*'`). 이 브랜치가 만든 것이
  아니다. [`../../../dev/architecture/code-graph.md`](../../../dev/architecture/code-graph.md)
  §2 는 아직 `violations: null` 이라고 적고 있으므로 그 문서는 다시 뽑아야 한다.
