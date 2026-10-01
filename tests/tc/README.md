# tests/tc — 레거시 스위트와 같은 구조로 정리한 테스트 케이스

`~/Work/github/packages/chainbench/tests` 의 셸 스위트를 그대로 따라가도록 배치했다. DSL 문서는 자기가 쓸 config·genesis 를 파일 안에 적으므로, 레거시의 `configs/` 같은 곁가지는 옮기지 않았다.

## 1. 구조

1단은 체인, 그 아래는 레거시의 도메인 이름을 그대로 쓴다.

```
tests/tc/
├── common/                 ← 세 체인에서 같은 목적으로 도는 것 (69건)
│   └── {node,tx,fee,contract,rpc,fault}/
├── go-stablenet/           ← 레거시 tests/stablenet
│   ├── regression/{ethereum,wbft,anzeon,
│   │                blacklist-authorized,system-contracts,api}
│   └── post-v1.0.0-change/{common-all,extra-state,
│                           effectivegasprice,string-handling,stand-alone}
├── go-wbft/                ← 레거시 tests/wemix4 중 wbft 대상 + wbft 전용
├── go-wemix/               ← 레거시 tests/wemix4 중 wemix(poa) 대상
└── basic/                  ← 레거시 동명 폴더 중 공통으로 가지 않은 것
```

`common/` 이 1단에 있는 것은 체인이 아니어서다. 나머지 1단은 한 체인의 것이고,
`common/` 은 어느 체인에서도 같은 목적으로 돌 수 있는 것이다. 무엇이 왜 거기 있는지는
[`common/README.md`](common/README.md) 가 적는다.

레거시 동명 폴더였던 `fault/`·`stress/`·`remote/`·`samples/` 는 담고 있던 케이스가
전부 공통으로 갔고, `basic/` 은 8건 중 6건이 갔다. 남은 둘은
`07-basic-wbft-consensus`(wbft 합의 전용)와 `08-attached-chain-produces`(attach 선언을
검사하는 유일한 케이스)다.

공통으로 갔다가 되돌아온 것이 여덟 있다. 2026-09-28 에 세 체인 소스를 대조하니
go-stablenet 에만 있는 `gasTip`·`MinBaseFee` 에 기대고 있었다. 내역은
[`common/README.md`](common/README.md) §4.1 이다.

파일명 규칙은 두 갈래다. `common/` 은 `CT-<영역>-<번호>-<간략설명>.json` 이고 **CT 하나에
파일 하나**다(2026-09-29·30 에 여섯 영역 모두 적용). 체인별 디렉터리는 레거시 대조를 위해
`<레거시 번호>-<테스트 id>.json` 을 그대로 쓴다. 번호는 레거시 스위트의 순번이고, 뒤의 id 가
무엇을 검증하는지 말한다. 레거시 하나가 여러 스펙으로 나뉜 경우 `11-`, `11b-` 처럼 뒤에
글자를 붙였다.

번호가 비어 있는 자리는 그 레거시 테스트가 DSL 이 아니라 `tests/e2e/` 의 Go 테스트로
옮겨진 자리다. 3절에 목록이 있다.

### 문서 한 장이 전부다

모든 문서가 `schemaVersion: "2"` 의 케이스이고, **체인 구성을 파일 안에 담는다.**
하나를 열면 어떤 체인 위에서 어떤 바이너리로 어떤 genesis·config 를 쓰는지, 그리고
무엇을 검증하는지가 그 안에 있다. 공용 `chain-preset/` 디렉터리는 없앴다 — id 로 부르면
파일만 봐서는 어떤 네트워크인지 알 수 없기 때문이다.

`description` 은 그 문서가 무엇을 검증하는지 사람이 읽으라고 있는 자리다. 레거시에
대응이 있는 것은 원본의 id 와 이름을 옮겨 적었다.

함께 돌릴 케이스는 chain-preset 이 같아야 한다. 같은 폴더에 있어도 다를 수 있다 —
`string-handling` 여섯 건은 각자 다른 genesis 를 요구한다. 실행기가 구성이 갈리는
묶음을 거부하므로(`sameComposition`) 조용히 틀린 네트워크에서 돌지는 않는다.

### chain-preset 이 같다고 같이 돌릴 수 있는 것은 아니다

