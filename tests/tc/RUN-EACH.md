# 라이브 케이스 209건 — 하나씩 돌리는 명령

`tests/tc/` 의 DSL 케이스 전부를 **케이스 하나당 명령 하나**로 적는다. 각 명령은 망을 새로
세우고, 케이스를 돌리고, 노드를 내린다. 케이스끼리 망을 나누지 않는 이유는
`scripts/tcsweep.sh` 머리 주석에 있다 — 같은 chain-preset 이어도 한 케이스가 바꾼 상태가
다음 케이스의 판정을 바꾼다.

2026-09-24 의 트리(`386a78ee`)에서 뽑았다. 케이스가 늘거나 줄면 이 목록도 고친다. 각 케이스의
체인·노드 수는 `chainbench run --plan` 이 푼 값이고, "직전" 은 2026-09-23 docker 15대
스위프의 판정과 소요 시간이다. 로컬 소요 시간은 재지 않았다.

---

## 1. 명령이 기대하는 것

명령은 **한 줄로 끝난다.** 미리 `export` 할 것도, `PATH` 를 바꿀 것도 없다. 줄마다 저장소로
`cd` 하고, 지난 워크스페이스를 지우고, 이 케이스에 필요한 바이너리를 `--binary` 나 줄 앞의
환경변수로 직접 넘긴다. 복사해서 어느 디렉터리에서 붙여도 같다.

대신 경로가 이 기계의 것으로 고정돼 있다. 다른 기계에서 쓰려면 문서 안의 경로를 바꿔야 한다.

바이너리를 넘기는 방식이 다른 여덟 가지(001 · 015 · 047 · 177 · 189 · 195 · 200 · 207)를 2026-09-24 에
`env -i` 로 비운 환경에서 `/` 에 서서 그대로 붙여 돌렸고 여덟 건 모두 pass 였다. 나머지 201건은 같은
모양의 명령이지만 이 방식으로는 돌려 보지 않았다.

| 무엇 | 경로 | 없으면 |
|---|---|---|
| chainbench | `~/work/github/0xmhha/chainbench/bin/chainbench` | 저장소에서 `make build` |
| go-stablenet 빌드 | `~/work/github/wemade/go-stablenet/build/bin/gstable` | 그 저장소에서 `make gstable` |
| go-wbft 빌드 | `~/work/github/wemade/go-wbft/build/bin/gwemix` | 그 저장소에서 `make gwemix` (wbft 도 이 이름으로 나온다) |
| go-wemix 빌드 | `~/work/github/wemade/go-wemix/build/bin/gwemix` | 그 저장소에서 `make gwemix` |
| go-stablenet **boho 이전** 빌드 | `~/cbw/gs-prefork/build/bin/gstable` | 아래. 두 케이스(015 · 047)만 쓴다 |
| 워크스페이스 | `~/cbw/manual/<케이스 id>` | 명령이 만든다 |

boho 이전 빌드는 원본 저장소를 건드리지 않도록 따로 복제해서 만든다. `ad0122af0` 은 boho 를
넣은 커밋의 부모다. 2026-09-24 에 이 기계에서 이미 만들어 두었다.

```sh
git clone -q --shared ~/work/github/wemade/go-stablenet ~/cbw/gs-prefork && git -C ~/cbw/gs-prefork checkout -q ad0122af0 && make -C ~/cbw/gs-prefork gstable
strings ~/cbw/gs-prefork/build/bin/gstable | grep -ci bohoblock    # 0 이어야 한다(현재 빌드는 0 이 아니다)
```

---

## 2. 한 케이스를 돌리고 읽는 법

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/basic-consensus" && bin/chainbench run tests/tc/basic/01-basic-consensus.json --workspace-dir "$HOME/cbw/manual/basic-consensus" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

끝에 이런 세 줄이 나온다.

```
1    basic-consensus  pass
pass=1 fail=0 blocked=0 skip=0
session: ~/.chainbench/sessions/<시각>/UTC-<시각>
```

- 판정은 넷이다. `pass`·`fail` 은 케이스가 답한 것이고, `blocked` 는 망이 준비되지
  않아 질문을 하지 못한 것, `skip` 은 케이스가 요구하는 capability 가 없어 건너뛴 것이다.
  종료 코드는 0 이 돌았음, 1 이 케이스 실패, 2 가 실행 자체가 진행되지 못함이다.
- `session:` 경로에 리포트와 실패 시점에 모은 노드 로그가 남는다. 워크스페이스를 지워도
  이 기록은 남는다.
- **명령이 끝나면 노드는 내려가 있다.** 워크스페이스의 datadir 와 로그는 들여다보라고 남는다.
  망을 띄운 채로 두려면 줄 끝에 `--keep-up` 을 붙이고, 다 본 뒤
  `~/work/github/0xmhha/chainbench/bin/chainbench chain stop --workspace-dir ~/cbw/manual/<id>` 로 내린다.
- 무엇이 세워질지만 보려면 줄 끝에 `--plan` 을 붙인다. 망을 만들지 않는다.

**줄마다 `rm -rf` 가 붙은 이유.** 한 번 돈 워크스페이스에 같은 명령을 다시 걸면 망을 세우지
못하고 `network not ready to test: node1 exhausted 1 restart(s)` 로 끝난다(2026-09-24 확인,
결함이다 — 재조립 신호를 받는 상태가 없다). 고쳐질 때까지 매번 새 워크스페이스로 돌린다.
워크스페이스 경로를 짧게 둔 것은 노드의 IPC 소켓이 그 아래 생기고 macOS 의 소켓 경로 한도가
104바이트이기 때문이다.

---

## 3. 따로 알아야 하는 케이스

| 케이스 | 무엇이 다른가 |
|---|---|
| `boho-crossed-by-restart` (015) | boho 이전 빌드로 올려 포크를 넘긴다. 직전 226s 로 가장 길다 |
| `signature-compat-across-swap` (047) | 노드 하나를 boho 이전 빌드로 바꿔 끼운다. 같은 빌드로 바꾸면 돌기는 하지만 "다른 바이너리" 를 검증하지 못한다 |
| 15노드 다섯 건 (127 · 178 · 182 · 189 · 191, "bp7 en7 pn1") | 한 기계에 노드 15개를 띄운다. 키는 케이스가 generate 로 선언해 새로 만든다 |
| `wemix-chain-up-15` (191) | docker 에서는 BLOCKED 로 끝났다. go-wemix 코어 결함으로 endpoint 하나가 합류하지 못한다(worklist G6). 로컬 한 기계에서는 돌려 보지 않았다 |
| go-wemix 13건 | etcd 를 쓰는 poa 라 올라오는 데 1~2분 걸린다 |
| `remote/*` · `samples/01` | 설명은 "붙은 엔드포인트" 라고 적지만 지금 선언은 `stablenet-bp4` 를 확장해 **망을 직접 세운다**. 이미 떠 있는 망에 붙이려면 `--attach --rpc <url>` 을 쓴다 |

---

## 4. 전체 목록

| 폴더 | 건수 |
|---|---|
| `basic` | 8 |
| `fault` | 6 |
| `go-stablenet/hardfork` | 1 |
| `go-stablenet/post-v1.0.0-change/common-all` | 19 |
| `go-stablenet/post-v1.0.0-change/effectivegasprice` | 4 |
| `go-stablenet/post-v1.0.0-change/extra-state` | 8 |
| `go-stablenet/post-v1.0.0-change/stand-alone` | 4 |
| `go-stablenet/post-v1.0.0-change/string-handling` | 6 |
| `go-stablenet/regression/anzeon` | 11 |
| `go-stablenet/regression/api` | 26 |
| `go-stablenet/regression/blacklist-authorized` | 9 |
| `go-stablenet/regression/ethereum` | 26 |
| `go-stablenet/regression/fee-delegation` | 7 |
| `go-stablenet/regression/system-contracts` | 23 |
| `go-stablenet/regression/wbft` | 9 |
| `go-stablenet/topology` | 1 |
| `go-stablenet/tx` | 1 |
| `go-stablenet/vocabulary` | 4 |
| `go-wbft/accounts` | 3 |
| `go-wbft/chain-up` | 2 |
| `go-wbft/consensus` | 2 |
| `go-wbft/fault` | 2 |
| `go-wbft/governance` | 2 |
| `go-wbft/network` | 1 |
| `go-wbft/tx` | 4 |
| `go-wemix/chain-up` | 2 |
| `go-wemix/consensus` | 1 |
| `go-wemix/fault` | 1 |
| `go-wemix/governance` | 1 |
| `go-wemix/hardfork` | 3 |
| `go-wemix/rpc` | 1 |
| `go-wemix/tx` | 3 |
| `go-wemix/vocabulary` | 1 |
| `remote` | 3 |
| `samples` | 2 |
| `stress` | 2 |
| **합계** | **209** |

### `basic` (8)

