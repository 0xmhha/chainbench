# 서버 세트 — 포트·호스트·접속 정보 (`server-set.yaml`)

> **[현행 설계]** 서버 세트.
> 지금 향하는 목표. 근거는 정본([[chainbench-requirements-review]]·[[chainbench-feature-spec]])이고,
> 작업 순서는 [[chainbench-worklist]] §1g 다.

> 노드가 **어디서** 뜨고 **어떤 포트로** 듣는지를 결정하는 단일 출처.
> 실제 파일은 **git 에 절대 올라가지 않는다**(gitignore). 추적되는 것은 템플릿
> `server-set.sample.yaml` 하나뿐이다.

---

## 1. 왜 별도 파일인가

호스트 주소와 포트 할당은 **사이트마다 다르고 민감정보**다. 코드 상수나 테스트 정의서에
들어가면 리포지토리에 박제되고, 사내 IP·포트가 공개 저장소로 새는 경로가 된다.

따라서 이 값들은 **데이터**이고, 그 데이터는 운영자의 로컬 파일에만 존재한다.

```
server-set.sample.yaml   추적됨 — 형식만 담은 템플릿
server-set.yaml          gitignore — 실제 값
```

`.gitignore` 는 `*server-set.yaml` / `.yml` 을 막고 `.sample.yaml` 만 예외로 둔다.

---

## 2. v2 — 파일의 주어는 서버 목록이 아니라 **자원 풀**이다 (2026-08-22)

v1 은 서버를 하나씩 나열하고 각 서버가 자기 포트·슬롯·접속을 가졌다. v2 는 **가용 자원**을
선언하고, 배치는 할당기가 정한다 — 운영자가 고르던 두 배치 모드가 실은 같은 격자였기 때문이다
([[netmap-design]] §2.2a).

```yaml
version: 2
pool:
  hosts:                       # 소비 순서. 문자열이면 주소가 곧 이름
    - { name: local, addr: 127.0.0.1 }
    # - { name: bp1, addr: 10.0.0.11 }
  slots: 8                     # 한 호스트가 담을 포트 슬롯 수(기본 1)
  ports:                       # 두 개의 겹치지 않는 대역(생략 시 내장 기본값)
    p2p: { base: 31000, step: 10 }
    rpc: { base: 8600,  step: 10 }
ssh:                           # 루프백이 아닌 호스트에 공통 적용
  port: 22
  sudo: false                  # 보관·전달만 — 소비는 기동 절차가 한다
dataRoot: /var/lib/chainbench  # 대상 위 데이터 플레인 루트
```

**호스트 먼저, 슬롯 나중**으로 소비한다. 5주소 × 4슬롯이면 node1~5 가 서로 다른 머신에
놓이고 node6 이 첫 주소로 돌아와 **다음 슬롯**을 받는다(포트가 겹치지 않는다). 한 호스트 ×
여러 슬롯은 이 머신 위의 네트워크, 여러 호스트 × 1슬롯은 한 서버당 한 노드다 — **같은 격자를 달리 읽은
것**이라 구성하는 쪽은 분기하지 않는다.

**local/remote 는 선언하지 않는다 — 주소가 정한다.** 루프백이면 이 머신, 아니면 SSH.
`kind` 필드는 옆에 있는 주소와 어긋날 수 있고, 그러면 파일이 두 가지를 말하게 된다.

| | 루프백 주소 | 그 외 주소 |
|---|---|---|
| 데이터 플레인 | 이 머신의 `dataRoot` | 그 호스트의 `dataRoot` |
| 접속 | 없음 | `ssh` (자격증명은 env 우선) |

> **v2 가 잃은 것**: 호스트별 개별 설정(호스트마다 다른 포트 대역·SSH 사용자·dataRoot).
> 풀은 균질하다. 이질적인 서버가 실제로 필요해지면 `hosts[]` 항목에 선택적 오버라이드를
> 얹는다 — 근거가 생길 때.

---

## 3. 포트 규칙

기본은 **파생 방식**이다. 두 대역에서 노드마다 stride 만큼 전진하고, 나머지는 http 에서 파생된다.

```
p2p   = p2pBase + (node-1) * p2pStep      wemix 계열만 etcd 를 p2p+1 로 파생
http  = rpcBase + (node-1) * rpcStep      ws = http+1, auth = http+2
                                          metrics = http+3 (rpcStep >= 4 일 때)
```

**용도별 대역**을 선언하면 파생이 꺼진다. 방화벽이 포트를 용도별 구간으로 여는
사이트(auth 85xx, http 86xx, ws 87xx, p2p 303xx)가 이 형태다:

```yaml
ports:
  p2p:     { base: 30301, step: 1 }
  rpc:     { base: 8601,  step: 1 }
  ws:      { base: 8701,  step: 1 }
  auth:    { base: 8501,  step: 1 }
  metrics: { base: 6060,  step: 0 }   # step 0 = 노드마다 같은 포트 (호스트당 1노드일 때)
```

