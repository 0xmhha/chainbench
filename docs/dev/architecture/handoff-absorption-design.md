# 하드포크 테스트를 보통 경로로 — 설계

**목적은 줄 수가 아니다.** 핸드오프가 다른 망과 **같은 규칙으로 도는 것**이 목적이고,
`internal/consensus/upgrade` 의 비테스트 1,959줄이 줄어드는 것은 그 결과다. 일부만
흡수하고 나머지를 남기는 결론도 정당하다.

**읽은 것과 돌려 본 것을 갈라 적는다.** 이 트랙에서 코드만 읽고 내린 판정이
2026-09-17 하루에 세 번 틀렸다(§11.2.5, §11.2.8-1, §11.2.9). 각 문장에 어느
쪽인지 표시한다.

---

## 0. 상태 — 끝났다 (2026-09-18)

**이 문서는 이제 계획이 아니라 기록이다.** §1·§3·§4 는 무엇을 하려 했는지이고,
§2·§5·§6 이 무엇이 됐는지다.

`internal/consensus/upgrade` 는 2,422줄에서 **118줄** 이 됐다. 남은 것은 하드포크
preset 로더 하나다. 핸드오프 본체·계획·기동·메시·타깃 해석이 전부 없어졌고,
`upgrade run`·`upgrade genesis` 명령과 `chainbench_upgrade` MCP 도구도 같이
내렸다. 합쳐서 4,894줄이 지워졌다.

하드포크는 이제 보통 경로로 구성된다. 선언 자리는 `env.upgrade` 하나이고,
방식이 둘이다.

| | concurrent | restart |
|---|---|---|
| 무엇이 바뀌나 | 누가 블록을 만드는가 | 어느 빌드가 도는가 |
| 노드 | 두 빌드가 동시에 | 한 빌드씩, 시간에 따라 |
| 포크 순간 | 후계자만 재기동 | 전부 재기동 |
| 포크 설정 | genesis (기본) 또는 config | **config** (§2.3) |
| 체인 | 둘 (wemix → wbft) | 하나 (stablenet boho) |

**라이브 통과 케이스 넷.**

```
go-wemix/hardfork/01-croissant-successors-take-over            구성이 넘긴다
go-wemix/hardfork/02-state-written-before-the-fork-survives-it 케이스가 넘긴다
go-wemix/hardfork/03-two-producers-hand-over                   생산자 둘
go-stablenet/hardfork/01-boho-crossed-by-restart               같은 체인, 빌드 교체
```

01 은 docker 서버셋 15대에도 올려 통과했다(§6).

---

## 1. 무엇이 막고 있었나 (착수 시점의 판정)

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

## 1.5 하드포크가 실제로 일어나는 절차 (2026-09-18, 정본)

**사용자가 준 순서다. 이것이 upgrade 와 보통 하드포크를 하나의 로직으로 돌리는
방식이고, 아래 설계는 여기에 맞춰야 한다.**

```
1. gwemix 를 포크 이전 genesis 로 init
2. 실행 → governance 배포 → NCP 등록 (블록 생성 자격)
3. 나머지 노드들(gwemix, gwbft) 모두 포크 이전 genesis 로 init → 실행
   단, gwbft 바이너리 중에 bp 가 있어서는 안 된다
4. 모든 노드가 생성된 블록을 제대로 동기화해야 한다
5. 노드 중단
6. 하드포크 정보가 든 genesis 로 모두 실행
7. 포크 블록에서 croissant 에 따라 후속 노드들이 validator 가 되어 블록 생성
```

### 여기서 읽히는 것

**genesis 는 바이너리별로 두 벌이 아니라 시간순으로 두 벌이다.** 포크 이전 것과
포크 정보가 든 것. 모든 노드가 같은 시점에는 같은 문서를 쓴다.

**포크 이전에 gwbft 노드는 생산자가 아니다**(3번). 동기화만 한다. 생산자가 되는
것은 **croissant 섹션이 validator 로 지명해서**이지 역할 때문이 아니다.

