# 응집도 후보 — 쓰임으로 다시 읽기 (2026-09-21) — [측정]

> **[측정]** 잰 것만 적는다. 판단이 필요한 자리는 판단이 필요하다고 적고 결론을 내리지 않는다.
>
> 재는 법: `go/parser` 로 `internal/` · `cmd/` 의 비-테스트 파일을 전부 읽어 선언 1,934개와
> 타입 677개를 뽑고, 선언마다 **무엇을 하는 것인지**(이름이 여는 동사 · 서명에 나오는 구체
> 타입 · 본문이 직접 만지는 파일과 코덱)를 기록했다. 산출물은 세션 스크래치패드의
> `intent.json` 이고, 아래 수치는 전부 그것과 `grep` 으로 재확인한 값이다.

## 0. 왜 이 문서가 있나

`preset` 정리가 끝난 방식이 기준이 됐다. 그 일이 잘 풀린 이유는 이름을 고쳐서가 아니라
**질문을 하나로 잡아서**다 — "이 문서를 누가 읽는가". 답이 "여러 모듈이 저마다 다른 이름으로"
였고, 읽기를 한 자리에 모으니 preset 을 더하는 일이 한 모듈 고치는 일이 됐다.

같은 질문을 나머지에 던졌다. **선언이 아니라 쓰임을 본다** — 문서가 무엇을 담고 있는지가
아니라 코드가 그중 무엇을 읽는지를 센다. 세 후보가 나왔고, 성격이 서로 다르다.

| | 무엇 | 성격 | 크기 |
|---|---|---|---|
| **A** | 한 원장에 두 종류가 섞여 있다 | 구조. 지금 깨지지 않는다 | 중 |
| **B** | `preset.Chain` 은 쓰임으로 보면 다른 것이다 | **이름 결정을 다시 봐야 한다** | 소~중 |
| **C** | `-Source` 접미사가 반대 성격 둘을 덮는다 | 어휘 | 소 |

---

## A. 단계 원장에 두 종류가 섞인다

### 잰 것

`markStep` 호출이 **18곳**이고 기록되는 단계 이름이 **17개**다. 그런데 그 17개는 성질이 둘로
갈린다.

| | 단계 이름 | 어디에 선언돼 있나 |
|---|---|---|
| **조립 사다리** 9 | `new` `place` `keys` `genesis` `config` `build` `deploy` `init` `start` | `composeNeeds`(steps_compose.go:412) 가 순서까지 안다 |
| **운영 기록** 8 | `stop` `rm` `restart` `hardfork` `cross-fork` `start-node` `stop-node` `swap-node` | **아무 데도 없다** |

둘 다 `w.state.Steps` 라는 **한 map** 에 들어간다. 그리고 그 map 에서 "내가 뜻하는 부분집합"
을 고르는 규칙이 **세 자리에 흩어져 있다.**

- `composeNeeds` — 조립 8단계의 선행 관계 (`steps_compose.go:412`)
- `upStepNames` — `firstUndone` 이 순회하는 목록 (`verbs_resume.go:116`)
- 이름을 직접 쓰는 곳 — `st.Steps["start"].Done` (`steps_preflight.go:22`)

### 지금 깨지지 않는 이유

읽는 쪽이 전부 **자기가 뜻하는 이름을 명시**하기 때문이다. `firstUndone` 은 `upStepNames` 만
돌고, preflight 는 `"start"` 만 본다. 운영 기록 8개는 아무 독자도 집지 않으므로 지나간다.

**그래서 이것은 버그 보고가 아니다.** 다만 다음 사람이 `markStep("foo")` 를 하나 더 부를 때
자기가 어느 종류를 더한 것인지 알 방법이 없고, `composeNeeds` 는 모르는 키에 nil 을 돌려주므로
**모든 선행 조건이 공허하게 충족된다**(direction.md §10 이 이미 적어 둔 위험이다).
래칫 `TestVerbNeedsStepsAreRealSteps` 는 동사가 **선언한** 단계만 검사하고, `markStep` 이
**기록하는** 단계는 보지 않는다.

