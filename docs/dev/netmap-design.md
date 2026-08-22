# netmap — 노드 배치의 단일 소유자

> **[현행 설계]** 노드 배치(역할·호스트·포트·피어링).
> 지금 향하는 목표. 근거는 정본([[chainbench-requirements-review]]·[[chainbench-feature-spec]])이고,
> 작업 순서는 [[chainbench-worklist]] §1g 다.

> [[keyring-design]] 이 *누가 있는가*(신원)를 정했다면, 이 문서는 **어디에 어떻게 있는가**
> (배치)를 정한다. 해석 순서(N9)의 두 번째 단계다:
> ① keyring → **② netmap** → ③ enode → ④ genesis → ⑤ config.
> 실측: 2026-08-22, AST 기반.

---

## 1. 문제 — 측정한 것만

### 1.1 "노드 배치"를 뜻하는 타입이 8개다

역할·호스트·포트를 함께 담는 구조체를 AST 로 전수 조사한 결과다.

| 타입 | 어디 | 담는 것 | 성격 |
|---|---|---|---|
| `place.NodePlacement` | `core/place` | Name·Host·Ports·DataPath | 할당 **결과** |
| `serverset.Server` | `serverset` | Host·Slots·Ports(밴드)·SSH | 인벤토리 항목 |
| `topology.Node` | `core/topology` | Role·SyncMode·Bootnode | **선언** |
| `driver.NodeSpec` | `core/driver` | Role·Host·Binary·DataDir·Config | 기동 입력 |
| `node.Node` | `core/node` | Role·Host·RPCURL·Ports·PID | **런타임** |
| `netcompose.NodeState` | `netcompose` | Role·Host·P2P·HTTP·… (평면) | 워크스페이스 영속 |
| `upgrade.NodeSpec` | `consensus/upgrade` | Role·Producer·Ports·Pubkey | 핸드오프 사본 |
| `deploy.Server` | `chains/wemix/deploy` | Host·Role·Binary·SyncMode | wemix 원격 사본 |

앞의 다섯은 **단계가 다르므로 타입이 다른 것이 정당하다**(선언→할당→기동→런타임).
문제는 뒤의 셋 — 같은 단계의 정보를 **파이프라인 밖에서 다시 정의**한 사본들이다. 그리고
정당한 다섯끼리도 **서로 변환되지 않아** 호출자가 필드를 손으로 옮긴다.

### 1.2 포트 집합의 표현이 3벌이고, 하나는 etcd 를 잃어버린다

| 타입 | 필드 | 문제 |
|---|---|---|
| `portplan.Ports` | P2P·**Etcd**·HTTP·WS·Auth·Metrics | 완전하다 |
| `node.Endpoints` | P2P·HTTP·WS·Auth·Metrics | **Etcd 가 없다** |
| `netcompose.NodeState` | P2P·HTTP·WS·Auth·Metrics (개별 int) | Etcd 가 없다 |

etcd 포트는 wemix 계열에서 합의에 필수이고(바이너리가 `p2p+1` 로 유도, [[server-inventory]] §3),
`p2pStep>=2` 규칙이 존재하는 이유 그 자체다. 그런데 **런타임 표현으로 넘어오는 순간 사라진다**
— 살아 있는 wemix 노드의 etcd 포트를 물어볼 방법이 없다.

### 1.3 역할 어휘가 3벌이고, 한 타입 안에서도 갈라져 있다

```go
node.Role:    validator | endpoint | boot | en     ← en 과 endpoint 가 공존한다
deploy.Role:  wemix_bp | wbft_bp | en | pn
validatorset: "validator" (문자열)
topology:     문자열 (검증은 topology.go 안에서)
```

`node.RoleEndpoint("endpoint")` 와 `node.RoleEN("en")` 은 **같은 것의 두 철자**다.
N0 이 정하는 목표 어휘는 **`bp` · `en` · `pn` 3종**이고 `boot` 는 역할이 아니라 속성이다.

### 1.4 static-nodes 조립이 4벌이고, 전부 풀메시다

"이 노드가 누구에게 붙는가"를 계산하는 코드가 네 곳에 있다.

