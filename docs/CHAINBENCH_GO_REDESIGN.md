# Chainbench Go-First 재설계 — 다체인 테스트벤치 아키텍처

> 작성일: 2026-07-24
> 상태: **설계 확정(방향) — 구현 미착수**. 이 문서가 전체 아키텍처의 **SSoT**.
> 목적: 요구사항 1–19를 반영해, chainbench를 **전면 Go**의 3단계 파이프라인 × **합의-알고리즘 중심**
> 플러그인 구조로 재설계한다. 여러 블록체인(go-stablenet / go-wbft / go-wemix)을 하나의 표면에서
> blind 하게 다루고, 코드 가독성·확장성·유지보수성을 최적화한다.
>
> 하위/참고:
> - `docs/CHAIN_EXTENSIBILITY_DESIGN.md` — 어댑터/매니페스트 상세(체인 축)
> - `docs/VISION_AND_ROADMAP.md` §1 — 모드 (A)/(B) 이중 표면 원칙
> - **참고 자산**:
>   - `../accounts` (`github.com/0xmhha/accounts`) — 계정·서명·tx·faucet SDK (다체인 확장 대상)
>   - `../script/wemix-upgrade` — wemix(poa)/wbft 체인 셋업 + 하드포크 업그레이드 절차 레퍼런스
>   - `../chain/{go-stablenet, go-wbft, go-wemix}` — 대상 체인 소스

---

## 0. 확정된 결정 (2026-07-24 검토 완료)