### 고친다면

세 가지 중 하나이고, 값과 비용이 다르다.

1. **어휘를 둘로 나눈다** — `markStep`/`markEvent`, 또는 `Steps`/`Events` 두 map. 가장 정직하지만
   기록 형식이 바뀌므로 워크스페이스 호환을 봐야 한다.
2. **이름만 선언한다** — 운영 기록 8개를 `opNames` 같은 목록에 적고, 래칫이 "기록되는 단계는
   두 목록 중 하나에 있어야 한다" 를 건다. 싸고, 섞임 자체는 남는다.
3. **둔다** — 지금 깨지지 않으므로. 다만 §A 를 읽은 사람이 있어야 한다.

> **단점.** 1번은 되돌리기 어렵고 이득이 "읽기 쉬움" 이다. 이 저장소는 소비자 없는 구조를
> 미리 배선하다 두 번 되돌렸다(direction.md §5). 2번이 그 교훈에 맞는 크기다.

### 곁가지 — 파일 이름은 실제로 성격을 말한다

처음에 `chainsetup` 34개 파일의 이름이 성격을 반만 말한다고 봤는데, **기준을 잘못 잡은
것이었다.** `steps_` 를 "Workspace 메서드" 로 읽으면 12개 파일이 어긋나 보인다. `markStep` 으로
읽으면 어긋나는 것은 넷뿐이고(`node_ops` `new` `workspace` `verbs_hardfork`), 그 넷은 전부
**운영 기록** 쪽이다. 즉 파일 이름은 §A 의 갈림을 이미 따르고 있었고, 갈림 자체가 적혀 있지
않았을 뿐이다. 이름을 고칠 일이 아니다.

---

## B. `preset.Chain` 은 쓰임으로 보면 하드포크 일정표다

### 잰 것

`presets/chain/wemix-upgrade.yaml` 은 최상위 절이 **11개**다. 코드가 그중 무엇을 읽는지 셌다.

| 절 | `internal/preset` 밖의 프로덕션 소비자 |
|---|---|
| `upgrade` | **4건** — 전부 `testengine/genesis_decl.go`, 그것도 `AtFork` 와 `ForkBlock` **두 스칼라만** |
| `name` `chains` `roles` `identities` `producers` `validators` `data` `ports` `nodes` | **0건** |

`preset.Chain` 을 **인자로 받는 함수가 저장소에 하나도 없다.** 타입의 메서드는
`PlanOrderOrDefault()` 하나이고 **호출자가 0**이다(처음에는 "프로덕션 0" 이라고 적었는데, 테스트도 안 부른다. 2026-09-21 에 지웠다). `poa.EnvFromPreset` 은
`preset.Chain` 이 아니라 하위 구조 `preset.Governance` 를 받고, **호출자가 테스트 하나뿐**이다.

### 왜 이것이 이름 결정을 다시 보게 하나

`direction.md` 는 이 타입의 이름을 **세 번** 정했다(§8 `upgrade.ChainPreset` → §13
`chainpreset.Preset` → §14 `preset.Chain`). 세 번 모두 근거가 같다. §8 이 그것을 이렇게 적는다.

> 실측이 그것을 보인다: `wemix-upgrade.yaml` 의 최상위 키 11개 중 하드포크 고유는 **`upgrade`
> 하나**이고, 나머지 열은 다른 종류의 preset 도 똑같이 적을 **일반 체인 설정**이다.

그 실측은 **문서를 센 것**이다. 같은 자리를 **쓰임으로** 세면 정반대가 나온다 — 열 절은 아무도
읽지 않고, 읽히는 것은 `upgrade` 하나다. 문서가 담은 것으로는 chain preset 이고, 코드가 읽는
것으로는 하드포크 일정표다.

**어느 쪽이 맞는지는 이 문서가 정하지 않는다.** 정하려면 답해야 하는 질문이 있다.