| 어디 | 코드 |
|---|---|
| `engine/launcher.go:269` | preset 전 노드 → enode 목록 → 모든 노드의 StaticNodes |
| `netcompose/steps_compose.go:367` | 위와 **거의 동일** (호스트 해석만 다름) |
| `consensus/upgrade/mesh.go:52` | `Enodes()` — 같은 계산, 핸드오프용 |
| `chainsetup/handoff_driver.go` | 위 결과를 `static-nodes.json` 으로 직접 파일에 |

네 곳 모두 **전 노드 풀메시**를 만든다. 그래서 N0b 의 요구 — `bp ↔ pn ↔ en`(en 은 bp 를
직접 모른다), poa 에 `pn` 선언 시 오류 — 는 **네 곳을 다 고치기 전에는 표현할 수 없다.**
keyring 이전의 `keys`/`keygen`/`keymat`/`keyreg` 와 정확히 같은 모양이다.

### 1.5 enode 조립 함수 자체는 이미 하나다 — 좋은 소식

`nodeconfig.Enode(publicKey, host, p2pPort)` 하나뿐이고 네 조립 지점 모두 이것을 쓴다.
중복은 **함수가 아니라 "누구를 목록에 넣는가"라는 정책**이다. 그 정책이 netmap 의 소유물이다.

---

## 2. 설계 — `core/netmap` (L1)

### 2.1 위치와 근거

```
L1  core/keyring   누가 있는가        ← 완료
L1  core/netmap    어디에 어떻게 있는가 ← 이 문서
```

- **L1 인 이유**: `place`(할당)·`portplan`(포트 계산)·`topology`(선언)와 같은 원시 계층이다.
  오케스트레이터를 모르고, 체인을 모른다. keyring 과 대칭.
- **의존**: `core/node`(Role 어휘의 최종 거처) · `core/portplan` 만. keyring 을 import
  하지 **않는다** — netmap 은 공개키를 받을 뿐, 키가 어디서 왔는지 모른다(신원과 배치의 분리,
  [[keyring-design]] §2.1 과 동일 원리).

### 2.2 소유하는 것

```
core/netmap
├─ (Role)      bp | en | pn — 상수는 **`core/node` 에 추가**된다(§2.5). netmap 은
│              재정의하지 않고 legacy 매핑(validator→bp, endpoint→en)만 소유
├─ Ports       P2P·Etcd·HTTP·WS·Auth·Metrics — portplan.Ports 승격
├─ Pool        가용 자원: IP 목록 × 포트베이스 목록 (인벤토리에서 읽음, §2.2a)
├─ Assign      Pool 을 소비해 노드별 (IP, 포트집합) 을 결정적으로 할당 (§2.2b)
├─ NodeLabel   "node1"·"bp1" — 노드를 지칭하는 명명된 타입 + 규칙 함수 (§2.5)
├─ Map         NodeLabel → Placement{Role, Host, Ports, DataDir}  (정방향)
│              Host:Port → NodeLabel                               (역방향, N7)
├─ Peering     역할에서 파생되는 그래프 (N0b)
│              mesh    — 현행과 동일 (기본)
│              proxied — bp↔bp 직결 + bp↔pn↔en, en 은 bp 를 모른다  ← **실사용 요구**
│              poa 에 pn 선언 시 **오류** (조용한 무시 금지)
└─ StaticNodes(label, pubkeyOf) → []enode
               "이 노드가 다이얼할 목록" — 4벌의 정책이 이 한 함수로
```

### 2.2a Pool — 가용 IP·포트는 목록이고, 민감정보다

호스트 주소와 포트는 [[server-inventory]] 의 원칙 그대로 **gitignore 된 설정 파일에서만**
온다. 파일이 주는 것은 개별 서버 항목이 아니라 **가용 자원의 풀**이다.

```yaml
# remote-server-config.yaml (gitignore) — 스키마 확장
version: 2
pool:
  hosts: [<ip1>, <ip2>, <ip3>, <ip4>, <ip5>]   # 가용 IP 목록
  slots: 4                                      # 한 호스트가 담을 포트 슬롯 수
  ports:                                        # 슬롯이 밟는 두 대역 (구현형)
    p2p: { base: 31000, step: 10 }
    rpc: { base: 8600,  step: 10 }
ssh:
  port: 22
  user: <user>          # 비밀은 env 가 이긴다 (기존 규칙 유지)
  sudo: true            # sudo 사용 가능 여부 — 기동 절차가 참조
dataRoot: /data/chainbench   # 설정 관리 루트. 노드 datadir 은 이 **하위**에 생성
```

