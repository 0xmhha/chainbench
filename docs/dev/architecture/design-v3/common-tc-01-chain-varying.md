# 공통 TC 가 세 체인에서 도는 법 — 체인별로 다른 것을 케이스가 부르는 방법

> **[제안] (2026-09-28).** 결정이 아니다. 아래 수치는 이날 `c08afbcd` 기준 실측이고,
> 인용하기 전에 다시 잰다. 상위 제안은
> [`chain-as-a-run-dimension.md`](chain-as-a-run-dimension.md) 이고 이 문서는 그 §4 의
> 1번과 3번을 연다.

## 1. 무엇이 문제인가

공통 테스트 99개가 `tests/tc/common/` 으로 옮겨졌다(`343c89a3`). 자리는 갈렸지만 **세
체인에서 도는지는 아직 아무도 재지 않았다.** 이 문서는 그것을 재고, 왜 안 도는지 밝히고,
어떻게 하면 도는지 정한다.

### 1.1 실측 — 지금 몇 개가 도나

케이스의 `requires` 를 각 체인이 제공하는 capability 집합과 대조했다. 체인이 제공하는
집합은 매니페스트에서 파생되며(`internal/core/registry/manifest_capability.go`
`DerivedCapabilities`), 코드를 실행해 뽑았다.

| 체인 | 제공하는 것 중 갈리는 부분 |
| --- | --- |
| stablenet | `engine:anzeon` `family:wbft` |
| wbft | `engine:croissant` `family:wbft` `precompile:secp256r1` |
| wemix | `family:poa` |

그 결과 capability 게이트만으로 이렇게 갈린다.

| 체인 | 99건 중 게이트를 통과하는 것 | 걸러지는 것 |
| --- | --- | --- |
| stablenet | 99 | 0 |
| wbft | 86 | 13 |
| wemix | 68 | 31 |

**세 체인 모두에서 게이트를 통과하는 것은 68건이다.** 걸러지는 이유는 둘뿐이다.

- `family:wbft` 를 요구하는 18건 — wemix 는 `family:poa` 라 걸린다.
- `engine:anzeon` 을 요구하는 13건 — stablenet 만 anzeon 이라 wbft·wemix 둘 다 걸린다.

### 1.2 걸러진다는 것이 무슨 뜻인가

`internal/testengine/capability.go` 의 `applicableWithCaps` 주석이 적는다. **"A spec that
requires a capability the target lacks is skipped, not failed."** 즉 wemix 에서 공통
테스트 31건은 실패하지 않는다. **조용히 건너뛴다.**

이것이 이 설계가 존재하는 이유다. 공통 테스트를 세 체인에 돌려 전부 통과했다고 적으면서
실제로는 wemix 에서 3분의 1을 묻지 않는 상태가 만들어진다. 저장소는 이 모양의 결함을 이미
알고 있다 — 작업 리스트 §1s F 가 "통과해도 아무것도 막지 못하는 검사" 로 따로 절을 두었다.

### 1.3 게이트를 통과해도 끝이 아니다

통과는 "돌 수 있다" 이지 "통과한다" 가 아니다. 게이트를 지나도 케이스 본문이 체인 고유의
RPC 를 부르면 그 자리에서 실패한다. 공통 99건에서 그런 호출을 세면 이렇다.

| 메서드 | 건수 | 어느 체인에 있나 |
| --- | --- | --- |
| `istanbul_getWbftExtraInfo` | 7 | stablenet·wbft (wemix 없음) |
| `istanbul_getValidators` | 5 | stablenet·wbft (wemix 는 `wemix_getValidators`) |

## 2. 갈리는 것은 네 가지다

99건을 하나씩 읽어 갈리는 이유를 분류하면 셋으로 떨어진다.

> **2026-09-28 정정.** 이 절은 처음에 "셋으로 떨어진다. 넷째는 없었다" 로 끝났다.
> 틀렸다. 넷째가 있었고, 읽어서는 안 보였다. 라이브로 돌려야 나오는 종류였기
> 때문이다. 아래 (D) 가 그것이며, 어떻게 드러났는지는 §2.1 에 적는다.