- 열 절이 **앞으로** 읽힐 것인가. `chains`·`roles`·`identities`·`producers` 는 핸드오프 구성이
  `chainsetup` 으로 흡수되기 전에 쓰이던 자리로 보인다(`consensus/upgrade` 가 1,949 → 164줄로
  줄고 없어진 그 작업이다). 흡수된 뒤에도 문서에 남아 있는 것이라면 **문서가 코드보다 낡은
  것**이고, 이름이 아니라 문서를 줄여야 한다.
- 아니면 그 열 절이 **다음 종류의 preset** 을 위한 자리인가. §8 의 논거가 그것이다. 그렇다면
  이름은 지금이 맞고, 대신 **읽히지 않는다는 사실이 어딘가 적혀 있어야 한다.**

### 딸린 것 — 밖에서 부르지 않는 표면 셋

같은 측정에서 `internal/preset` 이 내놓지만 패키지 밖 호출자가 0인 것이 셋 나왔다.

```
KeystoreAccount   LoadKeyPresetWithAccountsAt   NodeKeyAt
```

(처음에는 다섯이라고 적었는데 둘은 `TestGoldenPreset`·`TestPreset` 이었다. Go 테스트 함수
이름을 exported 함수로 센 것이라 표면이 아니다.) 방금 정리를 끝낸 모듈이므로, 흡수 과정에서
남은 것인지 앞으로 쓸 것인지 확인하기 좋은 시점이다.

> **단점.** B 를 건드리면 §8·§13·§14 의 판단을 네 번째로 뒤집는 것이 된다. 그 자체가 비용이고,
> 문서에 "또 바뀌었다" 가 한 줄 더 붙는다. **먼저 답해야 하는 것은 이름이 아니라 "열 절은
> 살아 있나"** 이고, 그 답이 나오면 이름은 따라온다.

### 답 (2026-09-21) — 문서가 코드보다 낡았다

"열 절이 앞으로 읽힐 것인가" 를 이력으로 물었다. 답은 **아니다**이고, 근거는 짐작이 아니라
커밋이다.

먼저 지금 누가 읽는지 다시 셌다. `preset.Chain` 을 담는 변수는 저장소에 다섯 곳뿐이라
전수로 셀 수 있다. 세 층으로 갈린다.

| 층 | 절 | 읽는 곳 |
|---|---|---|
| 프로덕션이 읽는다 | `upgrade` | `testengine/genesis_decl.go` 4곳. 그것도 `AtFork` 와 `ForkBlock` 두 스칼라만 |
| 테스트만 읽는다 | `roles` `identities` `producers` `validators` | `internal/preset` 의 테스트 셋과 `poa/preset_env_test.go`. 전부 문서를 문서 자신에게 맞춰 보는 검사다 |
| 아무도 읽지 않는다 | `chains` `data` `ports` `name` | 0곳 |

여기에 yaml 에만 있고 **구조체에 필드조차 없는** 키가 둘 더 있다. `description` 과 `nodes`
(`verbosity`·`gcmode`·`cache`)다. 디코딩되지 않으므로 무엇을 적어도 아무 일도 안 난다.

이력을 보면 왜 이렇게 됐는지가 나온다. `git log -S` 로 각 절을 마지막으로 읽은 커밋을
찾으면 `chains`·`ports`·`validators` 가 전부 #419(`6ed4b37c`, 하드포크 핸드오프 흡수)와
#422 에서 사라진다. 그 전에는 `internal/consensus/upgrade/` 가 읽고 있었다.

```
upgrade.go:114   prof.Roles.Producers + prof.Roles.Validators
upgrade.go:144   prof.Chains.From.Binary
handoff.go:847   h.Profile.Chains.To.NodekeyDir
handoff.go:926   prof.Producers.Governance
handoff.go:952   prof.Ports.BaseP2P
profile.go:145   p.Validators.ExtraData
```

