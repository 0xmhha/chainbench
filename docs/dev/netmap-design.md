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
├─ Placement   Label → {Role, Host, Ports, DataDir}  (정방향)
│              Host:Port → Label                      (역방향, N7)
├─ Peering     역할에서 파생되는 그래프 (N0b)
│              mesh    — 현행과 동일 (기본)
│              proxied — bp↔pn↔en, en 은 bp 를 모른다
│              poa 에 pn 선언 시 **오류** (조용한 무시 금지)
└─ StaticNodes(label, pubkeyOf) → []enode
               "이 노드가 다이얼할 목록" — 4벌의 정책이 이 한 함수로
```

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
| **NM1** | `core/netmap` 신설 — Role(bp/en/pn)·Ports(Etcd 포함)·Placement·legacy 매핑 | 기존 topology/NodeState 를 **읽기 호환**으로 흡수 · 문자열 역할이 남지 않음 |
| **NM2** | Peering 파생 + `StaticNodes` — mesh 는 **현행 argv 와 바이트 동일** | proxied: en 의 목록에 bp 가 없음 · poa+pn 오류 · 골든 비교 |
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
