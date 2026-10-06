# tests/tc — 테스트 돌리는 법

`tests/tc/` 아래 케이스를 자기 손으로 돌려 보게 하는 문서다. 무엇을 보는 케이스인지는
각 디렉터리의 README 가 설명한다. 여기서는 **무엇을 먼저 갖춰야 하는지, 명령이 왜 그렇게
생겼는지, 결과를 어떻게 읽는지**를 적고, 마지막에 케이스마다 복사해 붙일 명령을 모아 둔다.

chainbench 는 블록체인 테스트 망을 세우고 DSL(JSON)로 적힌 케이스를 그 망에 돌린 뒤
통과·실패를 판정하는 CLI 다. 케이스 하나를 돌린다는 것은 (1) 노드 몇 대짜리 망을 새로
세우고 (2) 케이스가 시키는 트랜잭션·조회·장애를 그 망에 걸고 (3) 케이스가 적어 둔
기대값과 실제를 견주는 세 단계를 한 번에 하는 것이다. 명령 한 줄이 이 셋을 다 한다.

---

## 1. 무엇을 먼저 갖춰야 하나

세 가지가 있어야 한다. 셋째(Docker)는 원격을 흉내 내 돌릴 때만 필요하다.

**chainbench 바이너리.** 저장소 루트에서 `make build` 를 하면 `bin/chainbench` 가 생긴다.
이 문서의 모든 명령은 저장소 루트에서 상대 경로 `bin/chainbench` 로 부른다.

```sh
cd <이 저장소>          # 예: cd ~/Work/github/chainbench
make build
bin/chainbench --help   # 여기까지 되면 준비 끝
```

**체인 노드 바이너리.** 케이스가 세우는 망의 노드가 실제로 돌 실행 파일이다. 저장소가
아니라 각 체인 소스에서 따로 빌드한다.

| 체인 | 빌드 산출물 | 이 문서에서 부르는 이름 |
| --- | --- | --- |
| go-stablenet | `build/bin/gstable` | `$GSTABLE` |
| go-wbft | `build/bin/gwemix` (`make gwemix` — 저장소가 go-wemix 에서 갈라져 이름이 남았다) | `$GWBFT` |
| go-wemix | `build/bin/gwemix` | `$GWEMIX` |

세 경로를 한 번 변수로 잡아 두면 5절의 명령을 그대로 붙여 쓸 수 있다. 자기 기계의 실제
경로로 바꾼다.

```sh
export GSTABLE="$HOME/work/github/chain/go-stablenet/build/bin/gstable"
export GWBFT="$HOME/work/github/chain/go-wbft/build/bin/gwemix"
export GWEMIX="$HOME/work/github/chain/go-wemix/build/bin/gwemix"
```

**셋 다 없어도 좋다.** 197건 중 175건이 stablenet 하나로 돈다. wbft 8건, wemix 7건만 그
체인의 빌드를 요구한다.

**(원격을 흉내 낼 때만) Docker 함대.** 로컬 바이너리가 없는 기계, 또는 노드가 서로 다른
기계에 있어야 도는 케이스는 `env/docker` 의 가상 서버 15대 위에서 돌린다. 준비는
[`env/docker/README.md`](../../env/docker/README.md) 가 처음부터 끝까지 적는다
(`./setup.sh` 한 줄로도 된다). 이 문서 4절은 그 위에서 케이스를 돌리는 부분만 다룬다.

**워크스페이스는 짧은 경로에 둔다.** macOS 는 유닉스 소켓 경로가 104바이트를 넘으면 끊는다.
노드가 datadir 안에 소켓을 만들므로 워크스페이스 경로가 길면 여기 걸린다. 이 문서는
`~/cbw/` 밑 짧은 이름을 쓴다.

---

## 2. 명령 한 줄의 구조

로컬에서 한 건 돌리는 명령은 늘 같은 모양이다.

```sh
bin/chainbench run <케이스.json> --workspace-dir ~/cbw/one/<이름> --binary "$GSTABLE"
```

세 조각이다.

- `bin/chainbench run <케이스.json>` — 무엇을 돌릴지. 저장소 루트 기준 상대 경로.
- `--workspace-dir ~/cbw/one/<이름>` — 이 실행이 세우는 망이 살 자리. 케이스마다 다르게 준다.
- `--binary "$GSTABLE"` — 노드가 돌 실행 파일. 케이스가 어느 체인인지에 따라 셋 중 하나(3절).

케이스 정의(JSON)의 `chainPreset` 이 노드 몇 대를 어떤 모양으로 세울지, 키는 어디서 올지를
이미 적고 있다. 그래서 더 줄 것이 없다. 예외는 3절의 세 갈래뿐이다.

**같은 워크스페이스를 다시 써도 된다.** preflight 가 같은 망에 노드만 멈춰 있는 것을
`relaunch` 로 보고 설정 단계를 건너뛴다. 다만 **다른 체인이나 다른 키로 세웠던 자리**라면
`incompatible genesis` 로 막히므로 그때는 지운다.

```sh
rm -rf ~/cbw/one/<이름>
```

---

## 3. 케이스가 갈리는 세 갈래

### 3.1 어느 바이너리를 주나

대부분은 stablenet 프리셋으로 선언돼 있다. 같은 케이스를 다른 체인에서 보려면 실행할 때
프리셋을 덮고 그 체인의 바이너리를 준다.

```sh
bin/chainbench run tests/tc/common/tx/CT-TX-001-value-transfer.json \
  --workspace-dir ~/cbw/one/x --chain-preset wemix-bp4 --binary "$GWEMIX"
```

세 체인이 모두 갖춘 프리셋은 `bp4`·`bp4-en1`·`bp4-en2-pn1`·`bp7-en7-pn1`·`bp9` 다섯이고
`presets/chain/` 에 `<체인>-<모양>.json` 으로 세 벌씩 있다. 체인을 세우는 네 갈래는
[`CHAIN-BRINGUP.md`](CHAIN-BRINGUP.md) 가 적는다.

### 3.2 환경변수를 더 받는 여섯

프리셋이 `${VAR}` 로 적어 둔 값이다. 안 걸면 실행이 멈추고 무엇을 걸라고 말한다.

| 케이스 | 변수 | 무엇 |
| --- | --- | --- |
| `basic/08-attached-chain-produces` | `GSTABLE_RPC` | 이미 떠 있는 망의 주소 |
| `go-stablenet/testnet/01`~`04` | `GSTABLE_TESTNET_RPC` | 실제 testnet 주소. 기본값 없음 |
| `go-stablenet/testnet/05` | 위 + `GSTABLE_TESTNET_KEY_FILE` | 자금을 쥔 계정의 평문 hex 키 파일 |
| `go-stablenet/.../boho-crossed-by-restart` | `GSTABLE_POSTFORK_BIN` | boho 이후 빌드. 안 걸면 대상의 `gstable-postfork` 를 찾는다 |
| `go-stablenet/.../signature-compat-across-swap` | `GSTABLE_UPGRADE_BIN` | 바꿔 낄 다른 빌드. 안 걸면 대상의 `gstable-hardfork` 를 찾는다 |

boho 이전 빌드는 원본 저장소를 건드리지 않게 따로 복제해 만든다. `ad0122af0` 은 boho 를
넣은 커밋의 부모다.

```sh
git clone -q --shared <go-stablenet> ~/cbw/gs-prefork
git -C ~/cbw/gs-prefork checkout -q ad0122af0 && make -C ~/cbw/gs-prefork gstable
strings ~/cbw/gs-prefork/build/bin/gstable | grep -ci bohoblock   # 0 이어야 한다
```

**두 케이스는 변수를 안 걸면 돌지 않는다.** 선언이 두 번째 빌드를 각각
`gstable-hardfork` 와 `gstable-postfork` 라는 **다른 이름**으로 부르기 때문이다. 이름이
곧 대상의 경로이므로, 예전처럼 둘 다 `gstable` 로 떨어지면 같은 파일을 바꿔 끼우고도
바꾼 것처럼 보인다. 그래서 이름을 갈랐다.

빌드를 그 이름으로 대상에 올려 두거나, 변수를 이미 있는 빌드로 가리키면 된다. 둘 다
안 하면 **망을 세우기 전에 BLOCKED 로 멈추고** 어느 선언의 어느 경로가 비었는지 적는다.

```
binaries.upgrade /data/chainbench/bin/gstable-hardfork: not on the target on server4
build that binary and place it on the target under the name the declaration uses,
or point the declaration's variable at a build that is already there
```

### 3.3 로컬에서 안 도는 하나

