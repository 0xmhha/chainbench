# Web 실행 경로를 CLI 어댑터로 바꾸는 결정

> 상태: **승인됨** (2026-10-12, main `b9e5be39` 기준). 6절의 네 가지를 사용자가 모두 권장안으로 정했다.

## 1. 왜 이 결정을 하는가

PR #443으로 들어온 Web 대시보드는 `internal/app` 유스케이스를 프로세스 안에서 직접 부르고,
그 위에 Web 전용 조율 코드를 따로 쌓았다. 같은 일을 하는 경로가 CLI와 Web 두 갈래가 되어서,
CLI 동작을 고치면 Web 쪽도 따로 고쳐야 한다. 누락 항목을 닫는 동안 나온 결함 대부분
(바이너리 지문 409, 여러 서버에 퍼진 네트워크의 노드 제어 409, fork로 늘어난 bp 때문에 정리가 거절된 문제)이
이 Web 전용 층에서 나왔다. 사용자는 PR을 병합한 뒤 Web이 CLI(또는 MCP)를 부르도록 바꾸기로 했다.

승인된 Seed의 문구("internal/app은 CLI/MCP 공통 usecase이며 Web은 같은 실행 의미를 호출한다")는
직접 호출로도 읽혀서 지금 구조를 막지 못했다. 이 결정은 그 문구를 하나의 뜻으로 고정하는 일도 포함한다.

## 2. 지금 무엇이 중복되어 있는가

규모(main `b9e5be39`, `wc -l`, 테스트 제외):

| 영역 | 줄 수 |
|---|---|
| `internal/app/web_*.go` (56개 파일) | 10,474 |
| 그 밖의 Web 전용 `internal/app` 파일(`deployment*.go`, `manifest_setup.go` 등) | 2,905 |
| `internal/core/session`의 Web 전용 저장소 | 1,245 |
| `internal/dashboard` | 2,351 |
| `web/src` (Svelte/JS) | 2,787 |

중복이 확인된 곳:

1. 체인 구성 순서를 Web이 두 번 다시 짠다(`web_chain_execution.go:216-331`, `manifest_setup.go:134-142`).
   CLI `chain up`은 `verb.ChainUp`이 작업공간 잠금을 쥔 채 같은 단계를 돈다.
2. 바이너리 업로드가 Web에 세 벌 있다(`web_chain_execution.go:295-319`, `web_test_run.go:393`,
   `web_node_binaries.go:286`). CLI에는 `app.Upload`가 있다.
3. Web은 `chain-record.json`을 14곳에서 직접 읽는다. CLI와 MCP는 `ChainStatus`, `ChainHealth`를 쓴다.
4. 노드 프로세스가 기록과 맞는지 확인하는 규칙이 둘이다. Web은 argv 전체를 비교하고
   (`web_node_probe.go:24`), 코어는 datadir과 config 경로를 읽는다(`chainsetup/observe_introspect.go:73-110`).
5. 로그 수집, SSH 자격증명 해석, 테스트 계획, 세션 이력도 양쪽에 따로 있다.

## 3. CLI와 MCP가 지금 줄 수 있는 것과 없는 것

둘 다 코드에서 확인했다.

**CLI**
- 명령 하나가 블로킹 프로세스 하나다. 첫 SIGINT/SIGTERM은 컨텍스트를 취소하고, 두 번째는 즉시 종료(130)한다.
- `--json`은 `run`, `validate`, `chain status/health/enode/show`, `resource pool/plan`에만 있다.
  `chain up`, 단계 명령, `node *`, `stop`, `status`, `report`, `log`, `clean`은 사람용 글만 낸다.
- 실시간 이벤트는 attach 모드 `run`과 `verify`만 대시보드 URL로 POST한다. compose 모드 `run`, `chain up`,
  `hardfork`는 진행 이벤트를 내지 않고, 이벤트에 작업 id가 없다.
- 종료 코드는 `run`만 1과 2를 구분하고 나머지는 실패면 모두 1이다.
- **없는 명령:** 노드 하나의 바이너리나 설정 교체(`node.swap`), 노드 리셋(`node.reset`).
  `ReadOnlyKeys`, `BinaryOverrides`, 선언형 attach의 `Nodes` 플래그도 없다.
