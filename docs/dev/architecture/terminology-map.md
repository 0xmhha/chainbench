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
| 1 | **bp 노드의 개수** | 우리 | `cmd/chainbench/{chaincmd,suitecmd,resourcecmd}`, `chainsetup.State` | 42 | **고쳤다 (V9-b)** |
| 2 | **계정 기능** — 키 집합에서 블록을 만드는 신원 | 우리 | `internal/core/keyring`, `cmd/chainbench/keyringcmd`, `registry.AccountValidator` | 108 | 맞다. 둔다 |
| 3 | **genesis 에 박히는 주소 집합** | 체인 | `internal/core/genesis`, `internal/core/blueprint`, `registry.GenesisParams` | 43 | 맞다. 둔다 |
| 4 | **실행 중인 합의 참여 집합** | 체인 | `ValidatorsMethod`, `RunningValidators`, `internal/core/health` | 31 | 맞다. 둔다 |
| 5 | **포크 뒤를 이어받는 바이너리** | 우리 | `dsl.BinaryAfter`, 정의서의 `binaries` 키 | 2 | **고쳤다 (V9-b)** |
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

### 표준 노드 구성

15대짜리 망은 `bp` 7대, `en` 7대, `pn` 1대로 짠다. `en` 은 `pn` 과 연결하고 `pn` 을
거쳐 `bp` 와 주고받는다. `en` 이 `bp` 에 직접 연결하지 않는다.

연결 규칙은 코드가 이미 그렇게 한다. `pn` 이 하나라도 있으면 구성이 자동으로 proxied
가 되고(`internal/testengine/compose.go`), proxied 에서 `en` 의 피어는 `pn` 뿐이다
(`internal/core/node/peering.go`).

`bp` 7대는 BFT 정족수를 floor(2n/3)+1 = 5 로 만든다. 두 대가 빠져도 돌고 세 대가
빠지면 멈춘다. 전에는 `bp` 13대에 `en` 1대였고 정족수가 9였다.

두 패밀리 모두 `pn` 을 돌린다. poa 에 프록시 계층이 없다고 적힌 주석이 여러 곳에
있었는데 코드와 달랐다(2026-09-14 정정).

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
| 체인 기록 | 한 체인이 무엇으로 요청됐고, 무엇으로 구성됐고, 지금 어떤 상태인지 | `chainsetup.State`, 파일은 `workspace.json` → **`chain-record.json` (N4·W0)** |
| 실행 세션 | 엔진 한 번의 실행이 남긴 아티팩트 | `internal/core/session` |
| 로컬 작업 공간 | 따로 지정하지 않았을 때 chainbench 가 자기 것을 두는 곳 | `internal/core/home`, `~/.chainbench` |

---

## 4. `preset`

세 가지로 쓰인다. **셋 다 이름을 받는다 (N5).** 맨 낱말 `preset` 은 어느 것도
뜻하지 않으므로 코드에도 문서에도 홀로 쓰지 않는다.

| 뜻 | 무엇인가 | 사는 곳 | 정한 이름 |
|---|---|---|---|
| 1 | 키 출처 — 미리 만들어 둔 키 집합 | 정의서의 `keys.nodekeys.source: "preset"` (케이스 199건), `internal/core/keyring/store` | `key-preset` |
| 2 | 대상에 **이미 있는** 입력 묶음 — genesis·키링·설정 | `resource.InputPreset`, `workspace-config.yaml` 의 `presets:` | `existing-inputs` |
| 3 | 테스트가 공통으로 참조할 체인 구성 | 아직 없다. P1 에서 만든다 | `chain-preset` |

2번을 `prepared` 가 아니라 `existing` 으로 부르는 이유는, 확인되는 사실이 "누가
준비했다" 가 아니라 **"대상에 이미 있고 chainbench 가 만들지 않는다"** 뿐이기
때문이다.

`InputPreset` 타입 주석에 이미 이렇게 적혀 있다. "이름을 일부러 Preset 으로 하지
않았다. keyring 이 이미 그 낱말을 키 출처로 쓰고 있고, 한 개념은 한 이름을 갖는다."
그 관찰이 맞았고, 해법은 셋을 갈라 부르는 것이다.

---

## 4.5 바이너리 이름

한 낱말 문제는 아니지만 같은 뿌리다. 바이너리를 부르는 이름이 네 곳에 있었고 주인이
정해져 있지 않았다.

