# Chainbench Go 재설계 — Dev HandOff

> 목적: **다른 머신/세션에서 이 문서만 읽고 모든 작업을 이어서 진행**할 수 있게 하는 완전한 컨텍스트.
> 작성: 2026-07-24. 대상 브랜치 커밋 시점의 상태 기준.
> 함께 볼 것:
> - **`docs/CHAINBENCH_GO_REDESIGN.md`** — 아키텍처 SSoT (전체 설계 + 로드맵 + §0.1 현황). **먼저 읽어라.**
> - `docs/A0-ACCOUNTS-MULTICHAIN-SCOPE.md` — accounts SDK 다체인 조사 결과(0x16 공통·Extra stablenet전용).
> - `docs/CHAIN_EXTENSIBILITY_DESIGN.md` — 체인 플러그인 축 상세.
> - `docs/dev/session-data/` — 이 작업을 만든 세션의 원본 로그(transcript jsonl) + memory 사본. 진행 맥락이 더 필요하면 읽어라.

---

## 0. 30초 요약

chainbench를 **go-stablenet 전용 bash 도구** → **다체인(stablenet/wbft/wemix) Go-first 테스트벤치**로 재설계 중. 조직 원리 = **세로 3단계 파이프라인(setup→verify→test) × 가로 공통코어 vs 합의-family 플러그인(wbft/poa)**. 두 표면(사람=CLI, 에이전트=MCP) + 대시보드가 **동일 Go 코어** 위에 있음. 계정/서명/tx/faucet은 별도 SDK `github.com/0xmhha/accounts`에 의존.

**핵심 코어·3표면·전체 로컬 라이프사이클은 이미 동작하고 전부 테스트 green.** 남은 건 대부분 (a) 실 체인 바이너리/etcd/프론트 툴체인이 필요한 실행·검증, (b) 레거시(`network/` Go 모듈 + bash `lib/*` + TS `mcp-server`) 폐기, (c) MCP 툴 파리티 확대.

---

## 1. 리포·브랜치·모듈 구조

두 리포가 로컬에 co-located:

| 리포 | 경로 | 모듈 | 역할 |
|---|---|---|---|
| chainbench | `/Users/wm-it-25_0220/Work/github/chainbench` | `github.com/0xmhha/chainbench` (루트) | 본체 |
| accounts | `/Users/wm-it-25_0220/Work/github/accounts` | `github.com/0xmhha/accounts` | 계정·서명·tx·faucet SDK |

- chainbench `go.mod`: `replace github.com/0xmhha/accounts => ../accounts` (로컬 co-dev). ⚠️ **CI/다른 머신에서는 이 상대경로가 깨진다.** accounts가 태그되면 pseudo-version으로 교체 필요. 다른 머신에서 이어가려면 두 리포를 형제 디렉토리로 클론해야 함.
- chainbench 루트에 신규 Go 모듈. **레거시 `network/`는 nested Go 모듈**(`github.com/0xmhha/chainbench/network`)로 그대로 존재 — `go list ./...`(루트)에서 자동 제외됨. 무변경 유지, 점진 흡수 예정. `internal/`이라 루트에서 import 불가.

### 워크플로 규칙 (중요)
- **작업 단위마다 새 feature 브랜치 → squash-merge PR → main 동기화 → 다음 청크는 새 브랜치.** (사용자가 직접 머지·동기화한다.)
- **commit guard**: chainbench 워킹디렉토리가 `main`이면 커밋 차단. **항상 feature 브랜치에서 커밋.**
- 커밋 메시지·PR 본문: **영어, 이모지 없음, co-author/Claude-Code 푸터 없음.**
- 이미 머지된 PR: chainbench #16(Go 재설계 G0–G8), #17(obs/report), accounts #5(protocol). main에 반영됨.
- persistent memory: `~/.claude/projects/-Users-wm-it-25-0220-Work-github-chainbench/memory/go-redesign-workflow.md` (사본은 `docs/dev/session-data/memory/`).

---

## 2. 현재 구현 상태 (main)

