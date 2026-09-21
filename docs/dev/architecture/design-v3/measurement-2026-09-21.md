# 리팩토링 설계 v3 — 실측 (2026-09-21) — [측정]

> **[측정]** 이 문서는 잰 것만 적는다. 방향과 제안은 `direction.md` 에 있다.
> 닫힌 세 계획([[consolidation-plan]]·[[module-plan]]·`refactoring-proposal/`)의 수치를
> 하나도 가져오지 않았다. 전부 이날 다시 쟀다.
>
> 재는 법: `go run docs/dev/codegraph/main.go .` (AST, `go/parser`+`go/ast`).
> 산출물은 [`codegraph.json`](../../codegraph/codegraph.json)·[`pipeline.md`](../../codegraph/pipeline.md).

## 1. 전체 모양

패키지 **67개**(`internal` 48 + `cmd` 19), 크로스패키지 호출 엣지 **1,610개**.

가장 많이 의존받는 것과 가장 많이 의존하는 것:

| 패키지 | fanIn | fanOut | 비-테스트 줄 | exported 타입 | exported 함수 |
|---|---|---|---|---|---|
| `internal/app` | 17 | **22** | 2,604 | **122** | 118 |
| `internal/chainsetup` | 2 | **20** | **8,406** | 68 | **169** |
| `internal/core/node` | **18** | 0 | — | 17 | 47 |
| `internal/core/registry` | 16 | 1 | — | 24 | 32 |
| `internal/core/filestore` | 12 | 0 | — | — | — |
| `internal/resource` | 6 | 4 | 3,094 | 41 | 92 |

표면 하나가 끌고 오는 패키지(전이 import): `cli/suitecmd` 40 · `cli/chaincmd` 38 · `mcp` 36 ·
`app` 35 · `testengine` 32 · `chainsetup` 23.

## 2. 레이어와 호출 관계 — 결함 없음

**여기는 문제가 아니다.** 세 가지를 확인했다.

- `layers.md` §3 이 실제 패키지 **48개를 모두** 배치한다(`arch.TestEveryPackageIsPlaced` 통과).
- 표가 이름 붙인 것 중 **없는 패키지는 0개**다. 같은 테스트의 역방향 검사가 본다.
- **상향 의존 0건**(`arch.TestNoUpwardDependency` 통과). 낮은 층이 높은 층을 알지 않는다.

Go 에서 호출은 import 를 수반하므로, import 층이 지켜지면 호출 관계도 그 안에 갇힌다.
**따라서 "레이어 구조와 호출 관계" 자체는 이미 명확하다.** 읽기 어려운 원인은 층이 아니라
아래 §4 의 크기와 §3 의 낱말이다.

> 처음에 이 절에 "layers.md 가 부르는 18개가 없다" 고 적었다가 지웠다. `readPlacement` 는
> 표의 **첫 칸만** 읽는데 나는 산문의 backtick 까지 긁었다. 흡수되어 사라진 모듈 이름이
> 설명 문장에 남아 있는 것을 배치로 오인한 것이고, 코드도 문서도 멀쩡했다.

## 3. 한 낱말이 다섯 가지를 가리킨다 — `preset` / `profile`

가장 큰 가독성 결함은 여기다. **`preset` 이 다섯 가지**를 가리킨다.

| 무엇 | 어디 | 부르는 이름 |
|---|---|---|
| 하드포크 골든 설정 파일 | `presets/chain/*.yaml` | 디렉터리는 **preset**, Go 타입은 `upgrade.Profile`, 읽는 함수는 `LoadProfile` |
| 키 픽스처 (노드키·계정 45파일) | `presets/keys/` | **preset** |
| 키셋 Go 타입 | `internal/core/keyring/preset.go` | `keyring.Preset` |
| DSL 의 env 선언 | `tests/tc/env/*.env.json` | 메인넷 워크리스트 P1~P3 이 **preset** 이라 부른다 |
| 원격 접속 설정 | `profiles/` | **profile** |

**같은 것을 두 이름으로 부르는 자리가 스키마에 살아 있다.** `dsl.UpgradeV2` 는
`Preset`(디렉터리 아래 이름)과 `Profile`(경로)을 **둘 다** 필드로 갖고, 둘을 같이 쓰면
거부한다(`spec_v2.go:395`). 그런데 저장소의 정의서 중 `profile` 을 쓰는 것은 **0개**,
`preset` 을 쓰는 것이 4개다. **`Profile` 필드는 소비자가 없다.**

