# 공통 테스트 분리를 위한 하네스 의존 분석

이미 있는 환경 선언·체인 plugin·계정 label·RPC helper를 재사용하되, 공개 RPC 서명 경로와 합의 판정은 별도 구현/체인별 처리가 필요하다. 이 문서는 실행 결과가 아닌 보존 소스의 정적 분석이다. 메인넷 명칭은 프로젝트 비교 이름이며, 로컬 기본 Chain ID나 엔드포인트를 운영망 설정으로 보증하지 않는다.

## 실행 구조

`suite run → app → testengine(compose 또는 attach) → DSL env 해석 → capability gate → interpreter → testhelper → RPC/accounts` 구조다. compose는 chainsetup과 consensus family가 노드 준비를 담당하며, attach는 기존 endpoint 또는 workspace 기록을 읽는다. `applicableChains`는 필터이고 실제 기능 지원의 증거가 아니다.

## 의존 요소와 변경 범위

### H01 체인 선택·환경 재사용 — 설정 + 실행 조합 생성

현재: EnvV2에 chain/binaries/keys/genesis/topology/hardforks/launch/config/accounts가 이미 있다. env ID 참조와 extends를 지원하지만 extends는 최상위 필드 단위 얕은 병합이다. compose --chain은 선언과 일치해야 하며 자동 변환하지 않는다.

변경 범위: 공통 case의 단계와 체인 env를 분리하고 3종 env와 실행 조합을 생성한다. 기존 JSON에 inline env가 있어 파일 정리가 필요하다. topology/launch 부분 덮어쓰기로 나머지가 사라지지 않도록 전체 필드를 전달한다. 임의 step 변수 주입을 env가 이미 지원한다고 간주하지 않는다.

근거:

- `sources/chainbench/internal/dsl/spec_v2.go:56` — `SchemaVersion string `json:"schemaVersion"``
- `sources/chainbench/internal/dsl/spec_v2.go:263` — `// The override form is a shallow top-level merge: each field the case names`
- `sources/chainbench/internal/dsl/files.go:30` — `b, err = InlineEnv(b, func(id string) ([]byte, error) {`
- `sources/chainbench/internal/testengine/compose.go:88` — `if in.Chain != "" && in.Chain != chain {`

### H02 applicableChains·requires — 설정 + capability 모델 보강

현재: applicableChains가 없으면 모든 체인으로 판정한다. 실제 동작 호환성을 검사하는 것이 아니다. env.capabilities도 실행 requires에 합쳐지며 plain RPC attach는 rpc만 기본 제공한다.

변경 범위: 공통 후보의 allowlist와 필요한 rpc/ws/process/consensus를 검토한다. tx type·hardfork·특정 RPC의 지원은 실제 노드와 소스로 확인하는 preflight가 필요하다. manifest tx_types는 3체인 모두 같은 목록이므로 지원 증거로 쓰지 않는다.

근거:

- `sources/chainbench/internal/testengine/capability.go:17` — `return func(s dsl.Spec) bool {`
- `sources/chainbench/internal/testengine/capability.go:52` — `func applicableWithCaps(chain string, provided []string) func(dsl.Spec) bool {`
- `sources/chainbench/internal/dsl/spec_v2.go:386` — `// The env's capabilities and the case's requires are both gating inputs, so`
- `sources/chainbench/internal/testengine/attach.go:201` — `PreSpec:    cfg.PreSpec,`
- `sources/chainbench/internal/chains/wemix/manifest.json:20` — `"probe": { "method": "wemix_getReward" },`

### H03 테스트 계정·서명 주체 — 설정 + 일부 signer 구현

현재: 키셋 label과 faucet alias, 생성 계정, 로컬 서명 sendTx가 이미 있다. faucet은 node1로 해석된다. 기존 기본 송금은 from 주소 literal과 preset keys 경로를 고정한다.

변경 범위: 주소 literal을 역할 label로 바꾸고 체인 env별 자금 계정·잔액·키 보관 참조를 지정한다. 실제 운영 계정/키 값은 보고서에 넣지 않는다. 격리 테스트에서는 node signing 가능, 공개 RPC에서는 로컬 signer가 필요하다.

