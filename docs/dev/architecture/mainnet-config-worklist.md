# 작업 리스트: 메인넷별 설정 구조 개선

> **[현행 설계]** 이 작업 트랙의 항목과 순서를 정한다. 모듈 경계는
> `architecture-v2.md` 와 `layers.md` 가 이긴다. 여기서 그 둘과 어긋나는
> 항목이 나오면 D9 처럼 열린 결정으로 기록하고 이 문서에서 결정하지 않는다.

> 최초 작성: 2026-09-14
> 대상 리비전: main `7f39c5627e0bdaccaa8e207a67767d71579283f5` (#418)
> 검증 기준선: `20260913T101856Z`
> 상위 맥락: PR [#419](https://github.com/0xmhha/chainbench/pull/419) `HANDOFF.md` 의 요구사항 9개
> 배경 분석: `docs/research/chainbench/analyses/12-mainnet-profile-plan.md`

## 이 문서의 용도

작업 목적은 "공통 테스트 코드에서 메인넷별 차이를 제거하고, 설정으로 관리하여
같은 테스트가 여러 메인넷에서 실행되게 한다" 이다. 그 목적에 닿기 위해 먼저
정리해야 하는 것들을 항목으로 끊어 놓은 문서다.

**이 문서가 작업의 정본이다.** 앞으로의 작업은 여기 적힌 항목을 하나씩 골라
진행한다. 여러 항목을 한꺼번에 열지 않는다.

모든 항목이 지금 완전히 정리된 상태가 아니다. 조사가 덜 된 것도 있고, 결정이
없어 손댈 수 없는 것도 있다. 그래서 항목마다 무엇이 확인됐고 무엇이 아직
아닌지를 함께 적는다.

## 관통하는 원칙

**하위호환을 남기지 않는다.** 이 트랙의 작업은 옛 형식을 읽어 주는 코드를 지우고,
최신 상태 하나만 남긴다. 그 최신 상태가 Z3 에서 v1 이 된다. 옛 형식을 위해 분기를
남기면 그 분기가 다음 사람의 판단을 흐린다. 실제로 그런 일이 있었다(§2 의 C1 배경).

**용어는 한 개념에 하나만 쓴다.** 같은 낱말이 여러 뜻을 갖는 곳에서 잘못된 판단이
나왔다. 아래 §1 의 용어 표가 정본이다.

---

## 진행 규칙

한 항목은 다섯 단계를 거친다.

1. **조사** — 그 항목이 실제로 어디에 어떻게 있는지 코드로 확인한다. 추정으로
   시작하지 않는다.
2. **결정** — 선택지가 갈리면 여기서 멈추고 사용자에게 묻는다. 결정 내용을 이
   문서에 적는다.
3. **변경** — 코드를 고친다. 커밋은 항목 하나에 하나다.
4. **검증** — 기준선과 대조한다. 아래 "검증 기준" 을 따른다.
5. **기록** — 상태를 갱신하고, 새로 드러난 것이 있으면 항목을 추가한다.

규칙 세 가지를 지킨다.

- **ID 는 재사용하지 않는다.** 항목이 없어져도 번호를 비워 둔다.
- **케이스 파일을 대량으로 고치는 작업은 목표 모양이 확정된 뒤 한 번만 한다.**
  여러 번 쓸면 매번 기준선 대조 비용이 든다.
- **조사하지 않은 것을 조사했다고 적지 않는다.** 수치는 어떻게 셌는지 함께 적는다.
- **낱말을 믿기 전에 `terminology-map.md` 를 본다.** 한 낱말이 여러 뜻을 가진 곳이
  어디인지, 그 낱말의 주인이 우리인지 체인인지를 적어 둔 문서다.
- **주석을 근거로 결정하지 않는다.** 주석이 무언가를 주장하면 코드로 확인한 뒤에
  쓴다. 2026-09-14 에 이 규칙이 없어서, "옛 철자는 영속 상태에 쓰여 있어 계속
  동작한다"(`role.go`)와 "poa 에는 프록시 계층이 없다"(`peering.go`)라는 두 주석을
  사실로 받아들여 잘못된 결론을 냈다. 둘 다 코드와 달랐다.
- **항목이 건드린 파일의 주석은 코드와 대조하고, 대조한 결과를 커밋에 적는다.**

**실패를 쫓을 때는 출력을 거르지 않는다.** `go test` 를 `grep -E "^--- (PASS|FAIL)"`
로 걸러 돌렸다가, 간헐적 실패를 두 번 만나고도 `t.Fatalf` 문구를 한 번도 못 봤다
(X14). 요약 줄은 "무엇이" 실패했는지만 말하고 "왜" 는 그 위에 있다. 통과가 예상될
때만 걸러도 되고, 원인을 모르는 실패를 쫓는 중에는 전량을 남긴다.

**게이트된 테스트는 CI 가 못 읽는다.** `//go:build e2e` 는 `go test ./...` 이
컴파일조차 하지 않고, `go vet -tags e2e` 는 컴파일은 해도 문자열 인자가 은퇴한
플래그인지 모른다. 이 트랙이 그 안에서 회귀를 둘 만들었다(X10, X12). **CLI 표면이나
액션 인자를 바꾸면 `tests/e2e` 를 실제 바이너리로 돌려 본다.**

## 측정한 사실 (2026-09-14, `tests/tc` 기준)

세는 방법은 케이스 JSON 을 읽어 `kind == "case"` 인 것만 골랐다. 체인을 띄워
확인한 것이 아니다.

| 사실 | 값 |
|---|---|
| 인라인 `env` 를 쓰는 케이스 | 205건 |
| 별도 env 파일을 참조하는 케이스 | 0건 |
| 서로 다른 `env` 모양 | 42개 |
| 가장 흔한 모양 하나가 쓰이는 케이스 | 93건 |
| 그 모양과 한 필드만 다른 변형 5개 합계 | 54건 (누적 147건) |
| `capabilities` 만 다르고 그 값이 "없음" 인 케이스 | 33건 |
| 모든 케이스에 있는 `env` 필드 | `chain`, `binaries`, `topology`, `keys` (각 205건) |
| 40자리 주소 리터럴이 있는 파일 | 138건 |
| 시스템 컨트랙트 주소(`0x…1000`~`0x…1004`)가 있는 파일 | 62건 |
| `gasTip` 을 읽는 파일 | 14건 |
| `${…}` 보간을 쓰는 파일 | 118건 |
| stablenet 바이너리 철자 | `gstable` 164건, `go-stablenet` 13건 |
| 환경변수 형태(`${VAR:-이름}`)로 바이너리를 적는 케이스 | 5건 |
| 대상 서버 절대 경로를 바이너리로 쓰는 케이스 | 3건 |
| `pn` 을 선언하는 토폴로지 | 6건 (`bp` 204건, `en` 21건). 그중 4건이 `bp` 13 · `en` 1 · `pn` 1 |
| 실행 옵션·설정 스코프에 옛 역할 철자를 쓰는 케이스 | **0건** (스코프를 쓰는 케이스 자체가 1건, 값은 `all`) |

---

## 용어

같은 낱말이 여러 뜻으로 쓰이고 있다. 이 표가 이 트랙의 정본이다.

| 쓸 말 | 무엇 | 지금 코드의 이름 |
|---|---|---|
| **대상 워크스페이스** | 실행 대상 머신의 `dataRoot` 아래 약속된 폴더 트리. bin, configs, genesis, keystore, keys, node, runtime, logs 여덟 칸. 24/7 개발 서버라면 여기에 키·genesis·설정이 이미 올라가 있고, 테스트마다 다시 올리지 않는다 | `resource.WorkspaceConfig` 의 `dataRoot` + `paths`, 선언 파일은 `workspace-config.yaml` |
| **체인 기록** | 한 체인이 무엇으로 요청됐고(`request`), 무엇으로 구성됐고(노드 표·키·genesis·설정·피어링·포트), 지금 어떤 상태인지(단계 진행·pid·argv). 테스트마다 하나씩 생기며, 결과 확인과 디버깅의 정본이다. 실행 이력은 이 파일이 아니라 옆의 `runs/<타임스탬프>/` 와 `chainstate.jsonl` 이 쌓는다 | `chainsetup.State`, 파일은 `chain-record.json` |
| **실행 세션** | 엔진 한 번의 실행이 남긴 아티팩트와 결과 | `internal/core/session` 의 per-run session |
| **로컬 작업 공간** | 따로 지정하지 않았을 때 chainbench 가 자기 것을 두는 곳. 기본 `~/.chainbench` | `internal/core/home`, `control.artifactRoot` |

"워크스페이스" 라는 말은 **대상 워크스페이스**에만 쓴다. 체인 기록이 담는 것은
대상 경로가 아니라 한 체인의 사실이므로, 그 파일을 워크스페이스라 부르지 않는다.

다만 그 파일이 든 **디렉터리**는 워크스페이스가 맞다. `runs/`, 노드 데이터 디렉터리,
genesis, 설정이 그 안에 있다. 그래서 `--workspace-dir` 플래그는 바꾸지 않는다.

---

## 1. 결정 항목 (코드 없음)

결정이 없으면 뒤가 막힌다. D1~D4 는 0순위다.

| ID | 결정할 것 | 지금 상태 | 권고 |
|---|---|---|---|
| D1 | 공유하는 구성 문서의 이름 | **확정 (2026-09-15)** | **`chain-preset`.** 미리 구성해 둔 체인 설정을 그대로 쓰는 것이므로 `preset` 이 맞는 낱말이고, 무엇의 preset 인지를 `chain-` 이 말한다. 파일은 `<id>.chain-preset.json`, Go 타입은 `ChainPreset`. 한때 "공용 env" 로 부르려 했으나 `.env` 와 혼동을 부른다 |
| D2 | `boot` 의 뜻 | **결정됨 (2026-09-14)** | DSL 표면에서 노드 역할 `boot` 를 없앤다. 모든 지원 체인에서 노드 사이의 연결은 `pn` 이 맡는다. 기동 단계 이름 `boot`(poa), 계정 기능 `validator`, 업그레이드의 바이너리 이름 `validator` 는 노드 역할이 아니므로 건드리지 않는다 |
| D3 | family 가 체인의 성질인가 망의 성질인가 | **해소 (2026-09-14)** | 질문이 틀렸다. 한 망의 합의는 하나이고 구성할 때 정해진다. 지금 핸드오프에서 둘로 보이는 것은 **바이너리 방언이 family 에 얹혀 있기** 때문이다. 축은 넷이다 — 합의, 계정·암호(accounts SDK `Protocol`), 바이너리 방언, 체인 상수. 공유 관계가 축마다 다르므로 한 축으로만 가를 수 없다. V10·V7·V11 이 축을 제자리에 놓는다 |
| D4 | 병합 규칙 | **확정 (2026-09-14)** | 맵은 깊게 병합하고 키 단위로 덮는다. `null` 은 삭제다. 배열은 통째 교체한다(합집합이면 항목을 뺄 수 없다). genesis·keyring 이 겹치면 오류다 — 정체성이라 조용히 바뀌면 재사용 판정과 fingerprint 가 실제와 어긋난다. 미지 키 거부는 **이미 있다**: 최상위 필드는 `parseStrict` 의 `DisallowUnknownFields`, 스코프는 `node.ValidScope`, 설정 노브는 `ApplyConfigOverride`, 실행 노브는 `ParseOverrides` 가 각각 거부한다 |
| D5 | 문서 종류를 새로 만들지 | **확정 (2026-09-15)** | 새 종류를 만들지 않는다. 지금 `env` 가 하는 일이 바로 체인 구성 선언이고, 참조(위로 8단계 탐색)와 병합(`extends`) 기계가 이미 거기 있다. **다만 이름은 `chain-preset` 으로 바꾼다**(N1) — `env` 는 무엇을 담는지 말해 주지 않고 `.env` 와 겹친다 |
| D6 | "메인넷" 이 인스턴스인가 패밀리인가 | 미정 | 인스턴스로 본다. 완료 조건 4번의 판정 기준이다 |
| D7 | 로컬 서명 전환(R4)을 이번 범위에 넣을지 | 미정 | 결론 없음. 넣지 않으면 실제 망에서 faucet·deploy·load 가 안 돈다. 넣으면 범위가 크게 는다 |
| D8 | `chain up` 성공이 보장하는 것 | 미정 | HANDOFF 의 최우선 질문. 배포된 입력 / 띄운 프로세스 / RPC 응답 / 블록 진행 중 무엇인가. 지금 단계는 `deploy` 와 `start` 둘뿐이다 |
| D9 | consolidation-plan §0-2 와 PR #419 F1 의 충돌 | 보류 | 이번 작업은 어느 쪽에서도 성립한다. 기록만 남기고 미룬다 |

### 결정 기록

**2026-09-14**

- **D2 확정.** 노드 역할 `boot` 를 없앤다. 예전에 부트노드 동작을 잘못 이해해 생긴
  이름이다. 지원하는 모든 체인에서 노드 사이의 연결은 `pn` 역할의 노드가 맡으면
  되므로 따로 `boot` 를 둘 이유가 없다.
- **표준 토폴로지 확정.** 노드 15대는 `bp` 7대, `en` 7대, `pn` 1대로 구성한다.
  `en` 은 `pn` 과 연결하고 `pn` 을 거쳐 `bp` 와 정보를 주고받는다. `en` 이 `bp` 에
  직접 연결하지 않는다. (V8)
- **용어 확정.** `bp` 는 블록을 만들고 제안하는 노드의 역할이다. `bp` 노드가 제안
  차례가 아닐 때 다른 `bp` 가 제안한 블록을 검증하며, 그 동작을 `validator` 라고
  부른다. 즉 `validator` 는 역할이 아니라 `bp` 의 동작이다. (V9)
- **문서 언어.** 이 저장소의 문서는 한국어로 쓴다. HANDOFF 의 영어 규칙은 PR 제목과
  본문, 커밋 메시지에 적용된다. 이전에 그 규칙이 지켜지지 않아 덧붙은 기록이다.
- **커밋 단위.** 항목 하나에 커밋 하나로 간다. 스쿼시 머지하므로 커밋 수는 문제되지
  않는다.

---

## 2. 주석과 코드의 정합 (C)

주석이 코드와 어긋나면 다음 사람이 잘못 판단한다. 2026-09-14 에 실제로 그랬다.

다만 주석은 18,885줄이고 Go 코드의 19% 다. 그중 파서가 참·거짓을 판정할 수 있는
주장은 일부뿐이고, 이번에 잘못된 판단을 부른 두 주석은 그 일부에 들지 않는다.
문법은 멀쩡하고 내용이 틀린 문장이었다. 전수 의미 감사는 저장소를 다시 리뷰하는
일이고, 앞으로의 작업이 그 주석들을 다시 바꾼다. 그래서 둘로 나눈다.

| ID | 항목 | 근거 | 의존 | 상태 |
|---|---|---|---|---|
| C1 | 기계로 판정되는 주석 오류를 고치고, 검사기를 회귀 테스트로 박는다 | 2026-09-14 측정: 없는 패키지 경로 38건, 없는 문서 경로 5건, 없는 파일 이름 2건, 사라진 패키지 이름을 말하는 패키지 주석 10건, 옛 이름으로 시작하는 문서 주석 25건 | — | **완료 (2026-09-14)** |
| C2 | 주석 의미 전수 감사 | 파서가 못 잡는 주장을 코드와 대조한다. 코드가 멈춘 뒤에 한다 | Z1, Z2 | Z3 과 함께 |

C1 은 V3 보다 먼저 했다. 검사기는 `internal/arch/comments.go` 이고
`TestCommentsDoNotContradictTheCode` 가 0건을 지킨다. 판정하는 것은 다섯 가지다.
인용한 파일·줄·패키지 경로·문서 경로가 있는지, 패키지 주석이 제 패키지를 말하는지,
문서 주석이 다른 심볼 이름으로 시작하는지.

산문으로 시작하는 문서 주석은 강제하지 않는다. 시나리오를 문장으로 적은 테스트
주석이 제 이름을 되뇌는 것보다 잘 읽히고, 그것은 거짓이 아니라 문체다.

**고친 것 중 무거운 것 셋.** `internal/chainsetup/workspace.go` 가 "Package
netcompose" 로 시작하는 옛 패키지 주석을 갖고 있어 `doc.go` 의 정본과 둘이었다.
`Preflight` 의 문서가 실제로는 `startPhase` 와 `checkVacant` 의 문서 두 개가 쌓인
것이었다. `app.Validate` 와 `app.Report` 의 문서가 각각 옆 타입 위에 떠 있었다.

---

## 2.5 이름과 실제 동작의 어긋남 (N)

이름이 그 자리가 하는 일을 말하지 않으면 읽는 사람이 틀린다. 이 트랙에서 실제로
네 번 틀렸고(§진행 규칙 참조), 매번 원인이 같았다.

**이름은 "무엇이 비었나" 가 아니라 "이것이 무엇인가" 로 정한다.** 2026-09-15 에
후보를 고르면서 이미 쓰인 낱말을 전부 배제하는 방식으로 접근했다가, 그러면 뜻이
맞는 낱말을 버리고 뜻이 안 맞는 낱말을 고르게 된다는 지적을 받았다. 이미 쓰인 이름이
틀렸으면 그 이름을 고치는 것이 맞다.

| ID | 어긋난 이름 | 실제로 무엇인가 | 정한 이름 | 상태 |
|---|---|---|---|---|
| N1 | `env` (정의서의 블록·파일) | 체인 구성 선언. 지금은 "구성해라" 만 말할 수 있고 "이미 있는 체인에 붙어라" 를 말할 수 없다 | 보류 — §2.5.1 | **보류 (P 묶음 뒤)** |
| N2 | `LayerCase` (`"case"`) | 테스트 정의서가 아니라 **CLI·MCP 가 넘긴 override** 다. 층을 밝히지 않은 값이 여기로 떨어진다(`builder.go:53`) | `LayerCommand` | **완료 (2026-09-15)** |
| N3 | `LayerEnv` (`"env.launch"`) | 선언이 정한 값. 정의서의 `launch` 블록만이 아니라 포트·HTTP·마이너 등 25곳이 이 층을 쓴다 | **그대로 둔다** | 판정 완료 |
| N4 | `workspace.json` | 대상 워크스페이스가 아니라 한 체인의 **요청·구성·상태 기록**이다. 실행 이력은 이 파일이 아니라 옆의 `runs/` 와 `chainstate.jsonl` 이 쌓으므로 "history" 는 붙이지 않는다 | `chain-record.json` | **완료 (2026-09-15)** — W0 흡수 |
| N5 | `preset` (세 뜻) | 키 출처(`keys.nodekeys.source`), 대상에 이미 있는 입력 묶음(옛 `resource.InputPreset`), 그리고 D1 이 정한 공유 체인 구성 | 정의서는 `keyPreset`·`chainPreset`, 워크스페이스 설정은 `existingInputs` | **완료 (2026-09-15)** — `chainPreset` 은 P1 에서 만든다 |
| N6 | `LayerFamily` (`"family"`) | 합의 family 가 정하지 않는다. V10 이후 이 층의 세 값은 전부 하니스가 정하고 방언이 거른다 | `LayerHarness` | **완료 (2026-09-15)** |
| N7 | `upgrade run --preset` · `HandoffInputs.PresetDir` | 다른 네 명령이 `--keys` 라고 부르는 것과 **같은 값**이다. 한 값이 이름 셋을 갖고 있었다 | `--keys` · `KeysDir` | **완료 (2026-09-15)** |

### N3 을 그대로 두는 이유

층 이름 넷이 두 가지 어휘를 섞어 쓴다. `family` 와 `role` 은 "누가 그 값을
계산했나" 를, `env` 와 `case` 는 "어느 문서에서 왔나" 를 말한다. 그래서 이 넷은
아래 해석 순서와 **같은 축이 아니다.**

1차 `chain-preset` — 공유하는 체인 구성
2차 정의서의 override — 그 테스트만 다른 것
3차 CLI·MCP 가 넘긴 값 — 실행할 때 덮는 것

1차와 2차는 병합이 끝나면 둘 다 `LayerEnv` 로 들어온다. 층은 그 둘을 구분하지
못한다. **구분할 필요가 없다** — 정의서 파일에 무엇을 덮었는지가 남아 있다. 그래서
`LayerEnv` 는 "선언이 정한 값" 이라는 뜻으로 맞고, 이름을 바꾸지 않는다.
바꾸는 것은 나머지 둘이다(N2·N6).

### 2.5.1 N1 — attach 선언 (완료 2026-09-18)

지금 케이스가 `env` 를 빠뜨리면 **테스트가 돌지 않는다.** 막는 곳이 둘이다.

- 스키마의 `caseSpec.required` 가 `schemaVersion`·`kind`·`id`·`env`·`steps`
  다섯을 요구한다(`internal/dsl/schema/v2.schema.json:196`).
- `lowerCase` 가 `dsl: v2 case %s needs "env"` 로 거부한다
  (`internal/dsl/spec_v2.go:372`). `env` 가 있어도 그 안에 `chain` 이 없으면
  또 거부한다(`spec_v2.go:387`).

즉 **모든 케이스가 "이 체인을 구성해라" 를 반드시 말해야 한다.** "이미 있는 체인에
rpc-url 로 붙어라" 를 말할 자리가 없다. `envSpec` 의 열여덟 속성 어디에도 rpc 가
없고, attach 는 명령행 플래그(`--attach`·`--rpc`·`--workspace-dir`,
`cmd/chainbench/suitecmd/run.go`)로만 존재한다.

**완료 (2026-09-18).** 정해 둔 방향대로 넣었고, 한 곳만 다르게 갔다.

- `env` 는 그대로 두고 **그 하위에 `attach` 를 더했다.** `{"rpc": [...],
  "keysDir": ..., "provides": [...]}` 세 가지다.
- **형태 둘은 배타다.** 붙는 선언이 만드는 쪽 키(topology·genesis·binaries·
  upgrade·hardforks·accounts·keys·blueprint·launch·config·target·manifest·
  genesisTemplate)를 하나라도 말하면 거부하고, **무엇을 지우라고 이름을 댄다.**
  이유는 어느 망을 두고 하는 주장인지 말하지 않은 선언이기 때문이다 — 망을 하나
  만들고 아무것도 돌리지 않은 채 다른 망을 보고하게 된다. 그건 통과로 읽힌다.
- **`chain` 은 요구한 그대로 뒀다.** "붙은 뒤 RPC 로 읽는다" 는 방향은 여전히 옳지만
  먼저 할 수 없다 — 컨트랙트를 이름으로 부르는 케이스는 첫 호출 전에 체인의 표가
  있어야 하고, 그 표는 매니페스트에서 온다. 문서에 이유를 적어 뒀다.
- **게이팅은 그대로 걸린다.** 아무도 구성하지 않은 망이라 광고된 능력이 없으므로,
  망을 세운 사람이 `attach.provides` 로 말한다. 이름이 `capabilities` 가 아닌 이유는
  `genesis.provides` 와 같다 — env 의 `capabilities` 는 **요구**이고, 한 낱말에 반대
  뜻 둘을 담았다가 케이스 6건이 영영 스킵됐다.
- **명령행이 이긴다.** `--rpc`·`--attach`·`--workspace-dir` 가 먼저 판정되고, 그중
  아무것도 없을 때만 선언에 묻는다. 실측으로 확인했다 — 선언은 8600 을 가리키는데
  `--rpc` 로 아무도 없는 포트를 주면 실패한다(선언이 이겼다면 통과했을 것이다).
- 엔드포인트에 `${VAR:-기본값}` 이 먹는다. 바이너리 경로와 같은 확장이며, 커밋된
  케이스가 특정 머신의 주소를 안 들고 있게 하는 자리다.

**실측 (2026-09-18).** `stablenet-bp4` 를 띄워 둔 뒤, **망을 가리키는 플래그를 하나도
주지 않고** `run tests/tc/basic/08-attached-chain-produces.json` 이 통과했다. 주지
않은 능력(`contract:govMinter`)을 요구하게 바꾸면 스킵되고 `--no-skips` 가 실패한다.

**아키텍처 가드가 한 번 잡았다.** 처음엔 `DeclaredAttach` 가 `*dsl.AttachV2` 를
돌려줬는데, 그러면 표면이 app 을 건너뛰고 모듈을 직접 부르게 된다(§2 규칙).
`app.AttachDecl` 을 app 의 타입으로 두고 고쳤다.

**남긴 것 — N1-b (§11.5).** `chain` 을 RPC 로 읽어 지우는 일이다. 막고 있는 것은
컨트랙트 표 하나뿐이다: `govMinter` 같은 이름을 쓰는 케이스는 첫 호출 **전에** 표가
있어야 하고, 표는 매니페스트에서 온다. 열려면 순서를 바꿔야 한다 — 붙어서
`eth_chainId` 를 읽고, 그 값으로 매니페스트를 고른 뒤, 그때 표를 세운다.

### 2.5.2 N5 — AST 로 센 결과와 수정 계획

`go/parser` 로 전 파일을 읽어 `preset` 이 든 식별자·문자열·태그를 전부 뽑고, 선언과
참조를 이어 뜻을 갈랐다. 총 528곳이다(선언 91, 패키지 경유 참조 200, 문자열 233,
태그 3).

| 뜻 | 대표 심볼 | 참조 | 판정 근거 |
|---|---|---|---|
| **키 preset** | `keyring.Preset`(106) · `store.LoadPreset` 계열(36) · `store.PresetKeys`(10) · `blueprint.FromPreset`/`PresetFrom` · `genesis.PresetSource` · `keys/preset/` · 정의서의 `"source": "preset"`(199건) | ~480 | 전부 키 집합을 읽거나 그 키로 genesis·설정을 만든다 |
| **대상에 이미 있는 입력** | `resource.InputPreset` · `WorkspaceConfig.Presets` · `Inputs.Preset` · `testengine.applyPreset`/`applyPresetConfigs` | ~40 | `inputs.mode: prepared` 일 때만 돈다. genesis·키링·설정 파일을 **가리킬 뿐 만들지 않는다** |

(뜻 2 의 심볼 이름은 아래 1번이 끝나 바뀌었다. 위 표는 바꾸기 전의 측정이다.)

**두 뜻이 한 함수 안에서 만나는 자리를 하나 찾았다.** `testengine.applyPreset` 은
뜻 2 인데 그 안에서 `up.KeysSource = "preset"` 으로 뜻 1 의 enum 값을 쓴다
(`internal/testengine/compose.go:320`). 이름이 같아서 읽는 사람이 구분할 단서가 없다.

#### 수정 순서

1. ~~**뜻 2 를 `existing-inputs` 로 옮긴다.**~~ **완료 (2026-09-15).**
   `resource.InputPreset` → `ExistingInputs`, `WorkspaceConfig.Presets` →
   `ExistingInputs`(yaml `existingInputs`), `Inputs.Preset` → `Inputs.Name`
   (yaml `name`), `InputPrepared` `"prepared"` → `InputExisting` `"existing"`,
   `applyPreset`/`applyPresetConfigs` → `applyExistingInputs`/
   `applyExistingConfigs`. 모드와 묶음이 같은 낱말을 쓴다. 파일 6개.
2. ~~**뜻 1 의 이름 없는 표면만 `keyPreset` 으로 옮긴다.**~~ **완료 (2026-09-15).**
   스키마 enum `["preset","generate"]` → `["keyPreset","generate"]`, 정의서 199건,
   `steps_compose` 의 switch 와 거부 문구, `--keys-source` 기본값 2곳과 구조체 태그,
   MCP 설명, `testengine` 이 기록하는 값. **한 낱말이 정의서·CLI·MCP·기록에서 같다.**
   205건을 `chainbench validate` 로 전수 검사해 전부 통과했다.
3. **뜻 1 의 Go 심볼은 그대로 둔다.** §2.5.3 참조. 정의서의 `keyPreset` 이 코드의
   `keyring.Preset` 에 대응한다 — 대응 지점은 `steps_compose.go` 의 switch 한 곳이다.

#### N7 — 한 값이 갖고 있던 이름 셋 (완료)

`upgrade run` 만 `--preset` 이라고 불렀다. 같은 값을 `chain new`·`chain up`·
`suite run`·`validator roster` 넷은 `--keys` 라고 부르고, 기본값도 다 `keys/preset`
이다. 코드 안에서는 한 줄에 두 이름이 있었다 — `PresetDir: keysDir`
(`internal/testengine/compose.go:154`).

처음에는 `--key-preset-dir` 을 제안했는데 그건 낱말을 하나 더 만드는 것이었다.
저장소에 이미 이 값의 이름이 있다. 셋을 `keys`/`KeysDir` 로 모았다: 플래그, MCP 인자,
`HandoffInputs.KeysDir`, `UpgradeRunIn.KeysDir`.

**옛 플래그를 별칭으로 남기지 않았다.** 동작하는 별칭은 이 트랙이 없애는 하위호환
그 자체다. 다만 실제로 쳐 보니 거부 문구는 `error: unknown flag: --preset` 한 줄이고
usage 를 찍지 않는다 — 새 이름을 알려면 `--help` 를 한 번 더 쳐야 한다. 플래그 오류에
usage 를 붙이는 것은 명령 전체에 걸리는 설정이라 이 항목에서 건드리지 않았다.

### 2.5.3 뜻 1 의 Go 심볼을 안 바꾸는 이유

`keyring.Preset` 을 `keyring.KeyPreset` 으로 바꾸면 패키지 이름이 이미 말한 것을
타입 이름이 되풀이한다. Go 는 이것을 stutter 라고 부르고 피하라고 한다
(`chain.Config`, `rpc.Client`). `store.LoadPreset` 도 같다 — 패키지가
`keyring/store` 다.

**애매했던 것은 심볼이 아니라 자격 없는 표면이었다.** `resource.InputPreset` 과
`keyring.Preset` 은 패키지가 이미 갈라 준다. 갈라 주는 것이 없는 자리는 셋이다:
정의서의 enum 값, workspace-config 의 yaml 키, 그리고 산문의 맨 낱말 "preset".
그 셋만 고친다.

맨 낱말 `preset` 은 코드에도 문서에도 홀로 쓰지 않는다. `internal/arch` 가 이것을
검사하게 할지는 뜻 1 의 표면을 옮긴 뒤 판단한다.

---

## 3. 어휘와 주인 정리 (V)

값의 주인이 정해져 있지 않아 생긴 문제들이다. 뒤 작업의 크기를 줄인다.

| ID | 항목 | 근거 | 의존 | 상태 |
|---|---|---|---|---|
| V1 | 바이너리 이름을 논리 이름으로 좁힌다. 매니페스트 기본값 → 워크스페이스 별칭 → 경로 조합 순으로 해석한다 | 지금 DSL 값은 검증 없이 `exec.CommandContext` 로 간다(`internal/core/process/local.go:48`). 매핑 함수 `BinaryPath` 는 있으나 호출처가 `internal/app/upgrade.go:249` 한 곳뿐이다 | D5 | 조사 완료, 미착수 |
| V2 | 테스트 정의서에서 절대 경로를 거부한다 | 파싱 단계에서 거부한다(`binaryRefIsAName`). 절대·상대 경로와 `~`, 그리고 `${VAR:-/경로}` 형태까지 본다. 5건(3건이 아니라 5건이었다)의 선언을 지웠고, 도커 워크스페이스 설정이 같은 경로를 만든다 | ~~V1~~ | **완료 (2026-09-14)** |
| V3 | 역할 어휘를 `bp`·`en`·`pn` 으로 좁힌다 | `RoleValidator`·`RoleEndpoint`·`RoleBoot` 상수를 지우고 `NormalizeRole` 의 접기를 없앴다. 카운트 형식의 `validators`·`endpoints` 별칭, 접기만 하던 두 `UnmarshalJSON`, 라벨의 옛 철자도 함께 지웠다. `chainbench validate` 가 노드 표의 역할을 오프라인으로 검사한다 | D2 | **완료 (2026-09-14)** |
| V4 | 스코프 규칙을 어휘에 묻게 한다 | 규칙이 네 군데에 흩어져 있었고 `node<N>` 정규식은 두 번 정의돼 있었다. 손으로 적은 두 목록이 `bp`·`en` 만 적어 `pn` 이 빠졌다. `node` 에 `ScopeAll`·`ValidScope`·`ScopeIndex`·`ScopeFor`·`ScopeWords` 를 두고 문법과 워크스페이스가 그것을 묻는다 | V3 | **완료 (2026-09-14)** |
| V5 | 설정 스코프에 역할을 더한다 | 설정과 실행 옵션이 같은 세 형태를 같은 순서로 받는다. 기록되는 스코프를 검사하지 않던 구멍도 막았다. 스키마의 두 패턴이 파서와 어긋나 있던 것도 고치고, 어긋나면 실패하는 테스트를 붙였다 | ~~V3~~, ~~V4~~ | **완료 (2026-09-14)** |
| V10 | `StartFlags` 에서 바이너리 몫을 뺀다 (완료) | 두 family 의 `StartFlags` 차이는 `--rpc.enabledeprecatedpersonal` 과 `--rpc.allow-unprotected-txs` 둘뿐이고, 둘 다 합의가 아니라 바이너리가 받는 플래그다. 방언이 이미 그 차이를 안다(`geth110Wemix` 가 앞의 것을 지운다). family 는 원시 문자열 대신 타입을 내놓는다. `ParseFamilyFlags` 주석이 이미 이 작업을 예고한다("It goes away when families declare a FamilyPolicy directly"). `registry.LaunchPolicy{Mine bool}` 하나가 남았고 shim 39줄이 사라졌다 | — | **완료 (2026-09-14)** |
| V6 | family 선택을 등록제로 바꾼다 | `registry.RegisterFamily`/`FamilyByName` 을 두고 각 family 가 `init()` 에서 등록한다. `external.Load` 의 switch 와 두 import 가 사라졌다. 링크 책임이 옮겨가므로 `chains/all` 이 family 를 명시적으로 끌어온다. **함께 고친 것**: `app.derivationFor` 가 family 이름을 비교해 BLS 파생 여부를 정하고 있었다. 등록제를 열면 새 family 가 그 else 로 떨어져 조용히 틀리므로 `ValidatorsCarryBLS()` 로 family 가 답하게 했다 | ~~V10~~ | **완료 (2026-09-14)** |
| V7 | 방언을 체인이 말한다 | `DialectFor(chainID)` 가 `if chainID == "wemix"` 로 갈린다. 매니페스트에 방언 필드가 없어 체인이 스스로 말할 방법이 없고, 옛 geth 세대를 쓰는 새 체인은 조용히 `Geth114` 를 받아 모르는 플래그를 달고 부팅에서 죽는다. 매니페스트에 `dialect` 를 필수로 두고(`network_id` 와 같은 "기본값 없음" 규칙), `nodeconfig` 은 이름으로 찾는다. 모르는 이름은 오류다. 핸드오프는 family 대신 플러그인을 받는다 — 방언은 매니페스트의 답이고, family 만 넘기던 경로는 방언이 빈 채로 argv 를 조립하고 있었다 | ~~V10~~, ~~V6~~ | **완료 (2026-09-14)** |
| V11 | 체인 폴더를 조립 지점으로 만든다 | 네 축(합의·계정/암호·방언·체인 상수)을 고르는 자리가 지금 네 곳에 흩어져 있다. 매니페스트, `familyByName` switch, `DialectFor` if, `protocol.ByName` 이다. 체인 파일이 `registry.StaticPlugin` 리터럴 하나로 넷을 고른다. 세 체인이 각자 손으로 쓰던 네 메서드짜리 타입이 사라졌다. 등록이 반쯤 배선된 플러그인을 거부하므로 빠뜨린 선택이 그 자리에서 잡힌다. 한 메서드만 다르면 임베딩으로 덮는다 | ~~V7~~ | **완료 (2026-09-14)** |
| V8 | 표준 15노드 구성을 `bp` 7 · `en` 7 · `pn` 1 로 맞춘다 | 케이스 5건을 옮겼다(wemix 것 하나가 처음 조사에서 빠져 있었다). 정족수가 9에서 5로 바뀌므로 fault 케이스의 단계도 다시 썼다. poa 에 프록시 계층이 없다던 주석 3곳도 고쳤다(X6) | ~~V3~~ | **완료 (2026-09-14)** |
| V9-a | `validator` 를 전수 분류하고 용어 지도를 쓴다 | 코드 797곳·테스트 823곳을 여섯 뜻으로 갈랐다. 넷은 맞고(계정 기능·genesis 주소·실행중 합의집합·wbft 합의) 둘이 틀렸다(bp 노드 개수·포크 뒤 바이너리). 결과는 `terminology-map.md` | D2 | **완료 (2026-09-14)** |
| V9-b | 틀린 두 뜻만 고친다 | CLI·MCP·Go 필드·기록 키·출력 문자열을 `bp`/`en`/`pn` 으로, 정의서의 바이너리 키를 `from`/`to` 로 맞췄다. 뜻이 맞는 `--validators`(신원 수, 실행 중 집합)는 그대로 뒀다 | ~~V9-a~~, ~~W5~~ | **완료 (2026-09-14)** |

### V 항목에 딸린 판정 기준

V6·V7 이 끝나면 다음이 성립해야 한다.

**기존 합의 알고리즘을 쓰는 EVM 체인을 하나 더할 때 새로 쓰는 Go 파일이 0개다.**
매니페스트와 genesis 템플릿만 더한다. 이 기준을 `internal/arch` 에 테스트로
박아 둔다. 나중에 체인별 분기가 들어오면 그 자리에서 실패한다.

---

## 4. 워크스페이스와 대상 경로 (W)

여기는 엣지 케이스가 많아 복잡도가 커지는 자리다. **설계가 확정되기 전에는 코드를
건드리지 않는다.** 아래는 전부 조사부터 시작한다.

배경은 이렇다. 대상이 원격이든 로컬이든, 파일을 어디에 올리고 실행할 때 어느 경로를
참조할지가 정해져야 한다. 24/7 도는 개발 검증 서버라면 약속된 트리 아래에 키·genesis·
설정이 이미 있다. 테스트마다 새로 만들어 올리면 느려지고, 만들 이유도 없다. 그래서
대상 워크스페이스의 폴더 트리를 약속하고 그 안의 파일을 쓴다.

그 기계는 이미 있다. `inputs.mode: existing` 과 `existingInputs` 가 그것이고, 키링은
`srv://` 참조를, 설정은 논리 이름을 대상 파일로 매핑한다. 정리할 것은 그 주변이다.

| ID | 항목 | 근거 | 의존 | 상태 |
|---|---|---|---|---|
| W0 | `workspace.json` 을 `chain-record.json` 으로 개명한다 (**N4**) | 고친 코드는 `session.chainRecordFile` 상수와 `CompositionFilePath`→`ChainRecordPath` 뿐이고, 나머지는 문자열을 직접 쓰던 파일 18개다. `--workspace-dir` 은 **바꾸지 않았다** — 그 디렉터리는 진짜 워크스페이스다(`runs/`·노드 데이터·genesis·설정이 그 안에 있다). 옛 이름만 있는 디렉터리는 조용히 "구성 안 됨" 으로 읽혀 두 번째 체인이 옆에 생기므로, 두 파일 이름을 대며 거부하는 가드를 같이 넣었다 | ~~D1~~ | **완료 (2026-09-15)** |
| W1 | 구성 식별자가 위치에서 파생되는 문제 | 디렉터리를 옮기면 동일성이 깨진다 (PR #419 CLI-C2) | W0 | 조사 필요 |
| W2 | 구성 기록 저장이 원자적이지 않다 | `Composition.Save` 가 `os.WriteFile` 을 직접 부른다. 동시 독자에게 어떻게 보이는지가 열린 문제다 (PR #419 CLI-C2) | W0 | 조사 필요 |
| W3 | 기록된 PID 는 살아 있다는 증거가 아니다 | `NetworkStatus` 가 기록을 읽을 뿐 실사하지 않는다 | W0 | 조사 필요 |
| W4 | CLI 와 엔진의 결과 경로 기본값이 서로 다르게 정해진다 | 모든 세션이 환경 옆에 쌓인다고 약속하기 전에 실효 경로를 확인해야 한다 (PR #419) | W0 | 조사 필요 |
| W5 | 구성 기록에 형식 버전이 없다 | `State.FormatVersion` 과 `StateFormatVersion` 을 두고, 다른 형식이면 이름을 대며 거부한다. `Composition.Load` 가 기록이 있었는지 함께 알려 주므로 "빈 기록" 과 "없는 기록" 이 갈린다. 옛 키(`serverConfig`)를 몰래 받던 코드도 함께 지웠다 | W0 | **완료 (2026-09-14)** |
| W6 | 대상 경로를 정하는 권한이 셋으로 갈려 있다 | `workspace-config.yaml`, 정의서의 값, 명령행 인자. V1 의 바이너리 문제와 같은 뿌리다 | V1, W0 | 조사 필요 |
| W7 | prepared 경로의 엣지 케이스를 정리한다 | 일부만 있을 때, 오래됐을 때, 여러 테스트가 같은 트리를 쓸 때. 여기가 복잡도가 커지는 자리다 | W6 | 조사 필요 |

---

## 5. 병합 기계 (M)

지금이 가장 싼 시점이다. `extends` 를 쓰는 케이스가 0건이라 바꿔도 깨질 것이 없다.

| ID | 항목 | 근거 | 의존 | 상태 |
|---|---|---|---|---|
| M1 | `extends` 를 깊은 병합으로 바꾼다 | 맵은 키 단위로 만난다. 전에는 필드를 통째 교체해서, 노드 하나만 덮으려 한 케이스가 공용 env 의 `all` 스코프와 다른 노드 설정을 조용히 잃었다 | ~~D4~~ | **완료 (2026-09-14)** |
| M2 | 삭제(`null`)와 배열 교체 규칙을 넣는다 | `null` 은 키를 지운다. 배열은 통째 교체한다 — 합집합이면 항목을 뺄 방법이 없어져 삭제가 없는 것과 같은 구멍이 된다 | ~~M1~~ | **완료 (2026-09-14)** |
| M3 | 미지 키를 거부한다 | **이미 있었다.** 병합 결과가 `parseStrict`(`DisallowUnknownFields`)를 지나고, 스코프는 `node.ValidScope`, 설정 노브는 `ApplyConfigOverride`, 실행 노브는 `ParseOverrides` 가 거부한다. 병합에서 한 번 더 보지 않는다 | ~~M1~~ | **확인 완료 (2026-09-14)** |
| M4-a | **실행 전에 병합 결과를 보여준다** | 선언은 세 층을 거쳐 합쳐지고 병합은 조용하다. 케이스 파일을 읽어도 어떤 체인이 뜨는지 알 수 없고, 워크스페이스를 읽으면 이미 뜬 뒤다. `ComposePlan` 은 **합성기가 받는 바로 그 값**에서 렌더링하므로 실제와 다른 망을 말할 수 없다. `RunSuiteIn.OnPlan` 이 이음매이고(라이브러리가 터미널에 직접 쓰지 않는다), `suite run` 은 stderr 로 찍는다 — `--json` 문서는 바이트 그대로 남는다. `--plan` 은 합성하지 않고 계획만 낸다 | ~~M1~~ | **완료 (2026-09-15)** |
| M4-b | 값마다 어느 층에서 왔는지 기록한다 | **완료 (2026-09-16, 3차).** **3차에서야 "기록" 을 했다** — 1차·2차는 출처를 화면에만 찍었고, 실행이 끝나면 사라졌다. 이 항목이 답하려는 질문("이 값이 왜 이 값인가")은 **일주일 뒤에 워크스페이스를 여는 사람이 묻는다.** 그래서 계획을 `compose-plan.json` 으로 `chain-record.json` 옆에 남긴다. 이력은 남기지 않는다(시각은 기록의 step 표시가 갖고 있고, 사본을 하나 더 두면 맞춰야 할 것이 하나 더 는다). 합성에 실패한 실행도 계획은 남긴다 — 실측으로 확인했다. **2차 내용:** **1차에서 launch 노브만 하고 닫은 것이 잘못이었다** — 항목의 문장은 "값마다" 다. 2차에서 출처가 둘 이상인 값 전부로 넓혔다: `binary`, `target`, `nodes.bp/en/pn`, `keys.source`, `keys.dir`. `PlanSource` 는 셋이다 — `declaration`(열 파일이 있다), `command`(방금 친 줄이 있다), `harness`(**고칠 것이 없다**, 그래서 가장 놀라는 값이다). 출처가 하나뿐인 값(chain, workspace, genesis, config)은 일부러 뺐다: 답이 늘 같은 낱말이면 아무 말도 안 하는 것이다. 화면에는 **선언이 안 고른 것만** 한 줄로 찍는다(`chosen by  binary: harness · target: harness`) — 매번 "declaration" 일곱 번을 찍으면 정작 다른 두 줄이 묻힌다. `VerifyLaunched` 의 키셋·노드 수 어긋남 문구도 출처를 댄다. 205건 전부 계획되고, 193건이 `binary: harness · target: harness` 다. **이 작업이 X15 를 드러냈다.** 1차 내용은 아래 | ~~M4-a~~ | **완료 (2026-09-16)** |
| M4-b (1차) | launch 노브의 출처 | launch 노브는 두 곳에서 온다 — 선언(`env.launch`)과 명령줄(`--launch-opt`, `--network-id`). `mergeScopes` 가 둘을 한 표로 합치면서 출처를 버렸다. `PlanKnob{Knob, From}` 으로 바꿔 `declaration`/`command` 를 들고 다니게 했고, `VerifyLaunched` 의 문구가 **"누가 요구했는지"** 를 말한다: 읽는 사람은 어딘가를 고치러 가야 하는데 선언과 명령줄은 다른 곳이다. `--plan` 도 `metrics=true (declaration), nodiscover (command)` 로 찍는다. **config 에는 붙이지 않았다** — 실행 경로에서 출처가 `spec.EnvConfig` 하나뿐이라 `From` 이 어디서나 같은 값이 된다(없는 구분을 필드로 만드는 꼴). 두 번째 출처가 생기면 그때 붙인다. 2026-09-15 에 지운 `WonBy` 와 닮았지만 같지 않다: 그때 지운 이유는 **읽는 자리가 없다** 였고, M4-c 가 그 자리를 만들었다 | ~~M4-a~~ | **완료 (2026-09-16)** |
| M4-c | 계획과 실제 실행 명령을 대조한다 | **완료 (2026-09-16, 2차).** 1차는 **바이너리와 배치를 안 견줬다.** 둘 다 양쪽이 대놓고 말하는 사실인데 빠져 있었다. 워크스페이스는 재사용되므로 계획은 이번 실행 것이고 기록은 지난 실행 것일 수 있다 — 다른 바이너리로 띄운 망, 다른 기계에 있는 망은 스위트의 질문에 기꺼이 답하고 **다른 것에 대해** 답한다. 배치는 문구가 아니라 값으로 견주려고 계획이 `Placement` 를 함께 든다. **1차 내용:** `VerifyLaunched` 가 계획(`ComposePlan`)과 기록(`chain-record.json`)을 견준다 — 체인, 키셋 경로, 역할별 노드 수, 그리고 **선언이 요구한 launch 노브가 그 스코프의 노드 argv 에 실제로 있는지**. 어긋나면 **테스트를 돌리지 않고 멈춘다**: 다른 망에서 도는 테스트는 실패하는 게 아니라 아무도 안 물은 질문에 답한다. 양쪽이 서로 대놓고 말하는 사실만 견준다 — 포트나 datadir 는 합성기가 유도하므로 계획에 견줄 상대가 없다. 값이 아니라 **존재**만 본다: 값의 철자는 방언이 정하므로 여기서 비교하면 같은 사실의 사본이 둘이 된다. 스코프 규칙은 `node.ScopeFor` 에 묻는다 | ~~M4-a~~ | **완료 (2026-09-16)** |
| M5 | fingerprint 를 병합 결과로 계산한다 | **구조상 이미 그렇다.** `ReadFiles` 가 파싱 전에 병합하므로 fingerprint 가 병합된 값을 해싱한다. 조용히 깨질 수 있는 성질이라 테스트로 고정했다 | ~~M1~~ | **확인 완료 (2026-09-14)** |
| M6 | genesis·keyring 겹침은 오류로 유지한다 | 지금 규칙이 이미 그렇다(`internal/testengine/compose.go`). 바꾸지 않았다 | ~~D4~~ | **확인 완료 (2026-09-14)** |

**M4 는 미루지 않는다.** 출처를 못 보는 override 는 유지보수를 쉽게 만드는 것이
아니라 어렵게 만든다. 값이 왜 그 값인지 알려면 사람이 파일 둘을 머릿속에서
합쳐야 한다. M4 없이 M1 만 하면 지금보다 나빠진다.

**M4-a 가 P1 보다 먼저인 이유.** P1 은 정의서 하나를 파일 둘로 쪼갠다. 쪼갠 뒤에
합쳐진 결과가 원래와 같은지 확인하려면 계획을 전후로 견주는 수밖에 없고, 그게
없으면 205건이 조용히 다른 체인에서 도는 것을 못 잡는다. `--plan` 은 합성하지 않으므로
205건을 전부 돌려도 몇 초다.

**M4-a 가 바로 결함을 하나 잡았다.** 205건 중 204건은 계획이 나오고 1건이 거부됐다
(X7). 고친 뒤 205건 전부가 계획된다.

---

## 6. preset 과 값 (P)

| ID | 항목 | 근거 | 의존 | 상태 |
|---|---|---|---|---|
| P1 | 반복되는 체인 선언을 파일로 뽑고 205건이 참조하게 한다 | 체인 선언의 밑바탕(chain·topology·keys·binaries)은 **17개 모양**뿐이었고 그중 하나가 161개 파일에 복사돼 있었다. `tests/tc/env/` 에 17개를 두고, 51건은 이름으로만 부르고 154건은 `extends` 로 다른 것만 덮는다. **205건의 `--plan` 출력이 전후로 바이트까지 같다** | ~~V1~~~V7, ~~M1~~~M5, ~~M4-a~~ | **완료 (2026-09-15)** |
| P2 | preset 에 `values` 블록을 둔다 | 주소·수수료는 `env` 항목이 아니라 step 인자다. preset 만으로는 내려가지 않는다 | D5, P1 | **미착수 — §11.5** |
| P3 | `${…}` 바인딩의 초기값을 preset 이 채운다 | 문법은 이미 있고 118개 파일이 쓴다(`internal/dsl/interp/binding.go:23`). 값의 출처가 실행 중 `read` 결과뿐이다. **R1·R5 가 여기 걸려 있다**(§8) | P2 | **미착수 — §11.5** |
| P4 | 케이스에 `needs`, preset 에 `provides` 를 둔다 | HANDOFF 요구사항 5. 못 주는 값이면 SKIP 하고 사유를 남긴다 | D4 | 미착수 |
| P5 | `applicableChains` 를 능력 요구로 바꾼다 — **오프라인으로 가능한 부분** | 판정표(`case-gate-table.md`)대로 `contract:`·`engine:`·`fork:` 로 표현되는 것을 옮겼다. 143 → **66건**. 판정 변화 0 | ~~V7~~, ~~P4~~, ~~P6~~, ~~매니페스트 능력~~ | **완료 (2026-09-15)** |
| P5-L1 | 남은 게이트를 넓히는 작업 — **라이브 필요** | `validate` 는 "돌 수 있다" 까지만 말하고 "통과한다" 는 말하지 못한다. 넓히는 것이 의도라 **판정표 비교도 X8 가드도 잡아 주지 못한다** — 오프라인 검사가 없는 유일한 묶음이다. 각 케이스를 `--env` 로 대상 체인에 올려 실제로 통과하는지 본 뒤 옮긴다 | ~~P5~~, `--env`(P6) | **완료 (2026-09-18): 66 → 0** — §P5-L1 참조 |
| P5-L2 | `gasTip` 이 wbft 헤더에 있는지 확인한다 | **해소 (2026-09-15). 없다.** 실제 wbft 망을 띄워 `istanbul_getWbftExtraInfo` 를 블록 1·5·32 에서 읽었더니 필드는 `committedSeal · epochInfo · preparedSeal · prevCommittedSeal · prevPreparedSeal · prevRound · randaoReveal · round · vanityData` 뿐이고 **`gasTip` 이 없다**. 그리고 `08-legacy-transfer` 를 `--env wbft-bp4` 로 실제로 올려 보니 `step 3 (read) failed: no "gasTip" in the result` 로 **FAIL** 했다. 즉 `applicableChains: "stablenet,wbft"` 라고 적힌 3건은 **wbft 에서 돌지 않는다** — 선언이 틀렸고, `engine:anzeon` 으로 좁히는 것이 맞았다 | **해소 (2026-09-15)** |
| P6 | 실행 시점에 체인 선언을 주입한다 | 완료 조건 3번. `suite run --env <id\|경로>` 가 케이스의 참조를 갈아끼우고, 케이스가 덮은 것은 남긴다. §P1 아래 참조 | ~~P1~~ | **완료 (2026-09-15)** |

### P5-L1 진행 (2026-09-18)

**낡은 내역을 버리고 다시 셌다.** "35건, `family:wbft` 14 / 게이트 불필요 21" 로
적혀 있었는데 맞지 않았다. 실제로 `applicableChains` 를 들고 있는 케이스는
**66건**이고 이렇게 갈렸다.

```
stablenet             32
stablenet,wbft        29   ← 이번에 처리
wbft                   4
stablenet,wbft,wemix   1   ← 이번에 처리
```

**`stablenet,wbft` 29건을 wbft 망에 올려 봤다.** 선언이 맞는지는 돌려 봐야만
안다 — P5-L2 가 이미 한 건에서 선언이 거짓인 것을 찾았다.

| 결과 | 수 | 옮긴 곳 |
|---|---|---|
| wbft 에서 실제로 통과 | 25 | `family:wbft` |
| wbft 에서 돌지 않음 | 4 | `engine:anzeon` |

**돌지 않는 넷의 이유는 넷 다 달랐다**(전부 새 망에서 재확인).

- `08-legacy-transfer`·`09-dynamic-fee-tx` — `istanbul_getWbftExtraInfo` 에
  `gasTip` 이 없다. P5-L2 가 잰 그대로다
- `12-dynamic-fee-below-basefee-rejected` — stablenet 이 거부하는 것을 wbft 는
  받는다
- `15-gas-limit-exceeds-block-rejected` — 거부하긴 하는데 다른 이유로 거부한다

**확인한 것.** 넷은 이제 wbft 에서 실패가 아니라 **스킵**된다(실측). 25건은 wbft
에서 통과한다(실측). 그리고 29건 전부 stablenet 에서 `--no-skips` 로 통과한다 —
**과도하게 막힌 것이 없다**는 뜻이다.

`stablenet,wbft,wemix` 1건(`20-admin-peers-populated`)은 게이트가 아무 말도 하지
않으므로 지웠다. 한 번도 안 올려 본 wemix 에서 실제로 통과하는 것을 확인했다.

**곁가지로 나온 사실 하나.** 정의서 29개를 한 망에 연속으로 올리면 **망이
나빠진다.** 긴 실행에서 다섯 건이 영수증 대기 시간 초과로 떨어졌는데, 새 망에서
따로 돌리니 다섯 다 통과했다. 이 종류의 라이브 스윕은 **나눠서 돌려야** 판정을
믿을 수 있다.

### 남은 36건도 올려 봤다 (2026-09-18)

`applicableChains` 자체가 스킵을 일으키므로 **그것만 뗀 사본**을 만들어 올렸다.

**`stablenet` 32건 — wbft 와 wemix 양쪽에.**

| 결과 | 수 | 옮긴 곳 |
|---|---|---|
| 세 체인 다 통과 | 14 | 게이트 삭제 |
| stablenet·wbft 통과, wemix 실패 | 7 | `family:wbft` |
| stablenet 전용 동작 | 7 | `engine:anzeon` |
| 케이스에 박힌 수수료 탓 | 4 | 수수료를 고쳐 **게이트 삭제** |

**가장 큰 발견은 수수료 대납 7건이다.** `stablenet` 전용이라고 적혀 있었는데
**세 체인에서 전부 통과한다.** 게이트가 사실이 아니었다.

**stablenet 전용 7건의 이유는 두 갈래다.** 최소 가스값 정책 — wbft 는 stablenet
이 거부하는 것을 받는다(`legacy`·`accesslist`·`feecap` 셋). 그리고 wbft 헤더에
`gasTip` 이 없다(`11-gaslimit-exceeded`). 여기에 `06-basefee-minimum` 과
블랙리스트 둘(0 주소·프리컴파일로의 전송이 wbft 에서는 성공한다)이 붙는다.

**남는 4건은 게이트 문제가 아니다** (세 저장소 코드로 확인, 2026-09-18).

처음엔 "빌드 기본값 탓" 으로 적었는데 **절반만 맞았다.** gwbft 의 상한이 방아쇠일
뿐, 원인은 **케이스에 박아 넣은 stablenet 수수료 값**이다.

`nonce-ordering`·`out-of-order-nonces-mine`·`replacement-tx`·
`same-nonce-replacement` 이 이렇게 적고 있다.

```json
"maxFeePerGas":         "100000000000000",   // 100,000 gwei
"maxPriorityFeePerGas":  "30000000000000"    // 30,000 gwei
```

**그 값은 stablenet 의 고정 tip 을 넘기려고 고른 것이다.** go-stablenet
`core/state_transition.go`:

```go
// If Anzeon is enabled, and the sender is authorized, use the tx's tx.GasTipCap()
// Otherwise, use the header's gas tip
if statedb != nil && !statedb.IsAuthorized(from) {
    gasTipCap = new(big.Int).Set(headerGasTip)
}
msg.GasPrice = min(GasTipCap + baseFee, GasFeeCap)
```

비인가 계정은 **tx 가 요청한 tip 이 버려지고 헤더 값이 강제된다.** 그 값이 genesis
템플릿의 `gasTip: 27600000000000`(27,600 gwei)이다. baseFee 쪽은 한산하면 바닥까지
계속 내려가므로(`CalcBaseFee` 의 `parent.GasUsed < decreasingTarget` 가지),
stablenet 에서 비용을 지배하는 것은 baseFee 가 아니라 **고정 tip** 이다. 그래서
케이스가 10만 gwei 를 적어 둔다.

wbft 에는 그 구조가 아예 없다 — `TransactionToMessage` 에 `headerGasTip` 인자가
없고 tx 자신의 tip 을 쓴다. 필요한 수수료가 ~1 gwei 인데 케이스는 **10만 배**를
적고 있고, `checkTxFee` 가 `gasPrice × gas` 로 보므로 2.1 ether 가 되어 gwbft 의
상한 1 ether 에 걸린다.

**그러니 이 넷은 체인 전용 테스트가 아니다.** nonce 순서와 tx 교체는 어느 EVM
에서나 같다. stablenet 에 묶어 놓은 것은 **케이스에 박힌 수수료 값 하나**다.

**그리고 그 값은 그냥 낮추면 됐다.** 산수가 답을 줬다.

```
stablenet 이 강제하는 tip  2.76e13 x 21000 = 0.58 ether
gwbft 의 상한                              = 1.00 ether
```

**고정 tip 을 다 내고도 상한 아래다.** 즉 두 제약을 동시에 만족하는 값이 있고,
케이스가 적어 둔 1e14(2.10 ether)는 필요보다 컸을 뿐이다. `feeCap` 만
`3.5e13`(0.735 ether)과 `4.2e13`(0.882 ether)으로 낮췄다 — tip 은 건드리지 않았다.
stablenet 에서는 어차피 버려지고, wbft 에서는 그것이 교체를 교체로 만든다.

**세 체인 전부 통과한다(실측).** 게이트를 지웠다.

P2·P3 이 열리면 이 값도 체인에서 오게 하는 것이 맞다. 다만 **그때까지 막혀 있을
이유는 없었다** — "표현할 수 없다" 고 적고 멈출 뻔했는데, 값을 한 번 계산해 보니
아니었다.

**`wbft` 4건 — stablenet 에.** `01-wbft-govcontracts-at-genesis` 는
`contract:govConfig`·`contract:govStaking` 으로 옮겼다(wbft 만 선언한다. stablenet
에서 스킵, wbft 에서 통과 실측). 나머지 셋(secp256r1 프리컴파일)은 **남긴다** —
"이 체인은 RIP-7212 프리컴파일이 있다" 를 말할 능력이 매니페스트에 없다. 셋 중
둘은 stablenet 에서 "통과" 하는데, 프리컴파일이 없어서 부정 케이스가 우연히
맞는 것이라 **넓히면 안 된다.**

**가드 하나를 넓혔다.** `TestCorpus_ACaseNamingASystemContractSaysWhereItRuns`
가 `applicableChains` 만 알고 있어서, 프리컴파일 주소를 쓰는 케이스가 게이트를
`engine:anzeon` 으로 바꾸자 걸렸다. 어느 체인도 선언하지 않은 주소는 대개
프리컴파일이고 프리컴파일은 엔진의 것이므로, `engine:` 요구를 받아들이게 했다.
`family:`·`fork:` 는 안 받는다 — 둘 다 체인 둘을 허용하고, 생주소는 그 둘에서
다른 것을 뜻할 수 있다.

**곁가지 — wemix 매니페스트가 부정확하다.** `tx_types` 에 `0x04`(setCode)를
적어 놨는데 `18-set-code-delegation` 이 wemix 에서
`transaction type not supported` 로 떨어진다. 그래서 이 케이스를 tx 타입 능력으로
게이트할 수 없어 `family:wbft` 로 뒀다. 매니페스트를 고치는 것이 맞는 순서다.

### 남은 둘을 마저 풀었다 (2026-09-18) — 7 → 0

**wemix 매니페스트의 tx 타입이 틀렸다.** go-wemix 가 실제로 정의하는 것은 넷뿐이고,
디코더가 그 밖을 거부한다.

```go
// core/types/transaction.go — 상수
LegacyTxType = iota               // 0x00
AccessListTxType                  // 0x01
DynamicFeeTxType                  // 0x02
FeeDelegateDynamicFeeTxType = 22  // 0x16

// decodeTyped — 받는 것이 전부다
switch b[0] {
case AccessListTxType, DynamicFeeTxType, FeeDelegateDynamicFeeTxType: ...
default: return nil, ErrTxTypeNotSupported
}
```

`BlobTx`·`SetCodeTx` 는 **타입 정의 자체가 `core/types/` 어디에도 없다.**
stablenet·wbft 는 둘 다 `case BlobTxType:`·`case SetCodeTxType:` 를 갖고 있다.
매니페스트는 여섯을 적고 있었고, 그래서 `18-set-code-delegation` 이 wemix 에서
`transaction type not supported` 로 떨어졌다. 둘을 뺐다.

**secp256r1 셋은 프리컴파일 능력을 만들어 풀었다.** 저장소를 읽어 보니 두 체인
**다** `p256Verify` 를 빌드하는데, EVM 이 **포크에 따라 다른 세트를 고른다**.

```go
// go-stablenet core/vm/evm.go
case evm.chainRules.IsBoho:      precompiles = PrecompiledContractsBoho      // 0x100 있음
case evm.chainRules.IsAnzeon:    precompiles = PrecompiledContractsAnzeon    // 0x100 없음
// go-wbft
case evm.chainRules.IsCroissant: precompiles = PrecompiledContractsCroissant // 0x100 있음
```

`PrecompiledContractsAnzeon` 에 `0x100` 이 없는 것을 직접 셌다(0건). boho 가 꺼진
기본 stablenet 망은 anzeon 세트로 떨어지므로 프리컴파일이 없다. 그리고 기본
genesis 는 이렇다.

```
wbft       croissantBlock: 0   → 산다
stablenet  bohoBlock 없음      → 안 산다
```

**빌드가 가진 것과 망에서 사는 것이 다르다.** `fork:croissant` 로 적으면
"크로아상이 필요하다" 는 거짓말이 된다 — 필요한 것은 프리컴파일이다. 그래서
`precompile:` 접두사를 만들고 wbft 매니페스트가 선언하게 했다. 실측: wbft 에서
`pass=1` 셋, stablenet 에서 `skip=1` 셋. boho 를 켠 stablenet 망은 env 의
`capabilities` 로 직접 알릴 수 있고, 그 통로는 이미 있다.

**가드를 한 번 더 넓혔다.** 어느 체인도 컨트랙트로 선언하지 않은 주소에 대해
`precompile:` 이 가장 정확한 답이다 — 그 자리에 사는 바로 그것을 지목한다.
`engine:` 도 계속 받는다.

**남은 것은 없다. 66 → 0.** `tests/tc` 의 어떤 케이스도 체인 이름으로 게이트되지
않는다.

### P1 이 한 것과 하지 않은 것

측정해 두지 않으면 다음 작업이 무엇을 남겼는지 알 수 없다. 205건 기준이다.

| 완료 조건 | 전 | 후 |
|---|---|---|
| (1) 메인넷별 설정을 따로 관리 | 별도 파일 참조 0건 | **205건** (17개 선언) |
| (2) 공통 테스트 코드에 메인넷 값 없음 | 체인 이름을 문장에 든 케이스 205 | **205** (줄지 않았다) |
| (3) 실행 시점에 주입 | 0% | 0% |
| (4) 설정만 추가하면 재사용 | 0% | 0% |

**(2) 가 줄지 않은 것이 P1 의 한계다.** 케이스가 여전히 `"env": "stablenet-bp4"` 라고
말한다. 체인 이름이 케이스 안에 있으므로, 같은 케이스를 wbft 에서 돌리려면 케이스를
고쳐야 한다. 남은 자리는 셋이다.

| 자리 | 건수 |
|---|---|
| env 참조(`extends` 또는 id) | 205 |
| `applicableChains` | 134 |
| 케이스 id·description | 37 · 23 |

**steps 안에는 체인 이름이 하나도 없다.** 남은 것은 전부 머리말이다.

(2)(4) 를 끝내려면 케이스가 체인을 이름으로 고르지 말아야 한다. 둘이 같이 가야 한다 —
실행할 때 망을 바꿔 넣고(P6, 완료), `applicableChains` 를 능력 요구로 바꾸는 것(P5)이다.

### P6 — 실행 시점 주입 (완료 2026-09-15)

`suite run --env <id|경로>` 가 케이스가 이름으로 부른 선언 대신 다른 선언 위에 올린다.
케이스가 덮은 것은 남는다 — 그건 체인이 아니라 그 테스트의 것이기 때문이다.

```
chainbench run --plan --workspace-dir /tmp/ws --env wbft-bp4 tests/tc/.../08-legacy-transfer.json
  chain      wbft        ← 정의서는 stablenet 이라고 쓰여 있다
  binary     gwbft
```

`dsl.UseEnv` 가 참조를 바꾸고 `dsl.ReadFilesWithEnv` 가 읽는다. 인라인 env 를 가진
케이스는 **거부한다** — 인라인 객체가 곧 그 케이스의 선언이라, 갈아끼우면 무엇이
중요했는지 알 수 없는 채로 버리게 된다. 값이 경로처럼 생겼으면(구분자가 있거나
`.json` 으로 끝나면) 파일로 읽는다. 디스크를 보고 정하지 않으므로 같은 입력이 늘 같은
뜻이다.

**이 기능이 남은 장벽을 처음으로 측정했다.** 205건을 통째로 다른 체인에 올려 봤다.

| | 계획이 나오는가 | `applicableChains` 를 통과하는가 |
|---|---|---|
| 그대로 (stablenet) | 205 | 202 |
| `--env wbft-bp4` | **204** | **104** |
| `--env wemix-bp4` | **204** | **72** |

계획이 안 나오는 1건은 핸드오프 케이스다. 포크 양쪽의 바이너리를 요구하므로 평범한
체인 선언 위에 올릴 수 없고, 거부 문구가 `binaries.from is missing` 이라고 말한다.
이건 결함이 아니라 성질이다.

**막고 있는 것은 사실상 하나다.** `applicableChains: "stablenet"` 한 값이 101건에
붙어 있다.

| 값 | 건수 |
|---|---|
| `stablenet` | **101** |
| (없음) | 71 |
| `stablenet,wbft` | 29 |
| `wbft` | 3 |
| `stablenet,wbft,wemix` | 1 |

그 101건이 진짜 stablenet 전용인지, 아니면 그렇게 적혀만 있는지는 아직 모른다.
**P5 가 답해야 할 질문이 그것이다.**

---

## 7. 하드코딩 제거 (H)

### 7.0 P5 와 H 를 함께 조사한 결과 (2026-09-15)

두 묶음은 같은 것을 다른 각도에서 본다. **"이 케이스가 왜 stablenet 전용인가" 의
답이 대개 "stablenet 시스템 컨트랙트 주소를 박아 뒀기 때문" 이다.** 그래서 한 번에
쟀다.

#### 주소 47개, 파일 138개

JSON 을 순회해 값이 정확히 40자리 hex 인 것만 셌다. 정규식으로 원문을 훑으면 이벤트
토픽·셀렉터·바이트코드가 섞인다.

| 갈래 | 값 | 쓰임 | 파일 |
|---|---|---|---|
| 낮은 주소 전체 | 32 | 280 | 106 |
| └ **체인 자체의 컨트랙트** | | | **74** |
| └ `0x…c0ffee**` (테스트가 배포) | | | 나머지 |
| └ 프리컴파일 `0x…0001` | | 2 | 2 |
| 계정 | 14 | 84 | 44 |
| zero | 1 | 5 | 3 |

**셋을 갈라야 한다.** `0x…1000`~`0x…1004`, `0x…b00003`, `0x…0100` 이 체인 자체의
컨트랙트다. `0x…c0ffee**` 는 테스트가 직접 배포하는 표식이라 체인과 무관하고,
`0x…0001` 은 ecrecover 프리컴파일이라 모든 EVM 에 있다. 처음 조사에서 프리컴파일을
시스템 컨트랙트로 세어 게이트 없는 케이스를 하나 더 많게 보고했다(10 → **9**).

#### `applicableChains: "stablenet"` 101건의 분해

| 분류 | 건수 |
|---|---|
| 시스템 컨트랙트 주소를 든다 | **62** |
| 포크·기능 단서는 있으나 주소는 없다 | 25 |
| 단서가 없다 | 14 |

**62 는 H1 이 적어 둔 파일 수와 정확히 같다.** 즉 P5 가 풀어야 할 101건의 절반 이상이
H1 이 치울 대상이고, 둘은 같은 작업의 앞뒤다.

단서가 없는 14건은 nonce 순서, tx 교체, revert, out-of-gas, ws 구독 같은 **평범한 EVM
동작**이다. stablenet 에서 쓰여서 그렇게 적혔을 뿐으로 보인다. 다만 그중 둘은 낱말
검색으로는 안 잡히는 stablenet 규칙이다 — "최소 가스비 하한선"(anzeon 수수료 정책)과
"0x0 주소 전송 거부". **낱말 검색은 근거가 약하다. 14건은 한 건씩 읽어야 한다.**

#### 게이트가 없는 10건

`applicableChains` 가 **비어 있는데 체인 자체의 컨트랙트 주소를 든 케이스가 9건**
있었다. 다른 체인에 올리면 돌다가 깨진다. 지금은 아무도 다른 체인에 올리지 않아
드러나지 않았을 뿐이다(P6 가 그걸 가능하게 만들었다). **X8 에서 해소했다.**

#### P5 를 막는 것: 매니페스트가 체인을 구분하지 않는다

세 체인의 `capabilities` 가 **완전히 같다** — `["process","rpc","ws","consensus"]`.
`tx_types` 도 같다. 즉 **능력 표시는 지금 체인 구분 정보를 하나도 담고 있지 않고**,
그래서 `applicableChains` 가 존재한다.

실제로 다른 것은 이쪽이다.

| 필드 | stablenet | wbft | wemix |
|---|---|---|---|
| `genesis.engine_field` | `anzeon` | `croissant` | (없음) |
| `genesis.hardforks` | istanbul, boho | istanbul, pangyo, applepie, brioche, croissant | istanbul, pangyo, applepie, brioche |
| `consensus.rpc_namespace` | istanbul | istanbul | wemix |
| `consensus_family` | wbft | wbft | poa |
| `bootstrap` | static | static | governance-etcd |

**P5 는 매니페스트가 자기 능력을 말하게 하는 것부터 시작한다.** 케이스가 요구할 말이
없으면 요구로 바꿀 수 없다.

#### H3 은 기계가 이미 있다

계정 주소 14개 중 5개가 `keys/preset` 의 node1~node5 다. 가장 많이 쓰인
`0xc17d4938…`(30회)는 node1 이다. 그리고 **DSL 은 이미 계정을 라벨로 부른다** —
`"from": "node1"` 이 키셋의 주소로 풀린다(`internal/testengine/accounts.go`,
`accountlabel_live_test.go`). H3 은 새 문법이 필요 없고 치환이다.

#### 7.0.1 매니페스트 능력 설계 (P5 의 전제)

##### 조사가 뒤집은 사실 하나

같은 주소가 체인마다 **다른 컨트랙트**를 담고 있다.

| 주소 | stablenet | wbft |
|---|---|---|
| `0x…1000` | nativeCoinAdapter | govConfig |
| `0x…1001` | govValidator | govStaking |
| `0x…1002` | govMasterMinter | govRewardeeImp |
| `0x…1003` | govMinter | govNCP |
| `0x…1004` | govCouncil | (없음) |

X8 을 하면서 "0x…1000~1003 은 두 체인 genesis 에 다 있으니 케이스를 둘 다에서
돌려도 되겠다" 고 잠깐 생각했는데, **틀렸다.** 주소는 같고 컨트랙트가 다르다. 그
케이스들을 `wbft` 로 좁힌 것이 맞았고, 넓혔으면 조용히 엉뚱한 컨트랙트를 불렀을
것이다.

**이 사실이 설계를 정한다. 케이스는 주소가 아니라 컨트랙트를 이름으로 불러야 한다.**

##### 원칙 — 있는 데이터를 다시 선언하지 않는다

능력은 세 군데서 온다. 어디서 오는지에 따라 다루는 법이 다르다.

| 출처 | 예 | 어떻게 |
|---|---|---|
| 매니페스트에 **이미 있는** 데이터 | `genesis.hardforks`, `genesis.engine_field`, `consensus_family`, `tx_types` | **파생한다.** `capabilities` 에 다시 적으면 같은 사실이 두 곳에 남는다 |
| 체인만 아는 것 | 시스템 컨트랙트 이름표, 바이너리가 주는 AccountManager | **매니페스트에 새로 선언한다** |
| 그 실행이 만든 것 | overlay, 지연 포크 | **이미 동작한다** (`networkCapabilities`) |

##### 더할 것: 매니페스트가 자기 컨트랙트를 이름으로 말한다

```json
"system_contracts": {
  "govValidator":      "0x0000000000000000000000000000000000001001",
  "nativeCoinAdapter": "0x0000000000000000000000000000000000001000",
  "govMasterMinter":   "0x0000000000000000000000000000000000001002",
  "govMinter":         "0x0000000000000000000000000000000000001003",
  "govCouncil":        "0x0000000000000000000000000000000000001004",
  "accountManager":    "0x00000000000000000000000000000000000b00003"
}
```

genesis 템플릿이 이미 같은 이름과 주소를 담고 있으므로, **둘이 어긋나면 실패하는
테스트를 같이 넣는다.** wemix 는 템플릿이 없고 컨트랙트를 실행 중에 배포하므로
매니페스트가 유일한 자리다. AccountManager 는 genesis 가 아니라 바이너리가 주므로
어차피 템플릿에 없다.

##### 파생하는 능력의 문법

접두사로 갈래를 밝힌다. 접두사가 없으면 지금처럼 하니스가 주는 능력이다.

| 형태 | 어디서 | 예 |
|---|---|---|
| `contract:<이름>` | `system_contracts` 의 키 | `contract:govMinter` |
| `fork:<이름>` | `genesis.hardforks` | `fork:boho` |
| `engine:<이름>` | `genesis.engine_field` | `engine:anzeon` |
| `family:<이름>` | `consensus_family` | `family:wbft` |
| `tx:<타입>` | `tx_types` | `tx:0x16` |
| `precompile:<이름>` | (아직 없다 — 아래) | `precompile:secp256r1` |
| (접두사 없음) | `capabilities` · overlay · 지연 포크 | `rpc`, `ws`, `account-extra`, `delayed-boho` |

`networkCapabilities` 가 이 목록을 만든다. 게이트(`satisfies`)와 SKIP 사유는 이미
있으므로 배선만 는다.

##### 케이스가 바뀌는 모양

```json
"applicableChains": "stablenet"          →   "requires": ["contract:govMinter"]
"to": "0x0000…1003"                      →   "to": "govMinter"
```

**한 번의 선언이 두 가지를 동시에 한다.** 게이트가 "이 체인에 govMinter 가 있는가"
로 바뀌고, 주소 리터럴이 사라진다. 그래서 P5 와 H1 은 같은 커밋이다.

이름이 겹치는 문제는 `ResolveAccount` 가 이미 푸는 방식대로 푼다 — 모양으로 가른다.
`0x…` 는 주소, 계정 라벨은 키셋에 있는 이름, 컨트랙트는 매니페스트에 있는 이름.
셋 중 어디에도 없으면 오류이고, 그 오류가 무엇을 찾아봤는지 말한다.

##### 1단계에서 나온 것: 프리컴파일도 체인마다 다르다

`0x…0100` 은 secp256r1(P-256, RIP-7212) 프리컴파일이다. 프리컴파일이라 시스템
컨트랙트가 아닌데, **모든 EVM 에 있는 것도 아니다.** 지금은 그것을 쓰는 케이스 3건이
`applicableChains: "wbft"` 로 막혀 있다.

`system_contracts` 에 넣으면 이름이 거짓말을 한다. 매니페스트에 `precompiles` 를
따로 두고 `precompile:<이름>` 으로 파생하는 것이 맞아 보이는데, **3건뿐이라 지금
정하지 않는다.** 4단계에서 그 3건을 만질 때 같이 정한다.

#### 7.0.2 4단계를 어떻게 검증할 것인가

1~3단계는 동작을 안 바꿨으므로 "전체 테스트 통과" 로 충분했다. **4단계는 다르다.**
62건의 케이스가 바뀌고, `--plan` 으로는 못 잡는다 — 계획에 steps 가 없어서 주소가
바뀌었는지 보이지 않는다.

##### 무엇이 조용히 틀릴 수 있나

| 위험 | 왜 안 보이나 |
|---|---|
| 이름을 잘못 골랐다 (`0x…1004` 를 `govMinter` 로) | 둘 다 해석되고, 컨트랙트가 다를 뿐이다 |
| 해석기가 안 닿는 자리에 이름을 넣었다 | H3 에서 실제로 저질렀다(`params` 안) |
| 게이트가 좁아져 돌던 케이스가 SKIP 된다 | SKIP 은 실패가 아니라 조용하다 |
| 게이트가 넓어져 못 돌 데서 돌다 깨진다 | X8 이 바로 그것이었다 |

**앞의 둘은 기계가 판정할 수 있고, 뒤의 둘은 판단이 필요하다.** 그래서 4단계를 둘로
가른다.

##### 4a — 주소를 이름으로 (기계가 판정)

주소 해석은 **오프라인으로 끝까지 돌릴 수 있다.** 확인했다: `keys/preset` 을
`LoadPresetWithKeys` 로 읽어 키셋을 만들고 매니페스트의 컨트랙트 표를 붙이면,
`ResolveAddress` 가 망 없이 답한다.

```
node1      -> 0xc17d…f9d8
faucet     -> 0xc17d…f9d8
govMinter  -> 0x0000…1003
```

그래서 **골든 파일**을 둔다. 모든 케이스의 모든 주소 자리를 해석해 `케이스 → 자리 →
주소` 로 적어 커밋한다. 4a 는 **그 파일을 한 줄도 바꾸면 안 된다.** 바뀌면 이름을
잘못 골랐거나 해석기가 안 닿는 자리에 넣은 것이다.

골든이 남는 이유는 마이그레이션 뒤에도 값이 있기 때문이다. 그 파일은 **이 코퍼스가
건드리는 주소의 전량 기록**이고, 이름이 여전히 같은 것을 가리키는지를 계속 지킨다.
오타 난 계정 라벨·컨트랙트 이름도 실행 전에 잡는다.

**단점**: 주소를 정당하게 바꿀 때마다 골든도 고쳐야 한다. 그게 목적이기도 하다 —
주소 변경은 보여야 한다.

##### 4b — 게이트를 옮긴다 (사람이 판정)

게이트는 **새 도구가 필요 없다.** `validate --chain <체인>` 이 이미 케이스마다
OK/SKIP 을 답한다. 세 체인에 대해 돌려 표를 뜨면 그것이 전후 비교다.

지금 기준선은 이렇다(205건).

| 체인 | OK | SKIP |
|---|---|---|
| stablenet | 194 | 11 |
| wbft | 97 | 108 |
| wemix | 63 | 142 |

규칙은 대칭이 아니다.

- **돌던 것이 멈추면 안 된다.** 그 케이스의 원래 체인에서 OK 였다면 뒤에도 OK 여야
  한다. 이건 기계가 판정한다.
- **새로 도는 것은 한 건씩 근거를 댄다.** 넓어지는 것이 이 작업의 목적이지만,
  넓어져서 깨진 것이 X8 이었다. 자동 통과시키지 않는다.

그래서 4b 는 **한 커밋에 다 하지 않고 묶음으로 나눈다.** 디렉터리 단위(anzeon,
system-contracts, extra-state, …)면 한 묶음이 5~15건이라 디프를 눈으로 읽을 수 있다.

##### 왜 4a 는 한 커밋이고 4b 는 나누나

4a 는 **같음** 을 요구하므로 62건을 한 번에 바꿔도 기계가 전부 검사한다. 4b 는
**달라지는 것이 목적**이라 기계가 "맞다" 고 말해 줄 수 없고, 사람이 읽을 수 있는
크기로 잘라야 한다.

##### 이 검증이 못 잡는 것

**실제로 돌려 보지 않았다는 사실은 그대로다.** 골든은 주소가 같다는 것까지만 말하고,
그 주소의 컨트랙트가 그 케이스가 기대하는 ABI 를 갖는지는 말하지 않는다. wbft 로
넓힌 케이스가 실제로 통과하는지는 **라이브 검증에서만** 알 수 있다. 4b 의 각 묶음은
그 점을 커밋 메시지에 적고 넘어간다.

#### 7.0.3 케이스별 판정표 — 어느 층에서 따질지 정하는 법

묶음 2 에서 "컨트랙트를 부른다고 그것이 이유의 전부는 아니다" 를 배운 뒤, 남은 107건을
추측 없이 처리하려고 절차를 세웠다. **판정표는 `case-gate-table.md` 에 있다.**

##### 절차

| | 단계 | 누가 |
|---|---|---|
| 1 | 능력마다 **제공 체인 수**를 세어 좁은 것부터 나열 | 기계 |
| 2 | 케이스마다 **증거**를 뽑는다 (컨트랙트·RPC 메서드·genesis 키·거부 사유·선언 능력) | 기계 |
| 3 | 증거를 보고 **가장 좁은 충분조건**을 정한다. 표현 불가면 그렇게 적는다 | 사람 |
| 4 | 표대로 묶음을 나눠 처리한다 | — |

**1번을 층 이름으로 하면 틀린다.** 세어 보니 `fork:boho` 는 1개 체인이고
`fork:istanbul` 은 3개다. 넓이는 층이 아니라 **그 값**의 성질이라, 제공 체인 수로
세어야 한다.

**3번은 결론만이 아니라 근거를 같이 적는다.** 근거 없는 칸은 나중에 아무도 검증할 수
없다.

##### 결과 (107건)

| 처리 | 건수 |
|---|---|
| 변환 가능 | 58 |
| 게이트 불필요 | 21 |
| 표현 불가 | 16 |
| 보류 (X9) | 6 |
| 보류 (remote) | 3 |
| 값 (P2) | 2 |
| 판단 필요 | 1 |

**"표현 불가" 16건이 이 작업의 한계선이다.** 바이너리의 거부 규칙(6), 프리컴파일
존재(4), EIP-7702 가스 비용, RPC 메서드 존재, `account-extra` 자기만족, 값 의존.
**억지로 층에 욱여넣으면 게이트는 통과하고 케이스는 실패하는 상태가 된다** — 지금 없는
문제를 새로 만드는 것이라 하지 않는다.

##### 적용 (2026-09-15)

`contract:`·`engine:`·`fork:` 로 판정된 41건을 옮겼다. 판정 변화 0. `applicableChains`
를 든 케이스 107 → **66**.

적용하면서 배운 셋은 `case-gate-table.md` §5 에 있다. 요약하면: 기존
`applicableChains` 값도 증거이고(추론이 그것을 조용히 뒤집으면 안 된다), 주소에서
이름을 되찾을 때는 그 케이스의 체인 표를 써야 하며(`0x…1003` 은 체인마다 다른
컨트랙트다), 요구 목록은 손판정이 아니라 케이스가 실제로 쓰는 것에서 뽑아야 한다.

X8 가드도 고쳤다. **컨트랙트 요구도 게이트로 인정**하되, **케이스가 쓴 주소와 요구한
컨트랙트가 같아야** 한다. 이 규칙이 내 손판정 오류 둘을 잡았다.

##### 조사로 나온 결함 (X9)

stablenet 케이스 8건이 오버레이로 `applepieBlock: 0` 을 켜는데, **stablenet 매니페스트의
포크 목록에는 applepie 가 없다**(`["istanbul","boho"]`). 그런데 go-stablenet 은 그 포크를
안다 — `internal/testengine/suite.go` 에 실측이 적혀 있다: "all four running nodes
reported `Applepie: #<nil>`". 노드가 포크 표에 찍었다는 것은 바이너리가 안다는 뜻이다.

**즉 매니페스트의 포크 목록이 불완전하다.** 2단계가 그 목록에서 `fork:` 능력을
파생하므로, 지금 `fork:applepie` 로 게이트하면 그 8건이 stablenet 에서 SKIP 된다.
매니페스트를 먼저 고쳐야 하고, 고치려면 go-stablenet 의 포크 목록을 확인해야 한다.

---

##### 이 설계의 단점

- **케이스가 접두사를 외워야 한다.** `contract:govMinter` 는 `stablenet` 보다 길고,
  처음 쓰는 사람에게 덜 분명하다. `chainbench validate` 가 모르는 접두사를 거부하고
  아는 목록을 보여 주는 것으로 덜어야 한다.
- **모든 것을 능력으로 바꿀 수는 없다.** "anzeon 수수료 정책이 basefee 하한을
  강제한다" 는 `engine:anzeon` 으로 표현되지만, 하한 값 자체는 아니다. 값이 필요한
  케이스는 P2(`values`)가 답한다.
- **파생은 매니페스트를 신뢰한다.** `genesis.hardforks` 가 실제 바이너리와 다르면
  게이트가 틀린 답을 낸다. 지금도 그 위험은 같으므로 새로 생기는 위험은 아니지만,
  게이트가 그 위에 얹히면 틀렸을 때의 결과가 커진다.

##### 착수 단위

1. ~~`system_contracts` 를 세 매니페스트에 더하고, genesis 템플릿과 대조하는 테스트.~~
   **완료 (2026-09-15).** stablenet 6개(genesis 5 + 바이너리가 주는 `accountManager`),
   wbft 4개, wemix 0개(거버넌스 주소를 `admin_wemixInfo` 로 실행 중에 읽으므로 선언할
   고정 주소가 없다 — 그래서 `contract:` 요구는 wemix 에서 SKIP 되는 것이 맞다).
   대조 테스트는 템플릿의 치환 토큰을 채워 파싱한 뒤 컨트랙트 블록을 읽는다
   (stablenet 은 `systemContracts`, wbft 는 `govContracts` 로 이름이 다르다).
   변이 둘로 실효성을 확인했다 — 주소를 틀리게 해도, 이름을 빠뜨려도 잡는다.
   **아무 동작도 바뀌지 않았다.**
2. ~~`networkCapabilities` 가 파생 능력을 내놓게 하고, `validate` 가 모르는 요구를
   거부하게 한다.~~ **완료 (2026-09-15).** `Manifest.DerivedCapabilities()` 가 규칙을
   갖고(사실이 사는 곳에 규칙을 둔다), 합성과 `validate` 가 그것을 쓴다.
   `MalformedCapability` 는 **오타와 안 맞는 요구를 가른다** — 안 맞는 요구는 SKIP 이
   정상이지만, 접두사 오타는 모든 체인에서 SKIP 이라 올바르게 걸러진 케이스와 구별되지
   않는다. 그래서 오타는 `validate` 와 `Precheck` 이 거부한다.
   확인: `contract:govMinter` 는 stablenet 에서 OK, wbft·wemix 에서 SKIP.
   205건 전부 통과하고 **아무 케이스도 아직 접두사를 쓰지 않으므로 동작이 안 바뀌었다.**
3. ~~컨트랙트 이름을 주소 자리에서 푼다(H1 의 기계).~~ **완료 (2026-09-15).**
   `ResolveAddress` 가 주소 자리를 푼다 — 0x 리터럴, 키셋의 계정 라벨, 체인이 선언한
   컨트랙트 이름 셋이다. 못 찾으면 **세 곳을 다 대며** 거부한다.
   **서명 자리와 주소 자리를 갈랐다.** `from`·`deployer`·`funder` 는 키셋만 본다 —
   컨트랙트에는 키가 없으므로 그 이름이 서명 자리에 서면 노드가 "unknown account" 로
   답하고, 그 문구는 이름이 아니라 주소를 말해서 원인을 가린다. `to`·`address` 와
   read 의 `of` 목록이 컨트랙트를 받는다.
   이름이 겹치면 **계정이 이긴다** — 키셋 라벨은 그 실행이 만든 것이고 컨트랙트 표는
   체인이 고정한 것이라, 충돌을 만든 실행은 방금 만든 쪽을 뜻한 것이다.
   **케이스는 아직 안 고쳤다.**
4. **4a 완료 (2026-09-15)** — 주소 197곳을 이름으로 바꾸고 H4 래칫을 넣었다.
   골든이 한 줄도 안 바뀌었다. 검증 설계는 §7.0.2.

   **4b 묶음 1 (system-contracts, 23건) 완료 (2026-09-15).**
   4a 가 주소를 이름으로 바꿔 둔 덕에 **케이스가 무엇을 필요로 하는지 파일에서 읽힌다** —
   `requires` 는 그 케이스가 부르는 컨트랙트 목록이다. `applicableChains` 를 지웠다.
   **615줄 판정표에서 OK/SKIP 이 바뀐 줄이 0이다.** 바뀐 것은 SKIP 의 사유뿐이다:
   `SKIP (chain not applicable)` → `SKIP (needs caps: contract:govMinter)`.
   메커니즘을 동작 변화 없이 증명하는 묶음이라 첫 번째로 골랐다.
   `applicableChains` 를 든 케이스 143 → **120**.

   **묶음 2 (blacklist-authorized 8건 + extra-state 6건) 완료 (2026-09-15).**
   같은 모양이고 판정 변화도 0이다. 120 → **107**.

   **여기서 규칙을 하나 배웠다. 컨트랙트를 부른다고 해서 그것이 그 케이스가
   체인에 매인 이유의 전부는 아니다.** `06-precompile-transfer-rejected` 는
   `accountManager` 를 부르므로 기계적으로는 변환 대상인데, 실제로 검증하는 것은
   **"프리컴파일 주소로의 값 전송을 바이너리가 거부한다"** 는 규칙이다.
   `applicableChains` 를 지우자 **X8 가드가 잡았다** — 게이트 없이 `0x…0100` 을
   든다고. 되돌렸다.

   그래서 묶음의 판정 절차는 둘이다: 판정표 비교로 **좁아지지 않았는지** 보고,
   X8 가드로 **넓어지면 안 될 것이 넓어지지 않았는지** 본다. 후자가 실제로 일했다.

   **아직 이름이 없는 의존 3종을 남겼다** — 바이너리의 거부 규칙(zero 주소 전송,
   프리컴파일 전송), Extra 비트맵 지원(`account-extra` 는 케이스가 스스로 선언하므로
   게이트 구실을 못 한다), 그리고 프리컴파일 존재 여부. 셋 다 `contract:` 로 표현할 수
   없고, 추측으로 이름을 붙이면 아무도 검사할 수 없는 게이트가 된다.
5. 남은 39건을 한 건씩 읽어 판정한다. 낱말 검색으로는 안 되는 자리다.

#### 순서

1. ~~**H3 먼저.**~~ **완료 (2026-09-15).** 35곳/20파일. 남은 14곳은 H3-b.
2. ~~**X8.**~~ **완료 (2026-09-15).** 9건에 게이트를 달고 재발 방지 테스트를 넣었다.
3. **매니페스트 능력.** 세 체인이 실제로 다른 것을 말하게 한다. P5 의 전제다.
4. **H1 + P5 + H4 를 한 묶음으로.** 62건이 겹치므로 따로 하면 같은 파일을 두 번
   건드린다. H4(arch 테스트)는 H1 과 같이 넣어 되돌아가지 않게 한다.
5. **H2.** gasTip 14건. 12건이 stablenet 이라 P5 뒤에 보는 것이 낫다.

---

| ID | 항목 | 규모 | 의존 | 상태 |
|---|---|---|---|---|
| H1 | 시스템 컨트랙트 주소를 이름으로 바꾼다 | **197곳/65파일 완료 (2026-09-15).** 케이스가 `"to": "govMinter"` 라고 쓰고, 그 체인의 표가 주소로 푼다. 골든이 **한 줄도 안 바뀌었다** — 197곳 전부가 전과 같은 주소로 풀린다 | ~~P3~~, ~~매니페스트 능력~~ | **완료** |
| H2 | 수수료 힌트를 값 또는 체인 어댑터로 돌린다 | 14개 파일 (12건이 stablenet) | P3, P5 | 미착수 |
| H3 | 계정 주소 리터럴을 라벨로 바꾼다 | **35곳/20파일 완료.** `keys/preset` 의 node1~5 주소가 49곳에 있었고, 그중 35곳이 해석기가 닿는 자리(`address`·`from`·`to`·`deployer`·`funder`, 그리고 read 의 `of` 목록)였다. `of` 는 이번에 열었다 | ~~없음~~ | **완료 (2026-09-15)** — 남은 14곳은 H3-b |
| H4 | arch 테스트로 주소 리터럴을 금지한다 | **래칫 1단계 완료 (2026-09-15).** "주소 리터럴 전면 금지" 가 아니다 — 이름이 없는 주소가 많다(테스트가 배포한 표식, 프리컴파일, 임의 수신자). 금지하는 것은 **체인이 이름을 가진 컨트랙트를, 이름이 통하는 자리에 주소로 쓰는 것**이다. 매니페스트에 컨트랙트를 더할 때마다 검사가 저절로 좁아진다. **2단계(허용 목록 줄이기)가 열려 있다 — §11.5** | ~~H1~~ | **1단계 완료, 2단계 대기** |

#### H3 이 남긴 14곳 (H3-b) — 완료 (2026-09-18)

**결정한 규칙: `of` 쪽이다.** 키 집합이 아는 이름이면 풀고, 아니면 적힌 그대로 둔다.
비교값과 원시 인자는 숫자·hex·불리언·블록 태그·이미 치환된 바인딩을 같이 담으므로,
못 푸는 것이 잘못이 아니다. 주소 인자만 못 풀면 오류로 남는다 — 그쪽은 반드시
계정이어야 하고, 오타가 값을 허공에 보내면 안 되기 때문이다.

**자리를 추측하지 않는 방법은 자리를 안 보는 것이었다.** `eth_getBalance` 는 첫
인자가 주소이고 `eth_createAccessList` 는 트랜잭션 객체 안에 있다. 그런데 어느
자리인지 알 필요가 없다 — `node1` 은 블록 태그도, 수량도, 해시도 될 수 없으므로
**풀리는 값은 계정으로 쓴 것**이다. 리스트와 객체를 재귀로 걸어 들어가면 tx 객체
안의 `.from` 까지 닿는다.

| 자리 | 곳 | 결과 |
|---|---|---|
| `waitFor.expected` · `rpc.is` · `rpc.is[]` | 9 | 라벨로 바꿨다 |
| `read.params[]` · `rpc.params[]` | 4 | 라벨로 바꿨다 |
| `params[0]` 안의 tx 객체 `.from` | 1 | 라벨로 바꿨다 |

**`tests/tc` 에 preset 주소 리터럴이 0곳이다** (49 → 35 → 0).

**규칙을 넓히자 소스 가드가 결함 여섯을 바로 잡았다.** `expected` 와 `params` 를
`addressShapedKeys` 에 넣으니, 그 값을 spec 에서 꺼내 쓰면서 해석기에 안 넘기는
함수가 여섯 나왔다. 그중 `waitFor` 와 `rpcAssertion` 은 **해석을 해 놓고 원본을
읽고 있었다** — X12 와 같은 모양이다. `rpcAssertion` 은 대상 노드마다 같은 spec 을
다시 풀면서 비교값은 끝내 안 풀었다.

그리고 `knownLabels` 가 키 집합이 없는 실행(attach, 단위 테스트)에서 **패닉했다.**
오류 문구를 만드는 쪽만 nil 검사가 빠져 있었다. 이 작업이 그 경로를 처음 밟았다.

주소 골든도 `params`·`is`·`expected` 를 더 이상 불투명하게 두지 않는다. 예전 주석은
"원시 JSON-RPC 안의 주소는 리터럴로 남는다" 였는데 그게 더 이상 참이 아니다.

**H4 는 마지막이 아니라 H1 과 함께 넣는다.** 허용 목록을 크게 두고 시작해,
묶음을 치울 때마다 목록을 줄인다. 그래야 되돌아가지 않는다.

**그 "줄일 때" 가 왔다 (2026-09-18).** H3-b 로 `tests/tc` 의 preset 주소 리터럴이
0곳이 됐다. 목록을 그대로 두면 래칫이 한 칸 헐거운 채로 남고, 다음에 누가 주소를
다시 써도 잡히지 않는다. **H4 2단계는 §11.5 에 있다.**

---

## 8. 실제 망 대응 (R)

여기부터는 로컬에서 확인할 수 없는 것이 섞인다.

**착수 순서는 의존이 정한다 (2026-09-18).** P3 없이 갈 수 있는 것은 **R2 · R3 · R4**
셋뿐이다. R1 과 R5 가 P3 에 의존하고 R6 이 R1 에 의존하는데, P3 는 값에 막힌 케이스가
2건뿐이라 §11.5 로 내려가 있다. 즉 **R 묶음의 절반은 P3 를 올리지 않으면 열리지
않는다.** 이 사실이 §11.4 의 순서 표기와 어긋나 있었다.

| ID | 항목 | 근거 | 의존 | 상태 |
|---|---|---|---|---|
| R1 | chainId 를 preset 값과 비교한다 | 지금 `WantChainID` 가 0이면 검사를 건너뛴다(`internal/testengine/nodegate.go:44-56`) | P3 | 미착수 |
| R2 | 엔드포인트에 역할·WS·metrics 를 담는다 | `--rpc` 는 URL 만 받는다. 워크스페이스 attach 는 역할을 안다 | P1 | 미착수 |
| R3 | 능력을 실제 RPC 로 확인한다 | 매니페스트의 `capabilities` 는 손으로 쓴 목록이다 | P4, V7 | 미착수 |
| R4 | faucet·deploy·load 를 로컬 서명으로 바꾼다 | 실제 망 노드는 남의 키를 열어주지 않는다. 지금 로컬 서명 sendTx 는 gas 21000 고정이고 컨트랙트 생성을 못 한다 | D7 | 미착수 |
| R5 | `expect: reject` 를 사유별로 판정한다 | 지금은 어떤 오류든 통과한다 | P3 | 미착수 |
| R6 | 한 스위트를 여러 preset 으로 실행하고 결과에 preset id 를 남긴다 | 완료 조건 4번을 증명한다 | P5, R1 | 미착수 |

### 공통 TC 가 요구하는 도구 기능과의 대응 (2026-09-20)

[`docs/tc/common/03-separate-implementation-items.md`](../../tc/common/03-separate-implementation-items.md)
§2 "실행 도구에 추가해야 하는 것" 이 여덟 가지를 적는다. **그 여덟과 위 R 묶음은 지금까지 어느
문서에서도 이어져 있지 않았다** — 같은 일을 두 이름으로 부르고 있었다. 아래가 대응이다.

| 공통 TC §2 기능 | 여기서 부르는 이름 | 덮는 정도 |
|---|---|---|
| 로컬 서명 전송 | R4 | 전부. R4 의 근거 문장이 "gas 21000 고정, 계약 생성 불가" 로 같은 결함을 적는다 |
| 기능 사전 확인 | R3 | 전부. "없으면 건너뜀과 이유" 까지 R3 의 `needs`/SKIP 이 맡는다 |
| 외부 주소 프로필 | R2 | 전부. 역할·WS·metrics 가 양쪽에 같이 적혀 있다 |
| 거부 판정 강화 | R5 | 전부. 사유 문구 앞부분 일치라는 판정 방식까지 같다 |
| 세 체인 일괄 실행 | R6 | 전부 |
| 실행 결과 기록 | R6 | **일부.** R6 은 preset id 만 남긴다. 프로그램 해시·버전·하드포크 상태·서명 방식·건너뛴 사유는 R6 에 없다 |
| 동기화 경로 관찰 | **없음** | 노드 로그나 지표로 어느 경로(snap/full)로 받았는지 확인하는 항목이 R 에도 V 에도 P 에도 없다 |
| 준비물 공유 | **없음** | 한 케이스 파일이 저장한 값(배포 주소·영수증)을 다른 파일이 이름으로 읽는 기능. P4 는 preset→케이스 방향(`provides`/`needs`)이라 **케이스 파일 사이**를 잇지 못한다. 회귀 실행 묶음 A·C 가 이것을 전제한다 |

즉 **R 묶음을 다 끝내도 공통 TC 는 열리지 않는다.** 빈 둘과 R6 의 모자란 부분이 남는다.
새 ID 를 붙이는 것은 여는 사람의 몫으로 남긴다 — 여기서 붙이면 착수 전에 의존을 지어내게 된다.

---

## 9. 마무리 (Z)

전체 작업이 끝난 뒤에 하는 일이다. 앞의 모든 항목과 별도 트랙이 끝나야 열린다.

| ID | 항목 | 근거 | 의존 | 상태 |
|---|---|---|---|---|
| Z1 | 모든 테스트를 DSL 문법으로 옮긴다 | 별도 트랙이다. 기존 기록은 `docs/dev/legacy-test-migration.md` | — | 별도 트랙 |
| Z2 | local 과 remote(도커 기반 개발 환경) 두 곳에서 테스트를 통과시킨다 | 최종 정리의 통과 조건이다 | Z1, 위 전 항목 | 미착수 |
| Z3 | DSL 코드의 `V` 버전 표기를 모두 없애고 최종을 v1 으로 재정리한다 | 지금 `EnvV2`, `CaseV2`, `spec_v2.go`, `schemaVersion: "2"`, `ParseV2`, `IsV2` 가 v2 를 달고 있다. 기계적 개명이지만 표면(정의서의 `schemaVersion`)까지 바뀐다 | Z1, Z2 | 미착수 |

**Z3 은 마지막이다.** 리팩토링이 끝나고, 모든 테스트가 DSL 로 옮겨지고, local 과
remote 검증이 끝난 뒤에 한 번에 한다. 중간에 하면 표면이 두 번 바뀐다.

---

## 10. 이미 드러난 결함 (X)

| ID | 항목 | 처리 | 상태 |
|---|---|---|---|
| X1 | stablenet 바이너리 철자가 둘이다 | 없어졌다. `go-stablenet` 은 바이너리 이름이 아니라 저장소 이름이었다(매니페스트의 `build.repo`). 중복 선언을 지우니 둘 다 사라졌다 | **해소 (2026-09-14)** |
| X2 | 대상 서버 절대 경로가 케이스에 있다 | 5건이었다. 전부 표준 15노드 케이스였고 도커 워크스페이스 설정이 같은 경로를 만든다 | **해소 (2026-09-14)** |
| X3 | `pn` 스코프를 못 쓴다 | V4 에서 해소 | **해소 (2026-09-14)** |
| X4 | wbft 매니페스트의 `make_target` 이 틀렸다 | **해소 (2026-09-15).** 세 저장소를 소스까지 확인했다 — go-stablenet 은 `cmd/gstable`/타깃 `gstable`, go-wbft 는 `cmd/gwemix`/타깃 `gwemix`, go-wemix 도 `cmd/gwemix`/타깃 `gwemix` 다. go-wbft 안에 `gwbft` 라는 낱말은 없다(`git grep -il gwbft` 0건). **두 필드를 갈라 봐야 한다**: `binary: gwbft` 는 **틀린 주장이 아니라 chainbench 의 요구**다 — 핸드오프가 go-wemix 의 `gwemix` 와 go-wbft 의 것을 **한 망에서 동시에** 쓰므로 두 이름이 필요하고, 운영자가 `GWBFT_BIN` 으로 실물을 잇는다(`tests/tc/CHAIN-BRINGUP.md` 가 이미 그렇게 적고 있다). 반면 `build.make_target` 은 **빌드 방법에 대한 사실 진술**이고 `make gwbft` 는 없다 — `gwemix` 로 고쳤다 | **해소 (2026-09-15)** |
| X5 | Stablenet 기준선 대조 | **Go 테스트 절반 완료 (2026-09-15).** 기준선 `20260913T101856Z` 는 58패키지·1,944 PASS·27 SKIP·테스트 없는 패키지 12 였고, 지금은 **58패키지·0 실패·2,030 PASS·27 SKIP·테스트 없는 패키지 12** 다. **SKIP 이 27 그대로**이고 PASS 가 86 늘었다. **라이브 절반에서 회귀를 하나 잡았다 (X10)** — `tests/e2e` 하니스가 은퇴한 `--validators`/`--endpoints` 로 `chain up` 을 부르고 있었다. 고친 뒤 `TestE2E_StablenetChain` 이 43.18초에 PASS 했다(기준선의 재실행은 42.47초 PASS). **다만 이 PASS 는 기준선의 열린 실패를 해소한 증거가 아니다** — §아래 | 절반 완료, 라이브는 X10 뒤 |
| X6 | `peering.go` 의 `RoleSupport` 주석이 실제와 다르다 | 같은 주장이 `registry.go` 와 `node.go` 에도 있었다. 셋 다 고쳤다 | **해소 (2026-09-14)** |
| X7 | `01-wemix-wbft-handoff.json` 은 구동될 수 없었다 | 업그레이드 env 인데 `topology: {bp: 4}` 를 들고 있었고, `compositionOf` 는 핸드오프 env 의 topology·launch·config·hardforks 를 거부한다. 케이스는 `3618dd7c`(#383) 부터 그 필드를 갖고 있었고 거부는 `dad54b37`(#363) 에 들어왔다 — 그 사이에 이 케이스가 `suite run` 으로 돌아간 적이 없다. e2e 는 `upgrade run` 을 직접 부르므로 잡지 못했다. **그 수는 쓰이지도 않았고 틀리기까지 했다**: 프로파일은 그 망을 producer 1 + validator 4, 즉 노드 5대로 잡는데 `bp: 4` 는 validator 수를 다른 뜻의 칸에 옮겨 적은 것이었다. 케이스에서 지웠고, 크기는 계획이 프로파일에서 읽어 보여준다 | **해소 (2026-09-15)** |
| X8 | 게이트 없이 체인 컨트랙트 주소를 든 케이스 9건 | `applicableChains` 가 비어 있는데 `0x…1000`대 주소를 쓴다. 다른 체인에 올리면 돌다가 깨진다. P6 이 그 시나리오를 처음 가능하게 만들어서 드러났다. 7건은 `stablenet`(boho·anzeon 포크와 `0x…1004`·`0x…b00003` 은 stablenet genesis·바이너리에만 있다), 2건은 `wbft`(케이스가 스스로 go-wbft 컨트랙트에 맞춰 쓰였다고 적고 있고 셀렉터가 그것을 뒷받침한다). **`internal/testengine` 의 코퍼스 테스트가 재발을 막는다** — 변이를 심어 잡는 것과 프리컴파일을 안 잡는 것을 둘 다 확인했다 | **해소 (2026-09-15)** |
| X9 | stablenet 매니페스트의 포크 목록이 불완전하다 | `genesis.hardforks` 가 `["istanbul","boho"]` 인데 케이스 8건이 `applepieBlock` 을 켜고, `suite.go` 의 실측 기록은 노드가 `Applepie: #<nil>` 을 찍었다고 적고 있다 — 바이너리는 그 포크를 안다. 2단계가 이 목록에서 `fork:` 능력을 파생하므로, 고치기 전에는 그 8건을 `fork:applepie` 로 게이트할 수 없다. **go-stablenet 의 포크 목록 확인이 필요하다** | §7.0.3 조사로 발견 (2026-09-15), 미해결 |
| X10 | 게이트된 e2e 테스트가 은퇴한 플래그를 부른다 | `tests/e2e/harness_test.go` 가 `chain up --validators/--endpoints` 를 부르는데 V9-b 가 그것을 `--bp`/`--en` 으로 바꿨다. **`//go:build e2e` 태그 때문에 `go test ./...` 이 컴파일조차 하지 않고, `go vet -tags e2e` 는 문자열 인자를 검사하지 않으므로 CI 가 잡을 수 없었다.** 체인 바이너리를 주고 실제로 돌려야만 드러난다. `--bp`/`--en` 으로 고쳤고 헬퍼 인자 이름도 역할 어휘에 맞췄다. `keyring new --validators` 는 신원 수를 뜻하는 다른 플래그라 그대로 둔다 | **해소 (2026-09-15)** |
| X11 | `tests/repro/*.sh` 가 없어진 `net up` 을 부른다 | 스크립트 3종이 `chainbench net up` 을 부르는데 그 명령은 은퇴했다(`unknown command "net"`). `tests/e2e/README.md` 는 이 계층이 게이트된 Go 테스트로 옮겨졌다고 적고 있으므로 **옮겨지지 않은 잔여인지, 지워야 할 것인지 판단이 필요하다.** 오늘 만든 회귀가 아니다 | 발견 (2026-09-15), 미해결 |
| X12 | 이름 지은 컨트랙트가 액션의 `to` 에서 안 풀린다 | 3단계는 `resolveAddressArgs`(리더·어세션이 쓰는 길)만 `ResolveAddress` 로 바꿨고, **액션 구현은 `ResolveAccount` 를 직접 부르고 있었다.** 그래서 H1 이 `"to": "govValidator"` 로 바꾼 케이스가 `sendTx` 에서 `unknown account "govValidator"` 로 죽었다. 오프라인 검사가 전부 못 잡았다 — 주소 골든은 `resolveAddressArgs` 를 쓰므로 **액션이 쓰는 다른 함수를 검사하지 않았다.** `sendTx`(양쪽 경로)와 `registerContract` 의 `to` 를 `ResolveAddress` 로 바꿨다. `faucet` 의 `to` 는 계정만 받는다(코퍼스에 컨트랙트로 보내는 케이스 0건). 재발 방지로 **`ResolveAccount` 가 컨트랙트 이름을 받으면 그렇게 말한다** — "unknown account" 는 없는 키를 찾게 만들지만, 실제 원인은 배선이다 . **같은 구멍 셋을 더 찾아 함께 막았다** — `sendRawTampered` 의 `to`, `callError` 의 `to`, `wsSubscribe` 의 `address` 는 아예 해석을 안 하고 있었다(오늘 코퍼스에 그 자리로 이름을 넣은 케이스는 0건이라 드러나지 않았다). **그리고 소스 수준 검사를 넣었다**: 주소 모양 인자를 맵에서 꺼내면서 그 값을 해석기에 넘기지 않는 함수를 `internal/testhelper` 전체에서 찾는다. **함수 단위가 아니라 키 단위다** — `sendTx` 는 `from` 을 해석하고 `to` 를 빠뜨렸으므로 "이 함수가 뭔가 해석하나" 로는 통과한다. 리더 면제 목록은 손으로 적지 않고 소스에서 뽑는다(어세션 표의 `read:` 와 `RegisterReader`), 그래서 새 리더는 저절로 덮이고 **새 액션은 안 덮인다** | **해소 (2026-09-16)** |
| X13 | `tests/e2e` 가 `tests/specs` 를 가리킨다 | `runCase` 가 `tests/specs/system-contracts/<이름>.json` 을 조립하는데 그 트리는 `#362`(2026-09-08) 의 tc 통합에서 사라졌다. 우리 트랙 이전이고, 게이트된 테스트라 컴파일도 CI 도 읽지 않아 드러나지 않았다. 경로를 조립하는 대신 `tests/tc` 아래에서 이름으로 **찾게** 했다 — 코퍼스가 이미 한 번 옮겨졌으므로 다음 이동도 견딘다. 못 찾거나 여럿이면 무엇을 찾았는지 대며 그 자리에서 실패한다 | **해소 (2026-09-16)** |
| X14 | `TestE2E_StablenetConsensusLifecycle` 가 간헐적으로 실패한다 | **6회 중 2회 실패.** 실패는 51.32s·50.79s 로 둘 다 50초대, 통과는 33~61초로 흩어진다. 단독·2건·stablenet 6건·**전체 19건 모두에서 통과한 적이 있으므로 "전체 실행에서만 깨진다" 는 가설은 반증됐다. **문구를 확보하지 못했다** — 실패한 두 번 다 출력을 `--- PASS\|FAIL` 줄만 남기고 걸렀다. 이 트랙이 만든 것으로 보이지는 않는다(이 테스트는 DSL 케이스를 안 쓰고 `nodeStop`/`nodeStart`/`head`/`blockField` 만 쓴다; 실패 두 번 사이에 X12·X13 을 고쳤는데 결과가 같았다) — **정황이지 증명은 아니다.** **다음에 실패하면 그때 분석한다**: 필요한 것은 `t.Fatalf` 문구 하나이고, 그것이 세 갈래를 가른다 — `waitAdvancing` 타임아웃이면 예산 문제, `b-10: parentHash chain broken` 이면 체인 분기, `b-08: production continued below quorum` 이면 정족수 아래 생산이라는 심각한 결함이다 | **재발 시 분석** (2026-09-16) |
| X15 | 계획의 `target` 줄이 선언의 배치를 무시했다 | `describeTarget` 이 `up.Server`(명령이 고른 서버셋 항목)만 읽고 `up.Target`(선언의 `env.target`)은 안 읽었다. 그래서 `env.target: "ops@<호스트>:/data/net1"` 인 케이스의 계획이 `target  this machine` 을 찍었다 — **뜨는 망과 다른 망을 말하는 계획**이고, M4-a 가 "합성기가 받는 그 값에서 렌더링하므로 다른 망을 말할 수 없다" 고 적어 둔 바로 그것이 깨졌다. 두 필드 다 읽게 고쳤고, 무엇이 원격인지는 `resource.Spec.Describe` 에 묻는다(그 규칙의 소유자이고, `internal/arch` 가 소비자의 직접 분기를 막는다). M4-b 의 값별 출처를 넣다가 드러났다 | **해소 (2026-09-16)** |

---

## 11. 우선순위

### 다음에 할 일 (2026-09-18 기준)

**오프라인으로 할 수 있는 일은 남아 있지 않다.** 남은 셋은 전부 체인을 띄워야 한다.

| | 무엇 | 얼마나 | 왜 지금 |
|---|---|---|---|
| ~~1~~ | ~~전체 회귀 1회 (§11.4 L1)~~ | — | **완료 (2026-09-20).** §11.4.1 |
| ~~2~~ | ~~노드 간 읽기 경합 전수 (§11.4 L2)~~ | — | **완료 (2026-09-20).** §11.4.2 |
| **1** | **HEAD 바이너리로 L1 재실행** | 약 4시간 | 위 L1 은 **2026-08-10 빌드**로 돌았다. §11.4.1 의 "무엇이 아직 안 닫혔나" 참조 |
| ~~2~~ | ~~`restart-at-boho` 1건~~ | — | **완료 (2026-09-20).** pre-boho 빌드를 golang 컨테이너 안에서 네이티브로 만들어(`GOOS`/`GOARCH` 교차가 아니라 네이티브라 CGO 가 되고 `blst` 가 빌드된다) 쌍으로 돌렸다 — `GSTABLE_BIN=gstable-v1`(v1.0.0, 포크를 모른다) · `GSTABLE_POSTFORK_BIN=gstable`(v1.1.0). 계획이 `per node: default=gstable-v1, postfork=gstable` 을 잡았고 케이스가 통과한다. **A29 29건이 이로써 전부 닫혔다** |
| 3 | 실제 망 대응 R2~R6 (§11.4) | 대 | 위가 닫힌 뒤 |

**단점도 적는다.** 1번은 2시간 동안 머신을 점유하고, 끝나도 "이번엔 통과했다" 이상은
말해 주지 못한다. 간헐 결함은 한 번 돌려서 없다고 할 수 없다.


### 11.1 끝난 것 (2026-09-18 기준 53항목)

C1 · N1 · N2~N7 · V1~V11 · W0 · W5 · M1~M6 · P1 · P5 · P5-L1 · P5-L2 · P6 ·
H1 · H3 · H3-b · H4(1단계) · X1~X3 · X4 · X6~X11 · X12 · X13 · X15 · D-A · D-B,
그리고 매니페스트 능력 1~3단계.

**완료 조건 네 개가 다 섰다 (2026-09-18).** 마지막이던 `applicableChains` 가 66 →
0 이 됐다. 남은 것은 조건이 아니라 품질이다.

**완료 조건 대비 위치와 이 트랙이 세운 검사 목록은 §12 에 있다.**

### 11.2 하니스 안정성 — S1·S2·S4·S5 완료 (2026-09-17)

**205건 라이브 검증이 드러낸 것은 케이스가 아니라 하니스였다.** 아래 셋은 서로
다른 증상이지만 뿌리가 하나다 — **생애 주기를 책임질 주체가 구현되지 않아 그 일이
호출자로 번졌다.** 설계는 `stage-lifecycle-design.md` 에 있다.

| 항목 | 무엇을 했나 | 상태 |
|---|---|---|
| **S1** | 수집을 `session.TestRecord` 에서 떼어냈다. 테스트 실패는 지금처럼 `observations/`, **세우다 실패는 워크스페이스의 `failures/<시각>/`**. 정리보다 **먼저** 모은다(건강 조사는 노드가 응답해야 한다). 못 모은 것은 `gather-problems.txt` 에 남아 사라지지 않는다. **실측: 0개 → 6개** | **완료 (2026-09-17)** |
| **S2** | `Step` 에 결과·시작 시각·오류를 더했다(`Done` 은 유지 — 옛 기록도 읽힌다). 실패는 **러너 한 곳**에서 적는다(실패한 단계는 자기 기록 호출에 도달하지 못한다). `init`·`start` 를 의존 표에 넣어 "초기화 뒤 기동" 이 순회 순서가 아니라 **규칙**이 됐다. **실측: `init failed` + 사유가 기록에 남는다** | **완료 (2026-09-17)** |
| **S3** | 노드 하나가 90초 안에 준비되지 않는다. 가벼운 구성 8회·무거운 구성 1회로는 **재현되지 않았다**(그 자체가 정보다 — 아무 구성에서나 나지 않는다). 앞서 3회 연속 났을 때는 205건 실행 직후라 **머신 부하가 쌓인 상태**였다는 것이 정황이다. **S1 덕에 다음에 나면 증거가 남는다**<br>**2026-09-20**: L1 의 wemix 배치에서 같은 문구가 2회, 뒤이어 6회 더 났다. **원인은 디스크였다** — 컨테이너 여유가 409M 까지 차 있었고, 쓰지 못하는 노드는 IPC 소켓을 못 만든다. 정리 후 같은 케이스가 전부 통과했다. 부하 정황이라는 앞의 짐작보다 단순한 설명이고, **S3 를 재현했다고 볼 근거는 못 된다** | **재발 시 분석** |
| **S4** | `Stop` 이 머신 해석 실패에 반환해 **남은 노드를 켜 둔 채 끝나던 것**을 고쳤다. 모든 노드를 시도하고 "2 of 3 stopped" 로 얼마나 됐는지 말한다. **동시 정지**로 바꿔 유예(15초)가 노드 수만큼 쌓이지 않는다. 테스트 스텁 둘이 락 없이 기록하던 것도 고쳤고, **새 테스트가 내 첫 구현의 실제 레이스를 잡았다** | **완료 (2026-09-17)** |
| **S5** | 증거의 로그가 **양끝**을 남긴다(앞 200 + 뒤 200, 사이에 생략 줄 수). 기동 실패는 로그 **머리**에 있는데 꼬리 200줄은 마지막 30초라 그 창에 없었다 | **완료 (2026-09-17)** |

### 11.2.1 DSL 어휘가 모자란 자리 (2026-09-17)

**선언이 표현할 수 없어서 막힌 것들이다.** 능력은 있는데 말할 방법이 없다.

| 항목 | 무엇 | 상태 |
|---|---|---|
| ~~**D-A**~~ | ~~선언이 망의 능력을 말할 수 없다~~ | **완료 (2026-09-18).** 진단이 반쯤 틀렸었다. overlay 파일을 가리킬 키가 없는 것이 아니라 — 하니스가 그 파일을 이미 쓰고 있다 — `{capabilities, genesis}` 문서의 **genesis 쪽만** 쓰고 있었다. env 의 `capabilities` 는 요구 목록으로 `spec.Requires` 에 합쳐지므로(142개 파일이 그 뜻으로 쓴다) 거기에 얹을 수 없었고, `genesis.provides` 를 새로 뒀다. 스킵 6건이 전부 통과한다(실측) |
| ~~**D-B**~~ | ~~케이스가 "이 망은 멈춘다" 를 말할 수 없다~~ | **완료 (2026-09-18).** `genesis.haltsAt` 을 뒀다. 준비 판정은 "진행하고 있나" 만 묻는데, 멈추는 것이 정답인 체인에는 그 질문의 답이 틀렸다. 하드포크 인계가 이미 같은 모양(모든 노드가 한 블록 앞에 서 있고 아무것도 진행하지 않음)이라 그 게이트를 넓혔다 — 인계자가 없는 쪽이 `preFork` 가 빈 집합이다. `unsupported-system-contract-version` 이 통과한다(실측) |

**D-A 를 푼 방법과 그 과정에서 나온 것 (2026-09-18).**

스킵 6건은 `capabilities: ["rpc", "account-extra"]` 를 env 에 적고 있었다. 그런데
env 의 `capabilities` 는 **요구**이고 `spec.Requires` 로 합쳐진다. 즉 여섯 케이스는
"이 능력이 필요하다" 를 두 번 적고 있었을 뿐, 망에 그 능력을 준 적이 없다. 그래서
영영 스킵됐다. 그 뜻은 142개 파일이 쓰고 있어서 바꿀 수 없으므로 광고하는 자리를
`genesis.provides` 로 새로 뒀고, 하니스가 이미 쓰고 있던 overlay 문서의
`capabilities` 절에 실어 보낸다.

env 는 셋이 됐다. `stablenet-bp4-account-extra` 는 alloc 세 계정에 `extra` 비트를
심고(권한 62번, 차단 63번, 둘 다), `stablenet-bp4-short-expiry` 는 네 거버넌스
컨트랙트의 `expiry` 를 604800 에서 30 으로 낮춘다. 소각 환불 케이스만 셋째
`stablenet-bp4-boho-short-expiry` 를 쓴다 — `refundableBalance(address)` 는
**GovMinter v2** 에만 있어서 boho 가 켜져 있지 않으면 읽기부터 revert 한다.

**그리고 케이스 하나가 깨져 있었다.** `05-burn-expire-refundable` 의 `proposeBurn`
calldata 가 **447바이트**였다 — 32의 배수가 아니다. 어느 워드의 패딩에서 0 바이트
하나가 빠져 그 뒤가 전부 한 칸씩 밀려 있었고, 문자열 길이 워드가 문자열 첫 글자를
물고 있었다. 스킵되는 케이스는 아무도 실행하지 않으므로 **아무도 못 봤다.** 잘
도는 형제 케이스(`03-burn-cancel-refundable`)의 배치대로 다시 인코딩했다.
정의서 전체를 훑어 같은 결함이 더 있는지 봤고, 32의 배수가 아닌 나머지 넷은 전부
ABI 호출이 아니라 배포 바이트코드였다.

### 11.2.13 남은 셋을 마저 고쳤다 (2026-09-18)

`04b`, `03-unsupported-version`, `04-anzeon-basefee-stable` 셋이 남아 있었다.
셋 다 고쳤고, 고치는 과정에서 어휘가 모자란 자리 네 곳이 드러났다.

**`04b` 는 주장 자체가 틀려 있었다.** 케이스 제목은 "멤버로 추가된 노드가 에폭
경계에서 검증자 집합에 들어간다" 인데, `GovValidator._onMemberAdded` 는 본문이
`// do nothing` 이다. 멤버 추가는 검증자를 만들지 않는다. 검증자가 되는 길은
`configureValidator(address, bytes blsKey, bytes blsSig)` 하나뿐이고, 이것은
**아직 자기 validator 가 없는 활성 멤버**만 부를 수 있다. genesis 는 멤버 넷을
전부 자기 자신의 operator 로 심어 두므로(`gov_validator.go` 의
`operatorToValidator[member] = val`), 그 넷 중 누가 불러도 교체가 되지 추가가
되지 않는다.

그래서 케이스가 다섯 번째 멤버를 **직접 만든다**: `newAccount` 로 계정을 하나
얻어 node1 이 채워 주고, 거버넌스가 그 계정을 멤버로 올린 뒤, 그 계정이
`configureValidator(en1, …)` 로 en1 을 자기 validator 로 등록한다. BLS 키와 소유
증명은 preset 의 node5 값이고, 컨트랙트가 `blsPoP` 프리컴파일로 검증하므로 지어낸
키는 거부된다. 에폭 경계를 지나면 검증자가 4 → 5 가 된다(실측).

**그 과정에서 간헐 실패가 하나 더 나왔다 — 그리고 그것이 더 중요하다.**
8회 중 2회가 `configureValidator` 에서 revert 했다. 진단용 읽기를 케이스에 심어
보니 제안 상태가 `Executed(3)` 가 아니라 `Voting(1)` 이었다. 원인은 거버넌스가
아니라 **노드가 갈린 것**이다. 승인은 `from: node2, on: node2` 로 보내 node2 가
영수증을 확인해 주는데, 그 다음 읽기와 전송은 전부 node1 이 답한다. node1 이 그
블록을 아직 들이지 않았으면 node1 의 상태에는 다섯 번째 멤버가 없고,
`onlyActiveMember` 가 revert 한다. 블록이 도착했는지에만 달렸으니 간헐적이다.

고치는 방법은 node1 이 그 영수증을 가질 때까지 기다리는 것이다(`waitFor` 로
`eth_getTransactionReceipt` 의 status). 8회 연속 통과한다.

> **이 경합은 이 케이스만의 것이 아니다.** "node1 이 제안하고 node2 가 승인한 뒤
> 기본 노드에서 읽는" 모양은 거버넌스 케이스 전반에 있다. 대부분은 뒤에 다른
> 대기가 끼어 있어 우연히 통과해 왔다. **전수 점검은 아직 안 했다** — 다음에
> 거버넌스 케이스가 간헐적으로 깨지면 여기를 먼저 본다.

**`04-anzeon-basefee-stable` 도 구조적으로 경합이었다.** 안정 구간은
`6% < 사용량 ≤ 20%` 이고 빈 블록은 사용량 0 이라 하락 구간이다
(`params/protocol_params.go`: IncreasingThreshold 20, DecreasingThreshold 6,
BaseFeeChangeRate 2). 케이스는 여섯 블록에 부하를 건 뒤 base fee 가 처음과 같은지
봤는데, 사이에 빈 블록이 하나만 끼어도 2% 떨어져 어긋난다. `load` 는 영수증을
기다리며 한 블록에 하나씩 보내므로 빈 블록을 막을 수 없다. 단독 5회는 다 통과했고
전체 실행(부하가 높은 상태)에서 깨졌던 것이 이것으로 설명된다.

그래서 **블록 짝**으로 다시 썼다. 마지막 소각이 담긴 블록과 그 다음 블록을 견주고,
그 블록의 `gasUsed` 가 실제로 안정 구간 안인지도 함께 확인한다(실측 1,823,397 /
20,000,000 = 9.1%). 경합이 사라지고, 통과해도 무의미해질 일이 없다.

**어휘가 모자라 넓힌 곳 넷.**

| 무엇 | 왜 필요했나 |
|---|---|
| `derive abiCall` 의 `{"bytes": "0x…"}` 인자 | 스칼라 32바이트 워드만 받아서 `configureValidator(address, bytes, bytes)` 를 **부를 방법이 아예 없었다**. 이런 서명에 닿는 유일한 길이 손으로 인코딩한 덩어리를 붙여 넣는 것이었고, `05-burn-expire-refundable` 이 워드 경계에서 한 바이트 모자랐던 것이 그 결과다 |
| `derive` 의 `format: "hex"` | 블록을 인자로 받는 RPC 는 0x-hex 만 받는다. 영수증에서 블록 번호를 읽고도 **그 다음 블록**을 물을 수 없었고, base fee 처럼 블록과 자식 블록에 걸쳐 정의된 규칙은 확인할 길이 없었다 |
| `uintArg` 가 0x-hex 도 받는다 | 체인이 돌려주는 블록 번호·가스·nonce 는 전부 0x-hex 인데 십진만 받아서, 체인에서 읽은 값을 그 값을 받는 인자에 그대로 넣을 수 없었다(`waitBlock` 에 영수증의 블록 번호를 줄 수 없었다). `parseBigValue` 는 처음부터 둘 다 받았다 |
| `genesis.haltsAt` | 위 D-B |

---

### 11.2.14 이 문서가 스스로 만든 결함 셋 (2026-09-18)

코드가 아니라 **이 문서**의 결함이다. 앞의 둘은 사용자가 "정말 다 적혀 있냐" 고
물어서 찾았고, 셋째는 그 둘을 적다가 나왔다.

> 이 제목도 한 번 틀렸다. 셋째를 덧붙이면서 "결함 둘" 을 그대로 뒀다 — 제목이
> 개수를 세는 순간 본문이 늘 때마다 같이 고쳐야 한다. 같은 결함의 네 번째 사례로
> 세지 않고 여기 적어만 둔다.

**하나 — 항목을 §11.5 에만 적었다.** N1-b 와 H4 2단계를 우선순위 표에만 남기고
원래 자리(§2.5.1 의 N1, §7 의 H4)에는 아무것도 안 적었다. 그 주제를 찾아 §2.5.1 을
펼친 사람은 **끝난 일로 읽는다** — 항목이 "완료" 로 닫혀 있고 남은 절반을 말하는 줄이
없기 때문이다. 우선순위 표는 순서를 보는 자리이지 주제를 찾는 자리가 아니다.

> **규칙.** 항목을 미루거나 쪼갤 때는 **원래 자리에도 한 줄**을 남긴다. 무엇이
> 남았고 무엇이 막고 있는지까지 적는다. 우선순위 표에만 적으면 그것은 기록이 아니라
> 대기열이다.

**둘 — 순서 표기가 자기 의존과 어긋났다.** §11.4 가 실제 망 묶음을
`R2 → R3 → R5 → R4 → R6` 으로 적고 있었는데, **R1 이 빠져 있는데 R6 이 R1 에
의존하고**, R5 는 그보다 앞에 놓였지만 P3 에 의존한다. 이 순서대로 시작하면 R5 에서
막힌다. 순서를 손으로 적고 의존을 표로 적으면 둘은 반드시 갈라진다.

> **아직 기계가 안 본다.** §12.2 의 검사 일곱은 케이스와 코드를 보지 이 문서를 보지
> 않는다. 의존 표에서 순서를 **유도**하면 이 결함은 구조적으로 사라지지만, 그러려면
> 표기를 기계가 읽을 수 있게 바꿔야 한다. 지금은 사람이 지키는 규칙으로 둔다.

**셋 — 절 번호가 겹쳤다.** 위 둘을 적다가 드러났다. 이 절을 `11.2.4` 로 붙였는데
§11.2.4 는 이미 있었고(결과물이 쌓이는 자리), 그 번호는 §11.2 표에서 **참조되고
있다.** `11.2.3` 도 둘이었다. 새 절을 파일 중간에 끼워 넣으면서 번호를 눈으로 고르면
이렇게 된다.

> 오늘 붙인 두 절만 빈 번호(`11.2.13`·`11.2.14`)로 옮겼다. **옛 번호는 건드리지
> 않았다** — 다른 문서가 `§11.2.5`·`§11.2.8`~`§11.2.12` 를 가리키고 있어서, 정렬을
> 위해 번호를 다시 매기면 그 참조가 전부 깨진다. 번호는 순서가 아니라 **이름**으로
> 쓰고 있다는 뜻이고, 그렇다면 §11.2 의 절들은 번호순으로 놓여 있지 않다.

---

### 11.2.2 내가 모르는 것 (2026-09-17, 정직한 기록)

**거버넌스 계약과 체인 설정의 층을 구분하지 못한다.** 오늘 이것 때문에 반복해서
틀렸다.

- `proposeAddMember`(거버넌스 멤버)와 `configureValidator`(검증자+BLS키)를 같은
  것으로 봤다. 후보 목록과 합의 검증자 집합이 다른 것도 사용자가 짚어 줘서 알았다.
- 정족수가 genesis 의 `govValidator.params.quorum` 에서 오고 제안 생성 시점의
  값을 스냅숏한다는 것을 뒤늦게 확인했다.
- **`short-expiry` 와 `account-extra` 를 "genesis 문제" 로 읽었다.** 사용자는 만료
  기한이 거버넌스 컨트랙트 파라미터이고 계정 부가 상태는 config 로 주입할 수 있다고
  짚었다. 2026-09-18 에 go-stablenet 코드로 확인해 보니 **절반씩 맞았다.** 만료
  기한은 컨트랙트 파라미터가 맞아서 `config.anzeon.systemContracts.*.params.expiry`
  로 심었다(`GovBase.sol` 이 초기화 때 한 번 읽고 이후 바꾸지 못한다). 계정 부가
  상태는 `types.Account.Extra` 라는 **genesis alloc 전용 필드**이고 `core/genesis.go`
  가 `statedb.SetExtra` 로 심는다 — config 경로가 없다. 그래서 `alloc[addr].extra`
  에 남겼다.

**이 목록은 다음에 이 영역을 건드리기 전에 먼저 배워야 할 것이다.** 증상만 쫓아
고치면 또 틀린다.

### 11.2.3 W 묶음 조사 결과 (2026-09-17)

**조사만 했다. 코드는 건드리지 않았다.** 여섯 항목 중 **셋은 이미 풀려 있었고**,
셋은 고칠 자리가 한 곳으로 좁혀졌다.

| 항목 | 조사 결과 | 다음 |
|---|---|---|
| **W6** 대상 경로 권한이 셋 | **절반은 이미 해결.** 데이터 루트는 `WorkspaceConfig.AdoptDataRoot` 가 단독 소유자이고, 다른 값이 오면 조용히 덮지 않고 **충돌로 거부**하며 어디를 고치라고 말한다. 주석이 그 역사도 적는다 — 두 호출자가 각자 비교를 복사해 갖고 있던 것을 한 곳으로 모았다. **남은 축은 "어느 머신인가"** 다: `up.Server`(명령행 `--server`/`--docker`)와 `up.Target`(선언 `env.target`)이 서로를 모른다. **실측: 선언이 `srv://alpha/data/net1` 을 말하고 명령이 `--server beta` 를 말하면 계획이 `server beta` 만 보여 주고 alpha 는 말없이 사라진다.** `--docker` 는 머신이 아니라 **서버셋 항목을 로컬 컨테이너로 다루는 방식**이라 이 검사의 대상이 아니다(처음에 충돌로 잘못 읽었다) | **완료 (2026-09-17)** — `refuseMachineConflict` 가 두 답을 다 대며 거부한다 |
| **W1** 구성 식별자가 위치에서 파생 | **이미 해소돼 있다. 워크리스트 설명이 낡았다.** `compositionID(dir)` 는 `new` 때 **한 번** 값을 정하는 씨앗이고, 그 뒤로는 기록이 소유자다(`new.go:95` — 비어 있을 때만 채운다). **실측: 워크스페이스를 옮겨도 `301ac9bbddbc` 그대로였다** | 없음 (항목 종료) |
| **W2** 기록 저장이 원자적이지 않다 | **고칠 자리가 한 줄이다.** `Composition.Save` 가 `os.WriteFile` 을 직접 부르는데(`composition.go:139`), **같은 패키지에 `WriteFileAtomic` 이 이미 있다**(`write.go:17`, 임시 파일 + rename). 다른 기록들은 그것을 쓴다 | 한 줄 교체 |
| **W3** 기록된 PID 는 생존 증거가 아니다 | **도구가 이미 있다.** `PIDAlive` 가 로컬·원격 양쪽에 구현돼 있다(`inspect.go:62,114`). `NetworkStatus` 가 안 부를 뿐이고, 서명이 `_ context.Context` 로 컨텍스트를 버리는 것이 그 증거다 — 확인할 생각이 없는 서명이다 | status 가 `PIDAlive` 를 묻게 한다 |
| **W4b** 계획이 설정의 바이너리 경로를 모른다 | **W4 확인 중 발견.** workspace-config 가 바이너리를 데이터 루트 아래에서 찾는데(`<dataRoot>/bin/gstable`), 계획은 매니페스트의 이름(`gstable`)을 적는다. M4-c 대조가 `asked for binary gstable, launched with /tmp/.../bin/gstable` 로 **실제 불일치를 잡았다** — 검사는 옳고, 계획이 설정 층을 안 읽는다 | **완료 (2026-09-17)** — 배치 규칙을 `chainsetup.PlaceBinary` 한 곳으로 빼고 계획이 그것을 거친다 |
| **W4** 결과 경로 기본값이 갈린다 | **셋이 다른 것보다 나쁘다.** `WorkspaceConfig.ArtifactRoot()` 가 있고 설정 파일이 그 값을 **필수로 요구**하는데(빈 값이면 거부), **그 함수를 부르는 코드가 저장소에 없다.** 사용자에게 적으라고 해 놓고 쓰지 않는다. 실제로 쓰이는 것은 CLI 기본값(`~/.chainbench/sessions`)과 엔진 기본값(`<워크스페이스>/sessions`) 둘뿐이다 | **완료 (2026-09-17)** — 설정값을 읽는다. 기본값도 하나로 합쳤다(§11.2.4) |
| **W7** prepared 경로 엣지 케이스 | **이름부터 이미 정리돼 있다.** `ExistingInputs` 이고 주석이 이유를 적는다 — "chainbench 가 확인할 수 있는 사실은 파일이 거기 있다는 것과 자기가 만들지 않았다는 것뿐이다. 누가 언제 준비했는지는 모른다." 모드도 `InputExisting` 이며 **없으면 오류이지 조용한 생성 대체가 아니다.** 존재 검사는 파싱 시점에 한다 | 남은 엣지(오래됨·공유)는 W6 결론 뒤 |

**조사 뒤 남은 것은 둘뿐이다.** W1·W2·W3·W7 은 끝났다(앞의 둘은 고쳤고, 뒤의
둘은 이미 그렇게 돼 있었다). 남은 **W4 와 W6 머신 축은 둘 다 "선언과 명령행이
같은 것을 다르게 말할 때" 를 다루지 않는 문제**이고, 데이터 루트가 이미 그 답을
갖고 있다 — `AdoptDataRoot` 처럼 **조용히 덮지 말고 충돌로 거부하며 고칠 자리를
말한다.** 같은 규칙을 두 곳에 더 적용하면 된다.

### 11.2.5 W4c — 재현·수정 완료 (2026-09-17)

**적어 뒀던 것보다 넓다.** 증상이 셋인데 뿌리는 하나다.

**뿌리.** 바이너리 참조는 **실행 직전에만** 배치된다(`Workspace.placeBinary`,
`Init`/`Start` 가 `state.Binary` 에 배치된 값을 적는다). 그런데 **그 실행과
대조하는 쪽들은 배치를 안 거친다.** 그래서 같은 바이너리를 한쪽은 이름으로,
다른 쪽은 경로로 말한다.

**증상 1 — 계획. 고쳤다(W4b).**

**증상 2 — 한 망 안에서 노드마다 다른 것을 실행한다.**

선언이 노드에 바이너리를 붙이면(`topology.nodes[].binary`) 그 이름이
`inlineTopologyOf` 에서 `resolved[name] = expand(path)` 로 들어가고
(`compose.go:597`), `state.Binaries` 에 **그대로** 저장되며
(`steps_compose.go:673`), `binaryFor` 가 그것을 **exec 경로로 바로 쓴다**
(`steps_lifecycle.go:41`). 단일 바이너리만 배치를 거친다.

실측 — `dataRoot: /data`, `paths.binaries: bin`, 선언은
`"binaries":{"default":"gstable","upgrade":"gstable-next"}` 와
`{"index":3,"role":"bp","binary":"upgrade"}`:

```
node1 execs "/data/bin/gstable-next"   ← 배치됨
node3 execs "gstable-next"             ← 배치 안 됨, 타깃 PATH 를 찾는다
```

PATH 에 없으면 node3 만 안 뜬다. PATH 에 **다른 빌드**가 있으면 한 망에서 두
빌드가 돌고 아무도 말해 주지 않는다.

`chainsetup` 쪽 필드 주석은 이 맵이 "its **resolved path**" 를 담는다고 적는다.
계약은 분명하고, 채우는 쪽이 계약을 안 지킨다.

**증상 3 — workspace-config 를 쓰면 재사용이 아예 안 된다.**

`Have.Binary` 는 기록된 **배치된 경로**이고(`steps_preflight.go:20`),
`WantOf` 는 요청의 **이름**을 그대로 쓴다(`steps_preflight.go:46`). 똑같은 요청을
다시 해도:

```
verdict = rebuild-all
  reason: binary: have "/data/bin/gstable", want "gstable"
```

**매번 체인 전체를 다시 만든다.** `--binary` 에 절대 경로를 주면 가려진다 —
그러니까 workspace-config 에 배치를 맡기는 **의도된 사용법에서만** 터진다.
이 경로는 `chain up` 과 `run` 양쪽이 공유한다(`preflightDecision` →
`ws.Compare`, `WantOf(up)`).

**재현 방법.** 셋 다 오프라인이고 바이너리가 필요 없다. `Workspace` 를
`&Workspace{state: State{...}}` 로 직접 세우면 `binary("")` 와 `binaryFor` 를
바로 부를 수 있고, 증상 3 은 `preflight.Compare(have, WantOf(in))` 한 줄이다.

**고쳤다 — (가), 2026-09-17.** `PlaceRequest` 가 요청의 모든 참조를 **기록·비교·
실행 전에 한 번** 배치한다. 진입점 둘이 다 부른다 — `chain up` 은 들어올 때,
테스트 엔진은 구성하면서(이 경로는 실행 전에 계획을 찍고 preflight 에 묻기
때문이다). `PlaceBinary` 는 멱등이라 두 번 배치해도 무해하고, **어느 한 곳도
혼자 짊어지지 않는다.** W4b 에서 넣었던 계획 전용 값은 없앴다 — 요청에 형태가
하나뿐이면 따로 들고 있을 것이 없다.

실측(`--plan`, 실제 workspace-config):

```
target     local /tmp/cbw4c/data
binary     /tmp/cbw4c/data/bin/gstable
```

**재구성 한 번**은 실제로 없었다. 증상 3 때문에 workspace-config 워크스페이스는
어차피 매번 재구성되고 있었다.

**막지 못한 것 — 넷째 자리.** 컴파일러는 여전히 이름과 경로를 구별하지 못한다.
이번에 무너진 것이 바로 규약이었다("이 함수를 불러라" — 세 호출자가 안 불렀다).
(가)는 그 자리를 둘로 좁히고 멱등으로 만들었지만, 종류 자체를 없애지는 못한다.
다음 안은 §11.2.6 에 적는다.

### 11.2.6 실행 경계에 타입을 붙이는 안 (미착수)

**아이디어.** `PlaceBinary` 의 반환에 이름 있는 타입을 준다(`node.BinaryPath`).
그러면 exec 으로 가는 자리(`node.LaunchReq.Binary`)가 그 타입을 요구하고,
컴파일러가 요구를 **거꾸로 끌고 간다** — `binaryFor` 가 그 타입을 돌려줘야 하고,
`state.Binaries` 값이 그 타입이어야 하고, 그것을 채우는 쪽은 `PlaceBinary` 를
거칠 수밖에 없다. `PlaceBinary` 가 유일한 생산자이기 때문이다.

**얻는 것.** 증상 2 — **배치를 안 거친 참조가 exec 에 닿는 것** — 이 구조적으로
불가능해진다. 규약으로 피하는 것이 아니라 컴파일이 안 된다. 셋 중 유일하게
**위험한** 증상이다(1 과 3 은 낭비와 헛된 거부이지 잘못된 바이너리가 도는 것이
아니다).

**막는 것이 무엇이 아닌지 분명히 해 둔다.** 한 망에서 여러 빌드가 도는 것은
**막을 대상이 아니라 지켜야 할 기능**이다. 실제로 셋이 그렇게 돈다 —
`stablenet-bp4-default-mismatch`(default+mismatch, swapNode),
`stablenet-bp4-en1-default-upgrade`(TC-3-1-04, 노드를 다른 바이너리로 재기동),
`wbft-from-to`(포크를 사이에 두고 from→to). 타입은 **값의 형태**를 강제하지
개수를 제한하지 않는다. 한 망에 배치된 경로가 둘이면 그대로 둘이다.

실측(`--plan`, workspace-config 적용):

```
binary  /tmp/cbw4c/data/bin/gstable
        (per node: default=/tmp/.../bin/gstable, upgrade=/tmp/.../bin/gstable)
```

### 11.2.7 핸드오프는 두 번째 컴포저다 (2026-09-17, 재검토)

**앞서 적었던 두 문장이 틀렸다.** 지우지 않고 무엇이 틀렸는지 남긴다.

> ~~"핸드오프는 설계상 로컬 전용이다"~~ — 아니다. `HandoffInputs` 에
> `MultiMachine`, `Machine func(index)`, `Placement`, `DialURL`, 원격
> `Files`/`Driver`/`Exec` 가 **다 있다.**
>
> ~~"배치하는 것이 옳은지 자체가 결정 사항이다"~~ — 아니다. `chainbench upgrade`
> 는 **이미** workspace-config 로 배치한다(`app/upgrade.go:144`,
> `resolveBinaryOn` → `wc.BinaryPath`).

#### 핸드오프는 구조적으로 특별하지 않다

노드마다 다른 것은 셋뿐이고, 셋 다 **"이 노드가 어떤 바이너리로 도는가"** 에서
따라온다.

1. **바이너리** — 포크 이전 것과 이후 것.
2. **genesis** — 같은 base 에 포크 이후 동작을 위한 설정이 더해진 두 번째 문서.
   이전 바이너리 노드는 첫 번째를, 이후 바이너리 노드는 두 번째를 쓴다.
3. **그 바이너리가 요구하는 레이아웃** — nodekey 디렉터리 이름
   (`Profile.Chains.From/To.NodekeyDir`), IPC 소켓 이름, RPC 네임스페이스.

나머지는 **보통 구성과 같다.** 어느 머신에 놓을지, 포트, 키 출처, config(공통이든
노드별이든), 부트스트랩 순서 — 다르지 않다. 실행 커맨드도 바이너리·genesis·config
의 이름만 바뀐다.

#### 보통 경로가 이미 하는 것

| 필요한 것 | 보통 경로 | 근거 |
|---|---|---|
| 노드별 바이너리 | ✅ | `node.Record.Binary` + `state.Binaries` + `binaryFor` |
| 노드별 config | ✅ | `node.Entry.Config`, `writeNodeConfig` |
| 배치·서버셋·원격·포트 | ✅ | `eachMachine`, `resource.Access` |
| poa 부트스트랩 (원격 포함) | ✅ | `runPhaseActions` + `poa.Bootstrap`, 노드별 `exec.Access` |
| 바이너리로 갈리는 IPC 경로 | ✅ | `node.Layout.IPCPath(label, binary)` |
| **노드별 genesis** | **❌** | `state.GenesisPath` 가 **망 전체에 하나**. `Init` 이 머신당 한 번 읽어 그 머신의 모든 노드에 같은 바이트를 준다(`steps_lifecycle.go:80-111`) |

**확인된 공백은 노드별 genesis 하나다.** 다른 것도 있는지는 전수로 보지 않았다.
찾은 작은 것 하나 — `runPhaseActions` 는 `spec.Binary = bin` 으로 **단일
바이너리**를 쓴다(`steps_lifecycle.go:1002`). `binaryFor` 가 아니다. 섞인 망에서
phase 액션은 노드가 실제로 도는 바이너리로 돌지 않는다.

#### 그래서 무엇이 중복인가

`internal/consensus/upgrade` 가 **비테스트 1,959줄**로 두 번째 컴포저를 들고 있다
(handoff 1039 · plan 326 · exec 263 · profile 153 · mesh 125 · launch 53).
자기만의 `WriteConfig` `BaseGenesis` `ComposePlan` `ApplyOverlay` `Launch`
`WireMesh` `machineFiles` `provisionKeys` `label` 포트 계산이 다 있다. 보통 경로에
있는 것과 **같은 일**이다.

#### 진짜 결함 — 표면 둘의 배선이 딴판이다

호출자가 둘인데 채우는 양이 다르다.

- **`chainbench upgrade`**(`app/upgrade.go:152`) — 전부 채운다. 서버 배치,
  `MultiMachine`, `DialURL`, `Machine`, 원격 `Files`/`Driver`/`Exec`, 그리고
  workspace-config 를 거친 바이너리 해석.
- **`chainbench run <업그레이드 케이스>`**(`testengine/compose.go:224`) — **일곱
  필드만** 채우고 끝이다. host 없음, placement 없음, machine 없음, files/driver
  없음, workspace-config 안 읽음, 바이너리 배치 안 함.

핸드오프 분기가 workspace-config 블록보다 **먼저 return 하기 때문**이다.

실측 — `--workspace-config` 가 `dataRoot: /tmp/cbw4c/data`, `paths.binaries: bin`
을 적었는데:

```
chain      wbft  (consensus handoff)
binaries   from gwemix -> to gwbft     ← 이름 그대로, target 줄 자체가 없다
```

**같은 분기가 genesis·launch·key-source override 는 소리 내어 거부한다.**
workspace-config 만 조용히 사라진다.

#### 배치 정책이 셋으로 갈려 있다

`wc.BinaryPath` 라는 규칙 자체는 하나인데, 그것을 감싸는 정책이 셋이다.

- `chainsetup.PlaceBinary` — wc 가 없으면 이름을 타깃 PATH 에 맡긴다.
- `app.resolveBinaryOn` — 원격에서 wc 가 없으면 **거부**하고, 배치 뒤 타깃에
  파일이 있는지까지 **확인한다**. 원격에서는 로컬 PATH 를 믿을 수 없으니 이쪽이
  더 엄격한 것은 타당하다.
- `app.ResolveBinary` — 로컬 `exec.LookPath`.

셋이 다른 것이 곧 결함은 아니지만, **어느 것을 언제 쓰는지 적힌 곳이 없다.**

#### 그래서 할 일

**2번 완료 (2026-09-17).** `phasePlan` 이 노드마다 자기 바이너리를 준다.
`poa.Bootstrap` 의 `Binary` 덮어쓰기도 없앴다 — 실행기는 **이미** 플랜의 노드
항목을 우선하고 있었는데 그것을 덮고 있었다. IPC 소켓 경로가 바이너리로
갈리므로, 섞인 망에서 부트스트랩은 그 노드가 만들지 않는 소켓을 기다렸다.

**1번 완료 (2026-09-17).** 선언이 바이너리별 genesis 를 말할 수 있다.

```json
"genesis": {"perBinary": {"next": {"set": {"config.croissantBlock": 20}}}}
```

genesis 단계가 그것을 **빌드된 genesis 위에** 병합해(`genesis.Customize`, 망
자신의 overlay 와 같은 병합·포크순서 재검증) `genesis-<binary>.json` 으로 모든
머신에 쓰고 어디에 놨는지 기록한다. 그 바이너리를 도는 노드는 그것으로 init
하고, 나머지는 망의 것을 그대로 쓴다. **템플릿에서 다시 빌드하지 않고 빌드된
바이트에 얹는다** — 두 번째 문서가 첫 번째 + 그쪽이 필요한 것임이 그 자체로
드러나고, genesis 를 자기 바이너리가 생성하는 계열을 두 번 돌리지 않는다.

`genesisFor` 는 `binaryFor` 를 의도적으로 그대로 따른다. 노드의 바이너리와
genesis 는 **같은 질문을 두 번 묻는 것**이고, 답이 둘로 갈리는 것이 이 결함의
모양이었다. 삭제·기록·기동 전 점검은 모든 문서를 훑는다. 아무 노드도 안 도는
바이너리에 genesis 를 선언하면 거부한다.

**딸려 나온 것.** 노드 표가 **일부** 노드에만 바이너리를 붙이면, 나머지가
선언의 `default` 가 아니라 **처음 지정된 노드의 바이너리**를 받고 있었다.
4노드 중 2개를 후속 바이너리에 올리면 **4개 다** 후속 바이너리로 돌았고,
계획도 그렇게 적었으며 이상해 보이는 곳이 없었다. 실측 후 고쳤다.

**3번 완료 (2026-09-17).** 배선이 `upgrade.Target` 으로 내려왔고 두 표면이 같은
것을 부른다. `app` 의 사본(`openHandoffTarget` `placeHandoff` `handoffMachines`
`spansHosts` `firstPlacedServer` `serverNamesByIndex` `handoffExec`)은 지웠다.

덤으로 하나 더 고쳤다. **로컬 핸드오프도 환경 파일을 읽는다.** 환경 파일은
파일이 어디 있는지를 말하고, 그것은 이 머신에도 똑같이 참이다. 실측:

```
workspace  /tmp/cbw4c/data
binaries   from /tmp/cbw4c/data/bin/gwemix -> to /tmp/cbw4c/data/bin/gwbft
```

환경 파일이 없으면 아무것도 안 바뀐다 — 데이터 루트는 워크스페이스, 이름은 이름.

**4번 판정은 §11.2.8.**

**확인하지 않은 것.** 노드별 genesis 를 넣었지만 핸드오프가 정말 보통 경로로
표현되는지는 **아직 안 돌려 봤다.** 부트스트랩 순서(etcd init, governance
deploy, await fork)가 phase 액션으로 다 표현되는지도 보지 않았다. 4번은 그
확인 뒤의 이야기다.

**그리고 지금 핸드오프는 genesis 를 하나만 쓴다.** `Launch` 가 `plan.Genesis`
한 벌을 모든 머신에 쓰고, `forkPrereqs` 가 후속 바이너리가 요구하는 필드
(`chainId`, `petersburgBlock`)를 **그 한 문서에** 더한다. 즉 지금 동작하는
이유는 **이전 바이너리가 그 필드들을 받아 주기 때문**이다. 문서가 둘 필요한
경우는 이전 바이너리가 거부하는 설정이 후속에 필요할 때이고, 그것이 이번에 넣은
기능이 대비하는 경우다.

### 11.2.8 `upgrade` 를 흡수할 수 있나 — 판정 (2026-09-17)

> **완료 (2026-09-18).** 흡수했다. `internal/consensus/upgrade` 는 2,422줄에서
> **118줄**(하드포크 preset 로더)이 됐고, `upgrade run`·`upgrade genesis` 명령과
> `chainbench_upgrade` MCP 도구도 같이 내렸다 — 합쳐서 4,894줄. 하드포크는
> `env.upgrade` 로 선언하고 보통 경로로 구성되며, 방식이 둘이다(concurrent /
> restart). 라이브 케이스 넷이 통과하고, 그중 하나는 docker 서버셋 15대에도
> 올렸다. 전 과정과 실측은
> [`handoff-absorption-design.md`](handoff-absorption-design.md) §0 에 있다.

**할 수 있다. 막는 것은 구조가 아니라 작고 구체적인 사실 넷이었고, 넷 다
2026-09-17 에 정리됐다.** 남은 것은 §11.2.9 가 지목한 **config 단계** 하나다.

**넷 중 셋이 같은 모양이었다.** 프로필이 **선택이 아니라 사실을 다시 적고**
있었다 — validator 집합은 키셋에서 유도되고(§11.2.10), 거버넌스 정책은
기본값과 동일하며(§11.2.11), PlanOrder 는 플랜이 생산자를 앞에 놓기로 한 탓에
생긴 보정이다(§11.2.12). 넷째(nodekey 디렉터리)는 내가 거꾸로 읽었다(§11.2.9).

#### 이미 공유하고 있는 것 (코드로 확인)

**부트업 순서는 이미 같은 선언을 읽는다.** `Handoff.phases()` 가
`From.Family().BringUpPhases(roles)` 를 부른다. 주석이 직접 적는다 — "순서는 이
함수가 지어낼 것이 아니다. consensus family 가 선언하고, 구성 경로는 F3 이후로 그
선언을 따라 왔다."

**부트 액션도 그 선언에 들어 있다.** `poa.Family.BringUpPhases` 가
`Actions: [deploy-governance, etcd-init, verify-etcd]` 를 돌려주고, 보통 경로의
`runPhaseActions` 가 그것을 실행한다. **그런데 `phases()` 는 `Nodes` 만 읽고
`Actions` 를 버린 뒤, `Run` 이 그 셋을 손으로 다시 부른다.** 중복이 가장 선명한
자리다.

**나머지 셋은 이번 작업으로 들어왔다** — 노드별 바이너리(2번), 노드별
genesis(1번), 타깃·배치 해석(3번).

**거버넌스 파라미터도 보통 경로에 있다.** `poa.GenesisSource.Env *Env` 가 그것
이고 "zero value 는 DefaultEnv" 다. 프로필의 값을 못 받는 것이 아니라 **받을 DSL
문법이 없을 뿐**이다.

#### 못 하는 것 넷

| # | 프로필이 가진 것 | 보통 경로 | 크기 |
|---|---|---|---|
| 1 | `Identities.PlanOrder` — 플랜 노드 k 가 프리셋 노드 N 의 키를 쓴다 | **노드 표가 순서를 고르면 재매핑이 필요 없다. §11.2.12** | 해소 |
| 2 | `Chains.From/To.NodekeyDir` — 바이너리마다 nodekey 를 찾는 디렉터리가 다르다 | **내가 거꾸로 적었다. §아래 참조** | — |
| 3 | `Validators.Addresses/BLSPublicKeys/ExtraData` — 미리 계산된 값 | **전부 유도된다. 대조 완료 — §11.2.10** | 해소 |
| 4 | `Producers.Governance` — 거버넌스 정책 | **기본값과 필드 하나하나 동일하다. §11.2.11** | 해소 |

그 밖에 피어링이 다르다. 핸드오프는 `static-nodes.json` 을 쓰고 **추가로**
`admin_addPeer` 로 메시를 건다. 보통 경로는 static-nodes 만 쓴다.

#### 순서 (착수 전)

1. **완료 (2026-09-17).** 조사해 보니 적어 둔 것보다 넓었다. 오케스트레이션이
   **둘**이었고 **둘 다** 가족의 선언을 안 따랐다.

   - `Handoff.Run` 은 첫 단계만 쓰고 나머지 단계를 **전부 하나로 뭉개** launch 했다.
   - DSL 경로(`handoffUp`)는 **다섯 노드를 한꺼번에 띄우고** 그 다음 governance 를
     배포했다 — `Run` 의 주석이 경고하는 바로 그 순서다(poa 클러스터는 생산자가
     혼자일 때만 형성된다).
   - 둘 다 deploy-governance·etcd-init·verify-etcd 를 **손으로** 불렀고,
     **`etcd-join` 은 아무도 안 돌렸다.**

   생산자가 1이면 셋 다 안 드러난다. 프로필이 1이다.

   `Handoff.BringUp` 이 선언된 단계를 걸으며 각 단계의 노드를 띄우고 그 단계가
   선언한 액션을 **가족 자신의 실행기**(보통 경로가 쓰는 그것)로 돌린다. 두 표면이
   그것을 부른다.

   두 번째 사본을 살려 두던 차이가 셋이었는데 둘은 이미 표현 가능했고(부트
   keystore, 그리고 양쪽이 genesis 옆에 같은 이름으로 두는 거버넌스 config),
   셋째만 없었다 — **비밀번호 경로**. 핸드오프는 그것을 키가 아니라 망 데이터 옆에
   쓴다. `poa.Bootstrap.Password` 로 받는다.

   **라이브 검증.** 바꾸기 전 기준선을 잡고 두 표면 다 돌렸다. 결과 동일 —
   거버넌스 배포, 클러스터 형성, 블록 21-30 을 후계 검증자 4/4 가 전부 봉인.
   순서만 바뀌었다.

   ```
   launch:boot: 1 node(s)        ← 전: launch: 5 node(s)
   deploy-governance: on node1
   etcd-init: on node1
   verify-etcd: on node1
   launch:endpoints: 4 node(s)
   mesh: 5 endpoint(s) meshed
   ```
2. ~~**#2** `Layout.NodekeyPath(label, binary)`~~ — **지을 것이 없었다(2026-09-17).
   §11.2.9 참조.**
3. ~~**#4** 거버넌스 정책의 DSL 문법~~ — **지금은 안 짓는다(2026-09-17).
   소비자가 없다. §11.2.11.**
4. ~~**#3** 유도값과 프로필 값 대조~~ — **완료(2026-09-17). 셋 다 일치.
   §11.2.10.**
5. ~~**#1** PlanOrder~~ — **해소(2026-09-17). §11.2.12.**
6. **흡수.** 설계는 `handoff-absorption-design.md` 에 있고 **결정까지 끝났다**
   (2026-09-17). 순서는 ① 노드별 config 파일 ② `binaries` 값에 체인
   ③ `upgrade` 선언 확장 ④ 보통 경로로 돌려 대조 ⑤ `upgrade` 축소.
   **①과 ②는 ④의 결과와 무관하게 이득이다.**

   정해진 것 셋. `env.upgrade` 는 **남기고 모든 하드포크의 자리로 넓힌다** —
   방식이 `concurrent`(지금 것)와 `restart`(보통 하드포크) 둘이고, restart 는
   **문법 자리만 두고 이름을 대며 거부한다**(쓰는 케이스가 없다). 프로필은
   **하드포크 preset** 이 된다. `AwaitFork` 는 **테스트 쪽**이다.

   **체인 코드로 확인한 것** — 하드포크 설정은 config 파일의 genesis 섹션으로
   적용되고(세 저장소 동일), 노드 초기화는 포크 이전 genesis 로 해야 하며,
   genesis 블록 해시가 같아야 하므로 **chain config 의 포크 블록만 달라질 수
   있다.** 런타임 override 플래그는 Cancun·Verkle 뿐이라 쓸 수 없다.

**라이브 전제 (2026-09-17 실측).** 핸드오프는 세 가지가 있어야 돈다 —
`GWEMIX_BIN`, `GWBFT_BIN`, 그리고 **`GOWEMIX_TEMPLATE`**(go-wemix 자신의
`wemix/scripts/genesis-template.json`). 셋째가 없으면 "a profile and a go-wemix
genesis template are required" 로 준비 단계에서 멈춘다.

**여전히 확인하지 않은 것.** 생산자가 **둘 이상인** 핸드오프는 안 돌려 봤다.
`etcd-join` 과 단계별 순차 기동은 이번에 비로소 돌 수 있게 됐지만, 실제로 도는
것은 못 봤다. 프로필이 생산자 1을 쓰기 때문이다.

### 11.2.12 PlanOrder 는 순서로 풀린다 (2026-09-17, 실측)

`plan_order: [5, 1, 2, 3, 4]` 가 왜 있는지 프리셋을 열어 보고 알았다.

**프리셋 node5 는 nodekey 주소와 keystore 계정이 다르다.**

```
node5/address                    0x5400d8b543eaf6738c7b44799623bea88fd0f5ee
node5/keystore/UTC--node5-...    0xf9593d358b373d354a348c00887b914b408f6984
                                 ↑ 프로필의 producers.members 가 이것
```

일부러 그렇게 돼 있다. **생산자가 봉인하려면 잠금 해제할 수 있는 계정이
필요한데, 그 계정의 keystore 가 node5 슬롯에 있다.** 그래서 생산자는 무슨 일이
있어도 node5 에 앉아야 한다.

**그런데 핸드오프 플랜은 생산자를 앞에 놓는다**(`nodes 1..P 가 from-chain
miner`). 앞에 놓기로 정해 두고 나니 "1번이 5번 신원을 쓴다"는 재매핑이 필요해진
것이다. **노드 표는 순서를 스스로 고른다.** 생산자를 node5 로 선언하면
프리셋 5가 그 자리에 자연히 오고, 후계자는 node1~4 에 프리셋 1~4 가 온다 —
**프로필이 만들어 내는 매핑과 같고, 재매핑은 없다.**

`topology.nodes[].key` 로도 절반은 된다(nodekey). 하지만 keystore 는 못 옮긴다 —
보통 경로는 keystore 를 `<키셋>/node<N>/keystore` 에서 **노드 인덱스로** 찾는다.
그러니 답은 키를 옮기는 것이 아니라 **순서를 고르는 것**이다.

**규칙으로도 재현된다** — "생산자가 프리셋의 뒤쪽 슬롯을 가져간다"로 계산하면
`[5 1 2 3 4]` 가 그대로 나온다. 다만 **생산자가 1개뿐이라 근거가 약하다.**
일반화는 확인하지 않았다.

**회귀 테스트는 목적 쪽에 걸었다** — 생산자 슬롯의 keystore 가 프로필이 스테이킹
하는 계정을 갖고 있는지. 둘이 어긋나면 생산자가 **키 없는 계정을 잠금 해제하려다**
"no key for given address or file" 로 죽는데, 그 메시지는 원인 파일을 말해 주지
않는다. `plan_order` 를 `[1,2,3,4,5]` 로 바꿔 실제로 잡히는 것을 확인했다.

### 11.2.11 거버넌스 정책은 기본값을 다시 적은 것이다 (2026-09-17, 대조 완료)

문법을 짓기 전에 **소비자를 먼저 찾았다. 없었다.**

처음에 anzeon basefee 케이스들이 소비자일 거라고 봤는데 **틀렸다** — 그것들은
go-stablenet 이고 wemix 거버넌스와 무관하다. 남은 소비자는 프로필뿐이었고,
프로필 값이 기본값과 다른지 봤더니 **필드 열셋이 하나하나 같았다.**

```
ballot_duration_min 86400 · ballot_duration_max 604800
staking_min/max · max_idle_block_interval 5 · block_creation_time 1000
block_reward_amount · max_priority_fee_per_gas · reward_distribution [4000 1000 2500 2500]
max_base_fee · block_gas_limit 105000000 · base_fee_max_change_rate 55
gas_target_percentage 30
                              ← 전부 poa.DefaultEnv() 과 동일
```

**그래서 §11.2.10 과 같은 모양이다.** 프로필이 선택을 적은 것이 아니라 기본값을
다시 적었다. 보통 경로의 genesis 소스는 `Env` 를 받고 바로 이 값으로 기본값을
잡으므로, 흡수를 막는 항목에서 빠진다.

**문법은 짓지 않았다.** 소비자 없는 구조를 넓게 배선하는 것이 이 워크리스트가
P2·P3 에서 스스로 경고한 바로 그것이다. 대신 **회귀 테스트**를 뒀다 — 프로필이
기본값에서 벗어나면 실패하고, 실패 메시지가 "이제 선언할 방법이 필요하다" 고
말한다. 즉 **문법이 필요해지는 순간을 테스트가 알려 준다.** `gas_target_percentage`
를 30→40 으로 바꿔 실제로 잡히는 것을 확인했다.

### 11.2.10 프로필의 validator 블록은 전부 유도된다 (2026-09-17, 대조 완료)

프로필이 후계자 집합의 주소·BLS 공개키·RLP extra-data 를 미리 적어 둔다. 그
**셋 다** 키셋에서 유도된다. 프로필 자신이 선언한 `plan_order` 를 따라
`keys/preset` 의 노드 1~4 를 읽고 `wbft.ExtraData` 를 돌린 결과다.

```
plan_order        [5 1 2 3 4]         ← 1번은 생산자, 나머지가 후계자
addresses         일치 (4/4)
bls_public_keys   일치 (4/4)
extra_data        일치               ← 0xf90147… 300여 자
```

**그래서 이 블록은 사실을 적은 것이지 선택을 적은 것이 아니다.** 흡수를 막는
항목에서 빠진다 — 키셋이 이미 아는 것뿐이라 보통 경로가 못 만들 것이 없다.

**그리고 사실을 두 번 적는 것은 두 번 틀릴 수 있다는 뜻이다.** 프리셋을 고치거나
`plan_order` 를 바꾸면 프로필은 옛 집합을 계속 주장한다. 그러면 genesis 가 실제로
뜬 노드가 아닌 검증자를 지목하고, **동기화는 되는데 아무도 봉인하지 않는 망**이
된다 — 원인 파일에서 한참 떨어진 증상이다. 회귀 테스트를 뒀고, `plan_order` 를
두 자리 바꿔 실제로 잡히는 것을 확인했다.

**유도되지 않는 것 둘.** `validators.members`(거버넌스 협의회로 seed 되는 검증자
하나)는 선택이고, `producers.members` 는 노드5의 **keystore 계정**이라 nodekey 에서
나오는 주소가 아니다(프로필 주석이 그렇게 적는다).

### 11.2.9 2번은 거꾸로였다 (2026-09-17, 실측)

**내가 §11.2.8 에 적은 2번은 틀렸다. 두 군데가.**

> ~~"`Layout.NodekeyPath(label)` 이 바이너리를 안 받는다"~~ — 받을 필요가 없다.
> 게다가 **그 함수는 프로덕션 호출자가 없다.** 테스트 하나뿐이다.
>
> ~~"`IPCPath` 와 같은 무늬, 함수 하나 차이"~~ — 같은 무늬가 아니다. IPC 소켓
> 이름은 바이너리가 정하지만 nodekey 는 **알려 주면 되는 것**이다.

**보통 경로는 관례를 알 필요가 없다.** `--nodekey <키셋>/node<N>/nodekey` 로
**파일을 직접 가리킨다**(`process/nodeconfig.go:26` — 경로가 datadir 이 아니라
**키셋** 아래다). static-nodes 도 마찬가지로 노드별 config 에 적는다.

**실측** — 보통 경로로 gwemix 망을 구성해 통과시켰다
(`01-wemix-etcd-and-governance`, pass). 워크스페이스를 보면:

```
보통 경로:  node1/geth/nodekey     ← 바이너리가 만든 것. config_node1.toml 존재
                                     static-nodes.json 없음 (config 안에 있다)
핸드오프:   node1/geth/nodekey     ← 핸드오프가 쓴 것. .toml 없음
            node2/gwemix/nodekey     node2/gwemix/static-nodes.json
```

**그래서 진짜 원인은 함수가 아니라 단계다.** 핸드오프는 **노드별 config 파일 없이**
띄운다(`LaunchArgs` 가 `ConfigPath` 를 `nodeconfig.Spec` 에 넣지 않는다). config 가
없으니 nodekey 도 static-nodes 도 **각 바이너리가 관례로 찾는 자리**에 있어야 하고,
그래서 프로필이 `nodekey_dir` 을 들고 있다. **보통 경로의 config 단계를 주면 그
필드는 사라진다.**

**드롭인은 아니다.** `nodeconfig.TOML(Spec)` 은 순수 함수라 재사용 가능하지만,
핸드오프는 계정·RPC 네임스페이스·unlock 을 `nodeconfig.Override` 로 **argv 층에**
얹는다. config 파일을 쓰면 `Argv` 가 일부를 argv 에서 빼므로(주석: "A spec with a
ConfigPath leaves the auth port to the file"), 그 override 들이 파일 쪽으로도 가야
한다. `ApplyConfigOverride` 가 같은 키 타입을 받으니 길은 있다.

**판정.** 2번은 독립된 작은 항목이 아니라 **흡수(4번)의 한 조각**이다. 순서에서
빼고 흡수 쪽으로 옮긴다. 얻는 것은 프로필 필드 하나가 아니라 **핸드오프가 관례
대신 선언으로 돌게 되는 것**이다.

### 11.2.4 결과물이 쌓이는 자리 — 완료 (2026-09-17)

설계는 `stage-lifecycle-design.md` §6 에 있고, 승인받은 대로 들어갔다.

```
~/.chainbench/sessions/          ← 기본 루트, --artifact-root 로 덮는다
  20260917-073226-57155/         ← 세션: 프로세스 시작 시각 + pid
    UTC-20260917-073226/         ← 실행
      tests/001_<테스트 id>/     ← 판정과 증거
      report.json
```

**셋을 고쳤다.**

1. **순서를 뒤집었다.** 세션을 망 구축 **전에** 연다. 그래서 망이 안 뜨면 그
   테스트의 폴더가 생기고 `blocked` 로 적히고 증거가 `observations/` 로 간다.
   워크스페이스의 `failures/` 는 없앴다. **실측: `--binary /nonexistent/x` 로
   구성을 깨뜨리면 `tests/001_basic-consensus/status.json` 에 이유가, 그 아래
   `observations/` 에 수집 결과가 남는다.** 요약도 `blocked=1` 로 센다 — 전에는
   오류 문자열뿐이었다.
2. **세션(프로세스) 층을 넣었다.** `session.New` 한 곳에서 붙이므로 구성·attach·
   워크스페이스 attach 가 저절로 같은 규칙을 따른다. `session.List` 가 한 단계 더
   내려가고 옛 평면 배치도 계속 읽힌다. 대시보드 경로·세션 gc·정렬을 맞췄다.
3. **루트를 워크스페이스 밖으로 옮겼다.** 워크스페이스를 지워도 판정과 로그가
   남는다.

**아직 안 한 것 둘.**

- **체인 구성만 한 실행의 폴더 이름.** 설계 §6.1 은 테스트가 없을 때 "무엇을
  구성했는지" 로 폴더를 만들라고 적는다. 지금은 그 경로가 세션을 열지 않는다.
- **이름 빚.** 코드의 `session.Session` 은 실행 하나를 뜻하는데 디스크에서 세션은
  그 위층이다. 고치려면 `session.json` 과 170여 군데를 건드린다.

### 11.3 지금 바로 할 수 있는 것 (오프라인)

| 순위 | 항목 | 크기 | 왜 이 자리인가 |
|---|---|---|---|
| ~~**1**~~ | ~~**X11** repro 스크립트 3종~~ | 소 | **완료 (2026-09-18).** 셋을 지웠다 — 은퇴한 `net up` 을 불러 **돌아간 적이 없다**. 주제는 셋 다 `tests/tc` 에 DSL 케이스로 있다. 다만 스크립트가 얹고 있던 것이 하나 있었다: 능력을 켠 망에 그 능력으로 게이트된 케이스를 올리고 **스킵되면 실패**로 봤다 — 조용한 스킵은 능력이 닿지 않았다는 뜻이기 때문이다. 그것을 `run --no-skips` 로 옮겼다. 셋이 각자 갖고 있던 가드가 러너 하나로 갔고, 이제 어떤 케이스 목록에도 붙일 수 있다 |
| ~~2~~ | ~~**H3-b** 비교값의 계정 라벨 14곳~~ | 소 | **완료 (2026-09-18).** `of` 규칙을 골랐다 — 아는 이름이면 풀고 아니면 그대로. 자리를 추측하는 대신 값이 풀리는지만 본다. `tests/tc` 의 preset 주소 리터럴 **0곳**. §H3-b 참조 |
| ~~2b~~ | ~~**X10** 영역에 검사~~ | 중 | **완료 (2026-09-18).** `cmd/chainbench/cliflags_test.go`. 붙이자마자 죽은 호출 둘을 잡았다 — §12.3 참조 |
| ~~3~~ | ~~**N1** attach 선언~~ | 대 | **완료 (2026-09-18).** `env.attach` 를 뒀다. 만드는 형태와 배타이고, 게이팅은 `attach.provides` 로 걸리며, 명령행이 이긴다. §2.5.1 참조 |

**이 표는 2026-09-18 로 비었다.** 오프라인으로 할 수 있는 항목이 남아 있지 않다.
다음 일감은 전부 §11.4(라이브)와 §11.5(마지막 묶음)에 있다.

**M4 묶음은 끝났다(M4-a·M4-b·M4-c).** 이제 실행 전에 계획을 볼 수 있고, 계획의 각
값이 어디서 왔는지 말하고, 뜬 망이 계획과 다르면 테스트를 돌리지 않는다.

**P2·P3(값)는 1순위에서 내려와 있다.** 실제로 세어 보니 값에 막힌 케이스가 2건뿐이고
(anzeon basefee 상·하한), H2 의 gasTip 14건은 **이미 체인에서 읽고 있다** — "14개
파일" 은 낱말이 나오는 파일 수였지 하드코딩 건수가 아니었다. 2건을 위해 DSL 에 문법을
더하는 것은 소비자 없는 구조를 넓게 배선하는 것이다. §11.5 로 옮겨 둔다.

> **다만 이 근거는 실제 망 대응을 시작할 때 다시 봐야 한다 (2026-09-18).** "2건"
> 은 케이스만 센 숫자다. 항목으로 세면 **R1·R5·R6 셋이 P3 에 걸려 있고**(§8),
> R6 은 완료 조건 4번을 증명하는 항목이다. 소비자가 2건이 아니라 "케이스 2건 +
> R 묶음 절반" 이면 계산이 달라진다. **지금 올리자는 말이 아니라, 내린 근거를
> 그때 다시 세라는 말이다.**

### 11.3.1 케이스 208건의 현재 상태 (2026-09-18)

직전 전체 실행은 **통과 199, 스킵 6, 실패 2, 블록 1** 이었다. 9건이 전부였고, 9건
다 고쳐서 `run --no-skips` 로 통과를 실측했다.

| 무엇이었나 | 몇 건 | 원인 | 지금 |
|---|---|---|---|
| 스킵 | 6 | env 의 `capabilities` 가 **요구**인데 그걸로 능력을 주려 했다. 망에 능력을 주는 자리가 DSL 에 없었다 | `genesis.provides` 를 만들고 env 셋을 새로 뒀다 |
| 실패 | 1 | `04b` 의 주장이 틀렸다 — 멤버 추가는 검증자를 만들지 않는다 | `configureValidator` 로 다시 썼다. 그 과정에서 노드 간 읽기 경합도 찾았다 |
| 실패(간헐) | 1 | `04-basefee` 가 빈 블록 하나에 깨지는 구조였다 | 블록 짝 비교로 바꿨다 |
| 블록 | 1 | 준비 판정이 "멈추는 것이 정답인 체인" 을 몰랐다 | `genesis.haltsAt` |

**그 뒤로 케이스가 하나 늘어 209건이다** (`08-attached-chain-produces`, N1 의
증거). 실측한 것은 그 9건과 H3-b 로 라벨을 바꾼 6건, 합쳐 **15건**이고 **나머지
194건은 안 돌렸다** — §11.4 의 L1 이 그것이다.

---

### 11.4 라이브가 있어야 하는 것

**오프라인 검사가 답을 주지 못하는 자리다.** 넓히는 것이 의도이므로 판정표 비교도
X8 가드도 "맞다" 고 말해 주지 않는다.

| 순위 | 항목 | 무엇 |
|---|---|---|
| ~~P5-L1~~ | ~~남은 게이트를 넓힌다~~ | **완료 (2026-09-18): 66 → 0.** 현재 쓰이는 게이트는 `rpc` 208, `contract:*` 82, `consensus` 45, `process` 15, `engine:anzeon` 15, `fork:boho` 6, 기타 소수다 |
| ~~**1**~~ | ~~**L1 전체 회귀 1회**~~ | **완료 (2026-09-20). 209건, 약 4시간, 도커 15서버.** 통과 151·실패 48·스킵 6·실행불가 4. **실패 48건이 하나도 빠짐없이 설명된다** — 원인 불명 0건. 아래 §11.4.1 |
| ~~**2**~~ | ~~**L2 노드 간 읽기 경합 전수**~~ | **완료 (2026-09-20).** 전수 감사 **123지점 56케이스**, 그중 **10건이 L1 에서 실제로 깨졌고** 고쳤다. L1 이 표본을 준다는 예상대로였다 — 반복 실행은 하지 않았다. 아래 §11.4.2 |
| 3 | **X5 라이브 절반** | 기준선의 Stablenet 간헐 실패. **단발 실행으로는 판정 불가** — 기준선 자신이 재실행 PASS 를 기록한다. §B1~B6 의 동시 관측이 필요하다 |
| 4 | **X14** | e2e 간헐 실패. **다음에 실패할 때** 문구를 잡아 분석한다. 2026-09-20 의 L1 은 이 테스트를 돌리지 않았다(게이트된 Go e2e 이지 `tests/tc` 케이스가 아니다) — 표본이 늘지 않았다 |
| 5 | R2 → R3 → R4 → (P3) → R1 → R5 → R6 | 실제 망 대응. **순서를 고쳤다 (2026-09-18)**: 예전 표기 `R2 → R3 → R5 → R4 → R6` 은 자기 의존과 어긋났다 — R1 이 아예 빠져 있는데 R6 이 R1 에 의존하고, R1·R5 는 둘 다 P3 에 의존한다. P3 는 지금 §11.5 로 내려가 있으므로 **R1·R5·R6 은 P3 가 올라오기 전까지 열리지 않는다.** P3 없이 갈 수 있는 것은 R2·R3·R4 다 |
| 6 | Z2 | local·remote(docker) 두 곳 통과 |

**L1 이 덮어야 할 변경 (2026-09-18).** 전부 이날 들어갔고, 아래 넷은 **모든 케이스가
지나는 해석 경로**다.

| 무엇 | 왜 넓히기만 하는가 | 그래도 남는 위험 |
|---|---|---|
| `uintArg` 가 0x-hex 도 받는다 | 전에는 그런 값이 실패했다 | 숫자 인자에 0x 로 시작하는 문자열을 **일부러** 넣어 실패를 기대하던 케이스가 있으면 뜻이 바뀐다 |
| `derive abiCall` 의 `{"bytes": …}` | 전에는 map 이 `parseBigValue` 에서 오류였다 | 없음에 가깝다 |
| `derive` 의 `format` | 키가 없으면 예전과 같다 | 없음에 가깝다 |
| 비교값·`params` 이름 해석 | 키 집합이 아는 이름만 바꾼다 | **비게 됐다 (2026-09-20).** 적어 둔 위험이 그대로 일어났다 — `admin_wemixInfo.self.name` 이 `"node1"` 을 돌려주는데 기대값이 주소로 바뀌어, 정상 동작하는 망에서 `default-on-routes-every-step` 이 깨졌다. 골든이 못 잡은 이유도 여기 적힌 그대로다(주소만 본다). 체인의 답이 hex 가 아니면 이름으로 비교한다 |
| overlay 문서에 `capabilities`·`haltsAt` | 둘 다 없으면 바이트가 같아 경로 해시가 안 변한다 | 없음에 가깝다 |
| 준비 게이트의 `haltsAt` 분기 | `haltsAt` 0 이면 예전 그대로다 | 없음에 가깝다 |
| `env.attach` | 만드는 형태는 `EnvAttach` 가 nil 이라 분기에 안 닿는다 | 없음에 가깝다 |
| 노드 계정 unlock 조건 | **되돌렸다** — 손대지 않았다 | 없음 |

**라이브가 실제로 된다는 것은 확인했다** — 세 체인 다 띄워 봤고, e2e 19건이 한 번은
전부 통과했다(673초). 전제는 §11.5 에 적는다.

### 11.4.1 L1 전체 회귀 — 결과 (2026-09-20)

**209건, 도커 15서버, 15노드 `bp7 · en7 · pn1` proxied, 약 4시간.**
통과 151 · 실패 48 · 스킵 6 · 실행불가 4.

케이스마다 망을 새로 세운다 — 한 `run` 이 여러 spec 을 받아도 그렇다(실측). 그래서
상태 누적은 없고, 대신 케이스당 약 114초라 §11.4 가 잡은 "약 2시간" 이 4시간이 됐다.
그 추정은 각 케이스의 작은 토폴로지 기준이었다.

**실패 48건이 하나도 빠짐없이 설명된다 — 원인 불명 0건.**

| 분류 | 건수 | 무엇 |
|---|---:|---|
| A | 19 | `--env` 가 env 의 *동작*(하드포크 일정·genesis overlay·바이너리 스왑·attach)을 지웠다 |
| B1·B1b | 4 | 키가 `keys/preset` → `generate` 로 바뀌어 계정·주소가 달라졌다 |
| B2 | 5 | `waitFor` 예산이 4노드 망 기준이다 |
| B3 | 5 | 어세션이 검증자 수를 값으로 박고 있다 |
| B4 | 4 | 정족수·피어 수가 4노드 전제다 |
| L2 | 10 | **노드 간 읽기 경합** — §11.4.2 |
| WS | 1 | ws 다이얼이 도커 localmap 을 건너뛴다 |
| **회귀** | **1** | **비교값 이름 해석** — 아래 |

**찾은 회귀는 하나다.** `admin_wemixInfo.self.name` 은 `"node1"` 을 돌려주는데 기대값의
`"node1"` 이 주소로 해석돼, 정상 동작하는 망에서 `default-on-routes-every-step` 이 깨졌다.
§11.4 의 위험 표가 **이것을 미리 적어 뒀고**, 골든이 못 잡는 이유(주소만 본다)까지 맞았다.
체인의 답이 hex 가 아니면 이름으로 비교하게 고쳤다 — 비교값에 이름이 들어가는 7곳 중
주소를 돌려주는 5곳은 그대로다.

**"전부 15노드" 로 돌린 값.** A·B 다섯 경로를 겹침 빼고 세면 **약 60건이 원래 의도대로
시험되지 않았다.** 그중 29건을 자기 env 로 다시 돌려(A29) **28건 통과**를 확인했다 —
진단이 맞았다는 증거이자, 그 조건의 비용이다. `*-chain-up-15` 와 `wbft-quorum-at-15-nodes`
가 **통과**한 것이 같은 말을 한다: 선언한 노드 수와 실제가 맞으면 통과한다.

**무엇이 아직 안 닫혔나 — 바이너리.** 이 실행은 컨테이너에 있던 **2026-08-10 빌드**
(`0937ac5c9`)로 돌았다. 호스트에 HEAD 를 빌드했지만 fleet 에 올라가지 않았다:
`PlaceBinary` 는 절대 경로를 **타깃 위의 경로**로 해석하고, 15노드 env 는 `binaries`
블록이 없어 매니페스트 이름을 풀어 이미 있던 파일을 썼다. 그러므로 **"209건이 현재 체인에서
통과한다" 는 말은 성립하지 않는다.** 회귀 1건과 L2·WS 는 chainbench 쪽이라 영향받지 않는다.

---

### 11.4.2 노드 간 읽기 경합 — 전수 감사와 수정 (2026-09-20)

§11.2.13 이 `04b` 한 건만 실측하고 "전수 점검은 아직 안 했다" 로 남긴 항목이다.

**감사는 양방향이어야 했다.** 처음 판은 `비기본 노드 → 기본 노드` 만 찾아 41케이스
63지점을 냈는데, 더 흔한 모양은 반대다 — **기본 노드가 계정에 자금을 넣고 그 계정이
en1 에서 행동**한다. en1 이 그 블록을 못 받았으면 잔고가 0 이라 **의도한 사유가 아닌
"insufficient funds" 로 거절**된다. `claim-zero-refund-reverts` 가 L1 에서 깨진 것이
이것이었다. 양방향으로 다시 세면 **123지점 56케이스**다.

**고친 것은 실제로 깨진 10건, 17곳이다.** 나머지 106지점은 그대로 뒀다 — 의도된 것이
섞여 있고(전파 테스트는 그 간격을 원한다), 측정하지 않은 자리에 대기를 넣으면 테스트가
약해지면서 고친 것처럼 보인다. 해법은 `04b` 가 쓴 것 그대로다: 읽는 노드에서
`eth_getTransactionReceipt` 를 기다린다. 10건 라이브 통과.

> **남은 106지점은 후보이지 결함 목록이 아니다.** 정적 분석은 뒤 스텝이 앞 변경에
> 실제로 의존하는지 모른다. 다음에 거버넌스 케이스가 간헐적으로 깨지면 이 목록을
> 먼저 본다 — 재생성은 `scripts/` 가 아니라 이 절의 규칙(변경을 확정한 노드와 그것을
> 쓰는 노드가 다른데 사이에 대기가 없다)으로 다시 돌리면 된다.

**왜 지금 드러났나.** 15노드 proxied 는 en 이 pn 을 거쳐야 해 전파가 느리다. 4노드에서
"우연히 통과" 하던 것이 그 여유를 잃었다. §11.2.13 이 적은 예측 그대로다.

---

### 11.5 마지막

| 항목 | 크기 | 내용 |
|---|---|---|
| Z1 | 대 | 모든 테스트를 DSL 문법으로 (별도 트랙) |
| Z3 + C2 | 중 | `V` 버전 표기 제거, v1 재정리, 주석 전수 감사 |
| P2 · P3 | 중 | preset 의 `values` 블록과 `${…}` 초기값. **막힌 케이스가 2건뿐이라 내려와 있다.** 다만 항목으로는 R1·R5·R6 셋이 P3 에 걸려 있으므로(§8), **R 묶음을 시작할 때 이 순위를 다시 센다** — §11.3 의 상자 |
| N1-b | 소 | attach 의 `chain` 을 RPC 로 읽어 지운다. 지금은 컨트랙트 표가 첫 호출 전에 있어야 해서 요구한다(§2.5.1) |
| H4 (2단계) | 소 | 허용 목록 줄이기. 묶음을 치울 때마다 줄인다고 정해 뒀고, H3-b 로 주소 리터럴이 0 이 됐으니 지금이 줄일 때다 |
| ~~X9~~ | — | **완료 (2026-09-18).** go-stablenet 을 읽어 닫았다. 그 체인의 고유 포크는 `applepie` 와 `boho` 둘인데 매니페스트에는 `boho` 만 있었다. `applepie` 는 죽은 이름이 아니다 — `IsApplepie` 가 수수료 대납 tx 타입을 막는 게이트다(`core/state_transition.go`, `core/txpool/validation.go`). 매니페스트에 넣었다. **`fork:applepie` 를 요구하는 케이스는 아직 없다**: 수수료 대납 케이스 여섯은 `requires: ["rpc"]` 뿐이라, applepie 가 없는 망에서는 건너뛰지 않고 실패한다 |

### 11.6 라이브를 돌리는 전제 (2026-09-16 실측)

| | 값 |
|---|---|
| 저장소 | `$CHAIN/{go-stablenet,go-wbft,go-wemix}`. **경로는 기계마다 다르다** — 2026-09-16 실측 기계는 `~/Work/github/chain`, 2026-09-19 기계는 `~/work/github/wemade` 였다 |
| stablenet | `GSTABLE_BIN=<go-stablenet>/build/bin/gstable` |
| wbft | `WBFT_BIN` 또는 `GWBFT_BIN=<go-wbft>/build/bin/gwemix` — **이름이 `gwemix` 다**(X4) |
| wemix | `WEMIX_BIN=<go-wemix>/build/bin/gwemix` |
| 핸드오프 | `GOWEMIX_TEMPLATE=<go-wemix>/wemix/scripts/genesis-template.json` |
| 워크스페이스 | **짧은 경로**. 긴 경로면 노드가 `bind: invalid argument` 로 죽는다(유닉스 소켓 104자) |
| e2e 전체 | 약 11분 |

### HANDOFF 검토 단위와의 대응

이 작업은 새 트랙이 아니라 HANDOFF 가 잡은 검토 단위 3번과 4번이다.

- 단위 3 (DSL 구성·테스트와 `.dsl` 이행) — D5, P1, P2
- 단위 4 (기존 RPC 테스트 적격성) — P4, P5, R1, R2, R3
- 단위 1·2 (로컬 CI, 격리망 다중 서버) — V1, V2, R2, R3
- 단위 5 (MCP 노출) — R6

---

## 12. 검증 기준

항목 하나를 끝낼 때마다 아래를 확인한다.

- `gofmt -l cmd internal tests` 가 비어 있다.
- `go vet ./...` 이 통과한다.
- 해당 항목이 건드린 패키지의 테스트가 통과한다.
- 케이스 파일을 건드린 항목은 기준선 `20260913T101856Z` 와 대조한다.

케이스를 건드릴 때 지키는 두 가지가 있다.

**Stablenet 의 FAIL 은 FAIL 로 남아야 한다.** 치환 때문에 PASS 나 SKIP 으로
바뀌면 그것은 진전이 아니라 신호 손실이다. 그 묶음을 되돌린다.

**SKIP 증가분은 사유별로 설명한다.** 기준선의 SKIP 은 27건이다. P4 로 능력 기반
적격성을 켜면 늘어날 수 있다. 늘어난 것마다 어느 능력이 없어서인지 적는다.
SKIP 은 PASS 가 아니다.

### 12.1 완료 조건 대비 현재 (2026-09-15, 205건 기준)

| 완료 조건 | 처음 | 지금 |
|---|---|---|
| (1) 메인넷별 설정을 따로 관리 | 별도 파일 참조 0 | **205** (선언 17개) |
| (2) 공통 코드에 메인넷 값 없음 | 체인 이름 게이트 143 · 시스템 컨트랙트 주소 197 | **게이트 66 · 주소 0** |
| (3) 실행 시점 주입 | 없음 | **`run --env <id\|경로>`** |
| (4) 설정만 추가하면 재사용 | 0 | **완료 (2026-09-18)** — `applicableChains` 66 → **0** |

**(1)~(4) 가 다 섰다.** 남은 것은 조건이 아니라 품질이다 — §11.3·§11.4 참조.

### 12.2 이 트랙이 세운 오프라인 검사

케이스를 건드리는 작업은 아래가 지킨다. 넷 다 변이를 심어 실효를 확인했다.

| 검사 | 무엇을 막나 |
|---|---|
| `run --plan` 전수 비교 | 선언을 쪼개거나 옮길 때 **망이 달라지는 것** |
| 주소 골든 (`corpus-addresses.golden`, 571줄) | 이름으로 바꿀 때 **다른 주소를 가리키는 것** |
| X8 가드 | 체인 고유 주소를 쓰면서 **게이트를 안 다는 것**, 그리고 쓴 주소와 요구한 컨트랙트가 **어긋나는 것** |
| H4 래칫 | 이름이 있는 컨트랙트를 **다시 주소로 쓰는 것** |
| `validate --chain` 판정표 | 게이트를 옮길 때 **좁아지는 것** |
| 주소 인자 소스 검사 | 액션이 주소 인자를 **해석기에 안 넘기는 것** (키 단위) |
| **`VerifyLaunched`** | **계획과 다른 망에서 테스트가 도는 것** — 이것만 "실제로 그렇게 떴나" 를 본다 |

**넓어지는 것을 잡는 오프라인 검사는 없다.** X8 가드가 일부를 잡지만 주소를 쓰는
케이스에 한한다. 그래서 P5-L1 이 라이브 항목이다.

### 12.3 이 검사들이 못 본 것 (2026-09-16)

X5 를 라이브로 돌리자 **이 트랙이 만든 회귀 둘이 나왔다.** 위 다섯 검사가 전부
통과하는 상태에서였다.

| | 무엇 | 왜 안 보였나 |
|---|---|---|
| X10 | e2e 하니스가 은퇴한 `--validators` 를 부른다 | `//go:build e2e` 라 `go test ./...` 이 **컴파일도 안 한다**. `go vet -tags e2e` 는 컴파일은 하지만 문자열 인자가 플래그 이름인지 모른다. **2026-09-18: 이 영역에 검사를 붙였다** — `cmd/chainbench/cliflags_test.go` |
| X12 | 이름 지은 컨트랙트가 액션의 `to` 에서 안 풀린다 | 주소 골든이 `resolveAddressArgs` 를 쓰는데 **액션은 다른 함수를 쓴다.** 길이 둘인데 하나만 검사했다 |

**둘 다 "검사를 더 촘촘히" 가 아니라 "검사가 닿지 않는 영역" 의 문제였다.**

X12 에는 소스 수준 검사를 붙여 그 영역을 덮었다(주소 인자를 꺼내면서 해석기에 안
넘기는 함수를 키 단위로 찾는다).

**X10 이 속한 영역도 2026-09-18 에 덮었다** — `cmd/chainbench/cliflags_test.go`.
결함이 글자 수준이라 소스에서 읽는다. **파싱은 빌드 태그를 안 보므로**, 평소
`go test ./...` 이 컴파일조차 하지 않는 파일도 여기서는 다른 파일과 똑같이 읽힌다.

판정 단위는 저장소가 아니라 **명령**이다. 문자열 리터럴이 명령 경로로 시작하면 그
명령을 부른 것으로 보고, 거기 적힌 긴 플래그가 그 명령이 받는 것인지 본다. 명령
경로로 시작하지 않으면 이 CLI 것이 아니다 — 이 저장소가 만드는 노드 argv 는
`--datadir`·`--http.port` 로 가득하다.

**긴 플래그가 하나라도 있어야 판정한다.** 그게 "이건 argv 다" 라는 증거다. 없이
판정했더니 `[]string{"validator", "endpoint"}`(역할 목록)까지 명령 호출로 읽혀
거짓 보고가 19건 났다.

셸 스크립트도 본다. 스크립트도 이 CLI 를 부르는데 **아무도 컴파일하지 않으므로**,
은퇴한 이름이 가장 오래 남는 자리다.

**붙이자마자 죽은 것 둘이 나왔다.**

| 어디 | 무엇 | 처리 |
|---|---|---|
| `scripts/chain-setup/handoff-wemix-wbft.sh` | `chain up --case/--profile/--from-binary/--to-binary/--template/--data-dir/--stop-after` 와 `chain down`. 일곱 중 남아 있는 것은 `--to-binary` 하나이고 그나마 `hardfork` 에 있다. **돌아갈 수 없는 상태였고 아무도 몰랐다** | 148줄이 존재하던 이유("`chain up --case wemix-wbft` 가 아직 예전 단일 단계 순서라 실패한다")가 이미 사라졌다 — 핸드오프는 구성기에 흡수됐고 DSL 케이스로 있다. 오늘 도는 명령 하나로 줄였다. 문서 둘이 이 경로를 가리키므로 파일은 남긴다 |
| `setup.sh` 의 "Next steps" | `chainbench setup --chain …`(그런 명령 없음), `verify --data-dir`(그런 플래그 없음) | 실제 명령으로 고쳤다 |

**변이로 실효를 확인했다.** `tests/e2e/harness_test.go` 를 `--validators`/`--endpoints`
로 되돌리자 파일·줄·플래그 이름을 집어 실패했고, 되돌리니 통과했다. 셸 쪽도 같은
방법으로 확인했다.

**못 잡는 것**: 조각을 이어 붙여 만든 플래그, 명령 낱말이 다른 곳에 있는 슬라이스에
`append` 하는 경우, 문서 안의 명령. 그물이지 증명이 아니다.

---

## 13. 아직 확인하지 않은 것

**읽어서 닫을 수 있는 것** (체인 저장소가 이 기계에 있다):

- go-wbft 저장소가 실제로 `gwbft` 를 만들어 내는지 (X4 의 남은 절반). 지금은
  `make` 결과가 `gwemix` 라는 것만 실측돼 있다
- go-wbft 가 정말 두 합의 알고리즘을 지원하는지. chainbench 매니페스트에는 `wbft`
  하나만 적혀 있다
- 깊은 병합으로 바꿨을 때 42개 모양이 몇 개 preset 으로 줄어드는지

**띄워야 닫히는 것**: §11.4 의 L1·L2.

**그리고 이 문서의 판단 상당수는 소스를 읽은 것이다.** 체인을 띄워 확인한 것은 각
항목이 "실측" 이라고 적은 곳뿐이다.

---

## 14. 변경 이력

| 날짜 | 내용 |
|---|---|
| 2026-09-20 | **L1·L2 완료 (§11.4.1·§11.4.2).** 209건을 도커 15서버에 올려 돌렸고 실패 48건이 **전량 설명된다**(원인 불명 0). 찾은 **회귀는 하나** — 비교값의 이름 해석이고, §11.4 의 위험 표가 예측해 둔 바로 그것이다. 기존 결함 둘을 함께 고쳤다: 노드 간 읽기 경합 10건(전수 123지점 중 실측된 것만), ws 다이얼이 도커 localmap 을 건너뛰던 것. **다만 이 실행은 2026-08-10 빌드로 돌았다** — HEAD 바이너리는 fleet 에 올라가지 않았고, 그래서 "현재 체인에서 통과한다" 는 아직 말할 수 없다 |
| 2026-09-18 | **오프라인 잔여가 0 이 됐다.** H3-b(비교값·원시 인자의 계정 라벨 14곳 → 리터럴 0곳), X10 영역의 검사(`cliflags_test.go` — 붙이자마자 죽은 호출 둘을 잡았다), N1(`env.attach`). 앞서 같은 날 P5-L1 이 66 → 0 으로 끝나 **완료 조건 네 개가 다 섰다**. 스킵 6·실패 2·블록 1 도 전부 통과로 돌렸고, 그 과정에서 DSL 어휘를 다섯 군데 넓혔다(`genesis.provides`·`genesis.haltsAt`·`abiCall` 의 bytes·`derive` 의 hex·`uintArg` 의 hex). **남은 것은 전부 라이브다** — §11 머리말 참조 |
| 2026-09-14 | 최초 작성. PR #419 `HANDOFF.md` 와 코드 조사 결과를 종합해 항목과 우선순위를 정리 |
| 2026-09-15 | D1·D5 재확정(`chain-preset`). 네이밍 묶음 N 신설. M4 를 출처 기록에서 "최종 설정 기록 + 실행 대조" 로 다시 잡음. `WonBy` 제거 — 2년간 아무도 읽지 않았고 주석은 없는 화면을 약속했다 |
| 2026-09-17 | **하니스 안정성 S1·S2·S4·S5 완료.** 실패 자료가 어디서 실패하든 남고(S1), 기록이 어느 단계에서 죽었는지 말하고(S2), 정지가 모든 노드를 동시에 끝까지 시도하고(S4), 로그가 양끝을 남긴다(S5). **오늘 겪은 세 가지 — 증거 없는 실패, 절반만 멈추는 정리, 잘린 로그 — 를 직접 겨냥했다.** S3(노드 간헐 실패)는 재현되지 않아 재발 대기로 남긴다. 다음에 나면 S1 이 증거를 남긴다 |
| 2026-09-17 | **205건 라이브 검증 완료 — 190건 통과.** 그 과정에서 찾은 결함 셋을 고쳤다(재구성이 데이터 폴더를 안 비움, 선언한 바이너리 이름표가 개수 형식에서 사라짐, 세우다 실패하면 정리를 건너뜀). 케이스 6건도 고쳤다. **그러나 더 큰 것이 드러났다 — 실패가 테스트 이전에 나면 아무 기록도 안 남는다.** 하니스 안정성을 §11.2 로 신설해 최우선에 두고, 단계 생애 주기 설계를 `stage-lifecycle-design.md` 에 썼다(AST 로 함수 307개 파싱해 측정) |
| 2026-09-17 | **§11.2.2 신설 — 내가 모르는 것을 적어 둔다.** 거버넌스 계약과 체인 설정의 층을 구분하지 못해 반복해서 틀렸다. 멤버 추가와 검증자 등록을 같은 것으로 봤고, `short-expiry`·`account-extra` 를 genesis 문제로 읽었다. 사용자가 각각 지적했다. **증상만 쫓아 고치면 또 틀리므로 배우고 나서 건드린다** |
| 2026-09-16 | **M4 를 세 번째로 다시 열었다.** M4-b 는 "기록한다" 인데 화면에만 찍고 있었다 — 계획을 `compose-plan.json` 으로 워크스페이스에 남긴다. M4-c 는 "대조한다" 인데 바이너리와 배치를 빼고 있었다. **두 항목 다 문장은 처음부터 그렇게 적혀 있었고, 내가 덜 한 채로 완료로 적었다.** 닫기 전에 항목의 문장을 글자 그대로 충족했는지 본다 |
| 2026-09-16 | **M4-b 를 일찍 닫은 것을 되돌렸다.** 항목은 "값마다" 인데 launch 노브만 하고 완료로 적었다. 출처가 둘 이상인 값 전부로 넓히고(`binary`·`target`·노드 수 셋·키 둘), 세 번째 출처 `harness` 를 세웠다 — 선언도 명령도 안 고른 값은 읽는 사람이 고칠 것이 없어서 가장 놀란다. 이 작업이 X15 를 드러냈다 |
| 2026-09-16 | M4-b 완료로 **M4 묶음이 끝났다**. launch 노브가 선언에서 왔는지 명령줄에서 왔는지 계획이 들고 다닌다. 9월 15일에 지운 `WonBy` 를 되살린 것이 아니다 — 그때의 기각 사유는 "읽는 자리가 없다" 였고 M4-c 가 그 자리를 만들었다. config 에는 안 붙였다(실행 경로의 출처가 하나뿐) |
| 2026-09-16 | 구성 기록의 중복 결함 하나. `recordLaunchSet`·`recordConfigSet` 이 무조건 append 라서 같은 선언을 한 워크스페이스에 두 번 구성하면 `launchSet: {bp: ["mine=true","mine=true"]}` 가 됐다. argv 조립은 마지막이 이기므로 뜨는 망은 같지만, **무엇을 요구받았는지 말하라고 만든 기록**이 실행 횟수만큼 길어졌다. 두 곳이 같은 규칙(키당 한 줄, 제자리 교체)을 쓰게 했다 |
| 2026-09-14 | M1·M2·M3·M5·M6 완료. `extends` 가 깊은 병합이 되고 `null` 로 지울 수 있다. M4(출처 기록)만 남음 |
| 2026-09-14 | D1·D4·D5 확정. 공용 문서는 `env` 이고 preset 은 새 뜻을 갖지 않는다. 병합 규칙 확정 |
| 2026-09-14 | V11 완료. 체인 파일이 네 축을 한 리터럴로 조립. 1순위 어휘·주인 묶음 종료 |
| 2026-09-14 | V7 완료. 방언을 매니페스트가 선언하고 `DialectFor` 의 체인이름 분기를 없앰 |
| 2026-09-14 | V6 완료. family 를 등록제로 바꾸고, family 이름을 비교하던 BLS 분기를 family 에게 물음 |
| 2026-09-14 | V10 완료. family 가 타입 있는 `LaunchPolicy` 를 내놓고, RPC 플래그 둘은 방언이 정한다. wemix 가 `--rpc.allow-unprotected-txs` 를 받게 된 것이 유일한 동작 변화 |
| 2026-09-14 | D3 해소. family/방언/계정·암호/체인상수 네 축을 확인하고 V6·V7 을 V10·V6·V7·V11 로 재구성 |
| 2026-09-14 | V2 완료. 정의서의 바이너리 경로를 파싱 단계에서 거부하고 5건을 워크스페이스 설정으로 옮김. X2 해소 |
| 2026-09-14 | V1 완료. 바이너리 해석을 한 곳으로 모으고 정의서 192건의 중복 선언을 지움. X1 해소 |
| 2026-09-14 | V8 완료. 15노드 케이스 5건을 bp 7 · en 7 · pn 1 로 옮기고 정족수 케이스를 다시 씀. X6 해소 |
| 2026-09-14 | V9-b 완료. 노드 개수 어휘를 `bp`/`en`/`pn` 으로 통일하고 핸드오프 바이너리 키를 `from`/`to` 로 바꿈 |
| 2026-09-14 | V9-a 완료. `terminology-map.md` 작성. V9 를 조사(V9-a)와 변경(V9-b)으로 나눔 |
| 2026-09-14 | W5 완료. 구성 기록에 형식 버전을 넣고 옛 키 수용을 지움. V9-b 가 기록 형식을 바꾸기 전에 필요했다 |
| 2026-09-14 | V5 완료. 설정 스코프에 역할을 더하고, 기록되는 스코프를 검사하게 하고, 스키마 패턴을 어휘에 묶음 |
| 2026-09-14 | V4 완료. 스코프 규칙을 `node` 로 모으고 `pn` 을 쓸 수 있게 함. X3 해소 |
| 2026-09-14 | V3 완료. 옛 역할 철자 3개를 지우고 검증기가 노드 표의 역할을 보게 함. 테스트 32개 파일 수정 |
| 2026-09-14 | C1 완료. 주석 오류 80건을 고치고 `internal/arch` 에 회귀 테스트를 넣음 |
| 2026-09-14 | 하위호환 제거를 원칙으로 올림. 용어 표 추가(대상 워크스페이스 / 구성 기록 / 실행 세션 / 로컬 작업 공간). 주석 관련 진행 규칙 2건 추가. C 묶음과 W 묶음 신설 |
| 2026-09-14 | D2 확정. 표준 토폴로지(bp 7 · en 7 · pn 1)와 `bp`/`validator` 용어를 확정해 V8·V9 추가. 마무리 묶음 Z1~Z3 추가. X6 추가 |
| 2026-09-14 | 착수 전 조사 2건 완료. 옛 역할 철자 사용 0건 확인(V3 위험 제거), X4 부분 확인. 바이너리 철자 수치를 164/13 으로 정정 |
