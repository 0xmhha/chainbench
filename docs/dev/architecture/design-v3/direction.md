# 리팩토링 설계 v3 — 방향 — [제안]

> **[제안] (2026-09-21).** 근거는 전부 [`measurement-2026-09-21.md`](measurement-2026-09-21.md) 에
> 있고, 이 문서는 그 위에서 무엇을 할지만 적는다. 닫힌 세 계획을 이어받지 않는다.

## 1. 무엇을 좋은 코드로 삼는가

**코드를 글처럼 읽어서 내용이 파악되면 좋은 코드다.** 이 기준을 설계 판정에 그대로 쓴다.
그러면 판정이 취향이 아니라 질문 셋이 된다.

1. **한 낱말이 한 가지를 가리키는가.** 부르는 자리마다 뜻이 달라지면 문장이 안 된다.
2. **동작이 자기 선행 조건을 말하는가.** "이 상태에서 이것을 한다" 가 코드에 적혀 있지 않으면
   읽는 사람이 호출 순서를 외워야 한다.
3. **한 파일을 열었을 때 그 파일이 하는 일이 제목으로 설명되는가.**

### 모듈의 역할 경계 (전제, 바꾸지 않는다)

- **`app`** — CLI 와 MCP 로 들어오는 **진입점**이다. chainbench 를 데몬으로 만들더라도 `main`
  이 들어가는 자리가 `app` 이다.
- **`chainsetup`** — 체인을 구성·설정해 **도는 체인을 만드는 일 전반**을 책임진다.
- **`testengine`** — **테스트 수행 lifecycle** 을 책임진다: 환경을 구성하고, 테스트를 수행하고,
  결과 레포트를 쓴다.
- `testengine` 이 환경을 구성할 때 **그 구성을 담당하는 것이 `chainsetup`** 이다.

이 셋은 합치지 않는다. 레이어도 손대지 않는다 — 측정 §2 가 결함 없음을 보인다.

---

## 2. 낱말을 하나로 (가장 먼저)

`preset` 이 다섯 가지를 가리킨다(측정 §3). **뜻마다 다른 낱말을 준다.**

| 지금 | 제안 | 왜 |
|---|---|---|
| `presets/chain/*.yaml` + `upgrade.Profile` + `LoadProfile` | ~~preset 으로~~ **완료 (2026-09-21): `upgrade.ChainPreset` · `LoadChainPreset`** | 파일 위치가 `presets/` 인데 타입이 `Profile` 이라 문서와 코드가 다른 말을 했다 |
| `dsl.UpgradeV2.Profile` 필드 | ~~삭제~~ **완료 (2026-09-21)** | 정의서 209개 중 쓰는 것이 0개였다. 지우면서 스키마의 `upgrade` 블록도 실제 구조에 맞췄다 — `required: [profile, template]` 에 `additionalProperties: false` 라 **실제로 쓰는 `preset` 을 거부하는 상태**였고, `fork`·`at`·`from`·`to`·`style`·`carry` 여섯 필드가 통째로 빠져 있었다 |
| `profiles/` 디렉터리 | ~~삭제~~ **완료 (2026-09-21)** | 측정 §4 |
| `presets/keys/` | **`keys/fixture/`** 로 | 키는 골든 설정이 아니라 테스트 픽스처다. `keyring.Preset` 타입도 같이 본다 |
| 메인넷 워크리스트의 "preset" (env 선언) | **env 로 통일** | 코드·파일·스키마가 이미 `env` 라 부른다. 문서만 다르다 |

경로 오기 둘도 같이 고친다: `v2.schema.json:142` 의 *"profiles/\*.yaml"*, 그리고
`chain-handover-2026-09-12.md:379` 의 `profiles/wemix-upgrade-15.yaml` 링크.

**단점.** `presets/keys` → `keys/fixture` 는 env 파일 26개의 `keys.nodekeys.ref` 와 코드의
`KeysDir` 기본값을 건드린다(참조 71곳). 이름 하나 때문에 넓게 퍼지므로, 나머지 넷을 먼저 하고
이것만 따로 떼어 판단해도 된다.

