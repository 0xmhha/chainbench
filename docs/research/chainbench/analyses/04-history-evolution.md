# git 히스토리에서 본 구조 진화

## 1. 월별 커밋 볼륨

| 월 | 커밋 수 |
|---|---:|
| 2026-03 | 21 |
| 2026-04 | 257 |
| 2026-05 | 13 |
| 2026-06 | 17 |
| 2026-07 | 151 |
| 2026-08 | 156 |

4월에는 초기 기능과 테스트 표면이 빠르게 확장됐고, 7~8월에는 Go-first 재설계와 모듈 경계 정리가 집중됐다.

## 2. 진화 단계

| 단계 | 기간 | 핵심 변화 | 구조적 의미 |
|---|---|---|---|
| Phase 0 | 3월 | 셸·MCP 중심 sandbox 시작 | 빠른 기능 검증 우선 |
| Phase 1 | 4월 | 회귀 테스트·CLI·MCP 기능 급증 | 표면과 도메인 로직이 함께 팽창 |
| Phase 2 | 7월 | Go-first multi-chain redesign, dashboard | 장기 구조로 전환 |
| Phase 3 | 8월 초중순 | `pkg`→`internal`, session/testspec/engine 도입 | 내부 계약과 실행 모델 형성 |
| Phase 4 | 8월 말 | resource, machine, keyring, launcher, chainsetup 분리 | 단일 소유자와 이름 정리 진행 |

## 3. 역할 분화 과정

8월 25~28일 커밋은 machine, keyring, process, chainsetup, testengine, launcher, testspec 순으로 책임을 분리했다. 현재 코드는 이 작업의 결과로 레이어 규칙과 상태 소유권을 테스트한다 (`internal/arch/layers_test.go:89`, `internal/arch/state_test.go:57`). 동시에 이전 경로를 안전하게 이행하기 위한 forwarding wrapper와 두 개의 local composition 경로가 남았다.

## 4. 운영 패턴

- 커밋 제목이 `feat(scope)`, `refactor(scope)`, `fix(scope)` 형태로 역할 단위를 드러낸다.
- 구조 변경마다 문서와 테스트 ratchet을 함께 추가하는 경향이 있다.
- 대규모 이동을 한 번에 끝내기보다 기존 표면을 유지한 채 shrink-only 예외를 줄이는 전략을 쓴다 (`internal/arch/mcp_imports_test.go:17`).
- 다음 리팩토링도 이 패턴을 유지해, canonical path를 먼저 만들고 CLI 계약 테스트로 옛 경로 제거를 증명하는 편이 안전하다.
