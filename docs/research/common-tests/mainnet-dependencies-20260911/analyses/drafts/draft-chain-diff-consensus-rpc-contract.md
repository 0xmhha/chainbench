# 세 클라이언트 차이 초안: 합의, RPC, 헤더, 시스템 계약, 포크, 보상, genesis, 플래그, 동기화

2026-09-11 현재 작업 트리를 다시 읽고 쓴 초안이다. 이전 분석(2026-09-09)의 결론과 줄번호를 가져오지 않았다. 인용은 `go list`로 고른 빌드 선택 파일 기준이며, 예외는 각 항목에 적었다. 실행 결과가 아니라 정적 분석이다.

| ID | 영역 | 한 줄 요약 |
|---|---|---|
| CD-B-01 | 합의 엔진 | WEMIX3.0은 etcd 토큰 PoA, WEMIX4.0은 PoA→WBFT 전환 엔진, StableNet은 WBFT 단일. WBFT 규칙은 두 체인이 같다. |
| CD-B-02 | RPC | istanbul namespace는 WBFT 두 체인만. wemix namespace는 Brioche 설정이 있어야 열리고 메서드 3개뿐. WEMIX3.0 상태는 admin_wemixInfo. |
| CD-B-03 | 헤더 extra | WBFTExtra는 같은 뼈대이나 StableNet에 GasTip 필드가 끼어들고 epochInfo 키가 stakers/candidates로 다르다. |
| CD-B-04 | 시스템 계약 | WEMIX3.0 Registry 조회, WEMIX4.0 Gov* 0x1000~0x1003, StableNet 시스템 계약 0x1000~0x1004. 일반 계약만 공통. |
| CD-B-05 | 하드포크 | WEMIX3.0 Croissant는 중단 조건. SetCode/P256은 WEMIX4.0 Croissant, StableNet Anzeon/Boho에서만. baseFee 공식 3종. |
| CD-B-06 | 블록 보상 | Brioche 보상은 WEMIX3.0+4.0. StableNet은 보상 없이 baseFee 배분. getBriocheBlockReward는 Brioche 설정 필요. |
| CD-B-07 | genesis | WBFT 두 체인은 템플릿+자리표, WEMIX3.0은 바이너리 생성. croissant/anzeon 섹션 필수. |
| CD-B-08 | 노드 플래그 | 플래그 이름은 거의 같다. 기본 syncmode가 WEMIX3.0 snap, 나머지 full. light는 WEMIX3.0만. |
| CD-B-09 | 동기화 | downloader/fetcher/snap 모듈은 셋 다 있다. WBFT 두 체인은 TD 보정 규칙, WEMIX3.0은 etcd syncCheck. |

## CD-B-01 합의 엔진과 블록 생산 규칙

**같은 점.** go-wbft와 go-stablenet은 같은 WBFT 코어를 쓴다. 기본 블록 주기 1초, epoch 10, RoundRobin, quorum = ceil(N - (N-1)/3)이 같다. RandaoReveal과 MixDigest 계산도 같다.

| 구분 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 다른 점 | 합의 engine 자리는 ethash faker를 beacon으로 감싼 것이다. 실제 블록 생산 권한은 wemix 패키지가 etcd 잠금(mining token)으로 결정한다. 블록 간격, gas limit, baseFee 상한은 governance 계약(EnvStorage)에서 매 블록 읽는다. 기본값은 blockInterval 15(단위 ms/1000 → 초). | CroissantEnabled면 WBFT다. CroissantBlock이 1 이상이면 Croissant 이전 블록을 위해 WemixPoA(wpoa)를 legacy로 감싼 전환 엔진을 만든다. CroissantBlock 0이면 순수 WBFT backend다. 검증자 집합은 GovStaking의 staker 목록에서 epoch마다 다시 만든다. | AnzeonEnabled(anzeon 섹션 존재)면 WBFT backend다. 전환 엔진이 없다. 검증자 후보는 GovValidator 계약에서 읽는다. |

설정으로 둘 값:

- 블록 주기(초), epoch 길이, requestTimeout, proposerPolicy: genesis의 croissant.wBFT / anzeon.wbft 값
- WEMIX3.0의 블록 간격, gasLimit, maxBaseFee: governance 배포 시 env 값
- 노드 수와 quorum 기대값(WBFT는 ceil(2N/3) 공식으로 계산)

체인별 구현이 필요한 부분:

- WEMIX3.0의 "합의 정상" 판정은 etcd 리더/토큰 상태와 governance 배포 여부를 봐야 하므로 별도 어댑터가 필요하다
- 장애 허용 수 계산: WBFT는 quorum 공식, WEMIX3.0은 etcd 과반 + 미이너 목록으로 다르다
- proposer 순환·commit seal 검증은 WBFT 두 체인에만 적용한다

근거:

