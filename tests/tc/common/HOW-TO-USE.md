# tests/tc/common — 테스트 돌리는 법 (처음 오는 사람용)

이 문서는 이 저장소를 처음 보는 사람이 `tests/tc/common/` 아래 케이스를 자기 손으로
돌려 보게 하는 것이 목적이다. 무엇을 보는 케이스인지는 [`README.md`](README.md) 가
설명하고, 케이스마다 복사해 쓸 수 있는 명령 한 줄씩은 그 문서 8절에 모아 두었다. 여기서는
그 명령이 왜 그렇게 생겼는지, 무엇을 먼저 갖춰야 하는지, 결과를 어떻게 읽는지를 적는다.

chainbench 는 블록체인 테스트 망을 세우고 DSL(JSON)로 적힌 케이스를 그 망에 돌린 뒤
통과·실패를 판정하는 CLI 다. 케이스 하나를 돌린다는 것은 (1) 노드 몇 대짜리 망을 새로
세우고 (2) 케이스가 시키는 트랜잭션·조회·장애를 그 망에 걸고 (3) 케이스가 적어 둔
기대값과 실제를 견주는 세 단계를 한 번에 하는 것이다. 명령 한 줄이 이 셋을 다 한다.

---

## 1. 무엇을 먼저 갖춰야 하나

세 가지가 있어야 한다. 셋째(Docker)는 원격을 흉내 내 돌릴 때만 필요하다.

**chainbench 바이너리.** 저장소 루트에서 `make build` 를 하면 `bin/chainbench` 가
생긴다. 이 문서의 모든 명령은 저장소 루트에서 상대 경로 `bin/chainbench` 로 부른다.

```sh
cd <이 저장소>          # 예: cd ~/Work/github/chainbench
make build
bin/chainbench --help   # 여기까지 되면 준비 끝
```

**체인 노드 바이너리.** 케이스가 세우는 망의 노드가 실제로 돌 실행 파일이다. 세 체인
패밀리가 있고 케이스마다 어느 것이 필요한지 정해져 있다. 저장소가 아니라 각 체인 소스에서
따로 빌드한다.

| 체인 | 빌드 산출물 | 이 문서에서 부르는 이름 |
| --- | --- | --- |
| go-stablenet | `build/bin/gstable` | `$GSTABLE` |
| go-wbft | `build/bin/gwemix` (`make gwemix` — 저장소가 go-wemix 에서 갈라져 이름이 남았다) | `$GWBFT` |
| go-wemix | `build/bin/gwemix` | `$GWEMIX` |

세 경로를 한 번 변수로 잡아 두면 8절의 명령을 그대로 붙여 쓸 수 있다. 자기 기계의 실제
경로로 바꾼다.

```sh
export GSTABLE="$HOME/work/github/go-stablenet/build/bin/gstable"
export GWBFT="$HOME/work/github/go-wbft/build/bin/gwemix"
export GWEMIX="$HOME/work/github/go-wemix/build/bin/gwemix"
```

공통 89개 중 stablenet 바이너리만 있으면 도는 것이 대부분이다. 이름에 `wbft`·`wemix` 가
붙은 8개(3절)만 각 체인 바이너리를 요구한다. **셋 다 없어도 좋다 — stablenet 하나로
공통의 대부분을 돌릴 수 있다.**

**(원격을 흉내 낼 때만) Docker 함대.** 로컬 바이너리가 없는 기계, 또는 노드가 서로 다른
기계에 있어야 도는 케이스(3.3의 `fault/004`)는 `env/docker` 의 가상 서버 15대 위에서
돌린다. 준비 절차는 [`env/docker/README.md`](../../../env/docker/README.md) 가 처음부터
끝까지 적어 두었다. 이 문서 6절은 그 위에서 케이스를 돌리는 부분만 다룬다.

macOS 는 유닉스 소켓 경로가 104바이트를 넘으면 끊는다. 노드가 datadir 안에 소켓을 만들어
워크스페이스 경로가 길면 여기 걸린다. 그래서 이 문서는 워크스페이스를 `~/cbw/` 밑 짧은
이름으로 둔다.

---

## 2. 명령 한 줄의 구조

로컬에서 한 건 돌리는 명령은 늘 같은 모양이다.

```sh
rm -rf ~/cbw/m/<이름> && bin/chainbench run <케이스.json> --workspace-dir ~/cbw/m/<이름> --binary "$GSTABLE"
```

네 조각이다.

- `rm -rf ~/cbw/m/<이름>` — **앞선 실행이 남긴 것을 먼저 지운다.** 워크스페이스에 다른
  체인·다른 키로 세운 datadir 이 남아 있으면 다음 실행이 그 genesis 를 만나
  `incompatible genesis` 로 막힌다. 케이스마다 워크스페이스를 따로 두고 돌리기 전에 비운다.
- `bin/chainbench run <케이스.json>` — 무엇을 돌릴지. 저장소 루트 기준 상대 경로.
- `--workspace-dir ~/cbw/m/<이름>` — 이 실행이 세우는 망이 살 자리. 케이스마다 다르게 준다.
- `--binary "$GSTABLE"` — 노드가 돌 실행 파일. 케이스가 어느 체인인지에 따라
  `$GSTABLE`·`$GWBFT`·`$GWEMIX` 중 하나를 준다(3절).

