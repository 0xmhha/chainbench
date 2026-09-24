# 16번 검토 결과를 확인하고 고치는 작업 — 작업 세션용 prompt

> 2026-09-22 작성. 대상은 HEAD `33ecf83b` 와 작업 트리다.
>
> **이 문서는 16번의 지적을 믿고 고치라는 문서가 아니다.** 16번
> (`16-hsm-implementation-review-2026-09-22.md`)은 코드를 읽고 일부는 돌려서 쓴 검토지만, 그 세션은
> 이 코드를 만든 세션이 아니다. 설계 의도를 잘못 읽었을 수 있다.
>
> 그래서 **항목마다 먼저 확인하고, 그 다음에 고친다.** 확인 결과가 "문제 아님" 이면 그렇게 적고
> 넘어가는 것이 옳은 결과다. 지적 여섯 개를 여섯 개 다 고치는 것이 목표가 아니다.
>
> 기술 용어(state, leaf state, parent state, enter/exit/process, self message, transition, controller,
> dispatch, drain)는 14번 문서 규약대로 원어로 쓴다.

---

## 시작 전 확인 — 라이브 스윕이 도는지

`internal/chainsetup` 의 포트 테스트는 고정 포트(8600 번대, 31000 번대)를 쓴다. 라이브 스윕
(`scripts/tcsweep.sh`)이 돌고 있으면 그 테스트 아홉 개가 "port(s) are already in use" 로 실패한다.
**코드 회귀가 아니다.** 16번 3.7 장에 그 판정 근거가 있다.

```
pgrep -fl 'gstable|tcsweep'
lsof -nP -iTCP -sTCP:LISTEN | grep -E ':(31[0-9]{3}|86[0-9]{2})'
```

무엇이 남아 있으면 **끄지 않는다.** 다른 세션의 것이다. 두 경우로 나뉜다.

- **스윕이 도는 중이다.** 아래 2.1 · 2.2 · 2.6 은 그대로 할 수 있다. 이 셋의 확인 테스트는 포트를
  쓰지 않는다. 2.3 · 2.4 · 2.5 는 통합 테스트가 필요하니 스윕이 끝난 뒤로 미룬다.
- **아무것도 없다.** `go build ./... && go vet ./... && go test ./...` 로 baseline 을 먼저 잡는다.
  여기서 실패가 있으면 그것을 먼저 보고하고 멈춘다. 이 작업으로 고치지 않는다.

---

## 0. 먼저 읽을 것 (이 순서대로)

1. `16-hsm-implementation-review-2026-09-22.md` — 이 작업의 입력. 특히 3장(결함)과 6장(제안과 단점).
2. `14-hsm-pattern-review-2026-09-21.md` — 설계의 정본. 코드와 어긋나면 **어느 쪽이 틀렸는지** 를
   이 문서로 판정한다.
3. `internal/core/statemachine/doc.go` — 60줄. 코어의 계약이 전부 여기 있다.
4. `internal/chainsetup/manager.go:158-192` — tree 를 만드는 Add 블록.
5. `internal/chainsetup/manager_test.go:84-128` — golden tree.

Android 원본을 봐야 하면 `/Users/wm-it-25_0220/Work/github/references/android/READING-ORDER.md` 가
읽는 순서를 준다. 이 작업에서 필요한 것은 둘뿐이다.
`modules-utils/java/com/android/internal/util/State.java`(enter/exit 가 `void` 인 것)와
같은 디렉터리의 `StateMachine.java`(`quit()` · `transitionToHaltingState()`).

---

## 1. 이 작업의 규칙 — 확인이 먼저다

항목마다 이 순서를 지킨다. 건너뛰지 않는다.

1. **주장을 읽는다.** 16번의 해당 장.
2. **직접 확인한다.** 아래 각 항목의 "확인" 에 적힌 방법으로. 코드를 읽는 것만으로 끝내지 말고,
   돌릴 수 있는 것은 돌린다.
