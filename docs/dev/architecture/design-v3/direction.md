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
| `presets/hardfork/*.yaml` + `upgrade.Profile` + `LoadProfile` | ~~preset 으로~~ **완료 (2026-09-21): `upgrade.HardforkPreset` · `LoadHardforkPreset`** | 파일 위치가 `presets/` 인데 타입이 `Profile` 이라 문서와 코드가 다른 말을 했다 |
| `dsl.UpgradeV2.Profile` 필드 | ~~삭제~~ **완료 (2026-09-21)** | 정의서 209개 중 쓰는 것이 0개였다. 지우면서 스키마의 `upgrade` 블록도 실제 구조에 맞췄다 — `required: [profile, template]` 에 `additionalProperties: false` 라 **실제로 쓰는 `preset` 을 거부하는 상태**였고, `fork`·`at`·`from`·`to`·`style`·`carry` 여섯 필드가 통째로 빠져 있었다 |
| `profiles/` 디렉터리 | ~~삭제~~ **완료 (2026-09-21)** | 측정 §4 |
| `keys/preset/` | **`keys/fixture/`** 로 | 키는 골든 설정이 아니라 테스트 픽스처다. `keyring.Preset` 타입도 같이 본다 |
| 메인넷 워크리스트의 "preset" (env 선언) | **env 로 통일** | 코드·파일·스키마가 이미 `env` 라 부른다. 문서만 다르다 |

경로 오기 둘도 같이 고친다: `v2.schema.json:142` 의 *"profiles/\*.yaml"*, 그리고
`chain-handover-2026-09-12.md:379` 의 `profiles/wemix-upgrade-15.yaml` 링크.

**단점.** `keys/preset` → `keys/fixture` 는 env 파일 26개의 `keys.nodekeys.ref` 와 코드의
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
| 4 | 실행 조건 어휘를 만들고 동사 몇 개를 옮긴 뒤 래칫을 건다 | 중 | 래칫이 지킨다 |
| 5 | 남은 동사를 옮긴다 | 대 | 4 가 끝나야 의미 있다 |
| 6 | `keys/preset` → `keys/fixture` | 중 | 참조 71곳. 따로 판단 |
| 7 | 큰 덩어리 | — | 요구가 생기면 |

1·2 는 서로 독립이고 오늘 끝난다. 3 은 1 뒤다. 4 가 이 설계의 본체다.

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

## 8. 3번을 하면서 정한 것 — 왜 `chainPreset` 이 아닌가 (2026-09-21)

`chainPreset` 이 후보로 나왔다. 그 문서가 담은 것을 읽고 **`HardforkPreset`** 으로 정했다.

`presets/hardfork/wemix-upgrade.yaml` 은 **두 체인과 그 사이의 포크**를 적는다 — `upgrade.from`
(wemix) · `upgrade.to`(wbft) · `at_fork`(croissant) · `fork_block` · 양쪽 바이너리 · 역할 수 ·
거버넌스 · 검증자 주소와 BLS 키 · 포트 계획. 한 체인의 설정이 아니라 **핸드오프 환경 전체**다.

그래서 `chainPreset` 은 두 가지가 어긋난다. 첫째, 단수로 읽혀 **주어가 틀린다** — 이 문서의
주제는 한 체인이 아니라 둘 사이의 이행이다. 둘째, 그 이름이 **정말 어울리는 자리가 따로 있다**:
`tests/tc/env/*.env.json` 이 한 체인 + 모양을 적고, 메인넷 워크리스트가 그것을 이미 "preset" 이라
부른다. 여기서 `chainPreset` 을 쓰면 겹침을 없애는 대신 **한 칸 옮기는** 셈이다.

`Preset` 만 쓰지 않은 이유도 측정이다. `keyring.Preset` 이 이미 그 낱말을 갖고 있어
`arch.TestNamesDoNotCollide` 의 새 빚이 된다. 그 목록은 **줄기만 한다**고 선언돼 있다.
6번(`keys/preset` → `keys/fixture`)이 낱말을 비우면 그때 `upgrade.Preset` 으로 줄일 수 있다.

## 9. 3번을 하면서 드러난 것

- **README 가 없는 명령을 안내하고 있었다.** `chainbench upgrade run --profile …` 과
  `upgrade genesis` 인데, `root.go` 가 등록하는 그룹에 `upgrade` 가 없다(있는 것은 `hardfork`).
  낱말만 바꾸면 거짓이 남으므로, 실제로 도는 경로(정의서 + `suite run`, 그리고
  `chainbench hardfork --workspace-dir …`)로 다시 썼다.
- **e2e 테스트 셋이 같은 죽은 명령을 부른다** — `cmd/chainbench/upgrade_run_e2e_test.go`,
  `upgrade_data_migration_e2e_test.go`, `upgrade_gov_ncp_lifecycle_e2e_test.go` 가
  `"upgrade", "run", "--profile", …` 를 넘긴다. `e2e` 태그 뒤에 있고 환경변수가 없으면
  건너뛰므로 **컴파일은 되고 아무도 실패를 보지 못한다.** 이번 범위 밖으로 두었다 — 고치는 일은
  이름이 아니라 없어진 명령의 문제다.
- **여섯째 `profile` 이 있다** — `internal/accounts` 의 "accounts SDK protocol profile".
  상류 SDK 의 어휘라 건드리지 않는다.
