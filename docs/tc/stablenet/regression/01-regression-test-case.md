# Regression Test Case

> 출처: Confluence [Regression Test Case](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2599813145) (페이지 ID 2599813145, 버전 7, 최종 수정 2026-04-24)  
> 상위 페이지: Regression Test  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김)

---

StableNet 핵심 기능에 대한 회귀 테스트 케이스 목록.

> **테스트 환경**: 폐쇄망 환경 (AnzeonBlock=0, BohoBlock=0, BP=7,EN=7,PN=1 환경)

| 범주 | TC 수 |
| --- | --- |
| 이더리움 기본 기능 | 25 |
| WBFT 합의 엔진 | 12 |
| Anzeon 가스 정책 | 7 |
| Fee Delegation | 4 |
| 블랙리스트 / 권한 계정 | 8 |
| 시스템 컨트랙트 & 거버넌스 (NativeCoinAdapter, GovMinter, GovValidator, GovMasterMinter, GovCouncil) | 27 |
| API 호출 (eth 조회, 가스/수수료, WBFT 커스텀, 관리/진단, StableNet 고유) | 21 |
| **합계** | **104** |

---

### A. 이더리움 기본 기능

#### A-1. 노드 실행 및 동기화

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RT-A-1-01 | 제네시스 블록으로 노드 초기화 | 유효한 genesis.json | 블록 0 해시가 모든 노드에서 동일 |
| RT-A-1-02 | Full Sync — 신규 노드가 기존 체인을 동기화 | `--syncmode full`(기본값), 동기화 제공 노드 블록 높이가 동기화 요청 노드(신규)보다 2 이상 높은 상태 | 동기화 요청 노드의 헤드 블록 번호·해시가 동기화 제공 노드와 일치 |
| RT-A-1-03 | Snap Sync — 신규 노드가 스냅으로 동기화 | `--syncmode snap`, 동기화 제공 노드에 128블록 이상 존재하고 블록 높이가 동기화 요청 노드(신규)보다 2 이상 높은 상태 | 동기화 요청 노드가 스냅 피벗 이후 state 동기화 완료, `eth_getBalance` 조회 가능 |
| RT-A-1-04 | 노드 재시작 후 블록 생산 재개 | 정상 운영 중인 노드를 종료 후 재시작 | 재시작 후 블록 간격(1초) 유지되어 블록 생산 재개 |
| RT-A-1-05 | P2P 피어 연결 — PN 부트노드를 통한 노드 간 연결 | PN이 `--bootnodes`로 설정된 상태, BP/EN 노드 기동 | 각 노드에서 `net_peerCount` ≥ 1, `admin_peers`에 상대 노드가 표시됨 |
| RT-A-1-06 | Downloader 경로 — 동작중인 노드가 블록 높이 차이로 인한 따라잡기 | Anzeon 활성화, 동기화 요청 노드 블록 높이가 동기화 제공 노드보다 2 이상 낮은 상태 (또는 `tdSyncInterval` 경과 후 높이 차이 1인 상태 지속) | 동기화 요청 노드의 Downloader가 동기화 제공 노드로부터 누락 블록 헤더·바디를 순서대로 요청하여 삽입, `eth_blockNumber`가 일치 |
| RT-A-1-07 | Block Fetcher 경로 — 동작중인 노드가 블록 전파로 블록 수신 | 동기화 요청 노드 정상 운영 중, 동기화 제공 노드가 신규 블록을 `NewBlockHashes`(해시 공지) 또는 `NewBlock`(전체 블록 전파)으로 전송, 동기화 요청 노드 로컬에 해당 블록 없음 | 동기화 요청 노드의 Block Fetcher가 블록 수신 후 삽입, `eth_blockNumber` 갱신 |

