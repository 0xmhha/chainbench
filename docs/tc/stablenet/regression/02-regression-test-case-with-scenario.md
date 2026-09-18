# Regression Test Case with scenario

> 출처: Confluence [Regression Test Case with scenario](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2599256076) (페이지 ID 2599256076, 버전 22, 최종 수정 2026-04-29)  
> 상위 페이지: Regression Test  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김)

---

[Regression Test Case 문서](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2599813145/Regression+Test+Case)의 103개 테스트 케이스에 대한 실행 가능한 상세 시나리오 작성.

---

| 상수/필드 | 값 / 출처 |
| --- | --- |
| `MinBaseFee` | `20000000000000` Wei (20 Gwei) — `params/protocol_params.go` |
| `MaxBaseFee` | `20000000000000000` Wei (20 000 Gwei) — `params/protocol_params.go` |
| `IncreasingThreshold` | 20% — `params/protocol_params.go` |
| `DecreasingThreshold` | 6% — `params/protocol_params.go` |
| `BaseFeeChangeRate` | 2% — `params/protocol_params.go` |
| `BlockPeriodSeconds` | 1 — `params/config_wbft.go` `DefaultAnzeonConfig.WBFT` |
| `Quorum` | `Ceil(2N/3)` (N=7 → 5) — `consensus/wbft/validator/default.go` `defaultSet.QuorumSize()` |
| 시스템 컨트랙트 주소 | NativeCoinAdapter `0x1000`, GovValidator `0x1001`, GovMasterMinter `0x1002`, GovMinter `0x1003`, GovCouncil `0x1004` |
| Native manager 주소 | NativeCoinManager `0xb00002`, AccountManager `0xb00003`, BLSPoP `0xb00001` |
| `WBFTExtra` 필드 | `VanityData, RandaoReveal, PrevRound, PrevPreparedSeal, PrevCommittedSeal, Round, PreparedSeal, CommittedSeal, GasTip, EpochInfo` (`core/types/istanbul.go`) |
| `WBFTAggregatedSeal` | `Sealers` (bitset), `Signature` (BLS aggregate) |
| 블랙리스트 에러 | `ErrBlacklistedAccount{Address}` — `core/state_transition.go`, `core/txpool/validation.go` |
| Zero address 에러 | `ErrZeroAddressTransfer` "transfer to zero address not allowed" — `core/vm/evm.go:211` |
| Precompile transfer 에러 | `ErrValueTransferToPrecompile` "value transfer to precompiled contract disallowed" — `core/vm/evm.go:215` |
| `eth_signRawFeeDelegateTransaction` | `internal/ethapi/api.go:2186` (`TransactionAPI`) / `:629` (`PersonalAccountAPI`) |

---

## Section A — 이더리움 기본 기능

### A-1. 노드 실행 및 동기화

#### RT-A-1-01 — 제네시스 블록으로 노드 초기화 ✓

go-stablenet은 3가지 체인 구성 모드를 지원한다. 하지만, 메인넷은 테스트에서 제외 한다. RT-A-1-01-A/B 2개 TC로 분할해 각 모드를 개별 검증한다.

| 모드 | 명령 | chainId | validator 수 | EpochLength | quorum | BohoBlock |
| --- | --- | --- | --- | --- | --- | --- |
| A: Testnet | `gstable --testnet` | 8283 | 7 | 140 | 2 | 100 |
| B: Private | `gstable init genesis.json` | 사용자 지정 | 사용자 지정 | 사용자 지정 | 사용자 지정 | 사용자 지정 |

> Anzeon 활성 시 3가지 모드 모두 `baseFeePerGas`는 `MinBaseFee` (20000 Gwei = `0x12309ce54000`)로 고정됨.

> 공통 환경 변수 (A/B Verify에서 사용): `RPC=<node_rpc_url>`, `GS=0x0000000000000000000000000000000000001000` (NativeCoinAdapter), `GV=0x0000000000000000000000000000000000001001` (GovValidator)


##### RT-A-1-01-A — Testnet 초기화 (`--testnet`) ✓

> Testnet genesis는 바이너리에 embedded (`stablenetTestnetAllocData`). A와 동일하게 `gstable init` 서브커맨드나 외부 `genesis.json` 파일이 **필요하지 않음** — `--testnet` 플래그만으로 기동하면 자동 초기화.

