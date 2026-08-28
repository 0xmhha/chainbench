# CLI 기준 체인 구동 절차 — 3체인 대조

> `chainbench` 명령만으로 세 체인을 띄우려면 어떤 step 이 필요한가.
> 각 스텝에 **현재 CLI 명령이 있는지** 표시한다 — 없는 것이 곧 만들 것이다.
>
> 실측: 2026-08-18 라이브(stablenet·wbft 는 `net up` 성공 / wemix 는 수동 절차로 성공).
> 케이스별 심층 분석은 [[case-1-wemix]](case-1-wemix.md) 등, 작업 순서는 [[chainbench-worklist]](../chainbench-worklist.md) §1g.

---

## 0. 먼저 — 기동 표면이 3개다

같은 일을 하는 CLI 가 셋 있고, 각각 할 수 있는 범위가 다르다.

| 표면 | 명령 | 대상 | governance-etcd |
|---|---|---|---|
| **net** | `net up` / `net <step>` | 로컬 + 원격(TargetSpec) | ❌ **없음** |
| **chain** | `chain up` (케이스 기반) | 로컬 | ❌ wemix 는 `unsupported` |
| **remote** | `remote deploy` + `remote bootstrap` | 원격 SSH 클러스터 전용 | ✅ **있음** |

**`remote bootstrap` 만이 거버넌스+etcd 를 안다.** 즉 wemix 를 CLI 로 띄우는 길은 오늘도 존재하지만,
**SSH 클러스터 경로에만** 있다. 로컬에서는 불가능하다.

이 문서의 목표는 `net` 하나로 세 체인을 모두 덮는 것이다.

---

## 1. 공통 9스텝 (`net`)

세 체인이 같은 순서를 탄다. `net up` 은 이 9개를 한 번에 실행한다.

| # | 스텝 | 명령 | 하는 일 |
|---|---|---|---|
| 1 | new | `net new --chain X --binary B --keys K --server S` | 워크스페이스 초기화: 체인·키셋·타깃 기록 |
| 2 | allocate | `net allocate --validators N [--topology T]` | 노드표: 역할·경로·포트 (인벤토리에서) |
| 3 | keys | `net keys [--keys-source generate --bootnode B]` | 키셋 확보 |
| 4 | genesis | `net genesis [--chain-id C --set K=V --overlay O]` | **★ 패밀리 분기 ★** |
| 5 | config | `net config` | 노드별 TOML 렌더 |
| 6 | launchopts | `net launchopts [--set K=V]` | argv 조립 (실행 안 함) |
| 7 | provision | `net provision` | 타깃에 입력 존재 확인(skip-if-exists) |
| 8 | init | `net init` | `<binary> init` 으로 datadir 초기화 |
| 9 | start | `net start` | **★ 패밀리 분기 ★** 노드 기동 |

확인: `net status` (스텝 진행) · `net health` (블록 전진) · `net logs --node i` · `net stop` · `net rm`

---

## 2. go-stablenet — 라이브 성공 ✅

```sh
CHAIN=/Users/…/Work/github/chain
chainbench net up --workspace-dir /tmp/cbs --chain stablenet \
  --binary $CHAIN/go-stablenet/build/bin/gstable \
  --keys keys/preset --validators 4 --server local

chainbench net health --data-dir /tmp/cbs
chainbench run --chain stablenet --rpc http://127.0.0.1:8545 tests/specs/api/*.json
chainbench net stop --workspace-dir /tmp/cbs
```

9스텝 전부 CLI 존재. 결과: 블록 97→110→122, 4노드 동기, api 9 pass.

---

## 3. go-wbft — 라이브 성공 ✅

stablenet 과 **명령이 완전히 동일**하다. 두 가지만 다르다.

```sh
chainbench net up --workspace-dir /tmp/cbw --chain wbft \
  --binary $CHAIN/go-wbft/build/bin/gwemix \   # ← 바이너리 이름이 gwemix 다
  --keys keys/preset --validators 4 --server local
```