근거:

- `sources/chainbench/internal/testhelper/account.go:71` — `func ResolveAccount(d *interp.Deps, ref string) (Account, error) {`
- `sources/chainbench/internal/testhelper/account.go:82` — `label := ref`
- `sources/chainbench/internal/testhelper/builtins.go:177` — `if sender.SignsLocally() {`
- `sources/chainbench/tests/tc/basic/05-basic-tx-send.json:42` — `주소 등 literal 필드 존재(값 생략)`

### H04 deploy/load/faucet/register의 서명 — 별도 공통 구현

현재: funder는 계정 주소만 반환하며 sendAndConfirm은 eth_sendTransaction으로 보낸다. 이 경로는 label에 로컬 키가 있어도 로컬 서명을 선택하지 않는다.

변경 범위: 공개 RPC에서 계약 배포·가스 부하·faucet·registerContract를 수행하려면 공통 transaction sender를 재사용하도록 통합하고 contract creation과 data/gas/fee/nonce를 모두 전달한다. from 설정만 변경하면 해결된다는 판정은 금지한다.

근거:

- `sources/chainbench/internal/testhelper/assets.go:190` — `func funder(ctx context.Context, c *rpc.Client, d *interp.Deps, args map[string]any) (string, error) {`
- `sources/chainbench/internal/testhelper/assets.go:210` — `func sendAndConfirm(ctx context.Context, c *rpc.Client, args rpc.SendTxArgs, opts map[string]any) (map[string]any, string, error) {`
- `sources/chainbench/internal/core/rpc/client.go:371` — `func (c *Client) SendTransaction(ctx context.Context, args SendTxArgs) (string, error) {`
- `sources/chainbench/internal/testhelper/load.go:70` — `if err := applyFeeArgs(&args, ac.Args); err != nil {`
- `sources/chainbench/internal/testhelper/assets.go:160` — `if g, ok := hexQuantity(ac.Args["gas"]); ok {`

### H05 RPC/WS endpoint·노드 역할 — 설정 + live profile 보강

현재: --rpc는 반복 입력 가능하고 workspace attach는 저장된 node roles와 capabilities를 재사용한다. bare endpoint에서는 producer/endpoint 역할을 알 수 없으며 en1 fallback이 node1을 가리킬 수 있다.

변경 범위: RPC/WS, 인증 참조, provider 제한, node role을 환경 프로필로 관리한다. 주소 나열만으로 bp1/en1 의미를 보장하지 않는다. 공개 provider의 admin/personal/debug 허용 여부를 명시한다. live endpoint의 실제 값은 별도 운영 설정에서 주입한다.

근거:

- `sources/chainbench/cmd/chainbench/suitecmd/run.go:114` — `"compose: where node identities come from — preset (use --keys as-is) | generate (create a fresh set in --keys)")`
- `sources/chainbench/internal/testengine/attach.go:51` — `// node: "en1" means the first node whose role is en, while "node1" means`
- `sources/chainbench/internal/testengine/attach.go:174` — `if len(cfg.Nodes.Nodes) > 0 {`
- `sources/chainbench/internal/testhelper/derived.go:398` — `func wsTargetURL(nodes []node.Node) (string, error) {`

### H06 Chain ID/Network ID/바이너리 — 설정 + 검증

현재: manifest마다 binary/build/chain_id/network_id가 있고 compose CLI가 chain-id/network-id/binary를 받는다. manifest 값은 하네스 기본값이다. remote-chain-info는 chainId>0만 검사한다. 보존 go-wbft Makefile의 실제 target은 gwemix이고 산출물은 build/bin/gwemix인데, chainbench wbft manifest는 binary 및 make_target를 gwbft로 기록한다. 현재 suite는 자동 빌드하지 않으며 MakeTarget의 실행 소비처는 없다. 따라서 make gwbft 실행 실패가 아니라 gwbft 이름을 실행하려는 경로의 탐색/실행 실패 가능성이 문제다.

