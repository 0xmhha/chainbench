# Web UI Seed 보강 기록

> **[현행 설계의 보강 기록]** — 기능 범위를 변경하지 않은 1회 보강이다.
> Seed `seed_2a20eeab7a55`, 1.0.0 → 1.0.1, 2026-10-07.

최초 advisory QA는 0.87 / REVISE였다. 사용자가 “docs/chain-analysis/ … 이것들도 활용해야해”,
“seed 보강은 진행해줘”로 체인 명령·옵션 활용과 기존 QA 제안 두 가지의 적용을 승인했다.
초기 Seed를 재생성하지 않고 YAML을 직접 수정했다. 원래 goal, 14개 outcome과
semantic_ac_key, ontology 이름, 확정 운영 정책은 유지한다.

## Wonder와 Reflect

| 후보 | 근거 | 유형 | 채택 상태 |
|---|---|---|---|
| A 도메인·관계·brownfield 근거 연결 | QA + 다섯 lateral 관점 | sharpen | 사용자 승인 보강에 포함, 적용 |
| B WEB별 실행 명령·관측·증거와 필수 skip 거절 | QA + 다섯 lateral 관점 | sharpen | 사용자 승인 보강에 포함, 적용 |
| C 체인 표면/엔진 지원/실제 binary의 출처와 교차 검증 | 사용자 명시 요청 + 다섯 lateral 관점 | sharpen | 사용자 명시 요청에 따라 적용 |
| D ontology 이름을 도메인 이름으로 교체 | architect 단독 | rename | 비채택; 원래 온톨로지 이름 보존 |

채택 후보의 균형은 expand 0 / sharpen 3 / remove 0이다. 논점의 공통 결론은
“바이너리가 받는 명령·옵션”과 “현재 엔진이 실행/방출하는 명령·옵션”을 구분해야 한다는 것이다.
모든 binary CLI를 새 엔진 기능으로 구현하거나 raw 명령 실행을 추가하는 방향은 후보 C의
의미로 채택하지 않았다. 기존 사용자 승인에 없는 추가 범위는 이번 보강에 넣지 않았다.

Socrates 대조는 인터뷰 ledger와 사용자 메시지를 기준으로 했다. 개인/팀 동시 지원,
공유 설정과 개인 자격증명의 분리, 현행 DSL 전체와 의미 보존, 실제 로컬/SSH 검증은
이미 확정된 조건이며 문서 분석을 이유로 축소하지 않았다. 새로운 요구 질문은 없었다.

Ouroboros lateral의 hacker/researcher/simplifier/architect/contrarian을 각각 독립 실행했다.
팬아웃 `fanout_7331eab2d6984d04beb66f06b21f54e4`의 5개 결과를 persona로 상관시켰고
반환 artifact를 읽어 누락 관점이 없음을 확인했다. 세부 산출물은 로컬 분석 ledger에 둔다.

## Refine와 Restate의 적용 차이

- brownfield의 빈 목록 → primary 저장소와 명세/체인 분석/읽기 전용 체인 소스의 경로·패턴·의존성.
- ontology의 빈 fields → 16개 도메인 개념과 소유·식별·관계·상태·상세 계약의 근거.
- “모든 내용은 host verbatim”이라는 최초 생성기 문구 → 확정 요구와 승인된 보강의 정확한 출처.
- 14개 acceptance criterion → 원래 outcome/key를 보존하고 `verify_command`, `expected_artifacts`,
  `output_assertion` 추가. runner는 구현 작업이 먼저 만들며 지금 통과했다고 주장하지 않는다.
- 체인별 네 산출물 → 지원 범위·command-local·hidden/absent·fingerprint·schema를 연결하는
  동작 명세와 `ChainSurface` API 계약. 계획에 고정할 chainSurfaces 추가.
- 종료 판정 → 필수 live skip·mock-only·이전 invocation 증거로 완료할 수 없음을 명시.
- 체인 분석 README의 이전 `internal/core/launchopt` 경로 → 현재 소유자 `internal/core/nodeconfig/launchopt.go`.

QA를 보강본에 한 번 다시 적용했다. advisory **PASS — 0.94 (bar 0.90)**이며
결과는 [seed-qa.json](chainbench-web-ui.seed-qa.json)에 저장했다. MCP evaluator는 설정된 Codex
실행 파일 부재로 판정을 반환하지 못해 로컬 QA Judge 역할로 평가했다. 수동 advisory이다.
모델·YAML·API/schema·링크·지원 참조·성공 계약 검증은 문서 품질 검증이며 제품 인수 검증이 아니다.
실제 브라우저/로컬/SSH 검증은 구현 이후 필수다. 추가 보강 패스는 별도 사용자 요청 없이 반복하지 않는다.

문서 검증 결과: Seed 모델 PASS, 14개 structured success contract, 16개 ontology field,
OpenAPI 37경로/47operation/56schema, 내부 참조 426개·예제 14개·문서 링크 37개 확인.
기존 `nodeconfig`의 `TestDialectSpellingsExistInTheBinaries`와
`TestDialectsDoNotClaimFlagsTheOtherGenerationLacks`는 둘 다 PASS, skip 0이었다.
이는 저장된 CLI 캡처와 현재 dialect의 대조이며 선택 바이너리의 신규 실측을 대체하지 않는다.

사용자 홈의 `~/.ouroboros/seed-revisions`는 이 작업의 쓰기 허용 경로 밖이어서 이 기록과
응답에 같은 audit를 남긴다. 기존 `.gitignore`와 별도 refactoring Seed 작업은 보존했다.
