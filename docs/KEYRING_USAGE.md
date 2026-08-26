# keyring 사용 설명서

`chainbench keyring` 은 키 재료의 단일 명령 그룹이다. 노드 키(nodekey), 계정 키,
검증자용 BLS 재료는 전부 **같은 비밀 하나에서 파생**되므로, 용도별로 명령을 나누지
않고 한 그룹이 담당한다. 예전의 `keys` 그룹과 `account new/import/list` 는 이
그룹으로 흡수되어 제거됐다(§8).

문서 끝의 **수동 검증 체크리스트(§9)** 는 이 문서의 모든 동작을 순서대로 직접
확인할 수 있게 구성했다. 각 항목은 자동 테스트와 1:1 로 대응한다.

## 1. 개념 — 링은 디렉토리다

링(ring)은 노드 신원들의 디렉토리다. `keyring new` 가 만드는 것:

```
<ring>/
  metadata.json      # 인덱스: 라벨·주소·(BLS 공개키)·검증자 명단·genesis alloc
  password           # 키스토어 공용 비밀번호
  node1/
    nodekey          # devp2p 비밀키 (0600)
    address, pubkey
    keystore/UTC--…  # 암호화 키스토어
  node2/ …
```

- **파생값은 저장하지 않는다.** 주소·BLS·extraData 는 비밀키에서 매번 파생한다.
  `--verify` 가 그 일치를 검사한다.
- 링의 기본 위치는 **운영자 머신**이다. `--keyring-dir` 에 target 문법을 쓰면
  **서버 위에 링을 만들 수도 있다**: `srv://server1/data/chainbench/ring` 처럼
  지정하면 생성·조회·가져오기 전부 그 서버의 경로에서 일어난다(파일 seam 경유,
  docker 서버들에서는 `--docker` 와 함께). 네트워크 기동용 배송은 여전히
  `net provision` 의 몫이다.
- 체인 바이너리는 전혀 필요 없다. BLS 파생까지 인프로세스다.

## 2. 어느 링을 쓰는가

우선순위: `--keyring-dir <dir>` > 환경변수 `CHAINBENCH_KEYRING` > 기본값 `./keys/default`.
경로는 로컬 디렉토리 또는 target 문법(`srv://<서버>/경로`, `user@host:/경로`)이다 —
서버 경로면 `--server-set`(서버 세트)와, docker 서버들라면 `--docker` 를 함께 쓴다.
모든 명령이 첫 줄에 **어떤 링을 왜 골랐는지** 보고하므로 경로를 추측할 일이 없다.

```
$ bin/chainbench keyring list --keyring-dir /tmp/myring
keyring: /tmp/myring (--keyring-dir)
```

## 3. 명령

### new — 링 생성

```
bin/chainbench keyring new --count 3 [--json]                 # ./keys/default 에
bin/chainbench keyring new --keyring-dir /tmp/r --count 5 \
    --with-bls --validators 3                                 # 지정 경로, wbft 용
```

| 플래그 | 의미 |
|---|---|
| `--count N` | 신원 N 개 (node1..nodeN) |
| `--with-bls` | BLS 파생 (wbft 계열 필수, wemix 는 불필요) |
| `--validators M` | M 명만 검증자 명단에. **new 의 기본은 전원**, `0` 은 "검증자 없음 선언" |
| `--password` / `--balance` | 키스토어 비밀번호(기본 "1") / genesis alloc 잔액 |

이미 링이 있는 경로에는 **거부**한다(“already holds a ring; add to it instead”).

### add — 신원 추가

```
bin/chainbench keyring add --keyring-dir /tmp/r --count 2 --with-bls
```

기존 신원과 주소는 그대로 유지되고, **추가분은 검증자로 승격되지 않는다**
(add 의 `--validators` 기본은 0 — 신원 추가와 검증자 변경은 다른 결정이다).

### list / show — 조회

```
bin/chainbench keyring list --keyring-dir /tmp/r [--verify] [--json]
bin/chainbench keyring show --keyring-dir /tmp/r --name node1 [--json]
```