3. **판정을 적는다.** 세 가지 중 하나다.
   - `확인됨` — 주장대로다. 고친다.
   - `문제 아님` — 의도된 설계다. **왜 의도인지 근거를 적고** 16번 해당 장에 한 줄 덧붙인다. 코드는
     건드리지 않는다.
   - `일부만` — 주장 중 맞는 부분과 틀린 부분을 나눠 적는다. 맞는 부분만 고친다.
4. **고친다면 재현 테스트를 먼저 쓴다.** RED 를 확인하고 고쳐서 GREEN 으로 만든다. 테스트 없이
   고치지 않는다.

**판정을 미루지 않는다.** "나중에 볼 것" 으로 남기지 말고 셋 중 하나를 적는다.

---

## 2. 확인할 것 여섯

### 2.1 `Comparing` 가지의 성공 경로가 끊겼다 (최우선)

**주장**(16번 3.1). `RestartingNodes` 와 `StoppingToRebuild` 가 성공했을 때 보내는 self message 를
받는 곳이 `composingState.Process`(`manager_states.go:143-148`)인데, `Composing` 은 두 state 의 조상이
아니다. 그래서 메시지가 아무에게도 안 닿고, 코어는 그것을 에러로 보지 않으므로
(`machine.go:329-342`) `Send` 가 `nil` 을 돌려준다. **네트워크를 내리고 재구성하지 않은 채 성공을
보고한다.**

**확인.** 아래 테스트를 `internal/chainsetup/` 에 두고 돌린다. 네트워크도 노드도 필요 없다. 0.02초에
끝나므로 라이브 스윕 중에도 돌릴 수 있다.

```go
package chainsetup

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// 이 machine 이 보내는 self message 는 자기 조상 사슬 위에서 처리돼야 한다.
// dispatch 는 parent 를 따라 위로만 가므로(machine.go:322-333), 형제 가지의
// 처리기는 영원히 닿지 않는다.
func TestComparingLeaves_SuccessMessagesAreHandledOnTheirAncestorChain(t *testing.T) {
	mg := newTestManager(t)
	root := &compositionState{mg: mg}

	for _, c := range []struct {
		name string
		from statemachine.State
		msg  statemachine.Message
	}{
		{"nodesRestarted", mg.comparing.restarting, nodesRestarted{}},
		{"stoppedToRebuild", mg.comparing.stopping, stoppedToRebuild{}},
	} {
		path := mg.m.Path(c.from)

		var handledBy string
		for _, s := range []statemachine.State{c.from, mg.comparing, root} {
			ok, err := s.Process(context.Background(), mg.m, c.msg)
			if err != nil {
				continue
			}
			if ok {
				handledBy = string(s.Name())
				break
			}
		}
		if handledBy == "" {
			ok, _ := mg.composing.Process(context.Background(), mg.m, c.msg)
			t.Errorf("%s 는 %s 어디서도 처리되지 않는다 (Composing 이 처리=%v, 그런데 Composing 은 이 경로에 없다=%v)",
				c.name, path, ok, !strings.Contains(path, string(nameComposing)))
		}
	}
}
```

이 테스트를 돌렸을 때 나온 출력이 16번 3.1 의 근거다. 확인용으로 한 번 더 적는다.

```
nodesRestarted: 보내는 state 의 경로 = "Composition/Comparing/RestartingNodes"
  조상 사슬 어디서도 처리되지 않는다
  Composing.Process(nodesRestarted) = true, 그런데 Composing 이 경로에 있나 = false
stoppedToRebuild: (같음)
```

**주의.** `mg.m.Start(ctx, mg.comparing.restarting)` 로 machine 을 걷게 해서 확인하려 하면 안 된다.
`RestartingNodes` 에 들어가면 parent 인 `Comparing` 의 Enter 도 같이 돌고, 그 Enter 가
`compareWorkspace` 를 부른다(`stage_compare.go:56-58`). 빈 request 로는 거기서 먼저 실패해
`Composition/Failed` 로 가므로 무엇도 증명하지 못한다. 위 테스트는 `dispatch` 가 하는 일을 손으로
흉내 내기 때문에 그 함정을 피한다.

