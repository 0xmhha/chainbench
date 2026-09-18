# 하네스 실행 구조와 메인넷 의존 (2026-09-11)

> 생성 스크립트: `scripts/build_harness_draft.py`. 인용 줄의 코드 조각은 생성 시점에 현재 파일에서 읽어 넣었으므로 줄이 밀리면 생성이 실패한다. 근거 표기의 `chainbench/...`는 chainbench 저장소 루트 기준 경로다.


chainbench HEAD cc501aed 작업 트리를 다시 읽고 썼다. 2026-09-09 분석의 결론과 줄번호는 승계하지 않았다. 모든 인용은 현재 파일에서 다시 확인했다. 실행하지 않은 정적 분석이다.

## 요약표

| ID | 영역 | 현재 | 오늘 설정으로 되는 것 | 추가 구현 |
|---|---|---|---|---|
| H-01 | 실행 경로 (suite run → app → testengine → dsl → interp → testhelper) | `suite run`은 --workspace-dir(compose), --workspace-dir --attach(워크스페이스 attach), --rpc(bare attach) 세 갈래로 나뉜다. --rpc와 --workspace-dir은 함께 쓸 수 없다. | compose: chain, binary, keys, validators, chain-id, network-id, launch-opt, workspace-config, server/docker | 세 체인을 한 번에 도는 실행 행렬이 없다. 한 suite는 한 chain, 한 composition만 허용하므로 공통 케이스는 체인별 env 3벌과 별도 호출 3회가 필요하다. |
| H-02 | 환경 선언(EnvV2) 필드와 런타임 효과 | EnvV2 필드: target, chain, manifest, genesisTemplate, binaries, keys, blueprint, genesis, topology, hardforks, launch, config, capabilities, accounts, upgrade. CaseV2는 applicableChains, requires, on, timeouts, hooks, steps를 가진다. | 테스트 계정 역할: keys(nodekeys) + accounts{label: fund} + newAccount 스텝 | step 상수(가스, 수수료, timeout, 주소, 기대 chainId)를 env에서 주입하는 문법이 없다. 공통 케이스가 체인별 기대값을 갖게 하려면 파라미터 바인딩(예: env.params → $binding) 또는 케이스 생성기가 필요하다. |
| H-03 | 체인 플러그인과 manifest | 세 manifest는 binary(gwemix/gwbft/gstable), chain_id(8285/8284/8283), network_id, consensus_family(poa/wbft/wbft), genesis.hardforks, consensus.rpc_namespace(wemix/istanbul/istanbul), tx_types(세 체인 동일 목록), capabilities(세 체인 동일: process,rpc,ws,consensus)를 담는다. | --manifest/--genesis-template(env.manifest/genesisTemplate)로 외부 manifest를 쓸 수 있다 | wbft manifest의 binary/make_target를 실제 산출물 이름(gwemix)에 맞추거나, 이름과 경로를 분리해 기록하는 교정이 필요하다. go-wemix 산출물도 gwemix이므로 파일명으로 체인을 구분하면 안 된다. |
| H-04 | 합의 family와 handoff | poa(wemix): RPC namespace wemix, validators는 governance 계약을 eth_call로 열거한다(getMemberLength/getMember). BuildGenesis는 거부하고 바이너리가 governance config로 genesis를 생성한다. 기동은 boot → join-nodeN(역순) → endpoints 단계다. etcd 때문에 포트 span 3이다. | env.chain으로 family 선택 | 장애 허용 수, 정족수, 블록 주기 같은 합의 기대값을 family가 제공하지 않는다. 케이스가 숫자를 직접 쓴다(derive quorum은 ceil(2n/3) 고정). PoA/etcd 장애 조건은 별도 판정이 필요하다. |
| H-05 | 계정과 서명 경로 | ResolveAccount: 0x 리터럴은 주소만, node<N>은 노드 keystore 서명(키 없음), 그 외 라벨은 키셋의 키로 로컬 서명. faucet은 node1의 별칭이다. 키셋이 없으면 라벨은 오류다. | 격리 네트워크: node 라벨, faucet, env.accounts, newAccount+key, 로컬 라벨(dev1 등) | 공개 RPC(언락 계정 없음)에서 공통 케이스를 돌리려면 faucet/deployContract/registerContract/load와 env.accounts 자금 지원을 로컬 서명으로도 보낼 수 있어야 한다. 현재는 eth_sendTransaction 고정이다. |
| H-06 | RPC endpoint와 노드 역할 | rpc.Client는 http.DefaultClient로 JSON-RPC over HTTP를 보낸다. timeout은 ctx로만, 인증 헤더는 없다(Content-Type만). | --rpc 반복으로 여러 endpoint | attach 프로필: endpoint별 역할(bp/en/pn), HTTP/WS URL, metrics URL, 인증 헤더, 허용 namespace, 기본 timeout을 선언하고 노드표로 만드는 기능. |
| H-07 | 장애 주입과 process control | faultTarget은 Deps.Nodes(NodeControl)가 없으면 'attach mode does not own the node processes'로 실패한다. control은 워크스페이스 compose/attach에서만 workspaceNodes로 주입된다. bare attach와 handoff는 nil이다. | compose/워크스페이스 attach: stopNode/startNode/restartNode/swapNode/readNodeLog/partition | 운영 메인넷 attach에서는 장애 케이스를 돌릴 수 없다. 공통 목록에서 실행 환경(격리/운영)을 분리해야 한다. |
| H-08 | 체인 정책이 스며든 assertion/observer | sendTx expect:reject는 '전송 오류가 있으면' 통과다. reason은 오류 문자열 부분 일치다. expect:revert는 receipt status 0x0이다. | compare/delta로 비교 방식 | 체인별 기대값(정족수, 블록 주기, 최소 수수료, 거부 사유, effectiveGasPrice 공식)을 케이스 밖 프로필에서 주입하는 장치가 없다. FeePolicy/ConsensusPolicy oracle 또는 파라미터 바인딩이 필요하다. |
| H-09 | capability 모델과 preflight | applicableChains가 비면 모든 체인에 적용된다. requires ⊆ provided일 때만 실행, 아니면 skip. skip 사유는 한 문장 고정이다. | applicableChains, requires, env.capabilities, --caps, genesis overlay의 capabilities | 체인별 기능 capability(예: tx-0x04, p256, fee-delegation, istanbul-rpc, wemix-rpc, account-extra, blacklist)를 manifest 또는 실제 노드 probe로 제공하고 케이스가 requires로 선언하는 모델. |
| H-10 | 실행 기록과 메타데이터 | session.json에 command/startedAt, env.json에 envId/fingerprint/dataPath/nodes, 테스트별 steps/assert/status/reason/artifacts.json이 남는다. report.json은 session/command/startedAt/summary/tests다. | --json 요약, artifact-root, dashboard 스트림 | 실행 행렬용 메타데이터: chain, chainId(RPC 확인값), 바이너리 경로+sha256+버전(web3_clientVersion), fork 활성 상태, signer 모드(node/local), attach 프로필 id, skip/blocked 사유 코드. 세 체인 결과를 같은 case id로 묶는 키. |

