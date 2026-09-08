# DSL 작성 가이드

이 문서는 사람이 정리한 요약이 아니라 **코드에서 뽑은 것**이다. `internal/testhelper`·`internal/dsl`·`internal/testengine`·`internal/core/session` 을 tree-sitter-go 로 파싱해 등록 지점(`RegisterAction`/`RegisterAssertion`/`RegisterReader`)을 찾고, 각 구현의 본문에서 읽는 인자 키를 모았다. 그래서 코드가 바뀌면 이 목록도 바뀐다.

등록된 어휘는 47개다. 액션 19, 단언 11, 리더 17.

---

## 1. 두 가지 문서 형태

DSL 문서는 `schemaVersion` 으로 갈린다. 어느 쪽을 쓸지는 **체인을 누가 세우는가**로 정해진다.

`schemaVersion: "1"` 은 스펙이다. 문서 안에 `chain` 을 직접 적고, 이미 떠 있는 체인에 붙어 돌 수 있다. 노드를 멈추거나 바꾸는 동작은 쓰지 않는다.

`schemaVersion: "2"` 는 케이스다. `env` 로 환경 선언을 이름으로 부르고, 그 선언대로 네트워크를 세운 뒤 돈다. 노드를 멈추고 살리고 바꾸는 동작은 여기서만 쓸 수 있다.

환경 선언은 `kind: "env"` 인 별도 문서다. `tests/tc/env/` 에 두면 아래 어느 깊이의 케이스든 id 로 찾아 쓴다.

샘플은 `tests/tc/samples/` 에 있다. `01-sample-spec.json` 이 v1, `02-sample-case.json` 이 v2 이고 `tests/tc/env/sample.env.json` 이 그 환경이다.

> v2 는 모르는 필드를 거부한다. v1 은 통과시킨다. v2 케이스에 `description` 을 넣으면 파싱이 실패한다 — 설명은 파일 밖(README)에 적는다.

---

## 2. 문장 쓰는 법

v1 과 v2 는 같은 어휘를 다른 표기로 쓴다. 실행기는 둘을 같은 문장 목록으로 낮춘다.

| | v1 | v2 |
|---|---|---|
| 동작 | `{"sendTx": {…}}` | `{"do": "sendTx", …}` |
| 읽기 | `{"read": {"source": "balanceAt", …}}` | `{"do": "read", "source": "balanceAt", …}` |
| 판정 | `{"assert": "txStatus", …, "expected": …}` | `{"expect": "txStatus", …, "is": …}` |

v2 의 `is` 는 낮추는 단계에서 `expected` 로 바뀐다. 둘 다 통하지만 v2 에서는 `is` 를 쓴다.

`expect` 는 자리에 따라 뜻이 다르다. 문장의 머리로 오면 판정이고, `do` 가 있는 문장에 붙으면 그 동작의 기대 결과다 — `{"do":"sendTx", "expect":"receipt"}` 처럼.

`rpc` 는 `rpcCall` 의 별칭이다.

---

## 3. 값을 잇는 법

어느 문장이든 `save: "이름"` 을 붙이면 결과가 그 이름에 묶인다. 뒤 문장은 `$이름` 으로 부른다. 문자열 안에 끼워 넣을 때는 `${이름}` 을 쓴다.

```json
{ "do": "read", "source": "rpcCall", "method": "eth_getTransactionReceipt",
  "params": ["$txHash"], "select": "logs.#", "save": "logCount" },
{ "do": "read", "source": "derive", "op": "diff",
  "of": ["$logCount", "1"], "save": "lastIdx" },
{ "do": "read", "source": "rpcCall", "method": "eth_getTransactionReceipt",
  "params": ["$txHash"], "select": "logs.${lastIdx}.topics.0", "save": "lastTopic" }
```

`select` 는 점으로 잇는 경로다. 배열은 숫자로 들어가고(`peers.0.id`), `#` 은 길이를 준다.

`params` 안의 `"@latest"` 는 호출 시점의 head 블록 번호로 바뀐다. `waitFor` 안에서는 매 폴링마다 다시 바뀌므로 head 를 따라간다.

리터럴에 이름을 붙이고 싶으면 `derive` 를 값 하나로 쓴다. `sum` 은 값이 하나면 그 값을 그대로 돌려주므로 상수 선언이 된다.

```json
{ "read": { "source": "derive", "op": "sum", "of": ["21000"],
            "save": "transferGas" } }
```

---

## 4. 노드와 계정 고르기

### 노드

