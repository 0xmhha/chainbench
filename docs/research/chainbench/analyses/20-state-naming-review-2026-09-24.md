# 상태 이름 재검토 — 02 규약으로 되돌리기

> **기록 문서다.** 여기 그림과 표의 상태 이름은 `cd9664b3` 시점의 옛 이름(진행형)이다. 이름은
> 2026-09-24 [state-machine-06](../../../dev/architecture/design-v3/state-machine-06-naming-and-contract.md)
> 대로 바뀌었고(§3 의 "지금" 열이 옛→새 대응표), 이 문서가 그린 결함은 같은 PR 에서 고쳐졌다.
> 옛 코드를 설명하는 기록이라 이름을 새로 바꿔 쓰지 않는다.

> 기준: `386a78ee` (main `cde3a08f` 위). 2026-09-24.
> 짝 문서: [`19-state-machine-gaps-2026-09-24.md`](19-state-machine-gaps-2026-09-24.md) — 같은 두 머신의 전이와 빈 경로.
> 이름의 기준: [`state-machine-02-states.md`](../../../dev/architecture/design-v3/state-machine-02-states.md) §2 (조립·운영),
> [`state-machine-05-test-failures.md`](../../../dev/architecture/design-v3/state-machine-05-test-failures.md) §2 (테스트).

## 1. 왜 다시 보나

02 문서는 상태 이름을 `STATE_<영역>_<단계>[_<세부>][_FAIL_<사유>]` 로 짓고, 단계 이름은 **그 단계가
하는 일**로 짓는다고 정했다. 영역은 `CHAIN` 과 `TEST`, 운영 동작은 `CHAIN_OP_` 이다.

리팩토링 전 코드는 이 규약을 따르고 있었다. `a5386b0e` 의 `internal/core/lifecycle/transitions.go` 는
`ChainOpenWorkspace`·`ChainEnsureKeysFromPreset`·`CompareChainNetworkDiffers`·`ChainOpCrossForkHandingOver`·
`TestReachNetwork` 처럼 117개 이름을 들고 있었다(실패 62개 포함).

PR #425(`cde3a08f`)가 이 이름들을 지우고 진행형 어휘로 바꿨다. 근거는 작업 지시서였던
`15-hsm-refactoring-handoff-2026-09-21.md` §1 "확정된 설계 — 바꾸지 않는다" 의 한 줄이다:
"큰 단계 = parent state(진행형, `ensuringKeys`, `buildingGenesis`). 구체 행동 = leaf state(행동 이름)".
이 줄은 14번 검토 문서의 어휘 표를 옮긴 것이고, 두 문서 어디에도 02 의 이름과 어떻게 대응하는지,
왜 02 를 바꾸는지는 적혀 있지 않다(`git show cde3a08f:docs/research/chainbench/analyses/15-hsm-refactoring-handoff-2026-09-21.md`).
그래서 [결정] 문서와 코드가 서로 다른 말을 하게 됐다.

이 문서는 두 머신의 상태를 하나씩 **코드가 실제로 하는 일**에 비추어 보고, 02 규약의 이름을 제안한다.
이름만 바꾸면 되는 것과, 이름이 가리키는 구조부터 02 와 다른 것을 나눠 적는다.

---

## 2. 표기 규칙 (제안)

02 는 이름의 **낱말**을 정했고 Go 표기는 정하지 않았다. 리팩토링 전 코드가 쓰던 표기를 되살린다.

| 자리 | 02 | Go (제안) |
|---|---|---|
| 상태 이름 값 | `STATE_CHAIN_OPEN_WORKSPACE` | `"ChainOpenWorkspace"` — `a5386b0e` 의 lifecycle 과 같은 모양 |
| const 이름 | — | `nameChainOpenWorkspace statemachine.StateName` |
| 세부 | `…_ENSURE_KEYS` + `_FROM_PRESET` | `"ChainEnsureKeysFromPreset"` — 부모 이름을 앞에 그대로 붙인다 |
| 운영 | `CHAIN_OP_STOP` | `"ChainOpStop"` |
| 테스트 | `TEST_READ_DECLARATION` | `"TestReadDeclaration"` |

세부가 부모 이름을 달고 다니므로 이름 하나만 로그에 찍혀도 어느 단계의 세부인지 알 수 있다.
그 대신 경로가 길어진다: `Chain/ChainCompose/ChainBuildGenesis/ChainBuildGenesisFromTemplate`.
→ **결정 D1** (7장).

