# 1st Test Cases (v1.0.0+)

> 출처: Confluence [1st Test Cases (v1.0.0+)](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2614231162) (페이지 ID 2614231162, 버전 3, 최종 수정 2026-08-05)  
> 상위 페이지: 테스트(v1.0.0이후 변경 사항)  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김. 4-5 표의 TC-4-5-03/04/12 행은 원본 표 자체가 깨져 있어 그대로 둠)

---

# DEV\_CHANGES 테스트 케이스

v1.0.0 이후 변경사항(`DEV_CHANGES.md`)에 대한 테스트 케이스 목록.

| 섹션 | 변경사항 | TC 수 |
| --- | --- | --- |
| 1-1 | GovMinter 시스템 컨트랙트 업그레이드 | 12 |
| 1-2 | secp256r1 서명 검증 지원 | 6 |
| 1-3 | 최소 가스비 하한선 적용 | 6 |
| 2-1 | 잘못된 KZG 증명 피어 차단 | - |
| 2-2 | 암호화 취약점 수정 - ECIES | - |
| 2-3 | secp256k1 좌표 유효성 검사 강화 | - |
| 2-4 | P2P 메시지 DoS 취약점 수정 | - |
| 3-1 | 암호화 서명 처리 속도 향상 | 4 |
| 4-1 | 체인 설정 초기화 오류 수정 | 3 |
| 4-2 | 가스 추정 오류 수정 | 3 |
| 4-3 | 설정 문자열 처리 수정 | 6 |
| 4-4 | 동일 블록에 복수 하드포크 적용 시 누락 버그 수정 | 4 |
| 4-5 | Genesis 인증 계정·블랙리스트 상태 불일치 수정 | 12 |
| 4-6 | Snap Sync 환경에서 EffectiveGasPrice 조회 오류 수정 | 4 |
| 5-1 | Go 런타임 1.23.12 업그레이드 | 3 |
| 5-2 | 하드포크 업그레이드 시스템 통합 | 6 |
| 5-3 | 제네시스 블록 구성 방식 표준화 | 1 |
| **합계** |  | **70** |

---

## 1. Boho 하드포크 추가

### 1-1. GovMinter 시스템 컨트랙트 업그레이드 (`bf17c9607`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-1-1-01 | 소각 제안 취소 시 refundableBalance 이동 | Boho 활성화, 소각 제안이 존재하며 상태가 Cancelled | 해당 제안의 burnBalance가 refundableBalance로 이동 |
| TC-1-1-02 | 소각 제안 거부 시 refundableBalance 이동 | Boho 활성화, 소각 제안이 존재하며 상태가 Rejected | 해당 제안의 burnBalance가 refundableBalance로 이동 |
| TC-1-1-03 | 소각 제안 만료 시 refundableBalance 이동 | Boho 활성화, 소각 제안이 존재하며 상태가 Expired | 해당 제안의 burnBalance가 refundableBalance로 이동 |
| TC-1-1-04 | 소각 실행 성공 시 refundableBalance 미이동 | Boho 활성화, 소각 제안이 정상 실행(Executed) | burnBalance가 소각되고 refundableBalance에 추가되지 않음 |
| TC-1-1-05 | claimBurnRefund 정상 출금 | refundableBalance \> 0인 계정 | 잔액이 0으로 변경되고 해당 금액이 계정에 전송됨 |
| TC-1-1-06 | claimBurnRefund 잔액 0 시 revert | refundableBalance == 0인 계정 | `NoRefundAvailable` revert 발생 |
| TC-1-1-07 | claimBurnRefund 중복 호출 방어 | 이전에 claimBurnRefund 호출 완료한 계정 | 두 번째 호출 시 `NoRefundAvailable` revert |
| TC-1-1-08 | \_cleanupBurnDeposit 멱등성 확인 | 동일 proposalId로 \_cleanupBurnDeposit 2회 호출 | 두 번째 호출 시 amount == 0이므로 상태 변경 없이 조기 리턴 |
| TC-1-1-09 | BurnRefundClaimed 이벤트 발생 확인 | claimBurnRefund 정상 실행 | `BurnRefundClaimed(address, amount)` 이벤트 emit |
| TC-1-1-10 | BurnDepositRefunded 이벤트 발생 확인 | 취소/거부/만료된 제안에 대해 \_cleanupBurnDeposit 실행 | `BurnDepositRefunded(proposalId, proposer, amount)` 이벤트 emit |
| TC-1-1-11 | Boho 하드포크 전후 GovMinter 바이트코드 변경 확인 | Boho 블록 도달 | GovMinter 컨트랙트 코드가 v1에서 v2로 교체됨 |
| TC-1-1-12 | v2 업그레이드 후 기존 v1 상태 보존 확인 | Boho 블록 도달, 기존 v1 상태(mintBalance 등) 존재 | v1에서 사용하던 스토리지 슬롯 값이 유지됨 |

