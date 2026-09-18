# 세 클라이언트 차이 (2026-09-11 현재 코드 기준)

WEMIX3.0=go-wemix, WEMIX4.0=go-wbft, StableNet=go-stablenet의 현재 작업 트리를 다시 읽고 정리했다. 2026-09-09 자료의 결론과 줄번호는 승계하지 않았다. 인용 파일은 각 프로젝트의 실제 빌드 선택 파일(`build/<project>-selected.json`)에 들어 있는지 검사했다(예외는 문서 끝 "인용 범위 주의"). 노드를 띄워 실행한 결과가 아니라 정적 분석이다. 두 부분으로 나뉜다. A부는 거래·수수료·계정(CD-A-01~09), B부는 합의·RPC·헤더·시스템 계약·포크·보상·genesis·플래그·동기화(CD-B-01~09)다. 각 항목의 근거는 `<repo>/<path>:<line>` — 현재 줄의 코드 조각 형식이다. 기계 판독용 목록은 [chain-differences.json](chain-differences.json)이다.

## 공통 테스트 분리에 바로 영향을 주는 결론

1. Chain ID로는 WEMIX3.0과 WEMIX4.0을 구분할 수 없다(둘 다 1111). client family를 명시하고 합의 namespace(`wemix_` vs `istanbul_`)로 확인한다(CD-A-01, CD-B-02).
2. 거래 타입 0x00/0x01/0x02/0x16과 `eth_createAccessList`, `eth_signRawFeeDelegateTransaction`은 세 클라이언트 공통이다. 0x04(7702)와 P256(0x100)은 go-wemix에 없고, 나머지 둘도 gate가 다르다(높이 vs 설정 존재, Croissant vs Boho)(CD-A-02, CD-A-04, CD-A-05, CD-B-05).
3. 수수료는 세 공식이다. baseFee 변화식·상하한, 최소 tip/feeCap 기준, effectiveGasPrice(StableNet 비인가 계정은 헤더 GasTip)가 다르다. 기대값은 체인별 oracle이어야 한다(CD-A-06).
4. 거부 사유 sentinel 문자열은 같지만 go-wbft/go-stablenet은 문맥을 덧붙여 감싼다. 부분 문자열 일치로 판정해야 한다(CD-A-08).
5. StableNet만 계정 blacklist/authorized 상태가 있다(CD-A-07). 시스템 계약은 세 체인이 주소·ABI·의미 모두 다르다(CD-B-04).
6. `wemix` namespace는 genesis에 Brioche 설정이 있을 때만 열리고 메서드는 3개뿐이다. chainbench wemix manifest가 적은 `wemix_getValidators`/`wemix_getReward`는 go-wemix에 없다(CD-B-02).
7. WBFTExtra는 StableNet에서 GasTip 필드가 끼어 RLP 배치와 epochInfo 키가 다르다(CD-B-03). StableNet에는 블록 보상이 없고 baseFee를 검증자에게 배분한다(CD-B-06).
8. go-wemix의 Croissant 높이는 기능이 아니라 중단 조건이다(CD-B-05). PUSH0은 go-wemix 기본 instruction set에 없다(ExtraEips로만 활성).
9. 기본 syncmode가 go-wemix snap, 나머지 full이다. light는 go-wemix만 있다(CD-B-08).

# A부. 거래·수수료·계정

| ID | 분야 | 제목 | 설정 항목 수 | 별도 구현 항목 수 | 근거 수 |
|---|---|---|---:|---:|---:|
| CD-A-01 | 네트워크 식별 | Chain ID와 Network ID 기본값 | 4 | 1 | 12 |
| CD-A-02 | 거래 타입 | 지원 거래 타입과 txpool의 포크 게이트 | 2 | 2 | 17 |
| CD-A-03 | 수수료 대납 | Fee delegation(0x16) 서명 구조와 대납자 검사 | 2 | 3 | 21 |
| CD-A-04 | EIP-7702 | SetCode(0x04) 게이트와 authorization 처리 | 2 | 2 | 8 |
| CD-A-05 | 프리컴파일 | P256VERIFY(0x100) 존재와 포크 게이트 | 2 | 2 | 9 |
| CD-A-06 | 수수료 정책 | baseFee 계산, 가스 가격 오라클, 최소 tip, 교체 bump, effectiveGasPrice | 5 | 3 | 34 |
| CD-A-07 | 계정 정책 | StableNet 블랙리스트·인가 상태(extra 비트) | 2 | 2 | 8 |
| CD-A-08 | 오류 문자열 | 거부 경로의 오류 문자열 비교 | 1 | 2 | 20 |
| CD-A-09 | 노드 기본값 | 가스 한도·가스 가격 관련 노드 플래그 기본값 | 2 | 2 | 18 |

## CD-A-01 Chain ID와 Network ID 기본값

