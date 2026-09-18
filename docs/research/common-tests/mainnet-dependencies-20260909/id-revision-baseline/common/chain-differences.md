# 세 클라이언트 공통 테스트의 의미 차이

현재 보존한 세 프로젝트 소스의 정적 차이 분석. 실제 배포 상태 및 실행 결과 미검증.

WEMIX3.0=go-wemix, WEMIX4.0=go-wbft, StableNet=go-stablenet으로 사용자 경로를 대응했다. 아래는 소스 능력과 설정 조건이며 운영 메인넷 배포 확인이 아니다. 기본 거래도 이식·환경 준비 후 실행해야 한다. `build/{project}-darwin-arm64.json`의 GoFiles/CgoFiles와 대조하여 45개 인용 파일 위치가 모두 빌드 선택 파일에 포함됨을 확인했다. 바이너리 빌드·실행을 검증한 것은 아니다.

| ID | 분야 | 공통화 범위 | 별도 처리 |
|---|---|---|---|
| CD-01 | 네트워크 식별 | 실행 전 RPC chain ID와 선택 프로필 대조 | 체인 ID로 구현체를 자동 선택하지 말고 명시적 clientFamily와 사전 검사 결합 |
| CD-02 | 일반 거래 | Legacy/type1/type2 전송·조회·receipt·nonce·계약 생성/호출은 공통 시나리오 후보 | 공통 거래 인코더를 사용하되 capability 검증 후 실행; 통과 여부는 미실행 |
| CD-03 | Fee Delegation | 세 코드 모두 type22 fee payer 별도 서명 구조가 있어 공통 테스트 후보 | 서명·전송은 공통화 가능하나 payer 비용·잔액·오류 오라클은 fork/체인 정책별 구현 |
| CD-04 | EIP-7702 | 3종 전체 공통 성공 테스트에서 제외; capability 기반 WBFT/StableNet 하위 suite | type4 fixture/authorization nonce·signature·execution 오라클을 별도 suite로 유지 |
| CD-05 | P256 precompile | 3종 전체 공통 성공 테스트에서 제외; WBFT/StableNet도 gate 구분 | WBFT Croissant/StableNet Boho 활성 후 동일 입력 벡터 재사용; 활성 이전 기대 결과는 별도 |
| CD-06 | gas·fee 판정 | 유효 수수료로 전송하고 실제 receipt와 잔액 변화를 검사하는 목적은 공통 | FeePolicy 인터페이스로 가격 제안·상승하락·잔액 기대값 계산 분리. 모든 체인에 Ethereum 단일 target 식이나 고정 gwei를 적용하지 않음 |
| CD-07 | 테스트 계정 | sender/recipient/fee payer 역할과 자금 준비는 공통 | AccountFixture가 상태 사전 검사; blacklist/권한 정책 자체의 테스트는 StableNet 전용 |
| CD-08 | RPC | eth/net 기반 일반 조회·raw tx 전송을 공통 action으로 사용 | 합의 조회/관리 RPC만 체인 adapter; endpoint와 인증은 외부 프로필. 오류 문자열 완전일치 대신 의미와 실제 상태 검증 |
| CD-09 | 컨트랙트 주소·ABI | 일반 ERC20/저장소/이벤트 fixture는 suite가 배포한 주소를 후속 step에 전달하여 공통화 | 일반 FixtureDeployer 공통; GovernanceAdapter와 contract resolver는 체인별. ABI 버전·주소의 code hash 검증 |
| CD-10 | 합의·노드 운영 | 블록 진행, 고정 높이 해시 일치, 노드 재접속·동기화 결과 검증은 공통 목적 | NodeLifecycle·ConsensusAdapter 구현 분리; 재시작과 네트워크 단절은 격리된 제어 가능 환경 조건 |
| CD-11 | 포크·보상·시스템 변경 | 공통 suite는 동일 목적의 일반 거래 결과만 공유 | 보상/거버넌스 전용 suite를 별도 유지; 공통 실행 harness와 결과 스키마만 재사용 |