목록·조회에는 **비밀키가 절대 나오지 않는다**. `--verify` 는 저장된 주소·BLS 가
비밀키에서 실제로 파생되는지 전수 검사한다(변조 검출).

### export — 비밀키 출력

```
bin/chainbench keyring export --keyring-dir /tmp/r --name node1 --yes
```

`--yes` 없이는 거부한다 — 스크롤백에 비밀키가 우연히 남는 것을 막는 장치다.

### import — 이미 있는 키를 링으로

출처는 **정확히 하나**만 지정한다. 둘 이상이거나 0개면 거부한다.

| 출처 | 예 |
|---|---|
| 16진 비밀키 | `--private-key 0x…` |
| BIP-39 니모닉 | `--mnemonic "word × 12/24" [--passphrase w25] [--hd-coin-type 60] [--hd-account 0] [--hd-change 0] [--hd-index 0]` |
| 로컬 파일 | `--from /path/key` — raw hex 또는 키스토어 JSON(`--password` 필요) |
| 서버 세트 서버 | `--from srv://server1/data/…/nodekey` (`--server-set` 는 모든 동사 공통 플래그) |
| 직접 호스트 | `--from user@10.0.0.7:/path/key` 또는 `ssh://user@host:port/path` |

```
bin/chainbench keyring import --keyring-dir /tmp/r --name faucet --private-key 0x…
bin/chainbench keyring import --keyring-dir /tmp/r --name hd0 \
    --mnemonic "test test test test test test test test test test test junk"
```

