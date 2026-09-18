# 공통 테스트 분리를 위한 설정과 변경 범위

> ID 표기는 Confluence 원래 명세 ID를 우선한다. 실행 ID는 별도 표기한다. ID 대응은 전체 명세 충족이나 PASS를 뜻하지 않는다. [ID 대응표](existing-tc-specs.md)

공통 테스트의 실행 단계와 관찰 방법은 공유하고, 네트워크 연결 정보와 계정은 설정으로 분리한다. 수수료, 합의, 거버넌스처럼 같은 입력에 대한 기대 결과가 다른 부분은 체인별 판정 코드로 둔다. 기존 EnvV2, 계정 label, RPC helper, consensus family를 재사용하는 범위부터 시작한다.

이 문서는 구현할 작업의 범위다. 아래에서 제안하는 프로필 필드나 어댑터가 현재 DSL에 모두 있다는 뜻은 아니다. 공통 후보별 원본 ID와 실제 의존 필드는 [전체 목록](all-tests.json), 실행 구조의 근거는 [하네스 분석](harness-dependencies.md), 체인 규칙의 근거는 [클라이언트 비교](chain-differences.md)에 있다.

## 설정으로 분리할 항목

| 의존 요소 | 프로필에 담을 값 또는 참조 | 현재 재사용할 기능 | 추가로 필요한 처리 |
|---|---|---|---|
| 테스트 계정 | sender, recipient, feePayer, funder 역할, 키 참조, 초기 잔액, nonce 시작 조건 | keys label, `env.accounts`, ResolveAccount | 주소 literal 제거. 역할별 계정을 격리하고 StableNet blacklist/authorized 상태를 사전 확인. 키 값 자체를 공통 JSON에 넣지 않음 |
| RPC Endpoint | 노드 역할별 HTTP/WS, 인증 참조, 허용 namespace, timeout, provider 제한 | 반복 `--rpc`, workspace attach, RPC client | endpoint만 나열한 경우 BP/EN 역할을 추론하지 않음. HTTP 존재를 WS 지원으로 세지 않음 |
| 일반 계약 주소 | 배포 결과 binding 또는 기존 주소, ABI, runtime code hash, 배포 블록 | deployContract, `$binding`, ABI calldata 조립 | 동일 EVM revision용 fixture 준비. 일반 송금의 `to` 주소와 계약 주소를 구분 |
| 시스템 계약 | 체인별 registry/system-contract 설정, ABI 버전, 권한 계정 | 기존 chain capability와 consensus bootstrap | 주소 외에 의미가 달라 아래 체인 어댑터가 필요 |
| Chain ID / Network ID | 기대 ID, 서명 ID, genesis hash, client family/version | CLI/manifest/env genesis, `eth_chainId` | WEMIX3/4의 코드상 mainnet ID는 같으므로 ID만으로 구현 선택 금지. 하네스 기본값과 운영값을 구분 |
| 바이너리 | 파일 경로, SHA, 소스 SHA, 빌드 대상, OS/arch, CGO/tags | `env.binaries`, explicit binary 경로 | go-wbft Makefile의 `gwemix`와 하네스 manifest의 `gwbft` 차이를 해결. 세 클라이언트의 이름을 일괄 치환하지 않음 |
| 포크 / genesis | 활성 높이·시간, genesis template 또는 generator 입력, overlay | `env.hardforks`, genesis, 기존 family | 미지원 opcode/tx type을 설정만으로 활성화할 수 없음. 실제 노드 fork와 대조 |
| 수수료 입력 | fee mode, gas, tip/fee cap, PriceBump, 유효성 기준 블록 | fee args, 서명 계층 | 숫자 입력은 설정화. receipt 비용·거부 사유·baseFee 변화는 체인별 계산 |
| 토폴로지 / 실행 환경 | BP/EN 수, peer 관계, datadir, ports, process 소유 여부 | topology, blueprint, compose/attach | `en1` 참조와 실제 EN 생성 일치. process가 없는 attach에서는 장애 주입 불가 |
| 시간 / 부하 | timeout, polling, 블록 관측 구간, tx량·gas량, 자금 예산 | step별 timeout, load, block 관찰기 | step 내부 상수에 프로필 값을 전달하는 생성기 또는 DSL 확장이 필요. 현재 env가 임의 step 변수 주입을 지원한다고 가정하지 않음 |

