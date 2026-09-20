# 체인을 실행 시점 차원으로 — 공통 TC 한 벌이 세 체인에 돈다 — [제안]

> **[제안] (2026-09-20).** 결정이 아니다. 아래 수치는 이날 실측이고, 인용 전에 다시 잰다.
> 이 문서는 닫힌 세 계획([[consolidation-plan]]·[[module-plan]]·`refactoring-proposal/`)을
> **이어받지 않는다.** 전제는 코드 실측과 아래 §1 의 역할 경계뿐이다.

## 1. 전제 — 모듈의 역할 경계

새 설계가 지켜야 하는 경계다. 이 셋은 합치지 않는다.

- **`app`** — CLI 와 MCP 로 들어오는 **진입점**이다. chainbench 를 데몬으로 만들더라도
  `main` 이 들어가는 자리가 `app` 이다.
- **`chainsetup`** — chainbench 로 체인을 구성·설정해서 **도는 체인을 만드는 일 전반**을
  책임진다.
- **`testengine`** — **테스트 수행 lifecycle** 을 책임진다: 테스트 환경을 구성하고, 테스트를
  수행하고, 결과 레포트를 쓴다.
- `testengine` 이 테스트 환경을 구성할 때 **그 구성을 담당하는 것이 `chainsetup`** 이다.
  둘 사이의 의존은 이 방향이고, 의도된 것이다.

공통 TC 도구 기능 8개가 떨어지는 자리는 이 셋이 아니다. 다섯이 `testhelper`(로컬 서명 전송 ·
거부 판정 · 기능 사전 확인 · 동기화 경로 관찰)와 `dsl`(준비물 공유), 둘이 `testengine` 의 레포트
책임(실행 결과 기록 · 세 체인 일괄 실행), 하나가 attach 경로(외부 주소 프로필)다.

## 2. 무엇을 바꾸나

**체인은 env 문서의 값이 아니라 실행 시점의 차원이다.**

체인 고유 정보는 이미 env 밖에 있다 — `internal/chains/<id>/manifest.json` 이 `binary`·
`chain_id`·`network_id`·`consensus`·`consensus_family`·`dialect`·`genesis`·`capabilities`·
`system_contracts`·`tx_types` 를 들고 있다. env 의 `chain` 필드는 **그 매니페스트를 가리키는
포인터 하나**다.

실측이 그것을 보인다: `tests/tc/env/{stablenet,wbft,wemix}-bp4.env.json` 세 파일은 `id` 와
`chain` 두 줄을 빼면 같다. `keys` 는 셋 다 `{"nodekeys":{"ref":"keys/preset","source":"keyPreset"}}`,
`topology` 는 셋 다 `{"bp":4}` 다. `stablenet-bp4` 에는 `binaries` 키가 아예 없고, 바이너리는
매니페스트의 `binary: gstable` 에서 나온다.

그래서 필요한 변경은 **포인터를 실행 시점에 덮는 것 하나**다. 새 스키마도, 새 문서 종류도,
파일 이름 규칙의 계약화도 필요 없다. 부수 효과로 체인별 env 중복이 지워진다 — `wbft-bp4` 와
`wemix-bp4` 는 `bp4` 하나로 줄고, 모양 19종을 세 벌씩 채우는 일이 사라진다.

**플래그 이름은 `--chain` 을 쓰지 않는다.** 그 이름은 attach 경로에서 이미
"specs 가 선언한 것과 **일치해야 한다**" 는 검사로 쓰인다. 같은 플래그가 경로에 따라 검사이기도
하고 덮어쓰기이기도 하면 설명할 수 없다.

## 3. 판정 규칙 — 손으로 쓴 목록이 아니라 유도된다

> **env override 가 체인 고유 이름을 부르면, 그 케이스의 `requires` 가 그것을 이름으로
> 선언해야 한다.**

체인 고유 이름은 포크 이름 · 엔진 섹션 · 시스템 컨트랙트 · upgrade preset 이다. 이 규칙이
있으면 거절 목록을 손으로 유지하지 않아도 되고, 규칙을 어긴 케이스가 **들어오는 시점에** 잡힌다.

세 체인이 아는 포크(`manifest.genesis.hardforks`)와 기본 genesis 템플릿이 켜는 것:

| 포크 | stablenet | wbft | wemix |
|---|---|---|---|
| `istanbul` · `applepie` | 안다 (템플릿은 안 켬) | 안다, 템플릿 0 | 안다, 템플릿 0 |
| `boho` | 안다 (템플릿은 안 켬) | — | — |
| `anzeon` (엔진 섹션) | 템플릿에 있다 | — | — |
| `pangyo` · `brioche` | — | 템플릿 0 | 템플릿 0 |
| `croissant` | — | 템플릿 0 + 엔진 섹션 | — |

## 4. 28건 판정 결과 (2026-09-20 전수)

케이스 209개 중 `genesis`·`hardforks`·`upgrade`·`accounts` 를 인라인 override 로 든 것이
28개다. 셋으로 갈린다.

### 거절 — 3건

override 가 체인 고유 이름을 부르는데 `requires` 가 그것을 말하지 않는다. 체인 교체를 주면
게이트를 통과한 뒤 다른 체인에 뜻 없는 overlay 가 얹힌다. **이름을 대고 거절한다.**

