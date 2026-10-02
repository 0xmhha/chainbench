# env/docker — 원격 서버 행세를 하는 로컬 docker

실 원격 서버 없이 chainbench 의 원격 코드 경로를 검증하기 위한 가상 서버들이다.
설계와 근거는 [`docs/dev/docker-remote-design.md`](../../docs/dev/docker-remote-design.md),
작업 상태는 worklist §1g R 트랙. 영어판은 [`README-EN.md`](README-EN.md) 에 있다.

- 컨테이너 = 빈 ubuntu 서버 + sshd. **접근은 실서버와 같은 모양**이다:
  `accounts.env` 의 첫 계정(기본 `devuser1`) + 공용 비밀번호, 키 로그인 없음,
  root 로그인 없음, **sudo 는 그 비밀번호를 요구**한다(NOPASSWD 아님 — 그 흐름을
  검증하는 것이 목적이므로). 실서버도 provision 된 dev 계정으로 접속하므로 같은
  모양이다. 체인 바이너리는 넣지 않는다 — 실서버처럼 provision 이 올린다.
  이미지에는 계정이 하나도 없다 — 모든 계정은 시작 시 주입된다.
- 기본 15대(`server1`~`server15`). **server15 는 pn 노드를 띄울 예정**이지만 서버
  계층에서는 구분하지 않는다 — 역할은 netmap 할당이 정한다.
- 퍼블리시 포트는 **127.0.0.1 에만** 바인딩된다. 이 머신 밖에서는 닿을 수 없다.
- **방화벽이 실서버와 같은 모양이다**: 허용 목록에 있는 것만 열고 나머지 인바운드를
  DROP 한다. 컨테이너마다 `firewall.sh` 가 여는 것은 TCP 10022·8501-8504·8601-8604·
  8701-8704·30301-30313·6060·3000·3001·9100·9090·1099·5901·5044·9200 와 UDP 30303 이다.
  p2p 대역만 넓은 것은 `firewall.sh` 가 그것을 슬롯 수에서 계산하기 때문이다 —
  `30301 + SLOTS × 3` 이라 기본 설정에서 13칸이 열린다. 나머지 대역은 슬롯당 한 칸이다.
  **열려 있는 칸 수와 세트가 쓰는 칸 수는 다르다.** 무엇을 쓸지는 server set 이 정한다.
- **sshd 는 10022** 에서 듣는다. 포트 배치도 실서버와 같은 규격이다:
  p2p 30301+1스텝, http 8601, ws 8701, auth 8501, metrics 6060.

## 쓰는 법

```bash
cd env/docker
cp accounts.env.sample accounts.env            # 열어서 실제 비밀번호로 바꾼다
./setup.sh                                     # 나머지 전부
```

`setup.sh` 가 아래 여섯 단계를 순서대로 한다. 단계 이름을 주면 그 단계만 다시
한다 — 컨테이너를 다시 만든 뒤 `./setup.sh binaries`, 남이 띄운 서버를 점검할 때
`./setup.sh verify`. `--recreate` 는 컨테이너를 다시 만들고, 그러면 `bin/` 이
비므로 바이너리를 다시 넣는 단계가 뒤따른다.

| 단계 | 하는 일 |
|---|---|
| `check` | docker 가 떠 있나, `accounts.env` 가 sample 그대로가 아닌가 |
| `generate` | `gen-env.sh` 로 `build/` 재생성 |
| `image` | 서버 이미지 빌드 |
| `up` | 컨테이너 기동 |
| `binaries` | 체인별 **리눅스** 바이너리를 골랑 컨테이너에서 빌드해 전 서버에 넣기 |
| `verify` | 서버마다 계정과 바이너리 3개가 ELF 로 있는지 |

손으로 하면 이렇다. `setup.sh` 가 하는 일이 이것이다.

