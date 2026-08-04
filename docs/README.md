# docs/ — 문서 인덱스

chainbench는 **go-stablenet/wbft/wemix용 Go-first 다체인 테스트벤치**다. 이 디렉토리는
현행 설계·운영 문서를 둔다. (Go 재설계 이전 bash/TS 아카이브와 구 재설계 SSoT는 정리되었다.)

## 설계 SSoT — 여기서 시작 (`docs/dev/`, 4종 세트)

| 문서 | 성격 |
|---|---|
| [`dev/chainbench-requirements-review.md`](dev/chainbench-requirements-review.md) | 요구사항 37 · 사양 검토 · 코드 격차 · **etcd flaky 실체** · 동시성/안전성. |
| [`dev/chainbench-design.md`](dev/chainbench-design.md) | **아키텍처 SSoT** — 구조·패키지 인터페이스(§3)·데이터 모델(§4)·동시성(§6)·마이그레이션. |
| [`dev/chainbench-feature-spec.md`](dev/chainbench-feature-spec.md) | F1~F15 동작 계약 · 수용기준(AC). |
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
| [`dev/wemix4-port-tracker.md`](dev/wemix4-port-tracker.md) · [`dev/wemix4-migration-plan.md`](dev/wemix4-migration-plan.md) · [`dev/repro-migration-remaining.md`](dev/repro-migration-remaining.md) | wemix4 테스트 포팅·마이그레이션 추적. |

> `dev/session-data/`(원본 세션 transcript)는 검증용 로컬 자료로 **git 미추적**(`.gitignore`).

## 하위 디렉토리

| 경로 | 내용 |
|---|---|
| [`claudedocs/`](claudedocs/) | 외부 컨텍스트 — chainbench가 속한 상위 자동화 시스템의 제안서/지시서(cross-reference). |
