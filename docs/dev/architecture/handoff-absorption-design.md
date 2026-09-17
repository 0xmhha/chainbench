# 핸드오프를 보통 경로로 — 설계

**목적은 줄 수가 아니다.** 핸드오프가 다른 망과 **같은 규칙으로 도는 것**이 목적이고,
`internal/consensus/upgrade` 의 비테스트 1,959줄이 줄어드는 것은 그 결과다. 일부만
흡수하고 나머지를 남기는 결론도 정당하다.

**읽은 것과 돌려 본 것을 갈라 적는다.** 이 트랙에서 코드만 읽고 내린 판정이
2026-09-17 하루에 세 번 틀렸다(§11.2.5, §11.2.8-1, §11.2.9). 아래 각 문장은 어느
쪽인지 표시한다.

---

## 1. 지금 무엇이 남아 있나

워크리스트가 흡수를 막는다고 적었던 넷은 **전부 정리됐다**(§11.2.9~12). 셋은
프로필이 **선택이 아니라 사실을 다시 적고** 있었고, 하나는 내가 거꾸로 읽었다.

**막는 것으로 남은 사실은 둘이다.**

### 1.1 핸드오프는 노드별 config 파일이 없다 (읽음 + 실측)

`LaunchArgs` 가 `ConfigPath` 를 `nodeconfig.Spec` 에 넣지 않는다. 그래서 nodekey 도
static-nodes 도 **각 바이너리가 관례로 찾는 자리**에 있어야 하고, 프로필이
`nodekey_dir` 을 들고 있다.

**그런데 그 관례가 이미 한쪽에서 깨져 있다.** 라이브 실행의 node2 로그(실측):

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

처음에 "노드마다 체인 플러그인이 다르다"를 큰 장애로 봤다. **재 보니 작다.**

- `LaunchPolicy` 는 **두 가족이 똑같다**(`{Mine: role==bp}`). poa 의 주석이 직접
  적는다 — "그 둘이 같은 것은 중복이 아니라, 봉인은 어느 알고리즘에서도
  생산자가 하는 일이라는 사실이다."
- 어휘(dialect)는 `geth110-wemix` 대 `geth114` 로 다르지만, **차이나는 19개
  플래그를 chainbench 가 무조건 세팅하지 않는다.** 5개는 삭제, 14개는 wemix 전용
  ChainExt 노브이고 후자는 `--launch-opt` 로 요청해야 나온다. 없는 노브는
  `EnableIfSupported` 가 이미 건너뛴다.

**즉 노드별로 진짜 필요한 것은 RPC 네임스페이스 한 가지다.** 나머지는 이미 같다.

---

## 2. 흡수 후의 모습

### 2.1 선언 (제안)