**그러니 역할은 재실행에서 바뀐다.** 5~6번의 중단과 재실행이 genesis 만 바꾸는
것이 아니라 "이제 누가 생산하는가" 도 함께 바꾼다.

**4번은 게이트다.** 동기화가 확인돼야 중단으로 넘어간다. 확인 없이 포크로 가면,
따라오지 못한 노드가 포크 뒤에 무엇을 하는지 알 수 없다.

**6번의 genesis 는 재init 없이 들어가야 한다.** 그래야 2~3번에서 쌓인
governance·NCP·블록이 남는다. 체인 코드에서 확인한 경로가 그것이다(§2.3) —
config 파일의 genesis 섹션이 이기고, **genesis 블록 해시는 같아야 하므로 chain
config 의 포크 블록만 달라질 수 있다.** chainbench 의 config 렌더러는 아직
`[Eth.Genesis]` 를 내지 않는다.

### 착수 시점의 구현과 다른 점

**chainbench 의 핸드오프는 이 절차를 밟지 않는다.** croissant 를 처음부터
genesis 에 넣고 한 번에 띄운다 — 중단도 재실행도 없다. 통과는 한다(블록 21-30 을
후계 검증자 4/4 가 봉인). 하지만 **절차가 아니다.**

**그리고 `style` 을 둘로 나눈 근거가 흔들린다.** `concurrent` 와 `restart` 를
"중단이 있느냐"로 갈랐는데, 정본에는 **둘 다 중단과 재실행이 있다.** 다른 것은
재실행 뒤에 **어떤 바이너리들이 도느냐** 뿐이다 — 같은 것들인지, 섞인 것들인지.

### 내 실행이 블록 19에서 멈춘 이유

이 절차로 설명된다. 나는 후계자를 `en` 으로 선언했고(3번대로 맞다), croissant 를
처음부터 genesis 에 넣었다. 그래서 포크 블록 20에서 wemix 는 넘겨주려 하는데
**받을 쪽이 생산자로 떠 있지 않았다.** 정본에서는 6번의 재실행이 그 역할 전환을
한다. **내게 없는 것은 역할이 아니라 재실행이다.**

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

### 2.3 restart 의 포크 설정은 config 로 간다 (실측 2026-09-18)

**두 번 틀리고 세 번째에 맞췄다.** 먼저 "config 파일이 유일한 길" 이라고 적었다가
(틀렸다 — genesis 로도 간다, §2.3b), restart 에서는 아예 필요 없다고 했다가
(틀렸다), 실측이 이유를 알려 줬다.

**진짜 이유는 누가 `init` 했느냐다.** 포크를 모르는 빌드가 init 하면, 그 빌드가
써 놓은 chain config 가 DB 에 남는다. **이후 모든 실행은 genesis.json 이 아니라
그것을 읽는다** — genesis 문서는 빈 DB 에만 들어간다. 그래서 새 빌드로 갈아타도
포크가 안 켜진다.

go-stablenet 을 boho 커밋 앞뒤로 빌드해 재 봤다.

```
체인이 블록 202 까지 감
저장된 chain config:  bohoBlock ABSENT
govMinter:            v1 그대로
```

config 파일은 **매 실행마다 읽힌다.** 그것이 이미 init 된 노드에 닿는 길이다.

### 2.3a 체인 코드에서 확인한 것 (읽음)

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
`OverrideVerkle` 뿐이라 boho·applepie·croissant 는 못 덮는다.

그래서 restart 는 이렇게 돈다.

1. 모든 노드를 **포크 이전 genesis** 로 init
2. 포크 이전 바이너리로 기동
3. 노드를 멈추고, **포크 블록이 적힌 config** 를 써 주고, 새 바이너리로 재기동
4. 선언한 블록에서 포크가 켜진다

3번에서 datadir 을 다시 만들지 않으므로 **genesis 는 처음 하나로 고정된다.**
이것이 §2.2 의 두 번째 거부 이유다.

**chainbench 쪽에 필요한 것 하나.** 지금 config 렌더러가 내는 섹션은 `[Eth]`,
`[Eth.Miner]`, `[Node]`, `[Node.P2P]`, `[Metrics]` 이다. `[Eth.Genesis]` 는
`nodeconfig.Spec.Genesis` 가 채워진 노드에만 붙는다(2026-09-18 추가). 멈추고
config 를 다시 쓰고 재기동하는 동작 자체는 `swapNode` 에 이미 있다.

