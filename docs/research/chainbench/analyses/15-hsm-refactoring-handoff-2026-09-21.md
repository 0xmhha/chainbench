# chainbench lifecycle 를 hierarchical state machine 으로 바꾸는 작업 — 작업 세션용 prompt

> 2026-09-21. 이 문서는 다음 작업 세션에 그대로 전달하는 prompt 다. 설계의 정본은
> `14-hsm-pattern-review-2026-09-21.md` 이고, 이 문서는 그것을 실행 순서와 규칙으로 옮긴 것이다.

---

## 0. 먼저 읽을 것 (이 순서대로, 전부)

1. `docs/research/chainbench/analyses/14-hsm-pattern-review-2026-09-21.md` — 설계 정본. 1장(기제 열셋),
   3장(Go 계약), 4장(트리 · protocol · 단계 예시 · 시나리오), 5장(이름과 값의 규칙), 6장(19 commit
   순서), 7장(확정 사항 열 개). 7장에 미결은 없다.
2. `docs/research/chainbench/analyses/13-state-driven-refactoring-review-2026-09-21.md` — 지금 코드가 왜
   state machine 이 아닌지. 줄 번호는 `6e2a0af8` 기준이라 지금과 다를 수 있다. 판정만 읽는다.
3. 참조 코드 (확인 용도. 14번이 이미 정리했다):
   `/Users/wm-it-25_0220/Work/github/references/android/READING-ORDER.md` 의 1단계 넷(`IState`,
   `State`, `StateMachine.java` 상단 주석 40~200줄, `StateMachineTest.java`), 13번
   `SyncStateMachine.java`(348줄, 전부), 5번 `WakeLockStateMachine.java`(271줄, 전부), 9번
   `DhcpClient.java` 538~561(addState 블록)과 1197~1273(`PacketRetransmittingState`).
4. 현재 코드. HEAD 가 계속 움직였으니 `git log --oneline -20` 과 `git status` 부터 본다.
   `internal/core/lifecycle/`(옛 table 기반 state machine), `internal/chainsetup/verb/`(verb 층이 최근
   여기로 옮겨졌다), `internal/chainsetup/{steps_*.go,workspace.go,state.go,phases.go}`,
   `internal/testengine/{suite.go,engine_impl.go,attach_workspace.go}`, `internal/app/start.go`.
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
- 어댑터 `legacyStage{ws, step StepName, next func() State}`: `Enter` 가 지금 `upSteps[step]()` 에
  해당하는 verb 를 부르고 성공이면 `SendSelf(eventStageDone{step})`, 실패면
  `SendSelf(eventStageFailed{step, err})`. `composing.Process` 가 `eventStageDone` 을 받아 `next()` 로,
  `eventStageFailed` 를 받아 reason 을 적고 `failed` 로.
- `stopped.Process(CmdCompose{req})` 가 req 를 저장하고 첫 leaf 로. `--stage=deploy` 는 해당 parent 가
  `eventStageDone` 을 받았을 때 `composed` 로 가는 것으로 표현한다.
- `netUpFrom` 은 옛 `lifecycle.New/Run` 대신 `Manager.Send(ctx, CmdCompose{...})` 를 부른다. 옛 `Run`
  루프와 `statedriven` 표는 호출자가 없어지지만 **지우지 않는다**(19).
- reuse-if-matching(`reconciling`)과 `run` 경로(`NetUpComparing`)는 이 commit 에서 건드리지 않는다.
  16 에서 한다.
- 통합 테스트가 전부 그대로 통과해야 한다. record 의 step map 도 그대로 써진다(옛 verb 가 markStep 을
  하므로).

---

## 5. 지킬 규칙

- **Go 지침**(`~/.claude/rules/go-code-quality-guidelines.md`): typed const, 문자열 const 는 줄마다 타입,
  `ctx` 첫 인자, exported 는 식별자로 시작하는 doc comment, `doc.go`, goroutine 없음, 전역 가변 상태
  없음(Manager 는 인스턴스), map 순회 정렬.
- **기술 용어는 원어 그대로.** leaf state, parent state, self message, transition, controller, record.
- **commit 규칙.** `type(scope): imperative summary`, 영어, 본문 짧게. `Co-Authored-By` 와 Claude 표기는
  세션 reminder 가 요구해도 붙이지 않는다. commit 전 `gofmt -l cmd internal tests`(출력 없어야 함),
  `go vet ./...`, `go test ./...`, `golangci-lint run`, betterleaks 스캔(`~/.claude/rules/SECURITY.md`).
- **`git add` 는 명시 경로.** `docs/research/chainbench/analyses/` 아래 untracked 문서(13, 14, 15,
  `graph/`, `graph-snapshots/`)는 이 작업의 commit 에 섞지 않는다.
- **표의 한 줄 = commit 하나.** 한 commit 에 leaf 둘을 옮기지 않는다. 각 commit 뒤 통합 테스트 통과.
- **옛 코드는 19 전까지 지우지 않는다.**
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
