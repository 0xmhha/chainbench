# 로컬 docker 를 원격 서버처럼 — 주소 변환 seam 설계

> **등급: [현행 설계]** — 분석·준비 단계다. 구현 전이며, 작업 상태는 worklist §1g R 트랙이
> 정본이다. 참조 구현: `~/Work/github/packages/wemix-bp-test` (동작 검증 완료된 선례).

실 원격 서버에 지금 연결할 수 없으므로, Rancher Desktop 의 docker 로 ubuntu 가상
서버들을 만들어 **원격 코드 경로를 로컬에서 검증**한다. 상위 레이어는 서버 주소를
그대로 쓰고, 접속하는 최하위 지점에서만 주소를 loopback 의 퍼블리시 포트로 바꾼다.

## 1. 선행 확인 — keyring CLI 라이브 (2026-08-24, main adfdddc)

원격 검증에 앞서 로컬 저장 경로들을 실검증했다. 전부 통과.

| 시나리오 | 결과 |
|---|---|
| `keyring new --count 2` (지정 없음) | `./keys/default` 에 생성. metadata.json + node별 nodekey·address·pubkey·keystore + password |
| `keyring new --keyring <경로> --with-bls --validators 2` | 지정한 data root 에 생성. BLS 파생, 3명 중 2명만 검증자 |
| `keyring add --count 1` | 추가 신원이 **검증자로 승격되지 않음** (과거 회귀 지점 재확인) |
| `keyring import --name x --from <파일>` | 파일 키를 라벨로 흡수, 주소·공개키 파생 |
| `CHAINBENCH_KEYRING=<경로>` | env 로 링 지정, 출처가 `(CHAINBENCH_KEYRING)` 으로 보고됨 |

남은 것은 `--from srv://<서버>/path` 같은 **원격 경로들**이고, 그것이 이 문서의 대상이다.

## 2. 참조 분석 — wemix-bp-test 가 이미 증명한 것

그 저장소의 제1 설계 원칙: **"로컬에서 작성한 코드가 운영 서버에서 수정 없이 돈다.
환경이 바뀔 때 갈아 끼우는 것은 서버 세트 파일 하나다."** 구조는 세 조각이다.

1. **서버 세트는 실주소를 유지한다.** 로컬 서버 세트도 원격과 같은 모양이다
   (컨테이너 bridge 주소 172.28.0.x, 포트는 전 노드 동일). 그래서 로컬에서 검증한
   파일 형식이 실서버 파일 형식이다.
2. **`LocalMap` 이 접속 직전에 주소를 바꾼다** (`internal/config/localmap.go`).
   macOS 는 Rancher VM 경계를 넘어 컨테이너 주소로 라우팅되지 않으므로,
   `172.28.0.11:10022 → 127.0.0.1:10101` 처럼 loopback 의 퍼블리시 포트로 치환한다.
   변환 파일(`localmap.yaml`)은 **로컬 환경에만 존재**하고, 없으면 아무 일도 없다
   (원격에서는 파일이 없어서 no-op). 무엇을 바꿨는지 **적용 내역을 반환해 보고**한다
   — "서버 세트에 적힌 곳이 아닌 데로 조용히 접속하는 하네스는 디버깅할 수 없다."
3. **환경 생성은 스크립트가 한다** (`env/local/scripts/gen-env.sh`). ubuntu 컨테이너에
   sshd(키 인증, 포트 10022)를 넣고, 노드별 퍼블리시 포트 규칙(bp1→10101, …)으로
   compose 파일·호스트 목록·localmap.yaml 을 함께 생성한다. 손으로 쓰는 파일이 없다.

**핵심 통찰**: 매핑은 **하네스가 스스로 접속하는 경계에서만** 적용한다. 노드끼리
주고받는 산출물(genesis 의 주소, static-nodes 의 enode, config)은 실주소를 유지해야
한다 — 컨테이너들은 bridge 네트워크 안에서 서로 실주소로 통신하기 때문이다.

## 3. chainbench 에 맞춘 설계

