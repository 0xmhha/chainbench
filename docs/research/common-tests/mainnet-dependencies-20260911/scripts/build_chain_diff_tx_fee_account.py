#!/usr/bin/env python3
"""Build draft-chain-diff-tx-fee-account.{json,md} from a verified evidence table.
Every evidence row is checked against the CURRENT file: the snippet (whitespace
collapsed) must occur on exactly the cited line, or the build fails."""
import json, os, re, sys
ROOT = '/Users/wm-it-25_0220/Work/github/chain'
OUT = os.environ['OUT']
SEL = {p: {s['path'] for s in json.load(open(f'{OUT}/build/{p}-selected.json'))['selected']} for p in ['go-wemix','go-wbft','go-stablenet']}
def ws(s): return re.sub(r'\s+', ' ', s).strip()
def ev(repo, path, line, snippet):
    src = open(os.path.join(ROOT, repo, path), errors='replace').read().split('\n')
    actual = src[line-1]
    if ws(snippet) not in ws(actual):
        sys.exit(f'MISMATCH {repo}/{path}:{line}\n want: {snippet}\n have: {actual}')
    if path not in SEL[repo]:
        sys.exit(f'NOT IN BUILD {repo}/{path}')
    return {'repo': repo, 'path': path, 'line': line, 'snippet': ws(snippet)}
W, B, S = 'go-wemix', 'go-wbft', 'go-stablenet'
items = []
items.append({
 'id': 'CD-A-01', 'area': '네트워크 식별', 'title': 'Chain ID와 Network ID 기본값',
 'same': 'Chain ID는 genesis config에서 읽는다. eth_chainId로 확인하는 절차는 세 클라이언트가 같다.',
 'differs': {
  'wemix': '코드 기본값 mainnet 1111, testnet 1112. NetworkId 기본값은 ethconfig에 1111로 고정돼 있다. --wemix 같은 프리셋 플래그가 NetworkId를 다시 정한다.',
  'wbft': '코드 기본값 mainnet 1111, testnet 1112로 go-wemix와 같다. NetworkId 기본값은 0이며 chainId를 그대로 따른다. Croissant 높이는 아직 TODO(200,000,000)다.',
  'stablenet': '코드 기본값 mainnet 8282, testnet 8283. NetworkId 기본값은 0이며 chainId를 따른다. mainnet은 Boho 0, testnet은 Boho 14,408,500이다.'},
 'config_items': ['expectedChainId(프로필별 실제 값)', 'expectedNetworkId', 'genesis hash', 'clientFamily(wemix|wbft|stablenet)를 명시'],
 'implementation_items': ['Chain ID로 구현체를 고르지 않는다. go-wemix와 go-wbft가 같은 1111을 쓰므로 clientFamily와 web3_clientVersion 또는 합의 RPC(wemix_ vs istanbul_)로 사전 확인한다.'],
 'evidence': [
  ev(W,'params/config.go',146,'ChainID: big.NewInt(1111),'), ev(W,'params/config.go',176,'ChainID: big.NewInt(1112),'),
  ev(B,'params/config.go',47,'ChainID: big.NewInt(1111),'), ev(B,'params/config.go',140,'ChainID: big.NewInt(1112),'),
  ev(B,'params/config.go',71,'CroissantBlock: big.NewInt(200_000_000), // TODO: decide the block number'),
  ev(S,'params/config.go',45,'ChainID: big.NewInt(8282),'), ev(S,'params/config.go',156,'ChainID: big.NewInt(8283),'),
  ev(S,'params/config.go',65,'BohoBlock: big.NewInt(0),'), ev(S,'params/config.go',169,'BohoBlock: big.NewInt(14408500),'),
  ev(W,'eth/ethconfig/config.go',75,'NetworkId: 1111,'), ev(B,'eth/ethconfig/config.go',67,'NetworkId: 0, // enable auto configuration of networkID == chainID'),
  ev(S,'eth/ethconfig/config.go',64,'NetworkId: 0, // enable auto configuration of networkID == chainID')]})