### 1-2. secp256r1 서명 검증 지원 - EIP-7951 (`9a5c62ae7`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-1-2-01 | Boho 활성화 후 secp256r1 프리컴파일 호출 성공 | Boho 하드포크 활성화 | `PrecompiledContractsBoho`에 secp256r1 포함, 정상 실행 |
| TC-1-2-02 | Boho 이전에는 secp256r1 프리컴파일 미존재 | Anzeon만 활성화 (Boho 미활성화) | secp256r1 주소 호출 시 일반 컨트랙트 호출로 처리 (프리컴파일 아님) |
| TC-1-2-03 | 유효한 secp256r1 서명 검증 성공 | Boho 활성화, 올바른 r1 서명 데이터 | 검증 성공, 결과값 1 반환 |
| TC-1-2-04 | 잘못된 secp256r1 서명 검증 실패 | Boho 활성화, 변조된 r1 서명 데이터 | 검증 실패, 결과값 0 반환 |
| TC-1-2-05 | 입력 길이 부족 시 처리 | Boho 활성화, 160바이트 미만 입력 | 빈 결과 반환 (에러 없음) |
| TC-1-2-06 | Boho 하드포크 활성 주소 목록에 secp256r1 포함 확인 | Boho 활성화 | `PrecompiledAddressesBoho`에 secp256r1 주소 포함 |

### 1-3. 최소 가스비 하한선 적용 (`47721b190`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-1-3-01 | GasFeeCap \< minBaseFee + minTip인 tx 거부 | Anzeon 활성화, London 포크 활성화 | `ErrUnderpriced` 에러로 txpool 진입 거부 |
| TC-1-3-02 | GasFeeCap == minBaseFee + minTip인 tx 허용 | Anzeon 활성화, London 포크 활성화 | txpool 진입 허용 |
| TC-1-3-03 | GasFeeCap \> minBaseFee + minTip인 tx 허용 | Anzeon 활성화, London 포크 활성화 | txpool 진입 허용 |
| TC-1-3-04 | LegacyTx의 GasPrice에 최소 가스비 적용 | Anzeon 활성화, LegacyTx(GasPrice 사용) | GasPrice \< minBaseFee + minTip이면 거부 |
| TC-1-3-05 | AccessListTx에 최소 가스비 적용 | Anzeon 활성화, AccessListTx | GasPrice \< minBaseFee + minTip이면 거부 |
| TC-1-3-06 | DynamicFeeTx에 최소 가스비 적용 | Anzeon 활성화, DynamicFeeTx(GasFeeCap 사용) | GasFeeCap \< minBaseFee + minTip이면 거부 |

---

## 2. 취약점 패치

### 2-1. 잘못된 KZG 증명 피어 차단 (`315f31e7e`)