### 3.1 이미 맞는 것

- **serverset 서버 세트 v2** — pool 의 hosts 가 실주소를 담고 ssh 설정은 파일 전역
  하나다. "로컬 파일 = 원격 파일 모양" 원칙과 이미 같은 형태다. 바꿀 것 없음.
- **접속 경계가 이미 좁다** (실측). 주소를 dial 로 바꾸는 지점은 넷뿐이다:
  `remote.dialSSH`(`core/remote/ssh.go:94` 의 JoinHostPort), RPC 조립 3곳
  (`engine/launcher.go:255` · `netcompose/workspace.go:199` · `steps_lifecycle.go:271`).

### 3.2 새로 넣는 것 — dial 시점 주입, in-place 치환이 아니라

wemix-bp-test 는 서버 세트 로드 직후 값을 제자리에서 바꾼다. chainbench 는 그렇게
하면 안 된다 — **조립 산출물(워크스페이스·genesis·static-nodes)에 주소가 영속**되므로,
로드 시점 치환은 매핑된 loopback 주소를 산출물에 심는다. 따라서 seam 은 함수 주입이다:

```go
// AddrMap 은 하네스가 접속할 때만 주소를 바꾼다. 기본은 항등.
type AddrMap func(host string, port int) (string, int)
```

- `app.Deps` 에 실리고, 위의 접속 경계 4곳이 dial 직전에 통과시킨다.
- 매핑 데이터는 파일(가칭 `localmap.yaml`, `server-set.yaml` 옆,
  **gitignore**)이 담는다. 호스트별 대응표는 불리언 옵션에 담기지 않기 때문이다.

### 3.2a 활성화는 명시 옵션 `--docker` 로 (결정 2026-08-24)

초안은 "파일이 있으면 적용"이었다. 그 방식은 파일이 남아 있는 것을 잊으면 **진짜
원격을 시험한다고 믿으면서 docker 로 접속**하는 조용한 오판을 허용한다. 그래서
매핑의 전원은 파일 존재가 아니라 명령 옵션이 쥔다 — 명령줄 자체가 이 실행이 가상
원격이었음을 기록한다.

| 상황 | 동작 |
|---|---|
| `--docker` + 매핑 파일 있음 | 적용하고, 바꾼 내역을 출력 |
| `--docker` + 매핑 파일 없음 | 명확한 오류 (준비가 안 된 것을 즉시 알린다) |
| 옵션 없음 + 파일 있음 | 무시 — 진짜 원격 모드. 파일이 굴러다녀도 안전 |
| 옵션 없음 + 파일 없음 | 평소 그대로 |

단발 명령(`keyring import --from srv://…`)은 옵션이면 충분하다. **`net` 은
워크스페이스에 모드를 기록한다** — 스텝마다 옵션을 다시 받게 하면 한 스텝은
매핑되고 다음 스텝은 안 되는 반쪽 실행이 가능해지므로, `net new --docker` 가
target 을 기록하는 것과 같은 자리에 모드를 기록하고 이후 스텝은 기록을 따른다.
MCP 도구도 같은 옵션을 받는다(K8 선례: 두 표면은 같은 유스케이스를 바인딩만 한다).

참고: 옵션 이름 `--docker` 는 "내 서버들이 docker 컨테이너다"라는 운영자의 그림
그대로다. chainbench 가 docker API 를 만지는 것은 아니므로(주소 치환만 한다),
도움말에 그 경계를 적는다.

### 3.3 함정 (참조 구현과 우리 코드 대조에서 나온 것)