#### A-2. 트랜잭션 처리

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RT-A-2-01 | Legacy Tx (type 0x0) 발행 | 잔액이 있는 계정 | 정상처리(tx가 블록에 포함되고 영수증의 status == 1), effectiveGasPrice = baseFee + min(tipCap, gasFeeCap-baseFee), gasLimit valid check (21000) |
| RT-A-2-02 | EIP-1559 DynamicFeeTx (type 0x2) 발행 | 잔액이 있는 계정, GasFeeCap ≥ baseFee+tip | tx 정상 처리, effectiveGasPrice = baseFee + min(tipCap, gasFeeCap-baseFee), gasLimit valid check |
| RT-A-2-03 | AccessListTx (type 0x1) 발행 | 잔액이 있는 계정 | AccessList 포함 tx 정상 처리 |
| RT-A-2-04 | Nonce 순서 보장 | 동일 계정에서 연속 nonce tx 발행 | 낮은 nonce 순서대로 블록에 포함 |
| RT-A-2-05a | GasTipCap \< MinTip tx 거부 (txpool 하한 검증) | 정상 노드 | txpool 진입 거부, `errors.Is(err, ErrUnderpriced) == true`, 메시지 prefix `transaction underpriced: gas tip cap` 일치 |
| RT-A-2-05b | GasFeeCap \< MinBaseFee+MinTip tx 거부 (Anzeon 활성) | Anzeon 활성화 (`AnzeonBlock=0`) | txpool 진입 거부, `errors.Is(err, ErrUnderpriced) == true`, 메시지 prefix `transaction underpriced: gas fee cap` 일치 |
| RT-A-2-06 | 잔액 부족 tx 거부 | 전송 금액 \> 잔액 | txpool 진입 시 `ErrInsufficientFunds` |
| RT-A-2-07 | Gas Limit 초과 tx 거부 | GasLimit \> 블록 gas limit | txpool 진입 거부 |
| RT-A-2-08 | eth\_getTransactionReceipt — effectiveGasPrice 확인 | 로컬 채굴된 tx | receipt.effectiveGasPrice != null, 올바른 값 |
| RT-A-2-09 | 트랜잭션 교체 (replacement tx) | pending 상태인 tx와 동일 nonce, 더 높은 GasFeeCap으로 tx 재발행 | 기존 pending tx가 새 tx로 교체되어 새 tx만 블록에 포함 |
| RT-A-2-10 | SetCodeTx (type 0x4) 발행 — EIP-7702 계정 코드 위임 | delegator 계정에 잔액 존재, 유효한 authorization 목록 | tx 정상 처리, authorization 목록에 지정된 계정 코드가 delegation prefix(`0xef0100`) + delegate 주소로 설정됨, effectiveGasPrice = baseFee + min(tipCap, gasFeeCap-baseFee), gasLimit valid check |

#### A-3. 스마트 컨트랙트

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RT-A-3-01 | 컨트랙트 배포 | 배포 tx 발행 | contractAddress가 receipt에 포함, eth\_getCode 비어있지 않음 |
| RT-A-3-02 | 컨트랙트 상태 변경 호출 | 배포된 컨트랙트 존재 | 호출 tx 정상 처리, 상태 변경이 eth\_call로 확인 |
| RT-A-3-03 | eth\_call — view 함수 조회 | 배포된 컨트랙트 존재 | 트랜잭션 발행 없이 결과 반환 |
| RT-A-3-04 | eth\_estimateGas 정상 동작 | 임의 tx 파라미터 | 실제 소비 가스 이상의 값 반환 |
| RT-A-3-05 | 컨트랙트 revert 처리 — eth\_call 에러 반환 | revert하는 함수를 eth\_call로 호출 | `execution reverted` 에러 및 revert reason 문자열 반환 |
| RT-A-3-06 | 컨트랙트 revert tx — receipt status == 0 | revert하는 함수를 tx로 실행 | receipt.status == 0, revert 시점까지 사용한 가스만 차감(잔여 가스 환불), 상태 변경 롤백 |
| RT-A-3-07 | 컨트랙트 out-of-gas tx — 가스 전량 소비 | gasLimit을 실제 소비보다 부족하게 설정하여 tx 실행 | receipt.status == 0, gasUsed == gasLimit (가스 전량 소비, 환불 없음), 상태 변경 롤백 |