- `go-wemix/eth/ethconfig/config.go:231` — `engine = ethash.New(ethash.Config{`
- `go-wemix/eth/ethconfig/config.go:245` — `return beacon.New(engine)`
- `go-wemix/wemix/miner/miner.go:57` — `func AcquireMiningToken(height *big.Int, parentHash common.Hash) (bool, error) {`
- `go-wemix/wemix/sync.go:145` — `if admin == nil || !admin.etcdIsRunning() {`
- `go-wemix/wemix/admin.go:1121` — `func getBlockBuildParameters(height *big.Int) (blockInterval int64, maxBaseFee, gasLimit *big.Int, baseFeeMaxChangeRate, gasTargetPercentage int64, err error) {`
- `go-wemix/wemix/admin.go:1139` — `blockInterval = 15`
- `go-wemix/miner/worker.go:1310` — `blockInterval, _, blockGasLimit, baseFeeMaxChangeRate, gasTargetPercentage, _ := wemixminer.GetBlockBuildParameters(parent.Number())`
- `go-wemix/consensus/ethash/consensus.go:327` — `if !wemixminer.IsPoW() && !wemixminer.VerifyBlockSig(header.Number, header.Coinbase, header.MinerNodeId, header.Root, header.MinerNodeSig, chain.Config().IsPangyo(header.Number)) {`
- `go-wbft/eth/ethconfig/config.go:195` — `if config.CroissantEnabled() {`
- `go-wbft/eth/ethconfig/config.go:204` — `return wemix.NewCroissantEngine(wpoa.NewWemixPoAEngine(govCli), wbftCfg, privKey, db), nil`
- `go-wbft/eth/ethconfig/config.go:206` — `return wbftBackend.New(wbftCfg, privKey, db), nil`
- `go-wbft/consensus/wemix/consensus.go:76` — `if chain.Config().IsCroissant(header.Number) {`
- `go-wbft/consensus/wbft/engine/engine.go:571` — `govStakingAddress := govContracts.GovStaking.Address`
- `go-stablenet/eth/ethconfig/config.go:192` — `if config.AnzeonEnabled() {`
- `go-stablenet/eth/ethconfig/config.go:200` — `return wbftBackend.New(wbftCfg, privKey, db), nil`
- `go-stablenet/consensus/wbft/engine/engine.go:609` — `govValidatorAddress := systemContracts.GovValidator.Address`
- `go-wbft/consensus/wbft/config.go:124` — `BlockPeriod:                 1,`
- `go-wbft/consensus/wbft/config.go:126` — `Epoch:                       10,`
- `go-stablenet/consensus/wbft/config.go:118` — `BlockPeriod:            1,`
- `go-stablenet/consensus/wbft/config.go:120` — `Epoch:                  10,`
- `go-wbft/consensus/wbft/validator/default.go:222` — `func (valSet *defaultSet) F() float64 { return float64(valSet.Size()-1) / 3 }`
- `go-wbft/consensus/wbft/validator/default.go:228` — `return int(math.Ceil(float64(valSet.Size()) - valSet.F()))`
- `go-stablenet/consensus/wbft/validator/default.go:228` — `return int(math.Ceil(float64(valSet.Size()) - valSet.F()))`
- `go-wbft/consensus/wbft/backend/engine.go:401` — `header.MixDigest = wbftengine.CalculateRandaoMix(parent.MixDigest, extra.RandaoReveal)`
- `go-stablenet/consensus/wbft/backend/engine.go:396` — `header.MixDigest = wbftengine.CalculateRandaoMix(parent.MixDigest, extra.RandaoReveal)`

## CD-B-02 합의 RPC namespace와 메서드

**같은 점.** eth, net, web3, txpool, admin, debug, miner, personal namespace는 세 클라이언트 모두 등록한다. eth_signRawFeeDelegateTransaction도 세 곳에 있다. HTTP/WS 기본 노출 모듈은 net, web3뿐이라 나머지는 --http.api/--ws.api로 열어야 한다.

| 구분 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 다른 점 | istanbul namespace가 없다. wemix namespace는 Brioche 설정이 genesis에 있을 때만 등록되며 메서드는 briocheConfig, halvingSchedule, getBriocheBlockReward 세 개뿐이다. 노드 상태와 governance 주소는 admin_wemixInfo로 읽는다. etcd 관리는 admin_etcd* 메서드다. | istanbul namespace는 전환 엔진이 WBFT backend의 API를 그대로 노출한다. Croissant 이전 블록 번호로 getWbftExtraInfo를 부르면 ErrIsNotWBFTBlock이다. wemix namespace(Brioche 3개 메서드)도 Brioche 설정이 있을 때 등록된다. | istanbul namespace만 있고 wemix namespace는 없다. getWbftExtraInfo 응답에 gasTip 키가 추가된다. |

설정으로 둘 값:

- 노드별 --http.api/--ws.api 모듈 목록(istanbul, txpool, admin 포함 여부)
- 체인별 합의 namespace 이름과 검증자 조회 방법: WBFT 두 체인은 istanbul_getValidators, WEMIX3.0은 admin_wemixInfo + governance eth_call

체인별 구현이 필요한 부분:

- WEMIX3.0 검증자·합의 상태 조회는 RPC 한 번으로 끝나지 않는다. chainbench poa/validators.go처럼 admin_wemixInfo → 계약 eth_call 경로가 필요하다
- chainbench wemix manifest의 wemix_getValidators, wemix_getReward는 go-wemix에 없는 메서드다. manifest 값과 실제 조회 경로가 다르므로 정리가 필요하다
- istanbul_* 검사는 WBFT 두 체인 공통 suite로 두고 WEMIX3.0은 SKIP 사유를 기록한다

근거:

- `go-wemix/eth/backend.go:295` — `if brioche := s.blockchain.Config().Brioche; brioche != nil {`
- `go-wemix/eth/backend.go:297` — `Namespace: "wemix",`
- `go-wemix/eth/api.go:706` — `func (api *PublicWemixAPI) BriocheConfig() BriocheConfigResult {`
- `go-wemix/eth/api.go:748` — `func (api *PublicWemixAPI) GetBriocheBlockReward(blockNumber rpc.BlockNumber) *hexutil.Big {`
- `go-wemix/eth/api.go:275` — `func (api *PrivateAdminAPI) RequestMinerStatus(id enode.ID) error {`
- `go-wemix/eth/api.go:285` — `func (api *PrivateAdminAPI) EtcdInit() error {`
- `go-wemix/internal/web3ext/web3ext.go:269` — `getter: 'admin_wemixInfo'`
- `go-wemix/internal/ethapi/api.go:2404` — `func (s *PublicTransactionPoolAPI) SignRawFeeDelegateTransaction(ctx context.Context, args TransactionArgs, input hexutil.Bytes) (*SignTransactionResult, error) {`
- `go-wemix/internal/ethapi/backend.go:117` — `Namespace: "txpool",`
- `go-wemix/node/defaults.go:56` — `HTTPModules:         []string{"net", "web3"},`
- `go-wbft/eth/backend.go:334` — `if brioche := s.blockchain.Config().Brioche; brioche != nil {`
- `go-wbft/eth/backend.go:336` — `Namespace: "wemix",`
- `go-wbft/eth/api_wemix.go:95` — `func (api *PublicWemixAPI) GetBriocheBlockReward(blockNumber rpc.BlockNumber) *hexutil.Big {`
- `go-wbft/consensus/wbft/backend/engine.go:253` — `Namespace: "istanbul",`
- `go-wbft/consensus/wemix/consensus.go:189` — `return we.wbft.APIs(chain)`
- `go-wbft/consensus/wbft/backend/api.go:80` — `func (api *API) NodeAddress() common.Address {`
- `go-wbft/consensus/wbft/backend/api.go:86` — `func (api *API) GetCommitSignersFromBlock(number *rpc.BlockNumber) (*BlockSigners, error) {`
- `go-wbft/consensus/wbft/backend/api.go:132` — `func (api *API) GetValidators(number *rpc.BlockNumber) ([]common.Address, error) {`
- `go-wbft/consensus/wbft/backend/api.go:165` — `func (api *API) Status(startBlockNum *rpc.BlockNumber, endBlockNum *rpc.BlockNumber) (*Status, error) {`
- `go-wbft/consensus/wbft/backend/api.go:347` — `func (api *API) IsValidator(blockNum *rpc.BlockNumber) (bool, error) {`
- `go-wbft/consensus/wbft/backend/api.go:420` — `if !api.chain.Config().IsCroissant(bNumber) {`
- `go-wbft/internal/ethapi/api.go:2188` — `func (s *TransactionAPI) SignRawFeeDelegateTransaction(ctx context.Context, args TransactionArgs, input hexutil.Bytes) (*SignTransactionResult, error) {`
- `go-wbft/internal/ethapi/backend.go:115` — `Namespace: "txpool",`
- `go-wbft/node/defaults.go:62` — `HTTPModules:          []string{"net", "web3"},`
- `go-stablenet/eth/backend.go:335` — `Namespace: "eth",`
- `go-stablenet/consensus/wbft/backend/engine.go:234` — `Namespace: "istanbul",`
- `go-stablenet/consensus/wbft/backend/api.go:416` — `func (api *API) GetWbftExtraInfo(number rpc.BlockNumber) (map[string]interface{}, error) {`
- `go-stablenet/consensus/wbft/backend/api.go:419` — `if !api.chain.Config().AnzeonEnabled() {`
- `go-stablenet/internal/ethapi/api.go:2207` — `func (s *TransactionAPI) SignRawFeeDelegateTransaction(ctx context.Context, args TransactionArgs, input hexutil.Bytes) (*SignTransactionResult, error) {`
- `go-stablenet/internal/ethapi/backend.go:115` — `Namespace: "txpool",`
- `chainbench/internal/chains/wemix/manifest.json:17` — `"validators_method": "wemix_getValidators"`
- `chainbench/internal/chains/wemix/manifest.json:20` — `"probe": { "method": "wemix_getReward" },`
- `chainbench/internal/consensus/poa/validators.go:15` — `deploys, not in a JSON-RPC method — there is no <ns>_getValidators. So the`
- `chainbench/internal/consensus/poa/validators.go:116` — `if err := c.Call(ctx, "admin_wemixInfo", &info); err != nil {`

## CD-B-03 블록 헤더 WBFTExtra 필드

**같은 점.** go-wbft와 go-stablenet의 WBFTExtra는 VanityData, RandaoReveal, PrevRound, PrevPreparedSeal, PrevCommittedSeal, Round, PreparedSeal, CommittedSeal, EpochInfo를 같은 순서로 갖는다. istanbul_getWbftExtraInfo가 같은 키 이름으로 돌려준다.

| 구분 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 다른 점 | WBFTExtra가 없다. extraData는 genesis에서 허용 bootnode id를 담고, 헤더에는 MinerNodeId/MinerNodeSig 필드로 서명자를 담는다. | EpochInfo는 Stakers(diligence)와 Stabilizing 플래그를 갖는다. JSON 키는 stabilizing, stakers, validators다. gasTip 키가 없다. | GasTip 필드가 CommittedSeal 뒤, EpochInfo 앞에 추가되어 RLP 배치가 다르다. Header.GasTip()으로 읽는다. EpochInfo는 Candidates이며 JSON 키는 candidates, validators다. |

설정으로 둘 값:

- 기대하는 epoch 길이와 epochInfo가 채워지는 블록 번호(epoch 마지막 블록)
- seal quorum 기대값(검증자 수에서 계산)

체인별 구현이 필요한 부분:

- extra 디코더는 두 RLP 배치를 모두 알아야 한다. GasTip 유무로 필드 위치가 달라진다
- epochInfo 검사는 stakers/stabilizing(WEMIX4.0)과 candidates(StableNet) 키를 체인별로 고른다
- WEMIX3.0에는 적용하지 않는다. 대신 MinerNodeSig 검증(Pangyo 이후 규칙)을 별도 검사로 둔다

근거:

- `go-wbft/core/types/istanbul.go:81` — `type WBFTExtra struct {`
- `go-wbft/core/types/istanbul.go:90` — `EpochInfo         *EpochInfo // epoch info is filled only for last block of epoch`
- `go-wbft/core/types/istanbul.go:99` — `Stakers       []*Staker // staker list for next epoch (staker index may be changed for each epoch)`
- `go-wbft/core/types/istanbul.go:102` — `Stabilizing   bool      // initial epochs are stabilizing epochs, which means that the stakers are less than `stabilizingStakersThreshold``
- `go-wbft/consensus/wbft/backend/api.go:411` — `"stabilizing": epoch.Stabilizing,`
- `go-stablenet/core/types/istanbul.go:90` — `GasTip            *big.Int   // tip value agreed through governance voting (in Wei)`
- `go-stablenet/core/types/istanbul.go:100` — `Candidates    []*Candidate // candidate list for next epoch (candidate index may be changed for each epoch)`
- `go-stablenet/core/types/block.go:111` — `func (h *Header) GasTip() *big.Int {`
- `go-stablenet/consensus/wbft/backend/api.go:447` — `"gasTip":            extra.GasTip.String(),`
- `go-stablenet/consensus/wbft/backend/api.go:411` — `"candidates": candidates,`
- `go-wemix/wemix/admin.go:170` — `//  1. extradata of genesis block, which is the id of the node that is allowed`
- `go-wemix/consensus/ethash/consensus.go:327` — `if !wemixminer.IsPoW() && !wemixminer.VerifyBlockSig(header.Number, header.Coinbase, header.MinerNodeId, header.Root, header.MinerNodeSig, chain.Config().IsPangyo(header.Number)) {`

## CD-B-04 시스템·거버넌스 계약과 일반 계약의 구분

**같은 점.** 세 체인 모두 임의 EVM 바이트코드를 배포·호출할 수 있다. 일반 계약은 배포 결과 주소를 binding하면 공통이다. 시스템 계약은 주소·ABI·권한·상태 의미가 체인마다 다르다.

| 구분 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 다른 점 | 고정 주소가 없다. Registry 계약을 boot owner에서 찾은 뒤 domain 이름(StakingReward, Ecosystem, Maintenance, FeeCollector, Gov, EnvStorage 등)으로 주소를 조회한다. 노드가 매 블록 governance 계약을 읽어 블록 파라미터와 보상 배분을 정한다. | Croissant genesis가 GovConfig 0x1000, GovStaking 0x1001, GovRewardeeImp 0x1002, GovNCP 0x1003을 alloc에 주입한다. GovStaking의 staker가 검증자 후보다. Upgrade 배열로 블록 높이별 코드 교체를 선언한다. | Anzeon genesis가 NativeCoinAdapter 0x1000, GovValidator 0x1001, GovMasterMinter 0x1002, GovMinter 0x1003, GovCouncil 0x1004를 주입한다. 계정별 blacklist/authorized 상태가 StateDB에 있고 tx 검증에 쓰인다. 하드포크 upgrade 오버레이가 genesis(블록 0)와 런타임 finalize 양쪽에서 적용된다. |

설정으로 둘 값:

- 일반 계약: bytecode fixture, 배포 계정, 배포 후 주소 binding
- 시스템 계약 주소 표: 체인별 genesis config(croissant.govContracts / anzeon.systemContracts)와 WEMIX3.0 Registry domain 목록
- 계약 version 문자열(v1 등)과 upgrade 블록

체인별 구현이 필요한 부분:

- 시스템 계약 호출은 체인별 ABI 어댑터로 분리한다. 주소만 바꿔서 같은 calldata를 쓰면 안 된다
- WEMIX3.0은 주소 조회 자체가 Registry eth_call 2단계라 resolver가 필요하다
- StableNet 계정 정책(blacklist/authorized)은 일반 송금 fixture 준비 단계에서 사전 검사로 넣는다

근거:

- `go-wemix/wemix/bind/structs.go:253` — `func GetRegistryByOwner(opts *bind.CallOpts, backend bind.ContractBackend, owner common.Address) (common.Address, *Registry, error) {`
- `go-wemix/wemix/admin.go:196` — `func (ma *wemixAdmin) getRegGovEnvContracts(ctx context.Context, height *big.Int) (*gov.GovContracts, error) {`
- `go-wemix/wemix/admin.go:335` — `staker, err := contracts.Registry.GetContractAddress(opts, metclient.ToBytes32(gov.DOMAIN_StakingReward))`
- `go-wbft/params/config_wbft.go:33` — `DefaultGovConfigAddress      = common.HexToAddress("0x1000")`
- `go-wbft/params/config_wbft.go:36` — `DefaultGovNCPAddress         = common.HexToAddress("0x1003")`
- `go-wbft/params/config_wbft.go:183` — `type GovContracts struct {`
- `go-wbft/params/config_wbft.go:213` — `type Upgrade struct {`
- `go-wbft/core/genesis.go:718` — `func InjectContracts(genesis *Genesis, config *params.ChainConfig) error {`
- `go-wbft/core/genesis.go:719` — `transition, err := govwbft.GetGovContractsTransition(config.Croissant.GovContracts)`
- `go-wbft/consensus/wbft/engine/engine.go:565` — `func (e *Engine) GetStakers(config *params.ChainConfig, latestEpochInfo *types.EpochInfo, state govwbft.StateReader, num *big.Int) ([]common.Address, bool) {`
- `go-stablenet/params/config_wbft.go:32` — `DefaultNativeCoinAdapterAddress = common.HexToAddress("0x1000")`
- `go-stablenet/params/config_wbft.go:44` — `DefaultGovCouncilAddress = common.HexToAddress("0x1004")`
- `go-stablenet/params/config_wbft.go:146` — `type SystemContracts struct {`
- `go-stablenet/params/config_wbft.go:174` — `type Upgrade struct {`
- `go-stablenet/core/genesis.go:735` — `func InjectContracts(genesis *Genesis, config *params.ChainConfig) error {`
- `go-stablenet/core/genesis.go:743` — `transition, err := systemcontracts.GetSystemContractsTransition(config.Anzeon.SystemContracts, &genesis.Alloc)`
- `go-stablenet/core/genesis.go:760` — `for _, upgrade := range config.CollectUpgrades() {`
- `go-stablenet/consensus/wbft/engine/engine.go:607` — `func (e *Engine) GetGovCandidates(config *params.ChainConfig, state systemcontracts.StateReader, num *big.Int) []common.Address {`
- `go-stablenet/core/state/statedb.go:311` — `func (s *StateDB) IsBlacklisted(addr common.Address) bool {`
- `go-stablenet/core/state/statedb.go:321` — `func (s *StateDB) IsAuthorized(addr common.Address) bool {`

## CD-B-05 하드포크 이름과 각 포크가 여는 기능

**같은 점.** Homestead~London 공통 포크 필드는 같다. Applepie 필드는 세 곳에 있다. 세 코드 모두 Rules에 자기 포크 플래그를 넣어 EVM/precompile/txpool을 gate한다.

| 구분 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 다른 점 | Pangyo, Applepie, Brioche, Croissant 블록 필드가 있다. Croissant는 지원이 아니라 중단 조건이다. Croissant 이후 높이는 채굴을 건너뛰고 검증도 거부한다. Shanghai/Cancun/Prague time 필드가 없다. SetCode(0x04)와 P256 precompile이 없다. precompile은 Berlin 집합이 최신이다. baseFee는 governance 값(maxBaseFee, 변화율, gas target)으로 계산한다. | Pangyo, Applepie, Brioche, Croissant 블록과 Shanghai/Cancun/Prague time이 있다. Croissant 블록 기준(IsCroissant(num))으로 WBFT 전환, SetCode tx 허용, P256(0x100) precompile이 열린다. baseFee는 London 표준식이며 Croissant gate가 없다. mainnet 설정의 CroissantBlock은 200_000_000 TODO다. | Applepie, Boho 블록과 Anzeon/Boho 섹션이 있다. Anzeon은 블록 번호가 아니라 섹션 존재(AnzeonEnabled)로 켜진다. Anzeon이 SetCode tx, blacklist/authorized 계정 정책, threshold 기반 baseFee(min/max clamp)를 연다. P256은 Boho precompile 집합에만 있다. mainnet 설정은 BohoBlock 0, testnet은 14408500이다. |

설정으로 둘 값:

- env.hardforks의 포크 높이(pangyo/applepie/brioche/croissant vs boho)
- anzeon/boho 섹션 존재 여부와 Anzeon 파라미터(threshold, minBaseFee, maxBaseFee)
- 기대 tx type 목록과 precompile 주소 목록은 체인·포크별 표에서 읽는다

체인별 구현이 필요한 부분:

- SetCode(0x04)와 P256 검사는 WEMIX3.0에서 SKIP이고, WEMIX4.0은 Croissant 이후, StableNet은 Anzeon(0x04)/Boho(P256) 이후에만 성공을 기대한다
- baseFee 기대값은 세 공식(governance 파라미터 / London 표준 / Anzeon threshold+clamp)을 체인별 oracle로 계산한다
- WEMIX3.0 Croissant 높이는 "여기서 멈춘다"는 뜻이므로 upgrade/handoff 테스트 외에는 설정하지 않는다

근거:

- `go-wemix/params/config.go:414` — `PangyoBlock         *big.Int `json:"pangyoBlock,omitempty"`         // Pangyo switch block (nil = no fork, 0 = already on pangyo)`
- `go-wemix/params/config.go:417` — `CroissantBlock      *big.Int `json:"croissantBlock,omitempty"`      // Croissant switch block (nil = no fork, 0 = already on croissant)`
- `go-wemix/params/config.go:841` — `IsPangyo, IsApplepie, IsBrioche, IsCroissant            bool`
- `go-wemix/miner/worker.go:1601` — `if w.chain.Config().IsCroissant(height) {`
- `go-wemix/consensus/ethash/consensus.go:324` — `return fmt.Errorf("go-wemix does not support blocks after Croissant hard fork")`
- `go-wemix/core/types/transaction.go:48` — `FeeDelegateDynamicFeeTxType = 22 // fee delegation`
- `go-wemix/core/vm/contracts.go:136` — `case rules.IsBerlin:`
- `go-wemix/consensus/misc/eip1559.go:81` — `_, maxBaseFeeGov, _, baseFeeMaxChangeRate, gasTargetPercentage, err := wemixminer.GetBlockBuildParameters(parent.Number)`
- `go-wbft/params/config.go:712` — `CroissantBlock      *big.Int `json:"croissantBlock,omitempty"`      // Croissant switch block (nil = no fork, 0 = already on Croissant)`
- `go-wbft/params/config.go:735` — `Croissant   *CroissantConfig `json:"croissant,omitempty"``
- `go-wbft/params/config.go:1058` — `func (c *ChainConfig) IsCroissant(num *big.Int) bool {`
- `go-wbft/params/config.go:1062` — `func (c *ChainConfig) CroissantEnabled() bool {`
- `go-wbft/params/config.go:71` — `CroissantBlock:      big.NewInt(200_000_000), // TODO: decide the block number`
- `go-wbft/core/txpool/validation.go:78` — `if !opts.Config.IsCroissant(head.Number) && tx.Type() == types.SetCodeTxType {`
- `go-wbft/core/vm/contracts.go:140` — `common.BytesToAddress([]byte{0x1, 0x00}): &p256Verify{},`
- `go-wbft/core/vm/contracts.go:182` — `case rules.IsCroissant:`
- `go-wbft/consensus/misc/eip1559/eip1559.go:59` — `if !config.IsLondon(parent.Number) {`
- `go-stablenet/params/config.go:809` — `BohoBlock           *big.Int `json:"bohoBlock,omitempty"`           // Boho switch block (nil = no fork, 0 = already on Boho)`
- `go-stablenet/params/config.go:831` — `Anzeon      *AnzeonConfig `json:"anzeon,omitempty"``
- `go-stablenet/params/config.go:1085` — `func (c *ChainConfig) AnzeonEnabled() bool {`
- `go-stablenet/params/config.go:1043` — `func (c *ChainConfig) IsBoho(num *big.Int) bool {`
- `go-stablenet/params/config.go:65` — `BohoBlock:           big.NewInt(0),`
- `go-stablenet/params/config.go:169` — `BohoBlock:           big.NewInt(14408500),`
- `go-stablenet/core/txpool/validation.go:78` — `if !opts.Config.AnzeonEnabled() && tx.Type() == types.SetCodeTxType {`
- `go-stablenet/core/txpool/validation.go:253` — `if opts.State.IsBlacklisted(from) {`
- `go-stablenet/core/state_transition.go:592` — `if rules.IsAnzeon && st.state.IsAuthorized(msg.From) {`
- `go-stablenet/core/vm/contracts.go:154` — `common.BytesToAddress([]byte{0x1, 0x00}): &p256Verify{},`
- `go-stablenet/core/vm/contracts.go:200` — `case rules.IsBoho:`
- `go-stablenet/core/vm/contracts.go:202` — `case rules.IsAnzeon:`
- `go-stablenet/consensus/misc/eip1559/eip1559.go:62` — `if config.AnzeonEnabled() {`
- `go-stablenet/consensus/misc/eip1559/eip1559.go:81` — `minBaseFee := config.MinBaseFee()`

## CD-B-06 블록 보상과 wemix_getBriocheBlockReward

**같은 점.** go-wemix와 go-wbft는 같은 BriocheConfig(blockReward, halving)와 같은 GetBriocheBlockReward 계산을 갖는다. 두 곳 모두 Brioche 설정이 있을 때만 wemix namespace를 연다.

| 구분 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 다른 점 | ethash Finalize가 wemixminer.CalculateRewards를 불러 governance의 배분 비율(staker, ecosystem, maintenance, feecollector)대로 나눈다. Brioche 이후는 BriocheConfig 값이 RewardAmount를 대체한다. | Croissant 이전 블록은 wpoa가 같은 WEMIX3.0 규칙으로 보상한다. Croissant 이후 WBFT 엔진은 Brioche 값 또는 wBFT.blockReward를 blockRewardBeneficiary 비율로 나눈다. | 블록 보상 자체가 없다. wBFT 설정에 blockReward 필드가 없고 Brioche 코드도 없다. London 이후 baseFee를 이전 epoch 검증자 집합에 배분한다(distributeBaseFee). wemix namespace가 없어 getBriocheBlockReward는 method not found다. |

설정으로 둘 값:

- Brioche 파라미터(blockReward, firstHalvingBlock, halvingPeriod, halvingTimes, halvingRate)
- WEMIX4.0 wBFT.blockReward와 blockRewardBeneficiary 비율
- WEMIX3.0 governance 배분 주소(staker/ecosystem/maintenance/feecollector)

체인별 구현이 필요한 부분:

- 보상 검증은 WEMIX3.0+WEMIX4.0 전용 suite다. StableNet은 baseFee 배분이라는 다른 검사가 필요하다
- 수령자와 금액 계산은 체인별 oracle로 두고 일반 송금 잔액 검사와 섞지 않는다

근거:

- `go-wemix/params/config.go:443` — `func (bc *BriocheConfig) GetBriocheBlockReward(defaultReward *big.Int, num *big.Int) *big.Int {`
- `go-wemix/consensus/ethash/consensus.go:705` — `rewards, err := wemixminer.CalculateRewards(`
- `go-wemix/wemix/admin.go:893` — `if config.IsBrioche(num) {`
- `go-wemix/wemix/admin.go:894` — `blockReward = config.Brioche.GetBriocheBlockReward(defaultBriocheBlockReward, num)`
- `go-wbft/params/config.go:750` — `func (bc *BriocheConfig) GetBriocheBlockReward(defaultReward *big.Int, num *big.Int) *big.Int {`
- `go-wbft/consensus/wpoa/consensus.go:412` — `func (wpoa *WemixPoA) accumulateRewards(config *params.ChainConfig, stateDB *state.StateDB, header *types.Header, uncles []*types.Header) {`
- `go-wbft/consensus/wpoa/consensus.go:459` — `blockReward = config.Brioche.GetBriocheBlockReward(params.DefaultBriocheBlockReward, num)`
- `go-wbft/consensus/wbft/engine/engine.go:1097` — `if chain.Config().IsBrioche(header.Number) {`
- `go-wbft/consensus/wbft/engine/engine.go:1100` — `cfgBlockReward := e.cfg.GetConfig(header.Number).BlockReward`
- `go-wbft/consensus/wbft/engine/engine.go:1109` — `beneficiaryInfo := e.cfg.GetConfig(header.Number).BlockRewardBeneficiary`
- `go-wbft/params/config_wbft.go:229` — `BlockReward                 *math.HexOrDecimal256 `json:"blockReward,omitempty"`            // Reward from start, works only on WBFT consensus protocol`
- `go-stablenet/params/config_wbft.go:186` — `type WBFTConfig struct {`
- `go-stablenet/consensus/wbft/engine/engine.go:943` — `if err := e.distributeBaseFee(chain, header, state); err != nil {`
- `go-stablenet/consensus/wbft/engine/engine.go:972` — `func (e *Engine) distributeBaseFee(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB) error {`

## CD-B-07 genesis 구성 방식

**같은 점.** WBFT 두 체인은 같은 절차다. init.validators와 init.blsPublicKeys로 초기 EpochInfo를 만들어 extraData에 RLP로 넣고, 시스템 계약 코드를 alloc에 주입한다. chainbench의 wbft/stablenet 템플릿도 같은 자리표(__VALIDATORS_JSON__, __BLS_PUBLIC_KEYS_JSON__, __EXTRA_DATA__)를 쓴다.

| 구분 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 다른 점 | gwemix wemix genesis 하위 명령이 템플릿과 governance 설정(members, env, accounts, staker/ecosystem/maintenance/feecollector 주소)을 읽어 genesis를 만든다. extraData는 bootnode id, coinbase는 admin이다. 헤더에 minerNodeId/minerNodeSig 필드가 있다. chainbench 템플릿의 config에는 pangyo/applepie/brioche 블록만 있고 anzeon/croissant 섹션이 없다. | config.croissant 섹션(wBFT, init, govContracts)이 필수다. CroissantBlock이 0일 때만 genesis에서 extraData와 계약을 만든다. 1 이상이면 PoA genesis로 시작해 전환한다. | config.anzeon 섹션(wbft, init, systemContracts)이 필수다. systemContracts.govValidator.params에 validators/members/quorum/gasTip 같은 초기 governance 값이 들어간다. alloc 계정의 Extra 비트를 검증한다. |

설정으로 둘 값:

- 체인별 genesis 템플릿과 자리표 값(chainId, validators, BLS 키, 초기 잔액)
- anzeon.wbft.epochLength(템플릿 140)와 croissant.wBFT.epochLength(템플릿 10)처럼 템플릿마다 다른 기본값

체인별 구현이 필요한 부분:

- WEMIX3.0 genesis는 템플릿 치환이 아니라 바이너리 생성 경로라 별도 generator가 필요하다(chainbench poa family가 이미 담당)
- genesis hash 고정값 검사는 체인·템플릿·키셋 조합마다 다르므로 기대값을 프로필에 둔다

근거:

- `go-wemix/cmd/gwemix/wemixcmd.go:86` — `Name:      "genesis",`
- `go-wemix/cmd/gwemix/wemixcmd.go:270` — `type genesisConfig struct {`
- `go-wemix/cmd/gwemix/wemixcmd.go:409` — `func genGenesis(ctx *cli.Context) error {`
- `go-wemix/wemix/admin.go:174` — `func (ma *wemixAdmin) getGenesisInfo() (string, common.Address, error) {`
- `go-wbft/core/genesis.go:239` — `func initializeCroissantGenesis(genesis *Genesis) error {`
- `go-wbft/core/genesis.go:243` — `extraData, err := wbft.CreateInitialExtraData(genesis.Config.Croissant)`
- `go-wbft/core/genesis.go:248` — `return InjectContracts(genesis, genesis.Config)`
- `go-wbft/consensus/wbft/config.go:231` — `func CreateInitialExtraData(config *params.CroissantConfig) ([]byte, error) {`
- `go-wbft/params/config_wbft.go:55` — `type CroissantConfig struct {`
- `go-stablenet/core/genesis.go:242` — `func initializeAnzeonGenesis(genesis *Genesis) error {`
- `go-stablenet/core/genesis.go:247` — `extraData, err := wbft.CreateInitialExtraData(genesis.Config.Anzeon)`
- `go-stablenet/params/config_wbft.go:55` — `type AnzeonConfig struct {`
- `chainbench/internal/chains/wemix/manifest.json:7` — `"bootstrap": { "type": "governance-etcd" },`
- `chainbench/internal/chains/wemix/genesis.json:25` — `"minerNodeId": "0x0",`
- `chainbench/internal/chains/wbft/genesis.json:18` — `"croissant": {`
- `chainbench/internal/chains/wbft/genesis.json:66` — `"extraData": "__EXTRA_DATA__",`
- `chainbench/internal/chains/stablenet/genesis.json:14` — `"anzeon": {`
- `chainbench/internal/chains/stablenet/genesis.json:96` — `"extraData": "__EXTRA_DATA__",`

## CD-B-08 테스트가 건드리는 노드 플래그

