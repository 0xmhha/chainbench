# 하드포크 테스트를 보통 경로로 — 설계

**목적은 줄 수가 아니다.** 핸드오프가 다른 망과 **같은 규칙으로 도는 것**이 목적이고,
`internal/consensus/upgrade` 의 비테스트 1,959줄이 줄어드는 것은 그 결과다. 일부만
흡수하고 나머지를 남기는 결론도 정당하다.

**읽은 것과 돌려 본 것을 갈라 적는다.** 이 트랙에서 코드만 읽고 내린 판정이
2026-09-17 하루에 세 번 틀렸다(§11.2.5, §11.2.8-1, §11.2.9). 각 문장에 어느
쪽인지 표시한다.

---

## 1. 지금 막고 있는 것

워크리스트가 흡수를 막는다고 적었던 넷은 **전부 정리됐다**(§11.2.9~12). 셋은
프로필이 **선택이 아니라 사실을 다시 적고** 있었고, 하나는 내가 거꾸로 읽었다.

**남은 사실은 둘이다.**

### 1.1 핸드오프는 노드별 config 파일이 없다 (읽음 + 실측)

`LaunchArgs` 가 `ConfigPath` 를 `nodeconfig.Spec` 에 넣지 않는다. 그래서 nodekey 도
static-nodes 도 **각 바이너리가 관례로 찾는 자리**에 있어야 하고, 프로필이
`nodekey_dir` 을 들고 있다.

**그 관례가 이미 한쪽에서 깨져 있다.** 라이브 실행의 node2 로그(실측):

```
ERROR The static-nodes.json file is deprecated and ignored.
      Use P2P.StaticNodes in config.toml instead.
```

gwbft 는 그 파일을 **무시한다.** 후계자들이 피어를 얻는 경로는 `WireMesh` 의
`admin_addPeer` 뿐이다. 보통 경로는 static nodes 를 config 에 적으므로 이 문제가
없다.

### 1.2 노드마다 RPC 네임스페이스가 다르다 (읽음)

`--http.api` 에 들어가는 이름이 `wemix`(poa) 와 `istanbul`(wbft) 로 갈린다.
`overrides()` 가 노드별로 넣고 있다.

### 1.3 겁먹을 필요 없던 것 — 플러그인 전체 (읽음)

처음에 "노드마다 체인 플러그인이 다르다"를 최대 난점으로 봤다. **재 보니 작다.**

- `LaunchPolicy` 는 **두 가족이 똑같다**(`{Mine: role==bp}`). poa 의 주석이 직접
  적는다 — "그 둘이 같은 것은 중복이 아니라, 봉인은 어느 알고리즘에서도
  생산자가 하는 일이라는 사실이다."
- 어휘는 `geth110-wemix` 와 `geth114` 로 다르지만, **차이나는 19개 플래그를
  chainbench 가 무조건 세팅하지 않는다.** 5개는 삭제, 14개는 wemix 전용 ChainExt
  노브이고 후자는 `--launch-opt` 로 요청해야 나온다. 없는 노브는
  `EnableIfSupported` 가 이미 건너뛴다.

**노드별로 진짜 필요한 것은 RPC 네임스페이스 하나다.**

---

## 2. 하드포크를 선언하는 자리

### 2.1 두 방식

`env.upgrade` 는 **모든 하드포크의 자리**로 넓힌다. 하드포크에는 방식이 둘 있고,
지금까지는 드문 쪽만 지원했다.

| | restart (보통) | concurrent (지금의 upgrade) |
|---|---|---|
| 바이너리 | 하나씩, 시간에 따라 바뀜 | 둘이 처음부터 동시에 |
| 노드 | 전부 같은 바이너리 | 노드마다 다름 |
| 전환 | 노드를 멈추고 새 바이너리로 재기동 | 없음 |
| 블록 생성자 | 안 바뀜 | 포크 후 바뀜 |
| genesis | **하나** (아래 §2.3) | 둘이어도 된다 |

`from` 과 `to` 는 두 방식에서 **같은 것을 가리킨다** — 포크 전을 맡는 쪽과 포크
후를 맡는 쪽이다. restart 는 같은 노드가 시간을 두고 바뀌고, concurrent 는 다른
노드가 동시에 있다. **시간으로 나뉘느냐 노드로 나뉘느냐**의 차이다.

### 2.2 선언

