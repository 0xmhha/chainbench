# 상태 주도 리팩토링 검토 — chainsetup · testengine — [측정]

> 처음 쓴 날 2026-09-21(커밋 `6e2a0af8` 기준). **2026-09-22 에 HEAD `ea267fd8` 기준으로 다시 쟀다.**
> 그 사이 17 커밋이 들어갔고, 이 문서의 수치·경로·줄 번호는 전부 새 트리의 것이다. 판정은
> 바뀌지 않았고, 무엇이 바뀌었는지는 5장에 있다. 대안 설계는 14번, 작업 순서는 15번이 정본이다.
>
> 그래프 자료: [`graph/code-graph.json`](graph/code-graph.json)(tree-sitter, 808 파일),
> [`graph/callgraph-state.json`](graph/callgraph-state.json)(go/types 호출 그래프, 799 함수 · 2,085 호출),
> [`graph/lifecycle-transitions.json`](graph/lifecycle-transitions.json)(전이표 추출, 도달 여부 포함).
> 9월 21일자(`6e2a0af8`)는 [`graph-snapshots/6e2a0af8/`](graph-snapshots/6e2a0af8/) 에, 9월 3일자
> (`cd22f8ed`)는 [`graph-snapshots/cd22f8ed/`](graph-snapshots/cd22f8ed/) 에 있다.
>
> 기술 용어(state, machine, handler, transition, record)는 원어대로 쓴다.

---

## 0. 결론부터

**지금 코드는 "state 를 검증하는 machine" 이지 "state 가 이끄는 machine" 이 아니다.** 9월 21일의
판정이 그대로다. 근거 셋도 그대로다.

1. **state 가 어디에도 남지 않는다.** `lifecycle.Machine` 은 `netUpFrom` 안에서 만들어져 함수가
   끝나면 사라진다(`internal/chainsetup/verb/verbs_up.go:263`, `:267`). record 에 적히는 것은 여전히
   이름 키의 step map 이다(`internal/chainsetup/state.go:104`). resume 은 그 map 과 상수 배열
   `UpStepNames` 로 "어디까지 왔나" 를 다시 계산한다(`internal/chainsetup/record.go:185`,
   `internal/chainsetup/state_request.go:122`).
2. **단계는 machine 밖에서도 그대로 호출된다.** `chain keys` 같은 단일 명령은 machine 없이 같은
   verb 를 부른다. 그래서 verb 는 `require()`(7곳) · `allow()`(8곳) · 손으로 쓴 "run `chain X` first"
   문구(14곳)를 그대로 들고 있다.
3. **handler 는 일을 하지 않고, 일이 끝난 뒤 지나온 state 를 재생한다.** verb 가 `Passed
   []lifecycle.Status` 를 돌려주면(16곳) handler 가 그것을 `m.Request` 로 되풀이한다
   (`internal/chainsetup/verb/statedriven.go:162`~`204`).

**9월 22일에 새로 보이는 것 하나.** `lifecycle.Status` 가 세 영역에서 **세 가지 다른 것**으로
쓰인다. 조립 영역(0x1000)에서는 machine 이 걷는 state 다. 운영 영역(0x3000)에서는 실패를 분류하는
값이고, cross-fork 가 돌려주는 `Passed` 를 걷는 machine 은 없다. 테스트 영역(0x8000)에서는 실행이
끝난 뒤 에러를 분류해 `RunSuiteOut.FailedAt` 에 넣는 값일 뿐이다. 이름은 하나인데 뜻이 셋이다.

---

## 1. 무엇을 어떻게 쟀나

- **모듈 그래프** — codemine `extract_graph.py`(tree-sitter). 808 파일, 24 모듈, 665 타입.
- **호출 그래프** — `go/packages` + `go/types` 로 `chainsetup`(`verb` 포함) · `testengine` · `app` ·
  `core/lifecycle` · `mcp` · `cmd/chainbench` 의 함수 799개 사이 호출 2,085건.
- **전이표 추출** — `internal/core/lifecycle/transitions.go` 의 `names`(117개)와 `allowed`(51 키,
  state 111개)를 파싱해 프로덕션 코드가 참조하는 state 를 셌다(4장).