**같은 점.** --syncmode, --mine, --miner.*, --ws, --http.api, --ws.api, --bootnodes, --nodekey, --metrics, --metrics.addr, --metrics.port 이름이 세 곳에 있다. init, dumpgenesis, account 하위 명령도 같다.

| 구분 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 다른 점 | syncmode에 light가 있고 light.* 플래그와 les 모듈이 있다. 기본 syncmode는 snap이다. wemix.block.interval 같은 wemix.* 플래그 8개와 wemix-testnet 플래그가 있다. 하위 명령 wemix genesis가 있다. | syncmode는 snap 또는 full이고 기본은 full이다. light 서버가 없다. wbft 전용 플래그 이름은 없다(합의 설정은 genesis에 있다). | go-wbft와 같다. 기본 syncmode full. |

설정으로 둘 값:

- env.launch로 넘길 플래그 값(metrics, ws, syncmode, bootnodes)
- 체인별 바이너리 이름(gwemix 2종, gstable)과 하위 명령

체인별 구현이 필요한 부분:

- Snap sync 검사는 WEMIX3.0 기본값이 snap, 나머지는 full이라 명시 플래그로 통일해야 한다
- light 모드 검사는 WEMIX3.0 전용이다

근거:

- `go-wemix/cmd/utils/flags.go:231` — `Usage: `Blockchain sync mode ("snap", "full" or "light")`,`
- `go-wemix/cmd/utils/flags.go:275` — `Name:  "light.serve",`
- `go-wemix/cmd/utils/flags.go:896` — `Name:  "wemix.block.interval",`
- `go-wemix/cmd/utils/flags.go:153` — `Name:  "wemix-testnet",`
- `go-wemix/cmd/utils/flags.go:760` — `Name:  "metrics",`
- `go-wemix/eth/ethconfig/config.go:65` — `SyncMode: downloader.SnapSync,`
- `go-wbft/cmd/utils/flags.go:263` — `Usage:    `Blockchain sync mode ("snap" or "full")`,`
- `go-wbft/cmd/utils/flags.go:837` — `Name:     "metrics",`
- `go-wbft/eth/ethconfig/config.go:63` — `SyncMode:       downloader.FullSync,`
- `go-stablenet/cmd/utils/flags.go:263` — `Usage:    `Blockchain sync mode ("snap" or "full")`,`
- `go-stablenet/cmd/utils/flags.go:640` — `Name:     "ws",`
- `go-stablenet/eth/ethconfig/config.go:60` — `SyncMode:       downloader.FullSync,`

## CD-B-09 동기화 경로(downloader, fetcher, snap)

**같은 점.** 세 빌드 모두 eth/downloader, eth/fetcher, eth/protocols/snap 모듈을 포함한다(AST 그래프 기준 파일 수 go-wemix 16/2/7, go-wbft 17/2/8, go-stablenet 17/2/8). full/snap 동기화 검사는 세 체인 공통 후보다.

| 구분 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 다른 점 | 동기화 판단은 표준 TD 비교다. 그 위에 wemix/sync.go가 etcd 토큰과 파트너 상태를 주기적으로 검사한다(syncCheck). les 모듈이 있어 light 동기화도 가능하다. | CroissantEnabled면 peer TD가 자기 TD + tdAdjustment 이내일 때 동기화 완료로 본다. WBFT 블록은 TD가 1씩 늘어서 표준 TD 비교를 쓰지 않는다. | AnzeonEnabled 조건으로 같은 TD 보정 규칙을 쓴다. les 없음. |

설정으로 둘 값:

- 동기화 노드의 syncmode와 시작 높이 차이(gap) 값
- 동기화 완료 판정 timeout

체인별 구현이 필요한 부분:

- "동기화 중"(eth_syncing) 관찰은 공통이지만 WEMIX3.0은 etcd 파트너 여부에 따라 syncCheck가 개입하므로 비파트너 EN으로 관찰한다
- downloader/fetcher 경로를 구분하려면 로그 계측이 필요하며 latest hash만으로는 증명되지 않는다

근거:

- `go-wemix/wemix/sync.go:242` — `func syncCheck() error {`
- `go-wemix/wemix/sync.go:243` — `if admin == nil || !admin.amPartner() || admin.self == nil || !admin.etcdIsRunning() {`
- `go-wemix/eth/downloader/modes.go:28` — `LightSync                 // Download only the headers and terminate afterwards`
- `go-wbft/eth/sync.go:214` — `} else if cs.handler.chain.Config().CroissantEnabled() && op.td.Cmp(new(big.Int).Add(ourTD, big.NewInt(tdAdjustment))) <= 0 {`
- `go-stablenet/eth/sync.go:214` — `} else if cs.handler.chain.Config().AnzeonEnabled() && op.td.Cmp(new(big.Int).Add(ourTD, big.NewInt(tdAdjustment))) <= 0 {`

## 인용 범위 주의

- `go-wemix/cmd/gwemix`는 `cmd/geth`를 가리키는 symlink다. 빌드 선택 목록에는 `cmd/geth/...` 경로로 들어 있다. 이 문서는 사용자가 부르는 이름인 `cmd/gwemix` 경로로 적었다.
- `chainbench/...` 인용은 체인 빌드 선택 파일이 아니다. 하네스가 체인 차이를 어떻게 다루는지 보여 주려고 넣었다.
- `go-wemix/internal/web3ext/web3ext.go`는 콘솔 JS 확장이다. RPC 메서드 이름(admin_wemixInfo)을 확인하는 용도로만 인용했다.