**확인되면.** 두 `case` 를 `manager_states.go:143-148` 에서 `comparing.Process`
(`stage_compare.go:69`)로 옮긴다. 두 state 는 비교의 결과이지 구성 단계가 아니고, golden tree 도
그렇게 그려져 있다. leaf state 를 `Composing` 밑으로 옮기는 반대 방향은 **고르지 않는다.** golden
tree 와 14번 문서를 둘 다 고쳐야 한다.

**고친 뒤 반드시 볼 것.** `RebuildAll` 경로는 지금까지 **한 번도 끝까지 돈 적이 없다.** 고치면 처음
돈다. `stoppingToRebuild` 의 doc(`stage_compare.go:106-110`)이 "init 과 start 는 recorded pid 를 가진
노드를 건너뛰므로, 돌고 있는 네트워크 위에 다시 composition 하면 genesis 만 덮어쓰고 모든 노드가 옛
것을 서빙한다" 고 적어뒀다. 그 경로가 이제 실제로 도니, 통합 테스트로 한 번 걸어본다. 라이브 스윕이
끝난 뒤에 한다.

### 2.2 부분 transition 을 되돌리지 않는다

**주장**(16번 3.2). `Enter` 나 `Exit` 가 실패하면 `performTransition`(`machine.go:361-402`)이 되돌리지
않는다. current 가 이미 나간 state 를 가리키고, 먼저 들어간 ancestor 는 영영 안 나오며, 이미 나간
state 가 `Exit` 를 두 번 받는다.

**확인.** 16번 7장에 재현 테스트가 통째로 있다. 그대로 복사해 `internal/core/statemachine/` 에 두고
`go test ./internal/core/statemachine/ -run TestZZ -v` 로 돌린다. 포트를 안 쓴다. 확인 뒤 지운다.

`t.Logf` 가 찍는 trace 가 증거다. 아래와 같이 나오면 주장대로다.

```
[X.process X.exit Y.enter]   Current() = "X"
[S3.process S3.exit P.exit Q.enter R.enter]  →  다음 Send: [S3.process S3.exit P.exit P.enter]
```

두 번째 줄에 `Q.exit` 가 없는 것이 핵심이다.

**확인되면.** 둘 중 하나를 고른다. 16번 6장에 각각의 단점이 적혀 있으니 읽고 고른다.

- **(a) `Enter`/`Exit` 의 `error` 를 없앤다.** Android 가 `void` 로 막은 방식이다
  (`State.java:44`, `:52`). 33곳이 이미 `nil` 만 돌려주므로 실제 수정은 시그니처와 호출부뿐이다.
  단점은 인터페이스를 깨는 것.
- **(b) `Machine` 에 고장 표시를 둔다.** `Enter`/`Exit` 가 실패하면 표시를 켜고 이후 `Send` 와
  `Start` 를 같은 에러로 거부한다. 단점은 망가진 상태를 정리하지 못하는 것. 새는 자원은 샌다.

**어느 쪽을 고르든 `state.go` 의 doc 에 계약을 적는다.** 지금은 "Enter 가 실패하면 그 state 는 Exit 를
받지 못한다" 가 어디에도 없다. 이것이 이 항목에서 가장 중요한 산출물이다. 코드를 안 고치기로 해도
이 문장은 적는다.

**(a) 를 고르면 14번 문서도 같이 고친다.** 계약 스케치(14번 3장)가 `error` 를 돌려주는 모양으로
적혀 있다.

### 2.3 취소되면 정리가 건너뛰어진다

**주장**(16번 3.3). `dispatch` 가 메시지마다 맨 앞에서 ctx 를 본다(`machine.go:315-317`). 테스트
실행의 teardown 은 `collecting.Enter`(`runmachine.go:518`)에 있어서 메시지가 한 번 더 전달돼야
닿는다. 취소되면 그 전달이 막히고, `SendSelf` 해둔 `stageStopped` 도 함께 버려진다. 그래서 취소에서
정리로 가는 길이 없다.

