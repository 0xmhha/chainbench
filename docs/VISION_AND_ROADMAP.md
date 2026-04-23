# chainbench Vision & Roadmap

> **작성일**: 2026-04-20
> **최종 업데이트**: 2026-04-20 (§5.16 Sub-decisions S1~S8 확정 · §5.17 Go Network 모듈 구현 상세 추가 · §5.12/§6 Go 기반 재작성)
> **목적**: 프로젝트 비전을 토대로 현 상태를 진단하고, 다체인·로컬/원격 통합을 위한 아키텍처 방향과 단계별 로드맵을 확정한다.

---

## 1. 프로젝트 비전

1. **EVM 다체인 지원** — `go-stablenet`(현재), `go-wbft`, `go-wemix`, `go-ethereum`(확장)
2. **로컬 또는 원격에 체인 네트워크 구성**
3. **이미 구성된 체인 네트워크에 연결**(attach)
4. **체인 네트워크에 transaction 전송**
5. **체인 네트워크에서 log 수집**

---

## 2. 현 프로젝트 지원 기능 요약

### 2.1 CLI (`lib/cmd_*.sh`)
| 영역 | 커맨드 |
|---|---|
| 체인 수명주기 | `init`, `start`, `stop`, `restart`, `status`, `clean` |
| 노드 개별 제어 | `node stop/start/log/rpc <N>` |
| 테스트 | `test list/run`, `report` |
| 로그 분석 | `log timeline/anomaly/search` |
| 프로파일 | `profile list/show/create` |
| 로컬 설정 오버레이 | `config set/get/unset/list` |
| MCP | `mcp enable/disable/status` |
| 원격 노드 | `remote add/list/info/remove/select` |

### 2.2 프로파일 (`profiles/*.yaml`)
`default`, `minimal`, `bft-limit`, `large`, `hardfork-*` (4종), `regression`, `remote-example`. `inherits:` 기반 상속.

### 2.3 테스트 (총 200+개)
- `basic/` (7), `fault/` (6), `stress/` (2)
- `regression/` 7 섹션 + hardfork + z-layer2-e2e (≈165)
  - `a-ethereum` 31, `b-wbft` 12, `c-anzeon` 7, `d-fee-delegation` 4, `e-blacklist-authorized` 9, `f-system-contracts` 28, `g-api` 23, `h-hardfork` 40, `z-layer2-e2e` 5
- `unit/tests/` 20 (83 assertions)
- `remote/` 4

### 2.4 Layer 2 테스트 라이브러리 (`tests/lib/`)
`contract.sh`, `event.sh`, `chain_state.sh`, `tx_builder.sh`, `system_contracts.sh` + RPC 모듈군.

### 2.5 MCP 서버 (41 tools, TypeScript)
lifecycle, node, test, schema, log, remote, consensus, network, config, spec.

### 2.6 Chain Adapter (`lib/adapters/`)
- `stablenet.sh` — 실구현
- `wbft.sh`, `wemix.sh` — stub
- ethereum — 없음

---

## 3. 비전 Gap Analysis

| 비전 요구사항 | 현 상태 | Gap |
|---|---|---|
| EVM 다체인 지원 | 어댑터 뼈대만 존재. stablenet만 실구현 | 🔴 **핵심 Gap** |
| 로컬 체인 구성 | `cmd_init/start/stop` 완비 | ✅ |
| 원격 체인 구성 | `cmd_remote` + `remote_state.sh` + `tests/remote/` | 🟡 local/remote 이원화 |
| 기존 네트워크 attach | `remote add` 플로우 | 🟡 테스트가 로컬 전제 다수 |
| Transaction 전송 | `tx_builder`, `rpc_tx`, fee-delegation | ✅ |
| Log 수집 | `cmd_log`, `log_utils`, timeline/anomaly/search | ✅ (application log에 한정) |

**핵심 판단**:
> 다체인 추상화가 유일한 **구조적 병목**. 나머지 4개 비전은 70~90% 구현됨. 단, local/remote가 상위에서 분기(`@alias` vs 숫자) → 상위 레이어 blind 상태로 통합 필요.

`lib/cmd_start.sh`, `cmd_stop.sh`, `cmd_node.sh`에 `gstable` 하드코딩이 7곳 잔존 — 어댑터 경유로 강제하지 않으면 다체인 성립 불가.

---

## 4. 권장 로드맵 (Phase 순서)

```
Phase 1 (Adapter Contract) ──┐
                             ├──▶ Phase 4 (Chain impl: wbft → eth → wemix)
Phase 2 (Network Abstraction)┘
         │
         └──▶ Phase 3 (Test Matrix) ──▶ Phase 5 (Lib Neutralization) ──▶ Phase 6 (MCP)
```

### Phase 1 — 어댑터 인터페이스 정식화 (Foundation)
- 현 `stablenet.sh` 사용 함수 목록화 + 누락 함수 도출
- `cmd_start/stop/node`의 `gstable` 하드코딩 7곳 → 어댑터 함수로 추출
- `docs/ADAPTER_CONTRACT.md` 명세 작성
- `unit/tests/adapter-contract-<chain>.sh` 계약 테스트 하네스

### Phase 2 — Network 추상화 (Local/Remote/Attach 통합)
**→ §5 참조** (본 문서 핵심)

### Phase 3 — 테스트 호환성 매트릭스
- 프론트매터 확장: `chain_compat: [stablenet, wbft, wemix, ethereum]`
- `requires_capabilities: [process, admin_rpc]`
- `cmd_test.sh` 필터링/skip + 리포트

### Phase 4 — 체인 어댑터 구현
1. **go-wbft** — stablenet 가장 가까움
2. **go-ethereum** — 가장 넓은 레퍼런스
3. **go-wemix** — etcd-raft, 가장 이질적 (마지막)

