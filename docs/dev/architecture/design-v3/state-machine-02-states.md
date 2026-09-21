# 상태 기계 설계 2 — 상태 표 — [결정]

> **[결정]** 1번의 측정([`state-machine-01-failures.md`](state-machine-01-failures.md))을
> 입력으로 삼아 상태를 확정한다. 전이는 이 문서가 정하지 않는다 — 3번이 정한다.
>
> 세부 상태는 지어내지 않았다. **지금 코드가 실제로 가르는 갈래**만 상태가 된다. 갈래의
> 근거를 상태마다 파일·줄로 적는다.

## 1. 값 규약

```
0xAB00 꼴의 값 하나가 블록이고, 블록 하나가 단계 하나다.

  BASE + 0x00          진입. 이 단계를 시작한다
  BASE + 0x01 ~ 0x7F   세부 진행. 코드가 실제로 가르는 갈래만
  BASE + 0x80 ~ 0xFF   실패. 1번이 센 만큼
```

값 하나로 두 가지를 판정할 수 있다.

```go
func IsFailure(s State) bool { return s&0xFF >= 0x80 }
func BlockOf(s State) State  { return s &^ 0xFF }
```

**왜 이 배치인가.** 실패를 상위 절반에 몰면 manager 가 상태 **이름을 몰라도** "지금 실패인가"
를 답한다. 블록 마스크로 "어느 단계의 실패인가" 도 답한다. 핸들러를 블록 단위로 등록할 수
있는 것이 여기서 나온다.

**간격.** 블록은 0x100(256칸), 영역은 0x1000 이다. 세부가 127칸·실패가 128칸이고, 지금 가장
많은 블록이 세부 4·실패 5 이므로 스무 배 넘게 남는다.

## 2. 이름 규약

```
STATE_<영역>_<단계>[_<세부>][_FAIL_<사유>]
```

| 자리 | 값 |
|---|---|
| 영역 | `CHAIN` · `TEST` |
| 단계 | 블록 이름 |
| 세부 | 없으면 진입 |
| 사유 | 실패일 때만 |

단계 이름은 **하는 일**로 짓는다. `PLACED` 같은 결과형은 무슨 일을 하는 단계인지 말하지 않는다.

## 3. 영역과 블록 배치

```
0x1000 ~ 0x1FFF   CHAIN 조립        아홉 블록 + 검증 + 완료
0x2000 ~ 0x2FFF   CHAIN 인수        이미 있는 체인을 받는다
0x3000 ~ 0x3FFF   CHAIN 운영        조립이 끝난 뒤의 동작
0x0F00 ~ 0x0FFF   CHAIN 공통 실패   어느 상태에서도 빠질 수 있는 것
0x8000 ~ 0x8FFF   TEST              testengine 의 생애주기
```

## 4. 조립 블록 아홉

### 4.1 `BASE_CHAIN_OPEN_WORKSPACE = 0x1100`

워크스페이스를 열고 요청을 기록한다. 지금 `NetNew` 다.

| 값 | 상태 | 근거 |
|---|---|---|
| +0x00 | `STATE_CHAIN_OPEN_WORKSPACE` | 진입 |
| +0x80 | `…_FAIL_NO_CHAIN` | `workspace_new.go:66` — `--chain or --manifest is required` |

세부 없음. 이 단계는 갈래가 없다.

### 4.2 `BASE_CHAIN_BUILD_NODE_TABLE = 0x1200`

노드 표를 만든다: 역할·경로·결정적 포트. 지금 `NetAllocate` 다.

| 값 | 상태 | 근거 |
|---|---|---|
| +0x00 | `STATE_CHAIN_BUILD_NODE_TABLE` | 진입 |
| +0x80 | `…_FAIL_TWO_LAYOUTS` | `verbs_steps.go:149` — blueprint 와 topology 를 둘 다 줬다 |
| +0x81 | `…_FAIL_SET_CONTENDED` | `setlock.go:61` — 다른 실행이 서버 집합을 잡고 있다. **재시도 대상** |

### 4.3 `BASE_CHAIN_ENSURE_KEYS = 0x1300`

키 집합이 있는지 보고 없으면 만든다. 지금 `NetKeys` 다.

