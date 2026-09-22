# 선언 모델 — 체인을 무엇으로 서술하고, 값은 어느 순서로 정해지나

> 상태: **[제안]** · 2026-09-22 · 리팩토링 설계
>
> 이 문서가 답하려는 것은 하나다. **테스트 정의서를 읽는 것만으로 어떤 망을 세워
> 무엇을 검사하는지 알 수 있는가.** 지금은 알 수 없고, 이 문서는 왜 알 수 없는지를
> 측정으로 보이고 목표 모델과 이행 순서를 적는다.

---

## 0. 한 장 요약

지금 체인을 서술하는 문서가 **네 자리에 세 이름**으로 있다. 같은 사실이 두 곳에 다른
낱말로 적혀 있고, 어느 쪽이 이기는지 정한 곳이 없다. 값의 우선순위를 말하는 어휘가
**셋**이고 셋 다 서로를 모른다.

그 결과로 결함 셋이 살아 있다. config 파일과 argv 가 서로 다른 network id 를 적고,
균일성을 지키라고 쓴 파일은 호출자가 0이며, `--chain-id` 를 바꿔도 network id 가
따라가지 않는다.

목표는 **서술 한 벌, 어휘 한 벌, 우선순위 한 줄**이다.

---

## 1. 지금 — 네 자리, 세 이름

| 무엇을 적나 | 어디 | 부르는 이름 | 형식 | 개수 |
|---|---|---|---|---|
| 체인 구현이 무엇인가 (chain_id·network_id·dialect·hardfork 목록·genesis 템플릿·빌드) | `internal/chains/<id>/manifest.json` | **manifest** | JSON, `go:embed` | 3 (+외부) |
| 이 망이 어떻게 생겼나 (노드 수·역할·바이너리·키·launch·config) | `tests/tc/env/*.json` | **env** | DSL JSON | 26 |
| 검증된 하드포크 환경 | `presets/chain/*.yaml` | **preset** (`preset.Chain`) | YAML | 2 |
| 노드 신원 | `presets/keys/` | **preset** (`preset.Key`) | 디렉터리 | 1 |

2번과 3번은 **같은 종류**다. 둘 다 "재사용되는 환경 선언"이고, 둘 다 케이스가 이름으로
가리키며, 둘 다 케이스의 override 를 받는다. 형식과 위치와 낱말만 다르다.

케이스는 `"env": "wemix-to-wbft-bp2"` 로 2번을 가리키고, 그 env 안에서
`"upgrade": {"preset": "wemix-upgrade"}` 로 3번을 가리킨다. **한 망을 세우는 데 선언이
두 겹**이고, 바깥 겹이 안쪽 겹의 값을 대부분 다시 적는다.

### 1.1 같은 사실이 두 곳에 있다

`internal/chains/wemix/manifest.json`

```json
"chain_id": 8285, "network_id": 8285,
"upgrade": { "to_chain": "wbft", "at_fork": "croissant", "validator_source": "croissant_init" }
```

`presets/chain/wemix-upgrade.yaml`

```yaml
upgrade:
  from: wemix   to: wbft   at_fork: croissant   fork_block: 20   network_id: 8285
```

preset 의 다섯 필드 중 **넷이 manifest 에 이미 있다.** 고유한 것은 `fork_block` 하나이고,
그것은 실행마다 달라지는 일정이라 원래 선언이 아니라 케이스의 것이다. 실제로 env 파일이
`"at": 20` 으로 직접 적고 있고, 그때 preset 은 읽히지도 않는다.

### 1.2 우선순위를 말하는 어휘가 셋이다

| 어휘 | 값 | 어디에 쓰나 |
|---|---|---|
| `blueprint.Origin` | blueprint · inventory · keyset · chain · default | 해결된 망의 값마다 출처 기록 |
| `nodeconfig.Layer` | harness · role · env.launch · command | argv 조립에서 어느 층이 이기나 |
| `testengine.PlanOrigin` | declaration · command · harness | 계획이 값마다 누가 골랐나 |

셋은 순서가 다르고 이름이 겹치며(`command`·`harness`·`default`) **셋 다 preset 과
server-set 을 모른다.** preset 의 값이 아무 데도 안 닿는 이유가 이것이다. 안 쓸모없어서가
아니라 **들어갈 단이 없어서**다.

---

## 2. 그래서 살아 있는 결함 셋

### D1. 같은 노드의 argv 가 띄우는 경로마다 다르다 — 실측으로 확인 (2026-09-22)

