# 공통 테스트 분리: 설정으로 뺄 것, 따로 구현할 것, 변경 범위 (2026-09-11)

이 문서는 작업 목적의 네 가지 완료 조건에 직접 답한다. 근거는 세 갈래다. 테스트 파일에서 기계적으로 뽑은 의존 필드([의존 필드 색인](dependency-index.md)), 현재 클라이언트 코드의 차이([클라이언트 차이](chain-differences.md), CD-A/CD-B 항목), 현재 하네스 구조([하네스 분석](harness-dependencies.md), H 항목)다. 판정 대상은 JSON 케이스 195개와 명세 문서 행 74개다([전체 목록](all-tests.md)).

숫자는 원본 항목 수다. 같은 목적의 문서 행과 JSON, 이미 세 체인으로 이식된 사본이 각각 세어진다. 실행해서 확인한 결과가 아니라 정적 판정이다.

| 구분 | JSON | 문서 | 합계 |
|---|---:|---:|---:|
| 세 체인 공통 후보 | 99 | 45 | 144 |
| 그중 설정만 분리 | 54 | 20 | 74 |
| 그중 설정 분리 + 체인별 기대값(oracle) | 32 | 22 | 54 |
| 그중 별도 구현·소유 환경 필요 | 13 | 3 | 16 |
| 2체인 후보 (WEMIX4.0+StableNet) | 17 | 27 | 44 |
| 2체인 후보 (WEMIX3.0+WEMIX4.0) | 1 | 1 | 2 |
| 2체인 후보 (WEMIX3.0+StableNet) | 1 | 0 | 1 |
| 공통 제외 (StableNet 전용·전환 전용·하네스 전용) | 77 | 1 | 78 |

Confluence 명세 ID 단위(330개)의 판정은 [별도 표](confluence-spec-catalog.md)에 있다. 세 체인 공통 104개 중 설정만 분리 38개, 체인별 기대값 필요 59개, 별도 구현 7개다. JSON·문서 판정과 명세 ID 판정이 갈리는 곳은 "명세 그대로"(예: gasPrice == baseFee + 헤더 GasTip)와 "일반 부분"(gasPrice == baseFee + tip 출처)의 차이이며 표의 `일반 부분` 열에 적었다.

---

## 1. 메인넷별 의존 요소 (완료 조건 1)

의존 요소는 요청한 다섯 범주로 나눴다. 각 범주마다 "테스트 파일에 실제로 어떻게 박혀 있는지"와 "세 클라이언트가 어디서 갈리는지"를 같이 적었다.

### 테스트 계정

테스트 파일에서 계정은 세 가지 모양으로 나온다. 노드 역할 label(`node1`, `en1`), 주소 literal, `newAccount`가 만든 `$binding`이다. 주소 literal은 35개 파일에 있다. 가장 많은 것은 preset 키셋의 node1 계정 주소(15개 파일)이고, 나머지는 수신용 고정 주소(`0x7099…`, `0x…C0FFEE..`)다.

세 체인이 갈리는 곳은 두 군데다. 첫째, `from: node1` 같은 노드 서명은 언락된 노드 계정이 있는 소유 네트워크에서만 된다. 공개 RPC에는 없다(H-05). 둘째, StableNet은 계정에 blacklist/authorized 상태 비트가 있어 같은 송금이 거부되거나 수수료 계산이 달라진다(CD-A-07). 다른 두 체인에는 이 상태가 없다.

### RPC Endpoint

케이스는 endpoint를 URL로 적지 않는다. `on`/`onEach`에 역할 이름을 적고 하네스가 노드표에서 URL을 찾는다. compose에서는 topology가, attach에서는 `--rpc` 순서가 노드표를 만든다. attach로 붙인 endpoint는 역할이 전부 EN이고 WS·metrics 포트가 없다(H-06). 그래서 `bp1`, `en1` 선택자와 `wsSubscribe`, `metric` 검사가 공개 RPC에서는 성립하지 않는다.