`common/fault/CT-FAULT-004-network-partition` 은 노드가 도는 기계에 방화벽 규칙을 넣어 망을
가른다. 로컬 한 대에서는 그럴 대상이 없어 `requires` 에 `target:remote` 가 붙어 있고,
로컬 실행에서는 **건너뛴다(skip).** Docker 함대(4절)에서만 실제로 돈다.

---

## 4. Docker(원격 흉내)에서 돌리기

함대 준비가 끝났다면 케이스 하나는 이렇게 돈다. 로컬과 다른 점은 `--binary` 를 안 주고
(컨테이너 안 바이너리를 이름으로 찾는다) Docker 플래그가 붙는다는 것이다.

```sh
bin/chainbench run tests/tc/common/node/CT-NODE-002-startup-15-nodes.json \
  --workspace-dir ~/cbw/dk \
  --server-set env/docker/build/server-set.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys-source generate
```

go-wemix 는 두 가지가 다르다. poa 노드가 p2p 옆에 포트를 셋씩 잡아 서버 세트가 다르고
(`server-set-wemix.yaml`), 늦게 합류하는 노드 때문에 대기를 늘려야 한다.

```sh
bin/chainbench run <wemix 케이스> \
  --workspace-dir ~/cbw/dk \
  --server-set env/docker/build/server-set-wemix.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys-source generate --node-monitor-timeout 5m
```

Docker 는 **원격 datadir 을 따로 지워야** 다음 실행의 genesis 가 안 막힌다. 로컬의
`rm -rf` 만으로는 컨테이너 안이 안 지워진다.

```sh
bin/chainbench chain stop --workspace-dir ~/cbw/dk   # rm 은 도는 노드를 거부한다
bin/chainbench chain rm   --workspace-dir ~/cbw/dk
rm -rf ~/cbw/dk
```

### 여러 건을 한꺼번에 (스위프)

`scripts/tcsweep.sh <out.log> [패턴]` 은 케이스마다 망 하나씩 차례로 돌리고 판정을 모은다.
같은 chain-preset 이어도 한 케이스가 바꾼 상태가 다음 판정을 바꾸므로 일부러 묶지 않는다.

```sh
scripts/tcsweep.sh ~/cbw/sweep.log tests/tc/common     # 공통만
scripts/tcsweep.sh ~/cbw/sweep.log CT-FEE              # 이름에 CT-FEE 가 든 것만
```

**전량 스위프는 세 번에 나눈다.** 체인마다 서버 세트가 다르고 `TCSWEEP_FLAGS` 는 전 케이스에
똑같이 적용되기 때문이다. 명령은 [`env/docker/README.md`](../../env/docker/README.md) 에 있다.

---

## 5. 판정을 읽는 법

실행이 끝나면 요약 한 줄이 나온다.

```
pass=1 fail=0 blocked=0 skip=0
session: ~/.chainbench/sessions/<시각>/UTC-<시각>
```

네 가지 판정이 있고 둘로 접으면 안 된다.

- **pass** — 케이스가 물은 것에 "그렇다" 가 나왔다.
- **fail** — 물은 것에 "아니다" 가 나왔다. 케이스가 실제로 답했다.
- **skip** — 이 체인·이 대상에서 물을 수 없어 아무 답도 안 했다. **통과가 아니다.**
- **blocked** — 망을 세우다 실패해 질문을 걸지도 못했다. 케이스에 대한 답이 아니라 설정 실패다.

종료 코드도 같이 본다. `0` 은 돌았다, `1` 은 케이스가 실패했다, `2` 는 실행 자체가 나아가지
못했다.

`session:` 경로에 리포트와 실패 시점에 모은 노드 로그가 남는다. **워크스페이스를 지워도 이
기록은 남는다.** 실패를 다시 볼 때 여기부터 본다.

```sh
S=$(ls -dt ~/.chainbench/sessions/*/ | head -1)
cat $S/UTC-*/tests/*/status.json      # 실패 사유 한 줄 — 가장 쓸모 있다
cat $S/UTC-*/tests/*/steps.json       # 단계별 결과와 오류
```

명령이 끝나면 노드는 내려가 있다. 망을 띄운 채 더 보려면 `--keep-up` 을 붙이고, 다 본 뒤
`bin/chainbench chain stop --workspace-dir <ws>` 로 내린다. 무엇이 세워질지만 보려면
`--plan` 을 붙인다 — 망을 만들지 않는다.

---

## 6. 자주 막히는 곳

| 증상 | 원인 | 손씀 |
| --- | --- | --- |
| `incompatible genesis` | 워크스페이스(또는 원격 datadir)에 다른 체인·키로 세운 것이 남았다 | 로컬은 `rm -rf <ws>`, Docker 는 `chain stop` → `chain rm` 뒤 `rm -rf <ws>` |
| 소켓 경로가 길다는 오류(macOS) | 워크스페이스 경로가 104바이트를 넘겼다 | `~/cbw/` 밑 짧은 이름으로 옮긴다 |
| `p2p_step must be >= 3` (Docker) | 일반 서버 세트로 go-wemix 를 올렸다 | `server-set-wemix.yaml` 로 바꾼다(4절) |
| 15대 wemix 가 뜨다 blocked | 늦게 합류하는 endpoint 를 게이트가 일찍 끊었다 | `--node-monitor-timeout 5m` 를 준다 |
| `CT-FAULT-004` 가 skip 으로 끝난다 | `target:remote` 케이스를 로컬에서 돌렸다 | 정상이다. Docker 함대에서 돌린다(3.3) |
| 컨테이너에서 바이너리가 안 돈다 | 맥에서 빌드한 Mach-O 를 넣었다 | 골랑 컨테이너로 리눅스 바이너리를 만들어 넣는다(`env/docker/README.md`) |
| 스위프가 전부 NOTRUN | `TCSWEEP_FLAGS` 없이 Docker 를 기대했다 | 플래그를 주거나 로컬 바이너리를 PATH 에 둔다 |
| attach 케이스가 "RPC URL is empty" | 3.2 의 환경변수를 안 걸었다 | 그 변수를 건다 |

---

## 7. 케이스별 실행 명령

아래 목록은 **손으로 쓰지 않는다.** `scripts/gen-case-commands.py` 가 트리에서 만든다.
케이스가 늘거나 줄면 다시 생성한다 — 이 저장소는 손으로 적은 목록이 "209건" 에서 멈춘 채
실제가 197건이 된 것을 한 번 겪었다.

```sh
python3 scripts/gen-case-commands.py --write
```

<!-- BEGIN generated: scripts/gen-case-commands.py -->

케이스 **197건**이다. 각 케이스에 두 갈래를 적는다 — `bin/chainbench` 로 한 건만 돌리는 것과, `scripts/tcsweep.sh` 로 같은 한 건을 돌리는 것이다. 스크립트 쪽은 망을 세우고 내리고 지우는 것까지 하고 판정 한 줄을 남긴다.

### `basic` — 2건

**07-basic-wbft-consensus.json** · `basic-wbft-consensus`
```sh
bin/chainbench run tests/tc/basic/07-basic-wbft-consensus.json \
  --workspace-dir ~/cbw/one/07-basic-wbft-consensus --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '07-basic-wbft-consensus'
```

**08-attached-chain-produces.json** · `attached-chain-produces`
```sh
GSTABLE_RPC=<값> bin/chainbench run tests/tc/basic/08-attached-chain-produces.json
```
> 이미 떠 있는 망에 붙는다. 망을 세우지 않으므로 `--workspace-dir` 도 `--binary` 도 주지 않는다. `tcsweep.sh` 는 이 갈래를 자기가 세운 망에 붙여 돌린다.

### `common/contract` — 7건

**CT-CONTRACT-001-deploy-and-call.json** · `contract-roundtrip`
```sh
bin/chainbench run tests/tc/common/contract/CT-CONTRACT-001-deploy-and-call.json \
  --workspace-dir ~/cbw/one/CT-CONTRACT-001-deploy-and-call --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-CONTRACT-001-deploy-and-call'
```

**CT-CONTRACT-002-storage-write-and-read.json** · `storage-write-and-read`
```sh
bin/chainbench run tests/tc/common/contract/CT-CONTRACT-002-storage-write-and-read.json \
  --workspace-dir ~/cbw/one/CT-CONTRACT-002-storage-write-and-read --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-CONTRACT-002-storage-write-and-read'
```

**CT-CONTRACT-003-view-call-leaves-state.json** · `view-call-leaves-state`
```sh
bin/chainbench run tests/tc/common/contract/CT-CONTRACT-003-view-call-leaves-state.json \
  --workspace-dir ~/cbw/one/CT-CONTRACT-003-view-call-leaves-state --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-CONTRACT-003-view-call-leaves-state'
```