argv 를 만드는 자리가 셋인데 서로 다른 plugin 을 쓴다.

```
steps_config.go:182  LaunchOpts       w.plugin()      망 전체의 체인
phases.go:85         startPhase       pluginFor(ns)   그 노드의 체인
steps_crossfork.go   포크 뒤 재기동    pluginFor(ns)   그 노드의 체인
```

`nodeconfig.Chain` 이 노드별 사실(dialect·RPC namespace)과 망 전체 사실(network id)을 한
구조체에 담고, 그것을 `ChainOf(plugin, role)` 가 **넘겨받은 plugin 하나**로 채우기 때문이다.
타입의 주석은 스스로 "none of them per node" 라고 적고 있는데, 부르는 쪽 둘이 노드별 plugin 을
넘긴다.

**실측.** wemix→wbft 핸드오버(`01-croissant-successors-take-over`)를 고치기 전 코드로 돌려
기록된 argv 를 읽었다.

```
옛 코드: node1=8284 node2=8284 node3=8284 node4=8284 node5=8285
D-a 뒤 : node1=8285 node2=8285 node3=8285 node4=8285 node5=8285
```

successor 넷과 producer 가 **다른 devp2p 망 번호로 떠 있었다.** 그런데도 케이스는 통과한다.
포크 전에는 `LaunchOpts` 가 만든 argv(8285)로 떠서 producer 와 붙어 동기화하고, 포크 뒤
재기동에서 8284 로 바뀌는데 그때는 producer 가 이미 멈췄고 넷끼리는 번호가 같기 때문이다.
**구성이 그렇게 생겨서 안 터진 것**이고, 포크 뒤에도 옛 빌드로 남는 노드가 하나라도 있으면
그 노드는 나머지와 peer 하지 못한다.

(config 파일은 이 값을 아예 적지 않는다. TOML 렌더러에 network id 가 없다. 처음에는 config 와
argv 가 어긋난다고 적었는데, 재보니 어긋나는 것은 argv 와 argv 였다.)

### D2. 균일성 검사가 배선돼 있지 않다 — 닫음 (2026-09-22)

`resource` 밑에 `netid.go` 라는 파일이 있었다. 이 함정을 막으려고 쓴 것이고, 주석이 직접 말한다.

> go-wemix 는 chain id 와 무관하게 1111 로 기본값을 잡고, go-wbft 는 chain id 에서
> 유도한다. 그래서 한 체인을 이루려는 두 바이너리는 같은 network id 를 명시하지 않으면
> peer 를 거부한다.

`Resolve`·`Flag`·`ValidateUniform` 세 함수의 **호출자가 0**이었다. 자리도 틀렸다 —
`resource` 의 package doc 이 자기가 가진 파일을 열거하는데 `netid.go` 가 그 목록에 없고,
그 패키지가 소유한다고 적은 것은 "어떤 서버가 있고 어느 포트를 주나" 다.

검사를 `nodeconfig` 로 옮겼다(`ValidateUniformNetworkID`). argv 의 어휘를 아는 패키지이고,
**조립된 argv 를 읽는다.** 입력이 아니라 산출물을 보는 이유는, 입력은 `network()` 하나에서
오므로 스스로와 어긋날 수 없기 때문이다. 어긋나는 길은 `network()` 를 안 거치는 길이다 —
한 스코프에만 걸린 launch override, 또는 나중에 쓰이는 네 번째 조립기.

옛 파일의 `Resolve`·`Flag` 는 지웠다. 해석은 이제 `NetworkOf` 가 하고, 플래그 철자는
dialect 표가 갖는다. `Flag` 가 없어지면서 이름 충돌 빚도 하나 줄었다.

### D3. chain id 를 바꿔도 network id 가 안 따라간다 — 닫음 (2026-09-22)

`suite run --chain-id` 는 genesis 의 chainId 를 덮는다. network id 는 `ChainOf()` 가
manifest 에서 그대로 읽는다(`nodeconfig.Chain` 에는 `ChainID` 필드가 아예 없다). 둘을 잇는
코드가 없다.

go-wbft 는 network id 를 chain id 에서 유도하므로, chain id 를 덮은 망에서 wbft 노드는
자기 chain id 와 다른 network id 를 명령줄로 받는다.

**실측.** `--chain-id 4242` 로 stablenet 케이스를 고치기 전 코드로 돌렸다.

```
옛 코드: genesis chainId = 4242, 노드 넷 전부 --networkid 8283  (manifest 값)
고친 뒤: genesis chainId = 4242, 노드 넷 전부 --networkid 4242
```

