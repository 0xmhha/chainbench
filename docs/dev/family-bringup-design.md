# 체인 패밀리별 기동 설계 — 상위 일관 · 하위 특화

> 목표: go-wemix / go-wbft / go-stablenet 을 **스크립트 없이 Go 함수 조합으로** 기동한다.
> 상위 레이어는 세 체인에 대해 동일하게 동작하고, 차이는 하위 레이어에만 존재한다.
>
> 근거: 2026-08-18 라이브 실측(3체인 기동) · `script/wemix-upgrade/{manage_chain.sh, upgrade_test/}` ·
> `go-wemix/build/bin/gwemix.sh` 코드 대조.

---

## 1. 이미 있는 것 (실측)

**wemix 기동 프리미티브는 이미 Go 로 존재한다.** 없는 것은 배선이다.

| 있는 것 | 위치 | 상태 |
|---|---|---|
| `poa.Config/Env/Member/Account` + `Validate()` | `consensus/poa/wemixconfig.go` | ✅ 형태 정확(idv5 128-hex 검증까지) |
| `poa.GenerateGenesis` / `DeployGovernance` / `EtcdInit` | `consensus/poa/bootstrap_exec.go` | ✅ 커맨드 정확, `Runner` seam 있음 |
| `poa.BootstrapPlan()` — 5단계 순서 데이터 | `consensus/poa/bootstrap.go` | ✅ 순서 정확 |
| `supervisor.Deps.LeaderGate` — "미배선이면 오류" 계약 | `core/supervisor` | ✅ 자리만 비어 있음 |
| `engine.GenesisSource` / `KeySource` seam | `internal/engine` | ✅ |

**그런데 이 프리미티브를 부르는 곳이 3군데로 흩어져 있다:**

```
cmd/chainbench/upgrade_run.go        (업그레이드 CLI)
internal/chainsetup/handoff_driver.go (핸드오프)
internal/chains/wemix/deploy/bootstrap.go (원격 fleet)
```

그리고 **정작 평범한 기동 경로에는 없다** — `chainsetup/wemix.go` 가 `NotImplemented` 로 남긴 그 자리다.
T7.11a 에서 고친 것과 같은 fan-out 이고, 같은 해법(유스케이스 1곳 수렴)이 필요하다.

---

## 2. 세 체인이 실제로 다른 지점

라이브 실행으로 확인한 차이만 적는다.

| 축 | stablenet / wbft (family `wbft`) | wemix (family `poa`) |
|---|---|---|
| **genesis 생성** | 템플릿 + 검증자셋 치환 (Go) | **바이너리가 생성** — `gwemix wemix genesis --data config.json --genesis <tmpl>` |
| **검증자셋 표현** | genesis `extraData` = RLP 인코딩 | `config.json` 의 `members[]`, genesis `extraData` = 부트노드 id |
| **기동 순서** | 전 노드 동시 | **boot 단독 → deploy-governance(IPC) → etcdInit → 나머지** |
| **합의 활성 조건** | genesis 만으로 충분 | 거버넌스 컨트랙트 배포 + etcd 클러스터 형성 |
| **포트 규칙** | p2p 대역 · rpc 대역 (etcd = p2p+1 예약) | `PORT` 하나가 셋 결정: http=PORT, **p2p=PORT+1**, ws=PORT+10, **etcd = p2p+1 및 p2p+2** |
| **노드 신원** | preset(address·128hex pubkey·nodekey) | **동일** — `id = "0x"+publicKey` 가 곧 idv5 |

**마지막 줄이 설계상 가장 중요하다.** `keys/preset` 의 `publicKey` 는 128-hex 비압축 공개키이고,
이는 `gwemix wemix nodeid` 의 `idv5` 와 같은 값이다. 즉 **wemix 전용 신원 생성기는 필요 없다** —
기존 preset 키셋이 그대로 `poa.Member{Addr, ID}` 를 채운다.

---

## 3. 상위 레이어가 깨지는 지점은 단 하나

현재 `supervisor.Deps.Launch(ctx, plan)` 는 **plan 의 전 노드를 한 번에** 띄운다.
wemix 는 이 가정 하나만 위반한다. 나머지(provision·health·teardown·재시도·진단)는 전부 공통이다.

따라서 **상위 레이어 전체를 바꿀 필요는 없다.** 바꿔야 할 것은 "언제 무엇을 띄우고, 그 사이에
무엇이 완료돼야 하는가"를 **패밀리가 데이터로 말하게** 하는 것뿐이다.

---

## 4. 설계 — 4개의 seam

### 4.1 `ConsensusFamily.BringUpPhases` — 기동 순서를 데이터로