#### A-4. RPC API

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RT-A-4-01 | eth\_blockNumber — 최신 블록 번호 조회 | 노드 운영 중 | 단조 증가하는 블록 번호 반환 |
| RT-A-4-02 | eth\_getBalance — 계정 잔액 조회 | 잔액 있는 계정 | 올바른 잔액 반환 (Wei 단위) |
| RT-A-4-03 | eth\_sendRawTransaction — 서명된 tx 전파 | 서명된 tx 직렬화 데이터 | txHash 반환, 블록에 포함 되기 전 txpool에서 조회 가능 |
| RT-A-4-04 | eth\_getLogs — 이벤트 로그 조회 | 이벤트를 발생시킨 tx | 필터 조건에 맞는 로그 반환 |
| RT-A-4-05 | eth\_chainId — genesis 설정과 일치 확인 | 노드 운영 중 | genesis.json의 `chainId` 값과 동일한 값 반환 |
| RT-A-4-06 | eth\_subscribe (newHeads) — WebSocket 신규 블록 구독 | WebSocket 연결, 블록 생산 중 | 블록 생성마다 블록 헤더 이벤트 수신 |
| RT-A-4-07 | eth\_subscribe (logs) — WebSocket 이벤트 로그 구독 | WebSocket 연결, address/topics 필터 지정 | 조건에 맞는 이벤트 발생 시 실시간 수신 |

---

### B. WBFT 합의 엔진

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RT-B-01 | 블록 생산 주기 — 1초 간격 | 정상 운영 중인 검증자 | 연속 블록의 timestamp 차이가 1초 |
| RT-B-02 | 블록 헤더 Extra 필드 — WBFT 서명 포함 확인 | 생산된 블록 | WBFTExtra 디코딩 성공, Current Seal(Committed, Prepared) 수 ≥ quorum |
| RT-B-03 | 에폭 전환 — 검증자 집합 갱신 | 에폭 길이만큼 블록 생산 | 에폭 블록에서 WBFTExtra.EpochInfo가 새 검증자 집합 반영 |
| RT-B-04 | 검증자 추가 — GovValidator 제안·승인·적용 | 현재 검증자가 신규 검증자 추가 제안 및 quorum 승인 | 다음 에폭부터 신규 검증자가 블록 서명에 참여 |
| RT-B-05 | 검증자 제거 — GovValidator 제안·승인·적용 | 현재 검증자가 기존 검증자 제거 제안 및 quorum 승인 | 다음 에폭부터 해당 검증자가 블록 생산에서 제외 |
| RT-B-06 | GasTip 헤더 반영 — worker → WBFTExtra.GasTip | GovValidator 컨트랙트 storage `gasTip` 값이 새 값(T2)으로 이미 변경된 상태 (RT-F-3-05의 후행 단계) | 다음 블록 생성 시 worker가 컨트랙트 storage를 읽어 `header.Extra`의 `WBFTExtra.GasTip = T2`로 기록, `istanbul_getWbftExtraInfo(N).gasTip == T2`, `eth_gasPrice` 응답값도 T2 기반으로 변경. 헤더 검증(`engine.verifyHeader`)에서 storage 값과 헤더 값 불일치 시 `GasTipMismatchError`로 거부 |
| RT-B-07 | `istanbul_getValidators` RPC — 현재 검증자 목록 조회 | 운영 중인 노드 | 현재 에폭의 검증자 주소 목록 반환 |
| RT-B-08 | 쿼럼 미달 | 쿼럼 미만의 서명이 포함된 블록 헤더 | 블록 수락 거부, 체인에 미포함 |
| RT-B-09 | 라운드 체인지 동작 — 제안자 노드 중단 시 | 현재 라운드 제안자 노드를 강제 중단 (RequestTimeout 초과) | 나머지 노드들이 RoundChange 메시지를 교환, 다음 라운드 제안자가 블록 생성 (`istanbul_getWbftExtraInfo` Round \> 0) |
| RT-B-10 | 라운드 체인지 후 블록 정상 연결 | RT-B-09 이후 블록 생산 재개 | 라운드 체인지로 생성된 블록이 이전 블록과 parentHash로 정상 연결, 이후 Round=0으로 복귀 |
| RT-B-11 | PrevCommittedSeal 수집 확인 | 정상 운영 중, 블록 N+1 생성 후 WBFTExtra 조회 | 블록 N+1의 `PrevCommittedSeal.Sealers`에 블록 N의 서명자 수 ≥ quorum (테스트 환경에서는 밸리데이터 수만큼 모으는지 확인) |
| RT-B-12 | PrevPreparedSeal 수집 확인 | 정상 운영 중, 블록 N+1 생성 후 WBFTExtra 조회 | 블록 N+1의 `PrevPreparedSeal.Sealers`에 블록 N의 Prepare 서명자 수 ≥ quorum (테스트 환경에서는 밸리데이터 수만큼 모으는지 확인) |