- 자격증명은 server-set 파일(비밀은 0600 파일 경로) 또는 환경 변수로만 받는다. ssh-agent는 지원하지 않는다.

**MCP**
- stdio 루프 하나가 요청을 하나씩 처리한다. 긴 `chainbench_run`이 끝날 때까지 다른 호출이 모두 기다린다.
- `notifications/cancelled`를 받고 무시한다(`internal/mcp/server.go:75`). 진행 알림을 보내지 않는다.
- 모든 핸들러가 빈 `app.Deps{}`를 넘긴다. 호출마다 다른 자격증명을 줄 방법이 없다.
- 인자도 CLI보다 적다(`chain_up`에 blueprint, target, launch-opt 등이 없고 `clean`, 파일 전송 도구가 없다).

**공통 위험**
- 작업공간 잠금이 원자적이지 않고(읽은 뒤 `os.WriteFile`, `session/lock.go:87-117`),
  같은 PID면 항상 다시 잡힌다(`:99-100`). 프로세스 하나 안의 여러 작업은 서로 막지 못한다.
- 포트 점유 계산은 `~/.chainbench/<x>/chainsetup` 아래 작업공간만 본다.

## 4. 선택지

### A. CLI 하위 프로세스 (권장)

Web은 작업마다 전용 디렉터리를 만들고, 검토한 입력을 YAML과 DSL 파일로 써서 `chainbench`를 실행한다.
결과는 CLI가 이미 쓰는 `chain-record.json`, `process.json`, 세션 폴더와 `--json` 출력에서 읽는다.
Web에 남는 것은 로그인과 역할, 암호화 자격증명 보관, 문서와 개정 관리, 계획 검토 화면, 이력과 모니터링 화면이다.

- 장점: 실행 의미가 CLI 하나에만 있다. CLI에 추가하는 기능(노드 교체와 리셋, JSON 출력, 진행 이벤트)은
  CLI 사용자도 쓴다. 작업이 별도 프로세스라서 취소는 신호로 하고, 대시보드가 죽어도 실행 기록이 남는다.
- 단점:
  1. SSH 키를 메모리로 넘길 수 없다. 작업 동안 0700 작업 디렉터리 안 0600 파일로 복호화해 두고 끝나면 지워야 한다.
     "비밀은 암호화 저장소에만 둔다"는 규칙에 대한 예외라서 승인이 필요하다(attach의 계정 키는 이미 이렇게 한다).
  2. 지금 다이얼마다 권한 철회를 다시 확인하는 `BeforeDial` 보호가 사라진다. 철회하면 하위 프로세스를
     취소 신호로 멈추는 방식으로 바뀐다. 철회 직후 진행 중인 SSH 명령 하나는 끝까지 갈 수 있다.
  3. CLI 쪽 선행 작업이 크다(5절 P0).
  4. 작업마다 프로세스를 띄우는 비용이 생긴다. 실행 시간이 분 단위라서 영향은 작다고 본다(측정하지 않았다).

### B. MCP 클라이언트

- 장점: 응답이 구조화되어 있다.
- 단점: 서버를 동시 처리, 취소, 진행 알림, 호출별 자격증명을 받도록 고쳐야 한다. MCP 인자는 CLI보다 뒤처져 있어서
  A의 CLI 작업을 MCP에도 다시 해야 한다. 대시보드가 MCP 서버 프로세스 하나에 묶이면 그 프로세스가 죽을 때
  모든 작업이 같이 멈춘다. A보다 일이 많고 얻는 것이 적다.

### C. 프로세스 안 호출을 유지하고 중복만 줄이기

Web이 `ChainUp`, `app.Upload`, `ChainStatus` 같은 공용 유스케이스를 쓰도록 바꾸고 Web 전용 조율을 줄인다.

- 장점: 변경이 가장 작고, 메모리 자격증명과 `BeforeDial` 보호를 그대로 둔다. Seed 문구와도 맞는다.
- 단점: 진입점이 여전히 둘이다. 사용자가 이미 정한 방향(Web은 CLI 또는 MCP를 부른다)과 다르다.

## 5. 권장안(A)의 단계

각 단계는 실패하는 테스트를 먼저 확인하고, 관련 WEB 기준의 실제 바이너리 증명으로 회귀를 확인한 뒤 커밋한다.