## 2026-09-09 결론과 달라진 점

- genesis mode existing이 생겼다. 워크스페이스 config의 prepared 프리셋이 genesis/keyring/configs를 공급한다. 이전 분석의 'template만 지원' 서술은 더 이상 맞지 않는다.
- metrics endpoint가 노드표에 MetricsURL로 기록된다. 이전 분석에서 metric assertion이 컨테이너 내부 주소를 찍던 문제는 해결됐다.
- handoff가 서버 세트와 노드별 machine을 받는다. 다만 테스트 엔진 관점에서는 여전히 노드표와 process control이 없다.
- 하네스 코드에 체인 이름 분기는 없다. 이전 분석이 우려한 '체인별 하드코딩'은 testhelper가 아니라 케이스 JSON과 SDK 프로토콜(세 체인 동일 tx type 목록)에 있다.
- sendTx 로컬 서명의 gas 21000 고정, faucet/deploy/load의 node-signed 고정은 그대로다. 이전 분석의 H04/H08 지적은 여전히 유효하다.
- wbft manifest의 gwbft와 go-wbft Makefile의 gwemix 불일치도 그대로다. tests/tc의 wbft 케이스 8건이 bare 'gwbft'를 선언하므로 PATH에 그 이름의 파일이 있어야 한다.

## H-01 실행 경로 (suite run → app → testengine → dsl → interp → testhelper)

현재:

- `suite run`은 --workspace-dir(compose), --workspace-dir --attach(워크스페이스 attach), --rpc(bare attach) 세 갈래로 나뉜다. --rpc와 --workspace-dir은 함께 쓸 수 없다.
- compose 경로: dsl.ReadFiles가 env 참조를 인라인하고, Parse → sameChain → sameComposition → Precheck → compositionOf → composeWorkspace 또는 handoffUp → wiredAttachEngine 순으로 간다.
- --chain은 attach에서 필수이고 compose에서는 spec의 env.chain과 같아야 한다. --binary/--keys/--keys-source/--validators/--chain-id/--network-id/--launch-opt는 compose 전용 override다.
- --workspace-config는 target dataRoot와 prepared inputs 프리셋(genesis/keyring/configs)을 공급한다. 같은 DSL을 파일 교체만으로 다른 target에 돌리는 장치다.
- attach 엔진은 accounts.ForChain(chain)으로 SDK 프로토콜을 고르고, 키셋(ringFor)과 testhelper.Registry()를 interp.Deps에 넣는다. 체인 이름이 SDK에 없으면 attach 자체가 거부된다.

오늘 설정으로 되는 것:

- compose: chain, binary, keys, validators, chain-id, network-id, launch-opt, workspace-config, server/docker
- attach: chain, rpc(반복), keys(라벨 해석용), dashboard
- 여러 정의를 순서대로 돌리면 네트워크를 유지하고 preflight가 재사용 여부를 정한다

세 체인 분리에 추가로 필요한 구현:

- 세 체인을 한 번에 도는 실행 행렬이 없다. 한 suite는 한 chain, 한 composition만 허용하므로 공통 케이스는 체인별 env 3벌과 별도 호출 3회가 필요하다.
- attach는 chain 이름만 받고 실제 노드의 chainId/fork/버전을 대조하지 않는다. 공개 RPC에 붙일 때 chain 오지정을 잡는 preflight가 없다.

근거:

- `chainbench/cmd/chainbench/suitecmd/run.go:66` — `case attach && len(rpcURLs) > 0:`
- `chainbench/cmd/chainbench/suitecmd/run.go:76` — `case len(rpcURLs) > 0 && workspaceDir != "":`
- `chainbench/cmd/chainbench/suitecmd/run.go:113` — `cmd.Flags().StringVar(&chain, "chain", "", "chain id (e.g. stablenet); required to attach, with --workspace-dir it must agree with what the specs decl`
- `chainbench/cmd/chainbench/suitecmd/run.go:115` — `cmd.Flags().StringVar(&workspaceConfig, "workspace-config", "", "compose: environment file owning the target dataRoot and its purpose directories; the`
- `chainbench/cmd/chainbench/suitecmd/run.go:119` — `cmd.Flags().StringArrayVar(&rpcURLs, "rpc", nil, "attach: node RPC URL (repeatable) — runs against a live network")`
- `chainbench/cmd/chainbench/suitecmd/run.go:122` — `cmd.Flags().StringVar(&binary, "binary", "", "compose: node binary path, overriding what the specs declare")`
- `chainbench/cmd/chainbench/suitecmd/run.go:123` — `cmd.Flags().StringVar(&keysDir, "keys", "keys/preset", "compose: key set directory, overriding what the specs declare")`
- `chainbench/cmd/chainbench/suitecmd/run.go:129` — `cmd.Flags().Int64Var(&chainID, "chain-id", 0, "compose: override the chain id in the built genesis (0 = declared/manifest)")`
- `chainbench/internal/dsl/files.go:30` — `b, err = InlineEnv(b, func(id string) ([]byte, error) {`
- `chainbench/internal/testengine/suite.go:257` — `if err := sameChain(parsed); err != nil {`
- `chainbench/internal/testengine/suite.go:260` — `if err := sameComposition(parsed); err != nil {`
- `chainbench/internal/testengine/suite.go:266` — `if err := Precheck(parsed); err != nil {`
- `chainbench/internal/testengine/suite.go:269` — `comp, err := compositionOf(ctx, parsed[0], in)`
- `chainbench/internal/testengine/suite.go:319` — `eng, err := wiredAttachEngine(sd, net, attachWiring{`
- `chainbench/internal/testengine/compose.go:93` — `if in.Chain != "" && in.Chain != chain {`
- `chainbench/internal/testengine/attach.go:156` — `accts, err := accounts.ForChain(cfg.Chain)`
- `chainbench/internal/testengine/attach.go:165` — `run := NewRunSpec(interp.Deps{`
- `chainbench/internal/app/workflow.go:97` — `if in.DataDir != "" {`
- `chainbench/internal/app/workflow.go:109` — `eng, err := testengine.NewAttachEngine(testengine.AttachConfig{`