```go
// core/registry
// Phase is one ordered group of a bring-up: nodes that start together, then
// the actions that must complete before the next phase may start.
type Phase struct {
    Name    string   // "boot" | "rest" | ...
    Nodes   []int    // 1-based indices launched in this phase
    Actions []string // action names run after these nodes are healthy
}

// ConsensusFamily gains:
//   BringUpPhases(nodes []node.Node) []Phase
```

- `wbft`: `[{Name:"all", Nodes:[1..N]}]` — 한 페이즈, 액션 없음. **현재 동작과 바이트 동일.**
- `poa`: `[{"boot",[1],["deploy-governance","init-etcd"]}, {"rest",[2..N],nil}]`

`Actions` 가 **문자열**인 것이 핵심이다. `testspec.rpcCall` 이 체인 어휘를 core 가 아니라 spec 에
두었던 것과 같은 이유로, core 는 `deploy-governance` 가 무엇인지 알지 않는다(C6 ACL 유지).

### 4.2 `supervisor.Deps.Action` — 액션 실행 seam

```go
// Action performs one named bring-up action against a node. What an action name
// means is chain-specific; the supervisor owns when it runs, how long it may
// take, and how a failure is classified. An action a phase requests but that is
// not wired is an error, not a silent pass.
Action func(ctx context.Context, name string, on node.Node) error
```

`LeaderGate`·`SwapBinary` 와 **완전히 같은 계약**이다(선언했는데 미배선이면 오류). 새 패턴이 아니다.

`Deps.Launch` 는 시그니처가 바뀐다:
```go
Launch func(ctx context.Context, plan driver.Plan, nodes []int) (LaunchResult, error)
```
`nodes` 가 nil 이면 전체 — 기존 호출자는 nil 을 넘기면 되므로 이관이 기계적이다.

### 4.3 `GenesisSource` — 산출물이 바이트 하나가 아니다

wemix 의 `config.json` 은 genesis 입력이자 **deploy-governance 입력**이다. 즉 genesis 단계의
부산물이 뒤 단계로 전달돼야 한다.

```go
// engine
type GenesisArtifacts struct {
    Genesis []byte
    // Extra are additional files the family needs on the target, by relative
    // path (poa: "wemix-config.json"). They are provisioned alongside the
    // genesis and their target paths are recorded, so a later action reads them
    // without reconstructing anything.
    Extra map[string][]byte
}
```
`GenesisSource.Build` 의 반환을 `[]byte` → `GenesisArtifacts` 로 넓힌다. wbft 는 `Extra` 가 nil.

### 4.4 `PortRule` — 패밀리가 대역 요구량을 말한다

현재 `portplan` 은 `etcd = p2p+1` 하나만 예약한다. wemix 는 **p2p+1(peer)·p2p+2(client)** 둘을 쓴다
(라이브 확인: `peerUrls https://127.0.0.1:8590`, `clientUrls http://localhost:8591`, p2p=8589).

```go
// ConsensusFamily gains:
//   PortReservation() portplan.Reservation   // {P2PSpan, RPCSpan}
```
- `wbft`: `{P2PSpan:2, RPCSpan:3}` (현행)
- `poa`: `{P2PSpan:3, RPCSpan:3}`

`serverset` 의 `p2pStep >= 2` 전역 검증은 **패밀리 기준으로 바뀐다.** 지금 값은 wemix 에서 틀렸다 —
어제 커밋한 "조용한 실패를 막는 규칙"이 정작 wemix 에서 한 칸 모자란다.

---

## 5. 지금 잘못된 것을 고치는 부분

### `poa.BuildGenesis` 는 genesis 생성기가 아니다

현재 `poa.Family.BuildGenesis` 는 템플릿의 `__CHAIN_ID__`/`__COINBASE__` 를 치환한다.
이것이 라이브에서 **구조적으로는 유효하지만 기능적으로 죽은** genesis 를 만들었다
(`alloc:{}`, `minerNodeId:"0x0"`) — 노드는 ethash 로 뜨고 `wemix` RPC 네임스페이스가 없다.
**오류보다 나쁘다.**

해법: 이 함수를 **템플릿 준비 단계**로 재배치한다.

```
poa.PrepareTemplate(template, chainID, coinbase)   // 치환된 템플릿
      ↓
poa.GenerateGenesis(binary, config.json, 준비된 템플릿)  // 바이너리가 진짜 genesis 생성
```
라이브에서 확인: `wemix genesis` 는 템플릿의 `config` 를 그대로 통과시키므로, 먼저 chainId 를
박아 넣으면 매니페스트의 8285 가 반영된다(현재는 템플릿 기본값 1111 로 떴다).