| 함정 | 내용 |
|---|---|
| 바이너리 이름 | go-wbft 의 make 산출물이 **`gwemix`** 다. `--binary` 를 명시 경로로 줘야 한다 |
| chain id | 8284 (stablenet 8283) |

결과: 블록 24→36→48, 4노드 동기, api 9 pass, consensus 3 pass.

---

## 4. go-wemix — `net` 으로는 실패 ❌

`net up` 은 9스텝을 다 통과하고 노드도 뜨지만 **블록이 0에서 멈춘다.**

```
ERROR Unavailable modules in HTTP API list  unavailable=[wemix]
INFO  Disk storage enabled for ethash DAGs        ← wemix 가 아니라 ethash 로 떴다
```

genesis 가 껍데기이기 때문이다 — `alloc:{}`, `minerNodeId:"0x0"`, `coinbase:0x0…0`.

### 4.1 실제로 필요한 절차 (수동으로 검증 ✅)

`net` 의 9스텝에 **더해** 아래가 필요하다. 굵게 표시한 것이 **CLI 에 없는 것**이다.

| # | 스텝 | 명령 (오늘) | 상태 |
|---|---|---|---|
| W1 | 노드 신원 확보 | `net keys` (preset 사용) | ✅ **그대로 됨** — preset 의 `publicKey`(128hex)가 곧 idv5 |
| W2 | **`wemix-config.json` 조립** | 없음 | ❌ **CLI 없음** (`poa.Config` 타입은 있음) |
| W3 | **genesis 를 바이너리로 생성** | 없음 (`net genesis` 는 템플릿 치환) | ❌ **CLI 없음** |
| W4 | datadir init | `net init` | ✅ |
| W5 | **부트노드 1대만 기동** | 없음 (`net start` 는 전부 기동) | ❌ **CLI 없음** |
| W6 | **거버넌스 컨트랙트 배포** | `remote bootstrap`(SSH 전용) | ◐ **로컬 없음** |
| W7 | **etcd 초기화** | `remote bootstrap`(SSH 전용) | ◐ **로컬 없음** |
| W8 | **etcd 클러스터 확인** | 없음 | ❌ (`chainsetup.WaitEtcdCluster` 는 있음) |
| W9 | **나머지 노드 기동** | 없음 (페이즈 구분이 없음) | ❌ **CLI 없음** |
| W10 | 블록 전진 확인 | `net health` | ✅ |

**갭은 6개다** — W2·W3·W5·W8·W9 는 CLI 자체가 없고, W6·W7 은 SSH 경로에만 있다.

### 4.2 수동으로 검증한 실제 명령 (2026-08-18)

```sh
G=$CHAIN/go-wemix/build/bin/gwemix
D=/tmp/wm                    # 짧게: IPC 소켓 104자 제한

# W1  신원 (preset 을 쓰면 생략 가능 — 아래는 신규 생성 경로)
$G wemix new-account --password $D/password --out $D/nodeN/keystore/accountN
$G wemix new-nodekey --out $D/nodeN/geth/nodekey
$G wemix nodeid $D/nodeN/geth/nodekey        # → idv5 (128 hex)

# W2  wemix-config.json  (members[]: addr·stake·name·id·ip·port·bootnode / accounts[] / env{})

# W3  genesis — 치환이 아니라 바이너리가 생성한다
$G wemix genesis --data $D/config.json --genesis <템플릿> --out $D/genesis.json

# W4  init
for i in 1 2 3 4; do $G --datadir $D/node$i init $D/genesis.json; done

# W5  부트노드 1대만
$G --datadir $D/node1 --nodiscover --http --http.port 8588 --ws --ws.port 8598 \
   --syncmode full --gcmode archive --port 8589 &
#     포트 규칙: http=PORT · p2p=PORT+1 · ws=PORT+10 · etcd=p2p+1, p2p+2

# W6  거버넌스
$G wemix deploy-governance --url $D/node1/gwemix.ipc --password $D/password \
   $D/config.json $D/node1/keystore/account1 1500000000000000000000000

# W7  etcd
$G attach $D/node1/gwemix.ipc --exec 'admin.etcdInit()'

# W8  확인 (반환값이 null 이라 이 조회가 유일한 판정 근거)
$G attach $D/node1/gwemix.ipc --exec 'JSON.stringify(admin.wemixInfo.etcd)'

# W9  나머지 노드
for i in 2 3 4; do P=$((8588+(i-1)*1000)); $G --datadir $D/node$i --mine \
   --http --http.port $P --ws --ws.port $((P+10)) --port $((P+1)) \
   --syncmode full --gcmode archive & done
```

