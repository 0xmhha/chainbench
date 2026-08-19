# DSL v2 문법 제안 · x-bar 정렬 갭 분석

> **[현행 설계]** DSL v2 문법(T7.8 구현됨).
> 지금 향하는 목표. 근거는 정본([[chainbench-requirements-review]]·[[chainbench-feature-spec]])이고,
> 작업 순서는 [[chainbench-worklist]] §1g 다.

> 지시 1(DSL 문법 신규 제안) · 지시 4(x-bar 기반 문서 대조·갭) 응답. 작성: 2026-08-11 · 기준 커밋 `2181191`.
> 대조 대상: [`chainbench-design.md`](chainbench-design.md) §3.2·§4.3 · [`chainbench-feature-spec.md`](chainbench-feature-spec.md) ·
> [`chainbench-component-architecture.md`](chainbench-component-architecture.md) · 구현 정본 `internal/testspec/spec.go`.

---

## 0. 요약

현행 DSL(schemaVersion 1)은 **실행 어휘(액션 11 · 어세션 16)는 충분히 넓지만, 선언 어휘(환경 구성)는
배경 요구의 절반을 표현하지 못한다.** 구체적으로 키/키스토어 소싱(배경 1.4·1.5, 알고리즘 2·3),
genesis 4모드 중 3모드(design §3.8), 노드 실행 옵션(배경 2, 알고리즘 7), metric 검증 소스(배경 3),
hook 의 override 시맨틱(배경 4)이 **정의서로 도달 불가능**하다.

v2 는 문법을 넓히는 것이 아니라 **선언부(environment)와 시나리오부(case)를 분리**하고, 빠진
선언 축 5개를 채우며, 스텝/어세션의 **문법 형태를 통일**한다.

---

## 1. 현행 문법 실측 (v1)

`internal/testspec/spec.go:26-41` 이 정본이다.

```
Spec := schemaVersion id [applicableChains] [requires] chain [topology] [hardforks]
        [placement] [defaultOn] [preActions] [steps] assertions [postActions] [timeouts]
chain := name (binary | binaries) [config] [genesisOverlay]
step      := { "<actionName>": { ...args, ["save"] } }        # head = 객체 키
assertion := { "assert": "<name>", ...fields }                # head = "assert" 값
```

필수: `schemaVersion, id, chain.name, chain.binary|binaries, assertions` (`spec.go:63-87`).

---

## 2. x-bar 정렬 분석

### 2.1 방법

X-bar 도식을 다음과 같이 사상한다.

| X-bar | DSL/문서에서의 의미 |
|---|---|
| **X⁰ (head)** | 범주를 이름 짓는 환원 불가능한 핵 — 액션/어세션 이름, 문서의 주제 |
| **Complement** | head 가 하위범주화하는 **필수** 논항 — 없으면 head 가 성립 안 됨 |
| **Adjunct** | 선택적·반복 가능한 수식 — 없어도 문장은 성립 |
| **Specifier** | 투사를 닫고 **지시(reference)를 고정** — 어느 노드/어느 환경에 대한 진술인가 |

**갭의 정의**: ① head 가 요구하는 complement 가 문서·문법에 부재, ② adjunct 가 head 자리를 차지,
③ 같은 범주가 서로 다른 구조로 투사(비대칭).

### 2.2 문서 x-bar 표

| 문서 | Specifier (지시 고정) | X⁰ (head) | Complement (필수) | Adjunct |
|---|---|---|---|---|
| `chainbench-design.md` | "구현 전 확정 설계(HOW)" | 패키지 인터페이스 계약 | §3 인터페이스 · §4 데이터모델 | §6 동시성 · §8 마이그레이션 · §9 미결 |
| `chainbench-feature-spec.md` | F1–F16 번호 | 기능 동작 계약 | 입력/동작/출력/에러 + AC | 요구↔F 추적 부록 |
| `chainbench-component-architecture.md` | "#225 시점 실측" | 컴포넌트 계층·조립 순서 | §1b DDD · §2b 실측 · §3 카탈로그 | §0 비평 · §6 리스크 |
| `chainbench-worklist.md` | "진행 단일 정본" | 작업 상태 | T0–T6 태스크 · 상태표기 | 폴더 트리 예상도 |
| `chain-cli-execution-plan.md` | "기준 커밋 2424ccc" | 6-지시 → 순서화 | §4 원자 명령 표면 · §5 페이즈 | §2.4 드리프트 |
| `chain-setup/README.md` | 12단계 파이프라인 | bring-up 절차 | 단계·2페이즈 계약·결함8 | §4 CLI |