EnvV2는 환경 재사용 단위를 이미 정의하며, `extends`는 최상위 필드 전체를 교체하는 얕은 병합이다. 예를 들어 topology의 EN만 덮어쓰면서 BP 구성이 보존된다고 기대하면 안 된다. 공통 case에서 환경 ID를 참조하고, 체인별 완성된 환경 선언을 제공하는 방식부터 적용한다. 근거: `sources/chainbench/internal/dsl/spec_v2.go:54`, `sources/chainbench/internal/dsl/spec_v2.go:263`.

## 별도 구현 또는 체인별 처리가 필요한 항목

| 구분 | 공유할 부분 | 분리할 부분 | 주요 대상 |
|---|---|---|---|
| 공통 transaction sender | 서명 선택, raw 전송, receipt 대기 | node signing / local signing, type별 필드 전달 | 송금, 계약 배포·호출, fee delegation, load, faucet |
| FeePolicy | tx/receipt/header 읽기, 금액 비교 | 수수료 제안, 실제 지불액, local/remote 거부, replacement, baseFee 공식 | transaction-types, fee-policy, fee-delegation |
| AccountFixture | 계정 생성·자금 준비·nonce 격리 | StableNet blacklist/authorized 상태 조건 | 일반 송금과 실패 거래, 대납 거래 |
| ContractResolver | ABI 호출과 반환값 디코딩 | WEMIX Registry domain 조회, WBFT governance, StableNet system-contract 의미 | 일반 계약은 binding 공유, 시스템 계약은 전용 suite |
| Consensus / Lifecycle | 블록 진행·고정 높이 hash 관찰, process 제어 인터페이스 | WEMIX etcd/token, WBFT quorum/epoch/finality, bootstrap | 장애, 재시작, 동기화, 노드 참여 |
| CapabilityPreflight | chain/fork/version/RPC/계정 조건 검사 결과 | 각 체인의 기능 gate와 provider API 노출 | 7702/P256, 전용 RPC, WS, txpool, debug/admin |
| Test parameter 전달 | case ID와 공통 steps 보존 | env에 없는 step 상수·정책값 주입 | 하드코딩 fee, timeout, 예상 chainId, 주소 |

공통 sender는 새로운 서명 라이브러리를 중복 도입하는 작업이 아니다. 기존 Wallet과 sendTx를 재사용하되 contract creation, data, gas, fee, nonce를 모든 경로에서 전달하도록 정리한다. 현재 `sendAndConfirm`은 `eth_sendTransaction`을 사용하고, 로컬 서명 단순 송금 분기는 gas를 21000으로 고정한다. 이 상태에서 공개 RPC 주소와 계정 label만 바꾸면 계약 배포·부하 테스트가 공통으로 실행된다고 판단할 수 없다. 근거: `sources/chainbench/internal/testhelper/assets.go:210`, `sources/chainbench/internal/testhelper/builtins.go:315`.

## 체인별 판정에서 보존할 차이

| 항목 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| baseFee 변화 | governance의 target/change rate/max 값 | post-Croissant EIP-1559 계산. 동일한 고정 max clamp 없음 | Anzeon 증가/감소 threshold와 min/max |
| baseFee 하한 검사 | 정상 초기 조건의 PoA 감소 경로에서 1 | 계산 경로의 0 하한 | 프로필의 MinBaseFee |
| 일반 계정 수수료 | tx 입력과 유효 tip 정책 | 활성 fork별 정책 | 일반 계정 tip을 headerGasTip으로 대체할 수 있음 |
| 계정 상태 | 잔액, nonce, 서명 | 잔액, nonce, 서명 및 fork | 추가 blacklist/authorized 상태 |
| 대납 거래 | type 0x16 | type 0x16, Croissant 조건별 잔액 검사 | type 0x16, fee payer blacklist 등 |
| 7702 성공 시험 | 공통 성공 대상 아님 | Croissant gate | Anzeon gate |
| P256 성공 시험 | 공통 성공 대상 아님 | Croissant gate | Boho gate |
| 합의 / 운영 | PoA, etcd, mining token, WEMIX governance | WBFT 및 Croissant 전환 경로 | WBFT 및 StableNet system-contract 정책 |