---

## 3. 쓰지 않는 자리를 지운다

`state/` 와 `profiles/` 를 지우고, `.gitignore` 에서 그것들을 위한 **13줄**을 걷어낸다.
`README.md:306`·`CONTRIBUTING.md:89·111` 의 서술도 같이 지운다 — 지금은 문서가 없는 기능을
있는 것처럼 안내한다.

**단점.** `state/` 의 `.gitignore` 9줄은 예전 셸 시절 산출물 이름이다. 지우면 누군가 옛
스크립트를 돌렸을 때 산출물이 git 에 잡힌다. 그 스크립트가 아직 있는지 먼저 확인해야 한다.

---

## 4. 상태 주도를 조립 밖으로 넓힌다

`composeNeeds` 는 **이미 옳은 모양**이다(측정 §5). 선행 조건을 한 곳에 선언하고, 상태 필드가
아니라 **기록된 단계**를 읽는다. 문제는 덮는 범위다 — 동사 38개 중 선행 조건을 선언하는 것이
6개다.

그래서 새 기계를 만들지 않는다. **같은 표를 조립 밖으로 넓힌다.**

지금 표는 "어떤 단계가 먼저 돌았는가" 만 묻는다. 운영 동사는 그것으로 부족하다 — `Stop` 은
"start 가 돌았나" 가 아니라 **"지금 도는 노드가 있나"** 를 묻고, `Rm` 은 **"멈춰 있나"** 를
묻는다. 그러니 선행 조건의 어휘를 둘로 나눈다.

- **단계 조건** — 그 단계가 기록되었는가. 지금 `composeNeeds` 가 하는 일.
- **실행 조건** — 노드가 도는가 / 멈췄는가. `state.Nodes[].PID` 가 이미 답을 갖고 있고,
  `phases.go` 의 `checkVacant` 가 이미 한 자리에서 그것을 묻는다.

동사마다 둘을 선언하게 하고, 선언하지 않은 동사를 래칫이 세어 준다. 지금 6/38 이고, 그 수가
줄어드는 것이 진척이다.

**단점.** 동사 32개를 한 번에 옮기면 큰 PR 하나가 된다. 그리고 몇몇은 **일부러** 조건이 없다 —
`Logs` 는 죽은 노드의 로그도 보여 줘야 하고, `Have`·`Dir`·`State` 는 조회다. 그래서 래칫의 수는
"0 이 목표" 가 아니라 **"선언하지 않은 것에는 이유가 적혀 있다"** 여야 한다. 그 형태는
`arch` 에 선례가 있다(`TestEveryFieldAProducerOwnsIsFilledOrExplained`).

---

## 5. 큰 덩어리는 마지막에, 요구가 생겼을 때

`chainsetup` 8,406줄·exported 함수 169개, `app` exported 타입 122개·fanOut 22 다(측정 §1).
크지만 **레이어는 어기지 않고 역할 경계도 §1 대로 옳다.** 그래서 지금 쪼갤 근거는 "크다" 뿐이다.

닫은 계획이 남긴 교훈이 여기 그대로 적용된다 — 이 저장소는 소비자 없는 구조를 미리 배선하다
두 번 되돌렸다(`Transport` 타입, 객체형 참조). **요구가 생겼을 때 연다.** §2·§3·§4 를 끝내고
나면 `chainsetup` 을 읽는 사람이 실제로 어디서 막히는지가 드러나고, 그때 자르는 선이 정해진다.

---

## 6. 순서와 크기