```json
{
  "schemaVersion": "2", "kind": "env", "id": "wemix-to-wbft",
  "chain": "wemix",
  "binaries": {
    "default": { "binary": "${GWEMIX_BIN:-gwemix}" },
    "next":    { "binary": "${GWBFT_BIN:-gwbft}", "chain": "wbft" }
  },
  "upgrade": {
    "preset": "wemix-to-wbft",
    "fork": "croissant",
    "at": 20,
    "from": "default",
    "to": "next",
    "style": "concurrent"
  },
  "genesis": {
    "mode": "template",
    "perBinary": { "next": { "set": { "config.croissant": { "...": "..." } } } }
  },
  "topology": { "nodes": [
    { "index": 1, "role": "en", "binary": "next" },
    { "index": 2, "role": "en", "binary": "next" },
    { "index": 3, "role": "en", "binary": "next" },
    { "index": 4, "role": "en", "binary": "next" },
    { "index": 5, "role": "bp" }
  ]}
}
```

읽는 법. **`chain` 은 포크 이전 체인**이다 — genesis 를 생성하는 바이너리이고,
부트업 순서를 선언하는 가족이다. `binaries.next.chain` 이 포크 이후 체인이다.

**생산자가 node5 인 것이 `plan_order` 를 대신한다**(§11.2.12). 프리셋 node5 가
생산자 계정의 keystore 를 들고 있으므로, 생산자를 그 자리에 선언하면 재매핑이
필요 없다.