케이스 정의(JSON)의 `chainPreset` 이 노드 몇 대를 어떤 모양으로 세울지, 키는 어디서 올지를
이미 적고 있다. 그래서 명령에는 위 네 조각 말고 더 줄 것이 없다. 예외는 두 가지뿐이다 —
바이너리를 환경변수로 받는 브링업 케이스 둘(3.2)과 Docker 실행(6절)이다.

---

## 3. 어느 바이너리를 주나

### 3.1 대부분은 stablenet

공통 89개 중 이름에 체인이 안 붙은 케이스는 전부 stablenet 프리셋으로 선언돼 있다.
`--binary "$GSTABLE"` 를 준다. 8절 목록에서 `$GSTABLE` 로 찍힌 것이 그것이다.

같은 케이스를 다른 체인에서 보고 싶으면 프리셋을 실행할 때 덮는다. 이때는 그 체인의
바이너리를 준다.

```sh
rm -rf ~/cbw/m/x && bin/chainbench run tests/tc/common/tx/001-value-transfer.json \
  --workspace-dir ~/cbw/m/x --chain-preset wemix-bp4 --binary "$GWEMIX"
```

세 체인이 모두 갖춘 프리셋은 `bp4`·`bp4-en1`·`bp4-en2-pn1`·`bp7-en7-pn1`·`bp9` 다섯이고
`presets/chain/` 에 `<체인>-<모양>.json` 으로 세 벌씩 있다.

### 3.2 이름에 체인이 붙은 8개

이 케이스들은 그 체인 바이너리가 있어야 돈다. stablenet 만 있으면 나머지는 건너뛴다.

| 케이스 | 바이너리 |
| --- | --- |
| `tx/015-wbft-insufficient-funds-rejected` · `contract/001-wbft-tx-and-contract` · `contract/006-wbft-revert-status-zero` · `fault/001-wbft-node-crash` | `$GWBFT` |
| `tx/015-wemix-insufficient-funds-rejected` · `contract/001-wemix-tx-and-contract` · `contract/006-wemix-revert-status-zero` · `fault/001-wemix-node-crash` | `$GWEMIX` |

`node/` 의 파일은 모두 세 체인 공통이다(2026-09-29 에 체인별 사본을 지우고 합쳤다). 다른 체인은
3.1 처럼 `--chain-preset` 과 `--binary` 를 바꿔 돌린다.

### 3.3 로컬에서 안 도는 하나

`fault/004-fault-network-partition` 은 노드가 도는 기계에 방화벽 규칙을 넣어 망을
가른다. 로컬 한 대에서는 그럴 대상이 없어 `requires` 에 `target:remote` 가 붙어 있고,
로컬 실행에서는 **건너뛴다(skip).** Docker 함대(6절)에서만 실제로 돈다.

---

## 4. 판정을 읽는 법

실행이 끝나면 마지막에 요약 한 줄이 나온다.

```
pass=1 fail=0 blocked=0 skip=0
```

네 가지 판정이 있고 둘로 접으면 안 된다.

- **pass** — 케이스가 물은 것에 "그렇다" 가 나왔다.
- **fail** — 물은 것에 "아니다" 가 나왔다. 케이스가 실제로 답했다.
- **skip** — 이 체인·이 대상에서 물을 수 없어 아무 답도 안 했다. 통과가 아니다.
- **blocked** — 망을 세우다 실패해 질문을 걸지도 못했다. 케이스에 대한 답이 아니라 설정 실패다.

종료 코드도 같이 본다. `0` 은 돌았다, `1` 은 케이스가 실패했다, `2` 는 실행 자체가 나아가지
못했다. 세부 기록(노드 로그, 각 단계 결과)은 `~/.chainbench/sessions/` 밑에 실행마다 쌓인다.
실패를 다시 볼 때 여기부터 본다.

---

## 5. 로컬에서 한 건 돌려 보기

가장 단순한 합의 케이스로 한 번 해 본다. 1절의 변수를 잡아 뒀다고 본다.

```sh
rm -rf ~/cbw/m/CT-NODE-009-head-hash-agreement && \
  bin/chainbench run tests/tc/common/node/CT-NODE-009-head-hash-agreement.json \
  --workspace-dir ~/cbw/m/CT-NODE-009-head-hash-agreement --binary "$GSTABLE"
```

노드가 뜨고 블록이 몇 개 진행되면 `pass=1 ...` 로 끝난다. 끝나면 chainbench 가 망을 스스로
내린다. 몇 가지 자주 쓰는 손잡이가 있다.

- `--keep-up` — 끝나고 망을 안 내린다. 노드에 붙어 더 보고 싶을 때. 다 보면
  `bin/chainbench chain stop --workspace-dir ~/cbw/m/CT-NODE-009-head-hash-agreement` 로 내린다.
- `--plan` — 돌리지 않고 무엇을 세울지만 보여 준다.
- `--chain-preset <이름>` — 케이스가 선언한 프리셋을 덮는다(3.1).

한 건이 끝난 워크스페이스는 다음에 그 자리를 다시 쓸 때 명령 앞의 `rm -rf` 가 지운다.
따로 정리할 것은 없다. 8절 목록의 명령이 전부 이 모양이라, 원하는 케이스 줄을 골라
그대로 붙이면 된다.