케이스는 옛 코드에서도 통과한다. 넷이 다 8283 이라 서로는 붙기 때문이다. 문제는 **그 번호가
손대지 않은 stablenet manifest 로 세운 다른 망의 번호와 같다**는 것이다. 옛 `netid.go` 주석이
경고한 "서로 보면 안 되는 두 망이 합의해 버린다" 가 그것이다.

---

## 3. 정해야 할 사실 — network id 규칙

**하드포크로 이어받는 노드는 포크 전과 같은 network id 로 떠야 한다.** go-wbft 를 하드포크로
쓸 때는 network id 를 그 망의 chain id 에 맞춰 명시해야 한다. go-wbft 로 처음부터 init 해서
띄우는 망이면 상관없다.

지금 세 manifest 는 전부 `chain_id == network_id` 다 (8285/8285, 8284/8284, 8283/8283).
즉 `network_id` 필드는 **오늘 하나도 정보를 더하지 않는다.**

그러므로 규칙을 파생으로 적는다.

> **network id 는 그 망의 chain id 다.** manifest 의 `network_id` 는 파생과 달라야 하는
> 체인만 적는 override 이고, `--network-id` 가 그 위를 덮는다.

필드는 남기되 **파생과 같은 값을 적으면 거부한다**(확정 2026-09-22). 그래야 필드가 적혀 있다는
사실 자체가 "일부러 다르게 했다" 는 선언이 된다. 래칫이 없으면 세 체인처럼 같은 값을 적어 둔
필드가 다시 쌓이고, 다음 사람은 그것이 결정인지 관성인지 알 수 없다.

이 한 줄이 D1·D2·D3 를 한꺼번에 닫는다. 하드포크에서는 망의 chain id 가 포크 전 값이므로
모든 노드가 같은 값을 받고, `--chain-id` 를 덮으면 network id 가 따라가며, 파생이 한 곳에서
일어나므로 config 와 argv 가 갈라질 자리가 없다.

---

## 4. 목표 모델

### 4.1 서술은 세 종류, 이름은 각각 하나

| 종류 | 무엇 | 누가 쓰나 | 바뀌는 때 |
|---|---|---|---|
| **체인 정의** | go-wemix 라는 소프트웨어가 무엇인가 | 도구가 안다. 사람은 읽기만 | 체인 프로젝트가 바뀔 때 |
| **환경 선언** | 이 망이 어떻게 생겼나 | 테스트 작성자 | 테스트를 만들 때 |
| **케이스** | 무엇을 검사하나 | 테스트 작성자 | 테스트를 만들 때 |

**`presets/chain/*.yaml` 은 환경 선언으로 흡수된다.** 지금 두 겹인 선언이 한 겹이 되고,
`preset.Chain` 이라는 타입과 `upgrade.preset` 이라는 문법 자리가 함께 사라진다. 하드포크는
특별한 종류가 아니라 환경 선언이 표현하는 여러 구성 중 하나가 된다.

### 4.2 값의 우선순위는 한 줄

```
체인 정의  <  환경 선언  <  케이스 override  <  server-set(자원 배정)  <  CLI/MCP
```

늦게 오는 것이 이긴다. **이 줄을 확정한다 (2026-09-22).** workspace-config 는 이 줄에
없다 — 값이 아니라 **경로**를 정하므로 충돌 대상이 아니다.

server-set 이 케이스보다 뒤에 오는 이유는 4번 논의 그대로다. 정의서는 저장소에 남아 관리되는
파일이고, 실행하는 기계는 매번 다르며, 민감한 주소와 포트를 정의서에 적을 이유가 없다. 그래서
호스트와 포트는 자원 배정 단계에서 server-set 이 정한다. 선언이 적은 포트 값은 server-set 이
없을 때의 바닥값이다.

어휘 셋(`blueprint.Origin`·`nodeconfig.Layer`·`testengine.PlanOrigin`)은 이 한 줄로 모은다.

치르는 값을 적어 둔다. server-set 이 케이스보다 뒤이므로 **케이스는 특정 포트를 못 박지
못한다.** 포트를 고정해야 하는 테스트는 CLI 로 가야 한다. 그런 테스트가 생기면 이 줄이 아니라
그 테스트가 무엇을 검사하려는지를 먼저 본다 — 포트 번호에 의존하는 검사는 대개 포트가 아니라
다른 것을 보려던 것이다.

