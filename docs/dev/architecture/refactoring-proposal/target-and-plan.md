# Chainbench 목표 구조와 단계별 리팩토링 제안 (AC2)

> Review update: this is the earlier P0–P5 proposal, not an approved schedule. Follow the [current CLI-first review](cli-composition-review.md) before accepting its boundaries. MCP follows CLI; dashboard/daemon follow MCP. Shared records, `.dsl` migration and `chainbenchd` require explicit additions to this plan.

세 geth 계열 체인의 네트워크 구성·검증·테스트를 공통 Go 코어와 CLI/MCP/dashboard에서 일관되게 제공한다. 합의 패밀리 재사용과 체인별 선언적 확장이 비전이다.

리팩토링 준비 제안만 작성한다. 개선 대상은 chainbench이며 참조 세 체인의 내부 개선, 제품 코드 수정, 추가 E2E·기준선 재수집은 제외한다.

검토 상태와 범위: [제안 개요](README.md). 진단의 사실·해석·원문은 [분석 근거](analysis-evidence.md), 검증 절차는 [검증 계획](verification.md)에 연결한다. 이 문서는 기존 AC2 제안을 PR 검토용으로 옮긴 것이며 최종 승인된 구조가 아니다.

## 책임 축과 의존 축

책임 축은 체인 구성과 구성된 체인 기반 테스트이며, 공용 기능은 두 책임을 지원한다. 의존 축은 표면→유스케이스/조립→구성·테스트 오케스트레이션→관측/서비스다. 책임과 의존을 동일한 레이어 번호로 분류하지 않는다.

아래 경로는 목표 배치다. `(제안)` 경로는 현재 제품에 존재한다고 주장하지 않는다. 목록의 의존 방향은 개선 경계의 목표 규칙이며 기존 전체 import가 이미 이를 만족한다는 뜻이 아니다. shared services 내부 전체 레이어 재설계는 하지 않는다.

```text
chainbench/
  cmd/chainbench, internal/mcp, internal/dashboard   표면
  internal/app                                     유스케이스·조립
    composition/ (제안)                            환경 포트 구현
  internal/chainsetup                              구성·readiness·lease
  internal/consensus/upgrade                        handoff 구성
  internal/testengine                              환경 요청·실행·결과
  internal/dsl                                     문법·Registry 소비
  internal/core/observation/ (제안)                 관측 경계
    progress, receipt, canonical, propagation       내부 모듈
  internal/core, accounts, resource, nodemonitor    기존 서비스 재사용
  internal/chains, consensus/poa, consensus/wbft    기존 어댑터 유지
```

testengine은 자신의 환경 포트를 호출하고 app이 구현체를 주입한다. 런타임에서는 테스트→구성 요청이 흐르지만 testengine→app import를 만들지 않는다. bridge는 소비자 타입을 구현할 때 testengine의 포트 정의를 참조할 수 있다. 이 예외는 명시적 타입 의존이며 testengine에서 bridge로의 역방향 import는 금지한다.

