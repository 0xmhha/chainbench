# Seed v1.0.2 — 기준선 반영 QA

QA: **0.92 / 1.00 PASS** (권고 기준 0.90). 이전 0.88 대비 +0.04. 세션 `qa-1a8dc6ec`, iteration 3. 한 번 평가했으며 점수 상승을 위한 자동 재수정은 수행하지 않았다.

이 점수는 Seed 명세의 품질이다. 그래프·목표 구조·계획 산출물의 완성이나 세 체인 전체 호환성 통과를 의미하지 않는다. Stablenet 실패와 미검증 범위는 유지된다.

## 반영 내역

- 사용자가 별도 승인해 수집한 `baselines/20260913T101856Z`를 근거로 연결.
- 기준선 전체가 후속이라는 문구를 수집 완료 부분과 후속 검증으로 구분.
- 기본 테스트·CLI·WBFT·handoff PASS, Stablenet FAIL, 27 SKIP 및 미검증 범위 명시.
- 소스/바이너리 해시 범위, 환경 실패 시도, 명령·제한 시간·정리·증거 연결 반영.
- 기존 목표·ontology 이름/필드·세 수용 기준 ID 유지, 버전 1.0.1 → 1.0.2.
- 기준선 파일 해시와 Ouroboros Seed 스키마 검증 통과.

## 잔여 자문

체인 바이너리와 CLI의 재빌드 여부를 더 명확히 구분하고, 분석 자료의 지시문 경계·기존 사용자 변경 보존·그래프 분석 명령의 제한 시간을 명시하라는 권고가 남았다. 이번 평가 후에는 추가 수정하지 않았다.

## QA 원문

```text
QA Verdict [Iteration 3]
============================================================
Session: qa-1a8dc6ec
Score: 0.92 / 1.00 [PASS]
Verdict: pass
Threshold: 0.90

Dimensions:
  Correctness          0.95
  Completeness         0.88
  Quality              0.90
  Intent Alignment     0.98
  Domain Specific      0.89

Differences:
  - '바이너리는 재빌드하지 않았다'는 참조의 '체인 바이너리' 한정을 빠뜨려 CLI 빌드와의 구분이 모호하다.
  - prompt_injection: 분석에 사용하는 문서·주석·로그 안의 지시를 근거 데이터로만 취급한다는 경계가 명시되지 않았다.
  - dirty_worktree: dirty 상태와 입력 해시는 기록하지만, 산출물 작성 시 기존 tracked/untracked 사용자 변경과 경로 충돌을 처리하는 보존 규칙이 없다.
  - hung_command: 기준선 수집의 제한 시간과 정리는 다루지만, 이번에 실행할 그래프 분석·기계 검증 명령의 시간 제한과 중단 판정은 명시하지 않았다.

Suggestions:
  - 해당 문구를 '체인 바이너리는 재빌드하지 않았고 CLI는 현재 소스로 빌드했다'로 한정한다.
  - 외부 근거에 포함된 지시는 Seed의 범위·검증 기준을 변경하거나 명령 실행을 승인하지 않는다는 제약을 추가한다.
  - 기존 사용자 변경을 보존하고 충돌하는 산출물은 별도 출력 경로에 작성하며 강제 덮어쓰기·reset·clean을 하지 않는다는 제약을 추가한다.
  - 그래프 분석·검증 명령에도 기록된 제한 시간과 자식 프로세스 정리를 적용하고 TIMEOUT·중단은 미완료로 판정하도록 명시한다.

Reasoning: 핵심 범위와 완료 상태가 구체적이며 ontology가 주요 산출물과 추적 관계를 포괄한다. 그래프 무결성·범위 대조·재현성 및 계약 매핑의 기계 검증을 의미 검토 면제와 명확히 분리했다. 제공된 부분 기준선의 Stablenet 실패, SKIP, 미검증 범위를 전체 호환성 성공으로 확대하지 않는다. 남은 차이는 국소적인 명확화와 실행 경계 보완이다. 문서 자체를 평가했으며 제품 테스트 성공이나 추가 실행 증거를 요구하지 않았다.

Loop Action: done
```

