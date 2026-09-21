# 상태 기계 설계 3 — 전이 표 — [결정]

> **[결정]** 2번이 정한 63개 상태([`state-machine-02-states.md`](state-machine-02-states.md))
> 사이에 **어떤 이동이 허용되는가**를 정한다. 코드는 아직 쓰지 않는다 — 4번이 쓴다.
>
> 열려 있던 질문 셋에 먼저 답한다. 답의 근거는 전부 지금 코드에서 잰 것이다.

## 1. 열린 질문 셋 — 권고와 근거

### 1.1 재시도 전이를 넣을 것인가 → **넣지 않는다**

후보가 둘이었다. 둘 다 읽어 보니 **재시도로 풀리는 것이 아니었다.**

**`BUILD_NODE_TABLE_FAIL_SET_CONTENDED` 는 이미 재시도를 다 쓴 상태다.**
`setlock.go:51~65` 가 10초 동안 폴링한다.

```go
deadline := time.Now().Add(setLockWait)          // setLockWait = 10초
for {
    held, prev, state, err := session.AcquireLock(...)
    if err == nil { return ... }
    if state != session.LockLive || time.Now().After(deadline) {
        return nil, fmt.Errorf("…the server set is being allocated by another run…")
    }
    time.Sleep(setLockPoll)
}
```

이 실패 상태는 **폴링이 끝난 뒤에만** 도달한다. 상태 기계가 또 재시도하면 재시도가 두 겹이
된다.

**`LAUNCH_NODES_FAIL_PORT_BUSY` 는 시간이 풀어 주지 않는다.** `occupancy.go:120~137` 이 쥐고
있는 쪽을 둘로 가른다.

```
recoverable — 이 워크스페이스가 pid 를 기록한 것. `chain stop` 이 푼다
byHand      — 다른 구성이거나 손으로 띄운 것. 사람이 찾아 멈춰야 한다
```

둘 다 **누가 무엇을 해야** 풀린다. 기다림이 아니다.

> **그래서 권고는 "넣지 않는다" 다.** 지금 재시도가 필요한 자리는 이미 그 자리에서 재시도하고
> 있고, 나머지는 재시도로 풀리지 않는다. 이 저장소는 소비자 없는 구조를 미리 배선하다 두 번
> 되돌렸다.
>
> **다만 문을 닫지는 않는다.** 전이 표는 `실패 → 같은 블록의 진입` 을 *문법상* 허용하되, 지금은
> 그런 전이를 **하나도 등록하지 않는다.** 나중에 진짜 재시도할 실패가 생기면 표에 한 줄을
> 더하면 된다.

### 1.2 `COMPARE_NETWORK_DIFFERS` → `OPEN_WORKSPACE` 되돌리기 → **허용하되 한 번만**

**지금도 되돌아간다.** `attach_workspace.go:172` 가 판정하고, `RebuildAll` 이면 `NetStop` 으로
멈춘 뒤 `NetUp` 이 처음부터 다시 조립한다. 새 동작이 아니다.

**판정은 한 번만 일어난다.** `preflightDecision` 호출은 저장소 전체에서 그 한 곳뿐이다.

> **그래서 권고는 "허용하되 한 번" 이다.** 지금 동작과 같고, 무한 루프만 막는다.
>
> manager 가 블록 진입 횟수를 센다. 같은 실행에서 `COMPARE` 에 **두 번째로** 닿으면
> `STATE_CHAIN_FAIL_LOOP` 로 끝낸다. 상한은 표에 적는 값이지 코드에 박는 수가 아니다.

### 1.3 `VERIFY` 를 조립 안에 둘 것인가 → **상태는 안에, 핸들러는 밖에**

`verifyAgainstPlan(suite.go:160)` 이 필요로 하는 것을 보면 답이 나온다.

- `ComposePlan` — `testengine` 의 타입이다
- `nodemonitor` — `chainsetup` 이 **import 하지 않는다**(확인함)

**옮기면 층이 뒤집힌다.** `chainsetup` 이 `testengine` 을 알아야 한다.