namespace도 의존이다. 케이스가 부르는 메서드는 eth/net/txpool/admin/web3 표준과 `istanbul_*`(31개 파일), `wemix_*`(1개 파일)다. `istanbul_*`은 go-wbft와 go-stablenet에만 있고 go-wemix에는 없다. `wemix` namespace는 go-wemix와 go-wbft에서 genesis에 Brioche 설정이 있을 때만 열리고, 메서드는 briocheConfig, halvingSchedule, getBriocheBlockReward 세 개뿐이다(CD-B-02). chainbench wemix manifest가 적은 `wemix_getValidators`와 `wemix_getReward`는 go-wemix에 없는 메서드다. 하네스의 poa family는 이미 `admin_wemixInfo`와 governance 계약 eth_call로 검증자를 읽으므로 manifest 값만 실제와 어긋나 있다. admin/txpool/personal은 공개 RPC에서 닫혀 있을 수 있다. 세 클라이언트의 HTTP/WS 기본 노출 모듈은 net, web3뿐이라 나머지는 `--http.api`/`--ws.api`로 열어야 한다.

### 컨트랙트 주소

컨트랙트 의존은 두 종류로 갈라야 한다. 일반 계약은 케이스가 bytecode를 배포하고 `$binding`으로 주소를 잇는다. 이 경로는 세 체인 공통이다. 단 bytecode가 어느 EVM revision을 요구하는지가 문제다. `regression/ethereum/23`의 fixture는 PUSH0(0x5f)을 쓰는데, go-wemix는 PUSH0을 기본 instruction set에 넣지 않았다(ExtraEips로만 켜진다). 나머지 fixture는 PUSH1/MSTORE/RETURN/REVERT/LOG 계열이라 문제없다.

시스템 계약은 주소가 고정돼 있다. StableNet의 `0x1000`(NativeCoinAdapter), `0x1001`(GovValidator), `0x1002`(GovMasterMinter), `0x1003`(GovMinter), `0x1004`(GovCouncil)를 62개 파일이 직접 부른다. go-wbft의 거버넌스 계약(GovConfig/GovStaking/GovNCP)과 go-wemix의 Registry 기반 거버넌스는 주소도 ABI도 의미도 다르다(CD-B-04). 주소를 바꿔 끼운다고 같은 테스트가 되지 않는다. P256 프리컴파일 `0x100`도 go-wemix에는 없다(CD-A-05).

### Chain ID

케이스는 `expect: chainId`를 9개 파일에서 쓰지만 전부 `> 0` 비교다. 실제 값 비교는 없다. 코드 기본값은 go-wemix와 go-wbft가 같은 1111이고 go-stablenet은 8282다. 하네스 manifest 기본값은 8285/8284/8283이다(CD-A-01, H-03). Chain ID만으로는 WEMIX3.0과 WEMIX4.0 구현체를 구분할 수 없다. 구분은 `env.chain`(client family)과 합의 namespace(`wemix_` vs `istanbul_`)로 한다.

### 기타 체인별 설정 및 실행 환경

- 포크: 케이스 26개가 genesis overlay나 hardforks를 쓴다. 키 이름이 체인별이다(StableNet `bohoBlock`/`anzeon`, WEMIX `brioche`, 공통 `applepieBlock`). 7702는 go-wbft가 Croissant 높이, go-stablenet은 anzeon 설정 존재로 gate한다(CD-A-02, CD-A-04).
- 수수료: 케이스 12개가 `istanbul_getWbftExtraInfo.gasTip`을 읽어(읽기 24회) fee 입력을 만든다. 이 필드는 StableNet 헤더에만 있다. baseFee 변화 규칙(WEMIX governance, WBFT EIP-1559, StableNet Anzeon 임계값·상하한)과 최소 수수료 기준(pool.gasPrice, miner.gasprice, MinBaseFee+MinTip)이 세 체인 모두 다르다(CD-A-06).
- 토폴로지·시간: bp/pn/en 수, `waitBlock target`, `timeout`, `blocks`가 케이스 상수다. 블록 주기가 다르면 전부 어긋난다.
- 프로세스 제어: 17개 파일이 stop/start/restart/swap/partition/load를 쓴다. 소유한 노드에서만 된다(H-07).
- 바이너리: 케이스가 `gwemix`/`gwbft`/`gstable` 이름을 적는다. go-wbft의 실제 산출물은 `gwemix`라 wbft manifest와 케이스 8건의 `gwbft`는 PATH에 그 이름으로 링크가 있어야 돈다(H-03). go-wemix 산출물도 `gwemix`이므로 파일 이름으로 체인을 구분하면 안 된다.
- 오류 문자열: 거부 사유 sentinel은 세 체인이 같지만 go-wbft/go-stablenet은 문맥을 덧붙여 감싼다. 현재 하네스의 `expect: reject`는 어떤 오류든 통과시키므로 잘못된 이유의 거부도 PASS가 된다(CD-A-08, H-08).