**같은 점.** Chain ID는 genesis config에서 읽는다. eth_chainId로 확인하는 절차는 세 클라이언트가 같다.

| 항목 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 차이 | 코드 기본값 mainnet 1111, testnet 1112. NetworkId 기본값은 ethconfig에 1111로 고정돼 있다. --wemix 같은 프리셋 플래그가 NetworkId를 다시 정한다. | 코드 기본값 mainnet 1111, testnet 1112로 go-wemix와 같다. NetworkId 기본값은 0이며 chainId를 그대로 따른다. Croissant 높이는 아직 TODO(200,000,000)다. | 코드 기본값 mainnet 8282, testnet 8283. NetworkId 기본값은 0이며 chainId를 따른다. mainnet은 Boho 0, testnet은 Boho 14,408,500이다. |

설정으로 분리할 값:
- expectedChainId(프로필별 실제 값)
- expectedNetworkId
- genesis hash
- clientFamily(wemix|wbft|stablenet)를 명시

별도 구현 또는 체인별 처리:
- Chain ID로 구현체를 고르지 않는다. go-wemix와 go-wbft가 같은 1111을 쓰므로 clientFamily와 web3_clientVersion 또는 합의 RPC(wemix_ vs istanbul_)로 사전 확인한다.

근거:
- `go-wemix/params/config.go:146` — `ChainID: big.NewInt(1111),`
- `go-wemix/params/config.go:176` — `ChainID: big.NewInt(1112),`
- `go-wbft/params/config.go:47` — `ChainID: big.NewInt(1111),`
- `go-wbft/params/config.go:140` — `ChainID: big.NewInt(1112),`
- `go-wbft/params/config.go:71` — `CroissantBlock: big.NewInt(200_000_000), // TODO: decide the block number`
- `go-stablenet/params/config.go:45` — `ChainID: big.NewInt(8282),`
- `go-stablenet/params/config.go:156` — `ChainID: big.NewInt(8283),`
- `go-stablenet/params/config.go:65` — `BohoBlock: big.NewInt(0),`
- `go-stablenet/params/config.go:169` — `BohoBlock: big.NewInt(14408500),`
- `go-wemix/eth/ethconfig/config.go:75` — `NetworkId: 1111,`
- `go-wbft/eth/ethconfig/config.go:67` — `NetworkId: 0, // enable auto configuration of networkID == chainID`
- `go-stablenet/eth/ethconfig/config.go:64` — `NetworkId: 0, // enable auto configuration of networkID == chainID`

## CD-A-02 지원 거래 타입과 txpool의 포크 게이트

**같은 점.** Legacy(0x00), AccessList(0x01), DynamicFee(0x02), FeeDelegate(0x16)는 세 클라이언트가 모두 디코딩한다. Berlin 전에는 typed tx를, London 전에는 0x02를, Applepie 전에는 0x16을 거부하는 순서도 같다.

| 항목 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 차이 | FeeDelegate 상수만 10진 22로 적혀 있다(값은 0x16과 같다). Blob(0x03)과 SetCode(0x04)는 상수도 디코더도 없다. txpool은 eip2718/eip1559/feedelegation 플래그를 head+1 기준으로 켠다. | 0x03, 0x04 상수와 디코더가 있다. txpool은 Cancun 전 0x03, Croissant 전 0x04를 거부한다. 거부 메시지는 "type %d rejected, pool not yet in <fork>"로 감싼다. | 0x03, 0x04 상수와 디코더가 있다. 0x04는 AnzeonEnabled()(config에 anzeon 섹션 존재)로 게이트한다. 블록 높이가 아니라 설정 존재 여부다. |

설정으로 분리할 값:
- supportedTxTypes(프로필별)
- activeForks(berlin/london/applepie/cancun/croissant/anzeon/boho)

별도 구현 또는 체인별 처리:
- 0x04 시험은 세 체인 공통 성공 대상이 아니다. go-wemix는 SKIP 사유를 "타입 미구현"으로 기록한다.
- 거부 메시지 비교는 core.ErrTxTypeNotSupported 포함 여부로 한다. go-wemix는 감싸지 않은 원문을 돌려주고 나머지는 fork 이름을 덧붙인다.