> **그래서 권고는 이것이다.** `STATE_CHAIN_VERIFY` 는 CHAIN 영역에 **상태로 둔다.** 그래야
> "조립은 끝났지만 아직 준비되지 않았다" 를 상태가 말한다. 대신 **그 블록의 핸들러는
> `testengine` 이 등록한다.**
>
> manager 는 핸들러를 **누가 등록했는지 모른다.** 블록 값으로 찾아 부를 뿐이다. 상태는 공용
> 어휘이고 핸들러는 소유자의 것이라는 갈림이 여기서 값을 낸다 — 층을 지키면서 상태 하나로
> 이야기가 이어진다.
>
> **따라 오는 규칙.** `chainsetup` 만 쓰는 경로(`chain up`)는 `VERIFY` 핸들러를 등록하지 않고,
> 목표 상태를 `LAUNCH_NODES` 로 둔다. 등록되지 않은 블록에 닿으면 manager 가 거절하므로,
> "핸들러를 안 붙였는데 그 상태로 갔다" 가 조용히 지나가지 않는다.

## 2. 전이 표를 읽는 법

```
S → T        S 에서 T 로 갈 수 있다
S ⇢ T        갈 수 있지만 한 번만 (횟수를 센다)
S ⊣          끝. 루프를 벗어난다
```

세 규칙이 전체에 걸린다.

1. **어느 상태에서든 공통 실패(`0x0F8x`)로 갈 수 있다.** 워크스페이스 자체가 무너지는 것은
   단계를 가리지 않는다.
2. **실패 상태에서 나가는 전이는 지금 하나도 없다.** 실패는 끝이다(§1.1).
3. **진입하지 않은 블록으로 건너뛸 수 없다.** 이것이 지금 `require()` 가 손으로 하는 일을
   대신한다.

## 3. 조립 경로

### 3.1 순서

지금 실행 순서는 `verbs_up.go:145` 의 목록이다.

```go
var upStepNames = []string{"new", "place", "keys", "genesis", "config", "build", "deploy", "init", "start"}
```

전이 표도 같은 선형이다. **바꾸지 않는다** — 순서를 바꾸는 것은 이 작업의 범위가 아니다.

```
STATE_CHAIN_OPEN_WORKSPACE        → STATE_CHAIN_BUILD_NODE_TABLE
STATE_CHAIN_BUILD_NODE_TABLE      → STATE_CHAIN_ENSURE_KEYS
STATE_CHAIN_ENSURE_KEYS           → STATE_CHAIN_BUILD_GENESIS
STATE_CHAIN_BUILD_GENESIS         → STATE_CHAIN_BUILD_NODE_CONFIG
STATE_CHAIN_BUILD_NODE_CONFIG     → STATE_CHAIN_BUILD_NODE_COMMAND
STATE_CHAIN_BUILD_NODE_COMMAND    → STATE_CHAIN_DEPLOY_NODES
STATE_CHAIN_DEPLOY_NODES          → STATE_CHAIN_INIT_NODES
STATE_CHAIN_INIT_NODES            → STATE_CHAIN_LAUNCH_NODES
STATE_CHAIN_LAUNCH_NODES          → STATE_CHAIN_VERIFY
STATE_CHAIN_VERIFY                → STATE_CHAIN_READY
STATE_CHAIN_READY                 ⊣
```

### 3.2 `composeNeeds` 는 어디로 가나

`steps_compose.go:412` 의 표는 **순서가 아니라 의존**이다.

```go
"place":   {"new"},        "keys":    {"new"},
"genesis": {"place"},      "config":  {"place", "keys"},
"build":   {"place", "keys"},
"deploy":  {"place", "genesis", "config"},
"init":    {"deploy"},     "start":   {"init"},
```

선형 순서와 이 의존은 **다른 것**이다. `keys` 는 `new` 만 필요한데 순서상 `place` 뒤에 온다.

> **의존 표는 전이 표가 대신하지 않는다. 전이 표를 검사하는 데 쓴다.**
>
> 래칫 하나가 이것을 붙든다: **선형 순서의 각 상태에 닿았을 때, 그 단계의 의존이 전부 이미
> 지나간 상태인가.** 지금은 그 검사가 실행 중에 `require()` 로 일어나고, 그때는 이미 늦다.
> 표 대 표 비교는 **빌드 때** 일어난다.

