# 공통 테스트 분류 반대 검토

현재 common-candidates 139개(exact-config45/adapter-required94) 정적 반대 검토. 원문 카탈로그 수정하지 않음.

오류 2종(영향 4개 케이스)을 확인했다. 분류가 달라지는 사항만 아래에 제시한다.

## CR-01 — TC-028, TC-029, TC-131

exact-config 판정인데 입력 준비 단계에서 istanbul_getWbftExtraInfo와 gasTip 필드에 직접 의존한다. WEMIX3에는 해당 WBFT RPC 구현이 없다.

변경 권고: adapter-required로 이동하고 체인별 fee source 또는 DSL 입력 준비 단계 교체를 implementation_items에 추가. 성공/거부 검증 목적은 공통 후보로 유지.

근거: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/08-legacy-transfer.json:51`, `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/09-dynamic-fee-tx.json:51`, `sources/chainbench/tests/tc/go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json:50`, `sources/go-wbft/consensus/wbft/backend/api.go:417`, `sources/go-stablenet/consensus/wbft/backend/api.go:416`

## CR-02 — TC-128

고정 baseFee 최대값 보장은 세 체인의 동일 공통 능력이 아니다. WBFT London/Croissant의 CalcBaseFee 증가 분기는 별도 max clamp 없이 parentBaseFee+delta를 반환한다. adapter가 유한 max를 선택하면 프로토콜에 없는 상한을 새로 요구하게 된다.

변경 권고: 고정 상한 검증은 configured-cap capability를 가진 체인 조건부 후보로 분리. 모든 세 체인 공통 목록에서 제외하거나 별도 derived 테스트로 부모기반 fee 계산식 검증임을 명시. TC-127도 min clamp와 단순 비음수 lower-bound를 구분한다.

근거: `sources/chainbench/tests/tc/go-stablenet/regression/anzeon/07-basefee-maximum.json:33`, `sources/go-wbft/consensus/misc/eip1559/eip1559.go:83`, `sources/go-stablenet/consensus/misc/eip1559/eip1559.go:72`

## 판정 유지

- **fee balance / effectiveGasPrice**: TC-016은 동일 tx의 BP/EN 값 비교이므로 Stable의 가격 계산 정책이 달라도 목적을 유지. TC-036은 양수 가격과 gasUsed만 검사. 문서형 fee 계산은 이미 adapter-required와 headerGasTip/authorized 정책을 명시.
- **WEMIX3 older EVM fixture**: 선택된 주요 runtime fixture는 PUSH1/MSTORE/RETURN/REVERT/LOG1/JUMP로 작성되어 PUSH0 의존 오류를 확인하지 못함. 문서는 compiler opcode 공통 fork 조건을 이미 명시.
- **WS / sync**: 세 구현의 NewHeads/Logs와 FullSync/SnapSync 경로 존재. 하네스 derived.go의 wsOpen/wsCollected/wsSubscribe 구현 확인. 실제 endpoint 노출과 full/snap 경로 검증은 준비·계측 필요로 명시되어 있어 지원=실행완료 오탐 없음.
- **Brioche / fee delegation**: Brioche 보상은 체인고유 suite 분리 유지. type22는 세 구현에 존재하므로 fork/payer 잔액 정책 adapter를 전제로 공통 후보 유지.
- **7702 / P256**: WBFT IsCroissant(head.Number), Stable AnzeonEnabled의 type4 gate와 WBFT Croissant/Stable Boho P256 map 분리를 코드 재확인. 모든 인용 파일 selected manifest 포함.

## 범위 검증

CR-01만 반영하면139=exact42+adapter97. CR-02를 모든3체인 공통에서 제외하면138=exact42+adapter96이며 TC128은 조건부 별도 목록 유지.

세 프로젝트 selected manifest와 대조하여 chain-differences 인용 45개 모두 빌드 포함을 재확인했다. AST graph도 각 선택파일 mirror 기반으로 존재한다. 함수 호출 그래프가 동적 실행 또는 운영 활성 fork까지 증명하지는 않는다.

TC-127은 약한 **프로필 하한 비교**로 유지한다. W3 정상 PoA 감소경로는 `min(max(parent-delta,1),maxBaseFee)`, WBFT는 `max(baseFee,0)`, Stable Anzeon은 `config.MinBaseFee`다. 동일한 양수 최소값 또는 설정 가능한 floor를 보장하는 테스트라고 해석하지 않는다. 유효 초기 baseFee와 정상 governance params가 전제다. 근거: `sources/go-wemix/consensus/misc/eip1559.go:140`, `sources/go-wbft/consensus/misc/eip1559/eip1559.go:93`, `sources/go-stablenet/consensus/misc/eip1559/eip1559.go:81`.
