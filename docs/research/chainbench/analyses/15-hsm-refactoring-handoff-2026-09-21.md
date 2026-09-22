# chainbench lifecycle 를 hierarchical state machine 으로 바꾸는 작업 — 작업 세션용 prompt

> 2026-09-21 작성, 2026-09-22 갱신. 이 문서는 다음 작업 세션에 그대로 전달하는 prompt 다. 설계의 정본은
> `14-hsm-pattern-review-2026-09-21.md` 이고, 현재 코드 측정은 `13-state-driven-refactoring-review-2026-09-21.md`
> (2026-09-22 재측정)이며, 이 문서는 그것을 실행 순서와 규칙으로 옮긴 것이다.

---

## 시작 전 확인 — 다른 세션의 테스트가 끝났는지

이 작업은 다른 세션이 리팩토링된 코드로 라이브 테스트(`scripts/tcsweep.sh`)를 끝낸 **뒤에** 시작한다.
아래 셋이 모두 맞을 때만 1장으로 간다.

1. **그 세션이 끝났다.** `pgrep -fl 'gstable|tcsweep'` 에 아무것도 없고, 31000 · 8600 번대 포트에
   listener 가 없다(`lsof -nP -iTCP -sTCP:LISTEN | grep -E ':(31[0-9]{3}|86[0-9]{2})'` 가 비어 있다).
   무엇이 남아 있으면 **끄지 말고** 그 세션의 것이라고 보고하고 멈춘다.
2. **최신 코드다.** `git fetch && git status && git log --oneline -20`. 그 세션이 테스트 중 고친 commit 이
   있으면 그것이 HEAD 다. 이 문서의 줄 번호는 `ea267fd8` 기준이니 옮겨졌을 수 있다. 함수 이름으로 찾는다.
3. **baseline 이 녹색이다.** `go build ./... && go vet ./... && go test ./...` 가 통과한다. 실패가 있으면
   그것을 먼저 보고한다. 이 작업으로 고치지 않는다.

그 뒤 `docs/research/chainbench/analyses/README.md` 색인과 메모리 `state-driven-refactoring-review-20260921.md`
를 읽고 1장으로 간다.

---

## 0. 먼저 읽을 것 (이 순서대로, 전부)

1. `docs/research/chainbench/analyses/14-hsm-pattern-review-2026-09-21.md` — 설계 정본. 1장(기제 열셋),
   3장(Go 계약), 4장(트리 · protocol · 단계 예시 · 시나리오), 5장(이름과 값의 규칙), 6장(19 commit
   순서), 7장(확정 사항 열 개). 7장에 미결은 없다.
2. `docs/research/chainbench/analyses/13-state-driven-refactoring-review-2026-09-21.md` — 지금 코드가 왜
   state machine 이 아닌지(2026-09-22, HEAD `ea267fd8` 기준 재측정). 0장 · 3장 · 5장을 읽는다. 5장 끝의
   "15번에 더할 것 셋" 은 이 문서 4장과 5장에 이미 들어 있다.
3. 참조 코드 (확인 용도. 14번이 이미 정리했다):
   `/Users/wm-it-25_0220/Work/github/references/android/READING-ORDER.md` 의 1단계 넷(`IState`,
   `State`, `StateMachine.java` 상단 주석 40~200줄, `StateMachineTest.java`), 13번
   `SyncStateMachine.java`(348줄, 전부), 5번 `WakeLockStateMachine.java`(271줄, 전부), 9번
   `DhcpClient.java` 538~561(addState 블록)과 1197~1273(`PacketRetransmittingState`).
