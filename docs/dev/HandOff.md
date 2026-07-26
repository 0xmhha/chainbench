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

### 남은 작업
- **A1(경미/부차). 구 binary-swap 하드포크의 target namespace config 재생성**: 검증된
  핸드오프는 concurrent 모델(`pkg/consensus/upgrade`)이며 `pkg/core/hardfork`는 균질
  fork용 binary-swap으로 문서화됨. 이 refinement는 우선순위 낮음.
- **A4(부분). accounts governance/token 타입 바인딩** 🟢: ABI/tx/event 바인딩 계층은
  코어에 있음(위). 체인별 거버넌스·토큰의 타입드 프로파일 바인딩은 케이스별 refinement.
- **E1(잔여). 노드 lifecycle RemoteDriver** 🟢: 원격 RPC 접근은 됨. ssh로 노드를
  provision/launch하는 드라이버는 별도.
- **D1. Svelte SPA** 🔴: 프론트 빌드 툴체인 필요. 현재 `pkg/dashboard/index.html`
  (build-free)이 SSE로 동작 — 데이터 계약(SSE Event JSON + `/api/runs`) 확정.
- **회귀 잔여 포팅**: f-system-contracts 거버넌스 write(mint/burn/proposal 다단계),
  c-anzeon basefee 버스트(타이밍 민감) 등. 바인딩 계층이 완비돼 케이스별 반복 작업.
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
