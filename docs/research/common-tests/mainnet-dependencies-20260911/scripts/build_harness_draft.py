#!/usr/bin/env python3
"""Build draft-harness.json/.md from verified (path,line) citations. Each
snippet is read from the CURRENT file at that line, so a citation that drifted
fails loudly instead of quoting stale text."""
import json, os, sys
ROOT = "/Users/wm-it-25_0220/Work/github/chainbench"
OUT = "/Users/wm-it-25_0220/Work/github/chainbench/docs/research/common-tests/mainnet-dependencies-20260911/analyses"
_cache = {}
FAIL = []
def line(path, n):
    if path not in _cache:
        _cache[path] = open(os.path.join(ROOT, path), errors="replace").read().split("\n")
    src = _cache[path]
    if n < 1 or n > len(src):
        sys.exit(f"{path}:{n} out of range")
    s = src[n-1].strip()
    if not s:
        sys.exit(f"{path}:{n} is blank")
    return s[:150]
def ev(path, n, must=None):
    s = line(path, n)
    if must and must not in s:
        FAIL.append(f"{path}:{n} does not contain {must!r}: {s}")
    return {"repo": "chainbench", "path": path, "line": n, "snippet": s}

H = []
def item(id, area, current, configurable, needs, evidence):
    H.append({"id": id, "area": area, "current": current, "configurable_today": configurable,
              "needs_implementation": needs, "evidence": evidence})

item("H-01", "실행 경로 (suite run → app → testengine → dsl → interp → testhelper)",
  ["`suite run`은 --workspace-dir(compose), --workspace-dir --attach(워크스페이스 attach), --rpc(bare attach) 세 갈래로 나뉜다. --rpc와 --workspace-dir은 함께 쓸 수 없다.",
   "compose 경로: dsl.ReadFiles가 env 참조를 인라인하고, Parse → sameChain → sameComposition → Precheck → compositionOf → composeWorkspace 또는 handoffUp → wiredAttachEngine 순으로 간다.",
   "--chain은 attach에서 필수이고 compose에서는 spec의 env.chain과 같아야 한다. --binary/--keys/--keys-source/--validators/--chain-id/--network-id/--launch-opt는 compose 전용 override다.",
   "--workspace-config는 target dataRoot와 prepared inputs 프리셋(genesis/keyring/configs)을 공급한다. 같은 DSL을 파일 교체만으로 다른 target에 돌리는 장치다.",
   "attach 엔진은 accounts.ForChain(chain)으로 SDK 프로토콜을 고르고, 키셋(ringFor)과 testhelper.Registry()를 interp.Deps에 넣는다. 체인 이름이 SDK에 없으면 attach 자체가 거부된다."],
  ["compose: chain, binary, keys, validators, chain-id, network-id, launch-opt, workspace-config, server/docker", "attach: chain, rpc(반복), keys(라벨 해석용), dashboard", "여러 정의를 순서대로 돌리면 네트워크를 유지하고 preflight가 재사용 여부를 정한다"],
  ["세 체인을 한 번에 도는 실행 행렬이 없다. 한 suite는 한 chain, 한 composition만 허용하므로 공통 케이스는 체인별 env 3벌과 별도 호출 3회가 필요하다.",
   "attach는 chain 이름만 받고 실제 노드의 chainId/fork/버전을 대조하지 않는다. 공개 RPC에 붙일 때 chain 오지정을 잡는 preflight가 없다."],
  [ev("cmd/chainbench/suitecmd/run.go",66,"attach && len(rpcURLs)"), ev("cmd/chainbench/suitecmd/run.go",76,"workspaceDir != \"\""), ev("cmd/chainbench/suitecmd/run.go",113,"\"chain\""), ev("cmd/chainbench/suitecmd/run.go",115,"workspace-config"), ev("cmd/chainbench/suitecmd/run.go",119,"\"rpc\""), ev("cmd/chainbench/suitecmd/run.go",122,"\"binary\""), ev("cmd/chainbench/suitecmd/run.go",123,"\"keys\""), ev("cmd/chainbench/suitecmd/run.go",129,"chain-id"),
   ev("internal/dsl/files.go",30,"InlineEnv"), ev("internal/testengine/suite.go",257,"sameChain"), ev("internal/testengine/suite.go",260,"sameComposition"), ev("internal/testengine/suite.go",266,"Precheck"), ev("internal/testengine/suite.go",269,"compositionOf"), ev("internal/testengine/suite.go",319,"wiredAttachEngine"),
   ev("internal/testengine/compose.go",93,"in.Chain != chain"), ev("internal/testengine/attach.go",156,"accounts.ForChain"), ev("internal/testengine/attach.go",165,"NewRunSpec"), ev("internal/app/workflow.go",97,"in.DataDir"), ev("internal/app/workflow.go",109,"NewAttachEngine")])