### 3.3 세부 전이

세부가 있는 블록만 적는다. 나머지는 진입에서 바로 다음 블록으로 간다.

**`ENSURE_KEYS` (0x1300)** — `steps_keys.go:93~121` 이 출처 셋을 가른다.

```
ENSURE_KEYS → ENSURE_KEYS_FROM_PRESET      keyPreset
            → ENSURE_KEYS_GENERATED        generate
            → ENSURE_KEYS_FROM_BLUEPRINT   declared
            → …_FAIL_UNKNOWN_SOURCE        그 밖

ENSURE_KEYS_FROM_PRESET     → BUILD_GENESIS
ENSURE_KEYS_GENERATED       → BUILD_GENESIS
ENSURE_KEYS_FROM_BLUEPRINT  → BUILD_GENESIS
```

**`BUILD_GENESIS` (0x1400)** — 갈래가 둘 있고 서로 곱해진다. 어디서 왔든(`템플릿`/`기존`)
포크가 선언돼 있으면 적용하고, 바이너리별 변형이 있으면 쓴다.

```
BUILD_GENESIS → BUILD_GENESIS_FROM_TEMPLATE      기본
              → BUILD_GENESIS_FROM_EXISTING      steps_compose.go:238

BUILD_GENESIS_FROM_TEMPLATE → BUILD_GENESIS_FORK_APPLIED      steps_genesis.go:202
                            → BUILD_GENESIS_VARIANTS_WRITTEN  steps_genesis.go:241
                            → BUILD_NODE_CONFIG               포크도 변형도 없다
BUILD_GENESIS_FROM_EXISTING → (같음)

BUILD_GENESIS_FORK_APPLIED      → BUILD_GENESIS_VARIANTS_WRITTEN
                                → BUILD_NODE_CONFIG
BUILD_GENESIS_VARIANTS_WRITTEN  → BUILD_NODE_CONFIG
```

**`DEPLOY_NODES` (0x1700)** — 대상이 로컬인지 원격인지로 갈린다(`steps_compose.go:107`).

```
DEPLOY_NODES → DEPLOY_NODES_VERIFIED_LOCAL  → INIT_NODES
             → DEPLOY_NODES_SHIPPED_REMOTE  → INIT_NODES
```

**`LAUNCH_NODES` (0x1900)** — 집안이 단계 수를 정한다. `wbft` 는 하나(`wbft.go:64`), `poa` 는
`boot` 다음 생산자 수만큼 `join-nodeN`(`poa.go:94~112`).

```
LAUNCH_NODES            → LAUNCH_NODES_PHASE_LAUNCHING
LAUNCH_NODES_PHASE_LAUNCHING → LAUNCH_NODES_PHASE_ACTIONS   뒷작업이 있다
                             → LAUNCH_NODES_PHASE_DONE      없다
LAUNCH_NODES_PHASE_ACTIONS   → LAUNCH_NODES_PHASE_DONE
LAUNCH_NODES_PHASE_DONE      → LAUNCH_NODES_PHASE_LAUNCHING 다음 단계가 있다
                             → VERIFY                        전부 끝
```

**여기가 이 설계에서 값이 가장 큰 자리다.** 지금은 `phases.go` 안의 루프라, poa 망이 세 번째
`join` 에서 죽어도 기록에 남는 것은 `start` 뿐이다. 상태로 올리면 **몇 번째 단계에서 멈췄는지가
상태값**이다.

**`VERIFY` (0x1A00)**

```
VERIFY → VERIFY_PRODUCING             → READY
       → VERIFY_HALTED_AS_DECLARED    → READY     genesis.haltsAt 대로 멈췄다
       → …_FAIL_PLAN_MISMATCH
       → …_FAIL_NOT_PRODUCING
```

## 4. 인수 경로

시작 상태가 셋 중 하나다. CLI 의 `switch` 네 갈래가 여기로 온다.

```
STATE_CHAIN_ADOPT → ADOPT_BY_RPC            --rpc
                  → ADOPT_BY_WORKSPACE      --attach --workspace-dir
                  → ADOPT_BY_DECLARATION    env.attach

ADOPT_BY_RPC          → COMPARE
ADOPT_BY_WORKSPACE    → COMPARE
ADOPT_BY_DECLARATION  → COMPARE
```

