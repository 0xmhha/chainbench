# 상태 기계 설계 1 — 지금 실패하는 자리를 전부 뽑는다 (2026-09-21) — [측정]

> **[측정]** 잰 것만 적는다. 상태 표는 이 문서가 정하지 않는다 — 2번이 정하고, 이 문서가 그
> 입력이다.
>
> **재는 법.** `go/parser` 로 `internal/chainsetup` 의 비-테스트 파일 34개를 읽어 (1) 모든
> 실패 지점과 (2) 패키지 안 호출 관계를 뽑았다. 단계마다 진입 함수가 있으므로(`ChainAllocate`,
> `ChainKeys` …) 거기서 호출 그래프를 훑어 **도달하는 실패를 그 단계에 귀속**시켰다.
> 도구는 세션 스크래치패드의 `failures.go` 이고, 산출물은 `failures.json` 이다.

## 0. 왜 이것을 먼저 하나

첫 상태 표가 틀렸던 이유가 **실패에 자리를 주지 않아서**였다. `BASE + 0` 을 진입 상태로 쓰고
`+1` 부터 다음 단계를 붙였더니, 한 단계가 실패했을 때 갈 상태가 없었다.

고치려면 블록마다 실패 상태를 몇 개 둘지 알아야 하고, 그 수는 **지금 코드가 실제로 구분하는
실패의 수**에서 나온다. 추측하면 또 모자란다.

## 1. 전체 수치

| | 건수 |
|---|---|
| 실패 지점 전체 | **481** |
| ├ 자기 말이 있는 것 (`fmt.Errorf` · `errors.New`) | 230 |
| └ 그대로 올려보내는 것 (`return …, err`) | 251 |
| 조립 9단계 중 **한 단계에만** 속하는 것 | **118** |
| 여러 단계가 공유하는 것 | 16 |
| 조립 단계에서 안 닿는 것 (운영 동사·접근자) | 96 |

118건이 상태 표의 입력이다. 나머지는 2번에서 따로 다룬다.

## 2. 118건을 핸들러가 다르게 굴 부류로 나눈다

**한 메시지에 한 상태를 주면 118개가 된다.** 그건 상태가 아니라 메시지 목록이다. 기준은
**"핸들러가 이 둘을 다르게 처리하는가"** 하나다.

읽어 보니 일곱 부류였다.

| 부류 | 무엇 | 핸들러가 무엇을 하나 |
|---|---|---|
| `PRECOND` | 앞 단계가 안 끝났다 | **상태 기계에서는 일어나지 않는다.** 전이 표가 막는다 |
| `BAD_INPUT` | 사람이 적은 것이 틀렸다 | 재시도 불가. 즉시 중단 |
| `MISMATCH` | 선언과 실제가 어긋난다 | 중단. 양쪽 값을 보여 준다 |
| `UNABLE` | 대상이 그 일을 못 한다 | 중단. 구성 문제다 |
| `CONTENDED` | 다른 실행이 쥐고 있다 | **기다렸다 재시도** |
| `FOREIGN` | 거기 있는 것이 우리 것이 아니다 | 다시 만들거나 중단 |
| `RESOURCE` | 파일·망 접근이 실패했다 | 재시도 가능할 수 있다 |

그리고 **순수 래핑 53건**이 따로 있다. `"keys: %w"` 처럼 접두사만 붙이고 자기 말이 없는 것들이라
새 실패가 아니다. 상태를 주지 않는다.

## 3. 단계별 실패 상태 수 — 상태 표의 입력

```
단계                      래핑   PRECOND  BAD_INPUT  MISMATCH  UNABLE  CONTENDED  FOREIGN  RESOURCE   실패상태
OPEN_WORKSPACE             1         ·         1         ·        ·         ·        ·         ·          1
BUILD_NODE_TABLE           2         ·         1         ·        ·         1        ·         ·          2
ENSURE_KEYS                3         3         5         2        ·         ·        ·         2          4
BUILD_GENESIS             13         ·         4         6        2         ·        1        10          5
BUILD_NODE_CONFIG          7         ·         3         ·        ·         ·        1         2          3
BUILD_NODE_COMMAND         5         ·         2         ·        ·         ·        ·         ·          1
DEPLOY_NODES               3         ·         ·         1        ·         ·        ·         1          2
INIT_NODES                 3         2         ·         ·        1         ·        ·         1          3
LAUNCH_NODES              16         2         ·         3        4         1        ·         3          5
─────────────────────────────────────────────────────────────────────────────────────────────────────
합계                      53         7        16        12        7         2        2        19         26
```

**맨 오른쪽 칸이 답이다.** 블록마다 실패 상태를 그만큼 둔다. 가장 많은 곳이 다섯이므로
`+0x80 ~ 0xFF` 128칸은 넉넉하다.

## 4. 이 표가 말하는 것 넷

### 4.1 `PRECOND` 7건은 상태 기계가 통째로 없앤다

지금 코드에 이런 것들이 흩어져 있다.