`sameComposition` 이 보는 것은 **어떤 망을 세우는가** 이지 그 망에 무엇을 하는가가
아니다. 체인 상태를 바꾸는 케이스끼리는 chain-preset 이 같아도 서로를 밟는다.

케이스가 "자기 완결" 이라고 적힌 것은 **깨끗한 망을 기준으로** 그렇다는 뜻이고,
서로에 대해 독립이라는 뜻이 아니다. 잰 예가 있다(2026-09-21).

| | 하는 일 |
|---|---|
| `regression/wbft/04-validator-add-member-executes` | `0x5c646aa6(node5, 4)` — node5 를 멤버로 추가 |
| `regression/wbft/05-validator-remove-member-executes` | `0x5c646aa6(node5, 2)` 로 **자기가 먼저 추가**한 뒤 제거 |

한 망에 04 를 먼저 올리면 node5 가 이미 멤버라 05 의 추가가 revert 하고, 05 는
`step 2 (sendTx) ... reverted (status 0x0)` 로 실패한다. 둘 다 단독으로는 통과한다.

멤버 집합이 바뀌면 그 뒤에 오는 **다른 거버넌스 케이스**도 영향을 받는다. 제안이
자동 실행되는 데 필요한 승인 수가 멤버 수에 달려 있기 때문이다.

그래서 **여러 케이스를 한 망에 올릴 때는 상태를 바꾸는 것이 있는지 먼저 본다.**
확실하게 하려면 케이스마다 망을 새로 세우고 끝나면 내린다 — 조립 시간이 케이스 수만큼
들지만, 결과가 그 케이스에 대한 것이 된다.

작성 방법은 `../../docs/guide/dsl-authoring.md` 에 있다.

## 2. 디렉터리별 내용

2026-10-01 에 센 값이다. 케이스가 늘거나 줄면 이 표도 고친다.

| 디렉터리 | 건수 | 무엇이 있나 |
|---|---|---|
| `common/node` | 15 | 기동·동기화·네트워크 |
| `common/tx` | 20 | 트랜잭션 전송·거부 |
| `common/fee` | 7 | 수수료·가스 정책 |
| `common/contract` | 7 | 컨트랙트 실행 |
| `common/rpc` | 14 | 조회·구독 API |
| `common/fault` | 6 | 장애·복구 |
| `go-stablenet/regression/*` | 68 | anzeon 11, api 14, blacklist-authorized 9, ethereum 3, system-contracts 23, wbft 8 |
| `go-stablenet/post-v1.0.0-change/*` | 35 | common-all 16, extra-state 8, string-handling 6, effectivegasprice 3, stand-alone 2 |
| `go-stablenet/{hardfork,vocabulary}` | 3 | |
| `go-stablenet/testnet` | 5 | 우리가 세우지 않은 go-stablenet 망에 붙어 도는 케이스 |
| `go-wbft/*` | 8 | accounts 3, governance 2, consensus 1, fault 1, tx 1 |
| `go-wemix/*` | 7 | hardfork 3, consensus 1, governance 1, rpc 1, vocabulary 1 |
| `basic` | 2 | 공통으로 가지 않고 남은 둘 |
| **합계** | **197** | |

케이스 하나하나가 무엇을 보는지는 디렉터리의 문서가 적는다. 여기에 같은 목록을 한 벌 더
두었다가 트리와 어긋났다 — 2026-09-30 까지 이 자리에는 재편 이전의 디렉터리(`fault/`,
`remote/`, `samples/`, `stress/`)와 없어진 파일 이름이 남아 있었다.

- `common/` — [`common/README.md`](common/README.md) 가 CT 별 파일과 그 사정을,
  [`common/HOW-TO-USE.md`](common/HOW-TO-USE.md) 가 케이스별 실행 명령을 적는다.
- 체인별 디렉터리 — 각 스펙 파일의 `description` 이 무엇을 왜 그렇게 보는지 적는다.
  `chainbench test list <디렉터리>` 로 훑을 수 있다.

## 3. Go e2e 로 옮겨진 레거시 테스트

아래 테스트는 노드 생명주기·동기화·합의 정지처럼 DSL 이 표현하기 어려운 것이라 `tests/e2e/` 의 Go 테스트가 맡는다. 그래서 tc/ 에는 그 번호가 비어 있다.