**확인.** 세 가지를 따로 본다.

1. `machine_test.go:502` `TestDrain_HonoursCancellation` 을 읽는다. 큐에 남은 self message 가 버려지는
   것을 정답으로 못 박아뒀는지.
2. `runmachine.go:115-126` 의 `Run` 을 읽는다. `Send` 가 에러를 돌려줄 때 teardown 을 부르는 곳이
   있는지.
3. 실제로 걸어본다. 라이브 스윕이 끝난 뒤, suite 를 `KeepUp=false` 로 돌리다가 Ctrl-C 를 한 번 누르고
   `lsof` 로 노드가 남는지 본다.

**판정에 넣을 것.** 16번은 심각도를 낮추는 근거도 같이 적었다. `InWorkspace` 가 실패 경로에서도
`ws.Save()` 를 불러(`workspace_open.go:52`) pid 가 디스크에 남으므로 `chain stop --data-dir` 로
회수된다. 그리고 `cmd/chainbench/interrupt.go:26-31` 이 Ctrl-C 에서 노드를 남기는 것을 **의도**라고
적어뒀다.

그러니 판정이 `문제 아님` 일 수 있다. 그 경우 **테스트 실행의 `KeepUp=false` 약속은 어떻게 되는지**
를 같이 답해야 한다. 자동으로 내리기로 한 것이 안 내려가는데 아무 말도 안 하는 것
(`r.out.Stopped` 가 `false` 로 남는다)이 맞는지.

**확인되어 고친다면.** 16번 6장 셋째 항목을 따른다. `Machine` 에 `Quit(ctx)` 를 더하고, 정리 경로만은
취소되지 않은 ctx 로 돈다. `runmachine.go:519` 의 `r.net.teardown(ctx)` 도 같이 고쳐야 한다. 취소된
ctx 를 그대로 받고 있어 메시지가 닿아도 실패한다.

**규모가 크면 여기서 멈추고 보고한다.** 2.1 과 2.2 를 먼저 닫는 것이 낫다.

### 2.4 `Exit` 를 구현한 state 가 0개다 — 판단이 필요하다

**주장**(16번 3.4). 두 machine 합쳐 `Enter` 33개, `Exit` 0개다. Android 의 `exit()` 는 `enter()` 가 건
것을 거두는 자리인데 절반만 쓰고 있다. 새는 것은 노드 프로세스다.

**확인.**

```
grep -rn ") Exit(" internal/chainsetup/*.go internal/testengine/*.go | grep -v _test
```

비어 있으면 주장대로다. 그리고 `phases.go:97-99` 를 읽어, 한 phase 에서 노드 3이 실패할 때 이미 뜬
1 · 2 가 남는지 확인한다.

**이것은 버그가 아니라 설계 선택일 수 있다.** 그래서 고칠지 말지를 판단해서 적는다.

`launching` 에 `Exit` 를 두는 안의 단점이 16번 6장에 있다. 성공해서 `Ready` 로 넘어갈 때도 `Exit` 가
불리므로(`machine.go:386-392`) 성공과 실패를 구분할 장치가 필요해지고, `Exit` 의 계약이 "Enter 가
잡은 것을 되돌린다" 에서 "실패했을 때만 되돌린다" 로 복잡해진다.

**지금처럼 ledger 에 남기고 사람이 `chain stop` 으로 치우는 것도 정당한 선택이다.** 그 경우 **코드는
안 고치고 `launching` 의 doc comment 에 그렇게 적는다.** 지금은 어디에도 없어서, 읽는 사람이 빠뜨린
것인지 고른 것인지 알 수 없다. 이 항목의 최소 산출물은 그 한 문단이다.

### 2.5 controller 와 child 기제가 미사용 — 결정이 필요하다

