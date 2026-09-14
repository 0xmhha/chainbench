# 용어 지도

> **[측정]** 2026-09-14 기준. 코드가 바뀌면 다시 뽑는다. 어긋나면 코드가 이긴다.
> 대상 리비전: `fc94e443` (브랜치 `codex/refactoring-proposal-review`)
> 세는 방법: `internal/`·`cmd/`·`scripts/` 의 `.go` 파일을 줄 단위로 훑고, 파일 경로와
> 줄 내용으로 뜻을 갈랐다. 체인을 띄워 확인한 것이 아니다.

## 이 문서가 있는 이유

한 낱말이 여러 뜻을 가지면 읽는 사람이 틀린다. 2026-09-14 하루 동안 그런 일이 네 번
있었다.

`role.go` 의 주석이 "옛 철자는 영속 상태에 쓰여 있어 계속 동작한다" 고 해서 사실로
받아들였다. 그런 데이터는 없었다. `peering.go` 의 주석이 "poa 에는 프록시 계층이
없다" 고 했는데 `poa.Family.SupportsRole` 은 `pn` 에 true 를 돌려준다. "옛
워크스페이스" 라고 쓰면서 서로 다른 세 가지를 섞어 불렀다. `RoleBoot` 를 주석만 고쳐
남기자고 했다가 철회했다.

네 번 모두 원인이 같다. **낱말을 믿고 코드를 안 봤다.** 그리고 그 낱말들이 실제로
여러 뜻을 갖고 있었다.

이 문서는 그런 낱말이 어디에 몇 개의 뜻으로 살고 있는지를 적는다. 작업 규칙
(`mainnet-config-worklist.md` §진행 규칙)의 "주석을 근거로 결정하지 않는다" 를 실제로
지키려면 무엇이 위험한지를 먼저 알아야 한다.

## 읽는 법

각 낱말마다 뜻을 나열하고 **주인**을 적는다.

- **우리 것** — chainbench 가 정한 이름이다. 틀렸으면 고친다.
- **체인 것** — go-wemix·go-stablenet·go-wbft 가 정한 이름이다. RPC 메서드 이름,
  genesis 필드 이름, 합의 도메인 용어가 여기 든다. **고치면 안 된다.** 고치는 순간
  우리 이름과 체인 이름 사이에 번역이 하나 생기고, 그 번역이 다음 헷갈림의 원인이 된다.

---

## 1. `validator`

가장 많이 쓰이고 가장 여러 뜻을 가진 낱말이다. 코드 797곳, 테스트 823곳이다.

| 뜻 | 무엇인가 | 주인 | 주로 사는 곳 | 코드 | 판정 |
|---|---|---|---|---|---|
| 1 | **bp 노드의 개수** | 우리 | `cmd/chainbench/{chaincmd,suitecmd,resourcecmd}`, `chainsetup.State` | 42 | **틀렸다. 고친다** |
| 2 | **계정 기능** — 키 집합에서 블록을 만드는 신원 | 우리 | `internal/core/keyring`, `cmd/chainbench/keyringcmd`, `registry.AccountValidator` | 108 | 맞다. 둔다 |
| 3 | **genesis 에 박히는 주소 집합** | 체인 | `internal/core/genesis`, `internal/core/blueprint`, `registry.GenesisParams` | 43 | 맞다. 둔다 |
| 4 | **실행 중인 합의 참여 집합** | 체인 | `ValidatorsMethod`, `RunningValidators`, `internal/core/health` | 31 | 맞다. 둔다 |
| 5 | **포크 뒤를 이어받는 바이너리** | 우리 | `dsl.BinaryAfter`, 정의서의 `binaries` 키 | 2 | **틀렸다. 고친다** |
| 6 | **wbft 합의에 참여하는 노드** | 체인 | `internal/consensus/wbft`, `internal/consensus/poa` | 81 | 맞다. 둔다 |

### 왜 2·3·4·6 은 맞는가

`en` 과 `pn` 은 합의에 참여하지 않는다. genesis 에도 들어가지 않는다. 그래서 "합의에
참여하는 것" 을 가리킬 때 `validator` 는 애매하지 않다. 그 자리에서 `validator` 는
**`bp` 노드가 제안 차례가 아닐 때 하는 동작**을 정확히 가리킨다.