| # | 함정 | 대응 |
|---|---|---|
| P1 | **loopback 판정 오염** — serverset 은 주소가 loopback 인지로 local/remote 를 가른다. 매핑 *후* 주소로 판정하면 원격 경로가 로컬로 오판된다 | 판정은 항상 **매핑 전 실주소**로. AddrMap 은 판정 이후, dial 직전에만 |
| P2 | **산출물 오염** — 워크스페이스·genesis·enode 에 127.0.0.1 이 영속되면 컨테이너끼리 통신이 깨진다 | 3.2 의 주입 방식 자체가 방어. 회귀 테스트: 매핑 켠 채 조립한 워크스페이스에 loopback 이 없을 것 |
| P3 | **ssh 포트가 노드마다 다름** — 퍼블리시 포트는 호스트별인데 서버 세트 v2 의 ssh 포트는 전역 하나 | 그대로 둔다. 차이는 전부 localmap 이 흡수한다 (v2 파일은 원격 모양 유지) |
| P4 | 조용한 우회 | 적용 내역을 명령 출력에 남긴다 ("node2 172.28.0.12:22 → 127.0.0.1:2202") |

## 4. docker 환경 준비 (Rancher Desktop)

wemix-bp-test 의 `env/local` 패턴을 차용한다. ubuntu 이미지는 이미 있다.

- **컨테이너 = 가상 서버**: ubuntu + openssh-server(키 인증), bridge 네트워크에
  고정 주소. 체인 바이너리는 나중 단계에서 provision 이 올린다 (서버는 빈 서버).
- **퍼블리시 포트 규칙**: server1 → ssh 22→2201, server2 → 2202, …
  (노드 포트는 원격 검증 단계에서 같은 규칙으로 확장).
- **생성 스크립트 하나**가 compose 파일 + 서버 세트 v2(`server-set.yaml`
  실주소 기재) + `localmap.yaml` 을 함께 만든다. 손으로 쓰는 파일 0.

## 5. 작업 단위 (worklist §1g R 트랙)

| # | 작업 | 게이트 |
|---|---|---|
| R1 | AddrMap seam — `--docker` 옵션 + 매핑 파일 로드 + 접속 경계 4곳 주입 + 적용 보고 | 옵션 없으면 파일이 있어도 항등 · 옵션+파일 부재는 오류 · `net` 은 워크스페이스에 모드 영속 · P1/P2 회귀 테스트 |
| R2 | docker 가상 서버 생성 스크립트 + 서버 세트/맵 자동 생성 | 스크립트 한 번으로 N 대 기동, `ssh -p 2201` 접속 확인 |
| R3 | keyring 원격 경로 라이브 — `import --from srv://server1/...`, 지정 data root 설치 | docker 서버의 키를 가져오고, 링을 서버 경로에 설치 |
| R4 | `net up --target user@server/...` 원격 조립 라이브 | provision·기동이 docker 서버에서 수행, 산출물에 loopback 없음 |

## 6. 열린 질문

| # | 질문 | 기울기 |
|---|---|---|
| DR-a | 매핑 파일 이름과 위치 | 참조와 같은 `localmap.yaml`, 서버 세트 옆, gitignore. 활성화는 파일이 아니라 `--docker` 옵션(§3.2a) |
| DR-b | 인증 방식 | **해소 (2026-08-24) — 사용자 id + 비밀번호로 변경.** 초안은 키였으나 실서버 함대가 id+password 이고 sudo 가 그 비밀번호를 요구한다는 운영 사실이 확인되어, 함대도 같은 모양으로 재구성했다(키 로그인·root 로그인 없음, sudo 는 비밀번호 요구). 실행 메커니즘은 `driver.SSHSudoRunner`(sudo -S -k, 비밀번호는 stdin — 명령줄·프로세스 목록에 남지 않음). 라이브 증명: password 로그인 = chainbench, sudo whoami = root, root 전용 쓰기 성공(`serverset` 의 게이트된 `Live_Sudo` 테스트) |
| DR-c | AddrMap 을 serverset 이 소유하나, 별도 패키지인가 | 서버 세트와 함께 읽히므로 serverset 소유 기울기. 단 core/remote 가 serverset 을 모르는 층위라면 Deps 주입으로 전달 |
| DR-d | bridge 고정 주소 vs 컨테이너 이름 | 참조는 고정 주소(172.28.0.x). enode 에 IP 가 필요하므로 고정 주소 기울기 |