| 소유자 / 목표 위치 | 책임 | 허용 의존 대상 | 진단 |
|---|---|---|---|
| surface / `cmd/chainbench; internal/mcp; internal/dashboard` | 입력 파싱·출력 렌더링만 담당. 구성/테스트 정책을 표면마다 복제하지 않는다. | app | F2, F3 |
| app / `internal/app` | 기존 입력·출력 타입, 기본값, 오류 및 의존 주입의 진입점. alias 일괄 제거 대신 변경되는 내부 경계만 어댑트한다. | bridge, testing, composition, services, chain-adapters | F2, F3, F5 |
| bridge / `internal/app/composition` | testengine의 환경 요청 포트를 구현. 정규화된 환경 요청을 ChainUpIn 또는 HandoffInputs로 변환하고 실행·lease 반환. DSL 문법을 소유하지 않는다. | composition, handoff, services, testing | F1, F2 |
| composition / `internal/chainsetup` | 키·genesis·설정·배치·프로세스·workspace·구성 readiness와 소유한 네트워크 해제. 테스트 assertion이나 결과 판정을 가져오지 않는다. | observation, services, chain-adapters | F1, F2, F6 |
| handoff / `internal/consensus/upgrade` | 기존 Wemix→WBFT 혼합 바이너리 handoff의 단계와 fork 전후 구성. 테스트 환경 요청에서 선택되며 세 체인 내부 구조는 변경하지 않는다. | services, chain-adapters | F1 |
| testing / `internal/testengine` | DSL 환경 정규화, compose/attach/reuse 요청, action/assertion, 결과·세션 조립. 소비자 소유 환경 포트로 lease를 받아 테스트하고 반환한다. | dsl, observation, services | F1, F4, F6 |
| dsl / `internal/dsl` | v1/v2 문법·이름 해석 및 interp.Registry 소비자 인터페이스. 구현 후보를 실제 호출 대상으로 간주하지 않는다. | services | F4 |
| observation / `internal/core/observation` | 성공 값과 RPC 오류를 구분한 관측 자료 및 진행·receipt·canonical·전파 판정. 내부 하위 모듈별 소유권은 boundaries에 명시. app/testengine/chainsetup를 import하지 않는다. | services | F6 |
| services / `internal/core; internal/accounts; internal/resource; internal/nodemonitor` | 기존 RPC·health·process·filestore·report·session·계정 기능 재사용. core/observation은 이 집합에서 제외한다. nodemonitor는 순수 분류를 유지하고 재시작 실행은 구성 소유자에게 요청한다. | 상위 의존 없음 | F1, F2, F4, F6 |
| chain-adapters / `internal/chains; internal/consensus/poa; internal/consensus/wbft` | chainbench 안의 Wemix/WBFT/Stablenet capability·manifest·합의 패밀리 연결을 유지. 모듈 이름 대신 repository/snapshot과 체인 identity로 식별. | services | F1, F6 |

유지보수성은 구성 기본값/실행 정책의 변경 지점을 구성 어댑터에 모으고, 테스트에는 환경 의도와 assertion을 남기는 것으로 개선한다. 가독성은 성공 receipt·합의 진행·canonical 상태·전파를 다른 결과로 읽을 수 있게 하는 데서 얻는다. 인터페이스 추가 비용과 DTO 중복 위험 때문에 F1·F2는 실제 변경 이유가 생긴 경계부터 도입한다.

## 진단 추적표

| 진단 | 원 분석의 지위 / 설계 대응 | 책임 / 의존 규칙 | 단계 | 검증 항목 |
|---|---|---|---|---|
| F1 | candidate: 위험 후보: 테스트가 환경 의도를 소유하고 구성 어댑터가 ChainUp/Handoff 정책 연결을 소유. 기존 체인 분기 없는 선언적 선택을 유지. | bridge, composition, handoff, testing, services, chain-adapters; 각 owner의 allowed_dependencies | P1, P5 | AC2-V2, AC2-V5 |
| F2 | candidate: 위험 후보: forwarding 공유 장점 유지. 변경 필요 경계에만 변환을 두고 alias 전체 재작성은 하지 않음. | surface, app, bridge, composition, services; 각 owner의 allowed_dependencies | P2, P5 | AC2-V3, AC2-V5 |
| F3 | confirmed: 현재 CLI→app 경로를 문서화하고 표면/유스케이스 책임을 구별. 번호 중심 재분류 제외. | surface, app; 각 owner의 allowed_dependencies | P0, P2, P5 | AC2-V1, AC2-V3 |
| F4 | confirmed: 기존 Registry 주입을 보존하는 설계 제약. implements 관계만으로 동적 대상을 고정하지 않음. | testing, dsl, services; 각 owner의 allowed_dependencies | P1, P5 | AC2-V4 |
| F5 | confirmed: 과거 중복 진단을 새 결함으로 취급하지 않음. 제거 대상은 현재 참조와 계약 시험으로 다시 확인. | app; 각 owner의 allowed_dependencies | P0, P2, P5 | AC2-V1 |
| F6 | confirmed: B1–B6 소유권과 실패 자료를 분리. 구조 추출과 판정 강화를 분리하고 원본 실패 원인은 미확정으로 유지. | composition, testing, observation, services, chain-adapters; 각 owner의 allowed_dependencies | P3, P4, P5 | AC2-V5, AC2-V6, AC2-V7, AC2-V8 |