- **초안의 `portBases: [8080, 8180, …]`(단일 숫자 목록)은 구현에서 두 대역으로 바뀌었다.**
  노드 하나는 포트 하나가 아니라 묶음(P2P·Etcd·HTTP·WS·Auth·Metrics)을 쓰는데, 그 묶음은
  `portplan` 이 **겹치지 않는 두 대역**(p2p / rpc)에서 파생한다 — p2p 를 1 간격으로 채우면
  wemix 가 유도하는 etcd(p2p+1)가 다음 노드의 p2p 를 덮는다. 단일 목록으로는 그 규칙을
  표현할 수 없고, 무엇보다 **현행 배치를 재현하지 못해** NM3 이 모든 넷의 포트를 옮기게 된다.
  슬롯 n 의 포트 = `portplan.Plan(n, p2p.base, p2p.step, rpc.base, rpc.step)`.
- `dataRoot` 는 노드 datadir 이 아니다 — 설정·산출물이 놓이는 서버 쪽 루트이고,
  노드별 datadir 은 `<dataRoot>/<label>/` 로 그 하위에 생긴다.
- **v2 단독으로 간다** (확정 2026-08-22). v1(서버별 항목) 호환을 두지 않는 이유: 실제 인벤토리
  파일은 gitignore 라 저장소에 없고 이 시점에 배포된 파일이 확인되지 않는다 — 전환 비용이 가장
  싼 때다. 호환 코드는 두 스키마를 아는 로더를 남기고, 그 로더가 곧 다섯 번째 폴딩표가 된다.
  `remote-server-config.sample.yaml` 을 v2 로 교체하고, v1 파일을 만나면 **무엇을 어떻게 고쳐야
  하는지 말하며 거부**한다(조용한 강등 금지).

### 2.2b Assign — 풀을 소비하는 결정적 할당

노드가 가용 IP 보다 많을 수 있다. 그때 **IP 를 재사용하되 포트 슬롯으로 구분**한다.

할당 순서는 **IP 먼저, 그다음 슬롯**이다: node1 부터 순서대로 IP 목록을 소비하고,
IP 가 소진되면 슬롯 인덱스를 올려 첫 IP 부터 다시 돈다. 다섯 주소를 적었다는 것은
네트워크를 그 다섯 대에 **펼치겠다**는 뜻이고, 첫 대를 채우고 넘어가려면 그렇게 적어야 한다.

```
hosts = [ip1..ip5], slots = 4, p2p 31000/10 · rpc 8600/10, 노드 15개
슬롯 n 의 포트 = portplan.Plan(n, …)  →  slot1 p2p 31000/http 8600, slot2 31010/8610, …

node1  → (ip1, slot1)   node6  → (ip1, slot2)   node11 → (ip1, slot3)
node2  → (ip2, slot1)   node7  → (ip2, slot2)   node12 → (ip2, slot3)
node3  → (ip3, slot1)   node8  → (ip3, slot2)   ...
node4  → (ip4, slot1)   node9  → (ip4, slot2)   node15 → (ip5, slot3)
node5  → (ip5, slot1)   node10 → (ip5, slot2)
```

- **결정적이다**: 같은 풀 + 같은 노드 수 → 항상 같은 할당. 재실행이 다른 배치를 만들지 않는다.
- **초과는 오류다**: 노드 수 > |hosts| × slots 면 몇 개가 부족한지 말하고 거부한다.
  조용히 겹치는 (ip, port) 를 만드는 것이 최악의 결과다.
- 같은 IP 의 두 노드는 포트베이스가 다르므로 (ip, port) 조합은 항상 유일하다 —
  `portplan.Validate` 가 최종 확인한다.
- 기존 `place.Allocator` 의 두 모드(LocalStepped·RemotePerHost)는 이 일반형의 특수한
  경우다: 로컬 = hosts 1개 × 베이스 N, 원격 호스트당 1노드 = hosts N × 베이스 1.
  **Assign 이 셋을 하나로 흡수**하고 `place` 는 그 구현이 된다.

**`StaticNodes` 가 공개키를 함수로 받는 이유**: enode = pubkey + host + port 인데 pubkey 는
keyring 의 것이다. netmap 이 keyring 을 import 하면 배치가 신원에 묶인다. 호출자(엔진·컴포즈)가
`func(Label) (pubkey string, ok bool)` 을 주입하면 두 소유자가 서로를 모른 채 조합된다.