**CT-CONTRACT-004-estimate-gas.json** · `estimate-gas`
```sh
bin/chainbench run tests/tc/common/contract/CT-CONTRACT-004-estimate-gas.json \
  --workspace-dir ~/cbw/one/CT-CONTRACT-004-estimate-gas --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-CONTRACT-004-estimate-gas'
```

**CT-CONTRACT-005-eth-call-revert-returns-error.json** · `eth-call-revert-returns-error`
```sh
bin/chainbench run tests/tc/common/contract/CT-CONTRACT-005-eth-call-revert-returns-error.json \
  --workspace-dir ~/cbw/one/CT-CONTRACT-005-eth-call-revert-returns-error --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-CONTRACT-005-eth-call-revert-returns-error'
```

**CT-CONTRACT-006-revert-tx-status-zero.json** · `revert-tx-status-zero`
```sh
bin/chainbench run tests/tc/common/contract/CT-CONTRACT-006-revert-tx-status-zero.json \
  --workspace-dir ~/cbw/one/CT-CONTRACT-006-revert-tx-status-zero --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-CONTRACT-006-revert-tx-status-zero'
```

**CT-CONTRACT-007-out-of-gas-consumes-all.json** · `out-of-gas-consumes-all`
```sh
bin/chainbench run tests/tc/common/contract/CT-CONTRACT-007-out-of-gas-consumes-all.json \
  --workspace-dir ~/cbw/one/CT-CONTRACT-007-out-of-gas-consumes-all --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-CONTRACT-007-out-of-gas-consumes-all'
```

### `common/fault` — 6건

**CT-FAULT-001-producer-crash-and-restart.json** · `fault-node-crash`
```sh
bin/chainbench run tests/tc/common/fault/CT-FAULT-001-producer-crash-and-restart.json \
  --workspace-dir ~/cbw/one/CT-FAULT-001-producer-crash-and-restart --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-FAULT-001-producer-crash-and-restart'
```

**CT-FAULT-002-node-recover-and-sync.json** · `fault-node-recover`
```sh
bin/chainbench run tests/tc/common/fault/CT-FAULT-002-node-recover-and-sync.json \
  --workspace-dir ~/cbw/one/CT-FAULT-002-node-recover-and-sync --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-FAULT-002-node-recover-and-sync'
```

**CT-FAULT-003-two-producers-down.json** · `fault-two-down`
```sh
bin/chainbench run tests/tc/common/fault/CT-FAULT-003-two-producers-down.json \
  --workspace-dir ~/cbw/one/CT-FAULT-003-two-producers-down --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-FAULT-003-two-producers-down'
```

**CT-FAULT-004-network-partition.json** · `fault-network-partition`
```sh
# 로컬에서는 건너뛴다 — 노드가 도는 기계의 셸이 필요하다
bin/chainbench run tests/tc/common/fault/CT-FAULT-004-network-partition.json \
  --workspace-dir ~/cbw/one/CT-FAULT-004-network-partition --server-set env/docker/build/server-set.yaml --workspace-config env/docker/build/workspace-config.yaml --docker --all-servers --keys-source generate
```

**CT-FAULT-005-hub-topology.json** · `fault-p2p-topology`
```sh
bin/chainbench run tests/tc/common/fault/CT-FAULT-005-hub-topology.json \
  --workspace-dir ~/cbw/one/CT-FAULT-005-hub-topology --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-FAULT-005-hub-topology'
```

**CT-FAULT-006-txpool-leader-change.json** · `fault-txpool-leader-change`
```sh
bin/chainbench run tests/tc/common/fault/CT-FAULT-006-txpool-leader-change.json \
  --workspace-dir ~/cbw/one/CT-FAULT-006-txpool-leader-change --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-FAULT-006-txpool-leader-change'
```

### `common/fee` — 7건

**CT-FEE-001-tip-below-min-rejected.json** · `tip-below-min-rejected`
```sh
bin/chainbench run tests/tc/common/fee/CT-FEE-001-tip-below-min-rejected.json \
  --workspace-dir ~/cbw/one/CT-FEE-001-tip-below-min-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-FEE-001-tip-below-min-rejected'
```

**CT-FEE-002-min-gas-price-boundary.json** · `legacy-gasprice-below-min-rejected`
```sh
bin/chainbench run tests/tc/common/fee/CT-FEE-002-min-gas-price-boundary.json \
  --workspace-dir ~/cbw/one/CT-FEE-002-min-gas-price-boundary --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-FEE-002-min-gas-price-boundary'
```

**CT-FEE-007-effective-gas-price.json** · `effective-gas-price`
```sh
bin/chainbench run tests/tc/common/fee/CT-FEE-007-effective-gas-price.json \
  --workspace-dir ~/cbw/one/CT-FEE-007-effective-gas-price --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-FEE-007-effective-gas-price'
```

**CT-FEE-008-effective-gas-price-across-nodes.json** · `effective-gas-price-regular-bp-en`
```sh
bin/chainbench run tests/tc/common/fee/CT-FEE-008-effective-gas-price-across-nodes.json \
  --workspace-dir ~/cbw/one/CT-FEE-008-effective-gas-price-across-nodes --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-FEE-008-effective-gas-price-across-nodes'
```

**CT-FEE-009-snap-receipt-gas-price.json** · `snap-receipt-gas-price`
```sh
bin/chainbench run tests/tc/common/fee/CT-FEE-009-snap-receipt-gas-price.json \
  --workspace-dir ~/cbw/one/CT-FEE-009-snap-receipt-gas-price --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-FEE-009-snap-receipt-gas-price'
```

**CT-FEE-010-suggested-gas-price.json** · `gas-price-positive`
```sh
bin/chainbench run tests/tc/common/fee/CT-FEE-010-suggested-gas-price.json \
  --workspace-dir ~/cbw/one/CT-FEE-010-suggested-gas-price --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-FEE-010-suggested-gas-price'
```

**CT-FEE-012-fee-history.json** · `fee-history-well-formed`
```sh
bin/chainbench run tests/tc/common/fee/CT-FEE-012-fee-history.json \
  --workspace-dir ~/cbw/one/CT-FEE-012-fee-history --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-FEE-012-fee-history'
```

### `common/node` — 15건

**CT-NODE-001-startup-block-production.json** · `stablenet-chain-up`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-001-startup-block-production.json \
  --workspace-dir ~/cbw/one/CT-NODE-001-startup-block-production --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-001-startup-block-production'
```

**CT-NODE-002-startup-15-nodes.json** · `stablenet-chain-up-15`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-002-startup-15-nodes.json \
  --workspace-dir ~/cbw/one/CT-NODE-002-startup-15-nodes --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-002-startup-15-nodes'
```

**CT-NODE-003-genesis-init.json** · `genesis-block-hash-consistent`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-003-genesis-init.json \
  --workspace-dir ~/cbw/one/CT-NODE-003-genesis-init --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-003-genesis-init'
```

**CT-NODE-004-full-sync.json** · `full-sync`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-004-full-sync.json \
  --workspace-dir ~/cbw/one/CT-NODE-004-full-sync --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-004-full-sync'
```

**CT-NODE-005-snap-sync.json** · `snap-sync`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-005-snap-sync.json \
  --workspace-dir ~/cbw/one/CT-NODE-005-snap-sync --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-005-snap-sync'
```

**CT-NODE-006-missing-block-catch-up.json** · `downloader-catch-up`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-006-missing-block-catch-up.json \
  --workspace-dir ~/cbw/one/CT-NODE-006-missing-block-catch-up --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-006-missing-block-catch-up'
```

**CT-NODE-007-live-block-receive.json** · `live-block-receive`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-007-live-block-receive.json \
  --workspace-dir ~/cbw/one/CT-NODE-007-live-block-receive --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-007-live-block-receive'
```

**CT-NODE-008-peers.json** · `basic-peers`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-008-peers.json \
  --workspace-dir ~/cbw/one/CT-NODE-008-peers --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-008-peers'
```

**CT-NODE-009-head-hash-agreement.json** · `basic-consensus`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-009-head-hash-agreement.json \
  --workspace-dir ~/cbw/one/CT-NODE-009-head-hash-agreement --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-009-head-hash-agreement'
```

**CT-NODE-010-sync-via-proxy.json** · `stablenet-proxied-pn-routing`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-010-sync-via-proxy.json \
  --workspace-dir ~/cbw/one/CT-NODE-010-sync-via-proxy --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-010-sync-via-proxy'
```

**CT-NODE-011-endpoint-first-layout.json** · `e1-mixed-producers`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-011-endpoint-first-layout.json \
  --workspace-dir ~/cbw/one/CT-NODE-011-endpoint-first-layout --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-011-endpoint-first-layout'