### Phase 5 — Layer 2 라이브러리 체인 중립화
- 체인 고유 로직을 어댑터로 위임
- `tx_builder`가 `adapter_supports_tx_type` 호출
- `regression/lib/common.sh` 인라인 함수 마이그레이션

### Phase 6 — MCP 다체인 노출
- `chainbench_network_use/attach` 신규
- 기존 툴 응답에 `chain_type`, `network_mode` 주입

---

## 5. Network 추상화 설계 (Deep Dive)

### 5.1 설계 원칙

> **"상위 레이어는 local/remote를 모른다. 알게 해서도 안 된다."**

- **Obliviousness** — 상위는 모드 분기(`if local / elif remote`) 없음
- **Substitutability (Liskov)** — Provider가 바뀌어도 상위 동작 불변
- **Capability 기반** — "local/remote" 대신 "이 provider가 할 수 있는 것"으로 차이를 표현
- **Orthogonality** — chain type × location 두 축을 독립 (adapter × provider)
- **Per-node dispatch** — hybrid 네트워크(일부 local + 일부 remote) 대비

### 5.2 오퍼레이션 분류 — 어디서 공통, 어디서 갈라지는가

| 계층 | 오퍼레이션 예시 | 공통/차이 | 이유 |
|---|---|---|---|
| **L0 Protocol** | JSON-RPC call, eth_sendRawTransaction, eth_getLogs, eth_subscribe | ✅ **완전 공통** | RPC 프로토콜은 어디든 동일 |
| **L1 Chain semantics** | `istanbul_getValidators`, `wbft_*`, `wemix_*` | 🔶 **chain-specific** | adapter 책임 |
| **L2 Process** | start/stop/restart node, crash recovery | 🔴 **location-specific** | remote는 PID 없음 |
| **L3 Filesystem** | application log tail, datadir inspect | 🔴 **location-specific** | remote는 FS 없음 (SSH 예외) |
| **L4 Composite** | "node stop 후 validator 재구성 관찰" | ⚙️ **조합** | L1+L2 조합, 상위에서 조립 |

**핵심 관찰**: 비전의 5개 요구사항 중 **tx 전송·log 수집·RPC 질의는 모두 L0/L1**이다. 즉 **상위 통합이 가능한 표면이 전체의 80% 이상**을 차지한다. 차이는 L2/L3에 국한되며, **L2/L3는 capability gate로 제어 가능**하다.

### 5.3 계층 구조 제안

```
┌────────────────────────────────────────────────────────┐
│ L5 Commands / Tests / MCP                              │
│   (chainbench test run, tests/**, MCP handlers)        │
├────────────────────────────────────────────────────────┤
│ L4 Composite Operations (tx_send, log_subscribe, ...)  │
├════════════════════════════════════════════════════════┤  ← Network Abstraction boundary
│ L3 Network Facade (Network Abstraction Interface, 네이티브 코드)       │    (Q4: 일관 command
│   NetworkController · NodeHandle · EventBus            │     in / event out)
├════════════════════════════════════════════════════════┤
│ L2 Driver Dispatch (per-node, 네이티브 코드)           │
├──────────────────┬─────────────────────────────────────┤
│ LocalDriver      │ RemoteDriver                        │
│   subprocess →   │   HTTP/WS RPC client                │
│     cmd_*.sh     │   + auth (API key/JWT)              │
│     pids.json    │                                     │
│                  │ SSHRemoteDriver (future)            │
│                  │   SSH tunnel + shell exec           │
├──────────────────┴─────────────────────────────────────┤
│ L1 Chain Adapter (orthogonal axis)                     │
│   stablenet / wbft / wemix / ethereum                  │
├────────────────────────────────────────────────────────┤
│ L0 JSON-RPC (raw HTTP/WS)                              │
└────────────────────────────────────────────────────────┘
```

**두 직교 축**:
- 세로축 = **Location** (local / remote / ssh-remote / attached-ro) → Provider
- 가로축 = **Chain** (stablenet / wbft / wemix / ethereum) → Adapter

L3 Facade는 두 축을 직교적으로 조합한다. "stablenet × local", "ethereum × remote" 모두 같은 상위 API.

### 5.4 Provider Interface Contract

Provider = "이 노드에 대해 어떤 조작이 가능한가"를 제공하는 계약.

**필수 (모든 provider)**:
```bash
provider_name                               # "local" | "remote" | "ssh-remote"
provider_capabilities                       # "rpc ws process fs admin"  (space-sep)
provider_has_capability <cap>               # 0/1

provider_node_list                          # 노드 ID 목록 JSON
provider_node_http_url <node_id>
provider_node_ws_url <node_id>
provider_node_health <node_id>              # alive 체크 (포트 또는 ping)

provider_rpc_call <node_id> <method> <params_json> [timeout]
```

**선택 (capability-gated)**:
```bash
# CAP: process
provider_node_start <node_id>
provider_node_stop <node_id> [signal]
provider_node_restart <node_id>
provider_node_pid <node_id>

# CAP: fs
provider_node_datadir <node_id>
provider_node_logfile <node_id>
provider_node_tail_log <node_id> [lines]

# CAP: network-topology (optional, local only)
provider_network_partition <from_ids> <to_ids>
provider_network_heal
```

**호출 규약**:
- 선택 함수는 capability 없으면 **정의조차 되지 않는다**. 상위에서 `provider_has_capability process` 체크 후 호출.
- 에러 시 exit 2 = `NOT_SUPPORTED` (비정상 실패 1과 구분) — 상위가 "capability는 있다 했으나 런타임 사유로 실패"를 판정 가능.

### 5.5 Capability Negotiation