| 케이스 | override | 왜 고유한가 | 지금 `requires` |
|---|---|---|---|
| `wemix-brioche-block-reward` | `config.brioche.{firstHalvingBlock,halvingPeriod,…}` | `brioche` 는 wbft·wemix 에만 있다 | `rpc` |
| `state-written-before-the-fork-survives-it` | `upgrade.preset="wemix-upgrade"`, `fork="croissant"` | `croissant` 는 wbft 에만, preset 은 wemix 전용 | `rpc`, `consensus` |
| `sample-lifecycle-node-restart` | `config.bohoBlock: 6` | `boho` 는 stablenet 만 안다 | `rpc`, `consensus`, `process` |

### 게이트가 이미 막는다 — 15건

`requires` 가 체인 고유 능력을 이미 이름으로 말한다. 체인을 바꾸면 **override 가 적용되기 전에
SKIP** 된다. 거절 장치가 따로 필요 없다.

| 무엇을 요구하나 | 건수 | 케이스 |
|---|---|---|
| `fork:boho` (+`contract:govMinter`/`govValidator`) | 5 | `stablenet-delayed-fork` · `prealloc-preserved-across-boho` · `boho-chain-config-active` · `anzeon-active-before-boho` · `unsupported-system-contract-version` |
| `engine:anzeon` + `contract:govCouncil` | 6 | `authorized-accounts-{no-space,space,trim,empty-item,single,empty}` |
| `engine:anzeon` + `contract:govValidator` | 1 | `validator-add-member-epoch-activates` |
| `contract:accountManager` | 3 | `stablenet-account-extra` · `extra-union-merge` · `extra-state-across-delayed-boho` |

### 중립 — 10건 (그중 8건은 선언이 부족하다)

override 가 세 체인 모두에서 뜻이 있다. 교체 가능하다.

| 케이스 | override | 판정 |
|---|---|---|
| `wbft-tx-and-contract` · `wemix-tx-and-contract` | `alloc.<addr>.balance` 뿐 | 그대로 교체 가능 |
| 대납 7건 (`fee-delegated-*` · `fd-*` · `feepayer-insufficient-rejected`) | `config.applepieBlock: 0` | **교체 가능하나 `tx:0x16` 선언이 없다** |
| `set-code-delegation` | `config.applepieBlock: 0` | **`family:wbft` 보다 `tx:0x04` 가 정확하다** |

`applepie` 는 세 체인이 **모두 아는** 포크다(wbft·wemix 는 템플릿이 0 으로 켠다). 그래서 이
override 는 다른 체인에서 무해하다 — 문제는 override 가 아니라 **시험 대상을 선언하지 않은
것**이다. 대납 7건은 `requires` 가 `rpc` 뿐인데, 매니페스트를 보면 대납 타입 `0x16` 은 세 체인이
다 받고 EIP-7702 의 `0x04` 는 **wemix 에 없다**(stablenet·wbft `["0x00","0x01","0x02","0x03","0x04","0x16"]`,
wemix `["0x00","0x01","0x02","0x16"]`). 선언이 없으면 체인이 타입을 빼는 날 **SKIP 이 아니라
단언 실패**로 나타난다 — `manifest_capability.go:55` 가 기록한 것과 같은 실패다.

## 5. 단점

- ~~키 파생이 우연에 기대고 있다~~ — **철회 (2026-09-20). 단점이 아니다.** 검토에서 나온 지적이
  맞았다: wemix 에서는 만들어진 BLS 키를 **쓰지 않으면 그만**이다. 근거 둘. ① `derive.Derive`
  는 주소와 devp2p 공개키를 개인키에서 똑같이 계산하고 `WithBLS` 는 `Identity.BLS` 를 **더할
  뿐**이다 — 키 자체도 주소도 달라지지 않는다. ② 공유 모양들이 쓰는 `source: "keyPreset"` 은
  저장소에 커밋된 픽스처 `keys/preset` 을 **읽기만 한다**(각 노드 디렉터리에 `bls`·`bls_pubkey`·
  `pop` 이 이미 있다). 그 경로에서는 체인별로 파생하는 일 자체가 없다. 따라서 모양은 파생 정책과
  무관하게 체인 독립이다.

  남은 N3 빚의 성격은 **표현**이다. `derive/identity.go` 가 *"a wemix node has no BLS key, and
  modelling that as absence rather than as zeroes keeps the two cases distinguishable downstream"*
  라고 적는데, `chainsetup/steps_keys.go:159·206·228` 이 패밀리를 묻지 않고 늘 `WithBLS` 를
  넘겨 그 의도를 어긴다. `generate`·`declared` 경로에서 BLS 를 읽지 않는 체인의 기록에 BLS 가
  남는다. 계산 비용도 CGO 도 걸림돌이 아니다 — `deriveBLS` 는 순수 Go(`kilic/bls12-381`)이고
  주석이 "CGO 를 꺼도 돈다" 고 적는다. **이 설계의 선행 조건이 아니다.**
- **`requires` 보강 8건은 이 설계의 선행 작업이다.** 지금 상태로 교체하면 대납 7건이 조용히
  다른 뜻으로 돈다.
- **새 플래그 이름이 어휘를 하나 늘린다.** `--chain` 과의 차이를 문서로 설명해야 한다.
