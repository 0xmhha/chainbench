# Android StateMachine 기제 정독과 chainbench 적용 — [측정 + 제안]

> 작성 2026-09-21. 이 문서는 같은 날의 초안을 **통째로 대체**한다. 초안은 참조 코드를 조각으로
> 읽고 세 번 고쳐 썼고, 그 과정에서 틀린 주장을 셋 남겼다(0장). 이번에는
> `/Users/wm-it-25_0220/Work/github/references/android/READING-ORDER.md` 가 가리키는 코드를
> 다섯 갈래로 전부 읽은 뒤 썼다. 근거는 전부 파일·줄이다.
>
> 대상 chainbench 코드는 커밋 `80e29fbe` 와 작업 트리다. **2026-09-22 에 HEAD `ea267fd8` 로 경로와 줄을
> 다시 맞췄다**(verb 층이 `internal/chainsetup/verb/` 로 갔고, 운영·테스트 영역의 `Status` 가 선언됐다. 13번 5장). 13번 문서의 판정("지금 `lifecycle` 는
> 상태를 추인하는 테이블 기반 state machine다")은 그대로 유효하고, 이 문서는 그 대안이 정확히 무엇인지를
> 적는다.
>
> 기술 용어(state, leaf state, parent state, enter/exit/processMessage, self message, deferred message,
> controller, record, transition)는 번역하지 않고 원어대로 쓴다.
>
> 아래에서 Android 파일은 처음 한 번 전체 경로로 적고 그 뒤로는 파일명만 쓴다. 전체 경로:
> `/Users/wm-it-25_0220/Work/github/references/android/modules-utils/java/com/android/internal/util/StateMachine.java`,
> `/Users/wm-it-25_0220/Work/github/references/android/modules-utils/javatests/com/android/internal/util/StateMachineTest.java`,
> `/Users/wm-it-25_0220/Work/github/references/android/NetworkStack/src/android/net/dhcp/DhcpClient.java`,
> `/Users/wm-it-25_0220/Work/github/references/android/NetworkStack/src/android/net/dhcp6/Dhcp6Client.java`,
> `/Users/wm-it-25_0220/Work/github/references/android/NetworkStack/src/android/net/ip/IpClient.java`,
> `/Users/wm-it-25_0220/Work/github/references/android/NetworkStack/src/com/android/server/connectivity/NetworkMonitor.java`,
> `/Users/wm-it-25_0220/Work/github/references/android/telephony/src/java/com/android/internal/telephony/data/DataNetwork.java`,
> `/Users/wm-it-25_0220/Work/github/references/android/Connectivity/staticlibs/device/com/android/net/module/util/SyncStateMachine.java`,
> `/Users/wm-it-25_0220/Work/github/references/android/Connectivity/Tethering/src/android/net/ip/IpServer.java`,
> `/Users/wm-it-25_0220/Work/github/references/android/Connectivity/Tethering/src/com/android/networkstack/tethering/Tethering.java`.

---

## 0. 결론과 정정

**기제는 한 문장이다.** 상태에 들어가면 `enter()` 가 그 단계의 일을 시작한다. 일의 결과는 언제나
메시지로 돌아온다. `processMessage()` 가 그 메시지를 받아 다음 상태로의 전이를 정하고, 다음
일은 다음 상태의 `enter()` 가 한다. `exit()` 는 `enter()` 가 걸어 둔 것을 거둔다. 상태는 이렇게
사슬로 이어지고, 사슬의 매듭마다 메시지가 하나씩 있다.

초안에서 틀린 것 셋을 먼저 고친다.

| 초안의 주장 | 참조 코드 | 근거 |
|---|---|---|
| `Enter` 는 준비만 하고 일은 `Process` 가 한다 | **반대다.** `enter()` 가 패킷을 보내고 타이머를 걸고 남에게 부탁한다. `processMessage()` 는 그 결과 메시지를 전이로 바꾼다 | DhcpClient 1212, Dhcp6Client 249~256, DataNetwork 1498~1520 |
| state machine는 자기에게 메시지를 보내지 않는다. self queue가 없다 | **self message가 기제의 핵심이다.** `enter()` 는 `transitionTo` 를 못 부르므로 자기에게 메시지를 남겨 전이가 끝난 뒤 처리하게 한다. 동기 변형도 self queue(`mSelfMsgQueue`)를 가진다 | IpClient 632~634, 2955; SyncStateMachine 53, 206~210 |
| child→parent는 리스너 콜백이다 | **parent state machine의 메시지 큐다.** parent가 child을 만들 때 자기 자신을 controller 로 넘기고, child은 `mController.sendMessage(...)` 로 보고한다. 동기 child(IpServer)도 콜백을 거쳐 결국 parent 큐에 넣는다 | IpClient 3046; DhcpClient 897; Tethering 3026~3029 |

chainbench 는 테스트를 하나씩 돌리므로 Looper 와 스레드는 필요 없다. 그러나 **self message
큐는 필요하다.** 동시성 때문이 아니라 "전이 도중에는 전이를 못 부른다" 는 규칙 때문이다(1.7).

---

## 1. 기제 — 참조 코드가 실제로 하는 것 열셋

### 1.1 트리는 `addState` 블록이 그림이고, parent는 데이터다

`addState(child, parent)` 로 넣는 실행 시점 데이터가 트리다. 실전 코드는 이 블록을 들여쓰기로
적어 트리를 보이게 한다(DhcpClient 538~561, IpClient 1331~1337, Dhcp6Client 171~182). Java
상속은 **다른 축**이다. `MessageExchangeState`(Dhcp6Client 230), `BaseServingState`(IpServer
1122), `ErrorState`(Tethering 2481)는 공통 행동을 물려주는 Java parent이지만 state machine에 등록되지
않는다. 트리의 parent는 "안 받은 메시지를 대신 받는 상태" 이고, Java parent는 "코드를 나눠 갖는 틀"
이다.

### 1.2 메시지 하나의 생애

StateMachine 803~838 의 `handleMessage` 가 전부다.

1. 현재 상태의 `processMessage` 를 부른다. `NOT_HANDLED` 면 parent로 올라간다. root까지 못 받으면
   `unhandledMessage`(993~1016).
2. `processMessage` 가 돌아온 **뒤에** `transitionTo` 로 적어 둔 목적지를 본다. 현재 상태와
   목적지의 가장 가까운 활성 조상을 찾아, 거기까지 `exit` 를 **깊은 쪽부터** 부르고, 거기서부터
   `enter` 를 **바깥부터** 부른다(844~908, 1022~1047).
3. deferred message를 큐 **맨 앞**에 오래된 순으로 넣는다(897, 1052~1065).
4. `enter()` 나 `exit()` 가 또 `transitionTo` 를 불렀으면 2 를 반복한다(899~905).

`enter()` 안에서 `sendMessage` 한 것은 전이가 다 끝난 뒤에 처리된다. Looper 가 현재
`handleMessage` 를 마쳐야 다음을 꺼내기 때문이다(1784~1790). 예제 출력이 이것을 보인다:
`mP2.enter` 가 보낸 `CMD_5` 는 미룬 `CMD_3` 과 먼저 쌓인 `CMD_4` 뒤에 온다(416~419, 테스트는
StateMachineTest 1998~2011).

### 1.3 `enter()` 가 일을 한다 — 세 꼴

| 꼴 | 예 | 근거 |
|---|---|---|
| **보내고 재전송 알람을 건다** | `PacketRetransmittingState.enter`: `initTimer(); sendMessage(CMD_KICK)`. 첫 송신도 self message다 | DhcpClient 1212 |
| **마감만 건다** | `TimeoutState.enter` → `mTimeoutAlarm.schedule(now + mTimeout)` | DhcpClient 1165, 1189~1193 |
| **남에게 부탁하고 reply message를 기다린다** | `WaitBeforeOtherState.enter` → `mController.sendMessage(CMD_PRE_DHCP_ACTION)`; `ConnectingState.enter` → `setupDataCall(..., obtainMessage(EVENT_SETUP_DATA_NETWORK_RESPONSE))`; `ProbingState.enter` → 스레드를 띄우고 그 스레드가 `CMD_PROBE_COMPLETE` 를 보낸다 | DhcpClient 1004; DataNetwork 1617~1621; NetworkMonitor 2106~2132 |

`enter()` 가 결과를 **돌려주지 않는다.** 결과는 언제나 메시지다.

### 1.4 결과는 메시지로 돌아온다

알람은 `WakeupMessage` 로 메시지가 된다(DhcpClient 566~572). 수신 스레드는 패킷을
`sendMessage(CMD_RECEIVED_PACKET, packet)` 으로 넣는다(706). 외부 서비스 호출은 회신
메시지를 인자로 넘긴다(DataNetwork 1621). child state machine는 parent 큐에 넣는다(DhcpClient 897).
콜백은 `sendMessage` 로 바꾼다(NetworkMonitor 1846). **어느 경로든 한 큐로 모인다.** 그래서
상태 코드는 스레드를 모른다.

### 1.5 `processMessage()` 는 메시지를 전이와 다음 일로 바꾼다

OFFER 를 받으면 `mOffer` 에 저장하고 `Requesting` 으로(DhcpClient 1360). NAK 이면 `Init` 으로
(1497~1499). 타임아웃이면 `Init` 으로(1504~1507). setup 응답이 성공이면 `Connected`, 실패면
사유를 필드에 적고 `Disconnected`(DataNetwork 1692, 1697). 전이 뒤의 "다음 일" 은 이 함수가
하지 않는다. 다음 상태의 `enter()` 가 한다.

### 1.6 `exit()` 는 `enter()` 가 건 것을 거둔다

알람 취소(DhcpClient 1238, 1183), self message 제거(`removeMessages`, 1333~1335), 소켓 닫기
(1117), watchdog timer 해제(DataNetwork 1524). parent state의 `exit()` 는 그 하위 트리 전체의 자원을
거둔다. `DhcpHaveLeaseState.exit` 가 임대 알람 셋을 취소하고 controller 에 주소 해제를 알린다
(DhcpClient 1525~1535).

### 1.7 `enter()` 안에서는 `transitionTo` 를 못 부른다 — 그래서 self message

비동기 state machine는 `Log.wtf` 를 찍고(StateMachine 1236~1243), 동기 state machine는 예외를 던진다
(SyncStateMachine 267~271). 대신 세 가지를 쓴다.

- `deferMessage(obtainMessage(CMD_JUMP_RUNNING_TO_STOPPING))`. 전이가 끝난 직후 큐 맨 앞에
  들어가 새 상태가 처리한다(IpClient 632~634, 3483).
- `sendMessage(CMD_KICK)`. 큐 뒤에 들어간다(DhcpClient 1212).
- 동기 state machine에서는 `sendSelfMessage`. 전이가 끝나고 `mCurrentState` 가 바뀐 뒤, 같은
  `processMessage` 호출 안에서 새 상태에 대해 처리된다(SyncStateMachine 249, 206~210; IpServer
  1144~1148).

즉 "enter 가 일을 하고 사건을 만든다" 의 실체는 **enter 가 self queue에 넣은 메시지**다.

### 1.8 기다리는 상태는 나머지를 붙든다, 재시도는 별도 상태다

비동기 결과를 기다리는 상태는 `default: deferMessage(msg)` 로 나머지를 전부 미룬다(IpClient
3010, 3200, 3240, 3381; NetworkMonitor 2188~2191; DhcpClient 1325). 아무것도 잃지 않고 다음
상태가 순서대로 본다. 재시도는 **따로 대기 상태**를 두고 그 `enter()` 가 배수 지연으로 자기
메시지를 건다(NetworkMonitor 2219~2226). 바쁜 상태가 미루기를 할 수 있게 하려는 분리다.

### 1.9 오래된 응답은 토큰으로 버린다

트랜잭션 id(Dhcp6Client 280), 탐침 토큰(NetworkMonitor 2139), 재평가 토큰(1522), 스레드 id
(2308). 새 단계에 옛 답이 오면 무시한다.

### 1.10 오류 — 사유는 필드에, terminal state가 한 번 보고, 분리할 때는 ErrorState

DataNetwork 는 실패 사유를 전이 **직전에 필드**(`mFailCause`, `mRetryDelayMillis`)에 적고
`Disconnected` 로 간다. terminal state의 `enter()` 가 그것을 읽어 위에 **한 번** 보고한다(1693~1698,
2028~2046). 전이 자체는 값을 나르지 않는다.

Tethering 은 오류를 **상태로 분리**한다. `ErrorState` 가 Java parent이고 오류 코드를 필드로 들고,
늦게 합류한 child에게도 같은 오류를 보내며, `CMD_CLEAR_ERROR` 로만 나간다(2481~2509). leaf state 다섯은
실패한 작업마다 하나이고 `enter()` 만 있다. 알리고, 되돌릴 것을 되돌린다(2511~2561). 트리에는
root 바로 아래 평평하게 등록된다(2000~2006).

IpClient 는 실패를 **한 깔때기**로 모은다. `transitionToStoppingState(code)` 가 reason code를
남기고 정지로 간다(1763~1766).

### 1.11 parent와 child

- parent가 child을 만들 때 **자기 자신을 controller 로** 넘긴다(IpClient 3046). child은
  `mController.sendMessage(CMD_POST_DHCP_ACTION, ...)` 로 보고한다(DhcpClient 897). parent는 자기
  상태의 `case DhcpClient.CMD_POST_DHCP_ACTION:` 으로 받는다(IpClient 3803).
- **왕복이 있다.** child이 `CMD_PRE_DHCP_ACTION` 을 올리면 parent가 프레임워크에 부탁하고, reply 를
  받아 child에게 `CMD_PRE_DHCP_ACTION_COMPLETE` 를 내려보낸다(3743, 3645, 3650). child이 못 하는
  특권 작업(주소 설정)도 같은 왕복이다(3755, 3784).
- **정지는 두 단계다.** child에게 `CMD_STOP_DHCP` 를 보내 자기 정리를 시키고 `doQuit()`, child의
  `CMD_ON_QUIT` 이 올 때까지 `StoppingState` 에서 기다린다(2957~2964, 2991~3008).
- child이 동기 state machine일 때는 **위로는 큐에 넣고(비동기) 아래로는 중첩 호출(동기)** 이다. 이
  비대칭이 재귀를 막는다(Tethering 2193~2194 아래로, 3026~3029 위로).

### 1.12 번호와 이름

- parent는 1~24, 점프 명령은 100~102, child DHCPv4 는 1000 대, DHCPv6 는 2000 대. child이 parent의
  handler 를 같이 쓰므로 겹치면 안 된다(IpClient 643~647).
- 한 state machine 안에서 `PUBLIC_BASE`(밖이 보내는 것)와 `PRIVATE_BASE = PUBLIC_BASE + 100`(자기 안에서
  도는 것)을 가른다(DhcpClient 219, 259).
- 접두는 `CMD_`(시키는 것)와 `EVENT_`(일어난 사실). child이 parent에게 올리는 보고도 `CMD_POST_*`
  같은 `CMD_` 이름을 쓴다. parent에게 "이제 이것을 하라" 는 뜻이기 때문이다.
- `MessageUtils` 가 리플렉션으로 상수 이름을 모아 로그에 이름을 찍는다(DhcpClient 279~281).

### 1.13 기록

`LogRec` 링 버퍼가 메시지마다 (시각, what, 처리한 상태, 처음 받은 상태, 목적지)를 남기고 `dump`
가 찍는다(StateMachine 452~459, 2088~2097). 이것이 "문제가 생겼을 때 디버깅이 수월하다" 의
실체다.

### 동기 변형의 정확한 규칙

SyncStateMachine 은 위 기제에서 Looper · 지연 메시지 · `deferMessage` 를 뺀 것이다.

- `processMessage()` 가 **완전한 단위**다: 처리 → 전이 하나 → self queue를 빌 때까지 소진. 돌아오면
  state machine는 정지 상태다(174~195).
- `transitionTo` 는 `processMessage` 안에서 **한 번**만. `enter` / `exit` 에서 부르면 던진다
  (267~271).
- `sendSelfMessage` 는 `enter` / `exit` / `processMessage` 안에서만. 전이가 끝난 뒤 새 상태에
  대해 처리된다(243~250, 206~210).
- 재진입은 던진다(177~180). 못 받은 메시지는 `Log.wtf` 다(227).

---

## 2. 지금 `internal/core/lifecycle` 와의 차이

| 축 | 참조 기제 | 지금 lifecycle | 근거 |
|---|---|---|---|
| 상태 | `enter/exit/processMessage` 를 가진 객체 | `uint32` 값. 행동은 `map[Status]Handler` | `internal/core/lifecycle/status.go:23` |
| 전이 | 상태가 `transitionTo`. 표 없음 | 전역 `allowed` 표가 `Request` 를 거절 | `internal/core/lifecycle/transitions.go` |
| 무엇이 움직이나 | 메시지 | 루프가 목표까지 핸들러를 부름 | `internal/core/lifecycle/machine.go:130` |
| 일은 어디서 | `enter()` | verb 가 밖에서 하고 `Passed` 로 지나온 state 를 재생 (16곳) | `internal/chainsetup/verb/statedriven.go:162`~`204` |
| 결과는 어떻게 | 메시지 | 반환값 | 위와 같음 |
| self message | 있음. 기제의 핵심 | 없음 | — |
| parent 위임 | 있음 | 없음 | — |
| 정리 | `exit()` | 없음 | — |
| parent·child | child이 parent 큐에 보고 | 두 state machine, parent 없음 | `internal/chainsetup/verb/compare.go:73` |
| 기록 | `LogRec` | 없음 | — |

한 줄로: 지금 것은 상태 **표를 검사하는 루프**이고, 참조는 **메시지로 이어지는 상태 객체
사슬**이다.

---

## 3. Go 로 옮길 때

### 상속 둘은 임베딩, 트리는 그대로

Android 가 Java 상속을 쓴 자리는 둘이다. 빈 기본 구현(`State.java`)과 공통 틀
(`MessageExchangeState`, `BaseServingState`, `ErrorState`). 둘 다 struct 임베딩으로 된다.
트리는 데이터라 잃는 것이 없다.

### Looper 는 빼고 self queue는 남긴다

테스트를 하나씩 돌리므로 스레드는 없다. SyncStateMachine 모델을 따른다. 그러나 1.7 의 규칙 때문에
self queue는 남긴다. 지연 메시지와 `deferMessage` 는 **지금은 안 만든다.** 조립 단계는 동기 호출이라
`enter()` 가 결과를 그 자리에서 알고, 호출자가 하나라 처리 중에 다른 메시지가 오지 않는다.
nodemonitor 의 사건이 조립 도중에 들어오는 날 `deferMessage` 가 필요해지고, 그때 만든다.

### 위로는 큐에 넣고, 아래로는 중첩 호출

두 state machine가 다 동기이면 child의 `Send` 안에서 parent의 `Send` 를 부를 수 없다(재진입). 참조가 parent를
비동기로 둔 이유다. 우리는 parent state machine에 **`Post(msg)`** 를 둔다. 큐에만 넣고 처리하지 않는다.
parent의 현재 `Send` 가 끝나면 self queue와 함께 소진한다. 이것이 "위로는 비동기" 의 동기 등가물이다.

### 계약 스케치

```go
package lifecycle

type What int

// Message crosses a machine's boundary or its own queue. The concrete type is
// what a state switches on; What is for logs and range checks.
type Message interface{ What() What }

type State interface {
	Name() StateName
	Enter(ctx context.Context, m *Machine) error // starts this state's work; results come back as messages
	Exit(ctx context.Context, m *Machine) error  // undoes what Enter armed
	Process(ctx context.Context, m *Machine, msg Message) (handled bool, err error)
}

type Base struct{}

func (Base) Enter(context.Context, *Machine) error                    { return nil }
func (Base) Exit(context.Context, *Machine) error                     { return nil }
func (Base) Process(context.Context, *Machine, Message) (bool, error) { return false, nil }

// Machine is the synchronous variant: Send is one complete unit of work.
type Machine struct {
	parent     map[State]State
	active     []State
	dest       State
	self       []Message // sendSelfMessage: drained after the transition, in the new state
	inbox      []Message // Post: what a child or an observer left for us
	processing bool
	log        []LogRec  // ring buffer, see 1.13
	controller *Machine  // the parent, or nil
}

func New(name string, controller *Machine) *Machine
func (m *Machine) Add(s, parent State)
func (m *Machine) Start(ctx context.Context, initial State) error
func (m *Machine) Send(ctx context.Context, msg Message) error // refuses re-entry; processes, transitions, drains self then inbox
func (m *Machine) Post(msg Message)                            // enqueue only; safe from inside a child's Send
func (m *Machine) TransitionTo(s State)                        // inside Process only, once
func (m *Machine) SendSelf(msg Message)                        // inside Enter/Exit/Process only
func (m *Machine) Current() State
func (m *Machine) Path(s State) string
func (m *Machine) Dump() []LogRec
```

`Send` 의 순서는 `SyncStateMachine.processMessage` 그대로다: 처리(parent로 올라가며) → 전이(exit
깊은 쪽부터, enter 바깥부터, 마지막에 current 갱신) → `self` 소진 → `inbox` 소진. 돌아오면
state machine는 정지 상태다. `LogRec` 은 메시지마다 (what, 처리 상태, 원 상태, 목적지)를 남긴다.

**한 곳만 SyncStateMachine 이 아니라 StateMachine 을 따른다 — 자기 자신으로의 transition**
(2026-09-22, commit 1 을 쓰면서 정함). SyncStateMachine 은 `performTransitions` 첫 줄에서
`mDestState == mCurrentState` 면 그냥 돌아간다(279). 그래서 `transitionTo(자기 자신)` 이 아무
일도 하지 않고, 부른 쪽은 그것을 알 방법이 없다. 비동기 StateMachine 은 반대로 목적지를 **언제나**
enter 한다 — "the destState must always be entered even if it is active. This can happen if we are
exiting/entering the current state"(StateMachine 1105~1112). 우리는 뒤쪽을 쓴다. 이유 둘이다.
첫째, "이 단계를 다시 한다" 가 여기서는 실제로 필요한 동작이고(재시도, resume 뒤 같은 leaf state
재실행), 그것을 말할 방법이 `TransitionTo(self)` 말고는 없다. 둘째, `Exit` 은 `Enter` 가 건 것을
거두는 자리라 둘이 짝이어야 하는데, 조용한 no-op 은 두 번째 `Enter` 가 첫 번째가 건 것 위에 다시
거는 모양이 된다. 그 밖의 순서·규칙은 전부 SyncStateMachine 그대로다.

---

## 4. chainbench 에 씌우면

### 어휘: 큰 단계는 parent state, 구체 행동은 leaf state, 메시지는 CMD 와 EVENT

참조에서 "genesis 를 템플릿에서 만든다" 와 "기존 것을 쓴다" 같은 갈래는 **leaf state**다.
`DhcpInitState` 와 `DhcpInitRebootState` 가 둘 다 상태이고(DhcpClient 540, 557), 어느 쪽으로
갈지는 앞 상태의 `processMessage` 가 요청을 보고 정한다(`StoppedState`: `CMD_START_DHCP` 를
받아 preconnection 이면 `Preconnecting`, 아니면 `InitReboot`, 1065~1072). 그래서 어휘는 이렇게
된다.

| 무엇 | 이름 | 예 |
|---|---|---|
| 큰 단계 | parent state, 진행형 | `ensuringKeys`, `buildingGenesis`, `launching` |
| 구체 행동 | leaf state, 행동 이름 | `keysFromPreset`, `keysGenerated`, `keysDeclared`, `genesisFromTemplate`, `genesisFromExisting` |
| 밖이 시키는 것 | `Cmd` 메시지 | `CmdCompose{req}`, `CmdStep{name}`, `CmdStop`, `CmdClearError` |
| 일어난 사실 | `Event` 메시지 | `EventKeysEnsured`, `EventGenesisBuilt`, `EventPhaseLaunched{n}`, `EventStageFailed{err}` |
| 위로 올리는 보고 | parent protocol 의 `Cmd` | `CmdPostCompose{ok, result}`, `CmdOnQuit` (DhcpClient 의 `CMD_POST_DHCP_ACTION` 과 같은 꼴) |

지금 `lifecycle` 의 "세부 상태" 열아홉(`FromPreset`, `ForkApplied` …)이 바로 leaf state다. 13번
문서가 "추인" 이라 부른 `Passed` 재생은, leaf state에 실제로 **들어가는** 것으로 바뀐다.

### 트리

```
composition                  root. 공통: CmdStop, 모르는 메시지 → unhandled 로 기록
  stopped                    초기. CmdCompose 를 받아 요청을 저장하고 첫 leaf state으로
  composing                  공통: EventStageFailed → failure 필드 → failed. CmdStep 이 지금 단계가 아니면 거절
    openingWorkspace
    buildingNodeTable
    ensuringKeys
      keysFromPreset · keysGenerated · keysDeclared
    reconciling              reuse-if-matching 일 때만
    buildingGenesis
      genesisFromTemplate · genesisFromExisting
    buildingNodeConfig
    buildingNodeCommand
    deployingInputs
      inputsVerifiedLocal · inputsShippedRemote
    initializingDatadirs
    launching                공통: 포트·바이너리 검사. EventPhaseLaunched 를 세어 다음 phase 또는 verifying
      launchingPhase
      runningPhaseActions
  composed                   --stage=deploy 처럼 중간에 멈춘 자리. record 에 남는다. CmdStep 은 여기서만 받는다
  verifying                  Enter 가 준비 게이트를 돌린다 (게이트 함수는 testengine 이 주입)
  ready                      운영 Cmd 를 받는 유일한 자리. EventNodeDied 도 여기만
    stopping · swapping · hardforking · restarting · crossingFork
  failed                     ErrorState 꼴. reason 필드, CmdClearError 로만 나감
    failedPortBusy · failedKeyNotLocal · …   되돌릴 것이 다른 실패만 leaf state으로
```

### 단계 하나

`buildingGenesis` parent와 leaf state 둘이다. leaf state의 `Enter` 가 일을 하고 결과를 **self message**로 남긴다.
parent의 `Process` 가 그 메시지를 전이로 바꾼다.

```go
// genesisFromTemplate is one way of building the genesis. Enter does the work
// and leaves the result as a self message; the parent turns it into a move.
type genesisFromTemplate struct {
	lifecycle.Base
	ws *Workspace
}

func (genesisFromTemplate) Name() lifecycle.StateName { return NameGenesisFromTemplate }

func (s genesisFromTemplate) Enter(ctx context.Context, m *lifecycle.Machine) error {
	gen, err := s.ws.buildGenesis(ctx, s.ws.request.Genesis) // 지금 ws.Genesis 의 본체
	if err != nil {
		m.SendSelf(EventStageFailed{Stage: NameBuildingGenesis, Err: err})
		return nil
	}
	s.ws.record.Genesis = gen
	m.SendSelf(EventGenesisBuilt{Bytes: len(gen)})
	return nil
}

func (s genesisFromTemplate) Exit(ctx context.Context, _ *lifecycle.Machine) error {
	return s.ws.save(ctx) // 이 leaf state을 떠날 때 한 번
}

// buildingGenesis is the stage. It maps the leaf's result to the next stage.
type buildingGenesis struct {
	lifecycle.Base
	ws   *Workspace
	next func() lifecycle.State
}

func (buildingGenesis) Name() lifecycle.StateName { return NameBuildingGenesis }

func (s buildingGenesis) Process(ctx context.Context, m *lifecycle.Machine, msg lifecycle.Message) (bool, error) {
	switch msg.(type) {
	case EventGenesisBuilt:
		if s.ws.request.Stage == StageGenesis {
			m.TransitionTo(s.ws.composed) // 여기까지만 하라고 했다
			return true, nil
		}
		m.TransitionTo(s.next())
		return true, nil
	}
	return false, nil // EventStageFailed 는 composing 이 받는다
}
```

leaf state을 고르는 것은 **앞 단계**다. `ensuringKeys` 의 parent `Process` 가 `EventKeysEnsured` 를 받으면
요청을 보고 `genesisFromTemplate` 인지 `genesisFromExisting` 인지 골라 `TransitionTo` 한다.
`StoppedState` 가 `CMD_START_DHCP` 를 받아 갈래를 고르는 것과 같다.

**고를 때 보는 것은 request 만이 아니다** (2026-09-22, commit 7 을 쓰면서 정함). `ws.Keys` 는 요청의
source 문자열보다 **앞 단계가 만든 노드 표**를 먼저 본다 — 키를 선언한 표는 이미 자기 신원의 출처를
말한 것이기 때문이다(`steps_keys.go` 의 주석, 그리고 13번이 "요청에서 읽던 것이 틀렸던 사례" 로 든
자리: 인라인 topology 로 조립한 것이 preset 을 썼다고 기록됐다). 그러니 문장을 한 번 넓힌다 —
**고르는 자리는 request 와 workspace 를 본다.**

그리고 고르는 일 자체에 준비가 필요할 때가 있다. 키 단계는 서버에 있는 ring 을 먼저 내려받아야
표를 읽을 수 있다. 그런 단계는 **parent 의 `Enter` 가 준비하고 고른 뒤 self message 로 말하고,
parent 의 `Process` 가 그것을 leaf 로 바꾼다.** 기제 그대로다 — Enter 가 일하고, 결과는 message 이고,
Process 가 transition 이다. 고르는 데 일이 필요 없는 단계는 앞 단계의 `Process` 에서 바로 골라도 된다.

### 시나리오

- **`chain up`.** `mg.Send(ctx, CmdCompose{req})` 하나. `stopped.Process` 가 요청을 저장하고
  `TransitionTo(openingWorkspace)`. 그 `Enter` 가 일을 하고 `SendSelf(EventWorkspaceOpened)`.
  전이가 끝난 뒤 self queue를 소진하며 `composing` 이 그 사건을 받아 다음 leaf state으로. 이 사슬이 `Send`
  하나 안에서 `ready` 까지 간다. `--stage=deploy` 는 `deployingInputs` 의 parent가
  `EventInputsDeployed` 를 받았을 때 `composed` 로 가는 것이다.
- **`chain genesis --existing X` 단독.** `mg.Send(ctx, CmdStep{Genesis, Existing: X})`.
  `composed.Process` 가 record 의 마지막 단계를 보고 다음이 genesis 면 `genesisFromExisting` 으로,
  아니면 "지금은 `X` 까지 됐다" 로 거절. 다른 상태에서는 `composing` 이 "조립 중이다" 로 거절.
  `require()` · `composeNeeds` · 손 문장이 이 두 분기가 된다.
- **실패.** leaf state이 `SendSelf(EventStageFailed{err})`. `composing.Process` 가 받아 `ws.failure` 에
  적고 `TransitionTo(failed 또는 그 leaf state)`. `failed.Enter` 가 사유를 기록하고 위에
  `CmdPostCompose{ok:false}` 를 **한 번** 올린다(`DisconnectedState.enter` 꼴). 이후 Cmd 는
  `CmdClearError` 만 받는다.
- **resume.** `Open` 이 record 의 경로로 `Start(state)`. leaf state에서 죽었으면 그 leaf state의 `Enter` 가 다시
  돌아 일을 다시 하고 사슬이 이어진다. `firstUndone` · `upStepNames` 가 필요 없다.
- **외부 사건.** nodemonitor 가 `mg.Post(EventNodeDied{3})`. `ready.Process` 만 받아
  `restarting` 으로. 조립 중이면 `composing` 이 안 받고 root가 unhandled 로 기록한다(지금은
  `deferMessage` 가 없으므로 버려진다. 필요해지는 날 1.8 대로 만든다).

### testengine 은 parent state machine, chainsetup 은 child

```
run                          root. 공통: CmdAbort
  pending
  composing                  Enter 가 child을 만들고(controller = 자기) CmdCompose 를 보낸다. child의 CmdPostCompose 를 받는다
  gating                     Enter 가 nodemonitor 를 돌린다. EventGateOK / EventGateFailed
  testing                    공통: EventSpecDone → 다음 정의서 또는 reporting
    applicable · buildingEnv · preSpecGate · executing · collecting
  reporting
  stopping                   child에게 CmdStop, CmdOnQuit 을 기다린다 (IpClient.StoppingState 꼴)
  done · blocked
```

child이 올리는 것은 parent protocol 의 `Cmd` 다. `CmdPostCompose{ok, endpoints|err}`,
`CmdOnQuit`. child은 `m.controller.Post(...)` 로 넣고, parent는 자기 `Send` 가 끝난 뒤 `inbox` 를
소진하며 `composing.Process` 의 `case CmdPostCompose:` 로 받는다. `engine.Run` 의 for 문
(`internal/testengine/engine_impl.go:95`)과 `Deps` 클로저 여섯은 `testing` 아래 leaf state 다섯의
`Enter` 가 된다. `app.StartFor` 는 첫 메시지(`CmdCompose` 인가 `CmdAttach` 인가)를 고른다.

---

## 5. 이름과 값의 규칙

전제는 하나다. 문자열과 숫자는 typed const 로 두고 리터럴은 선언 한 곳에만 있다. 문자열 const 는
줄마다 타입을 다시 써야 typed 가 된다.

### 5.1 protocol 파일 두 층

`internal/core/lifecycle/protocol.go` 는 **BASE 배정만** 한다. 도메인을 모르지만 번호는 안다.
IpClient 644~647 이 child의 BASE 를 정하는 것과 같은 자리다.

```go
const (
	BaseChain What = 0x1000
	BaseTest  What = 0x8000
)
```

`internal/chainsetup/protocol.go` 는 이 state machine가 **받는 Cmd**, **자기 안에서 도는 Event**,
**위로 올리는 Cmd** 를 한 파일에 적는다. 인자 struct 도 여기 둔다.

```go
package chainsetup

// ---- Public: what the controller (CLI/MCP via Manager, or testengine) sends down.
const (
	CmdCompose lifecycle.What = lifecycle.BaseChain + 1 + iota
	CmdStep
	CmdStop
	CmdClearError
)

// ---- Public: what this machine sends UP to its controller.
const (
	CmdPostCompose lifecycle.What = lifecycle.BaseChain + 0x40 + 1 + iota // ok/err + result
	CmdOnQuit
)

// ---- Private: what this machine says to itself. Not exported.
const (
	eventWorkspaceOpened lifecycle.What = lifecycle.BaseChain + 0x100 + 1 + iota
	eventNodeTableBuilt
	eventKeysEnsured
	eventGenesisBuilt
	eventInputsDeployed
	eventPhaseLaunched
	eventStageFailed
)

// ---- Public event: what an observer below (nodemonitor) posts to us.
const (
	EventNodeDied lifecycle.What = lifecycle.BaseChain + 0x180 + 1 + iota
)

type Compose struct{ Request Request }

func (Compose) What() lifecycle.What { return CmdCompose }

type PostCompose struct {
	OK        bool
	Endpoints []string
	Err       error
}

func (PostCompose) What() lifecycle.What { return CmdPostCompose }
```

규칙.

1. **state machine 하나에 BASE 하나.** chain 0x1000, test 0x8000.
2. **PUBLIC 과 PRIVATE 을 0x100 으로 가른다**(DhcpClient 259). 위로 올리는 Cmd 는 PUBLIC 안에서
   `+0x40` 부터, 아래서 받는 Event 는 PRIVATE 안에서 `+0x80` 부터. 값만 보면 방향이 보인다.
3. **접두는 `Cmd` 와 `Event` 둘.** Android 와 같다. `Cmd` 는 시키는 것(위→아래, 그리고 child이
   parent에게 "이제 이것을 하라"), `Event` 는 일어난 사실(자기 안, 또는 아래→위 관찰). 오류는
   메시지 종류가 아니라 **`Event…Failed` 의 `Err` 필드**와 **failed 상태**다(1.10).
   초안의 `Action` · `Error` 접두는 쓰지 않는다. `Action` 은 leaf state가 하는 일의 이름이지 메시지
   이름이 아니다(4장 어휘 표).
4. **exported 여부가 PUBLIC 여부.** Go 의 대소문자가 `PUBLIC_BASE` / `PRIVATE_BASE` 를 대신한다.
5. **이름 표 한 곳.** `whatNames map[What]string` 을 protocol 파일에 두고 `String()` 이 쓴다.
   `MessageUtils` 의 리플렉션 대신 테스트가 빠짐을 잡는다.

**구현하며 바뀐 것 둘 (2026-09-22, commit 2).**

- `CmdStep` 의 본체 이름은 `Step` 이 아니라 **`RunStep`** 이다. `chainsetup` 에 이미 `Step`
  (기록된 조립의 한 단계, `session.Step` 의 별칭)이 있고, 한 패키지에 같은 이름 둘은 Go 가 주지
  않는다. 나머지 본체 이름은 이 문서 그대로다.
- `String()` 은 `What` 에 붙일 수 없다. `What` 은 state machine 패키지의 타입이고 이름은 도메인의
  것이라, 도메인이 자기 이름을 그 패키지에 등록하는 전역 가변 상태를 만들지 않으려면 방향이 반대여야
  한다. 대신 도메인 패키지가 `WhatName(w) string` 을 내놓는다. 표와 "빠진 이름" 테스트는 그대로다.

### 5.2 상태 이름

| 자리 | 형태 | 예 |
|---|---|---|
| root | 명사 | `composition`, `run` |
| parent(큰 단계) | 진행형 | `composing`, `ensuringKeys`, `launching` |
| leaf state(구체 행동) | 행동을 말하는 이름 | `keysFromPreset`, `genesisFromExisting`, `launchingPhase` |
| terminal · idle | 형용사 | `stopped`, `composed`, `ready`, `failed` |

leaf state에 parent를 접두로 붙이지 않는다. 기록에는 state machine가 계산한 경로
`Composition/Composing/BuildingGenesis/GenesisFromTemplate` 를 쓴다(`Machine.Path`). 이름 const 는
도메인 패키지에 둔다. 비교는 `s == mg.ready` 처럼 값으로 한다.

### 5.3 실패

`failed` 는 `ErrorState` 꼴의 parent다. reason 필드, `CmdClearError` 로만 나감, 다른 Cmd 는 "실패한
상태다" 로 거절. leaf state은 **되돌릴 것이 다른 실패**만 둔다(Tethering 의 leaf state 다섯이 각각 다른 netd
되돌리기를 하듯). 그 밖의 실패는 parent의 reason 필드다. 사유는 전이 **직전 필드**에 적고 `failed.Enter`
가 한 번 보고한다.

