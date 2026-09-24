# 구현된 hierarchical state machine 검토 — core/statemachine · chainsetup · testengine — [측정 + 제안]

> 작성 2026-09-22. 대상은 HEAD `33ecf83b` 와 작업 트리다(작업 트리의 변경은 `docs/` 뿐이라 코드는 HEAD 와 같다).
>
> 14번 문서(`14-hsm-pattern-review-2026-09-21.md`)가 설계의 정본이고, 15번 문서가 그 실행 prompt 다.
> 이 문서는 그 뒤에 **실제로 들어온 구현을 검토한 것**이다. 14번이 "무엇을 만들 것인가" 라면 이 문서는
> "만들어진 것이 설계대로인가, 어디가 깨져 있는가" 를 답한다.
>
> 근거는 전부 파일과 줄이다. 코드를 읽어 확인한 것과 **실제로 돌려서 확인한 것**을 구분해 적었다.
> 돌려서 확인한 것은 7장에 재현 방법을 남겼다. 확인하지 못한 것은 그렇게 적었다.
>
> 기술 용어(state, leaf state, parent state, enter/exit/process, self message, controller, transition,
> dispatch, drain)는 14번 문서의 규약대로 번역하지 않고 원어대로 쓴다.
>
> Android 참조 코드 경로는 `/Users/wm-it-25_0220/Work/github/references/android/` 아래이고,
> 읽는 순서는 같은 디렉터리의 `READING-ORDER.md` 에 있다. 이 문서에서 인용하는 것은
> `modules-utils/java/com/android/internal/util/State.java` 와 같은 디렉터리의 `StateMachine.java` 둘이다.

---

## 0. 결론부터

`internal/core/statemachine` 은 Android `StateMachine` 을 Go 로 옮긴 것이고, 문서 수준은 원본보다 낫다.
다만 원본이 **막아둔 자리를 Go 판이 열어놨고**, 그 자리 하나는 이미 실제 버그로 터져 있다.

급한 것부터 셋이다.

1. **`Comparing` 가지의 성공 경로가 끊겼다.** `chain up` 이 "네트워크를 내리고 다시 구성하라" 고
   판정하면, 내리기만 하고 재구성하지 않은 채 **성공을 보고한다.** 3.1 장.
2. **`Enter`/`Exit` 가 실패하면 machine 이 망가진 채로 계속 돈다.** 이미 나간 state 가 current 로 남고,
   들어간 state 가 영영 안 나온다. 돌려서 재현했다. 3.2 장.
3. **취소되면 정리가 통째로 건너뛰어진다.** 테스트 실행이 네트워크를 안 내리고 끝난다. 3.3 장.

Android 는 1번을 `unhandledMessage()` 훅으로 드러내고, 2번을 `enter()`/`exit()` 를 `void` 로 만들어
원천 차단하며, 3번을 `quit()` 으로 보장한다. 셋 다 Go 판에 없다. 4장에 표로 정리했다.

---

## 1. 무엇을 어떻게 확인했나

세 갈래로 봤다.

**코어**(`internal/core/statemachine`)의 여섯 파일은 전부 읽었다. 435줄짜리 `machine.go` 가 핵심이다.
읽다가 의심이 간 자리는 **임시 test 파일을 패키지 안에 만들어 실제로 돌려** 확인했다. 확인한 뒤
파일을 지우고 `git status internal/` 로 저장소가 깨끗한 것을 확인했다. 그 test 코드는 7장에 남긴다.

**소비자** 두 패키지(`internal/chainsetup` 21,521줄, `internal/testengine`)는 두 갈래로 나눠 조사했고,
올라온 지적 중 중요한 것은 원본 코드로 다시 확인했다. 확인한 것만 아래에 단정형으로 적었다.

**Android 원본**은 `references/android/modules-utils` 의 소스를 직접 열어 대조했다.

`go vet ./internal/core/statemachine/ ./internal/chainsetup/ ./internal/testengine/` 는 통과한다.