**DSL 변경은 둘이다.** `binaries` 의 값이 문자열에서 객체가 되고(문자열은 "이
망의 체인을 쓰는 바이너리" 로 계속 읽어 기존 선언을 안 건드린다), `upgrade` 가
`{profile, template}` 에서 위 형태로 넓어진다.

**`style: restart` 는 지금 만들지 않는다.** 문법에는 자리를 두되, 주면 **이름을
대며 거부한다** — "restart 방식은 아직 구현되지 않았다". 쓰는 케이스가 하나도
없어서, 만들어도 돌려 볼 대상이 없기 때문이다. **조용히 무시하는 것과 이름을 대며
거부하는 것은 다르다.**

거부를 하나 더 건다. `style: restart` 인데 `genesis.perBinary` 가 적혀 있으면
받지 않는다. 이유는 §2.3 이다.

### 2.3 restart 는 왜 genesis 가 하나인가 (체인 코드 확인)

**세 체인 저장소를 읽고 확인했다.** go-stablenet `core/genesis.go`
`LoadChainConfigWithOverride`:

```go
newcfg := genesis.configOrDefault(stored)      // config 파일에 genesis 가 있으면 그 설정
...
if genesis == nil && ...사설망... {
    newcfg = storedcfg                          // 없을 때만 DB 에 저장된 것을 쓴다
}
```

그리고 `eth/ethconfig/config.go` 에 `Genesis *core.Genesis` 가 **toml 태그와 함께**
있다. go-wbft 와 go-wemix 도 같다.

**따라서 하드포크 설정은 config 파일로 적용된다.** 다만 조건이 붙는다.

```go
hash := genesis.ToBlock().Hash()
if hash != stored { return GenesisMismatchError }
```

genesis 블록 해시가 저장된 것과 **같아야 한다.** 그래서 alloc·extraData 같은
genesis 블록을 이루는 값은 그대로여야 하고, **chain config 의 포크 블록만 달라질
수 있다.** 포크가 0번 블록이면 genesis 블록 자체가 달라지므로 안 되지만,
하드포크 테스트는 0번이 아니다.

**런타임 플래그로는 안 된다.** `ChainOverrides` 에 `OverrideCancun` 과
`OverrideVerkle` 뿐이라 boho·applepie·croissant 는 못 덮는다. **config 파일이
유일한 길이다.**

그래서 restart 는 이렇게 돈다.

1. 모든 노드를 **포크 이전 genesis** 로 init
2. 포크 이전 바이너리로 기동
3. 노드를 멈추고, **포크 블록이 적힌 config** 를 써 주고, 새 바이너리로 재기동
4. 선언한 블록에서 포크가 켜진다

3번에서 datadir 을 다시 만들지 않으므로 **genesis 는 처음 하나로 고정된다.**
이것이 §2.2 의 두 번째 거부 이유다.

**chainbench 쪽에 필요한 것 하나.** 지금 config 렌더러가 내는 섹션은 `[Eth]`,
`[Eth.Miner]`, `[Node]`, `[Node.P2P]`, `[Metrics]` 이다. **`[Eth.Genesis]` 를 안
낸다.** restart 를 만들 때 이것이 추가할 한 가지다. 멈추고 config 를 다시 쓰고
재기동하는 동작 자체는 `swapNode` 에 이미 있다.

### 2.4 프로필을 하드포크 preset 으로

`profiles/wemix-upgrade.yaml` 은 이름이 무엇인지 말해 주지 않는다. **키와 체인에
preset 이 있으니 하드포크도 같은 자리로 옮긴다.**

**남는 값이 적다.** 오늘 확인한 바로 거버넌스 정책은 기본값과 하나하나 같고
(§11.2.11), 검증자 주소·BLS·extra_data 는 키셋에서 유도되며(§11.2.10),
`plan_order` 는 플랜이 생산자를 앞에 놓은 탓에 생긴 보정이다(§11.2.12). 빼고 나면
**포크 이름과 블록, network id, 역할 개수, 바이너리 둘** 정도다.

`upgrade.preset` 이 그것을 가리키고, 같은 키를 선언에 적으면 덮어쓴다 — 매니페스트
를 preset 으로 다루기로 한 것과 같은 규칙이다.

### 2.5 AwaitFork 는 테스트 쪽이다

포크 블록이 지난 뒤 후계자들이 실제로 블록을 봉인하는지 확인하는 일이다. **케이스가
`steps` 에 적어 원할 때 돌리는 것으로 둔다.**

구성 단계에서 필요한 경우가 나오면 같은 함수를 거기서도 부르면 된다. 지금 없는
쓰임을 위해 양쪽에 배선해 두지 않는다 — 이 트랙이 계속 걷어낸 모양이다.

---

## 3. 1,959줄이 어떻게 갈리나 (읽음)

| 파일 | 줄 | 판정 |
|---|---|---|
| `handoff.go` | 1039 | **대부분 사라진다.** `WriteConfig`·`BaseGenesis`·`ComposePlan`·`ApplyOverlay` 는 genesis 단계로, `LaunchPhase`·`BringUp` 은 start 단계로, `provisionKeys`·`overrides`·`machineFiles`·`label` 은 보통 경로에 이미 있다. **남을 후보**: `AwaitFork`·`confirmPostFork` — §2.5 대로 테스트 쪽으로 옮긴다 |
| `plan.go` | 326 | **사라진다.** 포트·순서·역할은 place 단계가 한다 |
| `exec.go` | 263 | **사라진다.** `BuildNodeSpecs`·`Launch` 는 start 단계가 한다 |
| `profile.go` | 153 | **축소해 하드포크 preset 의 로더로 남는다**(§2.4) |
| `mesh.go` | 125 | **조사 필요.** static-nodes 가 config 로 가면 `admin_addPeer` 메시가 필요 없을 수 있다 — §1.1 이 단서다 |
| `launch.go` | 53 | 사라진다 |
| `target.go` | 약 240 | **남는다.** 오늘 만들었고 두 표면이 쓴다. 보통 경로의 배치 해석과 합칠 여지가 있다 |

**이 표는 읽고 만든 것이다.** 각 항목은 4장의 순서를 밟으며 실제로 확인된다.

---

## 4. 순서 — 매 단계 끝에서 라이브로 통과해야 한다

중간에 깨진 채로 있는 구간을 두지 않는다. 각 단계마다 §5 를 돌린다.

1. **노드별 config 파일 — 완료 (2026-09-17).** 노드마다 `nodeconfig.Spec` 을 하나
   만들어 **config 파일과 argv 를 둘 다 거기서 낸다.** 그래서 파일과 명령줄이 한
   노드를 두고 어긋날 수 없다.

   **§1.1 과 §1.2 가 같이 없어졌다.** nodekey 는 `<datadir>/nodekey` 로 가고
   config 가 그 경로를 이름으로 말한다. static peers 는 `P2P.StaticNodes` 로
   간다. 그리고 config 의 `HTTPModules` 가 **각 노드 자신의 체인**에서 나오므로
   RPC 네임스페이스가 저절로 노드별로 갈린다(실측):

   ```
   node1 (생산자)  HTTPModules = [..., "wemix"]
   node2 (후계자)  HTTPModules = [..., "istanbul"]
   ```

   `Profile.Chains.*.NodekeyDir` 은 읽는 곳이 없어져 **지웠다.** 프로필에서도
   뺐다. gwbft 가 "deprecated and ignored" 로 답하던 `static-nodes.json` 도
   더는 쓰지 않는다(실측 — 로그에서 사라졌다).

   **설계에서 틀린 것 하나.** `ApplyConfigOverride` 로 argv override 를 Spec 에
   옮기면 된다고 적었는데, 그 함수는 `syncMode`·`httpHost`·`metricsHost` 세 개만
   받는다. 실제로는 계정 관련을 `Spec.Unlock`/`PasswordFile` 로 옮기고, RPC
   모듈은 config 에 맡기고, `--nat=none` 만 override 로 남겼다.

2. **`binaries` 값에 체인.** DSL 과 `NetUpIn` 이 바이너리별 체인을 나른다.
   `w.plugin()` 이 `w.pluginFor(ns)` 가 되는데, **10곳 중 노드별이 필요한 것은
   `Start`·`Config`·`swapNodeConfig`·`LaunchOpts` 넷**이고 나머지는 구성 전체의
   답으로 족하다(읽음 — 이 단계에서 컴파일러가 확인한다).

   **핸드오프 쪽 §1.2 는 1단계가 이미 풀었다.** 이 단계가 필요한 것은 **보통
   경로**가 섞인 망을 낼 수 있게 하기 위해서다.
3. **`upgrade` 선언을 넓힌다.** §2.2 의 문법을 파싱하고, `style: restart` 를 이름을
   대며 거부한다. 하드포크 preset 을 `upgrade.preset` 으로 읽는다.
4. **핸드오프 케이스를 보통 경로로 돌린다.** 같은 망이 서는지 §5 로 대조한다.
   **여기서 처음으로 흡수 가능 여부가 사실로 드러난다.** 생산자 2로도 돌려 본다.
5. **`upgrade` 축소.** 4가 통과한 뒤에야 지운다. `AwaitFork` 는 테스트 쪽으로.

**1과 2는 4의 결과와 무관하게 이득이다.** 1은 관례 의존과 무시되는 파일을 없애고,
2는 섞인 망의 어휘를 바로잡는다. 4가 실패해도 버려지지 않는다.

---

## 5. 라이브 검증 (실측 기준선)

환경변수 셋이 있어야 돈다 — `GWEMIX_BIN`, `GWBFT_BIN`,
**`GOWEMIX_TEMPLATE`**(go-wemix 의 `wemix/scripts/genesis-template.json`).

```
go run ./cmd/chainbench run --workspace-dir <ws> --artifact-root <out> \
  tests/tc/go-wemix/handoff/01-wemix-wbft-handoff.json
```

통과 기준선(2026-09-17 실측):

```
launch:boot: 1 node(s) → deploy-governance → etcd-init → verify-etcd
launch:endpoints: 4 node(s) → mesh: 5 endpoint(s)
await-fork: head 30; blocks 21-30 all sealed by the successor set, across 4 of 4
pass=1
```

`chainbench upgrade run` 쪽도 같이 돌린다. 두 표면이 같은 코드를 부르므로 한쪽만
보면 다른 쪽의 회귀를 놓친다.

---

## 6. 확인하지 않은 것

- **생산자가 둘 이상인 핸드오프.** `etcd-join` 과 단계별 순차 기동은 이제 돌 수
  있지만 **도는 것을 못 봤다.** 프로필이 생산자 1을 쓴다. 4단계에서 돌려 본다.
- **메시가 필요한지.** 1단계로 static peers 가 config 에 들어갔다(실측). 그래도
  `WireMesh` 를 뺀 채로는 **돌려 보지 않았다.** 빼고 돌려 봐야 안다.
- **`w.plugin()` 10곳의 분류.** 넷이 노드별이라는 것은 **읽고 내린 판단**이다.
  2단계에서 컴파일러가 확인한다.
- **원격 핸드오프.** `Target` 배선은 오늘 넣었지만 서버셋으로 **돌려 보지
  않았다.**
- **restart 방식 전체.** 문법만 열고 만들지 않는다. §2.3 의 config 경로는 체인
  코드를 읽어 확인했을 뿐, **chainbench 로 돌려 보지 않았다.**

---

## 7. 택하지 않은 안

**그대로 둔다.** 1,959줄은 돌아가고 라이브로 통과한다. 그런데 오늘 하루에만 그
안에서 결함 넷이 나왔다 — 두 오케스트레이션이 가족의 선언을 무시했고, 배선이
표면마다 달랐고, 무시되는 파일을 쓰고 있었고, 프로필이 사실을 세 번 다시 적고
있었다. **같은 일을 두 번 구현하면 배운 것이 한쪽에만 남는다**는 것이 이 파일
자신의 주석이 적은 말이다. 두는 쪽을 택하면 그 값을 계속 낸다.

**전부 한 번에 흡수한다.** 4장을 한 덩어리로 하면 중간에 라이브로 확인할 자리가
없다. 오늘 겪은 바로는 이 코드에서 확인 없이 두 걸음 이상 나가면 틀린다.

**restart 를 지금 같이 만든다.** 쓰는 케이스가 하나도 없어 돌려 볼 대상이 없다.
문법에 자리만 두고 거부해 두면, 케이스가 생길 때 채우면 된다.