| 레거시 | Go 테스트 |
|---|---|
| `post-v1.0.0-change/stand-alone/01-test-signature-compat-sync` | e2e TestE2E_StablenetHardforkSwap |
| `regression/ethereum/02-test-full-sync` | e2e TestE2E_StablenetSyncGap |
| `regression/ethereum/03-test-snap-sync` | e2e TestE2E_StablenetSyncGap / TestE2E_WbftSnapSync |
| `regression/ethereum/04-test-node-restart` | cases/fault-node-recover + e2e ConsensusLifecycle |
| `regression/ethereum/06-test-downloader-path` | e2e TestE2E_StablenetSyncGap |
| `regression/ethereum/07-test-block-fetcher-path` | e2e TestE2E_StablenetBlockPropagation |
| `regression/system-contracts/11-test-proposal-expiry` | specs/system-contracts/proposal-expiry-transitions + e2e ProposalExpiry |
| `regression/wbft/08-test-quorum-deficient` | e2e TestE2E_WbftFaultHalt / WbftQuorum* |
| `regression/wbft/09-test-round-change` | e2e TestE2E_WbftViewChange |
| `regression/wbft/10-test-post-round-change` | e2e TestE2E_WbftRoundRobinProposer |

## 4. 레거시 대비 커버리지

감사에서 나온 **누락 18건은 전부 옮겼다.** 부분 포팅으로 남아 있던 것 중
검증 범위가 눈에 띄게 좁았던 셋(effectivegasprice 3건의 노드 간 일치,
wbft add-validator 의 에폭 경계)도 채웠다.

남은 것은 사설망에서 값을 고정할 수 없어 등호를 부등호로 낮춘 항목들과,
fee-delegation 4건의 `personal_*` API 경로다. 후자는 로컬 서명으로 대체한 것이
의도인지 먼저 정해야 한다. 자세한 목록은
`../../docs/dev/legacy-port-audit/03-port-audit.md` 3.3 절에 있다.

### 4.1 이번에 추가한 스펙

| 새 스펙 | 레거시 | 원본과 달라진 점 |
|---|---|---|
| `regression/blacklist-authorized/05-zero-address-transfer-rejected` | RT-E-05 | 거부 사유를 `zero` 로 잡는다. 원본은 `zero address` 와 `ZeroAddress` 두 철자를 받았다 |
| `regression/blacklist-authorized/06-precompile-transfer-rejected` | RT-E-06 | 같음 (5개 주소 모두 제출 거부) |
| `regression/wbft/05-validator-remove-member-executes` | RT-B-05 | 자기 완결로 바꿨다. 원본은 04 가 먼저 추가해 둔 멤버를 지웠지만, 여기서는 추가와 제거를 한 스펙 안에서 한다 |
| `post-v1.0.0-change/common-all/05-burn-expire-refundable` | TC-1-1-03 | 원본 9개 단언 중 6개를 옮겼다. GovMinter 잔액 증가, Expired(5), refundableBalance 증가분, BurnDepositRefunded 이벤트는 그대로다. 빠진 것은 소각 계정 잔액 감소, `burnBalance == 0`, 환불 청구(claimBurnRefund) 후 출금 확인 세 가지다. `short-expiry` 를 요구한다 |
| `post-v1.0.0-change/common-all/19-upgrade-registry-order` | TC-5-2-01/02/03 | 같음 (block 0 / 0x63 / 0x64 코드 비교) |
| `post-v1.0.0-change/common-all/20-v1-params-init-storage` | TC-5-2-04 | gasTip 슬롯을 상수와 등호로 비교하지 않고 0이 아님으로 본다. 사설망마다 gasTip 이 다르다 |
| `post-v1.0.0-change/extra-state/03-extra-union-merge` | TC-4-5-05/06 | genesisOverlay 로 alloc.Extra 와 GovCouncil params 를 함께 넣어 자기 완결로 만들었다. 계정은 원본과 다르지만 합집합·중복 제거라는 명제는 같다 |
| `post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent` | TC-5-3-01 | 릴리스 고정 해시 대신 노드 간 일치와 `parentHash == 0` 으로 본다. 사설망은 env 마다 genesis 가 달라 고정값을 쓸 수 없다 |
| `post-v1.0.0-change/string-handling/01~06` | TC-4-3-01~06 | 입력 문자열을 genesisOverlay 로 넣는다. 원본이 시나리오마다 전용 바이너리를 바꿔 끼우던 자리다. `01` 의 입력은 레거시 원본대로 `0xaaa,0xbbb,0xccc` 를 쓴다 |