#419 가 그 합성기를 지웠다(1,949줄 → 151줄). 지금은 평범한 합성 경로가 같은 사실을 다른
데서 얻는다. 노드 수는 DSL 의 topology 에서, 바이너리는 체인 선언에서, 포트는 서버 셋과
portplan 에서, validator 는 키 셋에서 온다. **열 절은 다음 종류의 preset 을 위한 자리가
아니라, 지워진 합성기가 남긴 자국이다.**

`data` 와 `name` 은 한 술 더 뜬다. `git log -S 'Data.Directory'` 와 `-S 'prof.Name'` 이
저장소 전체 이력에서 **한 건도** 안 나온다. 태어나서 한 번도 읽힌 적이 없다.

그러므로 §8 의 전제 — "나머지 열은 다른 종류의 preset 도 똑같이 적을 일반 체인 설정" —
는 미래에 대한 짐작이었고, 이력이 그 짐작을 반박한다. 이름을 정하는 근거로 쓸 수 없다.

### 이번에 한 것

딸린 것부터 처리했다. 호출자 0인 셋(`KeystoreAccount`·`LoadKeyPresetWithAccountsAt`·
`NodeKeyAt`)은 죽은 코드가 아니라 **패키지 안에서만 쓰이는데 밖으로 내놓은 것**이었다.
`internal/preset` 밖에서도, `package preset_test` 에서도 부르는 곳이 없다. 소문자로 내려
공개 표면을 셋 줄였다. 동작 변화 0.

### 남은 결정 — 사람이 정해야 한다

두 가지가 남고, 둘 다 코드가 아니라 판단이다.

1. **골든 preset 에서 죽은 절을 지울 것인가.** `chains`·`data`·`ports`·`name`·`nodes`·
   `description` 은 지워도 코드가 안 바뀐다. 다만 이 파일은 핸드오프 환경을 서술하는
   문서이기도 해서, 읽는 사람에게는 정보다. 지운다면 대신 "이 환경은 이런 모양이었다" 를
   어디에 적을지 같이 정해야 한다.
2. **이름을 바꿀 것인가.** 쓰임으로는 하드포크 일정표가 맞다. 하지만 §8·§13·§14 를 네
   번째로 뒤집는 일이고, 이 저장소는 "읽기 쉬움" 을 근거로 배선했다가 두 번 되돌렸다.
   지금 새로 생긴 근거는 이력이지 취향이 아니라는 점이 이전 세 번과 다르다.

---

## C. `-Source` 가 반대 성격 둘을 덮는다 — 완료 (2026-09-21)

### 처음 잰 것이 좁았다

처음에는 접미사 `-Source` 를 가진 **exported 타입 9개**만 세고, 그중 꼬리표는
`testengine.PlanSource` 하나라고 적었다. 그래서 "한 줄이면 끝나고 값도 크지 않다" 고 닫았다.
**타입만 세고 필드를 안 센 것이 잘못이었다.** 다시 재니 꼬리표 쪽이 여섯이다.

| 성격 | 어디 | 값의 예 |
|---|---|---|
| **만드는 것** (지시, 일 전에 주어진다) | `keyring.Source`·`FileSource`·`MnemonicSource`·`PrivateKeySource`·`RandomSource`·`PasswordSource` · `store.KeySource` · `genesis.Source`·`PresetSource` · `poa.GenesisSource` · `keysSource` 문자열 5곳 · DSL 정의서의 `"source"` | `keyPreset` · `generate` · `declared` · `rpcCall` |
| **적어 두는 것** (영수증, 일 끝난 뒤 남는다) | `blueprint.Source` · `testengine.PlanSource` · `State.PortSource` · `resource.Pool.Source` · `operation.SetOut.Source` · `NetPoolOut.Source` | `inventory` · `harness` · `server-set.yaml` · `explicit` |

### "지금 헷갈릴 자리는 없다" 도 틀렸다