---

## 2. 설정으로 분리할 항목 (완료 조건 2)

아래 항목은 케이스의 단계와 assertion을 그대로 두고 값만 체인 프로필로 빼면 된다. "현재 자리"는 오늘 EnvV2로 표현되는지, "부족한 것"은 문법이나 기능이 없는 부분이다.

| 의존 요소 | 프로필에 넣을 값 | 현재 자리 | 부족한 것 |
|---|---|---|---|
| 테스트 계정 | sender/recipient/feePayer 역할별 label, 자금, 키셋 참조 | `env.keys`, `env.accounts{fund}`, `newAccount` | 주소 literal 35개 파일을 label로 치환. `env.accounts` 자금은 node1 노드 서명이라 공개 RPC에서는 안 됨 |
| RPC endpoint | 역할별 HTTP/WS/metrics URL, 인증 헤더, 허용 namespace, timeout | compose topology 또는 `--rpc` 순서 | attach 프로필 스키마 없음. env에 URL을 적는 자리 없음 |
| 일반 컨트랙트 | fixture bytecode/ABI, 기배포 주소·코드 해시 | `deployContract` + `$binding` | 기배포 계약 레지스트리 없음. fixture EVM revision 표시 없음 |
| Chain ID | expectedChainId, expectedGenesisHash, clientFamily | `env.chain`, manifest chain_id, `genesis.set config.chainId` | 실제 값 비교 assertion이 없음. RPC 값과 프로필 값 대조 preflight 없음 |
| 포크 | 활성 높이·시간, 체인별 overlay 키 | `env.hardforks`, `genesis.set/overlay`, `delayed-<fork>` capability | overlay 키 이름이 체인별이라 공통 env 하나로 못 씀. 체인별 env 3벌 필요 |
| 수수료 입력 | gas, tip, feeCap, priceBump, minTip, fillPercent | 케이스 상수 | step 상수를 env에서 주입하는 문법 없음(H-02) |
| 토폴로지·시간 | bp/pn/en 수, blockPeriod, timeout, waitBlock target | `env.topology`, step 상수 | 시간 상수도 env 주입 문법 없음 |
| 바이너리 | 절대경로, 소스 revision, 빌드 태그 | `env.binaries`, `${VAR:-default}` | wbft manifest binary/make_target 교정. SHA·버전 기록 없음(H-10) |
| 실행 권한 | 격리(compose) / 워크스페이스 attach / 공개 RPC | `requires: process`, `--caps` | 실행 환경 구분이 skip 사유에 남지 않음 |

`extends`는 최상위 필드 단위 얕은 병합이다. 공통 env에서 topology 한 키만 바꾸면 나머지 topology가 사라진다. 체인별 env는 전체 필드를 다시 써야 한다(H-02).

step 상수 주입 문법이 없다는 점이 설정 분리의 가장 큰 걸림돌이다. 수수료·시간·기대값 상수는 지금 케이스 JSON 안에만 있다. 선택지는 둘이다. `env.params → $binding` 같은 파라미터 바인딩을 DSL에 더하거나, 공통 템플릿에서 체인별 케이스 파일을 생성하는 생성기를 두는 것이다. 전자는 DSL 변경이고 후자는 파일이 3배로 는다.

---

## 3. 별도 구현 또는 메인넷별 처리가 필요한 항목 (완료 조건 3)

설정으로 값을 바꿔도 같은 입력에 대한 기대 결과가 다르거나, 하네스에 경로 자체가 없는 항목이다.

