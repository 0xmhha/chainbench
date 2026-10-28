# Seed v1.0.5 실행 결과

Seed v1.0.5 확정본으로 실행했으며 Ouroboros는 수용 기준 3/3 completed를 보고했다. 정식 3단계 평가는 자동 실행되지 않았으므로 **executed_unverified**로 취급한다. 각 산출물의 기계적 검증 보고서는 PASS지만, 이 상태를 정식 평가 승인이나 전체 체인 호환성 통과로 확대하지 않는다.

- Job: job_232f57ec45cf
- Session: orch_6cf3d0cff337
- Execution: exec_7849d1f703c0
- 실행 방식: quality_first / frugality assurance off
- 작업 트리: `/Users/0xtopaz/.ouroboros/worktrees/chainbench/orch_6cf3d0cff337`
- 브랜치: `ooo/orch_6cf3d0cff337` (실행기 생성)

산출물은 별도 작업 트리에 있으며 main으로 병합하지 않았다. 확인한 git status에는 준비 문서·분석/검증 스크립트·실행 기록의 신규 파일만 있고 추적 중인 제품 파일 변경은 없다.

## 산출물

1. [문서·코드 그래프와 근거 진단](/Users/0xtopaz/.ouroboros/worktrees/chainbench/orch_6cf3d0cff337/docs/dev/architecture/preparation-ac1-v2/artifacts-verified/README.md)
2. [그래프 탐색기](/Users/0xtopaz/.ouroboros/worktrees/chainbench/orch_6cf3d0cff337/docs/dev/architecture/preparation-ac1-v2/artifacts-verified/explorer.html) — 탐색기에서 같은 디렉터리의 JSON.gz 파일을 선택한다.
3. [AC1 재시도 검증 증거](/Users/0xtopaz/.ouroboros/worktrees/chainbench/orch_6cf3d0cff337/docs/dev/architecture/preparation-ac1-retry1/README.md)
4. [목표 구조와 단계별 계획](/Users/0xtopaz/.ouroboros/worktrees/chainbench/orch_6cf3d0cff337/docs/dev/architecture/preparation-ac2-v1/README.md)
5. [외부 계약별 검증 절차](/Users/0xtopaz/.ouroboros/worktrees/chainbench/orch_6cf3d0cff337/docs/dev/architecture/preparation-ac3-retry1/README.md)

## 핵심 결과

- Go 파일 723개, 패키지/테스트 변형 159개, 호출 지점 28,415개. 타입 변환 1,048개를 호출과 분리.
- 문서 108개, 절 1,685개. 의미 간선 confirmed 1,708개와 candidate 4,390개를 구별. 기계적 범위 검증 통과가 모든 의미 해석 확정을 뜻하지 않는다.
- 구현 관계 후보 208,280개를 value/pointer method set으로 처리. source 및 직접 참조 외부 interface가 범위이며 런타임 dispatch 확정과 구별한다.
- 진단 6개, 목표 책임 소유자 10개, Stablenet 경계 6개, 변경 단계 6개.
- 외부 동작군 계약 16개와 검증 항목 16개. 매핑 누락·필수 공백·미해소 근거·해시 불일치 0건. 모든 개별 CLI flag/MCP 도구의 golden fixture를 확보했다는 의미는 아니다.

## 목표 방향과 순서

체인 구성과 구성된 체인의 테스트를 책임 축으로 나누고, 표면→app 조립→구성/테스트 오케스트레이션→관측/서비스를 의존 축으로 구별한다. testengine 소유 환경 포트를 app/composition 어댑터가 구현하며, readiness·합의 진행·receipt·canonical 상태·노드 전파·실패 진단/정리의 소유권을 분리한다. 현재 유효한 DSL Registry 주입은 유지한다.

- P0: 근거·계약·현재 경로 고정
- P1: 환경 요청과 구성 실행 소유권 분리
- P2: 표면 타입과 내부 구성 사이 어댑터를 필요한 경계로 제한
- P3: 관측·수명주기 경계를 동작 보존 상태로 분리
- P4: 판정 강화를 별도 후속 검토
- P5: 미사용 연결 제거와 문서 통합

각 단계에는 선행 조건, 변경 대상, 보존 계약, 검증 항목, 되돌림 조건이 있다. 구성 타입 alias 일괄 제거와 광범위한 패키지 이동을 전제하지 않는다.

원본 Stablenet FAIL과 후속 restored/instrumented/unmodified 관측을 구분하며 직접 원인은 미확정으로 유지한다. 추가 E2E나 제품 리팩토링은 이 실행에서 수행하지 않았다.

다음 정식 평가: `ooo evaluate orch_6cf3d0cff337`.
