# F1 — 파일 영속·복구 (설계안, 2026-08-28)

> 실행 순서표의 마지막 항목이다. "chainbench 프로세스가 중간에 죽었을 때, 다시 실행하면
> 이전 진행을 이어받고 서버 상태를 다시 확인한다." §4 의 네 가지는 제안대로 결정됐고
> (2026-08-28), §6 이 구현 결과다.
> 상위 계획: [[module-plan]] §2a(Inventory), §4 P4.x(preflight), §7-6(워크스페이스 정의).

## 0. 원칙 — 사본을 만들지 않는다

P1.5 에서 정한 규칙이 그대로 간다: **같은 사실은 한 곳에만 있다.** 복구를 위해 "복구용
파일"을 새로 두면, 그 파일과 본래의 기록이 어긋나는 순간이 반드시 온다(module-plan §8.4
의 `Placement` 가 그 병이었다). 그래서 F1 은 새 기록을 최소로 더하고, 있는 기록을
**읽어서 다시 세우는** 쪽으로 간다. 더하는 기록은 "그 사실이 지금 어디에도 없다" 를
증명한 것뿐이다.

## 1. 지금 디스크에 있는 것 — 복구의 재료

한 워크스페이스(`<workspace-dir>/`)에 이미 이만큼이 남는다.

| 파일 | 소유자 | 무엇이 남나 | 복구에서의 쓸모 |
|---|---|---|---|
| `workspace.json` | `session.Composition` | 체인·바이너리·키셋·타깃·노드 레코드(경로·포트·argv·pid)·단계별 완료 표시(`steps`) | **진행 상황의 정본.** 어느 단계까지 끝났는지, 노드가 무엇으로 떴는지 |
| `process.json` | `process.Ledger` | 라벨별 pid·호스트·바이너리·명령줄 | 어느 프로세스를 우리가 띄웠는지 |
| `workspace.lock` | `session` | 잡은 프로세스의 pid·호스트·명령 | 죽은 실행의 흔적(`LockStale`) — 이미 인수 처리한다 |
| `runs/<stamp>/` | `chainsetup.recordRun` | 기동 한 번의 manifest·genesis·노드별 명령 | 사후 진단. 복구가 읽지는 않는다 |
| `sessions/` | `session` | 테스트 세션 아티팩트 | 테스트 결과. 복구 대상이 아니다 |

워크스페이스 밖에는 **자원 인벤토리**가 있다. 파일이 아니라 메모리다: 프로세스가 뜰 때
`~/.chainbench` 아래 워크스페이스를 `Discover` 로 찾아 `Inventory.Adopt` 로 다시 센다
(P1.5). 이것도 이미 "읽어서 다시 세우는" 방식이라 F1 의 원칙과 같다.

살아 있는지 보는 눈도 이미 있다. `inspector` 는 요청받으면 ip·포트·경로 사실을 답하고
(M7), `preflight.Check` 는 그것으로 노드별 생사를 판정한다(P4.x).

## 2. 무엇이 없나 — 세 가지 구멍

1. **요청이 남지 않는다.** `workspace.json` 은 결과(노드 4개, 포트 이것)를 적지,
   요청(`--validators 4 --set bohoBlock=10 --overlay x.json --launch-opt ...`)을 적지
   않는다. genesis 단계가 끝났으면 결과가 요청을 대신하지만, **genesis 전에 죽으면**
   다시 실행할 때 요청을 다시 받아야 한다. 이어받기가 아니라 다시 시작이다.
2. **죽은 pid 와 주인 없는 프로세스를 정리하는 순서가 없다.** 단계 함수들은 각자
   `checkUnmanaged`·`checkVacant` 로 방어하지만, "이전 실행이 남긴 것을 한 번에
   재확인하고 기록을 현실에 맞춘다" 는 동작은 어디에도 없다. 지금 `net start` 는
   자기 워크스페이스 밖에서 같은 바이너리가 돌면 **거부**한다 — 그것이 직전 실행이
   띄우고 기록 못 한 우리 노드여도.
3. **동시 실행이 자원을 겹쳐 잡을 수 있다.** 인벤토리는 프로세스마다 따로 세운다.
   두 chainbench 가 같은 서버 세트에 동시에 `net allocate` 하면 둘 다 같은 빈 슬롯을
   "빈 것" 으로 보고 가져간다. 파일로 저장한다고 풀리는 문제가 아니다 — 저장 사이의
   경합이 남는다.

## 3. 설계 — 세 구멍을 각각 가장 작게 막는다

### 3.1 요청을 기록한다 (구멍 1)