```bash
cd env/docker
cp accounts.env.sample accounts.env            # 열어서 실제 비밀번호로 바꾼다
./gen-env.sh                                   # build/ 에 전부 생성 (손으로 쓰는 파일 0)
docker build -t chainbench-server:ubuntu24 .
docker compose -f build/docker-compose.yml up -d
ssh -p 2201 devuser1@127.0.0.1 hostname   # password: accounts.env 값 -> server1
```

`accounts.env` 없이(또는 비밀번호가 sample 값 `change-me` 그대로인 채로)
`gen-env.sh` 를 실행하면 안내와 함께 멈춘다 — placeholder 비밀번호로 sudo 계정을
띄우지 않기 위해서다. 이 파일은 gitignore 대상이고 커밋되면 안 된다.

`gen-env.sh` 가 `build/` 에 만드는 것은 다섯 개다. 손으로 쓰는 파일은 없다.

| 파일 | 무엇 |
|---|---|
| `docker-compose.yml` | 서버 15대. 퍼블리시 포트는 루프백에만 바인딩 |
| `server-set.yaml` | stablenet·wbft 용. `p2p_step` 1, 서버당 슬롯 4 |
| `server-set-wemix.yaml` | poa 용. `p2p_step` 3, 서버당 1노드 |
| `workspace-config.yaml` | `dataRoot=/data/chainbench`, `paths.binaries=bin` |
| `localmap.yaml` | `--docker` 가 쓰는 주소 번역표 |

## 스크립트를 고쳤으면 다시 생성하고 다시 만든다

`build/` 는 `gen-env.sh` 의 산출물이고 컨테이너는 그 산출물로 만들어진다. 그래서 스크립트를
고치는 것만으로는 **도는 컨테이너가 바뀌지 않는다.** 셋을 차례로 해야 한다.

```bash
cd env/docker
./gen-env.sh                                                   # build/ 재생성
docker compose -f build/docker-compose.yml up -d --force-recreate
# 컨테이너를 다시 만들면 /data/chainbench/bin/ 이 비므로 아래 절차로 바이너리를 다시 넣는다
```

2026-09-30 에 이것을 건너뛰어 웹소켓 케이스 둘이 실패했다. `gen-env.sh` 는 2026-09-19 에
ws 밴드를 퍼블리시하도록 고쳐졌는데 `build/docker-compose.yml` 이 2026-09-10 것이었고,
컨테이너는 그 낡은 compose 로 만들어져 8701 을 열지 않았다. `ws-subscribe-new-heads` 와
`ws-subscribe-logs` 가 `dial ws://127.0.0.1:8701: connection refused` 로 떨어졌고, 원인을
찾는 데 시간이 들었다. 지금 도는 컨테이너가 무엇을 퍼블리시하는지는 `docker port
chainbench-server1` 로 바로 볼 수 있다.

## 바이너리를 먼저 넣는다

`./setup.sh binaries` 가 이 절의 내용을 그대로 한다(체인 이름을 주면 하나만:
`./setup.sh binaries go-wemix`). 아래는 그것이 무엇을 하는지와, 왜 그렇게 하는지다.

컨테이너의 `/data/chainbench/bin/` 에 대상 체인의 **리눅스** 바이너리가 있어야 한다. 맥에서 만든 것은 Mach-O 라 돌지 않는다. 골랑 컨테이너에서 만들어 넣는다(2026-09-29 에 이렇게 만들었다).

**산출물 디렉터리는 홈 아래에 둔다.** `/tmp` 는 쓰면 안 된다 — macOS 의 Docker
Desktop 에서 `-v /tmp/out:/out` 은 호스트가 아니라 **VM 안의 `/tmp`** 에 붙는다.
빌드는 성공하는데 다음 줄의 `docker cp` 는 호스트에서 돌아 그 파일을 찾지 못한다.
2026-10-01 에 재현했다: 컨테이너 안 `/out` 에는 `gstable` 이 보이는데 호스트의
`/tmp/out` 은 비어 있었다.