```

**CT-NODE-013-genesis-mismatch-refused.json** · `genesis-mismatch-refuses-to-start`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-013-genesis-mismatch-refused.json \
  --workspace-dir ~/cbw/one/CT-NODE-013-genesis-mismatch-refused --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-013-genesis-mismatch-refused'
```

**CT-NODE-014-sync-complete.json** · `chain-not-syncing`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-014-sync-complete.json \
  --workspace-dir ~/cbw/one/CT-NODE-014-sync-complete --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-014-sync-complete'
```

**CT-NODE-015-timestamp-monotonic.json** · `block-timestamp-monotonic`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-015-timestamp-monotonic.json \
  --workspace-dir ~/cbw/one/CT-NODE-015-timestamp-monotonic --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-015-timestamp-monotonic'
```

**CT-NODE-016-block-period.json** · `block-period-one-second`
```sh
bin/chainbench run tests/tc/common/node/CT-NODE-016-block-period.json \
  --workspace-dir ~/cbw/one/CT-NODE-016-block-period --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-NODE-016-block-period'
```

### `common/rpc` — 14건

**CT-RPC-001-block-number-advances.json** · `basic-rpc-health`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-001-block-number-advances.json \
  --workspace-dir ~/cbw/one/CT-RPC-001-block-number-advances --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-001-block-number-advances'
```

**CT-RPC-002-block-transactions-field.json** · `block-transactions-field`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-002-block-transactions-field.json \
  --workspace-dir ~/cbw/one/CT-RPC-002-block-transactions-field --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-002-block-transactions-field'
```

**CT-RPC-003-block-by-hash-consistency.json** · `block-by-hash-consistency`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-003-block-by-hash-consistency.json \
  --workspace-dir ~/cbw/one/CT-RPC-003-block-by-hash-consistency --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-003-block-by-hash-consistency'
```

**CT-RPC-004-transaction-by-hash-fields.json** · `transaction-by-hash-fields`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-004-transaction-by-hash-fields.json \
  --workspace-dir ~/cbw/one/CT-RPC-004-transaction-by-hash-fields --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-004-transaction-by-hash-fields'
```

**CT-RPC-005-transaction-receipt-fields.json** · `transaction-receipt-fields`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-005-transaction-receipt-fields.json \
  --workspace-dir ~/cbw/one/CT-RPC-005-transaction-receipt-fields --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-005-transaction-receipt-fields'
```

**CT-RPC-006-transaction-count-increments.json** · `transaction-count-increments`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-006-transaction-count-increments.json \
  --workspace-dir ~/cbw/one/CT-RPC-006-transaction-count-increments --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-006-transaction-count-increments'
```

**CT-RPC-007-balance-query.json** · `genesis-balance`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-007-balance-query.json \
  --workspace-dir ~/cbw/one/CT-RPC-007-balance-query --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-007-balance-query'
```

**CT-RPC-009-event-logs-query.json** · `contract-event-emitted`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-009-event-logs-query.json \
  --workspace-dir ~/cbw/one/CT-RPC-009-event-logs-query --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-009-event-logs-query'
```

**CT-RPC-010-signed-tx-seen-in-pool.json** · `signed-tx-seen-in-pool`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-010-signed-tx-seen-in-pool.json \
  --workspace-dir ~/cbw/one/CT-RPC-010-signed-tx-seen-in-pool --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-010-signed-tx-seen-in-pool'
```

**CT-RPC-011-txpool-status.json** · `txpool-status`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-011-txpool-status.json \
  --workspace-dir ~/cbw/one/CT-RPC-011-txpool-status --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-011-txpool-status'
```

**CT-RPC-012-txpool-content-well-formed.json** · `txpool-content-well-formed`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-012-txpool-content-well-formed.json \
  --workspace-dir ~/cbw/one/CT-RPC-012-txpool-content-well-formed --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-012-txpool-content-well-formed'
```

**CT-RPC-013-ws-subscribe-new-heads.json** · `ws-subscribe-new-heads`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-013-ws-subscribe-new-heads.json \
  --workspace-dir ~/cbw/one/CT-RPC-013-ws-subscribe-new-heads --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-013-ws-subscribe-new-heads'
```

**CT-RPC-014-ws-subscribe-logs.json** · `ws-subscribe-logs`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-014-ws-subscribe-logs.json \
  --workspace-dir ~/cbw/one/CT-RPC-014-ws-subscribe-logs --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-014-ws-subscribe-logs'
```

**CT-RPC-015-metric-head-block.json** · `stablenet-metric-head-block`
```sh
bin/chainbench run tests/tc/common/rpc/CT-RPC-015-metric-head-block.json \
  --workspace-dir ~/cbw/one/CT-RPC-015-metric-head-block --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-RPC-015-metric-head-block'
```

### `common/tx` — 20건

**CT-TX-001-value-transfer.json** · `basic-tx-send`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-001-value-transfer.json \
  --workspace-dir ~/cbw/one/CT-TX-001-value-transfer --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-001-value-transfer'
```

**CT-TX-002-legacy-transfer.json** · `legacy-value-transfer`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-002-legacy-transfer.json \
  --workspace-dir ~/cbw/one/CT-TX-002-legacy-transfer --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-002-legacy-transfer'
```

**CT-TX-003-dynamic-fee-transfer.json** · `dynamic-fee-transfer`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-003-dynamic-fee-transfer.json \
  --workspace-dir ~/cbw/one/CT-TX-003-dynamic-fee-transfer --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-003-dynamic-fee-transfer'
```

**CT-TX-004-access-list-tx.json** · `access-list-tx`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-004-access-list-tx.json \
  --workspace-dir ~/cbw/one/CT-TX-004-access-list-tx --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-004-access-list-tx'
```

**CT-TX-005-fee-delegated-transfer.json** · `fee-delegated-transfer`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-005-fee-delegated-transfer.json \
  --workspace-dir ~/cbw/one/CT-TX-005-fee-delegated-transfer --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-005-fee-delegated-transfer'
```

**CT-TX-006-fd-sender-sig-tampered-rejected.json** · `fd-sender-sig-invalid-rejected`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-006-fd-sender-sig-tampered-rejected.json \
  --workspace-dir ~/cbw/one/CT-TX-006-fd-sender-sig-tampered-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-006-fd-sender-sig-tampered-rejected'
```

**CT-TX-007-fd-feepayer-sig-tampered-rejected.json** · `fd-feepayer-sig-invalid-rejected`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-007-fd-feepayer-sig-tampered-rejected.json \
  --workspace-dir ~/cbw/one/CT-TX-007-fd-feepayer-sig-tampered-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-007-fd-feepayer-sig-tampered-rejected'
```

**CT-TX-008-feepayer-insufficient-rejected.json** · `feepayer-insufficient-rejected`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-008-feepayer-insufficient-rejected.json \
  --workspace-dir ~/cbw/one/CT-TX-008-feepayer-insufficient-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-008-feepayer-insufficient-rejected'
```

**CT-TX-009-fee-delegate-sign-rpc-present.json** · `fee-delegate-sign-rpc-present`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-009-fee-delegate-sign-rpc-present.json \
  --workspace-dir ~/cbw/one/CT-TX-009-fee-delegate-sign-rpc-present --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-009-fee-delegate-sign-rpc-present'
```

**CT-TX-010-fee-delegated-access-list.json** · `fee-delegated-access-list`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-010-fee-delegated-access-list.json \
  --workspace-dir ~/cbw/one/CT-TX-010-fee-delegated-access-list --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-010-fee-delegated-access-list'
```

**CT-TX-011-keystore-fee-delegate-sign.json** · `keystore-fee-delegate-sign`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-011-keystore-fee-delegate-sign.json \
  --workspace-dir ~/cbw/one/CT-TX-011-keystore-fee-delegate-sign --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-011-keystore-fee-delegate-sign'
```

**CT-TX-012-nonce-ordering.json** · `nonce-ordering`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-012-nonce-ordering.json \
  --workspace-dir ~/cbw/one/CT-TX-012-nonce-ordering --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-012-nonce-ordering'
```

**CT-TX-013-same-nonce-replacement.json** · `replacement-tx`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-013-same-nonce-replacement.json \
  --workspace-dir ~/cbw/one/CT-TX-013-same-nonce-replacement --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-013-same-nonce-replacement'
```

**CT-TX-014-carry-over-and-replace.json** · `carry-over-and-replace`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-014-carry-over-and-replace.json \
  --workspace-dir ~/cbw/one/CT-TX-014-carry-over-and-replace --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-014-carry-over-and-replace'
```

**CT-TX-015-insufficient-funds-rejected.json** · `insufficient-funds-rejected`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-015-insufficient-funds-rejected.json \
  --workspace-dir ~/cbw/one/CT-TX-015-insufficient-funds-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-015-insufficient-funds-rejected'
```