**(A) 같은 질문을 체인마다 다른 메서드로 묻는다.** 검증자 목록이 그렇다. stablenet·wbft
는 `istanbul_getValidators`, wemix 는 `wemix_getValidators` 다. 질문은 하나고 이름이
둘이다.

**(B) 같은 절차의 정답이 체인마다 다르다.** CT-FEE-002 가 그렇다. 목록 문서가 기대
결과를 이렇게 적었다 — "같음과 초과는 성공한다. 미만은 StableNet에서 거부된다.
WEMIX3.0과 WEMIX4.0에서는 거부되거나 블록에 포함되지 않은 채 남는다." 절차는 하나인데
정답이 둘이다.

**(C) 그 체인에 그 규칙이 없다.** anzeon 의 수수료 하한이 그렇다. wemix 에는 그 규칙
자체가 없으므로 "하한 미만이 거부되는가" 는 wemix 에서 물을 수 없는 질문이다.

**(D) 질문은 공통인데 재료가 체인보다 새롭다.** 케이스가 컨트랙트를 배포해 무언가를
묻는데, 그 컨트랙트를 만든 컴파일러가 가장 낡은 체인보다 새 옵코드를 쓴 경우다. 묻는
질문은 세 체인이 다 답할 수 있다. 답하지 못하는 것은 질문을 담은 재료다.

넷의 처방이 다르다. (A) 는 이름을 코드가 고르게 하면 끝난다. (B) 는 케이스가 정답을
체인별로 적을 자리가 필요하다. (C) 는 **공통이 아니다** — 건너뛰는 것이 옳고, 다만 그것이
결정이었음이 기록에 남아야 한다. (D) 는 재료를 가장 낡은 체인에 맞춰 다시 만들거나,
그러지 않기로 했다면 (C) 와 같이 공통에서 뺀다.

### 2.1 (D) 는 어떻게 드러났나

읽어서는 안 나왔다. B단계에서 공통 91건을 go-wemix 로 강제해 돌리다가 세 번째 건에서
나왔다.

`tests/tc/go-stablenet/vocabulary/03-register-contract.json` 의 배포가 되돌려졌다. 영수증의
`gasUsed` 가 케이스가 준 한도와 정확히 같은 1,500,000 이고 `status` 가 `0x0` 이었다.
962바이트 컨트랙트 배포에 드는 가스는 20만 남짓이니 모자란 것이 아니라, 없는 옵코드를
만나 전액을 태운 모양이다.

세 노드의 인터프리터를 비교해 이유를 찾았다. go-wemix 의 `core/vm/interpreter.go` 는
분기가 `IsMerge` 위로 올라가지 않고, `core/vm/jump_table.go` 에 그 위의 테이블이 없다.
`enable3855`(PUSH0) 는 `core/vm/eips.go` 에 구현만 있고 어떤 포크 테이블도 부르지
않는다. go-wbft 의 `newCroissantInstructionSet()` 과 go-stablenet 의
`newAnzeonInstructionSet()` 은 둘 다 `enable3855` 를 부른다. 문제의 바이트코드는 solc
0.8.28 의 기본 타깃(cancun)으로 컴파일되어 PUSH0 를 45번 쓴다.

공통 91건에서 컨트랙트를 만드는 모든 단계의 바이트코드를 디스어셈블해 세어 보니 걸리는
것은 둘뿐이었다 — 위 케이스와 `contract/005-eth-call-revert-returns-error.json`(PUSH0
9번). `rpc/009-contract-event-emitted.json` 과 `rpc/014-ws-subscribe-logs.json` 의
51바이트짜리는 London 에서 돈다.

