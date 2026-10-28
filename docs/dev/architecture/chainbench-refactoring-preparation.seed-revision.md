## Iteration 2 — 승인된 정제

- 원본 QA: 0.80 REVISE, 목표 0.90. 사용자 답변: “승인.”
- [QA + Simplifier] 인터뷰 기록 AC 삭제, 고유 사실은 맥락/ontology로 이동 — 승인 반영.
- [QA + Hacker + Researcher] 관계 종류, 해소 상태, snapshot 근거, 범위 대조와 자동 검증 필수화 — 승인 반영.
- [QA + Architect] 기존 ontology 필드 안에서 그래프·진단·계약·검증 ID 연결 — 승인 범위 구체화.
- [QA + Contrarian] 계약별 절차/관찰값/판정 조건과 SKIP·후속 실행 경계 명시 — 승인 반영.
- 새로운 범위 확대 0건. 실제 런타임 추적, 제품 리팩토링, 현재 동작 기준선 수집은 채택하지 않음.
- Hacker는 하위 에이전트 검토. 추가 에이전트는 런타임 thread limit으로 생성 불가하여 나머지 네 관점은 주 세션에서 순차 검토. MCP에는 미실행 하위 에이전트로 정확히 기록함.

### 원본 대비 변경

- 수용 기준 4개 → 3개. 기존 세 결과 기준의 ID 유지.
- goal, constraints, ontology 이름과 11개 필드, 평가 가중치 유지.
- 수용 기준의 자동 검사 면제를 의미/설계 판단으로 한정.
- 기존 schema 설명에 필수 데이터와 검증 연결 추가.
- metadata.version: 1.0.0 → 1.0.1. Seed ID 유지.
- 로컬 YAML 및 Ouroboros Seed 모델 검증 통과. 제약/ontology/ID 보존 검사 통과.

감사 기록은 현재 쓰기 가능한 프로젝트 안에 보존한다. 실제 그래프 생성 및 검증 절차 실행 결과가 아니라 Seed 명세 검증 기록이다.