`go test ./internal/chainsetup/` 은 이 머신에서 9개가 실패하는데, **코드 회귀가 아니다.** 3.7 장에서
설명한다.

---

## 2. 설계가 갈라진 지점

Android 의 `State` 는 이렇다.

```java
public void enter() { }     // State.java:44
public void exit()  { }     // State.java:52
```

`void` 다. 실패할 수 없다. 실패를 알리려면 메시지로 보내야 하고, 그래서 transition 은 **언제나 끝까지
간다.** 반쯤 옮겨간 machine 이라는 상태가 존재하지 않는다.

chainbench 는 여기에 `error` 를 붙였다.

```go
// internal/core/statemachine/state.go:29
Enter(ctx context.Context, m *Machine) error
Exit(ctx context.Context, m *Machine) error
```

Go 답게 보이지만, 그 순간 "transition 이 중간에 실패한다" 는 경우가 생긴다. `machine.go:361` 의
`performTransition` 은 그 경우를 되돌리지 않는다. 3.2 장이 그 결과다.

같은 패키지 문서가 이미 옳은 답을 적어뒀다는 점이 아깝다.

> 도메인에 속한 실패는 이것(error)이 아니다. 그건 에러를 담은 메시지다. (`doc.go:53-59`)

원칙은 적혀 있는데 API 가 어긋난 쪽도 허용한다.

---

## 3. 확인한 결함

### 3.1 `Comparing` 가지의 성공 경로가 끊겼다 (가장 급함)

저장소 자신의 golden tree 가 증거다(`internal/chainsetup/manager_test.go:86-126`).

```
Composition
  Comparing
    RestartingNodes
    StoppingToRebuild
  Composing            ← Comparing 의 형제다
    OpeningWorkspace
    ...
```

`RestartingNodes` 가 일을 마치고 보내는 self message 는 `nodesRestarted` 다
(`internal/chainsetup/stage_compare.go:161`). 그런데 이걸 받는 곳은 `composingState.Process` 다
(`internal/chainsetup/manager_states.go:143`).

`Composing` 은 `RestartingNodes` 의 조상이 아니다. 형제의 자식일 뿐이다. tree 를 만드는 곳에서
확인된다(`internal/chainsetup/manager.go:162-166`).

`dispatch` 는 parent 를 따라 **위로만** 올라간다(`internal/core/statemachine/machine.go:322-333`).
그래서 실제 경로는 이렇다.

```
RestartingNodes → Comparing → Composition → 끝
```

`Comparing` 은 `comparisonMade` 만 처리하고 나머지는 `false` 를 돌려준다(`stage_compare.go:69-73`).
`Composition` 은 `stageFailed` 만 본다(`manager_states.go:41-51`). 아무도 안 받는다.

그리고 코어는 **아무도 안 받은 메시지를 에러로 보지 않는다.** 기록만 남기고 `Send` 는 `nil` 을
돌려준다(`machine.go:329-342`). 그걸 못 박은 test 까지 있다
(`machine_test.go:204` `TestProcess_AMessageNobodyHandlesIsRecordedAndNotAnError`).

결과가 둘이다.

`RebuildNodes` 판정이면 `Verifying` 에 못 들어가고 `RestartingNodes` 에서 멈춘다. 호출자는 성공으로
본다. `ChainUpComparing` 이 `out.At = mgr.At()` 로 경로를 돌려주므로, 바깥에서는 "성공했는데 위치가
`Composition/Comparing/RestartingNodes`" 로 보인다.

`RebuildAll` 판정이 더 나쁘다. `StoppingToRebuild.Enter` 가 네트워크를 **실제로 내린 뒤**
(`stage_compare.go:120-132`) `stoppedToRebuild` 를 보내는데 이것도 안 받힌다. 네트워크는 멈추고
재구성은 안 되고 명령은 성공이라고 말한다. "rebuild all" 이 "stop only" 가 된다.

실패 경로는 멀쩡하다. 두 state 가 실패할 때 보내는 `stageFailed` 는 조상인 `Composition` 이 받는다
(`stage_compare.go:127`, `stage_compare.go:156`). 끊긴 것은 성공 경로뿐이다.