### `--networkid` 미방출

조립된 argv 에 없다. launchopt 의 poa 다이얼렉트에서 매니페스트 `network_id` 를 방출한다.

---

## 6. 패키지 배치와 의존 방향

```
core/registry     Phase · ConsensusFamily(+BringUpPhases, PortReservation)   ← 체인 무지
core/portplan     Reservation 로 대역 검증                                    ← 체인 무지
core/supervisor   페이즈 실행 · 액션 호출 · 진단 · 재시도 · teardown          ← 체인 무지
      ↑ 주입
consensus/poa     wemix 액션 구현(GenerateGenesis/DeployGovernance/EtcdInit) — 이미 존재
consensus/wbft    액션 없음
internal/engine   WemixGenesisSource · poaActions 배선
internal/app      유스케이스 1곳 (net up / setup / upgrade 가 공유)
```

새 패키지는 **0개**다. 기존 seam 확장 + 이미 있는 프리미티브 배선이다.

의존 방향은 현행 그대로 유지된다 — core 는 여전히 어떤 체인도 import 하지 않는다.

---

## 7. 작업 순서 (각 단계 독립 green)

| # | 작업 | 게이트 | 리스크 |
|---|---|---|---|
| **F1** | `PortReservation` — 패밀리별 대역 검증, `serverset` 전역 상수 제거 | 단위: poa 는 p2pStep 2 를 거부, wbft 는 허용 | 낮음 |
| **F2** | `--networkid` 방출 (poa 다이얼렉트) | 단위: argv 비교 | 낮음 |
| **F3** | `BringUpPhases` + `Deps.Launch(nodes)` + `Deps.Action` | 단위: wbft 는 1페이즈·현행과 동일 argv / 미배선 액션은 오류 | **중** — supervisor 시그니처 변경 |
| **F4** | `GenesisArtifacts` + `WemixGenesisSource`(`poa.PrepareTemplate`→`GenerateGenesis`) | 단위: wbft `Extra` nil · poa config.json 이 `Validate()` 통과 | 중 |
| **F5** | poa 액션 배선 + `app` 유스케이스 수렴(3곳 → 1곳) | **라이브: wemix 4노드 블록 생성 · 4노드 sealing 로테이션** | 중 |
| **F6** | `chainsetup/wemix.go` 의 `NotImplemented` 제거 | 라이브 재현 | 낮음 |

**F3 이 유일한 실질 리스크다.** `Deps.Launch` 시그니처가 바뀌면 기존 호출자(engine·netcompose·
chainsetup)가 전부 영향을 받는다. 다만 `nodes=nil → 전체` 규약이면 이관은 기계적이고,
wbft 계열은 argv·순서가 바이트 동일하게 유지되므로 **stablenet/wbft 회귀를 라이브로 즉시 확인**할 수 있다
(어제 절차 그대로).

---

## 8. 명시적 비목표

- **`gwemix.sh` 를 Go 로 포팅하지 않는다.** 재구현이 필요한 것은 *순서*이지 스크립트가 아니다.
  `wemix genesis`·`deploy-governance`·`admin.etcdInit()` 는 바이너리의 기능이고, 계속 바이너리를 호출한다
  (`keygen` 이 `bootnode` 를 호출하는 것과 같다). `Runner` seam 이 이미 그 경계다.
- **wemix 전용 신원 생성기를 만들지 않는다.** §2 마지막 줄 — preset 이 이미 idv5 를 담고 있다.
- **로컬/원격을 분기하지 않는다.** `Runner` 는 이미 로컬(`execRunner`)·SSH(`SSHPoaRunner`) 두 구현이 있다.

---

## 9. 검토가 필요한 열린 질문

1. **`Phase.Actions` 를 문자열로 둘 것인가, 타입 상수로 둘 것인가.**
   문자열이면 core 가 체인 어휘를 모르지만 오타가 런타임까지 간다. `testspec` 은 문자열 + `Unresolved`
   오프라인 검증으로 풀었다(`chainbench validate`). 같은 방식을 쓸 수 있다.
2. **부트노드 선정.** 현재 `poa.BootRole` 은 `RoleBoot || RoleValidator` 를 참으로 본다 —
   4검증자면 전부 참이라 선정 기준이 되지 못한다. 토폴로지의 `bootnode: true`(이미 `State.Bootnode` 로
   기록 중)를 정본으로 삼는 편이 맞다.
3. **`config.json` 의 `env` 정책값**(blockCreationTime·stakingMin 등)을 어디서 받을 것인가.
   DSL `env` 선언 / 서버 인벤토리 / 패밀리 기본값 중 어디에 두느냐는 사용자 결정 사항.