### 5.4 붙드는 테스트

1. `What` 마다 이름이 있고, BASE 가 안 겹치고, exported 여부와 PUBLIC/PRIVATE 띠가 맞는다.
2. 트리를 `addState` 블록처럼 들여쓰기 문자열로 뽑아 golden 과 비교한다.
3. record 의 경로가 트리에 있는 상태다.
4. StateMachineTest 가 하는 것: enter/exit 순서, parent 위임, self message가 전이 뒤에 새 상태에서
   처리됨, `Enter` 안의 `TransitionTo` 거절, 재진입 거절.

---

## 6. 지금 코드에서 거기까지

### 남기는 것

단계 본체(`ws.Keys` · `ws.Genesis` · `startPhase` · `runPhaseActions`)는 leaf state의 `Enter` 본체가
된다. record 형식은 상태 경로 하나를 더한다(`FormatVersion` 2). `Workspace` 는 state machine의 소유자가
된다. sentinel 에러(작업 트리의 `errLaunch*` 등)는 실패 사유로 남는다.

### 버리는 것

`lifecycle` 의 `Status` 상수 · `names` · `allowed` · 표를 붙드는 테스트. `statedriven.go`
(작업 트리에서는 `internal/chainsetup/verb/statedriven.go`)의 `composition` 표 · `handlersFor` ·
`run` · `failed` · `startFor` · `targetFor`. `composeNeeds` · `require` · `verbNeeds` · `allow`.
`verb/compare.go` 의 `compareHandlers` · `chainsetup.ReconcileHandler`. verb 의 `InWorkspace` / `WithWorkspace` 감싸기.
운영 영역의 값(`ChainOp*`)과 테스트 영역의 값(`Test*`)은 각각 17 · 18번 commit 에서 leaf state 와 `failed` reason 으로 흡수된다(13번 5장). `app.Start.At` 의 `Status`.