## 평가된 Seed 전체

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
  - path: /Users/0xtopaz/work/github/0xmhha/chainbench/docs/dev/architecture/baselines/20260913T101856Z/README.md
    role: reference
    summary: 사용자가 별도로 승인하여 2026-09-13 수집한 부분적 동작 기준선. 소스 snapshot·바이너리 해시·명령·테스트 JSON·실패 로그는 같은 디렉터리에
      보존한다. 전체 호환성 통과를 뜻하지 않는다.
  existing_patterns:
  - '사용자가 적용을 확정한 기존 X-bar 사상: X⁰=주제, complement=필수 논항, adjunct=선택 수식, specifier=대상·맥락.'
  - 기존 CLI·DSL·MCP·설정·결과·세션 등의 외부 계약과 세 체인 지원을 보존하는 내부 개선을 기본으로 한다.
  - 인터뷰 승인 범위와 변경 이력은 refactoring-preparation-scope.md 및 chainbench-refactoring-preparation.seed-review.md에
    보존한다. 승인 대화 자체는 수용 기준이 아니다.
  - 세 참조 체인은 동일한 github.com/ethereum/go-ethereum 모듈 경로를 사용하므로 모듈 경로만으로 저장소나 심볼을 식별하지 않는다.
  - 기존 scripts/inventory/code-graph와 docs/dev/codegraph는 재사용 후보이며 전체 호출·구현 분석 충족의 증거로 간주하지 않는다. tests/e2e
    문서의 빌드 태그·바이너리 조건은 후속 절차에 반영한다.
  - '기준선 20260913T101856Z: chainbench 7f39c5627e0bdaccaa8e207a67767d71579283f5, darwin/arm64 Go 1.25.13.
    기본 테스트 58 패키지 PASS, 테스트/하위 테스트 이벤트 1944 PASS·27 SKIP·0 FAIL, 테스트 없는 패키지 12 SKIP. CLI 빌드와 help/chains/capabilities
    PASS.'
  - 같은 기준선에서 WBFT 기본 E2E 및 Wemix→WBFT handoff PASS. Stablenet 기본 E2E는 블록 증가·receipt 검사 이후 수신 잔액 0→0으로
    FAIL, 컨트랙트 단계 미도달. 원인은 미확정이며 이미 알려진 실패와 이후 회귀를 구분한다.
  - 기준선의 binaries.json·source-snapshots.json·commands.json·test-summary.json·artifact-sha256.json 및 e2e-*.log를
    참조한다. 바이너리는 재빌드하지 않았으며 기록된 SHA-256에 대해서만 실행 증거가 유효하다. 소스 해시는 추적 파일 범위이고 untracked 파일 내용은 포함하지 않는다.
  - 기준선 수집에서 Go 자동 도구체인/오프라인 설정 오류와 샌드박스 포트 제한 시도는 별도 환경 실패로 보존했다. 권한 제한 해소 후 20260913T101856Z 결과만 동작
    기준선으로 사용한다.
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
- 검증 항목과 실행 절차는 기존 테스트·기록 및 별도 승인으로 수집 완료한 20260913T101856Z 동작 기준선을 근거로 준비한다. 이 기준선은 수집 시점과 기록된 입력에 한정되며
  과거 기록을 새 실행 증거로 취급하지 않는다. 추가 E2E·재수집과 제품 수정은 이번 Seed 반영으로 자동 실행하지 않는다.
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
    연동 보존 여부를 확인할 수 있다. 수집 완료한 부분 기준선의 결과와 아직 실행하지 않은 검증을 분리하고, 과거 기록·수집 시점 실행 증거·후속 실행을 구별한다. 외부 계약 inventory의
    각 항목은 고유 ID, 표면 종류(CLI/DSL/MCP/설정/결과/세션/체인 연동), 현재 문서·코드 근거와 적용 체인을 갖는다. 각 계약에 시나리오, 전제조건·도구/바이너리/환경변수·작업
    디렉터리·입력, 실제 명령 또는 구체적인 수동 단계, 관찰값·기대 결과·명시적 성공/실패 조건, 증거 보관 방법을 연결한다. 기존 검증은 파일·테스트명·기록 revision으로
    연결하고 부족한 검증은 추가 필요로 표시하되 후속 작성/실행 절차와 판정 기준을 제공한다. 적용되지 않는 조합은 근거를 기록한다. 계약별 매핑 누락과 필수 필드 공백이 0건인
    자동 대조 보고서가 제공된다. e2e 빌드 태그와 바이너리 준비를 기본 테스트와 구별하고 SKIP/미실행/환경 부족을 성공으로 세지 않는다. 호환성 성공은 해당 입력과 실제 PASS
    시나리오에 한정하고 전체 호환성 판정은 미검증 항목의 후속 실행까지 보류하며, 이번 완료 판정은 절차의 완전성과 근거 적합성에 한정한다. 소스/계약/도구/빌드 조건 변경 시 영향받는
    그래프·진단·검증 항목의 재확인 조건을 기록한다. 기준선 README 및 명령·해시·테스트 이벤트에 검증 항목을 연결하며, 27 SKIP과 Stablenet의 알려진 실패를 누락
    없이 표시한다. Stablenet 실패는 준비 문서 완료를 막지 않지만 Stablenet 지원이 검증됐다는 근거로 사용할 수 없다. 미실행 hardfork/장애/동기화/거버넌스·Wemix
    독립 전체·race·성능/장시간 항목은 미검증으로 남긴다. 기준선 시간·PID·임시 경로 등 가변값과 계약 관찰값을 구별하여 비교한다.
  semantic_ac_key: ac_116b24ae05a0fcc5
  verify_exemption_reason: 기준선은 수집된 PASS/FAIL/SKIP을 사실대로 반영하며 추가 체인 실행 성공을 이번 준비 완료 조건으로 요구하지 않는다. 면제는
    절차의 의미적 적합성 검토에 한정한다. 계약 매핑 누락·필수 필드 공백 검사와 인용된 기준선 파일 해시 확인은 필수다.
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
      분석 입력 snapshot 확보와 수집된 동작 기준선을 구별한다.
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
    description: 기존 테스트·기록에 근거한 검증 항목·실행 절차와 수집된 부분 기준선·후속 검증 작업의 구분. 필수 데이터는 계약 inventory와의 대조표, verification_id,
      contract_id, scenario, prerequisites, cwd, inputs, 명령/구체적 수동 단계, expected observations, pass/fail/skip
      조건, evidence 수집 위치, 기존 테스트/기록 참조 또는 추가 필요 표시, 실행 상태다. preparation_checks에는 그래프·매핑 검증의 실제 명령과 성공
      보고서를 보관하고 baseline_records에는 baseline_id, 수집 시각, source/binary 해시 범위, 명령·로그 참조, PASS/FAIL/SKIP,
      알려진 실패와 미검증 범위를 보관한다. deferred_checks에는 미실행 항목과 재수집 절차를 보관한다. 기록은 수집 시각으로 구분하며 새 실행으로 오인하지 않는다.
      snapshot·도구·계약 변경에 따른 재확인 조건을 포함한다. 수집 절차의 제한 시간, 중단·정리 방법, TIMEOUT/환경 실패 상태도 commands.json 및 collect.py에
      연결하고 성공에서 제외한다.
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
  description: 준비 이후 구현과 추가 실환경 검증은 별도 작업으로 남긴다.
  criteria: 이미 수집한 부분 기준선을 준비 산출물에 반영하고 종료한다. 제품 리팩토링, 알려진 실패 수정, 추가 E2E 또는 기준선 재수집으로 자동 진행하지 않는다.
metadata:
  seed_id: seed_f339f3d38f1e
  version: 1.0.2
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