뜻 2 에 조건이 하나 붙는다. 그 계정이 `bp` 의 신원일 때만 맞다. 만약 `en` 이나 `pn`
도 쓰는 계정을 가리키게 되면 그때는 `validator` 가 아니다.

### 왜 1 은 틀렸는가

`chain place --validators 4` 는 "bp 노드를 4대 만들어라" 다. 개수를 세는 대상이
노드의 **역할**이고, 역할 이름은 `bp` 로 정했다. V3 에서 역할 상수를 정리했지만 그
개수를 세는 플래그는 옛 낱말로 남았다.

여기에 더 나쁜 것이 있다. **같은 `--validators` 라는 이름이 명령마다 다른 것을 뜻한다.**

| 명령 | `--validators` 의 뜻 | 어느 뜻인가 |
|---|---|---|
| `chain place`, `chain up`, `run`, `resource plan` | bp 노드 개수 | 뜻 1 (틀림) |
| `chain keys`, `account validator`, `account new` | 검증자가 될 신원의 수 | 뜻 2 (맞음) |
| `verify --validators` | 실행 중인 집합을 대조할지 여부 | 뜻 4 (맞음) |

같은 이름이 한 CLI 안에서 셋을 가리킨다.

노드 개수 플래그끼리도 어휘가 갈려 있다. 하나는 없앤 역할 낱말(`--validators`), 하나는
옛 철자(`--endpoints`), 하나는 역할 이름도 아니다(`--proxies`). 그런데 `chain
blueprint` 는 같은 것을 이미 `--bp` 라고 부른다.

### 왜 5 는 틀렸는가

```go
BinaryBefore = "producer"
BinaryAfter  = "validator"
```

정의서에서 포크 전후 바이너리를 부르는 이름이다. 실제로는 이렇다.

`BinaryBefore` 는 **go-wemix** 다. wemix 3.0 이고 poa 로 포크 지점까지 블록을 봉인한다.
`BinaryAfter` 는 **go-wbft** 다. 포크 전까지는 `en` 으로 블록을 받아 동기화만 하다가,
정해진 블록 이후부터 새 합의로 블록을 만든다.

즉 둘은 **바이너리**이고 노드 역할이 아니다. 그런데 하나는 역할처럼 들리는 이름
(`producer`)이고 다른 하나는 방금 없앤 낱말(`validator`)이다.

코드는 이미 올바른 이름을 쓰고 있다. `FromBinary`·`ToBinary` 이고 CLI 도
`--from-binary`·`--to-binary`·`--to-chain`·`--from-genesis` 다. **정의서의 두 키만
다르다.**

---

## 2. `boot`

V3 에서 노드 역할로서의 `boot` 를 없앴다. 남은 것들은 서로 다른 것이고 그대로 둔다.

| 뜻 | 무엇인가 | 주인 | 사는 곳 | 판정 |
|---|---|---|---|---|
| 1 | ~~노드 역할~~ | 우리 | — | **제거됨 (V3)** |
| 2 | poa 기동 단계의 이름 | 우리 | `internal/consensus/poa` 의 `Phase{Name: "boot"}` | 순서의 이름이지 역할이 아니다. 둔다 |
| 3 | 노드 하나에 붙는 속성 | 우리 | `node.Entry.Bootnode`, poa 의 etcd 씨앗 지정 | 역할이 아니라 속성이다. 둔다 |
| 4 | BLS 를 파생하는 외부 바이너리 이름 | 우리 | `dsl.EnvV2.Bootnode` | 노드와 무관하다. 이름이 나쁘지만 V9 범위 밖 |

노드 사이의 연결은 모든 지원 체인에서 `pn` 이 맡는다. 따로 `boot` 를 둘 이유가 없다.

---

## 3. `workspace`

`mainnet-config-worklist.md` §용어가 정본이다. 여기서는 요약만 적는다.