| 값 | 상태 | 근거 |
|---|---|---|
| +0x00 | `STATE_CHAIN_ENSURE_KEYS` | 진입 |
| +0x01 | `…_FROM_PRESET` | `steps_keys.go:121` 의 `keyPreset` 갈래 |
| +0x02 | `…_GENERATED` | 같은 줄의 `generate` 갈래 |
| +0x03 | `…_FROM_BLUEPRINT` | 같은 줄의 `declared` 갈래 |
| +0x80 | `…_FAIL_UNKNOWN_SOURCE` | `steps_keys.go:121` |
| +0x81 | `…_FAIL_COUNT_MISMATCH` | `steps_keys.go:165` — blueprint 가 선언한 수와 노드 수가 다르다 |
| +0x82 | `…_FAIL_KEY_NOT_LOCAL` | `steps_keys.go:278·283·286` — 인라인 키·서버 참조·읽을 수 없는 파일 |
| +0x83 | `…_FAIL_KEY_UNREADABLE` | `steps_keys.go:355·361` |

세부 셋의 근거가 한 줄인 이유는 그 줄이 세 값을 거절하는 `default` 이기 때문이다. 갈래 자체는
`steps_keys.go:93~121` 이 가른다.

### 4.4 `BASE_CHAIN_BUILD_GENESIS = 0x1400`

키 집합에서 genesis 를 만들어 쓴다. 지금 `NetGenesis` 다. **실패가 가장 많은 블록이다.**

| 값 | 상태 | 근거 |
|---|---|---|
| +0x00 | `STATE_CHAIN_BUILD_GENESIS` | 진입 |
| +0x01 | `…_FROM_TEMPLATE` | 기본 갈래 |
| +0x02 | `…_FROM_EXISTING` | `steps_compose.go:238` — `opts.Existing != ""` |
| +0x03 | `…_FORK_APPLIED` | `steps_genesis.go:202` — `opts.Fork != nil` |
| +0x04 | `…_VARIANTS_WRITTEN` | `steps_genesis.go:241` — 바이너리별 genesis |
| +0x80 | `…_FAIL_EXISTING_INVALID` | `steps_compose.go:244` — 유효한 JSON 이 아니다 |
| +0x81 | `…_FAIL_EXISTING_FOREIGN` | `verify_genesis.go:39` — 구성한 키와 안 맞는다 |
| +0x82 | `…_FAIL_FORK_UNRESOLVED` | `steps_genesis.go:287·360·390·410` — 포크의 이름·체인·구간·상대가 없다 |
| +0x83 | `…_FAIL_DECL_UNUSED` | `steps_genesis.go:430·472` — 아무 노드도 안 쓰는 바이너리에 선언했다 |
| +0x84 | `…_FAIL_TARGET_UNABLE` | `steps_compose.go:285·293` — 생성할 생산자가 없거나 대상이 명령을 못 돌린다 |

**왜 세부가 넷인가.** 이 단계만 **두 체인을 다룬다.** 포크를 건너는 망은 넘겨주는 체인의
genesis 를 읽어 넘겨받는 체인 것을 만든다. `+0x03` 이 그 자리다.

### 4.5 `BASE_CHAIN_BUILD_NODE_CONFIG = 0x1500`

노드마다 TOML 설정을 만들어 쓴다. 지금 `NetConfig` 다.

| 값 | 상태 | 근거 |
|---|---|---|
| +0x00 | `STATE_CHAIN_BUILD_NODE_CONFIG` | 진입 |
| +0x80 | `…_FAIL_BAD_OVERRIDE` | `provenance.go:22·149·155` — `key=value` 가 아니거나 범위가 틀렸다 |
| +0x81 | `…_FAIL_READBACK` | `steps_config.go:145` — 쓴 것과 대상에 있는 것이 다르다 |
| +0x82 | `…_FAIL_PIN_UNREADABLE` | `steps_config.go:99·122` — 고정한 config·genesis 를 못 읽는다 |

### 4.6 `BASE_CHAIN_BUILD_NODE_COMMAND = 0x1600`

노드마다 실행 명령을 조립한다. 지금 `NetLaunchOpts` 다.