이 표는 보존 코드의 정책이다. 현재 운영 메인넷의 배포 버전·포크 활성 상태는 확인하지 않았다. 세부 코드 근거는 CD-03~CD-11과 [반대 검토](counter-review.md)에 연결되어 있다. `RT-C-07 (분석 TC-128)`처럼 한 체인에 대응 기능이 없는 경우 임의 상한을 설정해서 공통 PASS로 만들지 않는다. `RT-C-06 (분석 TC-127)`의 프로필 하한 비교도 “세 체인 모두 양수 최소 수수료 설정을 강제한다”는 시험으로 확대하지 않는다.

## 실행 환경별 분리

| 실행 환경 | 공통으로 묶을 범위 | 필요한 전제 |
|---|---|---|
| 운영망 조회 | block/tx/receipt/chain 정보, 허용된 읽기 RPC | 알려진 fixture hash/height, provider 기능, 동일 높이 비교 |
| 전용 테스트 계정의 상태 변경 | 승인된 송금·계약 fixture·fee delegation | 자금·nonce 격리, 로컬 signer, fork와 계정 정책 |
| 소유한 격리 네트워크 | bootstrap, 장애, restart, partition, sync, 포크, 부하 | 바이너리·genesis·process controller·노드 역할 전체 제어 |

138개 공통 후보는 이식과 격리 환경을 포함한 후보 수다. 운영망에 그대로 실행할 테스트 138개라는 뜻은 아니다. 미지원 기능, provider 제한, 자금 부족, process 제어 부재는 구분된 SKIP/BLOCKED 사유로 기록하며 PASS로 합산하지 않는다. plain attach의 process 제어 한계는 `sources/chainbench/internal/testengine/attach.go:81`, 장애 대상 검증은 `sources/chainbench/internal/testhelper/fault.go:341`에 근거한다.

## 변경 작업 단위

| 순서 | 작업과 산출물 | 변경 범위 | 완료 기준 |
|---|---|---|---|
| 1 | 원본 ID와 공통 ID 대응 고정 | 테스트 목록·실행 결과 메타데이터 | 263개 원본이 보존되고 중복·파생·전용 관계 추적 가능 |
| 2 | 세 체인 환경 선언과 실행 조합 | 공통 case, 체인 env, applicableChains, requires, binary 선택 | 환경 충돌·SKIP을 숨기지 않고 선언한 노드와 계정이 생성됨 |
| 3 | 설정만 분리하는 42개 후보부터 추출 | 기존 EnvV2/label/binding 재사용, step literal 정리 | 같은 의미의 assertion을 유지하고 세 격리 프로필에서 실행 |
| 4 | 공통 sender 정리 | testhelper의 builtins/assets/load, 기존 Wallet 경계 | node/local 서명에서 type/data/creation/gas/fee/nonce 손실 없이 동일 결과 |
| 5 | 수수료·계정·계약 판정 분리 | FeePolicy/AccountFixture/ContractResolver와 해당 case | 세 체인별 정책을 기대값으로 사용하고 잘못된 거부·비용 판정을 방지 |
| 6 | 합의·동기화·장애 처리 분리 | 기존 chainsetup/consensus family/NodeControl, 관찰기 | PoA와 WBFT의 정상·장애 조건을 각각 만족. sync 경로를 실제로 관측 |
| 7 | 별도 처리 96개 후보와 전용 suite 정리 | 공통 steps + 체인별 판정, WBFT/StableNet 전용 suite | 공통 기능과 전용 기능을 혼합 집계하지 않음 |
| 8 | 실행 전 검사와 결과 기록 | capability, reporter, 실행 matrix | binary/source SHA, profile, fork, signer mode, assertion 결과, skip reason 보존 |

실제 구현에서는 관련 기존 파일의 변경을 우선 검토한다. 새 디렉터리·인터페이스 이름은 설계 예시이며 현재 코드에 추가하지 않았다. 포크·거버넌스 테스트를 공통 테스트로 만들기 위해 클라이언트 소스의 동작을 바꾸는 작업은 이 범위에 없다.

## 요청한 완료 조건의 대응

- 메인넷별 의존 요소 식별: 전체 catalog의 항목별 dependency 필드와 H01~H12.
- 설정 분리 항목: 위 설정 표 및 각 원본의 configuration_items.
- 별도 구현 항목: 위 처리 표, CD-01~CD-11 및 각 원본의 implementation_items.
- 공통 테스트 분리 변경 범위: 작업 단위 1~8과 실행 환경별 조건.

분석 산출물의 완료와 구현·실행 검증의 완료는 구분한다. 이 작업에서는 전자를 수행했다.