| 쓸 말 | 무엇인가 | 코드의 이름 |
|---|---|---|
| 대상 워크스페이스 | 실행 대상 머신의 `dataRoot` 아래 약속된 폴더 트리 | `resource.WorkspaceConfig`, `workspace-config.yaml` |
| 구성 기록 | 한 번의 구성이 무엇을 만들었는지 남긴 기록 | `chainsetup.State`, 파일은 `workspace.json` (**이름이 틀렸다 — W0**) |
| 실행 세션 | 엔진 한 번의 실행이 남긴 아티팩트 | `internal/core/session` |
| 로컬 작업 공간 | 따로 지정하지 않았을 때 chainbench 가 자기 것을 두는 곳 | `internal/core/home`, `~/.chainbench` |

---

## 4. `preset`

세 가지로 쓰인다. 어느 것을 정본 이름으로 할지는 워크리스트 D1 의 결정 대상이다.

| 뜻 | 무엇인가 | 사는 곳 |
|---|---|---|
| 1 | 키 출처 — 미리 만들어 둔 키 집합 | 정의서의 `keys.nodekeys.source: "preset"` (케이스 93건), `internal/core/keyring/store` |
| 2 | 미리 준비된 입력 묶음 — genesis·키링·설정 | `resource.InputPreset`, `workspace-config.yaml` 의 `presets:` |
| 3 | (예정) 테스트가 공통으로 참조할 설정 문서 | 아직 없다 |

`InputPreset` 타입 주석에 이미 이렇게 적혀 있다. "이름을 일부러 Preset 으로 하지
않았다. keyring 이 이미 그 낱말을 키 출처로 쓰고 있고, 한 개념은 한 이름을 갖는다."
세 번째 뜻을 얹으면 그 원칙이 깨진다.

---

## 5. 앞으로의 규칙

**낱말을 새로 쓰기 전에 이 문서를 본다.** 이미 쓰이고 있으면 다른 낱말을 고르거나, 이
문서에 뜻을 하나 더 적고 주인을 밝힌다.

**체인 것은 번역하지 않는다.** RPC 메서드 이름, genesis 필드 이름, 합의 도메인 용어는
그 체인의 것이다. 우리 이름으로 바꾸면 번역이 생기고, 번역은 다음 세대의 헷갈림이다.
대신 필드 주석에 "이건 체인이 정한 이름" 이라고 적는다.

**한 이름이 명령마다 다른 것을 뜻하게 두지 않는다.** `--validators` 가 그렇게 됐다.

**이 문서는 낡는다.** 기계로 지킬 수 있는 부분은 테스트로 박는다. 지금
`internal/arch` 가 지키는 것은 주석이 없는 경로·심볼을 가리키지 않는 것과 패키지
주석이 제 패키지를 말하는 것이다. 낱말의 뜻은 기계가 판정할 수 없으므로 사람이 본다.

---

## 6. V9-b 가 바꿀 것

이 조사로 확정된 변경 대상은 둘뿐이다. 1,426곳 중 **44곳**이다.

| 대상 | 지금 | 바꿀 것 | 규모 |
|---|---|---|---|
| 노드 개수 플래그와 그 배선 | `--validators` / `--endpoints` / `--proxies` | `--bp` / `--en` / `--pn` | 플래그 10곳 + 배선 32곳 |
| 정의서의 바이너리 키 | `producer` / `validator` | 코드가 이미 쓰는 `from` / `to` 에 맞춘다 | 상수 2곳 + 정의서 1건 |

나머지 1,382곳은 **그대로 둔다.** 뜻이 맞거나, 체인의 것이다.

### 딸린 위험

**사용자 표면이 바뀐다.** `--validators` 는 사람이 치는 플래그다. 스크립트에 남아 있을
수 있다.

**기록 형식이 바뀐다.** `State.Validators` 필드 이름이 바뀐다. W5 에서 형식 버전을
넣었으므로 옛 기록은 조용히 이상해지지 않고 이름을 대며 거부된다.

**정의서 하나가 바뀐다.** `tests/tc/go-wemix/handoff/01-wemix-wbft-handoff.json` 이
`producer`/`validator` 키를 쓴다.

---

## 7. 변경 이력

| 날짜 | 내용 |
|---|---|
| 2026-09-14 | 최초 작성. V9-a 조사 결과. `validator` 6뜻, `boot` 4뜻, `workspace` 4뜻, `preset` 3뜻 |