```
steps_keys.go:65        keys: node count unknown — run `chain place` first or pass --nodes
steps_keys.go:145       keys: %w — run `chain place` first
steps_lifecycle.go:116  init: no node table — run `chain place` first
steps_lifecycle.go:119  init: no genesis — run `chain genesis` first
steps_lifecycle.go:220  start: no node table — run `chain place` first
phases.go:254           start: %d thing(s) the launch needs are missing on the target
```

**전부 "앞 단계가 안 끝났다" 를 각자 검사하고 각자 문구를 쓴다.** 전이 표가 있으면
`BUILD_NODE_TABLE` 을 지나지 않고 `ENSURE_KEYS` 로 갈 수 없으므로, 이 검사도 이 문구도
필요 없어진다.

절반은 이미 `verb_needs.go` 의 표가 덮고 있다. 나머지가 이 일곱이다.

### 4.2 `BAD_INPUT` 16건은 루프 안에서 실패하면 안 된다

```
verbs_steps.go:326   genesis override expects key=value, got %q
steps_compose.go:175 bad --set %q (want key=value or a bare boolean key)
provenance.go:22     config override %q must be key=value
provenance.go:85     launch scope %q must be %s
steps_keys.go:121    keys: unknown source %q (want keyPreset, generate or declared)
verbs_steps.go:149   allocate: a blueprint and a topology both describe the layout — give one
```

사람이 적은 것이 틀렸다는 뜻이다. **여섯 단계에 흩어져 있고, 전부 루프가 한참 돈 뒤에 나온다.**
`--set` 오타 하나 때문에 워크스페이스를 열고 노드 표를 만들고 키를 확보한 뒤에 죽는다.

상태 기계에서는 **루프 진입 전 요청 검증**이 맡는다. `STATE_CHAIN_IDLE` 의 핸들러가 한 번에
본다. 지금 `planUp` 이 그 자리인데 세 가지만 본다(디렉터리·stage·topology 키).

### 4.3 `BUILD_GENESIS` 와 `LAUNCH_NODES` 가 절반을 차지한다

118건 중 65건이 이 둘이다. 그럴 만하다.

- `BUILD_GENESIS` 는 **두 체인을 다룬다.** 하드포크를 건너는 망은 넘겨주는 체인의 genesis 를
  읽어 넘겨받는 체인의 것을 만든다. 실패 23건 중 여섯이 그 짝짓기다 — "그 포크를 봉인하는
  바이너리가 자기 체인을 말하지 않는다", "그 바이너리를 쓰는 노드가 없어 넘겨줄 상대가 없다".
- `LAUNCH_NODES` 는 **대상 머신을 만진다.** 실패 13건 중 넷이 바이너리 자체를 못 찾는 것이고
  (`binary: none is set`, `not on the target's PATH`), 하나는 워크스페이스 밖에서 이미 돌고
  있는 프로세스와의 충돌이다.

**두 블록은 세부 상태도 더 필요하다.** 나머지 일곱은 진입·완료·실패 한둘이면 된다.

### 4.4 래핑 53건이 실패를 멀리 보낸다

`"keys: %w"` 같은 순수 래핑이 53건이다. 실패가 일어난 자리와 이름이 붙는 자리가 떨어져 있다는
뜻이고, 그래서 지금 기록에 `init failed: chainsetup: chain up: init: ...` 처럼 접두사가 겹쳐
남는다.

상태 기계에서는 **상태값이 자리를 말하므로** 래핑이 그 일을 안 해도 된다.

## 5. 아직 안 정한 것

- **공유 실패 16건.** 여러 단계가 닿는 도우미 안에 있다(`require`, `eachMachine`, `Open`).
  블록마다 복제할지, 공유 실패 블록을 하나 둘지는 2번에서 정한다.
- **조립 밖 96건.** 운영 동사(`Stop`·`Rm`·`Restart`·`SwapNode`·`CrossFork`·`Hardfork`)와
  접근자의 실패다. `BASE_CHAIN_OP_*` 블록을 설계할 때 같은 방법으로 잰다.
- **재시도 정책.** `CONTENDED` 2건과 `RESOURCE` 19건이 재시도 후보인데, 지금 코드는 전부
  즉시 중단한다. 재시도를 넣을지는 상태 표가 아니라 별도 판단이다.

## 6. 다음 (2번)

이 표를 입력으로 상태 표를 확정한다. 블록마다:

```
BASE + 0x00          진입
BASE + 0x01 ~ 0x7F   세부 진행   ← BUILD_GENESIS·LAUNCH_NODES 만 여럿
BASE + 0x80 ~ 0xFF   실패        ← 위 표의 맨 오른쪽 칸만큼
```

단계 이름은 확정됐다.

```
OPEN_WORKSPACE · BUILD_NODE_TABLE · ENSURE_KEYS · BUILD_GENESIS · BUILD_NODE_CONFIG
BUILD_NODE_COMMAND · DEPLOY_NODES · INIT_NODES · LAUNCH_NODES
```
