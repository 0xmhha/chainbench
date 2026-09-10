# chainbench test code (requirement #16)

> **[가이드]** 검증 기준 2026-09-11 (`3cf992db`). 이 README 는 이전에 은퇴한 Go
> **testkit** 모델(`internal/testkit` · `testrun` 페이즈 · `testkit.Case` 등록 ·
> `ChainCompat`/`RequiresCaps` 게이팅 · coverage 지표)을 서술하고 있었다. 그 모델은
> 2026-09-01 (R5) 에 삭제됐다 — [`../docs/dev/archive/legacy-retirement-plan.md`](../docs/dev/archive/legacy-retirement-plan.md).
> 아래는 지금의 트리다.

이 디렉터리는 chainbench 의 **테스트 자산**을 담는다. 테스트가 어디에 있느냐는
"무엇을 검증하는가" 가 아니라 **"어떻게 실행되는가"** 로 갈린다.

| 경로 | 티어 | 무엇인가 | 어떻게 도는가 |
|---|---|---|---|
| [`tc/`](tc/README.md) | **DSL 케이스** | `schemaVersion: "2"` 문서 하나가 체인 구성과 검증을 함께 선언한다. 레거시 셸 스위트의 구조를 그대로 따른다. | `chainbench run tests/tc/<dir>` (compose) 또는 `chainbench run --attach --rpc <url>` |
| [`e2e/`](e2e/README.md) | **라이브 Go e2e** | 실제 체인 바이너리로 네트워크를 띄워 끝까지 몬다. `//go:build e2e` 로 게이트돼 `go test ./...` 에서는 컴파일되지 않는다. | `go test -tags e2e ./tests/e2e/` + `GSTABLE_BIN`/`WBFT_BIN`/`WEMIX_BIN` |
| [`repro/`](repro/README.md) | **재현 런북** | CI 에서 돌 수 없는 회귀 시나리오를 빌드된 CLI 로 재현하는 셸 스크립트. 대부분 `e2e/` 로 이관됐고 남은 것이 여기 있다. | 저장소 루트에서 직접 실행 (바이너리 없으면 exit 2) |
| [`env/`](env/README.md) | **환경 픽스처** | 공개 테스트용 `.env` 와 비밀 취급 경계 예시. 시크릿 스캐너가 공개 허용목록으로 참조한다. | 소비되는 자료, 실행되지 않는다 |

단위 테스트는 여기 없다 — 각 패키지 옆의 `*_test.go` 에 있고 `go test ./...` 가
돌린다. `internal/arch` 의 아키텍처 래칫 9개도 그 축이다.

## 새 테스트는 어디에 쓰나

1. **먼저 DSL 케이스(`tc/`)를 시도한다.** 어휘로 표현되면 문서 한 장이 끝이고, CLI 와
   MCP 양쪽에서 같은 것이 돈다. 작성법: [`../docs/guide/dsl-authoring.md`](../docs/guide/dsl-authoring.md).
2. **DSL 어휘로 표현이 안 되면** 그것 자체가 발견이다 — 어휘를 늘릴지
   (`internal/testhelper` 의 `RegisterAction`/`RegisterAssertion`) 아니면 Go e2e 로 갈지
   를 먼저 정한다.
3. **바이너리 스왑·하드포크 핸드오프처럼 프로세스 생애주기를 조작해야 하면** `e2e/`
   의 게이트된 Go 테스트다.

케이스를 함께 돌리려면 `env` 가 같아야 한다. 실행기가 구성이 갈리는 묶음을 거부하므로
(`sameComposition`) 조용히 틀린 네트워크에서 도는 일은 없다.

## Keys in test source — TEST FIXTURE ONLY

테스트 자산은 **평문 개인키를 인라인으로** 담는다(`faucetKeyHex` 류). 의도된 것이다:
전송을 검증하려면 키 관리 단계 없이 자금이 있어야 하고, 그 값이 실행마다 재현돼야 한다.

이런 키는 전부 `keys/preset/` 의 픽스처이며, faucet 키는 모든 geth 포크에 공개된
upstream go-ethereum 테스트 키다. **이 키나 여기서 파생된 주소를 공용 네트워크에 절대
올리지 않는다.** 이 디렉터리를 훑는 시크릿 스캐너는 이것들을 보고하며, 그 보고는
예상된 것이다 — [루트 README](../README.md) 의 주의 블록 참조.

새 케이스에 자금 계정이 필요하면 그 체인 preset 의 alloc 에서 가져온다. 새 키를
만들어 커밋하지 않는다 — **알려진 픽스처가 아닌 커밋된 키는 diff 를 읽는 사람에게 실제
유출과 구분되지 않는다.**