### 4.3 "망 전체 사실"과 "노드별 사실"을 가른다

D1 은 "노드별이냐 망 전체냐" 의 문제가 아니라 **어떤 사실이 망 전체의 것인가**를 안 정한
문제다.

| 망 전체 | 노드별 |
|---|---|
| chain id · network id · genesis · hardfork 일정 | 어떤 바이너리 · dialect(플래그 철자) · RPC namespace · 역할 |

지금 `nodeconfig.ChainOf()` 는 둘을 한 구조체에 섞어 담고, 그것을 config 는 노드의 plugin
에서, argv 는 망의 plugin 에서 채운다. 갈라 담으면 어느 쪽에서 채워야 하는지가 타입에 적힌다.

---

## 5. 이행 계획

앞의 둘은 결함이라 설계 결정을 기다리지 않는다.

| # | 무엇 | 게이트 |
|---|---|---|
| ~~**D-a**~~ | ~~`nodeconfig.Chain` 을 망 전체 사실과 노드별 사실로 가른다. network id 를 망의 chain id 에서 파생한다~~ | **완료 2026-09-22.** 혼합 바이너리 핸드오버에서 다섯 노드가 전부 8285 로 뜬다(라이브 확인) |
| ~~**D-b**~~ | ~~`ValidateUniform` 을 배선한다~~ | **완료 2026-09-22.** `nodeconfig.ValidateUniformNetworkID` 로 옮겨 조립 세 자리에 걸었다. 일부러 어긋낸 합성이 `build` 에서 `ChainBuildNodeCommandFailSplitNetwork` 로 막힌다(라이브 확인) |
| ~~**D-c**~~ | ~~`--chain-id` 가 network id 를 끌고 가는지 고정한다~~ | **완료 2026-09-22.** 합성 수준 테스트와 라이브 대조 둘 다. 옛 코드 8283 / 고친 뒤 4242 |
| **P-1** | 용어를 확정한다 — 체인 정의 / 환경 선언 / 케이스 | 문서와 타입 이름이 한 낱말만 쓴다 |
| **P-2** | `presets/chain/*.yaml` 두 개를 환경 선언으로 옮긴다. `upgrade.preset` 문법과 `preset.Chain` 타입을 없앤다 | 하드포크 케이스 넷이 그대로 돈다 |
| **P-3** | manifest 와 환경 선언의 소유권을 가른다. 중복 필드(`upgrade`·`network_id`)를 한쪽으로 모은다 | 같은 사실을 적는 자리가 하나다 |
| **P-4** | 우선순위 어휘 셋을 한 줄로 모은다 | 값마다 어느 단이 이겼는지 한 어휘로 읽힌다 |
| **P-5** | `suite run` 에 `--server-set` 을 단다 | 정의서 하나가 server-set 만 갈아 끼워 다른 기계에서 돈다 |
| **P-6** | 감시 테스트를 override 결과 기준으로 다시 쓴다 | 기본값을 의도적으로 바꿔도 실패하지 않고, 사슬을 거친 결과가 틀리면 실패한다 |
| **P-7** | preset 문서의 틀린 값과 `presets/chain/README.md` 를 고친다 | 문서가 적은 값이 실제로 쓰이는 값이다 |
| **P-8** | `manifest.network_id` 가 파생과 같으면 거부하는 래칫을 건다 | 세 체인의 현재 필드가 전부 걸린다(같은 값이므로), 지우면 통과한다 |

`presets/chain/wemix-upgrade-15.yaml` 은 그대로 둔다. 지금 부르는 데가 없을 뿐,
`tests/tc` 정의서를 훑을 때 연결될 자리다.

---

## 6. 코드 응집

선언을 읽는 코드가 지금 네 패키지에 흩어져 있다.

```
internal/preset           4파일   648줄   preset.Chain · preset.Key
internal/core/registry    8파일  1093줄   registry.Manifest · 패밀리 · 플러그인
internal/chains/external  1파일    75줄   외부 manifest 로드
internal/dsl             18파일  3268줄   env·case 문법과 병합
```

`preset.Chain` 과 `registry.Manifest` 가 `upgrade` 와 `network_id` 를 겹쳐 갖는다. P-2·P-3 이
끝나면 `internal/preset` 의 chain 쪽은 사라지고 키 쪽만 남는다. 그때 남는 것을 어디에 둘지
다시 본다 — 지금 정하면 아직 없는 모양을 배선하는 것이 된다.

---

## 7. 확정한 것 (2026-09-22)