| # | 무엇 | 크기 | 되돌리기 |
|---|---|---|---|
| ~~1~~ | ~~`dsl.UpgradeV2.Profile` 삭제 + 스키마·문서 경로 오기 정정~~ | — | **완료 2026-09-21** |
| ~~2~~ | ~~`profiles/`·`state/` 삭제 + `.gitignore` + README·CONTRIBUTING~~ | — | **완료 2026-09-21** |
| ~~3~~ | ~~`upgrade.Profile`/`LoadProfile` 개명~~ | — | **완료 2026-09-21** |
| ~~4~~ | ~~실행 조건 어휘 + 동사 이동 + 래칫~~ | — | **완료 2026-09-21** |
| ~~5~~ | ~~남은 동사를 옮긴다~~ | — | **부분 완료 2026-09-21 (§12)** |
| ~~6~~ | ~~`presets/keys` → `keys/fixture`~~ | — | **취소 (2026-09-21).** 갈래가 체인/키 둘이면 `presets/keys` 은 이미 정확한 이름이다 |
| 7 | 큰 덩어리 | — | 요구가 생기면 |
| 8 | 응집도 후보 A·B·C | — | **측정 완료 (2026-09-21)** — [`cohesion-candidates-2026-09-21.md`](cohesion-candidates-2026-09-21.md) |

1·2 는 서로 독립이고 오늘 끝난다. 3 은 1 뒤다. 4 가 이 설계의 본체다.

**8 은 preset 이 끝난 뒤에 같은 질문을 나머지에 던진 결과다.** preset 이 잘 풀린 이유가 이름을
고쳐서가 아니라 "이 문서를 누가 읽는가" 하나로 질문을 잡아서였으므로, **선언이 아니라 쓰임을**
세어 후보 셋을 찾았다. 그중 B 는 §8·§13·§14 가 세 번 정한 이름의 근거를 다시 보게 한다 —
그 근거는 문서가 담은 절을 센 것이고, 코드가 읽는 절을 세면 결과가 뒤집힌다.

---

## 7. 1·2 를 하면서 드러난 것 (2026-09-21)

- **스키마의 `upgrade` 블록은 `profile` 하나가 아니라 통째로 낡아 있었다.** `required: [profile,
  template]` + `additionalProperties: false` 였으므로, 강제되는 자리였다면 지금 쓰는 `preset`
  선언을 전부 거부했을 것이다. 강제되지 않아서 **아무도 몰랐다.** 스키마↔파서 동기화 테스트
  (`TestSchemaV2MatchesParsedFields`)는 `envSpec`·`caseSpec` 의 **최상위 속성만** 본다. 중첩
  객체는 사정거리 밖이다 — 다음에 갚을 빚으로 적어 둔다.
- **`chainbench remote` 명령은 없다.** `profiles/remote-example.yaml` 이 `chainbench remote add`
  사용법을 안내하고 있었는데, `root.go` 가 등록하는 27개 그룹에 `remote` 가 없다.
- **`scripts/merge_profile.py` 는 호출자가 0이다.** `lib/profile.sh` 에서 추출했다고 적혀 있는데
  `lib/` 자체가 없고, `profiles/` 와 `state/local-config.yaml` 을 읽는다 — 둘 다 이번에 지웠다.
  같이 지웠다.
- **README·CONTRIBUTING 의 트리에 없는 패키지 둘이 더 있었다** — `internal/netmap/` 과
  `internal/testkit/`. 같은 문단이라 함께 걷어냈다.
- **아직 남은 고아 둘**: `scripts/extract_json.py` · `scripts/json_backend.py` 도 외부 참조가
  0건이다. `profiles`/`state` 와 무관해 이번 범위 밖으로 두었다.

## 8. 3번을 하면서 정한 이름 (2026-09-21)

**preset 은 두 갈래다: 체인에 대한 것과 키에 대한 것.** 이름은 그 갈래를 말해야 한다.

> **이 표는 §14 가 대신한다 (2026-09-21).** 아래 이름 넷 중 지금 코드에 있는 것은 하나도 없다 —
> §13 이 패키지를 옮기고 §14 가 갈래를 한 모듈로 모으면서 전부 바뀌었다. 여기 남겨 두는 것은
> **그때 무엇을 정했는지의 기록**이고, 지금 이름을 찾는 사람은 §14 의 표를 봐야 한다.

| 갈래 | 문서 | 타입 | 읽는 함수 |
|---|---|---|---|
| **체인** | `presets/<종류>/*.yaml` (지금은 `hardfork` 하나) | ~~`upgrade.ChainPreset`~~ | ~~`upgrade.LoadChainPreset`~~ |
| **키** | `presets/keys/` | ~~`keyring.Preset`~~ | ~~`store.LoadPreset`~~ |