| `post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost` | TC-4-2-01/03 | 위임 대상을 고정 주소 대신 새로 만든 계정으로 잡는다. 경계값은 preActions 에서 이름을 붙여 한 번씩만 적는다 |
| `post-v1.0.0-change/stand-alone/02-genesis-mismatch` | TC-4-1-03 | 원본은 SSH 로 바이너리를 직접 실행해 stderr 를 봤다. 여기서는 `swapNode` 에 `expect: fail` 을 걸고 노드가 RPC 에 응답하지 않는 것과 그 이유가 genesis 임을 확인한다. mismatch 바이너리가 사전 조건이다 (env 의 `GSTABLE_MISMATCH_BIN`) |
| `post-v1.0.0-change/stand-alone/03-unsupported-version` | TC-5-2-06 | 원본은 BohoBlock 커밋 실패를 로그로 봤다. 여기서는 genesis overlay 로 지원하지 않는 버전을 선언하고, BohoBlock 직전까지 생산한 뒤 `blockStalled` 로 멈춤을 확인하며 노드 로그를 아티팩트에 남긴다 |

| `post-v1.0.0-change/extra-state/06-invalid-extra-reject` | TC-4-5-09 | 원본은 미정의 Extra 비트가 박힌 바이너리로 노드를 띄웠다. 여기서는 `swapNode` 의 `genesisOverlay` 로 EN 한 대에만 그 genesis 를 주고 `expect:"fail"` 로 부팅 거부를 확인한다. 원본의 cleanup 단계는 옮기지 않았다 — compose 는 실행마다 새 워크스페이스를 쓴다 |
| `post-v1.0.0-change/effectivegasprice/01·02b·03` | TC-4-6-01/02/04 | 원본이 보던 **BP 와 snap-sync EN 의 값 일치**를 되살렸다. 기존 단일 노드 스펙은 그대로 두고, snap 엔드포인트가 있는 env 위에서 두 노드를 비교하는 케이스를 더했다. 03 은 AuthorizedTxExecuted 가 로그의 *마지막*인지도 두 노드에서 확인한다 |
| `regression/wbft/04b-validator-add-member-epoch-activates` | RT-B-04 | 원본이 보던 **에폭 경계에서 검증자 집합이 커지는지**를 되살렸다. 승격 대상은 살아 있는 EN 노드 자신(`istanbul_nodeAddress`)이라 실제로 합의에 들어간다. 에폭 길이는 genesis overlay 로 10블록으로 줄였다 |

## 5. 이번에 추가한 DSL 문법

옮기지 못하던 테스트를 표현하려고 세 가지를 더했다.

| 문법 | 뜻 | 구현 |
|---|---|---|
| `{"do":"signAuthorization","authorityKey":…,"delegate":…,"save":…}` | EIP-7702 인증 튜플에 서명만 하고 보내지 않는다. `eth_estimateGas` 의 `authorizationList` 에 넣을 수 있다 | `internal/accounts.Wallet.SignAuthorization`, `internal/testhelper/txprobe.go` |
| `{"do":"readNodeLog","on":…,"maxBytes":…,"save":…}` | 특정 노드가 남긴 stdout/stderr 의 끝부분을 읽는다 | `interp.NodeLogReader`, `testengine.workspaceNodes.Log`, `internal/testhelper/fault.go` |
| `{"do":"swapNode"…,"expect":"fail","reason":…}` | 이 기동은 실패해야 한다. RPC 가 응답하지 않는 것을 확인하고, 런처 에러와 노드 로그를 합쳐 `reason` 을 맞춘다 | `internal/testhelper/fault.go` |
| `{"expect":"blockStalled","on":…,"timeout":…}` | `blockAdvance` 의 반대. 창 내내 head 가 움직이지 않아야 통과한다 | `internal/testhelper/read.go` |
| `{"do":"swapNode"…,"genesisOverlay":{…}}` | 네트워크 genesis 에 조각을 덮어 **그 노드만** 다시 init 한다. 한 대에만 다른 genesis 를 줄 수 있다 | `chainsetup.SwapNodeOpts`, `interp.NodeChange.GenesisOverlay` |