**주장**(16번 3.5). 두 machine 다 controller 없이 만들어지고(`manager.go:126`, `runmachine.go:86`),
`Post()` 와 `Controller()` 를 부르는 곳은 `machine_test.go` 뿐이다. 위로 올리는 대역의 메시지
(`CmdPostCompose`, `CmdOnQuit`, `EventNodeDied`)와 `CmdStop` 은 보내는 곳도 받는 곳도 없다.

**확인.**

```
grep -rn "statemachine.New(" internal/ --include='*.go' | grep -v _test
grep -rn "\.Post(\|Controller()" internal/ cmd/ --include='*.go' | grep -v _test
grep -rn "CmdPostCompose\|CmdOnQuit\|EventNodeDied\|CmdStop" internal/ --include='*.go' | grep -v "protocol.go:"
```

**이것도 버그가 아니라 미완성이다.** 14번 문서 4장이 testengine 을 parent, chainsetup 을 child 로 두고
`CmdStop` → `CmdOnQuit` 2단계 정지를 계획했다. 결정할 것은 하나다.

- **계획대로 잇는다.** 그러면 2.3 의 정리 문제도 같이 풀린다. IpClient 의 `StoppingState` 가 child 의
  `CMD_ON_QUIT` 을 기다리는 꼴이다(14번 1.11). 큰 작업이다.
- **안 잇기로 한다.** 그러면 `Post`, `Controller`, 죽은 메시지 넷, `core/statemachine/protocol.go` 의
  번호 대역 설명을 지운다. 번호를 나눈 근거가 "child 메시지가 controller 를 지나간다" 인데 지나갈
  controller 가 없으므로, 설명이 남아 있으면 다음 사람이 그게 도는 줄 알고 읽는다.

**어느 쪽이든 14번 문서를 먼저 고치고 코드를 고친다.** 설계 변경이다.

`chainsetup/protocol.go:425-428` 이 "나머지는 자기 leaf state 와 함께, commit 하나씩" 이라고 적어뒀다.
그 계획이 아직 유효하면 `문제 아님` 으로 적고, 언제까지인지만 덧붙인다.

### 2.6 작은 것 여섯

각각 독립이다. 확인이 5분이면 끝난다. 맞으면 고치고, 한 commit 에 묶어도 된다.

| # | 주장 | 확인 | 고치면 |
|---|---|---|---|
| a | run machine 에 test 가 하나도 없다 | `internal/testengine/*_test.go` 에서 `runner`/`nameRun`/`Tree()` 를 grep | golden tree test 와 transition test 를 붙인다. **2.1 같은 버그를 앞으로 막는 것이 이 항목의 목적이다** |
| b | `interrupt.go:27` 의 `app.withWorkspace` 는 없는 식별자 | `grep -rn withWorkspace .` 가 그 주석 한 줄만 내는지 | 주석을 실제 이름으로 고친다 |
| c | `AcquireSetLock` 이 ctx 를 안 받는다 (`setlock.go:48`, `:66`) | 시그니처와 `time.Sleep` 확인 | `ctx` 를 첫 인자로 받고 `select` 로 바꾼다. 바로 옆 `steps_crossfork.go:257-261` 이 본보기다. 시계도 `d.Clock` 으로 통일 |
| d | 모든 operation 이 기록 위치를 `Composition/Ready` 로 덮어쓴다 (`manager_states.go:191-194`) | `--stage=deploy` 로 멈춘 워크스페이스에 `stop` 을 걸고 `StatePath` 를 본다. **16번은 resume 이 실제로 어떻게 잘못 도는지까지는 확인하지 못했다.** 거기부터 확인한다 | 피해가 실재하면 고친다. 아니면 `문제 아님` |
| e | `Post(nil)` 은 조용히 무시, `SendSelf(nil)` 은 misuse (`machine.go:178`, `:218`) | 두 함수를 읽는다 | 한쪽으로 맞춘다. 어느 쪽이든 doc 에 적는다 |
| f | `failedState.Process` 의 `handled=true` 가 무의미 (`manager_states.go:236`), `launching.next` 의 미사용 인자 (`stage_launch.go:112`) | 읽으면 끝 | 정리한다 |