전 32 패키지 build/vet/test green. 3 바이너리 빌드: `chainbench`, `chainbench-mcp`, `chainbenchd`.

### 2.1 아키텍처 맵 (패키지 → 책임)

```
cmd/
  chainbench/        CLI (cobra). 10 commands (아래 2.2)
  chainbench-mcp/    MCP stdio 서버 (pkg/mcp.Default 위 얇은 루프)
  chainbenchd/       대시보드 데몬 (obs bus+store 위 HTTP/SSE)
pkg/core/
  registry/          ChainPlugin/ConsensusFamily 인터페이스 + Manifest(데이터) + Register/Get
  config/            3원 해석: 코드default ← file ← flag (Values dot-path 맵)
  obs/               Event/Bus(pub-sub) + Store(MemStore, FileStore) + slog Logger. 로그·상태·결과(#17)
  node/              Node/NodeSet(3단계 hand-off 객체) + Role + Endpoints + Offset(포트할당)
  driver/            Driver 인터페이스 + LocalDriver(exec 주입) + InitDatadir + Stop
  rpc/               범용 JSON-RPC 클라이언트(eth_/net_ + Call passthrough)
  keys/              preset 로더(metadata.json → validators/BLS/extraData + node pubkeys)
  nodeconfig/        per-node geth-family TOML 생성(포트/keystore/namespace/miner/static-nodes)
  genesis/           genesis.Build — family dispatch (wbft: 템플릿치환 / poa: base). Inputs
  consensus/         manifest-driven 검증자 조회(Validators)
  hardfork/          BuildPlan + Plan.Execute(stop+relaunch on new binary)
  state/             NodeSet ↔ nodeset.json 영속
  pipeline/
    setup/           BuildPlan(role/port) + Run(provision→launch→NodeSet+obs) + LaunchArgs
    verify/          Run(블록생성 감지 + 노드정보; Prober 주입)
    attach/          Build(기존 노드 rpc → NodeSet; setup 스킵 #7)
    testrun/         Run(chain_compat/capability 게이트 + obs/store)
pkg/consensus/
  wbft/              wbft family: istanbul namespace, StartFlags, BuildGenesis(anzeon/croissant 치환)
  poa/               poa family(wemix): wemix namespace, BuildGenesis(base), BootstrapPlan(etcd 시퀀스)
pkg/chains/
  stablenet,wbft,wemix/  얇은 플러그인(family+protocol+manifest+genesis템플릿 조합, init 등록)
  all/               전체 등록 blank-import
pkg/accounts/        AccountProvider 경계 + 0xmhha/accounts 백엔드(faucet/wallet/tx)
pkg/testkit/         테스트 헬퍼(Case/T/Report/Register) — 테스트 코드와 분리(#16)
pkg/mcp/             self-contained MCP JSON-RPC(initialize/tools/list/tools/call) + 7 tools
pkg/dashboard/       HTTP(/events SSE, /api/runs, POST /api/events) + 임베드 HTML + Forward
manifests/
  chains/*.json      선언적 체인 매니페스트(binary/chain_id/family/genesis/consensus/tx_types/probe/caps)
  genesis/*.json     체인별 genesis 템플릿(stablenet=anzeon, wbft=croissant, wemix=base)
tests/
  wbft/consensus/chain_id.go   테스트 케이스 샘플(네이밍+godoc 규약)
  all/               테스트케이스 등록 blank-import
  README.md          tests/ 규약
```

### 2.2 표면 (동작 확인됨)