| 모듈 | 파일 | 줄 | 함수 |
|---|---|---|---|
| `internal/chainsetup` (`verb/` 포함) | 103 | 18,650 | 553 |
| `internal/chainsetup/verb` | 10 (테스트 제외) | 2,708 | — |
| `internal/testengine` | 60 | 9,988 | 312 |
| `internal/app` | 35 | 4,421 | 201 |
| `internal/core/lifecycle` | 6 | 1,772 (테스트 포함; `status.go` 599) | — |

`Workspace` 의 메서드는 126개다.

**빌드와 테스트.** `go build ./...` 는 통과한다. `go test` 는 `internal/chainsetup` 에서 셋이 실패한다
(`TestNetStepPipeline`, `TestNetResume_ContinuesFromTheFirstUnfinishedStep`,
`TestNetUp_ReuseRefusalLeavesTheRunningCompositionUntouched`). 셋 다 "포트가 이미 사용 중" 이고,
잰 시각에 이 기기에서 `scripts/tcsweep.sh` 라이브 실행이 gstable 노드 셋을 31000 · 8600 번대에
띄워 두고 있었다. 코드 회귀가 아니라 환경이다. `verb` · `testengine` · `app` · `lifecycle` 은 통과한다.

---

## 2. 체인 셋업은 실제로 어떻게 도는가

### 진입: 네 가지 요청이 하나의 시작 state 가 된다

`chainbench run` 과 MCP `chainbench_run` 은 `app.StartFor` 를 부른다(`internal/app/start.go:73`).
`--attach`, `--rpc`, `--workspace-dir`, 정의서의 `env.attach` 를 우선순위대로 보고 시작 state 하나를
돌려준다. **그런데 이 값은 machine 에 들어가지 않는다.** `Adopts()` 가 참이면 `AttachRun`, 거짓이면
`RunSuite` 로 갈린다(`:118`). `Start.At` 은 네 값짜리 enum 이다.

### 경로 A — `chain up` / `chain resume`: machine 이 돈다

`NetUp` → `netUpFrom`(`internal/chainsetup/verb/verbs_up.go:22`, `:177`).

1. `planUp` 이 요청을 검사한다(`:60`).
2. workspace 를 열고 한 번 잠근다(`:200`).
3. reuse-if-matching 이면 스냅샷을 뜬다(`:213`).
4. `record` 클로저가 step 이름으로 verb 를 부르고 실패를 record 에 남긴다(`:228`).
5. 시작 state 는 `startFor`, 목표는 `targetFor`(`internal/chainsetup/verb/statedriven.go:271`, `:288`).
6. `lifecycle.New` 와 `Run`(`internal/chainsetup/verb/verbs_up.go:263`, `:267`). machine 은 현재 block 의 handler 를 부르고,
   handler 가 `Request` 한 state 를 보고 다시 돈다(`internal/core/lifecycle/machine.go:130`).
7. **machine 의 마지막 state 는 버려진다.**

### 경로 B — `run`(조립): 다른 machine 이 돈다

`RunSuite` → `runSuiteBody`(`internal/testengine/suite.go:253`, `:261`) → `composeWorkspace`(`:299`) →
`NetUpComparing`(`internal/chainsetup/verb/compare.go:73`). 이 machine 은 `CompareChain` 에서
시작한다(`:75`). `StartFor` 가 고른 시작 state 와 다르다. 바깥 잠금이 없다(`compare.go` 에 `Acquire`
없음). reuse-if-matching(`ReconcileChain`)에 닿지 않는다 — 그것은 `netUpFrom` 만 태운다
(`internal/chainsetup/verb/verbs_up.go:213`). 목표가 `ChainVerify` 인데 VERIFY handler 는 없고, 준비 검사는 machine 이 돌아온
뒤 `gateReady` 가 밖에서 한다(`internal/testengine/nodegate.go:277`).

### 경로 C — attach 셋: machine 이 없다