근거:
- `go-wemix/core/types/transaction.go:45` — `LegacyTxType = iota`
- `go-wemix/core/types/transaction.go:48` — `FeeDelegateDynamicFeeTxType = 22 // fee delegation`
- `go-wemix/core/types/transaction.go:198` — `case FeeDelegateDynamicFeeTxType:`
- `go-wbft/core/types/transaction.go:51` — `BlobTxType = 0x03`
- `go-wbft/core/types/transaction.go:52` — `SetCodeTxType = 0x04`
- `go-wbft/core/types/transaction.go:53` — `FeeDelegateDynamicFeeTxType = 0x16 // fee delegation(22)`
- `go-stablenet/core/types/transaction.go:52` — `SetCodeTxType = 0x04`
- `go-stablenet/core/types/transaction.go:53` — `FeeDelegateDynamicFeeTxType = 0x16 // fee delegation(22)`
- `go-wemix/core/tx_pool.go:651` — `if !pool.eip2718 && tx.Type() != types.LegacyTxType {`
- `go-wemix/core/tx_pool.go:660` — `if !pool.feedelegation && tx.Type() == types.FeeDelegateDynamicFeeTxType {`
- `go-wemix/core/tx_pool.go:1398` — `pool.feedelegation = pool.chainconfig.IsApplepie(next)`
- `go-wbft/core/txpool/validation.go:72` — `if !opts.Config.IsApplepie(head.Number) && tx.Type() == types.FeeDelegateDynamicFeeTxType {`
- `go-wbft/core/txpool/validation.go:78` — `if !opts.Config.IsCroissant(head.Number) && tx.Type() == types.SetCodeTxType {`
- `go-stablenet/core/txpool/validation.go:72` — `if !opts.Config.IsApplepie(head.Number) && tx.Type() == types.FeeDelegateDynamicFeeTxType {`
- `go-stablenet/core/txpool/validation.go:78` — `if !opts.Config.AnzeonEnabled() && tx.Type() == types.SetCodeTxType {`
- `go-stablenet/params/config.go:1085` — `func (c *ChainConfig) AnzeonEnabled() bool {`
- `go-wbft/params/config.go:1062` — `func (c *ChainConfig) CroissantEnabled() bool {`

## CD-A-03 Fee delegation(0x16) 서명 구조와 대납자 검사

**같은 점.** 세 클라이언트 모두 FeePayer 주소와 FV/FR/FS 서명을 가진 같은 구조체를 쓴다. RecoverFeePayer가 chainId로 대납자를 복구하고 실패하면 "fee delegation: invalid feePayer"를 낸다. txpool은 대납자 잔액이 가스비보다 적으면 ErrFeePayerInsufficientFunds, 송신자 잔액이 value보다 적으면 ErrSenderInsufficientFunds를 낸다.

| 항목 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 차이 | state transition의 buyGas가 Applepie 전이면 타입 미지원으로 거절한다. 대납자가 송신자와 같으면 gas*feeCap+value를 한 계정에서 검사한다. txpool 오류는 감싸지 않은 sentinel이다. | buyGas의 feeCheck가 Croissant 활성 여부와 대납 여부에 따라 gasLimit*feeCap 또는 gasLimit*gasPrice로 갈린다(CroissantEnabled이고 아직 Croissant 전이며 대납이면 gasPrice 기준). txpool 1차 검사는 sentinel, 2차(pending 합산) 검사는 "%w: fee payer balance ..."로 감싼다. | Anzeon이면 대납자 블랙리스트를 txpool과 실행 양쪽에서 검사해 ErrBlacklistedAccount를 낸다. buyGas는 AnzeonEnabled이고 대납이면 gasPrice 기준으로 feeCheck한다. txpool 1차 검사부터 "%w: balance %v, fee cost %v"로 감싼다. |

설정으로 분리할 값:
- senderAccountRef, feePayerAccountRef, 자금 액수
- activeForks(applepie/croissant/anzeon)

별도 구현 또는 체인별 처리:
- 대납 성공과 무효 서명 거부는 공통 시나리오로 둔다.
- 대납자 잔액 기대값(feeCap 기준인지 gasPrice 기준인지)과 오류 문자열 매칭(sentinel vs 감싼 메시지)은 체인별 오라클로 분리한다.
- StableNet의 대납자 블랙리스트 경로는 StableNet 전용 시험이다.