각 진단의 공통 근거 ID (새 레코드를 복제하지 않고 AC1 원장을 참조):

- F1: ev:20a1d59977e9aa47fa0ed2f7107bea8167b623c2666f7d2cb06d73b28d8ac390, ev:7551b641431c3bf631233dbc2754c89b1dd8cd6d8344e0e5844187a599994355, ev:2f25bc6d42da781775cf15188a332557252c635e2d12c225d10ab2be2c3fa231
- F2: ev:f95ba0f333b62b7ed8607ce214b8244b5e8f19fb99db292a08fdf3fc328699d7, ev:bf304cdf79a8be8a1f3c2aa001274558a25c48c25a35fd5aaf0449238dee13f4, ev:2927509ef84e189d697e8a9309c85d8d38a4943033a1fdd2a2ab9798927a6ed6
- F3: ev:fef97b7fae1488dcf8b264cc4038a311c5bfdcde761e20c3b774aba5ec0d895d, ev:b70f032c7ebf81797c757e4e2ba5760bb862e8b0f3f31aa9ce7088faf8fd741d, ev:50869b3af4d3aa8a6ef14edffa46108c28b73e73d306bab23067c41d378b9698
- F4: ev:5f910ba1e49a520e2c1a28cd73a01b2ba320621242e7782ecb67493cacc3b687, ev:33b99f2e8fcf9b655f80ae6b8a2b31ae9c7796b3fd904332deca78f0751eceb1, ev:e7fdf7d8902392dfb8f24acdb1a7d52e06b80a421a0c096099529c1b2402f914
- F5: ev:88191dccc80fba8b05e60d965a0edbb8bbe20d1f150eb09bbb268aa9d863a80e, ev:1f969aeb905aa8c49aeae3aa209bbe389812e64b0ac01f1ea08fb44ed75a6fb4
- F6: ev:5dfd021e8b04425e46a10b5f2ff552a71f6fc092fc3f270d6c8b8e25278801d2, ev:ec869ea971a42e671ffa5dc034a3e66006925dd931e527ca9b194a93970ab7fd, ev:c20f6d48b5963d3120e5f6d3adda7736647b7627b885f703003b4c3bbb9cacd3, ev:7285a01ec0cc18424086f478855ff5b0f0741046853403035be7c1eee7a1dc93, ev:a94dbc1853066c60ea94afa7bc7987824c4b34d51f8b970d1fcaeb7f6385e87d, ev:239a89c47f41fca599ca7d50be045b76a49bbdba44c37d7211783ffb7db733dc
## Stablenet 경계의 목표 소유권

20260913T101856Z는 과거 부분 기준선이다. 원본 Stablenet FAIL은 유지한다. restored node1/3/4의 block 1·1 ETH와 node2/5의 genesis는 저장 시점의 차이다. instrumented 150 잔액 응답 1 ETH 및 unmodified 1회 PASS(42.47초)는 원인 해결·안정성 확보를 증명하지 않는다. receipt/latest 관측 시점 차이는 미확정 후보이며 영구 송금 손실은 복원 상태와 부합하지 않는다.

아래 상태명은 내부 설계 어휘이며 새 CLI/MCP 오류 코드나 결과 schema의 제안이 아니다. 관측 누락은 성공으로 채우지 않는다. receipt block 지정 조회가 미지원이면 명시적으로 남긴다.