---

## 3. composition 머신 — 조립 (`CHAIN`)

"하는 일" 열은 각 상태의 `Enter`·`Process` 와 타입 주석에서 옮겼다.

| # | 지금 | 실제로 하는 일 | 02 / 리팩토링 전 | 제안 | 판정 |
|---|---|---|---|---|---|
| 1 | `Composition` | root. 어느 상태의 `stageFailed` 든 받아 `Failed` 로 보낸다 (`manager_states.go:41-50`) | 없음 | `Chain` | 이름만 |
| 2 | `Stopped` | 머신이 명령을 기다리는 시작 상태. `ClearError` 뒤에도 여기로 온다 (`manager_states.go:53`). **체인이 멈췄다는 뜻이 아니다** | 없음. 리팩토링 전의 `ChainStopped` 는 "체인을 멈췄다" 였다 | `ChainIdle` | 이름만. `Stopped` 를 그대로 두면 운영의 멈춤과 계속 섞인다 |
| 3 | `Composing` | 조립 스테이지들의 부모. 스테이지 보고를 받아 다음 스테이지로 보낸다 (`manager_states.go:131-161`) | 영역 `CHAIN 조립(0x1000)` — 상태 이름은 없음 | `ChainCompose` | 이름만 |
| 4 | `OpeningWorkspace` | 워크스페이스를 열고 요청을 기록한다 | `STATE_CHAIN_OPEN_WORKSPACE` / `ChainOpenWorkspace` | `ChainOpenWorkspace` | 이름만 |
| 5 | `BuildingNodeTable` | 역할·경로·포트를 정해 노드 표를 만든다 | `STATE_CHAIN_BUILD_NODE_TABLE` / `ChainBuildNodeTable` | `ChainBuildNodeTable` | 이름만 |
| 6 | `EnsuringKeys` | 키 출처를 정하고(원격 키링이면 가져온 뒤) 세부로 보낸다 (`stage_keys.go:65-106`) | `STATE_CHAIN_ENSURE_KEYS` / `ChainEnsureKeys` | `ChainEnsureKeys` | 이름만 |
| 7 | `KeysFromPreset` | 있는 키 세트를 그대로 쓴다 | `ChainEnsureKeysFromPreset` | `ChainEnsureKeysFromPreset` | 이름만 |
| 8 | `KeysGenerated` | 새 키 세트를 만든다. **만드는 중인 상태**인데 이름은 결과형이다 | `ChainEnsureKeysGenerated` | `ChainEnsureKeysGenerate` | 이름만. 02 는 결과형을 금한다 — 리팩토링 전 이름도 이 점에서 어긋나 있었다 |
| 9 | `KeysDeclared` | 선언(blueprint)이 노드마다 적은 키를 쓴다 | `ChainEnsureKeysFromBlueprint` | `ChainEnsureKeysFromBlueprint` | 이름만 |
| 10 | `Reconciling` | reuse-if-matching 일 때만, 키 다음에 돌고 있는 망과 원하는 것을 견준다 (`stage_reconcile.go:37-61`) | 02 §5.2: **COMPARE 와 한 블록으로 합친다**("답하는 자리가 하나다"). 리팩토링 전: `ReconcileChain` + `_AllKept`·`_SomeRedone` | `ChainReconcile` (임시) | **구조 차이** — S3 |
| 11 | `BuildingGenesis` | 요청을 보고 템플릿/기존 genesis 를 고른다 | `STATE_CHAIN_BUILD_GENESIS` / `ChainBuildGenesis` | `ChainBuildGenesis` | 이름만 |
| 12 | `GenesisFromTemplate` | 체인 템플릿으로 genesis 를 만든다 | `ChainBuildGenesisFromTemplate` | `ChainBuildGenesisFromTemplate` | 이름만 |
| 13 | `GenesisFromExisting` | 주어진 genesis 를 쓴다 | `ChainBuildGenesisFromExisting` | `ChainBuildGenesisFromExisting` | 이름만. 리팩토링 전의 `_ForkApplied`·`_VariantsWritten` 은 #425 에서 없어졌다 |
| 14 | `BuildingNodeConfig` | 노드마다 config 를 렌더한다 | `STATE_CHAIN_BUILD_NODE_CONFIG` | `ChainBuildNodeConfig` | 이름만 |
| 15 | `BuildingNodeCommand` | 노드마다 실행 인자(argv)를 조립한다 | `STATE_CHAIN_BUILD_NODE_COMMAND` | `ChainBuildNodeCommand` | 이름만 |
| 16 | `DeployingInputs` | 실행 입력(genesis·config·키)이 대상에 있는지 확인하고, 원격이면 보낸다 (`stage_deploy.go:50-79`) | `STATE_CHAIN_DEPLOY_NODES` / `ChainDeployNodes` | `ChainDeployNodes` | 이름만. 단 하는 일은 "입력 배포" 다 → **D3** |
| 17 | `InputsVerifiedLocal` | 일은 없다. 로컬이라 보낼 것이 없었다는 기록 (`stage_deploy.go:80-95`) | `ChainDeployNodesVerifiedLocal` | `ChainDeployNodesVerifiedLocal` | 이름만. 결과형이지만 하는 일이 없는 기록 상태라 결과가 곧 내용이다 |
| 18 | `InputsShippedRemote` | 위와 같고, 원격으로 보냈다는 기록 | `ChainDeployNodesShippedRemote` | `ChainDeployNodesShippedRemote` | 이름만 |
| 19 | `InitializingDatadirs` | 노드마다 datadir 를 genesis 로 init 한다 | `STATE_CHAIN_INIT_NODES` / `ChainInitNodes` | `ChainInitNodes` | 이름만 |
| 20 | `Launching` | 체인 집안의 phase 목록을 받아 phase 마다 띄운다 (`stage_launch.go:64-117`) | `STATE_CHAIN_LAUNCH_NODES` / `ChainLaunchNodes` | `ChainLaunchNodes` | 이름만 |
| 21 | `LaunchingPhase` | phase 하나의 노드를 띄운다 | `…_PHASE_LAUNCHING` / `ChainLaunchNodesPhaseLaunching` | `ChainLaunchNodesPhase` | 이름만 |
| 22 | `RunningPhaseActions` | 그 phase 가 선언한 뒷작업을 돌린다 | `…_PHASE_ACTIONS` / `ChainLaunchNodesPhaseActions` | `ChainLaunchNodesPhaseActions` | 이름만 |
| 23 | `RecordingRun` | 모든 phase 뒤에 실행 기록(`runs/<시각>`)을 쓴다 | 02 의 `…_PHASE_DONE` 은 "다음 phase 로, 또는 끝" 이라는 **판단**이었다. 코드는 그 판단을 부모의 `Process` 가 하고(`launching.next`), 이 상태는 기록을 쓴다 | `ChainLaunchNodesRecordRun` | 이름만. 02 의 `PHASE_DONE` 은 코드에 대응하는 상태가 없다 |
| 24 | `Composed` | `--stage` 가 요청한 단계에서 멈춘 조립 (`manager_states.go:165`). 끝까지 간 조립은 여기로 오지 않는다 | 없음 | `ChainComposeStoppedAtStep` | 이름만 → **D4** |
| 25 | `Ready` | 모든 스테이지를 마친 조립. **그리고** 운영 동작의 부모이고, 운영이 끝나면 다시 여기로 온다 (`manager_states.go:197-203`) | `STATE_CHAIN_READY` — "루프 탈출". 운영은 따로 `0x3000` 영역 | `ChainReady` | **구조 차이** — S2 |
| 26 | `Failed` | 실패 이유를 필드로 들고 멈춘다. `ClearError` 만 받는다 | 02: 블록마다 `+0x80` 실패 상태, 공통 실패 `0x0F00`. 리팩토링 전: 실패 이름 62개 | `ChainFailed` | **구조 차이** — S5 |