`on` 이 노드를 고른다. 세 가지 형태가 있다.

| 형태 | 예 | 뜻 |
|---|---|---|
| 번호 | `node1`, `node5` | 노드 표의 1번부터 |
| 역할+서수 | `bp1`, `en1` | 그 역할의 첫 번째. **1부터 센다** |
| 역할:색인 | `bp:0`, `en:any` | 콜론 뒤는 **0부터 센다**. `any` 는 아무거나 |

역할 이름은 `bp`(= `validator`), `en`(= `endpoint`), `pn`, `boot` 이다.

### 계정

주소 자리에는 세 가지를 쓸 수 있다. 생김새로 구분하므로 따로 선언하지 않는다.

| 쓰는 것 | 뜻 | 서명하는 곳 |
|---|---|---|
| `0x…` | 주소를 그대로 적은 것 | — |
| `node1`, `node2` … | 그 노드가 가진 계정 | 노드가 서명한다 |
| `dev1` 같은 이름 | 키 세트에 있는 신원 | 하네스가 서명하고 raw 로 보낸다 |
| `faucet` | `node1` 의 별칭. genesis 로 자금을 받은 계정 | 노드가 서명한다 |

`dev1` 같은 개발용 계정은 env 의 `accounts` 에서 선언한다. 선언하면 키가 키 링 아래에 저장되고, **같은 이름이 다음 실행에서도 같은 주소를 가리킨다.** `fund` 를 적으면 네트워크의 자금 계정에서 그만큼 보내고 잔액이 실제로 도착할 때까지 기다린 뒤 첫 문장이 돈다.

```json
"accounts": { "dev1": { "fund": "0x8AC7230489E80000" } }
```

이름이 키 세트에 없으면 그 자리에서 실패하고 무엇이 있는지 알려 준다. 0 주소로 조용히 떨어지지 않는다.

---

## 5. 비교 연산

`compare` 에 쓸 수 있는 것은 16개다.

`Equal` `NotEqual` `EqualCI` `Len` `Greater` `GreaterOrEqual` `Less` `LessOrEqual` `Contains` `NotContains` `Regexp` `True` `False` `Nil` `NotNil` `ElementsMatch`

`derive` 의 `op` 는 다섯 가지다. `sum` `diff` `word` `quorum` `abiCall`.

---

## 6. 어휘 전체

각 항목의 인자는 구현 본문에서 읽는 키를 뽑은 것이다. `save` 는 실행기가 처리하므로 여기 나오지 않지만 어느 문장에나 붙일 수 있다.

### 동작 (do) — 19개

| 이름 | 인자 | 스펙 사용 | 구현 |
|---|---|---:|---|
| `deployContract` | `bytecode`, `data`, `gas`, `on`, `value` | 2 | `internal/testhelper/assets.go` |
| `faucet` | `amount`, `gas`, `on`, `to` | 0 | `internal/testhelper/assets.go` |
| `healPartition` | `groups` | 2 | `internal/testhelper/fault.go` |
| `load` | `blocks`, `on` | 4 | `internal/testhelper/builtins.go` |
| `newAccount` | `saveKey` | 29 | `internal/testhelper/builtins.go` |
| `partition` | — | 2 | `internal/testhelper/fault.go` |
| `readNodeLog` | `maxBytes`, `on` | 4 | `internal/testhelper/fault.go` |
| `registerContract` | `data`, `gas`, `on`, `to`, `value` | 0 | `internal/testhelper/assets.go` |
| `restartNode` | `on` | 2 | `internal/testhelper/fault.go` |
| `sendRawTampered` | `feePayerKey`, `on`, `senderKey`, `to`, `value`, `which` | 4 | `internal/testhelper/builtins.go` |
| `sendSetCode` | `authorityKey`, `delegate`, `key`, `on` | 1 | `internal/testhelper/builtins.go` |
| `sendTx` | `accessList`, `data`, `from`, `gas`, `key`, `on`, `pollInterval`, `timeout`, `to`, `value`, `wait` | 83 | `internal/testhelper/builtins.go` |
| `signAuthorization` | `authorityKey`, `delegate`, `on` | 1 | `internal/testhelper/builtins.go` |
| `startNode` | `on` | 2 | `internal/testhelper/fault.go` |
| `stopNode` | `on` | 4 | `internal/testhelper/fault.go` |
| `swapNode` | `binary`, `config`, `genesisOverlay`, `on`, `purpose` | 3 | `internal/testhelper/fault.go` |
| `waitBlock` | `on`, `pollInterval`, `target`, `timeout` | 41 | `internal/testhelper/builtins.go` |
| `waitFor` | `compare`, `expected`, `on`, `pollInterval`, `source`, `timeout` | 18 | `internal/testhelper/builtins.go` |
| `wsOpen` | `event`, `params`, `save` | 1 | `internal/testhelper/derived.go` |