---

### C. Anzeon 가스 가격 정책

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RT-C-01 | 일반 계정 GasTip 강제 적용 | Anzeon 활성화, 비인증 계정 | 블록 헤더의 GasTip 값으로 tipCap 강제 적용 |
| RT-C-02 | 인증 계정(Authorized) GasTip 자유 설정 | Anzeon 활성화, 인증된 계정 | 계정이 지정한 GasTipCap 그대로 사용 |
| RT-C-03 | baseFee 증가 — 블록 사용률 \> 20% | 블록 gas 사용률 \> 20% | 다음 블록 baseFee가 2% 증가 |
| RT-C-04 | baseFee 유지 — 블록 사용률 6\~20% | 블록 gas 사용률 6\~20% | 다음 블록 baseFee 변동 없음 |
| RT-C-05 | baseFee 감소 — 블록 사용률 \< 6% | 블록 gas 사용률 \< 6% | 다음 블록 baseFee가 2% 감소 |
| RT-C-06 | baseFee MinBaseFee 하한 유지 | baseFee가 MinBaseFee(20000000000000) 이하 | baseFee가 MinBaseFee 아래로 내려가지 않음 |
| RT-C-07 | baseFee MaxBaseFee 상한 유지 | baseFee가 MaxBaseFee 근접 | baseFee가 MaxBaseFee를 초과하지 않음 |

---

### D. Fee Delegation (type 0x16)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RT-D-01 | FeeDelegateDynamicFeeTx 정상 처리 | Applepie 이상 활성화, Sender·FeePayer 서명 모두 유효 | tx가 블록에 포함, Sender 잔액 변동 없음, FeePayer 가스비 차감 |
| RT-D-03 | Sender 서명 검증 실패 시 거부 | Sender 서명이 조작된 tx | `ErrInvalidSig` 또는 검증 실패로 txpool 거부 |
| RT-D-04 | FeePayer 서명 검증 실패 시 거부 | FeePayer 서명이 조작된 tx | txpool 거부 |
| RT-D-05 | FeePayer 잔액 부족 시 거부 | FeePayer 잔액 \< 가스비 | 블록 실행 중 `ErrInsufficientFunds` |

---

### E. 블랙리스트 / 권한 계정

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RT-E-01 | 블랙리스트 계정이 Sender인 tx 거부 | GovCouncil이 계정 A를 blacklist 등록, 계정 A가 tx 발행 | `ErrBlacklistedAccount` — tx 실행 거부 |
| RT-E-02 | 블랙리스트 계정이 Recipient인 tx 거부 | GovCouncil이 계정 B를 blacklist 등록, 계정 B로 코인 전송 | `ErrBlacklistedAccount` — tx 실행 거부 |
| RT-E-03 | FeePayer가 블랙리스트 계정인 경우 거부 | FeePayer 주소가 블랙리스트에 등록됨 | `ErrBlacklistedAccount` 반환 |
| RT-E-04 | 블랙리스트 해제 후 tx 정상 처리 | 블랙리스트 등록 계정을 unBlacklist 처리 | 해제 이후 tx 정상 처리 |
| RT-E-05 | Zero Address 전송 차단 | Anzeon 활성화, `to=0x000...000`으로 코인 전송 | `ErrZeroAddressTransfer` 반환 |
| RT-E-06 | Precompile 주소로 코인 전송 차단 | Anzeon 활성화, precompile 주소로 value \> 0 전송 | `ErrValueTransferToPrecompile` 반환 |
| RT-E-07 | AccountManager.isBlacklisted 조회 | 블랙리스트 등록된 계정 | `true` 반환 |
| RT-E-08 | AccountManager.isAuthorized 조회 | 인증 계정(Authorized) | `true` 반환, 일반 계정은 `false` |
| RT-E-09 | 인증계정 tx 실행 시 AuthorizedTxExecuted 이벤트 발생 | Anzeon 활성화, 인증계정이 tx 발행 | receipt의 logs에 AccountManager(`0xB00003`) 주소로 `AuthorizedTxExecuted` 이벤트 포함 |

---

### F. 시스템 컨트랙트 & 거버넌스