### 순서 — 옛 state machine와 새 state machine를 같이 두고 leaf state을 하나씩

| 커밋 | 무엇 | 지키는 테스트 |
|---|---|---|
| 1 | 새 `lifecycle` 를 3장 계약으로 **별도 패키지**에 쓴다. 5.4 의 4번 테스트 | 새 패키지만 |
| 2 | `protocol.go` 두 층과 5.4 의 1번 테스트 | 1 과 같이 |
| 3 | record 에 `State string` 을 추가만 한다 | 기존 기록 테스트 |
| 4 | `Manager` 와 트리. leaf state은 전부 어댑터 `legacyStage{step}` 로, `Enter` 가 옛 verb 를 부르고 `SendSelf(eventStageDone)`. `netUpFrom` 이 `Manager.Send(CmdCompose)` 를 부른다 | `chain up` 통합 전부. 동작이 같아야 한다 |
| 5~13 | leaf state 하나씩 어댑터를 진짜 leaf state으로. 한 커밋에 하나 | 그 단계 + 통합 |
| 14 | `composed` 와 `CmdStep`. `require` · `composeNeeds` · 손 문장 삭제 | 단독 명령 |
| 15 | resume 이 record 의 경로로 `Start`. `upStepNames` 잔재 삭제 | resume |
| 16 | `reconciling` · `verifying` · `comparing` 을 트리에. `NetUpComparing` 삭제 | `run` 통합 |
| 17 | `ready` 아래 운영 leaf state. `verbNeeds` 삭제 | 운영 명령 |
| 18 | testengine `run` state machine. chainsetup 을 child으로, `Post` 로 보고 | 전체 |
| 19 | 어댑터와 옛 `lifecycle` 삭제 | 전체 |