### B1 · 구성 readiness

소유자: composition — `internal/chainsetup (기존 단계 내부 책임)`

**입력:** authoritative chain ID/genesis hash, node identity·role, endpoint, 설정 provenance, 실행 소유권·deadline

**출력:** 실행 lease + 노드별 RPC 성공/identity 일치 관측. readiness는 합의 진행이나 송금 성공을 보장하지 않는다.

**실패 상태:** CONFIG_INVALID, IDENTITY_MISMATCH, RPC_ERROR, PROCESS_EXITED, TIMEOUT, INTERRUPTED

**인수 항목:** AC2-V6

### B2 · 합의 진행

소유자: observation — `internal/core/observation/progress (제안)`

**입력:** B1 노드 집합, 성공한 최초 높이·hash, 체인별 기존 대기 정책, deadline

**출력:** 두 성공 관측 간 높이 증가 + 노드별 head/hash/시각. -1→0은 진행으로 세지 않는다. validator 참여·전체 수렴은 별도다.

**실패 상태:** RPC_ERROR, NO_VALID_INITIAL_SAMPLE, NO_PROGRESS, TIMEOUT, INTERRUPTED

**인수 항목:** AC2-V6

### B3 · 거래 제출/receipt

소유자: testing — `internal/testengine (시나리오); internal/core/observation/receipt (관측 제안)`

**입력:** run/node identity, 서명 요청 to/value/chain identity, 제출 endpoint, deadline; accounts 인코딩 재사용

**출력:** txHash, 제출 시각·오류, receipt status/blockNumber/blockHash. 성공 receipt는 canonical 상태나 모든 노드 전파를 대신하지 않는다.

**실패 상태:** SUBMIT_REJECTED, SUBMIT_OUTCOME_UNKNOWN, RPC_ERROR, RECEIPT_PENDING, RECEIPT_REVERTED, TIMEOUT, INTERRUPTED

**인수 항목:** AC2-V7

### B4 · canonical 상태 검증

소유자: observation — `internal/core/observation/canonical (제안)`

**입력:** B3 receipt block identity, 같은 노드의 해당 높이 header, 명시적 state block 참조, 주소/기대값·deadline

**출력:** 동일 blockHash의 canonical 확인과 그 block의 state 값·stateRoot·시각. number 조회만 가능하면 조회 전후 hash를 확인하고 불일치는 성공에서 제외. latest 관측은 별도 필드로 보존.

**실패 상태:** REORG_OR_HASH_MISMATCH, STATE_UNAVAILABLE, RPC_ERROR, ASSERTION_MISMATCH, UNSUPPORTED_BLOCK_REFERENCE, TIMEOUT, INTERRUPTED

**인수 항목:** AC2-V7

### B5 · 노드 간 전파

소유자: observation — `internal/core/observation/propagation (제안)`

**입력:** 명시한 대상 노드/role 집합, B4 기준 block identity, 기존 시나리오의 수렴 정책·deadline

**출력:** 각 노드의 같은 높이 hash/stateRoot/state 및 관측 시각; 도달 못한 노드는 lagging, 같은 높이 다른 hash는 disagreement. 단일 receipt로 전체 수렴을 추론하지 않는다.

**실패 상태:** LAGGING, HASH_DISAGREEMENT, STATE_DISAGREEMENT, RPC_ERROR, TIMEOUT, INTERRUPTED

**인수 항목:** AC2-V8

### B6 · 실패 진단/정리

소유자: testing — `internal/testengine/failuredata.go; internal/chainsetup (lease 해제)`

**입력:** B1–B5 관측 이력, run/node identity, original/restored/instrumented/unmodified 구분, lease의 owned PID/process identity·deadline

**출력:** 테스트 실행 소유자가 실패 bundle을 봉인한 뒤 구성 소유자에 release 요청. primary failure와 cleanup failure를 별도 보존; cleanup_status 및 remaining_processes 반환. attach lease는 사용자 노드를 종료하지 않는다.