> **`place.LocalOSAssigned` 는 실측상 프로덕션 소비자가 0이다**(2026-08-22). OS 가 빈 포트를
> 고르는 전략이라 격자로 흡수되지 않으며, 아무도 부르지 않으므로 `place` 잔여분과 함께 정리
> 대상이다. `MinValidators` 도 전 호출지에서 값이 1이었고, "프로듀서가 최소 하나"라는 규칙은
> `Assign` 이 직접 강제한다 — 블록을 만들 노드가 없는 네트워크는 어느 패밀리에서도 전진하지 않는다.

### 2.3 흡수·이동 목록

| 지금 | 어디로 | 방식 |
|---|---|---|
| `portplan.Ports` | `netmap.Ports` | **이동**. `portplan` 은 계산 함수(`Plan`·`Validate`)만 남기고 타입은 netmap 이 소유 |
| `node.Endpoints` | `netmap.Ports` | **대체**. Etcd 누락 해소. `node.Node.Ports` 의 타입이 바뀐다 |
| `node.RoleEN` | 삭제 | `en` 은 `RoleEndpoint` 의 새 철자가 아니라 중복이었다 |
| `place.NodePlacement.Name` | `netmap.NodeLabel` | 명명된 타입으로 승격. `keyring.Label` 과 이름부터 다르게 — 노드 라벨과 링 항목은 다른 개념이다 |
| static-nodes 조립 4곳 | `netmap.StaticNodes` | engine·netcompose·upgrade.mesh·chainsetup 이 호출자로 |
| `upgrade.NodeSpec`·`deploy.Server` 의 배치 필드 | netmap 참조 | **F4·F5 때** — A4b 와 같은 이유로 핸드오프 코드는 그때 함께 |

### 2.4 하지 않는 것

- `topology`(선언) 를 흡수하지 않는다. 선언은 청사진(N1)의 영역이고, netmap 은 **해석된 결과**다.
- `serverset`(인벤토리) 를 흡수하지 않는다. 인벤토리는 *가용 자원*, netmap 은 *배정된 결과*.
- `driver.NodeSpec`·`node.Node` 를 없애지 않는다. 기동 입력과 런타임은 다른 단계다.
  netmap 은 그들이 **배치 필드를 복제하는 대신 참조하게** 만든다.

### 2.5 DSL 과의 어휘 공유 — 방향은 netmap → DSL

DSL 이 노드를 지칭하는 라벨과 netmap 의 라벨이 **같은 문자열**이어야 한다. 지금 DSL 은
`"node" + strconv.Itoa(n.Index)` 를 **자체 하드코딩**한다(`testspec/builtins.go:423`) —
netmap 이 라벨 규칙을 바꾸면 DSL 이 조용히 어긋난다.

공유 방향은 레이어가 정한다 — 단, 소유자는 둘로 갈린다:

- **역할 상수는 `core/node`(L0) 에.** `node.Role` 이 이미 어휘의 거처이고 fan-in 25 다.
  netmap(L1) 이 재정의하면 `Role` 타입이 세 벌이 되고, L0 인 `node` 는 그것을 import 할
  수도 없다. bp/en/pn 은 기존 `node.Role` 의 새 값으로 들어가고, `RoleEN`/`RoleEndpoint`
  중복은 이때 해소된다(N0).
- **라벨 규칙은 `netmap`(L1) 에.** `NodeLabel` 타입과 규칙 함수를 netmap 이 소유하고,
  DSL(L3)·엔진이 import 한다.

```go
// core/node — 어휘 (DSL 은 이미 이 패키지를 import 한다)
const RoleBP, RoleEN, RolePN Role

// core/netmap — 라벨
type NodeLabel string
func LabelFor(index int) NodeLabel              // "node7"  신원
func RoleLabel(role node.Role, ord int) NodeLabel  // "en2"   별칭
func ParseRoleLabel(NodeLabel) (node.Role, int, error)
```

`keyring.Label`(링 항목)과는 **타입도 이름도 다르다** — 겹치면 이름 규칙(다른 개념=다른 이름)을
어긴다.

#### 2.5a 신원과 별칭 — 노드는 두 이름을 갖는다 (확정 2026-08-22)

노드를 **번호로** 부를지(`node7`) **역할로** 부를지(`en2`)는 하나를 고르는 문제가 아니었다.
두 표기는 **청중이 다르다**.

