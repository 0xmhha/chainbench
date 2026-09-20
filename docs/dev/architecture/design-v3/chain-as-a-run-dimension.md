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

### 거절 — 케이스 수준에는 없다. env 수준에 2건 (2026-09-21 정정)

처음에 3건을 거절 대상으로 적었는데 **틀렸다.** 게이트를 다시 읽어 보니 overlay 가 포크를 켜면
`networkCapabilities`(`chainsetup/steps_genesis.go:537-541`)가 그 `fork:<name>` 을 **제공한다** —
override 의 `<fork>Block` 으로도, overlay 안의 엔진 섹션으로도. 그래서 셋 중 둘은 거절이 아니라
**선언이 빠진 것**이었고, 요구를 달면 15건과 똑같이 게이트가 알아서 건너뛴다.

진짜 거절은 **케이스가 아니라 env** 에 있다. `wemix-to-wbft` 는 `chain: "wemix"` 인데
`binaries.next` 가 `{"binary": "${GWBFT_BIN}", "chain": "wbft"}` 다 — **구조상 두 체인을 이름으로
부른다.** 같은 모양이 `wemix-to-wbft-bp2` 까지 둘이고, 이들을 extends 하는 케이스는 교체 대상이
아니다. 거절은 여기 한 곳에 두면 된다.

| 교체 불가 env | 왜 |
|---|---|
| `wemix-to-wbft` · `wemix-to-wbft-bp2` | `chain` 과 `binaries.next.chain` 이 서로 다른 체인이다. 핸드오프 자체가 시험 대상이라 다른 체인으로 옮길 수 없다 |

**못 박았다 (2026-09-21).** `arch.TestTwoChainEnvsAreListed` 가 `tests/tc/env` 를 걸어 **두 체인을
부르는 env 의 집합**을 고정한다. 집합은 선언이 아니라 **유도**한다 — env 의 `binaries.<name>.chain`
이 그 env 의 `chain` 과 다르면 두 체인이다. 문서가 이미 말하고 있는 사실이라 새 필드를 더하지 않았다.
새 항목이 생기면 래칫이 막고, **체인 교체 플래그를 넣는 사람이 그것을 어떻게 다룰지 먼저 정하게 된다.**
변이 둘로 확인했다: 다른 env 에 두 번째 체인을 넣으면 잡히고, 목록에서 하나를 빼도 잡힌다.

거절을 케이스가 아니라 env 에 둔 이유는 **자리가 하나이기 때문**이다. 케이스에 두면 그 env 를
extends 하는 케이스가 늘 때마다 같은 거절을 반복해 적어야 한다.

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
| 대납 7건 (`fee-delegated-*` · `fd-*` · `feepayer-insufficient-rejected`) | `config.applepieBlock: 0` | ~~`tx:0x16` 선언 없음~~ **달았다 (2026-09-21)** |
| `set-code-delegation` | `config.applepieBlock: 0` | ~~`family:wbft`~~ → **`tx:0x04` 로 교체 (2026-09-21)** |

### 선언 보강 10건 — 완료 (2026-09-21)

실제 레지스트리로 제공 여부를 확인한 결과다(`networkCapabilities` 에 각 케이스의 overlay 를 넣어 계산).

| 요구 | stablenet | wbft | wemix | 뜻 |
|---|---|---|---|---|
| `tx:0x16` (대납 7건) | 제공 | 제공 | 제공 | 지금은 어디서도 안 걸린다. 체인이 타입을 빼는 날 **실패가 아니라 SKIP** 이 되는 것이 값이다 |
| `tx:0x04` (set-code 1건) | 제공 | 제공 | **없음** | 옛 `family:wbft` 와 판정이 같고, 의존을 이름으로 정확히 말한다 |
| `fork:brioche` (1건) | **없음** | 제공 | 제공 | wbft 도 brioche 를 안다. wemix 전용이 아니라 stablenet 만 제외된다 |
| `fork:boho` (1건) | 제공 | **없음** | **없음** | overlay 의 `bohoBlock` 이 켜서 제공된다 |

`tx:` 둘은 지금 아무 체인도 걸러내지 않는다 — 그래서 **게이트가 아니라 선언**이다. 값은
`manifest_capability.go:55` 가 기록한 실패(게이트를 통과한 뒤 단언에서 깨짐)를 미리 막는 데 있다.

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