**실패 상태:** DIAGNOSTIC_PARTIAL, CLEANUP_FAILED, REMAINING_PROCESS, TIMEOUT, INTERRUPTED

**인수 항목:** AC2-V8

관측 공통 필드: run/node identity, txHash, receipt blockNumber/blockHash, canonical 확인 여부, state 조회 block 참조, 시각, 노드별 head/stateRoot, RPC 오류, cleanup 상태. 실패 전 자료 수집에도 deadline을 적용하며 진단 실패 때문에 정리를 무한정 지연하지 않는다. 실환경 수집의 구체적 절차/시간 예산은 별도 검증 AC에서 정의한다.

## 단계별 제안

### P0 · 근거·계약·현재 경로를 고정

**선행 단계:** 없음

**선행 조건:** 새 작업 승인, 대상 해시·dirty 재확인 및 변경분 격리. 이전 단계의 인수 조건 통과; 미검증 항목은 통과로 간주하지 않는다.

**변경 대상:** `README.md`, `docs/dev/architecture/code-health-review-2026-09-10.md`, `internal/arch`

후속 작업 시작 시 현재 graph와 문서 충돌을 정리하고 기존 계약 fixture를 고정. 이력 문서는 수정된 과거 주장으로 표시. 기존 L0/L1/L3 번호를 재분류의 전제로 삼지 않는다.

**목표 소유자 / 진단:** surface, app / F3, F5

**보존 표면:** CLI, DSL, MCP, configuration, results, sessions, three-chain support

**인수 항목:** AC2-V1, AC2-V3, AC2-V5

**되돌림 조건·방법:** 현재 snapshot을 확정할 수 없거나 문서 수정이 새 동작을 약속하면 해당 문서 변경만 되돌리고 분석을 갱신.

### P1 · 환경 요청과 구성 실행 소유권 분리

**선행 단계:** P0

**선행 조건:** 새 작업 승인, 대상 해시·dirty 재확인 및 변경분 격리. 이전 단계의 인수 조건 통과; 미검증 항목은 통과로 간주하지 않는다.

**변경 대상:** `internal/testengine/compose.go`, `internal/testengine/attach.go`, `internal/testengine/wire.go`, `internal/app`, `internal/chainsetup`, `internal/consensus/upgrade`

먼저 compositionOf의 순수 환경 정규화와 I/O를 함수 단위로 식별·특성화. 소비자 포트는 testengine에 두고 app/composition에서 ChainUp/Handoff 변환·실행을 구현. overlay·key 기본값·override 거부·reuse fingerprint의 순서/내용은 그대로 이동. 기존 진입점 위임을 유지하고 앱 조립에서 주입한다. attach는 borrowed lease, compose는 owned lease를 반환.

**목표 소유자 / 진단:** testing, bridge, composition, handoff, dsl / F1, F4

**보존 표면:** CLI, DSL, MCP, configuration, results, sessions, three-chain support

**인수 항목:** AC2-V2, AC2-V4, AC2-V5

**되돌림 조건·방법:** 변환 출력·부수효과 순서·handoff·attach 소유권이 다르거나 import cycle 발생 시 어댑터 연결을 이전 구현으로 되돌림. 생성 데이터 삭제나 session 마이그레이션으로 해결하지 않음.

### P2 · 표면 타입과 내부 구성 변화 사이의 어댑터 제한

**선행 단계:** P1

**선행 조건:** 새 작업 승인, 대상 해시·dirty 재확인 및 변경분 격리. 이전 단계의 인수 조건 통과; 미검증 항목은 통과로 간주하지 않는다.

**변경 대상:** `internal/app/net.go`, `internal/mcp`, `cmd/chainbench/networkcmd/network.go`

모든 alias를 새 DTO로 복제하지 않는다. P1에서 실제로 내부 타입 변경이 필요한 요청만 기존 app 타입/직렬화 형식을 유지하는 변환으로 감싼다. forwarding에는 정책을 추가하지 않는다. 현상 유지가 더 단순한 alias는 유지하고 근거를 기록.