**Provider 시작 시 capability 선언**:
```bash
# lib/providers/local.sh
provider_capabilities() { printf 'rpc ws process fs admin debug network-topology\n'; }

# lib/providers/remote.sh
provider_capabilities() { printf 'rpc ws\n'; }

# lib/providers/ssh-remote.sh (future)
provider_capabilities() { printf 'rpc ws process fs\n'; }
```

**테스트가 요구사항 선언**:
```bash
#!/usr/bin/env bash
# tests/fault/node-crash.sh
# --- chainbench-meta ---
# description: Stop 1/4 validators and verify consensus continues
# requires_capabilities: [process]
# chain_compat: [stablenet, wbft]
# --- end-meta ---
```

**테스트 러너 게이트**:
```bash
cb_test_can_run() {
  local test_path=$1
  local active_net
  active_net=$(cb_network_active)
  local caps
  caps=$(cb_network_capabilities "$active_net")

  local required
  required=$(cb_test_meta_field "$test_path" requires_capabilities)

  for r in $required; do
    grep -qw "$r" <<<"$caps" || { echo "SKIP: $test_path (requires $r)"; return 1; }
  done
}
```

**UX 효과**: 사용자가 remote 네트워크를 선택하고 `test run all`을 치면, `requires_capabilities: [process]`인 fault 테스트는 자동 skip + 명확한 사유 리포트. 비전 3번("기존 네트워크 attach")이 테스트 실행과 자연스럽게 연결된다.

### 5.6 Network 개념 (L2/L3 경계)

Network = 노드 집합 + chain type + 노드별 provider 바인딩.

**데이터 모델** (`state/networks/<name>.json`):
```json
{
  "name": "my-local",
  "chain_type": "stablenet",
  "chain_id": 8283,
  "nodes": [
    {
      "id": "node1",
      "role": "validator",
      "provider": "local",
      "http": "http://127.0.0.1:8501",
      "ws":   "ws://127.0.0.1:9501",
      "provider_meta": { "pid_key": "node1" }
    }
  ]
}
```

**Hybrid 예시** (local 3 + remote 1):
```json
{
  "name": "mixed",
  "chain_type": "ethereum",
  "chain_id": 1337,
  "nodes": [
    { "id": "v1",  "provider": "local",  "http": "http://127.0.0.1:8545", ... },
    { "id": "v2",  "provider": "local",  "http": "http://127.0.0.1:8546", ... },
    { "id": "v3",  "provider": "local",  "http": "http://127.0.0.1:8547", ... },
    { "id": "rpc", "provider": "remote", "http": "https://devnet.example.com", ... }
  ]
}
```

→ 노드별 provider 바인딩. 상위는 `cb_net_rpc_call "v1" ...`와 `cb_net_rpc_call "rpc" ...`를 구분 없이 호출.

### 5.7 L3 Facade — 상위가 부르는 단 하나의 API

```bash
# lib/network.sh — L3 Facade (발췌)

cb_network_load <name>                    # → state/networks/<name>.json 로드 + per-node provider source
cb_network_active                         # 현재 활성 네트워크 이름
cb_network_capabilities <name>            # 모든 노드 provider capability의 교집합 출력

cb_net_rpc <node_id> <method> <params>    # 공통 (L0 위임)
cb_net_http_url <node_id>                 # 공통
cb_net_ws_url <node_id>                   # 공통
cb_net_node_health <node_id>              # 공통

cb_net_node_stop <node_id>                # CAP: process — 없으면 NOT_SUPPORTED
cb_net_node_start <node_id>               # CAP: process
cb_net_node_logfile <node_id>             # CAP: fs

cb_net_tx_send <from> <to> <value> ...    # L4 composite: adapter × provider 조합
cb_net_log_chain_timeline [filter]        # L0/L1 only → 공통
cb_net_log_node_tail <node_id> [lines]    # L3 fs → gated
```

### 5.8 Per-node Provider Dispatch (구현 기법)

Bash는 다형성이 없으므로 **함수 prefix 패턴**:

```bash
# lib/network.sh
_cb_node_provider_fn() {
  # $1 = node_id, $2 = op (rpc_call, node_stop, ...)
  local node_id=$1 op=$2
  local provider
  provider=$(cb_net_node_field "$node_id" provider)
  echo "provider_${provider}_${op}"
}

cb_net_rpc() {
  local node_id=$1; shift
  local fn; fn=$(_cb_node_provider_fn "$node_id" rpc_call)
  "$fn" "$node_id" "$@"
}
```

**Provider 구현 파일**:
```
lib/providers/
  _base.sh               # 공통 헬퍼
  local.sh               # provider_local_*
  remote.sh              # provider_remote_*
  ssh-remote.sh          # provider_ssh_remote_*  (future)
```

`cb_network_load`가 네트워크의 모든 노드가 사용하는 provider 집합을 수집하여 해당 파일들만 source. 미사용 provider는 로드 안 함.

### 5.9 Chain Adapter × Provider 직교 조합

두 축이 모두 필요한 오퍼레이션은 **L4 Composite**에서 조립.

예: `cb_net_validator_list(network)` — stablenet/wbft에서는 `istanbul_getValidators` RPC, ethereum에서는 clique/beacon 방식:

```bash
cb_net_validator_list() {
  local node_id=${1:-1}
  local method; method=$(adapter_consensus_validator_rpc_method)  # chain 축
  cb_net_rpc "$node_id" "$method" "[\"latest\"]"                   # location 축
}
```

상위는 체인도 위치도 모른다. Adapter와 Provider가 각자 맡은 축만 결정.

### 5.10 Log 수집 — 두 종류의 Log를 분리

현 `cmd_log`는 **application log**(stderr) 기반. 비전 5번 "log 수집"은 **chain log**(block events)도 포함해야 한다. Remote 네트워크에서 stderr 접근은 불가능하므로 둘을 명확히 분리한다.