`AttachRun`(`internal/app/workflow.go:97`)은 `At` 의 block 만 확인하고(`:119`) `AttachWorkspaceRun`
이나 `NewAttachEngine` 으로 간다. 어느 쪽도 `lifecycle.New` 를 부르지 않는다. 프로덕션에서
`lifecycle.New` 를 부르는 곳은 둘뿐이다(`internal/chainsetup/verb/verbs_up.go:263`, `internal/chainsetup/verb/compare.go:75`).

### 한 단계가 도는 방법

```
Machine.Run
 └ handlersFor 가 만든 handler (internal/chainsetup/verb/statedriven.go:162)
    └ composeStage.run (:206)
       └ record 클로저 (internal/chainsetup/verb/verbs_up.go:228)
          └ upSteps["keys"] 클로저 (:103)
             └ verb.NetKeys (internal/chainsetup/verb/verbs_steps.go)
                └ chainsetup.InWorkspace: Open → Acquire → ws.Keys → Save → Release (internal/chainsetup/workspace_open.go:28)
                   └ Workspace.Keys: 일을 하고 markStep 하고 Passed 를 돌려줌
       ← Passed 를 m.Request 로 하나씩 재생 (internal/chainsetup/verb/statedriven.go:162~204)
       ← 다음 block 으로 m.Request(stage.next)
```

여섯 겹이다. handler 서명이 `(ctx, m, at)` 이라 `Workspace` 를 받을 자리가 없고
(`internal/core/lifecycle/machine.go:59`), 단계마다 workspace 를 다시 열고 잠그고 저장한다.
`WithWorkspace` 호출자 18, `InWorkspace` 호출자 5, `Open` 호출자 23 이다(호출 그래프).

### 실패는 어떻게 state 가 되나

verb 가 실패를 돌려주면 `composeStage.failed` 가(`internal/chainsetup/verb/statedriven.go:241`) `Passed` 를 먼저 재생하고,
`classify` 로 실패 state 를 고른다. **9월 22일 기준 아홉 단계 전부 `classify` 가 있다**
(`stagesStillAssuming = 0`, `stagesStillOwing = 0`, `:115`, `:119`). 분류는 sentinel 에러를
`lifecycle.Mark` 로 붙이고(`internal/core/lifecycle/mark.go:31`) `errors.Is` 로 고르는 방식이다.
그러니 실패 하나가 여전히 세 어휘를 거친다: sentinel → `Status` → record 의 `StepFailed` + `Err`
문자열. record 에 남는 것은 셋째뿐이다.

### resume 은 state 를 읽지 않는다

`NetResume` 이 `ws.FirstUndone()` 을 부르고(`internal/chainsetup/verb/verbs_resume.go:66`), 그
함수는 `UpStepNames` 를 돌며 `Steps[name].Done` 을 본다(`internal/chainsetup/record.go:185`).
machine 의 state 값은 어디에도 없다.

### testengine 자체의 수명주기

`engine.Run`(`internal/testengine/engine_impl.go:95`)은 정의서마다 한 바퀴 도는 for 문이고 바뀌지
않았다. 새로 생긴 것은 **분류**다. `RunSuite` 가 `runSuiteBody` 의 에러를 `runFailure` 로 테스트
영역 state 에 대응시켜 `RunSuiteOut.FailedAt` 에 넣는다(`internal/testengine/suite.go:253`~`258`,
`:142`, `internal/testengine/failurekind.go:80`). 각 반환 지점은 `lifecycle.Mark(errX, err)` 로
sentinel 을 붙인다(suite.go 에 20곳 남짓). 테스트 영역의 진입 state 여섯(`TestReadDeclaration` …
`TestCollect`)과 `TestFinished` 를 프로덕션 코드가 참조하는 곳은 없다. machine 이 걷지 않는다.

---

## 3. "state 기반" 이라 보기 어려운 이유 — 9월 22일 재확인

**① "지금 어디인가" 를 네 곳이 따로 안다.** record 의 step map(`internal/chainsetup/state.go:104`) ·
`UpStepNames`(`internal/chainsetup/state_request.go:122`) · `composition` 표(`internal/chainsetup/verb/statedriven.go:59`) · 전이표 `allowed`.
9월 21일과 같다. 바뀐 것은 `UpStepNames` 와 `OpStepNames` 가 한 파일에 선언되고 ratchet
(`TestRecordedStepsAreDeclared`)이 record 에 적히는 이름을 두 목록에 묶는다는 점이다. 네 곳이
넷이라는 사실은 그대로다.