**단점.** 4번 커밋에서 두 state machine가 잠깐 겹친다. 5~13 사이에는 옛 세부 상태가 record 에 안 남는다(어댑터
leaf state은 하나이므로). 그 구간을 짧게 가져간다. 어댑터는 50줄 안쪽이고 19번에서 버린다.

---

## 7. 정한 것과 남은 것

**정했다 (2026-09-21).**

1. State 패턴 기반 hierarchical state machine를 도입한다. 13번의 R1~R10 은 접는다.
2. 옛 state machine와 새 state machine를 같이 두고 leaf state을 하나씩 옮긴 뒤 옛 것을 지운다(6장).
3. 큰 단계는 parent state, 구체 행동은 leaf state. leaf state을 고르는 것은 앞 단계의 `Process` 다(4장).
4. 메시지는 `Cmd`(시키는 것, 위→아래와 child→parent의 보고)와 `Event`(사실). 오류는 메시지 종류가
   아니라 `Err` 필드와 `failed` 상태(5장).
5. self message queue는 남긴다. Looper · 지연 · defer 는 필요해지는 날 만든다(3장).
6. child→parent는 parent의 `Post`(큐에만), parent→child은 `Send`(중첩 호출)(3장).

7. 정의서를 읽어 `Request` 로 바꾸는 것은 testengine 이 한다. `Request` 를 받아 첫 leaf state을 고르고, 단계마다
   다음 leaf state을 고르는 것은 chainsetup 의 각 parent state가 한다(4장). 지금 `compositionOf` 가 하던 일은
   앞쪽, `planUp` 이 하던 일은 뒤쪽으로 간다.
8. 메시지 접두는 `Cmd` 와 `Event` 둘이다. 초안의 `Action` · `Error` 는 쓰지 않는다(5.1 의 3번).

9. **`failed` 는 처음에는 parent state 하나와 reason 필드뿐이다.** 실패마다 leaf state 를 두는 것은
   리팩토링을 끝내고 라이브 테스트를 돌리면서 "실패 뒤에 해야 하는 일이 다르다" 고 확인된 것에만
   그때 추가한다. 5.3 의 기준은 그때 쓴다.
10. **record 에 state path 를 적고 `FormatVersion` 을 2 로 올린다.** 하위 호환은 지원하지 않는다.
   새 build 는 version 1 record 를 이름을 대며 거절한다. 저장소의 기존 정책 그대로다.

**남았다.** 없다. 시작할 수 있다.