---

## 4. composition 머신 — 비교와 인수 (`CHAIN` 0x2000)

| # | 지금 | 실제로 하는 일 | 02 / 리팩토링 전 | 제안 | 판정 |
|---|---|---|---|---|---|
| 27 | `Comparing` | 워크스페이스에 있는 것과 원하는 것을 견줘 판정 넷 중 하나를 내고 그리로 보낸다 (`stage_compare.go:57-103`) | `STATE_CHAIN_COMPARE` / `CompareChain` | `ChainCompare` | 이름만. 리팩토링 전 이름은 영역이 뒤에 있었다(`CompareChain`) — 02 순서로 맞춘다 |
| 28 | `RestartingNodes` | 판정이 "일부 노드가 다르다" 일 때 그 노드들만 다시 띄운다 | `…_NODES_DIFFER` / `CompareChainNodesDiffer` | `ChainCompareNodesDiffer` | 이름만. 02 는 세부를 **판정**으로 이름 지었다. 이 상태의 일(다시 띄우기)은 그 판정의 결과다 |
| 29 | `StoppingToRebuild` | 판정이 "망 전체가 다르다" 일 때 망을 내린다. 그 뒤 조립으로 가야 하는데 못 간다(19번 문서 §3) | `…_NETWORK_DIFFERS` / `CompareChainNetworkDiffers` | `ChainCompareNetworkDiffers` | 이름만 |
| — | (상태 없음) | 판정 "같다" 는 `Verifying` 으로, "아무것도 없다" 는 첫 스테이지로 바로 간다 | `…_SAME` → `READY`, `…_NOTHING_COMPOSED` → `OPEN_WORKSPACE` | 02 대로면 `ChainCompareSame` 뒤 `ChainReady` | **구조 차이** — S1 |
| 30 | `Verifying` | **일이 없다.** 경로만 기록하고 멈춘다. 들어오는 길은 판정 "같다" 하나뿐이다 (`stage_compare.go:170-182`) | `STATE_CHAIN_VERIFY` — 블록이 나오는지, 선언대로 멈췄는지 본다. 실제 검증은 run 머신의 `verifyAgainstPlan` 과 readiness gate 가 한다 | 없앤다. 판정 "같다" 는 `ChainReady` 로 | **구조 차이** — S1 |
| — | run 머신의 `AttachingToNetwork` | 이미 선 워크스페이스의 망을 찾는다 | `STATE_CHAIN_ADOPT` + `_BY_RPC`·`_BY_WORKSPACE`·`_BY_DECLARATION` / `AdoptChain…` | 6장 참조 | **구조 차이** — S4 |