> **운영 환경 에서는 테스트가 어렵다.**
> 
> - 정상 노드는 tx 전파 전 자체 검증을 수행하므로, 잘못된 KZG 증명 tx가 네트워크에 전파되지 않음
> - StableNet은 blob tx를 지원하지 않아 이 오류 경로가 자연 발생하지 않음
> - 테스트하려면 검증을 우회한 커스텀 빌드 노드(악의적 피어)를 별도 테스트넷에서 실행해야 함
> - 단위 테스트(`eth/fetcher/tx_fetcher_test.go` → `TestTransactionProtocolViolation`)로 검증

### 2-2. 암호화 취약점 수정 - ECIES (`c37ae123a`)

> **운영 환경에서는 테스트가 어렵다.**
> 
> - P2P 핸드셰이크는 노드 간 연결 시점에 발생하며, 정상 노드는 항상 유효한 공개키를 사용
> - 재현하려면 잘못된 타원 곡선 공개키(invalid-curve)를 의도적으로 전송하는 커스텀 빌드 노드 필요
> - 단위 테스트(`crypto/ecies`, `p2p/rlpx/rlpx_oracle_poc_test.go`)로 검증

### 2-3. secp256k1 좌표 유효성 검사 강화 (`f867ab623`)

> **운영 환경에서는 테스트가 어렵다.**
> 
> - 정상적인 키 생성·서명 과정에서는 곡선 외부 좌표가 생성되지 않음
> - 재현하려면 의도적으로 조작된 좌표값을 crypto 함수에 직접 전달해야 함
> - 단위 테스트(`crypto/secp256k1`)로 검증

### 2-4. P2P 메시지 DoS 취약점 수정 (`b23c5a831`)

> **운영 환경에서는 테스트가 어렵다.**
> 
> - 정상 노드는 규격에 맞는 크기의 P2P 메시지만 전송하므로 자연 발생하지 않음
> - 재현하려면 허용 크기를 초과하는 메시지를 의도적으로 전송하는 커스텀 피어 노드 필요
> - 단위 테스트(`p2p/tracker`, `rlp`, `eth/protocols/eth`, `eth/protocols/snap`)로 검증

---

## 3. 성능 개선

### 3-1. 암호화 서명 처리 속도 향상 (`f867ab623`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-3-1-01 | 서명 생성 성능 측정 (BenchmarkSign) | libsecp256k1 업데이트 적용 | 기존 대비 약 38% 속도 향상 (21μs vs 35μs/op 수준) |
| TC-3-1-02 | 서명 복원 성능 측정 (BenchmarkRecover) | libsecp256k1 업데이트 적용 | 기존 대비 약 28% 속도 향상 (33μs vs 47μs/op 수준) |
| TC-3-1-03 | 서명 검증 성능 측정 (BenchmarkVerifySignature) | libsecp256k1 업데이트 적용 | 기존 대비 약 29% 속도 향상 (29μs vs 41μs/op 수준) |
| TC-3-1-04 | 업데이트 후 기존 서명과의 호환성 확인 | 업데이트 전에 생성한 서명 | 새 라이브러리로 검증/복원 성공 |

> ```
> # TC-3-1-01: 서명 생성 (crypto/secp256k1/secp256_test.go)
> go test -bench=BenchmarkSign ./crypto/secp256k1/...
> 
> # TC-3-1-02: 서명 복원 (crypto/secp256k1/secp256_test.go)
> go test -bench=BenchmarkRecover ./crypto/secp256k1/...
> 
> # TC-3-1-03: 서명 검증 / ecrecover (crypto/signature_test.go)
> go test -bench=BenchmarkVerifySignature ./crypto/...
> go test -bench=BenchmarkEcrecoverSignature ./crypto/...
> ```

> **TC-3-1-04는 노드 동기화 과정에서 자동 검증된다.**  
> 업데이트 전에 생성된 기존 블록·트랜잭션의 서명은 노드가 체인을 동기화하는 과정에서 새 라이브러리로 재검증되므로,  
> 동기화가 정상적으로 완료되면 호환성이 보장된다.