items.append({
 'id': 'CD-A-02', 'area': '거래 타입', 'title': '지원 거래 타입과 txpool의 포크 게이트',
 'same': 'Legacy(0x00), AccessList(0x01), DynamicFee(0x02), FeeDelegate(0x16)는 세 클라이언트가 모두 디코딩한다. Berlin 전에는 typed tx를, London 전에는 0x02를, Applepie 전에는 0x16을 거부하는 순서도 같다.',
 'differs': {
  'wemix': 'FeeDelegate 상수만 10진 22로 적혀 있다(값은 0x16과 같다). Blob(0x03)과 SetCode(0x04)는 상수도 디코더도 없다. txpool은 eip2718/eip1559/feedelegation 플래그를 head+1 기준으로 켠다.',
  'wbft': '0x03, 0x04 상수와 디코더가 있다. txpool은 Cancun 전 0x03, Croissant 전 0x04를 거부한다. 거부 메시지는 "type %d rejected, pool not yet in <fork>"로 감싼다.',
  'stablenet': '0x03, 0x04 상수와 디코더가 있다. 0x04는 AnzeonEnabled()(config에 anzeon 섹션 존재)로 게이트한다. 블록 높이가 아니라 설정 존재 여부다.'},
 'config_items': ['supportedTxTypes(프로필별)', 'activeForks(berlin/london/applepie/cancun/croissant/anzeon/boho)'],
 'implementation_items': ['0x04 시험은 세 체인 공통 성공 대상이 아니다. go-wemix는 SKIP 사유를 "타입 미구현"으로 기록한다.', '거부 메시지 비교는 core.ErrTxTypeNotSupported 포함 여부로 한다. go-wemix는 감싸지 않은 원문을 돌려주고 나머지는 fork 이름을 덧붙인다.'],
 'evidence': [
  ev(W,'core/types/transaction.go',45,'LegacyTxType = iota'), ev(W,'core/types/transaction.go',48,'FeeDelegateDynamicFeeTxType = 22 // fee delegation'),
  ev(W,'core/types/transaction.go',198,'case FeeDelegateDynamicFeeTxType:'),
  ev(B,'core/types/transaction.go',51,'BlobTxType = 0x03'), ev(B,'core/types/transaction.go',52,'SetCodeTxType = 0x04'), ev(B,'core/types/transaction.go',53,'FeeDelegateDynamicFeeTxType = 0x16 // fee delegation(22)'),
  ev(S,'core/types/transaction.go',52,'SetCodeTxType = 0x04'), ev(S,'core/types/transaction.go',53,'FeeDelegateDynamicFeeTxType = 0x16 // fee delegation(22)'),
  ev(W,'core/tx_pool.go',651,'if !pool.eip2718 && tx.Type() != types.LegacyTxType {'), ev(W,'core/tx_pool.go',660,'if !pool.feedelegation && tx.Type() == types.FeeDelegateDynamicFeeTxType {'),
  ev(W,'core/tx_pool.go',1398,'pool.feedelegation = pool.chainconfig.IsApplepie(next)'),
  ev(B,'core/txpool/validation.go',72,'if !opts.Config.IsApplepie(head.Number) && tx.Type() == types.FeeDelegateDynamicFeeTxType {'),
  ev(B,'core/txpool/validation.go',78,'if !opts.Config.IsCroissant(head.Number) && tx.Type() == types.SetCodeTxType {'),
  ev(S,'core/txpool/validation.go',72,'if !opts.Config.IsApplepie(head.Number) && tx.Type() == types.FeeDelegateDynamicFeeTxType {'),
  ev(S,'core/txpool/validation.go',78,'if !opts.Config.AnzeonEnabled() && tx.Type() == types.SetCodeTxType {'),
  ev(S,'params/config.go',1085,'func (c *ChainConfig) AnzeonEnabled() bool {'), ev(B,'params/config.go',1062,'func (c *ChainConfig) CroissantEnabled() bool {')]})