4. 현재 코드(2026-09-22, HEAD `ea267fd8` 기준. 더 움직였으면 `git log --oneline -20` 과 `git status` 부터).
   - `internal/core/lifecycle/` — 옛 table 기반 state machine. `status.go` 에 조립(0x1000) · 인수(0x2000) ·
     운영(0x3000) · 테스트(0x8000) 네 영역 117개 이름, `transitions.go` 의 `allowed`, `machine.go`, `mark.go`.
   - `internal/chainsetup/verb/` — verb 층. `statedriven.go`(`composition` 표, `handlersFor`),
     `verbs_up.go`(`upSteps` :103, `netUpFrom` :177, `lifecycle.New` :263), `compare.go`(`NetUpComparing`),
     `verbs_resume.go`, `verbs_steps.go`, `verbs_lifecycle.go`, `verbs_network.go`.
   - `internal/chainsetup/` — `Workspace` 와 단계 본체(`steps_*.go`, `phases.go`), `workspace_open.go`
     (`WithWorkspace` :17, `InWorkspace` :28), `state.go`(record, `FormatVersion` 1), `state_request.go`
     (`UpStepNames` :122, `OpStepNames` :140), `record.go`(`FirstUndone` :185), `verb_needs.go`(`allow`),
     `steps_compose.go`(`composeNeeds` :442, `require` :463).
   - `internal/testengine/` — `suite.go`(`RunSuite` :253 → `runSuiteBody`, `RunSuiteOut.FailedAt` :142),
     `failurekind.go`(`runFailure` :80, sentinel 스물), `engine_impl.go`(`engine.Run` :95),
     `attach_workspace.go`, `nodegate.go`(`gateReady` :277).
   - `internal/app/start.go`(`StartFor` :73), `internal/app/workflow.go`(`AttachRun` :97).
5. 메모리 `state-driven-refactoring-review-20260921.md` 와 `keep-technical-terms-untranslated.md`.

---

## 1. 확정된 설계 — 바꾸지 않는다

- **기제.** state 에 들어가면 `Enter()` 가 그 state 의 일을 하고, 결과를 self message 로 남긴다.
  `Process()` 가 message 를 transition 으로 바꾼다. 다음 일은 다음 state 의 `Enter()` 가 한다.
  `Exit()` 는 `Enter()` 가 건 것을 거둔다. `Enter` / `Exit` 안에서 `TransitionTo` 는 금지이고
  `Machine` 이 거절한다.
- **동기 모델**(`SyncStateMachine` 과 같다). `Send` 는 완전한 단위다: 처리(현재 state 부터 parent 로)
  → transition 하나(exit 는 깊은 쪽부터, enter 는 바깥부터, current 갱신은 마지막) → self queue 소진
  → inbox 소진. 돌아오면 정지 상태. 재진입은 거절. Looper · `sendMessageDelayed` · `deferMessage` 는
  **만들지 않는다.**
- **child → parent 는 `parent.Post(msg)`** (큐에만 넣는다). **parent → child 는 `child.Send`**
  (중첩 호출). 이 비대칭이 재귀를 막는다.
- **어휘.** 큰 단계 = parent state(진행형, `ensuringKeys`, `buildingGenesis`). 구체 행동 = leaf state
  (행동 이름, `keysFromPreset`, `genesisFromExisting`). 어느 leaf state 로 갈지는 **앞 단계 parent 의
  `Process`** 가 request 를 보고 정한다.
- **message.** `Cmd`(시키는 것. 위→아래. child 가 parent 에게 올리는 보고도 `Cmd`, 예 `CmdPostCompose`)
  와 `Event`(일어난 사실). `Action` · `Error` 접두는 없다. 오류는 `Event…Failed` 의 `Err` 필드와
  `failed` state 다.
- **protocol 두 층.** `internal/core/<새 패키지>/protocol.go` 는 BASE 배정만(`BaseChain 0x1000`,
  `BaseTest 0x8000`). 각 state machine 패키지의 `protocol.go` 에 `Cmd` / `Event` 상수와 인자 struct
  (`Message interface{ What() What }`). PUBLIC(밖이 보내는 Cmd)은 `base+1..`, 위로 올리는 Cmd 는
  `base+0x40..`, PRIVATE(self message)은 `base+0x100..`, 아래서 받는 Event 는 `base+0x180..`.
  exported 여부가 PUBLIC 여부다. `whatNames` 표와 "빠진 이름" 테스트.
- **state 이름.** typed const, 도메인 패키지에. record 에는 `Machine.Path` 가 만든 경로 문자열
  (`Composition/Composing/BuildingGenesis/GenesisFromTemplate`). 비교는 값으로(`s == mg.ready`).
- **`failed`.** parent state 하나 + reason 필드. 실패별 leaf state 는 리팩토링 뒤 라이브 테스트에서
  필요가 확인될 때만 추가한다. 지금은 만들지 않는다.
- **record.** state path 필드 추가, `FormatVersion` 2. 하위 호환 없음. 새 build 는 version 1 record 를
  이름을 대며 거절한다.
- **정의서 → `Request`** 는 testengine. **`Request` → leaf state 선택** 은 chainsetup 의 parent state.
- **옛 lifecycle 와 새 것을 같이 둔다.** leaf state 를 하나씩 옮기고, 마지막 commit 에서 옛 것을 지운다.
  어댑터 leaf `legacyStage{step}` 을 쓴다.