| | 신원 (identity) | 별칭 (alias) |
|---|---|---|
| 형태 | `node7` — 전역 1-based | `en2` — 역할 내 서수 |
| 답하는 질문 | "지금 몇 개가 떠 있고 이건 몇 번인가" | "엔드포인트 하나에 보내라" |
| 저장 | ✅ `Index int` (워크스페이스·세션 — **현행 그대로**) | ❌ 배치에서 파생 |
| 디스크 | ✅ `logs/node7.log` · `<root>/node7/` | — |
| keyring 링 항목 | ✅ 일치 (`node7`) | — |
| DSL `on:` | ✅ 받음 | ✅ 받음 (`en2`·`en:1`·`en:any`) |

**신원을 `Index` 로 두는 이유** 셋: (1) 운영자가 전체를 세고 순회하는 단위가 번호다.
(2) keyring 링 항목이 이미 `node1..nodeN` 이라, 역할 라벨을 신원으로 삼으면 enode 조립
(`StaticNodes(label, pubkeyOf)`)에 `bp1 → node3 → pubkey` 번역기가 끼어든다 — 배치와 신원
사이에 번역이 생기는 건 지금 없애려는 부채다. (3) 디스크 경로가 이미 번호 파생이라 개명이
곧 마이그레이션이 된다.

**별칭이 필요한 이유**는 운영자 조망이 아니라 **정의서의 이식성**이다. spec 은 한 번 쓰고
여러 토폴로지에서 돈다. `en1` 은 4노드 넷에서도 40노드 넷에서도 "첫 엔드포인트"지만,
`node7` 은 넷이 바뀌면 다른 노드다. 메인넷 구성이 bp–pn–en 이고 **트랜잭션이 en 을 거쳐
전파**되므로, 정의서는 "en 에 보낸다"를 말할 수 있어야 한다.

**따라서 개명은 없다.** 영속되는 식별자는 이미 라벨 문자열이 아니라 `Index int` 이고,
라벨은 표시·주소지정 시점의 파생이다. 필요한 것은 두 표기를 **둘 다 받는 것**뿐이다.

**역할 연속성은 강제하지 않는다.** 카운트 경로는 bp 를 먼저 배치하지만, 토폴로지 파일은
사용자의 `Index` 순서가 권한을 갖는다 — poa 부트스트랩은 프로듀서가 **먼저 떠야** 하므로
정렬을 강제하면 그 요구와 충돌한다. 대신 `Ord`(역할 내 서수)를 **Index 오름차순으로 결정**해,
역할이 섞여 있어도 `en1` 은 언제나 "가장 앞선 en" 이다. 전체 조망은 `net map` 이 역할별
집계를 함께 찍어 해결한다.

### 2.6 체인별 차이 — 새 타입이 아니라 기존 두 seam

초안은 `PortProfile{NeedsEtcd, AllowsPN}` 하나를 뒀는데, **이름이 어색한 이유는 두 개념을
한 타입에 넣었기 때문**이었다. 포트 요구(파생·검증의 입력)와 역할 허용(선언의 검증)은
성격이 다르고 — 그리고 둘 다 **이미 이름이 있다.**

| 질문 | 기존 seam | 어디에 |
|---|---|---|
| 이 패밀리 노드가 요구하는 포트 span 은? | `PortReservation() portplan.Reservation` | [[family-bringup-design]] §seam |
| 이 역할이 이 패밀리에서 유효한가? | `SupportsRole(Role) bool` | [[chainbench-worklist]] N0b |

netmap 은 새 타입을 만들지 않고 이 둘을 **소비**한다: Assign 이 `PortReservation` 으로
베이스 간격을 검증하고(wemix 는 etcd=p2p+1 이라 span 2 이상), Peering 이 `SupportsRole` 로
poa+pn 선언을 거부한다. `serverset` 의 전역 `p2pStep>=2` 강제는 이렇게 패밀리 seam 으로
옮겨져 F1 게이트("poa 는 step 2 거부, wbft 는 허용")가 닫힌다.

| | wbft 계열 | poa(wemix) |
|---|---|---|
| etcd | 안 씀 | **p2p+1 로 유도** — span 요구의 이유 |
| pn 역할 | static-nodes 그래프로 표현 | **없음** — `SupportsRole(pn)=false`, 선언 시 오류 |

