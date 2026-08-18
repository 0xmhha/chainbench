# 서버 인벤토리 — 포트·호스트·접속 정보 (`remote-server-config.yaml`)

> 노드가 **어디서** 뜨고 **어떤 포트로** 듣는지를 결정하는 단일 출처.
> 실제 파일은 **git 에 절대 올라가지 않는다**(gitignore). 추적되는 것은 템플릿
> `remote-server-config.sample.yaml` 하나뿐이다.

---

## 1. 왜 별도 파일인가

호스트 주소와 포트 할당은 **사이트마다 다르고 민감정보**다. 코드 상수나 테스트 정의서에
들어가면 리포지토리에 박제되고, 사내 IP·포트가 공개 저장소로 새는 경로가 된다.

따라서 이 값들은 **데이터**이고, 그 데이터는 운영자의 로컬 파일에만 존재한다.

```
remote-server-config.sample.yaml   추적됨 — 형식만 담은 템플릿
remote-server-config.yaml          gitignore — 실제 값
```

`.gitignore` 는 `*remote-server-config.yaml` / `.yml` 을 막고 `.sample.yaml` 만 예외로 둔다.

---

## 2. local 과 remote 가 같은 구조인 이유

한 항목이 이 머신과 원격 호스트를 **동일한 필드로** 기술한다. 다른 것은 `kind` 와
`ssh` 블록뿐이다.

```yaml
servers:
  - name: local
    kind: local          # ssh 블록은 참조되지 않음
    host: 127.0.0.1
    slots: 8             # 한 호스트에 여러 노드 → 포트가 step 만큼 벌어짐

  - name: bp1
    kind: remote         # ssh 블록으로 접속
    host: 10.0.0.11
                         # slots 기본 1 → 호스트당 1노드, 모든 호스트 동일 포트
```

덕분에 네트워크를 구성하는 쪽은 **분기하지 않는다.** local/remote 차이는
`serverset.Placement` 가 흡수해 allocator 입력(`place.Config`·mode·capacity)으로만
드러난다.

| | local | remote |
|---|---|---|
| 배치 모드 | `LocalStepped` — 한 호스트, 포트를 step 만큼 증가 | `RemotePerHost` — 호스트마다 1노드, 포트 동일 |
| 데이터 플레인 | 이 머신의 `dataRoot` | 그 호스트의 `dataRoot` |
| 접속 | 없음 | `ssh` (자격증명은 env 우선) |

---

## 3. 포트 규칙

두 개의 **서로 겹치지 않는 대역**에서 노드마다 stride 만큼 전진한다.

```
p2p   = p2pBase + (node-1) * p2pStep      etcd 는 p2p+1 로 파생됨
http  = rpcBase + (node-1) * rpcStep      ws = http+1, auth = http+2
                                          metrics = http+3 (rpcStep >= 4 일 때)
```

**`p2pStep >= 2` 는 스타일이 아니라 필수다.** wemix 계열 바이너리는 etcd 포트를
`p2p+1` 로 유도하므로, step 이 1 이면 etcd 가 다음 노드의 p2p 포트와 충돌하고
**블록 생성이 원인 불명으로 멈춘다.** `rpcStep >= 3` 은 http/ws/auth 를 담기 위한 최소값.

로더가 이 두 조건을 파일 읽는 시점에 검증한다.

---

## 4. 상속과 폴백

```
서버 항목  →  defaults  →  내장 기본값
```

세 단계다. 인벤토리는 **바꿀 값만** 적으면 된다.

**인벤토리 파일이 없으면** 내장 기본값(p2p 31000 / http 8600, step 10)으로 동작하되,
스텝 출력이 출처를 밝힌다 — 포트가 왜 그 값인지는 추측 대상이 아니다.

```
allocate: 4 node(s); ports: built-in defaults (no server config); p2p from 31000, http from 8600
allocate: 4 node(s); ports: remote-server-config.yaml[local]; p2p from 30303, http from 8545
```

단, **`--server` 로 서버를 지정했는데 인벤토리가 없으면 오류다.** 운영자가 고른
서버를 조용히 무시하고 다른 포트로 띄우면 안 된다.

---

## 5. 자격증명은 파일이 아니라 환경에서

`ssh.password` / `ssh.key_file` 을 파일에 쓸 수는 있지만, **환경변수가 항상 이긴다.**
그래야 인벤토리 파일 자체를 secret-free 로 유지할 수 있다.

```
CHAINBENCH_REMOTE_USER
CHAINBENCH_REMOTE_PASS
CHAINBENCH_REMOTE_KEY_FILE  (+ CHAINBENCH_REMOTE_KEY_PASSPHRASE)
```

호스트키 정책은 별개다: `CHAINBENCH_SSH_KNOWN_HOSTS`, 또는 known_hosts 가 없는
폐쇄망에서 `CHAINBENCH_SSH_INSECURE_HOST_KEY=1`.

---

## 6. 사용

```sh
cp remote-server-config.sample.yaml remote-server-config.yaml
$EDITOR remote-server-config.yaml        # 실제 호스트·포트 기입

# 이 머신에서
chainbench net up --data-dir /tmp/n1 --chain stablenet \
  --binary $GSTABLE --server local

# 원격 호스트 하나에서
chainbench net up --data-dir /tmp/n1 --chain stablenet \
  --binary /srv/bin/gstable --server bp1

# 인벤토리 전체에 한 노드씩 펼쳐서
chainbench net up --data-dir /tmp/n1 --chain stablenet \
  --binary /srv/bin/gstable --fleet

# DSL 실행도 같은 인벤토리를 읽는다
chainbench run --chain stablenet --binary $GSTABLE --keys keys/preset \
  --server local tests/specs/api/*.json
```

선택 플래그: `--server-config <path>`(기본 `remote-server-config.yaml`) ·
`--server <name>` · `--server-index <n>` · `--fleet`.

---

## 7. 아직 인벤토리를 읽지 않는 경로

`core/config` 의 `ports.base_*` 와 이를 쓰는 `core/bringup`(레거시 `setup` 명령)은
여전히 자체 기본값을 쓴다. 이 스택은 netcompose 전환이 라이브 검증되면 삭제된다
([[chainbench-worklist]] §1f b-5·b-6). 그 전까지 `setup` 과 `net`/`run` 은 서로 다른
포트 대역을 쓴다 — 같은 머신에서 둘을 동시에 띄우면 충돌하지 않는다는 뜻이기도 하다.