**② state 는 휘발한다.** `Machine.At()` 을 읽는 곳은 `internal/chainsetup/verb/compare.go:81` 하나다. record ·
`session` · `collector` 어디에도 `lifecycle` 이 없다.

**③ 전이는 일으키는 것이 아니라 추인하는 것이다.** `Passed` 를 만드는 곳이 16곳으로 늘었다. 늘수록
재생하는 양이 늘 뿐, handler 가 일을 하는 것은 아니다.

**④ 선행 조건은 여전히 verb 안에 있다.**

| 장치 | 호출 지점 | 어디 |
|---|---|---|
| `require(step)` | 7 | `steps_config.go` ×2, `steps_lifecycle.go` ×2, `steps_compose.go`, `steps_genesis.go`, `verb_needs.go` |
| `allow(verb)` / `allowNode` | 6 + 2 | `internal/chainsetup/verb_needs.go:169` 가 강제 |
| 전이표 `permits` | machine 안 | `internal/core/lifecycle/machine.go:87` |
| "run `chain X` first" 손 문장 | 14 | `steps_lifecycle.go` ×3, `verb_needs.go` ×3, `steps_keys.go` ×3, `steps_compose.go`, `observe.go`, `node_ops.go`, `verb/verbs_enode.go` ×2 |

**⑤ machine 이 둘이고 시작점이 다르다.** `netUpFrom`(OpenWorkspace→Verify), `NetUpComparing`
(Compare→Verify), attach 는 없음. `StartFor` 는 enum.

**⑥ state 111개 중 22개는 프로덕션이 부르지 않는다.** 4장.

**⑦ testengine 은 machine 이 없다.** 어휘와 분류는 생겼고 walker 는 없다.

**⑧ (새로) `Status` 가 세 영역에서 세 가지 뜻이다.** 0장. 운영 영역은 `ChainOpCrossFork` 의 세부
셋을 `Passed` 로 돌려주지만(`internal/chainsetup/steps_crossfork.go:103`~`121`) 그것을 `Request`
하는 machine 이 없고, 나머지 운영 block 은 실패 값만 쓴다. 테스트 영역은 `FailedAt` 하나다.

---

## 4. 전이표의 도달 현황 (2026-09-22)

`allowed` 의 state 111개 중 89개를 프로덕션 코드(테스트 제외, `lifecycle` 자신 제외)가 참조한다.
참조하지 않는 22개:

| 묶음 | state | 왜 안 닿나 |
|---|---|---|
| VERIFY · READY 5 | `ChainVerifyProducing` `ChainVerifyHaltedAsDeclared` `ChainVerifyFailPlanMismatch` `ChainVerifyFailNotProducing` `ChainReady` | 두 machine 모두 `ChainVerify` 를 목표로 삼아 그 앞에서 멈춘다 |
| 조립 종착 2 | `ChainRemoved` `ChainStopped` | 새로 선언됐고 아무도 안 쓴다 |
| 운영 진입 6 | `ChainOpStopNodes` `ChainOpStartNodes` `ChainOpReplaceNode` `ChainOpCrossFork` `ChainOpRemoveNodes` `ChainOpHardfork` | 운영은 실패 값과 cross-fork 세부만 쓴다. 진입 state 를 걷는 machine 이 없다 |
| adopt 실패 2 | `AdoptChainFailUnreachable` `AdoptChainFailWrongChain` | adopt 는 machine 에 안 탄다 |
| 테스트 진입 7 | `TestReadDeclaration` `TestOpenSession` `TestReachNetwork` `TestPrepare` `TestRunCases` `TestCollect` `TestFinished` | 실패만 `FailedAt` 으로 분류한다 |

9월 21일에 없던 것: `FailLoop` · `FailNoHandler` · `FailRecordFormat` · `FailRecordSave` ·
`FailWorkspaceConfig` 는 이제 참조된다(대부분 `Mark` 의 kind 로). 반대로 운영 · 테스트 영역이 더해져
"선언은 됐으나 안 걷는" state 가 21 → 22 로 늘었다.