**proxied 는 선택 기능이 아니다** (확정 2026-08-22): 운영 중인 메인넷 구성이 **bp – pn – en**
이고 **트랜잭션이 en 을 거쳐 전파**된다 — bp 로 직접 보내지 않는다. 재현 대상이 그 구조이므로
mesh 만으로는 실제 전파 경로를 태우지 못하며, proxied 는 NM2 의 게이트에 포함된다.

**proxied 의 정확한 형태는 라이브가 정정했다** (2026-08-22, 실 gstable 5노드):
초안대로 **bp 가 pn 만 다이얼하게** 하면 **블록이 하나도 생성되지 않는다.** 모든 bp 가
`ROUND-CHANGE` 를 브로드캐스트하며 `currentRoundChanges.count=1`(자기 것만)에 머물렀고,
pn 로그에는 WBFT 라인이 2줄뿐이었다 — **pn 은 검증자가 아니라 합의 트래픽을 중계하지 않는다.**
프록시 계층이 나르는 것은 트랜잭션과 블록이지 합의 메시지가 아니다.

따라서 확정형은 **bp↔bp 직결 + bp↔pn + pn↔en** 이다. `en` 이 bp 를 모른다는 성질(이 구조가
존재하는 이유)은 그대로 유지된다. 정정 후 같은 토폴로지로 재기동해 **블록 전진·피어 4·api 9/9
pass·고아 0** 을 확인했다.

### 2.7 이름 충돌 검증 — AST 실측

이 설계가 도입하는 이름을 기존 타입 선언 전수와 대조했다([[layers]] §5b 규칙 적용).

| 초안 이름 | 충돌 | 확정 | 근거 |
|---|---|---|---|
| `PortProfile` | (신조어) | **폐기** → `PortReservation` + `SupportsRole` | 두 개념 한 타입이 어색함의 뿌리. 둘 다 기존 설계 어휘가 있다 |
| `netmap.Label` | `keyring.Label` | `netmap.NodeLabel` | 다른 개념은 다른 이름. blueprint 원안 복귀 |
| `netmap.Role` | `node.Role`·`deploy.Role` ×2 존재 | **정의하지 않음** — 상수를 `node.Role` 에 | 세 번째 Role 을 만들지 않는다. L0 이 어휘 거처 |
| `netmap.Placement`(맵) | `serverset.Placement` | 맵=`netmap.Map`, 항목=`netmap.Placement` | serverset.Placement 은 Assign 에 흡수·소멸(NM1b) — 이행기만 공존 |
| `netmap.Ports` | `serverset.Ports`(밴드) | 유지 | 충돌 상대는 v2 `pool.ports` 로 이동, `serverset.Ports` 는 파생 뷰로만 잔존(NM3 에서 소멸) |
| `Vocabulary`(서브모듈) | — | **삭제** | 아무것도 말하지 않는 이름. `NodeLabel` 함수 + `node.Role` 상수로 족하다 |

---

## 3. 표면 — cmd 와 MCP

**스텝은 늘리지 않는다.** `net allocate` 의 산출물이 netmap 이고, 바뀌는 것은 산출물의 소유자와
표현이다. 다만 **조회는 스텝이 아니다** — 스텝은 상태를 바꾸고 조회는 바꾸지 않는다. 그래서
조회 명령 둘(`net map`·`net pool`)을 더한다. `net pool` 이 특히 필요한 이유: **"15노드를
요청했는데 왜 거부됐는가"를 명령 하나로 답할 수 있어야** 한다(§2.2b 의 초과 거부).

| 표면 | 지금 | 후 |
|---|---|---|
| `net allocate` | 노드 테이블 생성(NodeState 평면) | 동일 명령, 산출 = netmap. 출력에 **역할·호스트·포트 전부**(etcd 포함) |
| `net allocate --peering` | (없음 — 항상 풀메시) | `mesh`(기본) \| `proxied`. poa+pn 은 오류 |
| **`net map`** ★신규 | (없음) | 대장 조회. `--node 7` · `--label en1` · `--host <ip>` · `--port 8080` — **네 방향** · `--json` |
| **`net pool`** ★신규 | (없음) | 가용 자원 요약: 출처 · hosts×bases · **cap** · 현재 사용량. **자격증명은 출력하지 않는다** |
| `net status` / MCP `chainbench_net_status` | 포트 일부 표시 | netmap 조회 — 역방향(host:port→label) 포함 |
| MCP `chainbench_network_topology` | 런타임 피어 조회 | 유지. **계획된** 그래프는 allocate 출력이 답한다 |