items.append({
 'id': 'CD-A-03', 'area': '수수료 대납', 'title': 'Fee delegation(0x16) 서명 구조와 대납자 검사',
 'same': '세 클라이언트 모두 FeePayer 주소와 FV/FR/FS 서명을 가진 같은 구조체를 쓴다. RecoverFeePayer가 chainId로 대납자를 복구하고 실패하면 "fee delegation: invalid feePayer"를 낸다. txpool은 대납자 잔액이 가스비보다 적으면 ErrFeePayerInsufficientFunds, 송신자 잔액이 value보다 적으면 ErrSenderInsufficientFunds를 낸다.',
 'differs': {
  'wemix': 'state transition의 buyGas가 Applepie 전이면 타입 미지원으로 거절한다. 대납자가 송신자와 같으면 gas*feeCap+value를 한 계정에서 검사한다. txpool 오류는 감싸지 않은 sentinel이다.',
  'wbft': 'buyGas의 feeCheck가 Croissant 활성 여부와 대납 여부에 따라 gasLimit*feeCap 또는 gasLimit*gasPrice로 갈린다(CroissantEnabled이고 아직 Croissant 전이며 대납이면 gasPrice 기준). txpool 1차 검사는 sentinel, 2차(pending 합산) 검사는 "%w: fee payer balance ..."로 감싼다.',
  'stablenet': 'Anzeon이면 대납자 블랙리스트를 txpool과 실행 양쪽에서 검사해 ErrBlacklistedAccount를 낸다. buyGas는 AnzeonEnabled이고 대납이면 gasPrice 기준으로 feeCheck한다. txpool 1차 검사부터 "%w: balance %v, fee cost %v"로 감싼다.'},
 'config_items': ['senderAccountRef, feePayerAccountRef, 자금 액수', 'activeForks(applepie/croissant/anzeon)'],
 'implementation_items': ['대납 성공과 무효 서명 거부는 공통 시나리오로 둔다.', '대납자 잔액 기대값(feeCap 기준인지 gasPrice 기준인지)과 오류 문자열 매칭(sentinel vs 감싼 메시지)은 체인별 오라클로 분리한다.', 'StableNet의 대납자 블랙리스트 경로는 StableNet 전용 시험이다.'],
 'evidence': [
  ev(W,'core/types/feedelegate_dynamic_fee_tx.go',27,'FeePayer *common.Address `rlp:"nil"`'), ev(W,'core/types/feedelegate_dynamic_fee_tx.go',29,'FV *big.Int `json:"fv" gencodec:"required"` // feePayer V'),
  ev(B,'core/types/tx_fee_delegation.go',29,'FeePayer *common.Address `rlp:"nil"`'), ev(S,'core/types/tx_fee_delegation.go',31,'FV *big.Int // feePayer V'),
  ev(W,'core/types/transaction_signing.go',35,'ErrInvalidFeePayer = errors.New("fee delegation: invalid feePayer")'), ev(B,'core/types/transaction_signing.go',189,'func RecoverFeePayer(chainID *big.Int, tx *Transaction) (common.Address, error) {'), ev(S,'core/types/transaction_signing.go',35,'ErrInvalidFeePayer = errors.New("fee delegation: invalid feePayer")'),
  ev(W,'core/tx_pool.go',713,'if pool.currentState.GetBalance(feePayer).Cmp(tx.FeePayerCost()) < 0 {'), ev(W,'core/tx_pool.go',714,'return ErrFeePayerInsufficientFunds'),
  ev(W,'core/state_transition.go',201,'if !st.evm.ChainConfig().IsApplepie(st.evm.Context.BlockNumber) {'), ev(W,'core/state_transition.go',207,'if feePayer == st.msg.From() {'),
  ev(B,'core/txpool/validation.go',257,'return ErrFeePayerInsufficientFunds'), ev(B,'core/txpool/validation.go',373,'return fmt.Errorf("%w: fee payer balance %v, needed %v, overshot %v", ErrFeePayerInsufficientFunds'),
  ev(B,'core/state_transition.go',267,'if !st.evm.ChainConfig().CroissantEnabled() || st.evm.ChainConfig().IsCroissant(st.evm.Context.BlockNumber) || !isFeeDelegation {'),
  ev(S,'core/txpool/validation.go',284,'if opts.Config.AnzeonEnabled() && opts.State.IsBlacklisted(feePayer) {'), ev(S,'core/txpool/validation.go',291,'return fmt.Errorf("%w: balance %v, fee cost %v, overshot %v", ErrFeePayerInsufficientFunds'),
  ev(S,'core/state_transition.go',282,'if !st.evm.ChainConfig().AnzeonEnabled() || !isFeeDelegation {'), ev(S,'core/state_transition.go',581,'if rules.IsAnzeon && st.state.IsBlacklisted(payer) {'),
  ev(W,'core/error.go',103,'ErrFeePayerInsufficientFunds = errors.New("fee delegation: insufficient feePayer\'s funds for gas * price")'),
  ev(B,'core/txpool/errors.go',70,'ErrFeePayerInsufficientFunds = errors.New("fee delegation: insufficient feePayer\'s funds for gas * price")'),
  ev(S,'core/txpool/errors.go',74,'ErrSenderInsufficientFunds = errors.New("fee delegation: insufficient sender\'s funds for value")')]})