`workspace.json` 에 `request` 를 더한다. `net up` 이 받은 `NetUpIn` 에서 **비밀이 아닌
것**(체인·바이너리·키셋 경로·노드 수·동기화 모드·피어링·서버 참조·genesis set/overlay
경로·launch set)을 `new` 단계에서 적는다. 이것은 사본이 아니다 — 요청은 지금 어디에도
남지 않는 새 사실이다. `run --workspace-dir` 가 선언(env)에서 만든 요청도 같은 자리에
남으므로, 복구는 케이스 파일을 다시 열지 않아도 된다.

`preflight.WantOf` 가 이미 `NetUpIn` 을 목표로 바꾸므로, 기록된 요청은 곧 "이 워크스
페이스가 되려던 것" 이다. 이어받기와 preflight 가 같은 값을 본다.

### 3.2 되짚기 — `net resume` (구멍 2)

새 verb 하나: `chainbench net resume --workspace-dir DIR`. 하는 일은 순서대로 넷이다.

1. **잠금 인수.** 지금처럼 `LockStale` 이면 넘겨받고, `LockLive` 면 거부한다(살아 있는
   실행을 되짚지 않는다).
2. **기록을 현실에 맞춘다.** 노드마다 `preflight` 의 liveness(pid 는 그 노드의 머신에서,
   RPC head 는 그 노드 주소에서)로 판정한다. 죽은 pid 는 지운다(ledger·record 둘 다).
   레코드에 pid 가 없는데 그 노드의 데이터 디렉터리·포트에 **우리 명령줄로 뜬**
   프로세스가 있으면 — `inspector.Ports` + ledger 의 `Command` 대조 — 그 pid 를
   **입양**한다(§4-4 결정 필요). 결과를 한 줄씩 보고한다: `node2: pid 4123 dead,
   cleared` / `node3: pid 4188 alive, adopted`.
3. **첫 미완 단계부터 잇는다.** `steps` 에서 `done` 이 아닌 첫 단계를 찾아, 3.1 의
   요청으로 `NetUp` 을 그 단계부터 돌린다. 이미 끝난 단계는 건너뛴다(각 단계가 이미
   "있으면 재사용" 으로 되어 있어 다시 돌려도 안전하지만, 건너뛰는 편이 보고가 정직하다).
   `start` 가 미완이면 pid 가 없는 노드만 띄운다 — `startPhase` 가 이미 그렇게 한다.
4. **서버를 다시 확인한다.** 끝나면 `preflight.Check` 로 전 노드 생사를 한 번 더 보고,
   `net status` 와 같은 표를 찍는다.

`run --workspace-dir` 는 `net resume` 을 부르지 않는다. 이미 preflight 로 reuse /
rebuild-nodes / rebuild-all 을 고르고, 죽은 노드는 재구성 목록에 넣는다. 복구가 필요한
쪽은 "조립 도중 죽음" 이고 그것이 `net resume` 의 일이다. 테스트 도중 죽었으면 세션은
케이스 단위로 기록돼 있으므로, 다시 `run` 하면 preflight 가 네트워크를 재사용하고 케이스는
처음부터 돈다. **테스트 단위 이어받기는 하지 않는다**(케이스는 짧고, 반쯤 돈 케이스의
상태를 믿을 수 없다).

### 3.3 세트 잠금 — 파일이 아니라 배타 (구멍 3)

인벤토리를 파일로 쓰지 않는다. 대신 **할당하는 동안만** 서버 세트 단위의 잠금을 잡는다:
`~/.chainbench/<set-name>.lock` (서버 세트가 없는 내장 로컬 풀은 `local.lock`). 잠금 안에서
`Discover` → `Adopt` → `Take` 를 하고, 워크스페이스에 레코드를 저장한 뒤 놓는다. 그러면
두 프로세스가 같은 슬롯을 볼 수 없다 — 두 번째는 첫 번째가 저장한 워크스페이스를
`Adopt` 한다. 잠금 파일은 `session.Lock` 과 같은 형식(pid·호스트·명령)이라 stale 판정도
같은 코드를 쓴다.

이것이 "메모리가 정본" 을 지키면서 동시성만 막는 가장 작은 방법이다. 인벤토리 파일을
두면 워크스페이스 레코드와 인벤토리 파일이라는 **같은 사실 두 벌**이 생긴다.

### 3.4 게이트

- 조립 도중(`genesis` 뒤, `start` 도중 노드 2/4 기동 뒤) 프로세스를 죽이고 `net resume`
  하면 **남은 단계만** 돌고 노드 4개가 모두 뜬다. 단위 테스트는 stub 드라이버로
  `steps` 와 pid 판정을 검사하고, 라이브(gstable)로 실제 kill → resume 을 한 번 증명한다.
