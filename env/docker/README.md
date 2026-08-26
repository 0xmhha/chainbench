# env/docker — 원격 서버 행세를 하는 로컬 docker 함대

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

생성물(`build/`, gitignore):

| 파일 | 내용 |
|---|---|
| `docker-compose.yml` | server1~N. bridge 고정 주소 172.30.0.11+, ssh 22→2201+, rpc 8600→18601+ |
| `server-set.yaml` | **서버 세트 v2, 실주소 기재** — 운영 서버 세트와 같은 모양 |
| `localmap.yaml` | 실주소→loopback 퍼블리시 포트 대응표. `--docker` 일 때만 적용(R1) |

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
