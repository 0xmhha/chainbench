# Chainbench 리팩토링 준비 Seed 검토 — v1.0.1

> 후속 기준선 반영본 v1.0.2의 QA는 0.92 PASS다. 최신 평가와 Seed 전문은 [기준선 반영 검토](chainbench-refactoring-preparation.seed-baseline-review.md)를 참조한다. 아래는 v1.0.1 당시 기록이다.

승인된 정제 1회 완료. QA는 0.80 → **0.88 / 0.90, REVISE**다. 자문 평가이며 자동 재수정은 수행하지 않는다.

기존 세 보완점은 반영했다. 남은 QA 제안은 timeout/정리 절차, 사용자 변경 보호, 분석 자료의 지시문 경계, evidence_id 유일성 범위 명시다. 추가 제안은 이번 수정본에 자동 반영하지 않았다.

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

## 현재 Seed 전체

```yaml
goal: 이해한 chainbench의 목적·비전을 바탕으로 문서 X-bar 의미 그래프와 코드 AST 그래프를 생성하고, 응집도와 책임·레이어 분리 문제에 대한 근거 기반 진단, 목표
  구조, 단계별 리팩토링 계획, 호환성 검증 기준·절차를 준비한다.
task_type: code
brownfield_context:
  project_type: brownfield
  context_references:
  - path: /Users/0xtopaz/work/github/0xmhha/chainbench
    role: primary
    summary: 문서·코드 분석과 구조적 진단, 목표 구조 및 리팩토링 계획을 준비할 개선 대상.
  - path: /Users/0xtopaz/work/github/0xmhha/chain/go-wemix
    role: reference
    summary: chainbench의 지원 유지와 연동 계약·호환성 판단을 위한 참조.
  - path: /Users/0xtopaz/work/github/0xmhha/chain/go-wbft
    role: reference
    summary: chainbench의 지원 유지와 연동 계약·호환성 판단을 위한 참조.
  - path: /Users/0xtopaz/work/github/0xmhha/chain/go-stablenet
    role: reference
    summary: chainbench의 지원 유지와 연동 계약·호환성 판단을 위한 참조.
  existing_patterns:
  - '사용자가 적용을 확정한 기존 X-bar 사상: X⁰=주제, complement=필수 논항, adjunct=선택 수식, specifier=대상·맥락.'
  - 기존 CLI·DSL·MCP·설정·결과·세션 등의 외부 계약과 세 체인 지원을 보존하는 내부 개선을 기본으로 한다.
  - 인터뷰 승인 범위와 변경 이력은 refactoring-preparation-scope.md 및 chainbench-refactoring-preparation.seed-review.md에
    보존한다. 승인 대화 자체는 수용 기준이 아니다.
  - 세 참조 체인은 동일한 github.com/ethereum/go-ethereum 모듈 경로를 사용하므로 모듈 경로만으로 저장소나 심볼을 식별하지 않는다.
  - 기존 scripts/inventory/code-graph와 docs/dev/codegraph는 재사용 후보이며 전체 호출·구현 분석 충족의 증거로 간주하지 않는다. tests/e2e
    문서의 빌드 태그·바이너리 조건은 후속 절차에 반영한다.
  existing_dependencies: []
constraints:
- 이번 작업은 리팩토링 준비이며 실제 제품 리팩토링을 수행하지 않는다.
- 개선 대상은 /Users/0xtopaz/work/github/0xmhha/chainbench로 한정한다.
- /Users/0xtopaz/work/github/0xmhha/chain/go-wemix, /Users/0xtopaz/work/github/0xmhha/chain/go-wbft, /Users/0xtopaz/work/github/0xmhha/chain/go-stablenet은
  연동 계약·호환성 분석을 위한 참조이며 해당 저장소 내부의 구조 개선 계획은 범위에서 제외한다.
- 체인 구성과 구성된 체인 기반 테스트라는 책임 축을 의존 레이어 축과 구별하며, 기존 assistant의 L0/L1/L3 재분류를 확정 설계로 취급하지 않는다.
- 문서 X-bar 분석은 문서·절의 의미 구조에 X⁰=주제, complement=필수 논항, adjunct=선택 수식, specifier=대상·맥락 사상을 적용하고 근거 구절을 연결한다.
  개별 문장 내부의 언어학적 구문 트리는 제외한다.
- 코드 그래프는 AST를 기반으로 chainbench 전체 패키지·심볼, 정적으로 확인 가능한 함수·메서드 호출, 인터페이스 구현 관계를 포함한다.
- 기본 계획은 CLI·DSL·MCP·설정·결과·세션 등의 외부 계약을 보존하는 내부 리팩토링이며 세 체인 지원을 유지한다.
- 꼭 필요한 외부 계약 변경은 이유·영향·전환 방법·검증 기준을 갖춘 별도 대안으로 제시한다. 대안 작성은 특정 비호환 변경의 구현 승인이 아니다.
- 검증 항목과 실행 절차는 기존 테스트·기록을 근거로 준비한다. 실제 세 체인 E2E 실행과 현재 동작 기준선 확보는 후속 작업이며 과거 기록을 현재 실행 증거로 취급하지 않는다.
- 산출물 파일명과 표시 형식 등 구현 세부는 실행자가 정한다.
acceptance_criteria:
- description: chainbench의 목적·비전과 구조적 문제를 검토할 수 있는 근거 기반 분석이 제공된다. 분석에는 근거 구절이 연결된 문서·절 X-bar 의미 그래프,
    chainbench 전체 패키지·심볼 및 정적으로 확인 가능한 함수·메서드 호출·인터페이스 구현 관계의 AST 기반 그래프가 포함되고, 문서와 코드 근거를 연결하여 기능별 응집도
    저하와 책임·레이어 분리의 모호함을 진단한다. 체인 구성/테스트 책임 축과 의존 레이어 축은 구별된다. 그래프는 사람이 탐색 가능한 표현과 기계 판독 가능한 노드·관계·근거 목록을
    함께 제공한다. 관계는 import/reference/call/implements와 문서 의미 관계를 구별하고, confirmed/candidate/unknown 상태, 근거
    위치, 미해결 사유를 보존한다. 단순 참조를 호출로, 인터페이스 만족을 실제 런타임 호출로 표시하지 않는다. 저장소별 snapshot에는 경로·commit·분석 대상 작업 트리
    내용 해시, 도구 버전·명령·빌드 조건을 기록한다. 전체 소스 inventory에는 제품·테스트·생성 코드·빌드 태그별 포함/제외/미해결과 사유를 기록하며, 파일·패키지·심볼·호출
    지점 및 구현 분석 대상별 분모와 처리 수를 대조한다. 정적으로 해소 가능한 범위를 누락하거나 분석 실패를 제외 처리하여 전체 분석 완료로 판정하지 않는다. 동적 호출·외부 의존성
    등 해소 불가능한 대상은 unknown으로 명시한다. 산출물에 실제 실행 가능한 검증 진입점과 실행 보고서가 있으며, ID 유일성, 모든 관계 끝점과 근거의 존재, snapshot
    내 근거 위치/내용 일치, inventory 누락·중복 0건, 상태별 합계 일치, 같은 입력·도구·옵션·보존된 의미 주석에 대한 정규화 결과 재현성이 모두 통과한다. 문서 목록과
    절별 처리/제외 사유도 대조되며, 문서의 정본·설계·측정·이력 구분과 상충 근거를 해석에 반영한다.
  semantic_ac_key: ac_cdfeb8f1531aa71c
  verify_exemption_reason: 면제는 X-bar 의미 분류, 목적·비전 해석, 구조적 진단의 타당성에 대한 사람의 근거 검토에만 적용한다. 이 AC에 명시한 그래프
    무결성·범위 대조·재현성 자동 검증과 실제 성공 보고서는 필수이며 면제되지 않는다. 진입점의 파일명과 실행 명령은 준비 산출물에서 확정한다.
- description: 근거 기반 진단에 대응하는 chainbench 목표 구조와 단계별 리팩토링 계획이 제공된다. 기본 계획은 외부 계약과 세 체인 지원을 보존하고 체인 구성/테스트
    책임과 의존 레이어를 구별하여 유지보수성·가독성 개선 방향을 설명한다. 꼭 필요한 외부 계약 변경을 제안하는 경우 이유·영향·전환 방법·검증 기준이 포함된 별도 대안으로 제시하며,
    세 체인 자체의 내부 개선 계획이나 제품 리팩토링 구현은 포함하지 않는다. 각 진단 ID는 목표 책임 배치·허용 의존 방향·단계별 변경 대상 및 검증 항목으로 추적 가능하고,
    단계 간 선행 조건과 되돌림 조건이 설명된다.
  semantic_ac_key: ac_1c195afafb391715
  verify_exemption_reason: 목표 구조와 단계별 계획이 진단에 적절히 대응하는지, 외부 변경이 필요한지, 제시된 전환 대안이 타당한지는 설계와 계약 근거에 대한 판단이
    필요하여 명령만으로 결정할 수 없다.
- description: 기존 테스트·기록에 근거한 검증 기준과 실행 절차가 제공되어, 후속 작업에서 CLI·DSL·MCP·설정·결과·세션 등 외부 계약과 go-wemix·go-wbft·go-stablenet
    연동 보존 여부를 확인할 수 있다. 실제 세 체인 E2E 실행과 현재 동작 기준선 확보는 후속 작업으로 명시되고, 과거 기록과 현재 실행 증거가 구별된다. 외부 계약 inventory의
    각 항목은 고유 ID, 표면 종류(CLI/DSL/MCP/설정/결과/세션/체인 연동), 현재 문서·코드 근거와 적용 체인을 갖는다. 각 계약에 시나리오, 전제조건·도구/바이너리/환경변수·작업
    디렉터리·입력, 실제 명령 또는 구체적인 수동 단계, 관찰값·기대 결과·명시적 성공/실패 조건, 증거 보관 방법을 연결한다. 기존 검증은 파일·테스트명·기록 revision으로
    연결하고 부족한 검증은 추가 필요로 표시하되 후속 작성/실행 절차와 판정 기준을 제공한다. 적용되지 않는 조합은 근거를 기록한다. 계약별 매핑 누락과 필수 필드 공백이 0건인
    자동 대조 보고서가 제공된다. e2e 빌드 태그와 바이너리 준비를 기본 테스트와 구별하고 SKIP/미실행/환경 부족을 성공으로 세지 않는다. 실제 호환성 성공 판정은 후속 실행까지
    보류하며, 이번 완료 판정은 절차의 완전성과 근거 적합성에 한정한다. 소스/계약/도구/빌드 조건 변경 시 영향받는 그래프·진단·검증 항목의 재확인 조건을 기록한다.
  semantic_ac_key: ac_116b24ae05a0fcc5
  verify_exemption_reason: 실제 체인 실행과 동작 기준선 수집은 후속 범위이므로 이번에는 실행 성공을 요구하지 않는다. 면제는 절차의 의미적 적합성에 대한 사람의
    검토에 한정하며, 계약 inventory 대비 매핑 누락·필수 필드 공백 검사의 성공 보고서는 필수다.
ontology_schema:
  name: ChainbenchRefactoringPreparation
  description: chainbench의 목적·비전, 문서 의미 구조, 코드 구조, 연동 계약과 진단 근거를 목표 구조·단계별 계획·검증 절차에 연결하는 리팩토링 준비 모델.
  fields:
  - name: purpose_and_vision
    type: object
    description: 문서 근거에 기반한 chainbench의 목적과 비전 해석.
    required: true
  - name: repositories
    type: array
    description: 고정된 네 저장소 경로와 chainbench 개선 대상 또는 체인 연동 참조 역할. 각 항목은 repository_id, root, role, commit,
      분석 입력의 파일별 내용 해시 및 dirty 상태, snapshot_id를 포함한다. snapshot_id는 모듈 경로와 독립적이며 동일 commit의 작업 트리 차이도 식별한다.
      분석 입력 snapshot 확보와 후속 동작 기준선 수집을 구별한다.
    required: true
  - name: document_graph
    type: object
    description: 문서·절의 X-bar 의미 요소와 관계 및 해당 분류를 뒷받침하는 근거 구절. 필수 데이터는 문서/절 inventory, 문서 지위, nodes(id,
      kind, X-bar 역할), edges(source, target, relation, status), evidence(repository_id, snapshot_id, path,
      위치, excerpt, 내용 해시), 분류 근거·상충/미해결 사유, 범위 대조 및 검증 보고서 참조다. 해석 주석과 그 입력을 보존한다.
    required: true
  - name: code_graph
    type: object
    description: chainbench 전체 패키지·심볼, 정적으로 확인 가능한 함수·메서드 호출과 인터페이스 구현 관계. 필수 데이터는 소스 inventory, repository/snapshot으로
      한정된 node ID·심볼 소속, typed edges(import/reference/call/implements), resolution 상태와 원인, 근거 위치, 도구/버전/명령/빌드
      조건, 범위별 분모·처리 수, 자동 검증 보고서다. pointer/value method set과 빌드 조건을 구현 관계 근거에 포함하고 구현 후보와 실제 호출 대상을 구별한다.
      미해결 대상은 유효한 unknown 노드로 나타내며 간선 끝점을 생략하지 않는다.
    required: true
  - name: responsibility_axes
    type: object
    description: 체인 구성/테스트 책임 축과 별도로 구분한 의존 레이어 축.
    required: true
  - name: integration_contracts
    type: array
    description: 세 체인 지원과 연동·호환성 판단에 관련된 계약. 세 체인 연동뿐 아니라 CLI·DSL·MCP·설정·결과·세션 외부 표면을 포함한다. 필수 데이터는 contract_id,
      surface, 적용 체인/조건, 보존할 동작·형식, 문서/코드 evidence ID, verification ID다.
    required: true
  - name: structural_findings
    type: array
    description: 응집도와 책임·레이어 분리 문제 및 이를 뒷받침하는 문서·코드 근거 연결. 각 finding_id에 사실/해석 구분, 문서·코드 evidence ID,
      영향받는 책임/의존 관계 및 계약 ID를 연결한다. 근거 부족은 확정 결함과 구별한다.
    required: true
  - name: target_structure
    type: object
    description: 진단에 대응하는 책임 배치와 의존 관계의 목표 구조. 목표 tree와 각 책임의 소유 위치, 허용 의존 방향을 함께 표현하며 finding_id에 연결한다.
    required: true
  - name: refactoring_plan
    type: array
    description: 외부 계약 보존을 기본으로 하는 단계별 내부 리팩토링 계획. 각 단계는 대상, 선행 조건, finding_id, 목표 책임, 보존 계약 ID, 검증 ID와
      되돌림 조건을 포함한다.
    required: true
  - name: external_change_alternatives
    type: array
    description: 필요한 경우에만 제시하는 외부 계약 변경 대안과 이유·영향·전환 방법·검증 기준.
    required: true
  - name: verification_plan
    type: object
    description: 기존 테스트·기록에 근거한 검증 항목·실행 절차와 후속 E2E·현재 기준선 확보 작업의 구분. 필수 데이터는 계약 inventory와의 대조표, verification_id,
      contract_id, scenario, prerequisites, cwd, inputs, 명령/구체적 수동 단계, expected observations, pass/fail/skip
      조건, evidence 수집 위치, 기존 테스트/기록 참조 또는 추가 필요 표시, 실행 상태다. preparation_checks에는 그래프·매핑 검증의 실제 명령과 성공
      보고서를 보관하고 deferred_checks에는 실제 체인 실행·동작 기준선 절차를 보관한다. 기존 기록은 historical로 표시한다. snapshot·도구·계약 변경에
      따른 재확인 조건을 포함한다.
    required: true
evaluation_principles:
- name: 근거 추적성
  description: 문서 의미 분류와 구조적 진단을 원문 구절 및 코드 관계로 추적할 수 있고, 사실과 해석을 구별한다.
  weight: 0.3
- name: 분석 범위 충족
  description: 문서·절 의미 분석과 chainbench 전체 패키지·심볼·정적 호출·인터페이스 구현 관계 분석이 승인된 범위를 충족한다.
  weight: 0.25
- name: 설계 타당성
  description: 목표 구조와 단계별 계획이 진단에 대응하며 책임 축과 의존 레이어 축을 구별하여 유지보수성과 가독성 개선을 설명한다.
  weight: 0.25
- name: 호환성과 검증 적합성
  description: 외부 계약과 세 체인 지원 보존을 기본으로 삼고, 검증 절차의 근거와 후속 실행 경계를 명확히 한다.
  weight: 0.2
exit_conditions:
- name: 준비 산출물 완료
  description: 승인된 리팩토링 준비 범위가 충족되면 종료한다.
  criteria: 근거 연결 문서·코드 그래프와 진단, 목표 구조, 단계별 계획, 검증 기준·절차가 모두 제공되고 수용 기준을 충족한다.
- name: 후속 실행 경계 도달
  description: 준비 이후의 구현과 실제 기준선 확보는 별도 작업으로 남긴다.
  criteria: 준비 산출물 완료 후 제품 리팩토링, 세 체인 실제 E2E 실행 또는 현재 동작 기준선 확보로 자동 진행하지 않는다.
metadata:
  seed_id: seed_f339f3d38f1e
  version: 1.0.1
  created_at: '2026-09-13T10:01:27.886805Z'
  ambiguity_score: 0.0375
  gate_forced: false
  interview_id: interview_20260913_090229
  parent_seed_id: null
  generation_mode: normal
  degraded: false
  unresolved_slots: []
  recovery_reason: null

```

