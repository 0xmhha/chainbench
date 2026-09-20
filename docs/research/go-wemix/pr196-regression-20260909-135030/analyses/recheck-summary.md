# 우선순위 후보와 추가 테스트 재검토 결과

> ID 표기는 Confluence 원래 명세 ID를 우선한다. 실행 ID는 별도 표기한다. ID 대응은 전체 명세 충족이나 PASS를 뜻하지 않는다. [ID 대응표](/Users/wm-it-25_0220/Work/github/chainbench/docs/research/common-tests/mainnet-dependencies-20260909/analyses/existing-tc-specs.md)

정적 재검토에서 후보 누락, 실행 준비 조건 누락, 정상 동작을 실패로 판정할 수 있는 기대값을 확인하여 두 문서와 대응 JSON을 교정했다. 소스·테스트 수정과 실행은 하지 않았다.

기존 PR head `5a93553cb25d41770c7419cd325229ac81bb3042`의 보존 소스를 기준으로 했다. 최신 PR을 다시 수집한 검토는 아니다. chainbench 실행 엔진·genesis 보강 자료는 별도 시점에 수집했고 `sources/recheck-chainbench-manifest.json`에 시각·해시를 기록했다. 검토 전 문서는 `review-before-corrections/`에 보존했다.

## 수행 후보 교정

| 구분 | 교정 내용 | 영향 |
|---|---|---|
| 누락 후보 | TC-4-6-02 <br> 실행: effective-gas-price-regular, RT-C-03 [부분 대응] <br> 실행: anzeon-basefee-increase, RT-C-04 [부분 대응] <br> 실행: anzeon-basefee-stable, RT-C-05 [부분 대응] <br> 실행: anzeon-basefee-decrease, RT-C-06 <br> 실행: basefee-minimum, RT-C-07 <br> 실행: basefee-maximum, TC-1-3-03 [부분 대응] <br> 실행: feecap-above-min-accepted, TC-1-3-02 [부분 대응] <br> 실행: feecap-exact-min-accepted, RT-A-2-07 / TX-012 [부분 대응] <br> 실행: gaslimit-exceeded-rejected, RT-G-2-01 [부분 대응] <br> 실행: gas-price-equals-basefee-plus-tip, RT-B-01 <br> 실행: block-period-one-second을 excluded에서 adapt로 복원 | 공통 수수료·gas·블록 시각 검증을 전용 기능으로 일괄 제외한 오류. 원래 Anzeon/WBFT 기대값을 그대로 쓰지는 않는다. |
| 실행 대상 | 기존 이식 후보 40개에 applicableChains 수정 안내 추가 | env.chain만 바꾸면 SKIP될 수 있다. 신규 복원 후보에도 동일 지침 적용. |
| 노드 준비 | RT-A-2-04 <br> 실행: nonce-ordering, TX-010 [부분 대응] <br> 실행: out-of-order-nonces-mine, RT-A-2-09 <br> 실행: replacement-tx, TX-013 [부분 대응] <br> 실행: same-nonce-replacement, RT-D-01 <br> 실행: fee-delegated-transfer, RT-D-03 <br> 실행: fd-sender-sig-invalid-rejected, RT-D-04 <br> 실행: fd-feepayer-sig-invalid-rejected, RT-D-05 <br> 실행: feepayer-insufficient-rejected, RT-D-03 / TX-014 [부분 대응] <br> 실행: fee-delegated-sender-sig-invalid-rejected, RT-D-04 / TX-015 [부분 대응] <br> 실행: fee-delegated-feepayer-sig-invalid-rejected, RT-D-05 / TX-016 [부분 대응] <br> 실행: fee-delegated-unfunded-feepayer-rejected, RT-A-4-07 <br> 실행: ws-subscribe-logs에 EN 정의 안내 | on=en1 참조와 topology 불일치 해소가 필요하다. |
| 우선순위 | RT-G-5-01 <br> 실행: fee-delegate-sign-rpc-present P1→P3, BRIOCHE-02 / RPC-008 [부분 대응] <br> 실행: wemix-brioche-block-reward P1→P2, RT-A-1-01 <br> 실행: chain-id P2→P3 | 서명 RPC 존재·Brioche 보상·chainId의 대응 명세와 정렬. nonce RT-A-2-04 <br> 실행: nonce-ordering, TX-010 [부분 대응] <br> 실행: out-of-order-nonces-mine는 실제 포함 순서 검사가 없어 P1 유지. |
| 서명 입력 | RT-D-03 / TX-014에 R=0 등 결정적 무효 서명 조건 추가 | 임의 서명 변경은 다른 유효 주소로 복구될 수 있어 invalid sender를 보장하지 않는다. |
| nonce 기대값 | RT-G-1-05 / RPC-015에 확정 블록·초기 nonce 명시 | 초기 N에서 실제 포함된 sender 거래 k개에 대해 N+k. pending pool nonce와 구분. |