item("H-02", "환경 선언(EnvV2) 필드와 런타임 효과",
  ["EnvV2 필드: target, chain, manifest, genesisTemplate, binaries, keys, blueprint, genesis, topology, hardforks, launch, config, capabilities, accounts, upgrade. CaseV2는 applicableChains, requires, on, timeouts, hooks, steps를 가진다.",
   "binaries.default 하나면 단일 바이너리, 그 외 키는 역할별/노드표 참조다. 값은 os.Expand로 $VAR, ${VAR:-default}를 치환한다.",
   "genesis.mode는 template(set/overlay 병합)와 existing(ref 파일 그대로)만 지원한다. hardforks는 {fork: height}를 <fork>Block=<n> override로 바꾼다.",
   "accounts는 체인 기동 후 node1이 eth_sendTransaction으로 자금을 보내 만드는 테스트 계정이다. genesis에 넣지 않는다.",
   "env.capabilities는 case.requires와 합쳐져 gate 입력이 된다. 즉 env가 선언한 capability는 '필요 조건'이지 '제공 조건'이 아니다.",
   "env 참조(\"env\": \"<id>\")는 <id>.env.json을 case 파일 상위 8단계까지 찾는다. extends는 최상위 필드 단위 얕은 병합이다.",
   "upgrade env는 profile/template와 producer/validator 바이너리만 받고 hardforks/topology/launch/config/genesis override를 거부한다."],
  ["테스트 계정 역할: keys(nodekeys) + accounts{label: fund} + newAccount 스텝", "RPC endpoint: compose에서는 topology와 target이 정하고, attach에서는 --rpc가 정한다. env에는 endpoint URL을 적는 자리가 없다", "Chain ID: env에는 없고 manifest chain_id 또는 --chain-id/genesis.set(config.chainId)으로 정한다", "fork: hardforks, genesis.set/overlay, delayed-<fork> capability", "topology: bp/en/pn 수, syncMode, nodes[] 표(role/binary/sync/bootnode/config/key)", "binary: binaries.default 또는 역할별, ${VAR:-default}", "launch/config: 노드 실행 플래그와 TOML 값 (scope: all/role/nodeN)"],
  ["step 상수(가스, 수수료, timeout, 주소, 기대 chainId)를 env에서 주입하는 문법이 없다. 공통 케이스가 체인별 기대값을 갖게 하려면 파라미터 바인딩(예: env.params → $binding) 또는 케이스 생성기가 필요하다.",
   "외부 RPC의 인증 헤더/토큰, 노드 역할(bp/en) 선언, WS URL, metrics URL을 env에 적을 수 없다. attach 프로필 스키마가 필요하다.",
   "contract address를 env에 선언할 수 없다. 배포 결과 $binding만 있다. 기배포 계약 주소/ABI/코드해시 레지스트리가 필요하다.",
   "extends가 얕은 병합이므로 topology 한 키만 바꾸면 나머지 topology가 사라진다. 공통 env를 체인별로 파생할 때 전체 필드를 다시 써야 한다."],
  [ev("internal/dsl/spec_v2.go",64,"Chain"), ev("internal/dsl/spec_v2.go",70,"Binaries"), ev("internal/dsl/spec_v2.go",76,"Genesis"), ev("internal/dsl/spec_v2.go",78,"Hardforks"), ev("internal/dsl/spec_v2.go",81,"Capabilities"), ev("internal/dsl/spec_v2.go",87,"Accounts"), ev("internal/dsl/spec_v2.go",91,"Upgrade"), ev("internal/dsl/spec_v2.go",101,"Fund"),
   ev("internal/dsl/spec_v2.go",189,"ApplicableChains"), ev("internal/dsl/spec_v2.go",268,"shallow top-level merge"), ev("internal/dsl/spec_v2.go",321,"for k, v := range envObj"), ev("internal/dsl/spec_v2.go",401,"len(env.Capabilities) > 0"), ev("internal/dsl/spec_v2.go",420,"\"default\""), ev("internal/dsl/spec_v2.go",456,"\"existing\""), ev("internal/dsl/spec_v2.go",468,"no runtime boundary yet"),
   ev("internal/dsl/files.go",13,"maxEnvSearchDepth"), ev("internal/testengine/compose.go",70,"os.Expand"), ev("internal/testengine/compose.go",137,"len(spec.Hardforks) > 0"), ev("internal/testengine/compose.go",540,"%sBlock=%d"), ev("internal/testengine/suite.go",311,"ring.Get(\"node1\")"), ev("internal/testengine/accounts.go",84,"SendTransaction")])