**구조적 문제 ①: `design` 의 head 는 "계약"인데, 정의서 문법의 complement 가 없다.**
§4.3 은 문법이 아니라 **jsonc 예시 한 덩어리**다. 정규 문법(EBNF/JSON Schema)이 없으므로
파서(`spec.go`)가 사실상 유일한 문법 정본이고, 문서-코드 드리프트는 구조적으로 보장된다.
실증: `requires` 필드는 코드(`spec.go:30`)에 존재하나 design §4.3 필수/옵션 목록에 없다
(T6.1 에서 추가되며 문서에 반영되지 않음).

**구조적 문제 ②: adjunct 가 SSoT 를 흐린다.**
`chainbench-audit-2026-08-09.md` 는 자기 §7 에서 "worklist 가 SSoT"라 자인하면서도 §1–§6 의
판정을 남겨 둔다 — head 없이 adjunct 만 남은 문서다. `chain-cli-execution-plan.md §2.4` 가 이미
드리프트로 지목했으나 배너 추가 이상의 조치가 없다.

### 2.3 DSL x-bar 표 — 비대칭 발견

| 구성소 | Specifier | X⁰ | Complement | Adjunct |
|---|---|---|---|---|
| step (v1) | (암묵: `defaultOn`) | 객체 **키** (`"sendTx"`) | 값 객체 내부 필드 | `save`도 **값 객체 내부** |
| assertion (v1) | `on` / `onEach` | `"assert"` 의 **값** | 형제 필드(`address`,`expected`) | `compare`, `at` |

**구조적 문제 ③(비대칭):** step 과 assertion 은 같은 범주(환경에 대한 술어)인데 **투사 구조가
다르다.** step 은 head-as-key, assertion 은 head-as-value. 결과로 (a) step 은 specifier(`on`)를
complement 와 같은 층에 넣을 수밖에 없고, (b) `save` 같은 adjunct 가 complement 와 구분되지 않으며,
(c) 파서·validator·문서가 두 벌씩 필요하다.

**구조적 문제 ④(specifier 분산):** 배치 target 의 specifier 가 `placement`(spec) · `remote.cluster`
(spec) · `--remote-host/--remote-user/--remote-port/--target-dir`(CLI, `cmd/chainbench/net.go:52-55`)
**3곳**에 흩어져 있다. key point 2 가 요구한 "local/remote 무관한 단일 경로 표현"과 불일치.

### 2.4 갭 목록 (배경/알고리즘 ↔ 문서 ↔ 코드)

| # | 요구 | 문서 | 코드 | 판정 |
|---|---|---|---|---|
| **G1** | 배경 1.4·1.5 / 알고리즘 2·3 — node key·keystore 를 **random 생성할지 기존 사용할지 결정** | design §3.5 `keyreg`(인터페이스만) | `keyreg.New` 는 **프로덕션 호출 지점 0**. `engine/attach.go:79`·`app.go:114` 가 `session.New(…, nil)` 로 nil 전달. 실경로는 `keys/preset` 하드코딩(`app.go:50`) | **미구현**. DSL 필드도 없음 |
| **G2** | 배경 1.2 — genesis 4모드 | design §3.8 (4모드 명시) | DSL 은 `chain.genesisOverlay` 1개만 노출 | **문법 갭 3/4** |
| **G3** | 배경 2 / 알고리즘 7 — 바이너리 sub-command·flag 로 http/ws/metric/chainId/networkId 설정 | **어느 문서에도 head 없음** (component-arch §2 책임귀속표에 행 자체가 없음) | 5곳 하드코딩 → [`chain-binary-flag-graph.md`](chain-binary-flag-graph.md) §2 | **문서·문법·코드 모두 부재** |
| **G4** | 배경 3 — 검증에 **log·rpc·metric** 활용 | design §3.6 collector 는 log·chainstate 만 | 어세션 16종 중 metric 소스 0 | **1/3 미구현** |
| **G5** | 배경 4 — pre/post hook 의 **override 동작** 정의 | design §3.2 는 액션 리스트로만 해석 | preActions/postActions = 액션 배열 | **시맨틱 부재** |
| **G6** | key point 2 — local/remote 를 단일 "경로"로 | design §7 은 `remote.cluster` 참조 | §2.3 문제 ④ 참조 | **표현 분산** |
| **G7** | 알고리즘 11–12 — 스텝과 검증의 **인터리브** | design §5 는 `runSteps → runAssertions` 순차 고정 | `tests/specs/README.md` 가 "이관불가 4건(순서…)" 로 기록 | **표현력 한계** (부분 해소: `save`/`$ref`) |