keyring 때처럼 유스케이스는 `app` 에, 표면은 바인딩·렌더링만 (K8 선례):
`app.NetMap`·`app.NetPool` 하나씩, CLI 명령 하나씩, MCP 도구 하나씩
(`chainbench_net_map`·`chainbench_net_pool`).

**MCP 의 비밀 취급** — keyring 이 `export` 를 MCP 에 **의도적으로 노출하지 않은** 판단(K8)을
그대로 잇는다. SSH user/password/key_file 은 **어떤 도구도 반환하지 않으며**, 그 부재를
테스트로 고정한다. 호스트 주소는 반환한다(노드를 지목하려면 필요하고 `net status` 가 이미
준다). `net pool` 은 자원의 **개수와 출처**를 말하되 인벤토리 경로 이상은 말하지 않는다.

**netmap 이 소유하지 않는 것 — SSH 자격증명** (확정 2026-08-22). 접속 수단의 주인은 이미
셋이다: `serverset`(인벤토리 읽기)·`core/remote`(전송)·`core/target`(해석). netmap 이 넷째가
되면 지금 없애려는 파편화를 재생산한다. 그리고 비밀이 대장 구조체에 섞이면 **대장 자체를
출력·직렬화할 수 없게 된다** — `net map`·MCP 응답·세션 아티팩트가 전부 마스킹 대상이 된다.
netmap 의 `Placement` 는 인벤토리 키(`Server string`)만 들고, 접속이 필요한 순간
`target.Resolve` 가 자격증명을 env/인벤토리에서 가져온다. **netmap 은 어디에 있는지를 알고,
어떻게 들어가는지는 모른다.** `sudo` 도 같다 — 보관·전달만 하고 소비는 기동 절차가 한다(NM-e).

---

## 4. 작업 단계

| # | 작업 | 게이트 |
|---|---|---|
| **NM1** | `core/netmap` 신설 — node.Role 에 bp/en/pn 추가·Ports(Etcd 포함)·Map/Placement·NodeLabel·legacy 매핑 | 기존 topology/NodeState 를 **읽기 호환**으로 흡수 · 문자열 역할이 남지 않음 · DSL 의 `"node"+i` 하드코딩이 `netmap.NodeLabel` 로 |
| **NM1c** | **선행 결함 수정** — 셀렉터 폴딩표를 `netmap.NormalizeRole` 경유로 · 신원/별칭 두 표기(§2.5a) | ☑ **완료 2026-08-22.** `on:"bp1"` 이 `RoleBP`·`RoleValidator` 양쪽에 매칭 · `pn` 셀렉터 · `on:"node7"`(신원) 해석 · `RoleLabel`/`ParseRoleLabel` · `Placement{Index, Ord}` |
| **NM1b** | Pool + Assign — 인벤토리 **v2 단독** 스키마(hosts×slots·ports 2대역·sudo·dataRoot) + 결정적 할당 | ☑ **완료 2026-08-22.** 5호스트×4슬롯×15노드 테이블 테스트 · 초과는 부족 수를 말하며 거부 · **`place` 두 결정적 모드를 바이트 동일 재현**(등가 테스트) · v1 은 고칠 방법을 말하며 거부 · 루프백 여부로 local/remote 판정 |
| **NM2** | Peering 파생 + `StaticNodes` + `SupportsRole` seam — mesh 는 **현행 argv 와 바이트 동일** | ☑ **완료 2026-08-22.** 골든(`engine.armSpecs` 산출 config 의 enode 목록 == `netmap.Mesh`, self 항목 포함까지) · proxied 는 en 목록에 bp 없음 · pn 없는 proxied 거부 · `ConsensusFamily.SupportsRole` 로 poa+pn 거부. **잔여**: `serverset` 전역 `p2pStep>=2` → `PortReservation` seam 은 F1 (패밀리 인터페이스가 포트 예약을 말하게 하는 일이라 F 트랙) |
| **NM2b** | **`Layout`** — dataRoot 하위 경로 파생(순수 계산, 쓰기 없음). `"node%d"` 32곳 중 경로 파생분을 흡수 | 같은 함수가 로컬 워크스페이스와 서버 destination 양쪽 경로를 만든다 · 파일 쓰기 0건 · [[key-and-material-design]] §4.3 레이아웃(`bin`/`material`/`run`)의 구현체 |
| **NM3** | 조립 4곳 → netmap 소비 (engine·netcompose 먼저, upgrade·chainsetup 은 F4·F5 와) · `node.Endpoints`→`netmap.Ports`(Etcd 부활) | ◐ **static-nodes + 할당 전환 완료 2026-08-22**: 철자 무관 술어(`netmap.Is`) 9곳 · engine·netcompose 가 `netmap.Peering.StaticNodes` 경유 · `--peering` CLI/MCP 노출 · **라이브**(stablenet mesh 4노드 api 9/9 · wbft mesh 4노드 블록 54 · stablenet **proxied** 5노드 블록 전진+api 9/9, 고아 0). **할당 전환**: `place.Allocator` 프로덕션 호출 **0** — engine·netcompose·chainsetup 이 `netmap.Assign` 경유, `serverset.Placement` 가 `Pool` 을 함께 나른다. 라이브 재확인(포트 동일·api 9/9·고아 0). **잔여**: `node.Endpoints`→`netmap.Ports`(fan-in 25, etcd 부활) |
| **NM4** | 표면 — `net map`·`net pool` 신설 + allocate 산출 강화 + `--peering` (§3) | CLI/MCP 동일 출력 · **자격증명 미노출을 테스트로 고정** · A2 표 갱신 |
| **NM5** | 라벨 영속 — 워크스페이스에 Label 기록, 로그의 host:port 역추적 | `place.NodePlacement.Name` 이 버려지지 않음 (N7 의 원래 동기) |