**001. `basic-consensus`** — stablenet · bp4 en1 · 직전 PASS 40s  
Verify blocks are being produced and all validators participate (원본 basic/consensus.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/basic-consensus" && bin/chainbench run tests/tc/basic/01-basic-consensus.json --workspace-dir "$HOME/cbw/manual/basic-consensus" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**002. `basic-peers`** — stablenet · bp4 en1 · 직전 PASS 44s  
Verify all nodes have proper peer connectivity (원본 basic/peers.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/basic-peers" && bin/chainbench run tests/tc/basic/02-basic-peers.json --workspace-dir "$HOME/cbw/manual/basic-peers" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**003. `basic-rpc-health`** — stablenet · bp4 en1 · 직전 PASS 29s  
Verify all node RPC endpoints are responding (원본 basic/rpc-health.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/basic-rpc-health" && bin/chainbench run tests/tc/basic/03-basic-rpc-health.json --workspace-dir "$HOME/cbw/manual/basic-rpc-health" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**004. `basic-sync`** — stablenet · bp4 en1 · 직전 PASS 29s  
Verify all running nodes have synchronized block heights (원본 basic/sync.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/basic-sync" && bin/chainbench run tests/tc/basic/04-basic-sync.json --workspace-dir "$HOME/cbw/manual/basic-sync" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**005. `basic-tx-send`** — stablenet · bp4 en1 · 직전 PASS 35s  
Send a transaction and verify it gets included in a block (원본 basic/tx-send.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/basic-tx-send" && bin/chainbench run tests/tc/basic/05-basic-tx-send.json --workspace-dir "$HOME/cbw/manual/basic-tx-send" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**006. `basic-txpool-propagation`** — stablenet · bp4 en1 · 직전 PASS 37s  
Verify TX propagation across nodes and txpool drain under load (원본 basic/txpool-propagation.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/basic-txpool-propagation" && bin/chainbench run tests/tc/basic/06-basic-txpool-propagation.json --workspace-dir "$HOME/cbw/manual/basic-txpool-propagation" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**007. `basic-wbft-consensus`** — stablenet · bp4 en1 · 직전 PASS 45s  
Verify WBFT protocol properties - validator participation, round stability, commit seals (원본 basic/wbft-consensus.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/basic-wbft-consensus" && bin/chainbench run tests/tc/basic/07-basic-wbft-consensus.json --workspace-dir "$HOME/cbw/manual/basic-wbft-consensus" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**008. `attached-chain-produces`** — stablenet · bp4 · 직전 PASS 19s  
이미 떠 있는 망에 선언만으로 붙어서 블록이 나오는지 본다. 명령행에 망을 가리키는 플래그가 하나도 없고, env.attach 가 그 자리를 대신한다.  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/attached-chain-produces" && bin/chainbench run tests/tc/basic/08-attached-chain-produces.json --workspace-dir "$HOME/cbw/manual/attached-chain-produces" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `fault` (6)

**009. `fault-network-partition`** — stablenet · bp4 · 직전 PASS 53s  
Simulate network partition via admin_removePeer - verify consensus halts and recovers after heal (원본 fault/network-partition.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fault-network-partition" && bin/chainbench run tests/tc/fault/01-fault-network-partition.json --workspace-dir "$HOME/cbw/manual/fault-network-partition" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**010. `fault-node-crash`** — stablenet · bp4 · 직전 PASS 58s  
Stop 1 validator and verify consensus continues with 3/4 (원본 fault/node-crash.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fault-node-crash" && bin/chainbench run tests/tc/fault/02-fault-node-crash.json --workspace-dir "$HOME/cbw/manual/fault-node-crash" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**011. `fault-node-recover`** — stablenet · bp4 · 직전 PASS 66s  
Stop a node, wait, restart, and measure sync time (원본 fault/node-recover.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fault-node-recover" && bin/chainbench run tests/tc/fault/03-fault-node-recover.json --workspace-dir "$HOME/cbw/manual/fault-node-recover" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**012. `fault-p2p-topology`** — stablenet · bp4 · 직전 PASS 57s  
Test consensus and TX propagation under restricted hub-spoke P2P topology (원본 fault/p2p-topology.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fault-p2p-topology" && bin/chainbench run tests/tc/fault/04-fault-p2p-topology.json --workspace-dir "$HOME/cbw/manual/fault-p2p-topology" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**013. `fault-two-down`** — stablenet · bp4 · 직전 PASS 69s  
Stop 2/4 validators - consensus should halt, recover when 1 returns (원본 fault/two-down.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fault-two-down" && bin/chainbench run tests/tc/fault/05-fault-two-down.json --workspace-dir "$HOME/cbw/manual/fault-two-down" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**014. `fault-txpool-leader-change`** — stablenet · bp4 · 직전 PASS 56s  
Verify pending transactions survive leader node failure and get processed by remaining validators (원본 fault/txpool-leader-change.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fault-txpool-leader-change" && bin/chainbench run tests/tc/fault/06-fault-txpool-leader-change.json --workspace-dir "$HOME/cbw/manual/fault-txpool-leader-change" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/hardfork` (1)

**015. `boho-crossed-by-restart`** — stablenet · bp4 · 직전 PASS 226s  
The ordinary hardfork: one chain, one build at a time. Four stablenet producers run the pre-fork build, every node is stopped well before…  
> boho 이전 빌드로 올리고 포크 전에 현재 빌드로 바꾼다. boho 이전 빌드가 있어야 한다(§1)

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/boho-crossed-by-restart" && GSTABLE_BIN="$HOME/cbw/gs-prefork/build/bin/gstable" GSTABLE_POSTFORK_BIN="$HOME/work/github/wemade/go-stablenet/build/bin/gstable" bin/chainbench run tests/tc/go-stablenet/hardfork/01-boho-crossed-by-restart.json --workspace-dir "$HOME/cbw/manual/boho-crossed-by-restart"
```


### `go-stablenet/post-v1.0.0-change/common-all` (19)

**016. `stablenet-delayed-fork`** — stablenet · bp4 · 직전 PASS 52s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stablenet-delayed-fork" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json --workspace-dir "$HOME/cbw/manual/stablenet-delayed-fork" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**017. `govminter-v2-code`** — stablenet · bp4 · 직전 PASS 52s  
TC-5-2-05: GovMinter v2 업그레이드는 코드만 교체하고 잔액은 건드리지 않는다. 하드포크 전후의 코드가 다르고 잔액이 같은지를 함께 본다.  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/govminter-v2-code" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json --workspace-dir "$HOME/cbw/manual/govminter-v2-code" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**018. `burn-cancel-refundable`** — stablenet · bp4 · 직전 PASS 56s  
TC-1-1-01, TC-1-1-10 — 소각 제안 취소 → refundableBalance 이동 및 BurnDepositRefunded 이벤트 검증 (원본 post-v1.0.0-change/common-all/03-test-burn-cancel…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/burn-cancel-refundable" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json --workspace-dir "$HOME/cbw/manual/burn-cancel-refundable" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**019. `burn-reject-refundable`** — stablenet · bp4 · 직전 PASS 55s  
TC-1-1-02 — 소각 제안 거부(reject) → refundableBalance 이동 → claimBurnRefund 정상 출금 검증 (원본 post-v1.0.0-change/common-all/04-test-burn-reject-refund)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/burn-reject-refundable" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json --workspace-dir "$HOME/cbw/manual/burn-reject-refundable" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**020. `burn-expire-refundable`** — stablenet · bp4 · 직전 PASS 94s  
TC-1-1-03: 소각 제안이 만료되면 GovMinter 로 옮겨진 예치금이 환불 가능 잔액이 된다. proposeBurn 직후 GovMinter 잔액 증가, 만료 후 상태 Expired(5), refundableBalance 증가분, Burn…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/burn-expire-refundable" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json --workspace-dir "$HOME/cbw/manual/burn-expire-refundable" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**021. `burn-execute-no-refundable`** — stablenet · bp4 · 직전 PASS 55s  
TC-1-1-04 — 소각 제안 승인(approve) → 자동 실행(execute) → refundableBalance == 0 검증 (원본 post-v1.0.0-change/common-all/06-test-burn-execute-no-refund)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/burn-execute-no-refundable" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json --workspace-dir "$HOME/cbw/manual/burn-execute-no-refundable" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**022. `claim-burn-refund-succeeds`** — stablenet · bp4 · 직전 PASS 55s  
TC-1-1-05, TC-1-1-09 — claimBurnRefund 정상 출금 및 BurnRefundClaimed 이벤트 검증 (원본 post-v1.0.0-change/common-all/07-test-claim-refund-success)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/claim-burn-refund-succeeds" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json --workspace-dir "$HOME/cbw/manual/claim-burn-refund-succeeds" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**023. `claim-zero-refund-reverts`** — stablenet · bp4 · 직전 PASS 56s  
TC-1-1-06 — refundableBalance 0 계정 claimBurnRefund revert 검증 (원본 post-v1.0.0-change/common-all/08-test-claim-refund-zero-revert)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/claim-zero-refund-reverts" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json --workspace-dir "$HOME/cbw/manual/claim-zero-refund-reverts" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**024. `claim-burn-refund-double-reverts`** — stablenet · bp4 · 직전 PASS 56s  
TC-1-1-07 — claimBurnRefund 중복 호출 revert 검증 (원본 post-v1.0.0-change/common-all/09-test-claim-refund-double-revert)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/claim-burn-refund-double-reverts" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json --workspace-dir "$HOME/cbw/manual/claim-burn-refund-double-reverts" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**025. `prealloc-preserved-across-boho`** — stablenet · bp4 · 직전 PASS 52s  
TC-1-1-11/12: 하드포크가 prealloc 계정의 잔액·nonce 를 보존하고, 시스템 컨트랙트의 스토리지 슬롯도 그대로 둔다.  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/prealloc-preserved-across-boho" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json --workspace-dir "$HOME/cbw/manual/prealloc-preserved-across-boho" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**026. `legacy-gasprice-below-min-rejected`** — stablenet · bp4 · 직전 PASS 55s  
TC-1-3-04 — LegacyTx gasPrice 최소 가스비 하한선 미만 거부 검증 (원본 post-v1.0.0-change/common-all/12-test-legacy-gasprice-below-min-revert)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/legacy-gasprice-below-min-rejected" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json --workspace-dir "$HOME/cbw/manual/legacy-gasprice-below-min-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**027. `accesslist-gasprice-below-min-rejected`** — stablenet · bp4 · 직전 PASS 52s  
TC-1-3-05 — AccessListTx gasPrice 최소 가스비 하한선 미만 거부 검증 (원본 post-v1.0.0-change/common-all/13-test-accesslist-gasprice-below-min-revert)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/accesslist-gasprice-below-min-rejected" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json --workspace-dir "$HOME/cbw/manual/accesslist-gasprice-below-min-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**028. `feecap-below-min-rejected`** — stablenet · bp4 · 직전 PASS 52s  
TC-1-3-06 — DynamicFeeTx gasTipCap 최소값 미만 거부 검증 (원본 post-v1.0.0-change/common-all/14-test-dynamic-fee-tipcap-below-min-revert)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/feecap-below-min-rejected" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json --workspace-dir "$HOME/cbw/manual/feecap-below-min-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**029. `boho-chain-config-active`** — stablenet · bp4 · 직전 PASS 57s  
TC-4-1-01 — Boho 가 genesis 부터 켜진 망은 블록 1 에 이미 v2 GovMinter 를 들고 있다. v2 전용 함수 refundableBalance(0xb03d36cd) 가 답하는지로 가른다 — v1 에는 그 함수가 없어 이…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/boho-chain-config-active" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json --workspace-dir "$HOME/cbw/manual/boho-chain-config-active" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**030. `anzeon-active-before-boho`** — stablenet · bp4 · 직전 PASS 52s  
TC-4-1-02 — Boho 하드포크 값의 체인 설정 반영 및 런타임 활성화 검증 (원본 post-v1.0.0-change/common-all/16-test-boho-chain-config-activation)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/anzeon-active-before-boho" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json --workspace-dir "$HOME/cbw/manual/anzeon-active-before-boho" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**031. `estimategas-authorizationlist-cost`** — stablenet · bp4 · 직전 PASS 52s  
TC-4-2-01/03: EIP-7702 authorizationList 를 1건·2건 붙였을 때 eth_estimateGas 가 그만큼 늘어난다. 경계값은 preActions 에서 이름을 붙여 한 번씩만 적는다. 상한은 노드의 estimateG…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/estimategas-authorizationlist-cost" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json --workspace-dir "$HOME/cbw/manual/estimategas-authorizationlist-cost" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**032. `upgrade-registry-order`** — stablenet · bp4 · 직전 PASS 117s  
TC-5-2-01/02/03: block 0 에 Anzeon baseline 이 등록돼 있고, BohoBlock(100) 전까지 중간 업그레이드가 없으며, BohoBlock 에서 GovMinter 코드가 교체된다.  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/upgrade-registry-order" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json --workspace-dir "$HOME/cbw/manual/upgrade-registry-order" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**033. `v1-params-init-storage`** — stablenet · bp4 · 직전 PASS 115s  
TC-5-2-04: v1 시스템 컨트랙트의 Params 가 genesis 에서 초기화된다. GovValidator gasTip 슬롯(0x39)과 GovMinter quorum 슬롯(0x04)이 0이 아니어야 한다.  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/v1-params-init-storage" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json --workspace-dir "$HOME/cbw/manual/v1-params-init-storage" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**034. `burn-refund-events`** — stablenet · bp4 · 직전 PASS 56s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/burn-refund-events" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json --workspace-dir "$HOME/cbw/manual/burn-refund-events" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/post-v1.0.0-change/effectivegasprice` (4)

**035. `effective-gas-price-authorized-bp-en`** — stablenet · bp4 en1 (snap) · 직전 PASS 36s  
TC-4-6-01 — 인가 계정 tx 의 effectiveGasPrice 가 블록 생산 노드와 snap-sync 엔드포인트에서 같다  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/effective-gas-price-authorized-bp-en" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json --workspace-dir "$HOME/cbw/manual/effective-gas-price-authorized-bp-en" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**036. `effective-gas-price-regular`** — stablenet · bp4 · 직전 PASS 52s  
TC-4-6-02 — EffectiveGasPrice for regular (non-authorized) account (BP vs snap-sync EN comparison) (원본 post-v1.0.0-change/effectivegaspri…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/effective-gas-price-regular" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json --workspace-dir "$HOME/cbw/manual/effective-gas-price-regular" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**037. `effective-gas-price-regular-bp-en`** — stablenet · bp4 en1 (snap) · 직전 PASS 32s  
TC-4-6-02 — 일반 계정 tx 의 effectiveGasPrice 가 두 노드에서 같다  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/effective-gas-price-regular-bp-en" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json --workspace-dir "$HOME/cbw/manual/effective-gas-price-regular-bp-en" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**038. `auth-tx-event-last-bp-en`** — stablenet · bp4 en1 (snap) · 직전 PASS 36s  
TC-4-6-04 — AuthorizedTxExecuted 가 영수증 로그의 마지막이고, 두 노드가 같은 값을 보고한다  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/auth-tx-event-last-bp-en" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json --workspace-dir "$HOME/cbw/manual/auth-tx-event-last-bp-en" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/post-v1.0.0-change/extra-state` (8)

**039. `authorized-extra-bit-synced`** — stablenet · bp4 · 직전 PASS 51s  
TC-4-5-01,TC-4-5-02 — Account Extra alloc bits reflected in AccountManager (authorized + blacklisted) (원본 post-v1.0.0-change/extra-state/…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/authorized-extra-bit-synced" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json --workspace-dir "$HOME/cbw/manual/authorized-extra-bit-synced" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**040. `blacklisted-extra-bit-synced`** — stablenet · bp4 · 직전 PASS 55s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/blacklisted-extra-bit-synced" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json --workspace-dir "$HOME/cbw/manual/blacklisted-extra-bit-synced" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**041. `stablenet-account-extra`** — stablenet · bp4 · 직전 PASS 52s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stablenet-account-extra" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json --workspace-dir "$HOME/cbw/manual/stablenet-account-extra" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**042. `extra-union-merge`** — stablenet · bp4 · 직전 PASS 57s  
TC-4-5-05/06: alloc.Extra 와 GovCouncil params 가 서로 다른 계정을 인가하면 합집합이 되고, 양쪽에 다 있는 계정은 중복 없이 한 번만 센다 (3개, 4개가 아님).  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/extra-union-merge" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json --workspace-dir "$HOME/cbw/manual/extra-union-merge" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**043. `dual-status-extra`** — stablenet · bp4 · 직전 PASS 52s  
TC-4-5-07 — 동일 주소 dual-status — authorized AND blacklisted 동시 반영 (원본 post-v1.0.0-change/extra-state/04-test-extra-dual-status)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/dual-status-extra" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json --workspace-dir "$HOME/cbw/manual/dual-status-extra" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**044. `extra-balance-preserved`** — stablenet · bp4 · 직전 PASS 52s  
TC-4-5-08 — 동기화 시 무관 계정 잔액 보존 (원본 post-v1.0.0-change/extra-state/05-test-extra-balance-preserved)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/extra-balance-preserved" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json --workspace-dir "$HOME/cbw/manual/extra-balance-preserved" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**045. `invalid-extra-reject`** — stablenet · bp4 en1 · 직전 PASS 51s  
TC-4-5-09 — 미정의 Extra 비트를 가진 genesis 를 받은 노드는 부팅에 실패한다  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/invalid-extra-reject" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json --workspace-dir "$HOME/cbw/manual/invalid-extra-reject" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**046. `extra-state-across-delayed-boho`** — stablenet · bp4 · 직전 PASS 56s  
TC-4-5-10/11/12 — 하드포크가 지연돼도 alloc.Extra 가 AccountManager 에 반영되고 잔액은 보존된다  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/extra-state-across-delayed-boho" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json --workspace-dir "$HOME/cbw/manual/extra-state-across-delayed-boho" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/post-v1.0.0-change/stand-alone` (4)

**047. `signature-compat-across-swap`** — stablenet · bp4 en1 · 직전 PASS 37s  
TC-3-1-04 — 노드가 다른 바이너리로 재기동해도 기존 tx 의 blockNumber·status·from·to 가 보존된다  
> 노드 하나를 boho 이전 빌드로 바꿔 끼운다. 그 빌드가 있어야 한다(§1)

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/signature-compat-across-swap" && GSTABLE_UPGRADE_BIN="$HOME/cbw/gs-prefork/build/bin/gstable" bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json --workspace-dir "$HOME/cbw/manual/signature-compat-across-swap" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**048. `genesis-mismatch-refuses-to-start`** — stablenet · bp4 · 직전 PASS 56s  
TC-4-1-03 — 이 체인과 다른 genesis 로 빌드된 바이너리는 GenesisMismatch 로 기동에 실패한다  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/genesis-mismatch-refuses-to-start" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json --workspace-dir "$HOME/cbw/manual/genesis-mismatch-refuses-to-start" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**049. `unsupported-system-contract-version`** — stablenet · bp4 · 직전 PASS 71s  
TC-5-2-06 — genesis 가 지원하지 않는 시스템 컨트랙트 버전을 선언하면 BohoBlock 을 커밋하지 못하고 멈춘다  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/unsupported-system-contract-version" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json --workspace-dir "$HOME/cbw/manual/unsupported-system-contract-version" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**050. `genesis-block-hash-consistent`** — stablenet · bp4 · 직전 PASS 52s  
TC-5-3-01: block 0 해시가 모든 노드에서 같고 parentHash 가 0 이다. 레거시는 릴리스 고정 해시와 비교했으나, 사설망은 env 마다 genesis 가 달라 노드 간 일치와 genesis 형태로 검증한다.  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/genesis-block-hash-consistent" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json --workspace-dir "$HOME/cbw/manual/genesis-block-hash-consistent" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/post-v1.0.0-change/string-handling` (6)

**051. `authorized-accounts-no-space`** — stablenet · bp4 · 직전 PASS 52s  
TC-4-3-01: GovCouncil authorizedAccounts splitAndTrim — 공백 없음 "0xaaa,0xbbb,0xccc" → 3  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/authorized-accounts-no-space" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json --workspace-dir "$HOME/cbw/manual/authorized-accounts-no-space" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**052. `authorized-accounts-space`** — stablenet · bp4 · 직전 PASS 56s  
TC-4-3-02: GovCouncil authorizedAccounts splitAndTrim — 항목 사이 공백 "0xaaa, 0xbbb, 0xccc" → 3  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/authorized-accounts-space" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json --workspace-dir "$HOME/cbw/manual/authorized-accounts-space" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**053. `authorized-accounts-trim`** — stablenet · bp4 · 직전 PASS 52s  
TC-4-3-03: GovCouncil authorizedAccounts splitAndTrim — 앞뒤 공백 " 0xaaa , 0xbbb " → 2  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/authorized-accounts-trim" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json --workspace-dir "$HOME/cbw/manual/authorized-accounts-trim" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**054. `authorized-accounts-empty-item`** — stablenet · bp4 · 직전 PASS 52s  
TC-4-3-04: GovCouncil authorizedAccounts splitAndTrim — 빈 항목 "0xaaa,,0xbbb" → 2  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/authorized-accounts-empty-item" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json --workspace-dir "$HOME/cbw/manual/authorized-accounts-empty-item" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**055. `authorized-accounts-single`** — stablenet · bp4 · 직전 PASS 52s  
TC-4-3-05: GovCouncil authorizedAccounts splitAndTrim — 단일 항목 "0xaaa" → 1  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/authorized-accounts-single" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json --workspace-dir "$HOME/cbw/manual/authorized-accounts-single" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**056. `authorized-accounts-empty`** — stablenet · bp4 · 직전 PASS 57s  
TC-4-3-06: GovCouncil authorizedAccounts splitAndTrim — 빈 문자열 "" → 0  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/authorized-accounts-empty" && bin/chainbench run tests/tc/go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json --workspace-dir "$HOME/cbw/manual/authorized-accounts-empty" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/regression/anzeon` (11)

**057. `regular-account-gastip-forced`** — stablenet · bp4 · 직전 PASS 53s  
RT-C-01 — 일반 계정의 tipCap이 header.GasTip()으로 강제 대체됨 (원본 regression/anzeon/01-test-regular-account-gastip-forced)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/regular-account-gastip-forced" && bin/chainbench run tests/tc/go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json --workspace-dir "$HOME/cbw/manual/regular-account-gastip-forced" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**058. `authorized-account-gastip-free`** — stablenet · bp4 · 직전 PASS 57s  
RT-C-02 — Authorized 계정은 tipCap을 자유롭게 설정할 수 있음 (원본 regression/anzeon/02-test-authorized-account-gastip-free)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/authorized-account-gastip-free" && bin/chainbench run tests/tc/go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json --workspace-dir "$HOME/cbw/manual/authorized-account-gastip-free" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**059. `anzeon-basefee-increase`** — stablenet · bp4 · 직전 PASS 57s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/anzeon-basefee-increase" && bin/chainbench run tests/tc/go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json --workspace-dir "$HOME/cbw/manual/anzeon-basefee-increase" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**060. `anzeon-basefee-stable`** — stablenet · bp4 · 직전 PASS 59s  
RT-A-04 — 사용량이 안정 구간(6% < 사용량 ≤ 20%) 인 블록은 base fee 를 그대로 다음 블록에 넘긴다. 부하를 건 뒤 마지막 소각 트랜잭션이 담긴 블록과 그 다음 블록을 짝으로 견준다. "부하를 걸고 나중에 한 번 읽는" 방…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/anzeon-basefee-stable" && bin/chainbench run tests/tc/go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json --workspace-dir "$HOME/cbw/manual/anzeon-basefee-stable" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**061. `anzeon-basefee-decrease`** — stablenet · bp4 · 직전 PASS 77s  
RT-A-05: 부하가 끝나면 basefee 가 다시 내려간다. 기다림은 절대 블록 번호가 아니라 부하가 끝난 블록에서 센 상대값이다 — 블록 주기가 1초라 부하가 오래 걸리면 절대 번호는 이미 지나 있고, 그러면 basefee 가 식을 틈 없이…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/anzeon-basefee-decrease" && bin/chainbench run tests/tc/go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json --workspace-dir "$HOME/cbw/manual/anzeon-basefee-decrease" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**062. `basefee-minimum`** — stablenet · bp4 · 직전 PASS 51s  
RT-C-06 — baseFee가 MinBaseFee(20 Gwei) 아래로 내려가지 않음 (원본 regression/anzeon/06-test-min-basefee)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/basefee-minimum" && bin/chainbench run tests/tc/go-stablenet/regression/anzeon/06-basefee-minimum.json --workspace-dir "$HOME/cbw/manual/basefee-minimum" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**063. `basefee-maximum`** — stablenet · bp4 · 직전 PASS 53s  
RT-C-07 — baseFee가 MaxBaseFee(20,000,000 Gwei) 상한을 초과하지 않음 (원본 regression/anzeon/07-test-max-basefee)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/basefee-maximum" && bin/chainbench run tests/tc/go-stablenet/regression/anzeon/07-basefee-maximum.json --workspace-dir "$HOME/cbw/manual/basefee-maximum" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**064. `feecap-above-min-accepted`** — stablenet · bp4 · 직전 PASS 52s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/feecap-above-min-accepted" && bin/chainbench run tests/tc/go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json --workspace-dir "$HOME/cbw/manual/feecap-above-min-accepted" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**065. `feecap-exact-min-accepted`** — stablenet · bp4 · 직전 PASS 56s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/feecap-exact-min-accepted" && bin/chainbench run tests/tc/go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json --workspace-dir "$HOME/cbw/manual/feecap-exact-min-accepted" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**066. `gaslimit-exceeded-rejected`** — stablenet · bp4 · 직전 PASS 52s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/gaslimit-exceeded-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json --workspace-dir "$HOME/cbw/manual/gaslimit-exceeded-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**067. `basefee-redistributed-not-burned`** — stablenet · bp4 · 직전 PASS 56s  
baseFee is redistributed to the epoch's validators and none of it is destroyed. The six basefee specs beside this one check only the form…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/basefee-redistributed-not-burned" && bin/chainbench run tests/tc/go-stablenet/regression/anzeon/12-basefee-redistributed-not-burned.json --workspace-dir "$HOME/cbw/manual/basefee-redistributed-not-burned" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/regression/api` (26)

**068. `block-transactions-field`** — stablenet · bp4 · 직전 PASS 51s  
RT-G-1-01 — eth_getBlockByNumber(latest) (원본 regression/api/01-test-get-block-by-number)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/block-transactions-field" && bin/chainbench run tests/tc/go-stablenet/regression/api/01-block-transactions-field.json --workspace-dir "$HOME/cbw/manual/block-transactions-field" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**069. `block-by-hash-consistency`** — stablenet · bp4 · 직전 PASS 52s  
RT-G-1-02 — eth_getBlockByHash (원본 regression/api/02-test-get-block-by-hash)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/block-by-hash-consistency" && bin/chainbench run tests/tc/go-stablenet/regression/api/02-block-by-hash-consistency.json --workspace-dir "$HOME/cbw/manual/block-by-hash-consistency" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**070. `transaction-by-hash-fields`** — stablenet · bp4 · 직전 PASS 52s  
RT-G-1-03 — eth_getTransactionByHash (원본 regression/api/03-test-get-tx-by-hash)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/transaction-by-hash-fields" && bin/chainbench run tests/tc/go-stablenet/regression/api/03-transaction-by-hash-fields.json --workspace-dir "$HOME/cbw/manual/transaction-by-hash-fields" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**071. `transaction-receipt-fields`** — stablenet · bp4 · 직전 PASS 57s  
RT-G-1-04 — eth_getTransactionReceipt (PR #70 fix 확인) (원본 regression/api/04-test-get-tx-receipt)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/transaction-receipt-fields" && bin/chainbench run tests/tc/go-stablenet/regression/api/04-transaction-receipt-fields.json --workspace-dir "$HOME/cbw/manual/transaction-receipt-fields" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**072. `transaction-count-increments`** — stablenet · bp4 · 직전 PASS 52s  
RT-G-1-05 — eth_getTransactionCount (nonce 조회) (원본 regression/api/05-test-get-tx-count)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/transaction-count-increments" && bin/chainbench run tests/tc/go-stablenet/regression/api/05-transaction-count-increments.json --workspace-dir "$HOME/cbw/manual/transaction-count-increments" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**073. `system-contracts-deployed`** — stablenet · bp4 · 직전 PASS 51s  
RT-G-1-06 — eth_getCode on NativeCoinAdapter (0x1000) (원본 regression/api/06-test-get-code-system)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/system-contracts-deployed" && bin/chainbench run tests/tc/go-stablenet/regression/api/06-system-contracts-deployed.json --workspace-dir "$HOME/cbw/manual/system-contracts-deployed" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**074. `gas-price-positive`** — stablenet · bp4 · 직전 PASS 52s  
RT-G-2-01 — eth_gasPrice == baseFee + GasTip (원본 regression/api/07-test-gas-price)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/gas-price-positive" && bin/chainbench run tests/tc/go-stablenet/regression/api/07-gas-price-positive.json --workspace-dir "$HOME/cbw/manual/gas-price-positive" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**075. `gas-price-equals-basefee-plus-tip`** — stablenet · bp4 · 직전 PASS 56s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/gas-price-equals-basefee-plus-tip" && bin/chainbench run tests/tc/go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json --workspace-dir "$HOME/cbw/manual/gas-price-equals-basefee-plus-tip" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**076. `max-priority-fee-equals-gastip`** — stablenet · bp4 · 직전 PASS 52s  
RT-G-2-02 — eth_maxPriorityFeePerGas == WBFTExtra.GasTip (원본 regression/api/08-test-max-priority-fee)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/max-priority-fee-equals-gastip" && bin/chainbench run tests/tc/go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json --workspace-dir "$HOME/cbw/manual/max-priority-fee-equals-gastip" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**077. `fee-history-well-formed`** — stablenet · bp4 · 직전 PASS 52s  
RT-G-2-03 — eth_feeHistory (원본 regression/api/09-test-fee-history)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fee-history-well-formed" && bin/chainbench run tests/tc/go-stablenet/regression/api/09-fee-history-well-formed.json --workspace-dir "$HOME/cbw/manual/fee-history-well-formed" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**078. `estimate-gas-token-transfer`** — stablenet · bp4 · 직전 PASS 51s  
RT-G-2-04 — eth_estimateGas (NativeCoinAdapter.transfer) (원본 regression/api/10-test-estimate-system-call)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/estimate-gas-token-transfer" && bin/chainbench run tests/tc/go-stablenet/regression/api/10-estimate-gas-token-transfer.json --workspace-dir "$HOME/cbw/manual/estimate-gas-token-transfer" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**079. `node-address-returned`** — stablenet · bp4 · 직전 PASS 51s  
RT-G-3-01 — istanbul_nodeAddress (원본 regression/api/11-test-node-address)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/node-address-returned" && bin/chainbench run tests/tc/go-stablenet/regression/api/11-node-address-returned.json --workspace-dir "$HOME/cbw/manual/node-address-returned" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**080. `validator-set-nonempty`** — stablenet · bp4 · 직전 PASS 52s  
RT-G-3-02 — istanbul_getValidators (원본 regression/api/12-test-get-validators)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/validator-set-nonempty" && bin/chainbench run tests/tc/go-stablenet/regression/api/12-validator-set-nonempty.json --workspace-dir "$HOME/cbw/manual/validator-set-nonempty" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**081. `validator-set-count`** — stablenet · bp4 · 직전 PASS 56s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/validator-set-count" && bin/chainbench run tests/tc/go-stablenet/regression/api/12b-validator-set-count.json --workspace-dir "$HOME/cbw/manual/validator-set-count" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**082. `validator-equal-power`** — stablenet · bp4 · 직전 PASS 55s  
Every validator carries the same weight: each one takes proposing turns, and each one's vote is counted on every block. The two specs bes…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/validator-equal-power" && bin/chainbench run tests/tc/go-stablenet/regression/api/12c-validator-equal-power.json --workspace-dir "$HOME/cbw/manual/validator-equal-power" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**083. `commit-signers-quorum`** — stablenet · bp4 · 직전 PASS 52s  
RT-G-3-03 — istanbul_getCommitSignersFromBlock (원본 regression/api/13-test-get-commit-signers)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/commit-signers-quorum" && bin/chainbench run tests/tc/go-stablenet/regression/api/13-commit-signers-quorum.json --workspace-dir "$HOME/cbw/manual/commit-signers-quorum" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**084. `wbft-extra-info-fields`** — stablenet · bp4 · 직전 PASS 52s  
RT-G-3-04 — istanbul_getWbftExtraInfo (원본 regression/api/14-test-get-wbft-extra)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-extra-info-fields" && bin/chainbench run tests/tc/go-stablenet/regression/api/14-wbft-extra-info-fields.json --workspace-dir "$HOME/cbw/manual/wbft-extra-info-fields" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**085. `istanbul-status-fields`** — stablenet · bp4 · 직전 PASS 57s  
RT-G-3-05 — istanbul_status (원본 regression/api/15-test-istanbul-status)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/istanbul-status-fields" && bin/chainbench run tests/tc/go-stablenet/regression/api/15-istanbul-status-fields.json --workspace-dir "$HOME/cbw/manual/istanbul-status-fields" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**086. `is-validator-flags`** — stablenet · bp4 · 직전 PASS 57s  
RT-G-3-06 — istanbul_isValidator (원본 regression/api/16-test-is-validator)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/is-validator-flags" && bin/chainbench run tests/tc/go-stablenet/regression/api/16-is-validator-flags.json --workspace-dir "$HOME/cbw/manual/is-validator-flags" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**087. `txpool-status`** — stablenet · bp4 · 직전 PASS 56s  
RT-G-4-02 — txpool_status: pending(연속 nonce) + queued(nonce gap) 분리 (원본 regression/api/18-test-txpool-status)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/txpool-status" && bin/chainbench run tests/tc/go-stablenet/regression/api/18-txpool-status.json --workspace-dir "$HOME/cbw/manual/txpool-status" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**088. `txpool-content-well-formed`** — stablenet · bp4 · 직전 PASS 51s  
RT-G-4-03 — txpool_content: pending/queued 분리 내용 확인 (원본 regression/api/19-test-txpool-content)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/txpool-content-well-formed" && bin/chainbench run tests/tc/go-stablenet/regression/api/19-txpool-content-well-formed.json --workspace-dir "$HOME/cbw/manual/txpool-content-well-formed" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**089. `admin-peers-populated`** — stablenet · bp4 · 직전 PASS 51s  
RT-A-1-05 — P2P 피어 연결 확인 (원본 regression/ethereum/05-test-p2p-peers)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/admin-peers-populated" && bin/chainbench run tests/tc/go-stablenet/regression/api/20-admin-peers-populated.json --workspace-dir "$HOME/cbw/manual/admin-peers-populated" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**090. `fee-delegate-sign-rpc-present`** — stablenet · bp4 · 직전 PASS 51s  
RT-G-5-01 — eth_signRawFeeDelegateTransaction (원본 regression/api/21-test-sign-raw-fee-delegate)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fee-delegate-sign-rpc-present" && bin/chainbench run tests/tc/go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json --workspace-dir "$HOME/cbw/manual/fee-delegate-sign-rpc-present" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**091. `token-total-supply-readable`** — stablenet · bp4 · 직전 PASS 52s  
RT-G-5-02 — eth_call NativeCoinAdapter.totalSupply (원본 regression/api/22-test-total-supply)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/token-total-supply-readable" && bin/chainbench run tests/tc/go-stablenet/regression/api/22-token-total-supply-readable.json --workspace-dir "$HOME/cbw/manual/token-total-supply-readable" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**092. `token-approve-sets-allowance`** — stablenet · bp4 · 직전 PASS 56s  
RT-G-5-03 — eth_call NativeCoinAdapter.allowance (원본 regression/api/23-test-allowance)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/token-approve-sets-allowance" && bin/chainbench run tests/tc/go-stablenet/regression/api/23-token-approve-sets-allowance.json --workspace-dir "$HOME/cbw/manual/token-approve-sets-allowance" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**093. `chain-not-syncing`** — stablenet · bp4 · 직전 PASS 52s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/chain-not-syncing" && bin/chainbench run tests/tc/go-stablenet/regression/api/24-chain-not-syncing.json --workspace-dir "$HOME/cbw/manual/chain-not-syncing" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/regression/blacklist-authorized` (9)

**094. `sender-blacklisted-rejected`** — stablenet · bp4 · 직전 PASS 55s  
RT-E-01 — 블랙리스트 계정이 Sender인 tx 거부 (ErrBlacklistedAccount) (원본 regression/blacklist-authorized/01-test-sender-blacklisted)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/sender-blacklisted-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json --workspace-dir "$HOME/cbw/manual/sender-blacklisted-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**095. `recipient-blacklisted-rejected`** — stablenet · bp4 · 직전 PASS 55s  
RT-E-02 — 블랙리스트 계정이 Recipient인 tx 거부 (원본 regression/blacklist-authorized/02-test-recipient-blacklisted)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/recipient-blacklisted-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json --workspace-dir "$HOME/cbw/manual/recipient-blacklisted-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**096. `feepayer-blacklisted-rejected`** — stablenet · bp4 · 직전 PASS 55s  
RT-E-03 — FeePayer가 블랙리스트 계정이면 거부 (원본 regression/blacklist-authorized/03-test-feepayer-blacklisted)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/feepayer-blacklisted-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json --workspace-dir "$HOME/cbw/manual/feepayer-blacklisted-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**097. `address-unblacklisted-event`** — stablenet · bp4 · 직전 PASS 56s  
RT-E-04 — 블랙리스트 해제 거버넌스 흐름 검증 (원본 regression/blacklist-authorized/04-test-unblacklist)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/address-unblacklisted-event" && bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json --workspace-dir "$HOME/cbw/manual/address-unblacklisted-event" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**098. `zero-address-transfer-rejected`** — stablenet · bp4 · 직전 PASS 53s  
RT-E-05: 0x0 주소로의 값 전송은 EVM 실행에서 거부된다 (ErrZeroAddressTransfer) — 트랜잭션은 채굴되고 영수증 status 가 0 이다.  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/zero-address-transfer-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json --workspace-dir "$HOME/cbw/manual/zero-address-transfer-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**099. `precompile-transfer-rejected`** — stablenet · bp4 · 직전 PASS 56s  
RT-E-06: 프리컴파일·시스템 컨트랙트 주소로의 값 전송은 5개 주소 모두 EVM 실행에서 거부된다 — 트랜잭션은 채굴되고 영수증 status 가 0 이다.  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/precompile-transfer-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json --workspace-dir "$HOME/cbw/manual/precompile-transfer-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**100. `account-blacklist-readable`** — stablenet · bp4 · 직전 PASS 52s  
RT-E-07 — AccountManager.isBlacklisted() 조회 검증 (원본 regression/blacklist-authorized/07-test-is-blacklisted)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/account-blacklist-readable" && bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json --workspace-dir "$HOME/cbw/manual/account-blacklist-readable" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**101. `account-authorization-readable`** — stablenet · bp4 · 직전 PASS 52s  
RT-E-08 — AccountManager.isAuthorized() 조회 검증 (원본 regression/blacklist-authorized/08-test-is-authorized)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/account-authorization-readable" && bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json --workspace-dir "$HOME/cbw/manual/account-authorization-readable" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**102. `authorized-tx-executed-event`** — stablenet · bp4 · 직전 PASS 56s  
RT-E-09 — AuthorizedTxExecuted 이벤트 발생 검증 (원본 regression/blacklist-authorized/09-test-authorized-tx-executed)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/authorized-tx-executed-event" && bin/chainbench run tests/tc/go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json --workspace-dir "$HOME/cbw/manual/authorized-tx-executed-event" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/regression/ethereum` (26)

**103. `stablenet-chain-up`** — stablenet · bp4 · 직전 PASS 52s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stablenet-chain-up" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/01-stablenet-chain-up.json --workspace-dir "$HOME/cbw/manual/stablenet-chain-up" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**104. `legacy-transfer`** — stablenet · bp4 · 직전 PASS 53s  
RT-A-2-01 — Legacy Tx (type 0x0) 발행 (원본 regression/ethereum/08-test-legacy-tx)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/legacy-transfer" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/08-legacy-transfer.json --workspace-dir "$HOME/cbw/manual/legacy-transfer" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**105. `dynamic-fee-tx`** — stablenet · bp4 · 직전 PASS 57s  
RT-A-2-02 — EIP-1559 DynamicFeeTx (type 0x2) 발행 (원본 regression/ethereum/09-test-dynamic-fee-tx)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/dynamic-fee-tx" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/09-dynamic-fee-tx.json --workspace-dir "$HOME/cbw/manual/dynamic-fee-tx" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**106. `access-list-tx`** — stablenet · bp4 · 직전 PASS 56s  
RT-A-2-03: eth_createAccessList 로 노드가 만든 접근 목록을 붙여 type 0x01 트랜잭션을 보낸다.  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/access-list-tx" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/10-access-list-tx.json --workspace-dir "$HOME/cbw/manual/access-list-tx" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**107. `nonce-ordering`** — stablenet · bp4 · 직전 PASS 56s  
RT-A-2-04 — Nonce 순서 보장 (원본 regression/ethereum/11-test-nonce-ordering)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/nonce-ordering" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/11-nonce-ordering.json --workspace-dir "$HOME/cbw/manual/nonce-ordering" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**108. `out-of-order-nonces-mine`** — stablenet · bp4 · 직전 PASS 56s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/out-of-order-nonces-mine" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json --workspace-dir "$HOME/cbw/manual/out-of-order-nonces-mine" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**109. `dynamic-fee-below-basefee-rejected`** — stablenet · bp4 · 직전 PASS 51s  
RT-A-2-05a — GasTipCap < MinTip tx 거부 검증 (원본 regression/ethereum/12-test-tipcap-underpriced)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/dynamic-fee-below-basefee-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json --workspace-dir "$HOME/cbw/manual/dynamic-fee-below-basefee-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**110. `insufficient-funds-rejected`** — stablenet · bp4 · 직전 PASS 52s  
RT-A-2-06 — 잔액 부족 tx 거부 (원본 regression/ethereum/14-test-insufficient-funds)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/insufficient-funds-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json --workspace-dir "$HOME/cbw/manual/insufficient-funds-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**111. `gas-limit-exceeds-block-rejected`** — stablenet · bp4 · 직전 PASS 52s  
RT-A-2-07 — Gas Limit 초과 tx 거부 (블록 gas limit 초과) (원본 regression/ethereum/15-test-gaslimit-exceeded)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/gas-limit-exceeds-block-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json --workspace-dir "$HOME/cbw/manual/gas-limit-exceeds-block-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**112. `effective-gas-price`** — stablenet · bp4 · 직전 PASS 56s  
RT-A-2-08 — eth_getTransactionReceipt의 effectiveGasPrice 검증 (원본 regression/ethereum/16-test-effective-gas-price)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/effective-gas-price" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/16-effective-gas-price.json --workspace-dir "$HOME/cbw/manual/effective-gas-price" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**113. `replacement-tx`** — stablenet · bp4 · 직전 PASS 56s  
RT-A-2-09 — 동일 nonce, 더 높은 GasFeeCap으로 tx 교체 (원본 regression/ethereum/17-test-replacement-tx)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/replacement-tx" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/17-replacement-tx.json --workspace-dir "$HOME/cbw/manual/replacement-tx" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**114. `same-nonce-replacement`** — stablenet · bp4 · 직전 PASS 54s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/same-nonce-replacement" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/17b-same-nonce-replacement.json --workspace-dir "$HOME/cbw/manual/same-nonce-replacement" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**115. `set-code-delegation`** — stablenet · bp4 · 직전 PASS 53s  
RT-A-2-10 — SetCodeTx (type 0x4 / EIP-7702) 계정 코드 위임 (원본 regression/ethereum/18-test-setcode-tx)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/set-code-delegation" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json --workspace-dir "$HOME/cbw/manual/set-code-delegation" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**116. `contract-roundtrip`** — stablenet · bp4 · 직전 PASS 53s  
RT-A-3-01 — 컨트랙트 배포 (원본 regression/ethereum/19-test-contract-deploy)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/contract-roundtrip" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/19-contract-roundtrip.json --workspace-dir "$HOME/cbw/manual/contract-roundtrip" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**117. `estimate-gas`** — stablenet · bp4 · 직전 PASS 52s  
RT-A-3-04 — eth_estimateGas 정상 동작 (원본 regression/ethereum/22-test-estimate-gas)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/estimate-gas" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/22-estimate-gas.json --workspace-dir "$HOME/cbw/manual/estimate-gas" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**118. `eth-call-revert-returns-error`** — stablenet · bp4 · 직전 PASS 52s  
RT-A-3-05 — eth_call로 revert하는 함수 호출 시 에러 반환 (원본 regression/ethereum/23-test-eth-call-revert)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/eth-call-revert-returns-error" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json --workspace-dir "$HOME/cbw/manual/eth-call-revert-returns-error" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**119. `revert-tx-status-zero`** — stablenet · bp4 · 직전 PASS 53s  
RT-A-3-06 — revert tx: receipt.status == 0x0, gasUsed만 차감 검증 (원본 regression/ethereum/24-test-revert-tx)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/revert-tx-status-zero" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/24-revert-tx-status-zero.json --workspace-dir "$HOME/cbw/manual/revert-tx-status-zero" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**120. `out-of-gas-consumes-all`** — stablenet · bp4 · 직전 PASS 53s  
RT-A-3-07 — out-of-gas tx: gasUsed == gasLimit, 잔액 전량 차감 검증 (원본 regression/ethereum/25-test-out-of-gas)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/out-of-gas-consumes-all" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json --workspace-dir "$HOME/cbw/manual/out-of-gas-consumes-all" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**121. `genesis-balance`** — stablenet · bp4 · 직전 PASS 57s  
RT-A-4-02 — eth_getBalance 정상 조회 (원본 regression/ethereum/27-test-eth-get-balance)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/genesis-balance" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/27-genesis-balance.json --workspace-dir "$HOME/cbw/manual/genesis-balance" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**122. `value-transfer`** — stablenet · bp4 · 직전 PASS 57s  
RT-A-4-03 — eth_sendRawTransaction 서명된 tx 전파 (원본 regression/ethereum/28-test-send-raw-tx)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/value-transfer" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/27b-value-transfer.json --workspace-dir "$HOME/cbw/manual/value-transfer" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**123. `logs-query-well-formed`** — stablenet · bp4 · 직전 PASS 52s  
RT-A-4-04 — eth_getLogs 이벤트 로그 조회 (원본 regression/ethereum/29-test-eth-get-logs)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/logs-query-well-formed" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/29-logs-query-well-formed.json --workspace-dir "$HOME/cbw/manual/logs-query-well-formed" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**124. `chain-id`** — stablenet · bp4 · 직전 PASS 54s  
RT-A-1-01 — 제네시스 블록으로 노드 초기화 (원본 regression/ethereum/01-test-genesis-init)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/chain-id" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/30-chain-id.json --workspace-dir "$HOME/cbw/manual/chain-id" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**125. `ws-subscribe-new-heads`** — stablenet · bp4 · 직전 PASS 53s  
RT-A-4-06 — eth_subscribe(newHeads) WebSocket 구독 (원본 regression/ethereum/31-test-ws-subscribe-heads)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/ws-subscribe-new-heads" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json --workspace-dir "$HOME/cbw/manual/ws-subscribe-new-heads" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**126. `ws-subscribe-logs`** — stablenet · bp4 · 직전 PASS 54s  
RT-A-4-07 — eth_subscribe(logs) WebSocket 구독 (원본 regression/ethereum/32-test-ws-subscribe-logs)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/ws-subscribe-logs" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/32-ws-subscribe-logs.json --workspace-dir "$HOME/cbw/manual/ws-subscribe-logs" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**127. `stablenet-chain-up-15`** — stablenet · bp7 en7 pn1 (proxied) · 직전 PASS 54s  
The standard 15-node shape brought up and producing: 7 bp, 7 en and 1 pn. Endpoints reach the producers through the pn rather than dialli…  
> 키를 새로 만든다(선언이 generate)

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stablenet-chain-up-15" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json --workspace-dir "$HOME/cbw/manual/stablenet-chain-up-15" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**128. `contract-event-emitted`** — stablenet · bp4 · 직전 PASS 58s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/contract-event-emitted" && bin/chainbench run tests/tc/go-stablenet/regression/ethereum/36-contract-event-emitted.json --workspace-dir "$HOME/cbw/manual/contract-event-emitted" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/regression/fee-delegation` (7)

**129. `fee-delegated-transfer`** — stablenet · bp4 · 직전 PASS 54s  
RT-D-01 — FeeDelegateDynamicFeeTx (type 0x16) 정상 처리 (원본 regression/fee-delegation/01-test-fee-delegate-normal)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fee-delegated-transfer" && bin/chainbench run tests/tc/go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json --workspace-dir "$HOME/cbw/manual/fee-delegated-transfer" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**130. `fd-sender-sig-invalid-rejected`** — stablenet · bp4 · 직전 PASS 57s  
RT-D-03 — Sender 서명 변조 시 거부 (원본 regression/fee-delegation/02-test-sender-sig-invalid)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fd-sender-sig-invalid-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json --workspace-dir "$HOME/cbw/manual/fd-sender-sig-invalid-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**131. `fd-feepayer-sig-invalid-rejected`** — stablenet · bp4 · 직전 PASS 53s  
RT-D-04 — FeePayer 서명 변조 시 거부 (원본 regression/fee-delegation/03-test-feepayer-sig-invalid)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fd-feepayer-sig-invalid-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json --workspace-dir "$HOME/cbw/manual/fd-feepayer-sig-invalid-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**132. `feepayer-insufficient-rejected`** — stablenet · bp4 · 직전 PASS 55s  
RT-D-05 — FeePayer 잔액 부족 시 거부 (원본 regression/fee-delegation/04-test-feepayer-insufficient)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/feepayer-insufficient-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json --workspace-dir "$HOME/cbw/manual/feepayer-insufficient-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**133. `fee-delegated-sender-sig-invalid-rejected`** — stablenet · bp4 · 직전 PASS 52s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fee-delegated-sender-sig-invalid-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json --workspace-dir "$HOME/cbw/manual/fee-delegated-sender-sig-invalid-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**134. `fee-delegated-feepayer-sig-invalid-rejected`** — stablenet · bp4 · 직전 PASS 53s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fee-delegated-feepayer-sig-invalid-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json --workspace-dir "$HOME/cbw/manual/fee-delegated-feepayer-sig-invalid-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**135. `fee-delegated-unfunded-feepayer-rejected`** — stablenet · bp4 · 직전 PASS 56s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/fee-delegated-unfunded-feepayer-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json --workspace-dir "$HOME/cbw/manual/fee-delegated-unfunded-feepayer-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/regression/system-contracts` (23)

**136. `native-coin-adapter-code`** — stablenet · bp4 · 직전 PASS 52s  
RT-F-1-01 — NativeCoinAdapter.transfer → 기본 코인 전송과 동일 (원본 regression/system-contracts/01-test-native-transfer)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/native-coin-adapter-code" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json --workspace-dir "$HOME/cbw/manual/native-coin-adapter-code" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**137. `token-transfer-emits-event`** — stablenet · bp4 · 직전 PASS 53s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/token-transfer-emits-event" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json --workspace-dir "$HOME/cbw/manual/token-transfer-emits-event" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**138. `token-balance-readable`** — stablenet · bp4 · 직전 PASS 52s  
RT-F-1-02 — NativeCoinAdapter.balanceOf == eth_getBalance (원본 regression/system-contracts/02-test-balance-of)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/token-balance-readable" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/02-token-balance-readable.json --workspace-dir "$HOME/cbw/manual/token-balance-readable" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**139. `token-transfer-from-moves-balance`** — stablenet · bp4 · 직전 PASS 56s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/token-transfer-from-moves-balance" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json --workspace-dir "$HOME/cbw/manual/token-transfer-from-moves-balance" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**140. `mint-transfer-event`** — stablenet · bp4 · 직전 PASS 54s  
RT-F-1-04 — Mint 실행 시 Transfer(0x0 → beneficiary) 이벤트 발생 (원본 regression/system-contracts/04-test-mint-transfer-event)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/mint-transfer-event" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/04-mint-transfer-event.json --workspace-dir "$HOME/cbw/manual/mint-transfer-event" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**141. `burn-transfer-event`** — stablenet · bp4 · 직전 PASS 52s  
RT-F-1-05 — Burn 실행 시 Transfer(account → 0x0) 이벤트 발생 (원본 regression/system-contracts/05-test-burn-transfer-event)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/burn-transfer-event" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/05-burn-transfer-event.json --workspace-dir "$HOME/cbw/manual/burn-transfer-event" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**142. `mint-proposal-executes`** — stablenet · bp4 · 직전 PASS 53s  
RT-F-2-01 — 코인 발행: proposeMint(proofData) → 승인 → execute (원본 regression/system-contracts/06-test-mint-proposal)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/mint-proposal-executes" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/06-mint-proposal-executes.json --workspace-dir "$HOME/cbw/manual/mint-proposal-executes" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**143. `burn-proposal-executes`** — stablenet · bp4 · 직전 PASS 53s  
RT-F-2-02 — 코인 소각: proposeBurn(proofData) payable → 승인 → execute (원본 regression/system-contracts/07-test-burn-proposal)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/burn-proposal-executes" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/07-burn-proposal-executes.json --workspace-dir "$HOME/cbw/manual/burn-proposal-executes" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**144. `quorum-deficient-stays-voting`** — stablenet · bp4 · 직전 PASS 55s  
RT-F-2-03 — quorum 미달 → proposal 상태 Voting 유지, 발행 미실행 (원본 regression/system-contracts/08-test-quorum-deficient)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/quorum-deficient-stays-voting" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json --workspace-dir "$HOME/cbw/manual/quorum-deficient-stays-voting" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**145. `validator-metadata-readable`** — stablenet · bp4 · 직전 PASS 55s  
RT-F-3-04 — validatorList() + validatorToOperator(v) + validatorToBlsKey(v) 다중 호출 (원본 regression/system-contracts/09-test-validator-metad…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/validator-metadata-readable" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/09-validator-metadata-readable.json --workspace-dir "$HOME/cbw/manual/validator-metadata-readable" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**146. `gastip-governance-updates-header`** — stablenet · bp4 · 직전 PASS 57s  
RT-B-06 — GasTip 거버넌스 변경 → 블록 헤더 WBFTExtra.GasTip 반영 검증 + 원복 (원본 regression/wbft/06-test-gastip-header-sync)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/gastip-governance-updates-header" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json --workspace-dir "$HOME/cbw/manual/gastip-governance-updates-header" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**147. `proposal-expiry-transitions`** — stablenet · bp4 · 직전 PASS 93s  
RT-F-3-06 — proposal expiry 초과 → Expired 상태 전환 → execute 불가 (원본 regression/system-contracts/11-test-proposal-expiry)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/proposal-expiry-transitions" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json --workspace-dir "$HOME/cbw/manual/proposal-expiry-transitions" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**148. `configure-minter-proposal-executes`** — stablenet · bp4 · 직전 PASS 54s  
RT-F-4-01 — GovMasterMinter.proposeConfigureMinter(address, uint256) → 승인 → execute (원본 regression/system-contracts/12-test-add-minter)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/configure-minter-proposal-executes" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json --workspace-dir "$HOME/cbw/manual/configure-minter-proposal-executes" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**149. `remove-minter-executes`** — stablenet · bp4 · 직전 PASS 56s  
RT-F-4-02 — GovMasterMinter.proposeRemoveMinter(address) → 승인 → execute (원본 regression/system-contracts/13-test-remove-minter)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/remove-minter-executes" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/13-remove-minter-executes.json --workspace-dir "$HOME/cbw/manual/remove-minter-executes" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**150. `masterminter-member-add-remove`** — stablenet · bp4 · 직전 PASS 56s  
RT-F-4-03 — GovMasterMinter 자체 멤버 추가/제거 (proposeAddMember, proposeRemoveMember) (원본 regression/system-contracts/14-test-masterminter-self…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/masterminter-member-add-remove" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json --workspace-dir "$HOME/cbw/manual/masterminter-member-add-remove" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**151. `non-member-configure-minter-rejected`** — stablenet · bp4 · 직전 PASS 53s  
RT-F-4-04 — 비멤버 계정의 GovMasterMinter.proposeConfigureMinter 호출 거부 (원본 regression/system-contracts/15-test-non-member-rejected)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/non-member-configure-minter-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json --workspace-dir "$HOME/cbw/manual/non-member-configure-minter-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**152. `blacklist-proposal-executes`** — stablenet · bp4 · 직전 PASS 55s  
RT-F-5-01 — GovCouncil blacklist proposal → execute → isBlacklisted == true (원본 regression/system-contracts/16-test-blacklist)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/blacklist-proposal-executes" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json --workspace-dir "$HOME/cbw/manual/blacklist-proposal-executes" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**153. `authorize-proposal-executes`** — stablenet · bp4 · 직전 PASS 53s  
RT-F-5-03 — GovCouncil authorized account proposal → execute → isAuthorized == true (원본 regression/system-contracts/18-test-authorize)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/authorize-proposal-executes" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json --workspace-dir "$HOME/cbw/manual/authorize-proposal-executes" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**154. `unauthorize-proposal-executes`** — stablenet · bp4 · 직전 PASS 55s  
f5-04-unauthorize (원본 regression/system-contracts/19-test-unauthorize)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/unauthorize-proposal-executes" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json --workspace-dir "$HOME/cbw/manual/unauthorize-proposal-executes" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**155. `direct-blacklist-call-rejected`** — stablenet · bp4 · 직전 PASS 54s  
RT-F-5-05 — 비멤버가 AccountManager.blacklist 직접 호출 → revert (원본 regression/system-contracts/20-test-direct-blacklist-rejected)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/direct-blacklist-call-rejected" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json --workspace-dir "$HOME/cbw/manual/direct-blacklist-call-rejected" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**156. `authorized-account-added-event`** — stablenet · bp4 · 직전 PASS 54s  
RT-F-5-08 — AuthorizedAccountAdded 이벤트 (원본 regression/system-contracts/23-test-authorized-account-added-event)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/authorized-account-added-event" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/23-authorized-account-added-event.json --workspace-dir "$HOME/cbw/manual/authorized-account-added-event" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**157. `token-metadata`** — stablenet · bp4 · 직전 PASS 51s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/token-metadata" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/25-token-metadata.json --workspace-dir "$HOME/cbw/manual/token-metadata" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**158. `minter-status-readable`** — stablenet · bp4 · 직전 PASS 51s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/minter-status-readable" && bin/chainbench run tests/tc/go-stablenet/regression/system-contracts/28-minter-status-readable.json --workspace-dir "$HOME/cbw/manual/minter-status-readable" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/regression/wbft` (9)

**159. `block-period-one-second`** — stablenet · bp4 · 직전 PASS 56s  
RT-B-01 — 블록 생산 주기 1초 간격 (원본 regression/wbft/01-test-block-period)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/block-period-one-second" && bin/chainbench run tests/tc/go-stablenet/regression/wbft/01-block-period-one-second.json --workspace-dir "$HOME/cbw/manual/block-period-one-second" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**160. `wbft-seals-quorum`** — stablenet · bp4 · 직전 PASS 56s  
RT-B-02 — WBFTExtra에 Committed Seal + Prepared Seal 모두 존재하고 quorum 이상 (원본 regression/wbft/02-test-wbft-extra-seal)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-seals-quorum" && bin/chainbench run tests/tc/go-stablenet/regression/wbft/02-wbft-seals-quorum.json --workspace-dir "$HOME/cbw/manual/wbft-seals-quorum" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**161. `epoch-transition-carries-epoch-info`** — stablenet · bp4 · 직전 PASS 155s  
RT-B-03 — 에폭 전환 — 검증자 집합 갱신 (원본 regression/wbft/03-test-epoch-transition)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/epoch-transition-carries-epoch-info" && bin/chainbench run tests/tc/go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json --workspace-dir "$HOME/cbw/manual/epoch-transition-carries-epoch-info" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**162. `validator-add-member-executes`** — stablenet · bp4 · 직전 PASS 54s  
RT-B-04 — 신규 검증자 추가 및 에폭 합의 참여 확인 (원본 regression/wbft/04-test-add-validator)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/validator-add-member-executes" && bin/chainbench run tests/tc/go-stablenet/regression/wbft/04-validator-add-member-executes.json --workspace-dir "$HOME/cbw/manual/validator-add-member-executes" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**163. `validator-add-member-epoch-activates`** — stablenet · bp4 en1 · 직전 PASS 86s  
RT-B-04 — configureValidator 로 등록된 노드가 에폭 경계에서 실제 검증자 집합에 들어간다. 멤버 추가만으로는 검증자가 되지 않는다(GovValidator._onMemberAdded 는 아무것도 하지 않는다). 검증자 등록은…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/validator-add-member-epoch-activates" && bin/chainbench run tests/tc/go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json --workspace-dir "$HOME/cbw/manual/validator-add-member-epoch-activates" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**164. `validator-remove-member-executes`** — stablenet · bp4 · 직전 PASS 56s  
RT-B-05: proposeRemoveMember 가 실행되면 GovValidator 멤버에서 빠진다. 제거 대상을 먼저 추가해 자기 완결로 만든다. 정족수는 2로 둔다 — 승인이 node1, node2 둘뿐이므로 추가할 때 정족수를 올려 두면…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/validator-remove-member-executes" && bin/chainbench run tests/tc/go-stablenet/regression/wbft/05-validator-remove-member-executes.json --workspace-dir "$HOME/cbw/manual/validator-remove-member-executes" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**165. `prev-seals-quorum`** — stablenet · bp4 · 직전 PASS 57s  
RT-B-11 — 블록 N+1의 PrevCommittedSeal이 블록 N의 committers를 포함 (원본 regression/wbft/11-test-prev-committed-seal)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/prev-seals-quorum" && bin/chainbench run tests/tc/go-stablenet/regression/wbft/11-prev-seals-quorum.json --workspace-dir "$HOME/cbw/manual/prev-seals-quorum" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**166. `randao-and-mixdigest-present`** — stablenet · bp4 · 직전 PASS 54s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/randao-and-mixdigest-present" && bin/chainbench run tests/tc/go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json --workspace-dir "$HOME/cbw/manual/randao-and-mixdigest-present" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**167. `stablenet-gastip-field`** — stablenet · bp4 · 직전 PASS 56s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stablenet-gastip-field" && bin/chainbench run tests/tc/go-stablenet/regression/wbft/14-stablenet-gastip-field.json --workspace-dir "$HOME/cbw/manual/stablenet-gastip-field" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/topology` (1)

**168. `stablenet-proxied-pn-routing`** — stablenet · bp3 en1 pn1 (proxied) · 직전 PASS 58s  
Compose the proxied tier (bp <-> pn <-> en) and prove the endpoint stays in sync through the pn: under proxied peering an endpoint never …  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stablenet-proxied-pn-routing" && bin/chainbench run tests/tc/go-stablenet/topology/01-proxied-pn-routing.json --workspace-dir "$HOME/cbw/manual/stablenet-proxied-pn-routing" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/tx` (1)

**169. `stablenet-negative-tx-revert`** — stablenet · bp4 · 직전 PASS 57s  
Negative path (WA25): deploy a contract whose runtime always reverts, then send it a transaction and require that it reverts (mined with …  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stablenet-negative-tx-revert" && bin/chainbench run tests/tc/go-stablenet/tx/01-negative-tx-revert.json --workspace-dir "$HOME/cbw/manual/stablenet-negative-tx-revert" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-stablenet/vocabulary` (4)

**170. `stablenet-derived-vocabulary`** — stablenet · bp4 · 직전 PASS 52s  
Exercise the pure-derivation vocabulary that no case used before (WA24): createAddress computes the CREATE address from a deployer and no…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stablenet-derived-vocabulary" && bin/chainbench run tests/tc/go-stablenet/vocabulary/01-derived-address-and-checksum.json --workspace-dir "$HOME/cbw/manual/stablenet-derived-vocabulary" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**171. `stablenet-faucet-funds`** — stablenet · bp4 · 직전 PASS 52s  
Exercise the faucet action (WA24): top up a freshly generated account and read back its balance. LIVE-UNVERIFIED: authored offline (passe…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stablenet-faucet-funds" && bin/chainbench run tests/tc/go-stablenet/vocabulary/02-faucet-funds-account.json --workspace-dir "$HOME/cbw/manual/stablenet-faucet-funds" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**172. `stablenet-metric-head-block`** — stablenet · bp4 · 직전 PASS 51s  
Exercise the metric assertion (WA24): read the node's chain_head_block Prometheus metric once the chain has advanced. LIVE-VERIFIED 2026-…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stablenet-metric-head-block" && bin/chainbench run tests/tc/go-stablenet/vocabulary/03-metric-head-block.json --workspace-dir "$HOME/cbw/manual/stablenet-metric-head-block" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**173. `stablenet-register-contract`** — stablenet · bp4 · 직전 PASS 54s  
registerContract against a contract with a real ABI and real storage. The earlier version of this case deployed a stub that accepted any …  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stablenet-register-contract" && bin/chainbench run tests/tc/go-stablenet/vocabulary/03-register-contract.json --workspace-dir "$HOME/cbw/manual/stablenet-register-contract" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `go-wbft/accounts` (3)

**174. `secp256r1-precompile-valid`** — wbft · bp4 · 직전 PASS 54s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/secp256r1-precompile-valid" && bin/chainbench run tests/tc/go-wbft/accounts/01-secp256r1-precompile-valid.json --workspace-dir "$HOME/cbw/manual/secp256r1-precompile-valid" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```

**175. `secp256r1-precompile-invalid`** — wbft · bp4 · 직전 PASS 51s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/secp256r1-precompile-invalid" && bin/chainbench run tests/tc/go-wbft/accounts/02-secp256r1-precompile-invalid.json --workspace-dir "$HOME/cbw/manual/secp256r1-precompile-invalid" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```

**176. `secp256r1-precompile-short-input`** — wbft · bp4 · 직전 PASS 52s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/secp256r1-precompile-short-input" && bin/chainbench run tests/tc/go-wbft/accounts/03-secp256r1-precompile-short-input.json --workspace-dir "$HOME/cbw/manual/secp256r1-precompile-short-input" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```


### `go-wbft/chain-up` (2)

**177. `wbft-chain-up`** — wbft · bp4 · 직전 PASS 56s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-chain-up" && GWBFT_BIN="$HOME/work/github/wemade/go-wbft/build/bin/gwemix" bin/chainbench run tests/tc/go-wbft/chain-up/01-wbft-chain-up.json --workspace-dir "$HOME/cbw/manual/wbft-chain-up"
```

**178. `wbft-chain-up-15`** — wbft · bp7 en7 pn1 (proxied) · 직전 PASS 56s  
The standard 15-node shape brought up and producing: 7 bp, 7 en and 1 pn. Endpoints reach the producers through the pn rather than dialli…  
> 키를 새로 만든다(선언이 generate)

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-chain-up-15" && bin/chainbench run tests/tc/go-wbft/chain-up/02-wbft-chain-up-15.json --workspace-dir "$HOME/cbw/manual/wbft-chain-up-15" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```


### `go-wbft/consensus` (2)

**179. `e1-mixed-producers`** — stablenet · bp3 en1 · 직전 PASS 55s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/e1-mixed-producers" && bin/chainbench run tests/tc/go-wbft/consensus/01-e1-mixed-producers.json --workspace-dir "$HOME/cbw/manual/e1-mixed-producers" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**180. `wbft-quorum-halt-and-recover`** — wbft · bp4 · 직전 PASS 98s  
Take a wbft network below quorum and back: with 4 validators the quorum is floor(2n/3)+1 = 3, so stopping two must halt production and re…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-quorum-halt-and-recover" && bin/chainbench run tests/tc/go-wbft/consensus/02-wbft-quorum-halt-and-recover.json --workspace-dir "$HOME/cbw/manual/wbft-quorum-halt-and-recover" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```


### `go-wbft/fault` (2)

**181. `wbft-node-crash`** — wbft · bp4 · 직전 PASS 56s  
Stop one wbft validator and verify consensus continues 3/4, then restart it (WA25 — wbft had no fault coverage).  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-node-crash" && bin/chainbench run tests/tc/go-wbft/fault/01-wbft-node-crash.json --workspace-dir "$HOME/cbw/manual/wbft-node-crash" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```

**182. `wbft-quorum-at-15-nodes`** — wbft · bp7 en7 pn1 (proxied) · 직전 PASS 129s  
The quorum boundary at the standard 15-node shape. With 7 bp the quorum is floor(2n/3)+1 = 5, so five producing must be enough and four m…  
> 키를 새로 만든다(선언이 generate)

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-quorum-at-15-nodes" && bin/chainbench run tests/tc/go-wbft/fault/02-wbft-quorum-at-15-nodes.json --workspace-dir "$HOME/cbw/manual/wbft-quorum-at-15-nodes" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```


### `go-wbft/governance` (2)

**183. `wbft-govcontracts-at-genesis`** — wbft · bp4 · 직전 PASS 53s  
A fresh wbft chain carries its governance contracts from genesis, and nobody has staked in them. The empty staker set is not an incidenta…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-govcontracts-at-genesis" && bin/chainbench run tests/tc/go-wbft/governance/01-wbft-govcontracts-at-genesis.json --workspace-dir "$HOME/cbw/manual/wbft-govcontracts-at-genesis" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```

**184. `wbft-governance-register-staker`** — wbft · bp4 · 직전 PASS 54s  
A governance WRITE on go-wbft. The read case (governance/01) pins what genesis carries; this one changes governance state and checks the …  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-governance-register-staker" && bin/chainbench run tests/tc/go-wbft/governance/02-wbft-governance-register-staker.json --workspace-dir "$HOME/cbw/manual/wbft-governance-register-staker" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```


### `go-wbft/network` (1)

**185. `wbft-proxied-routing`** — wbft · bp2 en1 pn1 (proxied) · 직전 PASS 33s  
WA25: verify the proxied peer graph's SHAPE — bp <-> bp direct, bp <-> pn and pn <-> en through the tier, and an endpoint that never lear…  
> 키를 새로 만든다(선언이 generate)

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-proxied-routing" && bin/chainbench run tests/tc/go-wbft/network/01-wbft-proxied-routing.json --workspace-dir "$HOME/cbw/manual/wbft-proxied-routing" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```


### `go-wbft/tx` (4)

**186. `wbft-tx-and-contract`** — wbft · bp4 · 직전 PASS 55s  
wbft value transfer and contract deploy/call: fund a fresh account, deploy a returner contract, and read it back (WA25 — wbft had no tx c…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-tx-and-contract" && bin/chainbench run tests/tc/go-wbft/tx/01-wbft-tx-and-contract.json --workspace-dir "$HOME/cbw/manual/wbft-tx-and-contract" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```

**187. `wbft-insufficient-funds-rejected`** — wbft · bp4 · 직전 PASS 54s  
WA25: a transfer above the sender's balance is refused before it reaches a block. Proven on stablenet (regression/ethereum/14); go-wbft n…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-insufficient-funds-rejected" && bin/chainbench run tests/tc/go-wbft/tx/02-wbft-insufficient-funds-rejected.json --workspace-dir "$HOME/cbw/manual/wbft-insufficient-funds-rejected" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```

**188. `wbft-revert-status-zero`** — wbft · bp4 · 직전 PASS 55s  
WA25: a call into a contract whose code reverts is mined with status 0x0 rather than vanishing or failing the block. Deploy uses runtime …  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-revert-status-zero" && bin/chainbench run tests/tc/go-wbft/tx/03-wbft-revert-status-zero.json --workspace-dir "$HOME/cbw/manual/wbft-revert-status-zero" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```

**189. `wbft-tx-crosses-the-proxied-topology`** — wbft · bp7 en7 pn1 (proxied) · 직전 PASS 60s  
A transaction submitted at the EDGE of the standard 15-node proxied network must reach the producers and come back as state every node ag…  
> 키를 새로 만든다(선언이 generate)

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wbft-tx-crosses-the-proxied-topology" && bin/chainbench run tests/tc/go-wbft/tx/04-wbft-tx-crosses-the-proxied-topology.json --workspace-dir "$HOME/cbw/manual/wbft-tx-crosses-the-proxied-topology" --binary "$HOME/work/github/wemade/go-wbft/build/bin/gwemix"
```


### `go-wemix/chain-up` (2)

**190. `wemix-chain-up`** — wemix · bp4 · 직전 PASS 103s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wemix-chain-up" && GWEMIX_BIN="$HOME/work/github/wemade/go-wemix/build/bin/gwemix" bin/chainbench run tests/tc/go-wemix/chain-up/01-wemix-chain-up.json --workspace-dir "$HOME/cbw/manual/wemix-chain-up"
```

**191. `wemix-chain-up-15`** — wemix · bp7 en7 pn1 (proxied) · 직전 BLOCKED 672s  
The standard 15-node shape brought up and producing: 7 bp, 7 en and 1 pn. Endpoints reach the producers through the pn rather than dialli…  
> 키를 새로 만든다(선언이 generate) · docker 에서 BLOCKED — go-wemix 코어 결함(worklist G6)으로 endpoint 하나가 합류하지 못한다

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wemix-chain-up-15" && bin/chainbench run tests/tc/go-wemix/chain-up/02-wemix-chain-up-15.json --workspace-dir "$HOME/cbw/manual/wemix-chain-up-15" --binary "$HOME/work/github/wemade/go-wemix/build/bin/gwemix"
```


### `go-wemix/consensus` (1)

**192. `wemix-etcd-and-governance`** — wemix · bp4 · 직전 PASS 104s  
A go-wemix (poa) network is only healthy if its etcd cluster formed and every governance member is producing. Blocks advancing shows neit…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wemix-etcd-and-governance" && bin/chainbench run tests/tc/go-wemix/consensus/01-wemix-etcd-and-governance.json --workspace-dir "$HOME/cbw/manual/wemix-etcd-and-governance" --binary "$HOME/work/github/wemade/go-wemix/build/bin/gwemix"
```


### `go-wemix/fault` (1)

**193. `wemix-node-crash`** — wemix · bp4 · 직전 PASS 144s  
Stop one wemix (poa) producer and verify block production continues, then restart it (WA25 — wemix had no fault coverage).  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wemix-node-crash" && bin/chainbench run tests/tc/go-wemix/fault/01-wemix-node-crash.json --workspace-dir "$HOME/cbw/manual/wemix-node-crash" --binary "$HOME/work/github/wemade/go-wemix/build/bin/gwemix"
```


### `go-wemix/governance` (1)

**194. `wemix-governance-staking-deposit`** — wemix · bp4 · 직전 PASS 109s  
A governance WRITE on go-wemix (poa). It is a different shape from the wbft one, which is what the worklist's B item warned about: wbft's…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wemix-governance-staking-deposit" && bin/chainbench run tests/tc/go-wemix/governance/01-wemix-governance-staking-deposit.json --workspace-dir "$HOME/cbw/manual/wemix-governance-staking-deposit" --binary "$HOME/work/github/wemade/go-wemix/build/bin/gwemix"
```


### `go-wemix/hardfork` (3)

**195. `croissant-successors-take-over`** — wemix · bp1 en4 · 직전 PASS 80s  
A wemix network composed on the ORDINARY path crosses the croissant fork: the wemix producer seals to the block before it and stops, and …  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/croissant-successors-take-over" && GWEMIX_BIN="$HOME/work/github/wemade/go-wemix/build/bin/gwemix" GWBFT_BIN="$HOME/work/github/wemade/go-wbft/build/bin/gwemix" bin/chainbench run tests/tc/go-wemix/hardfork/01-croissant-successors-take-over.json --workspace-dir "$HOME/cbw/manual/croissant-successors-take-over"
```

**196. `state-written-before-the-fork-survives-it`** — wemix · bp1 en4 · 직전 PASS 173s  
State written while the PRE-fork binary was sealing must still be there once the successors produce. The case crosses the fork itself, be…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/state-written-before-the-fork-survives-it" && GWEMIX_BIN="$HOME/work/github/wemade/go-wemix/build/bin/gwemix" GWBFT_BIN="$HOME/work/github/wemade/go-wbft/build/bin/gwemix" bin/chainbench run tests/tc/go-wemix/hardfork/02-state-written-before-the-fork-survives-it.json --workspace-dir "$HOME/cbw/manual/state-written-before-the-fork-survives-it"
```

**197. `two-producers-hand-over`** — wemix · bp2 en4 · 직전 PASS 100s  
A hardfork handed over by a network with TWO producers. The one-producer case never reaches the poa bring-up's join phase: with a single …  
> 키를 새로 만든다(선언이 generate)

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/two-producers-hand-over" && GWEMIX_BIN="$HOME/work/github/wemade/go-wemix/build/bin/gwemix" GWBFT_BIN="$HOME/work/github/wemade/go-wbft/build/bin/gwemix" bin/chainbench run tests/tc/go-wemix/hardfork/03-two-producers-hand-over.json --workspace-dir "$HOME/cbw/manual/two-producers-hand-over"
```


### `go-wemix/rpc` (1)

**198. `wemix-brioche-block-reward`** — wemix · bp4 · 직전 PASS 104s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wemix-brioche-block-reward" && GWEMIX_BIN="$HOME/work/github/wemade/go-wemix/build/bin/gwemix" bin/chainbench run tests/tc/go-wemix/rpc/01-wemix-brioche-block-reward.json --workspace-dir "$HOME/cbw/manual/wemix-brioche-block-reward"
```


### `go-wemix/tx` (3)

**199. `wemix-tx-and-contract`** — wemix · bp4 · 직전 PASS 106s  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wemix-tx-and-contract" && GWEMIX_BIN="$HOME/work/github/wemade/go-wemix/build/bin/gwemix" bin/chainbench run tests/tc/go-wemix/tx/01-wemix-tx-and-contract.json --workspace-dir "$HOME/cbw/manual/wemix-tx-and-contract"
```

**200. `wemix-insufficient-funds-rejected`** — wemix · bp4 · 직전 PASS 104s  
WA25: a transfer above the sender's balance is refused before it reaches a block. Proven on stablenet (regression/ethereum/14); go-wemix …  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wemix-insufficient-funds-rejected" && bin/chainbench run tests/tc/go-wemix/tx/02-wemix-insufficient-funds-rejected.json --workspace-dir "$HOME/cbw/manual/wemix-insufficient-funds-rejected" --binary "$HOME/work/github/wemade/go-wemix/build/bin/gwemix"
```

**201. `wemix-revert-status-zero`** — wemix · bp4 · 직전 PASS 109s  
WA25: a call into a contract whose code reverts is mined with status 0x0 rather than vanishing or failing the block. Deploy uses runtime …  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wemix-revert-status-zero" && bin/chainbench run tests/tc/go-wemix/tx/03-wemix-revert-status-zero.json --workspace-dir "$HOME/cbw/manual/wemix-revert-status-zero" --binary "$HOME/work/github/wemade/go-wemix/build/bin/gwemix"
```


### `go-wemix/vocabulary` (1)

**202. `wemix-default-on-routes-every-step`** — wemix · bp4 · 직전 PASS 104s  
The case-level "on" routes every statement that names no target (WA24: the vocabulary was registered and wired but no spec ran it; the in…  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/wemix-default-on-routes-every-step" && bin/chainbench run tests/tc/go-wemix/vocabulary/01-default-on-routes-every-step.json --workspace-dir "$HOME/cbw/manual/wemix-default-on-routes-every-step" --binary "$HOME/work/github/wemade/go-wemix/build/bin/gwemix"
```


### `remote` (3)

**203. `remote-rpc-health`** — stablenet · bp4 · 직전 PASS 53s  
레거시 remote/rpc-health: 붙은 엔드포인트가 살아 있고 기본 RPC 셋이 응답한다. chainbench run --rpc <endpoint> 로 실행한다.  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/remote-rpc-health" && bin/chainbench run tests/tc/remote/01-remote-rpc-health.json --workspace-dir "$HOME/cbw/manual/remote-rpc-health" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**204. `remote-chain-info`** — stablenet · bp4 · 직전 PASS 52s  
레거시 remote/chain-info: 붙은 체인이 chainId 를 보고하고 동기화가 끝나 있다.  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/remote-chain-info" && bin/chainbench run tests/tc/remote/02-remote-chain-info.json --workspace-dir "$HOME/cbw/manual/remote-chain-info" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**205. `remote-balance-check`** — stablenet · bp4 · 직전 PASS 56s  
레거시 remote/balance-check: 잔액 조회가 16진 수량으로 돌아온다. 레거시 기본값과 같이 0 주소를 본다 — 어느 체인에나 있고 값이 변하지 않는다.  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/remote-balance-check" && bin/chainbench run tests/tc/remote/03-remote-balance-check.json --workspace-dir "$HOME/cbw/manual/remote-balance-check" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```


### `samples` (2)

**206. `sample-minimal-value-transfer`** — stablenet · bp4 · 직전 PASS 53s  
v1 스펙 샘플. 이미 떠 있는 체인에 붙어 실행한다. steps 로 값을 모으고 assertions 로 판정한다. chainbench validate tests/tc/samples/01-sample-minimal.json  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/sample-minimal-value-transfer" && bin/chainbench run tests/tc/samples/01-sample-minimal.json --workspace-dir "$HOME/cbw/manual/sample-minimal-value-transfer" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**207. `sample-lifecycle-node-restart`** — stablenet · bp4 en1 · 직전 PASS 51s  
작성 샘플 — 노드를 멈췄다 살리고 체인이 이어지는지 확인한다 (docs/guide/dsl-authoring.md)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/sample-lifecycle-node-restart" && GSTABLE_BIN="$HOME/work/github/wemade/go-stablenet/build/bin/gstable" bin/chainbench run tests/tc/samples/02-sample-lifecycle.json --workspace-dir "$HOME/cbw/manual/sample-lifecycle-node-restart"
```


### `stress` (2)

**208. `stress-block-time`** — stablenet · bp4 · 직전 PASS 52s  
Measure block production time statistics over last 100 blocks (원본 stress/block-time.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stress-block-time" && bin/chainbench run tests/tc/stress/01-stress-block-time.json --workspace-dir "$HOME/cbw/manual/stress-block-time" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

**209. `stress-tx-flood`** — stablenet · bp4 · 직전 PASS 69s  
Send N transactions rapidly and measure throughput (원본 stress/tx-flood.sh)  

```sh
cd "$HOME/work/github/0xmhha/chainbench" && rm -rf "$HOME/cbw/manual/stress-tx-flood" && bin/chainbench run tests/tc/stress/02-stress-tx-flood.json --workspace-dir "$HOME/cbw/manual/stress-tx-flood" --binary "$HOME/work/github/wemade/go-stablenet/build/bin/gstable"
```

---

## 5. 같은 케이스를 docker 15대 위에서

로컬 대신 `env/docker` 의 서버 15대에 올리려면 줄 끝에 아래 플래그를 붙이고, `--binary` 와
줄 앞의 환경변수는 뺀다. docker 쪽은 컨테이너의 `/data/chainbench/bin/` 에 있는 바이너리를
이름으로 쓰기 때문이다. 컨테이너 준비는 `env/docker/README.md` 를 따른다.

```sh
# stablenet · wbft
--server-set env/docker/build/server-set.yaml --workspace-config env/docker/build/workspace-config.yaml --docker --all-servers --keys-source generate
# go-wemix: poa 는 노드마다 p2p 옆 포트 셋을 잡아 서버 세트가 다르고, 합류가 느려 대기를 늘린다
--server-set env/docker/build/server-set-wemix.yaml --workspace-config env/docker/build/workspace-config.yaml --docker --all-servers --keys-source generate --node-monitor-timeout 5m
```

docker 쪽은 원격 datadir 도 지워야 다음 케이스의 genesis 가 막히지 않는다. `rm -rf` 앞에
`bin/chainbench chain stop --workspace-dir <ws>` 와 `bin/chainbench chain rm --workspace-dir <ws>` 를 건다.

