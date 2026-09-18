# JSON 수행 후보 재검토

기존 후보 86개와 제외 103개의 JSON 조건을 확인했다. 실행 테스트는 하지 않았다. 원본 분석 문서와 JSON은 수정하지 않았다.

## 교정할 내용

### TC-R1 [P1] 이식 대상 40개의 applicableChains 변경 안내가 빠졌다

공통 이식 지침에 top-level applicableChains를 wemix로 변경 또는 wemix를 포함하도록 수정한다. env.chain만 바꾸면 SKIP되며 실행 커버리지로 세면 안 된다.

대상: TC-013, TC-014, TC-015, TC-019, TC-022, TC-023, TC-024, TC-027, TC-028, TC-029, TC-030, TC-031, TC-032, TC-033, TC-034, TC-035, TC-036, TC-037, TC-038, TC-039, TC-040, TC-041, TC-042, TC-043, TC-044, TC-045, TC-046, TC-047, TC-048, TC-049, TC-050, TC-051, TC-065, TC-067, TC-068, TC-069, TC-079, TC-080, TC-081, TC-082

근거: sources/recheck-chainbench/internal/testengine/capability.go:16, sources/recheck-chainbench/internal/testengine/validate.go:260

### TC-R2 [P1] 공통 기능의 JSON 11개를 전용 기능으로 일괄 제외했다

기존 adapt 정책과 같은 기준으로 별도 이식 후보로 복원한다. 고정된 Stablenet 수수료, WBFTExtra.GasTip, 1초를 그대로 기대하지 않고 아래 항목별 조건으로 바꾼다. 각각 원문 실행 가능으로 승격하는 뜻은 아니다.

대상: TC-105, TC-124, TC-125, TC-126, TC-127, TC-128, TC-129, TC-130, TC-131, TC-133, TC-178

근거: sources/pr-head/core/tx_pool.go:671, sources/pr-head/consensus/misc/eip1559.go:72, sources/pr-head/internal/ethapi/api.go:100, sources/pr-head/wemix/admin.go:436

### TC-R3 [P2] 동일 기능의 문서 명세와 JSON 우선순위가 엇갈린다

methodPresent TC-027은 DOC-C-038과 P3, Brioche TC-055는 DOC-D-001과 P2, chainId TC-067은 DOC-C-025와 P3로 통일한다. 환경 기동 순서와 위험 중요도를 구분한다.

대상: TC-027, TC-055, TC-067

근거: analyses/02-wemix-priority.md, analyses/source-crosswalk.md

## 누락된 이식 후보 11개

이 항목들은 현재 JSON 그대로 실행하는 대상이 아니다. 기존 78개 adapt와 동일하게 입력과 기대값을 go-wemix 정책으로 바꿔야 한다.