**CT-TX-016-gas-limit-exceeds-block-rejected.json** · `gas-limit-exceeds-block-rejected`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-016-gas-limit-exceeds-block-rejected.json \
  --workspace-dir ~/cbw/one/CT-TX-016-gas-limit-exceeds-block-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-016-gas-limit-exceeds-block-rejected'
```

**CT-TX-017-reject-vs-execution-failure.json** · `reject-vs-execution-failure`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-017-reject-vs-execution-failure.json \
  --workspace-dir ~/cbw/one/CT-TX-017-reject-vs-execution-failure --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-017-reject-vs-execution-failure'
```

**CT-TX-018-txpool-propagation.json** · `basic-txpool-propagation`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-018-txpool-propagation.json \
  --workspace-dir ~/cbw/one/CT-TX-018-txpool-propagation --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-018-txpool-propagation'
```

**CT-TX-019-block-progress-under-load.json** · `stress-tx-flood`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-019-block-progress-under-load.json \
  --workspace-dir ~/cbw/one/CT-TX-019-block-progress-under-load --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-019-block-progress-under-load'
```

**CT-TX-020-test-account-funding.json** · `stablenet-faucet-funds`
```sh
bin/chainbench run tests/tc/common/tx/CT-TX-020-test-account-funding.json \
  --workspace-dir ~/cbw/one/CT-TX-020-test-account-funding --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log 'CT-TX-020-test-account-funding'
```

### `go-stablenet/hardfork` — 1건

**01-boho-crossed-by-restart.json** · `boho-crossed-by-restart`
```sh
GSTABLE_POSTFORK_BIN=<값> bin/chainbench run tests/tc/go-stablenet/hardfork/01-boho-crossed-by-restart.json \
  --workspace-dir ~/cbw/one/01-boho-crossed-by-restart --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '01-boho-crossed-by-restart'
```

### `go-stablenet/post-v1.0.0-change/common-all` — 16건

**01-stablenet-delayed-fork.json** · `stablenet-delayed-fork`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json \
  --workspace-dir ~/cbw/one/01-stablenet-delayed-fork --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '01-stablenet-delayed-fork'
```

**02-govminter-v2-code.json** · `govminter-v2-code`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json \
  --workspace-dir ~/cbw/one/02-govminter-v2-code --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '02-govminter-v2-code'
```

**03-burn-cancel-refundable.json** · `burn-cancel-refundable`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json \
  --workspace-dir ~/cbw/one/03-burn-cancel-refundable --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '03-burn-cancel-refundable'
```

**04-burn-reject-refundable.json** · `burn-reject-refundable`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json \
  --workspace-dir ~/cbw/one/04-burn-reject-refundable --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '04-burn-reject-refundable'
```

**05-burn-expire-refundable.json** · `burn-expire-refundable`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json \
  --workspace-dir ~/cbw/one/05-burn-expire-refundable --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '05-burn-expire-refundable'
```

**06-burn-execute-no-refundable.json** · `burn-execute-no-refundable`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json \
  --workspace-dir ~/cbw/one/06-burn-execute-no-refundable --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '06-burn-execute-no-refundable'
```

**07-claim-burn-refund-succeeds.json** · `claim-burn-refund-succeeds`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json \
  --workspace-dir ~/cbw/one/07-claim-burn-refund-succeeds --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '07-claim-burn-refund-succeeds'
```

**08-claim-zero-refund-reverts.json** · `claim-zero-refund-reverts`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json \
  --workspace-dir ~/cbw/one/08-claim-zero-refund-reverts --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '08-claim-zero-refund-reverts'
```

**09-claim-burn-refund-double-reverts.json** · `claim-burn-refund-double-reverts`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json \
  --workspace-dir ~/cbw/one/09-claim-burn-refund-double-reverts --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '09-claim-burn-refund-double-reverts'
```

**10-prealloc-preserved-across-boho.json** · `prealloc-preserved-across-boho`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json \
  --workspace-dir ~/cbw/one/10-prealloc-preserved-across-boho --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '10-prealloc-preserved-across-boho'
```

**15-boho-chain-config-active.json** · `boho-chain-config-active`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json \
  --workspace-dir ~/cbw/one/15-boho-chain-config-active --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '15-boho-chain-config-active'
```

**16-anzeon-active-before-boho.json** · `anzeon-active-before-boho`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json \
  --workspace-dir ~/cbw/one/16-anzeon-active-before-boho --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '16-anzeon-active-before-boho'
```

**17-estimategas-authorizationlist-cost.json** · `estimategas-authorizationlist-cost`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json \
  --workspace-dir ~/cbw/one/17-estimategas-authorizationlist-cost --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '17-estimategas-authorizationlist-cost'
```

**19-upgrade-registry-order.json** · `upgrade-registry-order`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json \
  --workspace-dir ~/cbw/one/19-upgrade-registry-order --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '19-upgrade-registry-order'
```

**20-v1-params-init-storage.json** · `v1-params-init-storage`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json \
  --workspace-dir ~/cbw/one/20-v1-params-init-storage --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '20-v1-params-init-storage'
```

**21-burn-refund-events.json** · `burn-refund-events`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json \
  --workspace-dir ~/cbw/one/21-burn-refund-events --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '21-burn-refund-events'
```

### `go-stablenet/post-v1.0.0-change/effective-gas-price` — 3건

**01-effective-gas-price-authorized-bp-en.json** · `effective-gas-price-authorized-bp-en`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/effective-gas-price/01-effective-gas-price-authorized-bp-en.json \
  --workspace-dir ~/cbw/one/01-effective-gas-price-authorized-bp-en --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '01-effective-gas-price-authorized-bp-en'
```

**02-effective-gas-price-regular.json** · `effective-gas-price-regular`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/effective-gas-price/02-effective-gas-price-regular.json \
  --workspace-dir ~/cbw/one/02-effective-gas-price-regular --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '02-effective-gas-price-regular'
```

**03-auth-tx-event-last-bp-en.json** · `auth-tx-event-last-bp-en`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/effective-gas-price/03-auth-tx-event-last-bp-en.json \
  --workspace-dir ~/cbw/one/03-auth-tx-event-last-bp-en --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '03-auth-tx-event-last-bp-en'
```

### `go-stablenet/post-v1.0.0-change/extra-state` — 8건

**01-authorized-extra-bit-synced.json** · `authorized-extra-bit-synced`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json \
  --workspace-dir ~/cbw/one/01-authorized-extra-bit-synced --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '01-authorized-extra-bit-synced'
```

**01b-blacklisted-extra-bit-synced.json** · `blacklisted-extra-bit-synced`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json \
  --workspace-dir ~/cbw/one/01b-blacklisted-extra-bit-synced --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '01b-blacklisted-extra-bit-synced'
```

**02-stablenet-account-extra.json** · `stablenet-account-extra`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json \
  --workspace-dir ~/cbw/one/02-stablenet-account-extra --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '02-stablenet-account-extra'
```

**03-extra-union-merge.json** · `extra-union-merge`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json \
  --workspace-dir ~/cbw/one/03-extra-union-merge --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '03-extra-union-merge'
```

**04-dual-status-extra.json** · `dual-status-extra`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json \
  --workspace-dir ~/cbw/one/04-dual-status-extra --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '04-dual-status-extra'
```

**05-extra-balance-preserved.json** · `extra-balance-preserved`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json \
  --workspace-dir ~/cbw/one/05-extra-balance-preserved --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '05-extra-balance-preserved'
```

**06-invalid-extra-reject.json** · `invalid-extra-reject`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json \
  --workspace-dir ~/cbw/one/06-invalid-extra-reject --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '06-invalid-extra-reject'
```

**07b-extra-state-across-delayed-boho.json** · `extra-state-across-delayed-boho`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json \
  --workspace-dir ~/cbw/one/07b-extra-state-across-delayed-boho --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '07b-extra-state-across-delayed-boho'
```

### `go-stablenet/post-v1.0.0-change/stand-alone` — 2건

**01b-signature-compat-across-swap.json** · `signature-compat-across-swap`
```sh
GSTABLE_UPGRADE_BIN=<값> bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json \
  --workspace-dir ~/cbw/one/01b-signature-compat-across-swap --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '01b-signature-compat-across-swap'
```

**03-unsupported-version.json** · `unsupported-system-contract-version`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json \
  --workspace-dir ~/cbw/one/03-unsupported-version --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '03-unsupported-version'
```

### `go-stablenet/post-v1.0.0-change/string-handling` — 6건

**01-authorized-accounts-no-space.json** · `authorized-accounts-no-space`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json \
  --workspace-dir ~/cbw/one/01-authorized-accounts-no-space --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '01-authorized-accounts-no-space'