item("H-03", "체인 플러그인과 manifest",
  ["세 manifest는 binary(gwemix/gwbft/gstable), chain_id(8285/8284/8283), network_id, consensus_family(poa/wbft/wbft), genesis.hardforks, consensus.rpc_namespace(wemix/istanbul/istanbul), tx_types(세 체인 동일 목록), capabilities(세 체인 동일: process,rpc,ws,consensus)를 담는다.",
   "wbft manifest의 binary와 build.make_target는 gwbft다. 그러나 go-wbft Makefile의 타깃은 gwemix이고 산출물은 build/bin/gwemix다. 하네스는 자동 빌드하지 않으므로 make_target는 소비처가 없다.",
   "binary 이름은 세 곳에서 실행 파일로 쓰인다. 로컬 드라이버는 exec.Command(name)으로 PATH 검색, app.ResolveBinary는 exec.LookPath, 원격 target은 절대경로만 받는다. tests/tc의 wbft 케이스 8건은 bare 'gwbft'를 선언하므로 PATH에 gwbft라는 이름의 파일이 있어야만 돈다.",
   "compose 경로는 manifest.Binary를 기본값으로 쓰지 않는다. spec이 binary를 선언하지 않고 --binary도 없으면 오류다. manifest.Binary를 기본값으로 쓰는 곳은 hardfork/upgrade 명령뿐이다.",
   "capability catalog: common 3건(chains.list/info/hardforks), wemix 1건(bootstrap.plan), stablenet 10건(governance.*). wbft는 catalog가 없다. stablenet 핸들러는 SDK protocol.StableNet().Contract(RoleGovMinter)로 주소를 얻는다.",
   "accounts SDK 프로토콜은 세 체인 모두 baselineTxTypes(0x00~0x04, 0x16)를 반환한다. go-wemix에 type 0x04가 없어도 SupportsTxType(0x04)는 true다.",
   "external.Load는 프로젝트 manifest를 poa/wbft 내장 family 위에 올린다. protocol 필드로 SDK 프로토콜을 빌린다."],
  ["--manifest/--genesis-template(env.manifest/genesisTemplate)로 외부 manifest를 쓸 수 있다", "binaries에 절대경로나 ${GWBFT_BIN:-gwbft}를 써서 이름 불일치를 우회할 수 있다"],
  ["wbft manifest의 binary/make_target를 실제 산출물 이름(gwemix)에 맞추거나, 이름과 경로를 분리해 기록하는 교정이 필요하다. go-wemix 산출물도 gwemix이므로 파일명으로 체인을 구분하면 안 된다.",
   "manifest.tx_types와 SDK TxTypes는 실제 지원 증거가 아니다. 체인별 실제 tx type/fork 지원표를 하네스가 갖고 preflight에서 검사해야 한다.",
   "wbft 거버넌스(GovConfig/GovStaking/GovNCP) capability catalog가 없다. 시스템 계약 테스트를 공통화하려면 체인별 adapter가 필요하다."],
  [ev("internal/chains/wemix/manifest.json",3,"gwemix"), ev("internal/chains/wemix/manifest.json",4,"8285"), ev("internal/chains/wemix/manifest.json",16,"wemix"), ev("internal/chains/wbft/manifest.json",3,"gwbft"), ev("internal/chains/wbft/manifest.json",8,"make_target"), ev("internal/chains/wbft/manifest.json",19,"0x16"), ev("internal/chains/wbft/manifest.json",21,"capabilities"), ev("internal/chains/stablenet/manifest.json",4,"8283"), ev("internal/chains/stablenet/manifest.json",12,"boho"),
   ev("internal/core/registry/manifest.go",21,"gstable|gwbft|gwemix"), ev("internal/core/process/local.go",55,"exec.Command(name"), ev("internal/core/process/local.go",85,"spec.Binary"), ev("internal/app/upgrade.go",258,"LookPath"), ev("internal/app/upgrade.go",239,"must be an absolute path"), ev("internal/testengine/compose.go",168,"declares no binary"),
   ev("internal/chains/stablenet/caps.go",28,"RegisterHandler(\"v1\", \"stablenet\""), ev("internal/chains/stablenet/caps.go",54,"RoleGovMinter"), ev("internal/chains/external/external.go",25,"func Load"), ev("internal/chains/external/external.go",84,"func ResolveChain")])

