# 운영 영역의 실패를 잰다 — `areaChainOp` — [측정]

> 잰 날 2026-09-21, HEAD `4c1e05c4`. 도구는 조립 때와 같다 — 운영 동사에서 출발해
> 자기 수신자의 메서드만 따라가며 `fmt.Errorf` · `errors.New` 를 모았다. 깊이 4.
>
> 왜 재고 나서 표를 쓰는가: 전이표가 스스로 그렇게 정해뒀다.
>
> > 운영 영역은 아직 항목이 없다. 실패를 조립만큼 재지 않았고, **실패를 모르는 상태
> > 사이에 이동을 적는 것이 첫 표가 실패를 둘 데 없게 만든 방식이다.**

---

## 0. 결론부터

**운영 영역은 조립과 모양이 다르다. 여섯 블록을 나란히 두면 안 된다.**

조립은 아홉 단계가 각자 실패를 가졌다 — 한 단계에만 속하는 것이 118건, 공유가 16건.
운영은 반대다. 고유 지점 65건 중 **25건이 블록 사이에 공유**되고, 블록 여섯 중 넷은
자기 실패가 **한두 개**뿐이다.

공유하는 것은 동사가 아니라 **동작 세 가지**다. 바이너리를 정하고, 노드 config 를 다시
쓰고, 노드를 내렸다 올린다. 운영 동사는 이 셋을 조합한 것이고, 그래서 실패도 셋에서 온다.

그러니 상태를 **동사마다** 두는 것이 아니라 **공유하는 동작**에 두고, 진짜 자기 일이 있는
두 곳(`swap` · `cross-fork`)에만 고유 상태를 둔다.

---

## 1. 수치

| | 건수 |
|---|---|
| 고유 실패 지점 | **65** |
| ├ 한 블록에만 속하는 것 | 40 |
| └ 블록 사이에 공유되는 것 | **25** |

블록별로 닿는 수(중복 포함 110):

| 블록 | 닿는 실패 | 그중 자기 것 |
|---|---|---|
| `stop` | 4 | 2 |
| `rm` | 9 | 3 |
| `restart` | 18 | 3 |
| `swap` | 35 | **11** |
| `cross-fork` | 37 | **20** |
| `hardfork` | 7 | 1 |

`stop` · `rm` · `restart` · `hardfork` 넷을 합쳐도 자기 실패가 **아홉 개**다.

---

## 2. 공유하는 25건은 네 갈래다

### 2.1 선행 조건 — 6건, 네 블록

```
verb_needs.go:160  이 동사는 요구사항을 선언하지 않았다 — verbNeeds 에 넣어라
verb_needs.go:177  노드 표가 비었다 — `chain place` 먼저
verb_needs.go:182  node%d 가 실행 중이다 — `chain stop` 먼저
verb_needs.go:218  node%d 에 기록된 argv 가 없다 — `chain start` 먼저
verb_needs.go:222  node%d 가 이미 실행 중이다
steps_compose.go:466  %s: %s 가 안 돌았다 — `chain %s` 먼저   (require)
```

조립에서는 이런 것을 **전이표가 통째로 없앤다**고 판정했다. 앞 단계를 안 지나면 다음
단계로 못 가기 때문이다. **운영은 그렇게 못 없앤다.** 순서가 없기 때문이다 — `stop`
다음에 `rm` 이 올 수도 `restart` 가 올 수도 있고, 어느 쪽도 틀리지 않다.

그래서 이것들은 **상태로 남겨야 한다.** 다만 여섯 벌이 아니라 한 벌이다.

여덟 동사 중 실제로 검사하는 것은 넷뿐이다(`Rm` · `StartNode` · `SwapNode` ·
`Hardfork`). `Stop` · `Restart` · `StopNode` · `CrossFork` 는 아무것도 안 묻는다.
`Stop` 이 안 묻는 것은 `verbNeeds` 에 적힌 이유가 있다 — "이미 멈춘 것을 멈추는 것은
부르는 쪽이 원한 결과다". **`CrossFork` 가 안 묻는 것은 적힌 이유가 없다.**

### 2.2 어느 바이너리로 올릴 것인가 — 6건, 세 블록 (`restart` · `swap` · `cross-fork`)

```
binary.go:54   바이너리가 필요한데 체인도 모른다
binary.go:58   체인 %q 가 바이너리를 안 이름한다 — 하나 줘야 한다
binary.go:88   바이너리가 필요하다 (--binary, 또는 `chain new` 에서)
binary.go:99   %q 는 상대 경로다
binary.go:105  binary %q: %w
steps_keys.go:50  체인이 안 정해졌다 — `chain new` 먼저
```

노드를 다시 올리는 세 동사가 전부 같은 질문을 한다.

### 2.3 노드 config 를 다시 쓴다 — 10건, 두 블록 (`swap` · `cross-fork`)