---

## 6. Docker(원격 흉내)에서 돌리기

로컬에 바이너리가 없거나, 노드가 서로 다른 기계에 있어야 도는 케이스는 `env/docker` 의
가상 서버 15대 위에서 돌린다. **함대 준비(계정 주입, 이미지 빌드, 컨테이너 기동, 각
컨테이너에 리눅스 바이너리 넣기)는 [`env/docker/README.md`](../../../env/docker/README.md) 를
그대로 따른다.** 맥에서 빌드한 바이너리는 컨테이너에서 안 도니, 그 문서의 골랑 컨테이너
빌드 절차로 리눅스 바이너리를 만들어 각 컨테이너의 `/data/chainbench/bin/` 에 넣어야 한다.

준비가 끝났다면 케이스 하나는 이렇게 돈다. 로컬과 다른 점은 `--binary` 를 안 주고(컨테이너
안 바이너리를 이름으로 찾는다) Docker 플래그 넉 줄이 붙는다는 것이다.

```sh
bin/chainbench run \
  --workspace-dir ~/cbw/m/dk \
  --server-set env/docker/build/server-set.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers \
  --keys ~/cbw/m/dk/genkeys --keys-source generate \
  tests/tc/common/node/CT-NODE-002-startup-15-nodes.json
```

go-wemix 는 두 가지가 다르다. poa 노드가 p2p 옆에 포트를 셋씩 잡아 서버 세트가 다르고
(`server-set-wemix.yaml`), 늦게 합류하는 노드 때문에 대기를 늘려야 한다.

```sh
bin/chainbench run \
  --workspace-dir ~/cbw/m/dk \
  --server-set env/docker/build/server-set-wemix.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers \
  --keys ~/cbw/m/dk/genkeys --keys-source generate \
  --node-monitor-timeout 5m \
  --chain-preset wemix-bp7-en7-pn1 \
  tests/tc/common/node/CT-NODE-002-startup-15-nodes.json
```

Docker 는 원격 datadir 을 지워야 다음 실행의 genesis 가 안 막힌다. 로컬처럼 `rm -rf` 만으로는
컨테이너 안이 안 지워지니, 워크스페이스를 지우기 전에 원격을 먼저 내리고 지운다.

```sh
bin/chainbench chain stop --workspace-dir ~/cbw/m/dk   # rm 은 도는 노드를 거부한다
bin/chainbench chain rm   --workspace-dir ~/cbw/m/dk
rm -rf ~/cbw/m/dk
```

---

## 7. 여러 건을 한꺼번에 (스위프)

`scripts/tcsweep.sh` 는 `tests/tc` 밑 케이스를 한 건에 망 하나씩 차례로 돌리고 판정을
모은다. 케이스끼리 상태를 섞지 않으려 일부러 batch 하지 않고 한 건씩 걷는다.

```sh
scripts/tcsweep.sh sweep.log                     # 전부
scripts/tcsweep.sh sweep.log tests/tc/common     # 공통만
scripts/tcsweep.sh sweep.log common/fee          # 이름에 fee 가 든 것만
```

로컬 스위프는 `$GSTABLE` 등 로컬 바이너리로 돈다. Docker 함대로 돌리려면 매 실행에 붙일
플래그를 `TCSWEEP_FLAGS` 로 준다 — 로컬에 바이너리가 없는 기계에서는 이쪽이 유일한 길이다.

```sh
export TCSWEEP_FLAGS="--server-set $PWD/env/docker/build/server-set.yaml \
  --workspace-config $PWD/env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys-source generate"
scripts/tcsweep.sh sweep.log tests/tc/common
```

---

## 8. 자주 막히는 곳

| 증상 | 원인 | 손씀 |
| --- | --- | --- |
| `incompatible genesis` | 워크스페이스(또는 원격 datadir)에 다른 체인·키로 세운 것이 남았다 | 로컬은 `rm -rf <ws>`, Docker 는 `chain stop` → `chain rm` 뒤 `rm -rf <ws>` |
| 소켓 경로가 길다는 오류(macOS) | 워크스페이스 경로가 104바이트를 넘겼다 | `~/cbw/` 밑 짧은 이름으로 옮긴다 |
| `p2p_step must be >= 3` (Docker) | 일반 서버 세트로 go-wemix 를 올렸다 | `server-set-wemix.yaml` 로 바꾼다(6절) |
| 15대 wemix 가 뜨다 blocked | 늦게 합류하는 endpoint 를 게이트가 일찍 끊었다 | `--node-monitor-timeout 5m` 를 준다 |
| `fault/004` 가 skip 으로 끝난다 | `target:remote` 케이스를 로컬에서 돌렸다 | 정상이다. Docker 함대에서 돌린다(3.3) |
| 컨테이너에서 바이너리가 안 돈다 | 맥에서 빌드한 Mach-O 를 넣었다 | 골랑 컨테이너로 리눅스 바이너리를 만들어 넣는다(`env/docker/README.md`) |
| 이름에 `wbft`/`wemix` 가 든 케이스가 skip | 해당 체인 바이너리를 안 줬다 | `$GWBFT`/`$GWEMIX` 를 준다(3.2) |