| 종류 | 출처 | 공통성 | 커맨드 |
|---|---|---|---|
| **Chain log** | `eth_getLogs`, `eth_subscribe("logs"/"newHeads")` | ✅ 완전 공통 | `chainbench log chain {timeline,events,blocks}` |
| **Application log** | 노드 프로세스 stderr/stdout | 🔴 local + ssh-remote only | `chainbench log node {tail,search,anomaly} <N>` |

기존 `cmd_log timeline/anomaly/search`는 application log 기반이므로 **`log node`로 이동**, 신규 `log chain`을 추가. 테스트는 `requires_capabilities: [fs]`로 구분.

### 5.11 Transaction 전송 — 공통화 관점

비전 4번 "tx 전송"은 L0(RPC `eth_sendRawTransaction`)만 있으면 완전 공통. 현 `tests/lib/tx_builder.sh`는 이미 RPC 기반이므로 provider 추상화 위에서 그대로 동작한다. **추가 비용 없음**.

체인 고유 tx type (0x16 fee-delegation, EIP-7702 setcode 등)은 `adapter_supports_tx_type`로 체인 축에 격리. Provider와 무관.

### 5.12 마이그레이션 전략 (현 코드 → 신 구조)

기존 코드를 깨뜨리지 않고 점진 진화. **S1=Go 확정으로 Network Abstraction은 `network/` Go 모듈로 구현되며, bash CLI·MCP(TS)는 subprocess spawn으로 호출**.

| Step | 작업 | 하위호환 |
|---|---|---|
| **M0** | `network/` Go 모듈 초기화 (`go.mod`), 디렉토리 골격 생성 (§5.17.1). JSON Schema 단일 출처(`network/schema/*.json`) 확정 — command/event/network 3종 | — |
| **M1** | `network/internal/state` · `controller` — Network 객체 load/save/active. 기존 `state/pids.json`·`remote_state.json`을 **읽기만** 해서 Go 구조체로 변환. `network/cmd/chainbench-net network list` 구현 | ✅ 기존 state 그대로 |
| **M2** | `network/internal/drivers/local` · `remote` — LocalDriver는 초기엔 `os/exec`로 기존 `cmd_*.sh` 호출 (네이티브 Go 재구현은 후속). RemoteDriver는 `go-ethereum/ethclient` 기반 Go 클라이언트. `network/internal/wire` NDJSON 프로토콜 구현 | ✅ 쉘스크립트 재활용 |
| **M3** | bash CLI가 네트워크 바이너리를 spawn하여 호출 — `lib/network_client.sh` 추가 (stdin envelope 전송 + NDJSON 파싱). `rpc_client.sh`·`tests/lib/rpc.sh` 내부가 `cb_network_call`로 대체. 외부 시그니처(`rpc @alias` vs `rpc 1`) 유지 | ✅ 테스트 무수정 |
| **M4** | `cmd_start/stop/node`의 `gstable` 하드코딩 7곳을 Go adapter + driver로 이관. 쉘스크립트는 drive의 backing으로 유지되거나 단계적 Go 이식 | Phase 1과 병행 |
| **M5** | 신규 커맨드 `chainbench network list/use/attach/probe/info` — 네트워크 바이너리 직접 또는 bash 래퍼. 기존 `remote` 커맨드는 내부 Network Abstraction 사용 (CLI 표면 유지) | ✅ 기존 `remote add/list` 동작 |
| **M6** | 테스트 프론트매터에 `requires_capabilities` 점진 부여. 부여 안 된 테스트는 기본 `[process, fs]` (안전한 기본값) | ✅ 기존 테스트 동작 |
| **M7** | `current-remote` 파일 deprecated, `state/active-network` 로 이전. 마이그레이션 스크립트 제공 | ⚠️ 경고 출력 기간 후 제거 |
| **M8** | 보안 경계 검증 — `unit/tests/security-key-boundary.sh` + Go 테이블 테스트 `network/internal/signer/signer_test.go`. privatekey가 stdout/stderr/log 어디에도 노출되지 않음을 negative test로 확인 (§5.16 S4, §5.17.5) | — |
| **M9** | MCP 서버(TS)가 네트워크 바이너리 경유하도록 내부 리팩토링. `mcp-server/src/utils/exec.ts` 확장, 41 tool 중 RPC/lifecycle/node 관련 점진 이관 (§5.17.7) | ✅ MCP tool 시그니처 유지 |

### 5.13 예상 효과 (before / after)

**Before** — 테스트 코드:
```bash
if [[ -n "$CHAINBENCH_REMOTE" ]]; then
  rpc "@$CHAINBENCH_REMOTE" eth_blockNumber
else
  rpc 1 eth_blockNumber
fi
```

**After** — 테스트 코드:
```bash
cb_net_rpc 1 eth_blockNumber        # 또는 'endpoint', 'node1' 등 네트워크 정의에 따른 ID
```

활성 네트워크가 local이든 remote든 동일. 테스트는 "이 네트워크에 node1이 있다"만 알면 됨.

**Before** — fault 테스트 실행 시:
```
remote 네트워크에서 fault/node-crash.sh 실행 → 실패 (pid 없음)
```

**After**:
```
[SKIP] fault/node-crash.sh (requires capability: process, network 'devnet' provides: rpc ws)
```

### 5.14 확정된 설계 결정 (Resolved, 2026-04-20)