**CLI 10 명령** (`chainbench <cmd>`):
- `chains` — 등록 체인 목록
- `setup --chain --validators --endpoints [--provision] [--launch] [--binary] [--keys-dir] [--data-dir] [--dry-run]` — plan/환경구축/실행. `--provision`=genesis+TOML 작성, `--launch`=init+실행+nodeset.json 저장
- `stop --data-dir` — nodeset.json의 PID로 노드 종료
- `verify --rpc <url>... | --data-dir` — 블록생성+노드정보
- `test --rpc | --data-dir [--name] [--category]` — 케이스 실행, `--data-dir`시 runs.json 저장
- `node rpc --rpc --method [--params]` — 임의 JSON-RPC passthrough
- `consensus --chain --rpc` — 검증자 집합(manifest method)
- `hardfork --data-dir --to-chain --block [--to-binary] [--dry-run]` — 바이너리 교체 업그레이드(plan/execute)
- `report --data-dir` — 저장된 결과 조회
- `faucet --chain --rpc --from-key --to --amount` — genesis 계정 → 전송
- 영속 플래그 `--dashboard <url>` — setup/verify/test 이벤트를 chainbenchd로 SSE 전달

**MCP 7 tools**: `chainbench_{chains,faucet,verify,test,consensus,node_rpc,report}` (self-contained JSON-RPC, stdio).

**대시보드**: `chainbenchd --addr` → `/events`(SSE), `/api/runs`, `POST /api/events`, `/`(3-phase HTML). CLI `--dashboard`로 라이브 연결.

### 2.3 3 체인 현황

| 체인 | family | binary | chain_id | genesis | 검증자 |
|---|---|---|---|---|---|
| stablenet | wbft | gstable | 8283 | anzeon 템플릿 | in-genesis (preset BLS) |
| wbft | wbft | gwbft | 8284 | croissant 템플릿 | in-genesis |
| wemix | poa | gwemix | 8285 | base 템플릿 | bootstrap(etcd/governance) — `poa.BootstrapPlan` |

- chain_id 8284/8285는 **로컬 네트워크 기본값**(operator override 가능), 실 체인 canonical 값 아님.
- genesis 템플릿은 **consensus-critical placeholder만**(chainId/validators/BLS/extraData/coinbase). 실 systemContracts/alloc 파리티는 미포팅(§4 이월).

---

## 3. 빌드·테스트·실행 (명령 모음)

```bash
# chainbench 루트에서
go build ./...                    # 32 패키지 빌드
go test ./...                     # 전체 테스트 (httptest/fake로 결정적, 실 바이너리 불필요)
go vet ./...
gofmt -l pkg/ cmd/                # 포맷 확인(빈 출력=clean)
go build -o bin/chainbench ./cmd/chainbench
go build -o bin/chainbench-mcp ./cmd/chainbench-mcp
go build -o bin/chainbenchd ./cmd/chainbenchd

# accounts 리포에서
cd ../accounts && go test ./...   # 골든 벡터 포함

# 레거시 network 모듈(무변경 확인용)
cd network && go build ./...
```

**실 체인 없이 동작 확인 예 (fake 바이너리)**:
```bash
printf '#!/bin/sh\nif [ "$1" = "init" ]; then exit 0; fi\nsleep 30\n' > /tmp/fakegeth; chmod +x /tmp/fakegeth
chainbench setup --chain stablenet --validators 2 --data-dir /tmp/d --keys-dir keys/preset --binary /tmp/fakegeth --launch
chainbench stop --data-dir /tmp/d
```

**실 체인 E2E**(빌드된 바이너리 필요): `--binary`를 실 gstable/gwbft 경로로. `../chain/go-stablenet` 등에서 `make gstable` 후 `build/bin/gstable`.

---

## 4. 남은 작업 리스트

> 상태 표기: ✅ 완료 / 🟢 지금 구현 가능 / 🔴 외부 자원 블로커.
>
> **2026-07 갱신**: 레거시 3-스택(`network/` wire 모듈, TS `mcp-server/`, bash CLI
> `lib/`·`chainbench.sh`)이 모두 제거되어 저장소가 **Go-first 단일 아키텍처**로
> 수렴했다. 아래 A~E의 대부분이 완료됨. PR #30~#53 참조.
>
> **2026-07-27 갱신**: 다체인 분리 감사(공통 코어 vs 체인 특화)로 아래 "구조 분리·
> 데이터화 완결(S1~S6)"을 신규 최우선 블록으로 추가. 기존 A4/E1은 S4/S5로 승계·정정.