```bash
OUT=~/cbw/linuxbin && mkdir -p "$OUT"

# 저장소는 읽기 전용으로 붙인다 — 산출물이 기존 build/bin 을 덮지 않게.
# go-stablenet
docker run --rm -v ~/Work/github/chain/go-stablenet:/src:ro -v "$OUT":/out \
  -v cbgocache:/gocache -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
  -w /src golang:1.25 go build -o /out/gstable ./cmd/gstable

# go-wbft — ./cmd/gwemix 를 빌드해 gwbft 로 넣는다. 저장소가 go-wemix 에서
# 갈라져 나와 명령 이름이 남았고, 매니페스트는 gwbft 를 부른다.
docker run --rm -v ~/Work/github/chain/go-wbft:/src:ro -v "$OUT":/out \
  -v cbgocache:/gocache -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
  -w /src golang:1.25 go build -o /out/gwbft ./cmd/gwemix

# go-wemix — go.mod 이 1.19 를 요구한다.
docker run --rm -v ~/Work/github/chain/go-wemix:/src:ro -v "$OUT":/out \
  -v cbgocache:/gocache -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
  -w /src golang:1.19 go build -o /out/gwemix ./cmd/gwemix

for i in $(seq 1 15); do
  docker exec -u root chainbench-server$i mkdir -p /data/chainbench/bin
  for b in gstable gwbft gwemix; do
    docker cp "$OUT/$b" chainbench-server$i:/data/chainbench/bin/$b
    docker exec -u root chainbench-server$i chmod 755 /data/chainbench/bin/$b
  done
done

docker exec chainbench-server1 /data/chainbench/bin/gstable version | head -3
```

마지막 줄로 무엇이 올라갔는지 확인한다. `gstable` 과 `gwbft` 는 `Git Commit` 을
찍지만 `gwemix` 는 그 빌드가 커밋을 심지 않아 찍지 않는다 — 바이너리만 보고는
출처를 되짚을 수 없으므로, 어느 HEAD 에서 빌드했는지는 따로 적어 둔다.

## 어느 서버 세트인가

go-wemix 는 `server-set-wemix.yaml` 을 쓴다. poa 가 p2p 옆 포트를 3개 잡아 `p2p_step` 이 3이어야 하고, 기본 세트는 1이라 거절당한다.

노드를 여러 대에 퍼뜨리려면 `--all-servers` 가 필요하다. 세트에 서버가 15대라 이름을 대지 않으면 거절하고, 하나를 대면 그 서버의 슬롯(4개)만 쓴다. 방화벽으로 망을 가르는 케이스는 무리마다 기계가 달라야 하므로 이 옵션이 필수다.

## 15대에 테스트 돌리기

`--all-servers` 는 노드를 서버당 하나씩 15대에 퍼뜨리고, `--docker` 는 dial 을
localmap 으로 번역한다. 표준 15노드(bp 7 · en 7 · pn 1) 스모크 테스트:

```bash
# 각 컨테이너 /data/chainbench/bin/ 에 대상 체인 바이너리(Linux)가 있어야 한다.
bin/chainbench run \
  --workspace-dir <ws> \
  --server-set env/docker/build/server-set.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers \
  --keys <ws>/genkeys \
  tests/tc/common/node/CT-NODE-002-startup-15-nodes.json
```

`chain-up-15` 의 env 블록이 15노드 topology 를 선언한다. 바이너리 경로는 선언하지
않는다 — 어느 바이너리인지는 체인 매니페스트가 이름으로 답하고, 그 이름이 어디
있는지는 위에서 넘기는 `workspace-config.yaml` 의 `dataRoot` 와 `paths.binaries`
가 답한다. 둘을 합치면 `/data/chainbench/bin/gstable` 이 나온다. 정의서가 그 경로를
직접 적으면 그 케이스는 이 도커 환경에서만 돈다.

키는 15개가 필요해 preset(5개) 대신 generate 로 만든다 — 생성 세트는 topology 의
bp 수(7)만 validator 로 선언한다.

### 전량 스위프를 이 15대에서