## H-02 환경 선언(EnvV2) 필드와 런타임 효과

현재:

- EnvV2 필드: target, chain, manifest, genesisTemplate, binaries, keys, blueprint, genesis, topology, hardforks, launch, config, capabilities, accounts, upgrade. CaseV2는 applicableChains, requires, on, timeouts, hooks, steps를 가진다.
- binaries.default 하나면 단일 바이너리, 그 외 키는 역할별/노드표 참조다. 값은 os.Expand로 $VAR, ${VAR:-default}를 치환한다.
- genesis.mode는 template(set/overlay 병합)와 existing(ref 파일 그대로)만 지원한다. hardforks는 {fork: height}를 <fork>Block=<n> override로 바꾼다.
- accounts는 체인 기동 후 node1이 eth_sendTransaction으로 자금을 보내 만드는 테스트 계정이다. genesis에 넣지 않는다.
- env.capabilities는 case.requires와 합쳐져 gate 입력이 된다. 즉 env가 선언한 capability는 '필요 조건'이지 '제공 조건'이 아니다.
- env 참조("env": "<id>")는 <id>.env.json을 case 파일 상위 8단계까지 찾는다. extends는 최상위 필드 단위 얕은 병합이다.
- upgrade env는 profile/template와 producer/validator 바이너리만 받고 hardforks/topology/launch/config/genesis override를 거부한다.

오늘 설정으로 되는 것:

- 테스트 계정 역할: keys(nodekeys) + accounts{label: fund} + newAccount 스텝
- RPC endpoint: compose에서는 topology와 target이 정하고, attach에서는 --rpc가 정한다. env에는 endpoint URL을 적는 자리가 없다
- Chain ID: env에는 없고 manifest chain_id 또는 --chain-id/genesis.set(config.chainId)으로 정한다
- fork: hardforks, genesis.set/overlay, delayed-<fork> capability
- topology: bp/en/pn 수, syncMode, nodes[] 표(role/binary/sync/bootnode/config/key)
- binary: binaries.default 또는 역할별, ${VAR:-default}
- launch/config: 노드 실행 플래그와 TOML 값 (scope: all/role/nodeN)

세 체인 분리에 추가로 필요한 구현:

- step 상수(가스, 수수료, timeout, 주소, 기대 chainId)를 env에서 주입하는 문법이 없다. 공통 케이스가 체인별 기대값을 갖게 하려면 파라미터 바인딩(예: env.params → $binding) 또는 케이스 생성기가 필요하다.
- 외부 RPC의 인증 헤더/토큰, 노드 역할(bp/en) 선언, WS URL, metrics URL을 env에 적을 수 없다. attach 프로필 스키마가 필요하다.
- contract address를 env에 선언할 수 없다. 배포 결과 $binding만 있다. 기배포 계약 주소/ABI/코드해시 레지스트리가 필요하다.
- extends가 얕은 병합이므로 topology 한 키만 바꾸면 나머지 topology가 사라진다. 공통 env를 체인별로 파생할 때 전체 필드를 다시 써야 한다.

근거:

- `chainbench/internal/dsl/spec_v2.go:64` — `Chain       string `json:"chain"``
- `chainbench/internal/dsl/spec_v2.go:70` — `Binaries        map[string]string `json:"binaries,omitempty"``
- `chainbench/internal/dsl/spec_v2.go:76` — `Genesis      *GenesisV2                `json:"genesis,omitempty"``
- `chainbench/internal/dsl/spec_v2.go:78` — `Hardforks    map[string]int            `json:"hardforks,omitempty"``
- `chainbench/internal/dsl/spec_v2.go:81` — `Capabilities []string                  `json:"capabilities,omitempty"``
- `chainbench/internal/dsl/spec_v2.go:87` — `Accounts map[string]AccountV2 `json:"accounts,omitempty"``
- `chainbench/internal/dsl/spec_v2.go:91` — `Upgrade *UpgradeV2 `json:"upgrade,omitempty"``
- `chainbench/internal/dsl/spec_v2.go:101` — `Fund string `json:"fund,omitempty"``
- `chainbench/internal/dsl/spec_v2.go:189` — `ApplicableChains string            `json:"applicableChains,omitempty"``
- `chainbench/internal/dsl/spec_v2.go:268` — `// The override form is a shallow top-level merge: each field the case names`
- `chainbench/internal/dsl/spec_v2.go:321` — `for k, v := range envObj {`
- `chainbench/internal/dsl/spec_v2.go:401` — `if len(env.Capabilities) > 0 {`
- `chainbench/internal/dsl/spec_v2.go:420` — `if b, ok := env.Binaries["default"]; ok && len(env.Binaries) == 1 {`
- `chainbench/internal/dsl/spec_v2.go:456` — `case "existing":`
- `chainbench/internal/dsl/spec_v2.go:468` — `return Spec{}, fmt.Errorf("dsl: case %s: genesis mode %q has no runtime boundary yet (supported: template, existing)", c.ID, g.Mode)`
- `chainbench/internal/dsl/files.go:13` — `const maxEnvSearchDepth = 8`
- `chainbench/internal/testengine/compose.go:70` — `return os.Expand(s, func(name string) string {`
- `chainbench/internal/testengine/compose.go:137` — `if len(spec.Hardforks) > 0 || len(spec.Topology) > 0 || len(spec.EnvLaunch) > 0 || len(spec.EnvConfig) > 0 {`
- `chainbench/internal/testengine/compose.go:540` — `out = append(out, fmt.Sprintf("%sBlock=%d", name, forks[name]))`
- `chainbench/internal/testengine/suite.go:311` — `funder, ok := ring.Get("node1")`
- `chainbench/internal/testengine/accounts.go:84` — `_, err := c.SendTransaction(ctx, rpc.SendTxArgs{`