## 수정본 QA

```text
QA Verdict [Iteration 2]
============================================================
Session: qa-1a8dc6ec
Score: 0.88 / 1.00 [REVISE]
Verdict: revise
Threshold: 0.90

Dimensions:
  Correctness          0.93
  Completeness         0.83
  Quality              0.87
  Intent Alignment     0.96
  Domain Specific      0.81

Differences:
  - 실행 가능한 준비 검증과 후속 명령을 요구하지만, 명령의 시간 제한과 중단·정리·TIMEOUT 판정은 정의하지 않는다(hung_command).
  - 작업 트리의 dirty 상태와 해시는 기록하지만, 산출물 생성 시 기존 파일·미커밋 변경의 덮어쓰기를 방지하는 조건은 없다(dirty_worktree).
  - 문서·코드 내용을 의미 분석에 사용하지만, 자료에 포함된 지시문이 작업 범위나 실행 권한을 변경하지 못하도록 하는 경계는 없다(prompt_injection).
  - 계약과 진단이 evidence ID를 참조하지만, evidence 레코드의 식별자 필드와 유일성 범위가 명시되지 않아 참조 해소 규칙이 불완전하다.

Suggestions:
  - 준비 검증과 후속 실행 절차에 유한한 timeout, 종료·정리 방법, TIMEOUT 상태와 증거 기록을 요구하고 시간 초과를 성공에서 제외한다.
  - 산출물 쓰기는 지정된 생성 영역으로 제한하고, 기존 사용자 파일이나 미커밋 변경과 충돌하면 덮어쓰지 않고 충돌을 보고하도록 명시한다.
  - 분석 자료 속 지시문은 데이터로 취급하며 작업 범위·실행 권한·수용 기준을 변경할 수 없다는 제약을 추가한다.
  - evidence_id 필드와 유일성 범위를 정의하고, 그래프·진단·계약의 모든 evidence ID가 해당 snapshot의 근거 레코드로 해소되는 자동 검사를 명시한다.

Reasoning: 핵심 범위와 완료 상태는 구체적이며, 정적 분석과 런타임 검증의 경계 및 의미 검토 면제와 필수 기계 검증의 구분도 명확하다. 다만 실행 절차의 예외 처리, 작업 트리 보호, 자료의 지시문 처리 경계와 근거 식별자 정의가 부족해 0.90 기준에는 미달한다. 문서에 대한 단일 자문 평가이며, 실행 증거가 없다는 이유로 감점하지 않았다.

Loop Action: continue
```