items.append({
 'id': 'CD-A-04', 'area': 'EIP-7702', 'title': 'SetCode(0x04) 게이트와 authorization 처리',
 'same': 'go-wbft와 go-stablenet은 같은 구조로 SetCodeAuthorizations를 메시지에 싣고 실행 시 applyAuthorization을 돈다. authorization tuple이 비면 txpool이 거절한다.',
 'differs': {
  'wemix': 'SetCode 타입, 디코더, signer 처리가 전혀 없다. 0x04 raw tx는 디코딩 단계에서 실패한다.',
  'wbft': 'txpool 게이트는 IsCroissant(head.Number), 즉 블록 높이 기준이다. mainnet 설정의 Croissant 높이는 TODO 값이다.',
  'stablenet': 'txpool 게이트는 AnzeonEnabled(), 즉 config 섹션 존재 기준이다. 높이와 무관하게 anzeon 섹션이 있으면 받는다.'},
 'config_items': ['setCodeEnabled(체인별 capability)', 'authorization chainId·nonce 입력'],
 'implementation_items': ['세 체인 공통 성공 시험에서 제외한다. WBFT/StableNet 하위 suite로만 유지하고 go-wemix는 "타입 미구현" SKIP으로 기록한다.', '"지원 선언"과 "활성"을 구분한다. WBFT는 높이, StableNet은 설정 존재가 기준이므로 preflight가 다르다.'],
 'evidence': [
  ev(W,'core/types/transaction.go',47,'DynamicFeeTxType'), ev(W,'core/types/transaction.go',48,'FeeDelegateDynamicFeeTxType = 22 // fee delegation'),
  ev(B,'core/txpool/validation.go',79,'return fmt.Errorf("%w: type %d rejected, pool not yet in Croissant", core.ErrTxTypeNotSupported, tx.Type())'),
  ev(B,'core/txpool/validation.go',131,'if len(tx.SetCodeAuthorizations()) == 0 {'), ev(B,'core/state_transition.go',509,'st.applyAuthorization(msg, &auth)'),
  ev(S,'core/txpool/validation.go',79,'return fmt.Errorf("%w: type %d rejected, pool not yet in Anzeon", core.ErrTxTypeNotSupported, tx.Type())'),
  ev(S,'core/txpool/validation.go',141,'if len(tx.SetCodeAuthorizations()) == 0 {'), ev(S,'core/state_transition.go',535,'for _, auth := range msg.SetCodeAuthorizations {')]})
items.append({
 'id': 'CD-A-05', 'area': '프리컴파일', 'title': 'P256VERIFY(0x100) 존재와 포크 게이트',
 'same': 'Homestead/Byzantium/Istanbul/Berlin 프리컴파일 집합은 세 클라이언트가 같다.',
 'differs': {
  'wemix': 'ActivePrecompiles는 Berlin까지만 안다. p256Verify 구현이 없다. Cancun 집합도 없다.',
  'wbft': 'PrecompiledContractsCroissant에 0x100 p256Verify가 있다. rules.IsCroissant일 때만 활성이다.',
  'stablenet': 'Anzeon 집합에는 0x100이 없고 Boho 집합에만 있다. rules.IsBoho는 isAnzeon && IsBoho(num)이다. mainnet 코드 기본값은 Boho 0이지만 실행 genesis가 기준이다.'},
 'config_items': ['p256Enabled(체인·포크별)', 'precompile 주소 0x100 입력 벡터'],
 'implementation_items': ['세 체인 공통 성공 시험에서 제외한다. WBFT는 Croissant, StableNet은 Boho 활성 후에만 같은 입력 벡터를 재사용한다.', '활성 전 기대값(빈 반환 또는 실패)은 체인별로 따로 둔다. 미활성에서 빈 응답을 성공으로 세지 않는다.'],
 'evidence': [
  ev(W,'core/vm/contracts.go',136,'case rules.IsBerlin:'),
  ev(B,'core/vm/contracts.go',127,'var PrecompiledContractsCroissant = map[common.Address]PrecompiledContract{'), ev(B,'core/vm/contracts.go',140,'common.BytesToAddress([]byte{0x1, 0x00}): &p256Verify{},'), ev(B,'core/vm/contracts.go',182,'case rules.IsCroissant:'),
  ev(S,'core/vm/contracts.go',127,'var PrecompiledContractsAnzeon = map[common.Address]PrecompiledContract{'), ev(S,'core/vm/contracts.go',141,'var PrecompiledContractsBoho = map[common.Address]PrecompiledContract{'), ev(S,'core/vm/contracts.go',154,'common.BytesToAddress([]byte{0x1, 0x00}): &p256Verify{},'), ev(S,'core/vm/contracts.go',200,'case rules.IsBoho:'),
  ev(S,'params/config.go',1534,'IsBoho: isAnzeon && c.IsBoho(num),')]})