---

## 4. 버그 수정

### 4-1. 체인 설정 초기화 오류 수정 (`ac572e72b`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-4-1-01 | WBFT 체인 설정으로 엔진 정상 생성 | StableNet 체인 설정(Anzeon 포함) | WBFT 엔진이 올바른 체인 설정으로 초기화 |
| TC-4-1-02 | 하드포크 적용 시 체인 설정 반영 | BohoFork 하드포크 블록 지정 | 하드포크 값이 체인 설정에 반영됨 |
| TC-4-1-03 | 저장된 제네시스와 설정 불일치 감지 | DB에 저장된 genesis block hash와 다른 hash를 생성하는 genesis 설정(alloc, ChainID, ExtraData 등 변경)으로 노드 시작 | `GenesisMismatchError` 반환 |

### 4-2. 가스 추정 오류 수정 (`db7c4b43c`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-4-2-01 | EIP-7702 tx의 AuthorizationList 포함 가스 추정 | SetCode tx에 AuthorizationList 포함 | 가스 추정 결과에 AuthorizationList 비용 반영 |
| TC-4-2-02 | AuthorizationList 미포함 tx 가스 추정 | 일반 DynamicFeeTx | 기존과 동일하게 정상 동작 |
| TC-4-2-03 | 복수 AuthorizationList 항목 가스 추정 | AuthorizationList에 여러 항목 포함 | 모든 항목의 비용이 반영된 가스 추정 |

### 4-3. 설정 문자열 처리 수정 (`2e0469474`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-4-3-01 | 공백 없는 쉼표 구분 주소 목록 | `"0xAAA,0xBBB,0xCCC"` | 3개 주소 정상 파싱 |
| TC-4-3-02 | 공백 포함 쉼표 구분 주소 목록 | `"0xAAA, 0xBBB, 0xCCC"` | 공백 제거 후 3개 주소 정상 파싱 (TC-4-3-01과 동일 결과) |
| TC-4-3-03 | 앞뒤 공백이 있는 주소 | `" 0xAAA , 0xBBB "` | 앞뒤 공백 제거 후 2개 주소 정상 파싱 |
| TC-4-3-04 | 빈 항목이 포함된 목록 | `"0xAAA,,0xBBB"` | 빈 항목 무시, 2개 주소만 파싱 |
| TC-4-3-05 | 단일 주소 (쉼표 없음) | `"0xAAA"` | 1개 주소 정상 파싱 |
| TC-4-3-06 | 빈 문자열 | `""` | 빈 슬라이스 반환 |

### 4-4. 동일 블록에 복수 하드포크 적용 시 누락 버그 수정 (`05a81ec5e`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-4-4-01 | 블록 0에 Anzeon + Boho 하드포크 동시 적용 | AnzeonBlock=0, BohoBlock=0 | Anzeon의 시스템 컨트랙트가 v1으로 초기화되고, Boho 하드포크로 GovMinter 바이트코드가 v2로 교체된 상태로 제네시스 alloc 생성 |
| TC-4-4-02 | 동일 블록 N에 두 하드포크 적용 (런타임) | 두 하드포크의 블록 번호가 동일 (N \> 0) | 블록 N 처리 시 두 하드포크의 StateTransition이 순서대로 병합 실행 |
| TC-4-4-03 | InjectContracts : BohoBlock=0일 때 제네시스에 하드포크 적용 | BohoBlock=0 | Anzeon baseline이 alloc에 추가된 후, GovMinter Code만 v2로 교체되고 Balance·Storage는 보존됨 |
| TC-4-4-04 | InjectContracts : BohoBlock=N(\>0)일 때 제네시스에 하드포크 미적용 | BohoBlock=N (N \> 0) | 제네시스 alloc에는 Anzeon v1 코드만 존재, Boho 업그레이드는 포크 블록 N에서 런타임에서 처리됨 |