| # | 결정 | 근거 / 제약 |
|---|------|------------|
| **Q1** | **서명은 로컬(테스트 실행 머신)에서만 수행**. 서명된 tx만 RPC-URL로 전송. **remote/LLM/네트워크에 privatekey 전송 절대 금지** | 보안 원칙(non-negotiable). 로컬에 존재하는 키의 **주입 경로만** 설계 대상 — §5.16 S4 |
| **Q2** | **chain_type은 RPC probe로 자동 감지** | `eth_chainId` + chain-specific RPC 존재 여부 (예: `istanbul_*` → wbft/stablenet 계열, `wemix_*` → wemix). 사용자 명시 override 허용 (probe 실패/오탐 대비) |
| **Q3** | **다체인 · Hybrid 네트워크 v1부터 지원**. go-stablenet 외에 **go-wbft, go-wemix 동시 지원이 즉시 필요** | per-node provider dispatch로 비용 최소. `chain_type` 축이 adapter로 격리되어 있으면 신규 체인 추가는 adapter 구현만으로 가능 |
| **Q4** | **상위 레이어 = 네이티브 코드 Network Abstraction (Android HAL 영감)**. 일관된 command 입력 + event 출력. 하위 local/remote는 각 **드라이버 컴포넌트**로 분리. 기존 `cmd_init`, `cmd_start` 등 쉘스크립트는 **LocalDriver의 내부 구현**으로 재포지셔닝 | 쉘은 다형성·이벤트·타입 안전성 부족. Network Abstraction 경계 확립이 다체인/다로케이션 확장의 전제. §5.15 참조 |
| **Q5** | **WS 구독도 provider별 처리**. 체인별 RPC 방언 차이(`istanbul_*` 구독 등)가 WS에도 반영됨 | adapter(체인 축) × provider(위치 축) 모두 관여하는 L4 composite로 분류 |
| **Q6** | **원격 인증 지원 = API key / JWT / SSH(user+password)**. 3조합으로 대부분 케이스 커버 | `nodes[].auth` 스키마에 `type` discriminator + 각 type별 필드. SSH는 신규 `SSHRemoteDriver`의 필수 입력 |

**비전 5개 요구사항 vs 결정 매핑**:
- 비전 1 (다체인) ← Q3
- 비전 2 (로컬/원격 구성) ← Q4 (Network Abstraction + 드라이버 분리)
- 비전 3 (기존 네트워크 attach) ← Q2, Q6
- 비전 4 (tx 전송) ← Q1 (local sign + RPC transmit)
- 비전 5 (log 수집) ← §5.10 (chain log 공통, application log capability gated)

---

### 5.15 Network Abstraction 아키텍처 상세 (Q4 확장)

**영감 (Inspiration)**: Android HAL의 "상위는 하위 구현을 모르고 command-in / event-out 인터페이스로만 통신" 패턴에서 영감을 받음. 단, 본 프로젝트는 **하드웨어가 아닌 체인 네트워크**를 추상화하므로 `Hardware Abstraction Layer` 명칭 대신 **`Network Abstraction`** 으로 명명한다.

**chainbench 대응**:

```
┌──────────────────────────────────────────────────────────┐
│ Upper (상위)                                             │
│   CLI (bash) │ MCP Server │ Tests │ External integrators │
│        ↓ command            ↑ event                      │
├──────────────────────────────────────────────────────────┤
│ Network Abstraction Interface (네이티브 코드, 언어 중립 계약)            │
│   NetworkController / NodeHandle / EventBus              │
├──────────────────────────────────────────────────────────┤
│ Drivers (하위, 네이티브 코드)                            │
│ ┌──────────────┬──────────────┬──────────────┐           │
│ │ LocalDriver  │ RemoteDriver │ SSHRemote    │           │
│ │  subprocess  │  HTTP/WS     │  (future)    │           │
│ │  → cmd_*.sh  │   RPC client │              │           │
│ │  → pids.json │   + auth     │              │           │
│ └──────────────┴──────────────┴──────────────┘           │
├──────────────────────────────────────────────────────────┤
│ Low-level assets (기존 자산, 드라이버 내부로 이동)       │
│   lib/cmd_*.sh · lib/adapters/*.sh · tests/lib/*.sh      │
└──────────────────────────────────────────────────────────┘
```

**Network Abstraction 구현 언어**: Go (S1 확정). `go-ethereum` 패키지 직접 import로 서명·RLP·ABI 재사용. §5.17 참조.

**Network Abstraction 인터페이스 (언어 중립 계약, command in)**:
```
network.load(name)           → Network
network.probe(rpc_url)       → { chain_type, chain_id }    # Q2
network.capabilities()       → Set<Capability>
node.rpc(method, params)     → Result                       # 공통
node.start() / stop() / restart()          # CAP: process
node.tail_log(lines)                       # CAP: fs
tx.send(signed_raw)          → tx_hash                      # Q1: signed만 입력
subscription.open(filter)    → stream_id                    # Q5
```

**Network Abstraction 인터페이스 (event out, stream)**:
```
event: node.started          { node_id, pid?, ts }
event: node.stopped          { node_id, exit_code?, reason, ts }
event: node.health.changed   { node_id, from, to, ts }
event: chain.block           { height, hash, miner, ts }
event: chain.tx              { tx_hash, status, block, ts }
event: chain.log             { contract, topics, data, block }
event: network.quorum        { healthy_node_ids, ok: bool }
event: error                 { source, code, message, recoverable }
```

이벤트는 상위에서 pub/sub 또는 async stream으로 수신. 테스트는 **"tx 전송 후 N초 이내 `chain.tx` 수신 + status=success"** 같은 **선언적 검증**을 `assert_event` 헬퍼로 작성 가능.

**쉘스크립트 포지셔닝**:
- `lib/cmd_init.sh`, `cmd_start.sh`, `cmd_stop.sh` 등은 **유지하되 `LocalDriver`의 내부 구현**으로 재분류
- 외부 호출은 금지, 네트워크 바이너리 경유만 허용
- `chainbench init` 등 CLI 표면은 유지 (bash 래퍼가 네트워크 바이너리에 command 전송)