**`HardforkPreset` 으로 갔다가 되돌렸다.** 종류(hardfork)로 이름을 좁혔는데, 그럴 이유가 없다 —
하드포크는 체인 설정 preset 의 **한 종류**일 뿐이고 다른 종류가 더 생긴다. 실측이 그것을 보인다:
`wemix-upgrade.yaml` 의 최상위 키 11개 중 하드포크 고유는 **`upgrade` 하나**이고, 나머지 열
(`chains`·`roles`·`identities`·`producers`·`validators`·`data`·`ports`·`nodes`)은 다른 종류의
preset 도 똑같이 적을 **일반 체인 설정**이다. 디렉터리도 이미 `presets/<종류>/` 다.

되돌리면서 내 반대 근거 둘도 틀렸음을 확인했다. "두 체인을 부르니 단수형 `chain` 이 맞지 않는다"
는 **어느 체인을 언급하는가(카디널리티)를 어떤 갈래인가(범주)와 섞은 것**이다. `chains:` 는
이 문서의 필드 이름 그대로다. "`chainPreset` 은 env 선언에 어울리므로 겹침을 한 칸 옮긴다" 도
성립하지 않는다 — env 선언은 코드·파일·스키마가 이미 `env`(`kind: "env"`)라 부르고, §2 가
문서 쪽 표기도 `env` 로 통일하자고 적고 있다.

**다음 차례는 키 쪽의 비대칭이다.** 체인 쪽은 갈래를 이름에 달았는데 키 쪽은 `Preset`·
`LoadPreset` 으로 안 달았다. 6번에서 `presets/keys` 을 옮길 때 `KeyPreset`·`LoadKeyPreset` 으로
맞추면 둘이 짝이 된다.

## 9. 3번을 하면서 드러난 것

- **README 가 없는 명령을 안내하고 있었다.** `chainbench upgrade run --profile …` 과
  `upgrade genesis` 인데, `root.go` 가 등록하는 그룹에 `upgrade` 가 없다(있는 것은 `hardfork`).
  낱말만 바꾸면 거짓이 남으므로, 실제로 도는 경로(정의서 + `suite run`, 그리고
  `chainbench hardfork --workspace-dir …`)로 다시 썼다.
- **e2e 테스트 셋이 같은 죽은 명령을 부른다** — 세 파일이 `"upgrade", "run", "--profile", …`
  를 넘긴다. `e2e` 태그 뒤에 있고 환경변수가 없으면 건너뛰므로 **컴파일은 되고 아무도 실패를
  보지 못한다.** 이번 범위 밖으로 두었다 — 고치는 일은 이름이 아니라 없어진 명령의 문제다.
  (이후 `upgrade_run_e2e_test.go` 는 #422 에서 지웠다. `cmd/chainbench/upgrade_data_migration_e2e_test.go`
  와 `cmd/chainbench/upgrade_gov_ncp_lifecycle_e2e_test.go` 는 아직 그대로 죽은 명령을 부른다.)
- **여섯째 `profile` 이 있다** — `internal/accounts` 의 "accounts SDK protocol profile".
  상류 SDK 의 어휘라 건드리지 않는다.

## 10. 4번 결과 (2026-09-21)

`internal/chainsetup/verb_needs.go` 에 어휘와 표를 두고, `allow(verb)` 하나가 강제한다.
**동사 39개 전부가 선언한다** — 전에는 6개였다.

어휘는 둘이다. 질문이 둘이기 때문이다. **단계 조건**은 "내 앞 단계가 기록됐나" 를 묻고 답이
기록된 단계에 있다(`composeNeeds`, 그대로 둔다). **실행 조건**은 단계 기록이 답할 수 없는 것을
묻는다 — `placed`(노드 표가 있나) · `stopped`(전부 내려갔나) · `anyRun`(묻지 않는다). 노드는
어떤 단계도 고르지 않은 이유로 내려가 있을 수 있어서, 두 질문은 겹치지 않는다.