## CD-01 네트워크 식별

WEMIX3와 WEMIX4 소스 mainnet 설정은 모두 1111. StableNet 설정은 8282. 리포지토리 명칭을 WEMIX3/4에 대응한 분석이며 실제 배포 버전·활성 fork는 미확인. WEMIX4 Croissant 높이는 TODO가 남아 있어 운영 전환 완료로 해석할 수 없다.

설정 분리: `networkProfile`, `expectedChainId`, `expectedGenesisHash`, `clientVersion`, `forkOverrides`.

별도 처리: 체인 ID로 구현체를 자동 선택하지 말고 명시적 clientFamily와 사전 검사 결합.

근거:

- `sources/go-wemix/params/config.go:146` — `ChainID: big.NewInt(1111)`
- `sources/go-wbft/params/config.go:47` — `ChainID: big.NewInt(1111)`
- `sources/go-wbft/params/config.go:71` — `CroissantBlock: big.NewInt(200_000_000), // TODO`
- `sources/go-stablenet/params/config.go:45` — `ChainID: big.NewInt(8282)`

## CD-02 일반 거래

같은 거래 형식도 Berlin/London과 각 체인 fork 활성 및 계정 정책에 따라 접수 결과가 달라진다.

설정 분리: `supportedTxTypes`, `activeForks`, `fundingAmount`, `gasLimit`.

별도 처리: 공통 거래 인코더를 사용하되 capability 검증 후 실행; 통과 여부는 미실행.

근거:

- `sources/go-wemix/core/types/transaction.go:45` — `LegacyTxType = iota`
- `sources/go-wbft/core/types/transaction.go:50` — `DynamicFeeTxType = 0x02`
- `sources/go-stablenet/core/types/transaction.go:50` — `DynamicFeeTxType = 0x02`

## CD-03 Fee Delegation

WEMIX4 buyGas는 Croissant 전/후 fee payer 잔액 사전 검사에 분기. StableNet fee payer blacklist 정책 추가. 거래 타입 번호 일치만으로 오라클 동일 보장 안 됨.

설정 분리: `senderAccountRef`, `feePayerAccountRef`, `activeForks`, `feeCaps`.

별도 처리: 서명·전송은 공통화 가능하나 payer 비용·잔액·오류 오라클은 fork/체인 정책별 구현.

근거:

- `sources/go-wemix/core/types/transaction.go:48` — `FeeDelegateDynamicFeeTxType = 22`
- `sources/go-wbft/core/types/transaction.go:53` — `FeeDelegateDynamicFeeTxType = 0x16`
- `sources/go-stablenet/core/types/tx_fee_delegation.go:117` — `return FeeDelegateDynamicFeeTxType`
- `sources/go-wbft/core/state_transition.go:267` — `!st.evm.ChainConfig().CroissantEnabled()`
- `sources/go-stablenet/core/txpool/validation.go:284` — `opts.Config.AnzeonEnabled() && opts.State.IsBlacklisted(feePayer)`

## CD-04 EIP-7702

WEMIX3 typed decode는 type1/2/22이며 type4 처리 없음. WBFT txpool은 현재 블록 Croissant, StableNet은 Anzeon 설정 존재로 gate. Prague 필드 존재만 보고 활성 판정하면 오류.

설정 분리: `setCodeEnabled`, `authorizationChainId`, `authorityAccountRef`.

별도 처리: type4 fixture/authorization nonce·signature·execution 오라클을 별도 suite로 유지.

근거:

- `sources/go-wemix/core/types/transaction.go:189` — `case AccessListTxType:`
- `sources/go-wbft/core/txpool/validation.go:78` — `!opts.Config.IsCroissant(head.Number) && tx.Type() == types.SetCodeTxType`
- `sources/go-stablenet/core/txpool/validation.go:78` — `!opts.Config.AnzeonEnabled() && tx.Type() == types.SetCodeTxType`
- `sources/go-stablenet/params/config.go:1085` — `func (c *ChainConfig) AnzeonEnabled() bool`