### 2.3b 포크 설정은 genesis 로도 가고 config 로도 간다 (실측 2026-09-18)

go-wemix 4대와 go-wbft 4대로 chainId 1111 망을 띄워 croissant 를 블록 100 에
걸고 실제로 넘겨 보았다. 확인한 것은 다음과 같다.

**genesis 로 나르기 (기본값).** 포크 구간은 genesis 의 `croissant` 절에 넣는다.
go-wemix 는 자기 config 에 없는 절을 무시한다. genesis 의 JSON 디코더가 못 붙이는
키를 건너뛰기 때문이다. 대신 망에 genesis 가 두 벌이 된다.

**config 로 나르기.** genesis 에는 `croissantBlock` 만 두고, 절은 go-wbft 노드의
config `[Eth.Genesis]` 에 싣는다. 망의 genesis 는 한 벌이 되고 config 가 두 벌이
된다. 조건이 넷 붙는다.

1. `[Eth.Genesis]` 는 **genesis 전체**여야 한다. 일부만 실으면 저장된 genesis 와
   해시가 달라져 `database contains incompatible genesis` 로 거부된다.
2. TOML 은 **Go 필드 이름을 그대로** 맞춘다(`chainId` → `ChainID`,
   `eip155Block` → `EIP155Block`, `blsPublicKeys` → `BLSPublicKeys`). geth 의
   `NormFieldName` 이 항등 함수라서 그렇다.
3. 타입마다 표기가 다르다. `uint64` 는 정수, `[]byte` 는 정수 배열, `*big.Int`
   는 문자열이다. null 은 뺀다.
4. `core.Genesis` 에 없는 키는 **치명적**이다. go-wemix 가 쓰는
   `minerNodeId`·`minerNodeSig`·`rewards` 가 그렇다. JSON 은 무시하고 TOML 은
   죽는다. 그래서 체인이 manifest 의 `genesis.config_omit` 으로 선언한다.

`Params` 같은 표의 키는 **필드 이름이 아니라 데이터**다. 대문자로 바꾸면
config 는 읽히고 블록 실행에서 `invalid gov config params` 로 죽는다. 부팅 실패
보다 나쁘다.

**두 벌은 어느 쪽이든 피할 수 없다.** genesis 가 두 벌이 아니면 config 가 두
벌이 된다. genesis 쪽이 훨씬 단순하므로 기본값이다.

**공유 genesis 는 양쪽 init 을 통과해야 한다.** go-wbft 는 `petersburgBlock`
없이는 init 자체를 거부한다(`unsupported fork ordering`). 넣으면 go-wemix 도
받고 genesis 해시는 `4435de..383445` 로 동일하다.

### 2.4 프로필을 하드포크 preset 으로

`presets/chain/wemix-upgrade.yaml` 은 이름이 무엇인지 말해 주지 않는다. **키와 체인에
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

## 3. 1,959줄이 어떻게 갈리나 — 착수 시점의 예측 (읽음)

| 파일 | 줄 | 판정 |
|---|---|---|
| `handoff.go` | 1039 | **대부분 사라진다.** `WriteConfig`·`BaseGenesis`·`ComposePlan`·`ApplyOverlay` 는 genesis 단계로, `LaunchPhase`·`BringUp` 은 start 단계로, `provisionKeys`·`overrides`·`machineFiles`·`label` 은 보통 경로에 이미 있다. **남을 후보**: `AwaitFork`·`confirmPostFork` — §2.5 대로 테스트 쪽으로 옮긴다 |
| `plan.go` | 326 | **사라진다.** 포트·순서·역할은 place 단계가 한다 |
| `exec.go` | 263 | **사라진다.** `BuildNodeSpecs`·`Launch` 는 start 단계가 한다 |
| `profile.go` | 153 | **축소해 하드포크 preset 의 로더로 남는다**(§2.4) |
| `mesh.go` | 125 | **조사 필요.** static-nodes 가 config 로 가면 `admin_addPeer` 메시가 필요 없을 수 있다 — §1.1 이 단서다 |
| `launch.go` | 53 | 사라진다 |
| `target.go` | 약 240 | **남는다.** 오늘 만들었고 두 표면이 쓴다. 보통 경로의 배치 해석과 합칠 여지가 있다 |