결과: 거버넌스 5컨트랙트 → etcd 4멤버 전부 up → 블록 238→253→268,
**4노드 sealing 로테이션(20블록 중 5/6/5/4)**.

### 4.3 기존 문서를 정정하는 실측 3건

[[case-1-wemix]] §5 의 절차는 케이스 2(핸드오프)에서 **도출**한 것이라 단독 기동에는 불필요한 단계가 있다.
오늘 실측으로 다음이 확인됐다.

| 문서의 절차 | 실측 | 결론 |
|---|---|---|
| A6: 부트스트랩 후 **프로듀서 종료** → B1 전 노드 재기동 | 종료하지 않고 그대로 두고 나머지만 띄워도 **정상 동작** | 단독 기동에는 재기동 불필요 |
| B2: **풀메시 `admin_addPeer`** 필요 | 하지 않았는데 4노드 전부 `up`·sealing | 거버넌스 member 목록(ip/port)으로 자동 연결됨 |
| `--http.api` 미기재 | 생략하면 `eth,net,rpc,web3` 만 열려 **txpool 스펙 2건 실패** | `--http.api eth,net,web3,wemix,admin,miner,txpool,personal` 필요 |

세 번째는 chainbench 의 `nodeconfig` 가 이미 HTTPModules 를 채우므로 CLI 경로에서는 자동 해소된다.

---

## 5. 세 체인 비교 요약

| | stablenet | wbft | wemix |
|---|---|---|---|
| family | wbft | wbft | **poa** |
| 바이너리 | `gstable` | **`gwemix`**(이름 주의) | `gwemix` |
| chain id | 8283 | 8284 | 8285 |
| genesis | 템플릿 + extraData RLP 치환 | 동일 | **바이너리가 생성** |
| 기동 | 전 노드 동시 | 동일 | **2페이즈 + 액션 2개** |
| 포트 대역 | p2p+1(etcd 예약) | 동일 | **p2p+1, p2p+2 둘** |
| `net up` | ✅ | ✅ | ❌ |
| CLI 스텝 수 | 9 | 9 | **9 + 6 (미구현)** |

---

## 6. 그래서 CLI 에 무엇을 더해야 하는가

§4.1 의 갭 6개는 **새 명령 6개가 아니라, 기존 스텝 2개의 확장**으로 덮인다.

| 갭 | 어디에 흡수되나 | 트래커 |
|---|---|---|
| W2 config 조립 · W3 genesis 생성 | **`net genesis`** — 패밀리가 다르면 다르게 만든다(`GenesisArtifacts`) | F4 |
| W5 부트노드 단독 · W9 나머지 · W6·W7 액션 · W8 확인 | **`net start`** — 페이즈를 따라 기동하고 사이에 액션을 실행 | F3·F5 |

즉 **CLI 표면은 지금 그대로 두고**(`net up` 한 줄로 세 체인 전부),
바뀌는 것은 그 아래 `net genesis` 와 `net start` 의 내부다.

```sh
# 목표 — 세 체인 모두 이 한 줄
chainbench net up --workspace-dir /tmp/n1 --chain {stablenet|wbft|wemix} \
  --binary <path> --keys keys/preset --validators 4 --server local
```

보조로 필요한 것은 **관측**뿐이다: `net status` 가 어느 페이즈까지 갔는지,
어떤 액션이 실행됐는지 보여주면 부분 실패를 손으로 이어갈 수 있다(이미 스텝 스탬프 구조가 있다).
