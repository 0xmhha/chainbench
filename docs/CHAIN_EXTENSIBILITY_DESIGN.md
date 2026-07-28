# Chain Extensibility Design — 다체인 지원 구조 개편

> ⚠️ **부분 대체(2026-07-24)**: 이 문서의 어댑터/매니페스트(§3.1–3.4) 설계는 유효하나,
> 전면 Go 재설계 + 3단계 파이프라인 + 계정/faucet/대시보드 요구가 추가되어
> **상위 문서 `docs/CHAINBENCH_GO_REDESIGN.md`가 전체 아키텍처의 SSoT**다. 이 문서는 그 안의
> "체인 플러그인 축" 상세로 참조된다.
>
> 작성일: 2026-07-24
> 상태: **설계 확정(방향) — 구현 미착수**. Phase 별 착수 시 이 문서를 SSoT 로 갱신.
> 목적: `go-stablenet`(현재) 위에 `go-wbft`, `go-wemix` 를 **저비용으로 추가**할 수 있는
> 확장 구조를 확정한다. 여러 체인을 하나의 chainbench 표면(CLI/MCP)에서 blind 하게 다룬다.
> 관련(아카이브): `docs/legacy/VISION_AND_ROADMAP.md` §3·§5.17 · `docs/legacy/ADAPTER_CONTRACT.md` ·
> `docs/legacy/HARDCODING_AUDIT.md` · `docs/legacy/REFACTORING_PLAN.md` §0
>
> **대상 체인 소스**:
> - `/Users/wm-it-25_0220/Work/github/chain/go-stablenet` (gstable, WBFT/anzeon)
> - `/Users/wm-it-25_0220/Work/github/chain/go-wbft` (gwbft, WBFT/croissant)
> - `/Users/wm-it-25_0220/Work/github/chain/go-wemix` (gwemix, etcd 기반)

---

## 0. 확정된 결정 (2026-07-24 검토 완료)

| # | 결정 사항 | 확정 |
|---|---|---|
| D1 | genesis 생성 전략 | **하이브리드** — stablenet/wbft = 체인별 소유 템플릿(전략 B), wemix = 네이티브 도구/별도 트랙(전략 A) |
| D2 | 어댑터 SSOT | **Go 로 단일화** — `network/internal/adapters` 가 유일 소유자, `lib/adapters/*.sh`(Python 중복) 폐기 |
| D3 | 확장 우선순위 | **wbft 먼저, wemix 후순위** |
| D4 | 산출물 | 이 설계 문서 (`docs/CHAIN_EXTENSIBILITY_DESIGN.md`) |

---

## 1. 현재 구조 (As-Is)

### 1.1 레이어 맵

chainbench 는 **3 표면 + 1 기능 코어**로 층화되어 있으나, 기능 코어가 레거시 bash 트랙과
신규 Go 트랙으로 **이원화된 채 마이그레이션 중**이다 (`REFACTORING_PLAN.md` §0).

```
표면 A (사람): bash CLI  chainbench.sh → lib/cmd_*.sh
표면 B (에이전트): MCP 서버  mcp-server/src/tools/*.ts
        │  (lifecycle reroute 완료: init/start/stop/restart/clean → callWire)
        ▼
기능 코어 (이원화):
  [레거시 bash]  lib/cmd_*.sh + lib/adapters/*.sh (Python+template)
  [신규 Go]      network/chainbench-net (NDJSON wire) + internal/{adapters,drivers,probe,...}
        │
        ▼
  실제 체인 바이너리 (gstable / gwbft / gwemix)
```

### 1.2 체인 추상화 축 — 중복 정의

| 트랙 | 위치 | 계약 | stablenet | wbft | wemix |
|---|---|---|---|---|---|
| bash | `lib/adapters/*.sh` (+ `chain_adapter.sh`) | 함수 4~5개 | ✅ 실구현 | 🔴 stub | 🔴 stub |
| Go | `network/internal/adapters/{spec,stablenet,wbft,wemix}` | 인터페이스 6 메서드 | ✅ golden-pin | 🟡 skeleton | 🟡 skeleton |