items.append({
 'id': 'CD-A-06', 'area': '수수료 정책', 'title': 'baseFee 계산, 가스 가격 오라클, 최소 tip, 교체 bump, effectiveGasPrice',
 'same': 'eth_gasPrice는 세 클라이언트 모두 SuggestGasTipCap + head.BaseFee이고 eth_maxPriorityFeePerGas는 SuggestGasTipCap이다. txpool 교체 bump 기본값은 10%다. 첫 London 블록 baseFee는 InitialBaseFee다.',
 'differs': {
  'wemix': 'baseFee는 거버넌스(GetBlockBuildParameters)의 maxBaseFee, 변화율, gasTarget 비율로 계산하고 [1, maxBaseFee]로 clamp한다. 거버넌스 미초기화면 parent 값을 유지한다. tip 제안은 거버넌스 SuggestGasPrice이며 기본 100 gwei다. txpool 최소값은 거버넌스가 miner_setGasPrice로 주입한 maxPriorityFeePerGas이고, 로컬 tx도 DropUnderPriced로 effective tip을 검사한다.',
  'wbft': 'baseFee는 EIP-1559 표준식이다. 감소 하한은 0이고 상한 clamp가 없다. Croissant 전이면 tip 제안이 wpoa.SuggestGasPrice 고정값이고 후에는 샘플링 오라클이다. txpool 최소 tip은 miner.gasprice(기본 100 gwei)가 SetGasTip으로 들어간다.',
  'stablenet': 'Anzeon이면 사용률 20% 초과 시 2% 인상, 6% 미만 시 2% 인하, [MinBaseFee 20,000 gwei, MaxBaseFee 20,000,000 gwei] clamp다. txpool은 tip 외에 feeCap >= MinBaseFee + MinTip도 검사한다(legacy는 gasPrice가 feeCap). 비인가 계정의 tip은 tx 값이 아니라 헤더 GasTip으로 대체돼 effectiveGasPrice와 오라클 샘플링에 쓰인다. 인가 계정 tx는 마지막 로그에 AuthorizedTxExecuted 이벤트를 남긴다.'},
 'config_items': ['gas, tip, feeCap 숫자 입력', 'minTip(프로필별 노드 플래그)', 'priceBump 10%', 'MinBaseFee/MaxBaseFee/threshold(StableNet 상수)', '거버넌스 파라미터 참조(WEMIX3)'],
 'implementation_items': ['baseFee 변화·상한·하한 기대값은 FeePolicy 오라클로 체인별 계산한다. 고정 상한 시험은 WBFT에 해당 clamp가 없으므로 공통 대상이 아니다.', 'effectiveGasPrice 기대값은 StableNet에서 송신자 인가 여부와 헤더 GasTip을 읽어야 한다.', 'underpriced 거부 기준은 WEMIX3 거버넌스 값, WBFT miner.gasprice, StableNet MinBaseFee+MinTip으로 다르므로 프로필 입력 준비 단계를 체인별로 둔다.'],
 'evidence': [
  ev(W,'consensus/misc/eip1559.go',81,'_, maxBaseFeeGov, _, baseFeeMaxChangeRate, gasTargetPercentage, err := wemixminer.GetBlockBuildParameters(parent.Number)'),
  ev(W,'consensus/misc/eip1559.go',121,'return math.BigMin(x.Add(parent.BaseFee, baseFeeDelta), maxBaseFee)'), ev(W,'consensus/misc/eip1559.go',142,'common.Big1,'),
  ev(W,'eth/gasprice/gasprice.go',153,'return wemixminer.SuggestGasPrice(), nil'), ev(W,'wemix/miner/miner.go',134,'return big.NewInt(100 * params.GWei)'),
  ev(W,'wemix/admin.go',648,'err := ma.rpcCli.CallContext(ctx, &v, "miner_setGasPrice",'), ev(W,'core/tx_pool.go',697,'if local && params.DropUnderPriced && tx.EffectiveGasTipIntCmp(pool.gasPrice, pool.priced.urgent.baseFee) < 0 {'),
  ev(B,'consensus/misc/eip1559/eip1559.go',83,'return num.Add(parent.BaseFee, baseFeeDelta)'), ev(B,'consensus/misc/eip1559/eip1559.go',93,'return math.BigMax(baseFee, common.Big0)'),
  ev(B,'eth/gasprice/gasprice.go',155,'if oracle.backend.ChainConfig().CroissantBlock != nil && !oracle.backend.ChainConfig().IsCroissant(head.Number) {'), ev(B,'eth/gasprice/gasprice.go',156,'return wpoa.SuggestGasPrice(), nil'),
  ev(B,'core/txpool/validation.go',120,'return fmt.Errorf("%w: gas tip cap %v, minimum needed %v", ErrUnderpriced, tx.GasTipCap(), opts.MinTip)'), ev(B,'eth/backend.go',455,'s.txPool.SetGasTip(price)'),
  ev(B,'miner/miner.go',65,'DefaultGasPriceGWei = 100'), ev(B,'core/txpool/legacypool/legacypool.go',164,'PriceBump: 10,'),
  ev(S,'consensus/misc/eip1559/eip1559.go',62,'if config.AnzeonEnabled() {'), ev(S,'consensus/misc/eip1559/eip1559.go',73,'if maxBaseFee.Cmp(common.Big0) != 0 && baseFee.Cmp(maxBaseFee) > 0 {'), ev(S,'consensus/misc/eip1559/eip1559.go',82,'if baseFee.Cmp(minBaseFee) < 0 {'),
  ev(S,'params/protocol_params.go',132,'IncreasingThreshold uint64 = 20'), ev(S,'params/protocol_params.go',133,'DecreasingThreshold uint64 = 6'), ev(S,'params/protocol_params.go',135,'MinBaseFee uint64 = 20000000000000'), ev(S,'params/protocol_params.go',136,'MaxBaseFee uint64 = 20000000000000000'),
  ev(S,'core/txpool/validation.go',122,'if opts.Config.IsLondon(head.Number) && opts.Config.AnzeonEnabled() {'), ev(S,'core/txpool/validation.go',129,'return fmt.Errorf("%w: gas fee cap %v, minimum needed %v", ErrUnderpriced, tx.GasFeeCap(), minFee)'),
  ev(S,'core/state_transition.go',169,'if statedb != nil && !statedb.IsAuthorized(from) {'), ev(S,'core/state_transition.go',170,'gasTipCap = new(big.Int).Set(headerGasTip)'), ev(S,'core/state_transition.go',592,'if rules.IsAnzeon && st.state.IsAuthorized(msg.From) {'),
  ev(S,'core/types/receipt.go',365,'if err == nil && atEnv.stateReader != nil && !atEnv.stateReader.IsAuthorized(from) && atEnv.headerTip != nil {'), ev(S,'eth/gasprice/gasprice.go',254,'atEnv := types.NewInstantAnzeonTipEnv(signer, header.BaseFee, header.GasTip(), stateReader)'),
  ev(S,'core/txpool/legacypool/legacypool.go',163,'PriceBump: 10,'), ev(W,'core/tx_pool.go',182,'PriceBump: 10,'),
  ev(W,'internal/ethapi/api.go',98,'tipcap.Add(tipcap, head.BaseFee)'), ev(B,'internal/ethapi/api.go',78,'tipcap.Add(tipcap, head.BaseFee)'), ev(S,'internal/ethapi/api.go',77,'tipcap.Add(tipcap, head.BaseFee)')]})