### 완료됨 ✅
- **B1. `network/` 흡수 → 삭제**: remote RPC auth + SSH 터널(`pkg/core/remote`),
  체인 감지(`pkg/core/probe`), named-network 레지스트리(`pkg/core/state`)를 코어로
  흡수. wire/events는 `pkg/core/obs`, adapters는 genesis/nodeconfig/family/manifest로
  이미 대체. `network/`·`network/go.mod` 제거(#30–#33, #41).
- **B2. bash CLI 폐기**: `lib/`·`chainbench.sh`·`tests/unit/`·`logs/` 삭제. Go CLI가
  핵심 워크플로(setup/start/stop/status/clean/verify/test/…) 커버(#42, #43).
- **B3. TS `mcp-server/` 폐기**: Go `chainbench-mcp`(단일 바이너리, 30툴)로 전환,
  `setup.sh`/docs 갱신 후 삭제(#40, #41). **A2**(MCP 파리티)·**A3**(setup-plan)·
  **A5**(log/log_timeline)도 그 과정에서 완료.
- **C1/C2/C3. 실 체인 E2E + wemix 부트스트랩 + 하드포크**: go-wemix+etcd → go-wbft
  croissant 핸드오프를 프레임워크(`pkg/consensus/upgrade`)로 코드화하고 `chainbench
  upgrade run` + 라이브 E2E(`tests/repro/wemix-wbft-handoff.sh` + gated Go 테스트)로
  검증(#27–#29, #38, #39).
- **E1(부분). 원격 RPC 접근**: auth 트랜스포트 + SSH 터널을 `pkg/core/remote`로 흡수,
  `network_*`/`remote_rpc` MCP 툴로 노출(#30, #33, #34).
- **회귀 스위트 Go 포팅(진행 중)**: 바인딩 계층(ABI `Selector`/`EncodeCall`, tx 4종
  `SendLegacy`/`SendCoin`/`SendSetCode`/`SendFeeDelegated`, 이벤트 로그 디코딩) +
  카테고리 대표 케이스(node/api/consensus/tx/시스템컨트랙트 read·write·event)를
  `tests/`로 포팅(#44–#53). 원본 bash는 `tests/regression/`에 보존.

### 구조 분리·데이터화 완결 (2026-07-27 다체인 감사, 신규·최우선)

> 사용자 지적("공통 코어 vs 체인 특화가 아직 분리 안 됨")을 코드로 검증한 결과.
> **아래 S1~S6은 대부분 기존 리스트에 없던 신규 항목**이며, 확장성을 우선하므로
> 회귀 잔여 포팅보다 앞선다(그 포팅을 계속하면 S4 부채가 커진다).
> 근본원인: 체인 사실을 Manifest로 **데이터화했으나 일부 code path가 그 필드 대신
> 하드코딩 문자열을 읽음** → 설계 결함이 아니라 데이터화 완결로 해소.
> 검증 시 초안 2건 정정: skip은 요약에 정상 집계됨(숨김 아님), MinerRecommit은
> 死데이터 아님(upgrade 경로는 사용, setup 경로만 무시).

진행 상태: **S1–S6 전부 적용 완료**(#66·#67 + 후속 PR). S5 원격 배선 완료(실 원격 E2E는
Docker sshd로 별도 검증 필요), S6 잔여(Node.Auth·README) 완료. 이후는 회귀 write 포팅 재개.

1. ✅ **S1 테스트 커버리지 신호 + 체인 온보딩** (#66): `obs.RunSkipped` 신설로 skip을
   성공과 구분 기록, `chainbench test`·요약이 `coverage=ran/applicable` 노출,
   `tests/README.md`에 커버리지·온보딩 절차 명문화. — 아래는 착수 당시 분석.
   `testrun.go:106`이 persist
   RunRecord.Status에서 skip→success로 뭉갬(내부 Result엔 "skip" 보존). 체인별
   기대 커버리지 개념이 없어 대부분 skip인 체인이 fail=0으로 완전커버처럼 보임.
   → skip을 성공과 구분 기록 + 체인별 커버리지 수치, `tests/README.md`에 체인
   온보딩 절차 명문화. 테스트벤치 신뢰성 문제라 최우선.
2. ✅ **S2 Probe·MinerRecommit 데이터화 완결** (#66): probe가 `Manifest.Probe`(死데이터
   해소), nodeconfig가 `Manifest.MinerRecommit`을 읽게 전환, `=="wemix"` 제거. — 분석:
   `probe/signatures.go:17-35`가
   탐지표(namespace·probe method)를 하드코딩하고 `Manifest.Probe`는 死데이터(읽힘 0).
   `nodeconfig.go:47`은 `RPCNamespace=="wemix"`로 recommit 인코딩을 재유도하나
   `Manifest.MinerRecommit`을 무시(반면 upgrade `plan.go:192`는 정상 사용 → 두 경로
   불일치). → probe가 `Manifest.Probe`를, nodeconfig가 `Manifest.MinerRecommit`을
   읽게 하고 문자열 비교 제거. 독립·저위험 → **S1과 한 PR로 묶기 권장**.
3. ✅ **S3 genesis family dispatch를 인터페이스로** (#66): `ConsensusFamily.BuildGenesis`
   +`registry.GenesisParams` 추가, `genesis.Build`는 가상 디스패치 → `pkg/core`가
   concrete consensus를 import 안 함(경계 회복). — 분석: `genesis.go:13-14,40-64`가
   `pkg/core`에서 유일하게 concrete `pkg/consensus/{poa,wbft}`를 import+switch(레이어
   인버전). → `ConsensusFamily`에 `BuildGenesis` 추가해 가상 디스패치, concrete
   import 제거. 새 합의 family를 플러그인 등록만으로 추가 가능(D9 경계 회복).
4. ✅ **S4 accounts governance/token을 stablenet 스코프로 이전** (#66): GovBase 바인딩을
   `pkg/accounts` → `pkg/chains/stablenet/govbind`로 이전, `HasAccountExtra` doc 일반화.
   `pkg/accounts`는 generic ABI/event/tx 헬퍼만. SDK per-chain profile(A0)은 별도 repo.
   이제 회귀 write 포팅 재개 가능(부채 해소).
5. ✅ **S5 RemoteDriver 배선** (#66 + 후속 PR): config를 `NodeSpec.ConfigContent`로
   렌더 → driver의 Provision 경유(local=파일, remote=base64 전송), `LaunchOptions.Driver`
   주입. 후속 PR에서 선택적 `driver.Initializer` 인터페이스 추가 — Local/Remote가
   `InitDatadir(spec, genesis)` 구현(local=datadir에 genesis 기록 후 init, remote=genesis
   base64 전송 후 ssh init). setup.Launch가 type-assert로 원격 init까지 태움. provision·
   init·launch·stop 전부 Driver seam 뒤. 원격 명령은 fakeRunner 단위테스트로 검증.
   gated E2E(`remote_e2e_test.go`)에 원격 init 검증 단계 추가. ⚠️ 실 over-SSH 실행은
   Docker sshd로 시도했으나 이 샌드박스에서 SSH 세션이 hang(컨테이너·TCP는 정상) —
   실 원격 E2E green은 정상 환경에서 `tests/remote/sshd/run.sh`로 재확인 필요.
6. ✅ **S6 잔여 정리** (#67 + 후속 PR): hardfork→setup 결합 제거(`LaunchArgs`→`nodeconfig`),
   `pkg/core/node` 테스트 추가, `tests/wbft/accounts` preset 주석 정정(B2), `Node.Auth`
   명명 타입화(`type node.Auth map[string]any` + 문서), 노후 root README를 Go-first·
   다체인 상태로 재작성.

### 남은 작업 (기존)
- ✅ **레거시 bash 데이터 재배치(A2)**: bash 스키마 프로파일 9개(default/minimal/large/
  bft-limit/regression/hardfork-*)와 `templates/`를 `tests/regression/{profiles,templates}/`로
  이동(git mv, 이력 보존). 최상위 `profiles/`엔 Go 툴용 `wemix-upgrade.yaml`·`remote-example.yaml`·
  `custom/`만 남김. bash CLI 제거로 이 데이터는 런타임 미해석(주석/메타뿐)이라 이동 안전.
  `state/` 런타임은 이미 gitignore됨. → 최상위=Go 툴 데이터로 명확화("이게 툴 설정인가
  회귀 데이터인가" 혼란 해소). 이 데이터는 회귀 포팅 완료 시 tests/regression과 함께 폐기.
- ✅ **하이브리드 외부 매니페스트(프로젝트별 체인 공급)**: 사용자 결정 — first-party 체인은
  embed 유지하되, 프로젝트가 자기 체인을 `--manifest <path>`/`--genesis-template <path>`로
  공급(기존 family 재사용 시 chainbench 수정 0). `pkg/chains/external.Load`(매니페스트 파싱 →
  family를 이름으로 선택 → `protocol.ByName(manifest.protocol|id)` → 템플릿 로딩 →
  `registry.StaticPlugin`), Manifest에 optional `protocol` 필드(SDK 프로파일 차용), `setup`에
  `--manifest`/`--genesis-template` 배선. core 경계 유지(family switch는 composition 계층).
  end-to-end 확인(외부 foonet 체인 plan). parity 확대: `consensus`·`faucet`도 `--manifest`/
  `--genesis-template` 수용(`resolveChain`/`resolveAccountProvider`); verify/test는 nodeset.json의
  chain으로 동작(플래그 불필요). 잔여: hardfork/upgrade(복잡한 handoff)는 defer, 새 consensus
  family는 여전히 플러그인 필요.
- ✅ **A5 upgrade_run 하드코딩 제거**: `upgrade_run.go`의 `wemix-config.json`→
  `<fromID>-config.json`, `gwemix.ipc`→`<fromBin basename>.ipc`, `--http.api`의
  `wemix`/`istanbul`→`from/to.Family().RPCNamespace()`로 소싱. launch flag/경로에
  하드코딩된 chain 리터럴 0건.
- **h-hardfork 포팅(B-blk1, 부분)**: stablenet은 Anzeon/Boho가 genesis 활성이라 **post-fork
  state read**는 일반 케이스로 포팅 가능 — `p256-precompile-active`(0x100 precompile, NIST 벡터)·
  `p256-rejects-invalid`·`govminter-v2-code`(eth_getCode GOV_MINTER 비어있지 않음)·
  `boho-chain-config-active`(4종) 포팅(tests/anzeon, mock 검증). 포팅 66 케이스.
  **burn-refund 언블록**: govbind에 `CancelProposalCall`/`ClaimBurnRefundCall` + burn-refund
  이벤트 토픽 추가(단위검증), h-06/09/11 포팅(`burn-cancel-refundable`/`burn-execute-no-refundable`/
  `claim-zero-refund-reverts`, refundableBalance read + node-side 서명). burn-refund 잔여
  h-07/10/12/13(`DisapproveProposalCall` 바인딩 + burn-reject/claim-succeeds/double-reverts/
  events, 이벤트 토픽 FindLog) + explicit-gas(wallet에 `SendDynamicFeeGas`/`SendLegacyGas`/
  `SendAccessListGas` 커스텀 gas 3종 + h-20~24 boundary: exact/above min accept, below min reject)
  포팅. 포팅 78 케이스.
  **defer(인프라 선행)**: fork-transition before/after(delayed-fork harness 부재),
  h-20~24(커스텀 gasFeeCap send wallet 메서드 필요), h-49~51(빌드체크·체인테스트 아님),
  h-30/33/34(chainbench genesis에 없는 Extra test 계정), h-42~45(effectiveGasPrice/이벤트 순서).
- **A1(경미). 구 binary-swap 하드포크 refinement**: 검증된 핸드오프는 concurrent 모델
  (`pkg/consensus/upgrade`), `pkg/core/hardfork`는 균질 fork용 binary-swap으로 문서화됨.
  우선순위 낮음.
- ✅ **0x1/0x2 tx타입 SDK 바인딩 + a2-02/03 포팅(B-blk2)**: accounts.Wallet에 `SendDynamicFee`
  (0x02, EIP-1559: tip=MaxPriorityFeePerGas, feeCap=gasPrice+tip)·`SendAccessList`(0x01, EIP-2930,
  빈 access list, legacy-style gasPrice) 추가(SDK `tx.DynamicFeeTx`/`AccessListTx` 직접 사용,
  SDK 무변경). validation 단위테스트. 라이브 케이스 `dynamic-fee-tx`·`access-list-tx`(faucet
  wallet로 typed send → eth_getTransactionByHash.type == 0x2/0x1) 포팅. 포팅 62 케이스.
- **A4(부분). accounts governance/token 타입 바인딩** 🟢: ABI/tx/event 바인딩 계층은
  코어에 있음(위). 체인별 거버넌스·토큰의 타입드 프로파일 바인딩은 케이스별 refinement.
  → **S4로 승계**: 바인딩은 완비됐으나 **위치가 generic `pkg/accounts`** — 이전이 선행.
- **E1(잔여). 노드 lifecycle RemoteDriver** 🟢: 원격 RPC 접근은 됨. ssh로 노드를
  provision/launch하는 드라이버는 별도. → **S5로 승계·정정**(실은 구현·테스트 완료,
  배선만 남음).
- **D1. Svelte SPA** 🔴: 프론트 빌드 툴체인 필요. 현재 `pkg/dashboard/index.html`
  (build-free)이 SSE로 동작 — 데이터 계약(SSE Event JSON + `/api/runs`) 확정.
- **회귀 잔여 포팅**(진행 중): 노드측 서명 인프라 — `rpc.Client.Coinbase`+`SendTransaction`
  (httptest 단위검증). governance write 3종 포팅(공통 헬퍼 `tests/anzeon/gov_common.go`:
  discoverValidators/extractProposalID/approveToQuorum/proposalExecuted):
  `mint-proposal-executes`(f2-01), `burn-proposal-executes`(f2-02, payable proposeBurn),
  `validator-add-member-executes`(f3-01/02, GovValidator members() 확인). govbind에
  BurnProof/ProposeBurnCall 추가. registration/gating 검증됨. ⚠️ **live-tx라 실 gstable
  네트워크 없이는 실행 미검증** — quorum 수·proof 포맷은 회귀 f2/f3 앵커.
  read 케이스 추가(mock 완전검증): `account-blacklist-readable`(e-07 isBlacklisted),
  `basefee-minimum`(c-06)·`basefee-maximum`(c-07, baseFeePerGas가 anzeon 최소~최대 범위),
  `estimate-gas`(a3-04, eth_estimateGas >= 21000), `logs-query-well-formed`(a4-04, eth_getLogs
  배열 shape), `effective-gas-price`(a2-08, receipt.effectiveGasPrice). 포팅 32 케이스.
  잔여: blacklist write(node측 서명),
  c-anzeon basefee 버스트(타이밍), 나머지 f/e/h/z. 원본 bash는 stablenet 고정이라 Go 러너
  미실행(감사 B4).
- ✅ **C2 RemoteDriver CLI 노출**: `setup --remote-host/--remote-user/--remote-port`가
  RemoteDriver를 빌드·주입(`LaunchOptions.Driver`). **SSH 비밀번호는 `CHAINBENCH_REMOTE_PASS`
  env 전용**(cmdline 금지, 보안), host-key는 표준 SSH env로 해석. 원격이면 `--binary`는 원격
  경로(로컬 stat 안 함). end-to-end로 RemoteDriver.InitDatadir가 SSH로 genesis ship까지 도달
  확인(연결만 실패=예상). 단위테스트: 비밀번호 가드·드라이버 빌드.
  **identity 원격 shipping 완료**: 선택적 `driver.FileProvisioner`(RemoteDriver만 구현 → local
  무변경) 추가. remote일 때 `keyBase`를 `<DataRoot>/keys`로 두고 preset의 nodekey/keystore/
  password를 `shipIdentities`로 SSH ship + config KeystoreDir·launch args를 그 경로로 전환.
  local은 keyBase==keysAbs라 100% 동일. 단위테스트: `RemoteDriver.ProvisionFile`(base64+chmod),
  `shipIdentities`(경로 검증). CLI e2e로 provision→ship identities→init→launch가 SSH까지 도달
  확인. ⚠️ 실 over-SSH E2E green은 정상 환경 필요(S5 참고, 샌드박스 hang).
- **docs 정리(진행 중)**: 아래 §참고 문서 중 레거시 로드맵(REMAINING_WORK/NEXT_WORK/
  REFACTORING_PLAN/VISION_AND_ROADMAP)은 superseded 처리, `docs/superpowers/`는 역사 기록.

---

## 5. 확정된 설계 결정 (요약; 상세는 REDESIGN §0)

- D1 전면 Go. D2 MCP Go 재작성. D3 accounts는 `AccountProvider` 경계 + SDK 기본구현.
- D9 **1차 축 = 합의 알고리즘**(wbft=stablenet+wbft, poa=wemix). D10 accounts 다체인화.
- D7 genesis 하이브리드(family별 템플릿). D8 wbft 먼저, wemix 후.
- 대시보드 실시간 = **SSE** 확정.
- **A0 조사 결과(불변 사실)**: 0x16 fee-delegation은 **3체인 공통**(정적 소스 동등성 확인), 계정 `Extra` 비트맵은 **stablenet 전용**, wemix 시스템컨트랙트는 **Registry 런타임 해석**(고정주소 아님).

---

## 6. 함정·주의 (Gotchas)

- `replace ../accounts` — 다른 머신에선 두 리포를 형제로 클론해야 빌드됨. CI엔 부적합.
- 커밋 전 **feature 브랜치인지 확인**(main 커밋 차단 가드).
- `network/`는 nested 모듈 + internal → 루트에서 import 불가. 흡수 전엔 코드 재사용 불가(참고만).
- 테스트는 전부 httptest/fake로 결정적. **실 체인 바이너리 없이도 전체 green** — 그래서 C/D가 "이월"이지 미완이 아님(배선은 됨).
- genesis 템플릿은 consensus-critical만 채움 → 실 노드 부팅엔 systemContracts/alloc 파리티(§4 A4/이월) 필요할 수 있음. 확인은 C1에서.
- `chainbench setup --launch`는 `binary init`을 실행하므로 실 바이너리가 없으면 fake로만 검증됨.

---

## 7. 참고 자산 (외부)

- `../accounts` (`github.com/0xmhha/accounts`): 계정 SDK. `protocol`(다체인), `wallet`(faucet=SendCoin), `tx`(0x00–0x04+0x16), `signing.Scheme`(암호 격리), `account.Extra`(stablenet).
- `../script/wemix-upgrade`: wemix(poa,gwemix)↔wbft(geth) 하드포크 셋업 스크립트. poa 부트스트랩(node1→governance→etcd→others)·remote 모드 레퍼런스.
- `../chain/{go-stablenet,go-wbft,go-wemix}`: 대상 체인 소스. genesis 구조(anzeon/croissant/base), tx 인코딩, consensus namespace 확인처.

---

## 8. 이어서 진행하는 법 (체크리스트)

1. 두 리포를 형제 디렉토리로 클론(또는 존재 확인): `chainbench`, `accounts`.
2. `docs/CHAINBENCH_GO_REDESIGN.md` §0.1 현황 + 이 문서 §2/§4 읽기.
3. `go build ./... && go test ./...`로 green 확인(accounts도).
4. 새 feature 브랜치 생성(main 아님).
5. §4의 A1 또는 A2부터 착수. 각 증분: 코어 → 테스트(httptest/fake) → gofmt/vet → 커밋(영어, 이모지·푸터 없음).
6. 응집된 단위가 쌓이면 PR 생성(base main). 사용자가 squash-merge·동기화.
7. 맥락 더 필요하면 `docs/dev/session-data/session-305c46ea.jsonl`(원본 세션 transcript) 참고.