---

## 9. 케이스별 실행 명령 (89개 전부)

`tests/tc/common/` 아래 89개를 케이스마다 한 줄씩 뽑았다. 1절의 변수(`$GSTABLE`·`$GWBFT`·
`$GWEMIX`)를 잡아 두면 원하는 줄을 그대로 붙여 넣어 돌릴 수 있다. 각 줄은 로컬 실행이다 —
같은 케이스를 Docker 함대에서 돌리려면 6절의 형태에 그 줄의 케이스 경로만 넣는다
(`--binary` 는 빼고 Docker 플래그를 붙인다).

node/001·002·010 은 wbft·wemix 줄도 함께 적었다(같은 파일에 `--chain-preset` 만 바꾼다).
`fault/004-fault-network-partition` 은 `target:remote` 라 로컬에선 skip 되고
Docker 에서만 돈다(3.3).

### node — 노드·동기화·네트워크

```sh
rm -rf ~/cbw/m/CT-NODE-001-startup-block-production && bin/chainbench run tests/tc/common/node/CT-NODE-001-startup-block-production.json --workspace-dir ~/cbw/m/CT-NODE-001-startup-block-production --binary "$GSTABLE"
rm -rf ~/cbw/m/001-chain-up-wbft && bin/chainbench run tests/tc/common/node/CT-NODE-001-startup-block-production.json --workspace-dir ~/cbw/m/001-chain-up-wbft --chain-preset wbft-bp4 --binary "$GWBFT"
rm -rf ~/cbw/m/001-chain-up-wemix && bin/chainbench run tests/tc/common/node/CT-NODE-001-startup-block-production.json --workspace-dir ~/cbw/m/001-chain-up-wemix --chain-preset wemix-bp4 --binary "$GWEMIX"
rm -rf ~/cbw/m/CT-NODE-002-startup-15-nodes && bin/chainbench run tests/tc/common/node/CT-NODE-002-startup-15-nodes.json --workspace-dir ~/cbw/m/CT-NODE-002-startup-15-nodes --binary "$GSTABLE"
rm -rf ~/cbw/m/002-chain-up-15-wbft && bin/chainbench run tests/tc/common/node/CT-NODE-002-startup-15-nodes.json --workspace-dir ~/cbw/m/002-chain-up-15-wbft --chain-preset wbft-bp7-en7-pn1 --binary "$GWBFT"
rm -rf ~/cbw/m/002-chain-up-15-wemix && bin/chainbench run tests/tc/common/node/CT-NODE-002-startup-15-nodes.json --workspace-dir ~/cbw/m/002-chain-up-15-wemix --chain-preset wemix-bp7-en7-pn1 --binary "$GWEMIX"
rm -rf ~/cbw/m/CT-NODE-003-genesis-init && bin/chainbench run tests/tc/common/node/CT-NODE-003-genesis-init.json --workspace-dir ~/cbw/m/CT-NODE-003-genesis-init --binary "$GSTABLE"
rm -rf ~/cbw/m/CT-NODE-004-full-sync && bin/chainbench run tests/tc/common/node/CT-NODE-004-full-sync.json --workspace-dir ~/cbw/m/CT-NODE-004-full-sync --binary "$GSTABLE"
rm -rf ~/cbw/m/CT-NODE-005-snap-sync && bin/chainbench run tests/tc/common/node/CT-NODE-005-snap-sync.json --workspace-dir ~/cbw/m/CT-NODE-005-snap-sync --binary "$GSTABLE"
rm -rf ~/cbw/m/CT-NODE-006-missing-block-catch-up && bin/chainbench run tests/tc/common/node/CT-NODE-006-missing-block-catch-up.json --workspace-dir ~/cbw/m/CT-NODE-006-missing-block-catch-up --binary "$GSTABLE"
rm -rf ~/cbw/m/CT-NODE-007-live-block-receive && bin/chainbench run tests/tc/common/node/CT-NODE-007-live-block-receive.json --workspace-dir ~/cbw/m/CT-NODE-007-live-block-receive --binary "$GSTABLE"
rm -rf ~/cbw/m/CT-NODE-008-peers && bin/chainbench run tests/tc/common/node/CT-NODE-008-peers.json --workspace-dir ~/cbw/m/CT-NODE-008-peers --binary "$GSTABLE"
rm -rf ~/cbw/m/CT-NODE-009-head-hash-agreement && bin/chainbench run tests/tc/common/node/CT-NODE-009-head-hash-agreement.json --workspace-dir ~/cbw/m/CT-NODE-009-head-hash-agreement --binary "$GSTABLE"
rm -rf ~/cbw/m/CT-NODE-010-sync-via-proxy && bin/chainbench run tests/tc/common/node/CT-NODE-010-sync-via-proxy.json --workspace-dir ~/cbw/m/CT-NODE-010-sync-via-proxy --binary "$GSTABLE"
rm -rf ~/cbw/m/010-stablenet-proxied-pn-routing-wbft && bin/chainbench run tests/tc/common/node/CT-NODE-010-sync-via-proxy.json --workspace-dir ~/cbw/m/010-stablenet-proxied-pn-routing-wbft --chain-preset wbft-bp4-en2-pn1 --binary "$GWBFT"
rm -rf ~/cbw/m/010-stablenet-proxied-pn-routing-wemix && bin/chainbench run tests/tc/common/node/CT-NODE-010-sync-via-proxy.json --workspace-dir ~/cbw/m/010-stablenet-proxied-pn-routing-wemix --chain-preset wemix-bp4-en2-pn1 --binary "$GWEMIX"
rm -rf ~/cbw/m/CT-NODE-011-endpoint-first-layout && bin/chainbench run tests/tc/common/node/CT-NODE-011-endpoint-first-layout.json --workspace-dir ~/cbw/m/CT-NODE-011-endpoint-first-layout --binary "$GSTABLE"
rm -rf ~/cbw/m/CT-NODE-013-genesis-mismatch-refused && bin/chainbench run tests/tc/common/node/CT-NODE-013-genesis-mismatch-refused.json --workspace-dir ~/cbw/m/CT-NODE-013-genesis-mismatch-refused --binary "$GSTABLE"
rm -rf ~/cbw/m/CT-NODE-014-sync-complete && bin/chainbench run tests/tc/common/node/CT-NODE-014-sync-complete.json --workspace-dir ~/cbw/m/CT-NODE-014-sync-complete --binary "$GSTABLE"
rm -rf ~/cbw/m/CT-NODE-015-timestamp-monotonic && bin/chainbench run tests/tc/common/node/CT-NODE-015-timestamp-monotonic.json --workspace-dir ~/cbw/m/CT-NODE-015-timestamp-monotonic --binary "$GSTABLE"
rm -rf ~/cbw/m/CT-NODE-016-block-period && bin/chainbench run tests/tc/common/node/CT-NODE-016-block-period.json --workspace-dir ~/cbw/m/CT-NODE-016-block-period --binary "$GSTABLE"
```