`scripts/tcsweep.sh` 는 기본적으로 로컬 바이너리로 돈다. `TCSWEEP_FLAGS` 를 주면 같은
스위프가 이 컨테이너들을 향한다 — 로컬에 체인 바이너리가 없는 기기에서는 이쪽이
유일한 경로다. 케이스마다 `chain stop` 이 걸리고(포트를 놓아야 다음 케이스가 쓴다),
**통과한 것만** `chain rm` 까지 간다. 통과하지 못한 케이스는 컨테이너의 datadir·노드
로그와 로컬 워크스페이스를 남기고 그 경로를 로그에 찍는다 — 실패를 다시 볼 때 필요한
것은 `~/.chainbench` 의 증적이 아니라 그 뒤의 기계 상태다. 통과한 것을 지우는 이유는
남겨 두면 다음 케이스의 genesis 가 그것을 만나 `incompatible genesis` 로 막히기 때문이다.

**한 번에 다 돌릴 수 없다. 세 번에 나눈다.** 체인마다 서버 세트가 다르고,
`tcsweep.sh` 는 `TCSWEEP_FLAGS` 를 전 케이스에 똑같이 적용하기 때문이다.

```bash
cd <chainbench 디렉터리>      # env/ 와 scripts/ 가 보이는 곳
make build

export TCSWEEP_FLAGS="--server-set $PWD/env/docker/build/server-set.yaml \
  --workspace-config $PWD/env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys-source generate"

# 1차 — stablenet 177건 + wbft 8건. go-stablenet/testnet 만 뺀다(아래 참조).
scripts/tcsweep.sh ~/cbw/sweep-1.log \
  'tests/tc/basic\|tests/tc/common\|tests/tc/go-wbft\|go-stablenet/regression\|go-stablenet/post-v1.0.0-change\|go-stablenet/hardfork\|go-stablenet/vocabulary'

# 2차 — go-wemix 7건. 서버 세트와 기동 예산이 다르다.
TCSWEEP_FLAGS="--server-set $PWD/env/docker/build/server-set-wemix.yaml \
  --workspace-config $PWD/env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys-source generate --node-monitor-timeout 5m" \
scripts/tcsweep.sh ~/cbw/sweep-2.log 'tests/tc/go-wemix'
```

2차가 따로인 이유는 위 "어느 서버 세트인가" 와 같다. poa 는 `p2p_step` 3 이 필요한데
기본 세트는 1 이라 place 단계에서 거절당하고, 기동도 느려 `--node-monitor-timeout` 을
올려야 한다.

**3차는 도커가 아니다.** `tests/tc/go-stablenet/testnet/` 의 5건은 우리가 세우지 않은
망에 붙는 케이스다. `tcsweep.sh` 의 `runAttach` 는 attach 케이스에 `GSTABLE_RPC` 만
넣어 주는데 이 다섯은 `GSTABLE_TESTNET_RPC` 를 요구하므로, 1차에 섞으면 전부 BLOCKED
로 떨어진다. 실행 방법은
[`tests/tc/go-stablenet/testnet/README.md`](../../tests/tc/go-stablenet/testnet/README.md)
에 있다.

`--server` 없이 `--server-set` 만 주면 15대 중 어느 것인지 몰라 place 단계가 거절한다.
`--all-servers` 는 노드를 서버당 하나씩 퍼뜨려 그 질문을 없앤다.

케이스 하나는 빠른 것이 30초쯤이다(2026-10-01, CT-RPC-001 을 이 15대에서 실측).
9노드를 띄워 가르고 다시 잇는 CT-FAULT-004 같은 것은 몇 분씩 걸리므로, 1차는
`nohup` 이나 `tmux` 로 돌리고 `tail -f` 로 본다.

Go 테스트 쪽 게이트는 `CHAINBENCH_DOCKER_SERVERS` 다:

```bash
CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build go test -p 1 -timeout 60m ./...
```

**live 테스트는 없는 fixture 를 skip 하지 않고 fail 한다.** 각 테스트 주석 맨 위에
`docker exec ...` 로 무엇을 심어야 하는지 적혀 있다. 심지 않고 돌리면 회귀처럼 보인다.

