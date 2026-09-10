# 코드 건강도 검토 — AST 측정 (2026-09-10)

> **검토 문서.** 기준 커밋 `a983fc00`. 측정은 `scripts/inventory/code-graph`(패키지
> 그래프)와 임시 `go/ast` 분석기(함수 단위)로 뽑았다. 분석기는 저장소에 넣지 않았다 —
> 재현 방법은 §1 에 적었다.
>
> **후속 (2026-09-11): §4·§5 가 제안한 다섯 항목은 반영됐다.** 작업 리스트 §1s D2 에
> 무엇이 어떻게 끝났는지와, 그 과정에서 이 검토가 틀렸던 두 곳이 적혀 있다 —
> `deps(cmd)` 는 10곳이 아니라 **12곳이었고 세 갈래로 갈라져** 있었으며,
> `mcp` 의 `ArgString` 사본은 **아키텍처 래칫이 만들어낸 결과**라 core 로 직접 합칠 수
> 없었다(`app` 이 중계한다). 아래 본문은 측정 당시 그대로 둔다.

## 1. 측정 방법

패키지 그래프는 저장소의 도구를 그대로 썼다.

```sh
go run ./scripts/inventory/code-graph . > graph.json
```

함수 단위 지표는 그 도구가 내지 않는다. `go/ast`로 모든 비테스트 `.go` 파일을 파싱해
함수마다 길이·분기 수·중첩 깊이·인자 수를 뽑고, 본문에서 주석과 공백을 지운 뒤 해시로
묶어 완전 중복을 찾았다. 빌드는 하지 않는다.

같은 측정을 다시 하려면 `go/ast`로 `*ast.FuncDecl`을 순회하면서 `IfStmt`·`ForStmt`·
`RangeStmt`·`CaseClause`·`CommClause`를 세면 된다. 중복 판정은 시그니처 줄을 뺀 본문을
`\s+` 정규화한 뒤 sha256으로 묶었고, 4줄 미만과 정규화 후 40자 미만은 우연한 일치가
많아 제외했다.

## 2. 측정값

| 항목 | 값 |
|---|---|
| 패키지 | 70 (scripts 3 포함) |
| 간선 | 225 |
| 층 위반 | **0** |
| 총 라인 | 51,151 |
| 프로덕션 파일 / 테스트 파일 | 351 / 319 |
| 프로덕션 함수 | 1,830 |

함수 길이는 중앙값 11줄, 평균 16.8줄, 최대 191줄이다. 50줄을 넘는 것이 83건(4.5%),
80줄을 넘는 것이 19건이다. 분기 수는 중앙값 2, 최대 46이다. 중첩 깊이는 프로덕션 최대 5,
파일 길이는 중앙값 100줄이다.

`internal/arch`의 래칫 9개가 모두 통과하고 `golangci-lint`는 0건이다.

## 3. 좋은 부분

핵심 계층의 모양이 가장 좋은 신호다.

| 패키지 | fan-in | fan-out |
|---|---|---|
| `internal/core/node` | 18 | **0** |
| `internal/core/filestore` | 12 | **0** |
| `internal/core/rpc` | 8 | **0** |
| `cmd/chainbench/surface` | 9 | **0** |

많이 의존되면서 아무것도 의존하지 않는다. 프리미티브가 프리미티브답게 놓여 있다는 뜻이다.
반대로 `internal/app`은 in 16 / out 23 으로 허브인데, 유스케이스 계층이라 의도한 모양이다.

층 규칙이 사람 합의가 아니라 기계로 강제된다. 상향 의존 없음, 표면이 app 을 거침, MCP
직결 없음, 파일 쓰기 소유자 목록 준수까지 `internal/arch`가 검사한다.

대문자로 시작하는 에러 문자열 0건, `fmt.Println` 잔재 0건이다. 루프 안 `defer`가 하나
걸렸으나(`core/process/lifecycle.go:43`) 고루틴 본문 안이라 올바른 관용구였다.

## 4. 중복 — 네 갈래

본문 해시로 찾은 완전 중복은 7그룹이다. 성격이 갈린다.

### 4.1 고칠 가치가 있는 것

**`shellQuote` 4곳.** `core/process/remote.go:225`(이름만 `shq`),
`core/remote/files.go:73`, `chainsetup/steps_compose.go:1273`,
`resource/machine.go:347`. 바이트 단위로 같다.

셸 인용은 보안에 닿는 프리미티브다. 인용 규칙에 결함이 발견되면 네 곳을 다 찾아 고쳐야
하고, 한 곳을 놓치면 그 경로만 뚫린다. `core`에 하나 두고 나머지가 부르는 것이 맞다.

**`ArgString`/`ArgInt` 2곳.** `core/registry/args.go:9,20`이 공개 버전이고
`mcp/tool.go:42,74`가 비공개 사본이다. 표면이 core 프리미티브를 복제했다.

**`hostOf(rpc string)` 2곳.** `mcp/network_tools.go:165`와
`cmd/chainbench/networkcmd/network.go:210`. 표면 두 곳이 같은 6줄을 각자 갖고 있다.
(`chainsetup/discover.go`의 동명 함수는 인자와 본문이 달라 이름만 겹친다.)

**`deps(cmd)` 10곳.** 6줄짜리가 `cmd/chainbench/*` 전역에 퍼져 있다. 그런데
`cmd/chainbench/surface`라는 공용 패키지가 이미 있고 fan-in 이 9다. 자리가 있는데 안 쓴다.

### 4.2 두는 편이 맞는 것

`chains/{stablenet,wbft,wemix}`의 `init()` 3개는 blank import 등록이라 프로젝트 Go
지침이 명시적으로 허용하는 예외다. `poa.Family.SupportsRole`과
`wbft.Family.SupportsRole`은 두 합의 패밀리가 우연히 같은 답을 낼 뿐이라, 합치면 오히려
결합이 생긴다.

