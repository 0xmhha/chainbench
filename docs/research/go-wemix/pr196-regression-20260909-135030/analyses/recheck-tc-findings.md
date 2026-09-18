# JSON 수행 후보 재검토

> ID 표기는 Confluence 원래 명세 ID를 우선한다. 실행 ID는 별도 표기한다. ID 대응은 전체 명세 충족이나 PASS를 뜻하지 않는다. [ID 대응표](/Users/wm-it-25_0220/Work/github/chainbench/docs/research/common-tests/mainnet-dependencies-20260909/analyses/existing-tc-specs.md)

기존 후보 86개와 제외 103개의 JSON 조건을 확인했다. 실행 테스트는 하지 않았다. 원본 분석 문서와 JSON은 수정하지 않았다.

## 교정할 내용

### TC-R1 [P1] 이식 대상 40개의 applicableChains 변경 안내가 빠졌다

공통 이식 지침에 top-level applicableChains를 wemix로 변경 또는 wemix를 포함하도록 수정한다. env.chain만 바꾸면 SKIP되며 실행 커버리지로 세면 안 된다.

대상: TC-1-3-04 (분석 TC-013), TC-1-3-05 (분석 TC-014), TC-1-3-06 (분석 TC-015), TC-5-3-01 (분석 TC-019), RT-G-1-03 (분석 TC-022), RT-G-1-04 (분석 TC-023), RT-G-1-05 (분석 TC-024), RT-G-5-01 (분석 TC-027), RT-A-2-01 (분석 TC-028), RT-A-2-02 (분석 TC-029), RT-A-2-03 (분석 TC-030), RT-A-2-04 (분석 TC-031), TX-010 [부분 대응] (분석 TC-032), RT-A-2-05a (분석 TC-033), RT-A-2-06 (분석 TC-034), RT-A-2-07 (분석 TC-035), RT-A-2-08 (분석 TC-036), RT-A-2-09 (분석 TC-037), TX-013 [부분 대응] (분석 TC-038), RT-A-3-01 (분석 TC-039), RT-A-3-05 (분석 TC-040), RT-A-3-06 (분석 TC-041), RT-A-3-07 (분석 TC-042), RT-A-4-03 (분석 TC-043), RT-A-4-04 / RPC-014 [부분 대응] (분석 TC-044), RT-D-01 (분석 TC-045), RT-D-03 (분석 TC-046), RT-D-04 (분석 TC-047), RT-D-05 (분석 TC-048), RT-D-03 / TX-014 [부분 대응] (분석 TC-049), RT-D-04 / TX-015 [부분 대응] (분석 TC-050), RT-D-05 / TX-016 [부분 대응] (분석 TC-051), RT-A-4-02 (분석 TC-065), RT-A-1-01 (분석 TC-067), RT-A-4-06 (분석 TC-068), RT-A-4-07 (분석 TC-069), remote-rpc-health [명세 ID 미확인] (분석 TC-079), remote-chain-info [명세 ID 미확인] (분석 TC-080), remote-balance-check [명세 ID 미확인] (분석 TC-081), sample-minimal-value-transfer [명세 ID 미확인] (분석 TC-082)

근거: sources/recheck-chainbench/internal/testengine/capability.go:16, sources/recheck-chainbench/internal/testengine/validate.go:260

### TC-R2 [P1] 공통 기능의 JSON 11개를 전용 기능으로 일괄 제외했다

기존 adapt 정책과 같은 기준으로 별도 이식 후보로 복원한다. 고정된 Stablenet 수수료, WBFTExtra.GasTip, 1초를 그대로 기대하지 않고 아래 항목별 조건으로 바꾼다. 각각 원문 실행 가능으로 승격하는 뜻은 아니다.

대상: TC-4-6-02 (분석 TC-105), RT-C-03 [부분 대응] (분석 TC-124), RT-C-04 [부분 대응] (분석 TC-125), RT-C-05 [부분 대응] (분석 TC-126), RT-C-06 (분석 TC-127), RT-C-07 (분석 TC-128), TC-1-3-03 [부분 대응] (분석 TC-129), TC-1-3-02 [부분 대응] (분석 TC-130), RT-A-2-07 / TX-012 [부분 대응] (분석 TC-131), RT-G-2-01 [부분 대응] (분석 TC-133), RT-B-01 (분석 TC-178)