---

## 2. 작업 순서 — 14번 6장 표 그대로. 표의 한 줄이 commit 하나다

| commit | 무엇 | 지키는 테스트 |
|---|---|---|
| 1 | 새 state machine 을 **별도 패키지**에 쓴다(아래 3장). 옛 `lifecycle` 는 손대지 않는다 | 새 패키지 단위 테스트 |
| 2 | `protocol.go` 두 층(BASE 배정, chainsetup 의 Cmd/Event 상수 + struct)과 이름·띠 테스트 | 1 과 같이 |
| 3 | record 에 state path 필드를 **추가만** 한다. `FormatVersion` 은 아직 1. 읽는 쪽 없음 | 기존 record 테스트 |
| 4 | chainsetup `Manager` 와 트리. leaf state 는 전부 어댑터 `legacyStage{step}`(`Enter` 가 옛 verb 를 부르고 `SendSelf(eventStageDone)`). `failed` 는 parent + reason. `netUpFrom` 이 `Manager.Send(CmdCompose)` 를 부른다 | `chain up` 통합 전부. 동작이 같아야 한다 |
| 5~13 | leaf state 하나씩 어댑터를 진짜 leaf 로. 한 commit 에 하나. 순서: openingWorkspace, buildingNodeTable, ensuringKeys(3 leaf), buildingGenesis(2 leaf), buildingNodeConfig, buildingNodeCommand, deployingInputs(2 leaf), initializingDatadirs, launching(2 leaf) | 그 단계 + 통합 |
| 14 | `composed` state 와 `CmdStep`. `require` · `composeNeeds` · "run `chain X` first" 문장 삭제 | 단독 명령 |
| 15 | resume 이 record 의 state path 로 `Start`. `FormatVersion` 2. `upStepNames` 잔재 삭제 | resume |
| 16 | `reconciling` · `verifying` · `comparing` 을 트리에. `NetUpComparing` 삭제 | `run` 통합 |
| 17 | `ready` 아래 운영 leaf(stopping · swapping · hardforking · restarting · crossingFork). `verbNeeds` 삭제 | 운영 명령 |
| 18 | testengine `run` state machine. chainsetup 을 child 로, `Post` 로 보고 | 전체 |
| 19 | 어댑터와 옛 `lifecycle` 삭제 | 전체 |

**PR 묶음**(메모리 규칙: PR 은 phase 단위, commit 은 task 단위): PR-A = 1~3, PR-B = 4, PR-C = 5~13,
PR-D = 14~17, PR-E = 18~19.

**이번 세션의 목표는 1~4 (PR-A, PR-B) 다.** 4 가 끝나면 `chain up` 이 새 Manager 로 돌면서 동작이
전과 같아야 한다. 5 부터는 한 commit 에 leaf 하나이고, 시간이 남으면 이어 간다.

---

## 3. commit 1 상세 — 새 state machine 패키지

패키지 이름은 `internal/core/statemachine` 이다. Go 관례대로 전부 소문자 한 낱말이다(`stateMachine` 은
`revive` 가 경고한다). 옛 `lifecycle` 와 공존해야 하므로 다른 이름이어야 하고, 19 뒤에 `lifecycle` 을 지운다.

파일:
- `doc.go` — package doc. 기제 한 문단, 동기 규칙, 금지 사항.
- `state.go` — `State` 인터페이스(`Name() StateName`, `Enter(ctx, *Machine) error`, `Exit(ctx, *Machine)
  error`, `Process(ctx, *Machine, Message) (handled bool, err error)`), `Base`, `StateName`.
- `message.go` — `What`, `Message interface{ What() What }`.
- `machine.go` — `Machine`: `New(name, controller *Machine)`, `Add(s, parent)`, `Start(ctx, initial)`,
  `Send(ctx, msg)`, `Post(msg)`, `TransitionTo(s)`, `SendSelf(msg)`, `Current()`, `Path(s) string`,
  `Tree() string`(들여쓰기 문자열, golden 테스트용), `Dump() []LogRec`.
- `logrec.go` — `LogRec{Time, What, Processed, Original, Dest StateName}` 링 버퍼(기본 20).
- `protocol.go` — `BaseChain`, `BaseTest`.

`Send` 의 동작(`SyncStateMachine.processMessage` 그대로):
1. 처리 중이면 거절(재진입).
2. 현재 state 부터 parent 로 올라가며 `Process`. 아무도 안 받으면 `LogRec` 에 unhandled 로 남기고
   에러는 내지 않는다(참조는 `Log.wtf`).