- **P0 CLI 보강.** Web과 무관하게 CLI만으로 쓸모 있는 것부터 넣는다.
  1. `node swap`(설정·바이너리 교체)과 `node reset` 명령. `app.NodeSwap`, `app.NodeReset`은 이미 있다.
  2. `chain up`, `node *`, `stop`, `status`, `report`, `log`의 `--json`.
  3. stdout NDJSON 진행 이벤트(`--events ndjson`)와 이벤트의 작업 id.
  4. 원자적 잠금(O_EXCL 또는 flock)과 포트 점유 계산의 작업공간 범위 지정.
  5. `run`의 `--binary-override`, 선언형 attach 노드 입력. `ReadOnlyKeys`는 필요한지 다시 따진다.
- **P0 결과와 순서 조정 (2026-10-12).** `node swap`, `node reset` 명령과 MCP 도구(`3cd2dd05`), 두 프로세스가
  같은 잠금을 함께 잡던 경쟁의 수정(`7ab443f7`)을 마쳤다. 나머지 P0 항목은 처음 필요해지는 단계로 옮긴다.
  그래야 각 추가가 그 단계의 실제 사용으로 RED와 GREEN을 얻는다. 진행 상황은 CLI가 이미 쓰는 `chain-record.json`의
  단계 기록과 테스트별 `status.json`을 읽어서 보이고, 이것으로 부족하다는 것이 P1 테스트에서 드러날 때만 NDJSON 이벤트를
  더한다. JSON 출력과 `run` 입력 보강은 P2와 P3에서, 자원 점유 기록은 P3에서, 프로세스 신원 확인은 P4에서 CLI 코어로 옮긴다.
- **P1 Web 실행기.** 작업 디렉터리, 자격증명 파일 수명, 하위 프로세스 실행, 취소, NDJSON 수신을 맡는 Web 쪽
  실행기 하나. 먼저 읽기 전용 관찰을 `chain status --json`, `chain health --json`으로 옮긴다.
- **P2 테스트 작업.** `test.run`, `test.attach`를 `chainbench run --json`으로 옮긴다.
- **P3 체인 구성.** `chain.setup`, `chain.deploy`, 매니페스트 setup을 `chain up`으로 옮긴다.
- **P4 노드 제어.** start, stop, restart, reset, swap을 P0 명령으로 옮긴다.
- **P5 정리와 인수.** 쓰이지 않게 된 Web 전용 조율 코드를 지우고, 14개 기준을 고정 소스로 다시 통과시킨다.

## 6. 사용자 결정 (2026-10-12)

1. A, B, C 중 무엇으로 갈지. **결정: A(CLI 하위 프로세스).**
2. A라면 작업 동안 SSH 키를 0600 임시 파일로 두는 예외를 허용할지. **결정: 허용한다.** 0700 작업 디렉터리 안에 두고 작업이 끝나면 지운다. 철회는 하위 프로세스를 취소 신호로 멈춘다.
3. Web에만 있는 안전 장치를 어디에 둘지. 자원 점유 기록과 프로세스 신원 확인을 CLI 코어로 옮기면 CLI 사용자도
   보호를 받지만 P0가 커진다. Web 어댑터에 남기면 중복이 일부 남는다. 권장은 CLI 코어로 옮기는 것이다.
   계획 검토 지문은 화면 개념이라서 Web에 남긴다. **결정: CLI 코어로 옮긴다.**
4. Seed 문구를 "Web은 chainbench CLI를 하위 프로세스로 실행해서 같은 실행 의미를 얻는다"로 고칠지. **결정: 고친다.** `chainbench-web-ui.seed.yaml`의 existing_patterns와 constraints 두 문장을 고쳤다.

## 7. 확인하지 않은 것

- `/dev/fd/N`을 `CHAINBENCH_REMOTE_KEY_FILE`로 넘겨 디스크에 키를 쓰지 않는 방법이 동작하는지.
- 잠금 경쟁과 같은 초에 시작한 두 실행의 세션 폴더 충돌이 실제로 일어나는지(코드로만 추론했다).
- 하위 프로세스 기동 비용.
- PR #443이 Web을 위해 바꾼 공용 코어(chainsetup, testengine, `remote/ssh.go`, `process/ledger.go`)의 규모.