`COMPARE` 의 세부 넷이 `preflight.Verdict` 넷이다(`preflight.go:100~111`).

```
COMPARE → COMPARE_SAME                → READY               Reuse
        → COMPARE_NODES_DIFFER        → LAUNCH_NODES        RebuildNodes
        → COMPARE_NETWORK_DIFFERS     ⇢ OPEN_WORKSPACE      RebuildAll  (한 번만)
        → COMPARE_NOTHING_COMPOSED    ⇢ OPEN_WORKSPACE      Compose     (한 번만)
```

`COMPARE_NODES_DIFFER` 가 `LAUNCH_NODES` 로 가는 것은 지금 동작 그대로다 — `RebuildNodes` 는
`NetRestart` 를 노드별로 부르지 조립을 다시 하지 않는다(`attach_workspace.go:177~184`).

## 5. 목표 상태

`UpStage` 가 하던 일이다. `deploy` 로 멈추는 실행은 목표를 `DEPLOY_NODES` 로 준다.

```go
m.RunUntil(STATE_CHAIN_DEPLOY_NODES)   // chain up --stage=deploy
m.RunUntil(STATE_CHAIN_LAUNCH_NODES)   // chain up  (chainsetup 만 쓰는 경로)
m.RunUntil(STATE_CHAIN_READY)          // testengine
```

**루프 탈출 조건이 셋이 된다.**

```
목표 상태에 닿았다     → 정상 종료
실패 상태에 들어갔다   → 오류 종료. 상태값이 무엇이 왜 실패했는지 말한다
전이 상한을 넘었다     → 오류 종료. STATE_CHAIN_FAIL_LOOP
```

## 6. 이 표가 지금 코드에서 없애는 것

| 지금 | 어디 | 전이 표가 대신하는 방식 |
|---|---|---|
| `require()` 의 선행 조건 검사 | `steps_compose.go:436`, 6단계가 공유 | 진입하지 않은 블록으로 못 간다 |
| "run `chain place` first" 문구 7개 | `steps_keys.go:65·145`, `steps_lifecycle.go:116·119·220` 등 | 위와 같음 |
| `if stage == UpDeploy && …` | `verbs_up.go:351` 루프 안 | 목표 상태 |
| `if reuseMode && name == "keys"` | `verbs_up.go:361` 루프 안 | `COMPARE` 블록 |
| `upStepNames` 상수 배열 | `verbs_up.go:145` | 전이 표 |
| CLI 의 `switch` 네 갈래 | `suitecmd/run.go:79~93` | 시작 상태 |

**루프에서 `if` 둘이 빠진다.** 지금 `netUpFrom` 의 루프는 순회 중에 "여기까지만" 과 "이 단계
뒤에 재사용 판정" 을 문자열 비교로 끼워 넣는다. 둘 다 상태로 올라간다.

## 7. 아직 정하지 않은 것

- **운영 영역(0x3000)의 전이.** 2번이 블록 자리만 잡았고 세부·실패를 안 쟀다. 조립과 같은
  방법으로 잰 뒤에 쓴다.
- **테스트 영역(0x8000)의 전이.** 6번에서 한다.
- **전이 상한의 값.** "한 번만" 은 규칙이고 숫자는 표에 적는다. 지금 동작이 1회이므로 1이
  맞지만, 재구성 뒤 재비교를 허용할지에 따라 2가 될 수 있다.

## 8. 다음 (4번)

manager 와 이 전이 표만 먼저 만든다. **핸들러는 비운다.** 기존 경로는 건드리지 않는다.

4번이 만들 것은 셋이다.

1. `State` 타입과 63개 상수, `IsFailure`·`BlockOf`
2. `allowed` 전이 표와 `Manager.Request` 의 거절
3. 표를 붙드는 래칫 — **모든 상태가 전이 표에 나오는가**(도달 불가 상태 금지),
   **선형 순서가 `composeNeeds` 의 의존을 어기지 않는가**(§3.2)

핸들러가 비어 있어도 이 셋은 검사할 수 있다. 상태가 실제로 돌기 전에 표가 맞는지 먼저 안다.