```json
{
  "schemaVersion": "2", "kind": "env", "id": "wemix-to-wbft",
  "chain": "wemix",
  "binaries": {
    "default": { "binary": "${GWEMIX_BIN:-gwemix}" },
    "next":    { "binary": "${GWBFT_BIN:-gwbft}", "chain": "wbft" }
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

**DSL 변경은 하나뿐이다** — `binaries` 의 값이 문자열에서 객체가 된다. 문자열은
"이 망의 체인을 쓰는 바이너리" 로 계속 읽어 기존 선언을 안 건드린다.

### 2.2 결정이 필요한 것 (사용자)

| # | 질문 | 선택지 |
|---|---|---|
| A | 기존 `env.upgrade: {profile, template}` 선언은 어떻게 되나 | (ᄀ) 위 셋으로 **번역되는 축약**으로 남긴다 (ᄂ) 케이스 파일을 고쳐 축약을 없앤다 |
| B | 프로필은 어디에 사나 | 오늘 확인된 바로 내용의 대부분이 유도되거나 기본값이다(§11.2.10~12). 남는 것은 fork 이름·블록·network id·roles 정도다. (ᄀ) preset 파일로 남긴다 (ᄂ) env 선언에 흡수한다 |
| C | `AwaitFork` 는 구성인가 테스트인가 | 포크 이후 후계자가 봉인하는지 확인하는 것은 **단언**에 가깝다. (ᄀ) 케이스의 step 으로 내린다 (ᄂ) 구성 단계로 남긴다 |

A 는 표면 호환성, B 는 무엇이 preset 인지, C 는 구성과 검증의 경계다. 셋 다 내가
고를 것이 아니다.

---

## 3. 1,959줄이 어떻게 갈리나 (읽음)

| 파일 | 줄 | 판정 |
|---|---|---|
| `handoff.go` | 1039 | **대부분 사라진다.** `WriteConfig`·`BaseGenesis`·`ComposePlan`·`ApplyOverlay` 는 genesis 단계로, `LaunchPhase`·`BringUp` 은 start 단계로, `provisionKeys`·`overrides`·`machineFiles`·`label` 은 보통 경로에 이미 있다. **남을 후보**: `AwaitFork`·`confirmPostFork`(C 에 달림) |
| `plan.go` | 326 | **사라진다.** 포트·순서·역할은 place 단계가 한다 |
| `exec.go` | 263 | **사라진다.** `BuildNodeSpecs`·`Launch` 는 start 단계가 한다 |
| `profile.go` | 153 | **B 에 달림.** preset 으로 남으면 축소, 흡수하면 사라진다 |
| `mesh.go` | 125 | **조사 필요.** static-nodes 가 config 로 가면 `admin_addPeer` 메시가 필요 없을 수 있다 — §1.1 이 그 단서다 |
| `launch.go` | 53 | 사라진다 |
| `target.go` | 약 240 | **남는다.** 오늘 만들었고 두 표면이 쓴다. 다만 보통 경로의 배치 해석과 합칠 여지가 있다 |

**이 표는 읽고 만든 것이다.** 각 항목은 4장의 순서를 밟으며 실제로 확인된다.

---

## 4. 순서 — 매 단계 끝에서 핸드오프가 라이브로 통과해야 한다

중간에 깨진 채로 있는 구간을 두지 않는다. 각 단계마다 §5 의 라이브 검증을 돌린다.

1. **노드별 config 파일.** 핸드오프가 `nodeconfig.TOML` 로 노드별 config 를 쓰고
   `ConfigPath` 를 넘긴다. argv 층 override 를 `ApplyConfigOverride` 로 Spec 에
   옮긴다. **이 단계가 §1.1 을 없앤다** — nodekey 와 static-nodes 가 선언으로 가고
   `nodekey_dir` 의존이 사라진다. 덤으로 gwbft 가 무시하던 파일이 없어진다.
2. **`binaries` 값에 체인.** DSL 과 `NetUpIn` 이 바이너리별 체인을 나른다.
   `w.plugin()` 이 `w.pluginFor(ns)` 가 되는데, **10곳 중 노드별이 필요한 것은
   `Start`·`Config`·`swapNodeConfig`·`LaunchOpts` 넷**이고 나머지는 구성 전체의
   답으로 족하다(읽음 — 4장에서 확인된다). 이것이 §1.2 를 없앤다.
3. **핸드오프 케이스를 보통 경로로 선언해 돌린다.** 2.1 의 선언으로 같은 망이
   서는지 대조한다. **여기서 처음으로 흡수가 가능한지 사실로 드러난다.**
4. **`upgrade` 축소.** 3이 통과한 뒤에야 지운다. A·B·C 의 답에 따라 남길 것을
   남긴다.

**1과 2는 3의 결과와 무관하게 이득이다.** 1은 관례 의존과 무시되는 파일을 없애고,
2는 섞인 망의 어휘를 바로잡는다. 3이 실패해도 버려지지 않는다.

---

## 5. 라이브 검증 (실측 기준선)

핸드오프는 환경변수 셋이 있어야 돈다 — `GWEMIX_BIN`, `GWBFT_BIN`,
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
  있지만 **도는 것을 못 봤다.** 프로필이 생산자 1을 쓴다. 흡수 뒤에는 노드 표로
  생산자를 늘릴 수 있으니, **3단계에서 생산자 2로도 돌려 본다.**
- **메시가 필요한지.** §1.1 의 로그가 static-nodes 가 무시된다고 말하는데, config
  로 옮기면 `admin_addPeer` 가 불필요해지는지는 1단계에서 확인된다.
- **`w.plugin()` 10곳의 분류.** 넷이 노드별이라는 것은 **읽고 내린 판단**이다.
  2단계에서 컴파일러가 확인해 준다.
- **원격 핸드오프.** `Target` 배선은 오늘 넣었지만 서버셋으로 **돌려 보지
  않았다.**

---

## 7. 택하지 않은 안

**그대로 둔다.** 1,959줄은 돌아가고 라이브로 통과한다. 그런데 오늘 하루에만 그
안에서 결함 넷이 나왔다 — 두 오케스트레이션이 가족의 선언을 무시했고, 배선이
표면마다 달랐고, 무시되는 파일을 쓰고 있었고, 프로필이 사실을 세 번 다시 적고
있었다. **같은 일을 두 번 구현하면 배운 것이 한쪽에만 남는다**는 것이 이 파일
자신의 주석이 적은 말이다. 두는 쪽을 택하면 그 값을 계속 낸다.

**전부 한 번에 흡수한다.** 4장을 한 덩어리로 하면 중간에 라이브로 확인할 자리가
없다. 오늘 겪은 바로는 이 코드에서 확인 없이 두 걸음 이상 나가면 틀린다.