근거:
- `go-wemix/core/types/feedelegate_dynamic_fee_tx.go:27` — `FeePayer *common.Address `rlp:"nil"``
- `go-wemix/core/types/feedelegate_dynamic_fee_tx.go:29` — `FV *big.Int `json:"fv" gencodec:"required"` // feePayer V`
- `go-wbft/core/types/tx_fee_delegation.go:29` — `FeePayer *common.Address `rlp:"nil"``
- `go-stablenet/core/types/tx_fee_delegation.go:31` — `FV *big.Int // feePayer V`
- `go-wemix/core/types/transaction_signing.go:35` — `ErrInvalidFeePayer = errors.New("fee delegation: invalid feePayer")`
- `go-wbft/core/types/transaction_signing.go:189` — `func RecoverFeePayer(chainID *big.Int, tx *Transaction) (common.Address, error) {`
- `go-stablenet/core/types/transaction_signing.go:35` — `ErrInvalidFeePayer = errors.New("fee delegation: invalid feePayer")`
- `go-wemix/core/tx_pool.go:713` — `if pool.currentState.GetBalance(feePayer).Cmp(tx.FeePayerCost()) < 0 {`
- `go-wemix/core/tx_pool.go:714` — `return ErrFeePayerInsufficientFunds`
- `go-wemix/core/state_transition.go:201` — `if !st.evm.ChainConfig().IsApplepie(st.evm.Context.BlockNumber) {`
- `go-wemix/core/state_transition.go:207` — `if feePayer == st.msg.From() {`
- `go-wbft/core/txpool/validation.go:257` — `return ErrFeePayerInsufficientFunds`
- `go-wbft/core/txpool/validation.go:373` — `return fmt.Errorf("%w: fee payer balance %v, needed %v, overshot %v", ErrFeePayerInsufficientFunds`
- `go-wbft/core/state_transition.go:267` — `if !st.evm.ChainConfig().CroissantEnabled() || st.evm.ChainConfig().IsCroissant(st.evm.Context.BlockNumber) || !isFeeDelegation {`
- `go-stablenet/core/txpool/validation.go:284` — `if opts.Config.AnzeonEnabled() && opts.State.IsBlacklisted(feePayer) {`
- `go-stablenet/core/txpool/validation.go:291` — `return fmt.Errorf("%w: balance %v, fee cost %v, overshot %v", ErrFeePayerInsufficientFunds`
- `go-stablenet/core/state_transition.go:282` — `if !st.evm.ChainConfig().AnzeonEnabled() || !isFeeDelegation {`
- `go-stablenet/core/state_transition.go:581` — `if rules.IsAnzeon && st.state.IsBlacklisted(payer) {`
- `go-wemix/core/error.go:103` — `ErrFeePayerInsufficientFunds = errors.New("fee delegation: insufficient feePayer's funds for gas * price")`
- `go-wbft/core/txpool/errors.go:70` — `ErrFeePayerInsufficientFunds = errors.New("fee delegation: insufficient feePayer's funds for gas * price")`
- `go-stablenet/core/txpool/errors.go:74` — `ErrSenderInsufficientFunds = errors.New("fee delegation: insufficient sender's funds for value")`

## CD-A-04 SetCode(0x04) 게이트와 authorization 처리

**같은 점.** go-wbft와 go-stablenet은 같은 구조로 SetCodeAuthorizations를 메시지에 싣고 실행 시 applyAuthorization을 돈다. authorization tuple이 비면 txpool이 거절한다.

| 항목 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 차이 | SetCode 타입, 디코더, signer 처리가 전혀 없다. 0x04 raw tx는 디코딩 단계에서 실패한다. | txpool 게이트는 IsCroissant(head.Number), 즉 블록 높이 기준이다. mainnet 설정의 Croissant 높이는 TODO 값이다. | txpool 게이트는 AnzeonEnabled(), 즉 config 섹션 존재 기준이다. 높이와 무관하게 anzeon 섹션이 있으면 받는다. |

설정으로 분리할 값:
- setCodeEnabled(체인별 capability)
- authorization chainId·nonce 입력

별도 구현 또는 체인별 처리:
- 세 체인 공통 성공 시험에서 제외한다. WBFT/StableNet 하위 suite로만 유지하고 go-wemix는 "타입 미구현" SKIP으로 기록한다.
- "지원 선언"과 "활성"을 구분한다. WBFT는 높이, StableNet은 설정 존재가 기준이므로 preflight가 다르다.

근거:
- `go-wemix/core/types/transaction.go:47` — `DynamicFeeTxType`
- `go-wemix/core/types/transaction.go:48` — `FeeDelegateDynamicFeeTxType = 22 // fee delegation`
- `go-wbft/core/txpool/validation.go:79` — `return fmt.Errorf("%w: type %d rejected, pool not yet in Croissant", core.ErrTxTypeNotSupported, tx.Type())`
- `go-wbft/core/txpool/validation.go:131` — `if len(tx.SetCodeAuthorizations()) == 0 {`
- `go-wbft/core/state_transition.go:509` — `st.applyAuthorization(msg, &auth)`
- `go-stablenet/core/txpool/validation.go:79` — `return fmt.Errorf("%w: type %d rejected, pool not yet in Anzeon", core.ErrTxTypeNotSupported, tx.Type())`
- `go-stablenet/core/txpool/validation.go:141` — `if len(tx.SetCodeAuthorizations()) == 0 {`
- `go-stablenet/core/state_transition.go:535` — `for _, auth := range msg.SetCodeAuthorizations {`

## CD-A-05 P256VERIFY(0x100) 존재와 포크 게이트

**같은 점.** Homestead/Byzantium/Istanbul/Berlin 프리컴파일 집합은 세 클라이언트가 같다.