| 항목 | 공유할 부분 | 체인별로 갈리는 부분 | 근거 |
|---|---|---|---|
| 로컬 서명 sender | 서명 선택, raw 전송, receipt 대기 | 없음(하네스 문제). 현재 `faucet`/`deployContract`/`registerContract`/`load`/`env.accounts` 자금은 노드 서명 고정. 로컬 서명 `sendTx`는 계약 생성 불가, data+fee 동시 지정 불가, gas 21000 고정 | H-05 |
| FeePolicy oracle | tx/receipt/header 읽기, 금액 비교 | baseFee 변화식·상하한, 최소 tip/feeCap 기준, effectiveGasPrice 공식(StableNet 비인가 계정은 헤더 GasTip), 대납자 잔액 기준(feeCap vs gasPrice) | CD-A-03, CD-A-06 |
| fee 입력 source | derive로 feeCap 조립 | `istanbul_getWbftExtraInfo.gasTip`(StableNet 전용 필드) 대신 `eth_maxPriorityFeePerGas` 또는 프로필 값 | CD-A-06, 12개 파일 |
| 거부 사유 판정 | `expect: reject` | sentinel 부분 문자열 표를 체인별로. "아무 오류면 통과"를 sentinel 일치로 바꿈 | CD-A-08, H-08 |
| AccountFixture | 계정 생성·자금·nonce 격리 | StableNet blacklist/authorized 사전 검사 | CD-A-07 |
| ContractResolver | 일반 계약 배포·binding·ABI 호출 | 시스템 계약 주소·ABI·권한·이벤트 의미(WEMIX Registry / WBFT Gov* / StableNet 0x1000~0x1004) | CD-B-04 |
| EVM fixture | 배포·호출 단계 | PUSH0 등 revision별 opcode. go-wemix는 Shanghai 미적용 | CD-B-05 |
| Consensus oracle | 블록 진행·고정 높이 해시·peerCount 관측 | validators 조회(`istanbul_getValidators` vs `admin_wemixInfo` + governance 계약 eth_call), 허용 장애 수(WBFT ceil(2n/3) vs PoA/etcd 과반), 블록 주기(WBFT 1초 vs WEMIX3.0 governance blockInterval), epoch | CD-B-01, CD-B-02, H-04 |
| WBFTExtra 디코더 | `istanbul_getWbftExtraInfo` 키 읽기 | StableNet은 GasTip 필드가 끼어 RLP 배치가 다르고 epochInfo 키가 candidates, WEMIX4.0은 stakers/stabilizing. Croissant 이전 블록 번호로 부르면 오류 | CD-B-03 |
| 보상 검증 | 없음(2체인 suite) | Brioche 보상은 WEMIX3.0+4.0. StableNet은 블록 보상이 없고 baseFee를 이전 epoch 검증자에게 배분하므로 다른 검사 | CD-B-06 |
| genesis 생성 | env 선언 | WEMIX3.0은 바이너리가 governance 설정에서 생성(poa family), WBFT 두 체인은 템플릿 자리표. 세 체인 기대 genesis hash는 프로필 값 | CD-B-07, H-04 |
| 동기화 관측 | eth_syncing, 높이 추격 | 기본 syncmode(go-wemix snap, 나머지 full)를 launch로 통일. WEMIX3.0은 etcd 파트너 여부가 syncCheck에 개입하므로 비파트너 EN에서 관찰. downloader/fetcher 경로 증명은 로그 계측 필요 | CD-B-08, CD-B-09 |
| Capability preflight | requires/applicableChains gate | tx type(0x04), P256, fee-delegation fork, istanbul/wemix namespace, account-extra를 실제 노드에서 확인. 지금은 manifest tx_types와 SDK가 세 체인 동일 값 | H-09 |
| 실행 환경 분리 | 케이스 ID와 단계 | 격리/워크스페이스/공개 RPC별 SKIP 사유 코드. process·ws·admin 필요 케이스 | H-07, H-06 |
| 실행 행렬·메타데이터 | 결과 스키마 | chain, RPC 확인 chainId, 바이너리 sha·버전, fork 상태, signer 모드를 같은 case id로 묶기. 지금은 suite 하나가 chain 하나 | H-01, H-10 |

두 체인에만 있는 기능은 공통이 아니라 2체인 suite로 둔다. `istanbul_*`·WBFTExtra·7702·P256은 WEMIX4.0+StableNet, Brioche 보상은 WEMIX3.0+WEMIX4.0이다. 7702와 P256은 두 체인 사이에서도 gate가 다르다(Croissant 높이 vs anzeon 설정 존재, Croissant vs Boho). 지원 선언과 활성 상태를 구분하는 preflight가 있어야 SKIP과 FAIL이 갈린다.

---

## 4. 공통 테스트 분리를 위한 변경 범위 (완료 조건 4)