**목표 소유자 / 진단:** surface, app, bridge / F2, F3, F5

**보존 표면:** CLI, DSL, MCP, configuration, results, sessions, three-chain support

**인수 항목:** AC2-V1, AC2-V3, AC2-V5

**되돌림 조건·방법:** 필드 누락·zero/default·marshal 결과·오류 차이가 발견되면 해당 변환만 제거하고 기존 alias/forwarding으로 복귀.

### P3 · 관측과 수명주기 경계를 동작 보존 상태로 분리

**선행 단계:** P2

**선행 조건:** 새 작업 승인, 대상 해시·dirty 재확인 및 변경분 격리. 이전 단계의 인수 조건 통과; 미검증 항목은 통과로 간주하지 않는다.

**변경 대상:** `tests/e2e/harness_test.go`, `internal/testengine/nodegate.go`, `internal/testengine/failuredata.go`, `internal/core/rpc`, `internal/core/health`, `internal/chainsetup`

B1–B6의 소유권에 따라 관측 자료/내부 포트를 추출. 기존 RPC·health·accounts·nodemonitor를 재사용. 이 단계는 판정/대기 정책을 바꾸지 않고 기존 진입점에서 위임한다. B4/B5의 추가 관측과 더 엄격한 판정은 P4 인수 전 기본 경로에 연결하지 않는다.

**목표 소유자 / 진단:** composition, testing, observation, services / F6

**보존 표면:** CLI, DSL, MCP, configuration, results, sessions, three-chain support

**인수 항목:** AC2-V5, AC2-V6, AC2-V7, AC2-V8

**되돌림 조건·방법:** 기존 결과나 RPC 순서·대기시간이 바뀌거나 관측 추가로 타이밍이 변하면 해당 경계 연결을 되돌린다. 결과 판정 변경은 P4로 분리.

### P4 · 경계 판정 강화의 별도 후속 검토

**선행 단계:** P3

**선행 조건:** 새 작업 승인, 대상 해시·dirty 재확인 및 변경분 격리. 이전 단계의 인수 조건 통과; 미검증 항목은 통과로 간주하지 않는다.

**변경 대상:** `tests/e2e/harness_test.go`, `internal/testengine/nodegate.go`, `internal/testengine/failuredata.go`

B1–B6의 목표 실패 상태를 fake 관측으로 먼저 검증. -1 sentinel 제거, 성공 sample 간 진행, canonical block 일치, 명시 노드 전파, 실패/정리 결과 분리는 판정 강화 후보다. 승인된 기존 계약의 의미와 일치하는 내부 진단만 연결 가능. PASS/FAIL·timeout·외부 오류/출력 변경이 필요하면 기본 리팩토링에서 제외하고 별도 대안 검토로 돌린다. 영구 송금 손실이나 특정 race 수정이라고 부르지 않는다.

**목표 소유자 / 진단:** composition, testing, observation / F6

**보존 표면:** CLI, DSL, MCP, configuration, results, sessions, three-chain support

**인수 항목:** AC2-V6, AC2-V7, AC2-V8, AC2-V5

**되돌림 조건·방법:** 알려진 FAIL을 대기/재시도로 숨기거나 미확정 원인을 해결로 주장하면 수락 중단. 외부 판정/형식 변화는 연결 전 보류, 이미 실험한 경우 실험 경계만 복귀하고 로그 보존.

### P5 · 사용되지 않는 연결 제거 및 설명 통합

**선행 단계:** P4

**선행 조건:** 새 작업 승인, 대상 해시·dirty 재확인 및 변경분 격리. 이전 단계의 인수 조건 통과; 미검증 항목은 통과로 간주하지 않는다.

**변경 대상:** `internal/app`, `internal/testengine`, `internal/chainsetup`, `internal/arch`, `README.md`, `docs/dev/architecture`