`expect: "fail"` 은 `sendTx` 의 `expect: "reject"` 와 같은 모양이다. 실패를 기대하는
표현이 트랜잭션과 노드 기동 두 곳에서 같게 읽힌다.

## 6. env 참조 규칙

케이스는 `"chainPreset": "<id>"` 로 환경을 부른다. `internal/dsl.ReadFiles` 가 케이스 파일이 있는 디렉터리부터 위로 올라가며 `<id>.json` 과 `chain-preset/<id>.json` 을 찾는다. 그래서 `presets/chain/` 하나로 모든 깊이의 케이스가 같은 환경 선언을 공유한다.

## 7. 실행 전에 있어야 하는 바이너리

케이스 대부분은 `--binary` 로 준 하나면 돈다. **일곱 개 chain-preset 은 환경변수로
바이너리를 더 받는다.** 안 걸어두면 `${VAR:-이름}` 의 기본값인 맨 이름으로 떨어지고, 그 이름은
PATH 에 없으므로 실행이 이렇게 멈춘다.

```
exec: "gstable": executable file not found in $PATH
```

| env | 변수 | 무엇을 가리키나 |
|---|---|---|
| `wbft-bp4-binvar` | `GWBFT_BIN` | go-wbft 빌드 (make 가 `gwemix` 로 만든다) |
| `wemix-bp4-binvar` | `GWEMIX_BIN` | go-wemix 빌드 |
| `wemix-to-wbft`, `wemix-to-wbft-bp2` | `GWEMIX_BIN`, `GWBFT_BIN` | 넘겨주는 쪽과 넘겨받는 쪽 |
| `stablenet-bp4-en1-default-upgrade` | `GSTABLE_UPGRADE_BIN` | **default 와 다른** go-stablenet 빌드 |
| `stablenet-bp4-default-mismatch` | `GSTABLE_MISMATCH_BIN` | genesis 가 어긋나 기동을 거부해야 하는 빌드 |
| `stablenet-restart-at-boho` | `GSTABLE_BIN`, `GSTABLE_POSTFORK_BIN` | boho 도입 **양쪽**의 go-stablenet 빌드 |

### go-stablenet 두 빌드 만들기

뒤의 둘은 go-stablenet 을 **두 커밋에서** 빌드해야 한다. `ad0122af0` 은 boho 를 넣은
커밋의 부모라 `BohoBlock` 자체가 없고, 그 뒤 아무 커밋이나가 상대다.

```sh
cd <go-stablenet>
git worktree add /tmp/gs-prefork ad0122af0
cd /tmp/gs-prefork && make gstable          # boho 를 모르는 빌드

export GSTABLE_BIN=/tmp/gs-prefork/build/bin/gstable
export GSTABLE_POSTFORK_BIN=<go-stablenet>/build/bin/gstable
export GSTABLE_UPGRADE_BIN=$GSTABLE_BIN
```

두 빌드가 실제로 다른지는 이렇게 본다 — 앞은 0, 뒤는 0이 아니어야 한다.

```sh
strings $GSTABLE_BIN         | grep -ci bohoblock
strings $GSTABLE_POSTFORK_BIN | grep -ci bohoblock
```

`GSTABLE_UPGRADE_BIN` 을 안 걸면 `01b-signature-compat-across-swap` 은 **같은 빌드로**
스왑한다. 실행은 되지만 그 케이스가 검증한다는 것("노드가 **다른** 바이너리로 재기동해도
tx 가 보존된다")을 확인하지 못한다.

## 8. 함께 있는 문서

- `RUN-EACH.md` — 케이스 209건을 하나씩 돌리는 명령과 그 준비
- `SPECS.md` — 스펙 이관 기록 (레거시 시절 `tests/specs/README.md`)
- `CHAIN-BRINGUP.md` — 체인 구성 케이스 설명 (레거시 시절 `tests/cases/README.md`)
- `../../docs/dev/legacy-port-audit/` — 포팅 감사 (그래프 2종 + 대응표)