**기존 bash provider 설계(§5.4, §5.8) 위치 재정의**:
- 당초 제안한 `lib/providers/{local,remote}.sh` (bash)는 **과도기 래퍼**로 격하
- 최종 상태는 **네이티브 Network Abstraction 내부의 Driver 클래스/모듈**
- §5.4의 함수 목록은 언어 중립 **Network Abstraction 인터페이스 계약**으로 승격 (bash는 일 구현)

---

### 5.16 Sub-decisions 확정 (Resolved, 2026-04-20)

Q1/Q4/Q6 확정에서 파생된 하위 결정 S1~S8 전부 확정:

| # | 항목 | 확정 | 근거 / 비고 |
|---|------|------|------------|
| **S1** | Network Abstraction 구현 언어/런타임 | **Go** | 체인 생태계(go-stablenet/go-wbft/go-wemix)와 언어 통일. `go-ethereum` 패키지 직접 import로 tx 서명·RLP·ABI 재사용. 정적 바이너리 배포 용이. MCP(TS)와는 spawn IPC로 연결. §5.17 참조 |
| **S2** | CLI ↔ Network Abstraction IPC | **매 호출마다 spawn** | daemon 운용 복잡도 회피. Go 시작 시간 ~10ms 수용 가능. **필수 고려사항**: (1) 메모리 관리 (프로세스 수명 ≤ 호출 수명) (2) stdin 명령 전달 · stdout 응답 스트림 프로토콜 설계 (3) 구조화 로깅 별도 채널(stderr 또는 파일). §5.17.3 참조 |
| **S3** | 이벤트 버스 구현 | **(a)+(b) 조합** | Go 내부: `chan Event` 기반 pub/sub (EventEmitter 대응). 외부 노출: stdout에 NDJSON stream. 상위(bash/MCP)가 라인 단위 파싱 |
| **S4** | 서명 키 주입 메커니즘 | **(a)+(b) 조합** | 환경변수 `CHAINBENCH_SIGNER_<ALIAS>_KEY` + keystore 파일(`CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE` + 패스워드). 둘 다 Network Abstraction Go 프로세스 내부에서 서명, **key material은 stdout/stderr/log 어디에도 출력 금지**. §5.17.5 참조 |
| **S5** | 키 주입 시점 | **테스트 시작 시** | 네트워크 정의에는 signer alias만. 실제 key material은 테스트 세션 초기화(네트워크 바이너리 spawn의 env/stdin)로 주입. 영속 파일 저장 금지 |
| **S6** | SSH 자격증명 저장 | **세션 prompt** | 평문 파일 저장 금지. 자동화 필요 시 OS 키체인 연동(후속). SSH 세션은 Network 프로세스 수명 내 메모리에만 상주 |
| **S7** | chain_type probe 트리거 | **자동 + 수동** | `network attach <url>` 자동 probe + `chainbench network probe <url>` 수동 커맨드. probe 실패 시 사용자 override 가능 |
| **S8** | Network Abstraction 계약 스키마 관리 | **JSON Schema** | 언어 중립 유지. Go는 `go generate` + `jsonschema`로 struct 생성. TS(MCP) 측도 `json-schema-to-typescript`로 타입 동기화. schema 파일은 `network/schema/*.json` 단일 출처 |

**특별 강조 — S4/S5 (보안, non-negotiable)**:
- LLM 프롬프트, 로그, 텔레메트리, 네트워크 응답 어디에도 privatekey 노출 금지
- Go Network Abstraction이 서명 함수(`signer.Sign`)를 제공하되, 반환은 **signed raw tx bytes**만. Key material은 반환/로깅/전송 어디에도 포함 안 됨
- Go 특성상 `[]byte` 키는 사용 후 즉시 `crypto/subtle.ConstantTimeCompare` 류 zeroing 권장 (best-effort)
- 키 주입 경로 negative test: `unit/tests/security-key-boundary.sh` — privatekey 문자열이 네트워크 바이너리 stdout/stderr/log 어디에도 나타나지 않음을 grep 기반 검증

---

### 5.17 Go Network 모듈 구현 상세 (S1~S8 확정 기반)

#### 5.17.1 프로젝트 구조

`chainbench` 레포 루트에 Go 모듈 `network/` 신설:

```
network/                            # 독립 Go 모듈 (go.mod)
├── cmd/chainbench-net/main.go      # 바이너리 entry (cobra CLI)
├── internal/
│   ├── controller/                 # NetworkController (L3 Facade)
│   ├── node/                       # NodeHandle
│   ├── events/                     # 이벤트 타입 + EventBus(chan 기반)
│   ├── drivers/
│   │   ├── local/                  # LocalDriver: exec cmd_*.sh 또는 native Go
│   │   ├── remote/                 # RemoteDriver: HTTP/WS
│   │   └── sshremote/              # 후속
│   ├── adapters/
│   │   ├── stablenet/              # stablenet 어댑터 (Go)
│   │   ├── wbft/
│   │   ├── wemix/
│   │   └── ethereum/
│   ├── probe/                      # chain_type probe (S7)
│   ├── signer/                     # 서명 경계 (S4/S5) — 격리 패키지
│   ├── rpc/                        # JSON-RPC client (ethclient wrapper)
│   ├── state/                      # state/networks/*.json 읽기/쓰기
│   └── wire/                       # stdin/stdout 프로토콜 (NDJSON)
├── schema/                         # JSON Schema (S8 단일 출처)
│   ├── command.json
│   ├── event.json
│   └── network.json
├── pkg/                            # 외부 공개 타입 (필요 시)
└── go.mod
```

**배치 원칙**:
- `internal/` = 외부 import 금지 (Go 관례)
- `signer/`는 별도 패키지로 격리 — 다른 패키지가 key material에 접근 불가능하도록 캡슐화
- 어댑터는 adapter interface를 구현하는 독립 패키지, 신규 체인 추가 = 신규 디렉토리 추가