| 값 | 상태 | 근거 |
|---|---|---|
| +0x00 | `STATE_CHAIN_BUILD_NODE_COMMAND` | 진입 |
| +0x80 | `…_FAIL_BAD_OPTION` | `provenance.go:85`, `steps_compose.go:175` — 범위·`--set` 형식 |

### 4.7 `BASE_CHAIN_DEPLOY_NODES = 0x1700`

실행 입력을 대상에 놓고 확인한다. 지금 `NetProvision` 이다.

| 값 | 상태 | 근거 |
|---|---|---|
| +0x00 | `STATE_CHAIN_DEPLOY_NODES` | 진입 |
| +0x01 | `…_VERIFIED_LOCAL` | `steps_compose.go:105` — 로컬이면 확인만 한다 |
| +0x02 | `…_SHIPPED_REMOTE` | `steps_compose.go:104` — 원격이면 신원 파일을 올린다 |
| +0x80 | `…_FAIL_INPUT_MISSING` | `steps_compose.go:58` |
| +0x81 | `…_FAIL_INPUT_FOREIGN` | `steps_compose.go:69` — 우리가 만든 파일이 아니다 (체크섬 불일치) |

### 4.8 `BASE_CHAIN_INIT_NODES = 0x1800`

노드마다 datadir 을 초기화한다. 지금 `NetInit` 이다.

| 값 | 상태 | 근거 |
|---|---|---|
| +0x00 | `STATE_CHAIN_INIT_NODES` | 진입 |
| +0x80 | `…_FAIL_TARGET_UNABLE` | `steps_lifecycle.go:137` — 대상 드라이버가 못 한다 |
| +0x81 | `…_FAIL_GENESIS_UNREADABLE` | `steps_lifecycle.go:150` |
| +0x82 | `…_FAIL_DATADIR` | `steps_lifecycle.go:180` — 비우기 실패 |

### 4.9 `BASE_CHAIN_LAUNCH_NODES = 0x1900`

멈춘 노드를 띄우고 PID를 기록한다. 지금 `NetStart` 다. **세부가 가장 많은 블록이다.**

| 값 | 상태 | 근거 |
|---|---|---|
| +0x00 | `STATE_CHAIN_LAUNCH_NODES` | 진입 |
| +0x01 | `…_PHASE_LAUNCHING` | 한 단계의 노드를 띄운다 |
| +0x02 | `…_PHASE_ACTIONS` | 그 단계의 뒷작업을 돌린다 |
| +0x03 | `…_PHASE_DONE` | 다음 단계로, 또는 전부 끝 |
| +0x80 | `…_FAIL_NO_BINARY` | `phases.go:268·276·285` — 없거나 대상에 없거나 PATH 에 없다 |
| +0x81 | `…_FAIL_PORT_BUSY` | `occupancy.go:137` |
| +0x82 | `…_FAIL_OCCUPIED` | `occupancy.go:73` — 워크스페이스 밖에서 이미 돌고 있다 |
| +0x83 | `…_FAIL_NO_KEYSTORE` | `phases.go:179` |
| +0x84 | `…_FAIL_PHASE_EMPTY` | `phases.go:88` — 뒷작업을 돌릴 노드를 안 띄웠다 |

**왜 세부가 셋인가.** 이 단계는 **집안마다 단계 수가 다르다.**

- `wbft` — `{Name: "all"}` 하나 (`wbft.go:64`)
- `poa`/`wemix` — `boot`(생산자 하나를 혼자 띄우고 etcd 를 세운다) 다음 `join-nodeN` 을 생산자
  수만큼 (`poa.go:88~106`)

지금은 그 반복이 `phases.go` 안의 루프다. 상태로 올리면 **어느 단계에서 멈췄는지가 상태값**이
된다.

### 4.10 `BASE_CHAIN_VERIFY = 0x1A00`

뜬 망이 계획과 같은지 본다. 지금 `testengine` 의 `verifyAgainstPlan` 과 준비 게이트가 나눠 한다.

| 값 | 상태 | 근거 |
|---|---|---|
| +0x00 | `STATE_CHAIN_VERIFY` | 진입 |
| +0x01 | `…_PRODUCING` | 블록이 나온다 |
| +0x02 | `…_HALTED_AS_DECLARED` | `genesis.haltsAt` 이 선언한 대로 멈췄다 |
| +0x80 | `…_FAIL_PLAN_MISMATCH` | 계획과 다른 망이 떴다 |
| +0x81 | `…_FAIL_NOT_PRODUCING` | 준비 예산 안에 블록이 안 나온다 |