근거: sources/pr-head/core/tx_pool.go:671, sources/pr-head/consensus/misc/eip1559.go:72, sources/pr-head/internal/ethapi/api.go:100, sources/pr-head/wemix/admin.go:436

### TC-R3 [P2] 동일 기능의 문서 명세와 JSON 우선순위가 엇갈린다

methodPresent RT-G-5-01 (분석 TC-027)은 RT-G-5-01 (분석 DOC-C-038)과 P3, Brioche BRIOCHE-02 / RPC-008 [부분 대응] (분석 TC-055)는 RPC-008 (분석 DOC-D-001)과 P2, chainId RT-A-1-01 (분석 TC-067)은 RT-A-4-05 / RPC-013 (분석 DOC-C-025)와 P3로 통일한다. 환경 기동 순서와 위험 중요도를 구분한다.

대상: RT-G-5-01 (분석 TC-027), BRIOCHE-02 / RPC-008 [부분 대응] (분석 TC-055), RT-A-1-01 (분석 TC-067)

근거: analyses/02-wemix-priority.md, analyses/source-crosswalk.md

## 누락된 이식 후보 11개

이 항목들은 현재 JSON 그대로 실행하는 대상이 아니다. 기존 78개 adapt와 동일하게 입력과 기대값을 go-wemix 정책으로 바꿔야 한다.

| ID | 우선순위 | 필요한 교정 |
|---|---|---|
| TC-4-6-02 <br> 실행: effective-gas-price-regular (effective-gas-price-regular) | P1 | 영수증 effectiveGasPrice를 해당 tx의 type 및 실제 gasPrice/feeCap/tipCap, 포함 블록 baseFee로 계산한다. RT-A-2-08 <br> 실행: effective-gas-price과 중복으로 연결하고 BP/EN 비교는 실제 구현이 없음을 명시한다. |
| RT-C-03 [부분 대응] <br> 실행: anzeon-basefee-increase (anzeon-basefee-increase) | P1 | PoA governance gasTargetPercentage보다 높은 gasUsed를 실제로 확인한다. 단순 25% load로 증가를 단정하지 않고 maxBaseFee에서 떨어진 초기 baseFee를 쓴다. |
| RT-C-04 [부분 대응] <br> 실행: anzeon-basefee-stable (anzeon-basefee-stable) | P1 | gasUsed == gasLimit * gasTargetPercentage / 100인 부모 블록을 만들어 다음 baseFee 불변을 비교한다. 임의 10% load는 충분하지 않다. |
| RT-C-05 [부분 대응] <br> 실행: anzeon-basefee-decrease (anzeon-basefee-decrease) | P1 | 초기 baseFee가 최저값 1 wei보다 크고 목표보다 낮은 gasUsed인 부모 블록에서 다음 baseFee 감소를 검사한다. |
| RT-C-06 <br> 실행: basefee-minimum (basefee-minimum) | P1 | 20,000,000,000,000 wei를 go-wemix 최저값 1 wei로 교체한다. 단순 한 번 읽는 assertion의 한계를 명시한다. |
| RT-C-07 <br> 실행: basefee-maximum (basefee-maximum) | P1 | Stablenet 고정 상한 대신 PoA governance maxBaseFee를 사용하고 상한 도달 조건을 만든다. |
| TC-1-3-03 [부분 대응] <br> 실행: feecap-above-min-accepted (feecap-above-min-accepted) | P1 | eth_maxPriorityFeePerGas 및 대상 노드 txpool 하한으로 tip을 준비한다. feeCap 여유와 수신 승인/채굴 성공을 검증한다. |
| TC-1-3-02 [부분 대응] <br> 실행: feecap-exact-min-accepted (feecap-exact-min-accepted) | P1 | txpool의 해당 시점 baseFee + 유효 tip 경계로 설정한다. head 변화로 경계가 바뀌지 않도록 관측/채굴 제어한다. |
| RT-A-2-07 / TX-012 [부분 대응] <br> 실행: gaslimit-exceeded-rejected (gaslimit-exceeded-rejected) | P1 | Istanbul tip 읽기를 교체하고 나머지는 유효한 tx로 gasLimit+1에 따른 ErrGasLimit을 검사한다. 이미 채택한 RT-A-2-07 <br> 실행: gas-limit-exceeds-block-rejected와 동일 기능이다. |
| RT-G-2-01 [부분 대응] <br> 실행: gas-price-equals-basefee-plus-tip (gas-price-equals-basefee-plus-tip) | P2 | Istanbul GasTip 대신 같은 head 시점의 eth_maxPriorityFeePerGas 값을 사용하여 eth_gasPrice == baseFee + suggestedTip을 비교한다. 샘플링 중 head 변경을 통제한다. |
| RT-B-01 <br> 실행: block-period-one-second (block-period-one-second) | P2 | WBFT 1초 상수를 PoA blockCreationTime/idle 정책에 맞는 기대값으로 바꾼다. parentHash 연결로 시각을 읽는 공통 검증을 재사용한다. |