| 항목 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 차이 | ActivePrecompiles는 Berlin까지만 안다. p256Verify 구현이 없다. Cancun 집합도 없다. | PrecompiledContractsCroissant에 0x100 p256Verify가 있다. rules.IsCroissant일 때만 활성이다. | Anzeon 집합에는 0x100이 없고 Boho 집합에만 있다. rules.IsBoho는 isAnzeon && IsBoho(num)이다. mainnet 코드 기본값은 Boho 0이지만 실행 genesis가 기준이다. |

설정으로 분리할 값:
- p256Enabled(체인·포크별)
- precompile 주소 0x100 입력 벡터

별도 구현 또는 체인별 처리:
- 세 체인 공통 성공 시험에서 제외한다. WBFT는 Croissant, StableNet은 Boho 활성 후에만 같은 입력 벡터를 재사용한다.
- 활성 전 기대값(빈 반환 또는 실패)은 체인별로 따로 둔다. 미활성에서 빈 응답을 성공으로 세지 않는다.

근거:
- `go-wemix/core/vm/contracts.go:136` — `case rules.IsBerlin:`
- `go-wbft/core/vm/contracts.go:127` — `var PrecompiledContractsCroissant = map[common.Address]PrecompiledContract{`
- `go-wbft/core/vm/contracts.go:140` — `common.BytesToAddress([]byte{0x1, 0x00}): &p256Verify{},`
- `go-wbft/core/vm/contracts.go:182` — `case rules.IsCroissant:`
- `go-stablenet/core/vm/contracts.go:127` — `var PrecompiledContractsAnzeon = map[common.Address]PrecompiledContract{`
- `go-stablenet/core/vm/contracts.go:141` — `var PrecompiledContractsBoho = map[common.Address]PrecompiledContract{`
- `go-stablenet/core/vm/contracts.go:154` — `common.BytesToAddress([]byte{0x1, 0x00}): &p256Verify{},`
- `go-stablenet/core/vm/contracts.go:200` — `case rules.IsBoho:`
- `go-stablenet/params/config.go:1534` — `IsBoho: isAnzeon && c.IsBoho(num),`

## CD-A-06 baseFee 계산, 가스 가격 오라클, 최소 tip, 교체 bump, effectiveGasPrice

**같은 점.** eth_gasPrice는 세 클라이언트 모두 SuggestGasTipCap + head.BaseFee이고 eth_maxPriorityFeePerGas는 SuggestGasTipCap이다. txpool 교체 bump 기본값은 10%다. 첫 London 블록 baseFee는 InitialBaseFee다.

| 항목 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 차이 | baseFee는 거버넌스(GetBlockBuildParameters)의 maxBaseFee, 변화율, gasTarget 비율로 계산하고 [1, maxBaseFee]로 clamp한다. 거버넌스 미초기화면 parent 값을 유지한다. tip 제안은 거버넌스 SuggestGasPrice이며 기본 100 gwei다. txpool 최소값은 거버넌스가 miner_setGasPrice로 주입한 maxPriorityFeePerGas이고, 로컬 tx도 DropUnderPriced로 effective tip을 검사한다. | baseFee는 EIP-1559 표준식이다. 감소 하한은 0이고 상한 clamp가 없다. Croissant 전이면 tip 제안이 wpoa.SuggestGasPrice 고정값이고 후에는 샘플링 오라클이다. txpool 최소 tip은 miner.gasprice(기본 100 gwei)가 SetGasTip으로 들어간다. | Anzeon이면 사용률 20% 초과 시 2% 인상, 6% 미만 시 2% 인하, [MinBaseFee 20,000 gwei, MaxBaseFee 20,000,000 gwei] clamp다. txpool은 tip 외에 feeCap >= MinBaseFee + MinTip도 검사한다(legacy는 gasPrice가 feeCap). 비인가 계정의 tip은 tx 값이 아니라 헤더 GasTip으로 대체돼 effectiveGasPrice와 오라클 샘플링에 쓰인다. 인가 계정 tx는 마지막 로그에 AuthorizedTxExecuted 이벤트를 남긴다. |

설정으로 분리할 값:
- gas, tip, feeCap 숫자 입력
- minTip(프로필별 노드 플래그)
- priceBump 10%
- MinBaseFee/MaxBaseFee/threshold(StableNet 상수)
- 거버넌스 파라미터 참조(WEMIX3)

별도 구현 또는 체인별 처리:
- baseFee 변화·상한·하한 기대값은 FeePolicy 오라클로 체인별 계산한다. 고정 상한 시험은 WBFT에 해당 clamp가 없으므로 공통 대상이 아니다.
- effectiveGasPrice 기대값은 StableNet에서 송신자 인가 여부와 헤더 GasTip을 읽어야 한다.
- underpriced 거부 기준은 WEMIX3 거버넌스 값, WBFT miner.gasprice, StableNet MinBaseFee+MinTip으로 다르므로 프로필 입력 준비 단계를 체인별로 둔다.