두 계약의 메서드 집합이 다르며(bash 4 vs Go 6), 새 체인은 **두 번** 구현해야 한다. → D2 로 해소.

### 1.3 다체인을 막는 구조적 병목 (분석 근거 포함)

1. **`gstable` 하드코딩 9곳** — `cmd_init/start/stop/node.sh` (`HARDCODING_AUDIT.md`). `cmd_init.sh:198`
   의 `"${BINARY}" init` 는 geth 계열 공통이나 어댑터 미경유.
2. **어댑터 함수가 선언만 되고 실경로 미사용** — `adapter_extra_start_flags`,
   `adapter_consensus_rpc_namespace`, `adapter_binary_name` 은 **유닛 테스트에서만** 호출된다.
   `cmd_start.sh:177-193` 은 launch 플래그(`--allow-insecure-unlock … --mine`)를 **인라인 하드코딩**하고
   `cmd_start.sh` 는 `chain_adapter.sh` 를 **source 조차 안 한다**. → 어댑터가 start 경로를 실제로 제어 못 함.
3. **genesis 자체 재구현** — `templates/genesis.template.json` 이 `config.anzeon.{wbft,init,systemContracts}`
   구조로 **stablenet 전용 박제**. 세 체인의 genesis 구조가 근본적으로 다름:
   - stablenet: `config.anzeon.{wbft, init, systemContracts}` + `boho` 하드포크
   - wbft: `config.croissant.{WBFT, Init, GovContracts}` + `pangyo/applepie/brioche/croissant` + `Transitions[]` + `Brioche` 반감기
   - wemix: WBFT 아님. **etcd 임베디드 클러스터**(`go-wemix/wemix/admin.go`) 멤버십 + governance 배포(`wemix/bind`)
   → 단일 placeholder 템플릿으로 못 덮음.
4. **네이티브 genesis 도구 이미 존재** — `go-stablenet/cmd/genesis_generator`,
   `go-wbft/cmd/genesis_generator`, `gwemix wemix genesis`. chainbench 재구현은 드리프트 부채.
5. **consensus RPC 네임스페이스 분기** — stablenet/wbft=`istanbul_*`, wemix=`wemix_*`.
   `mcp-server/src/tools/consensus.ts` 가 `istanbul_getValidators` 하드코딩 → consensus 표면이 wemix 미지원.
6. **preset 키가 stablenet 전용** — `keys/preset/metadata.json` 에 BLS 공개키(WBFT 요구). wemix 키 모델 상이.
7. **프로파일이 체인 중립적이지 않음** — `profiles/default.yaml` 이 `genesis.overrides.wbft`, `anzeon.init`,
   stablenet gov 컨트랙트를 직접 담음.
8. **테스트가 체인 의미론 전제** — `tests/regression/{b-wbft,c-anzeon,d-fee-delegation}`.
   `chain_compat` 프론트매터는 2개 파일에만 존재하고 러너 미배선. (단 `requires_capabilities` 게이트는 동작 — 토대 존재.)

---

## 2. 설계 목표

1. 새 체인 추가 = **선언 데이터 + 얇은 어댑터 1개** (두 번 구현 금지, SSOT 단일화)
2. genesis/config 생성의 **드리프트 제거**
3. 상위 표면(CLI/MCP)이 체인 종류에 **blind** — 분기는 어댑터 내부로
4. 모드 (B) 사람 사용성 · 모드 (A) 구조화 출력 **동시 유지** (VISION §1.1)
5. wemix 의 이질성(etcd)이 나머지 둘의 진행을 **막지 않게 격리**

---

## 3. 제안 아키텍처 (To-Be)

### 3.1 어댑터 SSOT 를 Go 로 단일화 (D2)