items.append({
 'id': 'CD-A-07', 'area': '계정 정책', 'title': 'StableNet 블랙리스트·인가 상태(extra 비트)',
 'same': '잔액, nonce, 서명 검사는 세 클라이언트가 같다.',
 'differs': {
  'wemix': '계정 상태 비트가 없다. IsBlacklisted/IsAuthorized 호출 지점이 core에 없다.',
  'wbft': '계정 상태 비트가 없다.',
  'stablenet': 'StateAccount에 Extra uint64가 있고 StateDB가 IsBlacklisted/IsAuthorized/SetBlacklisted/SetAuthorized를 제공한다. Anzeon이면 txpool과 실행에서 송신자·수신자·대납자 블랙리스트를 검사해 "blacklisted account: <addr>"를 낸다. 인가 계정은 tip 정책과 수료 로그가 달라진다.'},
 'config_items': ['accountRoles(sender/recipient/feePayer)', 'StableNet 프로필의 blacklistState/authorizedState 사전 조건'],
 'implementation_items': ['공통 송금 fixture는 비차단·비인가 계정이어야 한다. AccountFixture가 StableNet에서 상태를 사전 검사한다.', '블랙리스트·인가 정책 자체의 시험은 StableNet 전용 suite로 둔다.'],
 'evidence': [
  ev(S,'core/types/state_account.go',37,'Extra uint64 `rlp:"optional"`'), ev(S,'core/state/statedb.go',311,'func (s *StateDB) IsBlacklisted(addr common.Address) bool {'), ev(S,'core/state/statedb.go',321,'func (s *StateDB) IsAuthorized(addr common.Address) bool {'),
  ev(S,'core/state/statedb.go',434,'func (s *StateDB) SetBlacklisted(addr common.Address) {'), ev(S,'core/txpool/validation.go',253,'if opts.State.IsBlacklisted(from) {'), ev(S,'core/txpool/validation.go',256,'if to := tx.To(); to != nil && opts.State.IsBlacklisted(*to) {'),
  ev(S,'core/state_transition.go',508,'if st.state.IsBlacklisted(msg.From) {'), ev(S,'core/error.go',147,'return fmt.Sprintf("blacklisted account: %s", e.Address.Hex())')]})