| 어디 | 무엇을 정하는가 | V1 전 | V1 후 |
|---|---|---|---|
| 매니페스트 `binary` | 체인이 제 바이너리를 부르는 이름 | 기본값으로 안 쓰였다 | 아무도 이름을 안 대면 이것이 답한다 |
| 워크스페이스 설정 `binaryAliases` + `paths.binaries` | 이 환경에서 그 이름이 어느 파일인가 | upgrade 경로만 썼다 | 일반 경로가 쓴다 |
| 정의서 `binaries` | 실제로 실행될 문자열 | 이것만 썼고 검증 없이 exec 로 갔다 | 이름만 적고, 안 적으면 매니페스트가 답한다 |
| `--binary` | 이번 실행이 쓸 것 | 최우선 | 최우선 (경로 허용) |

해석 순서는 `internal/chainsetup/binary.go` 한 곳이다. 가장 구체적인 것이 이긴다.

정의서에는 **이름만** 적는다. 경로를 적으면 파싱 단계에서 거부한다
(`binaryRefIsAName`). 정의서는 "어느 바이너리인가" 를 말하는 문서이고, "그 바이너리가
대상의 어디 있는가" 는 워크스페이스 설정이 말한다. 둘을 한 곳에 적으면 그 케이스는
한 환경에서만 돈다. 실제로 다섯 건이 그랬고, 같은 명령줄로 넘기던 환경 파일이 이미
같은 경로를 만들고 있었다.

이름이 경로가 되는 것은 워크스페이스 설정이 있을 때뿐이다. 없으면 이름은 이름으로
남고 대상의 PATH 가 푼다. 그래서 실행 전 검사도 맨 이름을 stat 하지 않고 PATH 로
확인한다(`inspector.OnPath`). 전에는 stat 해서, 실행하면 찾았을 바이너리를 없다고
보고했다.

---

## 4.6 합의와 바이너리 방언

한 낱말 문제의 또 다른 형태다. `ConsensusFamily.StartFlags` 가 두 가지를 뱉고 있었다.

| 플래그 | 실제로 무엇이 정하나 |
|---|---|
| `--mine` | 합의. 생산자가 봉인한다 |
| `--allow-insecure-unlock` | 하니스. 계정을 HTTP 로 연다. 두 family 가 무조건 붙이고 있었다 |
| `--rpc.enabledeprecatedpersonal` | **바이너리**. go-wemix 세대에는 이 플래그가 없다 |
| `--rpc.allow-unprotected-txs` | **바이너리**. 두 세대 다 있다 |

합의 family 가 "어떤 바이너리가 어떤 플래그를 받는가" 를 정하고 있었다. 그건 방언
(`nodeconfig.Dialect`)이 아는 것이고, 실제로 `Geth110Wemix` 가 첫 번째 키를 지운다.

**V7 에서 방언은 매니페스트가 선언한다.** 전에는 `DialectFor(chainID)` 가
`if chainID == "wemix"` 로 갈렸다. 옛 geth 세대를 쓰는 새 체인이 다른 이름이면
조용히 최신 어휘를 받아, 바이너리에 없는 플래그를 달고 부팅에서 죽었다. 지금은
매니페스트에 `dialect` 가 필수이고 모르는 이름은 오류다.

V10 에서 family 는 `registry.LaunchPolicy{Mine bool}` 만 내놓는다. 나머지는 하니스가
매번 요청하고 방언이 가진 것만 나간다. 문자열 배열을 만들어 다시 파싱하던 shim
(`ParseFamilyFlags`, 39줄)이 사라졌다.

**동작 변화가 하나 있다.** wemix 가 `--rpc.allow-unprotected-txs` 를 받게 됐다. 전에는
poa family 의 문자열 목록에 그 플래그가 없어서 빠졌는데, wemix 방언은 그 키를 갖고
있다. 즉 지원하는 플래그가 빠져 있던 것이다. 이에 의존하는 테스트나 문서는 없다.

---

## 4.7 체인 하나는 네 가지 선택이다

V11 이후 체인 폴더의 파일 하나가 그 넷을 고른다.

```go
registry.Register(registry.StaticPlugin{
    M:     registry.MustParseManifest(manifestJSON),  // 체인 상수 + 방언
    Fam:   poa.New(),                                 // 합의
    Proto: protocol.WeMix(),                          // 계정·암호
    Tmpl:  genesisTmpl,                               // genesis 템플릿
})
```

전에는 세 체인이 각자 똑같은 네 메서드짜리 타입을 손으로 썼다. 선택이 체인마다
네 곳에 흩어져 있었고, 하나를 빠뜨려도 검사하는 것이 없었다.

공용 구현은 복제되지 않는다. stablenet 과 wbft 는 서로 다른 프로젝트인데 같은
`wbft.Family` 를 고른다. 한 메서드만 달라야 하면 임베딩으로 덮는다.

```go
type family struct{ wbft.Family }
func (family) PortReservation() node.Reservation { ... }
```

