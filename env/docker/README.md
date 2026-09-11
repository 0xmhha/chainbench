# env/docker — 원격 서버 행세를 하는 로컬 docker

실 원격 서버 없이 chainbench 의 원격 코드 경로를 검증하기 위한 가상 서버들이다.
설계와 근거는 [`docs/dev/docker-remote-design.md`](../../docs/dev/docker-remote-design.md),
작업 상태는 worklist §1g R 트랙.

- 컨테이너 = 빈 ubuntu 서버 + sshd. **접근은 실서버와 같은 모양**이다:
  `accounts.env` 의 첫 계정(기본 `devuser1`) + 공용 비밀번호, 키 로그인 없음,
  root 로그인 없음, **sudo 는 그 비밀번호를 요구**한다(NOPASSWD 아님 — 그 흐름을
  검증하는 것이 목적이므로). 실서버도 provision 된 dev 계정으로 접속하므로 같은
  모양이다. 체인 바이너리는 넣지 않는다 — 실서버처럼 provision 이 올린다.
  이미지에는 계정이 하나도 없다 — 모든 계정은 시작 시 주입된다.
- 기본 15대(`server1`~`server15`). **server15 는 pn 노드를 띄울 예정**이지만 서버
  계층에서는 구분하지 않는다 — 역할은 netmap 할당이 정한다.
- 퍼블리시 포트는 **127.0.0.1 에만** 바인딩된다. 이 머신 밖에서는 닿을 수 없다.
- **방화벽이 실서버와 같다**: 컨테이너마다 `firewall.sh` 가 Wemix3.5 테스트 서버의
  허용 목록(TCP 10022·8501-8504·8601-8604·8701-8704·6060·3000·3001·9100·9090·
  30301-30304·1099·5901·5044·9200, UDP 30303)만 열고 나머지 인바운드를 DROP 한다.
- **sshd 는 10022** 에서 듣는다(실서버 포트). 포트 규격도 실서버 그대로다:
  p2p 30301+1스텝, http 8601, ws 8701, auth 8501, metrics 6060.

## 쓰는 법

```bash
cd env/docker
cp accounts.env.sample accounts.env            # 열어서 실제 비밀번호로 바꾼다
./gen-env.sh                                   # build/ 에 전부 생성 (손으로 쓰는 파일 0)
docker build -t chainbench-server:ubuntu24 .
docker compose -f build/docker-compose.yml up -d
ssh -p 2201 devuser1@127.0.0.1 hostname   # password: accounts.env 값 -> server1
```

`accounts.env` 없이(또는 비밀번호가 sample 값 그대로인 채로) `gen-env.sh` 를
실행하면 안내와 함께 멈춘다 — placeholder 비밀번호로 sudo 계정을 띄우지 않기
위해서다.

## 15대에 테스트 돌리기

`--all-servers` 는 노드를 서버당 하나씩 15대에 퍼뜨리고, `--docker` 는 dial 을
localmap 으로 번역한다. 15노드(4 bp + 11 en) 스모크 테스트:

```bash
# 각 컨테이너 /data/chainbench/bin/ 에 대상 체인 바이너리(Linux)가 있어야 한다.
bin/chainbench run \
  --workspace-dir <ws> \
  --server-set env/docker/build/server-set.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers \
  --keys <ws>/genkeys \
  tests/tc/go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json
```

`chain-up-15` 의 env 블록(정의서 안에 인라인, id `stablenet-docker15`)이 15노드
topology 와 컨테이너 바이너리 경로를 선언한다. 키는 15개가 필요해 preset(5개)
대신 generate 로 만든다 — 생성 세트는 topology 의 validator 수(4)만 validator 로
선언한다.

### 세 체인 패밀리 15대 스모크 (stablenet · wbft · go-wemix)

세 패밀리 모두 15대(4 bp + 11 en)에서 검증됐다:

```bash
# stablenet (wbft 패밀리)  — server-set.yaml, 기본 게이트
bin/chainbench run --workspace-dir <ws> --server-set env/docker/build/server-set.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys <ws>/genkeys --keys-source generate \
  tests/tc/go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json

# wbft                      — server-set.yaml, 기본 게이트
bin/chainbench run ... tests/tc/go-wbft/chain-up/02-wbft-chain-up-15.json

# go-wemix (poa)           — server-set-wemix.yaml 필요, 게이트 예산 상향
bin/chainbench run --workspace-dir <ws> --server-set env/docker/build/server-set-wemix.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys <ws>/genkeys --keys-source generate \
  --node-monitor-timeout 5m \
  tests/tc/go-wemix/chain-up/02-wemix-chain-up-15.json
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