```
steps_config.go:119·125·129·143·158·164·168   고정 config 읽기 · 피어 · 렌더 · 읽기 확인
provenance.go:28                              config override 가 key=value 가 아니다
steps_compose.go:227                          node%d: %w
workspace_refs.go:146                         %s 참조 %q 읽기
```

이미 **조립의 `ChainBuildNodeConfig` 가 가진 실패 셋과 같은 자리**다
(`FailBadOverride` · `FailReadback` · `FailPinUnreadable`). 두 번 이름 붙이지 않는다.

### 2.4 노드를 집고 내린다 — 3건

```
node_ops.go:34    표에 node %d 가 없다        (restart · stop · swap)
node_ops.go:54    stop node%d: %w             (restart · stop)
workspace_refs.go:27  workspace-config: %w      (restart · swap · cross-fork)
```

---

## 3. 자기 일이 있는 것은 둘이다

### 3.1 `swap` — 11건

한 노드만 바꿔 끼운다. 절반은 스왑 자체(`stop` → `launch`)이고, 절반은 **genesis 오버레이를
얹어 그 노드만 다시 init** 하는 일이다.

```
node_ops.go:139  바이너리·config 변경·genesis 오버레이 중 하나는 있어야 한다
node_ops.go:161·169·177·182·185   stop · 렌더 · 기록 · launch
node_ops.go:200  genesis 가 없다 — `chain genesis` 먼저
node_ops.go:204  대상 드라이버가 datadir 을 초기화 못 한다
node_ops.go:208·212·215  genesis 읽기 · 오버레이 병합 · 오버레이로 init
```

뒤의 넷은 **조립 `ChainInitNodes` 의 실패와 같은 종류**다
(`FailTargetUnable` · `FailGenesisUnreadable` · `FailDatadir`).

### 3.2 `cross-fork` — 20건

가장 많고, 유일하게 **시간과 체인 상태를 기다린다.** 스무 건이 전부
`steps_crossfork.go` 안에 있고 네 묶음이다.

| 묶음 | 줄 | 무엇 |
|---|---|---|
| 건널 포크가 없다 | 68 · 73 · 75 | 선언이 없거나, 넘길 노드가 없거나, 받을 바이너리를 쓰는 노드가 없다 |
| 아직 포크 전이 아니다 | 165 · 169 · 173 · 177 | 실행 중인 노드가 없거나, 헤드를 못 읽거나, 이미 포크를 지났다 |
| 넘기는 중 | 193 · 197 · 217 · 219 · 262 · 276 · 301 · 307 · 313 · 320 | 읽을 노드가 없다 · 헤드 대기 초과 · stop · 렌더 · 피어 · launch |
| 넘긴 뒤 | 345 · 349 · 358 | 아무도 안 돌아왔다 · 체인이 그새 지나갔다 |

**네 묶음이 그대로 세부 상태 후보다.** 조립의 `ChainLaunchNodes` 가 단계(phase)마다 상태를
가진 것과 같은 이유다 — 어디서 멈췄는지가 곧 무엇을 해야 하는지다.

---

## 4. 이 측정이 상태 설계에 말하는 것

**하나. 블록을 여섯 두지 않는다.** `stop` · `rm` · `restart` · `hardfork` 는 자기 실패가
아홉 개뿐이고 나머지는 전부 공유분이다. 여섯 벌을 만들면 같은 실패에 여섯 이름이 붙는다 —
init 이 알아챈 포트 충돌을 `ChainLaunchNodesFailPortBusy` 하나로 둔 것과 같은 판단이다.

**둘. 공유하는 동작에 상태를 준다.** 선행 조건, 바이너리 정하기, config 다시 쓰기, 노드
집고 내리기. 앞의 둘은 새 상태가 필요하고, 뒤의 둘은 **조립이 이미 가진 상태를 쓴다**
(`ChainBuildNodeConfigFail*` · `ChainInitNodesFail*`).

**셋. `cross-fork` 만 세부 상태를 가진다.** 스무 건이 네 묶음이고, 묶음이 곧 진행 단계다.

**단점.** 동사와 블록이 1:1 이 아니게 된다. `chain rm` 이 실패했을 때 그 상태가
`rm` 블록이 아니라 공유 블록일 수 있고, 읽는 사람이 "어느 동사였나" 를 기록에서 따로
읽어야 한다. 대신 같은 실패가 여섯 이름을 갖는 일은 없다.

---

## 5. 아직 안 잰 것

- **접근자**(`Logs` · `Health` · `NetworkStatus` 등)의 실패. 운영 동사가 아니라서 뺐다.
- `hardfork` 는 자기 실패가 하나(`node_ops.go:329`)뿐인데, 실제 일은
  `internal/core/hardfork` 가 한다. 그 안의 실패는 이 측정 범위 밖이다.
- 깊이 4 를 넘는 호출. 늘려보면 수가 늘겠지만, 늘어나는 것은 대부분 `core/` 패키지의
  거절이라 이 영역의 상태가 될 것이 아니다.