근거:
- `go-wemix/consensus/misc/eip1559.go:81` — `_, maxBaseFeeGov, _, baseFeeMaxChangeRate, gasTargetPercentage, err := wemixminer.GetBlockBuildParameters(parent.Number)`
- `go-wemix/consensus/misc/eip1559.go:121` — `return math.BigMin(x.Add(parent.BaseFee, baseFeeDelta), maxBaseFee)`
- `go-wemix/consensus/misc/eip1559.go:142` — `common.Big1,`
- `go-wemix/eth/gasprice/gasprice.go:153` — `return wemixminer.SuggestGasPrice(), nil`
- `go-wemix/wemix/miner/miner.go:134` — `return big.NewInt(100 * params.GWei)`
- `go-wemix/wemix/admin.go:648` — `err := ma.rpcCli.CallContext(ctx, &v, "miner_setGasPrice",`
- `go-wemix/core/tx_pool.go:697` — `if local && params.DropUnderPriced && tx.EffectiveGasTipIntCmp(pool.gasPrice, pool.priced.urgent.baseFee) < 0 {`
- `go-wbft/consensus/misc/eip1559/eip1559.go:83` — `return num.Add(parent.BaseFee, baseFeeDelta)`
- `go-wbft/consensus/misc/eip1559/eip1559.go:93` — `return math.BigMax(baseFee, common.Big0)`
- `go-wbft/eth/gasprice/gasprice.go:155` — `if oracle.backend.ChainConfig().CroissantBlock != nil && !oracle.backend.ChainConfig().IsCroissant(head.Number) {`
- `go-wbft/eth/gasprice/gasprice.go:156` — `return wpoa.SuggestGasPrice(), nil`
- `go-wbft/core/txpool/validation.go:120` — `return fmt.Errorf("%w: gas tip cap %v, minimum needed %v", ErrUnderpriced, tx.GasTipCap(), opts.MinTip)`
- `go-wbft/eth/backend.go:455` — `s.txPool.SetGasTip(price)`
- `go-wbft/miner/miner.go:65` — `DefaultGasPriceGWei = 100`
- `go-wbft/core/txpool/legacypool/legacypool.go:164` — `PriceBump: 10,`
- `go-stablenet/consensus/misc/eip1559/eip1559.go:62` — `if config.AnzeonEnabled() {`
- `go-stablenet/consensus/misc/eip1559/eip1559.go:73` — `if maxBaseFee.Cmp(common.Big0) != 0 && baseFee.Cmp(maxBaseFee) > 0 {`
- `go-stablenet/consensus/misc/eip1559/eip1559.go:82` — `if baseFee.Cmp(minBaseFee) < 0 {`
- `go-stablenet/params/protocol_params.go:132` — `IncreasingThreshold uint64 = 20`
- `go-stablenet/params/protocol_params.go:133` — `DecreasingThreshold uint64 = 6`
- `go-stablenet/params/protocol_params.go:135` — `MinBaseFee uint64 = 20000000000000`
- `go-stablenet/params/protocol_params.go:136` — `MaxBaseFee uint64 = 20000000000000000`
- `go-stablenet/core/txpool/validation.go:122` — `if opts.Config.IsLondon(head.Number) && opts.Config.AnzeonEnabled() {`
- `go-stablenet/core/txpool/validation.go:129` — `return fmt.Errorf("%w: gas fee cap %v, minimum needed %v", ErrUnderpriced, tx.GasFeeCap(), minFee)`
- `go-stablenet/core/state_transition.go:169` — `if statedb != nil && !statedb.IsAuthorized(from) {`
- `go-stablenet/core/state_transition.go:170` — `gasTipCap = new(big.Int).Set(headerGasTip)`
- `go-stablenet/core/state_transition.go:592` — `if rules.IsAnzeon && st.state.IsAuthorized(msg.From) {`
- `go-stablenet/core/types/receipt.go:365` — `if err == nil && atEnv.stateReader != nil && !atEnv.stateReader.IsAuthorized(from) && atEnv.headerTip != nil {`
- `go-stablenet/eth/gasprice/gasprice.go:254` — `atEnv := types.NewInstantAnzeonTipEnv(signer, header.BaseFee, header.GasTip(), stateReader)`
- `go-stablenet/core/txpool/legacypool/legacypool.go:163` — `PriceBump: 10,`
- `go-wemix/core/tx_pool.go:182` — `PriceBump: 10,`
- `go-wemix/internal/ethapi/api.go:98` — `tipcap.Add(tipcap, head.BaseFee)`
- `go-wbft/internal/ethapi/api.go:78` — `tipcap.Add(tipcap, head.BaseFee)`
- `go-stablenet/internal/ethapi/api.go:77` — `tipcap.Add(tipcap, head.BaseFee)`

## CD-A-07 StableNet 블랙리스트·인가 상태(extra 비트)