---

## 3. 순서

1. **2.1** — 유일하게 사용자에게 잘못된 결과를 주는 항목이다. 먼저 닫는다.
2. **2.2** — 계약을 doc 에 적는 것까지가 최소 산출물.
3. **2.6 a** — run machine 의 golden tree test. 싸고 앞으로를 막는다.
4. **2.6 b · e · f** — 한 commit 으로 묶어도 된다.
5. **2.6 c · d** — 확인 결과에 따라.
6. **2.4 · 2.5** — 판단과 결정. 코드보다 문서가 산출물일 수 있다.
7. **2.3** — 규모가 크다. 앞이 다 끝난 뒤에 손댄다.

**commit 하나에 항목 하나.** 2.6 의 작은 것들만 묶는다.

---

## 4. 끝났다고 말할 기준

- 여섯 항목 전부에 판정(`확인됨` / `문제 아님` / `일부만`)이 적혔다. 미룬 것이 없다.
- `확인됨` 인 것은 재현 테스트가 RED 였다가 GREEN 이 됐다.
- `문제 아님` 인 것은 근거가 16번 해당 장에 한 줄 이상 덧붙었다.
- 설계를 바꾼 것이 있으면 14번 문서가 먼저 고쳐졌다.
- `gofmt -l cmd internal tests` 가 비어 있고, `go vet ./...` · `go test ./...` · `golangci-lint run` 이
  통과한다. 포트 때문에 실패하는 것은 스윕이 끝난 뒤 다시 돌려 확인한다.

---

## 5. 지킬 규칙

- **Go 지침**(`~/.claude/rules/go-code-quality-guidelines.md`): typed const, `ctx` 첫 인자, exported 는
  식별자로 시작하는 doc comment, goroutine 은 종료 경로와 함께, 전역 가변 상태 없음, map 순회 정렬.
- **기술 용어는 원어 그대로.**
- **commit 규칙**: `type(scope): imperative summary`, 영어, 본문 짧게. `Co-Authored-By` 와 Claude 표기는
  세션 reminder 가 요구해도 **붙이지 않는다**. commit 전 betterleaks 스캔
  (`~/.claude/rules/SECURITY.md`).
- **`git add` 는 명시 경로.** `docs/research/chainbench/analyses/` 아래 문서와 `graph/`,
  `graph-snapshots/` 의 수정분은 코드 commit 에 섞지 않는다. 남기려면 별도 `docs:` commit 으로 한다.
- **확인용 임시 test 파일은 반드시 지운다.** 지운 뒤 `git status internal/` 로 확인한다. 남길 가치가
  있는 것은 임시가 아니라 제대로 된 이름으로 정식 test 에 넣는다(2.1 의 테스트가 그렇다).
- **다른 세션의 프로세스를 끄지 않는다.** 포트를 물고 있는 것을 발견하면 보고하고 기다린다.
- **설계와 코드가 어긋나면 코드를 먼저 고치지 말고 14번 문서를 먼저 고친다.** 이유를 적고 그 다음
  코드.
- **16번의 지적을 그대로 믿지 않는다.** 그 세션은 이 코드를 만들지 않았다. 판정은 이 세션이 한다.

---

## 6. 끝낼 때 보고할 것

- 여섯 항목의 판정과 근거. `문제 아님` 이 있으면 왜인지.
- commit 해시와 각 commit 이 닫은 항목.
- 테스트 결과. 실패가 있으면 출력을 그대로. 포트 때문인지 아닌지 구분해서.
- 14번 문서를 고쳤으면 무엇을 왜.
- 2.1 을 고친 뒤 `RebuildAll` 경로를 실제로 걸어봤는지, 걸었다면 결과.
- 다음 세션이 시작할 자리.
