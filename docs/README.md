# docs/ — 문서 인덱스

chainbench 는 **go-stablenet / wbft / wemix 용 Go-first 다체인 테스트벤치**다.
사용자가 체인 환경과 테스트를 선언하면 로컬·docker·원격 서버에 실제 다중 노드 체인을
구성하고, 테스트 중 노드를 개별 제어하며, 실행 증적을 모아 보고서를 낸다.
그 목표의 정본은 [`dev/chainbench-system-direction.md`](dev/chainbench-system-direction.md) 다.

> 인덱스 갱신: **2026-09-12** (`ea486e50`). 문서를 더하거나 옮기면 이 표도 같이 고친다 —
> 등급 체계는 인덱스가 정직할 때만 작동한다.

## 0. 무엇을 알고 싶은가

| 질문 | 여기부터 |
|---|---|
| **쓰는 법이 궁금하다** — 키·MCP·DSL·설정 파일 | [`guide/`](#1-guide--쓰는-법) |
| **무엇을 만들기로 했나** — 요구·계약·인터페이스 | [정본 4종](#2-정본--설계-ssot-4종-docsdev) |
| **다음에 무엇을 하나** | [`dev/chainbench-worklist.md`](dev/chainbench-worklist.md) — **작업 순서·상태의 단일 출처** |
| **지금 어떤 구조를 향하나** | [`dev/architecture/`](#4-devarchitecture--구조와-측정) · [현행 설계 문서들](#3-dev--현행-설계) |
| **코드가 실제로 어떤 모양인가** | [`dev/architecture/code-graph.md`](dev/architecture/code-graph.md) (AST 실측) · [`dev/codegraph/`](dev/codegraph/README.md) (호출 그래프) |
| **체인 바이너리의 플래그·RPC 가 궁금하다** | [`chain-analysis/`](chain-analysis/README.md) — **체인 소스를 읽기 전에 여기부터** |
| **테스트는 어디에 있나** | [`../tests/README.md`](../tests/README.md) |
| **왜 이렇게 결정했었나** | [`dev/archive/`](dev/archive/README.md) |

## 문서 등급과 권위 순서

문서끼리 어긋날 때 **무엇이 이기는지**를 먼저 정한다. `dev/` 의 모든 문서는 제목 바로
아래에 자기 등급을 선언한다.

| 등급 | 무엇을 말하는가 | 어긋나면 |
|---|---|---|
| **[정본]** | *무엇을 만들어야 하는가* — 요구·계약·인터페이스·작업 순서 | **정본이 이긴다.** 설계 제안을 고친다. |
| **[현행 설계]** | *지금 어떻게 만들 것인가* — 목표 구조 | 정본에 진다. 코드에 이긴다(코드가 아직 안 따라온 것). |
| **[측정]** | *언제 재보니 이랬다* | 기준 커밋이 붙는다. 다시 뽑으면 갱신된다 — **어긋나면 코드가 이긴다.** |
| **[이력]** | *그때 무엇을 측정·결정했는가* | **현재 상태를 말하지 않는다.** 근거로 인용할 수 없다. |
| **[대체됨]** | 제안이 구현됐거나 다른 문서로 옮겨감 | [`dev/archive/`](dev/archive/README.md) 로 이동. 새 작업의 근거 금지. |

위 등급 체계는 **chainbench 자신의 설계·작업 문서**(`dev/`)에 적용된다. 그 정본은
4종뿐이다. [`chain-analysis/`](chain-analysis/README.md) 와 [`claudedocs/`](claudedocs/README.md)
는 이 축 밖에 있다 — 우리 설계가 아니라 **외부 대상의 기록**이라 4종과 경쟁하지 않는다.

> 문서를 오래됐다고 지우지 않는다. 지우면 근거가 사라진다. 위험한 것은 오래된 문서가
> 아니라 **오래됐다고 표시되지 않은 문서**다 — 등급 표기가 그 표시다.

## 1. `guide/` — 쓰는 법

| 문서 | 내용 |
|---|---|
| [`guide/keyring.md`](guide/keyring.md) | **keyring 사용 설명서** — 키 세트 개념·명령 전부·가져오기 출처(니모닉/파일/`srv://`)·원격 환경변수·`--docker` 모드 · **수동 검증 체크리스트**(자동 테스트와 1:1 대응). |
| [`guide/mcp.md`](guide/mcp.md) | chainbench MCP 서버 사용 가이드 — 도구 호출, `.mcp.json` 등록. |
| [`guide/dsl-authoring.md`](guide/dsl-authoring.md) | **DSL 작성 가이드** — 코드에서 뽑은 등록 어휘 전수(액션·단언·리더)와 각 인자 키. `tests/tc` 케이스를 쓰는 사람이 읽는 문서. |
| [`guide/config-files.md`](guide/config-files.md) | **`server-set.yaml` 과 `workspace-config.yaml`** — 하나의 DSL 이 local/docker/remote 에서 그대로 돌게 하는 두 파일을 만드는 법. 실제 파일은 커밋되지 않는다(`*.sample.yaml` 만 추적). |
| [`guide/topology.md`](guide/topology.md) | **per-node 토폴로지** — `--topology` 파일 형식과 세 레이아웃 출처(counts → topology → blueprint)의 우선순위. |
| [`guide/validator-set.md`](guide/validator-set.md) | **`validator set`** — 커밋된 5노드 preset 보다 큰 검증자 세트를 만드는 법. |
| [`SECURITY_KEY_HANDLING.md`](SECURITY_KEY_HANDLING.md) | 키 취급 보안 정책·위협 모델. 프리셋 키는 **테스트 픽스처 전용**. |

## 2. 정본 — 설계 SSoT 4종 (`docs/dev/`)

| 문서 | 성격 |
|---|---|
| [`dev/chainbench-requirements-review.md`](dev/chainbench-requirements-review.md) | 요구사항 37 · 사양 검토 · 코드 격차 · **etcd flaky 실체** · 동시성/안전성. |
| [`dev/chainbench-design.md`](dev/chainbench-design.md) | **아키텍처 SSoT** — 구조·패키지 인터페이스(§3)·데이터 모델(§4)·동시성(§6)·마이그레이션. |
| [`dev/chainbench-feature-spec.md`](dev/chainbench-feature-spec.md) | F1~F16 동작 계약 · 수용기준(AC). |
| [`dev/chainbench-worklist.md`](dev/chainbench-worklist.md) | **[정본] 작업 순서·상태의 단일 출처.** 무엇을 다음에 하는지는 여기서 읽는다. |

## 3. `dev/` — 현행 설계

### 제품 방향

| 문서 | 내용 |
|---|---|
| [`dev/chainbench-system-direction.md`](dev/chainbench-system-direction.md) | **[제품 목표·확정 방향] 2026-09-02 사용자 확인** — local/remote/Docker 원격 모사, 자료 재사용, PID·command·노드별 제어, config/contract 테스트, DSL 사전검사, 환경 재사용, Node Monitor, 실행 증적과 최종 report. |
| [`dev/refactoring-follow-up-handoff-2026-09-02.md`](dev/refactoring-follow-up-handoff-2026-09-02.md) | **[현행 설계 보조] 후속 작업 인수인계** — AST·문서 대조에서 확인한 genesis identity 결함과 E0A~E9 의 완료 조건. 작업 상태는 worklist §1k 가 이긴다. |

### 구성 요소별 설계

| 문서 | 내용 |
|---|---|
| [`dev/keyring-design.md`](dev/keyring-design.md) | **키 자료의 단일 소유자** — 키가 세 체인 동일함을 실증 · BLS 를 Go 로 파생(외부 바이너리 제거) · preset 이 신원/결정/산출물을 섞어 담은 문제와 분해. |
| [`dev/key-and-material-design.md`](dev/key-and-material-design.md) | **키·자료 소유 구조** — 신원 타입 5개와 키 패키지 실측 · **재업로드 방지가 존재 여부만 봐서 틀린 자료를 재사용하는 결함** · destination 레이아웃(`bin`/`material`/`run`, 내용 해시 경로). |
| [`dev/netmap-design.md`](dev/netmap-design.md) | **노드 배치의 단일 소유자** — 배치 타입·포트 표현·역할 어휘·static-nodes 조립의 중복 실측과 통합 설계 · 피어링 그래프 파생. 통합된 결과가 지금의 `internal/resource` 다. |
| [`dev/network-blueprint-design.md`](dev/network-blueprint-design.md) | **네트워크 청사진** — 구성 요소 전수(네트워크 14 · 노드 13 · 연결 4)와 누락 지점 · 선언→해석→물질화 3단계 · preset 을 전제에서 **생성기**로 강등 · 출처 사슬. |
| [`dev/family-bringup-design.md`](dev/family-bringup-design.md) | **패밀리별 기동 설계** — 상위 일관/하위 특화. 3체인 차이 실측표 · 4 boundary(BringUpPhases·Action·GenesisArtifacts·PortReservation). |
| [`dev/server-set.md`](dev/server-set.md) | **서버 세트** — 노드 포트·호스트·접속 정보의 단일 출처(`server-set.yaml`, gitignore). local/remote 동일 구조·포트 규칙·자격증명 취급. 만드는 법은 [`guide/config-files.md`](guide/config-files.md). |
| [`dev/docker-remote-design.md`](dev/docker-remote-design.md) | **로컬 docker 를 원격 서버처럼** — 인벤토리는 실주소 유지, 접속 경계 4곳에서만 `AddrMap` 치환 · 함정 4개(loopback 판정·산출물 오염 등). |
| [`dev/surface-unification-design.md`](dev/surface-unification-design.md) | **표면 통일 리팩토링** — 기능을 한 번 등록하면 CLI/MCP/DSL 이 렌더링. **[일부 대체됨 2026-09-05]** — 해법의 형태가 "세 표면이 모두 app 을 지나고 게이트는 기능별 동등성 테스트" 로 바뀌었다. 규칙의 정본은 `architecture-v2` §2. |
| [`dev/dsl-v2-proposal.md`](dev/dsl-v2-proposal.md) | **DSL v2 문법** + x-bar 정렬 갭 분석(G1~G7). T7.8 에서 구현됨. 쓰는 법은 [`guide/dsl-authoring.md`](guide/dsl-authoring.md). |
| [`dev/dashboard-metrics-design.md`](dev/dashboard-metrics-design.md) | **대시보드 metric 시각화** — Prometheus·Grafana 서버 없이 자체 동작한다는 결정과 근거 · 목표 구조(스크레이프→링버퍼→SSE→차트) · 참조 오픈소스 5종(라이선스 포함). |

### 절차와 이관

| 문서 | 내용 |
|---|---|
| [`dev/chain-setup/`](dev/chain-setup/README.md) | **체인 구성 절차** — 공통 파이프라인과 변곡점, 케이스 4종(wemix · wemix→wbft · wbft · stablenet), `cli-steps`. 검증 기준일 2026-08-09 의 실측이며, 각 문서 상단에 **명령 표면 정정(2026-09-11)** 이 붙어 있다. |
| [`dev/legacy-test-migration.md`](dev/legacy-test-migration.md) | **레거시 셸 스위트 → DSL 이관** — 양쪽을 파싱해 만든 대응표와 남은 계획. local 계열(basic·fault·stress·anzeon)은 이관·라이브 검증 완료. |
| [`dev/legacy-port-audit/`](dev/legacy-port-audit/README.md) | **[측정] 이관 감사** — 셸 460파일과 신규 Go+DSL 을 각각 AST 그래프로 만들어(노드 943 / 1,413) 테스트 단위로 대조한 기록. |
| [`dev/wemix4-port-tracker.md`](dev/wemix4-port-tracker.md) | wemix4 케이스 포팅 추적(covered / ported / deferred). 상단에 **경로 정정** — 판정은 유효하고 위치만 `tests/tc`·`tests/e2e` 로 옮겼다. |
| [`dev/monitoring-issue-review-2026-09-10.md`](dev/monitoring-issue-review-2026-09-10.md) | **모니터링 이슈 16건 재판정과 수정 계획** (기준 `2cc82692`). 그중 14건은 `9b6b0930` 에서 해소됐다. |
| [`dev/chain-handover-2026-09-12.md`](dev/chain-handover-2026-09-12.md) | **[측정] 체인팀 인계 3건** — chainbench 를 고쳐서는 해소되지 않는 go-wemix / go-wbft 결함. `verifyBlockSig` 의 nil 역참조 패닉(원인 확정), 부트 etcd 붕괴(R6, 범위 좁힘), `istanbul_getWbftExtraInfo` 의 블록 태그. 증상·근거·재현·제안까지. |

## 4. `dev/architecture/` — 구조와 측정

| 문서 | 등급 | 내용 |
|---|---|---|
| [`architecture-v2.md`](dev/architecture/architecture-v2.md) | [현행 설계] | **아키텍처 v2 (2026-08-25 결정)** — CLI 는 core 직접·MCP 는 app 경유, 자원/노드정보 소유, low level 파라미터 주입, 소비자 측 interface 노출, 모듈 네이밍 규칙 7. **모듈 경계는 이 문서가 이긴다.** |
| [`layers.md`](dev/architecture/layers.md) | [현행 설계] | **레이어 아키텍처** — L0~L6 정의 · 패키지 전수 배치 · 의존 규칙 · **상태 소유 규칙**(control plane=session / data plane=FileSink) · `internal/arch` 가 기계로 강제하는 규칙. |
| [`module-responsibilities.md`](dev/architecture/module-responsibilities.md) | [현행 설계] | **관심사별 소유 모듈** 16개 · 소유자 부재 실측 · **3체인 실행 시뮬레이션**(분기점은 genesis·기동순서 2개뿐) · DSL 파서 4분할. |
| [`module-plan.md`](dev/architecture/module-plan.md) | [현행 설계] | **모듈 재편 계획** — 자원·노드정보·프로세스 3모듈 + genesis·nodeconfig·dsl 빌더 3종 · 합칠 것과 지울 것 · P1~P8 단계와 게이트. |
| [`consolidation-plan.md`](dev/architecture/consolidation-plan.md) | [현행 설계] | **통폐합 계획 (2026-08-31 사용자 확정)** — `core` 아래 평평한 형제들을 관심사 단위로. 확정된 목표 구조와 이동표. |
| [`target-architecture.md`](dev/architecture/target-architecture.md) | [현행 설계] | **목표 아키텍처 다이어그램 8종** — 디렉토리/호출 두 축 분리 · 레이어 · 청사진 파이프라인 · 패밀리 분기 · 키 파생 · 피어링 그래프 · 표면 통일. |
| [`f1-recovery.md`](dev/architecture/f1-recovery.md) | [현행 설계] | **F1 파일 영속·복구** — 프로세스가 죽어도 다시 실행하면 이전 진행을 이어받는다. §0 원칙: 복구용 사본을 만들지 않는다. |
| [`code-graph.md`](dev/architecture/code-graph.md) | [측정] | **AST 실측 패키지 그래프** — 2026-09-11 재측정(70패키지 · 225엣지 · 51,151줄 · **층 위반 0**), 레이어별 규모와 fan-in/out, 자원 소유자로의 수렴. 다시 뽑기: `go run ./scripts/inventory/code-graph .` |
| [`code-health-review-2026-09-10.md`](dev/architecture/code-health-review-2026-09-10.md) | [측정] | **코드 건강도 검토** — 같은 수치 위에서 함수 길이·분기·중첩·완전 중복까지. `shellQuote` 4중복 등 **네 갈래 중복**과 패키지 문서 공백 17개. 수정은 아직 하지 않았다(worklist §1s). |
| [`../codegraph/`](dev/codegraph/README.md) | [측정] | **호출(선택자) 그래프** — 패키지 그래프가 답하지 않는 "누가 무엇을 부르는가". `codegraph.json` + 파이프라인 mermaid. |

## 5. 하위 디렉토리 — 외부 축

| 경로 | 내용 |
|---|---|
| [`chain-analysis/`](chain-analysis/README.md) | **체인 바이너리의 CLI 표면과 배선** — gstable · gwbft · gwemix 각각의 명령/플래그 그래프와 RPC/metrics 그래프. 실행 옵션 질문은 **체인 소스를 읽기 전에 여기부터.** 바이너리에서 재생성되며 기준 체인 커밋을 스스로 적는다. 대상이 외부 바이너리라 설계 4종과는 별개 축이다. |
| [`claudedocs/`](claudedocs/README.md) | **[이력] 외부 컨텍스트** — chainbench 가 속한 상위 자동화 시스템의 제안서/지시서(2026-04). 제안 당시 문서라 존재하지 않는 명령이 그대로 있다. 명령 표면은 `chainbench --help` 가 이긴다. |
| `research/` | 다른 세션이 관리하는 분석 트리. 저장소에 추적되는 것은 [`08-dsl-15-node-analysis-response.md`](research/chainbench/analyses/08-dsl-15-node-analysis-response.md)(15노드 DSL 구성, 코드 변경 전 검토) 하나다. 여기서 참조하는 다른 번호 문서는 그 세션의 트리에 있고 이 저장소에는 없다. |
| [`dev/archive/`](dev/archive/README.md) | **대체·완료된 문서.** 지우지 않고 옮긴다 — 결정의 근거는 결정 자체와 별개의 정보다. 각 문서에 "무엇으로 대체됐나" 가 붙어 있다. |

> `dev/session-data/`(원본 세션 transcript)는 검증용 로컬 자료로 **git 미추적**(`.gitignore`).