### 4.3 공통점

네 갈래 모두 **표면이나 상위 계층이 프리미티브를 각자 다시 만든** 모양이다. 이 프로젝트가
스스로 금지한 바로 그 패턴이라, 개별 중복보다 이 경향이 더 중요하다.

## 5. 큰 함수

| 줄 | 분기 | 깊이 | 위치 |
|---|---|---|---|
| 191 | 24 | 3 | `internal/chainsetup/verbs_up.go:130` `netUpFrom` |
| 182 | 46 | 3 | `internal/dsl/spec_v2.go:357` `lowerCase` |
| 175 | 30 | 3 | `internal/testengine/compose.go:86` `compositionOf` |
| 172 | 33 | 4 | `internal/arch/surface.go:230` `Entries` |
| 171 | 38 | 3 | `internal/arch/reach.go:73` `Reach` |

`netUpFrom`은 191줄 중 31줄이 주석이고 86줄이 스텝별 클로저를 담은 `map` 리터럴이다.
얽힌 제어 흐름이 아니라 표라서 깊이가 3에 머문다. 그래도 세 가지 일을 한다 — 입력 검증,
스텝 표 구성, 재사용 훅이다.

**이 함수는 172줄이었고 모니터링 수정 트랙에서 191줄로 커졌다.** 이미 과대한 함수에
가드를 더 붙였고 추출은 하지 않았다.

`lowerCase`는 분기가 46개로 가장 많다. DSL v2 를 내부 형태로 낮추는 함수라 케이스가 많은
것은 자연스럽지만, 46 은 한 함수가 감당할 수를 넘는다.

`arch`의 두 함수는 래칫 구현이라 성격이 다르다. 도구 코드이고 소비자가 테스트뿐이다.

인자가 5개를 넘는 함수가 32건 있다. 최대는 9개
(`accounts/tamper.go:17`, `consensus/poa/executor.go:293`, `dsl/interp/run.go:171`)다.

## 6. 문서화 공백

**패키지 67개 중 17개에 package doc 이 없다.** 프로젝트 Go 지침이 `doc.go`로 package doc
을 쓰라고 명시하는데 지켜지지 않은 곳이다.

가장 큰 구멍이 `internal/testhelper`(3,671줄, 11파일)다. 패키지가 무엇인지 어디에도 적혀
있지 않다. `internal/core/keyring/store`(1,294줄)와 `internal/dsl/interp`(928줄)도 없다.
나머지는 얇은 `cmd/*` 명령 패키지라 덜 급하다.

이름 규칙(`util`/`common`/`helper`)으로 걸린 두 패키지는 확인해 보니 오탐이었다.
`internal/chains/common`의 "common"은 실제 체인 프로젝트 이름이고, `internal/testhelper`는
파일마다 관심사가 갈려 있어 덤핑 그라운드가 아니다. 다만 이름이 내용을 말해 주지 않는 것은
사실이고, package doc 이 없어서 더 그렇다.

## 7. 판단이 갈리는 것

**`panic` 5곳.** 넷은 `init()` 시점의 프로그래머 오류다(플러그인 중복 등록, embed 파싱
실패). 정당하다. 하나가 애매하다 — `consensus/wbft/extradata.go:118`의 `rlpEncode`가
미지원 타입에 panic 한다. Go 타입 스위치의 default 라 외부 입력으로는 닿지 않지만
라이브러리 인코딩 경로다.

**`time.Now()` 직접 사용.** 지침은 시계를 주입하라고 하고 `chainsetup.Workspace`는 실제로
`Clock`을 받는다. 그런데 `consensus/{poa,upgrade}`의 폴링 데드라인은 직접 쓴다. 로직이
아니라 벽시계 타임아웃이라 재현성 영향은 적지만, 같은 저장소 안에서 규칙이 갈린다.

**테스트 없는 프로덕션 패키지 4개**(150줄 초과 기준):
`cmd/chainbench/networkcmd`, `internal/chains/stablenet`,
`internal/core/keyring/derive`, `internal/core/keyring/operation`.
뒤의 둘이 키를 다루는 곳이라 눈에 걸린다.

## 8. 종합과 제안 순서

구조는 clean 하다. 층 위반 0, 래칫 통과, 린터 0건, 함수 길이 분포가 건강하고 핵심 계층이
fan-out 0 이다.

깨끗하지 않은 곳은 **중복과 문서** 두 갈래이고, 중복에는 §4.3 의 경향이 있다.

착수한다면 이 순서를 제안한다.

1. **`shellQuote` 통합** — 보안에 닿고 사본이 가장 많다.
2. **`ArgString`/`ArgInt` 사본 제거** — core 에 이미 공개돼 있어 지우기만 하면 된다.
3. **`deps` 10개를 `cmd/chainbench/surface` 로** — 자리가 이미 있어 비용이 거의 없다.
4. **`testhelper` package doc** — 가장 큰 문서 공백이다.

`netUpFrom` 분해는 별개로 다루는 편이 낫다. 재사용 훅이 스텝 루프와 얽혀 있어 기계적
추출이 아니고, 다른 변경에 얹으면 검토 범위가 흐려진다.

## 9. 이 검토의 한계

정적 측정만 했다. 실행하거나 프로파일링하지 않았으므로 성능과 동시성 결함은 이 문서가
말하지 않는다. 중복 판정은 **완전 일치**만 본다 — 이름이나 상수만 다른 준중복은 잡히지
않으므로, 실제 중복은 여기 적힌 것보다 많을 수 있다.

`-race`와 `goleak`은 기존 테스트가 이미 돌리고 있으나, 이 검토가 그 결과를 새로 확인한
것은 아니다.