**순서**: NM1 ☑ → **NM1c ☑** → **NM1b ☑** → **NM2 ☑** → → NM2b → NM3 → NM4 → NM5.
NM1c 를 앞세운 이유는 §2.5a 의 결함이 NM3 에서 터지기 때문이다 — 그 시점엔 배치·persist·argv 가
함께 움직여 원인이 셋으로 갈린다. NM2b(Layout)를 NM3 앞에 두는 이유는, NM3 이 만지는 32곳에
경로 파생이 섞여 있어 Layout 이 없으면 같은 파일을 두 번 고치게 되기 때문이다.

**N 계열과의 관계**: NM1~5 = N7 + N0 + N0b. 완료 시 워크리스트의 세 항목이 닫히고,
N1(청사진 선언)은 netmap 을 산출 타입으로 삼는다.

---

## 5. 열린 질문

| # | 질문 | 기울기 |
|---|---|---|
| NM-a | ~~`netmap.Label` vs `keyring.Label`~~ | 해소 — `NodeLabel` 로 이름부터 분리(§2.5) |
| NM-b | `node.Node.Ports` 타입 교체는 파급이 크다(25 fan-in). 단계적으로? | NM1 에서 alias 로 시작, NM3 에서 교체 |
| ~~NM-c~~ | ~~proxied 에서 pn 이 없으면?~~ | **해소** — 오류(NM2 구현). mesh 로 조용히 강등하지 않는다 |
| ~~NM-i~~ | ~~proxied 에서 bp 는 다른 bp 를 직접 아는가?~~ | **해소 — 그렇다** (라이브 2026-08-22). pn 경유 전파로는 합의가 형성되지 않는다(§2.6). bp↔bp 직결 |
| ~~NM-f~~ | ~~라벨을 `node7` 로 둘지 `en2` 로 둘지~~ | **해소 (2026-08-22)** — 둘 다. 신원=`Index`, 별칭=역할 라벨(§2.5a). 개명 없음 |
| ~~NM-g~~ | ~~인벤토리 v1 호환 기간~~ | **해소 (2026-08-22)** — v2 단독(§2.2a) |
| ~~NM-h~~ | ~~proxied 를 실제로 쓰는가~~ | **해소 (2026-08-22)** — 쓴다. 메인넷이 bp–pn–en 이고 tx 가 en 경유(§2.6) |
| NM-d | 역할별 IP 선호(예: bp 는 앞쪽 IP 고정)가 필요한가? | 당장은 아니오 — 순서 결정성으로 충분. 근거가 생기면 Assign 에 정책 주입 |
| NM-e | sudo 는 어디가 소비하나? | 기동 절차(driver/bringup). netmap 은 보관·전달만 — 실행하지 않는다 |