| ID | 우선순위 | 필요한 교정 |
|---|---|---|
| TC-105 (effective-gas-price-regular) | P1 | 영수증 effectiveGasPrice를 해당 tx의 type 및 실제 gasPrice/feeCap/tipCap, 포함 블록 baseFee로 계산한다. TC-036과 중복으로 연결하고 BP/EN 비교는 실제 구현이 없음을 명시한다. |
| TC-124 (anzeon-basefee-increase) | P1 | PoA governance gasTargetPercentage보다 높은 gasUsed를 실제로 확인한다. 단순 25% load로 증가를 단정하지 않고 maxBaseFee에서 떨어진 초기 baseFee를 쓴다. |
| TC-125 (anzeon-basefee-stable) | P1 | gasUsed == gasLimit * gasTargetPercentage / 100인 부모 블록을 만들어 다음 baseFee 불변을 비교한다. 임의 10% load는 충분하지 않다. |
| TC-126 (anzeon-basefee-decrease) | P1 | 초기 baseFee가 최저값 1 wei보다 크고 목표보다 낮은 gasUsed인 부모 블록에서 다음 baseFee 감소를 검사한다. |
| TC-127 (basefee-minimum) | P1 | 20,000,000,000,000 wei를 go-wemix 최저값 1 wei로 교체한다. 단순 한 번 읽는 assertion의 한계를 명시한다. |
| TC-128 (basefee-maximum) | P1 | Stablenet 고정 상한 대신 PoA governance maxBaseFee를 사용하고 상한 도달 조건을 만든다. |
| TC-129 (feecap-above-min-accepted) | P1 | eth_maxPriorityFeePerGas 및 대상 노드 txpool 하한으로 tip을 준비한다. feeCap 여유와 수신 승인/채굴 성공을 검증한다. |
| TC-130 (feecap-exact-min-accepted) | P1 | txpool의 해당 시점 baseFee + 유효 tip 경계로 설정한다. head 변화로 경계가 바뀌지 않도록 관측/채굴 제어한다. |
| TC-131 (gaslimit-exceeded-rejected) | P1 | Istanbul tip 읽기를 교체하고 나머지는 유효한 tx로 gasLimit+1에 따른 ErrGasLimit을 검사한다. 이미 채택한 TC-035와 동일 기능이다. |
| TC-133 (gas-price-equals-basefee-plus-tip) | P2 | Istanbul GasTip 대신 같은 head 시점의 eth_maxPriorityFeePerGas 값을 사용하여 eth_gasPrice == baseFee + suggestedTip을 비교한다. 샘플링 중 head 변경을 통제한다. |
| TC-178 (block-period-one-second) | P2 | WBFT 1초 상수를 PoA blockCreationTime/idle 정책에 맞는 기대값으로 바꾼다. parentHash 연결로 시각을 읽는 공통 검증을 재사용한다. |

## 오탐으로 판정하지 않은 의심

- **direct 5개는 모두 오탐인가**: 현재 코드에서 일괄 강등 근거 없음. 5개 모두 env.chain=wemix이며 바이너리/키/PoA 준비를 이미 문서가 조건으로 명시한다. TC-055는 현재 embedded genesis의 briocheBlock=0과 JSON brioche 설정으로 보상 조건이 구성된다. 기존 분석 시점 genesis 사본이 없으므로 현시점 보강 근거로 구분한다.
- **낮은 feeCap은 반드시 queued인가**: 아니다. PR-head sources/pr-head/core/tx_pool.go:697과 sources/pr-head/params/protocol_params.go:193에서 DropUnderPriced=true이며 local도 effective tip 하한 미만이면 거부한다. queued를 일률 기대값으로 고치면 오류다. local 여부, DropUnderPriced, txpool gasPrice를 명시한다.
- **eth_signRawFeeDelegateTransaction 또는 fee delegation 자체 미지원인가**: 아니다. PublicTransactionPoolAPI.SignRawFeeDelegateTransaction이 존재한다(sources/pr-head/internal/ethapi/api.go:2404). tx type=22(0x16)가 존재하며 fork 활성화 후 허용한다. 현재 chainbench helper도 feePayerKey 및 sendRawTampered를 지원한다. 통신 오류를 서명 거부로 세지 않는 추가 assertion은 기존 문서의 공통 주의사항을 적용한다.
- **제외된 전용 시스템 계약/WBFT/7702/P256**: 11개 복원 후보 외 나머지는 원문 oracle이 특정 전용 기능을 필요로 하므로 제외를 유지한다. 임의 EVM 계약 배포로 원래 시스템 계약 의미를 바꾸는 식의 확대는 하지 않았다.

현재 helper 증거는 sources/recheck-chainbench 아래에 별도로 복사했다. sources/recheck-chainbench-manifest.json에 파일별 UTC 시각과 SHA-256이 있다. 원래 PR-head 스냅샷과 같은 시점으로 취급하지 않는다.

## TC-R4: EN 노드 준비 누락

TC-031, TC-032, TC-037, TC-038, TC-045, TC-046, TC-047, TC-048, TC-049, TC-050, TC-051, TC-069는 on=en1을 사용하지만 topology는 bp=4만 정의한다. env.topology.en=1 추가를 이식 지침에 명시한다.

TC-031/032는 최종 nonce==3만 관찰하는 partial이므로 P1을 유지한다. DOC-C-008과 assertion이 동등하지 않다.

recheck-tc-findings.json의 proposed_changes에 case_id별 status/priority/reason/required_changes를 명시했다.