**같은 점.** 잔액, nonce, 서명 검사는 세 클라이언트가 같다.

| 항목 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 차이 | 계정 상태 비트가 없다. IsBlacklisted/IsAuthorized 호출 지점이 core에 없다. | 계정 상태 비트가 없다. | StateAccount에 Extra uint64가 있고 StateDB가 IsBlacklisted/IsAuthorized/SetBlacklisted/SetAuthorized를 제공한다. Anzeon이면 txpool과 실행에서 송신자·수신자·대납자 블랙리스트를 검사해 "blacklisted account: <addr>"를 낸다. 인가 계정은 tip 정책과 수료 로그가 달라진다. |

설정으로 분리할 값:
- accountRoles(sender/recipient/feePayer)
- StableNet 프로필의 blacklistState/authorizedState 사전 조건

별도 구현 또는 체인별 처리:
- 공통 송금 fixture는 비차단·비인가 계정이어야 한다. AccountFixture가 StableNet에서 상태를 사전 검사한다.
- 블랙리스트·인가 정책 자체의 시험은 StableNet 전용 suite로 둔다.

근거:
- `go-stablenet/core/types/state_account.go:37` — `Extra uint64 `rlp:"optional"``
- `go-stablenet/core/state/statedb.go:311` — `func (s *StateDB) IsBlacklisted(addr common.Address) bool {`
- `go-stablenet/core/state/statedb.go:321` — `func (s *StateDB) IsAuthorized(addr common.Address) bool {`
- `go-stablenet/core/state/statedb.go:434` — `func (s *StateDB) SetBlacklisted(addr common.Address) {`
- `go-stablenet/core/txpool/validation.go:253` — `if opts.State.IsBlacklisted(from) {`
- `go-stablenet/core/txpool/validation.go:256` — `if to := tx.To(); to != nil && opts.State.IsBlacklisted(*to) {`
- `go-stablenet/core/state_transition.go:508` — `if st.state.IsBlacklisted(msg.From) {`
- `go-stablenet/core/error.go:147` — `return fmt.Sprintf("blacklisted account: %s", e.Address.Hex())`

## CD-A-08 거부 경로의 오류 문자열 비교

**같은 점.** sentinel 원문은 세 클라이언트가 같다. "insufficient funds for gas * price + value", "insufficient funds for transfer", "exceeds block gas limit", "transaction underpriced", "replacement transaction underpriced", "nonce too low", "invalid sender", "invalid transaction v, r, s values", "fee delegation: invalid feePayer".

| 항목 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 차이 | txpool이 sentinel을 감싸지 않고 그대로 돌려준다. 타입 미지원도 원문만 온다. | txpool이 "%w: ..." 형식으로 문맥을 덧붙인다(nonce, balance, gas tip cap, type rejected). 대납자 잔액은 1차 검사만 원문이다. | go-wbft와 같은 감싸기에 더해 Anzeon 전용 메시지가 있다. "gas fee cap %v, minimum needed %v"(underpriced), "blacklisted account: <addr>". |

설정으로 분리할 값:
- 없음. 문자열은 코드에 고정돼 있다.

별도 구현 또는 체인별 처리:
- assertion은 완전 일치가 아니라 sentinel 부분 문자열 포함으로 판정한다.
- go-wemix에서만 성립하는 "원문 완전 일치" 기대값을 공통에 넣지 않는다.

근거:
- `go-wemix/core/error.go:68` — `ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")`
- `go-wbft/core/error.go:72` — `ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")`
- `go-stablenet/core/error.go:74` — `ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")`
- `go-wemix/core/tx_pool.go:83` — `ErrGasLimit = errors.New("exceeds block gas limit")`
- `go-wbft/core/txpool/errors.go:43` — `ErrGasLimit = errors.New("exceeds block gas limit")`
- `go-stablenet/core/txpool/errors.go:43` — `ErrGasLimit = errors.New("exceeds block gas limit")`
- `go-wemix/core/tx_pool.go:71` — `ErrUnderpriced = errors.New("transaction underpriced")`
- `go-wbft/core/txpool/errors.go:31` — `ErrUnderpriced = errors.New("transaction underpriced")`
- `go-stablenet/core/txpool/errors.go:35` — `ErrReplaceUnderpriced = errors.New("replacement transaction underpriced")`
- `go-wemix/core/error.go:48` — `ErrNonceTooLow = errors.New("nonce too low")`
- `go-wbft/core/txpool/validation.go:238` — `return fmt.Errorf("%w: next nonce %v, tx nonce %v", core.ErrNonceTooLow, next, tx.Nonce())`
- `go-stablenet/core/txpool/validation.go:262` — `return fmt.Errorf("%w: next nonce %v, tx nonce %v", core.ErrNonceTooLow, next, tx.Nonce())`
- `go-wemix/core/tx_pool.go:67` — `ErrInvalidSender = errors.New("invalid sender")`
- `go-wbft/core/txpool/errors.go:27` — `ErrInvalidSender = errors.New("invalid sender")`
- `go-wemix/core/types/transaction.go:35` — `ErrInvalidSig = errors.New("invalid transaction v, r, s values")`
- `go-wemix/core/tx_pool.go:721` — `return ErrInsufficientFunds`
- `go-wbft/core/txpool/validation.go:261` — `return fmt.Errorf("%w: balance %v, tx cost %v, overshot %v", core.ErrInsufficientFunds, balance, cost, new(big.Int).Sub(cost, balance))`
- `go-wemix/core/tx_pool.go:652` — `return ErrTxTypeNotSupported`
- `go-wbft/core/txpool/validation.go:67` — `return fmt.Errorf("%w: type %d rejected, pool not yet in Berlin", core.ErrTxTypeNotSupported, tx.Type())`
- `go-stablenet/core/error.go:147` — `return fmt.Sprintf("blacklisted account: %s", e.Address.Hex())`