**게이트가 막을 수 없었다는 점이 더 중요하다.** 무엇이 필요한지 말할 어휘가 없었다.
`fork:` 는 genesis 의 포크 이름에서 뽑는데 세 체인은 각각 brioche·croissant·anzeon 이라
부르고, 어느 genesis 에도 이더리움 포크 이름이 없다. 그래서 `evm:` 접두사를 선언형으로
더했다(`internal/core/registry/manifest_capability.go`). `precompile:` 이 선언형인 것과
같은 이유다. stablenet 과 wbft 가 `evm:shanghai` 를 선언하고 wemix 는 하지 않는다.

두 케이스는 **공통에서 뺀다.** 재료를 London 으로 다시 컴파일하면 세 체인에서 돌지만,
현대 컴파일러로 만든 컨트랙트를 쓰는 테스트를 공통이라 부르지 않기로 했다. 공통은 89건이
된다.

## 3. 처방 셋

### 3.1 (A) 합의 메서드를 이름이 아니라 역할로 부른다

**이 사실은 이미 코드에 한 곳으로 있다.** `internal/consensus/wbft/wbft.go:29-30` 과
`internal/consensus/poa/poa.go:34-35` 가 각각 `RPCNamespace()` 와 `ValidatorsMethod()`
를 답하고, 매니페스트도 `consensus.rpc_namespace`·`consensus.validators_method` 로 같은
것을 싣는다(`internal/core/registry/manifest.go:154`).

**DSL 만 그것을 부를 길이 없다.** 케이스가 메서드 이름을 글자로 적는다.

그래서 더할 것은 새 지식이 아니라 **경로 하나**다. `read` 의 `method` 자리에 역할 이름을
쓸 수 있게 한다.

```json
{ "do": "read", "source": "rpcCall", "method": "@validators", "save": "vs" }
```

`@validators` 는 실행 시점에 그 체인의 `ValidatorsMethod()` 로 풀린다. 접두 `@` 는 "이것은
메서드 이름이 아니라 역할" 이라는 표시다.

역할은 **매니페스트가 이미 답하는 것만** 둔다. 지금은 `@validators` 하나다. 두 번째
사용처가 생길 때 늘린다.

- 고치는 케이스: `istanbul_getValidators` 를 부르는 5건.
- 안 고쳐지는 것: `istanbul_getWbftExtraInfo` 7건. 이것은 역할이 아니라 wbft 계열
  고유의 정보(`gasTip`)이고 wemix 에 대응물이 없다. (C) 로 간다.

### 3.2 (B) 기대값을 체인별로 적을 자리를 만든다

케이스가 절차는 하나로 쓰고 기대값만 갈라 적는다.

```json
{
  "expect": "txStatus", "hash": "$h",
  "is": "0x1",
  "isPerChain": { "wemix": "0x0" }
}
```

`isPerChain` 에 지금 체인이 있으면 그 값이 `is` 를 대신하고, 없으면 `is` 를 쓴다.

**왜 케이스에 적고 설정으로 빼지 않나.** 설정으로 빼면 "wemix 에서는 0x0" 이라는 사실이
케이스에서 사라진다. 읽는 사람이 두 파일을 들고 맞춰야 하고, 그 둘이 어긋나도 아무도
모른다. 정답은 테스트의 일부다.

**단점.** 케이스가 체인 이름을 다시 안다. §2 가 없애려던 것이 체인 이름 의존인데 이
처방은 그것을 되살린다. 다만 되살리는 자리가 다르다 — `chainPreset` 은 "이 케이스는 이
체인 것" 이라는 소유의 선언이고, `isPerChain` 은 "이 체인에서는 답이 다르다" 는 관찰의
기록이다. 후자는 사실이고 지워지지 않는다.

**대안과 기각 이유.** 케이스를 체인 수만큼 쪼개는 방법이 있다. 지금 상태가 바로 그것이고,
`CT-FEE-002` 에 케이스가 다섯 개인 이유다. 쪼개면 절차가 복사되고, 절차를 고칠 때 한
곳을 빠뜨린다.