lifecycle reroute 완료(init/start/stop/restart/clean → `callWire`)를 전제로, 어댑터의 유일 소유자를
**Go `network/internal/adapters`** 로 확정한다. `lib/adapters/*.sh` 의 Python 중복은 **폐기**하고,
bash `cmd_*.sh` 는 wire 핸들러 호출만 담당(체인 지식 0). → 병목 #1·#2 가 함께 해소.

> ⚠️ 폐기 순서 주의: `REFACTORING_PLAN.md` §0 은 M4 까지 bash/Go 어댑터 공존을 "의도적 과도기"로 규정했다.
> D2 는 그 종착점(M4 이후)을 앞당겨 확정하는 것이므로, **Phase 1 에서 genesis/toml 의 Go 경로가 실제로
> init 을 담당하게 된 뒤**에 bash 어댑터를 제거한다(골든 테스트로 회귀 방지).

### 3.2 "선언(Manifest) + 행위(Adapter)" 2 분할

새 체인 비용의 대부분을 **선언 데이터**로 내리는 것이 핵심이다.

**(1) ChainManifest — 정적 사실 (데이터):** 체인당 1 파일, 스키마 검증.

```jsonc
// network/chains/wbft.json
{
  "id": "wbft",
  "binary": "gwbft",
  "build": { "repo": "go-wbft", "make_target": "gwbft" },
  "consensus": {
    "family": "wbft",                 // wbft | wemix-etcd
    "rpc_namespace": "istanbul",
    "validators_method": "istanbul_getValidators"
  },
  "keys":    { "model": "bls", "preset": "keys/preset/wbft" },
  "genesis": {
    "strategy": "template",           // template | native-tool
    "template": "templates/genesis.wbft.json",
    "engine_field": "croissant",
    "hardforks": ["pangyo","applepie","brioche","croissant"]
  },
  "tx_types": ["0x16"],
  "probe":    { "method": "istanbul_getValidators", "chain_ids": null },
  "capabilities": ["process","rpc","ws","consensus"]
}
```

이 한 파일이 병목 #1/#5/#6/#7/#8 을 데이터화한다. `probe/signatures.go` 의 시그니처,
`handlers_node_tx_fee_delegation.go` 의 `feeDelegationAllowedChains` 하드코딩,
`consensus.ts` 네임스페이스, preset 키 경로가 전부 여기서 **파생**된다.

**(2) Adapter 인터페이스 — 매니페스트로 안 되는 행위만:**

```go
type Adapter interface {
    Manifest() ChainManifest
    GenerateGenesis(ctx context.Context, in GenesisInput) error  // strategy dispatch
    GenerateConfig(ctx context.Context, in ConfigInput) error    // toml 등
    StartFlags(role Role) []string
    // namespace / tx_types / binary 등 정적값은 Manifest() 파생 → 메서드 축소
}
```

기존 `spec.Adapter`(6 메서드)에서 정적 게터(`ConsensusRpcNamespace`, `SupportedTxTypes`)를
`Manifest()` 파생으로 흡수하여 인터페이스 표면을 줄인다.

### 3.3 genesis 생성 — 하이브리드 (D1)

| 전략 | 대상 | 방식 | 트레이드오프 |
|---|---|---|---|
| **B: 체인별 소유 템플릿** | **stablenet, wbft** | `templates/genesis.<chain>.json` + 어댑터 채우기 | 재현성/preset-키 모델 유지, 결정적. 체인 진화 시 재구현 부채(관리) |
| **A: 네이티브 도구 위임** | **wemix (+ 향후 옵션)** | 각 프로젝트 `genesis_generator`/`gwemix wemix genesis` shell-out | 드리프트 0. preset-키/재현성 모델과 충돌 가능 → wemix 별도 트랙에서 흡수 |

근거: **wbft 은 stablenet 의 near-clone**(둘 다 WBFT/BLS/preset 공유)이라 전략 B 로 저비용 복제 가능.
현행 `templates/genesis.template.json` → `templates/genesis.stablenet.json` 으로 이름 확정 후
`genesis.wbft.json`(croissant 엔진 필드 + wbft 하드포크) 추가. wemix 는 etcd 라 템플릿 부적합.