item("H-04", "합의 family와 handoff",
  ["poa(wemix): RPC namespace wemix, validators는 governance 계약을 eth_call로 열거한다(getMemberLength/getMember). BuildGenesis는 거부하고 바이너리가 governance config로 genesis를 생성한다. 기동은 boot → join-nodeN(역순) → endpoints 단계다. etcd 때문에 포트 span 3이다.",
   "wbft(wbft/stablenet 공용): namespace istanbul, istanbul_getValidators. 템플릿 치환으로 genesis를 만든다. 모든 노드 동시 기동. StartFlags에 --allow-insecure-unlock, --rpc.enabledeprecatedpersonal, --rpc.allow-unprotected-txs가 붙는다.",
   "VerifyValidators는 키셋이 도출한 검증자 집합과 체인이 보고하는 집합을 대조한다. poa는 기동 후에만 알 수 있다.",
   "handoff(upgrade): profile yaml + go-wemix 자체 템플릿 + producer/validator 바이너리로 혼합 네트워크를 만든다. AwaitFork는 fork+10 블록까지 successor 검증자가 봉인했는지 확인한다. handoff 네트워크에는 workspace 노드표와 process control이 없다."],
  ["env.chain으로 family 선택", "upgrade.profile/template + binaries.producer/validator", "topology.pn으로 proxied 그래프"],
  ["장애 허용 수, 정족수, 블록 주기 같은 합의 기대값을 family가 제공하지 않는다. 케이스가 숫자를 직접 쓴다(derive quorum은 ceil(2n/3) 고정). PoA/etcd 장애 조건은 별도 판정이 필요하다.",
   "handoff 네트워크에서는 fault 스텝과 노드 역할 선택이 동작하지 않는다(nodes nil, control nil)."],
  [ev("internal/consensus/poa/poa.go",26,"\"wemix\""), ev("internal/consensus/poa/poa.go",40,"func (Family) BuildGenesis"), ev("internal/consensus/poa/poa.go",94,"ActionDeployGovernance"), ev("internal/consensus/poa/poa.go",149,"P2PSpan: 3"), ev("internal/consensus/poa/validators.go",38,"RuntimeValidators"), ev("internal/consensus/poa/validators.go",23,"govGetMemberLength"),
   ev("internal/consensus/wbft/wbft.go",20,"\"istanbul\""), ev("internal/consensus/wbft/wbft.go",30,"func (Family) BuildGenesis"), ev("internal/consensus/wbft/wbft.go",46,"enabledeprecatedpersonal"), ev("internal/consensus/wbft/wbft.go",62,"Name: \"all\""),
   ev("internal/chainsetup/verify_validators.go",62,"RunningValidators"), ev("internal/consensus/upgrade/handoff.go",50,"postForkBlocks = 10"), ev("internal/consensus/upgrade/handoff.go",699,"func (h *Handoff) AwaitFork"), ev("internal/consensus/upgrade/handoff.go",730,"WbftValidators"), ev("internal/testengine/suite.go",288,"handoffEndpoints"), ev("internal/testengine/compose.go",138,"do not apply")])