#### F-1. NativeCoinAdapter (0x1000)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RT-F-1-01 | ERC-20 transfer — 기본 코인 전송과 동일 효과 | NativeCoinAdapter.transfer 호출 | 수신자 eth\_getBalance 증가, Transfer 이벤트 emit |
| RT-F-1-02 | balanceOf — eth\_getBalance와 동일 값 반환 | 임의 계정 | `NativeCoinAdapter.balanceOf(addr) == eth_getBalance(addr)` |
| RT-F-1-03 | approve / transferFrom 정상 동작 | 임의 계정이 approve 후 transferFrom 호출 | allowance 차감 및 잔액 이동, Transfer 이벤트 emit |
| RT-F-1-04 | Mint 실행 시 Transfer(0x0→beneficiary) 이벤트 발생 | GovMinter mint 제안 실행 | receipt logs에 NativeCoinAdapter `Transfer(from=0x0, to=beneficiary, amount)` 이벤트 포함, beneficiary 잔액 증가 |
| RT-F-1-05 | Burn 실행 시 Transfer(account→0x0) 이벤트 발생 | GovMinter burn 제안 실행 | receipt logs에 NativeCoinAdapter `Transfer(from=account, to=0x0, amount)` 이벤트 포함, account 잔액 감소 |

#### F-2. GovMinter 발행/소각 거버넌스 (0x1003)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RT-F-2-01 | 코인 발행 — 제안·승인·실행 흐름 | Minter 멤버 quorum 이상이 제안 승인 | 지정 계정의 잔액이 제안 금액만큼 증가 |
| RT-F-2-02 | 코인 소각 — 제안·승인·실행 흐름 | Minter 멤버 quorum 이상이 소각 제안 승인, 소각 계정 잔액 충분 | 해당 계정 잔액이 소각 금액만큼 감소, 총 공급량 감소 |
| RT-F-2-03 | quorum 미달 — 발행 제안 미실행 | quorum 미만 승인 | 제안이 Voting 상태 유지, 발행 미실행 |

#### F-3. GovValidator (0x1001)

RT-F-3-01\~02 : RT-B-04 중복으로 제거  
RT-F-3-03 : RT-B-05와 중복으로 제거

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| ~~RT-F-3-01 ~~ | ~~검증자 추가 제안 생성 ~~ | ~~운영키(member) 계정으로 addValidator 제안 ~~ | ~~제안이 Pending 상태로 등록, 제안 ID 반환 ~~ |
| ~~RT-F-3-02~~ | ~~검증자 추가 제안 승인 — quorum 달성 시 실행 ~~ | ~~quorum 이상 멤버가 승인 ~~ | ~~제안이 Executed 상태로 전환, 다음 에폭에 검증자 반영 ~~ |
| ~~RT-F-3-03 ~~ | ~~검증자 제거 제안 — quorum 달성 시 실행 ~~ | ~~현재 검증자 제거 제안에 quorum 이상 승인 ~~ | ~~제안 실행, 다음 에폭부터 해당 검증자 제외 ~~ |
| RT-F-3-04 | 검증자 메타데이터 조회 — `validatorList()` / `validatorToOperator(v)` / `validatorToBlsKey(v)` 다중 호출 | GovValidator 컨트랙트 상태 | (1) `validatorList()`가 검증키 주소 배열 반환, (2) 각 검증키 v에 대해 `validatorToOperator(v)`가 운영키 반환, (3) 각 검증키 v에 대해 `validatorToBlsKey(v)`가 BLS키(48바이트) 반환 |
| RT-F-3-05 | GasTip 거버넌스 lifecycle — `proposeGasTip` → 승인 → execute | 운영키(member)가 `proposeGasTip(T2)` 호출, quorum 이상 멤버가 승인 후 execute | 제안 상태가 `Voting` → `Approved` → `Executed`로 전환, `_setGasTip(T2)` 호출 결과 컨트랙트 storage `gasTip == T2`, `GasTipUpdated(oldTip, T2, msg.sender)` 이벤트 발생. 동일 값(`SameGasTip`) 또는 0 제안은 revert |
| RT-F-3-06 | expiry 초과 제안 자동 만료 | 제안 생성 후 expiry 기간 경과 | 제안이 Expired 상태로 전환, 실행 불가 |