### 3.3 (C) 물을 수 없는 질문은 건너뛰되, 건너뛴 것을 세게 한다

`engine:anzeon` 13건은 wemix 에 그 규칙이 없으므로 건너뛰는 것이 옳다. 문제는 건너뛴
것이 통과와 섞여 보이지 않는다는 점이다.

그래서 **공통 영역에는 규칙을 하나 더 건다.** 공통 케이스가 어느 체인에서 SKIP 되면, 그
SKIP 은 케이스가 **미리 선언한** 것이어야 한다.

```json
{ "requires": ["engine:anzeon"], "skipsOn": ["wemix", "wbft"] }
```

선언이 없는데 SKIP 이 나면 그것은 FAIL 이다. 선언이 있는데 SKIP 이 안 나면 그것도 FAIL
이다(선언이 낡았다는 뜻이다).

**이것이 §1.2 의 결함을 닫는 유일한 지점이다.** 나머지 둘은 케이스를 고치지만 이것은
"아무도 안 물었다" 를 "아무도 안 물기로 했다" 로 바꾼다.

## 4. 어디에 무엇을 더하나

`testhelper` 를 먼저 읽고 정했다. 지금 있는 어휘는 action 9개(`sendTx` `waitBlock`
`waitFor` `read` `newAccount` `sendRawTampered` `sendSetCode` `signAuthorization`
`load`)와 assertion 20여 개이고, `derive` 가 `sum`·`diff`·`hex`·`dec`·`abiCall`·`word`·
`quorum` 을 푼다(`internal/testhelper/derived.go`).

| 처방 | 어디 | 왜 거기 |
| --- | --- | --- |
| 3.1 `@역할` 메서드 | `internal/testhelper/read.go` 의 `rpcCall` 경로 | 메서드 이름을 푸는 자리가 거기 하나다. 새 action 을 만들면 `read` 와 하는 일이 같은 것이 둘이 된다 |
| 3.2 `isPerChain` | `internal/dsl/lower_v1.go` 의 `lowerStatement` | `is` 가 `expected` 로 바뀌는 자리가 거기 한 곳이다(`:390-391`) |
| 3.3 `skipsOn` + 검사 | `internal/dsl/spec_v2.go`(자리) + `internal/testengine/capability.go`(판정) | SKIP 을 정하는 함수가 `applicableWithCaps` 하나다 |

### 4.1 체인 사실을 실행기에 넘기는 길 — 선례를 따른다

**처음 쓸 때 여기를 틀렸다.** "실행기가 이미 체인을 안다" 고 적었는데 사실이 아니다.
`interp.ActionCtx` 가 드는 것은 `Env NodeTable`(노드 표)·`Deps`·`Args` 뿐이고
(`internal/dsl/interp/interpreter.go:155-176`), `interp.Deps` 에도 체인 id 도
합의 family 도 없다. 케이스가 도는 동안 "지금 어느 체인인가" 를 물을 자리가 없다.

그런데 **같은 문제를 이미 한 번 푼 선례가 있다.** `Deps.Contracts` 다. 컨트랙트 주소도
체인마다 다르고, 케이스는 이름으로 부른다. 어떻게 되어 있느냐면, 실행을 세울 때
`chainContracts(cfg.Chain)` 가 매니페스트에서 표를 뽑아 `Deps` 에 넣는다
(`internal/testengine/attach.go:184`, `303-309`). 실행기는 체인을 모르고, **체인이 답한
것만** 든다.

(A)·(B)·(C) 도 같은 모양으로 한다. `Deps` 에 필드를 더하고, 실행을 세울 때
매니페스트에서 채운다. (D) 는 여기에 더할 것이 없다. 실행기가 할 일이 아니라 케이스가
쓰는 재료의 문제이고, 게이트 쪽은 `evm:` 접두사 하나로 끝난다.