**이 표는 읽고 만든 것이다.** 각 항목은 4장의 순서를 밟으며 실제로 확인됐고,
**예측이 맞았다** — `profile.go` 만 남았고(158 → 118줄) 나머지는 전부 없어졌다.
하나만 빗나갔다: `target.go` 를 "남는다" 로 봤는데, 그것을 쓰던 표면이 같이
내려가면서 지워졌다.

---

## 4. 순서 — 매 단계 끝에서 라이브로 통과해야 한다 (전부 완료)

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

2. **`binaries` 값에 체인 — 완료 (2026-09-17).** 선언이 바이너리마다 체인을 말할
   수 있다. 문자열은 "이 망의 체인" 이라는 뜻을 그대로 지킨다.

   ```json
   "binaries": { "default": "gwemix", "next": {"binary":"gwbft","chain":"wbft"} }
   ```

   `w.pluginFor(ns)` 가 `binaryFor`·`genesisFor` 와 같은 모양으로 들어갔다. **셋이
   같은 질문을 한다** — 이 노드는 이 망의 어느 빌드인가. 모양을 맞춰 두는 것이
   셋 중 하나만 다르게 답하는 일을 막는다.

   **노드별이 필요한 곳은 넷이 아니라 셋이었다.** `Start`·`Config`·
   `swapNodeConfig` 이고, `LaunchOpts` 는 플러그인을 peer 계획에만 쓰므로 구성
   전체의 답으로 족하다(읽음이 한 칸 빗나갔고, 컴파일러가 잡았다).

   swap 도 챙겼다. 이름을 노드 키로 바꿔 기록하므로 **체인이 그 이름과 함께
   따라가야 한다.** 안 그러면 다른 빌드로 갈아탄 노드가 원래 빌드의 어휘로
   재기동한다.

   **핸드오프 쪽 §1.2 는 1단계가 이미 풀었다.** 이 단계는 **보통 경로**가 섞인
   망을 낼 수 있게 한다 — 4단계의 전제다.
3. **`upgrade` 선언을 넓혔다 — 완료 (2026-09-18).** 선언이 `preset`·`fork`·`at`·
   `from`·`to`·`style` 을 받는다.

   **프로필이 하드포크 preset 이 됐다.** `profiles/` 는 외부 체인 접속 프로필과
   업그레이드 프로필을 같이 담고 있었다 — 같은 종류가 아닌데 디렉터리가 같다고
   말하고 있었다. 업그레이드 쪽을 `presets/chain/` 로 옮겼고, `upgrade.preset`
   이 이름으로 가리킨다. 경로로 가리키는 `profile` 은 남는다.

   **선언이 거짓말을 못 한다.** 케이스가 `fork`·`at` 을 적으면 preset 과 **대조**
   한다. 안 적으면 preset 이 답한다. 적고 틀리면 그 케이스는 다른 포크를 시험하며
   통과를 보고하게 되는데, 그게 제일 나쁜 실패다.

   **`style: restart` 는 이름을 대며 거부한다.** 그 밖에 스스로 모순인 선언도
   거부한다 — preset 과 profile 을 둘 다 적거나, 한쪽 이름을 양쪽에 적거나,
   `binaries` 가 모르는 이름을 적는 경우.
