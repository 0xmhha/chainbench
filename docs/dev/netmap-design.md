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
├─ Role        bp | en | pn — 어휘의 단일 정의 (N0)
│              legacy 매핑: validator→bp, endpoint→en (읽기 호환)
├─ Ports       P2P·Etcd·HTTP·WS·Auth·Metrics — portplan.Ports 승격
├─ Pool        가용 자원: IP 목록 × 포트베이스 목록 (인벤토리에서 읽음, §2.2a)
├─ Assign      Pool 을 소비해 노드별 (IP, 포트집합) 을 결정적으로 할당 (§2.2b)
├─ Placement   Label → {Role, Host, Ports, DataDir}  (정방향 — Assign 의 산출)
│              Host:Port → Label                      (역방향, N7)
├─ Vocabulary  노드 라벨 규칙·역할 상수 — DSL 이 import 하는 쪽 (§2.5)
├─ Peering     역할에서 파생되는 그래프 (N0b)
│              mesh    — 현행과 동일 (기본)
│              proxied — bp↔pn↔en, en 은 bp 를 모른다
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
  portBases: [8080, 8180, 8280, 8380]           # 노드 1개가 소비하는 포트 묶음의 베이스
ssh:
  port: 22
  user: <user>          # 비밀은 env 가 이긴다 (기존 규칙 유지)
  sudo: true            # sudo 사용 가능 여부 — 기동 절차가 참조
dataRoot: /data/chainbench   # 설정 관리 루트. 노드 datadir 은 이 **하위**에 생성
```

- `portBases` 가 개별 포트가 아니라 **베이스**인 이유: 노드 하나는 포트 하나가 아니라
  묶음(P2P·Etcd·HTTP·WS·Auth·Metrics)을 쓴다. 베이스에서 묶음을 파생하는 계산은
  `portplan.Plan` 이 이미 갖고 있고, 베이스 간 간격 검증(`p2pStep>=2` 등)은 로더가 한다.
- `dataRoot` 는 노드 datadir 이 아니다 — 설정·산출물이 놓이는 서버 쪽 루트이고,
  노드별 datadir 은 `<dataRoot>/<label>/` 로 그 하위에 생긴다.
- 기존 v1 스키마(서버별 항목)는 읽기 호환으로 유지한다 — v1 항목 하나 = 풀의 (host 1 × 밴드 1).

### 2.2b Assign — 풀을 소비하는 결정적 할당

노드가 가용 IP 보다 많을 수 있다. 그때 **IP 를 재사용하되 포트베이스로 구분**한다.

할당 순서는 **IP 먼저, 그다음 포트베이스**다: node1 부터 순서대로 IP 목록을 소비하고,
IP 가 소진되면 포트베이스 인덱스를 올려 첫 IP 부터 다시 돈다.

```
hosts = [ip1..ip5], portBases = [8080, 8180, 8280, 8380], 노드 15개