## H-03 체인 플러그인과 manifest

현재:

- 세 manifest는 binary(gwemix/gwbft/gstable), chain_id(8285/8284/8283), network_id, consensus_family(poa/wbft/wbft), genesis.hardforks, consensus.rpc_namespace(wemix/istanbul/istanbul), tx_types(세 체인 동일 목록), capabilities(세 체인 동일: process,rpc,ws,consensus)를 담는다.
- wbft manifest의 binary와 build.make_target는 gwbft다. 그러나 go-wbft Makefile의 타깃은 gwemix이고 산출물은 build/bin/gwemix다. 하네스는 자동 빌드하지 않으므로 make_target는 소비처가 없다.
- binary 이름은 세 곳에서 실행 파일로 쓰인다. 로컬 드라이버는 exec.Command(name)으로 PATH 검색, app.ResolveBinary는 exec.LookPath, 원격 target은 절대경로만 받는다. tests/tc의 wbft 케이스 8건은 bare 'gwbft'를 선언하므로 PATH에 gwbft라는 이름의 파일이 있어야만 돈다.
- compose 경로는 manifest.Binary를 기본값으로 쓰지 않는다. spec이 binary를 선언하지 않고 --binary도 없으면 오류다. manifest.Binary를 기본값으로 쓰는 곳은 hardfork/upgrade 명령뿐이다.
- capability catalog: common 3건(chains.list/info/hardforks), wemix 1건(bootstrap.plan), stablenet 10건(governance.*). wbft는 catalog가 없다. stablenet 핸들러는 SDK protocol.StableNet().Contract(RoleGovMinter)로 주소를 얻는다.
- accounts SDK 프로토콜은 세 체인 모두 baselineTxTypes(0x00~0x04, 0x16)를 반환한다. go-wemix에 type 0x04가 없어도 SupportsTxType(0x04)는 true다.
- external.Load는 프로젝트 manifest를 poa/wbft 내장 family 위에 올린다. protocol 필드로 SDK 프로토콜을 빌린다.

오늘 설정으로 되는 것:

- --manifest/--genesis-template(env.manifest/genesisTemplate)로 외부 manifest를 쓸 수 있다
- binaries에 절대경로나 ${GWBFT_BIN:-gwbft}를 써서 이름 불일치를 우회할 수 있다

세 체인 분리에 추가로 필요한 구현:

- wbft manifest의 binary/make_target를 실제 산출물 이름(gwemix)에 맞추거나, 이름과 경로를 분리해 기록하는 교정이 필요하다. go-wemix 산출물도 gwemix이므로 파일명으로 체인을 구분하면 안 된다.
- manifest.tx_types와 SDK TxTypes는 실제 지원 증거가 아니다. 체인별 실제 tx type/fork 지원표를 하네스가 갖고 preflight에서 검사해야 한다.
- wbft 거버넌스(GovConfig/GovStaking/GovNCP) capability catalog가 없다. 시스템 계약 테스트를 공통화하려면 체인별 adapter가 필요하다.

근거:

- `chainbench/internal/chains/wemix/manifest.json:3` — `"binary": "gwemix",`
- `chainbench/internal/chains/wemix/manifest.json:4` — `"chain_id": 8285,`
- `chainbench/internal/chains/wemix/manifest.json:16` — `"rpc_namespace": "wemix",`
- `chainbench/internal/chains/wbft/manifest.json:3` — `"binary": "gwbft",`
- `chainbench/internal/chains/wbft/manifest.json:8` — `"build": { "repo": "go-wbft", "make_target": "gwbft" },`
- `chainbench/internal/chains/wbft/manifest.json:19` — `"tx_types": ["0x00", "0x01", "0x02", "0x03", "0x04", "0x16"],`
- `chainbench/internal/chains/wbft/manifest.json:21` — `"capabilities": ["process", "rpc", "ws", "consensus"]`
- `chainbench/internal/chains/stablenet/manifest.json:4` — `"chain_id": 8283,`
- `chainbench/internal/chains/stablenet/manifest.json:12` — `"hardforks": ["istanbul", "boho"],`
- `chainbench/internal/core/registry/manifest.go:21` — `// Binary is the node binary name (gstable|gwbft|gwemix).`
- `chainbench/internal/core/process/local.go:55` — `return exec.Command(name, arg...)`
- `chainbench/internal/core/process/local.go:85` — `cmd := d.launch(ctx, spec.Binary, spec.Args...)`
- `chainbench/internal/app/upgrade.go:258` — `path, err := exec.LookPath(name)`
- `chainbench/internal/app/upgrade.go:239` — `return "", fmt.Errorf("binary %q must be an absolute path on server %q — a bare name would be looked up on this machine, not there",`
- `chainbench/internal/testengine/compose.go:168` — `return composition{}, fmt.Errorf("the spec declares no binary and none was given")`
- `chainbench/internal/chains/stablenet/caps.go:28` — `registry.RegisterHandler("v1", "stablenet", name, h)`
- `chainbench/internal/chains/stablenet/caps.go:54` — `gov, ok := protocol.StableNet().Contract(protocol.RoleGovMinter)`
- `chainbench/internal/chains/external/external.go:25` — `func Load(manifestPath, templatePath string) (registry.ChainPlugin, error) {`
- `chainbench/internal/chains/external/external.go:84` — `func ResolveChain(chain, manifestPath, templatePath string) (registry.ChainPlugin, error) {`

## H-04 합의 family와 handoff

현재:

- poa(wemix): RPC namespace wemix, validators는 governance 계약을 eth_call로 열거한다(getMemberLength/getMember). BuildGenesis는 거부하고 바이너리가 governance config로 genesis를 생성한다. 기동은 boot → join-nodeN(역순) → endpoints 단계다. etcd 때문에 포트 span 3이다.
- wbft(wbft/stablenet 공용): namespace istanbul, istanbul_getValidators. 템플릿 치환으로 genesis를 만든다. 모든 노드 동시 기동. StartFlags에 --allow-insecure-unlock, --rpc.enabledeprecatedpersonal, --rpc.allow-unprotected-txs가 붙는다.
- VerifyValidators는 키셋이 도출한 검증자 집합과 체인이 보고하는 집합을 대조한다. poa는 기동 후에만 알 수 있다.
- handoff(upgrade): profile yaml + go-wemix 자체 템플릿 + producer/validator 바이너리로 혼합 네트워크를 만든다. AwaitFork는 fork+10 블록까지 successor 검증자가 봉인했는지 확인한다. handoff 네트워크에는 workspace 노드표와 process control이 없다.

