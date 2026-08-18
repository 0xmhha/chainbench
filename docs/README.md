# docs/ — 문서 인덱스

chainbench는 **go-stablenet/wbft/wemix용 Go-first 다체인 테스트벤치**다. 이 디렉토리는
현행 설계·운영 문서를 둔다. (Go 재설계 이전 bash/TS 아카이브와 구 재설계 SSoT는 정리되었다.)

## 설계 SSoT — 여기서 시작 (`docs/dev/`, 4종 세트)

| 문서 | 성격 |
|---|---|
| [`dev/chainbench-requirements-review.md`](dev/chainbench-requirements-review.md) | 요구사항 37 · 사양 검토 · 코드 격차 · **etcd flaky 실체** · 동시성/안전성. |
| [`dev/chainbench-design.md`](dev/chainbench-design.md) | **아키텍처 SSoT** — 구조·패키지 인터페이스(§3)·데이터 모델(§4)·동시성(§6)·마이그레이션. |
| [`dev/chainbench-feature-spec.md`](dev/chainbench-feature-spec.md) | F1~F16 동작 계약 · 수용기준(AC). |
| [`dev/chainbench-refactoring.md`](dev/chainbench-refactoring.md) | 기존 코드 → 목표 설계 매핑 · **작업 단위(WP) 분해**. |

## 현행 참고 문서

| 문서 | 내용 |
|---|---|
| [`REMOTE_WEMIX_DEPLOY_DESIGN.md`](REMOTE_WEMIX_DEPLOY_DESIGN.md) | remote wemix+etcd 배포 설계(현행 remote 경로). |
| [`MCP_USAGE.md`](MCP_USAGE.md) | chainbench MCP 서버 사용 가이드(도구 호출, `.mcp.json` 등록). |
| [`SECURITY_KEY_HANDLING.md`](SECURITY_KEY_HANDLING.md) | 키 취급 보안 정책·위협 모델. 프리셋 키는 **테스트 픽스처 전용**. |

## `dev/` 개발 문서

| 문서 | 내용 |
|---|---|
| `dev/chainbench-*.md` | 위 설계 SSoT 4종. |
| [`dev/keys-generate.md`](dev/keys-generate.md) · [`dev/topology.md`](dev/topology.md) | 프리셋 키 생성 · 로컬 토폴로지 설정 가이드. |
| [`dev/family-bringup-design.md`](dev/family-bringup-design.md) | **패밀리별 기동 설계 제안** — 상위 일관/하위 특화. 3체인 차이 실측표 · 4 seam(BringUpPhases·Action·GenesisArtifacts·PortReservation) · 작업순서 F1~F6 · 열린 질문. |
| [`dev/server-inventory.md`](dev/server-inventory.md) | **서버 인벤토리** — 노드 포트·호스트·접속 정보의 단일 출처(`remote-server-config.yaml`, gitignore). local/remote 동일 구조·포트 규칙·자격증명 취급. |
| [`dev/wemix4-port-tracker.md`](dev/wemix4-port-tracker.md) · [`dev/wemix4-migration-plan.md`](dev/wemix4-migration-plan.md) · [`dev/repro-migration-remaining.md`](dev/repro-migration-remaining.md) | wemix4 테스트 포팅·마이그레이션 추적. |

### `dev/architecture/` — 아키텍처 · 다이어그램 (현재 코드 기준)

| 문서 | 내용 |
|---|---|
| [`dev/architecture/software-architecture.md`](dev/architecture/software-architecture.md) | 전체 소프트웨어 아키텍처 — 계층·컨텍스트·실행모델·환경 5요소·검증원·동시성. |
| [`dev/architecture/component-diagram.md`](dev/architecture/component-diagram.md) | 컴포넌트 맵 · C4 컨테이너 뷰 · 키 소싱 컴포넌트 · 목표 델타. |
| [`dev/architecture/sequence-diagrams.md`](dev/architecture/sequence-diagrams.md) | 전체 run · BuildEnv · Interpreter · 원자 스텝 CLI · 원격 SSH · 실패 경로. |
| [`dev/architecture/state-diagrams.md`](dev/architecture/state-diagrams.md) | TestRun · Environment · NodeProcess · Session/키 · 워크스페이스 스텝. |
| [`dev/architecture/layers.md`](dev/architecture/layers.md) | **레이어 아키텍처** — L0~L6 정의 · 57개 패키지 전수 배치 · 의존 규칙(상향 0건 실측) · **상태 소유 규칙**(control plane=session / data plane=FileSink)과 위반 6곳 · 규칙 강제 테스트 제안. |
| [`dev/architecture/code-graph.md`](dev/architecture/code-graph.md) | AST 실측 패키지 그래프 — 계층 검증 · fan-in/out · launch-args 분산 5지점 · launchopt 실행 순서. |

### 2026-08-11 재설계 검토 3종 (제안 — 미확정)

| 문서 | 내용 |
|---|---|
| [`dev/dsl-v2-proposal.md`](dev/dsl-v2-proposal.md) | DSL v2 문법 제안 + x-bar 정렬 갭 분석(G1~G7). |
| [`dev/structure-and-atomic-cli-proposal.md`](dev/structure-and-atomic-cli-proposal.md) | import 그래프 실측 · 오케스트레이션 3스택 문제 · `internal/app` 제안 · 원자 CLI 스텝 카탈로그. |
| [`dev/chain-binary-flag-graph.md`](dev/chain-binary-flag-graph.md) | 3체인 바이너리 CLI 그래프(AST 추출) · 실행옵션 모듈+builder 설계 비판 검토. |

> `dev/session-data/`(원본 세션 transcript)는 검증용 로컬 자료로 **git 미추적**(`.gitignore`).

## 하위 디렉토리

| 경로 | 내용 |
|---|---|
| [`claudedocs/`](claudedocs/) | 외부 컨텍스트 — chainbench가 속한 상위 자동화 시스템의 제안서/지시서(cross-reference). |