---

## 5. composition 머신 — 운영 (`CHAIN_OP` 0x3000)

| # | 지금 | 실제로 하는 일 | 02 / 리팩토링 전 | 제안 | 판정 |
|---|---|---|---|---|---|
| 31 | `Stopping` | 떠 있는 노드를 모두 내린다 | `CHAIN_OP_STOP` / `ChainOpStopNodes` | `ChainOpStop` | 이름만 |
| 32 | `Removing` | 조립한 망의 데이터를 지운다 | `CHAIN_OP_REMOVE` / `ChainOpRemoveNodes` | `ChainOpRemove` | 이름만 |
| 33 | `Restarting` | 노드 하나를 내렸다 올린다 | `CHAIN_OP_RESTART_NODE` / `ChainOpStartNodes` | `ChainOpRestartNode` | 이름만. #28 `ChainCompareNodesDiffer` 와 이제 헷갈리지 않는다 |
| 34 | `Swapping` | 노드 하나의 바이너리를 바꿔 다시 띄운다 | `CHAIN_OP_SWAP_NODE` / `ChainOpReplaceNode` | `ChainOpSwapNode` | 이름만 |
| 35 | `Hardforking` | 포크 블록에서 망 전체의 바이너리를 바꾼다. 데이터는 둔다 (`stage_operate.go:108`) | `CHAIN_OP_HARDFORK` / `ChainOpHardfork` | `ChainOpHardfork` | 이름만 |
| 36 | `CrossingFork` | 망이 포크를 넘을 때까지 지켜보며 생산이 넘어가게 한다 | `CHAIN_OP_CROSS_FORK` / `ChainOpCrossFork` | `ChainOpCrossFork` | 이름만 |
| 37 | `BeforeFork` | 포크 직전 블록까지 체인이 오기를 기다린다 | `ChainOpCrossForkBeforeFork` | `ChainOpCrossForkAwaitBoundary` | 이름만. 위치형 이름을 하는 일로 바꾼다 — **D5** |
| 38 | `HandingOver` | 포크 전 노드를 내리고 포크 후 빌드를 올린다 | `ChainOpCrossForkHandingOver` | `ChainOpCrossForkHandOver` | 이름만 |
| 39 | `Crossed` | 넘은 뒤 확인하고 `operationDone` 을 보낸다. 이미 넘은 망은 바로 여기로 온다 | `ChainOpCrossForkCrossed` | `ChainOpCrossForkConfirm` | 이름만 — **D5** |

---

## 6. run 머신 — 테스트 (`TEST`)