## CD-A-09 가스 한도·가스 가격 관련 노드 플래그 기본값

**같은 점.** genesis gasLimit 기본값 4,712,388, txpool.pricelimit 기본 1, txpool.pricebump 기본 10은 같다. --miner.gasprice, --txpool.pricelimit 플래그 이름도 같다.

| 항목 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |
|---|---|---|---|
| 차이 | miner.gasprice 기본 1 gwei, miner.gaslimit(GasCeil) 기본 30,000,000. txpool AccountSlots 100,000, GlobalSlots 500,000. 실제 최소 가격과 블록 gasLimit은 거버넌스가 덮어쓴다(miner_setGasPrice, GetBlockBuildParameters의 gasLimit). | miner.gasprice 기본 100 gwei, GasCeil 기본 105,000,000. txpool AccountSlots 16, GlobalSlots 5,120. | go-wbft와 같은 기본값(100 gwei, 105,000,000, 16/5,120). 여기에 MinBaseFee 20,000 gwei 상수가 더해져 실제 최소 feeCap은 플래그보다 높다. |

설정으로 분리할 값:
- launch 플래그(miner.gasprice, miner.gaslimit, txpool.pricelimit)를 프로필 launch에 명시
- genesis gasLimit

별도 구현 또는 체인별 처리:
- WEMIX3는 플래그를 설정해도 거버넌스 계약 값이 우선한다. 기대 최소 가격은 거버넌스 조회로 읽는 어댑터가 필요하다.
- "gas limit 초과 거부" 시험의 기준 gasLimit은 프로필 값이 아니라 head 블록에서 읽는다.

근거:
- `go-wemix/eth/ethconfig/config.go:87` — `GasCeil: 30000000,`
- `go-wemix/eth/ethconfig/config.go:88` — `GasPrice: big.NewInt(params.GWei),`
- `go-wemix/core/tx_pool.go:181` — `PriceLimit: 1,`
- `go-wemix/core/tx_pool.go:184` — `AccountSlots: 100000,`
- `go-wemix/wemix/miner/miner.go:140` — `func GetBlockBuildParameters(height *big.Int) (blockInterval int64, maxBaseFee, gasLimit *big.Int, baseFeeMaxChangeRate, gasTargetPercentage int64, err error) {`
- `go-wbft/miner/miner.go:75` — `GasCeil: 105000000,`
- `go-wbft/miner/miner.go:70` — `DefaultGasPrice = big.NewInt(DefaultGasPriceGWei * params.GWei)`
- `go-wbft/core/txpool/legacypool/legacypool.go:163` — `PriceLimit: 1,`
- `go-wbft/core/txpool/legacypool/legacypool.go:166` — `AccountSlots: 16,`
- `go-stablenet/miner/miner.go:75` — `GasCeil: 105000000,`
- `go-stablenet/miner/miner.go:65` — `DefaultGasPriceGWei = 100`
- `go-stablenet/core/txpool/legacypool/legacypool.go:162` — `PriceLimit: 1,`
- `go-wemix/params/protocol_params.go:25` — `GenesisGasLimit uint64 = 4712388`
- `go-wbft/params/protocol_params.go:29` — `GenesisGasLimit uint64 = 4712388`
- `go-stablenet/params/protocol_params.go:29` — `GenesisGasLimit uint64 = 4712388`
- `go-wemix/cmd/utils/flags.go:484` — `MinerGasPriceFlag = BigFlag{`
- `go-wbft/cmd/utils/flags.go:445` — `MinerGasPriceFlag = &flags.BigFlag{`
- `go-stablenet/cmd/utils/flags.go:313` — `TxPoolPriceLimitFlag = &cli.Uint64Flag{`


# B부. 합의·RPC·헤더·시스템 계약·포크·보상·genesis·플래그·동기화

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