경로도 어긋나 있다. JSON 스키마는 이 파일을 *"golden upgrade profile (profiles/\*.yaml)"* 이라
설명하는데(`v2.schema.json:142`) 실제 위치는 `presets/chain/*.yaml` 이다.
`docs/dev/chain-handover-2026-09-12.md:379` 도 `profiles/wemix-upgrade-15.yaml` 로 링크한다 —
**그 경로에 그 파일은 없다.**

이름 겹침 래칫(`arch.TestNamesDoNotCollide`)은 이것을 잡지 못한다. 그 래칫은 **Go 선언 이름**만
보고, 여기 겹침은 디렉터리 이름 · JSON 필드 이름 · 문서 낱말에 걸쳐 있다.

## 4. 쓰지 않는 자리

| 자리 | 내용물 | 코드에서 부르는가 | `.gitignore` 줄 |
|---|---|---|---|
| `state/` | `.gitkeep` 2개뿐 | **0건** (`state/pids.json`·`current-profile*`·`state/results`·`state/remotes.json`·`current-remote`·`local-config.yaml`·`failures/` 전부 아무도 쓰지 않는다) | **9줄** |
| `profiles/` | `remote-example.yaml` + `custom/.gitkeep` | **0건.** 자기 자신과 `README.md`·`CONTRIBUTING.md` 만 가리킨다 | 4줄 |

두 자리 모두 **문서는 살아 있는 것처럼 적고 있다** — `README.md:306` 이 `profiles/` 를
"remote-chain connection profiles" 로, `CONTRIBUTING.md:89·111` 이 "여기에 YAML 을 만들라" 고
적는다.

## 5. 상태 주도 — 있는 것과 없는 것

**있다.** `chainsetup/steps_compose.go:413` 의 `composeNeeds` 가 조립 순서를 **한 곳에 선언**하고,
`require(step)` 이 그 앞 단계가 실제로 **기록되었는지** 본다. 상태 필드가 아니라 기록된 단계를
읽는 것이 핵심이다 — 필드는 다른 것이 채웠을 수 있고, 반쯤 돈 단계는 "조립된 것처럼 보이는
상태" 를 남긴다. 주석이 왜 이렇게 됐는지까지 적고 있다.

```
place → new        keys  → new       genesis → place
config → place,keys        build → place,keys
deploy → place,genesis,config        init → deploy        start → init
```

**없다.** 이 표가 덮는 것은 **조립 8단계**이고 `require` 를 실제로 지나는 것은 **6개**다
(`build`·`config`·`deploy`·`genesis`·`init`·`start`). 그런데 `Workspace` 가 내놓는 동사는
**38개**다:

```
Acquire Allocate CheckBaseline Compare Config CrossFork Dir Genesis Hardfork Have Health
Init Keys LaunchOpts Lock LogExcerpt Logs MarkStepFailed Netmap New NodeSet ObserveBaseline
Preflight Provision Reconcile Restart Retarget Rm RPCHost Save SetDriver SetEnv Start
StartNode Stop StopNode SwapNode VerifyValidators
```

즉 **선행 조건을 선언하는 동사는 38 중 6**이다. 나머지 32개 — `Stop`·`Restart`·`SwapNode`·
`Rm`·`Hardfork`·`CrossFork`·`Health`·`Logs`·`Resume` 등 — 는 각자 손으로 검사하거나 검사하지
않는다. `composeNeeds` 의 주석이 적은 옛 결함("각 단계가 자기가 필요한 필드를 손으로 검사하고
자기 메시지를 썼다")이 **조립 밖에서는 그대로 남아 있다.**

## 6. 이름 — 이미 관리되는 것과 아닌 것

`arch.TestNamesDoNotCollide` 가 Go 선언 이름을 두 목록으로 관리한다. `nameShared` 는
**일부러 겹치는 것**(동사 `Parse`·`Build`·`Generate`, 구조적 역할 `Config`·`Options`·`Deps`)이고
`nameCollisionDebt` 는 **갚아야 할 것**(`Runner` — poa 는 명령 실행, process 는 원격 셸;
`Handler`·`NewServer` — registry/mcp, dashboard/mcp)이다. 이 목록은 **줄기만 한다.**

관리 밖에 있는 것이 §3 의 `preset`/`profile` 이다. Go 이름이 아니라 **디렉터리·JSON 필드·문서
낱말**에 걸쳐 있어 래칫의 사정거리 밖이다.
