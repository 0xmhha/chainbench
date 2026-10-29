# tests — 테스트 자산

이 디렉터리는 chainbench 의 **테스트 자산**을 담는다. 무엇이 어디에 있느냐는
"무엇을 검증하는가" 가 아니라 **"어떻게 실행되는가"** 로 갈린다.

| 경로 | 티어 | 무엇인가 | 어떻게 도는가 |
|---|---|---|---|
| [`tc/`](tc/README.md) | **DSL 케이스** | `schemaVersion: "2"` 문서 하나가 체인 구성과 검증을 함께 선언한다. | `chainbench run --workspace-dir <ws> tests/tc/<경로>`(선언한 네트워크를 구성), `chainbench run --attach --workspace-dir <ws> <케이스>`(그 작업공간이 이미 띄운 네트워크), `chainbench run --rpc <url> --chain <id> <케이스>`(살아 있는 네트워크) |
| [`e2e/`](e2e/README.md) | **라이브 Go e2e** | 실제 체인 바이너리로 네트워크를 띄워 끝까지 몬다. `//go:build e2e` 로 게이트돼 `go test ./...` 에서는 컴파일되지 않는다. | `go test -tags e2e ./tests/e2e/` + `GSTABLE_BIN`·`WBFT_BIN`·`WEMIX_BIN` |
| [`repro/`](repro/README.md) | **재현 런북** | CI 에서 돌 수 없는 회귀 시나리오를 빌드된 CLI 로 재현하는 셸 스크립트. | 저장소 루트에서 직접 실행 (바이너리가 없으면 exit 2) |
| [`webui/`](webui/README.md) | **Web UI 테스트** | 대시보드 SPA 와 `/api/v1` 을 검사하는 Node 테스트(`*.test.mjs`)와 Python 테스트(`test_*.py`), 그리고 실제 바이너리로 WEB-NN 수용 기준을 확인하는 라이브 러너다. | `node --test tests/webui/*.test.mjs`, `python3 -m unittest discover -s tests/webui -p 'test_*.py'`, 라이브 증명은 `tests/webui/verify.sh --criterion WEB-NN --require-live` |

단위 테스트는 여기 없다. 각 패키지 옆의 `*_test.go` 에 있고 `go test ./...` 가
돌린다. `internal/arch` 의 아키텍처 래칫도 그 축이다.

## 새 테스트는 어디에 쓰나

1. **먼저 DSL 케이스(`tc/`)를 시도한다.** 어휘로 표현되면 문서 한 장이 끝이고, CLI 와
   MCP 양쪽에서 같은 것이 돈다. 작성법은
   [`../docs/guide/dsl-authoring.md`](../docs/guide/dsl-authoring.md) 에 있다.
2. **DSL 어휘로 표현이 안 되면** 그것 자체가 발견이다. 어휘를 늘릴지
   (`internal/testhelper/builtins.go` 의 `Register` 에 action·assertion 을 더한다)
   아니면 Go e2e 로 갈지를 먼저 정한다.
3. **바이너리 교체나 하드포크 핸드오프처럼 프로세스 생애주기를 직접 조작해야 하면**
   `e2e/` 의 게이트된 Go 테스트다.

케이스를 묶어서 돌리려면 그 묶음이 같은 네트워크를 선언해야 한다. 구성이 갈리는 묶음은
실행기가 거부하므로(`internal/testengine` 의 `sameComposition`) 조용히 틀린 네트워크에서
도는 일은 없다.

## 테스트가 쓰는 키

테스트 자산은 개인키를 인라인으로 담지 않는다. 자금이 있는 계정이 필요하면
`presets/keys/` 의 픽스처를 참조한다. 예를 들어 `tests/e2e` 의 송금은 node1 키를,
`tests/repro/wemix-chain.sh` 는 `presets/keys` 디렉터리 전체를 가리킨다.

그 키들은 전부 커밋된 **테스트 픽스처**다. 시크릿 스캐너가 보고하는 것은 예상된
동작이며, 허용 범위와 그 이유는 [루트 README](../README.md) 의 주의 블록과
[`presets/keys/README.md`](../presets/keys/README.md) 에 적혀 있다.

새 케이스에 자금 계정이 필요하면 그 체인 preset 의 alloc 에서 가져온다. 새 키를 만들어
커밋하지 않는다. **알려진 픽스처가 아닌 커밋된 키는 diff 를 읽는 사람에게 실제 유출과
구분되지 않는다.**