| 무엇 | 확정 | 치르는 값 |
|---|---|---|
| 우선순위 줄 | §4.2 다섯 단. server-set 이 케이스보다 뒤 | 케이스가 포트를 못 박지 못한다 |
| 체인 정의 파일의 자리 | 지금 자리 유지. P-2 가 끝난 뒤 다시 본다 | 선언이 두 트리에 나뉜 상태가 그만큼 길어진다 |
| `manifest.network_id` | 남기되 파생과 같은 값이면 래칫이 거부 (P-8) | 규칙이 한 줄 는다 |

### P-1 — 낱말 (확정 2026-09-22)

| 개념 | 낱말 | 파일 | 누가 쓰나 |
|---|---|---|---|
| 체인 구현이 무엇인가 | **chain-manifest** | `internal/chains/<id>/manifest.json` (embed) | 도구가 안다 |
| 이 망이 어떻게 생겼나 | **chain-preset** | `presets/chain/<id>.json` | 테스트 작성자 |
| 무엇을 검사하나 | **case** | `tests/tc/**/*.json` | 테스트 작성자 |
| 노드 신원 | **key-preset** | `presets/keys/` | 테스트 작성자 |

`env` 와 `profile` 과 (체인 쪽) `preset` 은 없어진다. `profile` 은 이미 죽은 낱말이라 주석의
잔재와 없어진 명령을 부르는 e2e 둘에만 남아 있다.

**chain-preset 은 종속성이 없어야 한다.** 논리적 이름만 적고, 구체적인 바이너리와 기계는
chain-manifest 와 workspace-config 가 답한다. 지금 26개 중 8개가 `${GWBFT_BIN:-gwbft}` 로
구체적인 파일 이름을 적고 있는데, 그 일을 할 자리는 `WorkspaceConfig.BinaryAliases` 에 이미
있다. 이 기준이 그 8개를 걸러 낸다.

### P-1b — 개명이 닿는 곳 (잰 것)

| 무엇 | 지금 | 뒤 | 건수 |
|---|---|---|---|
| 케이스의 참조 키 | `"env": "<id>"` | `"chainPreset": "<id>"` | 209 |
| 선언의 kind | `"kind": "env"` | `"kind": "chain-preset"` | 26 |
| 파일 이름·자리 | `tests/tc/env/<id>.env.json` | `presets/chain/<id>.json` | 26 |
| Go 식별자 | `EnvV2`·`KindEnv`·`findEnvFile`·`UseEnv`·`InlineEnv`·`ReadFilesWithEnv`·`envRef` | 대응하는 chain-preset 이름 | 약 104곳 |
| 스키마 | `const: "env"`, 속성 `env` | `chain-preset`, `chainPreset` | 3곳 |
| CLI | `suite run --env` | `--chain-preset` | 1 |

**해석 규칙.** 지금은 케이스 파일의 디렉터리에서 위로 올라가며 `<dir>/<id>.env.json` 과
`<dir>/env/<id>.env.json` 을 찾는다. 그 걸음은 유지하고(스위트가 자기 preset 을 곁에 둘 수
있다) **저장소 루트의 `presets/chain/<id>.json` 을 마지막 자리로 더한다.** 케이스는 id 만
부르므로 파일이 옮겨 가도 케이스의 값은 바뀌지 않는다 — 바뀌는 것은 키 이름뿐이다.

게이트는 209개 케이스가 그대로 `validate` 를 통과하는 것이다(지금 209/209).

---

## 8. 체인 정의를 `presets/` 로 옮기는 것에 대하여

옮기면 선언이 한 트리에 모이고, 테스트를 쓰는 사람이 체인 정의와 환경 선언을 나란히 본다.

치르는 값은 셋이다. 첫째, `go:embed` 가 `..` 를 못 쓰므로 `presets/` 가 Go 패키지가 되어야
한다. 둘째, manifest 가 그것을 쓰는 플러그인 코드(`internal/chains/<id>/*.go`)에서 떨어진다.
셋째, 체인을 코드 없이 추가하는 길은 이미 `internal/chains/external` 이 열어 두었으므로,
옮겨서 새로 얻는 기능은 없다.

그래서 **환경 선언을 합치는 것(P-2)을 먼저 하고, 체인 정의의 이사는 P-2 가 끝난 뒤에 다시
본다(확정 2026-09-22).** 합치고 나면 `presets/` 밑에 무엇이 남는지가 달라지고, 그때 판단이
지금보다 싸다.