```

**02-authorized-accounts-space.json** · `authorized-accounts-space`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json \
  --workspace-dir ~/cbw/one/02-authorized-accounts-space --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '02-authorized-accounts-space'
```

**03-authorized-accounts-trim.json** · `authorized-accounts-trim`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json \
  --workspace-dir ~/cbw/one/03-authorized-accounts-trim --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '03-authorized-accounts-trim'
```

**04-authorized-accounts-empty-item.json** · `authorized-accounts-empty-item`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json \
  --workspace-dir ~/cbw/one/04-authorized-accounts-empty-item --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '04-authorized-accounts-empty-item'
```

**05-authorized-accounts-single.json** · `authorized-accounts-single`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json \
  --workspace-dir ~/cbw/one/05-authorized-accounts-single --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '05-authorized-accounts-single'
```

**06-authorized-accounts-empty.json** · `authorized-accounts-empty`
```sh
bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json \
  --workspace-dir ~/cbw/one/06-authorized-accounts-empty --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '06-authorized-accounts-empty'
```

### `go-stablenet/regression/anzeon` — 11건

**01-regular-account-gastip-forced.json** · `regular-account-gastip-forced`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json \
  --workspace-dir ~/cbw/one/01-regular-account-gastip-forced --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '01-regular-account-gastip-forced'
```

**02-authorized-account-gastip-free.json** · `authorized-account-gastip-free`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json \
  --workspace-dir ~/cbw/one/02-authorized-account-gastip-free --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '02-authorized-account-gastip-free'
```

**03-anzeon-basefee-increase.json** · `anzeon-basefee-increase`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json \
  --workspace-dir ~/cbw/one/03-anzeon-basefee-increase --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '03-anzeon-basefee-increase'
```

**04-anzeon-basefee-stable.json** · `anzeon-basefee-stable`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json \
  --workspace-dir ~/cbw/one/04-anzeon-basefee-stable --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '04-anzeon-basefee-stable'
```

**05-anzeon-basefee-decrease.json** · `anzeon-basefee-decrease`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json \
  --workspace-dir ~/cbw/one/05-anzeon-basefee-decrease --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '05-anzeon-basefee-decrease'
```

**06-basefee-minimum.json** · `basefee-minimum`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/anzeon/06-basefee-minimum.json \
  --workspace-dir ~/cbw/one/06-basefee-minimum --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '06-basefee-minimum'
```

**07-basefee-maximum.json** · `basefee-maximum`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/anzeon/07-basefee-maximum.json \
  --workspace-dir ~/cbw/one/07-basefee-maximum --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '07-basefee-maximum'
```

**08-feecap-above-min-accepted.json** · `feecap-above-min-accepted`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json \
  --workspace-dir ~/cbw/one/08-feecap-above-min-accepted --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '08-feecap-above-min-accepted'
```

**09-feecap-exact-min-accepted.json** · `feecap-exact-min-accepted`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json \
  --workspace-dir ~/cbw/one/09-feecap-exact-min-accepted --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '09-feecap-exact-min-accepted'
```

**11-gaslimit-exceeded-rejected.json** · `gaslimit-exceeded-rejected`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json \
  --workspace-dir ~/cbw/one/11-gaslimit-exceeded-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '11-gaslimit-exceeded-rejected'
```

**12-basefee-redistributed-not-burned.json** · `basefee-redistributed-not-burned`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/anzeon/12-basefee-redistributed-not-burned.json \
  --workspace-dir ~/cbw/one/12-basefee-redistributed-not-burned --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '12-basefee-redistributed-not-burned'
```

### `go-stablenet/regression/api` — 14건

**06-system-contracts-deployed.json** · `system-contracts-deployed`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/06-system-contracts-deployed.json \
  --workspace-dir ~/cbw/one/06-system-contracts-deployed --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '06-system-contracts-deployed'
```

**07b-gas-price-equals-basefee-plus-tip.json** · `gas-price-equals-basefee-plus-tip`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json \
  --workspace-dir ~/cbw/one/07b-gas-price-equals-basefee-plus-tip --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '07b-gas-price-equals-basefee-plus-tip'
```

**08-max-priority-fee-equals-gastip.json** · `max-priority-fee-equals-gastip`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json \
  --workspace-dir ~/cbw/one/08-max-priority-fee-equals-gastip --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '08-max-priority-fee-equals-gastip'
```

**10-estimate-gas-token-transfer.json** · `estimate-gas-token-transfer`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/10-estimate-gas-token-transfer.json \
  --workspace-dir ~/cbw/one/10-estimate-gas-token-transfer --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '10-estimate-gas-token-transfer'
```

**11-node-address-returned.json** · `node-address-returned`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/11-node-address-returned.json \
  --workspace-dir ~/cbw/one/11-node-address-returned --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '11-node-address-returned'
```

**12-validator-set-nonempty.json** · `validator-set-nonempty`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/12-validator-set-nonempty.json \
  --workspace-dir ~/cbw/one/12-validator-set-nonempty --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '12-validator-set-nonempty'
```

**12b-validator-set-count.json** · `validator-set-count`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/12b-validator-set-count.json \
  --workspace-dir ~/cbw/one/12b-validator-set-count --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '12b-validator-set-count'
```

**12c-validator-equal-power.json** · `validator-equal-power`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/12c-validator-equal-power.json \
  --workspace-dir ~/cbw/one/12c-validator-equal-power --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '12c-validator-equal-power'
```

**13-commit-signers-quorum.json** · `commit-signers-quorum`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/13-commit-signers-quorum.json \
  --workspace-dir ~/cbw/one/13-commit-signers-quorum --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '13-commit-signers-quorum'
```

**14-wbft-extra-info-fields.json** · `wbft-extra-info-fields`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/14-wbft-extra-info-fields.json \
  --workspace-dir ~/cbw/one/14-wbft-extra-info-fields --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '14-wbft-extra-info-fields'
```

**15-istanbul-status-fields.json** · `istanbul-status-fields`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/15-istanbul-status-fields.json \
  --workspace-dir ~/cbw/one/15-istanbul-status-fields --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '15-istanbul-status-fields'
```

**16-is-validator-flags.json** · `is-validator-flags`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/16-is-validator-flags.json \
  --workspace-dir ~/cbw/one/16-is-validator-flags --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '16-is-validator-flags'
```

**22-token-total-supply-readable.json** · `token-total-supply-readable`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/22-token-total-supply-readable.json \
  --workspace-dir ~/cbw/one/22-token-total-supply-readable --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '22-token-total-supply-readable'
```

**23-token-approve-sets-allowance.json** · `token-approve-sets-allowance`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/api/23-token-approve-sets-allowance.json \
  --workspace-dir ~/cbw/one/23-token-approve-sets-allowance --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '23-token-approve-sets-allowance'
```

### `go-stablenet/regression/blacklist-authorized` — 9건

**01-sender-blacklisted-rejected.json** · `sender-blacklisted-rejected`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json \
  --workspace-dir ~/cbw/one/01-sender-blacklisted-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '01-sender-blacklisted-rejected'
```

**02-recipient-blacklisted-rejected.json** · `recipient-blacklisted-rejected`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json \
  --workspace-dir ~/cbw/one/02-recipient-blacklisted-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '02-recipient-blacklisted-rejected'
```

**03-feepayer-blacklisted-rejected.json** · `feepayer-blacklisted-rejected`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json \
  --workspace-dir ~/cbw/one/03-feepayer-blacklisted-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '03-feepayer-blacklisted-rejected'
```

**04-address-unblacklisted-event.json** · `address-unblacklisted-event`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json \
  --workspace-dir ~/cbw/one/04-address-unblacklisted-event --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '04-address-unblacklisted-event'
```

**05-zero-address-transfer-rejected.json** · `zero-address-transfer-rejected`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json \
  --workspace-dir ~/cbw/one/05-zero-address-transfer-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '05-zero-address-transfer-rejected'
```

**06-precompile-transfer-rejected.json** · `precompile-transfer-rejected`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json \
  --workspace-dir ~/cbw/one/06-precompile-transfer-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '06-precompile-transfer-rejected'
```

**07-account-blacklist-readable.json** · `account-blacklist-readable`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json \
  --workspace-dir ~/cbw/one/07-account-blacklist-readable --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '07-account-blacklist-readable'
```

**08-account-authorization-readable.json** · `account-authorization-readable`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json \
  --workspace-dir ~/cbw/one/08-account-authorization-readable --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '08-account-authorization-readable'
```

**09-authorized-tx-executed-event.json** · `authorized-tx-executed-event`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json \
  --workspace-dir ~/cbw/one/09-authorized-tx-executed-event --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '09-authorized-tx-executed-event'