## CD-05 P256 precompile

WBFT Croissant precompile map에 0x0100. StableNet Anzeon map에는 없고 Boho map에만 존재. 테스트용 exported P256 map의 존재는 활성 증거가 아니다.

설정 분리: `p256Enabled`, `precompileAddress`, `activeForks`.

별도 처리: WBFT Croissant/StableNet Boho 활성 후 동일 입력 벡터 재사용; 활성 이전 기대 결과는 별도.

근거:

- `sources/go-wbft/core/vm/contracts.go:127` — `var PrecompiledContractsCroissant`
- `sources/go-wbft/core/vm/contracts.go:140` — `common.BytesToAddress([]byte{0x1, 0x00}): &p256Verify{}`
- `sources/go-wbft/core/vm/contracts.go:182` — `case rules.IsCroissant:`
- `sources/go-stablenet/core/vm/contracts.go:127` — `var PrecompiledContractsAnzeon`
- `sources/go-stablenet/core/vm/contracts.go:141` — `var PrecompiledContractsBoho`
- `sources/go-stablenet/core/vm/contracts.go:200` — `case rules.IsBoho:`

## CD-06 gas·fee 판정

WEMIX3 governance 기반 gas target/change rate/max fee. WBFT Croissant 이전 전용 gas oracle, 이후 일반 샘플링 oracle. StableNet Anzeon base fee는 증가/감소 threshold 두 개와 min/max clamp; 일반 계정 tip은 headerGasTip으로 대체될 수 있음.

설정 분리: `feeMode`, `gasTip`, `gasFeeCap`, `thresholds`, `gasTarget`, `minBaseFee`, `maxBaseFee`.

별도 처리: FeePolicy 인터페이스로 가격 제안·상승하락·잔액 기대값 계산 분리. 모든 체인에 Ethereum 단일 target 식이나 고정 gwei를 적용하지 않음.

근거:

- `sources/go-wemix/consensus/misc/eip1559.go:81` — `wemixminer.GetBlockBuildParameters(parent.Number)`
- `sources/go-wbft/eth/gasprice/gasprice.go:155` — `!oracle.backend.ChainConfig().IsCroissant(head.Number)`
- `sources/go-stablenet/consensus/misc/eip1559/eip1559.go:62` — `if config.AnzeonEnabled()`
- `sources/go-stablenet/core/state_transition.go:169` — `!statedb.IsAuthorized(from)`

## CD-07 테스트 계정

StableNet은 sender/recipient/fee payer blacklist와 authorized 계정에 따른 tip/추가 로그 처리. 기본 전송 공통 fixture는 일반 비차단 계정이어야 한다. authorized는 EIP-7702 authorization과 다른 개념.

설정 분리: `accountRefs`, `funding`, `nonceIsolation`, `accountRoles`, `blacklistState`, `authorizedState`.

별도 처리: AccountFixture가 상태 사전 검사; blacklist/권한 정책 자체의 테스트는 StableNet 전용.

근거:

- `sources/go-stablenet/core/state_transition.go:506` — `if rules.IsAnzeon`
- `sources/go-stablenet/core/state_transition.go:592` — `rules.IsAnzeon && st.state.IsAuthorized(msg.From)`
- `sources/go-stablenet/core/txpool/validation.go:253` — `opts.State.IsBlacklisted(from)`

## CD-08 RPC

WEMIX3/4에는 wemix namespace 등록. WBFT/StableNet 합의 RPC는 istanbul. 서버 등록과 HTTP/WS 노출은 별개이므로 활성 모듈을 검사해야 함.

설정 분리: `rpcHttp`, `rpcWs`, `rpcAuthRef`, `rpcModules`, `rpcTimeout`, `nodeRole`.

별도 처리: 합의 조회/관리 RPC만 체인 adapter; endpoint와 인증은 외부 프로필. 오류 문자열 완전일치 대신 의미와 실제 상태 검증.