---

## 3. DSL v2 제안

### 3.1 원리 3가지

1. **선언과 시나리오를 분리한다.** 환경(`env`)은 fingerprint 재사용 단위이자 수명이 길고,
   케이스(`case`)는 짧고 많다. v1 은 둘을 한 파일에 섞어 환경 선언이 케이스 수만큼 복제된다.
2. **모든 술어를 하나의 구조로 투사한다.** `{"do"|"expect": <head>, …complement, …adjunct}`.
   비대칭(§2.3-③) 제거 + 인터리브(G7) 동시 해소.
3. **specifier 는 `on` 하나, 경로는 문자열 하나.** `local:/path` 와 `user@host:/path` 를 같은
   문법으로(G6).

### 3.2 파일 종류

```
specs/env/<id>.env.json      kind:"env"    환경 선언 (재사용 단위)
specs/case/<id>.case.json    kind:"case"   시나리오
specs/suite/<id>.suite.json  kind:"suite"  케이스 묶음 + 공통 hook   (선택)
```

`case` 는 `"env": "<env-id>"` 로 참조하거나 `"env": { … }` 로 인라인한다(단일 케이스 편의).

### 3.3 EnvSpec 문법

```jsonc
{
  "schemaVersion": "2",
  "kind": "env",
  "id": "wbft-7bp",

  // ── Specifier: 이 환경이 어디에 있는가 (G6 — 단일 경로 문법)
  "target": "local:.chainbench/work",
  //  "target": "deploy@10.0.0.11:/srv/chainbench",     // remote
  //  "target": { "path": "deploy@10.0.0.11:/srv/chainbench", "cluster": "remote-server-config.yaml" },
  //  cluster 파일은 gitignore 대상. spec 은 참조만 한다(design §7 L6b 유지).

  // ── Head: 어느 체인인가
  "chain": "wbft",

  // ── Complement: 이 환경이 성립하기 위한 필수 재료 4종 (배경 1.1~1.5)
  "binaries": { "default": "gwbft", "bp1": "gwemix" },     // 1.1 (handoff = 역할별 상이)

  "keys": {                                                 // 1.4 · 1.5 — G1 해소
    "nodekeys": { "source": "preset", "ref": "keys/preset" },
    //   source: preset | random | import | remote
    //   random 이면 keyreg 가 생성하고 BLSDeriver 로 BLS/PoP 를 채운다(design §3.5)
    "accounts": {
      "source": "preset", "ref": "keys/preset",
      "extra": [
        { "name": "acctA", "source": "random", "balance": "100ether" },
        { "name": "op1",   "source": "import", "ref": "keys/ops/op1.key" }
      ]
    }
  },

  "genesis": {                                              // 1.2 — G2 해소 (design §3.8 4모드)
    "mode": "template",          // existing | build | template | inherit
    "base": "keys/preset/genesis.json",     // existing/template/inherit 의 원본
    "set":  { "config.chainId": 8284 },     // 점경로 단일값
    "overlay": { "config": { "bohoBlock": 100 } }   // 깊은 병합
    // mode:"inherit" 는 base 를 상위 환경의 산출 genesis 로 해석(업그레이드 케이스)
  },

  "topology": { "bp": 7, "en": 5, "boot": 0,
                "sync": { "bp1": "archive", "default": "full" } },

  // ── Adjunct: 있으면 적용, 없으면 기본값
  "hardforks": { "croissant": 100, "brioche": 50 },
  "ports":     { "mode": "os" },            // os | stepped, [base.p2p], [base.rpc]
  "launch": {                                               // 배경 2 · 알고리즘 7 — G3 해소
    "all": { "http.api": "eth,net,web3,istanbul,admin,txpool",
             "verbosity": 3, "metrics": true, "networkid": 8284 },
    "bp":  { "mine": true },
    "bp1": { "metrics.port": 6060 }
    //  키는 launchopt.Key(체인 무관 이름). 대상 바이너리 dialect 가 미지원이면
    //  "요청된 기능의 부재"로 오류 — 조용한 스킵 금지.
    //  (→ chain-binary-flag-graph.md §3.3)
  },
  "config": { "eth.txpool.globalslots": 8192 },   // TOML 로 나가는 튜닝값(플래그와 겹치지 않음)
  "capabilities": ["rpc", "ws", "process", "metrics"]
}
```