#### 5.17.2 빌드 · 배포

- `go build -o bin/chainbench-net ./network/cmd/chainbench-net` → 정적 바이너리
- `install.sh` 확장: Go 1.22+ 확인 → 빌드 → `$HOME/.chainbench/bin/chainbench-net` 배치
- 크로스 컴파일 지원 (`darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`)
- 버전 정보 embed: `-ldflags "-X main.version=..."`

#### 5.17.3 Spawn-per-call 프로토콜 (S2 상세)

사용자가 S2에서 명시한 **필수 고려사항 3가지** 구현 방안:

**(1) 메모리 관리**
- 각 spawn은 독립 프로세스 → GC 경계 = 프로세스 경계. 호출 간 메모리 공유 없음
- 호출별 상태는 `state/networks/`, `state/active-network.json` 파일에 영속화
- 장기 실행 케이스(subscription, node tail -f)는 spawn 프로세스가 **호출 수명 동안 살아있음** — stdout 스트림을 유지하며 상위가 프로세스를 종료(SIGTERM)할 때까지 유지
- 누수 방지: Go의 `context.Context`를 최상위에서 생성, 모든 goroutine에 전파. 프로세스 종료 시 context cancel → 리소스 해제

**(2) Command 전달 · Response 수신**

```
Invocation:   chainbench-net <subcommand> [--flags]  [stdin: JSON command envelope]
Stdin  (옵션): { "command": "node.rpc", "args": {...}, "env": {...} }
Stdout (항상): NDJSON 스트림 — 한 줄당 1 JSON
Stderr (항상): structured log (slog JSON)
Exit code: 0 = success, 2 = NOT_SUPPORTED, 1 = generic error, 3 = protocol error
```

**Stdout NDJSON 스키마** (schema/event.json 으로 정의):
```
{"type":"event",   "name":"node.started", "data":{...}, "ts":"..."}
{"type":"event",   "name":"chain.block",  "data":{...}}
{"type":"progress","step":"init","done":2,"total":4}              # 진행률
{"type":"result",  "ok":true, "data":{...}}                        # 마지막 줄 (터미네이터)
{"type":"result",  "ok":false,"error":{"code":"NOT_SUPPORTED",...}}
```

**규약**:
- 모든 호출은 정확히 하나의 `type=result`로 종료 (성공/실패 무관). 상위는 `result`를 만나면 파싱 종료
- `event`/`progress`는 0개 이상. 순서 보장 (단일 stdout 파이프)
- 바이너리 무결성: 각 JSON은 독립 줄, 버퍼링 flush는 `json.Encoder` + `os.Stdout.Sync()` 조합

**상위(bash) 파싱 예**:
```bash
chainbench-net network load my-local <<<'{"command":"load","args":{"name":"my-local"}}' \
  | while IFS= read -r line; do
      case "$(jq -r .type <<<"$line")" in
        event)    handle_event "$line" ;;
        progress) update_progress "$line" ;;
        result)   echo "$line"; break ;;
      esac
    done
```

**(3) Logging 별도 채널**
- **stdout = 이벤트/결과 전용 (프로그램 consumption)**
- **stderr = 구조화 로그 전용 (사람/운영 consumption)** — slog JSON
- 로그 파일 redirect: `CHAINBENCH_NET_LOG=/path/to/log.jsonl` env. 미설정 시 stderr
- 로그 레벨: `CHAINBENCH_NET_LOG_LEVEL=debug|info|warn|error` (기본 info)
- slog attr에 key material 포함 금지 — `signer` 패키지가 커스텀 `slog.LogValuer` 구현, 어떤 경우도 `"***"` redact 반환

#### 5.17.4 이벤트 버스 (S3 Go 구현)

```go
// internal/events/bus.go
type Event struct {
    Type string          // "node.started", "chain.block" ...
    Data json.RawMessage
    TS   time.Time
}

type Bus struct {
    subs []chan<- Event  // in-process subscribers
    out  *json.Encoder   // stdout NDJSON writer (외부 노출)
    mu   sync.RWMutex
}

func (b *Bus) Publish(ev Event) { /* fan-out to subs + encode to stdout */ }
func (b *Bus) Subscribe() <-chan Event { /* in-process */ }
```

- 내부 구독자(adapter, driver 간 통신): channel
- 외부 구독자(bash/MCP): stdout NDJSON 라인
- 두 경로는 동일 `Publish` 호출로 fan-out

#### 5.17.5 서명 경계 (S4/S5 보안 설계)

```go
// internal/signer/signer.go — 외부 공개 API 최소화
package signer

type SignerAlias string  // 네트워크 정의에 저장되는 alias

type Signer interface {
    Sign(unsignedTx []byte) (signedRaw []byte, err error)
    Address() common.Address
}

// 공개 API: alias로만 조회. key material 반환 경로 없음
func Load(alias SignerAlias) (Signer, error)  // env 또는 keystore 자동 선택
```

**계약**:
- `Signer`는 `Sign` 외 어떤 메서드도 제공하지 않음 (키 export 불가)
- `Load`는 프로세스 시작 시 env 스캔 (`CHAINBENCH_SIGNER_*_KEY`, `CHAINBENCH_SIGNER_*_KEYSTORE`) — 그 외 경로 없음
- `Signer` 인스턴스는 프로세스 메모리에만 존재. 파일/네트워크/stdout 이동 없음
- 프로세스 종료 시 GC + best-effort zeroing

**주입 경로 (S5 = 테스트 시작 시)**:
```
test session init → 네트워크 바이너리 spawn with env CHAINBENCH_SIGNER_alice_KEY=...
                 → signer.Load("alice") → in-memory Signer
                 → tx 전송 시 signer.Sign(unsigned) 호출
                 → 네트워크 바이너리 종료 시 프로세스와 함께 key 소멸
```