#### F-4. GovMasterMinter (0x1002)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RT-F-4-01 | Minter 등록 제안 — quorum 달성 시 실행 | GovMasterMinter 멤버 quorum 이상 승인 | 신규 Minter 주소가 GovMinter의 멤버로 추가됨 |
| RT-F-4-02 | Minter 삭제 제안 — quorum 달성 시 실행 | GovMasterMinter 멤버 quorum 이상 승인 | 해당 Minter가 GovMinter 멤버에서 제거됨 |
| RT-F-4-03 | GovMasterMinter 자체 멤버 추가/제거 제안 — quorum 달성 시 실행 | GovMasterMinter 멤버 quorum 이상 승인 | MasterMinter 멤버 집합 변경 반영 |
| RT-F-4-04 | 비멤버 계정의 Minter 등록 제안 거부 | GovMasterMinter 멤버가 아닌 계정이 제안 | 컨트랙트 revert 발생 |

#### F-5. GovCouncil (0x1004)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RT-F-5-01 | blacklist 등록 제안 — quorum 달성 시 실행 | GovCouncil 멤버 quorum 이상 승인 | 대상 계정이 AccountManager에 블랙리스트 등록, isBlacklisted == true |
| RT-F-5-02 | unBlacklist 제안 — quorum 달성 시 실행 | GovCouncil 멤버 quorum 이상 승인 | 대상 계정의 블랙리스트 해제, isBlacklisted == false |
| RT-F-5-03 | authorize 제안 — quorum 달성 시 실행 | GovCouncil 멤버 quorum 이상 승인 | 대상 계정이 AccountManager에 Authorized 등록, isAuthorized == true |
| RT-F-5-04 | unAuthorize 제안 — quorum 달성 시 실행 | GovCouncil 멤버 quorum 이상 승인 | 대상 계정 Authorized 해제, isAuthorized == false |
| RT-F-5-05 | 비멤버 계정의 직접 blacklist 호출 거부 | GovCouncil 멤버가 아닌 계정이 AccountManager.blacklist 직접 호출 | 컨트랙트 revert 발생 |
| RT-F-5-06 | blacklist 등록 시 AddressBlacklisted 이벤트 발생 | GovCouncil blacklist 제안 실행 | receipt logs에 GovCouncil `AddressBlacklisted(account, proposalId)` 이벤트 포함 |
| RT-F-5-07 | unBlacklist 시 AddressUnblacklisted 이벤트 발생 | GovCouncil unBlacklist 제안 실행 | receipt logs에 GovCouncil `AddressUnblacklisted(account, proposalId)` 이벤트 포함 |
| RT-F-5-08 | authorize 시 AuthorizedAccountAdded 이벤트 발생 | GovCouncil authorize 제안 실행 | receipt logs에 GovCouncil `AuthorizedAccountAdded(account, proposalId)` 이벤트 포함 |
| RT-F-5-09 | unAuthorize 시 AuthorizedAccountRemoved 이벤트 발생 | GovCouncil unAuthorize 제안 실행 | receipt logs에 GovCouncil `AuthorizedAccountRemoved(account, proposalId)` 이벤트 포함 |

---

### G. API 호출

#### G-1. eth 블록/트랜잭션 조회 API

| ID | 메서드 | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- | --- |
| RT-G-1-01 | `eth_getBlockByNumber` | latest 블록 조회 | 블록이 1개 이상 생성된 상태 | 블록 객체 반환 (number, hash, transactions 포함) |
| RT-G-1-02 | `eth_getBlockByHash` | 블록 해시로 조회 | `eth_getBlockByNumber`로 얻은 hash | RT-G-1-01과 동일한 블록 객체 반환 |
| RT-G-1-03 | `eth_getTransactionByHash` | tx 해시로 tx 조회 | 블록에 포함된 tx | tx 객체 반환 (blockNumber, from, to, value 포함) |
| RT-G-1-04 | `eth_getTransactionReceipt` | tx 영수증 조회 | 블록에 포함된 tx | status, effectiveGasPrice, logs 포함된 receipt 반환 |
| RT-G-1-05 | `eth_getTransactionCount` | 계정 nonce 조회 | tx를 발행한 계정 | 발행한 tx 수와 일치하는 nonce 반환 |
| RT-G-1-06 | `eth_getCode` | 시스템 컨트랙트 코드 조회 | NativeCoinAdapter 주소(`0x1000`) | 비어있지 않은 bytecode 반환 |

#### G-2. 가스/수수료 API