| 처방 | 더할 것 | 어디서 채우나 |
| --- | --- | --- |
| 3.1 `@역할` | `Deps.ConsensusMethods map[string]string` | `p.Manifest().Consensus`(`ConsensusSpec`: `RPCNamespace`·`ValidatorsMethod`) |
| 3.2 `isPerChain` | 없음 — lowering 에서 풀린다 (아래) | — |
| 3.3 `skipsOn` | 없음 — 게이트가 판정하며 그 자리는 체인을 안다 | — |

### 4.2 `isPerChain` 은 실행이 아니라 lowering 에서 푼다

**두 번째로 틀렸던 자리다.** 처음에는 기대값을 읽는 곳이 `outcome.go` 하나라고 적었는데
아니다. 낮춘 뒤의 `expected` 를 읽는 곳은 최소 다섯이다 — `read.go:93`,
`txprobe.go:56`, `metric.go:50`, `websocket.go:201`, 그리고 `account.go:230` 의 값 인자
목록. 거기를 고치면 다섯 곳을 고치게 된다.

**단일 지점은 그 앞에 있다.** `is` 가 `expected` 로 바뀌는 곳이 `lower_v1.go:390-391`
한 줄이다.

```go
case "is":
    args["expected"] = v
```

그리고 **체인은 이미 그 함수의 한 칸 위에 와 있다.** `lowerStatements(c CaseV2, _
ChainPresetV2, spec *Spec)` 가 preset 을 받고 `_` 로 버리는데, `ChainPresetV2` 에
`Chain string` 필드가 있다(`spec_v2.go`). 그 `_` 를 이름 붙여 `lowerStatement` 로
내려보내면 된다.

`--chain-preset` 으로 갈아끼운 경우도 맞는다. 갈아끼우기는 읽을 때 일어나고
(`ReadFilesWithChainPreset(paths, presetRef)`, `files.go:45`), lowering 은 그 뒤이므로
lowering 이 보는 preset 이 실제로 돌 preset 이다.

**그래서 실행기에 넘길 것은 3.1 의 `ConsensusMethods` 하나뿐이다.** 3.2 는 케이스가
실행기에 닿기 전에 이미 풀려 있다.

## 5. 이 설계가 닫지 못하는 것

정직하게 적는다. 이것들을 다 해도 아래는 남는다.

- **CT 12개는 케이스 자체가 없다.** 새로 써야 하고, 그중 동기화 넷(`CT-NODE-004`~`007`)은
  어느 경로로 블록을 받았는지 보는 기능이 도구에 없어 막혀 있다.
- **`istanbul_getWbftExtraInfo` 7건.** (C) 로 분류하면 wemix 에서 7건이 더 SKIP 된다.
  `gasTip` 에 해당하는 값을 wemix 에서 어떻게 구하는지는 이 문서가 답하지 못한다.
- **`evm:` 는 적어야 듣는다.** 게이트는 케이스가 `requires` 에 적은 것만 본다. 새 옵코드를
  쓰는 바이트코드를 넣고 아무것도 적지 않으면 예전과 똑같이 가스만 태우고 FAIL 한다.
  바이트코드를 기계가 디스어셈블해 필요한 옵코드 집합을 스스로 판정하게 하는 것이 옳은
  마무리인데, 이 문서는 거기까지 가지 않았다.
- **도구 기능 여덟 중 다섯.** 로컬 서명 전송·외부 주소 프로필·세 체인 일괄 실행·실행
  결과 기록·준비물 공유는 이 설계의 밖이다. 대응표는
  [`mainnet-config-worklist.md` §8](../mainnet-config-worklist.md) 에 있다.

## 6. 끝났다고 말할 수 있는 조건

1. 공통 89건을 세 체인에 각각 돌렸을 때, **SKIP 은 전부 `skipsOn` 이 선언한 것**이다.
2. 선언되지 않은 SKIP 이 0건이다.
3. `@validators` 를 쓰는 케이스가 세 체인에서 같은 질문을 묻고 각자의 답을 받는다.
4. 위 셋을 라이브로 확인한 기록이 남는다.