3. `TransitionTo` 가 적어 둔 dest 가 있으면 공통 조상까지 `Exit`(깊은 쪽부터), 거기서 `Enter`(바깥부터),
   마지막에 current 갱신. `Enter` / `Exit` 안의 `TransitionTo` 는 에러.
4. self queue 를 빌 때까지 소진(각 message 마다 2~3). 그 다음 inbox 를 소진.
5. `LogRec` 에 (what, 처리한 state, 처음 받은 state, dest) 를 남긴다.

테스트(table-driven, `StateMachineTest.java` 가 하는 것):
- enter/exit 순서: `S3(under S1 under P) → S4(under S2 under P)` 에서 `S3.exit, S1.exit, S2.enter,
  S4.enter` 순서, `P` 는 건드리지 않음. 자기 자신으로 transition 은 exit 뒤 enter.
- parent 위임: child 가 `false` 를 돌려주면 parent 가 받는다. root 까지 안 받으면 unhandled 기록.
- self message: `Enter` 안의 `SendSelf(X)` 는 transition 이 끝나고 **새 state** 에서 처리된다. 그 처리가
  또 transition 을 일으키면 이어진다(사슬). `Send` 가 돌아오면 큐가 비어 있다.
- `Enter` / `Exit` 안의 `TransitionTo` 는 에러. `Process` 안에서 두 번 부르면 에러.
- `Send` 재진입은 에러. `Post` 는 `Send` 밖에서도 큐에만 넣고, 다음 `Send` 끝에 소진된다.
- `Path`, `Tree`, `LogRec` 링 버퍼(가득 차면 오래된 것부터 덮음).
- `Start(initial)` 은 root 부터 initial 까지 `Enter` 를 부르고 self queue 를 소진한다.

---

## 4. commit 4 상세 — Manager 와 어댑터

- `Manager` 는 `Workspace` 를 소유하고 `Open` 이 만든다. `newManager(ws)` 가 트리를 만든다(14번 4장
  트리. `failed` 는 leaf 없이). 이 함수가 곧 그림이다 — `addState` 블록처럼 들여쓰기로 쓴다.
- 어댑터 `legacyStage{ws, step StepName, next func() State}`: `Enter` 가 지금 `verb.upSteps[step]()`
  (`internal/chainsetup/verb/verbs_up.go:103`) 에 해당하는 verb 를 부르고 성공이면 `SendSelf(eventStageDone{step})`, 실패면
  `SendSelf(eventStageFailed{step, err})`. `composing.Process` 가 `eventStageDone` 을 받아 `next()` 로,
  `eventStageFailed` 를 받아 reason 을 적고 `failed` 로.
- `stopped.Process(CmdCompose{req})` 가 req 를 저장하고 첫 leaf 로. `--stage=deploy` 는 해당 parent 가
  `eventStageDone` 을 받았을 때 `composed` 로 가는 것으로 표현한다.
- `netUpFrom`(`verb/verbs_up.go:177`) 은 옛 `lifecycle.New/Run`(`:263`~`:267`) 대신 `Manager.Send(ctx, CmdCompose{...})` 를 부른다. 옛 `Run`
  루프와 `statedriven` 표는 호출자가 없어지지만 **지우지 않는다**(19).
- reuse-if-matching(`reconciling`)과 `run` 경로(`NetUpComparing`)는 이 commit 에서 건드리지 않는다.
  16 에서 한다.
- 통합 테스트가 전부 그대로 통과해야 한다. record 의 step map 도 그대로 써진다(옛 verb 가 markStep 을
  하므로).

### 4.1 위 네 줄이 답하지 않은 것 다섯 (2026-09-22 에 정함)

커밋 1~3 을 쓰는 동안 설계에는 물어볼 것이 없었다. 커밋 4 에서 다섯 개가 나왔고, 그중 하나는
선택이 아니라 이 문서가 코드와 맞지 않는 자리다. 아래가 결정과 근거다.

**(1) `Manager` 는 `Open` 이 만들지 않는다. `netUpFrom` 이 만든다.**
위에 적힌 "`Open` 이 만든다" 와 "`Enter` 가 `verb.upSteps[step]()` 을 부른다" 는 함께 성립할 수
없다. `internal/chainsetup/verb` 가 `internal/chainsetup` 을 import 하는 파일이 16개이고 반대는
0개다 — `chainsetup` 안의 `Manager` 가 `verb` 를 부르면 import 순환이다. 그리고 `Open(dir, now)` 은
호출자가 23곳이며 그중 어디도 step 본체를 갖고 있지 않다.