### tx — 트랜잭션 전송·거부

```sh
rm -rf ~/cbw/m/001-basic-tx-send && bin/chainbench run tests/tc/common/tx/001-basic-tx-send.json --workspace-dir ~/cbw/m/001-basic-tx-send --binary "$GSTABLE"
rm -rf ~/cbw/m/001-sample-minimal && bin/chainbench run tests/tc/common/tx/001-sample-minimal.json --workspace-dir ~/cbw/m/001-sample-minimal --binary "$GSTABLE"
rm -rf ~/cbw/m/001-value-transfer && bin/chainbench run tests/tc/common/tx/001-value-transfer.json --workspace-dir ~/cbw/m/001-value-transfer --binary "$GSTABLE"
rm -rf ~/cbw/m/002-legacy-value-transfer && bin/chainbench run tests/tc/common/tx/002-legacy-value-transfer.json --workspace-dir ~/cbw/m/002-legacy-value-transfer --binary "$GSTABLE"
rm -rf ~/cbw/m/003-dynamic-fee-transfer && bin/chainbench run tests/tc/common/tx/003-dynamic-fee-transfer.json --workspace-dir ~/cbw/m/003-dynamic-fee-transfer --binary "$GSTABLE"
rm -rf ~/cbw/m/004-access-list-tx && bin/chainbench run tests/tc/common/tx/004-access-list-tx.json --workspace-dir ~/cbw/m/004-access-list-tx --binary "$GSTABLE"
rm -rf ~/cbw/m/005-fee-delegated-transfer && bin/chainbench run tests/tc/common/tx/005-fee-delegated-transfer.json --workspace-dir ~/cbw/m/005-fee-delegated-transfer --binary "$GSTABLE"
rm -rf ~/cbw/m/006-fd-sender-sig-invalid-rejected && bin/chainbench run tests/tc/common/tx/006-fd-sender-sig-invalid-rejected.json --workspace-dir ~/cbw/m/006-fd-sender-sig-invalid-rejected --binary "$GSTABLE"
rm -rf ~/cbw/m/007-fd-feepayer-sig-invalid-rejected && bin/chainbench run tests/tc/common/tx/007-fd-feepayer-sig-invalid-rejected.json --workspace-dir ~/cbw/m/007-fd-feepayer-sig-invalid-rejected --binary "$GSTABLE"
rm -rf ~/cbw/m/008-feepayer-insufficient-rejected && bin/chainbench run tests/tc/common/tx/008-feepayer-insufficient-rejected.json --workspace-dir ~/cbw/m/008-feepayer-insufficient-rejected --binary "$GSTABLE"
rm -rf ~/cbw/m/009-fee-delegate-sign-rpc-present && bin/chainbench run tests/tc/common/tx/009-fee-delegate-sign-rpc-present.json --workspace-dir ~/cbw/m/009-fee-delegate-sign-rpc-present --binary "$GSTABLE"
rm -rf ~/cbw/m/010-fee-delegated-access-list && bin/chainbench run tests/tc/common/tx/010-fee-delegated-access-list.json --workspace-dir ~/cbw/m/010-fee-delegated-access-list --binary "$GSTABLE"
rm -rf ~/cbw/m/011-keystore-fee-delegate-sign && bin/chainbench run tests/tc/common/tx/011-keystore-fee-delegate-sign.json --workspace-dir ~/cbw/m/011-keystore-fee-delegate-sign --binary "$GSTABLE"
rm -rf ~/cbw/m/012-nonce-ordering && bin/chainbench run tests/tc/common/tx/012-nonce-ordering.json --workspace-dir ~/cbw/m/012-nonce-ordering --binary "$GSTABLE"
rm -rf ~/cbw/m/013-replacement-tx && bin/chainbench run tests/tc/common/tx/013-replacement-tx.json --workspace-dir ~/cbw/m/013-replacement-tx --binary "$GSTABLE"
rm -rf ~/cbw/m/014-carry-over-and-replace && bin/chainbench run tests/tc/common/tx/014-carry-over-and-replace.json --workspace-dir ~/cbw/m/014-carry-over-and-replace --binary "$GSTABLE"
rm -rf ~/cbw/m/015-insufficient-funds-rejected && bin/chainbench run tests/tc/common/tx/015-insufficient-funds-rejected.json --workspace-dir ~/cbw/m/015-insufficient-funds-rejected --binary "$GSTABLE"
rm -rf ~/cbw/m/015-wbft-insufficient-funds-rejected && bin/chainbench run tests/tc/common/tx/015-wbft-insufficient-funds-rejected.json --workspace-dir ~/cbw/m/015-wbft-insufficient-funds-rejected --binary "$GWBFT"
rm -rf ~/cbw/m/015-wemix-insufficient-funds-rejected && bin/chainbench run tests/tc/common/tx/015-wemix-insufficient-funds-rejected.json --workspace-dir ~/cbw/m/015-wemix-insufficient-funds-rejected --binary "$GWEMIX"
rm -rf ~/cbw/m/016-gas-limit-exceeds-block-rejected && bin/chainbench run tests/tc/common/tx/016-gas-limit-exceeds-block-rejected.json --workspace-dir ~/cbw/m/016-gas-limit-exceeds-block-rejected --binary "$GSTABLE"
rm -rf ~/cbw/m/017-reject-vs-execution-failure && bin/chainbench run tests/tc/common/tx/017-reject-vs-execution-failure.json --workspace-dir ~/cbw/m/017-reject-vs-execution-failure --binary "$GSTABLE"
rm -rf ~/cbw/m/018-basic-txpool-propagation && bin/chainbench run tests/tc/common/tx/018-basic-txpool-propagation.json --workspace-dir ~/cbw/m/018-basic-txpool-propagation --binary "$GSTABLE"
rm -rf ~/cbw/m/019-stress-tx-flood && bin/chainbench run tests/tc/common/tx/019-stress-tx-flood.json --workspace-dir ~/cbw/m/019-stress-tx-flood --binary "$GSTABLE"
rm -rf ~/cbw/m/020-faucet-funds-account && bin/chainbench run tests/tc/common/tx/020-faucet-funds-account.json --workspace-dir ~/cbw/m/020-faucet-funds-account --binary "$GSTABLE"
```