오늘 설정으로 되는 것:

- env.chain으로 family 선택
- upgrade.profile/template + binaries.producer/validator
- topology.pn으로 proxied 그래프

세 체인 분리에 추가로 필요한 구현:

- 장애 허용 수, 정족수, 블록 주기 같은 합의 기대값을 family가 제공하지 않는다. 케이스가 숫자를 직접 쓴다(derive quorum은 ceil(2n/3) 고정). PoA/etcd 장애 조건은 별도 판정이 필요하다.
- handoff 네트워크에서는 fault 스텝과 노드 역할 선택이 동작하지 않는다(nodes nil, control nil).

근거:

- `chainbench/internal/consensus/poa/poa.go:26` — `func (Family) RPCNamespace() string     { return "wemix" }`
- `chainbench/internal/consensus/poa/poa.go:40` — `func (Family) BuildGenesis(template []byte, p registry.GenesisParams) ([]byte, error) {`
- `chainbench/internal/consensus/poa/poa.go:94` — `Actions: []string{ActionDeployGovernance, ActionEtcdInit, ActionVerifyEtcd},`
- `chainbench/internal/consensus/poa/poa.go:149` — `return node.Reservation{P2PSpan: 3, RPCSpan: 3}`
- `chainbench/internal/consensus/poa/validators.go:38` — `func (Family) RuntimeValidators(ctx context.Context, c registry.Caller) ([]string, error) {`
- `chainbench/internal/consensus/poa/validators.go:23` — `govGetMemberLength = "0xd965ea00"`
- `chainbench/internal/consensus/wbft/wbft.go:20` — `func (Family) RPCNamespace() string     { return "istanbul" }`
- `chainbench/internal/consensus/wbft/wbft.go:30` — `func (Family) BuildGenesis(template []byte, p registry.GenesisParams) ([]byte, error) {`
- `chainbench/internal/consensus/wbft/wbft.go:46` — `"--rpc.enabledeprecatedpersonal",`
- `chainbench/internal/consensus/wbft/wbft.go:62` — `return []registry.Phase{{Name: "all"}}`
- `chainbench/internal/chainsetup/verify_validators.go:62` — `method, actual, err := registry.RunningValidators(ctx, p, caller)`
- `chainbench/internal/consensus/upgrade/handoff.go:50` — `postForkBlocks = 10`
- `chainbench/internal/consensus/upgrade/handoff.go:699` — `func (h *Handoff) AwaitFork(ctx context.Context, ns node.NodeSet, timeout time.Duration) (string, error) {`
- `chainbench/internal/consensus/upgrade/handoff.go:730` — `for _, v := range h.Plan.Network.WbftValidators {`
- `chainbench/internal/testengine/suite.go:288` — `net = composed{endpoints: handoffEndpoints(ns), caps: chainCaps(chain), teardown: teardown}`
- `chainbench/internal/testengine/compose.go:138` — `return composition{}, fmt.Errorf("a handoff composes from its profile and template; env hardforks, topology, launch, and config do not apply")`

## H-05 계정과 서명 경로

현재:

- ResolveAccount: 0x 리터럴은 주소만, node<N>은 노드 keystore 서명(키 없음), 그 외 라벨은 키셋의 키로 로컬 서명. faucet은 node1의 별칭이다. 키셋이 없으면 라벨은 오류다.
- sendTx 노드 서명 경로(eth_sendTransaction): to, data, accessList, value, gas, gasPrice|maxFee+tip, nonce를 모두 전달한다. 계약 생성(to 없음)도 가능하다.
- sendTx 로컬 서명 경로(key 인자 또는 로컬 라벨): to가 필수라 계약 생성이 불가하다. feePayerKey가 있으면 SendFeeDelegated(value만, data/gas/nonce/fee 지정 불가). data가 있으면 Execute(fee 자동, gas/nonce 지정 불가). fee/nonce를 명시하면 SendDynamicFeeTx로 가되 gas가 21000 고정이고 data가 빠진다. 아니면 SendCoin(type 0x02 자동 수수료).
- faucet/deployContract/registerContract/load는 funder()가 주소만 돌려주고 sendAndConfirm이 eth_sendTransaction을 쓴다. 로컬 키 라벨을 from에 줘도 노드 서명으로 간다. 즉 이 네 verb는 언락된 노드 계정이 있어야 한다.
- env.accounts 자금 지원도 node1의 eth_sendTransaction이다.
- sendRawTampered/sendSetCode/signAuthorization은 로컬 서명 raw 전송이다. SupportsTxType 게이트는 SDK 값이라 세 체인 모두 통과한다.
- Precheck의 misdirectedSends는 from: nodeN을 다른 노드로 보내는 스텝을 막는다. 공개 RPC(엔드포인트만 있음)에서는 node 라벨 자체가 의미가 없다.

오늘 설정으로 되는 것:

- 격리 네트워크: node 라벨, faucet, env.accounts, newAccount+key, 로컬 라벨(dev1 등)
- attach: --keys로 키셋 디렉터리를 주면 라벨이 해석된다

세 체인 분리에 추가로 필요한 구현:

- 공개 RPC(언락 계정 없음)에서 공통 케이스를 돌리려면 faucet/deployContract/registerContract/load와 env.accounts 자금 지원을 로컬 서명으로도 보낼 수 있어야 한다. 현재는 eth_sendTransaction 고정이다.
- 로컬 서명 sendTx에 계약 생성, data+gas+nonce+fee 동시 지정, accessList, 0x16+data가 없다. Wallet.SendDynamicFeeTx는 Data와 창조를 이미 지원하므로 builtins 쪽 배선만 부족하다.
- 체인별 tx type 지원표가 SDK에 없다(세 체인 동일). go-wemix에서 sendSetCode는 노드 거부로만 실패한다. 체인별 capability preflight가 필요하다.

