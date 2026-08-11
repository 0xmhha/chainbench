# 체인 바이너리 CLI 그래프 · 실행옵션 모듈 설계 검토

> 지시 5 응답. 작성: 2026-08-11 · 기준: `go-stablenet@0937ac5c9` · `go-wbft@f6515366b` · `go-wemix@1350376a6`
> 근거 데이터는 AST 추출기 [`scripts/inventory/chain-flag-graph`](../../scripts/inventory/chain-flag-graph/main.go) 로 재생성 가능:
> ```sh
> go run ./scripts/inventory/chain-flag-graph <chain-repo-root> <binary-dir> > graph.json
> ```
> 추출기는 `cmd/utils` · `cmd/<bin>` · `internal/debug` 를 `go/ast` 로 파싱한다(타입체크·빌드 불필요).
> 추출 대상: 플래그 선언(`XFlag = &cli.*Flag{Name,Usage,Category,Value}`) · 플래그 그룹(`nodeFlags`/`rpcFlags`/…) ·
> 커맨드 트리(`cli.Command{Name,Flags,Subcommands}`) · `app.Commands`.

---

## 1. 추출 결과 요약 (AST 그래프)

| 바이너리 | repo | 플래그 | 커맨드 | 플래그 그룹 |
|---|---|---:|---:|---|
| `gstable` | go-stablenet | **179** | 35 | nodeFlags 93 · rpcFlags 29 · metricsFlags 14 · debug.Flags 18 · DatabaseFlags 6 · consoleFlags 3 · Testnet 4 · Deprecated 14 |
| `gwemix` (wbft) | go-wbft | **177** | 34 | 위와 동일 수치 |
| `gwemix` (poa) | go-wemix | **183** | 36 | rpcFlags 26 · **wemixFlags 19** · metricsFlags 14 · debug.Flags 12 · Testnet 6 · consoleFlags 2 |

### 1.1 결정적 관찰 — 표면은 3개가 아니라 **2개**다

```
gstable  ∩ gwbft  = 177  (대칭차 2개: --check.url, --check.version — version-check 커맨드 전용)
gstable ∩ gwbft ∩ gwemix = 134
gstable \ gwemix = 43        gwemix \ gstable = 47
```

- **go-stablenet 과 go-wbft 의 CLI 표면은 사실상 동일**(대칭차 2). 두 repo 는 같은 go-ethereum
  베이스(1.13/1.14 계열, `flags.*Category` 도입 후)에서 갈라졌고, 체인 차이는 CLI 가 아니라
  genesis·합의 코드에 있다.
- **go-wemix 만 다른 세대**다. 구 go-ethereum(1.10 계열: `Category` 필드 없음, `--ropsten`/`--rinkeby`,
  ethash 플래그 잔존)에 `wemixFlags` 19개가 얹혀 있다.

이것이 이번 검토의 가장 중요한 사실이다: **"체인마다 실행옵션 어댑터가 필요하다"는 전제가 틀렸다.**
필요한 것은 체인 3개가 아니라 **바이너리 세대(generation) 2개 + wemix 확장 1개**의 어댑터다.

### 1.2 go-wemix 전용 플래그 (`wemixFlags`, 19개)

| 플래그 | 의미 |
|---|---|
| `--consensusmethod` | 1=PoW · 2=PoA · 3=ETCD · 4=PBFT |
| `--fixeddifficulty` · `--fixedgaslimit` | PoW 무력화 · 블록 크기 고정 |
| `--blocksperturn` · `--maxidleblockinterval` | PoA 턴 길이 · 빈 블록 생성 간격 |
| `--noncelimit` · `--maxtxsperblock` · `--prefetchcount` | 비거버넌스 계정 nonce 상한 · 블록당 tx 상한 · prefetch |
| `--wemix.block.interval` · `.timeadjblocks` · `.minbuildtime` · `.minbuildtxs` · `.trailtime` | 블록 생성 타이밍 5종 |
| `--wemix.publicrequests.cache` · `.max` · `--wemix.bootnodecount` | 퍼블릭 요청 캐시·상한 · 부트노드 수 |
| `--userocksdb` · `--hub` · `--log` | LevelDB/RocksDB · 메시지 허브 id · 로테이팅 로그파일 |