item("H-05", "계정과 서명 경로",
  ["ResolveAccount: 0x 리터럴은 주소만, node<N>은 노드 keystore 서명(키 없음), 그 외 라벨은 키셋의 키로 로컬 서명. faucet은 node1의 별칭이다. 키셋이 없으면 라벨은 오류다.",
   "sendTx 노드 서명 경로(eth_sendTransaction): to, data, accessList, value, gas, gasPrice|maxFee+tip, nonce를 모두 전달한다. 계약 생성(to 없음)도 가능하다.",
   "sendTx 로컬 서명 경로(key 인자 또는 로컬 라벨): to가 필수라 계약 생성이 불가하다. feePayerKey가 있으면 SendFeeDelegated(value만, data/gas/nonce/fee 지정 불가). data가 있으면 Execute(fee 자동, gas/nonce 지정 불가). fee/nonce를 명시하면 SendDynamicFeeTx로 가되 gas가 21000 고정이고 data가 빠진다. 아니면 SendCoin(type 0x02 자동 수수료).",
   "faucet/deployContract/registerContract/load는 funder()가 주소만 돌려주고 sendAndConfirm이 eth_sendTransaction을 쓴다. 로컬 키 라벨을 from에 줘도 노드 서명으로 간다. 즉 이 네 verb는 언락된 노드 계정이 있어야 한다.",
   "env.accounts 자금 지원도 node1의 eth_sendTransaction이다.",
   "sendRawTampered/sendSetCode/signAuthorization은 로컬 서명 raw 전송이다. SupportsTxType 게이트는 SDK 값이라 세 체인 모두 통과한다.",
   "Precheck의 misdirectedSends는 from: nodeN을 다른 노드로 보내는 스텝을 막는다. 공개 RPC(엔드포인트만 있음)에서는 node 라벨 자체가 의미가 없다."],
  ["격리 네트워크: node 라벨, faucet, env.accounts, newAccount+key, 로컬 라벨(dev1 등)", "attach: --keys로 키셋 디렉터리를 주면 라벨이 해석된다"],
  ["공개 RPC(언락 계정 없음)에서 공통 케이스를 돌리려면 faucet/deployContract/registerContract/load와 env.accounts 자금 지원을 로컬 서명으로도 보낼 수 있어야 한다. 현재는 eth_sendTransaction 고정이다.",
   "로컬 서명 sendTx에 계약 생성, data+gas+nonce+fee 동시 지정, accessList, 0x16+data가 없다. Wallet.SendDynamicFeeTx는 Data와 창조를 이미 지원하므로 builtins 쪽 배선만 부족하다.",
   "체인별 tx type 지원표가 SDK에 없다(세 체인 동일). go-wemix에서 sendSetCode는 노드 거부로만 실패한다. 체인별 capability preflight가 필요하다."],
  [ev("internal/testhelper/account.go",20,"FaucetLabel"), ev("internal/testhelper/account.go",76,"addressLiteral.MatchString"), ev("internal/testhelper/account.go",80,"no key set"), ev("internal/testhelper/account.go",84,"\"node1\""), ev("internal/testhelper/account.go",92,"!nodeLabel.MatchString"),
   ev("internal/testhelper/builtins.go",158,"\"key\""), ev("internal/testhelper/builtins.go",177,"SignsLocally"), ev("internal/testhelper/builtins.go",193,"accessList"), ev("internal/testhelper/builtins.go",205,"SendTransaction"), ev("internal/testhelper/builtins.go",262,"requires \\\"to\\\" when this harness signs"), ev("internal/testhelper/builtins.go",290,"SendFeeDelegated"), ev("internal/testhelper/builtins.go",296,"w.Execute"), ev("internal/testhelper/builtins.go",316,"Gas: 21000"), ev("internal/testhelper/builtins.go",328,"SendCoin"),
   ev("internal/testhelper/assets.go",196,"acct.Address"), ev("internal/testhelper/assets.go",211,"SendTransaction"), ev("internal/testhelper/load.go",73,"sendAndConfirm"), ev("internal/testengine/accounts.go",84,"SendTransaction"),
   ev("internal/testhelper/txprobe.go",142,"eth_sendRawTransaction"), ev("internal/testhelper/txprobe.go",162,"SupportsTxType(setCodeTxType)"), ev("internal/accounts/wallet.go",80,"empty ToHex is a contract creation"), ev("internal/testengine/validate.go",133,"misdirectedSends")])

item("H-06", "RPC endpoint와 노드 역할",
  ["rpc.Client는 http.DefaultClient로 JSON-RPC over HTTP를 보낸다. timeout은 ctx로만, 인증 헤더는 없다(Content-Type만).",
   "bare attach는 모든 endpoint를 RoleEN, 포트 미상으로 등록한다. 역할이 전혀 없으면 선택자 bp1/en1이 순서 기반으로 폴백한다. 워크스페이스 attach는 기록된 역할을 쓴다.",
   "WS URL은 node.Ports.WS로만 만든다. attach 노드는 포트가 없어 wsSubscribe/wsOpen이 실패한다. metrics도 MetricsURL 또는 Ports.Metrics가 있어야 한다.",
   "rpcCall 리더는 임의 메서드를 호출하고 dot-path로 값을 뽑는다. @latest 파라미터를 head 번호로 치환한다. 체인 namespace(istanbul_/wemix_)는 케이스에만 나온다.",
   "composed wbft 노드는 personal/unprotected-tx 플래그가 켜진다. 공개 RPC에는 admin/personal/txpool이 닫혀 있을 수 있다."],
  ["--rpc 반복으로 여러 endpoint", "케이스의 on/onEach 선택자(bp1, en:any, node3)", "step별 timeout/pollInterval"],
  ["attach 프로필: endpoint별 역할(bp/en/pn), HTTP/WS URL, metrics URL, 인증 헤더, 허용 namespace, 기본 timeout을 선언하고 노드표로 만드는 기능.",
   "namespace/메서드 노출 preflight(methodPresent는 있지만 케이스마다 수동).",
   "HTTP 클라이언트 timeout/재시도/인증 주입."],
  [ev("internal/core/rpc/client.go",30,"http.DefaultClient"), ev("internal/core/rpc/client.go",73,"Content-Type"), ev("internal/core/node/attached.go",37,"RoleEN"), ev("internal/core/session/environment_impl.go",114,"len(matched) == 0"), ev("internal/testengine/attach.go",51,"\"en1\""),
   ev("internal/testhelper/derived.go",404,"no WebSocket port"), ev("internal/testhelper/metric.go",64,"no metrics port"), ev("internal/testhelper/derived.go",217,"c.Call(ctx, method"), ev("internal/testhelper/derived.go",235,"@latest"), ev("internal/consensus/wbft/wbft.go",47,"allow-unprotected-txs")])