### fee — 수수료·가스 정책

```sh
rm -rf ~/cbw/m/001-tip-below-min-rejected && bin/chainbench run tests/tc/common/fee/001-tip-below-min-rejected.json --workspace-dir ~/cbw/m/001-tip-below-min-rejected --binary "$GSTABLE"
rm -rf ~/cbw/m/002-accesslist-gasprice-above-min-accepted && bin/chainbench run tests/tc/common/fee/002-accesslist-gasprice-above-min-accepted.json --workspace-dir ~/cbw/m/002-accesslist-gasprice-above-min-accepted --binary "$GSTABLE"
rm -rf ~/cbw/m/002-accesslist-gasprice-below-min-rejected && bin/chainbench run tests/tc/common/fee/002-accesslist-gasprice-below-min-rejected.json --workspace-dir ~/cbw/m/002-accesslist-gasprice-below-min-rejected --binary "$GSTABLE"
rm -rf ~/cbw/m/002-dynamic-feecap-above-min-accepted && bin/chainbench run tests/tc/common/fee/002-dynamic-feecap-above-min-accepted.json --workspace-dir ~/cbw/m/002-dynamic-feecap-above-min-accepted --binary "$GSTABLE"
rm -rf ~/cbw/m/002-feecap-below-min-rejected && bin/chainbench run tests/tc/common/fee/002-feecap-below-min-rejected.json --workspace-dir ~/cbw/m/002-feecap-below-min-rejected --binary "$GSTABLE"
rm -rf ~/cbw/m/002-legacy-gasprice-above-min-accepted && bin/chainbench run tests/tc/common/fee/002-legacy-gasprice-above-min-accepted.json --workspace-dir ~/cbw/m/002-legacy-gasprice-above-min-accepted --binary "$GSTABLE"
rm -rf ~/cbw/m/002-legacy-gasprice-below-min-rejected && bin/chainbench run tests/tc/common/fee/002-legacy-gasprice-below-min-rejected.json --workspace-dir ~/cbw/m/002-legacy-gasprice-below-min-rejected --binary "$GSTABLE"
rm -rf ~/cbw/m/007-effective-gas-price && bin/chainbench run tests/tc/common/fee/007-effective-gas-price.json --workspace-dir ~/cbw/m/007-effective-gas-price --binary "$GSTABLE"
rm -rf ~/cbw/m/008-effective-gas-price-regular-bp-en && bin/chainbench run tests/tc/common/fee/008-effective-gas-price-regular-bp-en.json --workspace-dir ~/cbw/m/008-effective-gas-price-regular-bp-en --binary "$GSTABLE"
rm -rf ~/cbw/m/009-snap-receipt-gas-price && bin/chainbench run tests/tc/common/fee/009-snap-receipt-gas-price.json --workspace-dir ~/cbw/m/009-snap-receipt-gas-price --binary "$GSTABLE"
rm -rf ~/cbw/m/010-gas-price-positive && bin/chainbench run tests/tc/common/fee/010-gas-price-positive.json --workspace-dir ~/cbw/m/010-gas-price-positive --binary "$GSTABLE"
rm -rf ~/cbw/m/012-fee-history-well-formed && bin/chainbench run tests/tc/common/fee/012-fee-history-well-formed.json --workspace-dir ~/cbw/m/012-fee-history-well-formed --binary "$GSTABLE"
```