05 문서가 02 의 네 블록(`PENDING`·`RUNNING`·`REPORTING`·`DONE`)을 실측으로 여섯 블록으로 고쳤다. 그것이 기준이다.

| # | 지금 | 실제로 하는 일 | 05 / 리팩토링 전 | 제안 | 판정 |
|---|---|---|---|---|---|
| 40 | `Run` | root. 스테이지가 `stageStopped` 를 보내면 `Collecting` 으로 보낸다 (`runmachine.go:188-193`) | 없음 | `Test` | 이름만 |
| 41 | `Pending` | 시작 상태. `startRun` 을 받아 선언 읽기로 간다 | 05 에 없음(02 의 `TEST_PENDING` 을 05 가 뺐다) | `TestIdle` | 이름만. composition 의 `ChainIdle` 과 짝 |
| 42 | `ReadingDeclaration` | 스펙과 chain-preset 을 읽어 요청과 계획을 만든다 | `TEST_READ_DECLARATION` / `TestReadDeclaration` | `TestReadDeclaration` | 이름만 |
| 43 | `OpeningSession` | 계획을 적고 아티팩트 자리를 연다 | `TEST_OPEN_SESSION` / `TestOpenSession` | `TestOpenSession` | 이름만 |
| 44 | `ReachingNetwork` | 망을 세우거나 이미 선 망을 찾는다. 두 갈래 모두 끝에 readiness gate 를 지난다 | `TEST_STAND_UP_NETWORK` / `TestReachNetwork` | `TestStandUpNetwork` | 이름만. 05 가 리팩토링 전 이름(`Reach`)을 `STAND_UP` 으로 고쳤다 |
| 45 | `ComposingNetwork` | **composition 머신으로 들어가** 조립을 끝내고(`ChainUpComparing`), 계획과 대조하고, gate 를 지난 뒤 `networkReached` 를 보낸다 (`runmachine.go:392-410`) | 05: "체인 영역에 위임" | `TestStandUpNetworkCompose` | 이름만. 사용자 설명 2번의 "체인 셋업 단계" 가 이 상태다 |
| 46 | `AttachingToNetwork` | 기존 워크스페이스의 망을 읽고 gate 를 지난다 | 05: 붙는 실행은 **`TEST_OPEN_SESSION` 에서 시작해 0x8200 을 건너뛴다**. 02: `CHAIN_ADOPT_BY_WORKSPACE` | `TestStandUpNetworkAttach` (임시) | **구조 차이** — S4 |
| 47 | `Preparing` | 포크·높이·계정 — 살아 있는 망 위의 준비 | `TEST_PREPARE` / `TestPrepare` | `TestPrepare` | 이름만 |
| 48 | `RunningCases` | 케이스를 돈다 | `TEST_RUN_CASES` / `TestRunCases` | `TestRunCases` | 이름만 |
| 49 | `Collecting` | 증거를 모은다. 실패한 실행도 반드시 지난다 | `TEST_COLLECT` / `TestCollect` | `TestCollect` | 이름만 |
| 50 | `Done` | 끝. 케이스 판정과 무관하게 실행이 끝났다 | `TEST_DONE` / `TestFinished` | `TestDone` | 이름만 |
| 51 | `Failed` | 실행이 어느 스테이지에서 더 가지 못했다. 그 자리는 경로가 말한다 | 없음 | `TestFailed` | 이름만. composition 의 `Failed` 와 이제 구분된다 |

---

## 7. 이름만으로는 안 되는 것 — 구조 차이

이름을 02 로 바꿔도 아래 다섯은 **02 가 그린 모양과 코드의 모양이 다르다.** 이름을 바꾸기 전에
어느 쪽에 맞출지 정해야 한다.

**S1. 판정 "같다" 가 `Verifying` 으로 간다.** 02 는 `COMPARE_SAME → READY` 다. 코드는 일이 없는
`Verifying` 으로 가서 멈추고, 이름이 `VERIFY` 여서 검증을 하는 것처럼 읽힌다. 실제 검증(블록이
나오는가)은 run 머신에 있고 02 도 "이 블록은 지금 `chainsetup` 밖에 있다" 고 적었다.
제안: `Verifying` 을 없애고 `ChainCompareSame → ChainReady` 로 둔다. 검증을 chainsetup 으로 들이는 날
`ChainVerify` 를 그 일과 함께 만든다.