### 4-5. Genesis 인증 계정·블랙리스트 상태 불일치 수정 (`9b8df257a`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-4-5-01 | genesis alloc의 Authorized 비트가 GovCouncil authorized 목록에 동기화 | genesis alloc의 `Account.Extra`에 Authorized 비트가 설정된 주소 존재, GovCouncil params에는 해당 주소 미포함 | `statedb.IsAuthorized()`와 GovCouncil authorized 목록이 일치하며, 해당 계정이 거버넌스 관리 대상에 포함됨 |
| TC-4-5-02 | genesis alloc의 Blacklisted 비트가 GovCouncil blacklist에 동기화 | genesis alloc의 `Account.Extra`에 Blacklisted 비트가 설정된 주소 존재, GovCouncil params에는 해당 주소 미포함 | `statedb.IsBlacklisted()`와 GovCouncil blacklist가 일치하며, 해당 계정이 거버넌스 관리 대상에 포함됨 |
| TC-4-5-03 | GovCouncil params의 authorized 주소가 alloc.Extra에도 동기화 | GovCouncil params에 authorized 주소 지정, genesis alloc에는 해당 주소의 Authorized 비트 미설정 또는 엔트리 미존재 |  |
| ovCouncil storage slot과 alloc.Extra Authorized 비트 양쪽에 동일 상태가 반영됨 \| |  |  |  |
| TC-4-5-04 | GovCouncil params의 blacklist 주소가 alloc.Extra에도 동기화 | GovCouncil params에 blacklist 주소 지정, genesis alloc에는 해당 주소의 Blacklisted 비트 미설정 또는 엔트리 미존재 |  |
| ovCouncil storage slot과 alloc.Extra Blacklisted 비트 양쪽에 동일 상태가 반영됨 \| |  |  |  |
| TC-4-5-05 | alloc.Extra와 GovCouncil params의 주소 집합을 union으로 병합 | 일부 주소는 alloc.Extra에만, 일부 주소는 GovCouncil params에만 존재 | 최종 authorized/blacklist 집합이 두 소스의 합집합으로 구성되고 누락이 없음 |
| TC-4-5-06 | alloc.Extra와 params에 중복으로 존재하는 주소가 한 번만 등록 | 동일 주소가 alloc.Extra와 GovCouncil params 양쪽에 모두 존재 | GovCouncil authorized/blacklist count와 목록에 중복 없이 1회만 반영됨 |
| TC-4-5-07 | 동일 주소가 blacklist와 authorized 양쪽 상태를 동시에 가질 때 두 목록 모두에 반영 | 같은 주소가 alloc.Extra의 Blacklisted 비트와 GovCouncil params의 authorizedAccounts에 동시에 존재 | 해당 주소의 Extra 비트와 GovCouncil 두 목록이 모두 일치하며 어느 한쪽도 누락되지 않음 |
| TC-4-5-08 | 동기화 과정에서 기존 alloc 엔트리의 balance와 무관 계정 상태가 보존 | sync 대상 외 alloc 계정 존재 또는 기존 alloc 엔트리에 잔액이 설정됨 | authorized/blacklist 동기화 후에도 무관 계정의 balance와 기존 alloc 정보가 변경되지 않음 |
| TC-4-5-09 | 정의되지 않은 Account.Extra 비트는 genesis 초기화 단계에서 거부 | genesis alloc의 `Account.Extra`에 Authorized/Blacklisted 외 비트가 설정됨 | `initializeAnzeonGenesis` 또는 동등한 초기화 단계가 에러를 반환하고 체인 초기화가 실패함 |
| TC-4-5-10 | decodePrealloc: Authorized 계정 Extra 복원 | mkalloc으로 Extra = Authorized인 계정 인코딩한 prealloc 바이너리 | account, 컨트랙트 슬롯 모두 반영 |
| TC-4-5-11 | decodePrealloc: Blacklisted 계정 Extra 복원 | mkalloc으로 Extra = Blacklisted인 계정 인코딩한 prealloc 바이너리 | account, 컨트랙트 슬롯 모두 반영 |
| TC-4-5-12 | decodePrealloc: Authorized + Blacklisted 계정 Extra 복원 | mkalloc으로 Extra = Authorized \\ | Blacklisted인 계정 인코딩한 prealloc 바이너리 |