근거:

- `sources/go-wemix/eth/backend.go:297` — `Namespace: "wemix"`
- `sources/go-wbft/eth/backend.go:336` — `Namespace: "wemix"`
- `sources/go-stablenet/eth/backend.go:335` — `Namespace: "eth"`
- `sources/go-wbft/consensus/wbft/backend/engine.go:253` — `Namespace: "istanbul"`
- `sources/go-stablenet/consensus/wbft/backend/engine.go:234` — `Namespace: "istanbul"`

## CD-09 컨트랙트 주소·ABI

WEMIX3 Registry는 boot owner 기반 탐색 후 domain 조회. WBFT GovConfig/GovStaking/GovRewardeeImp/GovNCP, StableNet GovValidator/NativeCoinAdapter/GovMinter/GovMasterMinter/GovCouncil은 의미와 ABI가 다름. 주소 문자열 교체만으로 거버넌스 테스트 공통화 불가.

설정 분리: `deployedContracts`, `registryOwnerRef`, `systemContractAddresses`, `abiVersion`, `deploymentBlock`.

별도 처리: 일반 FixtureDeployer 공통; GovernanceAdapter와 contract resolver는 체인별. ABI 버전·주소의 code hash 검증.

근거:

- `sources/go-wemix/wemix/bind/structs.go:253` — `func GetRegistryByOwner`
- `sources/go-wemix/wemix/admin.go:335` — `contracts.Registry.GetContractAddress`
- `sources/go-wbft/params/config_wbft.go:183` — `type GovContracts struct`
- `sources/go-stablenet/params/config_wbft.go:146` — `type SystemContracts struct`

## CD-10 합의·노드 운영

WBFT는 Croissant 전환 engine 또는 WBFT engine 선택. StableNet Anzeon은 WBFT backend. WEMIX3 token/거버넌스와 WBFT quorum/epoch/validator 조작은 서로 다른 절차. 장애 허용 노드 수나 동일 타임아웃을 고정하지 않는다.

설정 분리: `clientBinary`, `nodeCount`, `nodeRoles`, `dataDirs`, `p2pPorts`, `blockPeriod`, `timeouts`, `consensusProfile`.

별도 처리: NodeLifecycle·ConsensusAdapter 구현 분리; 재시작과 네트워크 단절은 격리된 제어 가능 환경 조건.

근거:

- `sources/go-wbft/eth/ethconfig/config.go:195` — `if config.CroissantEnabled()`
- `sources/go-wbft/eth/ethconfig/config.go:204` — `wemix.NewCroissantEngine`
- `sources/go-stablenet/eth/ethconfig/config.go:192` — `if config.AnzeonEnabled()`
- `sources/go-stablenet/consensus/wbft/backend/api.go:132` — `func (api *API) GetValidators`

## CD-11 포크·보상·시스템 변경

Brioche reward와 Croissant migration, StableNet system contract upgrade는 서로 같은 기능이 아님. 보상 수신자와 계산식을 일반 전송 오라클에 섞지 않는다.

설정 분리: `forkHeights`, `systemContractVersions`, `rewardRecipients`.

별도 처리: 보상/거버넌스 전용 suite를 별도 유지; 공통 실행 harness와 결과 스키마만 재사용.

근거:

- `sources/go-wemix/params/config.go:162` — `BriocheBlock:`
- `sources/go-wbft/params/config_wbft.go:229` — `BlockReward`
- `sources/go-stablenet/eth/ethconfig/config.go:265` — `wbftCfg.SystemContractUpgrades = append`

공통 테스트 완료 판정은 입력/전송/검증의 세 단계가 모든 대상 프로필에서 연결되는지로 나눈다. capability 없음은 SKIP 사유로 기록하며 공통 성공 테스트 수에 포함하지 않는다. 메인넷별 API 응답·합의 구성·활성 fork를 확인하지 않은 상태에서 PASS로 판정하지 않는다.