4. **핸드오프를 보통 경로로 돌렸다 — 실측 완료, 막힌 곳 둘 (2026-09-18).**

   §2.2 의 선언을 그대로 써서 돌렸다. **구성은 끝까지 갔다** — place, keys,
   genesis, config, build, deploy, init 이 전부 통과하고 기동에서 멈췄다.

   ```
   init: 5 datadir(s) initialized
   verify: the launched network matches the plan
   error: ... "deploy-governance": the node produced no block within 1m0s
   ```

   **막힌 것 하나 — 생산자가 계정을 열지 못한다 (실측).** 생산자 로그:

   ```
   Fatal: Failed to unlock account 0x5400d8b5… (no key for given address or file)
   ```

   보통 경로는 bp 에게 **그 노드의 nodekey 가 만들어 내는 주소**를 준다
   (`preset.Node(i).Address`). 그런데 프리셋 node5 의 keystore 에는 **다른
   계정**(`0xf9593d…`)이 들어 있다 — 일부러 그렇고, 그것이 핸드오프의 생산자
   계정이다(§11.2.12).

   **보통 경로에는 "이 노드는 이 계정을 연다" 고 말할 방법이 없다.** 길은 둘 —
   프리셋을 앞뒤가 맞게 고치거나, 노드 표가 계정을 말할 수 있게 하거나.

   *(로그의 `Unavailable modules ... unavailable=[wemix]` 는 무해하다. 통과하는
   보통 wemix 망에도 같은 줄이 있다 — 실측.)*

   **막힌 것 둘 — 포크 섹션을 선언할 수 없다 (읽음).** 핸드오프의 genesis 는
   from-체인 genesis 에 to-체인의 포크 섹션을 **계산해서** 얹은 것이다
   (`plan.go:166` — to-체인 genesis 를 그 템플릿으로 빌드하고,
   `ExtractConfigSection(…, "croissant")` 로 떼어, `SetConfigSection` 으로 얹는다).

   `genesis.perBinary` 는 **리터럴** overlay 를 받으므로 "X 체인의 포크 섹션을
   가져와라" 를 말할 수 없다. 게다가 그 섹션의 검증자는 **`to` 바이너리를 도는
   노드들**이라, 생산자를 고르는 `presetNetwork` 도 그대로는 못 쓴다.

   필요한 것은 작다 — `upgrade` 선언이 이미 `fork`·`at`·`from`·`to` 를 갖고
   있으니, **genesis 단계가 그것을 보고 빌드-추출-병합을 하면 된다.** 조각
   (`genesis.Build`·`ExtractConfigSection`·`SetConfigSection`·`ValidateForks`)은
   전부 `internal/core/genesis` 에 이미 있다.

   **덤으로 고친 것.** 이 실행이 결함 하나를 드러냈다 — `binaries` 의 `default`
   값이 **확장되지 않고** 있었다. 노드별 값은 확장되는데 default 만 아니어서,
   계획에 `${GWEMIX_BIN:-gwemix}` 가 그대로 찍혔고 exec 에도 그대로 갔을 것이다.

   **막힌 것 하나는 고쳤다 (2026-09-18).** 봉인 계정을 키셋이 답하게 했다.

   노드의 **신원**(devp2p 키)과 **봉인 계정**(keystore)은 다른 것이다. 일관되게
   쓰인 링에서는 겹치지만 겹쳐야 하는 것은 아니다 — wemix 생산자는 거버넌스
   멤버이고, 멤버는 피어가 아니라 계정이다. 프리셋 node5 가 일부러 그렇다.

   `Entry.Account` 와 `Entry.SealingAccount()` 를 두고, `LoadPresetWithAccounts`
   가 keystore 에서 읽는다. `LoadPresetWithKeys` 와 같은 모양이다 — 인덱스 한 번
   읽기는 그대로 두고, **답이 필요한 호출자가 값을 치른다.** genesis(계정에 잔고와
   스테이크를 준다)와 기동(그 계정을 연다) **둘 다 같은 함수를 읽는다.**

   고치면서 두 번 실측했다. 처음엔 열지 못했고
   (`no key for given address or file`), 열게 하니 이번엔 잔고가 없었다
   (`insufficient funds for gas * price + value`) — genesis 가 여전히 nodekey
   주소를 채우고 있었기 때문이다. **한쪽만 고치면 다른 쪽에서 막힌다**는 것이
   그대로 나왔다.

   **그래서 망이 떴다 (실측).** 생산자가 **블록 128** 까지 봉인했다.

   **그리고 남은 하나가 실측으로 확인됐다.** 후계자 넷은 **블록 0** 에 멈춰
   `Synchronisation failed, dropping peer` 를 8번 찍었다. genesis 에
   `croissant` 섹션도 `croissantBlock` 도 없으니, go-wbft 가 생산자의 블록을
   검증할 수 없다. **읽어서 낸 판정이 돌려서 확인됐다.**

   **포크 섹션도 넣었다 (2026-09-18).** `upgrade` 선언이 노드 표와 함께 오면
   **보통 경로로 구성한다.** 노드 표가 없으면 지금까지처럼 핸드오프 컴포저로
   간다 — 그쪽은 preset 의 역할 수로 망 크기를 정하기 때문이다.

   genesis 단계가 세 걸음을 한다. 포크 이후를 맡는 바이너리의 **체인 genesis 를
   그 체인 템플릿으로 빌드**하고, 그 **포크 섹션을 떼어**(`ExtractConfigSection`),
   이 genesis 에 **얹는다**(`SetConfigSection` + `<이름>Block`, 그리고 포크 순서
   재검증). 조각은 전부 이미 있었다.

   **검증자를 바이너리로 고른다.** 포크 이후의 검증자는 `to` 바이너리를 도는
   노드들이다 — 포크 전에는 엔드포인트이고 포크 후에 생산한다. 역할이 아니라
   빌드가 포크의 어느 쪽인지를 말한다. `binaryFor`·`genesisFor`·`pluginFor` 에
   이은 네 번째 같은 질문이다.

   **실측 — 여기까지 왔다.**

   ```
   genesis   croissant 섹션 있음, croissantBlock 20
   node1(gwbft) 블록 19 · node5(gwemix) 블록 19
   Synchronisation failed  0회   ← 이전 실행에서는 8회
   ```

   후계자들이 **동기화한다.** 이전 실행에서 블록 0에 멈춰 있던 것이 포크 섹션이
   없어서였음이 확인됐다.

   **그리고 다음 관문이 드러났다 — 체인이 블록 19, 정확히 포크 직전에서 멈춘다.**
   후계자가 이어받지 못한다. 이유는 **역할로는 "누가 생산하는가" 를 말할 수 없기
   때문**이다. `--mine` 은 `role == bp` 에서 나오는데, 핸드오프에서는 **포크 전후로
   생산하는 노드가 다르다.** 실제 핸드오프의 후계자 로그에는
   `Commit new sealing work` 가 있다 — 즉 그쪽은 생산자로 뜬다.

   후계자를 `bp` 로 선언하면 이번엔 wemix genesis 가 그들까지 거버넌스 멤버로
   앉힌다(`poa.GenesisSource.config` 가 역할로 고른다). **포크 이전의 생산자는
   `from` 바이너리를 도는 노드들**이어야 한다 — 같은 규칙의 다섯 번째 적용이다.

   **아직 안 한 것.** 위 하나. 그리고 생산자 2 이상.

