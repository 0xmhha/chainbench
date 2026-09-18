# 문서 수행 후보 독립 재검토

문서 수행 후보 47행과 Dual 제외 27행 정적 재검토. 코드/노드 테스트 미실행.

기존 후보 분류를 뒤집을 만한 미지원 오탐은 찾지 못했다. 실행 판정을 잘못 만들 수 있는 명세 누락 2건을 아래에 분리했다. 이는 구현 버그 판정이 아니다. 우선순위 재배치 근거는 발견하지 못했다.

## DOC-RECHECK-01 — Sender 서명 오류 테스트의 결정적 입력 조건 누락

원문의 임의 V/R/S 변경은 다른 유효 sender로 복구될 수 있다. R=0 등 구조적으로 무효한 서명으로 고정해야 invalid-sender 분기를 입증한다.
R=0 등 ValidateSignatureValues가 확실히 거부하는 입력을 고정하고 invalid sender 분기 도달을 확인한다. 임의 비트 변경 후 잔액 부족/feePayer 오류를 서명 오류 검증 성공으로 세지 않는다.

근거: sources/common_test_scenarios.md:33, sources/pr-head/core/types/transaction_signing.go:619, sources/pr-head/core/tx_pool.go:688

## DOC-RECHECK-02 — 온체인 nonce oracle의 블록 태그·초기값 조건 누락

원문의 온체인 전송 수 기대값은 latest/확정 블록과 초기 nonce를 명시해야 성립한다. pending은 pool nonce를 반환하므로 구분 필요.
동일 확정 블록 기준 초기 nonce N 대비 포함된 sender 거래 수 k만큼 N+k를 기대한다. pending 조회는 별도 pool nonce 검증으로 분리한다.

근거: sources/pr-head/internal/ethapi/api.go:1779

## 전체 47행 판정

| ID | 중요도 | 판정 | 근거 | 검토 |
|---|---|---|---|---|
| DOC-C-008 | P0 | 유지 | sources/pr-head/miner/worker.go:1146 | 역순 제출 후 (blockNumber, transactionIndex)로 sender nonce 순서를 대조하는 명세는 유효. pool 수락만으로 채굴 순서를 판정하면 안 된다. |
| DOC-C-040 | P0 | 유지 | sources/pr-head/eth/handler.go:150 | 지원. 동일 높이의 hash/stateRoot 비교 및 실제 full 경로 관측 필요. |
| DOC-C-044 | P0 | 유지 | sources/pr-head/eth/handler.go:282 | 높이 gap만으로 경로 확정하지 않는 조건 적절. |
| DOC-C-045 | P0 | 유지 | sources/pr-head/eth/handler.go:288 | 실제 fetcher 진입 관측 조건 적절. |
| DOC-C-001 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:1879 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-002 | P1 | 유지 | sources/pr-head/core/tx_pool.go:655 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-003 | P1 | 유지 | sources/pr-head/core/tx_pool.go:651 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-004 | P1 | 유지 | sources/pr-head/core/state_transition.go:198 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-005 | P1 | 명세 보완 필요 | sources/pr-head/core/types/transaction_signing.go:619 | 원문의 임의 V/R/S 변경은 다른 유효 sender로 복구될 수 있다. R=0 등 구조적으로 무효한 서명으로 고정해야 invalid-sender 분기를 입증한다. |
| DOC-C-006 | P1 | 유지 | sources/pr-head/core/types/transaction_signing.go:204 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-007 | P1 | 유지 | sources/pr-head/core/tx_pool.go:713 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-010 | P1 | 유지 | sources/pr-head/core/tx_pool.go:720 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-011 | P1 | 유지 | sources/pr-head/core/tx_pool.go:673 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-012 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:1879 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-013 | P1 | 유지 | sources/pr-head/core/tx_list.go:285 | 실제 PriceBump와 두 fee cap 인상을 명시하여 원문 10% 고정 오류를 피했다. 원문은 queued 교체 시나리오이며 pending 교체 전체 커버리지로 확대 해석하지 않는다. |
| DOC-C-014 | P1 | 유지 | sources/pr-head/core/vm/evm.go:499 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-015 | P1 | 유지 | sources/pr-head/core/vm/evm.go:168 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-019 | P1 | 유지 | sources/pr-head/core/vm/evm.go:236 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-020 | P1 | 유지 | sources/pr-head/core/vm/evm.go:237 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-021 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:738 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-023 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:1992 | 마이닝 전 txpool 관찰이 필요하므로 이식 시 채굴 제어 또는 nonce gap 등 관측 창 확보. |
| DOC-C-028 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:950 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-029 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:970 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-031 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:1838 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-032 | P1 | 명세 보완 필요 | sources/pr-head/internal/ethapi/api.go:1779 | 원문의 온체인 전송 수 기대값은 latest/확정 블록과 초기 nonce를 명시해야 성립한다. pending은 pool nonce를 반환하므로 구분 필요. |
| DOC-C-042 | P1 | 유지 | sources/pr-head/core/blockchain.go:220 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-009 | P1 | 유지 | sources/pr-head/core/tx_pool.go:693 | 이미 로컬 DropUnderPriced/EffectiveGasTip과 원격 GasTipCap을 구분했으므로 오탐 아님. |
| DOC-C-041 | P1 | 유지 | sources/pr-head/eth/handler.go:368 | 조건부 및 fallback 제외 문구 적절. |
| DOC-C-016 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:1206 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-017 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:1218 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-018 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:1170 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-022 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:843 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-024 | P2 | 유지 | sources/pr-head/eth/filters/api.go:327 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-026 | P2 | 유지 | sources/pr-head/eth/filters/api.go:208 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-027 | P2 | 유지 | sources/pr-head/eth/filters/api.go:238 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-030 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:1798 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-033 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:92 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-034 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:104 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-035 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:119 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-036 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:241 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-037 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:191 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-039 | P2 | 유지 | sources/pr-head/core/genesis.go:239 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-043 | P2 | 유지 | sources/pr-head/p2p/server.go:584 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-D-001 | P2 | 유지 | sources/pr-head/eth/api.go:748 | Brioche config·wemixapi.Info 조건 적절. P2 간접 회귀 분류 유지. |
| DOC-D-026 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:1218 | 원문 자체가 authorization 없는 일반 전송 baseline이다. 조건부 일반 estimateGas 이식은 정당하며 EIP-7702 지원으로 세지 않는 문구도 적절. |
| DOC-C-025 | P3 | 유지 | sources/pr-head/internal/ethapi/api.go:729 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| DOC-C-038 | P3 | 유지 | sources/pr-head/internal/ethapi/api.go:2402 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |

## 제외 항목 대조

- DOC-D-002..023: 제외 유지: WBFT 고유 oracle을 PoA로 그대로 적용할 수 없음 (sources/pr-head/eth/ethconfig/config.go:217)
- DOC-D-024..025: 제외 유지: Type 0x4 decoder 및 authorization gas 지원 없음 (sources/pr-head/core/types/transaction.go:193)
- DOC-D-027..029: 제외 유지: 0x100 P256 precompile 미등록. 빈 반환은 지원 증거가 아님 (sources/pr-head/core/vm/contracts.go:84)