`--consensusmethod` · `--wemix.block.*` 는 **테스트 파라미터로 직결**된다(블록 간격을 줄여 테스트를
빠르게, 합의 방식을 바꿔 케이스를 분기). 현재 chainbench 는 이 중 **어느 것도 노출하지 않는다.**

### 1.3 커맨드 트리 (3 바이너리 공통 핵심)

```
<bin> [global flags]                    # 기본 액션 = 노드 실행
├─ init <genesis.json>   --datadir      # chainbench 가 실제로 쓰는 유일한 서브커맨드
├─ account {new,list,import,update}     # keystore — chainbench 는 자체 구현으로 대체
├─ attach <ipc|url> --exec              # chainsetup/state.go:144 가 사용(admin.wemixInfo)
├─ dumpgenesis / dumpconfig             # ← 미사용. genesis/config 정합 검증에 쓸 수 있음
├─ db {inspect,stat,compact,get,put,…}  # 미사용
├─ snapshot / verkle / export / import  # 미사용
└─ wemix {…}                            # go-wemix 전용 (governance deploy 계열)
```

chainbench 가 현재 접촉하는 것은 `init`(`internal/core/driver/init.go:35`) · `attach`
(`internal/chainsetup/state.go:144`) · 기본 액션(노드 실행) **3개뿐**이다.

---

## 2. 현재 chainbench 의 실행 command 조립 방식 (실측)

노드 실행 인자는 **5곳에서 따로 조립**된다.

| # | 위치 | 무엇을 붙이나 |
|---|---|---|
| 1 | `internal/core/nodeconfig/nodeconfig.go:112` `LaunchArgs` | `--datadir --config --port --http --http.port --ws --ws.port` (하드코딩 7종) |
| 2 | `internal/consensus/wbft/wbft.go:38` `StartFlags(role)` | `--allow-insecure-unlock --rpc.enabledeprecatedpersonal --rpc.allow-unprotected-txs` (+role=bp면 `--mine`) |
| 3 | `internal/consensus/poa/poa.go:32` `StartFlags(role)` | `--allow-insecure-unlock` (+bp/boot면 `--mine`) |
| 4 | `internal/engine/launcher.go:194-201` `armSpecs` | `--nodekey`, (validator면) `--unlock --password --miner.etherbase` |
| 5 | `internal/chainsetup/handoff_driver.go:255-261` | 핸드오프 전용 하드코딩 `--nat none --http.api … --miner.etherbase --unlock --password` |

나머지(포트 외 RPC 설정·metrics·txpool·sync·cache)는 **인자가 아니라 TOML config**
(`nodeconfig.Generate`)로 나간다.

### 2.1 이 구조의 실제 결함 (설계 취향이 아니라 사실)

1. **동일 개념이 두 표면에 쪼개져 있다.** WS 포트는 `--ws.port`(인자)와 `[Node] WSPort`(TOML) 양쪽에
   나간다(`nodeconfig.go:69` 주석이 자인). 두 값이 어긋나면 어느 쪽이 이기는지 코드에 근거가 없다.
2. **테스트가 실행옵션을 못 만진다.** DSL spec 에 노드 플래그를 넣을 자리가 없다 →
   `--wemix.block.interval` 로 블록을 빠르게 만들거나 `--txpool.globalslots` 로 풀 포화를 만드는
   테스트는 **표현 자체가 불가능**하다. 배경 2 · 알고리즘 7 의 미충족 지점.
3. **`--metrics` 가 인자로 나가지 않는다.** TOML `[Metrics] Enabled=true` 만 쓰는데, 이는
   구 geth 에서 `--metrics` 없이는 무시된다. 배경 3 의 "metric 정보 활용"이 검증 소스로
   존재하지 않는 이유의 절반이 여기다.