## 오탐으로 판정하지 않은 의심

- **direct 5개는 모두 오탐인가**: 현재 코드에서 일괄 강등 근거 없음. 5개 모두 env.chain=wemix이며 바이너리/키/PoA 준비를 이미 문서가 조건으로 명시한다. BRIOCHE-02 / RPC-008 [부분 대응] (분석 TC-055)는 현재 embedded genesis의 briocheBlock=0과 JSON brioche 설정으로 보상 조건이 구성된다. 기존 분석 시점 genesis 사본이 없으므로 현시점 보강 근거로 구분한다.
- **낮은 feeCap은 반드시 queued인가**: 아니다. PR-head sources/pr-head/core/tx_pool.go:697과 sources/pr-head/params/protocol_params.go:193에서 DropUnderPriced=true이며 local도 effective tip 하한 미만이면 거부한다. queued를 일률 기대값으로 고치면 오류다. local 여부, DropUnderPriced, txpool gasPrice를 명시한다.
- **eth_signRawFeeDelegateTransaction 또는 fee delegation 자체 미지원인가**: 아니다. PublicTransactionPoolAPI.SignRawFeeDelegateTransaction이 존재한다(sources/pr-head/internal/ethapi/api.go:2404). tx type=22(0x16)가 존재하며 fork 활성화 후 허용한다. 현재 chainbench helper도 feePayerKey 및 sendRawTampered를 지원한다. 통신 오류를 서명 거부로 세지 않는 추가 assertion은 기존 문서의 공통 주의사항을 적용한다.
- **제외된 전용 시스템 계약/WBFT/7702/P256**: 11개 복원 후보 외 나머지는 원문 oracle이 특정 전용 기능을 필요로 하므로 제외를 유지한다. 임의 EVM 계약 배포로 원래 시스템 계약 의미를 바꾸는 식의 확대는 하지 않았다.

현재 helper 증거는 sources/recheck-chainbench 아래에 별도로 복사했다. sources/recheck-chainbench-manifest.json에 파일별 UTC 시각과 SHA-256이 있다. 원래 PR-head 스냅샷과 같은 시점으로 취급하지 않는다.

## TC-R4: EN 노드 준비 누락

RT-A-2-04 (분석 TC-031), TX-010 [부분 대응] (분석 TC-032), RT-A-2-09 (분석 TC-037), TX-013 [부분 대응] (분석 TC-038), RT-D-01 (분석 TC-045), RT-D-03 (분석 TC-046), RT-D-04 (분석 TC-047), RT-D-05 (분석 TC-048), RT-D-03 / TX-014 [부분 대응] (분석 TC-049), RT-D-04 / TX-015 [부분 대응] (분석 TC-050), RT-D-05 / TX-016 [부분 대응] (분석 TC-051), RT-A-4-07 (분석 TC-069)는 on=en1을 사용하지만 topology는 bp=4만 정의한다. env.topology.en=1 추가를 이식 지침에 명시한다.

RT-A-2-04 (분석 TC-031), TX-010 [부분 대응] (분석 TC-032)는 최종 nonce==3만 관찰하는 partial이므로 P1을 유지한다. RT-A-2-04 / TX-010 (분석 DOC-C-008)과 assertion이 동등하지 않다.

recheck-tc-findings.json의 proposed_changes에 case_id별 status/priority/reason/required_changes를 명시했다.