승인된 단계에서 실제 도입한 경계만 문서·import 규칙에 반영. P4가 계약 변화로 보류되면 P3 구조만 정리하고 P4 미실행을 명시. 정적 참조 외 name dispatch/등록 사용도 확인한 후 옛 내부 위임을 제거. 패키지 수 감소 자체를 목표로 삼지 않는다.

**목표 소유자 / 진단:** app, bridge, testing, composition, observation / F1, F2, F3, F4, F5, F6

**보존 표면:** CLI, DSL, MCP, configuration, results, sessions, three-chain support

**인수 항목:** AC2-V1, AC2-V2, AC2-V3, AC2-V4, AC2-V5, AC2-V6, AC2-V7, AC2-V8

**되돌림 조건·방법:** 이름 등록·반사·기존 세션 재개·세 체인 capability 회귀가 보이면 제거한 위임을 복구. 사용자 workspace와 결과 파일은 롤백 대상이 아님.

각 단계는 후속 구현 작업의 제안이며 현재 실행 상태는 모두 PROPOSED_NOT_EXECUTED다. P4는 판정 강화의 별도 수락 관문이다. P5는 P4의 수락 또는 명시적 보류 결정을 선행 조건으로 삼으며 보류를 검증 PASS로 기록하지 않는다.

## 단계 인수 항목

아래는 진단→단계 추적에 필요한 항목만 정의한다. 완전한 외부 계약 inventory와 실행 절차는 별도 검증 AC의 산출물에서 연결해야 한다. 기존 테스트 경로는 근거/재사용 후보이며 새 경계의 모든 조건이 이미 시험된다는 의미가 아니다.

### AC2-V1

README의 현재 경로와 AST import/call 근거를 대조. 목표/현재를 분리하고 승인된 단계의 import 변화만 허용. 과거 중복 숫자로 새 변경을 정당화하지 않음.

기존 시험: `internal/arch/layers_test.go`, `internal/arch/packagetree_test.go`

추가 필요: 새 bridge/observation이 생기는 후속 단계에서 규칙 갱신 전 현재 테스트 의미와 실제 graph 대조.

상태: DEFERRED_DESIGN_GATE

### AC2-V2

v1/v2, caller override, generated/prepared keys, topology/role/binary, placement, attach/reuse 및 handoff의 요청·오류·부수효과 순서가 이동 전후 동일.

기존 시험: `internal/testengine/compose_internal_test.go`, `internal/testengine/compose_preset_configs_test.go`, `internal/chainsetup/compose_test.go`, `internal/app/attachrun_test.go`

추가 필요: 환경 포트 fake로 old/new 변환 차등 시험과 lease release/attach 소유권 시험. 기존 시험만으로 전체 보존을 주장하지 않음.

상태: DEFERRED_DESIGN_GATE

### AC2-V3

CLI/MCP 요청과 출력·오류, 생략/zero/default, workspace 경로, JSON/YAML 필드·세션·결과 형식이 동일. alias 유지 또는 어댑터 변환의 모든 필드를 검사.

기존 시험: `internal/app/net_test.go`, `internal/app/workspaceconfig_test.go`, `internal/mcp/network_tools_test.go`, `cmd/chainbench/chaincmd/parity_test.go`

추가 필요: 변경되는 타입에 한해 roundtrip/golden fixture 추가; marshal·reflection 차이가 있으면 alias 교체 보류.

상태: DEFERRED_DESIGN_GATE

### AC2-V4

action 이름·인자·capability·v1/v2 해석 및 등록 표면 일치. pointer/value implements 간선과 name dispatch 실제 경로를 구별.

기존 시험: `internal/dsl/catalog_test.go`, `internal/dsl/schemasync_test.go`, `internal/testengine/wire_test.go`, `internal/mcp/capability_tools_test.go`

추가 필요: 새 포트 주입이 필요한 지점의 fake registry 시험; DSL 이름/구문 변경 없음.