근거:

- `chainbench/internal/testhelper/account.go:20` — `const FaucetLabel = "faucet"`
- `chainbench/internal/testhelper/account.go:76` — `if addressLiteral.MatchString(ref) {`
- `chainbench/internal/testhelper/account.go:80` — `return Account{}, fmt.Errorf("dsl: %q is a label but this run has no key set to resolve it against", ref)`
- `chainbench/internal/testhelper/account.go:84` — `label = "node1"`
- `chainbench/internal/testhelper/account.go:92` — `if !nodeLabel.MatchString(label) {`
- `chainbench/internal/testhelper/builtins.go:158` — `if keyHex, ok := ac.Args["key"].(string); ok && keyHex != "" {`
- `chainbench/internal/testhelper/builtins.go:177` — `if sender.SignsLocally() {`
- `chainbench/internal/testhelper/builtins.go:193` — `if al, ok := ac.Args["accessList"]; ok {`
- `chainbench/internal/testhelper/builtins.go:205` — `hash, err := c.SendTransaction(ctx, args)`
- `chainbench/internal/testhelper/builtins.go:262` — `return fmt.Errorf("dsl: sendTx requires \"to\" when this harness signs")`
- `chainbench/internal/testhelper/builtins.go:290` — `hash, err = w.SendFeeDelegated(ctx, fp, to, value)`
- `chainbench/internal/testhelper/builtins.go:296` — `hash, err = w.Execute(ctx, to, b, value)`
- `chainbench/internal/testhelper/builtins.go:316` — `dargs := accounts.DynamicTxArgs{ToHex: to, Value: value, Gas: 21000, Nonce: noncePtr}`
- `chainbench/internal/testhelper/builtins.go:328` — `hash, err = w.SendCoin(ctx, to, value)`
- `chainbench/internal/testhelper/assets.go:196` — `return acct.Address, nil`
- `chainbench/internal/testhelper/assets.go:211` — `hash, err := c.SendTransaction(ctx, args)`
- `chainbench/internal/testhelper/load.go:73` — `receipt, hash, err := sendAndConfirm(ctx, c, args, ac.Args)`
- `chainbench/internal/testengine/accounts.go:84` — `_, err := c.SendTransaction(ctx, rpc.SendTxArgs{`
- `chainbench/internal/testhelper/txprobe.go:142` — `sendErr := c.Call(ctx, "eth_sendRawTransaction", &h, "0x"+hex.EncodeToString(raw))`
- `chainbench/internal/testhelper/txprobe.go:162` — `if !ac.Deps.Accounts.SupportsTxType(setCodeTxType) {`
- `chainbench/internal/accounts/wallet.go:80` — `// account's next nonce; an empty ToHex is a contract creation. Value, GasFeeCap,`
- `chainbench/internal/testengine/validate.go:133` — `if bad := misdirectedSends(s); len(bad) > 0 {`

## H-06 RPC endpoint와 노드 역할

현재:

- rpc.Client는 http.DefaultClient로 JSON-RPC over HTTP를 보낸다. timeout은 ctx로만, 인증 헤더는 없다(Content-Type만).
- bare attach는 모든 endpoint를 RoleEN, 포트 미상으로 등록한다. 역할이 전혀 없으면 선택자 bp1/en1이 순서 기반으로 폴백한다. 워크스페이스 attach는 기록된 역할을 쓴다.
- WS URL은 node.Ports.WS로만 만든다. attach 노드는 포트가 없어 wsSubscribe/wsOpen이 실패한다. metrics도 MetricsURL 또는 Ports.Metrics가 있어야 한다.
- rpcCall 리더는 임의 메서드를 호출하고 dot-path로 값을 뽑는다. @latest 파라미터를 head 번호로 치환한다. 체인 namespace(istanbul_/wemix_)는 케이스에만 나온다.
- composed wbft 노드는 personal/unprotected-tx 플래그가 켜진다. 공개 RPC에는 admin/personal/txpool이 닫혀 있을 수 있다.

오늘 설정으로 되는 것:

- --rpc 반복으로 여러 endpoint
- 케이스의 on/onEach 선택자(bp1, en:any, node3)
- step별 timeout/pollInterval

세 체인 분리에 추가로 필요한 구현:

- attach 프로필: endpoint별 역할(bp/en/pn), HTTP/WS URL, metrics URL, 인증 헤더, 허용 namespace, 기본 timeout을 선언하고 노드표로 만드는 기능.
- namespace/메서드 노출 preflight(methodPresent는 있지만 케이스마다 수동).
- HTTP 클라이언트 timeout/재시도/인증 주입.

근거:

- `chainbench/internal/core/rpc/client.go:30` — `return &Client{url: url, http: http.DefaultClient}`
- `chainbench/internal/core/rpc/client.go:73` — `httpReq.Header.Set("Content-Type", "application/json")`
- `chainbench/internal/core/node/attached.go:37` — `Role:   RoleEN,`
- `chainbench/internal/core/session/environment_impl.go:114` — `if len(matched) == 0 {`
- `chainbench/internal/testengine/attach.go:51` — `// node: "en1" means the first node whose role is en, while "node1" means`
- `chainbench/internal/testhelper/derived.go:404` — `return "", fmt.Errorf("dsl: node%d has no WebSocket port (an attached node's ports are unknown)", n.Index)`
- `chainbench/internal/testhelper/metric.go:64` — `err := fmt.Errorf("dsl: metric: %s has no metrics port — was it launched with --metrics?", t.name)`
- `chainbench/internal/testhelper/derived.go:217` — `if err := c.Call(ctx, method, &result, params...); err != nil {`
- `chainbench/internal/testhelper/derived.go:235` — `const latestBlockParam = "@latest"`
- `chainbench/internal/consensus/wbft/wbft.go:47` — `"--rpc.allow-unprotected-txs",`

## H-07 장애 주입과 process control

현재:

- faultTarget은 Deps.Nodes(NodeControl)가 없으면 'attach mode does not own the node processes'로 실패한다. control은 워크스페이스 compose/attach에서만 workspaceNodes로 주입된다. bare attach와 handoff는 nil이다.
- partition/healPartition은 admin_removePeer/admin_addPeer다. 발견(discovery)이 켜진 네트워크는 다시 붙을 수 있다고 코드가 명시한다.
- swapNode는 binary/config/genesisOverlay 중 하나가 필요하고 NodeSwapper가 있어야 한다. readNodeLog는 NodeLogReader가 필요하며 워크스페이스의 로컬 로그 경로를 읽는다.
- requires에 process를 선언한 케이스는 13건이다. attach는 rpc만 제공하므로 자동 skip된다.