전체 전이(실패 제외 73개)는 `graph/lifecycle-transitions.mmd` 에 있다.

---

## 5. 9월 21일 이후 무엇이 바뀌었나 (`6e2a0af8` → `ea267fd8`)

| 바뀐 것 | 어디 | 판정에 미치는 영향 |
|---|---|---|
| verb 층이 `internal/chainsetup/verb/` 로 분리. `InWorkspace` · `WithWorkspace` 가 exported | `internal/chainsetup/workspace_open.go:17`, `:28` | 없음. 여섯 겹은 그대로이고 패키지 경계만 생겼다. 15번의 어댑터(`legacyStage`)가 감쌀 대상이 `verb.upSteps` 로 분명해졌다 |
| 아홉 단계 전부 `classify`. `owed` · `assumed` 가 0 | `verb/statedriven.go:115`, `:119` | ③ 은 그대로. 재생하는 `Passed` 가 늘었다 |
| `Mark` 가 `chainsetup/failurekind.go` 에서 `core/lifecycle/mark.go` 로 | `internal/core/lifecycle/mark.go:31` | sentinel → `Status` 두 겹은 그대로 |
| 운영 영역(0x3000) state 선언. 실패 분류와 cross-fork 세부만 사용 | `internal/core/lifecycle/status.go:301` 이하, `steps_crossfork.go`, `node_ops.go`, `steps_lifecycle.go` | ⑧. `Status` 가 오류 분류 어휘로도 쓰이기 시작했다 |
| 테스트 영역(0x8000) state 선언. `RunSuiteOut.FailedAt` | `internal/core/lifecycle/status.go:503` 이하, `internal/testengine/suite.go:142`, `internal/testengine/failurekind.go:80` | ⑦ 은 그대로(machine 없음). 분류는 생겼다 |
| `UpStepNames` · `OpStepNames` 를 한 파일에 선언 + ratchet | `internal/chainsetup/state_request.go:122`, `:140` | ① 의 "네 곳" 은 그대로. cohesion 후보 A 의 2안이 들어갔다 |
| `ChainRemoved` · `ChainStopped` 선언 | `internal/core/lifecycle/status.go:279` | 미참조 2 증가 |
| `ChainBuildNodeCommandFailSplitNetwork` 등 실패 추가 | `status.go` | 분류만 |

**9월 21일의 후보 R1~R10 은 접었다.** 14번 7장에서 State 패턴 기반으로 가기로 정했고, 순서는
15번이다. 이번 재검토로 15번에 더할 것은 셋이다.

1. 어댑터 `legacyStage` 가 감쌀 것은 `internal/chainsetup/verb/verbs_up.go:103` 의 `upSteps` 다.
   `netUpFrom` 이 `Manager.Send(CmdCompose)` 를 부르는 자리는 `:263`~`:267` 이다.
2. 운영 영역의 값들(`ChainOp*`)은 17번 commit 에서 `ready` 아래 leaf state 가 될 때 흡수된다. 그
   전까지 `steps_crossfork.go` 의 `Passed` 는 아무도 안 읽는다. 지우지 말고 두되, 새 leaf state 가 그
   세 세부(`BeforeFork` · `HandingOver` · `Crossed`)를 self message 로 바꾼다.
3. 테스트 영역의 `runFailure` 분류는 18번 commit 에서 `run` machine 의 `failed` reason 이 된다.
   `Mark` 로 붙인 sentinel 스물은 그대로 쓸 수 있다.

---

## 6. 정본 위치

- 대안 설계와 기제: [`14-hsm-pattern-review-2026-09-21.md`](14-hsm-pattern-review-2026-09-21.md)
- 작업 순서와 규칙: [`15-hsm-refactoring-handoff-2026-09-21.md`](15-hsm-refactoring-handoff-2026-09-21.md)
- 저장소의 자체 측정: `docs/dev/architecture/design-v3/state-machine-04-operational-failures.md`,
  `state-machine-05-test-failures.md`