### contract — 컨트랙트 실행

```sh
rm -rf ~/cbw/m/001-contract-roundtrip && bin/chainbench run tests/tc/common/contract/001-contract-roundtrip.json --workspace-dir ~/cbw/m/001-contract-roundtrip --binary "$GSTABLE"
rm -rf ~/cbw/m/001-wbft-tx-and-contract && bin/chainbench run tests/tc/common/contract/001-wbft-tx-and-contract.json --workspace-dir ~/cbw/m/001-wbft-tx-and-contract --binary "$GWBFT"
rm -rf ~/cbw/m/001-wemix-tx-and-contract && bin/chainbench run tests/tc/common/contract/001-wemix-tx-and-contract.json --workspace-dir ~/cbw/m/001-wemix-tx-and-contract --binary "$GWEMIX"
rm -rf ~/cbw/m/002-storage-write-and-read && bin/chainbench run tests/tc/common/contract/002-storage-write-and-read.json --workspace-dir ~/cbw/m/002-storage-write-and-read --binary "$GSTABLE"
rm -rf ~/cbw/m/003-view-call-leaves-state && bin/chainbench run tests/tc/common/contract/003-view-call-leaves-state.json --workspace-dir ~/cbw/m/003-view-call-leaves-state --binary "$GSTABLE"
rm -rf ~/cbw/m/004-estimate-gas && bin/chainbench run tests/tc/common/contract/004-estimate-gas.json --workspace-dir ~/cbw/m/004-estimate-gas --binary "$GSTABLE"
rm -rf ~/cbw/m/005-eth-call-revert-returns-error && bin/chainbench run tests/tc/common/contract/005-eth-call-revert-returns-error.json --workspace-dir ~/cbw/m/005-eth-call-revert-returns-error --binary "$GSTABLE"
rm -rf ~/cbw/m/006-negative-tx-revert && bin/chainbench run tests/tc/common/contract/006-negative-tx-revert.json --workspace-dir ~/cbw/m/006-negative-tx-revert --binary "$GSTABLE"
rm -rf ~/cbw/m/006-revert-tx-status-zero && bin/chainbench run tests/tc/common/contract/006-revert-tx-status-zero.json --workspace-dir ~/cbw/m/006-revert-tx-status-zero --binary "$GSTABLE"
rm -rf ~/cbw/m/006-wbft-revert-status-zero && bin/chainbench run tests/tc/common/contract/006-wbft-revert-status-zero.json --workspace-dir ~/cbw/m/006-wbft-revert-status-zero --binary "$GWBFT"
rm -rf ~/cbw/m/006-wemix-revert-status-zero && bin/chainbench run tests/tc/common/contract/006-wemix-revert-status-zero.json --workspace-dir ~/cbw/m/006-wemix-revert-status-zero --binary "$GWEMIX"
rm -rf ~/cbw/m/007-out-of-gas-consumes-all && bin/chainbench run tests/tc/common/contract/007-out-of-gas-consumes-all.json --workspace-dir ~/cbw/m/007-out-of-gas-consumes-all --binary "$GSTABLE"
```

### rpc — 조회·구독 API