```

### `go-stablenet/regression/ethereum` — 3건

**08-legacy-transfer.json** · `legacy-transfer`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/ethereum/08-legacy-transfer.json \
  --workspace-dir ~/cbw/one/08-legacy-transfer --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '08-legacy-transfer'
```

**09-dynamic-fee-tx.json** · `dynamic-fee-tx`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/ethereum/09-dynamic-fee-tx.json \
  --workspace-dir ~/cbw/one/09-dynamic-fee-tx --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '09-dynamic-fee-tx'
```

**18-set-code-delegation.json** · `set-code-delegation`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json \
  --workspace-dir ~/cbw/one/18-set-code-delegation --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '18-set-code-delegation'
```

### `go-stablenet/regression/system-contracts` — 23건

**01-native-coin-adapter-code.json** · `native-coin-adapter-code`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json \
  --workspace-dir ~/cbw/one/01-native-coin-adapter-code --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '01-native-coin-adapter-code'
```

**01b-token-transfer-emits-event.json** · `token-transfer-emits-event`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json \
  --workspace-dir ~/cbw/one/01b-token-transfer-emits-event --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '01b-token-transfer-emits-event'
```

**02-token-balance-readable.json** · `token-balance-readable`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/02-token-balance-readable.json \
  --workspace-dir ~/cbw/one/02-token-balance-readable --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '02-token-balance-readable'
```

**03-token-transfer-from-moves-balance.json** · `token-transfer-from-moves-balance`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json \
  --workspace-dir ~/cbw/one/03-token-transfer-from-moves-balance --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '03-token-transfer-from-moves-balance'
```

**04-mint-transfer-event.json** · `mint-transfer-event`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/04-mint-transfer-event.json \
  --workspace-dir ~/cbw/one/04-mint-transfer-event --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '04-mint-transfer-event'
```

**05-burn-transfer-event.json** · `burn-transfer-event`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/05-burn-transfer-event.json \
  --workspace-dir ~/cbw/one/05-burn-transfer-event --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '05-burn-transfer-event'
```

**06-mint-proposal-executes.json** · `mint-proposal-executes`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/06-mint-proposal-executes.json \
  --workspace-dir ~/cbw/one/06-mint-proposal-executes --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '06-mint-proposal-executes'
```

**07-burn-proposal-executes.json** · `burn-proposal-executes`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/07-burn-proposal-executes.json \
  --workspace-dir ~/cbw/one/07-burn-proposal-executes --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '07-burn-proposal-executes'
```

**08-quorum-deficient-stays-voting.json** · `quorum-deficient-stays-voting`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json \
  --workspace-dir ~/cbw/one/08-quorum-deficient-stays-voting --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '08-quorum-deficient-stays-voting'
```

**09-validator-metadata-readable.json** · `validator-metadata-readable`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/09-validator-metadata-readable.json \
  --workspace-dir ~/cbw/one/09-validator-metadata-readable --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '09-validator-metadata-readable'
```

**10-gastip-governance-updates-header.json** · `gastip-governance-updates-header`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json \
  --workspace-dir ~/cbw/one/10-gastip-governance-updates-header --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '10-gastip-governance-updates-header'
```

**11-proposal-expiry-transitions.json** · `proposal-expiry-transitions`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json \
  --workspace-dir ~/cbw/one/11-proposal-expiry-transitions --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '11-proposal-expiry-transitions'
```

**12-configure-minter-proposal-executes.json** · `configure-minter-proposal-executes`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json \
  --workspace-dir ~/cbw/one/12-configure-minter-proposal-executes --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '12-configure-minter-proposal-executes'
```

**13-remove-minter-executes.json** · `remove-minter-executes`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/13-remove-minter-executes.json \
  --workspace-dir ~/cbw/one/13-remove-minter-executes --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '13-remove-minter-executes'
```

**14-masterminter-member-add-remove.json** · `masterminter-member-add-remove`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json \
  --workspace-dir ~/cbw/one/14-masterminter-member-add-remove --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '14-masterminter-member-add-remove'
```

**15-non-member-configure-minter-rejected.json** · `non-member-configure-minter-rejected`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json \
  --workspace-dir ~/cbw/one/15-non-member-configure-minter-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '15-non-member-configure-minter-rejected'
```

**16-blacklist-proposal-executes.json** · `blacklist-proposal-executes`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json \
  --workspace-dir ~/cbw/one/16-blacklist-proposal-executes --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '16-blacklist-proposal-executes'
```

**18-authorize-proposal-executes.json** · `authorize-proposal-executes`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json \
  --workspace-dir ~/cbw/one/18-authorize-proposal-executes --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '18-authorize-proposal-executes'
```

**19-unauthorize-proposal-executes.json** · `unauthorize-proposal-executes`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json \
  --workspace-dir ~/cbw/one/19-unauthorize-proposal-executes --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '19-unauthorize-proposal-executes'
```

**20-direct-blacklist-call-rejected.json** · `direct-blacklist-call-rejected`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json \
  --workspace-dir ~/cbw/one/20-direct-blacklist-call-rejected --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '20-direct-blacklist-call-rejected'
```

**23-authorized-account-added-event.json** · `authorized-account-added-event`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/23-authorized-account-added-event.json \
  --workspace-dir ~/cbw/one/23-authorized-account-added-event --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '23-authorized-account-added-event'
```

**25-token-metadata.json** · `token-metadata`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/25-token-metadata.json \
  --workspace-dir ~/cbw/one/25-token-metadata --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '25-token-metadata'
```

**28-minter-status-readable.json** · `minter-status-readable`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/28-minter-status-readable.json \
  --workspace-dir ~/cbw/one/28-minter-status-readable --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '28-minter-status-readable'
```

### `go-stablenet/regression/wbft` — 8건

**02-wbft-seals-quorum.json** · `wbft-seals-quorum`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/wbft/02-wbft-seals-quorum.json \
  --workspace-dir ~/cbw/one/02-wbft-seals-quorum --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '02-wbft-seals-quorum'
```

**03-epoch-transition-carries-epoch-info.json** · `epoch-transition-carries-epoch-info`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json \
  --workspace-dir ~/cbw/one/03-epoch-transition-carries-epoch-info --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '03-epoch-transition-carries-epoch-info'
```

**04-validator-add-member-executes.json** · `validator-add-member-executes`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/wbft/04-validator-add-member-executes.json \
  --workspace-dir ~/cbw/one/04-validator-add-member-executes --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '04-validator-add-member-executes'
```

**04b-validator-add-member-epoch-activates.json** · `validator-add-member-epoch-activates`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json \
  --workspace-dir ~/cbw/one/04b-validator-add-member-epoch-activates --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '04b-validator-add-member-epoch-activates'
```

**05-validator-remove-member-executes.json** · `validator-remove-member-executes`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/wbft/05-validator-remove-member-executes.json \
  --workspace-dir ~/cbw/one/05-validator-remove-member-executes --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '05-validator-remove-member-executes'
```

**11-prev-seals-quorum.json** · `prev-seals-quorum`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/wbft/11-prev-seals-quorum.json \
  --workspace-dir ~/cbw/one/11-prev-seals-quorum --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '11-prev-seals-quorum'
```

**13-randao-and-mixdigest-present.json** · `randao-and-mixdigest-present`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json \
  --workspace-dir ~/cbw/one/13-randao-and-mixdigest-present --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '13-randao-and-mixdigest-present'
```

**14-stablenet-gastip-field.json** · `stablenet-gastip-field`
```sh
bin/chainbench run tests/tc/go-stablenet/regression/wbft/14-stablenet-gastip-field.json \
  --workspace-dir ~/cbw/one/14-stablenet-gastip-field --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '14-stablenet-gastip-field'
```

### `go-stablenet/testnet` — 5건

**01-chain-identity.json** · `testnet-chain-identity`
```sh
GSTABLE_TESTNET_RPC=<값> bin/chainbench run tests/tc/go-stablenet/testnet/01-chain-identity.json
```
> 이미 떠 있는 망에 붙는다. 망을 세우지 않으므로 `--workspace-dir` 도 `--binary` 도 주지 않는다. `tcsweep.sh` 는 이 갈래를 자기가 세운 망에 붙여 돌린다.

**02-block-advances.json** · `testnet-block-advances`
```sh
GSTABLE_TESTNET_RPC=<값> bin/chainbench run tests/tc/go-stablenet/testnet/02-block-advances.json
```
> 이미 떠 있는 망에 붙는다. 망을 세우지 않으므로 `--workspace-dir` 도 `--binary` 도 주지 않는다. `tcsweep.sh` 는 이 갈래를 자기가 세운 망에 붙여 돌린다.