### 3.4 wemix 격리

wemix 는 consensus·부트스트랩이 근본적으로 다르다(**etcd 클러스터 멤버십**). LocalDriver 의
"N 프로세스 + genesis 검증자 목록" 가정이 깨지므로, **별도 드라이버/부트스트랩 경로**(etcd 초기화 포함)로
격리하고 매니페스트 `consensus.family: "wemix-etcd"` 로 분기 표시한다. wbft 완료 후 착수(D3).

### 3.5 표면 파생 (분기 제거)

- **MCP `consensus.ts`**: `istanbul_getValidators` 하드코딩 → manifest `validators_method` 파생.
  `consensus` capability 없는 체인은 툴 비활성(`network.capabilities` 게이트 재사용).
- **테스트 러너**: 동작 중인 `requires_capabilities` 게이트(`cmd_test.sh:108`)에 **`chain_compat` 게이트를 추가 배선**.
  프론트매터만 확장하면 매트릭스 실행 가능.
- **프로파일**: 체인 중립 코어 + `chain: wbft` 선택 시 매니페스트가 genesis 구조 결정.
  stablenet 전용 gov 컨트랙트는 profile override 로 이동.

---

## 4. 단계 (Phasing)

| Phase | 내용 | 완료 기준 | 위험 |
|---|---|---|---|
| **P0** | ChainManifest 스키마 + 3 매니페스트 데이터화 (기존 하드코딩 → 데이터 이관, 동작 불변) | `network/chains/*.json` + 로더, 기존 테스트 green | 낮음 |
| **P1** | 어댑터 SSOT Go 확정, start 경로가 `StartFlags`/binary 를 매니페스트에서 취득(하드코딩 9곳 제거), 그 후 `lib/adapters/*.sh` 폐기 | init/start 완전 어댑터 경유, 골든 테스트 통과 | 중 |
| **P2** | **wbft 실구현** (전략 B, stablenet near-clone) + wbft preset 키 + wbft 프로파일 + probe/tx_type 매니페스트 파생 | `chainbench init --chain wbft` E2E 성공 | 중 |
| **P3** | MCP consensus/tx 표면 매니페스트 파생 + 테스트 `chain_compat` 게이트 배선 | 다체인 매트릭스 실행 | 낮음 |
| **P4** | **wemix** — etcd 드라이버 + 부트스트랩 (전략 A/별도 트랙) | `chainbench init --chain wemix` | 높음 |

wbft(P2)는 저비용 빠른 승리, wemix(P4)는 별도 프로젝트급이라 후순위(D3).

### 4.1 Exit criteria (문서 닫힘 조건)

1. `scripts/inventory/scan-binary-hardcoding.sh` 가 0 라인 (`HARDCODING_AUDIT.md` §Exit 와 동일)
2. `lib/adapters/*.sh` 폐기 완료, Go 어댑터가 유일 경로
3. `chainbench init --chain {wbft}` 가 비-stub 어댑터로 E2E 성공
4. 세 체인 매니페스트가 `network/chains/*.json` 로 존재, probe/consensus/tx 표면이 전부 파생

---

## 5. 미해결/추적 항목

- **wbft 네이티브 도구 vs 템플릿 재검토**: P2 착수 시 `go-wbft/cmd/genesis_generator` 의 입력 계약을
  확인해, 템플릿(전략 B)이 croissant `Transitions[]`·`Brioche` 반감기를 정확히 재현하는지 골든 비교로 검증.
  재현 부채가 크면 해당 체인만 전략 A 로 승격 가능(하이브리드는 체인별 선택이므로 열려 있음).
- **preset 키(wbft)**: `keys/preset/wbft/` 신규 fixture 필요(BLS 키 세트). 생성 방법은 P2 spec 에서 확정.
- **wemix etcd 드라이버**: LocalDriver 확장 범위(단일 호스트 임베디드 etcd vs 다중)는 P4 전 별도 설계.