### 4-6. Snap Sync 환경에서 EffectiveGasPrice 조회 오류 수정 (`c55bf2a86`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-4-6-01 | 인증 계정 tx의 EffectiveGasPrice 재계산 | Snap Sync 환경, 인증 계정의 tx receipt (EffectiveGasPrice == nil) | AuthorizedTxExecuted 로그 감지 → 원본 GasTipCap으로 재계산 |
| TC-4-6-02 | 일반 계정 tx의 EffectiveGasPrice 재계산 | Snap Sync 환경, 일반 계정의 tx receipt (EffectiveGasPrice == nil) | headerGasTip 사용하여 EffectiveGasPrice 재계산 |
| TC-4-6-03 | EffectiveGasPrice가 이미 존재하는 경우 덮어쓰지 않음 | 로컬 생성 receipt (EffectiveGasPrice != nil) | 기존 값 유지, 재계산 없음 |
| TC-4-6-04 | AuthorizedTxExecuted 이벤트가 항상 마지막 로그인지 확인 | 인증 계정의 tx 실행 | 해당 이벤트가 receipt의 마지막 로그로 기록됨 |

---

## 5. 내부 개선

### 5-1. Go 런타임 1.23.12 업그레이드 (`8d7930a48`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-5-1-01 | 전체 바이너리 빌드 성공 | Go 1.23.12 환경 | `make all` 성공, 13개 바이너리 정상 생성 |
| TC-5-1-02 | 전체 테스트 통과 | Go 1.23.12 환경 | `make test` 전체 통과 |
| TC-5-1-03 | 런타임 버전 확인 | 빌드된 gstable 바이너리 | `runtime.Version()` 이 "go1.23.12" 반환 |

### 5-2. 하드포크 업그레이드 시스템 통합 (`05a81ec5e`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-5-2-01 | CollectUpgrades가 등록된 모든 하드포크 반환 | Anzeon + Boho 설정 | 두 하드포크의 SystemContractsTransition 모두 반환 |
| TC-5-2-02 | CollectUpgrades 등록 순서 확인 | 여러 하드포크 등록 | 등록 순서대로 반환, 블록 번호 순 정렬 |
| TC-5-2-03 | SetConfigFromChainConfig가 CollectUpgrades 사용 확인 | 노드 시작 시 | 수동 append 대신 CollectUpgrades()를 통해 설정 로드 |
| TC-5-2-04 | v1 버전만 Params 초기화 수행 확인 | GovCouncil v1, GovMinter v1 | Params != nil && Version == "v1"일 때만 state 초기화 |
| TC-5-2-05 | v2 이상 버전은 코드 교체만 수행하며 Balance·Storage는 변경되지 않음 | GovMinter v2 | 코드만 교체, 기존 스토리지 상태 유지 |
| TC-5-2-06 | 미등록 컨트랙트 타입/버전 요청 시 에러 | 존재하지 않는 컨트랙트 버전 | `getContractCode` 에러 반환 |

### 5-3. 제네시스 블록 구성 방식 표준화 (`aa28927fb`)

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TC-5-3-01 | decodePrealloc 방식으로 생성한 제네시스 블록이 기존 JSON 방식과 동일한지 확인 | testnet 각각 JSON 참조 alloc과 decodePrealloc 결과 비교 | 두 방식으로 생성된 genesis block의 alloc, Config, 해시가 완전히 일치 |
