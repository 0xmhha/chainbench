# docs/ — 문서 인덱스

chainbench는 **go-stablenet/wbft/wemix용 Go-first 다체인 테스트벤치**다. 이 디렉토리는
현행 설계·운영 문서를 둔다. (Go 재설계 이전 bash/TS 아카이브와 구 재설계 SSoT는 정리되었다.)

## 문서 등급과 권위 순서

문서끼리 어긋날 때 **무엇이 이기는지**를 먼저 정한다. `dev/` 의 모든 문서는 제목 바로
아래에 자기 등급을 선언한다.

| 등급 | 무엇을 말하는가 | 어긋나면 |
|---|---|---|
| **[정본]** | *무엇을 만들어야 하는가* — 요구·계약·인터페이스·작업 순서 | **정본이 이긴다.** 설계 제안을 고친다. |
| **[현행 설계]** | *지금 어떻게 만들 것인가* — 목표 구조 | 정본에 진다. 코드에 이긴다(코드가 아직 안 따라온 것). |
| **[이력]** | *그때 무엇을 측정·결정했는가* | **현재 상태를 말하지 않는다.** 근거로 인용할 수 없다. |
| **[대체됨]** | 제안이 구현됐거나 다른 문서로 옮겨감 | [`dev/archive/`](dev/archive/) 로 이동. 새 작업의 근거 금지. |

위 등급 체계는 **chainbench 자신의 설계·작업 문서**(`dev/`)에 적용된다. 그 정본은
4종뿐이다 — `chainbench-requirements-review` (요구·결정) · `chainbench-feature-spec`
(동작 계약) · `chainbench-design` (인터페이스·데이터 모델) · `chainbench-worklist`
(**작업 순서·상태의 단일 출처**).

[`chain-analysis/`](chain-analysis/) 는 이 축 밖에 있다. 우리 설계가 아니라 **외부 체인
바이너리의 실측**이고, 바이너리에서 다시 뽑는다. 그 대상에 한해서는 그쪽이 정본이지만,
위 4종과 경쟁하지 않는다.

> 문서를 오래됐다고 지우지 않는다. 지우면 근거가 사라진다. 위험한 것은 오래된 문서가
> 아니라 **오래됐다고 표시되지 않은 문서**다 — 등급 표기가 그 표시다.

## 설계 SSoT — 여기서 시작 (`docs/dev/`, 4종 세트)

| 문서 | 성격 |
|---|---|
| [`dev/chainbench-requirements-review.md`](dev/chainbench-requirements-review.md) | 요구사항 37 · 사양 검토 · 코드 격차 · **etcd flaky 실체** · 동시성/안전성. |
| [`dev/chainbench-design.md`](dev/chainbench-design.md) | **아키텍처 SSoT** — 구조·패키지 인터페이스(§3)·데이터 모델(§4)·동시성(§6)·마이그레이션. |
| [`dev/chainbench-feature-spec.md`](dev/chainbench-feature-spec.md) | F1~F16 동작 계약 · 수용기준(AC). |
| [`dev/chainbench-worklist.md`](dev/chainbench-worklist.md) | **[정본] 작업 순서·상태의 단일 출처.** 무엇을 다음에 하는지는 여기서 읽는다. 현재 착수 지점은 §1i 의 P1.1(모듈 재편). |

## 현행 참고 문서