오늘 설정으로 되는 것:

- compose/워크스페이스 attach: stopNode/startNode/restartNode/swapNode/readNodeLog/partition
- expect: fail + reason으로 기동 실패 사유 검사

세 체인 분리에 추가로 필요한 구현:

- 운영 메인넷 attach에서는 장애 케이스를 돌릴 수 없다. 공통 목록에서 실행 환경(격리/운영)을 분리해야 한다.
- partition은 admin namespace가 열린 노드에서만 되고 실제 단절 관측(peerCount 감소)은 케이스가 따로 해야 한다.
- 원격 target에서 workspaceNodes.Log는 로컬 경로를 읽으므로 빈 로그가 될 수 있다(확인 필요).

근거:

- `chainbench/internal/testhelper/fault.go:342` — `if ac.Deps == nil || ac.Deps.Nodes == nil {`
- `chainbench/internal/testhelper/fault.go:344` — `"dsl: %s needs node control, which this run has none of (attach mode does not own the node processes)", action)`
- `chainbench/internal/testengine/suite.go:528` — `control: workspaceNodes{sd: sd, dataDir: dataDir},`
- `chainbench/internal/testengine/attach.go:84` — `Control interp.NodeControl`
- `chainbench/internal/app/workflow.go:112` — `KeysDir: in.KeysDir, Bus: in.Bus, Nodes: in.Nodes,`
- `chainbench/internal/testhelper/fault.go:397` — `if err := c.RemovePeer(ctx, enodes[to.Index]); err != nil {`
- `chainbench/internal/testhelper/fault.go:372` — `// Note: this severs current connections. A network with peer discovery enabled`
- `chainbench/internal/testhelper/fault.go:274` — `return fmt.Errorf("dsl: swapNode requires a \"binary\", \"config\", or \"genesisOverlay\"")`
- `chainbench/internal/testhelper/fault.go:156` — `return fmt.Errorf("dsl: readNodeLog node%d: this run's node control cannot read logs", n.Index)`
- `chainbench/internal/testengine/suite.go:200` — `path := node.Layout{Root: w.dataDir}.LogPath(node.LabelFor(n.Index))`

## H-08 체인 정책이 스며든 assertion/observer

현재:

- sendTx expect:reject는 '전송 오류가 있으면' 통과다. reason은 오류 문자열 부분 일치다. expect:revert는 receipt status 0x0이다.
- 기본 comparator: chainId/balanceAt/codeAt/call/txStatus/receiptLog/logs/rpcCall/derive는 Equal, blockNumber/peerCount/baseFee/estimateGas/gasPrice/metric은 GreaterOrEqual. InDelta도 있다.
- blockAdvance 기본 30s/500ms, blockHalt 10s 창에 1블록 허용, blockInterval 20샘플, waitBlock 60s. 블록 주기 기대값은 케이스가 적는다.
- derive: sum/diff/quorum(ceil(2n/3))/abiCall/word. 수수료 공식(baseFee+tip 등)은 케이스가 derive로 조립한다.
- callError는 eth_call 오류면 통과, methodPresent는 -32601이 아니면 통과. 오류 사유 문자열은 체인마다 다를 수 있다.
- 체인 이름 분기는 testhelper에 없다. 체인 어휘(istanbul_getWbftExtraInfo 24건, wemix_getBriocheBlockReward 3건)는 케이스 JSON에 있다.

오늘 설정으로 되는 것:

- compare/delta로 비교 방식
- timeout/pollInterval/within/maxAdvance
- reason 부분 일치

세 체인 분리에 추가로 필요한 구현:

- 체인별 기대값(정족수, 블록 주기, 최소 수수료, 거부 사유, effectiveGasPrice 공식)을 케이스 밖 프로필에서 주입하는 장치가 없다. FeePolicy/ConsensusPolicy oracle 또는 파라미터 바인딩이 필요하다.
- reject 판정이 '아무 오류'라서 잘못된 이유의 거부도 통과한다. 공통 케이스는 reason을 체인별 문자열 표로 관리해야 한다.

근거:

- `chainbench/internal/testhelper/builtins.go:462` — `if submitErr == nil {`
- `chainbench/internal/testhelper/builtins.go:466` — `if !strings.Contains(strings.ToLower(submitErr.Error()), strings.ToLower(reason)) {`
- `chainbench/internal/testhelper/builtins.go:479` — `return status == "0x0" || status == "0x00"`
- `chainbench/internal/testhelper/read.go:314` — `{name: assertChainID, defaultOp: "Equal", read: readChainID},`
- `chainbench/internal/testhelper/read.go:325` — `{name: assertBaseFee, defaultOp: "GreaterOrEqual", read: readBaseFee},`
- `chainbench/internal/testhelper/read.go:285` — `if op == "InDelta" {`
- `chainbench/internal/testhelper/builtins.go:67` — `defaultBlockAdvanceTimeout = 30 * time.Second`
- `chainbench/internal/testhelper/blockprobe.go:25` — `defaultBlockHaltWindow = 10 * time.Second`
- `chainbench/internal/testhelper/derived.go:474` — `// ceil(2n/3) = (2n + 2) / 3 in integer arithmetic.`
- `chainbench/internal/testhelper/derived.go:445` — `case "sum":`
- `chainbench/internal/testhelper/derived.go:421` — `if op == "abiCall" {`
- `chainbench/internal/testhelper/txprobe.go:319` — `if callErr == nil {`
- `chainbench/internal/testhelper/txprobe.go:330` — `// (-32601) means the method is absent. Args: method (required), params`
- `chainbench/internal/testhelper/derived.go:191` — `// getWbftExtraInfo.gasTip, istanbul_getValidators, a wemix_ reward query — are`

## H-09 capability 모델과 preflight

현재:

- applicableChains가 비면 모든 체인에 적용된다. requires ⊆ provided일 때만 실행, 아니면 skip. skip 사유는 한 문장 고정이다.
- provided: bare attach는 rpc(+--caps 수동), compose/워크스페이스는 manifest capabilities + ws + delayed-<fork>(override로 fork를 옮긴 경우) + overlay가 선언한 capabilities. handoff는 manifest capabilities만.
- manifest capabilities는 세 체인 동일(process, rpc, ws, consensus)이라 체인 간 기능 차이를 표현하지 않는다.
- tests/tc 현황: requires rpc 195, consensus 34, process 13, account-extra 4, short-expiry 2, ws 2. applicableChains는 stablenet 99, 미지정 63, stablenet+wbft 29, wbft 3, 세 체인 1. env.capabilities는 requires로 합쳐진다.
- tx type/fork/RPC namespace/계정 상태(blacklist) 지원 여부를 검사하는 preflight가 없다. SDK SupportsTxType은 세 체인 동일하다.

오늘 설정으로 되는 것:

- applicableChains, requires, env.capabilities, --caps, genesis overlay의 capabilities

세 체인 분리에 추가로 필요한 구현:

- 체인별 기능 capability(예: tx-0x04, p256, fee-delegation, istanbul-rpc, wemix-rpc, account-extra, blacklist)를 manifest 또는 실제 노드 probe로 제공하고 케이스가 requires로 선언하는 모델.
- skip 사유에 '어느 capability가 없었는지'를 기록.

근거:

- `chainbench/internal/testengine/capability.go:11` — `const attachCapability = "rpc"`
- `chainbench/internal/testengine/capability.go:19` — `if len(list) == 0 {`
- `chainbench/internal/testengine/capability.go:55` — `return chainOK(s) && satisfies(s.Requires, provided)`
- `chainbench/internal/testengine/engine_impl.go:131` — `rec.Reason("does not apply to this target (chain or required capabilities)")`
- `chainbench/internal/chainsetup/steps_compose.go:808` — `caps = append(caps, "ws")`
- `chainbench/internal/chainsetup/steps_compose.go:815` — `caps = append(caps, "delayed-"+strings.ToLower(fork))`
- `chainbench/internal/chainsetup/verbs_steps.go:345` — `opts.Capabilities = overlay.Capabilities`
- `chainbench/internal/testengine/suite.go:552` — `return p.Manifest().Capabilities`
- `chainbench/internal/chains/wemix/manifest.json:21` — `"capabilities": ["process", "rpc", "ws", "consensus"],`
- `chainbench/internal/dsl/spec_v2.go:398` — `// The env's capabilities and the case's requires are both gating inputs, so`

## H-10 실행 기록과 메타데이터

현재:

- session.json에 command/startedAt, env.json에 envId/fingerprint/dataPath/nodes, 테스트별 steps/assert/status/reason/artifacts.json이 남는다. report.json은 session/command/startedAt/summary/tests다.
- fingerprint 입력은 binary 문자열, binaries, config, genesisOverlay, topology, hardforks, placement, resolved config다. 바이너리 SHA나 버전은 없다.
- artifacts.json은 워크스페이스 compose에서 genesis 참조 하나만 기록한다. 워크스페이스 state는 Binary 경로, KeysDir, Capabilities, LaunchInputs 해시를 기록한다.
- StepResult는 signer/nonce/gas/hash/receipt/error를, AssertResult는 expected/actual/provenance를 담는다. Scrub이 key/privateKey/saveKey/password 값을 지운다.
- skip 사유는 고정 문장이라 어느 capability가 빠졌는지 남지 않는다.

오늘 설정으로 되는 것:

- --json 요약, artifact-root, dashboard 스트림

세 체인 분리에 추가로 필요한 구현:

- 실행 행렬용 메타데이터: chain, chainId(RPC 확인값), 바이너리 경로+sha256+버전(web3_clientVersion), fork 활성 상태, signer 모드(node/local), attach 프로필 id, skip/blocked 사유 코드. 세 체인 결과를 같은 case id로 묶는 키.
- bare attach와 handoff에도 artifacts 참조를 남기는 방법.

근거:

- `chainbench/internal/core/session/read.go:16` — `Command   string       `json:"command"``
- `chainbench/internal/core/session/environment_impl.go:144` — `Fingerprint string      `json:"fingerprint"``
- `chainbench/internal/dsl/interp/fingerprint.go:18` — `Binary         string            `json:"binary"``
- `chainbench/internal/dsl/interp/fingerprint.go:30` — `// comes from resolved; the rest come from the spec. It never touches a chain.`
- `chainbench/internal/testengine/suite.go:455` — `return []session.ArtifactRef{{Kind: "genesis", Ref: "genesis.json"}}`
- `chainbench/internal/chainsetup/workspace.go:61` — `Binary       string        `json:"binary,omitempty"``
- `chainbench/internal/chainsetup/workspace.go:75` — `LaunchInputs map[string]string `json:"launchInputs,omitempty"``
- `chainbench/internal/core/session/record.go:9` — `Signer  string`
- `chainbench/internal/core/session/scrub.go:14` — `secretField = regexp.MustCompile(`("(?:key|privateKey|saveKey|password|mnemonic|secret)"\s*:\s*)"[^"]*"`)`
- `chainbench/internal/core/report/report.go:33` — `type Report struct {`
- `chainbench/internal/testengine/engine_impl.go:131` — `rec.Reason("does not apply to this target (chain or required capabilities)")`

## 공개 RPC(언락 계정 없음)에서 공통 케이스가 필요로 하는 것

1. 서명: 모든 상태 변경 verb(sendTx, faucet, deployContract, registerContract, load, env.accounts 자금)가 로컬 키로 raw 전송할 수 있어야 한다. 현재 sendTx만 부분 지원한다.
2. 계정: 키셋 디렉터리(--keys)와 자금이 있는 운영 계정 라벨. faucet 별칭은 node1이라 공개 RPC에서는 무의미하다.
3. 노드표: endpoint별 역할, WS/metrics URL. 없으면 wsSubscribe/metric/en1 선택이 실패하거나 엉뚱한 노드를 가리킨다.
4. capability: rpc 외에는 --caps로 수동 선언해야 한다. process/consensus 필요 케이스는 skip된다.
5. namespace: admin/personal/txpool/istanbul/wemix 노출 여부를 케이스가 methodPresent로 직접 확인해야 한다.