기존 test 가 못 잡는 이유도 확인했다. `TestComparing_EachVerdictGoesItsOwnWay`
(`manager_test.go:564-589`)는 `c.nextFor(...)` 를 **직접** 부를 뿐 machine 을 걷지 않는다. 실제 walk 를
거치는 test 는 `TestComposeComparing_NothingComposedComposes` 하나인데, 그건 `Compose` 판정만 탄다.

고칠 곳은 둘 중 하나다. 두 `case` 를 `comparing.Process` 로 옮기거나, 두 leaf state 를 `Composing`
밑으로 옮기거나. 앞쪽이 맞아 보인다. 두 state 는 비교의 결과이지 구성 단계가 아니고, golden tree 도
그렇게 그려져 있다.

### 3.2 부분 transition 을 되돌리지 않는다

`performTransition` 은 나가는 쪽을 먼저 다 돌리고 들어가는 쪽을 돌린다. 들어가다 실패하면 그냥
반환한다. `m.current` 는 갱신되지 않는다(`machine.go:393-400`).

**돌려서 확인한 흔적이다.** 7장에 재현 코드가 있다.

```
[X.process  X.exit  Y.enter]   ← Y.enter 가 실패
Current() = "X"                ← 이미 나간 X 가 current 다
```

다음 `Send` 를 하면 나간 state 의 `Process` 가 또 돌고, `Exit` 가 **두 번째로** 돌아간다.

세 단계 tree 에서는 더 나쁘다.

```
[S3.process  S3.exit  P.exit  Q.enter  R.enter]   ← R.enter 가 실패
다음 Send:
[S3.process  S3.exit  P.exit  P.enter]
```

`Q` 는 들어갔는데 **영영 안 나온다.** current 가 `S3` 라서 나가는 걸음이 `Q` 를 지나가지 않기
때문이다. `S3` 과 `P` 는 `Exit` 가 두 번씩 돌았다. 파일을 두 번 닫고 타이머를 두 번 해제하는 모양이다.

`Exit` 가 실패할 때도 비슷하다. current 는 그대로인데 그 state 의 `Exit` 는 이미 한 번 돌았고,
재시도하면 또 돈다.

```
[B.process  B.exit]   ← B.exit 가 실패, Current() = B
[B.process  B.exit]   ← 다음 Send 에서 또
```

지금 이 구멍에 안 닿는 이유는 **두 machine 의 33개 `Enter` 가 전부 `nil` 을 돌려주기 때문이다.**
실패는 `mg.fail`(`manager.go:461-465`) 또는 `r.fail`(`runmachine.go:135-142`)을 거쳐 메시지로 간다.
규율은 지키고 있다.

문제는 그 규율을 강제하는 것이 아무것도 없다는 점이다. 린터도 타입도 test 도 없다. 새 state 를 쓰는
사람이 `return err` 한 줄을 넣으면 조용히 열린다. 그리고 그 계약이 `state.go` 의 doc 에 적혀 있지 않다.

기존 test `TestErrorFromAState_AbortsTheSend`(`machine_test.go:436`)는 에러가 나오는지만 본다. 그 뒤
machine 이 어떤 꼴로 남는지는 안 본다.

### 3.3 취소되면 정리가 건너뛰어진다

`dispatch` 는 메시지마다 맨 앞에서 ctx 를 본다(`machine.go:315-317`).

테스트 실행에서 네트워크를 내리는 곳은 `collecting.Enter` 다
(`internal/testengine/runmachine.go:518`). 거기까지 가려면 메시지가 한 번 더 전달돼야 한다.

ctx 가 취소되면 그 전달이 막힌다. `SendSelf` 로 넣어둔 `stageStopped` 도 같은 `dispatch` 를
지나가므로 함께 버려진다. 그래서 **취소에서 정리로 가는 길이 아예 없다.**

이 동작은 사고가 아니라 명시된 정책이다. `TestDrain_HonoursCancellation`(`machine_test.go:502`)이
그렇게 되는 것을 정답으로 못 박아뒀다.