- 같은 이름이 이미 있으면 **덮어쓰지 않고 거부**한다.
- 니모닉의 표준 파생(m/44'/60'/0'/0/0)은 개발용 니모닉 → `0xf39F…92266` 골든
  벡터로 고정되어 있다. `--hd-coin-type` 을 바꾸면 다른 키가 나온다(체인 등록
  코인타입 사용 시 필수).
- `--expect-address 0x…` 를 주면 가져온 키가 **정확히 그 주소를 파생하는지**
  먼저 확인하고, 다르면 아무것도 쓰지 않고 거부한다. 어떤 키여야 하는지 아는
  호출자가 전송 무결성을 증명시키는 방법이다.

### import --from-ring — 링 전체를 통째로

단건 대신 **링 전체**를 복제한다. 경로 문법은 `--keyring-dir` 와 같다
(로컬 경로, `srv://server1/…`, `user@host:/…`).

```
bin/chainbench keyring import --keyring-dir ./keys/pulled \
    --from-ring srv://server1/data/chainbench/ring \
    --server-set env/docker/build/server-set.yaml --docker
```

| 항목 | 동작 |
|---|---|
| 신원 | 라벨·순번 그대로 전부 복제 |
| validator 선언 | 원본의 선언(BLS 목록·alloc 포함)을 그대로 가져온다 — 단건 N회 복제와 달리 잃지 않는다 |
| 무결성 | 항목마다 키에서 주소·공개키·BLS 를 **재파생해 원본 인덱스와 대조**하고, 하나라도 다르면 전체를 거부한다 |
| 비밀번호 | 기본은 원본 링의 것을 유지, `--password` 로 재암호화 |
| 목적지 | 이미 링이 있으면 거부 (new 와 같은 규칙) |

## 4. 원격에서 가져오기 — 서버 세트가 접속을 결정한다

주소와 자격증명은 명령줄이 아니라 서버 세트 파일에 둔다. 서버 세트에 이름이
있는 서버는 **파일이 유일한 출처**다. 환경변수는 보지 않으므로, 다른 작업에서
export 해 둔 값이 접속을 조용히 바꾸는 일이 없다. 비밀번호를 파일에 직접 적기
싫으면 `ssh.password_file: <경로>` 로 한 줄짜리 0600 파일을 참조한다.

환경변수는 두 경우에만 쓰인다.

| 환경변수 | 의미 |
|---|---|
| `CHAINBENCH_REMOTE_USER` / `_PASS` / `_KEY_FILE` (+`_KEY_PASSPHRASE`) | **직접 표기(`user@host:/path`) 전용** SSH 자격 — 참조할 서버 세트가 없는 형태라서다 |

호스트키 정책은 환경변수가 아니라 **서버 세트의 `ssh:` 절**이 정한다:
`known_hosts_file: <경로>`(검증 파일 지정) 또는 폐쇄망 전용
`insecure_host_key: true`(정확히 하나만). 지정이 없으면 `~/.ssh/known_hosts`.

`srv://<이름>/경로` 는 서버 세트(`server-set.yaml`, 기본 위치 또는
`--server-set`)에서 이름을 찾는다. 모르는 이름·없는 서버 세트는 dial 전에
명확한 오류로 거부한다.

## 5. docker 가상 서버 모드 (`--docker`)

실서버 없이 원격 경로를 검증할 때 쓴다. 준비는 한 번:

```
cd env/docker
cp accounts.env.sample accounts.env    # 열어서 실제 비밀번호로 바꾼다
./gen-env.sh
docker build -t chainbench-server:ubuntu24 .
docker compose -f build/docker-compose.yml up -d
```

`--docker` 의 규칙 (옵션이 전원, `localmap.yaml` 이 데이터):

| 상황 | 동작 |
|---|---|
| `--docker` + localmap 있음 | 접속 직전에만 주소 치환, 내역 출력 (`docker: dialing 172.30.0.11:22 as 127.0.0.1:2201`) |
| `--docker` + localmap 없음 | 오류 (생성 방법 안내) |
| 옵션 없음 + 파일 있음 | 무시 — 진짜 원격 모드 오염 없음 |

서버들의 접근은 **실서버와 같은 모양**이다: `env/docker/accounts.env` 의 첫 계정
(기본 `devuser1`) + 공용 비밀번호, sudo 는 그 비밀번호를 요구한다. srv:// 경로는
자격을 생성된 서버 세트에서 읽고, 호스트키 정책(`insecure_host_key`)도 세트가 선언하므로 켤 환경변수가 없다.

```
bin/chainbench keyring import --keyring-dir /tmp/r --name srv1 \
    --from srv://server1/data/chainbench/live-test/rawkey \
    --server-set env/docker/build/server-set.yaml --docker
```

직접 표기(`user@host:path`)를 쓸 때만 비밀번호를 env 로 준다:
`CHAINBENCH_REMOTE_PASS=<accounts.env 의 비밀번호>`.

산출물(genesis·static-nodes·워크스페이스)에는 치환 주소가 절대 들어가지 않는다 —
노드끼리는 실주소로 통신한다.

## 6. validator 그룹 — 키를 합의의 눈으로 본다

키 재료는 keyring 소관이고, `validator` 는 그 키를 **체인 관점**으로 보여준다.

```
bin/chainbench validator new --chain stablenet [--json]     # 새 키 + BLS/PoP 파생 뷰
bin/chainbench validator import --chain wemix --private-key 0x…
bin/chainbench validator roster --chain stablenet --keys keys/preset
bin/chainbench validator set --out /tmp/preset --nodes 6 --validators 6
```

`validator set` 은 프리셋 묶음 생성기로 `keyring new` 와 산출물이 같은 계보다
(정리 후보로 기록되어 있다). `roster` 는 키셋을 패밀리 규칙으로 읽어 검증자
명단·BLS 유무를 보고한다.

## 7. MCP 도구

같은 유스케이스가 MCP 로도 노출된다: `chainbench_keyring_new/add/list/show/import`.
**export 도구는 의도적으로 없다** — 에이전트 대화록에 비밀키가 남으면 안 된다.

## 8. 제거·대체된 명령

| 예전 | 지금 |
|---|---|
| `keys new` / `keys import` | `keyring new` / `keyring import` (니모닉 포함) |
| `account new` / `account import` / `account list` | `keyring new` / `keyring import` / `keyring list` |
| `account fund` / `account state` | 그대로 (온체인 동작이라 account 에 남음) |

## 9. 수동 검증 체크리스트

각 항목 옆은 같은 것을 검증하는 자동 테스트다. 전체 자동 실행:
`go test ./cmd/chainbench/keyringcmd/ -v` (로컬),
`CHAINBENCH_DOCKER_FLEET=$PWD/env/docker/build go test ./cmd/chainbench/keyringcmd/ -run Live_Keyring -v` (원격).

**A. 로컬** (사전 준비 없음)

| # | 할 일 | 기대 결과 | 자동 테스트 |
|---|---|---|---|
| A1 | `keyring new --count 2` (지정 없음) | `keys/default` 생성, `keyring: keys/default (default)` 보고 | ReportsWhichRingItUsed |
| A2 | `keyring new --keyring-dir /tmp/r --count 3 --with-bls --validators 2` | 3신원·2검증자, BLS yes | NewCreatesAUsableRing |
| A3 | 같은 경로에 다시 `new` | 거부 (add 안내) | (core) Generate 거부 |
| A4 | `add --count 1 --with-bls` | 4신원·**여전히 2검증자**, 기존 주소 불변 | AddDoesNotPromote / AddKeeps |
| A5 | `list --verify` | 통과. metadata.json 한 글자 변조 후 재실행 → 실패 | VerifyCatchesDrift |
| A6 | `export --name node1` → 거부, `--yes` → 키 출력 | `--yes` 게이트 동작 | ExportRequiresConfirmation |
| A7 | `import --private-key <A6의 키> --name copy` / 같은 이름 재실행 | 성공 / 덮어쓰기 거부 | ImportRefusesToOverwrite |
| A8 | `import --mnemonic "test … junk"` | 주소 `0xf39F…92266` | MnemonicGolden |
| A9 | `import --mnemonic … --private-key …` 동시 | 거부 ("exactly one") | RefusesMixedSources / ExactlyOneOrigin |
| A10 | `CHAINBENCH_KEYRING=/tmp/r keyring list` | env 출처 보고 | ReportsWhichRingItUsed |

**B. 원격 (docker 서버들)** — §5 의 준비만 하면 된다(환경변수 불필요)

| # | 할 일 | 기대 결과 | 자동 테스트 |
|---|---|---|---|
| B1 | 로컬 링에서 키 export → 그 hex 를 server1 에 파일로 두기 | (준비) | Live 스위트가 자동 수행 |
| B2 | `import --from srv://server1/… --docker` | 치환 보고 + **로컬과 같은 주소** | Live_…RawKeyFromAServer |
| B3 | 키스토어 파일을 server1 에 두고 `--from srv://… --password 1` | 같은 주소 복원 | Live_…KeystoreFromAServer |
| B4 | B2 명령에서 `--docker` 만 제거 | 실주소(172.30.0.x)로 dial 하다 timeout | (설계 증명 — 수동) |
| B5 | `--docker` 인데 localmap 삭제 | 생성 방법을 안내하는 오류 | DockerNeedsTheLocalmap |
| B6 | `keyring new --keyring-dir srv://server1/<경로> --docker --count 2 --with-bls` → `list --verify` | 서버 위에 링 생성(치환 보고 출력), 원격 검증 통과. `--docker` 를 빼면 실주소 dial | Live_…CreatesARingOnAServer |
| B7 | `import --from-ring srv://server1/<경로> --docker` 로 서버 링을 로컬로 | 라벨·validator 선언 그대로, 항목별 재파생 대조 후 기록. 원본 인덱스를 1바이트 변조하면 전체 거부 | Live_…ClonesARingFromAServer, …Import_FromRing |

C. **net 과의 결합**(참고): `net up --keys <ring> --keys-source generate --server server1
--docker` 가 링 생성→배송→기동까지 잇는다. R4 라이브로 검증된 경로다.