### 판정 (expect / assert) — 11개

| 이름 | 인자 | 스펙 사용 | 구현 |
|---|---|---:|---|
| `blockAdvance` | `on`, `pollInterval`, `timeout` | 9 | `internal/testhelper/builtins.go` |
| `blockHalt` | `maxAdvance`, `on`, `within` | 1 | `internal/testhelper/builtins.go` |
| `blockInterval` | `blocks`, `maxSeconds`, `minSeconds`, `on` | 1 | `internal/testhelper/builtins.go` |
| `blockStalled` | `on`, `pollInterval`, `timeout` | 1 | `internal/testhelper/builtins.go` |
| `callError` | `data`, `on`, `to` | 1 | `internal/testhelper/builtins.go` |
| `methodPresent` | `method`, `on`, `params` | 1 | `internal/testhelper/builtins.go` |
| `metric` | `compare`, `expected`, `name` | 0 | `internal/testhelper/builtins.go` |
| `sameBlockHash` | `block`, `on` | 10 | `internal/testhelper/builtins.go` |
| `txMined` | `expected`, `on` | 2 | `internal/testhelper/builtins.go` |
| `wsCollected` | `count`, `sub`, `timeout` | 1 | `internal/testhelper/derived.go` |
| `wsSubscribe` | `count`, `event`, `expected`, `params`, `timeout` | 1 | `internal/testhelper/derived.go` |

### 읽기 source (read / waitFor) — 17개

| 이름 | 인자 | 스펙 사용 | 구현 |
|---|---|---:|---|
| `balanceAt` | `address` | 12 | `internal/testhelper/read.go` |
| `baseFee` | — | 7 | `internal/testhelper/read.go` |
| `blockNumber` | — | 48 | `internal/testhelper/read.go` |
| `call` | `data`, `to` | 46 | `internal/testhelper/read.go` |
| `chainId` | — | 9 | `internal/testhelper/read.go` |
| `codeAt` | `address` | 7 | `internal/testhelper/read.go` |
| `contractChecksum` | `address`, `bytecode`, `data` | 0 | `internal/testhelper/read.go` |
| `createAddress` | `deployer`, `from`, `nonce` | 0 | `internal/testhelper/read.go` |
| `derive` | `of`, `op` | 56 | `internal/testhelper/read.go` |
| `estimateGas` | `data`, `from`, `to` | 2 | `internal/testhelper/read.go` |
| `gasPrice` | — | 6 | `internal/testhelper/read.go` |
| `logs` | `address`, `fromBlock`, `index`, `select`, `toBlock`, `topics` | 15 | `internal/testhelper/read.go` |
| `nonceAt` | `address` | 2 | `internal/testhelper/read.go` |
| `peerCount` | — | 3 | `internal/testhelper/read.go` |
| `receiptLog` | `address`, `hash`, `index`, `select`, `topic`, `topic0` | 31 | `internal/testhelper/read.go` |
| `rpcCall` | `method`, `params`, `select` | 75 | `internal/testhelper/read.go` |
| `txStatus` | `hash` | 60 | `internal/testhelper/read.go` |

"스펙 사용" 은 `tests/tc` 아래 문서에서 그 이름이 나온 파일 수다. 0인 것은 구현은 있는데 아직 아무 테스트도 쓰지 않는 어휘다.

---

## 7. 쓰기 전에 확인할 것

문서를 쓰면 돌리기 전에 검사한다. 네트워크 없이 파싱과 참조 해석까지 본다.

```
chainbench validate tests/tc/<path>.json
```

`UNRESOLVED` 가 나오면 없는 어휘를 불렀거나, 정의되지 않은 `$이름` 을 참조했거나, 나중에 저장할 값을 먼저 쓴 것이다. 참조는 실행 순서대로 검사한다 — preActions, steps, assertions, postActions 순이다.

`tests/tc` 아래 전체를 한 번에 보려면 다음처럼 한다.

```
chainbench validate $(find tests/tc -name '*.json' ! -name '*.env.json')
```

`go test ./cmd/chainbench/ -run TestValidateCmd` 가 같은 검사를 CI 에서 돌린다. 새 문서를 넣으면 이 테스트가 자동으로 집어 간다.