<details>
<summary>최초 생성본과 최초 QA 기록</summary>

# Chainbench 리팩토링 준비 Seed — 생성 및 품질 검토

QA advisory: REVISE — score 0.80 (bar 0.90).

Seed ID: `seed_f339f3d38f1e`  
Interview: `interview_20260913_090229`

생성은 완료됐다. 아래 Seed는 생성 원문이며 QA 수정 제안은 아직 적용하지 않았다. 분석 작업·제품 리팩토링·실제 E2E 기준선 확보는 실행하지 않았다.

주요 검토 사항:
- AC4에 인터뷰 기록이 들어 있어 독립적인 완료 조건으로 정리할 필요가 있다.
- 그래프의 참조·호출·구현 관계 및 미해결 관계 구분, 출처·분석 범위·무결성 검증 기준을 강화할 필요가 있다.
- 외부 계약별 검증 시나리오·전제 조건·실행 절차·합격 조건의 연결을 명시할 필요가 있다.

## 전체 Seed YAML

```yaml
goal: 이해한 chainbench의 목적·비전을 바탕으로 문서 X-bar 의미 그래프와 코드 AST 그래프를 생성하고, 응집도와 책임·레이어 분리
  문제에 대한 근거 기반 진단, 목표 구조, 단계별 리팩토링 계획, 호환성 검증 기준·절차를 준비한다.
task_type: code
brownfield_context:
  project_type: brownfield
  context_references:
  - path: /Users/0xtopaz/work/github/0xmhha/chainbench
    role: primary
    summary: 문서·코드 분석과 구조적 진단, 목표 구조 및 리팩토링 계획을 준비할 개선 대상.
  - path: /Users/0xtopaz/work/github/0xmhha/chain/go-wemix
    role: reference
    summary: chainbench의 지원 유지와 연동 계약·호환성 판단을 위한 참조.
  - path: /Users/0xtopaz/work/github/0xmhha/chain/go-wbft
    role: reference
    summary: chainbench의 지원 유지와 연동 계약·호환성 판단을 위한 참조.
  - path: /Users/0xtopaz/work/github/0xmhha/chain/go-stablenet
    role: reference
    summary: chainbench의 지원 유지와 연동 계약·호환성 판단을 위한 참조.
  existing_patterns:
  - '사용자가 적용을 확정한 기존 X-bar 사상: X⁰=주제, complement=필수 논항, adjunct=선택 수식, specifier=대상·맥락.'
  - 기존 CLI·DSL·MCP·설정·결과·세션 등의 외부 계약과 세 체인 지원을 보존하는 내부 개선을 기본으로 한다.
  existing_dependencies: []
constraints:
- 이번 작업은 리팩토링 준비이며 실제 제품 리팩토링을 수행하지 않는다.
- 개선 대상은 /Users/0xtopaz/work/github/0xmhha/chainbench로 한정한다.
- /Users/0xtopaz/work/github/0xmhha/chain/go-wemix, /Users/0xtopaz/work/github/0xmhha/chain/go-wbft,
  /Users/0xtopaz/work/github/0xmhha/chain/go-stablenet은 연동 계약·호환성 분석을 위한 참조이며 해당 저장소
  내부의 구조 개선 계획은 범위에서 제외한다.
- 체인 구성과 구성된 체인 기반 테스트라는 책임 축을 의존 레이어 축과 구별하며, 기존 assistant의 L0/L1/L3 재분류를 확정 설계로
  취급하지 않는다.
- 문서 X-bar 분석은 문서·절의 의미 구조에 X⁰=주제, complement=필수 논항, adjunct=선택 수식, specifier=대상·맥락
  사상을 적용하고 근거 구절을 연결한다. 개별 문장 내부의 언어학적 구문 트리는 제외한다.
- 코드 그래프는 AST를 기반으로 chainbench 전체 패키지·심볼, 정적으로 확인 가능한 함수·메서드 호출, 인터페이스 구현 관계를 포함한다.
- 기본 계획은 CLI·DSL·MCP·설정·결과·세션 등의 외부 계약을 보존하는 내부 리팩토링이며 세 체인 지원을 유지한다.
- 꼭 필요한 외부 계약 변경은 이유·영향·전환 방법·검증 기준을 갖춘 별도 대안으로 제시한다. 대안 작성은 특정 비호환 변경의 구현 승인이 아니다.
- 검증 항목과 실행 절차는 기존 테스트·기록을 근거로 준비한다. 실제 세 체인 E2E 실행과 현재 동작 기준선 확보는 후속 작업이며 과거 기록을
  현재 실행 증거로 취급하지 않는다.
- 산출물 파일명과 표시 형식 등 구현 세부는 실행자가 정한다.
acceptance_criteria:
- description: chainbench의 목적·비전과 구조적 문제를 검토할 수 있는 근거 기반 분석이 제공된다. 분석에는 근거 구절이 연결된
    문서·절 X-bar 의미 그래프, chainbench 전체 패키지·심볼 및 정적으로 확인 가능한 함수·메서드 호출·인터페이스 구현 관계의 AST
    기반 그래프가 포함되고, 문서와 코드 근거를 연결하여 기능별 응집도 저하와 책임·레이어 분리의 모호함을 진단한다. 체인 구성/테스트 책임 축과
    의존 레이어 축은 구별된다.
  semantic_ac_key: ac_cdfeb8f1531aa71c
  verify_exemption_reason: 그래프의 기계적 정합성은 자동 점검할 수 있으나 이 기준 전체의 충족 여부는 문서 의미 분류, 목적·비전
    해석 및 구조적 진단이 원문과 코드 근거에 부합하는지 검토해야 하므로 명령만으로 결정할 수 없다.
- description: 근거 기반 진단에 대응하는 chainbench 목표 구조와 단계별 리팩토링 계획이 제공된다. 기본 계획은 외부 계약과 세
    체인 지원을 보존하고 체인 구성/테스트 책임과 의존 레이어를 구별하여 유지보수성·가독성 개선 방향을 설명한다. 꼭 필요한 외부 계약 변경을
    제안하는 경우 이유·영향·전환 방법·검증 기준이 포함된 별도 대안으로 제시하며, 세 체인 자체의 내부 개선 계획이나 제품 리팩토링 구현은 포함하지
    않는다.
  semantic_ac_key: ac_1c195afafb391715
  verify_exemption_reason: 목표 구조와 단계별 계획이 진단에 적절히 대응하는지, 외부 변경이 필요한지, 제시된 전환 대안이 타당한지는
    설계와 계약 근거에 대한 판단이 필요하여 명령만으로 결정할 수 없다.
- description: 기존 테스트·기록에 근거한 검증 기준과 실행 절차가 제공되어, 후속 작업에서 CLI·DSL·MCP·설정·결과·세션 등 외부
    계약과 go-wemix·go-wbft·go-stablenet 연동 보존 여부를 확인할 수 있다. 실제 세 체인 E2E 실행과 현재 동작 기준선
    확보는 후속 작업으로 명시되고, 과거 기록과 현재 실행 증거가 구별된다.
  semantic_ac_key: ac_116b24ae05a0fcc5
  verify_exemption_reason: 이번 범위는 검증 실행이 아닌 절차 준비이므로 명령 실행 결과로 절차의 충분성과 계약별 검증 적합성을
    확정할 수 없으며 기존 테스트·기록 및 연동 계약과 대조하는 검토가 필요하다.
- description: '[from-user]

    사용자는 내부 개선/외부 변경 설명 후 제시한 권장안을 승인했다. 원문: "권한 방식으로 승인."

    승인된 사전 제시안: 내부 구조 개선을 기본으로 삼고, 외부 변경이 꼭 필요한 지점은 이유·영향·전환 방법을 붙인 별도 대안으로 제시한다.

    구체적인 의미: CLI·DSL·MCP·설정·결과·세션 등의 외부 계약을 보존하는 내부 리팩토링을 기본 계획으로 한다. 꼭 필요한 외부 계약
    변경은 이유, 영향, 전환 방법, 검증 기준을 포함한 별도 대안으로 준비한다. 세 체인 지원은 유지한다. 이번 준비 산출물에 대안을 담는 승인이지
    특정 비호환 변경의 구현을 확정한 것은 아니다.

    확정사항 누적: 산출물은 문서 X-bar 그래프+코드 AST 그래프+근거 기반 진단+목표 구조+단계별 계획+검증 기준. 개선 대상 chainbench
    한정, 세 체인은 연동 계약 분석 참조.

    부모가 확인한 추가 사실: 세 체인의 go.mod는 모두 github.com/ethereum/go-ethereum 모듈 경로이므로 그래프 식별자에
    repository/revision 구분 필요. tests/README.md에 실제 체인 e2e는 별도 e2e 빌드 태그, 기본 go test
    ./... 미포함 명시. 기존 docs/dev/dsl-v2-proposal.md §2.1은 X⁰=head/주제, complement=필수 논항,
    adjunct=선택 수식, specifier=대상·맥락으로 사상한다. 기존 scripts/inventory/code-graph에는 패키지 import+symbol
    inventory가 있고, docs/dev/codegraph는 메서드 호출이 누락된다고 설명한다. 이 사실이 이번 분석 깊이를 확정하지는 않는다.'
  semantic_ac_key: ac_1a72c27d831987d8
ontology_schema:
  name: ChainbenchRefactoringPreparation
  description: chainbench의 목적·비전, 문서 의미 구조, 코드 구조, 연동 계약과 진단 근거를 목표 구조·단계별 계획·검증 절차에
    연결하는 리팩토링 준비 모델.
  fields:
  - name: purpose_and_vision
    type: object
    description: 문서 근거에 기반한 chainbench의 목적과 비전 해석.
    required: true
  - name: repositories
    type: array
    description: 고정된 네 저장소 경로와 chainbench 개선 대상 또는 체인 연동 참조 역할.
    required: true
  - name: document_graph
    type: object
    description: 문서·절의 X-bar 의미 요소와 관계 및 해당 분류를 뒷받침하는 근거 구절.
    required: true
  - name: code_graph
    type: object
    description: chainbench 전체 패키지·심볼, 정적으로 확인 가능한 함수·메서드 호출과 인터페이스 구현 관계.
    required: true
  - name: responsibility_axes
    type: object
    description: 체인 구성/테스트 책임 축과 별도로 구분한 의존 레이어 축.
    required: true
  - name: integration_contracts
    type: array
    description: 세 체인 지원과 연동·호환성 판단에 관련된 계약.
    required: true
  - name: structural_findings
    type: array
    description: 응집도와 책임·레이어 분리 문제 및 이를 뒷받침하는 문서·코드 근거 연결.
    required: true
  - name: target_structure
    type: object
    description: 진단에 대응하는 책임 배치와 의존 관계의 목표 구조.
    required: true
  - name: refactoring_plan
    type: array
    description: 외부 계약 보존을 기본으로 하는 단계별 내부 리팩토링 계획.
    required: true
  - name: external_change_alternatives
    type: array
    description: 필요한 경우에만 제시하는 외부 계약 변경 대안과 이유·영향·전환 방법·검증 기준.
    required: true
  - name: verification_plan
    type: object
    description: 기존 테스트·기록에 근거한 검증 항목·실행 절차와 후속 E2E·현재 기준선 확보 작업의 구분.
    required: true
evaluation_principles:
- name: 근거 추적성
  description: 문서 의미 분류와 구조적 진단을 원문 구절 및 코드 관계로 추적할 수 있고, 사실과 해석을 구별한다.
  weight: 0.3
- name: 분석 범위 충족
  description: 문서·절 의미 분석과 chainbench 전체 패키지·심볼·정적 호출·인터페이스 구현 관계 분석이 승인된 범위를 충족한다.
  weight: 0.25
- name: 설계 타당성
  description: 목표 구조와 단계별 계획이 진단에 대응하며 책임 축과 의존 레이어 축을 구별하여 유지보수성과 가독성 개선을 설명한다.
  weight: 0.25
- name: 호환성과 검증 적합성
  description: 외부 계약과 세 체인 지원 보존을 기본으로 삼고, 검증 절차의 근거와 후속 실행 경계를 명확히 한다.
  weight: 0.2
exit_conditions:
- name: 준비 산출물 완료
  description: 승인된 리팩토링 준비 범위가 충족되면 종료한다.
  criteria: 근거 연결 문서·코드 그래프와 진단, 목표 구조, 단계별 계획, 검증 기준·절차가 모두 제공되고 수용 기준을 충족한다.
- name: 후속 실행 경계 도달
  description: 준비 이후의 구현과 실제 기준선 확보는 별도 작업으로 남긴다.
  criteria: 준비 산출물 완료 후 제품 리팩토링, 세 체인 실제 E2E 실행 또는 현재 동작 기준선 확보로 자동 진행하지 않는다.
metadata:
  seed_id: seed_f339f3d38f1e
  version: 1.0.0
  created_at: '2026-09-13T10:01:27.886805Z'
  ambiguity_score: 0.0375
  gate_forced: false
  interview_id: interview_20260913_090229
  parent_seed_id: null
  generation_mode: normal
  degraded: false
  unresolved_slots: []
  recovery_reason: null
```