- 시나리오:
    1. Setup: 빈 `--datadir` (초기화되지 않은 상태). `init` 서브커맨드 수행 **금지**
    2. Action: 모든 노드(BP×7 + EN×7 + PN×1)에서 `gstable --testnet --datadir <dir>` 기동
    3. Expected: 블록 0 기록, testnet validator/BLS 키 7개 주입, 15개 노드 전부 동일 상태
    4. Verify:
        - Genesis hash (최우선 sentinel)
            - `cast block 0 --rpc-url $RPC --json | jq -r .hash` → `0x2bdf79b3d3cc49f9e6638ff81f3bb85065c79945a8fe4556cd0ff47bbfc02490`
        - 체인 식별
            - `cast chain-id --rpc-url $RPC` → `8283`
        - 헤더 필드
            - `cast block 0 --rpc-url $RPC --json | jq -r .baseFeePerGas` → `0x12309ce54000` (20000 Gwei, A와 공통)
        - 시스템 컨트랙트 5개 주입 (A와 동일 패턴, 5개 주소 모두 `!= "0x"`)
            - `cast code 0x0000000000000000000000000000000000001000 --block 0 --rpc-url $RPC` 외 4개
        - GovValidator (0x1001) — validator 7개 순서 검증
            - `cast call $GV 'validatorList()(address[])' --block 0 --rpc-url $RPC` → 길이 7, 순서:
                - \[0\] `0x9f06600b2c17108662e3840e76bb27c9468eb73d`
                - \[1\] `0x1aa18ec0b3131171b1b1ddba2dffd81410b30a5a`
                - \[2\] `0xe63b413353e1ba4ac99f2bd1892328e2365ec574`
                - \[3\] `0x20f681210071932dbe6387378adf6f26af029f7d`
                - \[4\] `0x5803c14973690550d6ffc2014b7bffd8005f6021`
                - \[5\] `0x59f6e6add1fbeab316a7b17c9f1966b3655efedb`
                - \[6\] `0x0a8dd92ce7ce53bbf6aed40228f9029bb3f92702`
            - 각 validator v에 대해 `cast call $GV 'validatorToBlsKey(address)(bytes)' <v> --block 0 --rpc-url $RPC` → 48 bytes, `StableNetTestnetChainConfig.Anzeon.Init.BLSPublicKeys[i]` 순서와 일치
            - `cast call $GV 'quorum()(uint256)' --block 0 --rpc-url $RPC` → `2` (A는1)
            - `cast call $GV 'gasTip()(uint256)' --block 0 --rpc-url $RPC` → `27600000000000` (A와 동일)
        - NativeCoinAdapter (0x1000)
            - `cast call $GS 'minterAllowance(address)(uint256)' 0x0000000000000000000000000000000000001003 --block 0 --rpc-url $RPC` → `100000000000000000000000000000` (1e29, A는 1e28)
        - BohoBlock = 100 미활성 검증 (RT-E-06 / PR #67과 교차)
            - 블록 0 시점: `cast call 0x0000000000000000000000000000000000000100 <p256Verify 입력> --block 0 --rpc-url $RPC` → precompile 미적용 결과 (call to empty address)
            - 블록 100 이후: 동일 호출이 성공 응답
        - EpochLength = 140 검증 (A의 10과 대비)
            - `curl -s $RPC -d '{"jsonrpc":"2.0","id":1,"method":"istanbul_getWbftExtraInfo","params":["0x8b"]}' | jq -r '.result.epochInfo.candidates | length'` (블록 139) → `7`
        - Prealloc 잔액 (testnet prealloc 주소 샘플 5\~10개 각각)
            - `cast balance <addr> --block 0 --rpc-url $RPC` → `stablenetTestnetAllocData` 디코드 결과와 일치
        - 노드 간 일관성
            - 위 모든 명령을 15개 노드 각각의 `$RPC`에서 반복 → 모든 응답 동일 (테스트는 1개의 노드만 검증)


##### RT-A-1-01-B — Private network 초기화 (사용자 `genesis.json`) ✓

> A/B와 달리 Private network은 embedded genesis가 없으므로 사용자가 작성한 `genesis.json`을 `gstable init` 서브커맨드로 **반드시 명시적으로 초기화**해야 한다 (`cmd/gstable/chaincmd.go:191-250` `initGenesis()`). 이후 네트워크 플래그 없이 기동.

- 시나리오:
    1. Setup: 폐쇄망용 `genesis.json` 준비 (필수 필드)
        - `config.chainId` (예: 9999, mainnet/testnet과 충돌하지 않는 값)
        - `config.anzeonBlock: 0`, `config.bohoBlock: 0`
        - `config.anzeon.init.validators` (예: 7개)
        - `config.anzeon.init.blsPublicKeys` (동일 길이)
        - `config.anzeon.systemContracts.*.params` (`quorum`, `members`, `gasTip`, `maxProposals` 등)
        - `alloc` — 테스트용 계정 잔액
        - `gasLimit` — 원하는 값
    2. Action:
        - (2-1) 모든 노드(BP×7 + EN×7 + PN×1)에서 `gstable --datadir <dir> init genesis.json` 실행 → 초기화 성공 로그 확인 (`Successfully wrote genesis state`)
        - (2-2) 초기화 완료 후 `gstable --datadir <dir>` 기동 (네트워크 플래그 없음)
    3. Expected: 블록 0 기록, 사용자 지정 값이 그대로 반영, 15개 노드 전부 동일 상태
    4. Verify:
        - Genesis hash 일관성 (최우선 sentinel — private는 하드코드 상수 없음, 첫 실행 결과를 `$EXPECTED_HASH`로 문서화해 재실행 시 비교)
            - `cast block 0 --rpc-url $RPC --json | jq -r .hash` → 15개 노드 전부 동일 값
        - 체인 식별
            - `cast chain-id --rpc-url $RPC` → `genesis.json`의 `config.chainId` 값
        - 헤더 필드
            - `cast block 0 --rpc-url $RPC --json | jq -r .gasLimit` → `genesis.json`의 `gasLimit`을 hex로 변환한 값
            - `cast block 0 --rpc-url $RPC --json | jq -r .baseFeePerGas` → `0x12309ce54000` (Anzeon 강제)
            - 음성: `genesis.json`에 다른 `baseFeePerGas`를 입력해도 위 값으로 고정됨
        - 시스템 컨트랙트 5개 주입 (A와 동일 패턴)
            - `cast code 0x0000000000000000000000000000000000001000 ~ 0x0000000000000000000000000000000000001004 --block 0 --rpc-url $RPC` → 5개 전부 `!= "0x"`
        - GovValidator (0x1001) — `genesis.json`과 일치
            - `cast call $GV 'validatorList()(address[])' --block 0 --rpc-url $RPC` → 순서와 주소가 `genesis.json`의 `anzeon.init.validators`와 정확히 일치
            - 각 validator v에 대해 `cast call $GV 'validatorToBlsKey(address)(bytes)' <v> --block 0 --rpc-url $RPC` → `genesis.json`의 `blsPublicKeys[i]`와 일치
            - `cast call $GV 'quorum()(uint256)' --block 0 --rpc-url $RPC` → `genesis.json`의 `govValidator.params.quorum`
            - `cast call $GV 'gasTip()(uint256)' --block 0 --rpc-url $RPC` → `genesis.json`의 `govValidator.params.gasTip`
        - Alloc 전수 (`genesis.json`의 `alloc`에 등록한 모든 주소)
            - `cast balance <addr> --block 0 --rpc-url $RPC` → `alloc[addr].balance`와 정확히 일치
        - 노드 간 일관성
            - 위 모든 명령을 15개 노드 각각의 `$RPC`에서 반복 → 모든 응답 동일 (테스트는 1개의 노드만 검증)


#### RT-A-1-02 — Full Sync ✓ 

- 검증: `eth/downloader/downloader.go` — `--syncmode full` 시 헤더→바디→receipts를 순차 다운로드, Anzeon에서는 WBFT extra 검증 포함
- 검증: `core/types/receipt.go:396-411` — Full Sync에서는 BP의 `EffectiveGasPrice`가 receipt에 이미 설정된 채로 전파됨 (DeriveFields fallback 미트리거)
- **시나리오 (3-Phase 구조)**:
    1. **Phase 1 — Setup + Tx 배치 전송 (BP에서)**:
        - 계정 A (일반), 계정 B (Authorized — 테스트 내 GovCouncil authorize 수행) 준비
        - **Batch 1**: Legacy Tx (type 0x0) × 3 — 계정 A, nonce N/N+1/N+2 → **RT-A-2-01 커버**
        - **Batch 2**: DynamicFeeTx (type 0x2) × 3 — 계정 A (일반), nonce N+3/N+4/N+5 → **RT-A-2-02 일반 경로 커버**
        - **Batch 3**: DynamicFeeTx (type 0x2) × 3 — 계정 B (Authorized), nonce M/M+1/M+2 → **RT-A-2-02 Authorized 경로 커버**
        - **Batch 4**: AccessListTx (type 0x1) × 3 — 계정 A, nonce N+6/N+7/N+8, accessList 포함 → **RT-A-2-03 커버**
        - **Batch 5**: Mixed × 8 — 계정 A, nonce N+9\~N+16을 **의도적 랜덤 순서** `[14,11,16,9,13,10,15,12]` 로 전송 → **RT-A-2-04 커버** (txpool nonce 정렬 검증)
        - 각 Batch 사이 1-2초 대기 (블록 분산 유도), 20개 tx 전부 receipt 확인 후 다음 Phase
    2. **Phase 2 — Full Sync**: EN 노드를 `--syncmode full --bootnodes <BP1_enode>` 로 기동 → head 도달 대기
    3. **Phase 3 — 동기화 후 검증 (EN에서)**:
        - (기존) `eth_blockNumber` BP == EN (diff ≤ 2), 임의 블록 hash 일치, `eth_syncing == false`
        - **(RT-A-2-01)** Batch 1 receipt: `type == 0x0`, `status == 0x1`, `effectiveGasPrice == gasPrice`
        - **(RT-A-2-02 일반)** Batch 2 receipt: `type == 0x2`, `effectiveGasPrice == min(headerGasTip + baseFee, gasFeeCap)` (일반 계정 — `receipt.go:403` headerGasTip 경로)
        - **(RT-A-2-02 Authorized)** Batch 3 receipt: `effectiveGasPrice == min(tx.gasTipCap + baseFee, gasFeeCap)` + 마지막 log가 `AuthorizedTxExecuted()` 이벤트
        - **(RT-A-2-03)** Batch 4: `type == 0x1`, `accessList` 필드 보존
        - **(RT-A-2-04)** Batch 5: 동일 블록 내 tx가 **nonce 오름차순 정렬** (랜덤 전송 순서와 무관)
        - **(RT-A-2-08)** **전 20개 tx에 대해** `BP receipt == EN receipt` (jq -S deep equal) — effectiveGasPrice/gasUsed/status/logs 전체 동일


#### RT-A-1-03 — Snap Sync ✓ 

- 검증: `eth/downloader/snap/` — snap sync는 특정 피벗 블록까지 헤더만 다운로드 후 그 state를 병렬 다운로드. 피벗 이후는 정상 full sync
- 검증: snap sync는 **128 블록 이상 필요** (pivot depth와 관련, `downloader.go` `fullMaxForkAncestry`/`MaxForkAncestry` 상수)
- **시나리오**:
    1. **Setup**: BP에서 최소 150 블록 생산 (pivot 여유). 신규 노드 stop 상태
    2. **Action**: `chainbench node start --syncmode snap --bootnodes <BP_enode>`
    3. **Expected**: snap 피벗까지 헤더+state 다운로드 → 이후 healing → head 도달
    4. **Verify**:
        - 동기화 완료 후 `eth_getBalance(<alloc 계정>, "latest")` 정상 잔액 반환 (state 접근 가능)
        - `eth_getBlockByNumber("latest").stateRoot`가 BP와 동일
        - 임의 블록 번호 N에 대해 `eth_getBlockByNumber(N).hash`가 BP1 == EN 동일


#### RT-A-1-04 — 노드 재시작 후 블록 생산 재개 ✓

- 검증: `params/config_wbft.go` `DefaultAnzeonConfig.WBFT.BlockPeriodSeconds = 1`
- **시나리오**:
    1. **Setup**: BP 노드 운영 중, 블록 N 생산 직후 `SIGTERM` 종료
    2. **Action**: 동일 datadir로 재시작
    3. **Expected**: 재시작 직후 다음 블록부터 1초 간격 유지
    4. **Verify**: 
        - `eth_getBlockByNumber("latest", false)` 폴링으로 5개 연속 블록의 `timestamp` 차이가 모두 1
        - seal 정보를 통해 재시작한 BP 가 블록 생성에 참여하는지 검증


#### RT-A-1-05 — P2P 피어 연결 (PN 부트노드 경유) ✓

- 검증 : RT-A-1-02(Full-Sync 테스트) 를 통해 검증
- **시나리오**:
    1. **Setup**: PN 노드를 먼저 기동, `admin_nodeInfo.enode` 추출
    2. **Action**: 모든 BP/EN 노드를 `--bootnodes <PN_enode>` 로 기동
    3. **Expected**: discovery v4 통해 BP↔PN↔EN 상호 발견
    4. **Verify**: 각 노드 `net_peerCount` ≥ 1, `admin_peers` 응답 배열에 상대 노드 enode/ID 포함


#### RT-A-1-06 — Downloader 경로: peer 재연결 시 큰 gap (≥ 2) 따라잡기 ✓ \[v2 신규\]

- 검증: `eth/sync.go:214` (Anzeon 분기) — `peer.TD > ourTD + tdAdjustment`이면 Downloader 트리거. 기본 `tdAdjustment = 1` → **gap ≥ 2일 때 즉시 동작**
- 검증: `eth/sync.go:106, 138-145` — `tdCheckTimer` (기본 10초)가 경과하고 로컬 TD가 정체되어 있으면 `forceAdjustTD = true` → 다음 iteration에서 `tdAdjustment = 0` → **gap 1도 Downloader 강제 트리거**
- 검증: `eth/ethconfig/config.go:61-62` — `ForceSyncCycle: 10s`, `TdSyncInterval: 10s` 기본값
- 검증: `eth/sync.go:33` — `defaultMinSyncPeers = 5`. 5 노드 환경(BP4+EN1)에서는 peer 수 4개 \< 5이므로 `forceSyncCycle(10초)` 만료 시 `forced=true`로 `minPeers=1`로 낮춰 sync 시작
- **⚠️ 전제조건 — 축소 테스트베드 필수 (BP×4 + EN×1 = 5 노드)**:
    - 본 TC는 문서 헤더의 기본 환경(BP×7+EN×7+PN×1 = 15 노드)이 **아닌** 축소 테스트베드(BP×4 + EN×1)에서 실행해야 합니다.
    - **축소가 필요한 이유**: `defaultMinSyncPeers = 5` 문턱을 의도적으로 **미달**시켜 `forced=true` 경로(`forceSyncCycle` 만료 시 `minPeers=1`로 낮춤)를 재현해야 하기 때문. 15 노드 환경에서는 EN이 항상 5개 이상의 peer에 연결되어 있어 `forced` 분기가 트리거되지 않습니다.
    - **기본 환경에서 발생할 수 있는 문제점**:
        1. `admin_removePeer`로 peer를 끊어도 `StaticNodes`/`TrustedNodes`에 등록된 노드라면 `staticDialer`가 수 초 내 재접속을 시도 → `net_peerCount`가 즉시 회복되어 "peer 0 상태"를 유지할 수 없음
        2. 15 노드 중 일부만 제거해도 나머지 peer가 `defaultMinSyncPeers = 5`를 초과 → `forced=true` 분기 미진입 → 검증하려는 경로가 실행되지 않음
        3. 결과적으로 `eth/sync.go:214`의 `tdAdjustment` 동적 변경(1→0) 경로를 관찰할 수 없어 TC가 "통과"처럼 보이지만 실제로는 **검증 경로를 한 번도 밟지 않은 거짓 통과** 발생
    - **축소 테스트베드 구성 방법**:
        - 별도의 chainbench 프로파일로 BP×4 + EN×1을 기동하거나
        - 기본 15 노드 환경에서 임시로 BP 3대, EN 6대를 중단(`chainbench node stop`)하여 5 노드만 유지
        - `--bootnodes` 외에 `--discovery.dns` 및 정적 peer 설정을 비활성화하여 자동 재연결 방지
- **시나리오 (RPC 기반 peer 조작)**:
    1. **Setup**: 축소 테스트베드(BP1\~BP4 + EN5), 모두 head 동기화 상태
    2. **Action 1 — peer 제거**: EN5에서 `admin_peers` 조회 → 각 peer의 `enode` 추출 → 모든 peer에 대해 `admin_removePeer(enode)` 호출 → EN5는 연결된 peer 0개 상태로 전환
    3. **Action 2 — gap 생성**: 15\~20초 대기 (BP들은 계속 블록 생산, EN5는 peer 0이라 수신 불가 → 로컬 head 정지, gap ≥ 10 블록 발생)
    4. **Action 3 — 단일 peer 재연결**: `admin_addPeer(<BP1_enode>)` 호출로 EN5가 BP1과 재연결. 이 시점 EN5의 `ourTD` \<\< `peer.TD`
    5. **Expected**:
        - 재연결 후 `ForceSyncCycle(10초)` 내에 `chainSyncer.loop`가 깨어나 `nextSyncOp()` 실행
        - `peer.TD > ourTD + 1` 조건 충족 → **Downloader 경로 활성화**
        - Downloader가 누락 블록 헤더·바디를 일괄 요청 → 순차 삽입
        - EN5의 head가 BP head에 근접(diff ≤ 2)까지 수렴
    6. **Verify**:
        - 제거 후 EN5의 `net_peerCount == 0` 또는 `admin_peers == []`
        - 대기 후 gap(BP - EN) ≥ 10
        - 재연결 후  `bp - en_head ≤ 2`
        - 중간 블록 샘플 해시가 BP == EN 일치
        - EN5 로그에 `"Imported new chain segment"` 또는 `"Downloader"` 메시지 관찰
- **Notes**:
    - chainbench는 `StaticNodes`로 연결되므로 `admin_removePeer` 후 약 30초 이내에 `staticDialer`가 재접속 시도 가능 → 짧은 창에서 빠르게 테스트
    - 4 peer뿐인 환경에서 `forced=true`에 의존하므로 재연결 직후 \~10초는 sync가 시작되지 않을 수 있음 (정상)


#### RT-A-1-07 — Block Fetcher 경로: peer 재연결 시 작은 gap (= 1) 또는 정상 전파 ✓ 

- 검증: RT-A-1-06 테스트에서 일부노드가 Block Fetcher 로 동기화 되므로, 통합 테스트
- 검증: `eth/fetcher/block_fetcher.go` — 피어가 `NewBlockHashes` 또는 `NewBlock` 메시지를 전송할 때 동작. Downloader와 별개의 단일 블록 import 경로
- 검증: `eth/sync.go:214`에서 `peer.TD <= ourTD + 1`이면 (Anzeon, 기본 `tdAdjustment=1`) `nextSyncOp`이 `nil` 반환 → Downloader 트리거 안 됨 → 블록 수신은 Fetcher 경로로만 가능
- **⚠️ 전제조건 — 축소 테스트베드 필수 (BP×4 + EN×1 = 5 노드)**:
    - 본 TC는 RT-A-1-06과 동일한 축소 테스트베드(BP×4 + EN×1)에서 실행해야 합니다.
    - **축소가 필요한 이유**:
        - "경로 2(peer 재연결 후 작은 gap = 1)" 시나리오는 RT-A-1-06과 동일하게 EN의 peer count를 **의도적으로 0으로 만들어야** 재현 가능하며, 기본 15 노드 환경에서는 `StaticNodes` 자동 재연결로 불가능
        - "경로 1(정상 운영 중 NewBlock 전파)"은 이론적으로 15 노드 환경에서도 관찰 가능하나, peer 수가 많을수록 **Fetcher와 Downloader 경로가 혼재**하여 "Fetcher 우세" 관찰이 어려워짐(여러 피어에서 오는 tdCheckTimer가 `forced=true`를 간헐적으로 유발 가능)
    - **기본 환경에서 발생할 수 있는 문제점**:
        1. `admin_removePeer` 후 `StaticNodes` 재연결로 gap = 1을 유지할 창이 짧음 → 재연결 시점에 BP의 TD가 이미 ≥ 2 증가하여 Downloader 경로로 흘러들어감
        2. 15 노드 환경에서 다수 피어로부터 `NewBlock` broadcast가 쏟아져 **메시지 순서/중복 제거(**`fetcher.Insert`의 `seen` 맵) 경합이 발생 → 단일 경로 관찰이 흐려짐
        3. **Downloader vs Fetcher 결정 트리**가 의도한 분기(gap=1 → Fetcher)로 재현되지 않아 **검증 경로를 밟지 않은 거짓 통과** 발생 가능
    - **축소 테스트베드 구성 방법**: RT-A-1-06과 동일
- **시나리오 (경로 1 — 정상 운영 중 NewBlock 전파)**:
    1. **Setup**: 정상 운영 중인 BP1\~BP4 + EN5, 모두 head 동기화 상태
    2. **Action**: 아무 조작 없이 5\~10 블록 생산 동안 관찰
    3. **Expected**:
        - BP1이 블록 N+1 생산 → `NewBlock` 메시지 broadcast → 다른 BP와 EN5 모두 Fetcher 경유 수신
        - EN5의 `eth_blockNumber`가 각 블록 생산 후 1\~2초 이내 갱신 (Downloader 개입 없이)
    4. **Verify**:
        - 5 블록 연속 관찰 중 `max_lag(BP, EN) ≤ 1`
        - 각 블록의 해시가 BP == EN 일치
- **시나리오 (경로 2 — peer 재연결 후 작은 gap 1)**:
    1. **Setup**: EN5 동기화 상태, BP1이 새 블록 생성 직전 상태
    2. **Action 1**: EN5에서 모든 peer 제거 (`admin_removePeer`)
    3. **Action 2**: **1\~2초 짧게 대기** (BP가 1 블록만 생산 → gap = 1)
    4. **Action 3**: BP1 peer 재연결 (`admin_addPeer`)
    5. **Expected**:
        - `peer.TD == ourTD + 1` → `nextSyncOp`이 `nil` 반환 (Downloader 미동작)
        - 대신 BP1이 다음 `NewBlock` 메시지를 전파 → Fetcher가 단일 블록 수신
        - 또는 `tdCheckTimer` 경과 시 `forceAdjustTD = true` 상태에서 1 블록 downloader로 수신
    6. **Verify**:
        - 재연결 후 20초 이내 EN5의 head가 BP와 일치
        - EN5 로그에 `"Imported new block"` 빈도가 downloader 메시지보다 높음 (Fetcher 우세 검증 — 선택)
- `Downloader vs Fetcher 결정 트리:`

| 조건 | 경로 |
| --- | --- |
| \| `peer.TD > ourTD + 1` (gap ≥ 2 | \| **Downloader** (일괄 헤더·바디 |
| \| `peer.TD == ourTD + 1` & \`forceAdjustTD == false | \| **Fetcher** (`NewBlock` 전파 기반 |
| \| `peer.TD == ourTD + 1` & `forceAdjustTD == true` (tdCheckTimer 경과 후 | \| **Downloader** (gap 1도 트리거 |


### A-2. 트랜잭션 처리

#### RT-A-2-01 — Legacy Tx (type 0x0) ✓

- 검증: RT-A-1-02 테스트에서 Tx 를 블록 마다 전송하면서 포함
- 검증: Legacy Tx는 `GasPrice == GasFeeCap == GasTipCap`이며, `effectiveGasPrice = baseFee + min(tipCap, gasFeeCap-baseFee) = baseFee + (gasPrice - baseFee) = gasPrice`로 계산됨. 단, Anzeon 일반 계정은 oracle effectiveTipForOracle 경로로 tip이 `header.GasTip()`로 강제될 수 있음 (RT-C-01)
- 검증: 단순 송금(value transfer)은 EIP-2028 기준 정확히 **21000 가스** 소비
- **시나리오**:
    1. **Setup**: 잔액 있는 계정 A, 현재 baseFee = B (예: 20000 Gwei)
    2. **Action**: `cast send --legacy --from A --to B --value 1ether --gas-price <B + tip>` → `eth_sendRawTransaction`
    3. **Expected**: 
        - 다음 블록에 포함, `receipt.status == 1`
        - `effectiveGasPrice` = `baseFee + min(tipCap, gasFeeCap - baseFee)` (Legacy는 gasPrice 그대로)
        - **gasLimit valid check**: 단순 송금이면 `receipt.gasUsed == 21000`, tx의 `gasLimit ≥ 21000` 확인
    4. **Verify**:
        - `eth_getTransactionReceipt(txHash).status == "0x1"`
        - `eth_getTransactionByHash(txHash).type == "0x0"`
        - `receipt.effectiveGasPrice` 값이 baseFee + tip 공식과 일치
        - `receipt.gasUsed == "0x5208"` (21000), `tx.gas ≥ "0x5208"`
        - 송신자 잔액 변화 = `value + (gasUsed × effectiveGasPrice)`

#### RT-A-2-02 — DynamicFeeTx (type 0x2) ✓ 

- 검증: RT-A-1-02 테스트에서 Tx 를 블록 마다 전송하면서 포함
- 검증: `core/types/tx_dynamic_fee.go` `effectiveGasPrice = baseFee + min(tipCap, gasFeeCap-baseFee)`
- 검증: 단순 송금(value transfer, data 없음)은 21000 가스, 컨트랙트 호출 시 EIP-2028 calldata + opcode 비용 추가
- **시나리오**:
    1. **Setup**: Anzeon 활성, 현재 baseFee = B (예: 20Gwei). 정상 로직 검증을 위해 **Authorized 계정** 사용 권장 (일반 계정은 header.GasTip()으로 강제됨 — RT-C-01 참고)
    2. **Action**: type 0x2 tx, `gasFeeCap = B + 5Gwei`, `gasTipCap = 3Gwei`, 단순 송금
    3. **Expected**: 
        - 블록 포함, `effectiveGasPrice = B + min(3Gwei, 5Gwei) = B + 3Gwei`
        - **gasLimit valid check**: 단순 송금이면 `receipt.gasUsed == 21000`, tx의 `gasLimit ≥ 21000`
        - `tx.gasLimit ≤ 블록 gasLimit` (RT-A-2-07과 직교)
    4. **Verify**:
        - `receipt.effectiveGasPrice == B + 3Gwei` (PR #70 fix로 BP/EN 양쪽 모두 동일 — RT-A-2-08 참고)
        - `receipt.gasUsed == "0x5208"` (단순 송금) 또는 컨트랙트 호출 시 `eth_estimateGas` 결과 이하
        - tx 본문 디코드 시 `gasLimit` 값이 입력값과 일치

#### RT-A-2-03 — AccessListTx (type 0x1) ✓

- 검증: RT-A-1-02 테스트에서 Tx 를 블록 마다 전송하면서 포함
- **시나리오**:
    1. **Setup**: 컨트랙트 C 배포 (특정 storage slot 사용)
    2. **Action**: `cast send --from A --to C --gas-price <baseFee+tip> --access-list '[{"address":"<C>","storageKeys":["0x..."]}]' "set(uint256)" 1`
    3. **Expected**: 블록 포함
    4. **Verify**: `eth_getTransactionByHash(tx).accessList`이 입력값과 일치

#### RT-A-2-04 — Nonce 순서 보장 ✓

- 검증: RT-A-1-02 테스트에서 Tx 를 블록 마다 전송하면서 포함
- **시나리오**:
    1. **Setup**: 계정 A nonce = N
    2. **Action**: 짧은 간격으로 nonce N, N+1, N+2 tx 3건 전송
    3. **Expected**: 모두 동일/연속 블록에 nonce 오름차순 포함
    4. **Verify**: 블록 transaction list에서 A의 tx 순서를 확인

#### RT-A-2-05a — GasTipCap \< MinTip 거부 ✓ 

- 검증: `core/txpool/validation.go:118-120` — `if tx.GasTipCapIntCmp(opts.MinTip) < 0 { return fmt.Errorf("%w: gas tip cap %v, minimum needed %v", ErrUnderpriced, ...) }` (Anzeon 여부 무관, 항상 수행)
- **시나리오**:
    1. **Setup**: 정상 노드, `opts.MinTip` = T
    2. **Action**: `gasTipCap = T - 1` 인 type 0x2 tx 전송
    3. **Expected**: txpool 진입 거부
    4. **Verify**: `errors.Is(err, txpool.ErrUnderpriced) == true` AND 메시지가 `transaction underpriced: gas tip cap` prefix로 시작

#### RT-A-2-05b — GasFeeCap \< MinBaseFee+MinTip 거부 (Anzeon) ✓ 

- 검증: `core/txpool/validation.go:122-130` — Anzeon 활성 시에만 `minFee = MinBaseFee + MinTip`을 계산하여 `gasFeeCap < minFee`이면 단일 `ErrUnderpriced`로 거부
- **시나리오**:
    1. **Setup**: Anzeon 활성(`AnzeonBlock=0`), `MinBaseFee = B0`, `MinTip = T`
    2. **Action**: `gasFeeCap = B0 + T - 1`, `gasTipCap ≥ T` 인 type 0x2 tx 전송
    3. **Expected**: txpool 진입 거부
    4. **Verify**: `errors.Is(err, txpool.ErrUnderpriced) == true` AND 메시지가 `transaction underpriced: gas fee cap` prefix로 시작
- **dev 변화**: v1.0.0에선 `ErrFeeCapTooLow`/`ErrUnderpriced` 두 경로 가능했으나 dev에선 단일 `ErrUnderpriced`로 통합. 메시지 포맷이 표준 geth `max fee per gas less than block base fee`가 **아님**에 주의. 05a(tip 하한)와 05b(fee 하한)는 동일 sentinel을 쓰지만 **메시지 prefix로 경로 식별**해야 함

#### RT-A-2-06 — 잔액 부족 ✓

- **시나리오**:
    1. **Setup**: 계정 A 잔액 = 1 WKRC
    2. **Action**: A → B value=2 WKRC tx 전송
    3. **Expected**: txpool 거부
    4. **Verify**: 응답 에러 메시지 `insufficient funds for gas * price + value`

#### RT-A-2-07 — Gas Limit 초과 ✓

- **시나리오**:
    1. **Setup**: 블록 gasLimit = G (예: 30 000 000)
    2. **Action**: tx.gasLimit = G + 1 전송
    3. **Expected**: txpool 거부
    4. **Verify**: 에러 메시지 `exceeds block gas limit`

#### RT-A-2-08 — receipt.effectiveGasPrice 확인 ✓ 

- 검증: RT-A-1-02 테스트에서 Tx 를 블록 마다 전송하면서 포함
- 검증: `core/types/receipt.go:396-411` (dev) — PR #70 merge로 Anzeon 활성 시에도 `DeriveFields()`가 fallback 계산:
    - Authorized 계정(RT-E-09 log 존재) → `txs[i].GasTipCap()` 사용
    - 일반 계정 → `headerGasTip` 사용
    - 최종: `min(gasTipCap + baseFee, gasFeeCap)`
- **시나리오**:
    1. **Setup**: BP 노드 = 채굴, EN 노드 = sync
    2. **Action**: BP에 type 0x2 tx 전송 → 블록 N에 포함 → BP/EN 양쪽에서 `eth_getTransactionReceipt(txHash)` 호출
    3. **Expected**: 양쪽 모두 `effectiveGasPrice != null` 그리고 동일 값
    4. **Verify**: 
        - BP receipt = EN receipt (동일 값)
        - 일반 계정 tx: `effectiveGasPrice == min(headerGasTip + baseFee, gasFeeCap)`
        - Authorized 계정 tx: `effectiveGasPrice == min(tx.GasTipCap + baseFee, gasFeeCap)`
- **회귀 검증 포인트**: 
    - **PR #70 회귀 방지** — receipt 인코딩/derive 변경 시 BP/EN 비교가 반드시 통과해야 함
    - **RT-E-09(AuthorizedTxExecuted log)와 결합** — log 마지막 위치 가정이 깨지면 derive에서 isAuthorized 판별이 깨짐. log 순서를 변경하는 PR이 들어오면 이 TC와 RT-E-09 둘 다 깨짐 → sentinel test 권장

#### RT-A-2-09 — Replacement tx ✓

- 검증: RT-A-2-04 테스트에 포함하여 같이 검증
- **시나리오**:
    1. **Setup**: A nonce=N+2 tx1 (gasFeeCap=F1, gasTipCap=T1) pending 상태로 유지 (낮은 fee)
    2. **Action**: 동일 nonce(N+2), gasFeeCap=F1×1.10 이상, gasTipCap=T1×1.10 이상으로 tx2 전송
    3. **Expected**: txpool 내부에서 tx1 제거, tx2로 교체
    4. **Verify**: `txpool_content.pending[A]` 에 tx2 hash만 존재, 다음 블록에 tx2만 포함

#### RT-A-2-10 — SetCodeTx (type 0x4 / EIP-7702) ✓ 

- 검증: RT-A-3-01 테스트에 통합하여 진행
- 검증: `core/types/tx_setcode.go` (PR #42 — `feat: EIP-7702 Set Code Transaction support`)
- 검증: delegation prefix `0xef0100` + 20-byte 위임 주소 → 해당 EOA `eth_getCode` 결과
- 검증: SetCodeTx도 DynamicFeeTx와 동일하게 `effectiveGasPrice = baseFee + min(tipCap, gasFeeCap-baseFee)` 적용
- 검증: gasLimit는 EIP-7702 intrinsic 가스 (21000 + per-auth `PerEmptyAccountCost`/`PerAuthBaseCost` 등) 이상이어야 함. `state_transition.go:applyAuthorization`이 `params.CallNewAccountGas - params.TxAuthTupleGas` 환불 처리
- 검증: EOA-\> CA 로 설정 + CA → EOA 로 설정 묶어서 진행 (제거시에는 0x0 으로 설정)
- 검증: 제거 후, EOA 로 동작하는지 검증
- **시나리오**:
    1. **Setup**: 
        - delegator EOA `D` (잔액 충분)
        - delegate 컨트랙트 `C` 배포 (예: 단순 storage 컨트랙트)
        - delegator의 chain authorization 객체 작성: `Authorization{ChainID, Address: C, Nonce: D.nonce}` → ECDSA 서명
        - 현재 baseFee = B (예: 20000 Gwei), tip 결정 (Authorized 계정 권장)
    2. **Action**: type 0x4 tx 구성 (`SetCodeTx { ChainID, Nonce, GasTipCap, GasFeeCap, Gas, To, Value, Data, AccessList, AuthList }`) → `eth_sendRawTransaction`
    3. **Expected**: 
        - `tx.receipt.status == 1`
        - `eth_getCode(D)` 반환값 == `0xef0100` + 20 bytes(`C`) (총 23 bytes)
        - **effectiveGasPrice** = `baseFee + min(tipCap, gasFeeCap - baseFee)`
        - **gasLimit valid check**: tx의 `gasLimit`이 `21000 + len(authList) × TxAuthTupleGas` 이상, `receipt.gasUsed ≤ gasLimit`
        - 이후 D 호출 시 C의 코드가 컨텍스트로 실행됨
    4. **Verify**: 
        - `eth_getCode(D, "latest")` hex 길이 = 48 (`0x` + 23×2)
        - prefix 확인: `code[0:8] == "0xef0100"`
        - `receipt.effectiveGasPrice` 값이 baseFee + tip 공식과 일치
        - `receipt.gasUsed` 값이 `eth_estimateGas` 결과 이하, `tx.gasLimit ≥ receipt.gasUsed`
        - 추후 D를 호출하는 별도 tx로 D.storage가 변경되는지 확인
- **주의**: EIP-7702는 EOA가 컨트랙트처럼 동작하게 되므로, 폐쇄망 환경에서 인증/블랙리스트 모듈과의 상호작용 별도 검증 권장


### A-3. 스마트 컨트랙트

#### RT-A-3-01 — 컨트랙트 배포 ✓

- **시나리오**:
    1. **Setup**: 단순 storage 컨트랙트 컴파일 (creation bytecode)
    2. **Action**: `eth_sendRawTransaction` (to=null, data=bytecode)
    3. **Expected**: receipt.contractAddress != null
    4. **Verify**: `eth_getCode(contractAddress)` 길이 \> 2

#### RT-A-3-02 — 상태 변경 함수 호출 ✓

- **시나리오**:
    1. **Setup**: RT-A-3-01에서 배포된 컨트랙트, 함수 `setX(uint256)`
    2. **Action**: `setX(42)` tx 전송
    3. **Expected**: tx 포함 후 상태 변경
    4. **Verify**: `eth_call(getter)` 또는 `eth_getStorageAt(contract, slot)` 결과 = 42

#### RT-A-3-03 — eth\_call (view 함수) ✓

- **시나리오**:
    1. **Setup**: RT-A-3-02에서 설정된 컨트랙트 + 상태 값 존재
    2. **Action**: `eth_call({to: contract, data: <getter selector>}, "latest")`
    3. **Expected**: tx 발행 없이 응답에 ABI 인코딩된 반환값
    4. **Verify**: 디코드 결과가 기대값

#### RT-A-3-04 — eth\_estimateGas ✓

- **시나리오**:
    1. **Setup**: 임의 함수 + 파라미터
    2. **Action**: `eth_estimateGas({from, to, data})`
    3. **Expected**: 양수 hex 반환
    4. **Verify**: 동일 호출의 실제 receipt.gasUsed 와 비교 → 추정값 ≥ 실제

#### RT-A-3-05 — eth\_call revert ✓

- **시나리오**:
    1. **Setup**: 컨트랙트에 `require(false, "BAD_INPUT")` 함수
    2. **Action**: `eth_call`로 해당 함수 호출
    3. **Expected**: JSON-RPC 에러 응답 (code 3 등), `data` 필드에 ABI 인코딩된 revert reason
    4. **Verify**: 디코드 결과 == "BAD\_INPUT", error.message에 `execution reverted`

#### RT-A-3-06 — revert tx receipt status==0 ✓ 

- 검증: EVM `REVERT` opcode 표준 동작 — revert 시점까지 사용한 가스만 차감, 잔여 가스는 환불, 상태 변경은 롤백
- **시나리오**:
    1. **Setup**: `require(false, "BAD")` 등 명시적 revert 함수 보유 컨트랙트, gasLimit = G (충분히 큼)
    2. **Action**: revert 함수를 tx로 호출
    3. **Expected**: tx는 블록에 포함, status=0, **gasUsed \< G** (잔여 환불)
    4. **Verify**: 
        - `eth_getTransactionReceipt(tx).status == "0x0"`
        - `receipt.gasUsed < gasLimit` (환불 발생 확인)
        - tx 실행 전후 컨트랙트 storage 동일 (`eth_getStorageAt` 비교) — 상태 변경 롤백 검증
        - sender 잔액 변화 = `gasUsed × effectiveGasPrice` (value 이동 없음)

#### RT-A-3-07 — out-of-gas tx ✓ 

- 검증: EVM `ErrOutOfGas` — gas 부족 시 전체 가스 소비, 환불 없음, 상태 롤백
- **시나리오**:
    1. **Setup**: 임의 함수(예: 큰 storage write loop)에 대해 `eth_estimateGas`로 정상 추정값 E를 얻음
    2. **Action**: `gasLimit = E - 1000` (즉, 실제 소비보다 부족한 한계)로 tx 전송
    3. **Expected**: tx는 블록에 포함되지만 실패, gasUsed == gasLimit
    4. **Verify**: 
        - `eth_getTransactionReceipt(tx).status == "0x0"`
        - `receipt.gasUsed == gasLimit` (전량 소비, 환불 없음)
        - 상태 변경 롤백 (`eth_getStorageAt` 비교)
        - sender 잔액 변화 = `gasLimit × effectiveGasPrice` (전량 차감)
- **RT-A-3-06 vs RT-A-3-07 차이**: 
    - 06: 명시적 `REVERT` → **잔여 가스 환불**
    - 07: gas 소진(`OUT_OF_GAS`) → **잔여 환불 없음**


### A-4. RPC API

#### RT-A-4-01 — eth\_blockNumber ✓

- **시나리오**: 1초 간격으로 폴링하여 단조 증가 확인

#### RT-A-4-02 — eth\_getBalance ✓

- **시나리오**: alloc/transfer 후 `eth_getBalance(addr, "latest")` Wei 단위 반환 검증

#### RT-A-4-03 — eth\_sendRawTransaction ✓

- **시나리오**: 오프라인 서명 → `eth_sendRawTransaction(rlp_hex)` → txHash 반환 → `txpool_content` 또는 `eth_getTransactionByHash`로 진입 확인

#### RT-A-4-04 — eth\_getLogs ✓

- **시나리오**: 컨트랙트가 이벤트 emit 한 tx 후 `eth_getLogs({fromBlock, toBlock, address, topics})` 호출 → 매칭 로그 배열 반환

#### RT-A-4-05 — eth\_chainId ✓

- 검증: genesis.json `config.chainId` (예: 8282) — `core/stablenet_genesis.go` `StableNetMainnetChainConfig`
- **시나리오**: `eth_chainId` 응답 hex → 십진수 변환 → genesis 값과 비교

#### RT-A-4-06 — eth\_subscribe newHeads ✓

- **시나리오**:
    1. **Setup**: WS 활성화 , WebSocket 연결 (`wscat -c ws://...`)
    2. **Action**: `{"method":"eth_subscribe","params":["newHeads"]}`
    3. **Expected**: subscription id 반환, 이후 매 블록마다 헤더 객체 push
    4. **Verify**: 5개 이상 연속 헤더 수신, 각각 number 증가

#### RT-A-4-07 — eth\_subscribe logs ✓

- **시나리오**:
    1. **Setup**: WebSocket + `eth_subscribe("logs",{address, topics})`
    2. **Action**: 별도 RPC로 매칭 이벤트 발생 tx 전송
    3. **Expected**: 실시간으로 log 객체 push
    4. **Verify**: 수신 log의 address, topics, data 확인

---

## Section B — WBFT 합의 엔진

#### RT-B-01 — 1초 블록 주기 ✓

- 검증: `params/config_wbft.go` `BlockPeriodSeconds: 1`
- **시나리오**: `eth_getBlockByNumber` 폴링으로 10블록 timestamp 차이 = 1초 검증

#### RT-B-02 — WBFTExtra 서명 포함 ✓

- 검증: `core/types/istanbul.go` `WBFTExtra.CommittedSeal *WBFTAggregatedSeal { Sealers, Signature }`, `WBFTExtra.PreparedSeal *WBFTAggregatedSeal`
- **시나리오**:
    1. **Setup**: 정상 노드, 블록 N≥1
    2. **Action**: `istanbul_getWbftExtraInfo(N)`
    3. **Expected**: 
        - `committedSeal.sealers` bitmap 존재, `committedSeal.signature` 비어있지 않음
        - `preparedSeal.sealers` bitmap 존재, `preparedSeal.signature` 비어있지 않음
    4. **Verify**: 
        - bit count(`committedSeal.sealers`) ≥ quorum (N=7 → 5)
        - bit count(`preparedSeal.sealers`) ≥ quorum
        - 테스트 환경(밸리데이터 7)에서 두 sealers bit count 모두 7로 모이는지 확인 권장

#### RT-B-03 — 에폭 전환 (검증자 집합 갱신) ✓

- 검증: RT-B-04, RT-B-05 와 함께 테스트
- 검증: `WBFTExtra.EpochInfo { Validators []uint32, Candidates []*Candidate, BLSPublicKeys [][]byte }` (`core/types/istanbul.go`), 에폭 마지막 블록에 채워짐
- **시나리오**:
    1. **Setup**: GovValidator 컨트랙트를 통해 validator 멤버를 추가 proposal, 승인 처리, epoch length E (params에서 확인 후 명시 — 예: E=1000)
    2. **Action**: 블록 E-1, E, E+1 의 `istanbul_getWbftExtraInfo` 호출
    3. **Expected**: 에폭 경계 블록(E-1)에서 `epochInfo` 필드가 존재, 새 검증자 집합 반영
    4. **Verify**: `epochInfo.candidates`/`validators` 길이 = 7

#### RT-B-04 — 검증자 추가 (제안→승인→다음 에폭 적용) ✓ (단, F-3-02 모델 이슈 참고)

- **시나리오**:
    1. **Setup**: V1\~V7 검증자, GovValidator 활성
    2. **Action**: GovValidator 멤버 quorum이 신규 검증자 V8 추가 절차 수행 (configureValidator + BLS PoP)
        1. 기존 active member가 `proposeAddMember(M8, newQuorum)` proposal을 만들고 quorum까지 승인하여 **V8 operator(M8)를 GovValidator member로 추가**
        2. 새 member M8가 `configureValidator(V8, blsKey, blsSig)`를 호출하여 **자기 validator 주소와 BLS 키를 등록**
    3. **Expected**: 다음 에폭 시작 블록에 V8 추가
    4. **Verify**: 에폭 시작 후 `istanbul_getValidators(<epoch_start_block>)` 응답에 V8 포함, V8 노드가 실제로 서명에 참여 (이후 블록의 sealers bitmap 분석)

#### RT-B-05 — 검증자 제거 ✓

- **시나리오**: RT-B-04 역과정. 다음 에폭부터 V7 sealers bitmap에서 제외 확인

#### RT-B-06 — GasTip 헤더 동기화 (worker → WBFTExtra.GasTip) ✓

- **책임**: GovValidator storage 변경 후 worker가 다음 블록 헤더에 자동 반영하는 경로만 검증 (거버넌스 lifecycle은 RT-F-3-05에서 별도 검증)
- **시나리오**:
    1. **Setup**: RT-F-3-05를 선행 또는 동등 셋업으로 GovValidator storage `gasTip = T2`인 상태 확보
    2. **Action**: 다음 블록 import 발생 → `worker.updateGasTipFromContract(state)` 콜백 트리거
    3. **Expected**: 새 블록의 `header.Extra.GasTip == T2`, `worker.gasTip == T2`, `txpool.gasTip == T2`
    4. **Verify**:
        - `istanbul_getWbftExtraInfo(N).gasTip == T2`
        - `eth_gasPrice` 응답값이 T2 기반으로 변경
- **근거**: `miner/worker.go:1200-1224` `updateGasTipFromContract`/`getGasTipFromContract`, `:874-883` import 콜백, `consensus/wbft/engine/engine.go:537,542` `WriteGasTip(gasTip)`, `:1291-1292` `GasTipMismatchError`

#### RT-B-07 — `istanbul_getValidators` ✓

- **근거**: `consensus/wbft/backend/engine.go:234`의 `Namespace: "istanbul"` 단일 등록 + `consensus/wbft/backend/api.go:136` `GetValidators()` → Go RPC 규약에 따라 `istanbul_getValidators`로 노출 (전체 코드베이스에 `Namespace: "wbft"` 없음)
- **시나리오**: `istanbul_getValidators("latest")` → 현재 에폭 검증자 주소 목록 반환

#### RT-B-08 — 쿼럼 미달 블록 거부 ✓

- 검증: `consensus/wbft/validator/default.go` `defaultSet.QuorumSize() = Ceil(2N/3)`
- **시나리오**:
    1. **Setup**: 7 validators (quorum=5)
    2. **Action**: BP 노드 3대를 stop
    3. **Expected**: 블록 검증 실패, 체인 미포함
    4. **Verify**: 노드 로그에 서명 부족 에러, `eth_blockNumber`가 변하지 않음

#### RT-B-09 — 라운드 체인지 ✓

- 검증: `consensus/wbft/core/` `RequestTimeout` 만료 시 round change
- **시나리오**:
    1. **Setup**: 라운드 0에서 V1이 제안자
    2. **Action**: V1 노드 강제 종료 (kill -9), 약 2\~5초 대기 (RequestTimeout 초과)
    3. **Expected**: 나머지 노드가 round change 후 다음 라운드 제안자(V2)로 블록 생성
    4. **Verify**: 새 블록의 `istanbul_getWbftExtraInfo(N).round > 0`

#### RT-B-10 — 라운드 체인지 후 정상 연결 ✓

- 검증: RT-B-09 와 통합 테스트
- **시나리오**:
    1. **Setup**: RT-B-09 결과 블록 N (round \> 0)
    2. **Action**: 모니터링 지속
    3. **Expected**: N+1 = parent N, N+1 round=0 (정상 복귀)
    4. **Verify**: parentHash 체인 일치 + round 시퀀스 확인

#### RT-B-11 — PrevCommittedSeal 수집 ✓

- 검증: `WBFTExtra.PrevCommittedSeal *WBFTAggregatedSeal`
- **시나리오**:
    1. **Setup**: 정상 운영
    2. **Action**: `istanbul_getWbftExtraInfo(N+1)`
    3. **Expected**: `prevCommittedSeal.sealers` bit count ≥ quorum (테스트 환경에선 7)
    4. **Verify**: bitmap 디코드 후 1인 bit 수가 7

#### RT-B-12 — PrevPreparedSeal 수집 ✓

- 검증: `WBFTExtra.PrevPreparedSeal *WBFTAggregatedSeal`
- **시나리오**: RT-B-11과 동일하되 `prevPreparedSeal` 필드 검증


---

## Section C — Anzeon 가스 가격 정책

#### RT-C-01 — 일반 계정 GasTip 강제 적용 ✓

- 검증: 일반 계정의 `gasTipCap`은 `header.GasTip()`으로 \*\*대체(replace)\*\*됨 (단순 clamp 아님)
- **시나리오**:
    1. **Setup**: header.GasTip = G (예: 1Gwei). 비-Authorized 계정 A
    2. **Action**: type 0x2 tx, gasTipCap = 5Gwei 로 전송
    3. **Expected**: tx 정상 처리되되 effective tip은 G로 적용됨 (검증자 보상 = G)
    4. **Verify**: 로컬 receipt.effectiveGasPrice = baseFee + G (5Gwei가 아님)

#### RT-C-02 — Authorized 계정 GasTip 자유 ✓

- **시나리오**:
    1. **Setup**: GovCouncil로 계정 A를 authorize (RT-F-5-03)
    2. **Action**: A가 type 0x2 tx, gasTipCap = T 임의 지정
    3. **Expected**: tipCap이 그대로 적용
    4. **Verify**: receipt.effectiveGasPrice = baseFee + min(T, gasFeeCap-baseFee)

#### RT-C-03 — baseFee 증가 (\>20%) ✓

- 검증: `IncreasingThreshold=20`, `BaseFeeChangeRate=2`
- **시나리오**:
    1. **Setup**: 현재 baseFee = B, blockGasLimit = G
    2. **Action**: 블록 N의 gasUsed \> G × 0.20 가 되도록 다수 tx 전송
    3. **Expected**: 블록 N+1 baseFee ≈ B × 1.02
    4. **Verify**: `eth_getBlockByNumber("0x"+N+1).baseFeePerGas` 값 비교

#### RT-C-04 — baseFee 유지 (6\~20%) ✓

- **시나리오**: 블록 N gasUsed가 \[6%, 20%\] 범위 → N+1 baseFee == N baseFee

#### RT-C-05 — baseFee 감소 (\<6%) ✓

- **시나리오**: 블록 N gasUsed \< 6% → N+1 baseFee ≈ B × 0.98 (단, MinBaseFee 하한)

#### RT-C-06 — MinBaseFee 하한 ✓

- 검증: `MinBaseFee = 20000000000000` Wei (20000 Gwei)
- **시나리오**:
    1. **Setup**: 노드를 충분히 idle 상태로 두어 baseFee가 계속 감소
    2. **Expected**: baseFee가 20000 Gwei에 도달 후 더 이상 감소하지 않음
    3. **Verify**: 연속 블록 baseFee 모니터링, 20000 Gwei 미만 발생 안함

#### RT-C-07 — MaxBaseFee 상한 ✓

- 검증: `MaxBaseFee = 20000000000000000` Wei (20,000,000 Gwei)
- **시나리오**: 부하 테스트로 baseFee를 상한까지 끌어올린 후 추가 부하에도 초과하지 않음 확인. (MaxBaseFee가 매우 큰 값이라 실제 부하 시뮬은 비실용적 → 단위/통합 테스트로 대체 가능)

---

## Section D — Fee Delegation (type 0x16)

> **카운트 안내**: v2 본문은 RT-D-01, 03, 04, 05만 보유 (4건, RT-D-02는 의도적으로 제거됨). v2 헤더 표 `Fee Delegation | 4` / 합계 `103`으로 정정 완료.

#### RT-D-01 — 정상 처리 ✓

- 검증: `core/state_transition.go` `buyGas()`가 FeePayer에서 가스비 차감, Sender는 value만 차감
- **시나리오**:
    1. **Setup**: Sender A (value 송금용 잔액), FeePayer P (가스비용 잔액). Applepie 활성 (폐쇄망에서 0 블록).
    2. **Action**: 
        - A가 FeeDelegateDynamicFeeTx 본체 서명(V/R/S) 후 RLP 직렬화
        - `eth_signRawFeeDelegateTransaction({feePayer: P, ...}, rlp_hex)` 호출 (P 키가 노드에 unlock)
        - 반환 RLP를 `eth_sendRawTransaction`으로 브로드캐스트
    3. **Expected**: 블록 포함, status=1
    4. **Verify**:
        - `eth_getBalance(A)` 변화 = -value (가스비 차감 0)
        - `eth_getBalance(P)` 변화 = -gasUsed × effectiveGasPrice
        - `eth_getBalance(B/recipient)` 변화 = +value
        - receipt.status == 0x1

#### RT-D-03 — Sender 서명 검증 실패 거부 ✓

- 검증: `core/types/transaction_signing.go` `recoverPlain()` → `crypto.ValidateSignatureValues()` → `ErrInvalidSig`
- **시나리오**:
    1. **Setup**: 정상 FeeDelegateTx 생성 후 SR을 1바이트 변조
    2. **Action**: `eth_sendRawTransaction(broken_rlp)` 또는 FeePayer 재서명 후 전송
    3. **Expected**: txpool 거부
    4. **Verify**: 응답 에러 메시지에 `invalid sender` / `invalid transaction v, r, s values` 포함

#### RT-D-04 — FeePayer 서명 검증 실패 거부 ✓ 

- 검증: FeePayer 복구는 `feeDelegateSigner.Sender()` (`core/types/transaction_signing.go`)
- **시나리오**:
    1. **Setup**: 정상 Sender 서명 → `eth_signRawFeeDelegateTransaction`으로 FeePayer 서명 받음 → FV/FR/FS 중 1바이트 변조
    2. **Action**: `eth_sendRawTransaction`
    3. **Expected**: 거부
    4. **Verify**: txpool 또는 블록 실행 단계에서 에러. 에러 메시지에 `invalid feePayer` 또는 `invalid transaction v, r, s values`. **검증 시점이 txpool인지 블록 실행인지는 코드 경로 추가 확인 필요** (현 경로는 buyGas/state transition)

#### RT-D-05 — FeePayer 잔액 부족 ✓

- 검증: `core/state_transition.go` `buyGas()` `if have := state.GetBalance(payer); have.Cmp(feeCheck) < 0 { return ErrInsufficientFunds(feePayer ...) }`
- **시나리오**:
    1. **Setup**: FeePayer P 잔액 \< gasLimit × gasFeeCap
    2. **Action**: tx 전송
    3. **Expected**: 블록 실행 단계에서 거부
    4. **Verify**: 에러 메시지에 `feePayer ... have ... want ...`, tx 미포함 또는 receipt 없음

---

## Section E — 블랙리스트 / 권한 계정

#### RT-E-01 — Sender 블랙리스트 거부 ✓

- 검증: txpool `core/txpool/validation.go:240`, state `core/state_transition.go:505` → `ErrBlacklistedAccount{Address}`
- **시나리오**:
    1. **Setup**: GovCouncil quorum 합의로 A를 blacklist 등록 (`proposeAddBlacklist(A) → vote → execute`)
    2. **Action**: A에서 임의 tx 전송
    3. **Expected**: txpool 거부
    4. **Verify**: 에러 메시지 `blacklisted account: 0x...A`. `eth_call(AccountManager.isBlacklisted(A))` 가 `true`

#### RT-E-02 — Recipient 블랙리스트 거부 ✓

- 검증: 동일 위치, recipient (msg.to  != nil) 체크
- **시나리오**: B를 블랙리스트 등록 → C가 to=B value=1 tx 전송 → 거부

#### RT-E-03 — FeePayer 블랙리스트 거부 ✓

- 검증: `core/txpool/validation.go:271` (Anzeon 활성 시), `core/state_transition.go:578`
- **시나리오**: P를 블랙리스트 등록 → FeeDelegateTx로 P를 FeePayer 지정 → 거부

#### RT-E-04 — 블랙리스트 해제 후 정상 처리 ✓

- 검증: `GovCouncil.proposeRemoveBlacklist` → `AccountManager.unBlacklist`
- **시나리오**: RT-E-01의 A에 대해 unBlacklist 거버넌스 실행 → 직후 A 송신 tx 정상 처리. `isBlacklisted(A) == false` 확인

#### RT-E-05 — Zero Address 전송 차단 ✓

- 검증: `core/vm/evm.go:204-211` — `value > 0 && Anzeon && to == 0x0` → `ErrZeroAddressTransfer`
- **시나리오**:
    1. **Setup**: Anzeon 활성
    2. **Action**: `to=0x0000000000000000000000000000000000000000`, value=1 tx 전송
    3. **Expected**: EVM 단계 abort
    4. **Verify**: receipt.status==0 + 에러 `transfer to zero address not allowed`. `eth_call`로 사전 검증 시 동일 에러 반환

#### RT-E-06 — Precompile 주소 코인 전송 차단 ✓ 

- 검증: `core/vm/evm.go:200-219` — `evm.Call()`이 Anzeon 활성 시 `evm.precompile(addr)` + `evm.nativeManager(addr)`로 **현재 활성 하드포크 기준의 precompile/native manager를 동적으로 판단**하여 `value > 0` 전송 시 `ErrValueTransferToPrecompile` 반환

```go
m, isNativeManager := evm.nativeManager(addr)
p, isPrecompile := evm.precompile(addr)
if !value.IsZero() {
    if evm.chainConfig.AnzeonEnabled() {
        if isNativeManager || isPrecompile {
            return nil, gas, ErrValueTransferToPrecompile
        }
    }
}
```
- **TC의 의미**: "precompile 주소" = 현재 활성 하드포크 기준 precompile — TC는 규칙을 검증하는 것이지 모든 주소를 열거하지 않음. 코드가 `evm.precompile()`로 동적 판단하므로 하드포크 변경 시에도 TC를 수정할 필요 없음
- **시나리오**:
    1. **Setup**: Anzeon 활성 환경, 현재 활성 하드포크 확인
    2. **Action**: 임의의 활성 precompile 또는 native manager 주소로 `to=<addr>, value=1` tx 전송
    3. **Expected**: `ErrValueTransferToPrecompile` 반환, receipt.status==0 또는 tx 미포함
    4. **Verify**: 에러 메시지 `value transfer to precompiled contract disallowed`
- **참고 — 테스트 실행 가이드** (TC 텍스트 아님, 실행 시 주소 선택 참고용):
    - 폐쇄망 환경 `AnzeonBlock=0, BohoBlock=0` (Boho 동시 활성) 기준 활성 precompile/native manager:
        - 0x01-0x0a: ecrecover, sha256, ripemd, identity, modexp, bn256\*, blake2f, kzg
        - 0x100: p256Verify (secp256r1) — Boho에서 활성 (PR #67)
        - 0xb00001: BLSPoP
        - 0xb00002: NativeCoinManager, 0xb00003: AccountManager (native managers)
    - 테스터는 이 중 임의 주소를 대상으로 테스트 (전수 테스트는 선택)

#### RT-E-07 — AccountManager.isBlacklisted ✓

- 검증: native\_manager selector `isBlacklisted(address)`
- **시나리오**: blacklist된 D에 대해 `eth_call({to:0xb00003, data: keccak("isBlacklisted(address)")[:4] + pad32(D)})` → return 1

#### RT-E-08 — AccountManager.isAuthorized ✓

- 검증: selector `isAuthorized(address)`
- **시나리오**: authorized E와 일반 F 각각 호출 → E=1, F=0

#### RT-E-09 — 인증계정 tx 실행 시 AuthorizedTxExecuted 이벤트 ✓ 

- 검증: `core/state_transition.go:583-595` — Anzeon 활성 시 **Sender(**`msg.From`)가 Authorized 계정이면 tx 실행 후 마지막 log로 추가:

```go
if rules.IsAnzeon && st.state.IsAuthorized(msg.From) {
    st.state.AddLog(&types.Log{
        Address: params.AccountManagerAddress,           // 0xb00003
        Topics:  []common.Hash{params.AuthorizedTxExecutedEventSig},
    })
}
```
- **⚠️ 중요 — FeePayer는 이 조건에 포함되지 않음**:
    - 코드는 `IsAuthorized(msg.From)` **단일 체크**만 수행합니다. FeeDelegateDynamicFeeTx(`0x16`)에서 **FeePayer만 authorized**이고 Sender는 일반 계정이면 `AuthorizedTxExecuted()` log는 **발생하지 않습니다**.
    - 파급: `DeriveFields()`의 fallback(`hasAuthorizedTxLog`)이 이 경우 "authorized가 아님"으로 판단 → `effectiveGasPrice`를 `headerGasTip` 기반으로 계산 (즉, 일반 계정 경로와 동일하게 `Miner.GetGasTip()` 값을 tipCap으로 사용). FeePayer authorize로 `tx.GasTipCap()`을 그대로 반영하려는 의도가 있다면 별도 설계가 필요.
- 검증: 이벤트 시그니처: `crypto.Keccak256Hash([]byte("AuthorizedTxExecuted()"))` = `0x40e728a89c7f5b192cf1c1b747fb64d51d81c7a2b3ed4607b94d3a1e6a3e0373` (`params/protocol_params.go:223`)
- **주의**: 코드 주석에 따르면 **마지막 log여야 함** — `DeriveFields()`가 마지막 log만 검사해 effective gas price 계산에 활용. log 순서를 변경하는 PR은 본 TC와 RT-A-2-08/G-1-04를 **동시에 깨뜨림**.
- **시나리오 (양성 — Sender Authorized)**:
    1. **Setup**: GovCouncil로 계정 A를 authorize (RT-F-5-03 절차)
    2. **Action**: A가 임의 tx 발행 (예: 단순 송금, type 0x0/0x1/0x2/0x4 중 하나)
    3. **Expected**: receipt.logs 마지막 항목이 AccountManager(0xb00003) 주소에 `AuthorizedTxExecuted()` 이벤트
    4. **Verify**:
        - `eth_getTransactionReceipt(tx).logs[-1].address == "0x0000...000b00003"`
        - `logs[-1].topics[0] == "0x40e728a89c7f5b192cf1c1b747fb64d51d81c7a2b3ed4607b94d3a1e6a3e0373"`
        - `logs[-1].data == "0x"` (no data)
- **시나리오 (음성 1 — 비-Authorized Sender)**:
    1. **Setup**: 일반 계정 N (authorize 없음)
    2. **Action**: N이 임의 tx 발행
    3. **Expected**: receipt.logs 어디에도 `AuthorizedTxExecuted()` 이벤트 없음
    4. **Verify**: `logs` 배열에서 `topics[0] == AuthorizedTxExecutedEventSig`인 항목이 0건
- **시나리오 (음성 2 — FeeDelegateTx, Sender 일반계정 + FeePayer authorized)**:
    1. **Setup**: 일반 계정 N(Sender), Authorized 계정 P(FeePayer)
    2. **Action**: N이 서명한 FeeDelegateDynamicFeeTx(type 0x16)를 P가 `eth_signRawFeeDelegateTransaction`으로 2차 서명 → 전송
    3. **Expected**: `AuthorizedTxExecuted()` log가 발생하지 않음 (Sender 기준 체크이므로)
    4. **Verify**:
        - receipt.logs에 `AuthorizedTxExecutedEventSig` 항목이 0건
        - `receipt.effectiveGasPrice`가 **일반 계정 경로**(=`min(headerGasTip + baseFee, gasFeeCap)`)로 계산되었는지 확인
        - (회귀 방지) 만약 향후 FeePayer authorize도 log를 발생시키도록 변경된다면 본 음성 케이스가 양성으로 전환되어야 하며, 그 시점에 RT-A-2-08 `effectiveGasPrice` 기대식도 함께 수정되어야 함

---

## Section F — 시스템 컨트랙트 & 거버넌스

### F-1. NativeCoinAdapter (0x1000)

#### RT-F-1-01 — transfer ✓

- 검증: NativeCoinAdapter.sol `transfer()` → `_transfer()` → `__coinManager.call(...)` → `core/vm/native_manager.go nativeTransfer()` → `evm.AddTransferLog()`
- **시나리오**:
    1. **Setup**: A 잔액 \> 0, B 잔액 = 0
    2. **Action**: `cast send 0x1000 "transfer(address,uint256)" B 1ether --from A`
    3. **Expected**: B 네이티브 잔액 +1ether
    4. **Verify**: `eth_getBalance(B)` 증가, `eth_getLogs({address:0x1000, topics:[Transfer]})` 에 (A→B,1e18) 이벤트

#### RT-F-1-02 — balanceOf ✓

- 검증: `_balanceOf(account) → _account.balance` (직접 native 잔액)
- **시나리오**: 임의 계정 X에 대해 `eth_call(0x1000, balanceOf(X))` 와 `eth_getBalance(X)` 동일 반환

#### RT-F-1-03 — approve / transferFrom ✓

- **시나리오**:
    1. **Setup**: A → B에 대해 `approve(B, 5ether)`
    2. **Action**: B가 `transferFrom(A, C, 3ether)` 호출
    3. **Expected**: A에서 C로 3ether 이동, allowance(A,B) = 2ether
    4. **Verify**: `eth_getBalance` 변화 + `eth_call(0x1000, allowance(A,B)) == 2ether` + Transfer 이벤트

#### RT-F-1-04 — Mint 시 Transfer(0x0→beneficiary) 이벤트 ✓ 

- 검증: `core/vm/native_manager.go:202-216` `coinManagerMint.Run()` → `nativeTransfer(evm, zeroAddress, to, amount)` → `evm.AddTransferLog(zero, to, amount)` (`native_manager.go:186`)
- 검증: `NativeCoinAdapter.sol:152-153` `emit Mint(...); // Transfer event is emitted by the EVM`
- **시나리오**:
    1. **Setup**: GovMinter 멤버 권한 계정, beneficiary 주소 B (mint 대상)
    2. **Action**: `proposeMint(proof)` → quorum 승인 → `executeProposal(id)` (RT-F-2-01 흐름)
    3. **Expected**: execute tx receipt.logs에 NativeCoinAdapter Transfer 이벤트 포함
    4. **Verify**:
        - `eth_getTransactionReceipt(execTx).logs` 내 `address == "0x...1000"` 인 log 검색
        - 해당 log: `topics[0] == keccak256("Transfer(address,address,uint256)")`, `topics[1] == 0x0...0` (from=zero), `topics[2] == B` (to=beneficiary), `data == amount`
        - `eth_getBalance(B)` 변화 = +amount
        - `eth_call(0x1000, totalSupply())` 증가량 = amount

#### RT-F-1-05 — Burn 시 Transfer(account→0x0) 이벤트 ✓ 

- 검증: `core/vm/native_manager.go:225-249` `coinManagerBurn.Run()` → `evm.StateDB.SubBalance(from, amount)` → `evm.AddTransferLog(from, zeroAddress, amount)`
- 검증: `NativeCoinAdapter.sol:172-173` `emit Burn(...); // Transfer event is emitted by the EVM`
- **시나리오**:
    1. **Setup**: GovMinter 멤버 권한 계정 M (burn 대상도 동일 — `proposeBurn`은 `payable`로 `msg.value == proof.amount` 선납)
    2. **Action**: `proposeBurn(proof) {value: proof.amount}` → quorum 승인 → execute (RT-F-2-02 흐름)
    3. **Expected**: execute tx receipt.logs에 NativeCoinAdapter Transfer(M→0x0) 이벤트 포함
    4. **Verify**:
        - `eth_getTransactionReceipt(execTx).logs` 내 `address == "0x...1000"` log
        - 해당 log: `topics[0] == keccak256("Transfer(address,address,uint256)")`, `topics[1] == M` (from=account), `topics[2] == 0x0...0` (to=zero), `data == amount`
        - `eth_getBalance(M)` 변화 = -amount (msg.value 선납 고려)
        - `eth_call(0x1000, totalSupply())` 감소량 = amount

> **제거된 TC**: 구 RT-F-1-03 "가스비 지불 Transfer 이벤트"는 TC가 잘못 작성된 것으로 확인되어 제거됨. 가스비 처리 경로(`buyGas`/TransitionDb fee 분배/`distributeBaseFee`)는 모두 `state.SubBalance`/`AddBalance`만 호출하며 `evm.AddTransferLog()`가 호출되지 않음 → `Transfer(sender→validator, gasFee)` 이벤트는 발생하지 않음. v2에서 F-1 카테고리가 5개로 재번호매김됨.

### F-2. GovMinter (0x1003)

#### RT-F-2-01 — 코인 발행 (proposeMint → 승인 → execute) ✓

- 검증: `proposeMint(proofData)` → quorum 승인 → `executeProposal` → `_executeCustomAction(ACTION_MINT_WITH_DEPOSIT)` → `fiatToken.mint(beneficiary, amount)`
- **시나리오**:
    1. **Setup**: GovMinter 멤버 권한 계정 M1\~Mn (genesis에서 설정), quorum=Q
    2. **Action**: M1이 `proposeMint(proof)` 호출 → M1\~MQ가 `approve(proposalId)` → 마지막 승인 시점 또는 이후 `executeProposal(proposalId)`
    3. **Expected**: beneficiary 잔액 += proof.amount, totalSupply 증가
    4. **Verify**: `eth_getBalance(beneficiary)` 변화, `eth_call(0x1000, totalSupply())` 증가, `Mint` 이벤트

#### RT-F-2-02 — 코인 소각 ✓

- 검증: v1.0.0의 `GovMinter.proposeBurn(proofData)`은 `payable`이며 `msg.value == proof.amount` 선납 모델. `_safeBurn(from, amount, withdrawalId)` → `fiatToken.burn(...)`
- **시나리오**:
    1. **Setup**: Minter 멤버
    2. **Action**: `proposeBurn(proof) {value: proof.amount}` → quorum 승인 → execute
    3. **Expected**: from 잔액 차감, totalSupply 감소
    4. **Verify**: `eth_call(0x1000, totalSupply())` 감소, `Burn` 이벤트

#### RT-F-2-03 — quorum 미달 발행 미실행 ✓ 

- 검증: `systemcontracts/solidity/abstracts/GovBase.sol:96-105` `ProposalStatus` enum `None, Voting, Approved, Executed, Cancelled, Expired, Failed, Rejected` — "Pending" 없음
- 검증: quorum 미달 상태에서 `executeProposal` 호출 → `ExecutionCheckResult.NotApproved` → revert. 해당 enum 주석(line 112)이 "may be Voting"이라고 명시
- **v2 변경**: v1의 RT-F-2-04(중복)는 제거됨. v2에서 RT-F-2-03 상태명 표기도 "Pending" → "Voting"으로 정정됨
- **시나리오**:
    1. **Setup**: Minter 멤버 quorum=5/7
    2. **Action**: 4명만 승인 → `executeProposal(id)` 호출 시도
    3. **Expected**: 제안 상태 = `Voting` (생성 시 상태 유지), execute tx revert
    4. **Verify**:
        - `proposalState(id)` == `Voting` (enum 값 1)
        - execute tx `receipt.status == 0` 또는 tx가 revert 에러 반환 (`ProposalNotExecutable` / `NotApproved`)
        - 제안 금액에 해당하는 beneficiary 잔액 변화 없음

### F-3. GovValidator (0x1001)

> **운영 모델**: `GovValidator.configureValidator()`는 **초기 설정 전용** — member가 자신의 validator 주소/BLS 키를 등록할 때 사용. **운영 중 검증자 변경(추가/제거)은 GovBase member 거버넌스 투표 흐름**으로 진행:
> 
> - **추가**: `proposeAddMember` → quorum 승인 → execute → 새 member 추가 → 새 member가 `configureValidator`로 자기 validator 주소/BLS 키 등록
> - **제거**: `proposeRemoveMember` → quorum 승인 → execute → `_onMemberRemoved` 훅이 `_removeValidator` 자동 호출
> - 참고: `_onMemberAdded`는 `// do nothing` — member 추가만으로는 validator가 자동 등록되지 않으며, 새 member의 `configureValidator` 호출 필요
> - GasTip은 별도 proposal 흐름(`proposeGasTip` → ACTION\_SET\_GAS\_TIP)

#### RT-F-3-01 — 검증자 추가 제안 생성 ✓

- 검증: GovBase `proposeAddMember(newMember)` → 제안 ID 반환, 상태 `Voting`
- **시나리오**:
    1. **Setup**: GovValidator 기존 member 계정, 추가할 신규 member 주소 M\_new (validator 운영자가 될 계정)
    2. **Action**: `proposeAddMember(M_new)` 호출
    3. **Expected**: 제안 생성, `proposalState(id) == Voting`, 제안 ID 반환
    4. **Verify**: `eth_getTransactionReceipt(tx).logs`에 `ProposalCreated(id, ...)` 이벤트 포함, `isProposalInVoting(id) == true`

#### RT-F-3-02 — 검증자 추가 제안 승인 — quorum 달성 시 실행 ✓

- 검증: member quorum 승인 → `executeProposal` → `_onMemberAdded`(`// do nothing`) → 새 member가 이어서 `configureValidator(self, blsKey, blsSig)` 호출 → `__validators.add()`
- **시나리오 (2단계)**:
    1. **Setup**: RT-F-3-01에서 생성한 제안 ID, quorum 수만큼의 기존 member 계정
    2. **Action 1 (투표)**: member quorum이 `approveProposal(id)` 호출 → `executeProposal(id)` → M\_new가 GovValidator member로 등록
    3. **Action 2 (validator 등록)**: M\_new가 자신의 BLS 키 페어를 준비하여 `configureValidator(validatorAddr, blsKey, blsSig)` 호출
    4. **Expected**: 
        - 제안 상태 `Executed`
        - `__validators` 집합에 validatorAddr 추가 → `validatorList()` 응답에 포함
        - **다음 에폭 블록부터** WBFTExtra.EpochInfo에 신규 validator 반영 (현재 에폭 내에서는 실제 서명 참여 X)
    5. **Verify**:
        - `proposalState(id) == Executed`
        - `validatorList()`에 validatorAddr 포함
        - 에폭 경계 블록의 `istanbul_getWbftExtraInfo(epochBlock).epochInfo.candidates`에 validatorAddr 포함
        - 이후 블록의 `committedSeal.sealers` bitmap에 해당 validator의 bit가 set

#### RT-F-3-03 — 검증자 제거 제안 — quorum 달성 시 실행 ✓

- 검증: GovBase `proposeRemoveMember(member)` → quorum 승인 → `executeProposal` → `_onMemberRemoved` override → `_removeValidator(validator, member)` 자동 호출
- **시나리오**:
    1. **Setup**: 현재 GovValidator member이자 validator 운영자인 계정 M\_target, quorum 수만큼의 다른 member 계정
    2. **Action**: `proposeRemoveMember(M_target)` → quorum member가 `approveProposal(id)` → `executeProposal(id)`
    3. **Expected**:
        - 제안 상태 `Executed`
        - `_onMemberRemoved(M_target)` 훅이 `operatorToValidator[M_target]`를 조회하여 해당 validator 자동 제거
        - `__validators.remove(validator)` 실행 → `validatorList()`에서 제외
        - **다음 에폭 블록부터** WBFTExtra.EpochInfo에서 제외
    4. **Verify**:
        - `validatorList()`에서 해당 validator 주소 제외
        - `validatorToOperator[validator] == 0`, `validatorToBlsKey[validator].length == 0` (delete 처리)
        - 다음 에폭 블록 `istanbul_getWbftExtraInfo` `epochInfo.candidates`에서 제외
        - 이후 블록 `committedSeal.sealers` bitmap에서 해당 validator 제외

#### RT-F-3-04 — 검증자 메타데이터 다중 호출 조회 ✓

- 검증: 단일 `getValidators()` 함수가 없으므로 `validatorList()` + 두 public mapping을 조합해 운영키·검증키·BLS키 세트를 구성. v2에서 다중 호출 시퀀스로 명시 정정 완료
- **시나리오**:
    1. **Action**: 
        - `eth_call(0x1001, validatorList())` → `[v1, v2, ..., v7]` (검증키 배열)
        - 각 vi에 대해 `eth_call(0x1001, validatorToOperator(vi))` → 운영키
        - 각 vi에 대해 `eth_call(0x1001, validatorToBlsKey(vi))` → BLS키
    2. **Expected**: 7개 트리플 (operator, validator, blsKey) — 총 1 + 2N 호출
    3. **Verify**: 각 BLS key 길이 = 48 bytes (`BLS_PUBLIC_KEY_LENGTH`), `validatorToOperator`/`operatorToValidator` 매핑 일관성

#### RT-F-3-05 — GasTip 거버넌스 lifecycle (proposeGasTip → 승인 → execute) ✓

- **책임**: 거버넌스 함수 호출 결과로 컨트랙트 storage `gasTip`이 변경되고 이벤트가 발생하는 부분만 검증 (헤더 반영은 RT-B-06에서 별도 검증)
- 검증: `proposeGasTip(newTip)` → `_createProposal(ACTION_SET_GAS_TIP, ...)` → quorum → `executeProposal()` → `_executeCustomAction(ACTION_SET_GAS_TIP, ...)` → `_setGasTip(newTip)` → storage `gasTip` 갱신 + `GasTipUpdated` 이벤트
- **시나리오**:
    1. **Setup**: 현재 컨트랙트 storage `gasTip = T1`
    2. **Action**: 운영키(member)가 `proposeGasTip(T2)` 호출 → quorum 이상 멤버가 `voteProposal(id, true)` → `executeProposal(id)`
    3. **Expected**:
        - 제안 상태 전이: `Voting` → `Approved` → `Executed`
        - `eth_call(0x1001, gasTip())` 결과 == T2
        - `_setGasTip` 내부에서 발생한 `GasTipUpdated(oldTip=T1, newTip=T2, updater=msg.sender)` 이벤트가 receipt logs에 포함
    4. **Verify**:
        - `proposalState(id) == Executed`
        - `gasTip` storage slot(0x39) 직접 조회로 T2 확인
        - 이벤트 `topics[0] == keccak256("GasTipUpdated(uint256,uint256,address)")`
- **음성 케이스**:
    - 동일 값 제안 시 `SameGasTip` revert (`GovValidator.sol:180`)
    - non-member 호출 시 `onlyActiveMember` revert
- **근거**: `systemcontracts/solidity/v1/GovValidator.sol:178-190` `proposeGasTip`/`_setGasTip`, `:148-155` `_executeCustomAction`

#### RT-F-3-06 — expiry 초과 자동 만료 ✓

- 검증: `GovBase.expireProposal(id)` — `block.timestamp > createdAt + proposalExpiry` 시 상태 → `Expired`
- **시나리오**:
    1. **Setup**: `proposalExpiry`를 짧게 설정한 환경 또는 충분한 시간 경과
    2. **Action**: 제안 생성 → expiry 초과까지 대기 → `expireProposal(id)`
    3. **Expected**: 상태 = `Expired`, 이후 `executeProposal(id)` 호출 시 revert
    4. **Verify**: `proposalState(id) == Expired`, execute tx revert

### F-4. GovMasterMinter (0x1002)

#### RT-F-4-01 — Minter 등록 ✓

- 검증: `proposeConfigureMinter(minter, allowance)` → ACTION\_CONFIGURE\_MINTER → `_safeConfigureMinter()` → `fiatToken.configureMinter(...)`
- **시나리오**: GovMasterMinter 멤버 quorum 승인 → execute → GovMinter `isMinter[m] == true`

#### RT-F-4-02 — Minter 삭제 ✓

- 검증: `proposeRemoveMinter(m)` → `_safeRemoveMinter()` → `fiatToken.removeMinter(m)`
- **시나리오**: 동일 흐름, 결과 `isMinter[m] == false`

#### RT-F-4-03 — GovMasterMinter 멤버 추가/제거 ✓

- 검증: GovBase 상속 — `proposeAddMember`/`proposeRemoveMember` (ACTION\_ADD\_MEMBER 등)
- **시나리오**: quorum 승인 → execute → `versionedMemberList[ver]` 변경 → 새 멤버가 propose 가능

#### RT-F-4-04 — 비멤버 제안 거부 ✓

- 검증: `onlyActiveMember` modifier → `NotAMember()` revert
- **시나리오**: 비멤버 X가 `proposeConfigureMinter(...)` 호출 → tx revert

### F-5. GovCouncil (0x1004)

#### RT-F-5-01 — blacklist 등록 ✓

- 검증: `proposeAddBlacklist(account)` → ACTION\_ADD\_BLACKLIST → `__accountManager.blacklist(account)`
- **시나리오**: GovCouncil 멤버 quorum 승인 → execute → `isBlacklisted(account) == true`

#### RT-F-5-02 — unBlacklist ✓

- 검증: `proposeRemoveBlacklist(account)` → `__accountManager.unBlacklist(account)`

#### RT-F-5-03 — authorize ✓

- 검증: `proposeAddAuthorizedAccount(account)` → `__accountManager.authorize(account)`

#### RT-F-5-04 — unAuthorize ✓

- 검증: `proposeRemoveAuthorizedAccount(account)` → `__accountManager.unAuthorize(account)`

#### RT-F-5-05 — 비멤버 직접 blacklist 호출 거부 ✓

- 검증: AccountManager의 mutator는 GovCouncil(또는 시스템)만 호출 가능 (access control)
- **시나리오**: 일반 EOA가 `eth_sendTransaction({to:0xb00003, data: blacklist(X)})` 호출 → revert

#### RT-F-5-06 — blacklist 시 AddressBlacklisted 이벤트 ✓ \[v2 신규\]

- 검증: `systemcontracts/solidity/v1/GovCouncil.sol:87` `event AddressBlacklisted(address indexed account, uint256 indexed proposalId);`
- **시나리오**:
    1. **Setup**: GovCouncil 멤버 권한 계정, 대상 계정 X
    2. **Action**: `proposeAddBlacklist(X)` → quorum 승인 → `executeProposal(id)` (RT-F-5-01 흐름)
    3. **Expected**: execute tx receipt에 `AddressBlacklisted(X, id)` 이벤트
    4. **Verify**:
        - `eth_getTransactionReceipt(execTx).logs` 내 `address == "0x...1004"` (GovCouncil) log 검색
        - `topics[0] == keccak256("AddressBlacklisted(address,uint256)")`
        - `topics[1] == X` (32 bytes padded), `topics[2] == proposalId` (32 bytes)
        - 동시에 `eth_call(0xb00003, isBlacklisted(X)) == true`

#### RT-F-5-07 — unBlacklist 시 AddressUnblacklisted 이벤트 ✓ \[v2 신규\]

- 검증: `GovCouncil.sol:92` `event AddressUnblacklisted(address indexed account, uint256 indexed proposalId);`
- **시나리오**:
    1. **Setup**: 사전에 blacklist된 계정 X (RT-F-5-01/06으로 등록 완료)
    2. **Action**: `proposeRemoveBlacklist(X)` → quorum 승인 → execute (RT-F-5-02 흐름)
    3. **Expected**: receipt에 `AddressUnblacklisted(X, id)` 이벤트
    4. **Verify**:
        - GovCouncil(0x...1004) log: `topics[0] == keccak256("AddressUnblacklisted(address,uint256)")`, `topics[1] == X`, `topics[2] == proposalId`
        - `eth_call(0xb00003, isBlacklisted(X)) == false`

#### RT-F-5-08 — authorize 시 AuthorizedAccountAdded 이벤트 ✓ 

- 검증: `GovCouncil.sol:97` `event AuthorizedAccountAdded(address indexed account, uint256 indexed proposalId);`
- **시나리오**:
    1. **Setup**: GovCouncil 멤버, 대상 계정 Y
    2. **Action**: `proposeAddAuthorizedAccount(Y)` → quorum 승인 → execute (RT-F-5-03 흐름)
    3. **Expected**: receipt에 `AuthorizedAccountAdded(Y, id)` 이벤트
    4. **Verify**:
        - GovCouncil log: `topics[0] == keccak256("AuthorizedAccountAdded(address,uint256)")`, `topics[1] == Y`, `topics[2] == proposalId`
        - `eth_call(0xb00003, isAuthorized(Y)) == true`

#### RT-F-5-09 — unAuthorize 시 AuthorizedAccountRemoved 이벤트 ✓ \[v2 신규\]

- 검증: `GovCouncil.sol:102` `event AuthorizedAccountRemoved(address indexed account, uint256 indexed proposalId);`
- **시나리오**:
    1. **Setup**: 사전에 authorized된 계정 Y (RT-F-5-03/08로 등록)
    2. **Action**: `proposeRemoveAuthorizedAccount(Y)` → quorum 승인 → execute (RT-F-5-04 흐름)
    3. **Expected**: receipt에 `AuthorizedAccountRemoved(Y, id)` 이벤트
    4. **Verify**:
        - GovCouncil log: `topics[0] == keccak256("AuthorizedAccountRemoved(address,uint256)")`, `topics[1] == Y`, `topics[2] == proposalId`
        - `eth_call(0xb00003, isAuthorized(Y)) == false`
        - 직후 Y가 발행하는 tx에 RT-E-09의 `AuthorizedTxExecuted` 이벤트가 더 이상 발생하지 않음 (음성 검증)

---

## Section G — API 호출

### G-1. eth 블록/트랜잭션 조회

#### RT-G-1-01 — eth\_getBlockByNumber ✓

- **시나리오**: `eth_getBlockByNumber("latest", true)` → number/hash/transactions 포함 객체

#### RT-G-1-02 — eth\_getBlockByHash ✓

- **시나리오**: G-1-01의 hash로 `eth_getBlockByHash(hash, true)` → 동일 객체

#### RT-G-1-03 — eth\_getTransactionByHash ✓

- **시나리오**: 블록에 포함된 tx hash로 호출 → blockNumber/from/to/value 포함

#### RT-G-1-04 — eth\_getTransactionReceipt ✓ \[dev에서 해결\]

- 검증: `core/types/receipt.go:396-411` (dev, PR #70 merge 후) — Anzeon 활성 시 derive 단계에서 EffectiveGasPrice fallback 계산
- **시나리오**:
    1. **Setup**: BP1(채굴) + EN1(snap-synced)
    2. **Action**: BP1에 type 0x2 tx 전송 → 블록 N 포함
    3. **Action 2**: BP1, EN1 양쪽에 `eth_getTransactionReceipt(txHash)`
    4. **Expected**: 양쪽 모두 `effectiveGasPrice != null` 동일 값
    5. **Verify**: 
        - 두 응답 receipt 객체의 `effectiveGasPrice`, `status`, `logs`, `gasUsed` 모두 동일
        - **이 비교 자체가 PR #70 회귀 검증 핵심** — sync 경로의 derive 동작이 미래에 깨지면 이 TC가 fail해야 함

#### RT-G-1-05 — eth\_getTransactionCount ✓

- **시나리오**: 발행 tx 수와 일치하는 nonce 반환

#### RT-G-1-06 — eth\_getCode (시스템 컨트랙트) ✓

- **시나리오**: `eth_getCode("0x0000000000000000000000000000000000001000", "latest")` → 비어있지 않은 bytecode

### G-2. 가스/수수료

#### RT-G-2-01 — eth\_gasPrice = baseFee + GasTip ✓

- 검증: `internal/ethapi/api.go:2105` `tipcap = oracle.SuggestGasTipCap; tipcap.Add(tipcap, head.BaseFee)`
- **시나리오**:
    1. **Setup**: 현재 baseFee=B, header.GasTip=T (oracle은 일반적으로 T 반환)
    2. **Action**: `eth_gasPrice`
    3. **Expected**: 응답 hex == B + T
    4. **Verify**: `eth_getBlockByNumber("latest").baseFeePerGas` + `istanbul_getWbftExtraInfo(latest).gasTip` 합과 일치

#### RT-G-2-02 — eth\_maxPriorityFeePerGas ✓ 

- 검증: 호출 경로 — `EthereumAPI.MaxPriorityFeePerGas()` → `b.SuggestGasTipCap()` → `EthAPIBackend.SuggestGasTipCap()` (`eth/api_backend.go:365-372`):

```go
if b.eth.blockchain.Config().AnzeonEnabled() {
    return b.eth.Miner().GetGasTip(), nil  // ← Anzeon에서 oracle 우회, GovValidator 직접 반환
}
return b.gpo.SuggestTipCap(ctx)
```
- 검증: `Miner.GetGasTip()`의 출처는 `worker.gasTip`, 이 값은 `worker.updateGasTipFromContract()` (`miner/worker.go:1200-1225`)이 매 블록 import 시 GovValidator 컨트랙트의 gasTip 슬롯을 읽어 갱신
- **결론**: `eth_maxPriorityFeePerGas`는 Anzeon 환경에서 항상 **현재 GovValidator 컨트랙트의 gasTip = WBFTExtra.GasTip 출처**와 동일한 값을 반환. **TC 기대값과 정확히 일치**
- **시나리오**:
    1. **Setup**: Anzeon 활성, 현재 GovValidator gasTip = T (= 새 블록 헤더의 WBFTExtra.GasTip)
    2. **Action**: `eth_maxPriorityFeePerGas`
    3. **Expected**: 응답 == T
    4. **Verify**: `eth_maxPriorityFeePerGas` == `istanbul_getWbftExtraInfo("latest").gasTip`
- **회귀 검증 포인트**:
    - GasTip 거버넌스 변경 후(RT-B-06/F-3-05) 다음 블록 import 시점에 `eth_maxPriorityFeePerGas` 응답이 새 값으로 즉시 반영되는지 확인
    - 비-Anzeon 환경(테스트 픽스처 등)에서는 oracle 경로(`gpo.SuggestTipCap`)를 타므로 별도 분기
- **이전 리뷰 오류 노트**: 초기 리뷰는 `internal/ethapi/api.go:83-89`만 보고 "oracle 기반"이라고 단정했으나, **Backend 인터페이스의 다형성**(`EthAPIBackend.SuggestGasTipCap()`의 Anzeon 분기)을 놓침. dev/v1.0.0 모두 동일한 구조

#### RT-G-2-03 — eth\_feeHistory ✓

- 검증: `MinBaseFee = 20000000000000`
- **시나리오**:
    1. **Action**: `eth_feeHistory("0x10", "latest", [])`
    2. **Expected**: `baseFeePerGas[]` 모두 ≥ 20000000000000 (16진수 비교)
    3. **Verify**: 배열 각 원소를 BigInt로 변환 후 검증

#### RT-G-2-04 — eth\_estimateGas (NativeCoinAdapter.transfer) ✓

- **시나리오**: `eth_estimateGas({from:A, to:0x1000, data: encode("transfer(address,uint256)",B,1ether)})` → 양수, 실제 receipt.gasUsed ≤ 추정값

### G-3. WBFT 커스텀 API (`istanbul_*`)

#### RT-G-3-01 — istanbul\_nodeAddress ✓

- 검증: `consensus/wbft/backend/api.go:NodeAddress() → backend.Address()`
- **시나리오**: 검증자 노드에서 `istanbul_nodeAddress` 호출 → coinbase 주소 (RT-G-3-02 결과의 한 원소)

#### RT-G-3-02 — istanbul\_getValidators ✓

- 검증: `API.GetValidators(*rpc.BlockNumber) []common.Address`
- **시나리오**: `istanbul_getValidators("latest")` → 7개 주소 배열

#### RT-G-3-03 — istanbul\_getCommitSignersFromBlock ✓

- 검증: `BlockSigners { Number, Hash, Author, Committers []common.Address }`
- **시나리오**: 블록 N에 대해 호출 → `{author, committers}`, len(committers) ≥ 5 (quorum)

#### RT-G-3-04 — istanbul\_getWbftExtraInfo ✓

- 검증: `GetWbftExtraInfo(rpc.BlockNumber) map[string]interface{}` — vanityData/randaoReveal/prevRound/prevPreparedSeal/prevCommittedSeal/round/preparedSeal/committedSeal/gasTip/epochInfo
- **시나리오**: 블록 N에 대해 호출 → 위 필드 모두 존재

#### RT-G-3-05 — istanbul\_status ✓

- 검증: `Status { SealerActivity, AuthorCounts, BlockRange, RoundStats }`
    - `SealerActivity { Total, Prepared, Committed, PrevPrepared, PrevCommitted }`
    - `RoundStats { RoundDistribution map, TotalRounds }`
- **시나리오**: `istanbul_status(<start>, <end>)` → 통계 객체. 합계 일관성 검증 (예: BlockRange.TotalBlocks = end-start+1)

#### RT-G-3-06 — istanbul\_isValidator ✓

- 검증: `IsValidator(*rpc.BlockNumber) bool` — 현재 노드 주소를 GetValidators 결과와 비교
- **시나리오**: BP 노드 = true, EN 노드 = false

### G-4. 관리/진단 API

#### RT-G-4-01 — net\_peerCount ✓

- **시나리오**: 2노드 이상 연결 환경에서 ≥ 1 반환

#### RT-G-4-02 — txpool\_status ✓ \[v2에서 강화, TC와 코드 일치\]

- 검증: `internal/ethapi/api.go:223-229` `TxPoolAPI.Status()`가 `b.Stats()` 반환값(pending, queue)을 `{pending, queued}` 맵으로 반환
- 검증: legacypool에서 현재 계정 nonce부터 연속되는 tx는 `pending`, nonce gap이 있는 tx는 `queue`로 분리
- **시나리오**:
    1. **Setup**: 계정 A의 현재 onchain nonce = N
    2. **Action 1 (pending)**: nonce N, N+1 두 tx를 채굴 차단 상태(또는 즉시) 주입 → pending에 2건
    3. **Action 2 (queued)**: nonce N+10 (gap 있음) tx를 추가 주입 → queued에 1건
    4. **Action 3**: `txpool_status` 호출
    5. **Expected**: 응답 `{pending: "0x2", queued: "0x1"}` (정확한 수)
    6. **Verify**: 주입 수와 응답 수 일치, 오류 없이 응답

#### RT-G-4-03 — txpool\_content ✓ 

- 검증: `internal/ethapi/api.go:173-197` `TxPoolAPI.Content()`가 `b.TxPoolContent()` 반환값(pending, queue)을 각각 `content["pending"]`/`content["queued"]`에 `account.Hex() → nonce → NewRPCPendingTransaction` 구조로 채워 반환 (from, nonce, gasPrice/maxFeePerGas, value, to, hash 등 모든 필드 포함)
- **시나리오**:
    1. **Setup**: RT-G-4-02와 동일한 pending(N, N+1)/queued(N+10) 상태
    2. **Action**: `txpool_content`
    3. **Expected**:
        - `pending[A]` 객체에 N, N+1 두 tx 포함
        - `queued[A]` 객체에 N+10 tx 포함
    4. **Verify**:
        - 각 tx 객체의 `from`, `nonce`, `gasPrice`/`maxFeePerGas`, `value`, `to`, `hash` 필드 정확
        - pending 객체에 N+10이 포함되지 않음 (음성)
        - queued 객체에 N, N+1이 포함되지 않음 (음성)

#### RT-G-4-04 — admin\_peers ✓

- **시나리오**: 배열, 각 원소 `id/name/network/protocols` 등 포함

### G-5. StableNet 고유 API

#### RT-G-5-01 — eth\_signRawFeeDelegateTransaction ✓

- 검증: `internal/ethapi/api.go:2186 TransactionAPI.SignRawFeeDelegateTransaction(args, input hexutil.Bytes)` (그리고 `:629 PersonalAccountAPI`도 password 버전 존재)
- **시나리오**:
    1. **Setup**: `Sender A가 DynamicFeeTx(type 0x2) 본체 서명(V/R/S) → RLP 직렬화 → hex`
    2. **Action**: `eth_signRawFeeDelegateTransaction({feePayer: P, ...args}, "0x<rlp>")` 호출 (P 키가 노드에 unlock된 상태)
    3. **Expected**: FV/FR/FS 채워진 완성된 RLP가 `result.raw`에 반환
    4. **Verify**: 반환 RLP를 디코드하여 FV/FR/FS 존재 확인 → `eth_sendRawTransaction(result.raw)`로 정상 전파

#### RT-G-5-02 — eth\_call NativeCoinAdapter.totalSupply ✓

- 검증: `NativeCoinAdapter.totalSupply() → __totalSupply`
- **시나리오**:
    1. **Action**: `eth_call({to:"0x1000", data:"0x18160ddd"})` (totalSupply selector)
    2. **Expected**: 현재 총 발행량 (uint256 ABI)
    3. **Verify**: alloc에 등록된 모든 계정의 `eth_getBalance` 합과 일치 (소수의 계정 환경에서 검증)

#### RT-G-5-03 — eth\_call NativeCoinAdapter.allowance ✓

- 검증: `allowance(owner, spender) → __allowed[owner][spender]`
- **시나리오**:
    1. **Setup**: A가 B에 `approve(B, 5ether)` 실행
    2. **Action**: `eth_call({to:"0x1000", data: encode("allowance(address,address)", A, B)})`
    3. **Expected**: 5×10^18
    4. **Verify**: 정확한 hex 값 확인

---