**기존 합의와 계정 프로토콜을 빌려 쓰는 EVM 체인은 Go 파일 없이 더할 수 있다.**
매니페스트와 genesis 템플릿 두 파일이면 된다(2026-09-14 실측: `--manifest` 로
새 체인을 열어 확인). 계정 모델이나 시스템 컨트랙트 방식이 다르면 accounts SDK
작업이 필요하고, 그건 다른 저장소다.

---

## 4.8 이름을 정하는 방법

**"이것이 무엇인가" 로 정한다. "무엇이 비었나" 로 정하지 않는다.**

2026-09-15 에 공유 구성 문서의 이름을 고르면서, 이미 쓰이는 낱말을 전부 배제하고
남은 것 중에서 골랐다. 그러면 뜻이 맞는 낱말을 버리고 뜻이 덜 맞는 낱말을 고르게
된다. 이미 쓰인 이름이 틀렸다면 **그 이름을 고치는 것**이 맞다.

그렇게 정한 이름이 `chain-preset` 이다. 미리 구성해 둔 체인 설정을 그대로 쓰는
것이니 `preset` 이 맞는 낱말이고, 무엇의 preset 인지를 `chain-` 이 말한다.

### 설정의 해석 순서

| 차수 | 무엇 | 지금 코드의 이름 |
|---|---|---|
| 1차 | 공유하는 체인 구성 | `chain-preset` (아직 없다. P1 에서 만든다) |
| 2차 | 그 테스트만 다른 것 | `LayerEnv` |
| 3차 | CLI·MCP 가 실행할 때 덮는 것 | `LayerCommand` |

**층 이름 넷은 이 해석 순서와 같은 축이 아니다.** `LayerHarness`·`LayerRole` 은
"누가 그 값을 계산했나" 를, `LayerEnv`·`LayerCommand` 는 "어느 문서에서 왔나" 를
말한다. 1차와 2차는 병합이 끝나면 둘 다 `LayerEnv` 로 들어오고 층은 그 둘을
구분하지 못한다 — 정의서에 무엇을 덮었는지가 남으므로 구분할 필요가 없다.

3차가 무조건 이긴다. 그래서 병합을 아무리 잘해도 **마지막에 실행 명령이 달라지면
다른 구성으로 도는 것**이고, 계획과 실제를 대조하는 자리가 필요하다(M4-c).

어긋난 이름의 목록은 워크리스트 §2.5 에 있다.

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

## 6. V9-b 가 바꾼 것 (2026-09-14 완료)

이 조사로 확정된 변경 대상은 둘뿐이다. 1,426곳 중 **44곳**이다.

| 대상 | 전 | 후 |
|---|---|---|
| CLI 플래그 (`chain place`·`chain up`·`run`·`resource plan`) | `--validators` / `--endpoints` / `--proxies` | `--bp` / `--en` / `--pn` |
| MCP 도구 인자와 스키마 | `validators` / `endpoints` / `proxies` | `bp` / `en` / `pn` |
| Go 필드 | `Validators` / `Endpoints` / `Proxies` / `Producers` | `BPCount` / `ENCount` / `PNCount` |
| 구성 기록의 키 | `"validators"` | `"bp"` |
| 정의서의 바이너리 키 | `producer` / `validator` | `from` / `to` |
| `chain place` 출력 | `2 validator(s) + 2 endpoint(s)` | `2 bp + 1 en + 1 pn` |
| `chain status` 출력 | `validators: 2` | `bp: 2` |

뜻이 다른 `--validators` 는 그대로 뒀다. `chain keys` 와 `account` 의 것은 신원 수
(뜻 2)이고, `verify --validators` 는 실행 중 집합 대조(뜻 4)다.

`chain place` 출력은 역할별로 세도록 바꿨다. 전에는 생산자가 아닌 노드를 전부
endpoint 로 셌고, 그래서 `pn` 이 endpoint 로 보고됐다. 무엇이 놓였는지 말하는 한 줄이
프록시 계층을 감추고 있었다.

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
| 2026-09-15 | 이름을 정하는 방법과 설정의 해석 순서를 적음. `chain-preset` 확정 |
| 2026-09-14 | V11 반영. 체인 하나가 네 선택이라는 것을 적음 |
| 2026-09-14 | V7 반영. 방언의 주인은 매니페스트 |
| 2026-09-14 | V10 반영. 합의와 바이너리 방언을 가름 |
| 2026-09-14 | V2 반영. 정의서는 이름만 적는다 |
| 2026-09-14 | V1 반영. 바이너리 이름의 주인을 적음 |
| 2026-09-14 | V9-b 반영. 뜻 1·5 를 고치고 나머지 4뜻은 그대로 뒀다 |
| 2026-09-14 | 최초 작성. V9-a 조사 결과. `validator` 6뜻, `boot` 4뜻, `workspace` 4뜻, `preset` 3뜻 |