## 단일 QA 검토 원문

```text
QA Verdict [Iteration 1]
============================================================
Session: qa-1a8dc6ec
Score: 0.80 / 1.00 [REVISE]
Verdict: revise
Threshold: 0.90

Dimensions:
  Correctness          0.78
  Completeness         0.75
  Quality              0.80
  Intent Alignment     0.94
  Domain Specific      0.73

Differences:
  - AC4 mixes approval history, source observations, duplicated requirements, and a repository/revision identity requirement. It is not a coherent predicate about finished work.
  - The code-graph requirements do not explicitly distinguish symbol references from calls or resolved relationships from unknown targets. An incomplete or misleading graph could satisfy the current wording.
  - AC1 acknowledges mechanically checkable graph integrity but establishes no mandatory validation results or coverage denominator. Its whole-criterion exemption leaves integrity, inventory reconciliation, and reproducibility unenforced.
  - The ontology names the major deliverables but leaves repository/revision identity, source evidence, graph relationships, and external contracts beyond chain integration insufficiently specified. Source snapshots and derived-artifact freshness are also unspecified despite the recorded shared module paths.
  - The compatibility criterion requires verification criteria and procedures without defining their minimum contents or a complete contract-to-check mapping. A generic checklist could satisfy it without making preservation objectively assessable.

Suggestions:
  - Remove AC4 as a standalone criterion. Move approval history and source observations into provenance/context, and merge its unique identity and E2E-tag requirements into the existing graph and verification outcomes.
  - Extend AC1 and code_graph with relationship kinds and resolution states: references remain distinct from calls, confirmed targets are distinguished from candidates or unknowns, and unresolved relationships retain source locations and reasons.
  - Require AC1's delivered graphs to have passing checks for unique identifiers, valid endpoints, resolvable evidence, and coverage reconciled against a defined source/package/symbol inventory and build scope. Require reproducibility against recorded inputs, and limit the exemption to semantic interpretation and diagnostic judgment.
  - Expand ontology descriptions or nested fields to represent repository-qualified identities, revisions and relevant working-tree content, evidence locations, typed relationships, and all external contract surfaces. Define when source changes invalidate derived graphs.
  - Strengthen AC3 to require every inventoried external contract and chain integration to map to source/test evidence, a scenario, prerequisites, execution procedure, and observable pass/fail conditions. Include the separate E2E build tag and explicitly defer current baseline collection and live execution.

Reasoning: The Seed closely preserves the approved preparation scope, compatibility defaults, repository boundaries, and deferred execution. Its first three criteria describe distinct user-visible outcomes and do not need fragmentation. Revision is needed because transcript material remains an acceptance criterion and graph semantics, mechanical verification, ontology coverage, and verification-plan sufficiency are not enforceable enough for the 0.90 threshold. No deduction is made for missing execution evidence from this document.

Loop Action: continue
```


</details>