item("H-07", "장애 주입과 process control",
  ["faultTarget은 Deps.Nodes(NodeControl)가 없으면 'attach mode does not own the node processes'로 실패한다. control은 워크스페이스 compose/attach에서만 workspaceNodes로 주입된다. bare attach와 handoff는 nil이다.",
   "partition/healPartition은 admin_removePeer/admin_addPeer다. 발견(discovery)이 켜진 네트워크는 다시 붙을 수 있다고 코드가 명시한다.",
   "swapNode는 binary/config/genesisOverlay 중 하나가 필요하고 NodeSwapper가 있어야 한다. readNodeLog는 NodeLogReader가 필요하며 워크스페이스의 로컬 로그 경로를 읽는다.",
   "requires에 process를 선언한 케이스는 13건이다. attach는 rpc만 제공하므로 자동 skip된다."],
  ["compose/워크스페이스 attach: stopNode/startNode/restartNode/swapNode/readNodeLog/partition", "expect: fail + reason으로 기동 실패 사유 검사"],
  ["운영 메인넷 attach에서는 장애 케이스를 돌릴 수 없다. 공통 목록에서 실행 환경(격리/운영)을 분리해야 한다.",
   "partition은 admin namespace가 열린 노드에서만 되고 실제 단절 관측(peerCount 감소)은 케이스가 따로 해야 한다.",
   "원격 target에서 workspaceNodes.Log는 로컬 경로를 읽으므로 빈 로그가 될 수 있다(확인 필요)."],
  [ev("internal/testhelper/fault.go",342,"ac.Deps.Nodes == nil"), ev("internal/testhelper/fault.go",344,"attach mode does not own"), ev("internal/testengine/suite.go",528,"workspaceNodes"), ev("internal/testengine/attach.go",84,"Control interp.NodeControl"), ev("internal/app/workflow.go",112,"KeysDir: in.KeysDir"),
   ev("internal/testhelper/fault.go",397,"RemovePeer"), ev("internal/testhelper/fault.go",372,"peer discovery enabled"), ev("internal/testhelper/fault.go",274,"swapNode requires"), ev("internal/testhelper/fault.go",156,"cannot read logs"), ev("internal/testengine/suite.go",200,"LogPath(node.LabelFor")])

item("H-08", "체인 정책이 스며든 assertion/observer",
  ["sendTx expect:reject는 '전송 오류가 있으면' 통과다. reason은 오류 문자열 부분 일치다. expect:revert는 receipt status 0x0이다.",
   "기본 comparator: chainId/balanceAt/codeAt/call/txStatus/receiptLog/logs/rpcCall/derive는 Equal, blockNumber/peerCount/baseFee/estimateGas/gasPrice/metric은 GreaterOrEqual. InDelta도 있다.",
   "blockAdvance 기본 30s/500ms, blockHalt 10s 창에 1블록 허용, blockInterval 20샘플, waitBlock 60s. 블록 주기 기대값은 케이스가 적는다.",
   "derive: sum/diff/quorum(ceil(2n/3))/abiCall/word. 수수료 공식(baseFee+tip 등)은 케이스가 derive로 조립한다.",
   "callError는 eth_call 오류면 통과, methodPresent는 -32601이 아니면 통과. 오류 사유 문자열은 체인마다 다를 수 있다.",
   "체인 이름 분기는 testhelper에 없다. 체인 어휘(istanbul_getWbftExtraInfo 24건, wemix_getBriocheBlockReward 3건)는 케이스 JSON에 있다."],
  ["compare/delta로 비교 방식", "timeout/pollInterval/within/maxAdvance", "reason 부분 일치"],
  ["체인별 기대값(정족수, 블록 주기, 최소 수수료, 거부 사유, effectiveGasPrice 공식)을 케이스 밖 프로필에서 주입하는 장치가 없다. FeePolicy/ConsensusPolicy oracle 또는 파라미터 바인딩이 필요하다.",
   "reject 판정이 '아무 오류'라서 잘못된 이유의 거부도 통과한다. 공통 케이스는 reason을 체인별 문자열 표로 관리해야 한다."],
  [ev("internal/testhelper/builtins.go",462,"submitErr == nil"), ev("internal/testhelper/builtins.go",466,"strings.Contains(strings.ToLower(submitErr.Error())"), ev("internal/testhelper/builtins.go",479,"status == \"0x0\""), ev("internal/testhelper/read.go",314,"assertChainID"), ev("internal/testhelper/read.go",325,"assertBaseFee"), ev("internal/testhelper/read.go",285,"InDelta"),
   ev("internal/testhelper/builtins.go",67,"defaultBlockAdvanceTimeout"), ev("internal/testhelper/blockprobe.go",25,"defaultBlockHaltWindow"), ev("internal/testhelper/derived.go",474,"ceil(2n/3)"), ev("internal/testhelper/derived.go",445,"\"sum\""), ev("internal/testhelper/derived.go",421,"abiCall"),
   ev("internal/testhelper/txprobe.go",319,"callErr == nil"), ev("internal/testhelper/txprobe.go",330,"32601"), ev("internal/testhelper/derived.go",191,"istanbul_getValidators")])

