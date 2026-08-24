# env/docker — 원격 서버 행세를 하는 로컬 docker 함대

실 원격 서버 없이 chainbench 의 원격 코드 경로를 검증하기 위한 가상 서버들이다.
설계와 근거는 [`docs/dev/docker-remote-design.md`](../../docs/dev/docker-remote-design.md),
작업 상태는 worklist §1g R 트랙.

- 컨테이너 = 빈 ubuntu 서버 + sshd(키 인증만). 체인 바이너리는 넣지 않는다 —
  실서버처럼 provision 이 올린다.
- 기본 15대(`server1`~`server15`). **server15 는 pn 노드를 띄울 예정**이지만 서버
  계층에서는 구분하지 않는다 — 역할은 netmap 할당이 정한다.
- 퍼블리시 포트는 **127.0.0.1 에만** 바인딩된다. 이 머신 밖에서는 닿을 수 없다.

## 쓰는 법

```bash
cd env/docker
./gen-env.sh                                   # build/ 에 전부 생성 (손으로 쓰는 파일 0)
docker build -t chainbench-server:ubuntu24 .
docker compose -f build/docker-compose.yml up -d
ssh -i build/ssh/id_ed25519 -p 2201 root@127.0.0.1 hostname   # -> server1
```

생성물(`build/`, gitignore):

| 파일 | 내용 |
|---|---|
| `docker-compose.yml` | server1~N. bridge 고정 주소 172.30.0.11+, ssh 22→2201+, rpc 8600→18601+ |
| `remote-server-config.yaml` | **인벤토리 v2, 실주소 기재** — 운영 인벤토리와 같은 모양 |
| `localmap.yaml` | 실주소→loopback 퍼블리시 포트 대응표. `--docker` 일 때만 적용(R1) |
| `ssh/` | 개발 전용 ed25519 키쌍 + authorized_keys |

대수를 바꾸려면 `SERVERS=20 ./gen-env.sh` 후 compose 를 다시 올린다.

정리:

```bash
docker compose -f build/docker-compose.yml down
```

## 왜 인벤토리에 컨테이너 실주소를 쓰나

노드끼리는 bridge 안에서 실주소로 통신해야 한다(genesis·static-nodes 에 들어가는
주소). loopback 치환은 **하네스가 스스로 접속하는 순간에만** 일어나며, 그 스위치가
`--docker` 옵션이다. 파일이 남아 있어도 옵션이 없으면 아무 일도 하지 않는다.