**옮긴 것은 다섯이다.** `Health`·`Preflight`·`Hardfork`·`VerifyValidators` 가 같은 "노드 표가
없다" 를 **네 가지 문구**로 말하고 있었다 — 앞의 둘은 `chain place` 를, 셋째는 "compose the
network first" 를 가리켰고, 넷째는 아무 안내도 하지 않았다. `Rm` 은 실행 중 노드를 손으로
막던 유일한 자리였다. 이제 한 문장 모양으로 모인다.

**"아무것도 필요 없다" 는 여전히 정답이지만 이유를 적어야 한다.** 접근자가 정말 아무것도
필요로 하지 않는 것과, 아무도 살피지 않은 구멍은 코드에서 같아 보인다. 래칫이 그 둘을 가른다:
`reflect` 로 메서드 집합을 걸어 **선언이 없으면 빌드를 세우고**, 조건도 이유도 없는 선언도 세운다.
역방향(없어진 동사가 표에 남는 것)과 단계 이름 오타(`composeNeeds` 가 모르는 키는 nil 슬라이스를
돌려주어 **모든 선행 조건이 공허하게 충족된다**)도 본다.

변이 셋으로 확인했다 — 선언 삭제 · 이유 없는 빈 선언 · 없는 단계 이름. (첫 시도에서 셋 다
"통과" 했는데 **변이가 적용되지 않은 것**이었다. gofmt 가 맵 정렬을 바꿔 문자열 치환이 빗나갔다.
변이가 파일을 실제로 바꿨는지 확인하는 단계를 넣고 다시 돌렸다.)

### 남은 것

- **동사 25개가 아직 `why` 로 면제돼 있다.** 그중 몇은 진짜 접근자지만, `Stop`·`StopNode`·
  `CrossFork`·`StartNode`·`SwapNode` 는 "더 가는 검사를 스스로 한다" 는 이유다. 그 검사들을
  표로 올릴 수 있는지는 5번에서 본다.
- ~~`package-tree.md` 의 패키지별 숫자는 래칫 밖이다~~ — **닫혔다 (2026-09-21). §11 참조.**

## 11. package-tree 본문을 재는 래칫 (2026-09-21)

§0 표 세 줄은 재고 있었고 그 아래 240줄은 재지 않았다. 그래서 세 줄은 계속 맞았고 **사람이
실제로 읽는 자리가 낡았다.** 실측: 패키지별 숫자 70개 중 **32개가 틀렸고**, 일부는 크게 틀렸다 —
`consensus/upgrade` 가 1,949줄로 적혀 있고 실제는 164줄, `dsl` 이 1,192 이고 실제는 1,889.
절 제목 다섯 중 둘은 **패키지 개수조차** 틀렸다(`10패키지`→12, `16패키지`→14).

`[측정]` 등급 문서의 틀린 숫자는 숫자가 없는 것보다 나쁘다 — **누가 확인한 것처럼 읽히기
때문이다.** 그래서 본문도 잰다. `arch.TestPackageTreeBodyIsMeasured` 가 70개 항목을 전부
대조하고 **빠진 패키지도 본다**(조용히 빠진 트리는 완전한 것처럼 읽힌다).
`arch.TestPackageTreeSectionsSumTheirOwnEntries` 는 절 제목을 그 절이 나열한 것의 합과 맞춘다.

변이 셋으로 확인했다 — 숫자 하나 틀리기 · 절 요약 틀리기 · 항목 하나 통째 삭제.

## 12. 5번 — 노드 단위 조건 (2026-09-21)

4번이 표 단위로 없앤 중복이 **노드 단위로 그대로 남아 있었다.** `"node%d has no recorded argv —
run \`chain start\` first"` 가 세 곳에 복사돼 있었다: `StartNode`(node_ops.go:73) ·
`SwapNode`(node_ops.go:148) · `Hardfork`(verbs_hardfork.go:123). 4번을 하면서 이것을 보지 못한
것은 그때 표 단위 조건만 찾았기 때문이다.