item("H-09", "capability 모델과 preflight",
  ["applicableChains가 비면 모든 체인에 적용된다. requires ⊆ provided일 때만 실행, 아니면 skip. skip 사유는 한 문장 고정이다.",
   "provided: bare attach는 rpc(+--caps 수동), compose/워크스페이스는 manifest capabilities + ws + delayed-<fork>(override로 fork를 옮긴 경우) + overlay가 선언한 capabilities. handoff는 manifest capabilities만.",
   "manifest capabilities는 세 체인 동일(process, rpc, ws, consensus)이라 체인 간 기능 차이를 표현하지 않는다.",
   "tests/tc 현황: requires rpc 195, consensus 34, process 13, account-extra 4, short-expiry 2, ws 2. applicableChains는 stablenet 99, 미지정 63, stablenet+wbft 29, wbft 3, 세 체인 1. env.capabilities는 requires로 합쳐진다.",
   "tx type/fork/RPC namespace/계정 상태(blacklist) 지원 여부를 검사하는 preflight가 없다. SDK SupportsTxType은 세 체인 동일하다."],
  ["applicableChains, requires, env.capabilities, --caps, genesis overlay의 capabilities"],
  ["체인별 기능 capability(예: tx-0x04, p256, fee-delegation, istanbul-rpc, wemix-rpc, account-extra, blacklist)를 manifest 또는 실제 노드 probe로 제공하고 케이스가 requires로 선언하는 모델.",
   "skip 사유에 '어느 capability가 없었는지'를 기록."],
  [ev("internal/testengine/capability.go",11,"attachCapability"), ev("internal/testengine/capability.go",19,"len(list) == 0"), ev("internal/testengine/capability.go",55,"satisfies(s.Requires"), ev("internal/testengine/engine_impl.go",131,"does not apply"), ev("internal/chainsetup/steps_compose.go",808,"\"ws\""), ev("internal/chainsetup/steps_compose.go",815,"delayed-"), ev("internal/chainsetup/verbs_steps.go",345,"overlay.Capabilities"), ev("internal/testengine/suite.go",552,"Manifest().Capabilities"), ev("internal/chains/wemix/manifest.json",21,"capabilities"), ev("internal/dsl/spec_v2.go",398,"gating inputs")])

item("H-10", "실행 기록과 메타데이터",
  ["session.json에 command/startedAt, env.json에 envId/fingerprint/dataPath/nodes, 테스트별 steps/assert/status/reason/artifacts.json이 남는다. report.json은 session/command/startedAt/summary/tests다.",
   "fingerprint 입력은 binary 문자열, binaries, config, genesisOverlay, topology, hardforks, placement, resolved config다. 바이너리 SHA나 버전은 없다.",
   "artifacts.json은 워크스페이스 compose에서 genesis 참조 하나만 기록한다. 워크스페이스 state는 Binary 경로, KeysDir, Capabilities, LaunchInputs 해시를 기록한다.",
   "StepResult는 signer/nonce/gas/hash/receipt/error를, AssertResult는 expected/actual/provenance를 담는다. Scrub이 key/privateKey/saveKey/password 값을 지운다.",
   "skip 사유는 고정 문장이라 어느 capability가 빠졌는지 남지 않는다."],
  ["--json 요약, artifact-root, dashboard 스트림"],
  ["실행 행렬용 메타데이터: chain, chainId(RPC 확인값), 바이너리 경로+sha256+버전(web3_clientVersion), fork 활성 상태, signer 모드(node/local), attach 프로필 id, skip/blocked 사유 코드. 세 체인 결과를 같은 case id로 묶는 키.",
   "bare attach와 handoff에도 artifacts 참조를 남기는 방법."],
  [ev("internal/core/session/read.go",16,"Command"), ev("internal/core/session/environment_impl.go",144,"Fingerprint"), ev("internal/dsl/interp/fingerprint.go",18,"Binary"), ev("internal/dsl/interp/fingerprint.go",30,"never touches a chain"), ev("internal/testengine/suite.go",455,"Kind: \"genesis\""), ev("internal/chainsetup/workspace.go",61,"Binary"), ev("internal/chainsetup/workspace.go",75,"LaunchInputs"), ev("internal/core/session/record.go",9,"Signer"), ev("internal/core/session/scrub.go",14,"secretField"), ev("internal/core/report/report.go",33,"type Report struct"), ev("internal/testengine/engine_impl.go",131,"does not apply")])