**이 블록은 지금 `chainsetup` 밖에 있다.** 조립의 일부로 들여올지는 5번에서 정한다.

### 4.11 `BASE_CHAIN_READY = 0x1F00`

| 값 | 상태 |
|---|---|
| +0x00 | `STATE_CHAIN_READY` — **루프 탈출** |

## 5. 인수 영역 (0x2000)

말씀된 "이미 구성된 체인도 준비된 것" 이 여기다. 지금은 CLI 의 `switch` 네 갈래다.

### 5.1 `BASE_CHAIN_ADOPT = 0x2000`

| 값 | 상태 | 지금 어디 |
|---|---|---|
| +0x00 | `STATE_CHAIN_ADOPT` | 진입 |
| +0x01 | `…_BY_RPC` | `run.go` 의 `--rpc` 갈래 |
| +0x02 | `…_BY_WORKSPACE` | `--attach --workspace-dir` 갈래 |
| +0x03 | `…_BY_DECLARATION` | `env.attach` 갈래 |
| +0x80 | `…_FAIL_UNREACHABLE` | 붙을 수 없다 |
| +0x81 | `…_FAIL_WRONG_CHAIN` | chainId 가 다르다 |

### 5.2 `BASE_CHAIN_COMPARE = 0x2100`

있는 것과 원하는 것을 견준다. 지금 `preflight.Compare` 다. **판정 넷이 그대로 상태가 된다.**

| 값 | 상태 | 지금 |
|---|---|---|
| +0x00 | `STATE_CHAIN_COMPARE` | 진입 |
| +0x01 | `…_SAME` | `preflight.Reuse` → `STATE_CHAIN_READY` |
| +0x02 | `…_NODES_DIFFER` | `preflight.RebuildNodes` → 노드별 재구성 |
| +0x03 | `…_NETWORK_DIFFERS` | `preflight.RebuildAll` → `OPEN_WORKSPACE` 로 되돌림 |
| +0x04 | `…_NOTHING_COMPOSED` | `preflight.Compose` → `OPEN_WORKSPACE` |
| +0x80 | `…_FAIL_UNREADABLE` | 기록을 못 읽는다 |

**여기서 중복이 하나 사라진다.** 지금은 같은 질문을 두 곳이 서로 모르는 채 답한다 —
`preflight.Compare`(조립 전)와 `reconcileUp`(조립 중, `verbs_up.go:363` 의 `name == "keys"`).
한 블록이 되면 답하는 자리가 하나다.

## 6. 운영 영역 (0x3000)

조립이 끝난 뒤의 동작이다. 지금 단계 장부에 조립과 섞여 들어간 여덟이 여기로 온다.

| BASE | 블록 | 지금 장부의 이름 |
|---|---|---|
| 0x3000 | `CHAIN_OP_STOP` | `stop` · `stop-node` |
| 0x3100 | `CHAIN_OP_REMOVE` | `rm` |
| 0x3200 | `CHAIN_OP_RESTART_NODE` | `restart` · `start-node` |
| 0x3300 | `CHAIN_OP_SWAP_NODE` | `swap-node` |
| 0x3400 | `CHAIN_OP_CROSS_FORK` | `cross-fork` |
| 0x3500 | `CHAIN_OP_HARDFORK` | `hardfork` |

**세부와 실패는 아직 안 쟀다.** 1번이 조립만 쟀고, 운영 쪽 실패 96건은 같은 방법으로 따로
재야 한다. 그 전까지 이 표는 블록 자리만 잡아 둔 것이다.

## 7. 공통 실패 (0x0F00)

1번이 찾은 공유 실패 16건을 셋으로 갈랐다.

**첫째, 상태 기계가 없애는 것 (2건).** `require`(6단계가 공유)와 `plugin`(7단계)은 전부
"앞 단계가 안 끝났다" 를 묻는다. 전이 표가 그것을 막으므로 상태가 필요 없다.

**둘째, 어느 상태에서도 빠질 수 있는 것 (3건).** 워크스페이스 자체의 실패다.