변경 범위: WEMIX3.0=wemix, WEMIX4.0=wbft, StableNet=stablenet 비교에서는 각 실제 실행 프로필의 expected chainId/genesisHash/clientVersion을 기록하고 RPC와 대조한다. 저장소/manifest 숫자를 운영 메인넷 값으로 단정하지 않는다. --chain-id는 attach 대상 체인의 ID를 바꾸지 않는다. 일반 단일 바이너리 suite는 --binary 또는 env.binaries.default에 go-wbft에서 빌드한 gwemix의 절대경로를 넣어 우회할 수 있다. per-node/role binary가 선언된 경우 그 매핑도 바꿔야 한다. 기존 explicit 경로 제공 테스트와 RPC attach에는 이 이름 불일치가 직접 적용되지 않는다. 별도로 내장 wbft manifest의 binary/build.make_target 메타데이터와 기본 env 이름을 실제 저장소 산출물에 맞추는 교정이 필요하다. go-wemix 산출물도 gwemix이므로 파일명만으로 체인을 구별하지 말고 절대경로와 소스 revision을 기록한다.

근거:

- `sources/chainbench/internal/chains/wemix/manifest.json:3` — `"binary": "gwemix",`
- `sources/chainbench/internal/chains/wbft/manifest.json:3` — `"binary": "gwbft",`
- `sources/chainbench/internal/chains/stablenet/manifest.json:3` — `"binary": "gstable",`
- `sources/chainbench/cmd/chainbench/suitecmd/run.go:127` — `return cmd`
- `sources/chainbench/tests/tc/remote/02-remote-chain-info.json:35` — `"is": "0"`
- `sources/go-wbft/Makefile:17` — `gwemix:`
- `sources/go-wbft/Makefile:20` — `@echo "Run \"$(GOBIN)/gwemix\" to launch gwemix."`
- `sources/chainbench/internal/chains/wbft/manifest.json:8` — `"build": { "repo": "go-wbft", "make_target": "gwbft" },`
- `sources/chainbench/cmd/chainbench/suitecmd/run.go:29` — `// handoff env composes the handoff) and runs against that; with --rpc it`
- `sources/chainbench/internal/testengine/compose.go:153` — `binary := in.Binary`
- `sources/chainbench/internal/testengine/compose.go:155` — `binary = expand(spec.Chain.Binary)`
- `sources/chainbench/internal/app/upgrade.go:123` — `path, err := exec.LookPath(name)`
- `sources/chainbench/internal/chainsetup/steps_lifecycle.go:38` — `if path := w.state.Binaries[ns.Binary]; path != "" {`

### H07 일반 컨트랙트·시스템 컨트랙트 — 설정 + 체인 adapter

현재: 일반 deployContract의 주소 저장과 $binding, registerContract 및 ABI calldata 조립 기능이 있다. 시스템 governance는 별도의 체인 capability catalog/handler로 존재한다.

변경 범위: 일반 계약은 같은 지원 EVM revision으로 컴파일한 bytecode를 배포하고 얻은 주소를 binding한다. 이미 배포된 계약은 address와 runtime code hash/ABI/version을 프로필에 둔다. 체인 시스템 계약은 주소만 바꾸지 말고 ABI·권한·이벤트·상태변경 의미를 adapter로 분리한다.

근거:

- `sources/chainbench/internal/testhelper/assets.go:85` — `func (deployContractAction) Do(ctx context.Context, ac *interp.ActionCtx) error {`
- `sources/chainbench/internal/testhelper/assets.go:141` — `func (registerContractAction) Do(ctx context.Context, ac *interp.ActionCtx) error {`
- `sources/chainbench/internal/dsl/interp/binding.go:73` — `func resolveString(s string, b Bindings) (any, error) {`
- `sources/chainbench/internal/testhelper/derived.go:487` — `func deriveAbiCall(spec map[string]any) (any, error) {`
- `sources/chainbench/internal/chains/stablenet/caps.go:2` — `// stablenet chain (its governance system contracts), separate from the common`

### H08 수수료·nonce·transaction type — 설정 + signer/판정 보강