node1  → (ip1, 8080)    node6  → (ip1, 8180)    node11 → (ip1, 8280)
node2  → (ip2, 8080)    node7  → (ip2, 8180)    node12 → (ip2, 8280)
node3  → (ip3, 8080)    node8  → (ip3, 8180)    ...
node4  → (ip4, 8080)    node9  → (ip4, 8180)    node15 → (ip5, 8280)
node5  → (ip5, 8080)    node10 → (ip5, 8180)
```

- **결정적이다**: 같은 풀 + 같은 노드 수 → 항상 같은 할당. 재실행이 다른 배치를 만들지 않는다.
- **초과는 오류다**: 노드 수 > |hosts| × |portBases| 면 몇 개가 부족한지 말하고 거부한다.
  조용히 겹치는 (ip, port) 를 만드는 것이 최악의 결과다.
- 같은 IP 의 두 노드는 포트베이스가 다르므로 (ip, port) 조합은 항상 유일하다 —
  `portplan.Validate` 가 최종 확인한다.
- 기존 `place.Allocator` 의 두 모드(LocalStepped·RemotePerHost)는 이 일반형의 특수한
  경우다: 로컬 = hosts 1개 × 베이스 N, 원격 호스트당 1노드 = hosts N × 베이스 1.
  **Assign 이 셋을 하나로 흡수**하고 `place` 는 그 구현이 된다.

**`StaticNodes` 가 공개키를 함수로 받는 이유**: enode = pubkey + host + port 인데 pubkey 는
keyring 의 것이다. netmap 이 keyring 을 import 하면 배치가 신원에 묶인다. 호출자(엔진·컴포즈)가
`func(Label) (pubkey string, ok bool)` 을 주입하면 두 소유자가 서로를 모른 채 조합된다.

### 2.3 흡수·이동 목록

| 지금 | 어디로 | 방식 |
|---|---|---|
| `portplan.Ports` | `netmap.Ports` | **이동**. `portplan` 은 계산 함수(`Plan`·`Validate`)만 남기고 타입은 netmap 이 소유 |
| `node.Endpoints` | `netmap.Ports` | **대체**. Etcd 누락 해소. `node.Node.Ports` 의 타입이 바뀐다 |
| `node.RoleEN` | 삭제 | `en` 은 `RoleEndpoint` 의 새 철자가 아니라 중복이었다 |
| `place.NodePlacement.Name` | `netmap.Label` | 명명된 타입으로 승격 (keyring.Label 과 **별개 타입** — 노드 라벨과 링 항목 라벨은 우연히 겹칠 뿐 같은 개념이 아니다) |
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

공유 방향은 레이어가 정한다: `testspec` 은 L3, `netmap` 은 L1 이므로 **netmap 이 어휘를
소유하고 DSL 이 import 한다.** (반대는 상향 의존 — A1 이 거부한다.)

```go
// netmap 이 소유
func NodeLabel(index int) Label      // "node1" — 지금 DSL 이 하드코딩하는 규칙
const RoleBP, RoleEN, RolePN Role    // DSL 의 role 선택자가 같은 상수를 참조
```

DSL 의 `on:"node1"`·역할 셀렉터가 이 함수·상수를 쓰게 되면, 라벨 규칙의 변경은 한 곳이고
DSL 은 따라온다.

### 2.6 체인별 차이 — 패밀리 포트 프로필

노드가 소비하는 포트 묶음은 패밀리마다 다르다. 이것이 F1(PortReservation)의 실체이고,
netmap 의 seam 으로 들어온다.

| | wbft 계열 | poa(wemix) |
|---|---|---|
| etcd | 안 씀 | **p2p+1 로 유도** — 베이스 간격의 이유 |
| pn 역할 | static-nodes 그래프로 표현 | **없음** — etcd 가 그 자리. 선언 시 오류 |
| 베이스 최소 간격 | 2 (p2p, 여유) | 2 이상 (p2p + etcd) |

```go
// 패밀리가 자기 요구를 선언하고, Assign/Validate 가 그것으로 검증한다
type PortProfile struct {
    NeedsEtcd bool     // 베이스 검증과 Ports 파생에 반영
    AllowsPN  bool     // false 면 pn 선언이 오류
}
```

현재 `serverset` 의 전역 `p2pStep>=2` 강제는 이 프로필로 옮겨진다 — wbft 만 있는 배치에서
과잉 제약을 걸지 않게 되고, F1 의 "poa 는 step 2 거부, wbft 는 허용" 게이트가 여기서 닫힌다.

---

## 3. 표면 — cmd 와 MCP

새 명령을 만들지 않는다. **`net allocate` 의 산출물이 netmap 이다** — 지금도 그 명령이
노드 테이블을 만들고 있고, 바뀌는 것은 산출물의 소유자와 표현이다.

| 표면 | 지금 | 후 |
|---|---|---|
| `net allocate` | 노드 테이블 생성(NodeState 평면) | 동일 명령, 산출 = netmap. 출력에 **역할·호스트·포트 전부**(etcd 포함) |
| `net allocate --peering` | (없음 — 항상 풀메시) | `mesh`(기본) \| `proxied`. poa+pn 은 오류 |
| `net status` / MCP `chainbench_net_status` | 포트 일부 표시 | netmap 조회 — 역방향(host:port→label) 포함 |
| MCP `chainbench_network_topology` | 런타임 피어 조회 | 유지. **계획된** 그래프는 allocate 출력이 답한다 |

keyring 때처럼 유스케이스는 `app` 에, 표면은 바인딩·렌더링만 (K8 선례).

---

## 4. 작업 단계

| # | 작업 | 게이트 |
|---|---|---|
| **NM1** | `core/netmap` 신설 — Role(bp/en/pn)·Ports(Etcd 포함)·Placement·Vocabulary·legacy 매핑 | 기존 topology/NodeState 를 **읽기 호환**으로 흡수 · 문자열 역할이 남지 않음 · DSL 의 `"node"+i` 하드코딩이 `netmap.NodeLabel` 로 |
| **NM1b** | Pool + Assign — 인벤토리 v2 스키마(hosts×portBases·sudo·dataRoot) + 결정적 할당 | §2.2b 예시(5 IP × 4 베이스 × 15노드)가 테이블 테스트로 · 초과는 부족 수를 말하며 거부 · `place` 2모드가 특수형으로 흡수 · v1 읽기 호환 |
| **NM2** | Peering 파생 + `StaticNodes` + PortProfile — mesh 는 **현행 argv 와 바이트 동일** | proxied: en 의 목록에 bp 가 없음 · poa+pn 오류 · 골든 비교 · `serverset` 전역 `p2pStep>=2` 가 패밀리 프로필로 (F1) |
| **NM3** | 조립 4곳 → netmap 소비 (engine·netcompose 먼저, upgrade·chainsetup 은 F4·F5 와) | 앞 2곳 전환 후 `net up` 라이브 재검증(stablenet·wbft) |
| **NM4** | 표면 — allocate 산출 강화 · `--peering` · 역방향 조회 | CLI/MCP 동일 출력 · A2 표 갱신 |
| **NM5** | 라벨 영속 — 워크스페이스에 Label 기록, 로그의 host:port 역추적 | `place.NodePlacement.Name` 이 버려지지 않음 (N7 의 원래 동기) |

**N 계열과의 관계**: NM1~5 = N7 + N0 + N0b. 완료 시 워크리스트의 세 항목이 닫히고,
N1(청사진 선언)은 netmap 을 산출 타입으로 삼는다.

---

## 5. 열린 질문

| # | 질문 | 기울기 |
|---|---|---|
| NM-a | `netmap.Label` 과 `keyring.Label` — 별개 타입 유지? | 별개. 노드 라벨(`node1`)과 링 항목(`faucet`)은 다른 공간 |
| NM-b | `node.Node.Ports` 타입 교체는 파급이 크다(25 fan-in). 단계적으로? | NM1 에서 alias 로 시작, NM3 에서 교체 |
| NM-c | proxied 에서 pn 이 없으면? | 오류 — mesh 로 조용히 강등하지 않는다 |
| NM-d | 역할별 IP 선호(예: bp 는 앞쪽 IP 고정)가 필요한가? | 당장은 아니오 — 순서 결정성으로 충분. 근거가 생기면 Assign 에 정책 주입 |
| NM-e | sudo 는 어디가 소비하나? | 기동 절차(driver/bringup). netmap 은 보관·전달만 — 실행하지 않는다 |