### 세 체인 패밀리 15대 스모크 (stablenet · wbft · go-wemix)

세 패밀리 모두 15대에서 검증됐다(검증 당시 4 bp + 11 en, 지금 표준은 bp 7 · en 7 · pn 1):

```bash
# stablenet (wbft 패밀리)  — server-set.yaml, 기본 게이트
bin/chainbench run --workspace-dir <ws> --server-set env/docker/build/server-set.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys <ws>/genkeys --keys-source generate \
  tests/tc/common/node/CT-NODE-002-startup-15-nodes.json

# wbft                      — server-set.yaml, 기본 게이트
bin/chainbench run ... --chain-preset wbft-bp7-en7-pn1 tests/tc/common/node/CT-NODE-002-startup-15-nodes.json

# go-wemix (poa)           — server-set-wemix.yaml 필요, 게이트 예산 상향
bin/chainbench run --workspace-dir <ws> --server-set env/docker/build/server-set-wemix.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys <ws>/genkeys --keys-source generate \
  --node-monitor-timeout 5m \
  --chain-preset wemix-bp7-en7-pn1 \
  tests/tc/common/node/CT-NODE-002-startup-15-nodes.json
```

go-wemix(poa)는 stablenet/wbft 와 두 가지가 다르다:

1. **서버 세트가 다르다** — poa 는 노드마다 p2p 옆에 3연속 포트(p2p+etcd)를 잡으므로
   `p2p step ≥ 3` 이 필요하다. `gen-env.sh` 가 이 용도로 `server-set-wemix.yaml`
   (slots 1, p2p step 3)을 함께 찍는다. 일반 `server-set.yaml`(step 1)로 wemix 를
   올리면 place 단계에서 `p2p_step must be >= 3` 로 거절된다.
2. **게이트 예산을 올린다** — poa 는 producer 부트(4 bp) → 거버넌스 배포 → etcd 형성 →
   endpoint(11 en) 조인 순으로 뜨는데, 늦게 조인한 endpoint 의 sync 가 nodemonitor 기본
   대기(90s)보다 오래 걸린다. `--node-monitor-timeout 5m` 로 게이트가 형성 중인 망을
   조기 종료하지 않게 한다.

**체인을 바꿔 다시 돌릴 때는 그 구성을 먼저 지운다.** genesis 가 다르면(다른 키·다른
패밀리) 남아 있는 datadir·genesis 가 init 을 `incompatible genesis` 로 막는다. `chain rm`
이 원격에서도 동작하므로(2026-09-11) 컨테이너를 손으로 비울 필요가 없다:

```bash
bin/chainbench chain stop --workspace-dir <ws>   # rm 은 실행 중인 노드를 거부한다
bin/chainbench chain rm   --workspace-dir <ws>
```

**자기 구성만 지운다** — 같은 서버의 다른 구성과 `bin/` 은 건드리지 않는다(구성 id 로
경로가 갈리고, 삭제는 target 의 dataRoot 안으로 구속된다). 라이브 확인: stablenet 3노드
→ stop → rm → **수동 정리 없이** wbft 3노드가 같은 서버에 올라가 블록 8까지 진행.

워크스페이스를 잃어버려 `chain rm` 을 걸 수 없을 때만 손으로 비운다:

```bash
for i in $(seq 1 15); do docker exec chainbench-server$i sh -c \
  'cd /data/chainbench && find . -maxdepth 1 -mindepth 1 ! -name bin -exec rm -rf {} +'; done
```

생성물(`build/`, gitignore):