**fingerprint 대상** = EnvSpec 전체(`id` 제외) + resolved config. v1 의 6요소에 `keys`·`launch`·
`ports` 가 추가된다 — 이 셋은 환경을 실제로 바꾸므로 재사용 판정에 반드시 포함해야 한다.

### 3.4 CaseSpec 문법

```jsonc
{
  "schemaVersion": "2",
  "kind": "case",
  "id": "GOV-005",
  "env": "wbft-7bp",
  "applicableChains": ["wbft", "stablenet"],
  "requires": ["rpc", "process"],

  "on": "bp:any",                                  // Specifier 기본값
  "timeouts": { "case": "10m", "receipt": "30s" },

  "hooks": {                                       // G5 해소 — override 시맨틱
    "pre":  [
      { "override": { "env.launch.bp1": { "verbosity": 5 } } },   // 환경 선언 덮어쓰기
      { "do": "ensureStaker", "name": "A" }                        // idempotent 가드
    ],
    "post": [ { "do": "unstake", "name": "A" } ],
    "onFail": [ { "do": "collectLogs", "tail": 500 } ]
  },

  // ── 단일 시퀀스: do 와 expect 를 자유롭게 인터리브 (G7 해소)
  "steps": [
    { "do": "sendTx", "on": "bp1", "signer": "op1",
      "call": "registerStaker(address,uint256)", "args": ["$acctA", "1"],
      "gas": "auto", "expect": "receipt", "save": "h1" },

    { "expect": "txStatus", "hash": "$h1", "is": "0x1" },

    { "do": "waitBlock", "n": 5 },

    { "expect": "rpc", "method": "istanbul_getValidators", "on": "bp1",
      "compare": "Len", "is": 7 },

    { "expect": "log", "on": "bp1", "match": "block reward",
      "within": "60s", "compare": "NotNil" },

    { "expect": "metric", "on": "bp1", "name": "chain/head/block",   // G4 해소
      "compare": "GreaterOrEqual", "is": 5 },

    { "do": "sendTx", "on": "bp1", "signer": "op2",
      "call": "unstake(uint256)", "args": ["1"], "expect": "revert" }
  ]
}
```

### 3.5 문법 규칙 (정규형)

```ebnf
CaseSpec   ::= schemaVersion kind id env [applicableChains] [requires]
               [on] [timeouts] [hooks] steps
Statement  ::= DoStmt | ExpectStmt | OverrideStmt
DoStmt     ::= '"do"' ':' ActionName  Complement*  Adjunct*
ExpectStmt ::= '"expect"' ':' SourceName  Complement*  Adjunct*
OverrideStmt ::= '"override"' ':' { DotPath : Value }

Adjunct    ::= '"on"' | '"onEach"' | '"save"' | '"timeout"' | '"within"' | '"compare"'
Complement ::= <액션/소스별 하위범주화 — 스키마가 정의>
```