**금지 사항 (negative test로 검증)**:
- `fmt.Printf`, `log.*`, `slog.*` 등 어떤 출력 경로에도 key bytes 등장 금지
- Panic stack trace, error message에도 포함 금지 (Signer가 error wrap 시 redact)
- 테스트: `network/cmd/chainbench-net` 을 privatekey 주입하여 실행 → stdout/stderr/log 전체를 grep → 키의 hex/base64 부분 문자열 발견 시 FAIL

#### 5.17.6 Subscription / 장기 실행 호출 (S2 edge case)

spawn-per-call이지만 `subscription.open`이나 `log node tail --follow`는 수명이 길다:

- 호출자(bash/MCP)가 프로세스를 계속 살려두고 stdout 스트림 소비
- 네트워크 바이너리는 graceful shutdown: SIGTERM 수신 시 구독 해제 + 정리 + `{"type":"result","ok":true}` 출력 후 종료
- 다중 구독 동시 필요 시 별개 프로세스 병렬 spawn (프로세스당 1 subscription이 단순)

#### 5.17.7 MCP 서버(TypeScript) 연동

MCP 서버도 네트워크 바이너리를 spawn하여 사용:
- `mcp-server/src/utils/exec.ts` 확장 — `chainbench-net` 실행 + NDJSON 파싱 헬퍼
- 기존 41 tool 중 RPC/lifecycle/node 관련은 점진적으로 네트워크 바이너리 경유로 전환
- bash CLI와 동일 프로토콜 → 중복 로직 제거

---

## 6. 즉시 착수 가능한 작업

Phase 1과 Phase 2 병렬 진행을 가정한 초기 3 스프린트 예시:

**Sprint 1 — Go 모듈 + 인벤토리 + 스키마**
- [ ] `network/` Go 모듈 초기화 (`go.mod`, Go 1.22+, cobra CLI)
- [ ] `network/schema/` JSON Schema 3종 초안 — command.json / event.json / network.json (S8)
- [ ] Go struct 생성 파이프라인 (`go generate` + `jsonschema`)
- [ ] `lib/adapters/stablenet.sh` 사용 함수 전수 목록화 → `docs/ADAPTER_CONTRACT.md` 초안
- [ ] `cmd_start/stop/node` 하드코딩 7곳 위치·조건 정리
- [ ] 이벤트 카탈로그 확정 (§5.15) — event.json에 반영

**Sprint 2 — Network Abstraction 뼈대 + Wire Protocol + LocalDriver (exec 래핑)**
- [ ] `network/internal/{controller,node,events,wire,state}` 기본 구현
- [ ] `network/cmd/chainbench-net` entry — `network list` 첫 커맨드
- [ ] NDJSON stdout 스트림 + slog stderr 로깅 프로토콜 (§5.17.3)
- [ ] `drivers/local` — 초기엔 `os/exec`로 기존 `cmd_*.sh` 호출
- [ ] bash 클라이언트 `lib/network_client.sh` — spawn + NDJSON 파서
- [ ] Unit test: Go 테이블 테스트 + bash `unit/tests/network-wire-protocol.sh`

**Sprint 3 — RemoteDriver + chain_type probe + 첫 체인 확장**
- [ ] `drivers/remote` — `go-ethereum/ethclient` 기반. API key / JWT 인증 (S6)
- [x] `probe` 패키지 + `network probe <url>` 커맨드 (S7 자동+수동) — Sprint 3a 완료 (2026-04-23)
- [ ] `adapters/stablenet` Go 포팅 — 기존 `adapter-contract-stablenet.sh` 계약 테스트 통과
- [ ] `adapters/wbft` 실구현 시작

**Sprint 4 — 서명 경계 + 보안 검증 (S4/S5)**
- [ ] `network/internal/signer` — env + keystore 주입 경로 (§5.17.5)
- [ ] `SignerAlias` 타입 + `Signer` 인터페이스 (key export 불가 보장)
- [ ] Go 테스트: `signer_test.go` — key material이 error/log에 leak되지 않음
- [ ] bash 테스트: `unit/tests/security-key-boundary.sh` — 네트워크 바이너리 실행 후 stdout/stderr/log grep
- [ ] 문서화 — 키 관리 정책 (`docs/SECURITY_KEY_HANDLING.md`)

**Sprint 5 — Capability gate + Hybrid 네트워크 + MCP 이관 시작**
- [ ] 테스트 프론트매터 `requires_capabilities` 점진 부여 (§5.5)
- [ ] `drivers/sshremote` 설계 초안 (Q6, S6 세션 prompt)
- [ ] Hybrid 네트워크 샘플 프로파일 (`profiles/hybrid-example.yaml`)
- [ ] MCP (`mcp-server/src/utils/exec.ts`) 확장 — 첫 네트워크 바이너리 경유 tool 1~2개 전환 (§5.17.7)

각 스프린트 완료 시 `/milestone` 플로우로 delta-log · rolling-summary 업데이트.

---

## 7. 참고

- 관련 기존 문서
  - `docs/chainbench-test-system-design.md` — 테스트 시스템 설계
  - `docs/REMAINING_TASKS.md` — 완료/보류 작업 현황
- 관련 코드 진입점
  - `lib/chain_adapter.sh`, `lib/adapters/*` — 어댑터 계층
  - `lib/rpc_client.sh`, `tests/lib/rpc.sh` — 현 local/remote 분기 초기 형태
  - `lib/cmd_remote.sh`, `lib/remote_state.sh` — 원격 연결 관리
  - `tests/lib/*.sh` — Layer 2 테스트 라이브러리 (체인 중립화 대상)