| 파일 | 내용 |
|---|---|
| `docker-compose.yml` | server1~N. bridge 고정 주소 172.30.0.11+, ssh 22→2201+, rpc 8600→18601+, metrics 6060→16061+ |
| `server-set.yaml` | **서버 세트 v2, 실주소 기재** — 운영 서버 세트와 같은 모양. slots 4, p2p step 1 |
| `server-set-wemix.yaml` | 같은 서버·같은 자격증명에 poa 용 노브만 다름 — slots 1, p2p step 3 |
| `workspace-config.yaml` | 대상 `dataRoot` 와 용도별 디렉터리. server-set 이 더는 `dataRoot` 를 갖지 않으므로 짝으로 넘겨야 한다 |
| `localmap.yaml` | 실주소→loopback 퍼블리시 포트 대응표. `--docker` 일 때만 적용(R1) |

> **매핑에 없는 포트는 조용히 그대로 나간다.** `AddrMap` 은 호스트만 바꾸고 매핑이 없는
> 포트는 원본을 유지하므로, 퍼블리시하지 않은 포트로 dial 하면 `127.0.0.1:<컨테이너 포트>`
> 라는 그럴듯한 주소가 만들어지고 아무것도 답하지 않는다. metrics 6060 이 정확히 그랬다
> (2026-09-11 수정). 새 포트를 쓰기 시작하면 compose 의 publish 와 localmap 양쪽에 더한다.

대수를 바꾸려면 `SERVERS=20 ./gen-env.sh` 후 compose 를 다시 올린다.

## 돌아가는 것을 어떻게 보나

스위프를 띄워 놓고 무엇을 보면 되는지. 아래는 모두 읽기만 하므로 도는 스위프에
영향을 주지 않는다.

### 어디까지 왔나

```bash
tail -f ~/cbw/sweep-1.log
```

한 줄에 한 케이스다. 판정은 넷이다 — `PASS`, `FAIL`(케이스가 틀렸다),
`BLOCKED`(물어보지도 못했다), `NOTRUN`(바이너리를 못 찾았다). **`NOTRUN` 이 보이면
`TCSWEEP_FLAGS` 를 넘기지 않은 것이다**: 그러면 모든 run 이 로컬로 떨어져 체인
바이너리를 이 기계의 PATH 에서 찾고, 그것은 컨테이너에만 있다.

지금 어느 케이스를 돌고 있는지는 프로세스가 답한다.

```bash
ps -eo etime,args | grep '[b]in/chainbench run' | sed -E 's#(--workspace-dir [^ ]*).*#\1#'
```

### 체인이 정말 도커에 올라갔나

```bash
W=~/cbw/sweep/c11                                  # 스위프는 cN 을 쓴다
bin/chainbench chain status --workspace-dir $W     # 8단계 중 어디까지
bin/chainbench chain show   --workspace-dir $W     # 노드 표
```

**로컬로 떨어졌는지는 `show` 의 `host` 가 바로 답한다.** `172.30.0.x` 면 컨테이너로
간 것이고, 맥 경로가 보이면 로컬이다. 노드를 고르는 방법은 여섯 가지다 — `--node`,
`--label`(`node7`·`en2`), `--host`, `--addr`, `--port`, 그리고 `--json`.

### 노드가 무슨 말을 하나

```bash
bin/chainbench chain logs --workspace-dir $W --node 1 --lines 50
```

노드가 올라간 **컨테이너의 로그를 SSH 로 읽어 온다**(필요하면 sudo 로 올라간다).
로그만 보고 고장이라 읽지 않도록 어느 케이스인지 함께 본다: fault 케이스는 합의를
일부러 멈추므로 ROUND-CHANGE 반복이 정상이다.

### 테스트가 무엇을 판정했나

증적은 `~/.chainbench/sessions/<id>/UTC-<ts>/` 에 쌓인다.

```bash
S=$(ls -dt ~/.chainbench/sessions/*/ | head -1)
cat $S/UTC-*/tests/*/status.json      # 실패 사유 한 줄 — 가장 쓸모 있다
cat $S/UTC-*/tests/*/steps.json       # 단계별 결과와 오류
cat $S/UTC-*/session.json             # 케이스별 verdict
```

여러 세션을 한 판정으로 묶으려면:

```bash
bin/chainbench report --workspace-dir ~/.chainbench/sessions --all
```

**플래그 이름이 `--workspace-dir` 인데 받는 것은 세션 디렉터리다.** `--session-dir`
는 없다.

### 실패한 케이스를 다시 볼 때

스위프는 통과한 것만 지우고, 실패하면 워크스페이스와 컨테이너의 datadir·로그를
남기고 그 경로를 찍는다(`kept for inspection: ...`). `chain-record.json` 에 구성 id,
노드별 호스트·포트·pid, `docker`·`serverSet` 설정이 들어 있다. 다 본 뒤 치운다 —
레코드가 docker 설정을 들고 있으므로 `--docker` 를 다시 줄 필요가 없다.

```bash
bin/chainbench chain stop --workspace-dir $W   # rm 은 돌고 있는 노드를 거부한다
bin/chainbench chain rm   --workspace-dir $W
```

### 컨테이너를 직접 볼 때

하네스를 못 믿을 때의 길이다.

```bash
for i in $(seq 1 15); do
  n=$(docker exec chainbench-server$i sh -c 'ps -eo args | grep -c "[/]data/chainbench/bin/"')
  echo "server$i: $n"
done

docker exec chainbench-server1 ls /data/chainbench/logs      # <구성id>/node1.log
docker exec chainbench-server1 ls /data/chainbench/node      # <구성id>/node1
docker exec chainbench-server1 ls /data/chainbench/runtime   # <구성id>/genesis·config
docker port chainbench-server1                               # 지금 무엇을 열었나
```

**로그는 노드가 있는 서버에만 있다.** `--all-servers` 는 서버당 한 노드를 두므로
server1 에는 `node1.log` 뿐이고 node7 의 로그는 server7 에 있다.

### 스위프를 둘 동시에 돌리지 않는다

경로는 `TCSWEEP_WS` 로 가를 수 있지만 **서버 포트가 겹친다.** 할당기가 서버마다
슬롯 1을 주므로 두 스위프가 모두 8601 을 원하고, `another composition holds it` 로
BLOCKED 가 난다. 그러면 남의 구성을 남겨진 것으로 오독하게 된다. 2026-10-01 에 그렇게
한 번 틀렸다. 먼저 확인한다:

```bash
ps -eo etime,args | grep '[t]csweep'
```

## 개발 계정 주입 (accounts.env)

공용 개발 계정은 `accounts.env`(gitignore, `accounts.env.sample` 을 복사해
만든다)에 `name:uid` 목록 + 공용 비밀번호로 적는다. 컨테이너가 시작할
때마다 `setup-accounts.sh` 가 이 파일을 읽어 계정을 만든다(compose command 에
연결). 값을 바꾸면 `./gen-env.sh` 로 다시 생성한 뒤 `docker compose -f
build/docker-compose.yml up -d` 로 재생성해야 반영된다 — `restart` 는 예전
compose 명령을 그대로 다시 돌리므로 계정 이름 변경을 적용하지 못한다(비밀번호만
바꿨다면 restart 로도 충분하다). 이미지 재빌드는 어느 쪽이든 필요 없다. 다른
도구가 같은 계정 정보를 쓸 때도 이 파일을 읽는다.

하네스 로그인도 이 파일에서 나온다: `gen-env.sh` 가 **첫 계정**을 서버 세트
(`build/server-set.yaml`)의 `ssh: user/password` 로 찍어내고,
dataRoot 도 그 계정 소유로 만든다. 계정 정보의 출처는 이 파일 하나다.

정리:

```bash
docker compose -f build/docker-compose.yml down
```

## 왜 서버 세트에 컨테이너 실주소를 쓰나

노드끼리는 bridge 안에서 실주소로 통신해야 한다(genesis·static-nodes 에 들어가는
주소). loopback 치환은 **하네스가 스스로 접속하는 순간에만** 일어나며, 그 스위치가
`--docker` 옵션이다. 파일이 남아 있어도 옵션이 없으면 아무 일도 하지 않는다.