- `workspace.json` 의 `request` 만으로 같은 네트워크를 다시 조립할 수 있다(플래그 없이).
- 같은 세트에 두 프로세스가 동시에 `net allocate` 해도 슬롯이 겹치지 않는다(잠금 테스트).
- 사본 검사: 파일에 적힌 사실 중 다른 파일에도 있는 것이 **0** — `request` 는 결과와
  다른 사실이고, 잠금은 사실이 아니라 신호다.

## 4. 결정이 필요한 것

| # | 질문 | 제안 | 대안 |
|---|---|---|---|
| 1 | 요청을 `workspace.json` 에 기록하나 | **기록한다**(3.1). genesis 전 실패를 이어받는 유일한 길 | 기록하지 않고 재실행 시 플래그를 다시 받는다 — "다시 시작" 이지 복구가 아니다 |
| 2 | 복구 진입점 | **`net resume` 한 verb**(3.2). 되짚기는 명시적으로 | 모든 `net <step>` 이 stale lock 을 보면 자동으로 되짚는다 — 사용자가 모르는 새 pid 를 지우거나 입양하게 되어 놀랄 수 있다 |
| 3 | 인벤토리 동시성 | **세트 잠금**(3.3). 파일 정본을 만들지 않는다 | `inventory.json` 정본 — 사본 금지 원칙과 충돌 |
| 4 | 주인 없는 프로세스 | 명령줄이 ledger 의 것과 같으면 **입양**, 아니면 지금처럼 거부 | 항상 거부(안전하지만, 죽기 직전 띄운 우리 노드를 손으로 죽여야 한다) |

## 5. 하지 않는 것

- 테스트 단위 이어받기(3.2 끝).
- 원격 타깃의 데이터 플레인 정리(`rm` 이 원격을 아직 지원하지 않는 것과 같은 선).
- `sessions/`·`runs/` 의 정리나 압축 — F1 의 일이 아니다.

## 6. 구현 결과 (2026-08-28, 제안대로)

| 조각 | 어디에 | 무엇 |
|---|---|---|
| 요청 기록 | `chainsetup.State.Request` (`workspace.json` 의 `request`) | `net up` 의 `new` 단계 직후 `NetUpIn` 을 적는다(`DataDir` 은 비움). `NetUpIn`·`resource.ServerRef` 에 JSON 태그가 붙었다 |
| 되짚기 | `chainsetup.NetResume` (`verbs_resume.go`), CLI `net resume --workspace-dir [--binary]`, MCP `chainbench_net_resume`, `app.NetResume` | ① `withWorkspace` 가 stale 잠금을 인수하고 live 는 거부 ② `Workspace.Reconcile`: 기록된 pid 는 그 노드 머신에서 `PIDAlive`, 죽었으면 ledger·record 둘 다 지움; pid 없는 노드는 `FindBinary` + `ps -o command=` 로 우리 argv 와 같은 프로세스를 찾아 입양 ③ `firstUndone` 부터 `netUpFrom` 으로 잇기(요청의 stage 가 provision 이면 init·start 는 미완이 아니다) ④ stage 가 start 면 pid 없는 노드를 `StartNode` 로 되살림 ⑤ `NetworkStatus` 로 읽어 보고 |
| 단계 목록 | `upStepNames` (`verbs_up.go`) | `net up` 과 `resume` 이 같은 순서 하나를 쓴다 |
| 세트 잠금 | `chainsetup/setlock.go` + `session.AcquireLock` | `NetAllocate` 가 `~/.chainbench/<set>.lock`(내장 풀은 `local.lock`)을 잡고 워크스페이스를 저장한 뒤 놓는다. live 홀더는 10초까지 기다린다. `session.AcquireLock` 은 워크스페이스 잠금과 같은 코드다(stale 인수·같은 프로세스 중첩) |

게이트(§3.4) 실측:
- 단위: 죽은 pid 정리 후 재기동 · 주인 없는 프로세스 입양(같은 바이너리 다른 argv 는 무시) ·
  요청 없는 워크스페이스 거부 · provision 에서 죽은 start 요청을 `init` 부터 잇기(가짜 바이너리) ·
  잠금 중첩/해제 — `internal/chainsetup/resume_test.go`.
- 라이브(gstable): `net up --stage provision` → 요청을 start 로 바꿔 "start 도중 죽음" 을 재현 →
  `net resume` 이 `init`·`start` 만 돌려 4노드 기동 → node3 을 `kill -9` → `net resume` 이
  `pid 33693 dead, cleared` 후 `node3 started (pid 33719)` → `net health` 전 노드 블록 7 →
  `net stop` 4노드. 스텝 목록·요청·pid 외에 새 파일은 세트 잠금 하나뿐이고, 그것은 사실이
  아니라 신호다.