**03-block-fields-well-formed.json** · `testnet-block-fields-well-formed`
```sh
GSTABLE_TESTNET_RPC=<값> bin/chainbench run tests/tc/go-stablenet/testnet/03-block-fields-well-formed.json
```
> 이미 떠 있는 망에 붙는다. 망을 세우지 않으므로 `--workspace-dir` 도 `--binary` 도 주지 않는다. `tcsweep.sh` 는 이 갈래를 자기가 세운 망에 붙여 돌린다.

**04-block-by-hash-consistency.json** · `testnet-block-by-hash-consistency`
```sh
GSTABLE_TESTNET_RPC=<값> bin/chainbench run tests/tc/go-stablenet/testnet/04-block-by-hash-consistency.json
```
> 이미 떠 있는 망에 붙는다. 망을 세우지 않으므로 `--workspace-dir` 도 `--binary` 도 주지 않는다. `tcsweep.sh` 는 이 갈래를 자기가 세운 망에 붙여 돌린다.

**05-value-transfer.json** · `testnet-value-transfer`
```sh
GSTABLE_TESTNET_KEY_FILE=<값> GSTABLE_TESTNET_RPC=<값> bin/chainbench run tests/tc/go-stablenet/testnet/05-value-transfer.json
```
> 이미 떠 있는 망에 붙는다. 망을 세우지 않으므로 `--workspace-dir` 도 `--binary` 도 주지 않는다. `tcsweep.sh` 는 이 갈래를 자기가 세운 망에 붙여 돌린다.

### `go-stablenet/vocabulary` — 2건

**01-derived-address-and-checksum.json** · `stablenet-derived-vocabulary`
```sh
bin/chainbench run tests/tc/go-stablenet/vocabulary/01-derived-address-and-checksum.json \
  --workspace-dir ~/cbw/one/01-derived-address-and-checksum --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '01-derived-address-and-checksum'
```

**03-register-contract.json** · `stablenet-register-contract`
```sh
bin/chainbench run tests/tc/go-stablenet/vocabulary/03-register-contract.json \
  --workspace-dir ~/cbw/one/03-register-contract --binary $GSTABLE

scripts/tcsweep.sh ~/cbw/one.log '03-register-contract'
```

### `go-wbft/accounts` — 3건

**01-secp256r1-precompile-valid.json** · `secp256r1-precompile-valid`
```sh
bin/chainbench run tests/tc/go-wbft/accounts/01-secp256r1-precompile-valid.json \
  --workspace-dir ~/cbw/one/01-secp256r1-precompile-valid --binary $GWBFT

scripts/tcsweep.sh ~/cbw/one.log '01-secp256r1-precompile-valid'
```

**02-secp256r1-precompile-invalid.json** · `secp256r1-precompile-invalid`
```sh
bin/chainbench run tests/tc/go-wbft/accounts/02-secp256r1-precompile-invalid.json \
  --workspace-dir ~/cbw/one/02-secp256r1-precompile-invalid --binary $GWBFT

scripts/tcsweep.sh ~/cbw/one.log '02-secp256r1-precompile-invalid'
```

**03-secp256r1-precompile-short-input.json** · `secp256r1-precompile-short-input`
```sh
bin/chainbench run tests/tc/go-wbft/accounts/03-secp256r1-precompile-short-input.json \
  --workspace-dir ~/cbw/one/03-secp256r1-precompile-short-input --binary $GWBFT

scripts/tcsweep.sh ~/cbw/one.log '03-secp256r1-precompile-short-input'
```

### `go-wbft/consensus` — 1건

**02-wbft-quorum-halt-and-recover.json** · `wbft-quorum-halt-and-recover`
```sh
bin/chainbench run tests/tc/go-wbft/consensus/02-wbft-quorum-halt-and-recover.json \
  --workspace-dir ~/cbw/one/02-wbft-quorum-halt-and-recover --binary $GWBFT

scripts/tcsweep.sh ~/cbw/one.log '02-wbft-quorum-halt-and-recover'
```

### `go-wbft/fault` — 1건

**02-wbft-quorum-at-15-nodes.json** · `wbft-quorum-at-15-nodes`
```sh
bin/chainbench run tests/tc/go-wbft/fault/02-wbft-quorum-at-15-nodes.json \
  --workspace-dir ~/cbw/one/02-wbft-quorum-at-15-nodes --binary $GWBFT

scripts/tcsweep.sh ~/cbw/one.log '02-wbft-quorum-at-15-nodes'
```

### `go-wbft/governance` — 2건

**01-wbft-govcontracts-at-genesis.json** · `wbft-govcontracts-at-genesis`
```sh
bin/chainbench run tests/tc/go-wbft/governance/01-wbft-govcontracts-at-genesis.json \
  --workspace-dir ~/cbw/one/01-wbft-govcontracts-at-genesis --binary $GWBFT

scripts/tcsweep.sh ~/cbw/one.log '01-wbft-govcontracts-at-genesis'
```

**02-wbft-governance-register-staker.json** · `wbft-governance-register-staker`
```sh
bin/chainbench run tests/tc/go-wbft/governance/02-wbft-governance-register-staker.json \
  --workspace-dir ~/cbw/one/02-wbft-governance-register-staker --binary $GWBFT

scripts/tcsweep.sh ~/cbw/one.log '02-wbft-governance-register-staker'
```

### `go-wbft/tx` — 1건

**04-wbft-tx-crosses-the-proxied-topology.json** · `wbft-tx-crosses-the-proxied-topology`
```sh
bin/chainbench run tests/tc/go-wbft/tx/04-wbft-tx-crosses-the-proxied-topology.json \
  --workspace-dir ~/cbw/one/04-wbft-tx-crosses-the-proxied-topology --binary $GWBFT

scripts/tcsweep.sh ~/cbw/one.log '04-wbft-tx-crosses-the-proxied-topology'
```

### `go-wemix/consensus` — 1건

**01-wemix-etcd-and-governance.json** · `wemix-etcd-and-governance`
```sh
bin/chainbench run tests/tc/go-wemix/consensus/01-wemix-etcd-and-governance.json \
  --workspace-dir ~/cbw/one/01-wemix-etcd-and-governance --binary $GWEMIX

scripts/tcsweep.sh ~/cbw/one.log '01-wemix-etcd-and-governance'
```

### `go-wemix/governance` — 1건

**01-wemix-governance-staking-deposit.json** · `wemix-governance-staking-deposit`
```sh
bin/chainbench run tests/tc/go-wemix/governance/01-wemix-governance-staking-deposit.json \
  --workspace-dir ~/cbw/one/01-wemix-governance-staking-deposit --binary $GWEMIX

scripts/tcsweep.sh ~/cbw/one.log '01-wemix-governance-staking-deposit'
```

### `go-wemix/hardfork` — 3건

**01-croissant-successors-take-over.json** · `croissant-successors-take-over`
```sh
bin/chainbench run tests/tc/go-wemix/hardfork/01-croissant-successors-take-over.json \
  --workspace-dir ~/cbw/one/01-croissant-successors-take-over --binary $GWEMIX

scripts/tcsweep.sh ~/cbw/one.log '01-croissant-successors-take-over'
```

**02-state-written-before-the-fork-survives-it.json** · `state-written-before-the-fork-survives-it`
```sh
bin/chainbench run tests/tc/go-wemix/hardfork/02-state-written-before-the-fork-survives-it.json \
  --workspace-dir ~/cbw/one/02-state-written-before-the-fork-survives-it --binary $GWEMIX

scripts/tcsweep.sh ~/cbw/one.log '02-state-written-before-the-fork-survives-it'
```

**03-two-producers-hand-over.json** · `two-producers-hand-over`
```sh
bin/chainbench run tests/tc/go-wemix/hardfork/03-two-producers-hand-over.json \
  --workspace-dir ~/cbw/one/03-two-producers-hand-over --binary $GWEMIX

scripts/tcsweep.sh ~/cbw/one.log '03-two-producers-hand-over'
```

### `go-wemix/rpc` — 1건

**01-wemix-brioche-block-reward.json** · `wemix-brioche-block-reward`
```sh
bin/chainbench run tests/tc/go-wemix/rpc/01-wemix-brioche-block-reward.json \
  --workspace-dir ~/cbw/one/01-wemix-brioche-block-reward --binary $GWEMIX

scripts/tcsweep.sh ~/cbw/one.log '01-wemix-brioche-block-reward'
```

### `go-wemix/vocabulary` — 1건

**01-default-on-routes-every-step.json** · `wemix-default-on-routes-every-step`
```sh
bin/chainbench run tests/tc/go-wemix/vocabulary/01-default-on-routes-every-step.json \
  --workspace-dir ~/cbw/one/01-default-on-routes-every-step --binary $GWEMIX

scripts/tcsweep.sh ~/cbw/one.log '01-default-on-routes-every-step'
```

<!-- END generated -->