현재: node-signed 경로는 gasPrice/maxFee/tip/nonce를 전달한다. 로컬 sendTx는 단순 송금 분기만 명시 dynamic fee/nonce를 처리하고 gas를 21000으로 둔다. calldata/fee delegation 분기는 별도 Wallet 메서드를 호출한다.

변경 범위: 공통 sender가 모든 경로에서 txType/gas/nonce/fee를 명시적으로 전달하도록 한다. baseFee 하한·underpriced 즉시거절 여부·replacement bump·fee payer 정책은 체인 규칙별 기대값으로 분리한다. 저수수료가 무조건 RPC 거절이어야 한다는 helper 주석을 세 체인의 규칙으로 사용하지 않는다.

근거:

- `sources/chainbench/internal/testhelper/assets.go:234` — `func applyFeeArgs(args *rpc.SendTxArgs, in map[string]any) error {`
- `sources/chainbench/internal/testhelper/builtins.go:278` — `// account signs as the sender (moves value) while feePayerKey covers the gas.`
- `sources/chainbench/internal/testhelper/builtins.go:294` — `return fmt.Errorf("dsl: sendTx: data: %w", derr)`
- `sources/chainbench/internal/testhelper/builtins.go:318` — `주소 등 literal 필드 존재(값 생략)`

### H09 genesis·bootstrap·fork 활성화 — 기존 adapter 재사용 + 설정

현재: wemix plugin은 poa/WeMix protocol, wbft와 stablenet은 wbft family와 각 accounts protocol을 선택한다. WEMIX는 governance/etcd 단계가 있고 WBFT family는 genesis validators/BLS/extraData를 사용한다.

변경 범위: 공통 테스트에서 시작/준비를 직접 구현하지 말고 기존 chainsetup/consensus family에 위임한다. 각 env는 hardfork와 genesis overlay, 바이너리·빌드 revision을 제공한다. go-wemix genesis를 단순 WBFT template 치환으로 만들지 않는다.

근거:

- `sources/chainbench/internal/chains/wemix/wemix.go:34` — `func (p plugin) Family() registry.ConsensusFamily { return poa.New() }`
- `sources/chainbench/internal/chains/wbft/wbft.go:33` — `func (p plugin) Family() registry.ConsensusFamily { return wbftfam.New() }`
- `sources/chainbench/internal/chains/stablenet/stablenet.go:34` — `func (p plugin) Family() registry.ConsensusFamily { return wbft.New() }`
- `sources/chainbench/internal/consensus/poa/poa.go:42` — `"poa: a wemix genesis is generated by the binary from a governance config, not substituted from a template — use the wemix genesis source (PrepareTemplate then GenerateGenesis)")`
- `sources/chainbench/internal/consensus/poa/bootstrap.go:19` — `func BootstrapPlan() []Step {`
- `sources/chainbench/internal/consensus/wbft/wbft.go:25` — `func (Family) BuildGenesis(template []byte, p registry.GenesisParams) ([]byte, error) {`

### H10 합의·quorum·보상·finality — 별도 체인 판정

현재: blockAdvance/blockHalt/interval 같은 관찰기는 재사용 가능하다. derive quorum은 ceil(2n/3)을 계산한다. 각 manifest의 consensus RPC namespace가 다르다.

변경 범위: 관찰 방법은 공유하고 장애 허용 수, 중단/회복 조건, finality 기준, 보상 수령/분배 및 fork별 정책은 별도 adapter/oracle로 둔다. WEMIX etcd 가용성과 WBFT 검증인 quorum에 동일 공식을 적용하지 않는다. same latest hash만으로 합의 안정성을 증명하지 않는다.

근거:

- `sources/chainbench/internal/testhelper/blockprobe.go:43` — `func (blockHaltAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {`
- `sources/chainbench/internal/testhelper/derived.go:462` — `func deriveQuorum(spec map[string]any) (any, error) {`
- `sources/chainbench/internal/consensus/poa/poa.go:26` — `func (Family) RPCNamespace() string     { return "wemix" }`
- `sources/chainbench/internal/consensus/wbft/wbft.go:20` — `func (Family) RPCNamespace() string     { return "istanbul" }`