세 번째 어휘 `nodeNeed` 를 더했다 — **네트워크가 아니라 노드에 대해 묻기 때문**이다.
"모든 노드가 멈췄나" 와 "3번 노드가 멈췄나" 는 다른 질문이고, 노드 하나를 다시 띄우는 동사를
앞의 것에 묶으면 안 된다. 값은 `launched`(기록된 argv 가 있다 — 다시 실행할 명령이 있다)와
`down`(멈춰 있다)이다.

**조건은 목록이다.** `StartNode` 는 `{down, launched}` 둘 다 필요하고 둘은 서로를 함의하지
않는다. `SwapNode` 는 `{launched}` 만이다 — 스왑은 스스로 멈추므로 도는 노드를 거절하면 안 된다.
값 하나였다면 이 차이를 표현할 수 없었다.

`nodeAll` 은 같은 조건을 표 전체에 건다. `Hardfork` 가 "모든 노드에 argv 가 있어야 한다" 를
자기 루프로 쓰고 있었고, 이제 선언으로 말한다.

변이 셋으로 확인했다 — `StartNode` 에서 `down` 제거 · `Hardfork` 의 `nodeAll` 끄기 ·
`launched` 검사 무력화.

### 남은 면제 22개

`Stop`·`StopNode`·`CrossFork` 는 읽어 본 결과 **면제가 맞다.** `Stop` 은 노드별 결과를 모아
"몇 개 중 몇 개가 멈췄다" 로 보고하므로 앞단 거절이 오히려 정보를 줄인다. `CrossFork` 의 거절
셋은 상태가 아니라 **포크 선언**에 대한 것이다("건널 포크가 선언되지 않았다", "그 바이너리를
쓰는 노드가 없다"). 나머지는 접근자·배선·기록이다.

### 덤

`package-tree` 래칫이 이 작업 중 **곧바로 일했다** — `chainsetup` 이 8,556 → 8,625줄이 되자
빌드를 세웠다. 숫자를 다시 재는 스크립트를 쓰지 않고 손으로 고치면 또 어긋나므로, 래칫과 같은
규칙으로 갱신하는 스크립트를 세션 스크래치패드에 두었다(`retree.py`). **저장소에 넣지 않았다** —
도구를 늘리는 대신 래칫이 틀린 숫자를 막는 쪽이 값이 크다고 보았고, 이 판단은 뒤집을 수 있다.

> **뒤집었다 (2026-09-21, `ea48ad9b`).** 스크래치패드는 다음 사람에게 건너가지 않는 자리다.
> `scripts/refresh-package-tree.py` 로 옮겼고, 래칫의 실패 문구가 그 이름을 댄다.

## 13. 갈래를 패키지로 (2026-09-21)

**`upgrade.ChainPreset` 이 아직 거꾸로였다.** 업그레이드는 체인 구성의 **한 종류**인데 chain
preset 이 그 아래 살고 있었다 — 3번에서 종류로 이름을 좁힌 것과 같은 실수를, 이번에는 패키지에서
하고 있었다.

실측이 그것을 굳혔다: `internal/consensus/upgrade` 는 **`preset.go` 164줄이 전부**였다.
핸드오프 본문·계획·기동은 이미 다른 데로 갔고(1,949 → 164), 실제로 import 하는 곳도
`testengine` 한 곳뿐이다(`core/hardfork` 는 주석 언급). **패키지가 곧 chain preset 이었다.**

그래서 `internal/chainpreset` 으로 옮기고, 패키지가 갈래를 말하므로 타입은 `Preset`,
읽는 함수는 `Load` 로 줄였다. 호출부가 문장이 된다 — `chainpreset.Load(path)`.

### 래칫 다섯이 동시에 울렸고, 그것이 설계대로였다

패키지를 옮기자 `TestEveryPackageIsPlaced`(배치 누락 + 유령) ·
`TestCommentsDoNotContradictTheCode`(죽은 경로를 말하는 주석 둘) · `TestPackageTreeBodyIsMeasured`
(없는 패키지를 부르는 항목 + 빠진 패키지) · `TestPackageTreeSectionsSumTheirOwnEntries` ·
`TestNamesDoNotCollide` 가 **한 번에** 무엇을 고쳐야 하는지 목록으로 말했다. 문서를 손으로 쫓지
않았다.

### 충돌이 갈래를 완성시켰다

`chainpreset.Preset` 과 `keyring.Preset` 이 겹쳤다. 래칫의 요구는 "하나를 개명하라" 였고,
빚으로 기록하는 대신 **키 쪽을 갈래 이름으로 올렸다**: `keyring.KeyPreset`(77파일, 컴파일러가
검증). 이제 짝이 맞는다.

> **이 표도 §14 가 대신한다 (2026-09-21).** `chainpreset` 패키지는 §14 에서 `preset` 으로
> 다시 옮겨졌다. 아래는 그 직전의 상태다.

| 갈래 | 문서 | 타입 | 읽는 함수 |
|---|---|---|---|
| 체인 | `presets/<종류>/*.yaml` | ~~`chainpreset.Preset`~~ | ~~`chainpreset.Load`~~ |
| 키 | `presets/keys/` | ~~`keyring.KeyPreset`~~ | ~~`store.LoadPreset`~~ |

로더까지 `LoadKeyPreset` 으로 바꾸려다 되돌렸다 — **충돌은 타입 하나였고**, 래칫이 요구하지 않은
것까지 바꾸면서 `LoadPresetWithAccountsAt` 같은 이름이 길어지기만 했다.

### 6번 취소

`direction.md` §2 가 제안하던 `presets/keys` → `keys/fixture` 를 **취소했다.** 그 제안은
"`preset` 이 여러 뜻이니 키 쪽이 비켜 주자" 는 내 논리에서 나왔는데, 갈래가 **체인/키 둘**이면
`presets/keys` 은 이미 정확한 이름이다. 비용도 반대를 가리킨다 — 경로 문자열 157곳, env 파일
21개가 그것을 참조한다.

## 14. preset 을 한 갈래·한 모듈로 (2026-09-21)

검토에서 나온 방향을 그대로 반영했다. **preset 은 두 갈래(체인·키)이고, 정의는 한 모듈에 모으고,
쓰는 모듈은 정의하지 않고 쓰기만 한다.**

### 디스크

`keys/preset/` 과 `presets/hardfork/` 가 같은 갈래를 두 자리에서, 한쪽은 **종류(hardfork)로**
부르고 있었다. `presets/keys/` · `presets/chain/` 으로 모았다. 경로를 들고 있던 파일이 119개였고,
그중 21개는 체인 선언이 키 세트를 경로로 참조하는 자리다. `filepath.Join` 으로 조립한 것은 문자열
치환이 못 잡아 세그먼트를 따로 옮겼다.

### 코드

**이 표가 정본이다.** §8·§13 에 같은 모양의 표가 둘 더 있는데 둘 다 이것의 이전 단계이고,
거기 적힌 이름은 코드에 없다.

| 갈래 | 문서 | 타입 | 읽는 함수 |
|---|---|---|---|
| 체인 | `presets/chain/*.yaml` | `preset.Chain` | `preset.LoadChainPreset` |
| 키 | `presets/keys/` | `preset.Key` | `preset.LoadKeyPreset` |

`internal/preset` 이 두 타입과 두 로더, 그리고 키 preset 의 **디스크 형식**(`KeyFile`·`KeyNode`·
`KeyIndexFile`·`NodeLabel`)을 갖는다. `keyring` 은 키 **모델**(`Entry`·`Network`·`Label`·`Source`)만
남기고 preset 을 정의하지 않는다. `keyring/store` 는 키를 **만들고 가져오는 일**(Generate·Extend·
Import)만 남았다 — 1,503 → 1,164줄.

### 순환이 두 번 방향을 가르쳐 줬다

**`preset` → `poa`.** `Chain.GovernanceEnv() poa.Env` 가 문서를 소비자에게 묶고 있었다. 어댑터를
`poa.EnvFromPreset` 으로 옮겼다 — **문서가 독자를 알아야 하면 그것은 문서가 아니다.** 그 테스트도
poa 로 따라갔다.

**`preset` → `store`.** 키 로더가 `store` 에 있었는데 `store` 는 `preset.Key` 를 반환해야 했다.
로더를 preset 으로 옮기니 풀렸다. 옮기면서 드러난 것: 로더 파일이 `store` 의 쓰기 쪽과 **형식
타입을 공유**하고 있었고, 그 형식은 preset 문서의 일부지 저장소의 것이 아니다.

### 래칫이 배치까지 고쳐 줬다

`preset` 을 옮기자 `TestNoUpwardDependency` 가 **L1 다섯이 L2 를 import 한다**고 말했다. 옳은
지적이었다 — poa 의존이 빠진 `preset` 은 `keyring` 하나만 import 하므로 **L1 이다.** 배치를 옮겼다.
`TestNamesDoNotCollide` 는 `Key` 가 `nodeconfig` 와 겹친다고 했고, 실행 옵션의 **키 이름** 타입인
쪽을 `nodeconfig.OptionKey` 로 올렸다(외부 참조 3건 대 110건).

## 15. A·B (2026-09-21)

### A — 스키마 검사가 중첩 객체로 내려간다

`TestSchemaV2MatchesParsedFields` 는 `$defs.envSpec`·`caseSpec` 의 **최상위 속성만** 비교했다.
그 아래 `upgrade` 블록이 통째로 낡은 것이 1번에서 드러났는데, **아무도 보지 않았기 때문**이다.

`TestSchemaV2NestedObjectsMatchTheirTypes` 가 envSpec·caseSpec 아래 **properties 를 가진 객체를
전부 걸어** 각각을 파싱하는 Go 타입과 대조한다. 짝짓기 자체도 래칫이다 — 새 중첩 객체는 짝을
선언해야 통과하고, 없어진 객체의 짝이 남아 있어도 걸린다.

**걸자마자 넷을 더 찾았다.** `genesis` 블록에 `ref`·`provides`·`haltsAt`·`perBinary` 가 없었다.
`haltsAt` 은 실제 케이스(`unsupported-system-contract-version`)가 쓰는 필드다. 넷을 채우자
`perBinary` 안의 중첩 객체가 또 짝을 요구했고(`GenesisSideV2`), 지금 **7개**가 검사된다.
변이 셋으로 확인했다 — 필드 삭제 · 스키마에만 있는 필드 · 짝 없는 중첩 객체.

### B — 죽은 명령을 부르는 e2e

`cmd/chainbench/e2e_commands_exist_test.go` 가 이 패키지의 **모든 테스트가 조립하는 argv 를 AST 로
읽어** 그 명령이 CLI 에 있는지 묻는다. 게이트 뒤 테스트도 파싱은 태그를 가리지 않으므로 함께 본다.

만들면서 두 번 틀렸다. 처음엔 `SetArgs` 의 **인라인 리터럴만** 읽어 argv 를 변수에 담는 테스트
하나를 놓쳤고, 넓혔더니 hex 표와 Solidity 시그니처 목록을 명령으로 읽었다. `SetArgs` 에 실제로
넘어가는 변수만 따라가고 명령어 모양(`^[a-z][a-z0-9-]*$`, 24자 이하)으로 좁혀 **거짓양성 0**이 됐다.

판정은 셋이 달랐다. `upgrade_run_e2e_test.go` 는 **지웠다** — 그 단언 셋(핸드오프 확인 · 자금이
후계 체인에 남는지 · 포크 뒤 tx)을 DSL 케이스 `croissant-successors-take-over` 와
`state-written-before-the-fork-survives-it` 이 그대로 한다. 나머지 둘은 **다른 곳이 덮지 않는
것**을 시험하므로(go-wbft 가 go-wemix chaindata 로 init 되는지 · 거버넌스 NCP 생애주기)
`invocationDebt` 에 **무엇이 걸려 있는지와 함께** 적었다. 그 목록은 줄기만 하고, 낡으면 걸린다.

`chainbench upgrade run` 을 안내하던 기록 셋(`tests/repro/README.md`·`tests/e2e/README.md`·
`wemix-chain.sh`)도 살아 있는 경로로 고쳤다.
