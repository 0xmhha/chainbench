# Confluence 명세 ID 단위 판정 (2026-09-11)

Chainbench 폴더 하위 페이지 29개를 Atlassian MCP로 읽어(`../sources/confluence/`) 명세 페이지 11개에서 찾은 테스트 ID 330개를 하나씩 판정했다. PR196 관리 ID 37개(PR196-TC-001~037)는 JSON 테스트의 별칭이므로 표 끝에 따로 둔다. 판정은 "명세에 적힌 기대 결과 그대로"를 기준으로 했고, 일반 부분만 떼면 세 체인이 되는 경우는 `일반 부분` 열에 적었다. 2nd Change의 T-1~T-5는 StableNet 페이지와 WEMIX4.0 페이지에 같은 번호로 있어 페이지별로 나눴다.

## 페이지별 판정 수

| 페이지 | 세 체인 공통 | 2체인 | 체인 전용 | 실행 테스트 아님 | 전환 |
|---|---:|---:|---:|---:|---:|
| [WEMIX3.0] 테스트 시나리오 | 1 | 3 | 10 | 0 | 0 |
| [PR196] 기존 테스트 전체 144개 우선순위 목록 | 85 | 6 | 2 | 0 | 0 |
| [PR196] 집중 테스트 28개 상세 실행 절차 | 29 | 2 | 6 | 0 | 0 |
| [WEMIX4.0] Test | 31 | 26 | 28 | 0 | 2 |
| [WEMIX 4.0] 테스트 시나리오 | 66 | 40 | 30 | 0 | 2 |
| Regression Test Case with scenario | 51 | 17 | 47 | 0 | 0 |
| Regression Test Case | 51 | 17 | 46 | 0 | 0 |
| 2nd Change Test Cases (StableNet) | 2 | 3 | 0 | 0 | 0 |
| [WEMIX 4.0] 2nd Change Test Cases | 2 | 3 | 0 | 0 | 0 |
| 1st Test Scenarios | 15 | 10 | 57 | 11 | 0 |
| 1st Test Cases (v1.0.0+) | 10 | 9 | 45 | 6 | 0 |

전체(중복 페이지 포함 ID 330개): 세 체인 공통 104(설정 분리 38, 설정 분리+기대값 59, 별도 구현 7), 2체인 63, StableNet 전용 105, WEMIX4.0 전용 28, WEMIX3.0 전용 17, 전환 2, 실행 테스트 아님 11.

## Confluence 원문을 읽고 바뀐 것

- RT-A-2-05b(feeCap < MinBaseFee+MinTip)와 RT-A-1-01-A/B는 표 페이지에 없고 시나리오 페이지에만 있다. 05b와 01-A는 StableNet 전용, 01-B는 세 체인 공통(설정 분리 + genesis 생성 경로)이다.
- RT-A-2-01/02의 기대식 `effectiveGasPrice = baseFee + min(tipCap, feeCap-baseFee)`는 표준식이지만 StableNet 비인가 계정은 tip이 헤더 GasTip으로 바뀐다. 시나리오 페이지 자체가 이 예외를 적어 두었다. 체인별 기대값이 필요하다.
- RT-G-2-01/02(eth_gasPrice = baseFee + tip, maxPriorityFee = tip)는 식이 세 클라이언트 같고 tip 출처만 다르다. JSON 07b/08과 함께 세 체인 공통(설정 분리 + 체인별 기대값)으로 올렸다.
- RT-C-07(baseFee 상한)은 WEMIX3.0 governance maxBaseFee가 있어 2체인(WEMIX3.0+StableNet)으로 바꿨다. WBFT에는 상한이 없다.
- TC-3-1-04(바이너리 교체 전후 서명 호환)와 TC-4-1-03(genesis 불일치 기동 거부)은 PR196 144 목록이 WEMIX3.0 후보로 뽑았고 목적이 공통이라 세 체인 공통(별도 구현)으로 올렸다. 교체 바이너리와 로그 문구는 체인별이다.
- WEMIX3.0 시나리오 14개 중 MINING-01(타임스탬프 단조 증가)만 세 체인 관측이 가능하고, BRIOCHE-01~03은 WEMIX4.0과 2체인, 나머지 10개(etcd, time-it, miner limit, 거버넌스 멤버 연동, admin_wemixInfo, 헤더 필드)는 WEMIX3.0 전용이다.
- 2nd Change T-1(txpool 누적 잔액)은 go-wbft/go-stablenet의 pending 합산 검사를 겨냥한다. go-wemix `core/tx_pool.go`는 단건 검사만 확인돼 2체인으로 뒀다. T-2/T-3(AccessList 대납, 대납 서명 경로)은 세 체인 목적이지만 go-wemix 경로는 실행으로 확인해야 한다.
- PR196 신규 N-008/N-009는 관측이 공통이라 세 체인 후보, N-010~N-015는 PR196 크기 제한값(8 MiB, 10 MiB) 기준이라 WEMIX3.0 전용으로 뒀다.
- PR196 144 목록의 JSON 97개 중 88개가 이번 세 체인 판정과 일치한다. 나머지는 2026-04-13 결과 페이지가 StableNet에서만 실행한 stablenet 전용 5개, 2체인 1개, 전환 1개, 하네스 전용 1개다(아래 표의 PR196 우선순위 열).

## 전체 표