순서는 의존 관계를 따른다. 앞 단계 없이 뒤 단계를 하면 케이스가 늘어도 세 체인에서 같은 의미로 돌지 않는다.

| 순서 | 작업 | 바뀌는 곳 | 완료 기준 |
|---|---|---|---|
| 1 | 케이스 ID와 명세 ID 고정 | [ID 대응표](existing-tc-specs.md), 결과 메타데이터 | 195개 실행 ID가 명세 ID·2체인/3체인 범위와 함께 기록됨 |
| 2 | 체인별 env 3벌과 공통 케이스 분리 | tests/tc의 inline env → `*.env.json` 3종, `applicableChains` 정리 | 같은 케이스 파일이 세 env로 각각 돈다(호출 3회). wbft `gwbft` 이름 교정 |
| 3 | 주소 literal 제거와 계정 label 정리 | 35개 파일, `env.accounts` | 케이스에 0x 주소 literal이 없다(zero address, 7702 delegate 상수 제외) |
| 4 | step 파라미터 주입 문법 또는 생성기 | `internal/dsl` 또는 케이스 생성기 | 수수료·시간·기대값 상수가 env에서 온다 |
| 5 | 로컬 서명 sender 통합 | `internal/testhelper/builtins.go`, `assets.go`, `load.go`, `testengine/accounts.go` | 계약 생성·data·gas·nonce·fee·accessList·0x16이 로컬 서명으로도 같은 결과. faucet/deploy/load가 이를 공유 |
| 6 | FeePolicy·Consensus oracle과 fee source 치환 | `derived.go` 파생 op, 체인 plugin | `istanbul_getWbftExtraInfo.gasTip` 의존 12개 파일이 체인 무관 입력으로 바뀜. baseFee·최소 수수료·quorum 기대값이 프로필에서 온다 |
| 7 | 거부 사유 sentinel 표와 reject 판정 강화 | `builtins.go checkSubmitRejected`, 케이스 `reason` | 잘못된 이유의 거부가 PASS 되지 않는다 |
| 8 | capability preflight와 SKIP 사유 코드 | `testengine/capability.go`, manifest 또는 probe | tx-0x04/p256/istanbul-rpc/wemix-rpc/account-extra가 실제 노드 값으로 gate 된다 |
| 9 | attach 프로필(역할·WS·metrics·인증) | `testengine/attach.go`, 노드표 | 공개 RPC에서 en1/bp1·ws·metric이 프로필대로 동작 |
| 10 | 실행 행렬과 결과 스키마 | `core/session`, `core/report` | 세 체인 결과가 같은 case id로 묶이고 바이너리 sha·chainId·fork·signer 모드가 남는다 |

이 표는 하네스와 케이스의 변경이다. 클라이언트 소스를 바꿔 기능을 맞추는 일은 범위에 없다.

### 단점과 확인하지 못한 것

- 4번은 DSL 문법 변경이라 기존 195개 케이스의 strict 파서와 스키마(`schema/v2.schema.json`)에 영향을 준다. 생성기로 대신하면 파일 수가 늘고 원본과 사본이 갈라질 수 있다.
- 5번은 SDK Wallet의 기존 메서드를 재사용하는 배선이지만, 세 체인에서 로컬 서명 결과가 노드 서명과 같은지는 실행으로 확인해야 한다. 이번 분석은 실행하지 않았다.
- 6번의 oracle 값 중 WEMIX3.0은 governance 계약 상태를 읽어야 한다. 코드 기본값이 아니라 실제 네트워크 값이다.
- go-wbft mainnet 설정의 Croissant 높이는 TODO 값이다. WEMIX4.0 운영망의 실제 fork 상태·validator 수는 배포된 genesis에서 확인해야 한다.
- Confluence 페이지는 읽었지만 `[Common] Test` 페이지는 비어 있고, 두 시나리오 명세서 Markdown은 Confluence에 없다. 공통 테스트의 정본이 어디인지는 정해지지 않았다.
- 2nd Change T-2/T-3(AccessList 대납, keystore 대납 서명)과 PR196 N-008/N-009는 세 체인 목적이지만 go-wemix 경로는 코드만 보고 판단했다. 실행 확인이 필요하다.
- 공개 RPC(운영 메인넷)에서 어떤 namespace가 열려 있는지는 확인하지 않았다.