if FAIL:
    print("\n".join(FAIL)); sys.exit(1)
json.dump(H, open(os.path.join(OUT, "draft-harness.json"), "w"), ensure_ascii=False, indent=1)

# Markdown
md = []
md.append("# 하네스 실행 구조 분석 초안 (2026-09-11, 현재 코드 기준)\n")
md.append("chainbench HEAD cc501aed 작업 트리를 다시 읽고 썼다. 2026-09-09 분석의 결론과 줄번호는 승계하지 않았다. 모든 인용은 현재 파일에서 다시 확인했다. 실행하지 않은 정적 분석이다.\n")
md.append("## 요약표\n")
md.append("| ID | 영역 | 현재 | 오늘 설정으로 되는 것 | 추가 구현 |")
md.append("|---|---|---|---|---|")
for h in H:
    md.append(f"| {h['id']} | {h['area']} | {h['current'][0]} | {h['configurable_today'][0]} | {h['needs_implementation'][0]} |")
md.append("")
md.append("## 2026-09-09 결론과 달라진 점\n")
md.append("- genesis mode existing이 생겼다. 워크스페이스 config의 prepared 프리셋이 genesis/keyring/configs를 공급한다. 이전 분석의 'template만 지원' 서술은 더 이상 맞지 않는다.")
md.append("- metrics endpoint가 노드표에 MetricsURL로 기록된다. 이전 분석에서 metric assertion이 컨테이너 내부 주소를 찍던 문제는 해결됐다.")
md.append("- handoff가 서버 세트와 노드별 machine을 받는다. 다만 테스트 엔진 관점에서는 여전히 노드표와 process control이 없다.")
md.append("- 하네스 코드에 체인 이름 분기는 없다. 이전 분석이 우려한 '체인별 하드코딩'은 testhelper가 아니라 케이스 JSON과 SDK 프로토콜(세 체인 동일 tx type 목록)에 있다.")
md.append("- sendTx 로컬 서명의 gas 21000 고정, faucet/deploy/load의 node-signed 고정은 그대로다. 이전 분석의 H04/H08 지적은 여전히 유효하다.")
md.append("- wbft manifest의 gwbft와 go-wbft Makefile의 gwemix 불일치도 그대로다. tests/tc의 wbft 케이스 8건이 bare 'gwbft'를 선언하므로 PATH에 그 이름의 파일이 있어야 한다.\n")
for h in H:
    md.append(f"## {h['id']} {h['area']}\n")
    md.append("현재:\n")
    for c in h['current']: md.append(f"- {c}")
    md.append("\n오늘 설정으로 되는 것:\n")
    for c in h['configurable_today']: md.append(f"- {c}")
    md.append("\n세 체인 분리에 추가로 필요한 구현:\n")
    for c in h['needs_implementation']: md.append(f"- {c}")
    md.append("\n근거:\n")
    for e in h['evidence']:
        md.append(f"- `chainbench/{e['path']}:{e['line']}` — `{e['snippet']}`")
    md.append("")
md.append("## 공개 RPC(언락 계정 없음)에서 공통 케이스가 필요로 하는 것\n")
md.append("1. 서명: 모든 상태 변경 verb(sendTx, faucet, deployContract, registerContract, load, env.accounts 자금)가 로컬 키로 raw 전송할 수 있어야 한다. 현재 sendTx만 부분 지원한다.")
md.append("2. 계정: 키셋 디렉터리(--keys)와 자금이 있는 운영 계정 라벨. faucet 별칭은 node1이라 공개 RPC에서는 무의미하다.")
md.append("3. 노드표: endpoint별 역할, WS/metrics URL. 없으면 wsSubscribe/metric/en1 선택이 실패하거나 엉뚱한 노드를 가리킨다.")
md.append("4. capability: rpc 외에는 --caps로 수동 선언해야 한다. process/consensus 필요 케이스는 skip된다.")
md.append("5. namespace: admin/personal/txpool/istanbul/wemix 노출 여부를 케이스가 methodPresent로 직접 확인해야 한다.")
open(os.path.join(OUT, "draft-harness.md"), "w").write("\n".join(md) + "\n")
print("items", len(H), "evidence", sum(len(h['evidence']) for h in H))