```sh
rm -rf ~/cbw/m/001-basic-rpc-health && bin/chainbench run tests/tc/common/rpc/001-basic-rpc-health.json --workspace-dir ~/cbw/m/001-basic-rpc-health --binary "$GSTABLE"
rm -rf ~/cbw/m/001-remote-rpc-health && bin/chainbench run tests/tc/common/rpc/001-remote-rpc-health.json --workspace-dir ~/cbw/m/001-remote-rpc-health --binary "$GSTABLE"
rm -rf ~/cbw/m/002-block-transactions-field && bin/chainbench run tests/tc/common/rpc/002-block-transactions-field.json --workspace-dir ~/cbw/m/002-block-transactions-field --binary "$GSTABLE"
rm -rf ~/cbw/m/003-block-by-hash-consistency && bin/chainbench run tests/tc/common/rpc/003-block-by-hash-consistency.json --workspace-dir ~/cbw/m/003-block-by-hash-consistency --binary "$GSTABLE"
rm -rf ~/cbw/m/004-transaction-by-hash-fields && bin/chainbench run tests/tc/common/rpc/004-transaction-by-hash-fields.json --workspace-dir ~/cbw/m/004-transaction-by-hash-fields --binary "$GSTABLE"
rm -rf ~/cbw/m/005-transaction-receipt-fields && bin/chainbench run tests/tc/common/rpc/005-transaction-receipt-fields.json --workspace-dir ~/cbw/m/005-transaction-receipt-fields --binary "$GSTABLE"
rm -rf ~/cbw/m/006-transaction-count-increments && bin/chainbench run tests/tc/common/rpc/006-transaction-count-increments.json --workspace-dir ~/cbw/m/006-transaction-count-increments --binary "$GSTABLE"
rm -rf ~/cbw/m/007-genesis-balance && bin/chainbench run tests/tc/common/rpc/007-genesis-balance.json --workspace-dir ~/cbw/m/007-genesis-balance --binary "$GSTABLE"
rm -rf ~/cbw/m/007-remote-balance-check && bin/chainbench run tests/tc/common/rpc/007-remote-balance-check.json --workspace-dir ~/cbw/m/007-remote-balance-check --binary "$GSTABLE"
rm -rf ~/cbw/m/009-contract-event-emitted && bin/chainbench run tests/tc/common/rpc/009-contract-event-emitted.json --workspace-dir ~/cbw/m/009-contract-event-emitted --binary "$GSTABLE"
rm -rf ~/cbw/m/009-logs-query-well-formed && bin/chainbench run tests/tc/common/rpc/009-logs-query-well-formed.json --workspace-dir ~/cbw/m/009-logs-query-well-formed --binary "$GSTABLE"
rm -rf ~/cbw/m/010-signed-tx-seen-in-pool && bin/chainbench run tests/tc/common/rpc/010-signed-tx-seen-in-pool.json --workspace-dir ~/cbw/m/010-signed-tx-seen-in-pool --binary "$GSTABLE"
rm -rf ~/cbw/m/011-txpool-status && bin/chainbench run tests/tc/common/rpc/011-txpool-status.json --workspace-dir ~/cbw/m/011-txpool-status --binary "$GSTABLE"
rm -rf ~/cbw/m/012-txpool-content-well-formed && bin/chainbench run tests/tc/common/rpc/012-txpool-content-well-formed.json --workspace-dir ~/cbw/m/012-txpool-content-well-formed --binary "$GSTABLE"
rm -rf ~/cbw/m/013-ws-subscribe-new-heads && bin/chainbench run tests/tc/common/rpc/013-ws-subscribe-new-heads.json --workspace-dir ~/cbw/m/013-ws-subscribe-new-heads --binary "$GSTABLE"
rm -rf ~/cbw/m/014-ws-subscribe-logs && bin/chainbench run tests/tc/common/rpc/014-ws-subscribe-logs.json --workspace-dir ~/cbw/m/014-ws-subscribe-logs --binary "$GSTABLE"
rm -rf ~/cbw/m/015-metric-head-block && bin/chainbench run tests/tc/common/rpc/015-metric-head-block.json --workspace-dir ~/cbw/m/015-metric-head-block --binary "$GSTABLE"
```

### fault — 장애·복구

```sh
rm -rf ~/cbw/m/001-fault-node-crash && bin/chainbench run tests/tc/common/fault/001-fault-node-crash.json --workspace-dir ~/cbw/m/001-fault-node-crash --binary "$GSTABLE"
rm -rf ~/cbw/m/001-sample-lifecycle && bin/chainbench run tests/tc/common/fault/001-sample-lifecycle.json --workspace-dir ~/cbw/m/001-sample-lifecycle --binary "$GSTABLE"
rm -rf ~/cbw/m/001-wbft-node-crash && bin/chainbench run tests/tc/common/fault/001-wbft-node-crash.json --workspace-dir ~/cbw/m/001-wbft-node-crash --binary "$GWBFT"
rm -rf ~/cbw/m/001-wemix-node-crash && bin/chainbench run tests/tc/common/fault/001-wemix-node-crash.json --workspace-dir ~/cbw/m/001-wemix-node-crash --binary "$GWEMIX"
rm -rf ~/cbw/m/002-fault-node-recover && bin/chainbench run tests/tc/common/fault/002-fault-node-recover.json --workspace-dir ~/cbw/m/002-fault-node-recover --binary "$GSTABLE"
rm -rf ~/cbw/m/003-fault-two-down && bin/chainbench run tests/tc/common/fault/003-fault-two-down.json --workspace-dir ~/cbw/m/003-fault-two-down --binary "$GSTABLE"
rm -rf ~/cbw/m/004-fault-network-partition && bin/chainbench run tests/tc/common/fault/004-fault-network-partition.json --workspace-dir ~/cbw/m/004-fault-network-partition --binary "$GSTABLE"   # target:remote — 로컬은 skip, docker에서만 돈다
rm -rf ~/cbw/m/005-fault-p2p-topology && bin/chainbench run tests/tc/common/fault/005-fault-p2p-topology.json --workspace-dir ~/cbw/m/005-fault-p2p-topology --binary "$GSTABLE"
rm -rf ~/cbw/m/006-fault-txpool-leader-change && bin/chainbench run tests/tc/common/fault/006-fault-txpool-leader-change.json --workspace-dir ~/cbw/m/006-fault-txpool-leader-change --binary "$GSTABLE"
```