5. **그리고 내가 만든 것 하나는 아직 소비자가 없다.** 바이너리별 genesis
   (`genesis.perBinary`)는 두 바이너리가 같은 문서를 못 받아들일 때를 위한 것인데,
   **wemix·wbft 쌍은 받아들인다**(하나의 genesis 로 라이브 통과). `restart` 방식도
   genesis 가 하나다. 그러니 지금 이 기능을 쓰는 것이 없다. 사용자가 말한 모델은
   실재하지만, **이 쌍에는 해당하지 않는다.**
6. **`upgrade` 축소.** 4가 통과한 뒤에야 지운다. `AwaitFork` 는 테스트 쪽으로.

**1과 2는 4의 결과와 무관하게 이득이다.** 1은 관례 의존과 무시되는 파일을 없애고,
2는 섞인 망의 어휘를 바로잡는다. 4가 실패해도 버려지지 않는다.

---

## 5. 라이브 검증 (실측 기준선)

### 5.0 흡수 전의 기준선 (2026-09-17, 역사)

전용 컴포저를 돌리던 시절의 것이다. 그 케이스도 컴포저도 없어졌으므로 재현할 수
없고, 무엇을 지키며 옮겼는지를 말하기 위해 남긴다.

```
launch:boot: 1 node(s) → deploy-governance → etcd-init → verify-etcd
launch:endpoints: 4 node(s) → mesh: 5 endpoint(s)
await-fork: head 30; blocks 21-30 all sealed by the successor set, across 4 of 4
pass=1
```

