# 기능 목록과 구현 모듈

chainbench는 여러 EVM 계열 체인을 로컬·원격 환경에 구성하고, 선언형 테스트를 실행해 세션 산출물로 남기는 테스트 벤치다.

## 네트워크 구성

| 기능 | 설명 | 구현 모듈 |
|---|---|---|
| workspace 생성·재개 | 단계 상태를 저장하고 미완료 지점부터 재개 | `internal/chainsetup` |
| 서버·슬롯·포트 배정 | server set을 열고 노드 placement 계산 | `internal/resource` |
| genesis·설정·기동 | 체인 패밀리 전략과 launcher 조립 | `internal/chainsetup`, `internal/consensus`, `internal/core/launcher` |
| 프로세스 추적 | PID와 datadir을 ledger로 보존 | `internal/core/process` |

## 테스트 실행

| 기능 | 설명 | 구현 모듈 |
|---|---|---|
| DSL 읽기·파싱 | v1/v2를 공통 `Spec`으로 낮춤 | `internal/testspec` |
| 액션·검증 실행 | registry로 액션·어설션·리더 연결 | `internal/testspec`, `internal/testhelper` |
| 환경 구성·실행 | local/attach 엔진과 직렬 spec 실행 | `internal/testengine` |
| 결과·아티팩트 | 환경·테스트·요약 저장 | `internal/core/session` |

## 사용자 표면

| 기능 | 설명 | 구현 모듈 |
|---|---|---|
| CLI | 명령·플래그·표/JSON 출력 | `cmd/chainbench`, 하위 `*cmd` 패키지 |
| MCP | 도구 스키마와 use case 호출 | `internal/mcp`, `cmd/chainbench-mcp` |
| 대시보드 | 관측 이벤트와 세션 결과 표시 | `internal/dashboard`, `web` |

## 기능 지도가 보여주는 것

- 네트워크 구성 기능이 `chainsetup`과 `testengine`에 중복되어 기능→모듈 대응이 1:1이 아니다.
- DSL “언어”와 DSL “실행”이 모두 `testspec`에 있어 순수 파서만 독립적으로 사용하기 어렵다.
- CLI 하위의 `netcmd`, `keyringcmd`, `resourcecmd` 분리는 좋은 방향이지만 root에는 아직 많은 명령 생성자와 직접 조립 코드가 남아 있다 (`cmd/chainbench/root.go:28`, `cmd/chainbench/run.go:192`).