| ID | 메서드 | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- | --- |
| RT-G-2-01 | `eth_gasPrice` | 현재 가스 가격 조회 | Anzeon 활성화 | baseFee + GasTip 합산값 반환 |
| RT-G-2-02 | `eth_maxPriorityFeePerGas` | 최대 우선 수수료 조회 | Anzeon 활성화 | 블록 헤더의 WBFTExtra.GasTip 값 반환 |
| RT-G-2-03 | `eth_feeHistory` | 최근 N개 블록 수수료 이력 조회 | 블록이 N개 이상 생성된 상태 | baseFeePerGas 배열이 MinBaseFee 이상인 값으로 반환 |
| RT-G-2-04 | `eth_estimateGas` | 시스템 컨트랙트 호출 가스 추정 | NativeCoinAdapter.transfer 호출 데이터 | 실제 소비 가스 이상의 추정값 반환, 0 아님 |

#### G-3. WBFT 커스텀 API (`istanbul_*`)

| ID | 메서드 | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- | --- |
| RT-G-3-01 | `istanbul_nodeAddress` | 현재 노드의 블록 서명 주소 조회 | 검증자 노드 | 노드의 coinbase 주소 반환 |
| RT-G-3-02 | `istanbul_getValidators` | 특정 블록의 검증자 목록 조회 | 블록 번호 지정 | 해당 블록 에폭의 검증자 주소 배열 반환 |
| RT-G-3-03 | `istanbul_getCommitSignersFromBlock` | 특정 블록의 서명자 목록 조회 | 블록 번호 지정 | Author(제안자) + Committers(커밋 서명자) 반환, Committers 수 ≥ quorum |
| RT-G-3-04 | `istanbul_getWbftExtraInfo` | 블록의 WBFT Extra 상세 정보 조회 | 블록 번호 지정 | GasTip, EpochInfo, CommittedSeal 등 파싱된 Extra 데이터 반환 |
| RT-G-3-05 | `istanbul_status` | 검증자 활동 통계 조회 | 블록 범위(start\~end) 지정 | SealerActivity, AuthorCounts, RoundStats 포함된 통계 반환 |
| RT-G-3-06 | `istanbul_isValidator` | 현재 노드 검증자 여부 확인 | 검증자 노드 / 비검증자 노드 각각 | 검증자 노드 `true`, 비검증자 노드 `false` |

#### G-4. 관리/진단 API

| ID | 메서드 | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- | --- |
| RT-G-4-01 | `net_peerCount` | 연결된 피어 수 조회 | 2노드 이상 연결된 상태 | 1 이상의 정수 반환 |
| RT-G-4-02 | `txpool_status` | 트랜잭션 풀 상태 조회 | pending tx(연속 nonce)와 queued tx(nonce gap이 있는 tx)를 각각 txpool에 주입한 상태 | `pending` 건수와 `queued` 건수가 주입한 수와 일치하고 오류 없이 응답 |
| RT-G-4-03 | `txpool_content` | 트랜잭션 풀 상세 내용 조회 | pending tx(연속 nonce)와 queued tx(nonce gap이 있는 tx)를 각각 txpool에 주입한 상태 | `pending` 항목에 연속 nonce tx, `queued` 항목에 nonce gap tx가 각각 포함되어 from·nonce·gasPrice 등 tx 상세 필드 반환 |
| RT-G-4-04 | `admin_peers` | 연결된 피어 상세 정보 조회 | 2노드 이상 연결된 상태 | 피어 ID, 네트워크 주소, 프로토콜 정보 포함된 객체 배열 반환 |

#### G-5. StableNet 고유 API

| ID | 메서드 | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- | --- |
| RT-G-5-01 | `eth_signRawFeeDelegateTransaction` | Fee Delegation tx FeePayer 서명 | Applepie 활성화, Sender가 서명한 FeeDelegateTx RLP | FeePayer 서명(FV/FR/FS)이 추가된 완성된 tx RLP 반환 |
| RT-G-5-02 | `eth_call` (NativeCoinAdapter.totalSupply) | 총 발행량 조회 | 발행된 코인이 존재 | 현재 총 공급량 반환, `eth_getBalance` 합산과 일치 |
| RT-G-5-03 | `eth_call` (NativeCoinAdapter.allowance) | 위임 한도 조회 | approve 완료된 계정 쌍 | 설정한 allowance 값 반환 |