검증은 두 겹이다. **파일 시점**에는 stride 가 전진 가능한지(`p2pStep >= 1`)와,
파생 방식일 때 `rpcStep >= 3`(http/ws/auth 를 담을 최소)만 본다. **할당 시점**에는
패밀리가 자기 요구를 말한다 — wemix 는 etcd 를 p2p+1(클라이언트 p2p+2)로 유도하므로
p2p step 이 그만큼 넓어야 하고, 좁으면 allocate 가 패밀리 이름을 대며 거부한다.
wbft 는 p2p 하나만 들으므로 step 1 이 합법이다.

---

## 4. 상속과 폴백

```
pool.ports  →  내장 기본값
```

두 단계다. 서버 세트는 **바꿀 값만** 적으면 된다 — 주소만 나열하고 포트를 생략하면 내장
대역(p2p 31000/10, rpc 8600/10)을 쓴다.

**서버 세트 파일이 없으면** 내장 기본값(p2p 31000 / http 8600, step 10)으로 동작하되,
스텝 출력이 출처를 밝힌다 — 포트가 왜 그 값인지는 추측 대상이 아니다.

```
allocate: 4 node(s); ports: built-in defaults (no server config); p2p from 31000, http from 8600
allocate: 4 node(s); ports: server-set.yaml[local]; p2p from 30303, http from 8545
```

단, **`--server` 로 서버를 지정했는데 서버 세트가 없으면 오류다.** 운영자가 고른
서버를 조용히 무시하고 다른 포트로 띄우면 안 된다.

---

## 5. 자격증명의 단일 출처는 서버 세트 파일이다

서버 세트에 이름이 있는 서버는 **파일이 말하는 그대로 접속된다. 환경변수는
읽지 않는다** — 다른 환경에서 export 해 둔 변수가 로그인을 조용히 바꿔치면
안 되기 때문이다. 비밀값을 파일 밖에 두려면 `ssh.password_file` /
`ssh.key_passphrase_file` 로 한 줄짜리 0600 파일을 가리킨다(상대 경로는 서버
세트 파일 기준, `~` 사용 가능).

`CHAINBENCH_REMOTE_USER` / `_PASS` / `_KEY_FILE` / `_KEY_PASSPHRASE` 는
직접 표기(`user@host:/path`)에만 남아 있다 — 그 형태에는 참조할 파일이 없다.

호스트키 정책도 이 파일의 데이터다: `ssh.known_hosts_file: <경로>` 로 검증 파일을
지정하거나, known_hosts 를 둘 수 없는 폐쇄망에서만 `ssh.insecure_host_key: true`
(정확히 하나만). 지정이 없으면 `~/.ssh/known_hosts` 로 검증한다 — 환경변수는 없다.

---

## 6. 사용

```sh
cp server-set.sample.yaml server-set.yaml
$EDITOR server-set.yaml        # 실제 호스트·포트 기입

# 이 머신에서
chainbench chain up --workspace-dir /tmp/n1 --chain stablenet \
  --binary $GSTABLE --server local

# 원격 호스트 하나에서
chainbench chain up --workspace-dir /tmp/n1 --chain stablenet \
  --binary /srv/bin/gstable --server bp1

# 서버 세트 전체에 한 노드씩 펼쳐서
chainbench chain up --workspace-dir /tmp/n1 --chain stablenet \
  --binary /srv/bin/gstable --all-servers

# DSL 실행도 같은 서버 세트를 읽는다
chainbench run --chain stablenet --binary $GSTABLE --keys keys/preset \
  --server local tests/specs/api/*.json
```

선택 플래그: `--server-set <path>`(기본 `server-set.yaml`) ·
`--server <name>` · `--server-index <n>` · `--all-servers`.

---

## 6b. v1 파일은 거부한다

v1(`servers:` 목록)을 만나면 **무엇을 어떻게 고쳐야 하는지 말하며 거부**한다 — 조용히
기본값으로 강등하면 운영자가 의도한 포트가 아닌 곳에서 네트워크가 뜬다.

```
serverset: <path> looks like the pre-v2 format: v2 replaced the server list with a pool:
put every address under `pool.hosts`, move `slots` and `ports` up to `pool`, and lift
`ssh`/`dataRoot` to the top level (see server-set.sample.yaml)
```

## 7. 아직 서버 세트를 읽지 않는 경로

`core/config` 의 `ports.base_*` 와 이를 쓰는 `core/bringup`(레거시 `setup` 명령)은
여전히 자체 기본값을 쓴다. 이 스택은 netcompose 전환이 라이브 검증되면 삭제된다
([[chainbench-worklist]] §1f b-5·b-6). 그 전까지 `setup` 과 `net`/`run` 은 서로 다른
포트 대역을 쓴다 — 같은 머신에서 둘을 동시에 띄우면 충돌하지 않는다는 뜻이기도 하다.