### 5.1 흡수 뒤의 기준선 (2026-09-18, 현행)

보통 경로의 것이다. `GOWEMIX_TEMPLATE` 은 아무도 읽지 않는다 — 체인 플러그인이
자기 템플릿을 들고 있다.

**concurrent — 두 체인, 두 빌드가 동시에.**

```
GWEMIX_BIN=... GWBFT_BIN=... go run ./cmd/chainbench run \
  --workspace-dir <ws> --artifact-root <out> \
  tests/tc/go-wemix/hardfork/*.json
```

```
cross-fork: croissant at 20: head 19, 4 successor(s) now produce
cross-fork: croissant at 30: head 29, 4 successor(s) now produce
01-croissant-successors-take-over            pass=1
02-state-written-before-the-fork-survives-it pass=1
03-two-producers-hand-over                   pass=1
```

**restart — 한 체인, 빌드를 갈아탄다.** 두 빌드가 필요하고, 만드는 법은 케이스
description 에 적혀 있다(go-stablenet 의 `bf17c9607` 앞뒤).

```
GSTABLE_BIN=<gstable-ad0122> GSTABLE_POSTFORK_BIN=<gstable-740526> \
  go run ./cmd/chainbench run --workspace-dir <ws> --artifact-root <out> \
  tests/tc/go-stablenet/hardfork/01-boho-crossed-by-restart.json
```

```
cross-fork: boho at 200: 4 node(s) relaunched on postfork at head 5, before the fork
pass=1
```

**대조군도 같이 돌린다.** `GSTABLE_POSTFORK_BIN` 을 이전 빌드로 두면 망이 뜨지
않아야 한다(`field 'BohoBlock' is not defined`). 통과하면 그 케이스는 새 빌드가
필요하다는 것을 더는 증명하지 못한다.

`chainbench upgrade run` 은 없어졌다. 두 표면이 같은 코드를 부르던 구조가 이
트랙에서 계속 문제였고, 한쪽이 없어지면 그 값을 더는 내지 않는다.

---

## 6. 확인한 것과 남은 것

### 확인됐다 (2026-09-18)

- **생산자가 둘 이상인 핸드오프.** 노드 표로 선언하니 프리셋 30개가 필요 없다.
  생성한 6노드 키셋으로 생산자 2 · 후계자 4 를 띄웠고, 두 생산자가 etcd 군집을
  이루어 번갈아 봉인했다(`admin_wemixInfo` 의 `etcd.members` 2, `miners` 가
  `node5/up node6/up/*`). 생산자가 하나면 부트 노드가 혼자 군집을 만들고 끝나서
  합류 단계 자체가 돌지 않았다.
- **메시가 필요한지.** 필요 없다. `WireMesh` 를 지운 채로 세 케이스가 통과한다.
  static peers 는 config 의 `P2P.StaticNodes` 로 간다.
- **`w.plugin()` 10곳의 분류.** 2단계에서 컴파일러가 확인했다 — 넷이 아니라
  셋이었다.

- **원격 하드포크.** docker 서버셋(`env/docker`, 15 컨테이너)에 `--all-servers
  --docker` 로 돌렸다. 노드 다섯이 server1~server5 에 하나씩 놓였고, 포크를
  넘어 통과했다(`pass=1`).

  **그 실행이 결함 하나를 드러냈다.** 계획의 placement 는 **env 가 선언한** 것이고
  명령줄로 고른 서버는 별개의 항목이라, `--server`·`--all-servers` 로 놓은 실행은
  placement 가 기계를 안 적는다. 그런데 기록은 항상 구성한 기계를 적는다. 그래서
  `VerifyLaunched` 가 "asked for nodes on local /data/chainbench, launched with
  nodes on server server1:/data/chainbench" 로 거부했다 — 같은 배치를 두 가지로
  말한 것이다. 기계를 적은 placement 는 그 기계에, 안 적은 것은 데이터 루트에
  묶도록 고쳤다. **`chainbench run` 으로 원격을 돌리면 항상 걸리던 것이라,
  하드포크만의 문제가 아니었다.**

