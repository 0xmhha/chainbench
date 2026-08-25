# 아키텍처 v2 — 모듈 재편 (레이어 · 책임 · 노출 규칙)

> **[현행 설계]** 모듈 재편의 목표 구조.
> 사용자 결정 2026-08-25. 근거는 정본([[chainbench-requirements-review]]·[[chainbench-feature-spec]])과
> 이 세션의 실측(서버 세트 nil 전달 결함 등)이고, 작업 순서는 [[chainbench-worklist]] §1h 다.
> [[layers]](layers.md)·[[target-architecture]](target-architecture.md) 의 레이어 규칙을 승계하며,
> 모듈 경계와 표면 경로를 재정의한다. 어긋나는 부분은 이 문서가 이긴다.

---

## 1. 전체 그림

```
CLI ──직접 호출──▶ core 모듈들            (각 기능의 동작 확인이 목적)

MCP ──▶ app (워크플로 층)
         └ DSL 파싱 → chainsetup → testengine → 수집 → 레포트

core
  chainsetup    체인을 구성해서 블록이 생성되는 상태까지 만든다 (netcompose 흡수)
                실행마다 기록 폴더를 남긴다 (§6)
  testengine    이미 구성된 체인 위에서, 테스트만 일관된 로직으로 수행 (기존 engine 재정의)
  keyring       키 역학 (생성·파생·검증) + store: 링 저장·읽기
  netmap        서버 정보 관리(서버 세트) · 자원 분배(ip·port 할당) ·
                enode 생성(공개키는 입력) · low level 접근 wrapper (유일 통로)

low level      (전부 파라미터 주입, 순수 기능)
  machine       ip + 경로 한 규칙으로 머신을 지정. local = loopback 머신 (현 core/target)
  process       머신별 실행 대장: 어떤 바이너리를 어떤 명령으로 띄워 pid 몇 번인지
  FileStore / driver / SSH   읽기·쓰기·실행·포트 점검의 원시 기능
```

의존은 아래로만 흐른다. low level 은 상위를 모르고, core 모듈끼리는 netmap 의
wrapper 를 통해서만 서버에 닿는다.

## 2. 표면 경로 — CLI 와 MCP 는 다르다

| 표면 | 경로 | 이유 |
|---|---|---|
| CLI | core 모듈 **직접 호출** | CLI 의 용도는 각 기능의 동작 확인이다. 층을 끼우면 확인 대상이 흐려진다 |
| MCP | **app 경유** | MCP 는 워크플로를 제공한다. DSL 파싱 → 체인 셋업 → 테스트 → 수집 → 레포트를 app 이 한 흐름으로 묶고, MCP 서버는 그것을 노출한다 |

app 은 흐름만 갖는다. "무엇을 순서대로 부를지"는 알고, "어떻게"는 core 가 안다.

## 3. 모듈 책임

| 모듈 | 소유 | 소유하지 않음 |
|---|---|---|
| `chainsetup` | 셋업 단계의 순차 진행, 사전 점검(기동 중 노드 검사), 실행 기록 폴더 | 단계의 내용(genesis·config 는 기능 모듈), 서버 접근 배선(netmap) |
| `testengine` | 구성된 체인 위에서의 테스트 수행, 일관된 실행 로직 | 체인 구성, 결과 종합(레포트는 app 흐름의 다른 모듈) |
| `keyring` | 키 생성·파생(주소·devp2p·BLS)·검증 | 저장(store 하위), 서버 접근(netmap) |
| `keyring/store` | 링 레이아웃·metadata·암호화 파일의 저장과 읽기, 링 위치 해석(플래그>env>기본) | 키 내용의 해석 |
| `netmap` | 서버 세트 관리, ip·port 할당, enode 조합(공개키는 입력), **접근 wrapper** | 키 파생, 체인 의미론, 프로세스 정책 |
| `machine` | 머신 지정(ip+경로 한 규칙)과 능력 손잡이 해석 | 서버 세트 내용(호출자가 lookup 주입) |
| `process` | 머신별 실행 대장(바이너리·명령·pid), 기동 전 점검 질의 | 실행 자체의 원시 기능(driver) |
| `driver`/`provision`/`remote` | 실행·읽기·쓰기·포트 점검의 원시 기능, 파라미터로 완결 | 어떤 머신인지 아는 것(입력으로 받는다) |