4. **핸드오프 경로가 별도 하드코딩**(#5)이라 일반 경로의 개선이 전파되지 않는다.
5. **역할(role) 축 하나로만 분기**한다. 실제 필요한 축은 role × 바이너리 세대 × 배치(local/remote) ×
   테스트 오버라이드 4개다.

---

## 3. 제안 검토 — "옵션 모듈 + builder 조립" 설계

사용자 제안: *설정 옵션을 별도 모듈로 나누고, builder 가 있으면 적용/없으면 넘어가는 방식.*

### 3.1 비판 — 순진하게 구현하면 실패하는 3가지

**(a) "있으면 적용, 없으면 스킵"은 조합 오류를 조용히 통과시킨다.**
`--mine` 없이 `--miner.etherbase` 만, `--http.api` 는 켜고 `--http` 는 안 켠, `--unlock` 은 있고
`--password`/`--allow-insecure-unlock` 은 없는 조합은 전부 "빌드 성공 → 런타임 실패"다. geth 는
이 중 일부만 오류를 내고 나머지는 조용히 무시한다. **옵션 모듈은 방출(emit)뿐 아니라 검증(validate)
책임을 함께 져야 하고, builder 는 전체 조립 후 cross-module 불변식을 한 번 더 검사해야 한다.**

**(b) 플래그 1개 = 모듈 1개는 과분해다.** 179개 플래그에 179개 타입을 만들면
component-architecture §0-2-2 가 경고한 "과분해로 추상화 비용↑"에 정확히 걸린다.
분해 단위는 플래그가 아니라 **관심사(concern)** 여야 한다 — 아래 §3.3의 10개.

**(c) 세대 차이를 모듈 내부 if 로 흡수하면 ACL 이 무너진다.**
`--syncmode`(gstable) ↔ 없음(gwemix), `--miner.recommit` 문자열 ↔ 나노초 정수 같은 차이를
각 옵션 모듈이 `if chain == "wemix"` 로 처리하기 시작하면 체인 지식이 core 전역에 번진다
(C6 ACL 불변식 위반). **세대 차이는 모듈이 아니라 `Dialect`(플래그 사전) 한 곳이 소유**해야 한다.

### 3.2 동의 — 이 설계가 옳은 3가지 이유

1. **§1.1 의 데이터가 뒷받침한다.** 표면이 2세대뿐이므로 dialect 테이블 2장이면 전부 덮는다.
   체인이 늘어도(4번째 체인) 대부분 기존 dialect 재사용이다.
2. **현재의 5-군데 분산(§2)이 이미 비용을 내고 있다.** 조립 지점을 1개로 모으는 것은 새 추상화
   도입이 아니라 **중복 제거**다 — 추상화는 값을 낼 때만 도입하라는 원칙에 부합한다.
3. **precedence 를 표현 가능하게 만든다.** 지금은 "테스트가 값을 덮어쓴다"를 표현할 데이터 구조가
   없다. 옵션을 값 객체로 만들면 `default < family < role < env.launch < case override` 를
   결정적으로 병합할 수 있다.

### 3.3 제안 설계 (수정판)

**레고 블록의 단위 = 플래그가 아니라 "관심사" 10개.** 각각은 순수함수 `Apply(*Args)` 이고,
자기 불변식만 검사한다.

```go
package launchopt   // internal/core/launchopt

// Dialect is the flag vocabulary of one binary generation. It is the ONLY place
// that knows a binary's spelling, so option modules stay chain-agnostic.
type Dialect struct {
    ID       string              // "geth114" | "geth110-wemix"
    Flag     map[Key]string      // Key -> actual flag name ("" = unsupported)
    Bool     map[Key]bool        // flag takes no value
    Recommit RecommitForm        // duration | nanos
}

// Key is the chain-agnostic name of a knob. Typed const, never a magic string.
type Key string
const (
    KeyDataDir   Key = "datadir"
    KeyHTTPPort  Key = "http.port"
    KeyMine      Key = "mine"
    KeyBlockInterval Key = "block.interval"   // gstable: 미지원("") / gwemix: --wemix.block.interval
    // ...
)

// Args is the accumulating command line: subcommand + ordered flags.
type Args struct {
    Subcommand string
    kv         []pair            // 순서 결정적(정렬 아님 — 삽입 순서 보존)
    problems   []error
}
func (a *Args) Set(k Key, v string)   // Dialect 가 미지원이면 problems 에 기록
func (a *Args) Enable(k Key)
func (a *Args) Has(k Key) bool

// Module is one concern. Pure: no I/O, no globals.
type Module interface {
    Name() string
    Apply(d Dialect, a *Args) error
}
```

10개 모듈:

| 모듈 | 소유하는 플래그 | 자기 불변식 |
|---|---|---|
| `Identity` | `--nodekey --unlock --password --miner.etherbase --allow-insecure-unlock --keystore` | unlock 이 있으면 password + allow-insecure-unlock 필수 |
| `Storage` | `--datadir --datadir.ancient --config --gcmode --syncmode --state.scheme --db.engine` | datadir 필수(절대경로) |
| `P2P` | `--port --bootnodes --nodiscover --maxpeers --nat --netrestrict --discovery.port` | port>0, static-node 사용 시 nodiscover |
| `HTTPRPC` | `--http --http.addr --http.port --http.api --http.vhosts --http.corsdomain` | api 지정 시 http 필수 |
| `WSRPC` | `--ws --ws.addr --ws.port --ws.api --ws.origins` | 동일 |
| `AuthIPC` | `--authrpc.* --ipcdisable --ipcpath` | ipcpath 길이 < 104(유닉스 소켓 한계, T4.4e 에서 실제로 물린 제약) |
| `RPCPolicy` | `--rpc.gascap --rpc.txfeecap --rpc.allow-unprotected-txs --rpc.enabledeprecatedpersonal` | — |
| `Mining` | `--mine --miner.gaslimit --miner.gasprice --miner.extradata --miner.recommit` | etherbase 없이 mine 금지 |
| `Metrics` | `--metrics --metrics.addr --metrics.port --metrics.expensive` | port 지정 시 `--metrics` 필수 ← §2.1-3 결함 해소 |
| `ChainExt` | `--consensusmethod --wemix.block.* --blocksperturn --noncelimit --maxtxsperblock` | dialect 미지원이면 **명시적 오류**(조용한 스킵 금지) |

빌더:

```go
// Builder assembles modules into a command line in a fixed order and then runs
// the cross-module checks the individual modules cannot see.
type Builder struct {
    dialect Dialect
    mods    []Module          // 순서 = 위 표 순서(결정적 출력)
    over    []Override        // 우선순위가 높은 마지막 층
}
func (b *Builder) Build() ([]string, error)   // problems 가 있으면 전부 모아 errors.Join
```

**핵심 수정 3가지 (§3.1 비판에 대한 응답):**

1. **"없으면 스킵"이 아니라 "없으면 분류된 결과"**다. 미지원 플래그는 세 갈래로 나뉜다 —
   ① *무해한 부재*(dialect 가 기본으로 켜는 것) → 스킵, ② *요청된 기능의 부재* → **오류**,
   ③ *deprecated 대체* → 대체 플래그로 매핑 + 경고. 지금 supervisor 가 `LeaderGate` 미배선을
   조용한 통과가 아니라 오류로 처리하기로 한 결정(T3.2b)과 같은 규율이다.
2. **cross-module 검증은 builder 소유**다. 모듈은 자기 것만 보므로 "mine 은 있는데 etherbase 는
   Identity 모듈 소관" 같은 조합은 모듈이 못 잡는다.
3. **precedence 는 층으로 표현**한다: `family default → role → env.launch → case override`.
   같은 Key 의 마지막 층이 이긴다. 어느 층이 이겼는지 `Args` 에 기록해 `net status` 로 노출한다
   (config 의 flag>file>default 규약과 동형).

### 3.4 config(TOML) 와의 경계 — 반드시 먼저 정할 것

§2.1-1 의 이중 표면을 방치한 채 builder 만 도입하면 결함이 하나 늘어난다. 규칙:

> **프로세스 아이덴티티·엔드포인트·기동 모드는 플래그, 튜닝 파라미터는 TOML.**
> 같은 값을 양쪽에 쓰지 않는다. 겹치면 플래그가 정본이고 TOML 에서 제거한다.

구체적으로 `WSPort`/`HTTPPort`/`ListenAddr` 는 TOML 에서 **삭제**하고 플래그만 남긴다
(`nodeconfig.Generate` 축소). `[Eth.Miner] Recommit` 의 duration/nanos 세대 차이는 `Dialect.Recommit`
로 옮겨 `--miner.recommit` 플래그로 통일 가능한지 실측 확인 후 결정한다.

---

## 4. 적용 여부 판정

**적용 권고 — 단, 범위를 좁혀서.**

| | 판정 | 근거 |
|---|---|---|
| 옵션을 관심사 10개 모듈로 분해 | **채택** | 현재 5곳 분산의 중복 제거이지 신규 추상화가 아님 |
| Dialect 로 세대차 흡수(2장) | **채택** | §1.1 — 표면이 2세대뿐이라 테이블 2장으로 끝남 |
| builder 조립 + cross-module 검증 | **채택** | 조합 오류의 유일한 포착 지점 |
| "있으면 적용/없으면 스킵" | **수정 후 채택** | 3분류(스킵/오류/대체매핑)로 대체 — 조용한 스킵 금지 |
| 플래그 1개 = 모듈 1개 | **기각** | 과분해(§3.1-b) |
| 179개 플래그 전량 노출 | **기각** | 실사용 근거 없는 표면. §3.3 10모듈이 덮는 ~60개로 시작하고 요청 시 확장 |
| 커맨드 트리 전체 래핑 | **기각** | 실사용은 `init`·`attach`·기본액션 3개(§1.3). `dumpgenesis`/`dumpconfig` 는 검증용으로 **추가 권고** |

### 4.1 도입 순서 (기존 구조를 깨지 않는 경로)

1. `internal/core/launchopt` 신설 — Dialect 2장 + 모듈 10개 + Builder. **순수함수라 전부 단위 TDD.**
   출력이 현재 `LaunchArgs+StartFlags+armSpecs` 조합과 **바이트 동일**함을 골든 테스트로 고정.
2. `engine/launcher.go armSpecs` 를 Builder 호출로 교체(동작 변화 0 — 골든 테스트가 게이트).
3. `nodeconfig.Generate` 에서 중복 엔드포인트 항목 제거(§3.4).
4. `chainsetup/handoff_driver.go` 하드코딩(#5)을 Builder 로 흡수 → 5곳 → 1곳.
5. DSL `env.launch` / `case.override` 를 Builder 의 상위 2개 층에 배선 → 배경 2 · 알고리즘 7 충족.
6. `ChainExt` 모듈로 wemix 전용 노브 노출 → `--wemix.block.interval` 을 줄인 고속 테스트 환경 가능.

1~2 단계까지가 최소 유의미 단위이며, 그 자체로 "동일 개념이 5곳에 흩어짐" 결함을 해소한다.

### 4.2 이 설계가 무엇을 해결하지 *못하는가* (정직한 한계)

- **genesis 정합 문제는 손대지 않는다.** wbft 계열 ExtraData 가 preset 에 baked 되어 있다는 제약
  (T4.4b)은 실행옵션 계층 밖이다.
- **AST 추출은 정적이다.** 플래그 이름은 신뢰할 수 있지만 **동작**(어떤 조합이 유효한지)은
  실행해봐야 안다. `dumpconfig` 로 실효 설정을 덤프해 Builder 출력과 대조하는 회귀 테스트를
  4.1-1 단계에 함께 넣을 것을 권고한다.
- **`debug.Flags`(`--verbosity` 등 18/12개)는 `cmd/` 밖**(`internal/debug`)에 있어 추출기가 별도
  스캔한다. 다른 repo 구조로 바뀌면 추출기 `dirs` 를 갱신해야 한다.