- **restart 방식.** `tests/tc/go-stablenet/hardfork/01-boho-crossed-by-restart.json`
  으로 돈다(`pass=1`).

  **처음에 잘못 읽었다.** restart 를 "포크 섹션을 나중에 나르는 것" 으로 보고
  config 캐리어를 강제했는데, 아니다. **restart 는 같은 체인의 하드포크다.**
  포크 블록도 섹션도 genesis 에 처음부터 있고, **바뀌는 것은 바이너리뿐**이다.
  포크를 모르는 빌드는 그 블록에서 아무것도 안 하므로 빌드를 바꿔야 하는 것이다.

  **wemix→wbft 로는 안 된다.** 두 빌드가 datadir 안의 이름을 다르게 쓴다 —
  go-wemix 의 gwemix 는 `geth/`, go-wbft 의 gwbft 는 `gwemix/` 다. 같은
  `--datadir` 를 주고 물으면 gwemix 는 29, gwbft 는 0 을 답한다(실측). 제자리
  교체가 성립하지 않는다. 그리고 그 쌍이 실제로 배포되는 방식은 concurrent 다.

  **시점이 핵심이다.** 보통 하드포크는 체인을 멈추지 않는다. 그래서 핸드오버처럼
  At-1 을 기다리면 안 된다 — 실행기 넷을 바꿀 시간이 한 블록뿐이고 되는지는
  운이다. 재시작은 **지금 즉시** 하고, 끝난 뒤에도 체인이 포크를 안 넘었는지
  확인한다. 넘었으면 거부한다(실측으로 발동 확인).

  **준비 확인의 두 상태는 restart 에 해당하지 않는다.** parked 를 restart 에
  적용했더니 높이 0에서 게이트가 통과했고, retired 를 적용하면 아무것도 안 도는
  망이 ready 로 읽힌다. 둘 다 핸드오버 전용으로 막았다.

  **두 빌드로 확인했다.** go-stablenet 의 boho 커밋(`bf17c9607`) 앞뒤로 빌드했다 —
  `gstable-ad0122`(부모, `BohoBlock` 이 아예 없다)와 `gstable-740526`. 두 커밋의
  `params/config.go` 차이는 **정확히 `Boho`·`BohoBlock` 둘뿐**이라 다른 변수가
  섞이지 않는다.

  ```
  두 빌드      pass=1, govMinter 가 v2 로 바뀐다
  이전 빌드만  망이 뜨지도 않는다 — field 'BohoBlock' is not defined in params.ChainConfig
  ```

  대조군이 실패하는 방식이 더 강하다. 포크가 조용히 안 일어나는 것이 아니라,
  **이전 빌드가 그 설정을 아예 거부한다.**

  **그리고 restart 가 왜 config 로 날라야 하는지가 여기서 드러났다.** 처음엔
  "섹션을 늦게 나른다" 로 잘못 짚었다가 아예 빼 버렸는데, 다시 넣어야 했다.
  진짜 이유는 이것이다 — **노드를 init 한 것이 포크를 모르는 빌드**다. 그 빌드가
  써 놓은 chain config 가 DB에 남고, 이후 모든 실행은 genesis.json 이 아니라 그것을
  읽는다. 실측: 블록 202 까지 갔는데 저장된 config 에 `bohoBlock` 이 없고 govMinter
  는 v1 그대로였다.

  **변환기에서 셋을 더 고쳤다**(전부 stablenet genesis 가 드러냈다). `baseFeePerGas`
  의 Go 필드는 `BaseFee` 다. alloc 주소를 `0x` 없이 적는 문서가 있는데 그 키는
  자유 문자열이 아니라 주소 타입이라 접두사가 필요하다. 엔진 섹션을 `wbft` 로
  적는 템플릿과 `wBFT` 로 적는 템플릿이 있고 Go 필드는 하나다.

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