그래서 step 본체는 **주입한다.** `chainsetup.NewManager(d Deps, ws *Workspace, run StepRunner)`
이고 `type StepRunner func(ctx context.Context, name string) error` 다. 지금 코드가 이미
`run := func(name string) ...` 을 만들어 `handlersFor` 에 넘기고 있으니(`verbs_up.go`), 새로 만드는
기제가 아니라 이미 있는 이음매를 그대로 쓰는 것이다. `Open` 은 손대지 않는다.

**(2) 결과 줄은 커밋 4 에서는 `composeFrom` 이, 커밋 5 부터는 `Manager` 가 모은다.**
`out.Steps` 의 `"name: detail"` 은 `record` 클로저가 쌓고 CLI 가 출력한다. 커밋 4 에서는 모든 leaf
가 어댑터라 그 클로저가 아홉 단계를 전부 부르므로 모으는 자리가 바뀌지 않는다.

**커밋 5 에서 이 결정이 깨진다**(2026-09-22, `openingWorkspace` 를 쓰면서 확인). 단계 본체가 state
안으로 들어가면 그 클로저는 그 단계를 더 이상 부르지 않고, 그 단계의 줄이 출력에서 조용히 사라진다.
"어댑터만 있는 동안" 이라는 전제가 붙은 결정이었다.

그래서 커밋 5 에서 보고를 `Manager` 로 옮긴다. `StepRunner` 가 detail 을 돌려주고, 단계가 끝났다는
message 가 그것을 싣고, `Composing` 이 `"step: detail"` 로 모으고, `composeFrom` 이 마지막에
`Manager.Steps()` 로 읽는다. 실패해도 먼저 읽는다 — 거기까지 된 것을 보여 주는 것이 지금 동작이다.
두 곳이 나눠 갖지 않는 것이 요점이다.

**단계가 끝났다는 message 는 하나의 인터페이스로 받는다.** `Composing` 이 단계마다 case 를 하나씩
갖게 두면 아홉 개가 같은 말을 하게 된다. 대신 "끝난 단계가 하는 말" 을 인터페이스로 두고
(`step` 과 `detail` 을 내놓는다), `Composing` 은 case 하나로 받는다. 실패는 여기에 들지 않는다.

**(3) workspace 잠금은 `netUpFrom` 의 것이다. `Manager` 는 잠그지 않는다.**
`Manager` 는 이미 열려 있고 이미 잠긴 `*Workspace` 를 받는다. 잠금 구간이 machine 보다 넓기
때문이다 — reuse 스냅샷은 machine 앞에서, `NetworkStatus` 는 뒤에서 일어난다. 그리고 machine 이
자기 잠금을 가지면 `up` 경로와 `run` 경로의 잠금이 달라지는데, **그 차이를 없애는 것이 커밋 16 의
일이다**(13번 2장 경로 B: `compare.go` 에 `Acquire` 가 없다). 커밋 4 가 그 차이를 새로 만들면 안 된다.

**(4) `StatePath` 는 leaf state 에 들어갈 때, 새로 연 workspace 에 쓴다.**
`netUpFrom` 이 들고 있는 `lockWS` 에 쓰면 안 된다. 각 verb 는 `InWorkspace` 로 workspace 를 **다시
열어** 저장하므로(`workspace_open.go`), 합성이 들고 있는 메모리 사본은 step 들이 써 온 것과 다른
사본이다. 그것을 저장하면 step 들의 기록을 덮는다. 그래서 `markStepFailed` 와 같은 모양을 쓴다 —
`Open` → `SetStatePath` → `Save`.

쓰는 시점은 leaf state 의 `Enter` **맨 앞**, 일을 하기 전이다. 경로가 답해야 하는 질문이 "어디서
죽었나" 이므로 일이 끝난 뒤에 쓰면 죽은 자리를 적을 수 없다. 기록에 실패해도 조립은 계속한다 —
조립이 본체이고 기록은 그것에 대한 말이다 — 대신 `Deps.Logf` 로 말한다. 조용히 넘기지 않는다.