상태: DEFERRED_DESIGN_GATE

### AC2-V5

세 체인 등록·capability와 기존 설정/결과/세션 읽기 유지. 과거 WBFT 및 Wemix→WBFT PASS와 Stablenet FAIL을 각각 보존.

기존 시험: `internal/chains/wemix`, `internal/chains/wbft`, `internal/chains/stablenet`, `internal/app/sessions_test.go`, `internal/testengine/runtoreport_internal_test.go`

추가 필요: 후속 호환성 AC의 계약별 검증 ID에 연결 후 단계 수락. 추가 E2E·바이너리 재빌드/기준선 재수집은 별도 승인 전 실행하지 않음.

상태: DEFERRED_DESIGN_GATE

### AC2-V6

readiness와 progress 분리. RPC error→0, valid 0→0, valid 0→1, wrong identity, deadline/cancel을 구별. wanted identity는 관측값에서 역산하지 않음.

기존 시험: `tests/e2e/harness_test.go`, `internal/testengine/nodegate_test.go`, `internal/testengine/nodegatefacts_internal_test.go`

추가 필요: fake RPC/clock 기반 경계 시험 필요. 기존 E2E 하네스 취약점과 원본 실패의 인과관계는 미확정.

상태: DEFERRED_DESIGN_GATE

### AC2-V7

제출 불명 상태에서 자동 재송금 금지; receipt success와 same-block state 검증 분리. receipt hash 변경, latest 지연, state unavailable, reverted receipt, block 참조 미지원이 성공으로 소거되지 않음.

기존 시험: `internal/app/txwait_test.go`, `internal/accounts/accounts_test.go`, `tests/e2e/stablenet_chain_test.go`

추가 필요: fake RPC 응답열로 receipt/canonical boundary 시험 추가 필요. 명시 block 지원을 체인마다 확인하지 못하면 UNKNOWN/UNSUPPORTED 유지.

상태: DEFERRED_DESIGN_GATE

### AC2-V8

1/0/1/1/0 높이는 지연과 fork를 구분. 실패 자료 보존 후 owned lease만 정리; 정리 실패·잔여 프로세스는 별도 오류. restored/instrumented/unmodified 기록 혼합 금지.

기존 시험: `tests/e2e/stablenet_block_propagation_test.go`, `internal/testengine/collect_test.go`, `internal/chainsetup/reuse_targeted_stop_test.go`

추가 필요: 노드별 same-height hash/stateRoot, log write 실패, cleanup timeout/cancel, attach lease 시험 추가 필요. 새 네트워크 실행은 별도 승인 대상.

상태: DEFERRED_DESIGN_GATE

## 외부 계약 및 범위 결정

현재 진단만으로 꼭 필요한 외부 계약 변경은 입증되지 않았다. 따라서 별도 비호환 대안은 제안하지 않는다. 후속 P4에서 필요성이 입증되면 이유·영향·전환 방법·검증 기준을 갖춘 독립 대안을 먼저 작성해야 하며, 이 문서는 그 변경이나 구현을 승인하지 않는다.

기본 계획은 CLI 이름/flag/default/exit·출력, DSL 문법/이름/인자, MCP tool/schema/오류, 설정 우선순위/경로/직렬화, 결과·세션 파일/재개, 세 체인 capability와 handoff를 보존한다. 내부 타입 이동을 이유로 외부 이름·파일 형식을 변경하지 않는다.

## 검토 및 검증 상태

목표 구조·계획은 정식 의미 평가에서 통과했지만 전체 준비 산출물은 승인되지 않았다. 원본 그래프의 최신 입력 검증과 전역 ID·근거 참조 검사, 평가기의 기계 명령 처리 문제는 [남은 검토 항목](README.md)에 기록한다. 이 문서의 인수 항목은 후속 구현의 조건이며 현재 구현 완료나 호환성 통과를 뜻하지 않는다.