| # | 결정 사항 | 확정 |
|---|---|---|
| D1 | 구현 언어 | **전면 Go** (#18). bash `lib/*` · TS `mcp-server` 폐기 대상 |
| D2 | MCP 서버 | **Go 재작성** (`pkg/mcp`) — 코어 함수 직호출 |
| D3 | accounts SDK 결합 | **인터페이스 경계 + SDK 기본구현 (1+2 결합)** — chainbench `AccountProvider` 뒤에 `0xmhha/accounts`를 기본 구현으로 |
| **D9** | **플러그인 1차 축** | **합의 알고리즘 중심** — `wbft`(stablenet+wbft), `poa`(wemix/etcd). 체인은 합의 family를 조합하는 얇은 계층 |
| **D10** | **accounts SDK 개선** | `../accounts`를 **다체인 구조로 리팩토링 후 의존**. 암호(secp256k1)는 공통, tx 인코딩·계정상태를 체인/합의별 선택 가능하게 |
| D4 | 전환 전략 | **Strangler 점진** |
| D5 | 대시보드 | **코어 Go 완료 후 착수** — 실시간 계약은 구현 시점에 확정 |
| D6 | 어댑터 SSOT | **Go 단일화**, `lib/adapters/*.sh` 폐기 |
| D7 | genesis 전략 | **하이브리드** — 합의 family별 템플릿/네이티브 도구 |
| D8 | 체인 우선순위 | **wbft(stablenet 재사용) 먼저, poa(wemix) 후순위** |
| D-mod | Go 모듈 구조 | **루트 모듈 승격** — `network/` → `github.com/0xmhha/chainbench` 루트로 흡수 (G0) |

---

## 1. 요구사항 → 모듈 매핑

조직 원리 = **세로: 3단계 파이프라인** × **가로: 공통 코어 vs 합의-family 플러그인**.

| 요구 | 핵심 | 담당 |
|---|---|---|
| #1 | genesis 생성 or 코드내장 사용 | `core/genesis` + `consensus/<family>` |
| #2 | genesis 초기 계정 + gas balance | `core/genesis` alloc |
| #3 | faucet: genesis 계정 → 타 계정 전송 | `pkg/accounts` (SDK wallet) |
| #4 | config: 파일/실행옵션/코드 default | `core/config` (3원 해석) |
| #5 | local / remote(id·pw, ssh key) | `core/driver` |
| #6 | 동일서버=port분리, 다중서버=IP변동 | `core/node` |
| #7 | 기존 노드 attach, 3단계 구분 | `core/pipeline` + `driver/attach` |
| #8 | 셋업 = 준비→적용/환경구축→실행 | `pipeline/setup` |
| #9 | 검증 = 블록생성 + 노드정보 | `pipeline/verify` |
| #10 | 테스트 = 준비된 노드에 수행 | `pipeline/testrun` + `tests/` |
| #11 | 모든 체인 동일 1–10 → 분리·분업 | `core/pipeline` (척추) |
| #12 | 체인별 계정/암호/tx/RPC 상이 | `consensus/<family>` + `registry` |
| #13 | 공통 vs 상이 → 확장체계 | `core` vs `consensus`/`chains` 경계 |
| #14 | MCP 별도 모듈 | `pkg/mcp` |
| #15 | cmd 하위 CLI, 신뢰 flag 시스템 | `cmd/*` (cobra) |
| #16 | 헬퍼 ↔ 테스트코드 분리, 네이밍+godoc | `pkg/testkit` ↔ `tests/` |
| #17 | 로깅·디버그·상태·결과 저장 (분리) | `core/obs` |
| #18 | 전면 Go | 전체 |
| #19 | Svelte 대시보드, 3메뉴, 실시간 | `dashboard/` + `chainbenchd` |

---

## 2. 목표 모듈 아키텍처

```
chainbench/  (module github.com/0xmhha/chainbench)
├── cmd/                          # (#15) cobra ✅ chains/setup(plan|provision|launch)/verify/test/faucet
│   ├── chainbench/               #  CLI: setup / verify / test / faucet / attach / hardfork
│   └── chainbenchd/              #  데몬: 이벤트버스 + 대시보드 API + 상태저장 (#17,#19)
├── pkg/core/                     # ── 공통, 체인·합의 불문 (#13) ──
│   ├── pipeline/                 #  3단계 오케스트레이터 (#7,#11)
│   │   ├── setup/                #   준비→적용/환경구축→실행 (#8)
│   │   ├── verify/               #   블록생성 확인 + 노드정보 (#9)
│   │   └── testrun/              #   준비된 노드에 테스트 (#10)
│   ├── config/                   #  3원 해석: default ← file ← flag (#4)
│   ├── genesis/                  #  생성 | 코드내장 | alloc+faucet 골격 (#1–3)
│   ├── node/                     #  노드 모델, role(boot/normal/en), port/ip 할당 (#6)
│   ├── driver/                   #  local | remote(ssh-key/id·pw) | attach (#5–7)
│   ├── rpc/                      #  JSON-RPC, health/블록 프로빙
│   ├── obs/                      #  로깅·디버그·상태·결과 + 이벤트버스 (#17)
│   └── registry/                 #  플러그인 레지스트리 + Manifest (#12,#13)
├── pkg/consensus/                # ── 합의-family 플러그인 (D9) = 1차 확장축 ──
│   ├── wbft/                     #  BFT: BLS validator, istanbul 네임스페이스, anzeon/croissant genesis
│   │                             #     → stablenet · wbft 공유
│   └── poa/                      #  etcd 멤버십, governance 배포, wemix 네임스페이스
│                                 #     → wemix (bootnode→governance→etcd→others)
├── pkg/chains/                   # ── 체인 프로파일 (얇은 계층) ──
│   ├── stablenet/  wbft/  wemix/ #   합의 family 선택 + chain_id/하드포크/템플릿 파라미터
├── pkg/accounts/                 # AccountProvider 경계 + 0xmhha/accounts(다체인화) (#3,#12,D10)
├── pkg/testkit/                  # 테스트 헬퍼 (#16): assert/fixture/report/wait
├── pkg/mcp/                      # MCP 표면 (#14, Go)
├── tests/                        # 테스트 코드 (#16, 헬퍼와 분리)
│   └── <family>/<category>/*_test.go   # 네이밍 규칙 + godoc 헤더 의무
├── manifests/chains/*.json       # 선언적 체인 매니페스트
├── dashboard/                    # (#19) Svelte SPA
└── docs/
```

규율: `pkg/core/*`는 특정 체인·합의를 **import 하지 않는다**. 합의 지식은 `pkg/consensus/*`, 체인
파라미터는 `pkg/chains/*`에만 있고 core는 `registry`로만 접근(의존 역전) → #13 경계를 컴파일러가 강제.

---

## 3. 3단계 파이프라인 = 조직 척추 (#7–11)

```
        ┌─────────── Setup (#8) ───────────┐
전체 →  │ 1.설정준비  2.적용/환경구축  3.실행 │ → NodeSet ─┐
플로우  └──────────────────────────────────┘            │
                                                        ▼
        ┌──── Verify (#9) ────┐        ┌──── Test (#10) ────┐
        │ 블록진행? + 노드정보 │  ───▶  │ testkit + tests/    │
        └─────────────────────┘        └─────────────────────┘
             ▲
   attach(#7): setup 스킵, rpc url/port로 NodeSet 구성 후 직행
```

- **Setup(#8)**: `config` 해석 → `consensus/<family>`가 genesis/키/멤버십/config 준비 → `driver` 실행.
- **Verify(#9)**: `rpc`로 `eth_blockNumber` 진행 관찰 + peer/validator/chain_id 수집.
- **Test(#10)**: `testkit` + `tests/<family>/`를 검증 통과 NodeSet에 실행.
- **attach(#7)**: 기존 노드를 1급 진입점으로. setup 없이 `NodeSet{rpcURL,port}` → verify.

단계 간 유일 전달 객체 = **`NodeSet`**(노드목록+접속정보+capability). 각 단계는 `NodeSet`을 받고
`NodeSet`(+obs 기록)을 반환 → 단계 독립 테스트·재실행 가능.

### 3.1 wemix-upgrade 레퍼런스 매핑

`../script/wemix-upgrade`가 각 단계의 구체 절차와 합의 분기를 실증한다. Go 재설계는 이를 계승:

| wemix-upgrade | chainbench 대응 | 비고 |
|---|---|---|
| `node_setup.sh {init,config,run,stop,clean,setup}` | `pipeline/setup` 서브스텝 | 커맨드 표면 그대로 |
| `init-gwemix.sh` / `run-gwemix.sh` (v3.0) | `consensus/poa` | gwemix, etcd |
| `init-geth.sh` / `run-geth.sh` (v3.5+) | `consensus/wbft` | geth 계열, BFT |
| role: `boot-node` / `node` / `ennode` | `core/node.Role` | boot=governance 배포 주체 |
| poa 부팅: node1→governance 배포→etcd init→others | `consensus/poa.Setup()` | #1 코드내장 genesis + 배포 스텝 |
| `--remote` + PUBLIC_IP, BASE_PORT | `driver/remote` + `core/node` | #5,#6 |
| genesis `montBlancBlock:100` 에서 gwemix→geth 교체 | `cmd/chainbench hardfork` | poa→wbft 하드포크 업그레이드 테스트 |

---

## 4. 합의-family 플러그인 계약 (D9, #12, #13)

**1차 축은 합의 알고리즘.** 새 체인 대부분은 기존 family 재사용 + 파라미터만 추가.

```go
// pkg/core/registry
type ConsensusFamily interface {
    ID() string                              // "wbft" | "poa"
    Genesis(ctx, ChainParams) (Genesis, error)  // 구조/validator/멤버십 (#1,#12)
    Bootstrap(ctx, NodeSet) error            // BLS extraData | governance배포+etcd (#8)
    ValidatorsMethod() string                // istanbul_getValidators | wemix_*
    RpcNamespace() string                    // istanbul | wemix
    StartFlags(role Role) []string
    Capabilities() []Capability
}

type ChainPlugin interface {
    ID() string                  // "stablenet" | "wbft" | "wemix"
    Family() ConsensusFamily     // 합의 family 선택 (D9)
    Params() ChainParams         // chain_id, 하드포크 필드, genesis 템플릿, tx타입
    Accounts() AccountProvider   // #12,#3
}

func Register(p ChainPlugin)     // chains/<x>/init.go 에서 호출
```

| 관심사 | core(공통) | consensus/wbft | consensus/poa | chains/*(파라미터) |
|---|---|---|---|---|
| 파이프라인·config·driver·rpc·obs·port/ip | ✅ | — | — | — |
| genesis 구조 | 골격(alloc/faucet) | anzeon/croissant | wemix+montBlanc | chain_id/하드포크값 |
| validator/멤버십 | 인터페이스 | BLS 목록 | etcd + governance | — |
| RPC 네임스페이스 | 프로빙 골격 | istanbul | wemix | — |
| tx 규칙·암호 | AccountProvider 계약 | secp256k1 + tx타입 | secp256k1 + tx타입 | 지원 tx타입 |

정적 사실(binary, 네임스페이스, api list, 하드포크, tx타입, probe)은 `manifests/chains/*.json`으로
데이터화 → 현행 `probe/signatures.go`·`feeDelegationAllowedChains` 하드코딩 제거.

---

## 5. 계정·faucet & accounts SDK 다체인화 (D3, D10, #3, #12)

### 5.1 chainbench 경계 (D3)

```go
// pkg/accounts — chainbench 소유 인터페이스(느슨한 결합)
type AccountProvider interface {
    Scheme() signing.Scheme
    NewWallet(ctx, key, rpc) (Wallet, error)
    Faucet(ctx, from, to, amount) (TxHash, error)   // genesis alloc 계정 → 타 계정 (#3)
    BuildTx(TxRequest) (SignedTx, error)            // tx 인코딩/디코딩 (#12)
}
// 기본 구현 = 0xmhha/accounts (재구현 0). chainbench internal/signer·abiutil·핸들러 tx빌딩 삭제.
```

### 5.2 accounts SDK 리팩토링 방향 (D10 — 별도 워크스트림 A)

현재 `../accounts`는 **go-stablenet 전용 clean-room**. 조사 결과 **암호(secp256k1)는 체인 공통**이고,
분기는 **tx 인코딩(0x16 fee delegation 등)·계정상태(`Extra` 비트맵)·token/governance 바인딩**에 있다.
→ 합의/체인 중심으로 다음과 같이 분리:

| accounts 패키지 | 성격 | 다체인화 |
|---|---|---|
| `crypto` `signing` `keystore` `types` `hdwallet` `vault` `abi` | **공통 프리미티브** | 변경 최소 (secp256k1 공유) |
| `tx` | 체인/합의별 tx타입 세트 | `Protocol` 프로파일로 지원 tx타입 선택 (baseline 0x00–0x04 + 체인확장) |
| `account`(Extra) `token` `governance` `transport` | 체인 온체인 규격 의존 | 체인별 구현/프로파일 분리 |

- accounts는 **독립 repo(자체 ADR·spec·골든벡터·TDD)** 이므로, 이 리팩토링은 **그 repo의 별도 설계 사이클**로
  진행(스펙 우선 + 골든 벡터 회귀). chainbench는 `AccountProvider` 경계로 이를 소비 → 두 프로젝트 병행 가능.
- **faucet(#3)**: genesis 잔액 계정(#2)을 faucet 지갑으로 승격 → `wallet.SendCoin`. CLI `chainbench faucet send`
  + MCP tool + setup 산출물로 노출.

---

## 6. Config 3원 해석 (#4)

우선순위 **코드 default ← config 파일 ← 실행옵션(flag/env)**. 단일 `core/config.Resolve()`가 병합해
`EffectiveConfig` 산출 → 모든 단계·드라이버가 이 하나만 소비(현행 `defaults.generated.sh`/YAML/flag 3중
SSOT 문제 해소). node 실행 flag는 `ConsensusFamily.StartFlags(role)`가 family별 구성.

---

## 7. 드라이버: local / remote / attach (#5–7)

```go
type Driver interface {
    Provision(ctx, NodeSpec) error       // datadir/config/genesis/keys 배치
    Launch(ctx, NodeSpec) (Handle, error)
    Stop(ctx, Handle) error
}
```

- **LocalDriver**: 동일 호스트 → port offset 할당(#6).
- **RemoteDriver**: 호스트별 1노드 → port 고정·IP 변동(#6). 인증(#5): ssh key / id·pw / agent.
  현행 `drivers/sshremote`(host-key 검증)·`drivers/remote`(API key/JWT) 확장. 자격증명은 env-var 이름만
  전달, 값 미로그.
- **AttachDriver**: Provision/Launch 무동작. 기존 rpc url/port → `NodeSet`(#7).

`obs`가 드라이버 불문 동일 이벤트 방출 → 상위 blind.

---

## 8. 관측·상태·대시보드 (#17, #19)

- **obs 분리**(D): setup·verify도 런타임 기록을 낳으므로 테스트 전용이 아님. `pkg/core/obs` = 구조적
  로깅(slog, 체인/앱 로그 분리) + run/state/result 저장 + 이벤트버스(현행 `internal/events/bus` 계승).
  `testkit`은 obs를 **소비**만.
- **대시보드**(#19, D5): `chainbenchd`가 obs 이벤트를 대시보드에 스트리밍. **3메뉴 = 3단계**(Setup/Verify/Test)
  분리, anti-slop 디자인. 실시간 전송 방식(SSE vs WS)·이벤트 스키마 버저닝은 **코어 Go 완료 후 구현 시점 확정.**

---

## 9. 테스트 체계 (#16)

- 헬퍼(`pkg/testkit`) ↔ 테스트 코드(`tests/`) 물리 분리.
- 네이밍: `tests/<family>/<category>/<name>_test.go` (예: `tests/wbft/consensus/validator_set_test.go`).
- **모든 테스트 상단 godoc 의무**(무엇·왜·전제·판정기준).
- 체인별 가능 테스트 차이 = `Capabilities` + `chain_compat` 게이트로 런너 필터.

---

## 10. 실행 계획 (Strangler — D4)

두 워크스트림 병행: **A = accounts SDK 다체인화(별도 repo)**, **G = chainbench Go 재설계**.

| Phase | 워크스트림 | 내용 | 완료 기준 |
|---|---|---|---|
| **A0** ✅ | accounts | tx/account/token/governance의 stablenet 결합 조사 + `Protocol` 프로파일 설계 | **완료** — `docs/A0-ACCOUNTS-MULTICHAIN-SCOPE.md`. 0x16 공통·Extra stablenet전용 확정 |
| **A1** ✅ | accounts | 공통/체인분리 리팩토링 + 체인별 tx타입/계정/컨트랙트 프로파일 | **완료(코어)** — `accounts/protocol` 패키지: **stablenet+wbft+wemix** preset, `ContractModel`(fixed vs wemix registry), `ByName`/`Register`. additive·하위호환, 전체 골든 green, branch `feat/multichain-protocol`. **의도적 이월**(사유 명시) → §11 |
| **G0** ✅ | chainbench | 루트 모듈 승격(D-mod) + `core/{registry,config,obs,node}` + `NodeSet`/family 인터페이스 + 매니페스트 데이터화 | **완료(코어)**: 루트 go.mod(`github.com/0xmhha/chainbench`, accounts replace) + `pkg/core/{node,registry,config,obs}` + `pkg/consensus/{wbft,poa}` + `pkg/accounts`(AccountProvider) + `pkg/chains/{stablenet,wbft,wemix}` 등록 + `manifests/chains/*.json`. 빌드/vet/test green, `network/` 무변경. **이월**: `network/` 흡수(드라이버/wire 이관)는 소비자(pipeline)가 생기는 G2–G3에서 수행 |
| **G1** 🟡 | chainbench | `pkg/accounts`(AccountProvider + A1 의존) + faucet, `internal/signer`·abiutil 대체 | **코어 완료**: `AccountProvider` 확장(AddressForKey/OpenWallet/**Faucet #3**) + `Wallet`(SDK `wallet` 백엔드). 오프라인 주소파생(골든) + **RPC-mock faucet E2E**(0x02 raw tx 제출 확인) green. **이월**: `network/internal/signer`·abiutil 실삭제는 tx 핸들러 흡수(G2–G3) 시점, 0x16 live golden은 빌드된 노드 필요 |
| **G2** 🟡 | chainbench | `pipeline/setup` + `driver/local` + `consensus/wbft` → bash `cmd_init/start/stop` 폐기(하드코딩 제거) | **코어 완료**: `pkg/core/driver`(Driver + LocalDriver, exec 주입) + `pkg/core/pipeline/setup`(BuildPlan role/port 배치 + Run provision→launch→NodeSet + obs 이벤트) + `consensus/wbft.BuildGenesis` + `pkg/core/nodeconfig`(per-node TOML). **`setup --provision`이 genesis.json + node TOML(포트/네임스페이스/miner/static-node enode) 완전 환경 생성**. 전부 테스트 green. **이월**: 실 systemContracts/alloc 파리티·live launch(바이너리 init/실행)는 `network/` 흡수와 함께 |
| **G3** 🟡 | chainbench | `pipeline/verify` + `driver/{remote,attach}` → bash `cmd_status/remote` 폐기 | **코어 완료**: `pkg/core/rpc`(범용 JSON-RPC) + `pkg/core/pipeline/verify`(블록생성 감지 + 노드정보, Prober 주입) + `pkg/core/pipeline/attach`(#7 기존 노드 NodeSet 구성). httptest/fake로 테스트 green. **이월**: remote(ssh key/id·pw) 드라이버는 검증된 `network/internal/drivers/{remote,sshremote}`에 있으므로 `network/` 흡수 시 이관, bash `cmd_status/remote` 실삭제도 그때 |
| **G4** ✅ | chainbench | `pipeline/testrun` + `testkit` + `tests/` 이관(godoc) | **완료(코어)**: `pkg/testkit`(Case/T/Report + Register, 헬퍼와 코드 분리) + `pkg/core/pipeline/testrun`(chain_compat/capability 게이트 + obs/store) + `tests/wbft/consensus/chain_id.go`(네이밍+godoc 규칙 샘플, `tests/README.md`). 게이트·pass/fail/skip·store 기록 테스트 green. **이월**: 실 테스트 이관(기존 200+ 케이스)은 `network/` 흡수 후 점진 |
| **G5** 🟡 | chainbench | **wbft 체인 플러그인**(consensus/wbft 재사용 + 파라미터) + wbft preset 키 | **코어 완료**: manifest에 chain_id/genesis(engine_field/hardforks/template) 데이터화 + 체인별 genesis 템플릿(anzeon/croissant) 임베드 + `pkg/core/genesis.Build`(family dispatch) + `pkg/core/keys`(preset 로더). **`setup --provision`이 실 preset 검증자로 진짜 genesis.json 생성**(stablenet 4 검증자, chainId 8283 확인). **이월**: wbft preset BLS 키 + per-node config(TOML) + live launch(gwbft/gstable 바이너리)는 `network/` 흡수와 함께 |
| **G6** | chainbench | `pkg/mcp` Go 재작성 → TS `mcp-server` 폐기 | tool 등가 + 통합테스트 |
| **G7** | chainbench | `consensus/poa` + `chains/wemix` (wemix-upgrade 절차: bootnode→governance→etcd) | `chainbench setup --chain wemix` E2E |
| **G8** | chainbench | `hardfork` 커맨드 (poa→wbft 업그레이드, wemix-upgrade 재현) + `chainbenchd`/대시보드(D5) | 하드포크 테스트 + 3메뉴 실시간 |

순서: accounts(A0–A1) → 코어(G0–G5, wbft 축 먼저 D8) → MCP(G6) → wemix poa(G7) → 하드포크+대시보드(G8).

### 10.1 종료 조건

1. `lib/*.sh` · `mcp-server/`(TS) 폐기, Go 단일 모듈 수렴
2. `chainbench {setup,verify,test,faucet,attach}`가 wbft/poa 두 family에서 동작
3. 3단계 파이프라인이 obs 기록, 대시보드 실시간 반영
4. 새 체인 추가 = `pkg/chains/<x>/`(family 선택 + 파라미터) + `manifests/chains/<x>.json`. 새 합의만 신규
   `pkg/consensus/<family>/` 추가
5. poa→wbft 하드포크 업그레이드 테스트 통과 (wemix-upgrade 등가)

---

## 11. 미해결 · 리스크

- **accounts SDK 다체인화(A0+A1)**: ✅ **해소** — `docs/A0-ACCOUNTS-MULTICHAIN-SCOPE.md` + `accounts/protocol`
  패키지(stablenet/wbft/wemix preset, additive, 골든 green). **의도적 이월**(날조 회피 — 외부 입력 필요):
  - *wbft/wemix governance·token ABI 바인딩*: 각 체인 시스템컨트랙트의 온체인 ABI 스펙이 있어야 정확. wemix는
    Registry 해석 모델이라 배포 후 주소 확정. → 실제 governance 테스트를 쓰는 시점(G5/G7)에 스펙 기반 추가.
    현재 `governance`/`token`/`account.Extra`는 **stablenet 프로파일**로 유지(문서 명시), 재배치는 미실행(공개 API
    안정성 우선).
  - *0x16 live golden*: 정적 소스 동등성은 확인(§ A0). build한 3개 노드로 raw tx 바이트 비교는 G1에서 회귀로 추가.
- **poa(wemix) 로컬 부트스트랩**: etcd 멤버십 + governance 배포 순서(node1→배포→etcd→others)를
  `consensus/poa.Bootstrap`으로 이식. 로컬 다노드 etcd 토폴로지(단일호스트 임베디드 범위)는 G7 전 확정.
  wemix-upgrade의 `init-gwemix.sh`/`deploy-gevernance.js`가 1차 레퍼런스.
- **하드포크 커맨드**: 동일 체인 데이터 위 바이너리 교체(gwemix→geth) + 하드포크 블록 트리거 재현.
  체인 데이터 호환성·중단 지점은 G8 spec에서 확정.
- **대시보드 실시간 계약**: SSE vs WS, 이벤트 스키마 버저닝 — 구현 시점(G8) 확정(D5).