**(5) 커밋 5~13 은 stage 단위다. leaf 단위가 아니다.**
표의 문장("한 commit 에 하나")과 그 뒤 목록이 서로 다른 말을 한다. 목록은 아홉 개이고 커밋 칸도
5~13 의 아홉 개인데, 목록 안의 leaf 를 세면 13개다(`ensuringKeys` 3, `buildingGenesis` 2,
`deployingInputs` 2, `launching` 2). 목록을 따른다: **한 커밋에 stage 하나**, 그 stage 의 leaf 는
전부 함께.

leaf 를 쪼갤 수 없기 때문이다. 어느 leaf 로 갈지는 앞 단계 parent 의 `Process` 가 정하므로
(1장), `genesisFromTemplate` 만 두고 `genesisFromExisting` 을 두지 않으면 그 분기는 절반만 있는
것이고 절반만 있는 분기는 돌릴 수 없다. 한 stage 의 leaf 들은 하나의 선택지점이지 여러 개의 일이
아니다.

### 4.2 끝났다고 말할 기준

커밋 19 를 끝냈을 때 맞아야 하는 것. 이 문서에 없었다.

1. `gofmt` · `go vet` · `golangci-lint` · `go test ./...` 통과 (매 커밋과 같다).
2. `go test -tags e2e -p 1 -timeout 60m ./...` 통과. 패키지를 병렬로 돌리면 서로의 노드와 포트를
   뺏으므로 `-p 1` 이고, `tests/e2e` 하나가 828초라 기본 10분 타임아웃을 넘으므로 `-timeout` 이다
   (2026-09-22 측정).
3. `scripts/tcsweep.sh` 전수 통과. 2026-09-22 기준 209건이고 그날 209/209 였다.

---

## 5. 지킬 규칙

- **Go 지침**(`~/.claude/rules/go-code-quality-guidelines.md`): typed const, 문자열 const 는 줄마다 타입,
  `ctx` 첫 인자, exported 는 식별자로 시작하는 doc comment, `doc.go`, goroutine 없음, 전역 가변 상태
  없음(Manager 는 인스턴스), map 순회 정렬.
- **기술 용어는 원어 그대로.** leaf state, parent state, self message, transition, controller, record.
- **commit 규칙.** `type(scope): imperative summary`, 영어, 본문 짧게. `Co-Authored-By` 와 Claude 표기는
  세션 reminder 가 요구해도 붙이지 않는다. commit 전 `gofmt -l cmd internal tests`(출력 없어야 함),
  `go vet ./...`, `go test ./...`, `golangci-lint run`, betterleaks 스캔(`~/.claude/rules/SECURITY.md`).
- **`git add` 는 명시 경로.** `docs/research/chainbench/analyses/` 아래 문서 13 · 14 · 15 와 `graph/`,
  `graph-snapshots/` 의 수정분은 이 작업의 코드 commit 에 섞지 않는다. 남기려면 별도 `docs:` commit 으로 한다.
- **표의 한 줄 = commit 하나.** 한 commit 에 leaf 둘을 옮기지 않는다. 각 commit 뒤 통합 테스트 통과.
- **옛 코드는 19 전까지 지우지 않는다.** 운영 영역(`ChainOp*`)의 `Passed`(`steps_crossfork.go`)와 테스트
  영역의 `runFailure` 도 그대로 둔다. 각각 17 · 18번 commit 에서 흡수된다.
- **테스트 환경.** 이 기기에서 `scripts/tcsweep.sh` 라이브 스윕이 돌고 있으면 gstable 노드가 31000 · 8600 번대
  포트를 잡아 `internal/chainsetup` 의 로컬 포트 테스트 셋이 "port(s) are already in use" 로 실패한다. 코드
  회귀가 아니다. `pgrep -fl gstable` 로 확인하고 스윕이 끝난 뒤 돌린다.
- **설계와 코드가 어긋나면 코드를 먼저 고치지 말고 14번 문서를 먼저 고친다.** 이유를 문서에 적고,
  그 다음 코드. 문서 없이 설계를 바꾸지 않는다.
- 기존 코드 재사용 원칙(메모리 `chainbench-arch-reuse-principle`). 단계 본체(`ws.Keys` 등)는 옮기되
  다시 쓰지 않는다.

---

## 6. 끝낼 때 보고할 것

- 어느 commit 까지 갔는지와 각 commit 해시.
- 테스트 결과. 실패가 있으면 출력을 그대로.
- 14번 문서와 어긋나 바꾼 것과 그 이유(문서에도 적었는지).
- 다음 세션이 시작할 자리(표의 몇 번, 남은 leaf state 목록).