다만 심각도는 확인한 두 가지로 낮아진다.

`InWorkspace` 는 실패 경로에서도 `ws.Save()` 를 부르고(`internal/chainsetup/workspace_open.go:52`),
그게 pid 장부를 디스크에 쓴다(`internal/chainsetup/workspace.go:252-258`). 그래서 남은 노드는 기록돼
있고 `chain stop --data-dir` 로 회수된다.

그리고 `cmd/chainbench/interrupt.go:26-31` 은 Ctrl-C 에서 노드를 남기는 것이 **의도**라고 적어뒀다.
기록을 먼저 지키려는 선택이고, 근거도 적혀 있다.

그래도 남는 문제가 있다. 테스트 실행은 `KeepUp=false` 면 자동으로 내리기로 한 것인데 그 약속이
깨진다. `r.out.Stopped` 는 `false` 로 남고 아무 말도 안 한다. 취소 이유가 Ctrl-C 가 아니라 상위
타임아웃이어도 똑같이 건너뛴다. 그리고 메시지가 도달한다 해도 `r.net.teardown(ctx)`
(`runmachine.go:519`)가 취소된 ctx 를 그대로 받으므로 바로 실패한다.

### 3.4 `Exit` 를 구현한 state 가 0개다

두 machine 을 합쳐 `Enter` 는 33개, `Exit` 는 **0개**다. 전부 `statemachine.Base` 의 빈 `Exit`
(`state.go:52`)를 쓴다.

Android 원본의 요점 중 하나가 정확히 이것이다. `exit()` 는 `enter()` 가 건 것을 거둔다. chainbench
자신의 `state.go:29` 도 같은 말을 적어뒀다. 절반만 쓰고 있다.

대부분은 안 샌다. `Enter` 가 자원을 state 의 수명이 아니라 **함수 호출의 수명**으로 잡기 때문이다.
워크스페이스 락은 `defer held.Release()` 로 같은 함수에서 놓고(`workspace_open.go:42-46`), 서버셋
할당 락도 같다(`steps_place.go:472-476`).

새는 것은 노드 프로세스 하나다. `launchingPhase.Enter`(`stage_launch.go:129-142`)가
`process.LaunchAndRecord` 로 띄운 프로세스는 detached 이고 어떤 `Exit` 도 안 내린다. 한 phase 에서
노드 3이 실패하면 이미 뜬 1·2 는 남는다(`phases.go:97-99`). phase 가 여럿일 때 나중 것이 실패해도
먼저 뜬 것이 남는다.

테스트 실행 쪽은 정리를 후속 state 인 `collecting.Enter` 로 옮겼다. 순서에는 이유가 있다. 이미 죽인
노드에서 증거를 못 걷기 때문이다(`runmachine.go:129-134`). 대신 정리가 "자원을 잡은 state 를 떠날 때"
가 아니라 "특정한 다른 state 에 도착할 때" 에 매달리게 됐고, 그게 3.3 의 원인이다.

### 3.5 controller 와 child 기제가 통째로 미사용

`doc.go:39-44` 가 길게 설명하는 부분이다. child 는 `Post` 로 올리고 controller 는 `Send` 로 내린다.
두 동기 machine 이 서로 `Send` 하면 재진입이 되니까 방향을 다르게 둔 것이다.

확인해보니 **안 쓴다.**

두 machine 다 controller 없이 만들어진다(`chainsetup/manager.go:126`, `testengine/runmachine.go:86`).
`Post()` 와 `Controller()` 를 부르는 곳은 `machine_test.go` 뿐이다.

메시지 번호 대역도 마찬가지다. `core/statemachine/protocol.go` 는 `+0x040` 을 "controller 에게 올리는
Cmd", `+0x180` 을 "아래에서 올라오는 Event" 로 정해뒀다. 그 대역에 있는 `CmdPostCompose`,
`CmdOnQuit`, `EventNodeDied` 는 **보내는 곳도 받는 곳도 없다**(`chainsetup/protocol.go:252-271`).