**S2. `Ready` 가 운영의 부모이고, 운영 뒤에도 `Ready` 다.** 02 는 `READY` 를 조립의 끝으로, 운영을
`0x3000` 의 별도 영역으로 두었다. 코드는 운영을 `Ready` 아래에 두고 끝나면 `Ready` 로 돌아가므로,
`ChainOpStop` 으로 망을 내린 뒤에도 상태가 "준비됨" 이다. 리팩토링 전에는 `ChainStopped`·`ChainRemoved`
라는 결과 상태가 있었다.
제안: 운영의 부모를 `ChainOp` 로 두고(`ChainReady` 와 형제), 운영이 끝난 뒤의 상태를 운영마다 정한다 —
멈춤·지움은 각각 `ChainStopped`·`ChainRemoved`, 나머지는 `ChainReady`.

**S3. 비교가 두 곳에 있다.** 02 §5.2 는 `preflight.Compare`(조립 전)와 `reconcileUp`(조립 중)이 같은
질문에 따로 답하는 것을 중복으로 보고 한 블록으로 합친다고 정했다. 코드에는 여전히 `Comparing`(조립
전)과 `Reconciling`(키 다음)이 따로 있다. 이름으로는 풀리지 않는다.

**S4. 붙기(attach)의 자리.** 05 는 붙는 실행이 `TEST_OPEN_SESSION` 에서 시작해 망 세우기 블록을
건너뛴다고 했고, 02 는 붙기를 체인 영역의 `CHAIN_ADOPT` 로 두었다. 코드는 run 머신의
`ReachingNetwork` 아래 leaf 로 두었다(선언 읽기부터 늘 같은 길). `ADOPT` 의 세 갈래(`BY_RPC`·
`BY_WORKSPACE`·`BY_DECLARATION`) 중 코드 상태로 있는 것은 워크스페이스 하나다.

**S5. 실패.** 02 는 실패를 블록마다 `+0x80` 상태로 두었고 리팩토링 전 코드에 62개가 있었다. 지금은
`Failed` 하나와 이유 필드이고, "어디서" 는 경로가 말한다(`FailedAt`·`ComposeFailedAt`). 15번 지시서가
"실패별 leaf state 는 라이브 테스트에서 필요가 확인될 때만 추가한다" 로 정한 결과다.

---

## 8. 결정할 것

| 번호 | 무엇 | 선택지 |
|---|---|---|
| D1 | 세부 이름에 부모 이름을 붙이나 | (가) 붙인다 — `ChainBuildGenesisFromTemplate`. 이름 하나로 어디인지 안다, 경로가 길다 · (나) 세부만 — `FromTemplate`. 경로가 짧다, 이름만 찍히면 모른다 |
| D2 | S1~S5 를 이름 변경과 **같이** 고치나, 이름 먼저 바꾸고 따로 고치나 | 같이 하면 한 번에 02 모양이 된다, 변경이 커진다 · 따로 하면 이름 변경이 기계적이다, 한동안 이름과 구조가 어긋난다 |
| D3 | #16 은 02 의 `DEPLOY_NODES` 인가, 하는 일대로 `DEPLOY_INPUTS` 인가 | 02 를 그대로 둔다 · 02 를 고친다 |
| D4 | #24 `Composed` 의 이름 | `ChainComposeStoppedAtStep` · 다른 이름 |
| D5 | 포크 넘기의 세 순간을 위치·결과형(`BeforeFork`·`Crossed`)으로 둘까, 하는 일(`AwaitBoundary`·`Confirm`)로 바꿀까 | 리팩토링 전 이름 유지 · 02 규칙대로 바꿈 |

## 9. 바꾸면 닿는 곳

- 상태 이름 const: `internal/chainsetup`·`internal/testengine` 의 20개 파일.
- 경로 문자열 리터럴 19곳: `statemachine/machine.go`(주석), `chainsetup/state.go`(주석),
  `manager_test.go`·`state_format_test.go`(트리 golden), `suitecmd/sequence_exit_internal_test.go`.
- **기록 형식**: `chain-record.json` 의 `statePath`(`state.go:117`). 이름이 바뀌면 옛 기록의 경로를
  새 빌드가 모른다. `StateFormatVersion`(지금 2)을 올릴지 정해야 한다.
- 리포트: `RunSuiteOut.FailedAt`·`ComposeFailedAt` 가 경로를 그대로 싣는다.
- 문서: 19번 문서의 그림.