| 명세 ID | 페이지 | 공통 범위 | 분리 방식 | family | 판정 이유 | 일반 부분 | PR196 우선순위 | StableNet 2026-04-13 결과 | 연결된 JSON/문서 |
|---|---|---|---|---|---|---|---|---|---|
| BRIOCHE-01 | [WEMIX3.0] 테스트 시나리오 | 2체인(WEMIX3.0+WEMIX4.0) | 설정 분리 + 체인별 기대값 | reward | wemix_briocheConfig/halvingSchedule/getBriocheBlockReward. 두 클라이언트 모두 Brioche 설정 시 등록 (CD-B-06) | - | - | - | - |
| BRIOCHE-02 | [WEMIX3.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 2체인(WEMIX3.0+WEMIX4.0) | 설정 분리 + 체인별 기대값 | reward | wemix_briocheConfig/halvingSchedule/getBriocheBlockReward. 두 클라이언트 모두 Brioche 설정 시 등록 (CD-B-06) | - | P2 | - | json:go-wemix/rpc/01-wemix-brioche-block-reward.json |
| BRIOCHE-03 | [WEMIX3.0] 테스트 시나리오 | 2체인(WEMIX3.0+WEMIX4.0) | 별도 구현/처리 | reward | 실제 보상 배분. WEMIX3.0 governance 배분 vs WEMIX4.0 beneficiary (CD-B-06) | - | - | - | - |
| ETCD-01 | [WEMIX3.0] 테스트 시나리오 | WEMIX3.0 전용 | 공통 아님 | wemix-etcd | etcd work/token 키 (CD-B-01) | - | - | - | - |
| ETCD-02 | [WEMIX3.0] 테스트 시나리오 | WEMIX3.0 전용 | 공통 아님 | wemix-etcd | etcd work/token 키 (CD-B-01) | - | - | - | - |
| ETCD-03 | [WEMIX3.0] 테스트 시나리오 | WEMIX3.0 전용 | 공통 아님 | wemix-etcd | etcd work/token 키 (CD-B-01) | - | - | - | - |
| ETCD-04 | [WEMIX3.0] 테스트 시나리오 | WEMIX3.0 전용 | 공통 아님 | wemix-etcd | etcd work/token 키 (CD-B-01) | - | - | - | - |
| ETCD-05 | [WEMIX3.0] 테스트 시나리오 | WEMIX3.0 전용 | 공통 아님 | wemix-etcd | etcd work/token 키 (CD-B-01) | - | - | - | - |
| GOV-001 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-01 | [WEMIX3.0] 테스트 시나리오 | WEMIX3.0 전용 | 공통 아님 | wemix-governance | 거버넌스 멤버 변경 시 BP·피어 갱신 | - | - | - | - |
| GOV-002 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-003 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-004 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-005 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-006 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-007 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-008 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-009 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-010 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-011 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-012 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-013 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-014 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-015 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-016 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-017 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-018 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-019 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-020 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-021 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-022 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-023 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| GOV-024 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04) | - | - | - | - |
| MINING-01 | [WEMIX3.0] 테스트 시나리오 | 세 체인 공통 | 설정 분리 | consensus-wbft | 타임스탬프 단조 증가. 관측은 세 체인 공통 | - | - | - | - |
| MINING-02 | [WEMIX3.0] 테스트 시나리오 | WEMIX3.0 전용 | 공통 아님 | wemix-mining | time-it catch-up, miner limit 창은 PoA 규칙 | - | - | - | - |
| MINING-03 | [WEMIX3.0] 테스트 시나리오 | WEMIX3.0 전용 | 공통 아님 | wemix-mining | time-it catch-up, miner limit 창은 PoA 규칙 | - | - | - | - |
| N-001 | [PR196] 기존 테스트 전체 144개 우선순위 목록 | WEMIX3.0 전용 | 공통 아님 | wemix-pr196 | PR196 신규 테스트 중 집중 목록 제외분(N-005 혼합 노드 구성 등). 상세 절차 페이지에 없어 내용 미열람 | - | - | - | - |
| N-008 | [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | nonce-replacement | 미포함 tx 이월·교체. 크기 한도 유도는 WEMIX3.0 PR196 값 | - | - | - | - |
| N-009 | [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | invalid-transaction | 거부 vs 실행 실패 상태. 공통 관측 | - | - | - | - |
| N-010 | [PR196] 집중 테스트 28개 상세 실행 절차 | WEMIX3.0 전용 | 공통 아님 | wemix-pr196 | 블록 크기(8 MiB)·메시지 크기(10 MiB)·생성 종료 조건은 PR196 hotfix 값. 다른 두 클라이언트의 대응 제한값은 확인하지 않음 | - | - | - | - |
| N-011 | [PR196] 집중 테스트 28개 상세 실행 절차 | WEMIX3.0 전용 | 공통 아님 | wemix-pr196 | 블록 크기(8 MiB)·메시지 크기(10 MiB)·생성 종료 조건은 PR196 hotfix 값. 다른 두 클라이언트의 대응 제한값은 확인하지 않음 | - | - | - | - |
| N-012 | [PR196] 집중 테스트 28개 상세 실행 절차 | WEMIX3.0 전용 | 공통 아님 | wemix-pr196 | 블록 크기(8 MiB)·메시지 크기(10 MiB)·생성 종료 조건은 PR196 hotfix 값. 다른 두 클라이언트의 대응 제한값은 확인하지 않음 | - | - | - | - |
| N-013 | [PR196] 집중 테스트 28개 상세 실행 절차 | WEMIX3.0 전용 | 공통 아님 | wemix-pr196 | 블록 크기(8 MiB)·메시지 크기(10 MiB)·생성 종료 조건은 PR196 hotfix 값. 다른 두 클라이언트의 대응 제한값은 확인하지 않음 | - | - | - | - |
| N-014 | [PR196] 집중 테스트 28개 상세 실행 절차 | WEMIX3.0 전용 | 공통 아님 | wemix-pr196 | 블록 크기(8 MiB)·메시지 크기(10 MiB)·생성 종료 조건은 PR196 hotfix 값. 다른 두 클라이언트의 대응 제한값은 확인하지 않음 | - | - | - | - |
| N-015 | [PR196] 집중 테스트 28개 상세 실행 절차 | WEMIX3.0 전용 | 공통 아님 | wemix-pr196 | 블록 크기(8 MiB)·메시지 크기(10 MiB)·생성 종료 조건은 PR196 hotfix 값. 다른 두 클라이언트의 대응 제한값은 확인하지 않음 | - | - | - | - |
| NODE-001 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 전환 전용 | 공통 아님 | transition | go-wemix 데이터로 go-wbft 기동·하드포크 전후 비교. 전환 시나리오 | - | - | - | - |
| NODE-002 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 전환 전용 | 공통 아님 | transition | go-wemix 데이터로 go-wbft 기동·하드포크 전후 비교. 전환 시나리오 | - | - | - | - |
| NODE-003 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | sync-lifecycle | Full/Snap sync (CD-B-08/09) | - | P0 | - | doc:DOC-C-040 |
| NODE-004 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | sync-lifecycle | Full/Snap sync (CD-B-08/09) | - | P1 | - | doc:DOC-C-041 |
| NODE-005 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 별도 구현/처리 | fault-topology | 1/7 장애 후 복구. 허용 장애 수는 family별 (CD-B-01, H-07) | - | P1 | - | doc:DOC-C-042 |
| NODE-006 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | NCP 주소 파싱. StableNet TC-4-3이 대응 개념 | - | - | - | - |
| NODE-007 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | NCP 주소 파싱. StableNet TC-4-3이 대응 개념 | - | - | - | - |
| RPC-001 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | eth_blockNumber | - | P1 | - | doc:DOC-C-021 |
| RPC-01 | [WEMIX3.0] 테스트 시나리오 | WEMIX3.0 전용 | 공통 아님 | wemix-rpc | admin_wemixInfo | - | - | - | - |
| RPC-002 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | rpc-basics | 블록 조회 "WBFTExtra 포함"(명세 그대로) | three-chain: 번호/해시 조회 일치 | P1 | - | doc:DOC-C-028; doc:DOC-C-029 |
| RPC-02 | [WEMIX3.0] 테스트 시나리오 | WEMIX3.0 전용 | 공통 아님 | wemix-rpc | rewards/fees/minerNodeSig 헤더 필드 | - | - | - | - |
| RPC-003 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_* | - | - | - | doc:DOC-D-019 |
| RPC-004 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_* | - | - | - | doc:DOC-D-020 |
| RPC-005 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_* | - | - | - | doc:DOC-D-021 |
| RPC-006 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_* | - | - | - | doc:DOC-D-022 |
| RPC-007 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | receipt | - | P1 | - | doc:DOC-C-031 |
| RPC-008 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 2체인(WEMIX3.0+WEMIX4.0) | 설정 분리 + 체인별 기대값 | reward | wemix_getBriocheBlockReward. Brioche 설정 필요 (CD-B-06) | - | P2 | - | json:go-wemix/rpc/01-wemix-brioche-block-reward.json; doc:DOC-D-001 |
| RPC-009 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | 거버넌스 계약 eth_call | - | - | - | - |
| RPC-010 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_nodeAddress/isValidator | - | - | - | doc:DOC-D-018 |
| RPC-011 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_nodeAddress/isValidator | - | - | - | doc:DOC-D-023 |
| RPC-012 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | eth_getBalance/chainId/getLogs/getTransactionCount | - | P2 | - | doc:DOC-C-022 |
| RPC-013 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 | rpc-basics | eth_getBalance/chainId/getLogs/getTransactionCount | - | P3 | - | doc:DOC-C-025 |
| RPC-014 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 | rpc-basics | eth_getBalance/chainId/getLogs/getTransactionCount | - | P1 | - | json:go-stablenet/regression/ethereum/36-contract-event-emitted.json; doc:DOC-C-024 |
| RPC-015 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | eth_getBalance/chainId/getLogs/getTransactionCount | - | P1 | - | doc:DOC-C-032 |
| RPC-016 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-observation | eth_gasPrice ≥ baseFee. tip 출처 체인별 | - | P2 | - | doc:DOC-C-033 |
| RPC-017 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | fee-observation | eth_feeHistory | - | P2 | - | doc:DOC-C-035 |
| RPC-018 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 | rpc-basics | txpool_*/admin_peers. namespace 노출 | - | P2 | - | doc:DOC-C-036 |
| RPC-019 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 세 체인 공통 | 설정 분리 | rpc-basics | txpool_*/admin_peers. namespace 노출 | - | - | - | - |
| RPC-020 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | logs-subscription | WS 구독 | - | P2 | - | doc:DOC-C-026 |
| RPC-021 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | logs-subscription | WS 구독 | - | P2 | - | doc:DOC-C-027 |
| RPC-022 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | consensus-wbft | epochInfo 존재. Stakers 키는 WEMIX4.0, StableNet은 candidates (CD-B-03) | - | - | - | - |
| RPC-023 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WEMIX4.0 전용 | 공통 아님 | wbft-governance | 언스테이킹 후 검증자 제외 | - | - | - | - |
| RT-A-1-01 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | sync-lifecycle | genesis 초기화·블록 0 해시 동일. WEMIX3.0 genesis는 바이너리 생성, WBFT 두 체인은 템플릿 (CD-B-07) | - | P2 | - | json:go-stablenet/regression/ethereum/30-chain-id.json; doc:DOC-C-039 |
| RT-A-1-02 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | sync-lifecycle | Full/Snap sync. 기본 syncmode가 다르고 WEMIX3.0은 etcd syncCheck 개입 (CD-B-08, CD-B-09) | - | P0 | - | doc:DOC-C-040 |
| RT-A-1-03 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | sync-lifecycle | Full/Snap sync. 기본 syncmode가 다르고 WEMIX3.0은 etcd syncCheck 개입 (CD-B-08, CD-B-09) | - | P1 | - | doc:DOC-C-041 |
| RT-A-1-04 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 별도 구현/처리 | fault-topology | 노드 재시작. 프로세스 제어 필요 (H-07). 기대 "1초 간격"은 WBFT 값 | - | P1 | - | doc:DOC-C-042 |
| RT-A-1-05 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | PN bootnode 경유 피어 연결. bootnode 프로필 | - | P2 | - | json:go-stablenet/regression/api/20-admin-peers-populated.json; doc:DOC-C-043 |
| RT-A-1-06 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 별도 구현/처리 | sync-lifecycle | downloader/fetcher 경로. 경로 증명에 로그·metrics 계측 필요 (H-08). 전제 "Anzeon 활성"은 StableNet 표기 | - | P0 | - | doc:DOC-C-044 |
| RT-A-1-07 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 별도 구현/처리 | sync-lifecycle | downloader/fetcher 경로. 경로 증명에 로그·metrics 계측 필요 (H-08). 전제 "Anzeon 활성"은 StableNet 표기 | - | P0 | - | doc:DOC-C-045 |
| RT-A-2-01 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | transaction-types | type 0x0/0x2. 명세의 effectiveGasPrice 공식은 표준식이나 StableNet 비인가 계정은 헤더 GasTip으로 tip이 바뀐다 (CD-A-06) | - | P1 | - | json:go-stablenet/regression/ethereum/08-legacy-transfer.json; doc:DOC-C-001 |
| RT-A-2-02 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | transaction-types | type 0x0/0x2. 명세의 effectiveGasPrice 공식은 표준식이나 StableNet 비인가 계정은 헤더 GasTip으로 tip이 바뀐다 (CD-A-06) | - | P1 | - | json:go-stablenet/regression/ethereum/09-dynamic-fee-tx.json; doc:DOC-C-002 |
| RT-A-2-03 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 | transaction-types | type 0x1. 세 클라이언트 공통 | - | P1 | - | json:go-stablenet/regression/ethereum/10-access-list-tx.json; doc:DOC-C-003 |
| RT-A-2-04 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 | nonce-replacement | nonce 순서 | - | P0 | - | json:go-stablenet/regression/ethereum/11-nonce-ordering.json; doc:DOC-C-008 |
| RT-A-2-06 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | invalid-transaction | 잔액 부족·gasLimit 초과 거부. sentinel은 같고 감싼 문맥이 다르다 (CD-A-08) | - | P1 | - | json:go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json; json:go-wbft/tx/02-wbft-insufficient-funds-rejected.json; json:go-wemix/tx/02-wemix-insufficient-funds-rejected.json; doc:DOC-C-010 |
| RT-A-2-07 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | invalid-transaction | 잔액 부족·gasLimit 초과 거부. sentinel은 같고 감싼 문맥이 다르다 (CD-A-08) | - | P1 | - | json:go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json; json:go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json; doc:DOC-C-011 |
| RT-A-2-08 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | fee-observation | receipt.effectiveGasPrice 존재 | - | P1 | - | json:go-stablenet/regression/ethereum/16-effective-gas-price.json; doc:DOC-C-012 |
| RT-A-2-09 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | nonce-replacement | replacement. priceBump 프로필 | - | P1 | - | json:go-stablenet/regression/ethereum/17-replacement-tx.json; doc:DOC-C-013 |
| RT-A-2-10 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | EIP-7702. go-wemix 미지원. gate가 Croissant 높이 vs anzeon 설정 존재 (CD-A-04) | - | - | - | json:go-stablenet/regression/ethereum/18-set-code-delegation.json; doc:DOC-D-024 |
| RT-A-3-01 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | evm-contract | 배포/호출/eth_call/estimateGas/revert/OOG. fixture EVM revision(PUSH0) 공통화 필요 (CD-B-05) | - | P1 | - | json:go-stablenet/regression/ethereum/19-contract-roundtrip.json; doc:DOC-C-014 |
| RT-A-3-02 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | evm-contract | 배포/호출/eth_call/estimateGas/revert/OOG. fixture EVM revision(PUSH0) 공통화 필요 (CD-B-05) | - | P1 | - | doc:DOC-C-015 |
| RT-A-3-03 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | evm-contract | 배포/호출/eth_call/estimateGas/revert/OOG. fixture EVM revision(PUSH0) 공통화 필요 (CD-B-05) | - | P2 | - | doc:DOC-C-016 |
| RT-A-3-04 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | evm-contract | 배포/호출/eth_call/estimateGas/revert/OOG. fixture EVM revision(PUSH0) 공통화 필요 (CD-B-05) | - | P2 | - | json:go-stablenet/regression/ethereum/22-estimate-gas.json; doc:DOC-C-017 |
| RT-A-3-05 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | evm-contract | 배포/호출/eth_call/estimateGas/revert/OOG. fixture EVM revision(PUSH0) 공통화 필요 (CD-B-05) | - | P1 | - | json:go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json; doc:DOC-C-018 |
| RT-A-3-06 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | evm-contract | 배포/호출/eth_call/estimateGas/revert/OOG. fixture EVM revision(PUSH0) 공통화 필요 (CD-B-05) | - | P1 | - | json:go-stablenet/regression/ethereum/24-revert-tx-status-zero.json; json:go-wbft/tx/03-wbft-revert-status-zero.json; json:go-wemix/tx/03-wemix-revert-status-zero.json; doc:DOC-C-019 |
| RT-A-3-07 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | evm-contract | 배포/호출/eth_call/estimateGas/revert/OOG. fixture EVM revision(PUSH0) 공통화 필요 (CD-B-05) | - | P1 | - | json:go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json; doc:DOC-C-020 |
| RT-A-4-01 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | 표준 eth_* 조회 | - | P1 | - | doc:DOC-C-021 |
| RT-A-4-02 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | 표준 eth_* 조회 | - | P2 | - | json:go-stablenet/regression/ethereum/27-genesis-balance.json; doc:DOC-C-022 |
| RT-A-4-03 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | 표준 eth_* 조회 | - | P1 | - | json:go-stablenet/regression/ethereum/27b-value-transfer.json; doc:DOC-C-023 |
| RT-A-4-04 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 | rpc-basics | 표준 eth_* 조회 | - | P1 | - | json:go-stablenet/regression/ethereum/29-logs-query-well-formed.json; json:go-stablenet/regression/ethereum/36-contract-event-emitted.json; doc:DOC-C-024 |
| RT-A-4-05 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 | rpc-basics | eth_chainId == genesis chainId. expectedChainId 프로필 (CD-A-01) | - | P3 | - | doc:DOC-C-025 |
| RT-A-4-06 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | logs-subscription | WS 구독. ws endpoint 프로필 | - | P2 | - | json:go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json; doc:DOC-C-026 |
| RT-A-4-07 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | logs-subscription | WS 구독. ws endpoint 프로필 | - | P2 | - | json:go-stablenet/regression/ethereum/32-ws-subscribe-logs.json; doc:DOC-C-027 |
| RT-A-2-05a | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | tip 하한 미달 거부. 하한 출처가 체인별(pool.gasPrice / miner.gasprice / MinTip). 메시지 prefix "gas tip cap"은 go-wbft/go-stablenet 형식 (CD-A-06, CD-A-08) | - | P1 | - | json:go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json |
| RT-A-2-05b | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | StableNet 전용 | 공통 아님 | fee-policy | feeCap < MinBaseFee+MinTip 거부는 Anzeon 전용 검사 (CD-A-06) | - | - | - | - |
| RT-B-01 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | 1초 주기(명세 그대로). blockPeriod 프로필로 두면 세 체인 관측 가능 | three-chain: 블록 주기 profile 값 비교 | P2 | - | json:go-stablenet/regression/wbft/01-block-period-one-second.json; doc:DOC-D-002 |
| RT-B-02 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | WBFTExtra seal/epoch/validators (istanbul_*) | - | - | - | json:go-stablenet/regression/wbft/02-wbft-seals-quorum.json; doc:DOC-D-003 |
| RT-B-03 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | WBFTExtra seal/epoch/validators (istanbul_*) | - | - | - | json:go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json; doc:DOC-D-004 |
| RT-B-04 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | GovValidator(0x1001) 제안·승인. WEMIX4.0 GovStaking과 다름 (CD-B-04) | - | - | - | json:go-stablenet/regression/wbft/04-validator-add-member-executes.json; json:go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json |
| RT-B-05 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | GovValidator(0x1001) 제안·승인. WEMIX4.0 GovStaking과 다름 (CD-B-04) | - | - | - | json:go-stablenet/regression/wbft/05-validator-remove-member-executes.json |
| RT-B-06 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-anzeon-fee | WBFTExtra.GasTip 필드는 StableNet 전용 (CD-B-03) | - | - | - | json:go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json; json:go-stablenet/regression/wbft/14-stablenet-gastip-field.json; doc:DOC-D-005; doc:DOC-D-008 |
| RT-B-07 | Regression Test Case with scenario, Regression Test Case | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | WBFTExtra seal/epoch/validators (istanbul_*) | - | - | - | json:go-stablenet/regression/api/12b-validator-set-count.json |
| RT-B-08 | Regression Test Case with scenario, Regression Test Case | 2체인(WEMIX4.0+StableNet) | 별도 구현/처리 | consensus-wbft | 쿼럼 미달 블록 거부. 조작 블록 주입 도구 필요(하네스에 없음) | - | - | - | doc:DOC-D-012 |
| RT-B-09 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 별도 구현/처리 | consensus-wbft | round change. 제안자 정지에 프로세스 제어 (H-07) | - | - | - | doc:DOC-D-006 |
| RT-B-10 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 별도 구현/처리 | consensus-wbft | round change. 제안자 정지에 프로세스 제어 (H-07) | - | - | - | doc:DOC-D-007 |
| RT-B-11 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | PrevCommittedSeal/PrevPreparedSeal ≥ quorum | - | - | - | json:go-stablenet/regression/wbft/11-prev-seals-quorum.json; doc:DOC-D-009 |
| RT-B-12 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | PrevCommittedSeal/PrevPreparedSeal ≥ quorum | - | - | - | doc:DOC-D-010 |
| RT-C-01 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-anzeon-fee | 비인가/인가 계정 tip 강제는 StableNet 정책 (CD-A-06, CD-A-07) | - | - | - | json:go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json |
| RT-C-02 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-anzeon-fee | 비인가/인가 계정 tip 강제는 StableNet 정책 (CD-A-06, CD-A-07) | - | - | - | json:go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json |
| RT-C-03 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | baseFee 증가/유지/감소. 임계값·target이 세 공식 (CD-A-06) | - | P1 | - | json:go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json |
| RT-C-04 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | baseFee 증가/유지/감소. 임계값·target이 세 공식 (CD-A-06) | - | P1 | - | json:go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json |
| RT-C-05 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | baseFee 증가/유지/감소. 임계값·target이 세 공식 (CD-A-06) | - | P1 | - | json:go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json |
| RT-C-06 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | baseFee 하한. StableNet 상수 / WEMIX3.0 governance clamp 1 / WBFT 0 → 프로필 하한 비교 | - | P1 | - | json:go-stablenet/regression/anzeon/06-basefee-minimum.json |
| RT-C-07 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 2체인(WEMIX3.0+StableNet) | 설정 분리 + 체인별 기대값 | fee-policy | baseFee 상한. WBFT EIP-1559에는 상한이 없다 (CD-A-06) | - | P1 | - | json:go-stablenet/regression/anzeon/07-basefee-maximum.json |
| RT-D-01 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-delegation | type 0x16 대납·서명 변조·잔액 부족. 세 클라이언트 구현. 대납자 잔액 기준(feeCap/gasPrice)과 거부 시점(txpool/실행)이 다르다 (CD-A-03) | - | P1 | - | json:go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json; doc:DOC-C-004 |
| RT-D-03 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-delegation | type 0x16 대납·서명 변조·잔액 부족. 세 클라이언트 구현. 대납자 잔액 기준(feeCap/gasPrice)과 거부 시점(txpool/실행)이 다르다 (CD-A-03) | - | P1 | - | json:go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json; json:go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json; doc:DOC-C-005 |
| RT-D-04 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-delegation | type 0x16 대납·서명 변조·잔액 부족. 세 클라이언트 구현. 대납자 잔액 기준(feeCap/gasPrice)과 거부 시점(txpool/실행)이 다르다 (CD-A-03) | - | P1 | - | json:go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json; json:go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json; doc:DOC-C-006 |
| RT-D-05 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-delegation | type 0x16 대납·서명 변조·잔액 부족. 세 클라이언트 구현. 대납자 잔액 기준(feeCap/gasPrice)과 거부 시점(txpool/실행)이 다르다 (CD-A-03) | - | P1 | - | json:go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json; json:go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json; doc:DOC-C-007 |
| RT-E-01 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-account-policy | blacklist/authorized/zero·precompile 전송 차단 (CD-A-07) | - | - | - | json:go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json |
| RT-E-02 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-account-policy | blacklist/authorized/zero·precompile 전송 차단 (CD-A-07) | - | - | - | json:go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json |
| RT-E-03 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-account-policy | blacklist/authorized/zero·precompile 전송 차단 (CD-A-07) | - | - | - | json:go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json |
| RT-E-04 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-account-policy | blacklist/authorized/zero·precompile 전송 차단 (CD-A-07) | - | - | - | json:go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json |
| RT-E-05 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-account-policy | blacklist/authorized/zero·precompile 전송 차단 (CD-A-07) | - | - | - | json:go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json |
| RT-E-06 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-account-policy | blacklist/authorized/zero·precompile 전송 차단 (CD-A-07) | - | - | - | json:go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json |
| RT-E-07 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-account-policy | blacklist/authorized/zero·precompile 전송 차단 (CD-A-07) | - | - | - | json:go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json |
| RT-E-08 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-account-policy | blacklist/authorized/zero·precompile 전송 차단 (CD-A-07) | - | - | - | json:go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json |
| RT-E-09 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-account-policy | blacklist/authorized/zero·precompile 전송 차단 (CD-A-07) | - | - | - | json:go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json |
| RT-F-1-01 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json; json:go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json |
| RT-F-1-02 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/02-token-balance-readable.json |
| RT-F-1-03 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json |
| RT-F-1-04 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/04-mint-transfer-event.json |
| RT-F-1-05 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/05-burn-transfer-event.json |
| RT-F-2-01 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/06-mint-proposal-executes.json |
| RT-F-2-02 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/07-burn-proposal-executes.json |
| RT-F-2-03 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json |
| RT-F-2-04 | Regression Test Case with scenario | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | - |
| RT-F-3-01 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | - |
| RT-F-3-02 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | - |
| RT-F-3-03 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | - |
| RT-F-3-04 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/09-validator-metadata-readable.json |
| RT-F-3-05 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | - |
| RT-F-3-06 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json |
| RT-F-4-01 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json |
| RT-F-4-02 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/13-remove-minter-executes.json |
| RT-F-4-03 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json |
| RT-F-4-04 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json |
| RT-F-5-01 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json |
| RT-F-5-02 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | - |
| RT-F-5-03 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json |
| RT-F-5-04 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json |
| RT-F-5-05 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json |
| RT-F-5-06 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | - |
| RT-F-5-07 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | - |
| RT-F-5-08 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/23-authorized-account-added-event.json |
| RT-F-5-09 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복) | - | - | - | json:go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json |
| RT-G-1-01 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | eth 블록/tx 조회 | - | P1 | - | json:go-stablenet/regression/api/01-block-transactions-field.json; doc:DOC-C-028 |
| RT-G-1-02 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | eth 블록/tx 조회 | - | P1 | - | json:go-stablenet/regression/api/02-block-by-hash-consistency.json; doc:DOC-C-029 |
| RT-G-1-03 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | eth 블록/tx 조회 | - | P1 | - | json:go-stablenet/regression/api/03-transaction-by-hash-fields.json; doc:DOC-C-030 |
| RT-G-1-04 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | eth 블록/tx 조회 | - | P1 | - | json:go-stablenet/regression/api/04-transaction-receipt-fields.json; doc:DOC-C-031 |
| RT-G-1-05 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | rpc-basics | eth 블록/tx 조회 | - | P1 | - | json:go-stablenet/regression/api/05-transaction-count-increments.json; doc:DOC-C-032 |
| RT-G-1-06 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | StableNet 전용 | 공통 아님 | stablenet-system-contract | 0x1000 코드 조회. 시스템 계약 주소는 체인별 | three-chain: 배포한 일반 계약의 eth_getCode | - | - | json:go-stablenet/regression/api/06-system-contracts-deployed.json |
| RT-G-2-01 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-observation | eth_gasPrice = tip + baseFee, eth_maxPriorityFeePerGas = tip. 식은 같고 tip 출처만 다르다 (CD-A-06) | - | P2 | - | json:go-stablenet/regression/api/07-gas-price-positive.json; json:go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json; doc:DOC-C-033 |
| RT-G-2-02 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-observation | eth_gasPrice = tip + baseFee, eth_maxPriorityFeePerGas = tip. 식은 같고 tip 출처만 다르다 (CD-A-06) | - | P2 | - | json:go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json; doc:DOC-C-034 |
| RT-G-2-03 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-observation | feeHistory 형태는 공통. "MinBaseFee 이상" 조건은 StableNet | - | P2 | - | json:go-stablenet/regression/api/09-fee-history-well-formed.json; doc:DOC-C-035 |
| RT-G-2-04 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 별도 구현/처리 | stablenet-system-contract | NativeCoinAdapter.transfer estimateGas. 일반 계약 fixture면 three-chain(RT-A-3-04) | three-chain: 일반 계약 estimateGas | - | - | json:go-stablenet/regression/api/10-estimate-gas-token-transfer.json |
| RT-G-3-01 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_* (CD-B-02) | - | - | - | json:go-stablenet/regression/api/11-node-address-returned.json; doc:DOC-D-018 |
| RT-G-3-02 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_* (CD-B-02) | - | - | - | json:go-stablenet/regression/api/12-validator-set-nonempty.json; doc:DOC-D-019 |
| RT-G-3-03 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_* (CD-B-02) | - | - | - | json:go-stablenet/regression/api/13-commit-signers-quorum.json; doc:DOC-D-020 |
| RT-G-3-04 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_* (CD-B-02) | - | - | - | json:go-stablenet/regression/api/14-wbft-extra-info-fields.json; doc:DOC-D-021 |
| RT-G-3-05 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_* (CD-B-02) | - | - | - | json:go-stablenet/regression/api/15-istanbul-status-fields.json; doc:DOC-D-022 |
| RT-G-3-06 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_* (CD-B-02) | - | - | - | json:go-stablenet/regression/api/16-is-validator-flags.json; doc:DOC-D-023 |
| RT-G-4-01 | Regression Test Case with scenario, Regression Test Case | 세 체인 공통 | 설정 분리 | rpc-basics | net/txpool/admin 조회. namespace 노출 프로필 | - | - | - | - |
| RT-G-4-02 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 | rpc-basics | net/txpool/admin 조회. namespace 노출 프로필 | - | P1 | - | json:go-stablenet/regression/api/18-txpool-status.json; doc:DOC-C-036 |
| RT-G-4-03 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 | rpc-basics | net/txpool/admin 조회. namespace 노출 프로필 | - | P1 | - | json:go-stablenet/regression/api/19-txpool-content-well-formed.json; doc:DOC-C-037 |
| RT-G-4-04 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 세 체인 공통 | 설정 분리 | rpc-basics | net/txpool/admin 조회. namespace 노출 프로필 | - | - | - | - |
| RT-G-5-01 | Regression Test Case with scenario, Regression Test Case, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | fee-delegation | eth_signRawFeeDelegateTransaction 세 클라이언트 구현 (CD-B-02) | - | P3 | - | json:go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json; doc:DOC-C-038 |
| RT-G-5-02 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter totalSupply/allowance | - | - | - | json:go-stablenet/regression/api/22-token-total-supply-readable.json |
| RT-G-5-03 | Regression Test Case with scenario, Regression Test Case | StableNet 전용 | 공통 아님 | stablenet-system-contract | NativeCoinAdapter totalSupply/allowance | - | - | - | json:go-stablenet/regression/api/23-token-approve-sets-allowance.json |
| T-1@stablenet | 2nd Change Test Cases (StableNet) | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | txpool | txpool 누적 잔액 검사. go-wbft/go-stablenet은 pending 합산 검사, go-wemix core/tx_pool.go는 단건 검사만 확인됨 (CD-A-03) | - | - | - | - |
| T-2@stablenet | 2nd Change Test Cases (StableNet) | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-delegation | AccessList 포함 0x16. go-wemix 경로는 실행 확인 필요 | - | - | - | - |
| T-3@stablenet | 2nd Change Test Cases (StableNet) | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-delegation | eth_sendTransaction/eth_signTransaction 0x16 서명. keystore 지원은 클라이언트별 확인 필요 | - | - | - | - |
| T-4@stablenet | 2nd Change Test Cases (StableNet) | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | vanityData raw hex | - | - | - | - |
| T-5@stablenet | 2nd Change Test Cases (StableNet) | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_status 안정성 | - | - | - | - |
| T-1@wemix4 | [WEMIX 4.0] 2nd Change Test Cases | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | txpool | txpool 누적 잔액 검사. go-wbft/go-stablenet은 pending 합산 검사, go-wemix core/tx_pool.go는 단건 검사만 확인됨 (CD-A-03) | - | - | - | - |
| T-2@wemix4 | [WEMIX 4.0] 2nd Change Test Cases | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-delegation | AccessList 포함 0x16. go-wemix 경로는 실행 확인 필요 | - | - | - | - |
| T-3@wemix4 | [WEMIX 4.0] 2nd Change Test Cases | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-delegation | eth_sendTransaction/eth_signTransaction 0x16 서명. keystore 지원은 클라이언트별 확인 필요 | - | - | - | - |
| T-4@wemix4 | [WEMIX 4.0] 2nd Change Test Cases | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | vanityData raw hex | - | - | - | - |
| T-5@wemix4 | [WEMIX 4.0] 2nd Change Test Cases | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-rpc-istanbul | istanbul_status 안정성 | - | - | - | - |
| TC-1-1-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovMinter v2 burn refund / Boho 업그레이드 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json |
| TC-1-1-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovMinter v2 burn refund / Boho 업그레이드 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json |
| TC-1-1-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovMinter v2 burn refund / Boho 업그레이드 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json |
| TC-1-1-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovMinter v2 burn refund / Boho 업그레이드 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json |
| TC-1-1-05 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovMinter v2 burn refund / Boho 업그레이드 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json |
| TC-1-1-06 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovMinter v2 burn refund / Boho 업그레이드 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json |
| TC-1-1-07 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovMinter v2 burn refund / Boho 업그레이드 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json |
| TC-1-1-08 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovMinter v2 burn refund / Boho 업그레이드 | - | - | - | - |
| TC-1-1-09 | 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovMinter v2 burn refund / Boho 업그레이드 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json; json:go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json |
| TC-1-1-10 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovMinter v2 burn refund / Boho 업그레이드 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json; json:go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json |
| TC-1-1-11 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovMinter v2 burn refund / Boho 업그레이드 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json |
| TC-1-1-12 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovMinter v2 burn refund / Boho 업그레이드 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json |
| TC-1-2-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | P256(0x100). go-wemix 없음. WBFT Croissant / StableNet Boho gate (CD-A-05). TC-1-2-02(Boho 이전 미존재)는 StableNet gate 전용 | - | - | Pass | doc:DOC-D-027; doc:DOC-D-028; doc:DOC-D-029 |
| TC-1-2-02 | 1st Test Cases (v1.0.0+) | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | P256(0x100). go-wemix 없음. WBFT Croissant / StableNet Boho gate (CD-A-05). TC-1-2-02(Boho 이전 미존재)는 StableNet gate 전용 | - | - | Pass | doc:DOC-D-027; doc:DOC-D-028; doc:DOC-D-029 |
| TC-1-2-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | P256(0x100). go-wemix 없음. WBFT Croissant / StableNet Boho gate (CD-A-05). TC-1-2-02(Boho 이전 미존재)는 StableNet gate 전용 | - | - | Pass | doc:DOC-D-027; doc:DOC-D-028; doc:DOC-D-029 |
| TC-1-2-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | P256(0x100). go-wemix 없음. WBFT Croissant / StableNet Boho gate (CD-A-05). TC-1-2-02(Boho 이전 미존재)는 StableNet gate 전용 | - | - | Pass | doc:DOC-D-027; doc:DOC-D-028; doc:DOC-D-029 |
| TC-1-2-05 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | P256(0x100). go-wemix 없음. WBFT Croissant / StableNet Boho gate (CD-A-05). TC-1-2-02(Boho 이전 미존재)는 StableNet gate 전용 | - | - | Pass | doc:DOC-D-027; doc:DOC-D-028; doc:DOC-D-029 |
| TC-1-2-06 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | P256(0x100). go-wemix 없음. WBFT Croissant / StableNet Boho gate (CD-A-05). TC-1-2-02(Boho 이전 미존재)는 StableNet gate 전용 | - | - | Pass | doc:DOC-D-027; doc:DOC-D-028; doc:DOC-D-029 |
| TC-1-3-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | 최소 가스비 하한. 기준값이 체인별 (CD-A-06) | - | - | Pass | - |
| TC-1-3-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+), [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | 최소 가스비 하한. 기준값이 체인별 (CD-A-06) | - | P1 | Pass | json:go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json |
| TC-1-3-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+), [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | 최소 가스비 하한. 기준값이 체인별 (CD-A-06) | - | P1 | Pass | json:go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json |
| TC-1-3-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+), [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | 최소 가스비 하한. 기준값이 체인별 (CD-A-06) | - | P1 | Pass | json:go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json |
| TC-1-3-05 | 1st Test Scenarios, 1st Test Cases (v1.0.0+), [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | 최소 가스비 하한. 기준값이 체인별 (CD-A-06) | - | P1 | Pass | json:go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json |
| TC-1-3-06 | 1st Test Scenarios, 1st Test Cases (v1.0.0+), [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | 최소 가스비 하한. 기준값이 체인별 (CD-A-06) | - | P1 | Pass | json:go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json |
| TC-3-1-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 실행 테스트 아님(단위/빌드) | 공통 아님 | unit-benchmark | Go 벤치마크. 노드 실행 테스트가 아님(명세도 코드 기반 불가로 표시) | - | - | - | - |
| TC-3-1-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 실행 테스트 아님(단위/빌드) | 공통 아님 | unit-benchmark | Go 벤치마크. 노드 실행 테스트가 아님(명세도 코드 기반 불가로 표시) | - | - | - | - |
| TC-3-1-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 실행 테스트 아님(단위/빌드) | 공통 아님 | unit-benchmark | Go 벤치마크. 노드 실행 테스트가 아님(명세도 코드 기반 불가로 표시) | - | - | - | - |
| TC-3-1-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+), [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 별도 구현/처리 | sync-lifecycle | 바이너리 교체 전후 서명 호환. swapNode 바이너리·로그 문구 체인별 | - | P1 | Pass | json:go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json |
| TC-4-1-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | Anzeon 설정으로 WBFT 엔진 초기화·Boho 반영 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json |
| TC-4-1-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | Anzeon 설정으로 WBFT 엔진 초기화·Boho 반영 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json |
| TC-4-1-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+), [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 별도 구현/처리 | sync-lifecycle | 저장 genesis와 불일치 시 기동 거부. 로그 문구 체인별 | - | P1 | Pass | json:go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json |
| TC-4-2-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | 7702 authorizationList estimateGas (CD-A-04) | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json; doc:DOC-D-025 |
| TC-4-2-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+), [PR196] 기존 테스트 전체 144개 우선순위 목록 | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | 7702 authorizationList estimateGas (CD-A-04) | - | P2 | Pass | doc:DOC-D-026 |
| TC-4-2-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | 7702 authorizationList estimateGas (CD-A-04) | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json; doc:DOC-D-025 |
| TC-4-3-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovCouncil authorized 주소 문자열 파싱 (WEMIX4.0의 NODE-006/007이 대응 개념) | - | - | Pass | json:go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json |
| TC-4-3-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovCouncil authorized 주소 문자열 파싱 (WEMIX4.0의 NODE-006/007이 대응 개념) | - | - | Pass | json:go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json |
| TC-4-3-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovCouncil authorized 주소 문자열 파싱 (WEMIX4.0의 NODE-006/007이 대응 개념) | - | - | Pass | json:go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json |
| TC-4-3-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovCouncil authorized 주소 문자열 파싱 (WEMIX4.0의 NODE-006/007이 대응 개념) | - | - | Pass | json:go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json |
| TC-4-3-05 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovCouncil authorized 주소 문자열 파싱 (WEMIX4.0의 NODE-006/007이 대응 개념) | - | - | Pass | json:go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json |
| TC-4-3-06 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | GovCouncil authorized 주소 문자열 파싱 (WEMIX4.0의 NODE-006/007이 대응 개념) | - | - | Pass | json:go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json |
| TC-4-4-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | Anzeon+Boho 동일 블록 적용 | - | - | - | - |
| TC-4-4-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | Anzeon+Boho 동일 블록 적용 | - | - | - | - |
| TC-4-4-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | Anzeon+Boho 동일 블록 적용 | - | - | - | - |
| TC-4-4-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | Anzeon+Boho 동일 블록 적용 | - | - | Pass | - |
| TC-4-5-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | alloc.Extra·GovCouncil 동기화 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json; json:go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json |
| TC-4-5-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | alloc.Extra·GovCouncil 동기화 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json; json:go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json; json:go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json |
| TC-4-5-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | alloc.Extra·GovCouncil 동기화 | - | - | Pass | - |
| TC-4-5-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | alloc.Extra·GovCouncil 동기화 | - | - | Pass | - |
| TC-4-5-05 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | alloc.Extra·GovCouncil 동기화 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json |
| TC-4-5-06 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | alloc.Extra·GovCouncil 동기화 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json |
| TC-4-5-07 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | alloc.Extra·GovCouncil 동기화 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json; json:go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json |
| TC-4-5-08 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | alloc.Extra·GovCouncil 동기화 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json |
| TC-4-5-09 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | alloc.Extra·GovCouncil 동기화 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json |
| TC-4-5-10 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | alloc.Extra·GovCouncil 동기화 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json |
| TC-4-5-11 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | alloc.Extra·GovCouncil 동기화 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json |
| TC-4-5-12 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | alloc.Extra·GovCouncil 동기화 | - | - | Pass | - |
| TC-4-6-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-anzeon-fee | 인증 계정 egp 재계산·AuthorizedTxExecuted 로그 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json |
| TC-4-6-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+), [PR196] 기존 테스트 전체 144개 우선순위 목록 | StableNet 전용 | 공통 아님 | stablenet-anzeon-fee | headerGasTip으로 egp 재계산(명세 그대로) | three-chain: BP/EN receipt effectiveGasPrice 동일 (JSON 02b) | P1 | Pass | json:go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json; json:go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json |
| TC-4-6-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-observation | egp가 있으면 덮어쓰지 않음. snap sync 노드 필요 | - | - | - | - |
| TC-4-6-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-anzeon-fee | 인증 계정 egp 재계산·AuthorizedTxExecuted 로그 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json |
| TC-5-1-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 실행 테스트 아님(단위/빌드) | 공통 아님 | build | 빌드·CI·runtime.Version 확인 | - | - | Pass | - |
| TC-5-1-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 실행 테스트 아님(단위/빌드) | 공통 아님 | build | 빌드·CI·runtime.Version 확인 | - | - | - | - |
| TC-5-1-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 실행 테스트 아님(단위/빌드) | 공통 아님 | build | 빌드·CI·runtime.Version 확인 | - | - | - | - |
| TC-5-2-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | CollectUpgrades/시스템 계약 버전 레지스트리 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json |
| TC-5-2-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | CollectUpgrades/시스템 계약 버전 레지스트리 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json |
| TC-5-2-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | CollectUpgrades/시스템 계약 버전 레지스트리 | - | - | Pass | - |
| TC-5-2-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | CollectUpgrades/시스템 계약 버전 레지스트리 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json |
| TC-5-2-05 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | CollectUpgrades/시스템 계약 버전 레지스트리 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json |
| TC-5-2-06 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 전용 | 공통 아님 | stablenet-post-v1 | CollectUpgrades/시스템 계약 버전 레지스트리 | - | - | Pass | json:go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json |
| TC-5-3-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+), [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 | sync-lifecycle | genesis 블록 해시 노드 간 일치 | - | P1 | Pass | json:go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json |
| TS-1-2 | 1st Test Scenarios | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | TC-1-2 시나리오 | - | - | - | - |
| TS-2-1 | 1st Test Scenarios | 실행 테스트 아님(단위/빌드) | 공통 아님 | unit-test | 취약점 패치 단위 테스트(명세가 통합 테스트 제외) | - | - | - | - |
| TS-2-2 | 1st Test Scenarios | 실행 테스트 아님(단위/빌드) | 공통 아님 | unit-test | 취약점 패치 단위 테스트(명세가 통합 테스트 제외) | - | - | - | - |
| TS-2-3 | 1st Test Scenarios | 실행 테스트 아님(단위/빌드) | 공통 아님 | unit-test | 취약점 패치 단위 테스트(명세가 통합 테스트 제외) | - | - | - | - |
| TS-2-4 | 1st Test Scenarios | 실행 테스트 아님(단위/빌드) | 공통 아님 | unit-test | 취약점 패치 단위 테스트(명세가 통합 테스트 제외) | - | - | - | - |
| TS-4-2 | 1st Test Scenarios | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | TC-4-2 시나리오 | - | - | - | - |
| TS-4-4 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TC-4-4 시나리오 | - | - | - | - |
| TS-4-6 | 1st Test Scenarios | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-observation | TC-4-6 시나리오(snap sync egp). 인증 계정 부분은 StableNet | - | - | - | - |
| TS-1-1-01 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TS는 TC-1-1 시나리오 | - | - | - | - |
| TS-1-1-02 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TS는 TC-1-1 시나리오 | - | - | - | - |
| TS-1-1-03 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TS는 TC-1-1 시나리오 | - | - | - | - |
| TS-1-1-04 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TS는 TC-1-1 시나리오 | - | - | - | - |
| TS-1-3-01 | 1st Test Scenarios | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | TC-1-3 시나리오 | - | - | - | - |
| TS-1-3-02 | 1st Test Scenarios | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | TC-1-3 시나리오 | - | - | - | - |
| TS-4-1-01 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TC-4-1-01/02 시나리오 | - | - | - | - |
| TS-4-1-02 | 1st Test Scenarios | 세 체인 공통 | 별도 구현/처리 | sync-lifecycle | TC-4-1-03 시나리오 | - | - | - | - |
| TS-4-3-01 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TC-4-3 시나리오 | - | - | - | - |
| TS-4-5-01 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TC-4-5 시나리오 | - | - | - | - |
| TS-4-5-02 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TC-4-5 시나리오 | - | - | - | - |
| TS-4-5-03 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TC-4-5 시나리오 | - | - | - | - |
| TS-5-1-01 | 1st Test Scenarios | 실행 테스트 아님(단위/빌드) | 공통 아님 | build | TC-5-1 시나리오 | - | - | - | - |
| TS-5-2-01 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TC-5-2 시나리오 | - | - | - | - |
| TS-5-2-02 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TC-5-2 시나리오 | - | - | - | - |
| TS-5-2-03 | 1st Test Scenarios | StableNet 전용 | 공통 아님 | stablenet-post-v1 | TC-5-2 시나리오 | - | - | - | - |
| TS-5-3-01 | 1st Test Scenarios | 세 체인 공통 | 설정 분리 | sync-lifecycle | TC-5-3 시나리오 | - | - | - | - |
| TX-001 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 세 체인 공통 | 설정 분리 | tx-basics | 일반 송금 | - | - | - | - |
| TX-002 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-policy | baseFee 미달 거부. 기준이 체인별 (CD-A-06) | - | - | - | - |
| TX-003 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | transaction-types | 0x2/0x0. effectiveGasPrice 식 (CD-A-06) | - | P1 | - | doc:DOC-C-002 |
| TX-004 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-delegation | 0x16 (CD-A-03) | - | P1 | - | doc:DOC-C-004 |
| TX-005 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | evm-contract | 배포·상태 변경·view·revert·OOG. fixture revision (CD-B-05) | - | P1 | - | doc:DOC-C-014 |
| TX-006 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | transaction-types | 0x2/0x0. effectiveGasPrice 식 (CD-A-06) | - | P1 | - | doc:DOC-C-001 |
| TX-007 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 | transaction-types | 0x1 | - | P1 | - | doc:DOC-C-003 |
| TX-008 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | 7702 (CD-A-04) | - | - | - | doc:DOC-D-024 |
| TX-009 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | P256 (CD-A-05) | - | - | - | json:go-wbft/accounts/01-secp256r1-precompile-valid.json; doc:DOC-D-027 |
| TX-010 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록, [PR196] 집중 테스트 28개 상세 실행 절차 | 세 체인 공통 | 설정 분리 | nonce-replacement | nonce 순서 | - | P0 | - | json:go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json; doc:DOC-C-008 |
| TX-011 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | invalid-transaction | 거부 sentinel (CD-A-08) | - | P1 | - | json:go-wbft/tx/02-wbft-insufficient-funds-rejected.json; json:go-wemix/tx/02-wemix-insufficient-funds-rejected.json; doc:DOC-C-010 |
| TX-012 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | invalid-transaction | 거부 sentinel (CD-A-08) | - | P1 | - | json:go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json; doc:DOC-C-011 |
| TX-013 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | nonce-replacement | queued 교체. priceBump | - | P1 | - | json:go-stablenet/regression/ethereum/17b-same-nonce-replacement.json; doc:DOC-C-013 |
| TX-014 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-delegation | 대납 서명 변조·잔액 부족 (CD-A-03) | - | P1 | - | json:go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json; doc:DOC-C-005 |
| TX-015 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-delegation | 대납 서명 변조·잔액 부족 (CD-A-03) | - | P1 | - | json:go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json; doc:DOC-C-006 |
| TX-016 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | fee-delegation | 대납 서명 변조·잔액 부족 (CD-A-03) | - | P1 | - | json:go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json; doc:DOC-C-007 |
| TX-017 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | evm-contract | revert/OOG. fixture revision | - | P1 | - | json:go-wbft/tx/03-wbft-revert-status-zero.json; json:go-wemix/tx/03-wemix-revert-status-zero.json; doc:DOC-C-019 |
| TX-018 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오, [PR196] 기존 테스트 전체 144개 우선순위 목록 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | evm-contract | revert/OOG. fixture revision | - | P1 | - | doc:DOC-C-020 |
| TX-019 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | P256 (CD-A-05) | - | - | - | json:go-wbft/accounts/02-secp256r1-precompile-invalid.json; doc:DOC-D-028 |
| TX-020 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 + 체인별 기대값 | modern-evm | P256 (CD-A-05) | - | - | - | json:go-wbft/accounts/03-secp256r1-precompile-short-input.json; doc:DOC-D-029 |
| WBFT-001 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | Finalize/주기/epoch/prevSeal | - | - | - | doc:DOC-D-003 |
| WBFT-002 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | Finalize/주기/epoch/prevSeal | - | - | - | doc:DOC-D-002 |
| WBFT-003 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 별도 구현/처리 | consensus-wbft | view change·proposer 순환·장애 수. 프로세스 제어 (H-07) | - | - | - | doc:DOC-D-006; doc:DOC-D-007 |
| WBFT-004 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | WBFT-003에 통합 | - | - | - | - |
| WBFT-005 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | Finalize/주기/epoch/prevSeal | - | - | - | doc:DOC-D-004 |
| WBFT-006 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 별도 구현/처리 | consensus-wbft | view change·proposer 순환·장애 수. 프로세스 제어 (H-07) | - | - | - | doc:DOC-D-008 |
| WBFT-007 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 별도 구현/처리 | consensus-wbft | view change·proposer 순환·장애 수. 프로세스 제어 (H-07) | - | - | - | doc:DOC-D-013 |
| WBFT-008 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 별도 구현/처리 | consensus-wbft | view change·proposer 순환·장애 수. 프로세스 제어 (H-07) | - | - | - | doc:DOC-D-014 |
| WBFT-009 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | Finalize/주기/epoch/prevSeal | - | - | - | doc:DOC-D-009; doc:DOC-D-010 |
| WBFT-010 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 설정 분리 | consensus-wbft | RandaoReveal/MixDigest | - | - | - | json:go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json; doc:DOC-D-011 |
| WBFT-011 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 별도 구현/처리 | consensus-wbft | quorum 계산 장애 시험. 프로세스 제어 | - | - | - | doc:DOC-D-015 |
| WBFT-012 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 별도 구현/처리 | consensus-wbft | quorum 계산 장애 시험. 프로세스 제어 | - | - | - | doc:DOC-D-016 |
| WBFT-013 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 2체인(WEMIX4.0+StableNet) | 별도 구현/처리 | consensus-wbft | quorum 계산 장애 시험. 프로세스 제어 | - | - | - | doc:DOC-D-017 |

## PR196 관리 ID (JSON 테스트 별칭)

| 관리 ID | 실행 ID | 공통 범위 | 분리 방식 | PR196 우선순위 |
|---|---|---|---|---|
| PR196-TC-001 | wemix-chain-up | 세 체인 공통 | 설정 분리 | P1 |
| PR196-TC-002 | wemix-chain-up-15 | 세 체인 공통 | 설정 분리 | P1 |
| PR196-TC-003 | wemix-node-crash | 세 체인 공통 | 별도 구현/처리 | P1 |
| PR196-TC-004 | wemix-tx-and-contract | 세 체인 공통 | 설정 분리 | P1 |
| PR196-TC-005 | basic-consensus | 세 체인 공통 | 설정 분리 + 체인별 기대값 | P1 |
| PR196-TC-006 | basic-peers | 세 체인 공통 | 설정 분리 | P1 |
| PR196-TC-007 | basic-rpc-health | 세 체인 공통 | 설정 분리 | P1 |
| PR196-TC-008 | basic-sync | 세 체인 공통 | 설정 분리 | P1 |
| PR196-TC-009 | basic-tx-send | 세 체인 공통 | 설정 분리 | P1 |
| PR196-TC-010 | basic-txpool-propagation | 세 체인 공통 | 설정 분리 | P1 |
| PR196-TC-011 | fault-network-partition | 세 체인 공통 | 별도 구현/처리 | P1 |
| PR196-TC-012 | fault-node-crash | 세 체인 공통 | 별도 구현/처리 | P1 |
| PR196-TC-013 | fault-node-recover | 세 체인 공통 | 별도 구현/처리 | P1 |
| PR196-TC-014 | fault-p2p-topology | 세 체인 공통 | 별도 구현/처리 | P1 |
| PR196-TC-015 | fault-two-down | 세 체인 공통 | 별도 구현/처리 | P1 |
| PR196-TC-016 | fault-txpool-leader-change | 세 체인 공통 | 별도 구현/처리 | P1 |
| PR196-TC-017 | stress-block-time | 세 체인 공통 | 설정 분리 + 체인별 기대값 | P1 |
| PR196-TC-018 | stress-tx-flood | 세 체인 공통 | 별도 구현/처리 | P1 |
| PR196-TC-019 | chain-not-syncing | 세 체인 공통 | 설정 분리 | P2 |
| PR196-TC-020 | stablenet-chain-up | 세 체인 공통 | 설정 분리 + 체인별 기대값 | P2 |
| PR196-TC-021 | stablenet-chain-up-15 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | P2 |
| PR196-TC-022 | stablenet-negative-tx-revert | 세 체인 공통 | 설정 분리 | P2 |
| PR196-TC-023 | wbft-chain-up | 세 체인 공통 | 설정 분리 + 체인별 기대값 | P2 |
| PR196-TC-024 | wbft-chain-up-15 | 세 체인 공통 | 설정 분리 + 체인별 기대값 | P2 |
| PR196-TC-025 | e1-mixed-producers | 세 체인 공통 | 별도 구현/처리 | P2 |
| PR196-TC-026 | wbft-node-crash | 세 체인 공통 | 별도 구현/처리 | P2 |
| PR196-TC-027 | wbft-tx-and-contract | 세 체인 공통 | 설정 분리 | P2 |
| PR196-TC-028 | remote-rpc-health | 세 체인 공통 | 설정 분리 | P2 |
| PR196-TC-029 | remote-chain-info | 세 체인 공통 | 설정 분리 | P2 |
| PR196-TC-030 | remote-balance-check | 세 체인 공통 | 설정 분리 | P2 |
| PR196-TC-031 | sample-minimal-value-transfer | 세 체인 공통 | 설정 분리 | P2 |
| PR196-TC-032 | sample-lifecycle-node-restart | 세 체인 공통 | 별도 구현/처리 | P2 |
| PR196-TC-033 | stablenet-proxied-pn-routing | 세 체인 공통 | 설정 분리 | P2 |
| PR196-TC-034 | wemix-wbft-handoff | 전환 전용 | 공통 아님 | P2 |
| PR196-TC-035 | stablenet-derived-vocabulary | 하네스 전용 | 설정 분리 | P3 |
| PR196-TC-036 | stablenet-register-contract | 세 체인 공통 | 설정 분리 | P3 |
| PR196-TC-037 | stablenet-faucet-funds | 세 체인 공통 | 설정 분리 | P3 |