### H11 장애/재시작/partition 실행 권한 — 격리 환경 전용 + adapter

현재: faultTarget은 NodeControl이 없으면 오류를 반환한다. plain attach는 기본적으로 process를 소유하지 않는다. partition은 admin_removePeer로 기존 연결을 끊는다.

변경 범위: 장애·restart·genesis 변경·대량 부하는 운영 mainnet attach 후보에서 분리한다. 소유한 격리 로컬/원격 네트워크의 process controller와 topology만 사용한다. partition은 discovery/reconnect를 통제하고 실제 단절을 관찰해야 한다. workspace attach도 자동 process 제어 가능으로 간주하지 말고 Control 주입을 확인한다.

근거:

- `sources/chainbench/internal/testhelper/fault.go:341` — `func faultTarget(ac *interp.ActionCtx, action string) (node.Node, interp.NodeControl, error) {`
- `sources/chainbench/internal/testengine/attach.go:81` — `// Control, when non-nil, lets fault steps (stopNode/startNode/restartNode)`
- `sources/chainbench/internal/testhelper/fault.go:370` — `// left alone.`
- `sources/chainbench/internal/testhelper/fault.go:397` — `if err := c.RemovePeer(ctx, enodes[to.Index]); err != nil {`

### H12 부하·시간·동기화 판정 — 설정 + 일부 새 구현

현재: load는 블록마다 gas-burn 계약 생성 한 건을 제출하고 receipt를 기다린다. stress-tx-flood는 fillPercent=30, blocks=15와 blockAdvance를 검사한다. 이름과 달리 TPS 측정/대량 transaction byte 부하가 아니다.

변경 범위: 블록 시간·timeout·polling·funding budget은 env profile에서 정의하되 현재 step 필드에 주입할 generator/parameter 기능을 마련한다. TPS/txpool saturation/large-byte block 검증을 원하면 별도 generator와 측정 oracle가 필요하다. full/snap sync 경로 증명에는 노드 설정·진행 상태 관찰을 추가한다.

근거:

- `sources/chainbench/internal/testhelper/load.go:63` — `var lastHash string`
- `sources/chainbench/internal/testhelper/load.go:74` — `if err != nil {`
- `sources/chainbench/tests/tc/stress/02-stress-tx-flood.json:40` — `},`
- `sources/chainbench/internal/testhelper/blockprobe.go:26` — `defaultBlockHaltMaxAdv = 1`

## 변경 순서와 산출물

1. **기존 JSON 정리:** 공통 steps와 3종 env를 분리하고 계정 literal/바이너리/토폴로지/fork/기대 chainId를 환경별 참조로 바꾼다. env extends의 얕은 병합을 지킨다.
2. **환경 프로필:** 계정 역할·RPC/WS·실제 Chain ID·계약 주소/코드 hash·수수료/시간 정책·실행 권한을 선언한다. 현재 EnvV2에 없는 임의 step parameter는 generator 또는 명시적 schema 확장 대상으로 기록한다.
3. **공통 구현 보강:** 로컬 서명으로 contract creation/data/gas/fee/nonce를 모두 다루는 transaction sender를 기존 sendTx/Wallet 경로와 통합한다. 배포·faucet·load·register가 이를 공유하도록 한다.
4. **체인 adapter:** consensus/bootstrap은 기존 family를 재사용하고 governance/보상/특수 tx/finality/fee-policy oracle을 분리한다.
5. **실행 matrix:** read-only live, 상태변경 가능한 전용 테스트 계정, 소유한 격리 네트워크 세 범위를 구분한다. 동일 의미의 공통 test id에 chain/env/build/fork/profile/skip reason을 기록한다.

소스 수정과 실제 검증 실행은 이 분석 범위에 포함하지 않았다. 모든 항목은 추후 구현 범위이며, 기존 기능이 있다고 표시한 부분도 세 체인 런타임 통과를 뜻하지 않는다.