| 문서 | 내용 |
|---|---|
| [`MCP_USAGE.md`](MCP_USAGE.md) | chainbench MCP 서버 사용 가이드(도구 호출, `.mcp.json` 등록). |
| [`SECURITY_KEY_HANDLING.md`](SECURITY_KEY_HANDLING.md) | 키 취급 보안 정책·위협 모델. 프리셋 키는 **테스트 픽스처 전용**. |
| [`KEYRING_USAGE.md`](KEYRING_USAGE.md) | **keyring 사용 설명서** — 키 세트 개념·명령 전부·가져오기 출처(니모닉/파일/srv://)·원격 환경변수·`--docker` 모드 · **수동 검증 체크리스트**(자동 테스트와 1:1 대응). |

### `docs/` 루트의 이력 문서

| 문서 | 등급 | 내용 |
|---|---|---|
| [`REMOTE_WEMIX_DEPLOY_DESIGN.md`](REMOTE_WEMIX_DEPLOY_DESIGN.md) | [이력] | wemix4 의 SSH 배포·하드포크 스위트를 Go 로 옮기기 위한 **착수 전 설계**(2026-08). 1~5 단계는 이미 구현됐다 — 원격 키 읽기는 `internal/core/remote/files.go`, 원격 파일 접근은 `internal/core/driver/remote_store.go`, 클러스터·계획·부트스트랩·핸드오프는 `internal/chains/wemix/deploy/` 에 있다. 본문은 이것들을 앞으로 만들 것처럼 서술하므로 **현재 동작의 근거로 인용하지 않는다.** 지금 무엇이 되는지는 코드와 [`internal/chains/wemix/deploy/README.md`](../internal/chains/wemix/deploy/README.md), 남은 일은 worklist 가 말한다. |

## `dev/` 개발 문서

| 문서 | 내용 |
|---|---|
| `dev/chainbench-*.md` | 위 설계 SSoT 4종. |
| [`dev/chainbench-refactoring.md`](dev/chainbench-refactoring.md) | **[이력]** pkg→internal 시기의 리팩토링 감사(WP1~6, 대부분 완료). 정본이 아니다 — 그때 무엇을 유지·재구성하기로 했는지의 기록이며, 현재 상태는 worklist 와 코드가 말한다. |
| [`dev/keys-generate.md`](dev/keys-generate.md) · [`dev/topology.md`](dev/topology.md) | 프리셋 키 생성 · 로컬 토폴로지 설정 가이드. |
| [`dev/keyring-design.md`](dev/keyring-design.md) | **keyring 설계 (첫 착수 대상)** — 키가 세 체인 동일함을 실증 · BLS 를 Go 로 파생(외부 바이너리 제거) · preset 이 신원/결정/산출물을 섞어 담은 문제와 분해 · 5패키지 → 1 · 명령. 작업 순서는 worklist §1g. |
| [`dev/chainbench-system-direction.md`](dev/chainbench-system-direction.md) | **[제품 목표·확정 방향] 2026-09-02 사용자 확인** — local/remote/Docker 원격 모사, 자료 재사용, PID·command·노드별 제어, config/contract 테스트, DSL 사전검사, 환경 재사용, Node Monitor, 실행 증적과 최종 report. 작업 순서는 worklist §1k. |
| [`dev/refactoring-follow-up-handoff-2026-09-02.md`](dev/refactoring-follow-up-handoff-2026-09-02.md) | **[현행 설계 보조] 후속 작업 인수인계** — AST·문서 대조에서 확인한 genesis identity 결함과 E0A~E9의 완료 조건. 작업 상태는 worklist §1k가 이긴다. |
| [`dev/network-blueprint-design.md`](dev/network-blueprint-design.md) | **네트워크 청사진** — 구성 요소 전수(네트워크 14 · 노드 13 · 연결 4)와 누락 지점 · 선언→해석→물질화 3단계 · preset 을 전제에서 **생성기**로 강등 · 출처 사슬. 작업 순서는 worklist §1g. |
| [`dev/surface-unification-design.md`](dev/surface-unification-design.md) | **표면 통일 리팩토링** — 기능을 한 번 등록하면 CLI/MCP/DSL 이 렌더링. **명령 표면 재설계**(72 엔드포인트 → 약 32, 최상위 26 → 9) · `cmd/` 박막화(4,569줄 중 21파일이 app 우회) · **모듈 인벤토리**(신설 5·변경 6·삭제 6) · 3단계 골격(Compose/Test/Report) · **3체인 구성요소 8개와 스텝 9개의 공통/특화 분리**(특화는 5지점뿐). 작업 순서는 worklist §1g. |
| [`dev/family-bringup-design.md`](dev/family-bringup-design.md) | **패밀리별 기동 설계 제안** — 상위 일관/하위 특화. 3체인 차이 실측표 · 4 boundary(BringUpPhases·Action·GenesisArtifacts·PortReservation) · 비목표 · 열린 질문. 작업 순서는 worklist §1g. |
| [`dev/handoff-2026-08-22.md`](dev/handoff-2026-08-22.md) | **[이력]** 2026-08-22 인수인계 — K 완결·A1~A4½·NM1/NM1b 까지의 지도, NM2 착수 컨텍스트, 잊기 쉬운 이행기 장치 목록. |
| [`dev/dashboard-metrics-design.md`](dev/dashboard-metrics-design.md) | **대시보드 metric 시각화** — Prometheus·Grafana 서버 없이 자체 동작한다는 결정과 근거 · 목표 구조(스크레이프→링버퍼→SSE→차트) · 참조할 오픈소스 코드베이스 5종(라이선스 포함). 작업 순서는 worklist §1g D. |
| [`dev/docker-remote-design.md`](dev/docker-remote-design.md) | **로컬 docker 를 원격 서버처럼** — 인벤토리는 실주소 유지, 접속 경계 4곳에서만 `AddrMap` 치환 · wemix-bp-test 의 LocalMap 선례 분석 · 함정 4개(loopback 판정·산출물 오염 등) · keyring 라이브 검증 기록. 작업 순서는 worklist §1g R. |
| [`dev/netmap-design.md`](dev/netmap-design.md) | **노드 배치의 단일 소유자** — 배치 타입 8개·포트 표현 3벌(etcd 소실)·역할 어휘 3벌·static-nodes 조립 4벌 실측 · `core/netmap`(L1) 설계 · 피어링 그래프 파생(N0b) · 표면은 `chain place` 산출 강화. 작업 순서는 worklist §1g. |
| [`dev/key-and-material-design.md`](dev/key-and-material-design.md) | **키·자료 소유 구조** — 신원 타입 5개와 키 6패키지(1,565줄) 실측 · **`keyreg` 는 keyring 의 `import` 다**(주입 이유를 K1 이 없앰) · god function 2건 · **재업로드 방지가 존재 여부만 봐서 틀린 자료를 재사용하는 결함** · destination 레이아웃(`bin`/`material`/`run`, 내용 해시 경로). 작업 순서는 worklist §1g. |
| [`dev/server-set.md`](dev/server-set.md) | **서버 세트** — 노드 포트·호스트·접속 정보의 단일 출처(`server-set.yaml`, gitignore). local/remote 동일 구조·포트 규칙·자격증명 취급. |
| [`dev/wemix4-port-tracker.md`](dev/wemix4-port-tracker.md) · [`dev/wemix4-migration-plan.md`](dev/wemix4-migration-plan.md) · [`dev/repro-migration-remaining.md`](dev/repro-migration-remaining.md) | wemix4 테스트 포팅·마이그레이션 추적. 참조하는 `tests/wemix4/`(`lib/*.sh` 포함)는 **외부 읽기 전용 트리**이며 이 저장소에 없다. |
| [`dev/chain-setup/`](dev/chain-setup/) | 체인 구성 절차 — 공통 파이프라인과 변곡점, 케이스 4종(wemix · wemix→wbft · wbft · stablenet), `cli-steps`, 자동화 인수인계. 검증 기준일 2026-08-09 의 실측이다. |
| [`dev/remaining-work.md`](dev/remaining-work.md) | 잔여 작업(DSL 이관·레거시 은퇴·후속 발견), 기준일 2026-08-14. **상태는 worklist 가 이긴다.** |
| [`dev/legacy-retirement-plan.md`](dev/legacy-retirement-plan.md) · [`dev/chainbench-component-architecture.md`](dev/chainbench-component-architecture.md) · [`dev/chainbench-audit-2026-08-09.md`](dev/chainbench-audit-2026-08-09.md) | **[이력]** 각각 레거시 은퇴 계획 · 컴포넌트 아키텍처 · 2026-08-09 감사. 현재 상태의 근거로 인용하지 않는다. |
| [`dev/stablenet-post-v1.0.0-change-test-catalog.md`](dev/stablenet-post-v1.0.0-change-test-catalog.md) | 폐기된 bash 회귀 테스트 51건의 **케이스 카탈로그**(포팅 참고용). 코드가 아니라 목록이다. |

### `dev/architecture/` — 아키텍처 · 다이어그램

이 묶음은 **등급이 섞여 있다.** `[현행 설계]` 는 지금 향하는 목표를 말한다.
`[이력]` 은 작성 시점의 스냅샷이라 현재 코드와 어긋날 수 있고, 근거로 인용할 수
없다. 어느 쪽인지는 아래 표의 등급 열이 말한다.

| 문서 | 등급 | 내용 |
|---|---|---|
| [`dev/architecture/architecture-v2.md`](dev/architecture/architecture-v2.md) | [현행 설계] | **아키텍처 v2 (2026-08-25 결정)** — CLI 는 core 직접·MCP 는 app 경유, netmap 이 서버 정보·자원 분배·enode·접근 wrapper 소유, low level 파라미터 주입, 소비자 측 interface 노출, 모듈 네이밍 규칙 7. 모듈 경계는 이 문서가 이긴다. 작업은 worklist §1h. |
| [`dev/architecture/module-plan.md`](dev/architecture/module-plan.md) | [현행 설계] | **모듈 재편 계획(2026-08-27 실측)** — 자원·노드정보·프로세스 3모듈 + genesis·nodeconfig·dsl 빌더 3종 · 관심사별 현 위치 실측(노드 타입 10개·기동 진입점 8개·경로 계산 4곳) · 합칠 것과 지울 것 · P1~P8 단계와 게이트 · M1 이름 후보 5. 순서는 worklist 가 이긴다. |
| [`dev/architecture/v2-move-map.md`](dev/architecture/v2-move-map.md) | [현행 설계] | 아키텍처 v2 **이동표** — 8개 패키지 541 심볼 실측, 태스크별(V1~V6) 파일·심볼 목적지. |
| [`dev/architecture/software-architecture.md`](dev/architecture/software-architecture.md) | [이력] | 전체 소프트웨어 아키텍처 — 계층·컨텍스트·실행모델·환경 5요소·검증원·동시성. 작성 시점 기준이라 패키지 경로는 현재와 다를 수 있다. |
| [`dev/architecture/component-diagram.md`](dev/architecture/component-diagram.md) | [이력] | 컴포넌트 맵 · C4 컨테이너 뷰 · 키 소싱 컴포넌트 · 목표 델타. |
| [`dev/architecture/sequence-diagrams.md`](dev/architecture/sequence-diagrams.md) | [이력] | 전체 run · BuildEnv · Interpreter · 원자 스텝 CLI · 원격 SSH · 실패 경로. |
| [`dev/architecture/state-diagrams.md`](dev/architecture/state-diagrams.md) | [이력] | TestRun · Environment · NodeProcess · Session/키 · 워크스페이스 스텝. |
| [`dev/architecture/target-architecture.md`](dev/architecture/target-architecture.md) | [현행 설계] | **목표 아키텍처 다이어그램 8종** — 디렉토리/호출 두 축 분리 · 레이어 · 청사진 파이프라인 · 패밀리 분기 2곳 · 키 파생 · 피어링 그래프 · 표면 통일. 산문 결정을 그림으로 검토하는 용도. |
| [`dev/architecture/layers.md`](dev/architecture/layers.md) | [현행 설계] | **레이어 아키텍처** — L0~L6 정의 · 57개 패키지 전수 배치 · 의존 규칙(상향 0건 실측) · **상태 소유 규칙**(control plane=session / data plane=FileSink)과 위반 6곳 · 규칙 강제 테스트 제안. 작업 순서는 worklist §1g. |
| [`dev/architecture/module-responsibilities.md`](dev/architecture/module-responsibilities.md) | [현행 설계] | **관심사별 소유 모듈**(체인구성 5요소·노드생명주기·DSL 등 16개) · 현재 소유자 부재 실측(genesis 17곳·키 17곳·노드 11곳) · **3체인 실행 시뮬레이션**(분기점은 genesis·기동순서 2개뿐) · DSL 파서 4분할 제안(dsl · assert · bind · interp). 작업 순서는 worklist §1g. |
| [`dev/architecture/code-graph.md`](dev/architecture/code-graph.md) | 측정 | AST 실측 패키지 그래프 — **§2~§3 은 2026-08-27 재측정**(75패키지·268엣지·46.6k줄 · 계층 위반 0 · netmap 계열 서브그래프), §4~§5 는 launchopt 브랜치 **[이력]**. 다시 뽑으면 갱신된다: `go run ./scripts/inventory/code-graph .` |

### 2026-08-11 재설계 검토

| 문서 | 등급 | 내용 |
|---|---|---|
| [`dev/dsl-v2-proposal.md`](dev/dsl-v2-proposal.md) | 현행 설계 | DSL v2 문법 + x-bar 정렬 갭 분석(G1~G7). T7.8 에서 구현됨. |
| [`dev/chain-binary-flag-graph.md`](dev/chain-binary-flag-graph.md) | 이력 | 3체인 바이너리 CLI 그래프(AST 추출) · 실행옵션 모듈+builder 설계 비판 검토. |

### [`dev/archive/`](dev/archive/) — 대체됨

| 문서 | 무엇으로 대체됐나 |
|---|---|
| `structure-and-atomic-cli-proposal.md` | 제안한 `internal/app` · `internal/core/launchopt` 는 **구현 완료**. 남은 표면 논의는 `surface-unification-design`. |
| `chain-cli-execution-plan.md` | 자체 헤더가 "진행 정본은 worklist" 라고 선언한다. 순서는 worklist §1g. |

> `dev/session-data/`(원본 세션 transcript)는 검증용 로컬 자료로 **git 미추적**(`.gitignore`).

## 하위 디렉토리

| 경로 | 내용 |
|---|---|
| [`chain-analysis/`](chain-analysis/) | **체인 바이너리의 CLI 표면과 배선** — 실행 옵션 질문은 체인 소스를 읽기 전에 여기부터. 바이너리에서 재생성되며 기준 체인 커밋을 스스로 적는다. 대상이 외부 바이너리라 설계 문서 4종과는 별개 축이다(§문서 등급). |
| [`claudedocs/`](claudedocs/) | 외부 컨텍스트 — chainbench가 속한 상위 자동화 시스템의 제안서/지시서(cross-reference). **[이력]** 제안 당시의 문서라 `chainbench init`/`start` 같은 존재하지 않는 명령이 그대로 있다. 명령 표면은 `chainbench --help` 가 이긴다. |