두 뜻이 한 파일에서 만나는 곳이 있다. `internal/core/keyring/operation/import.go` 는
지역변수 `source` 가 영수증인데, 여덟 줄 아래 메서드 `source()` 는 `keyring.Source` 를
만들어 돌려준다. 같은 낱말, 같은 파일, 반대 뜻이다.

JSON 에서도 만난다. `State.Request *NetUpIn` 이라 워크스페이스 파일 하나가
`request.keysSource: "generate"`(명령)와 `portSource: "server-set.yaml"`(기록)을 나란히
담는다. 읽는 사람이 구분할 방법이 없다.

### 고친 것

**코드가 이미 답을 반쯤 쓰고 있었다.** blueprint 의 상수는 `FromBlueprint`·`FromDefault` 로
"from" 을 쓰고, testengine 의 필드 이름은 `From` 이다. 타입 이름만 `Source` 로 남아
`FromDefault Source = "default"` 한 줄에 충돌이 그대로 보였다.

영수증 쪽 여섯을 `Origin` 으로 바꿨다. 만드는 쪽은 뜻이 맞으므로 건드리지 않았다.

- `blueprint.Source` → `Origin`, `ResolvedNetwork.Sources` → `Origins` (`json:"origins"`)
- `testengine.PlanSource` → `PlanOrigin`, 상수 셋도 `Origin…` 으로. plan.json 의 키는
  이미 `"from"` 이라 JSON 은 안 바뀐다
- `State.PortSource` → `PortOrigin` (`json:"portOrigin"`)
- `resource.Pool.Source` → `Origin`, `builtinSource` → `builtinOrigin`
- `operation.SetOut.Source` → `Origin`, 이 패키지의 영수증 지역변수도 `origin` 으로
- `NetPoolOut.Source` → `Origin` (`json:"origin"`), MCP 응답 키도 `origin`

건드리지 않은 것도 분명히 적는다. 인터페이스 아홉은 생산자라는 뜻이 맞다. `keysSource`
문자열 다섯 곳과 `tests/tc` 정의서 209개가 쓰는 `"source": "rpcCall"` 도 "어느 방식으로
가져오나" 라는 지시라서 그대로다. 결과로 사용자가 보는 `source` 는 전부 "어느 방식" 이고,
`origin` 은 전부 "누가 정했나" 다.

> **치른 값.** JSON 키 세 개가 바뀐다(`sources`·`portSource`·`source`). 셋 다 진단용이고
> 판단에 쓰이지 않는다 — `PortOrigin` 은 `steps_place.go` 가 쓰기만 하고 읽는 코드가 없다.
> 다만 이전에 만든 워크스페이스를 resume 하면 `portOrigin` 이 빈 채로 보인다. 값 하나가
> 빌 뿐 동작은 그대로다.

## 순서 제안

| # | 무엇 | 왜 이 자리인가 |
|---|---|---|
| ~~1~~ | ~~**B 의 앞부분** — 열 절이 살아 있는지 읽는다~~ | **완료 2026-09-21.** 답은 "아니다" — 지워진 합성기가 남긴 자국이다. B 의 §답 |
| ~~2~~ | ~~**B 의 딸린 것** — 호출자 0인 셋~~ | **완료 2026-09-21.** 죽은 게 아니라 안에서만 쓰였다. 소문자로 내렸다 |
| 3 | **A 의 2안** — 운영 기록 8개를 목록으로 선언하고 래칫을 건다 | 싸고, 다음 사람이 어느 종류를 더하는지 알게 된다 |
| ~~4~~ | ~~**C**~~ | **완료 2026-09-21.** "한 줄" 이 아니었다 — 타입만 세고 필드를 안 세서 여섯 배로 작게 잡았다 |

**A 의 1안(어휘를 둘로)과 B 의 개명은 여기 넣지 않았다.** 둘 다 1번의 답이 나오기 전에는
근거가 "읽기 쉬움" 뿐이고, 이 저장소는 그 근거로 배선했다가 두 번 되돌렸다.