번호를 나눈 근거 자체가 "child 메시지가 controller 를 지나가니까 겹치면 안 된다" 인데, 지나갈
controller 가 없다.

`CmdStop`(`chainsetup/protocol.go:236-239`)도 죽어 있다. `Manager.Stop`(`manager.go:353`)은 이 메시지를
안 쓰고 `Operate{Name: "Stopping"}` 를 보낸다. 같은 일을 가리키는 어휘가 둘인데 하나는 안 돈다.

14번 문서는 testengine 을 parent 로, chainsetup 을 child 로 두고 `CmdStop` → `CmdOnQuit` 2단계 정지를
계획했다(14번 4장, IpClient 의 `StoppingState` 꼴). 그 계획이 구현에 안 들어왔다.

`chainsetup/protocol.go:425-428` 은 이 상태를 인정하는 주석을 달아뒀다("나머지는 자기 leaf state 와
함께, commit 하나씩"). 의도된 미래 대비로 읽힌다. 다만 지금 읽는 사람에게는 도는 줄로 보인다.

### 3.6 작은 것들

**run machine 에 test 가 하나도 없다.** `internal/testengine/*_test.go` 36개를 훑어 확인했다. transition
을 한 줄도 안 본다. `runprotocol_test.go` 는 메시지 이름표만 본다. `chainsetup` 쪽은 golden tree test
가 있다(`manager_test.go:84`). `machine.go:266-269` 가 "아무도 안 보는 그림은 코드와 어긋난다" 고
적어뒀는데 한쪽이 그 상태다. 3.1 의 버그도 그 test 가 있었으면 잡혔다.

**`cmd/chainbench/interrupt.go:27` 의 `app.withWorkspace` 는 없는 식별자다.** 저장소 전체 grep 에서 이
주석 한 줄만 나온다. 이름이 바뀌었는데 주석이 안 따라갔다.

**`AcquireSetLock` 이 ctx 를 안 받는다.** 시그니처가 `AcquireSetLock(setPath string, d Deps)` 이고
`time.Sleep` 으로 돈다(`setlock.go:66`). 10초 상한이 있으니 멈추진 않지만 취소에 반응 못 한다. 그리고
`session.AcquireLock` 에는 `d.Clock` 을 주입하면서 자기 deadline 과 poll 은 `time.Now()`/`time.Sleep`
을 직접 쓴다. 시계 주입이 절반만 돼 있어 test 에서 이 대기를 제어할 수 없다. 바로 옆
`steps_crossfork.go:257-261` 은 `select` 로 제대로 하고 있어 대비된다.

**모든 operation 이 기록된 위치를 `Composition/Ready` 로 덮어쓴다.** `readyState.Enter` 가
`recordPath` 를 부르는데(`manager_states.go:191-194`), operation state 들이 `Ready` 의 자식이라
어떤 operation 이든 `Ready` 를 먼저 거친다. `--stage=deploy` 로 멈춰둔 워크스페이스에 `stop` 을 걸면
`Composition/Composed` 였던 기록이 `Composition/Ready` 가 된다. 코드로 확인한 사실이고, `ResumeStep`
이 그걸 "끝났음" 으로 읽어 실제로 어떻게 잘못 도는지까지는 **확인하지 못했다.**

**`Post(nil)` 은 조용히 무시하고 `SendSelf(nil)` 은 misuse 로 잡는다**(`machine.go:178-183`,
`machine.go:218-227`). 같은 종류의 실수에 답이 둘이다.

**`failedState.Process` 가 `handled=true` 와 에러를 함께 돌려준다**(`manager_states.go:236`).
`dispatch` 는 에러가 있으면 handled 를 보기 전에 빠져나가므로(`machine.go:324-328`) `true` 는 의미가
없다. 동작 문제는 아니고 읽는 사람을 헷갈리게 한다.

**`launching.next` 가 받은 `*statemachine.Machine` 인자를 안 쓴다**(`stage_launch.go:112`).

### 3.7 test 실패는 회귀가 아니다

`go test ./internal/chainsetup/` 에서 9개가 실패한다. 원인을 찾아보니 **다른 세션이 지금 chainbench
라이브 테스트를 돌리고 있어서** 포트 8600 번대와 31000 번대를 물고 있었다. `lsof` 로 `gstable` 프로세스
두 개(pid 21270, 21271)와 `scripts/tcsweep.sh` 실행을 확인했다. 15번 문서가 시작 전에 확인하라고 적어둔
바로 그 상황이다. 아무것도 끄지 않았다.

코드 문제가 아니다. 다만 test 가 고정 포트를 쓰기 때문에 실제 실행과 같이 못 돈다는 점은 그 자체로
짚을 만하다.

---

## 4. Android 대비 빠진 기제

| 기제 | Android | chainbench | 영향 |
|---|---|---|---|
| `quit()` / `onQuitting()` | 있음. 활성 state 전부에 `exit()` 를 돌린다 (`StateMachine.java:2024`, `:914-921`) | **없음** | 끝난 machine 이 자원을 안 놓는다 |
| `transitionToHaltingState()` | 있음 (`:1431`) | **없음** | 망가진 machine 을 세울 방법이 없다 |
| `unhandledMessage()` 훅 | 있음 | 기록만 하고 훅 없음 | **3.1 이 조용히 지나간다** |
| `enter`/`exit` 실패 불가 | `void` (`State.java:44`, `:52`) | `error` 반환 | **3.2 의 원인** |
| `deferMessage` | 있음 | 의도적으로 뺌 (`doc.go:29-32`) | 지금은 문제 없다 |
| 지연 메시지 | 있음 | 의도적으로 뺌 | 대기를 `Enter` 안 블로킹으로 처리 |

뺀 둘은 근거가 적혀 있고 타당하다. 문제는 안 뺀 것이 아니라 **넣어야 할 셋을 안 넣은 것**이다.

---

## 5. 맞게 된 것

깎아내리려고 쓴 문서가 아니니 이쪽도 적는다.

`doc.go` 는 Go 코드베이스에서 보기 드문 수준의 설계 문서다. "왜 이렇게 했는지" 와 "이건 일부러 뺐고
언제 넣어야 하는지" 가 같이 적혀 있다. Android 원본 주석보다 낫다.

계층이 장식이 아니다. parent 가 공통 메시지를 받는 자리가 다섯 군데다. 루트가 어디서 올라오든
`stageFailed` 를 받고(`manager_states.go:41-51`), `Composing` 이 여덟 종류의 완료 보고를 `stageReport`
하나로 받는다(`manager_states.go:133-142`). 각 stage 는 자기 다음이 누구인지 모른다. 원본의 의도
그대로다. testengine 쪽도 `ReachingNetwork` 의 두 leaf 가 `networkReached` 를 직접 처리하지 않고
parent 에 올린다(`runmachine.go:408`, `:433`, `:360-380`).

`TransitionTo` 25곳이 전부 `Process` 안이다. `Enter` 에서 움직이려 할 때 self message 를 쓰는 규약도
지켰다. 코어가 이 오용을 거부하도록 만들어뒀는데(`machine.go:197-198`), 거부당할 일이 한 번도 없다.

`context.Background()` 와 `context.TODO()` 가 두 패키지 전체에 하나도 없다.

state 안에서 직접 띄우는 goroutine 이 없다. `Workspace.Stop` 의 goroutine 은 같은 함수에서
`wg.Wait()` 로 전부 회수한다(`steps_lifecycle.go:433-441`).

`maxDispatchPerSend`(`machine.go:17`)로 무한 루프를 잡는 것, 로그 ring 크기를 원본의 20에 맞춘 것
(`logrec.go:10`), `Tree()` 를 golden 파일과 대조하게 만든 것은 원본을 제대로 읽은 결과다.

---

## 6. 제안과 그 단점

우선순위 순이다.

**첫째, 3.1 을 고친다.** `nodesRestarted` 와 `stoppedToRebuild` 의 `case` 둘을
`manager_states.go:143-148` 에서 `comparing.Process`(`stage_compare.go:69`)로 옮긴다.

단점은 없다. 지금 안 도는 코드다. 다만 고치면 `RebuildAll` 경로가 **처음으로 실제로 돌게** 되므로,
그 경로에 다른 문제가 숨어 있을 수 있다. 재현 test 를 먼저 써서 RED 로 만든 뒤 고치는 편이 좋다.

**둘째, 부분 transition 을 막는다.** 두 갈래가 있다.

Android 처럼 `Enter`/`Exit` 에서 `error` 를 없애는 쪽이 근본적이다. 33곳이 이미 `nil` 만 돌려주므로
실제 수정량은 시그니처 변경뿐이다. 단점은 인터페이스를 깨는 것이고, `Enter` 가 진짜 못 할 일이
생겼을 때 표현 수단이 사라진다.

덜 과격한 쪽은 `Machine` 에 고장 표시를 두는 것이다. `Enter` 나 `Exit` 가 실패하면 표시를 켜고 이후
모든 `Send` 와 `Start` 를 같은 에러로 거부한다. 단점은 망가진 상태를 정리하지는 못한다는 것이다.
새는 자원은 그대로 샌다. 다만 **조용히 잘못 도는 것보다는 낫다.**

어느 쪽을 고르든 `state.go` 의 doc 에 계약을 적어야 한다. 지금은 적혀 있지 않다.

**셋째, 정리 경로를 만든다.** `Machine` 에 `Quit(ctx)` 를 더해 활성 state 를 안쪽부터 바깥쪽으로
`Exit` 시킨다. Android 의 `quit()` 이 하는 일이다. 그리고 그 경로만은 취소되지 않은 ctx 로 돌려야
한다. `context.WithoutCancel` 에 정리 예산을 붙이는 형태다.

단점이 분명하다. 지금 `Exit` 가 0개라 `Quit` 을 넣어도 당장은 아무 일도 안 한다. 정리 로직을
`collecting.Enter` 에서 `Exit` 로 옮기는 작업이 따라붙는다. 그리고 `runmachine.go:519` 의
`r.net.teardown(ctx)` 도 같이 고쳐야 한다.

`launching` 에 `Exit` 를 두어 "이번 진입에서 내가 띄운 노드" 만 내리는 안도 있다. 단점은 성공해서
`Ready` 로 넘어갈 때도 `Exit` 가 불린다는 것이다(`machine.go:386-392`). 성공과 실패를 구분할 장치가
필요해지고, 그러면 `Exit` 의 계약이 "Enter 가 잡은 것을 되돌린다" 에서 "실패했을 때만 되돌린다" 로
복잡해진다. 지금처럼 ledger 에 남기고 사람이 `chain stop` 으로 치우는 것도 선택지다. 다만 그게
의도라면 `launching` 의 doc 에 적혀 있어야 하는데 지금은 어디에도 없다.

**넷째, run machine 에 golden tree test 와 transition test 를 붙인다.** 비용이 가장 싸고 3.1 같은
버그를 앞으로 막는다. 단점은 tree 를 바꿀 때마다 golden 을 같이 고쳐야 하는 것인데, 그게 목적이다.

**다섯째, controller 와 child 기제를 결정한다.** 쓸 거면 14번 문서대로 잇고, 안 쓸 거면 `Post`,
`Controller`, 죽은 메시지 넷, 번호 대역 설명을 지운다. 지금처럼 놔두면 다음 사람이 그게 도는 줄 알고
읽는다.

---

## 7. 3.2 를 다시 확인하는 방법

아래 파일을 `internal/core/statemachine/` 에 두고 `go test ./internal/core/statemachine/ -run TestZZ -v`
로 돌린다. **확인 뒤에는 지운다.** 두 test 모두 실패하는 것이 정상이고, 실패 메시지가 아니라
`t.Logf` 가 찍는 trace 가 증거다.

```go
package statemachine

import (
	"context"
	"errors"
	"testing"
)

type zzProbe struct {
	n        StateName
	tr       *[]string
	enterErr error
}

func (p *zzProbe) Name() StateName { return p.n }
func (p *zzProbe) Enter(context.Context, *Machine) error {
	*p.tr = append(*p.tr, string(p.n)+".enter")
	return p.enterErr
}
func (p *zzProbe) Exit(context.Context, *Machine) error {
	*p.tr = append(*p.tr, string(p.n)+".exit")
	return nil
}
func (p *zzProbe) Process(context.Context, *Machine, Message) (bool, error) {
	*p.tr = append(*p.tr, string(p.n)+".process")
	return true, nil
}

type zzMover struct {
	n  StateName
	tr *[]string
	to State
}

func (p *zzMover) Name() StateName { return p.n }
func (p *zzMover) Enter(context.Context, *Machine) error {
	*p.tr = append(*p.tr, string(p.n)+".enter")
	return nil
}
func (p *zzMover) Exit(context.Context, *Machine) error {
	*p.tr = append(*p.tr, string(p.n)+".exit")
	return nil
}
func (p *zzMover) Process(_ context.Context, m *Machine, _ Message) (bool, error) {
	*p.tr = append(*p.tr, string(p.n)+".process")
	m.TransitionTo(p.to)
	return true, nil
}

type zzNote int

func (n zzNote) What() What { return What(n) }

// 실패한 Enter 뒤에 current 가 이미 나간 state 를 가리킨다.
func TestZZ_FailedEnterLeavesCurrentOnAnExitedState(t *testing.T) {
	var tr []string
	m := New("zz", nil)
	y := &zzProbe{n: "Y", tr: &tr, enterErr: errors.New("boom")}
	mv := &zzMover{n: "X", tr: &tr, to: y}
	m.Add(mv, nil)
	m.Add(y, nil)
	if err := m.Start(context.Background(), mv); err != nil {
		t.Fatal(err)
	}
	tr = nil
	if err := m.Send(context.Background(), zzNote(1)); err == nil {
		t.Fatal("실패한 Enter 가 Send 를 실패시키지 않았다")
	}
	t.Logf("trace = %v", tr)                      // [X.process X.exit Y.enter]
	t.Logf("Current() = %q", m.Current().Name())  // "X" — 이미 나간 state
	tr = nil
	_ = m.Send(context.Background(), zzNote(1))
	t.Logf("두 번째 Send trace = %v", tr)          // X.process 와 X.exit 가 또 돈다
	t.Fail()
}

// Enter 가 중간에 실패하면 먼저 들어간 ancestor 가 영영 안 나오고,
// 이미 나간 state 들이 Exit 를 두 번 받는다.
func TestZZ_PartialEnterLeaksAncestorAndDoubleExits(t *testing.T) {
	var tr []string
	m := New("zzB", nil)
	p := &zzProbe{n: "P", tr: &tr}
	q := &zzProbe{n: "Q", tr: &tr}
	r := &zzProbe{n: "R", tr: &tr, enterErr: errors.New("boom")}
	s3 := &zzMover{n: "S3", tr: &tr}
	m.Add(p, nil)
	m.Add(s3, p)
	m.Add(q, nil)
	m.Add(r, q)
	s3.to = r
	if err := m.Start(context.Background(), s3); err != nil {
		t.Fatal(err)
	}
	tr = nil
	if err := m.Send(context.Background(), zzNote(1)); err == nil {
		t.Fatal("실패한 Enter 가 Send 를 실패시키지 않았다")
	}
	t.Logf("trace = %v", tr)  // [S3.process S3.exit P.exit Q.enter R.enter]
	tr = nil
	s3.to = p
	_ = m.Send(context.Background(), zzNote(1))
	t.Logf("두 번째 Send trace = %v", tr)  // [S3.process S3.exit P.exit P.enter] — Q.exit 가 없다
	t.Fail()
}
```

`Exit` 실패 쪽은 `zzProbe` 에 `exitErr` 를 더해 같은 꼴로 확인한다.