items.append({
 'id': 'CD-A-08', 'area': '오류 문자열', 'title': '거부 경로의 오류 문자열 비교',
 'same': 'sentinel 원문은 세 클라이언트가 같다. "insufficient funds for gas * price + value", "insufficient funds for transfer", "exceeds block gas limit", "transaction underpriced", "replacement transaction underpriced", "nonce too low", "invalid sender", "invalid transaction v, r, s values", "fee delegation: invalid feePayer".',
 'differs': {
  'wemix': 'txpool이 sentinel을 감싸지 않고 그대로 돌려준다. 타입 미지원도 원문만 온다.',
  'wbft': 'txpool이 "%w: ..." 형식으로 문맥을 덧붙인다(nonce, balance, gas tip cap, type rejected). 대납자 잔액은 1차 검사만 원문이다.',
  'stablenet': 'go-wbft와 같은 감싸기에 더해 Anzeon 전용 메시지가 있다. "gas fee cap %v, minimum needed %v"(underpriced), "blacklisted account: <addr>".'},
 'config_items': ['없음. 문자열은 코드에 고정돼 있다.'],
 'implementation_items': ['assertion은 완전 일치가 아니라 sentinel 부분 문자열 포함으로 판정한다.', 'go-wemix에서만 성립하는 "원문 완전 일치" 기대값을 공통에 넣지 않는다.'],
 'evidence': [
  ev(W,'core/error.go',68,'ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")'), ev(B,'core/error.go',72,'ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")'), ev(S,'core/error.go',74,'ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")'),
  ev(W,'core/tx_pool.go',83,'ErrGasLimit = errors.New("exceeds block gas limit")'), ev(B,'core/txpool/errors.go',43,'ErrGasLimit = errors.New("exceeds block gas limit")'), ev(S,'core/txpool/errors.go',43,'ErrGasLimit = errors.New("exceeds block gas limit")'),
  ev(W,'core/tx_pool.go',71,'ErrUnderpriced = errors.New("transaction underpriced")'), ev(B,'core/txpool/errors.go',31,'ErrUnderpriced = errors.New("transaction underpriced")'), ev(S,'core/txpool/errors.go',35,'ErrReplaceUnderpriced = errors.New("replacement transaction underpriced")'),
  ev(W,'core/error.go',48,'ErrNonceTooLow = errors.New("nonce too low")'), ev(B,'core/txpool/validation.go',238,'return fmt.Errorf("%w: next nonce %v, tx nonce %v", core.ErrNonceTooLow, next, tx.Nonce())'), ev(S,'core/txpool/validation.go',262,'return fmt.Errorf("%w: next nonce %v, tx nonce %v", core.ErrNonceTooLow, next, tx.Nonce())'),
  ev(W,'core/tx_pool.go',67,'ErrInvalidSender = errors.New("invalid sender")'), ev(B,'core/txpool/errors.go',27,'ErrInvalidSender = errors.New("invalid sender")'), ev(W,'core/types/transaction.go',35,'ErrInvalidSig = errors.New("invalid transaction v, r, s values")'),
  ev(W,'core/tx_pool.go',721,'return ErrInsufficientFunds'), ev(B,'core/txpool/validation.go',261,'return fmt.Errorf("%w: balance %v, tx cost %v, overshot %v", core.ErrInsufficientFunds, balance, cost, new(big.Int).Sub(cost, balance))'),
  ev(W,'core/tx_pool.go',652,'return ErrTxTypeNotSupported'), ev(B,'core/txpool/validation.go',67,'return fmt.Errorf("%w: type %d rejected, pool not yet in Berlin", core.ErrTxTypeNotSupported, tx.Type())'),
  ev(S,'core/error.go',147,'return fmt.Sprintf("blacklisted account: %s", e.Address.Hex())')]})