- **`expect` 의 값은 "소스"**(`rpc`|`log`|`metric`|`call`|`txStatus`|`balanceAt`|…)다.
  배경 3 의 세 검증원(log·rpc api·metric)이 문법 최상위에서 대등하게 보인다.
- **`compare` 는 adjunct, `is` 는 complement.** `compare` 생략 시 `Equal`.
- **`$ref` 바인딩**(v1 §3.2b)은 그대로 유지 — 규칙 3개(`save` / `$name`·`${name}` / 미바인딩=오류).
- **미지 필드는 오류**(strict). v1 이 `map[string]any` 를 관통시켜 오타가 런타임까지 가던 것을 막는다.

### 3.6 v1 → v2 마이그레이션 (desugar 규칙)

v1 은 **v2 의 부분집합으로 기계 변환 가능**하다. 파서에 desugar 층을 두고 v1 파일을 그대로 받는다.

| v1 | v2 |
|---|---|
| `chain{name,binary,binaries,config,genesisOverlay}` + `topology`/`hardforks`/`placement` | 인라인 `env` 객체 |
| `{"sendTx": {…, "save":"x"}}` | `{"do":"sendTx", …, "save":"x"}` |
| `{"assert":"call", "expected":E, "compare":C}` | `{"expect":"call", "is":E, "compare":C}` |
| `preActions` / `postActions` | `hooks.pre` / `hooks.post` |
| `steps` + `assertions` | `steps` (steps 전부 → assertions 전부 순서로 이어붙임) |
| `defaultOn` | `on` |

변환기는 `chainbench migrate-spec <v1.json>` 로 제공하고, 기존 `examples/specs/*.json` 22건과
`tests/specs/**` 를 일괄 변환해 골든 비교(v1 실행 결과 == v2 실행 결과)로 검증한다.

### 3.7 문법 정본화 (구조적 문제 ① 해소)

- `internal/testspec/schema/v2.schema.json` 을 **문법 정본**으로 두고 `Parse` 가 이를 강제한다.
- design §4.3 의 jsonc 예시는 스키마를 **참조**만 하고 필드 목록을 중복 기재하지 않는다.
- CI 가드: `chainbench validate` 가 `examples/specs/**` 전량 + 스키마 self-check 를 돌린다
  (T6.7 의 가드를 스키마 기반으로 승격).

---

## 4. 우선순위 권고

| 순위 | 항목 | 이유 |
|---|---|---|
| 1 | **G1 keyreg 배선** (`keys` 선언 + `session.New(…, keyreg.New(…))`) | 알고리즘 2·3 이 통째로 미구현. `keyreg` 는 이미 구현·테스트되어 있고 **호출 지점만 없다** — 문서가 경고한 "테스트 있음 ≠ 배선됨"의 재발 |
| 2 | **G3 launch 옵션** | 배경 2·알고리즘 7 미충족. 설계는 [`chain-binary-flag-graph.md`](chain-binary-flag-graph.md) §3.3 |
| 3 | **§3.5 문법 통일 + 스키마 정본화** | 이후 모든 이관(106건 잔여)이 이 문법 위에 쌓임. 늦출수록 재작업 비용 증가 |
| 4 | G2 genesis 4모드 노출 | 코드(`core/genesis`)는 이미 4모드 지원 — DSL 필드만 열면 됨 |
| 5 | G4 metric 어세션 | collector 에 metrics 스크레이프 추가 필요(신규 작업) |
| 6 | G6 단일 경로 문법 | `netcompose.TargetSpec`(`internal/netcompose/target.go:29`)이 이미 유사 모델 — 통합 |
| 7 | G5 override hook | 위 항목들이 자리 잡은 뒤 |

**주의:** 3(문법 통일)을 1·2보다 뒤에 둔 것은 의도적이다. 문법을 먼저 바꾸면 아직 표현할 대상이
없는 필드를 설계하게 된다. 1·2 로 표현 대상을 실재화한 뒤 문법을 확정하는 편이 안전하다.