| 값 | 상태 | 근거 |
|---|---|---|
| 0x0F80 | `STATE_CHAIN_FAIL_RECORD_FORMAT` | `workspace.go:87` — 기록 형식이 이 빌드가 모르는 판이다 |
| 0x0F81 | `STATE_CHAIN_FAIL_RECORD_SAVE` | `verbs_steps.go:51` — 저장 실패 |
| 0x0F82 | `STATE_CHAIN_FAIL_WORKSPACE_CONFIG` | `workspace_refs.go:27` |

**셋째, 여러 단계가 같은 일을 하는 것 (11건).** 바이너리 해석(`binary.go:88·99·105`,
`binary.go:54·58`), 포트 점검(`occupancy.go:137·191`), blueprint 읽기(`verbs_steps.go:209·213`),
입력 참조(`workspace_refs.go:146`), 노드 조회(`steps_compose.go:201`).

**이것들은 공통 블록으로 두지 않는다.** 실패한 자리를 상태가 말해야 하는데, 공통 블록에 넣으면
"바이너리를 못 찾았다" 는 알아도 "명령을 조립하다가인지 띄우다가인지" 를 잃는다. 그래서
**각 블록이 자기 실패 상태로 갖는다.** 위 4장의 `LAUNCH_NODES_FAIL_NO_BINARY` 와
`BUILD_NODE_COMMAND` 의 것이 그렇게 갈라진 결과다.

## 8. 테스트 영역 (0x8000)

`testengine` 의 생애주기다. 체인 영역과 값이 겹치지 않으므로 한 manager 가 둘을 들고 있어도
섞이지 않는다.

| BASE | 블록 |
|---|---|
| 0x8000 | `TEST_PENDING` |
| 0x8100 | `TEST_RUNNING` |
| 0x8200 | `TEST_REPORTING` |
| 0x8F00 | `TEST_DONE` |

**세부는 이 문서가 정하지 않는다.** 6번에서 체인 쪽이 끝난 뒤 같은 방법으로 잰다.

## 9. 합계

| | 수 |
|---|---|
| 블록 | 조립 11 · 인수 2 · 운영 6 · 공통 1 · 테스트 4 = **24** |
| 상태 (조립+인수, 확정분) | 진입 13 · 세부 19 · 실패 31 = **63** |
| 아직 안 잰 것 | 운영 6블록의 세부·실패, 테스트 4블록 |

## 10. 이 표가 지금의 어휘 다섯을 어떻게 흡수하나

| 지금 | 새 구조 |
|---|---|
| `state.Steps` 의 조립 9 | `0x1100`~`0x1900` 아홉 블록 |
| `state.Steps` 의 운영 8 | `0x3000`~`0x3500` 여섯 블록 |
| `UpStage` (`deploy`/`start`) | **목표 상태**. `RunUntil(STATE_CHAIN_DEPLOY_NODES)` |
| `preflight.Verdict` 넷 | `BASE_CHAIN_COMPARE` 의 세부 넷 |
| CLI 의 네 갈래 | **시작 상태**. `OPEN_WORKSPACE` 또는 `ADOPT_BY_*` 셋 |

## 11. 다음 (3번)

전이 표를 쓴다. 이 문서가 정한 63개 상태 사이에 **어떤 이동이 허용되는가**이고, 그 입력은
지금 `composeNeeds` 가 가진 순서와 위 4장의 세부 갈래다.

전이 표를 쓸 때 답해야 하는 것 셋을 미리 적어 둔다.

- **재시도 전이를 넣을 것인가.** `BUILD_NODE_TABLE_FAIL_SET_CONTENDED` 와
  `LAUNCH_NODES_FAIL_PORT_BUSY` 는 기다리면 풀릴 수 있다. 지금 코드는 전부 즉시 중단한다.
- **`COMPARE_NETWORK_DIFFERS` 가 `OPEN_WORKSPACE` 로 되돌아가는 것을 허용할 것인가.** 허용하면
  루프가 한 바퀴 더 돈다. 무한 루프를 막는 것은 전이 횟수 상한이다.
- **`VERIFY` 를 조립 안에 둘 것인가.** 지금은 `testengine` 에 있다.