items.append({
 'id': 'CD-A-09', 'area': '노드 기본값', 'title': '가스 한도·가스 가격 관련 노드 플래그 기본값',
 'same': 'genesis gasLimit 기본값 4,712,388, txpool.pricelimit 기본 1, txpool.pricebump 기본 10은 같다. --miner.gasprice, --txpool.pricelimit 플래그 이름도 같다.',
 'differs': {
  'wemix': 'miner.gasprice 기본 1 gwei, miner.gaslimit(GasCeil) 기본 30,000,000. txpool AccountSlots 100,000, GlobalSlots 500,000. 실제 최소 가격과 블록 gasLimit은 거버넌스가 덮어쓴다(miner_setGasPrice, GetBlockBuildParameters의 gasLimit).',
  'wbft': 'miner.gasprice 기본 100 gwei, GasCeil 기본 105,000,000. txpool AccountSlots 16, GlobalSlots 5,120.',
  'stablenet': 'go-wbft와 같은 기본값(100 gwei, 105,000,000, 16/5,120). 여기에 MinBaseFee 20,000 gwei 상수가 더해져 실제 최소 feeCap은 플래그보다 높다.'},
 'config_items': ['launch 플래그(miner.gasprice, miner.gaslimit, txpool.pricelimit)를 프로필 launch에 명시', 'genesis gasLimit'],
 'implementation_items': ['WEMIX3는 플래그를 설정해도 거버넌스 계약 값이 우선한다. 기대 최소 가격은 거버넌스 조회로 읽는 어댑터가 필요하다.', '"gas limit 초과 거부" 시험의 기준 gasLimit은 프로필 값이 아니라 head 블록에서 읽는다.'],
 'evidence': [
  ev(W,'eth/ethconfig/config.go',87,'GasCeil: 30000000,'), ev(W,'eth/ethconfig/config.go',88,'GasPrice: big.NewInt(params.GWei),'), ev(W,'core/tx_pool.go',181,'PriceLimit: 1,'), ev(W,'core/tx_pool.go',184,'AccountSlots: 100000,'),
  ev(W,'wemix/miner/miner.go',140,'func GetBlockBuildParameters(height *big.Int) (blockInterval int64, maxBaseFee, gasLimit *big.Int, baseFeeMaxChangeRate, gasTargetPercentage int64, err error) {'),
  ev(B,'miner/miner.go',75,'GasCeil: 105000000,'), ev(B,'miner/miner.go',70,'DefaultGasPrice = big.NewInt(DefaultGasPriceGWei * params.GWei)'), ev(B,'core/txpool/legacypool/legacypool.go',163,'PriceLimit: 1,'), ev(B,'core/txpool/legacypool/legacypool.go',166,'AccountSlots: 16,'),
  ev(S,'miner/miner.go',75,'GasCeil: 105000000,'), ev(S,'miner/miner.go',65,'DefaultGasPriceGWei = 100'), ev(S,'core/txpool/legacypool/legacypool.go',162,'PriceLimit: 1,'),
  ev(W,'params/protocol_params.go',25,'GenesisGasLimit uint64 = 4712388'), ev(B,'params/protocol_params.go',29,'GenesisGasLimit uint64 = 4712388'), ev(S,'params/protocol_params.go',29,'GenesisGasLimit uint64 = 4712388'),
  ev(W,'cmd/utils/flags.go',484,'MinerGasPriceFlag = BigFlag{'), ev(B,'cmd/utils/flags.go',445,'MinerGasPriceFlag = &flags.BigFlag{'), ev(S,'cmd/utils/flags.go',313,'TxPoolPriceLimitFlag = &cli.Uint64Flag{')]})

json.dump(items, open(f'{OUT}/analyses/draft-chain-diff-tx-fee-account.json','w'), indent=1, ensure_ascii=False)

L = []
L.append('# 세 클라이언트 차이 (거래·수수료·계정) — 2026-09-11 현재 코드 기준\n')
L.append('go-wemix(WEMIX3.0), go-wbft(WEMIX4.0), go-stablenet(StableNet)의 현재 작업 트리를 다시 읽고 정리했다. 2026-09-09 자료의 결론과 줄번호는 승계하지 않았다. 인용한 파일은 모두 각 프로젝트의 실제 빌드 선택 파일(`build/<project>-selected.json`)에 들어 있다. 노드를 띄워 실행한 결과가 아니라 정적 분석이다.\n')
L.append('| ID | 분야 | 제목 | 설정 항목 수 | 별도 구현 항목 수 | 근거 수 |')
L.append('|---|---|---|---:|---:|---:|')
for it in items:
    L.append(f"| {it['id']} | {it['area']} | {it['title']} | {len(it['config_items'])} | {len(it['implementation_items'])} | {len(it['evidence'])} |")
L.append('')
for it in items:
    L.append(f"## {it['id']} {it['title']}\n")
    L.append(f"**같은 점.** {it['same']}\n")
    L.append('| 항목 | WEMIX3.0 / go-wemix | WEMIX4.0 / go-wbft | StableNet / go-stablenet |')
    L.append('|---|---|---|---|')
    d = it['differs']
    L.append(f"| 차이 | {d['wemix']} | {d['wbft']} | {d['stablenet']} |")
    L.append('')
    L.append('설정으로 분리할 값:')
    for c in it['config_items']: L.append(f'- {c}')
    L.append('')
    L.append('별도 구현 또는 체인별 처리:')
    for c in it['implementation_items']: L.append(f'- {c}')
    L.append('')
    L.append('근거:')
    for e in it['evidence']:
        L.append(f"- `{e['repo']}/{e['path']}:{e['line']}` — `{e['snippet']}`")
    L.append('')
open(f'{OUT}/analyses/draft-chain-diff-tx-fee-account.md','w').write('\n'.join(L))
print('items', len(items), 'evidence', sum(len(i['evidence']) for i in items))