현재 수행 후보는 **144개 원본 항목**이다. 문서 47행 + JSON 97개이며 중복 제거된 실행 횟수가 아니다. JSON은 direct 5개, adapt 89개, conditional 3개다. 전체 263개 중 제외는 119개다.

| 우선순위 | 이전 | 교정 후 |
|---|---:|---:|
| P0 | 4 | 4 |
| P1 | 82 | 89 |
| P2 | 42 | 44 |
| P3 | 5 | 7 |
| 합계 | 133 | 144 |

## 추가 테스트 교정

| ID | 기존 기대값의 문제 | 교정 |
|---|---|---|
| N-001 | known/ancestor보다 먼저라는 순서를 전체 InsertChain에 확대할 여지 | ValidateBody 직접 호출과 선행 header 검증을 구분 |
| N-002 | 크기로 제외된 tx는 실행 자체가 없어야 한다 | 별도 prefetch 실행·캐시는 허용. 정식 env 반영과 committedTxs 표시를 금지 |
| N-006 | ETH payload 상한과 wire 크기·전송 가능 범위 혼동 | msg.Size를 기준으로 측정. 핸들러 직접 주입과 RLPx 통합 시험을 분리 |
| N-008 | 미포함 tx의 무조건 보존·교체 hash 포함 보장 | pool 용량·TTL·잔액·fee·reset·교체 수락/새 작업 시작 시점을 통제 |
| N-009 | 무효 tx 뒤 정상 tx가 반드시 처리되어야 한다 | size 검사에서 break하면 뒤 tx 미처리가 현 정책. 크기 gate 통과/초과를 분리하고 StateDB와 gasPool rollback을 구분 |
| N-010 | receipt 복사를 깊은 복사로 해석할 여지 | 정상 tx 추가 경로만 검증하고 Logs 내부 참조의 임의 변경 독립성은 요구하지 않음 |
| N-012 | ETH65에도 request ID envelope를 지정 | ETH65 raw list, ETH66/68 request ID wrapper를 각각 인코딩 |
| N-016 | decode 전 차단을 압축 해제·할당 전 방어로 해석할 여지 | ETH packet RLP decode 전으로 한정하고 버퍼 재사용을 허용하는 자원 안정화 기준 적용 |

추가 제안 **16개와 우선순위 P0 7개/P1 8개/P2 1개는 유지**한다. 8개 항목을 교정했다는 의미이며 8개의 구현 버그를 발견했다는 뜻은 아니다. historical 과대블록 존재, 실제 receipt 응답 상한 초과, 최종 블록 크기 위반은 아직 입증되지 않았다. N-004/005는 배포 정책과 복구 절차를 정한 뒤 판단하며 자동 수렴을 가정하지 않는다.

## 기각한 의심과 검토 한계

- direct 5개를 모두 실행 불가로 강등할 근거는 찾지 못했다. 환경 준비가 필요하고 통과 여부는 미확인이다.
- 낮은 feeCap 거래가 항상 queued라는 기대는 틀리다. 로컬 여부와 DropUnderPriced·txpool 하한을 구분해야 한다.
- Fee Delegation 및 서명 RPC가 없다는 의심은 코드로 반증했다.
- RT-A-2-04 (분석 TC-031), TX-010 [부분 대응] (분석 TC-032)의 최종 nonce 값만으로 RT-A-2-04 / TX-010 (분석 DOC-C-008)의 실제 포함 순서까지 검증됐다고 세지 않는다.

코드 실행 없이 확인할 수 있는 분류·입력·기대값을 검토한 결과다. 패치 적용 후 실제 동작과 부작용 여부는 최종 통합 SHA에서 시험해야 한다.

## 교정 문서와 근거

- [우선순위별 수행 후보](02-wemix-priority.md)
- [필수 추가 테스트](additional-tests.md)
- [JSON 후보 독립 검토](recheck-tc-findings.md)
- [문서 명세 독립 검토](recheck-document-findings.md)
- [추가 테스트 독립 검토](recheck-additional-findings.md)
- [구조·개수·해시 검증](recheck-validation.json)

원본 263항목, 선별 144항목, 추가 16항목의 문서·JSON 대응과 정렬을 확인했다. 코드 인용 1,725개, Mermaid 3개 구조 검사, 추적 모델 검증을 통과했다. 원본 소스 4,305개와 보강 파일 9개의 해시가 보존되었다. 모델의 기존 경고(mod-wemix 기능 미할당)는 유지되며 실행 커버리지 검증을 뜻하지 않는다.