## 4. 접근 규칙 — netmap wrapper 가 유일 통로

서버 이름 → 접속의 결합(서버 세트 조회, `--docker` 치환, 호스트키 정책, 치환 보고)은
netmap 의 wrapper **한 곳**에서만 일어난다. 소비자는 능력 손잡이(FileStore·Driver)만
받는다. 자격증명·SSH 세부·서버 세트 내부 구조는 반환 타입에 없다.

근거는 실측이다. keyring 은 서버 세트를 넘기고 netcompose 는 nil 을 넘겨서, 같은
서버에 대해 한쪽은 붙고 한쪽은 "no SSH auth" 로 죽었다(2026-08-25 라이브에서 발견).
배선이 두 곳이면 반드시 갈라진다.

**무분기 규칙**: machine 의 소비자는 local/remote 로 분기하지 않는다. local 은
loopback 주소의 머신일 뿐이고, 구현 차이는 machine 내부의 관심사다.

## 5. 노출 규칙 — 경계는 소비자 측 작은 interface

- 소비자는 자기가 쓰는 메서드 1~3개만 담은 interface 를 **자기 쪽에** 선언한다.
- 제공자는 최소한의 구체 타입을 반환한다. 반환 타입에 이미 있는 작은 interface
  (FileStore·Driver)를 그대로 쓴다.
- 제공 함수 전부를 담은 큰 interface 는 만들지 않는다 — 제한이 아니라 전시장이 된다.
- 상위 레이어는 관심 있는 것만 보고, 나머지는 각 모듈 내부의 관심사로 남는다.

## 6. 실행 기록 폴더 (chainsetup 소유)

지정 폴더 아래 실행마다 폴더 하나. 내용: 체인 id, 입력 사본, 노드 배치표, genesis,
실행 명령. 디버깅의 출발점이 되는 "어떤 체인을 어떤 정보로 구성했는가"다.

**보안 경계**: 서버 세트의 `ssh:` 절(자격증명)은 기록에 **절대** 들어가지 않는다.
"pool 은 어디서 도는지를 말하고, 어떻게 로그인하는지는 말하지 않는다"를 기록
폴더에도 적용하며, 테스트로 고정한다(worklist V5.2).

## 7. 모듈 네이밍 규칙

새 모듈을 만들거나 개명할 때 이 표로 판정한다.

1. **소문자 한 덩어리** — 하이픈·언더스코어·대문자 금지. 폴더명 = 패키지명.
2. **소유하는 명사로 짓는다** — 동작이 역할이면 "대상+동작명사" 합성, 최대 2단어
   (`chainsetup`, `testengine`).
3. **덤핑 단어 금지** — util·common·helpers·shared·misc·base·manager. 관리자는
   패키지가 아니라 타입이다 (`process.Manager`).
4. **stutter 금지** — 패키지명이 타입명에 반복되지 않게 (`machine.Spec` ○ /
   `machine.Machine` ✗. 현 `target.Target` 이 위반 사례).
5. **단수형** — 집합 자체가 주제일 때만 집합 명사 (`serverset` ○, 현 `accounts` 위반).
6. **계층은 경로가 말한다** — 이름에 core·app 같은 계층 접두어 금지.
7. **약어는 업계 표준만** — rpc·ssh·mcp·dsl ○ / `netreg` ✗ (registry 인지 register 인지
   읽히지 않음).

접미사 관례: CLI 표면 패키지는 `<그룹명>cmd` (keyringcmd·netmapcmd·netcmd),
저장 하위 패키지는 `<도메인>/store` (keyring/store).

## 8. 이동 계획

함수 단위 이동표는 [[v2-move-map]](v2-move-map.md), 작업 순서와 상태는
[[chainbench-worklist]] §1h (V0~V7) 이 정본이다.
