# 메인넷 의존 필드 색인 (tests/tc, 2026-09-11)

`scripts/extract_tc_dependencies.py`가 현재 195개 JSON에서 뽑은 필드다. 값은 테스트 fixture이므로 그대로 적었다. 개인 키처럼 보이는 값은 없었다(있으면 redacted 표기). 각 항목의 전체 목록은 [tc-dependencies.json](tc-dependencies.json)에 있다.

## 요약

| 분류 | 필드 수 | 파일 수 | 값 종류 |
|---|---:|---:|---|
| 테스트 계정 | 650 | 195 | literal-string 387, role 166, literal-address 53, binding 43, literal-hex 1 |
| RPC Endpoint·namespace·capability | 889 | 195 | literal-string 595, role 294 |
| 컨트랙트 주소·calldata | 589 | 123 | literal-address 254, binding 138, literal-hex 124, literal-string 73 |
| Chain ID·client family | 336 | 195 | literal-string 327, literal-number 9 |
| 포크·합의 전용 RPC | 86 | 55 | literal-string 63, literal-number 20, literal-address 2, literal-hex 1 |
| 수수료·가스 입력 | 339 | 106 | literal-number 199, literal-hex 96, literal-string 26, binding 18 |
| 토폴로지·시간·프로세스 제어 | 499 | 195 | literal-number 291, literal-string 195, literal-bool 13 |
| 바이너리 | 202 | 195 | literal-string 192, binding 10 |

## 하드코딩된 주소 literal

| 주소 | 파일 수 | 의미 | 처리 |
|---|---:|---|---|
| `0x0000000000000000000000000000000000001004` | 20 | SYSTEM CONTRACT: GovCouncil (stablenet) | StableNet 시스템 계약. 체인 adapter |
| `0x0000000000000000000000000000000000001003` | 19 | SYSTEM CONTRACT: GovMinter (stablenet) | StableNet 시스템 계약. 체인 adapter |
| `0x0000000000000000000000000000000000B00003` | 16 | reserved low address (precompile/system range) literal | 검토 |
| `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | 15 | signer/account reference (hardcoded address literal) | preset 키셋 node1 계정(node-signed sender). 계정 label로 치환 |
| `0x0000000000000000000000000000000000001000` | 14 | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) | StableNet 시스템 계약. 체인 adapter |
| `0x0000000000000000000000000000000000001001` | 9 | SYSTEM CONTRACT: GovValidator (stablenet) | StableNet 시스템 계약. 체인 adapter |
| `0x70997970C51812dc3A010C7d01b50e0d17dc79C8` | 8 | hardcoded recipient address literal | 수신용 고정 주소. label 또는 newAccount binding으로 |
| `0x00000000000000000000000000000000C0FFEE0E` | 6 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x0000000000000000000000000000000000000100` | 5 | SYSTEM CONTRACT: P256VERIFY precompile (RIP-7212) | StableNet 시스템 계약. 체인 adapter |
| `0x0000000000000000000000000000000000001002` | 5 | SYSTEM CONTRACT: GovMasterMinter (stablenet) | StableNet 시스템 계약. 체인 adapter |
| `0x00000000000000000000000000000000C0FFEE10` | 5 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE07` | 3 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE11` | 2 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x90F79bf6EB2c4f870365E785982E1f101E93b906` | 2 | hardcoded address literal | 검토 |
| `0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65` | 2 | hardcoded address literal | 검토 |
| `0x00000000000000000000000000000000C0FFEE05` | 2 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE13` | 2 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x0000000000000000000000000000000000000000` | 2 | reserved low address (precompile/system range) literal | zero address. 그대로 둔다 |
| `0x0000000000000000000000000000000000000001` | 2 | reserved low address (precompile/system range) literal | 검토 |
| `0x1111111111111111111111111111111111111111` | 2 | signer/account reference (hardcoded address literal) | EIP-7702 delegate 대상·createAddress 입력 상수. 그대로 둔다 |
| `0x71562b71999873db5b286df957af199ec94617f7` | 1 | hardcoded address literal in RPC params | 검토 |
| `0x00000000000000000000000000000000C0FFEE06` | 1 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE04` | 1 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE08` | 1 | hardcoded address literal in RPC params | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE32` | 1 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x0000000000000000000000000000000000b00001` | 1 | reserved low address (precompile/system range) literal | 검토 |
| `0x0000000000000000000000000000000000b00002` | 1 | reserved low address (precompile/system range) literal | 검토 |
| `0x0000000000000000000000000000000000b00003` | 1 | reserved low address (precompile/system range) literal | 검토 |
| `0x00000000000000000000000000000000C0FFEE03` | 1 | hardcoded address literal in RPC params | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE0C` | 1 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE0D` | 1 | hardcoded address literal in RPC params | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE0B` | 1 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE12` | 1 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE01` | 1 | hardcoded address literal in RPC params | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE02` | 1 | hardcoded address literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE20` | 1 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE21` | 1 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE22` | 1 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE23` | 1 | reserved low address (precompile/system range) literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |
| `0x00000000000000000000000000000000C0FFEE30` | 1 | hardcoded address literal | 수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로 |

## 분류별 필드 목록

### 테스트 계정

| 파일 | 필드 | 값 종류 | 값 | 의미 |
|---|---|---|---|---|
| basic/01-basic-consensus.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| basic/01-basic-consensus.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| basic/02-basic-peers.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| basic/02-basic-peers.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| basic/03-basic-rpc-health.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| basic/03-basic-rpc-health.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| basic/04-basic-sync.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| basic/04-basic-sync.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| basic/05-basic-tx-send.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| basic/05-basic-tx-send.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| basic/05-basic-tx-send.json | `steps[2].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| basic/05-basic-tx-send.json | `steps[3].address` | binding | `$r` | address by role/label/binding |
| basic/06-basic-txpool-propagation.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| basic/06-basic-txpool-propagation.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| basic/06-basic-txpool-propagation.json | `steps[2].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| basic/06-basic-txpool-propagation.json | `steps[3].address` | binding | `$r` | address by role/label/binding |
| basic/07-basic-wbft-consensus.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| basic/07-basic-wbft-consensus.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| fault/01-fault-network-partition.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| fault/01-fault-network-partition.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| fault/02-fault-node-crash.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| fault/02-fault-node-crash.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| fault/03-fault-node-recover.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| fault/03-fault-node-recover.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| fault/04-fault-p2p-topology.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| fault/04-fault-p2p-topology.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| fault/04-fault-p2p-topology.json | `steps[6].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| fault/04-fault-p2p-topology.json | `steps[7].address` | binding | `$r` | address by role/label/binding |
| fault/05-fault-two-down.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| fault/05-fault-two-down.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| fault/06-fault-txpool-leader-change.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| fault/06-fault-txpool-leader-change.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| fault/06-fault-txpool-leader-change.json | `steps[2].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[6].params[0]` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | hardcoded address literal in RPC params |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[7].params[0]` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | hardcoded address literal in RPC params |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[2].from` | role | `node2` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[5].from` | role | `node2` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[2].from` | role | `node3` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[5].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[6].from` | role | `node2` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[7].from` | role | `node4` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[3].from` | role | `node3` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[10].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[5].from` | role | `node2` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[2].from` | role | `node4` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[5].from` | role | `node4` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[7].from` | role | `node4` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[0].from` | role | `node2` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[3].from` | role | `node2` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[4].from` | role | `node2` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[5].from` | role | `node2` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[0].params[0]` | literal-address | `0x71562b71999873db5b286df957af199ec94617f7` | hardcoded address literal in RPC params |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[1].params[0]` | literal-address | `0x71562b71999873db5b286df957af199ec94617f7` | hardcoded address literal in RPC params |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[4].params[0]` | literal-address | `0x71562b71999873db5b286df957af199ec94617f7` | hardcoded address literal in RPC params |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[5].params[0]` | literal-address | `0x71562b71999873db5b286df957af199ec94617f7` | hardcoded address literal in RPC params |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[4].authorityKey` | binding | `$authority1Key` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[4].delegate` | binding | `$delegate` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[5].authorityKey` | binding | `$authority2Key` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[5].delegate` | binding | `$delegate` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[6].params[0].from` | binding | `$sponsor` | account reference in RPC params |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[6].params[0].to` | binding | `$sponsor` | account reference in RPC params |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[7].params[0].from` | binding | `$sponsor` | account reference in RPC params |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[7].params[0].to` | binding | `$sponsor` | account reference in RPC params |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[8].params[0].from` | binding | `$sponsor` | account reference in RPC params |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[8].params[0].to` | binding | `$sponsor` | account reference in RPC params |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[0].from` | role | `node4` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[3].from` | role | `node4` | signer/account reference |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[5].from` | role | `node4` | signer/account reference |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[4].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[7].from` | role | `node2` | signer/account reference |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[8].to` | literal-address | `0x70997970C51812dc3A010C7d01b50e0d17dc79C8` | hardcoded recipient address literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[1].to` | literal-address | `0x70997970C51812dc3A010C7d01b50e0d17dc79C8` | hardcoded recipient address literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[4].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[7].from` | role | `node2` | signer/account reference |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[8].to` | literal-address | `0x70997970C51812dc3A010C7d01b50e0d17dc79C8` | hardcoded recipient address literal |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | `steps[0].address` | literal-address | `0x90F79bf6EB2c4f870365E785982E1f101E93b906` | hardcoded address literal |
| go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | `steps[1].address` | literal-address | `0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65` | hardcoded address literal |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[1].params[0]` | literal-address | `0x90F79bf6EB2c4f870365E785982E1f101E93b906` | hardcoded address literal in RPC params |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[2].params[0]` | literal-address | `0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65` | hardcoded address literal in RPC params |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[8].params[0]` | literal-address | `0x90F79bf6EB2c4f870365E785982E1f101E93b906` | hardcoded address literal in RPC params |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[9].params[0]` | literal-address | `0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65` | hardcoded address literal in RPC params |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[1].to` | literal-address | `0x70997970C51812dc3A010C7d01b50e0d17dc79C8` | hardcoded recipient address literal |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `steps[3].to` | literal-address | `0x70997970C51812dc3A010C7d01b50e0d17dc79C8` | hardcoded recipient address literal |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[6].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[12].to` | literal-address | `0x70997970C51812dc3A010C7d01b50e0d17dc79C8` | hardcoded recipient address literal |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `steps[2].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `steps[2].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `steps[1].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| go-stablenet/regression/anzeon/06-basefee-minimum.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/06-basefee-minimum.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/07-basefee-maximum.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/07-basefee-maximum.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `steps[4].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `steps[4].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `steps[6].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/api/01-block-transactions-field.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/01-block-transactions-field.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/02-block-by-hash-consistency.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/02-block-by-hash-consistency.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/api/05-transaction-count-increments.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/05-transaction-count-increments.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/05-transaction-count-increments.json | `steps[0].params[0]` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | hardcoded address literal in RPC params |
| go-stablenet/regression/api/05-transaction-count-increments.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/api/05-transaction-count-increments.json | `steps[2].params[0]` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | hardcoded address literal in RPC params |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/07-gas-price-positive.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/07-gas-price-positive.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/09-fee-history-well-formed.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/09-fee-history-well-formed.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/10-estimate-gas-token-transfer.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/10-estimate-gas-token-transfer.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/10-estimate-gas-token-transfer.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/api/11-node-address-returned.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/11-node-address-returned.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/12-validator-set-nonempty.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/12-validator-set-nonempty.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/12b-validator-set-count.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/12b-validator-set-count.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/14-wbft-extra-info-fields.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/14-wbft-extra-info-fields.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/15-istanbul-status-fields.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/15-istanbul-status-fields.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/16-is-validator-flags.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/16-is-validator-flags.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/18-txpool-status.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/18-txpool-status.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/19-txpool-content-well-formed.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/19-txpool-content-well-formed.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/20-admin-peers-populated.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/20-admin-peers-populated.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json | `steps[0].params[0].from` | literal-address | `0x00000000000000000000000000000000C0FFEE08` | hardcoded address literal in RPC params |
| go-stablenet/regression/api/22-token-total-supply-readable.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/22-token-total-supply-readable.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/api/24-chain-not-syncing.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/api/24-chain-not-syncing.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[6].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[3].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[5].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[6].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[9].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[5].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[8].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[9].feePayerKey` | binding | `$feePayerKey` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[3].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[4].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[7].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[4].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[6].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[7].to` | literal-address | `0x70997970C51812dc3A010C7d01b50e0d17dc79C8` | hardcoded recipient address literal |
| go-stablenet/regression/ethereum/01-stablenet-chain-up.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/01-stablenet-chain-up.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[4].params[0]` | literal-address | `0x00000000000000000000000000000000C0FFEE03` | hardcoded address literal in RPC params |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[5].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[6].params[0]` | literal-address | `0x00000000000000000000000000000000C0FFEE03` | hardcoded address literal in RPC params |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `steps[4].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `steps[2].params[0].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | hardcoded address literal in RPC params |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `steps[2].params[0].to` | literal-address | `0x00000000000000000000000000000000C0FFEE0D` | hardcoded address literal in RPC params |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[5].address` | binding | `$acct` | address by role/label/binding |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[7].address` | binding | `$acct` | address by role/label/binding |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[5].address` | binding | `$acct` | address by role/label/binding |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[7].address` | binding | `$acct` | address by role/label/binding |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `steps[0].address` | role | `node1` | address by role/label/binding |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[3].authorityKey` | binding | `$authorityKey` | signer/account reference |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[3].delegate` | literal-address | `0x1111111111111111111111111111111111111111` | signer/account reference (hardcoded address literal) |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[4].address` | binding | `$authority` | address by role/label/binding |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[8].address` | binding | `$authority` | address by role/label/binding |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `steps[1].address` | binding | `$contract` | address by role/label/binding |
| go-stablenet/regression/ethereum/22-estimate-gas.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/22-estimate-gas.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `steps[2].address` | binding | `$addr` | address by role/label/binding |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/27-genesis-balance.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/27-genesis-balance.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/27-genesis-balance.json | `steps[0].address` | role | `node1` | address by role/label/binding |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `steps[0].params[0]` | literal-address | `0x00000000000000000000000000000000C0FFEE01` | hardcoded address literal in RPC params |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `steps[2].params[0]` | literal-address | `0x00000000000000000000000000000000C0FFEE01` | hardcoded address literal in RPC params |
| go-stablenet/regression/ethereum/29-logs-query-well-formed.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/29-logs-query-well-formed.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/30-chain-id.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/30-chain-id.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[2].address` | binding | `$addr` | address by role/label/binding |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `env.keys.nodekeys.source` | literal-string | `generate` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `steps[5].address` | binding | `$addr` | address by role/label/binding |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[4].feePayerKey` | binding | `$feePayerKey` | signer/account reference |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[8].address` | literal-address | `0x00000000000000000000000000000000C0FFEE02` | hardcoded address literal |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `steps[2].feePayerKey` | binding | `$acctKey` | signer/account reference |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `steps[2].senderKey` | binding | `$acctKey` | signer/account reference |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `steps[2].feePayerKey` | binding | `$acctKey` | signer/account reference |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `steps[2].senderKey` | binding | `$acctKey` | signer/account reference |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `steps[3].feePayerKey` | binding | `$feePayerKey` | signer/account reference |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `steps[3].to` | literal-address | `0x70997970C51812dc3A010C7d01b50e0d17dc79C8` | hardcoded recipient address literal |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `steps[2].feePayerKey` | binding | `$acctKey` | signer/account reference |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `steps[2].senderKey` | binding | `$acctKey` | signer/account reference |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `steps[2].feePayerKey` | binding | `$acctKey` | signer/account reference |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `steps[2].senderKey` | binding | `$acctKey` | signer/account reference |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `steps[3].feePayerKey` | binding | `$feePayerKey` | signer/account reference |
| go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/02-token-balance-readable.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/02-token-balance-readable.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[4].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[0].address` | literal-address | `0x00000000000000000000000000000000C0FFEE30` | hardcoded address literal |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[5].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[9].address` | literal-address | `0x00000000000000000000000000000000C0FFEE30` | hardcoded address literal |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[0].address` | literal-address | `0x00000000000000000000000000000000C0FFEE05` | hardcoded address literal |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[5].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[8].address` | literal-address | `0x00000000000000000000000000000000C0FFEE05` | hardcoded address literal |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[3].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[7].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/09-validator-metadata-readable.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/09-validator-metadata-readable.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[6].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[9].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[12].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[7].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[3].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[3].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[5].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[8].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[5].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[11].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[14].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[15].from` | role | `node3` | signer/account reference |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[3].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[3].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[3].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[4].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[7].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[3].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/system-contracts/25-token-metadata.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/25-token-metadata.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/28-minter-status-readable.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/system-contracts/28-minter-status-readable.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[4].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[5].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[8].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[1].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[4].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[7].from` | role | `node1` | signer/account reference |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[10].from` | role | `node2` | signer/account reference |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/14-stablenet-gastip-field.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/regression/wbft/14-stablenet-gastip-field.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/topology/01-proxied-pn-routing.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/topology/01-proxied-pn-routing.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/tx/01-negative-tx-revert.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/tx/01-negative-tx-revert.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[1].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[2].address` | binding | `$rv` | address by role/label/binding |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[3].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| go-stablenet/vocabulary/01-derived-address-and-checksum.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/vocabulary/01-derived-address-and-checksum.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/vocabulary/01-derived-address-and-checksum.json | `steps[1].deployer` | literal-address | `0x1111111111111111111111111111111111111111` | signer/account reference (hardcoded address literal) |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `steps[2].do` | role | `faucet` | node-signed asset helper |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `steps[3].address` | binding | `$acct` | address by role/label/binding |
| go-stablenet/vocabulary/03-metric-head-block.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/vocabulary/03-metric-head-block.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/vocabulary/03-register-contract.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/vocabulary/03-register-contract.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-stablenet/vocabulary/03-register-contract.json | `steps[1].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| go-wbft/accounts/01-secp256r1-precompile-valid.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/accounts/01-secp256r1-precompile-valid.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/accounts/02-secp256r1-precompile-invalid.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/accounts/02-secp256r1-precompile-invalid.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/accounts/03-secp256r1-precompile-short-input.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/accounts/03-secp256r1-precompile-short-input.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/chain-up/01-wbft-chain-up.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/chain-up/01-wbft-chain-up.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `env.keys.nodekeys.source` | literal-string | `generate` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/consensus/01-e1-mixed-producers.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/consensus/01-e1-mixed-producers.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/fault/01-wbft-node-crash.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/fault/01-wbft-node-crash.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/network/01-wbft-proxied-routing.json | `env.keys.nodekeys.source` | literal-string | `generate` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[2].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[3].address` | binding | `$r` | address by role/label/binding |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[4].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[5].address` | binding | `$c` | address by role/label/binding |
| go-wbft/tx/02-wbft-insufficient-funds-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/tx/02-wbft-insufficient-funds-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/tx/02-wbft-insufficient-funds-rejected.json | `steps[0].address` | role | `node1` | address by role/label/binding |
| go-wbft/tx/02-wbft-insufficient-funds-rejected.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-wbft/tx/03-wbft-revert-status-zero.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/tx/03-wbft-revert-status-zero.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wbft/tx/03-wbft-revert-status-zero.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-wbft/tx/03-wbft-revert-status-zero.json | `steps[3].from` | role | `node1` | signer/account reference |
| go-wemix/chain-up/01-wemix-chain-up.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/chain-up/01-wemix-chain-up.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/chain-up/02-wemix-chain-up-15.json | `env.keys.nodekeys.source` | literal-string | `generate` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/fault/01-wemix-node-crash.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/fault/01-wemix-node-crash.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[2].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[3].address` | binding | `$r` | address by role/label/binding |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[4].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[5].address` | binding | `$c` | address by role/label/binding |
| go-wemix/tx/02-wemix-insufficient-funds-rejected.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/tx/02-wemix-insufficient-funds-rejected.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/tx/02-wemix-insufficient-funds-rejected.json | `steps[0].address` | role | `node1` | address by role/label/binding |
| go-wemix/tx/02-wemix-insufficient-funds-rejected.json | `steps[2].from` | role | `node1` | signer/account reference |
| go-wemix/tx/03-wemix-revert-status-zero.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/tx/03-wemix-revert-status-zero.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| go-wemix/tx/03-wemix-revert-status-zero.json | `steps[0].from` | role | `node1` | signer/account reference |
| go-wemix/tx/03-wemix-revert-status-zero.json | `steps[3].from` | role | `node1` | signer/account reference |
| remote/01-remote-rpc-health.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| remote/01-remote-rpc-health.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| remote/02-remote-chain-info.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| remote/02-remote-chain-info.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| remote/03-remote-balance-check.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| remote/03-remote-balance-check.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| remote/03-remote-balance-check.json | `steps[0].address` | literal-address | `0x0000000000000000000000000000000000000000` | hardcoded address literal |
| remote/03-remote-balance-check.json | `steps[1].params[0]` | literal-address | `0x0000000000000000000000000000000000000000` | hardcoded address literal in RPC params |
| samples/01-sample-minimal.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| samples/01-sample-minimal.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| samples/01-sample-minimal.json | `steps[0].address` | role | `node2` | address by role/label/binding |
| samples/01-sample-minimal.json | `steps[1].from` | role | `node1` | signer/account reference |
| samples/01-sample-minimal.json | `steps[1].to` | role | `node2` | recipient by role/label |
| samples/01-sample-minimal.json | `steps[5].address` | role | `node2` | address by role/label/binding |
| samples/02-sample-lifecycle.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| samples/02-sample-lifecycle.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| samples/02-sample-lifecycle.json | `env.accounts.dev1.fund` | literal-hex | `0x8AC7230489E80000` | declared test account funded at run time |
| samples/02-sample-lifecycle.json | `steps[3].from` | literal-string | `dev1` | signer/account reference |
| samples/02-sample-lifecycle.json | `steps[4].address` | binding | `$acct` | address by role/label/binding |
| stress/01-stress-block-time.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| stress/01-stress-block-time.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| stress/02-stress-tx-flood.json | `env.keys.nodekeys.source` | literal-string | `preset` | node identity key source (validators/producers and node-signed sender) |
| stress/02-stress-tx-flood.json | `env.keys.nodekeys.ref` | literal-string | `keys/preset` | node identity key source (validators/producers and node-signed sender) |
| stress/02-stress-tx-flood.json | `steps[1].from` | literal-address | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | signer/account reference (hardcoded address literal) |

### RPC Endpoint·namespace·capability

| 파일 | 필드 | 값 종류 | 값 | 의미 |
|---|---|---|---|---|
| basic/01-basic-consensus.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| basic/01-basic-consensus.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| basic/01-basic-consensus.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| basic/02-basic-peers.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| basic/02-basic-peers.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| basic/02-basic-peers.json | `steps[2].on` | role | `node2` | target node role (endpoint selection) |
| basic/02-basic-peers.json | `steps[3].on` | role | `node3` | target node role (endpoint selection) |
| basic/02-basic-peers.json | `steps[4].on` | role | `node4` | target node role (endpoint selection) |
| basic/02-basic-peers.json | `steps[5].on` | role | `en1` | target node role (endpoint selection) |
| basic/03-basic-rpc-health.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| basic/03-basic-rpc-health.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| basic/03-basic-rpc-health.json | `steps[2].on` | role | `node2` | target node role (endpoint selection) |
| basic/03-basic-rpc-health.json | `steps[3].on` | role | `node3` | target node role (endpoint selection) |
| basic/03-basic-rpc-health.json | `steps[4].on` | role | `node4` | target node role (endpoint selection) |
| basic/03-basic-rpc-health.json | `steps[5].on` | role | `en1` | target node role (endpoint selection) |
| basic/04-basic-sync.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| basic/04-basic-sync.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| basic/05-basic-tx-send.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| basic/05-basic-tx-send.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| basic/05-basic-tx-send.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| basic/06-basic-txpool-propagation.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| basic/06-basic-txpool-propagation.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| basic/06-basic-txpool-propagation.json | `steps[3].on` | role | `node2` | target node role (endpoint selection) |
| basic/07-basic-wbft-consensus.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| basic/07-basic-wbft-consensus.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| fault/01-fault-network-partition.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| fault/01-fault-network-partition.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| fault/01-fault-network-partition.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| fault/01-fault-network-partition.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| fault/01-fault-network-partition.json | `steps[4].on` | role | `node3` | target node role (endpoint selection) |
| fault/01-fault-network-partition.json | `steps[6].on` | role | `node3` | target node role (endpoint selection) |
| fault/02-fault-node-crash.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| fault/02-fault-node-crash.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| fault/02-fault-node-crash.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| fault/02-fault-node-crash.json | `steps[2].on` | role | `node3` | target node role (endpoint selection) |
| fault/02-fault-node-crash.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| fault/02-fault-node-crash.json | `steps[4].on` | role | `node3` | target node role (endpoint selection) |
| fault/02-fault-node-crash.json | `steps[5].on` | role | `node3` | target node role (endpoint selection) |
| fault/03-fault-node-recover.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| fault/03-fault-node-recover.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| fault/03-fault-node-recover.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| fault/03-fault-node-recover.json | `steps[1].on` | role | `node3` | target node role (endpoint selection) |
| fault/03-fault-node-recover.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| fault/03-fault-node-recover.json | `steps[3].on` | role | `node3` | target node role (endpoint selection) |
| fault/03-fault-node-recover.json | `steps[4].on` | role | `node3` | target node role (endpoint selection) |
| fault/04-fault-p2p-topology.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| fault/04-fault-p2p-topology.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| fault/04-fault-p2p-topology.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| fault/04-fault-p2p-topology.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| fault/04-fault-p2p-topology.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| fault/04-fault-p2p-topology.json | `steps[6].on` | role | `node1` | target node role (endpoint selection) |
| fault/04-fault-p2p-topology.json | `steps[7].on` | role | `node1` | target node role (endpoint selection) |
| fault/04-fault-p2p-topology.json | `steps[9].on` | role | `node4` | target node role (endpoint selection) |
| fault/05-fault-two-down.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| fault/05-fault-two-down.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| fault/05-fault-two-down.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| fault/05-fault-two-down.json | `steps[2].on` | role | `node3` | target node role (endpoint selection) |
| fault/05-fault-two-down.json | `steps[3].on` | role | `node4` | target node role (endpoint selection) |
| fault/05-fault-two-down.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| fault/05-fault-two-down.json | `steps[5].on` | role | `node3` | target node role (endpoint selection) |
| fault/05-fault-two-down.json | `steps[6].on` | role | `node1` | target node role (endpoint selection) |
| fault/05-fault-two-down.json | `steps[7].on` | role | `node4` | target node role (endpoint selection) |
| fault/06-fault-txpool-leader-change.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| fault/06-fault-txpool-leader-change.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| fault/06-fault-txpool-leader-change.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| fault/06-fault-txpool-leader-change.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| fault/06-fault-txpool-leader-change.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| fault/06-fault-txpool-leader-change.json | `steps[4].on` | role | `node2` | target node role (endpoint selection) |
| fault/06-fault-txpool-leader-change.json | `steps[5].on` | role | `node2` | target node role (endpoint selection) |
| fault/06-fault-txpool-leader-change.json | `steps[5].method` | literal-string | `txpool_status` | RPC method; namespace txpool: standard |
| fault/06-fault-txpool-leader-change.json | `steps[6].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[1].method` | literal-string | `eth_getCode` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[2].method` | literal-string | `eth_getCode` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[3].method` | literal-string | `eth_getCode` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[4].method` | literal-string | `eth_call` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[5].method` | literal-string | `eth_call` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[6].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[7].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `steps[0].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `steps[1].method` | literal-string | `eth_getCode` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `steps[4].method` | literal-string | `eth_getCode` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `steps[5].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[2].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[5].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[2].on` | role | `node3` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[5].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[6].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[7].on` | role | `node4` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `env.capabilities[]` | literal-string | `short-expiry` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `requires[]` | literal-string | `short-expiry` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[3].on` | role | `node3` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[6].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[8].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[10].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[11].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[5].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[2].on` | role | `node4` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[5].on` | role | `node4` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[7].on` | role | `node4` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[0].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[3].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[4].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[5].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[0].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[1].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[2].method` | literal-string | `eth_getStorageAt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[4].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[5].method` | literal-string | `eth_getTransactionCount` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[6].method` | literal-string | `eth_getStorageAt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `steps[1].method` | literal-string | `eth_getCode` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[3].method` | literal-string | `eth_coinbase` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[6].method` | literal-string | `eth_estimateGas` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[7].method` | literal-string | `eth_estimateGas` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[8].method` | literal-string | `eth_estimateGas` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `steps[0].method` | literal-string | `eth_getCode` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `steps[1].method` | literal-string | `eth_getCode` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `steps[2].method` | literal-string | `eth_getCode` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `steps[3].method` | literal-string | `eth_getCode` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `steps[0].method` | literal-string | `eth_getStorageAt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `steps[1].method` | literal-string | `eth_getStorageAt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `steps[2].method` | literal-string | `eth_getCode` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[0].on` | role | `node4` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[3].on` | role | `node4` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[4].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[5].on` | role | `node4` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[6].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[7].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[8].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[9].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[9].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[10].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[10].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[11].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[11].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[12].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[12].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[13].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[13].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `steps[1].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `steps[2].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `steps[3].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[2].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[3].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[3].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[4].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[5].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[5].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[7].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[8].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[9].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[9].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[10].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[10].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[11].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[11].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[13].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[13].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[14].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[14].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[16].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[16].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[17].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[17].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[18].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[18].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `env.capabilities[]` | literal-string | `account-extra` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `requires[]` | literal-string | `account-extra` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `env.capabilities[]` | literal-string | `account-extra` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `requires[]` | literal-string | `account-extra` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[6].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[8].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `env.capabilities[]` | literal-string | `account-extra` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `requires[]` | literal-string | `account-extra` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | `env.capabilities[]` | literal-string | `account-extra` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | `requires[]` | literal-string | `account-extra` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `steps[1].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `steps[1].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `steps[3].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[1].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[2].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[8].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[9].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[2].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[3].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[4].method` | literal-string | `eth_getTransactionByHash` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[5].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[5].method` | literal-string | `eth_getTransactionByHash` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[6].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[7].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[7].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[8].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[8].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[9].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[9].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[10].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[10].method` | literal-string | `eth_getTransactionByHash` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[11].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[11].method` | literal-string | `eth_getTransactionByHash` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[12].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `steps[2].on` | role | `node4` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `steps[3].on` | role | `node4` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `steps[1].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `steps[4].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `steps[5].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | `steps[0].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | `steps[1].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | `steps[3].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `steps[0].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `steps[4].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `steps[5].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `steps[6].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[6].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[9].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[12].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[13].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[14].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[15].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/anzeon/06-basefee-minimum.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/anzeon/06-basefee-minimum.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/anzeon/07-basefee-maximum.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/anzeon/07-basefee-maximum.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `steps[0].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `steps[1].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `steps[0].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `steps[1].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `steps[0].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `steps[1].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `steps[4].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/01-block-transactions-field.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/01-block-transactions-field.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/01-block-transactions-field.json | `steps[0].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/01-block-transactions-field.json | `steps[1].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/01-block-transactions-field.json | `steps[2].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/02-block-by-hash-consistency.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/02-block-by-hash-consistency.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/02-block-by-hash-consistency.json | `steps[0].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/02-block-by-hash-consistency.json | `steps[1].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/02-block-by-hash-consistency.json | `steps[2].method` | literal-string | `eth_getBlockByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/02-block-by-hash-consistency.json | `steps[3].method` | literal-string | `eth_getBlockByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `steps[1].method` | literal-string | `eth_getTransactionByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `steps[2].method` | literal-string | `eth_getTransactionByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `steps[3].method` | literal-string | `eth_getTransactionByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `steps[4].method` | literal-string | `eth_getTransactionByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `steps[1].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `steps[2].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `steps[3].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/05-transaction-count-increments.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/05-transaction-count-increments.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/05-transaction-count-increments.json | `steps[0].method` | literal-string | `eth_getTransactionCount` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/05-transaction-count-increments.json | `steps[2].method` | literal-string | `eth_getTransactionCount` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/07-gas-price-positive.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/07-gas-price-positive.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | `steps[1].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | `steps[2].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | `steps[0].method` | literal-string | `eth_maxPriorityFeePerGas` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | `steps[1].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/09-fee-history-well-formed.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/09-fee-history-well-formed.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/09-fee-history-well-formed.json | `steps[0].method` | literal-string | `eth_feeHistory` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/09-fee-history-well-formed.json | `steps[1].method` | literal-string | `eth_feeHistory` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/10-estimate-gas-token-transfer.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/10-estimate-gas-token-transfer.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/11-node-address-returned.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/11-node-address-returned.json | `env.capabilities[]` | literal-string | `consensus` | capability the env claims to provide |
| go-stablenet/regression/api/11-node-address-returned.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/11-node-address-returned.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/11-node-address-returned.json | `steps[0].onEach` | role | `bp1` | target node role (endpoint selection) |
| go-stablenet/regression/api/11-node-address-returned.json | `steps[0].onEach` | role | `bp2` | target node role (endpoint selection) |
| go-stablenet/regression/api/11-node-address-returned.json | `steps[0].onEach` | role | `bp3` | target node role (endpoint selection) |
| go-stablenet/regression/api/11-node-address-returned.json | `steps[0].onEach` | role | `bp4` | target node role (endpoint selection) |
| go-stablenet/regression/api/12-validator-set-nonempty.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/12-validator-set-nonempty.json | `env.capabilities[]` | literal-string | `consensus` | capability the env claims to provide |
| go-stablenet/regression/api/12-validator-set-nonempty.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/12-validator-set-nonempty.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/12b-validator-set-count.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/12b-validator-set-count.json | `env.capabilities[]` | literal-string | `consensus` | capability the env claims to provide |
| go-stablenet/regression/api/12b-validator-set-count.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/12b-validator-set-count.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `env.capabilities[]` | literal-string | `consensus` | capability the env claims to provide |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/14-wbft-extra-info-fields.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/14-wbft-extra-info-fields.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/14-wbft-extra-info-fields.json | `steps[0].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/15-istanbul-status-fields.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/15-istanbul-status-fields.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/16-is-validator-flags.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/16-is-validator-flags.json | `env.capabilities[]` | literal-string | `consensus` | capability the env claims to provide |
| go-stablenet/regression/api/16-is-validator-flags.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/16-is-validator-flags.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/16-is-validator-flags.json | `steps[0].onEach` | role | `bp1` | target node role (endpoint selection) |
| go-stablenet/regression/api/16-is-validator-flags.json | `steps[0].onEach` | role | `bp2` | target node role (endpoint selection) |
| go-stablenet/regression/api/16-is-validator-flags.json | `steps[0].onEach` | role | `bp3` | target node role (endpoint selection) |
| go-stablenet/regression/api/16-is-validator-flags.json | `steps[0].onEach` | role | `bp4` | target node role (endpoint selection) |
| go-stablenet/regression/api/18-txpool-status.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/18-txpool-status.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/18-txpool-status.json | `steps[0].method` | literal-string | `txpool_status` | RPC method; namespace txpool: standard |
| go-stablenet/regression/api/18-txpool-status.json | `steps[1].method` | literal-string | `txpool_status` | RPC method; namespace txpool: standard |
| go-stablenet/regression/api/19-txpool-content-well-formed.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/19-txpool-content-well-formed.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/19-txpool-content-well-formed.json | `steps[0].method` | literal-string | `txpool_content` | RPC method; namespace txpool: standard |
| go-stablenet/regression/api/19-txpool-content-well-formed.json | `steps[1].method` | literal-string | `txpool_content` | RPC method; namespace txpool: standard |
| go-stablenet/regression/api/20-admin-peers-populated.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/20-admin-peers-populated.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/20-admin-peers-populated.json | `steps[0].method` | literal-string | `admin_peers` | RPC method; namespace admin: standard (admin, usually not public) |
| go-stablenet/regression/api/20-admin-peers-populated.json | `steps[1].method` | literal-string | `admin_peers` | RPC method; namespace admin: standard (admin, usually not public) |
| go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json | `steps[0].method` | literal-string | `eth_signRawFeeDelegateTransaction` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/22-token-total-supply-readable.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/22-token-total-supply-readable.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `steps[1].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/api/24-chain-not-syncing.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/api/24-chain-not-syncing.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/api/24-chain-not-syncing.json | `steps[0].method` | literal-string | `eth_syncing` | RPC method; namespace eth: standard |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[6].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[7].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[3].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[5].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[6].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[9].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[5].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[8].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[9].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[3].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[7].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[8].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[6].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[7].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/01-stablenet-chain-up.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/01-stablenet-chain-up.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/01-stablenet-chain-up.json | `steps[4].method` | literal-string | `net_peerCount` | RPC method; namespace net: standard |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[0].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[1].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[4].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[6].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[8].method` | literal-string | `eth_getTransactionByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `steps[0].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `steps[1].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `steps[6].method` | literal-string | `eth_getTransactionByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `steps[2].method` | literal-string | `eth_createAccessList` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `steps[5].method` | literal-string | `eth_getTransactionByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[3].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[4].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[3].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[4].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `steps[0].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `steps[2].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `steps[3].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[3].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[4].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[3].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[4].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[3].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[7].method` | literal-string | `eth_getTransactionByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/22-estimate-gas.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/22-estimate-gas.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `steps[1].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `steps[1].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `steps[4].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `steps[1].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `steps[3].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/27-genesis-balance.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/27-genesis-balance.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `steps[0].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `steps[2].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/29-logs-query-well-formed.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/29-logs-query-well-formed.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/30-chain-id.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/30-chain-id.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/30-chain-id.json | `steps[0].method` | literal-string | `eth_chainId` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `env.capabilities[]` | literal-string | `ws` | capability the env claims to provide |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `requires[]` | literal-string | `ws` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `steps[0].expect` | literal-string | `wsSubscribe` | WebSocket endpoint |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `steps[0].on` | role | `bp1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `env.capabilities[]` | literal-string | `ws` | capability the env claims to provide |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `requires[]` | literal-string | `ws` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[1].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[2].do` | literal-string | `wsOpen` | WebSocket endpoint |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[6].expect` | literal-string | `wsCollected` | WebSocket endpoint |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `steps[4].method` | literal-string | `net_peerCount` | RPC method; namespace net: standard |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `steps[1].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `steps[3].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[4].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `steps[3].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `steps[3].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `steps[1].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/system-contracts/02-token-balance-readable.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/02-token-balance-readable.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[6].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[8].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[5].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[6].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `steps[1].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[5].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[3].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[7].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/09-validator-metadata-readable.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/09-validator-metadata-readable.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[0].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[6].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[9].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[12].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `env.capabilities[]` | literal-string | `short-expiry` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `requires[]` | literal-string | `short-expiry` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[3].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[5].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[7].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[3].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[3].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[5].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[8].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[5].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[11].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[14].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[15].on` | role | `node3` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[3].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[4].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[3].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[3].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[7].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[8].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[0].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[3].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[4].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-stablenet/regression/system-contracts/25-token-metadata.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/25-token-metadata.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/system-contracts/28-minter-status-readable.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/system-contracts/28-minter-status-readable.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `steps[0].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `steps[1].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `steps[2].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `steps[3].method` | literal-string | `eth_getBlockByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `steps[4].method` | literal-string | `eth_getBlockByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `steps[5].method` | literal-string | `eth_getBlockByHash` | RPC method; namespace eth: standard |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `env.capabilities[]` | literal-string | `consensus` | capability the env claims to provide |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `steps[0].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `env.capabilities[]` | literal-string | `consensus` | capability the env claims to provide |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[4].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[1].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[3].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[5].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[8].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[11].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[12].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[4].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[7].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[10].on` | role | `node2` | target node role (endpoint selection) |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `env.capabilities[]` | literal-string | `consensus` | capability the env claims to provide |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | `steps[0].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | `steps[2].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-stablenet/regression/wbft/14-stablenet-gastip-field.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-stablenet/regression/wbft/14-stablenet-gastip-field.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/regression/wbft/14-stablenet-gastip-field.json | `steps[0].method` | literal-string | `eth_blockNumber` | RPC method; namespace eth: standard |
| go-stablenet/topology/01-proxied-pn-routing.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/topology/01-proxied-pn-routing.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/topology/01-proxied-pn-routing.json | `steps[1].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/topology/01-proxied-pn-routing.json | `steps[2].on` | role | `en1` | target node role (endpoint selection) |
| go-stablenet/tx/01-negative-tx-revert.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/vocabulary/01-derived-address-and-checksum.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/vocabulary/03-metric-head-block.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/vocabulary/03-metric-head-block.json | `steps[1].expect` | literal-string | `metric` | Prometheus metrics endpoint (launch --metrics) |
| go-stablenet/vocabulary/03-register-contract.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-stablenet/vocabulary/03-register-contract.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/vocabulary/03-register-contract.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-stablenet/vocabulary/03-register-contract.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-wbft/accounts/01-secp256r1-precompile-valid.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-wbft/accounts/01-secp256r1-precompile-valid.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/accounts/02-secp256r1-precompile-invalid.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-wbft/accounts/02-secp256r1-precompile-invalid.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/accounts/03-secp256r1-precompile-short-input.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| go-wbft/accounts/03-secp256r1-precompile-short-input.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/chain-up/01-wbft-chain-up.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/chain-up/01-wbft-chain-up.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/chain-up/01-wbft-chain-up.json | `steps[4].method` | literal-string | `net_peerCount` | RPC method; namespace net: standard |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `steps[4].method` | literal-string | `net_peerCount` | RPC method; namespace net: standard |
| go-wbft/consensus/01-e1-mixed-producers.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/consensus/01-e1-mixed-producers.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/consensus/01-e1-mixed-producers.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-wbft/fault/01-wbft-node-crash.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/fault/01-wbft-node-crash.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/fault/01-wbft-node-crash.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/fault/01-wbft-node-crash.json | `steps[1].on` | role | `node3` | target node role (endpoint selection) |
| go-wbft/fault/01-wbft-node-crash.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-wbft/fault/01-wbft-node-crash.json | `steps[3].on` | role | `node3` | target node role (endpoint selection) |
| go-wbft/network/01-wbft-proxied-routing.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[1].on` | role | `en1` | target node role (endpoint selection) |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[2].on` | role | `pn1` | target node role (endpoint selection) |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[3].on` | role | `bp1` | target node role (endpoint selection) |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[4].on` | role | `bp2` | target node role (endpoint selection) |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[5].onEach` | role | `bp1` | target node role (endpoint selection) |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[5].onEach` | role | `bp2` | target node role (endpoint selection) |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[5].onEach` | role | `pn1` | target node role (endpoint selection) |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[5].onEach` | role | `en1` | target node role (endpoint selection) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[5].on` | role | `node1` | target node role (endpoint selection) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[6].on` | role | `node1` | target node role (endpoint selection) |
| go-wbft/tx/02-wbft-insufficient-funds-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/tx/03-wbft-revert-status-zero.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wbft/tx/03-wbft-revert-status-zero.json | `steps[1].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-wbft/tx/03-wbft-revert-status-zero.json | `steps[4].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-wemix/chain-up/01-wemix-chain-up.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wemix/chain-up/01-wemix-chain-up.json | `steps[3].method` | literal-string | `net_peerCount` | RPC method; namespace net: standard |
| go-wemix/chain-up/02-wemix-chain-up-15.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wemix/chain-up/02-wemix-chain-up-15.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-wemix/chain-up/02-wemix-chain-up-15.json | `steps[3].method` | literal-string | `net_peerCount` | RPC method; namespace net: standard |
| go-wemix/fault/01-wemix-node-crash.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wemix/fault/01-wemix-node-crash.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| go-wemix/fault/01-wemix-node-crash.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| go-wemix/fault/01-wemix-node-crash.json | `steps[1].on` | role | `node3` | target node role (endpoint selection) |
| go-wemix/fault/01-wemix-node-crash.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-wemix/fault/01-wemix-node-crash.json | `steps[3].on` | role | `node3` | target node role (endpoint selection) |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `steps[2].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[5].on` | role | `node1` | target node role (endpoint selection) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[6].on` | role | `node1` | target node role (endpoint selection) |
| go-wemix/tx/02-wemix-insufficient-funds-rejected.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wemix/tx/03-wemix-revert-status-zero.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| go-wemix/tx/03-wemix-revert-status-zero.json | `steps[1].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| go-wemix/tx/03-wemix-revert-status-zero.json | `steps[4].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| remote/01-remote-rpc-health.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| remote/01-remote-rpc-health.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| remote/01-remote-rpc-health.json | `steps[1].method` | literal-string | `web3_clientVersion` | RPC method; namespace web3: standard |
| remote/01-remote-rpc-health.json | `steps[2].method` | literal-string | `net_peerCount` | RPC method; namespace net: standard |
| remote/02-remote-chain-info.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| remote/02-remote-chain-info.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| remote/02-remote-chain-info.json | `steps[1].method` | literal-string | `eth_syncing` | RPC method; namespace eth: standard |
| remote/02-remote-chain-info.json | `steps[2].method` | literal-string | `eth_getBlockByNumber` | RPC method; namespace eth: standard |
| remote/03-remote-balance-check.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| remote/03-remote-balance-check.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| remote/03-remote-balance-check.json | `steps[1].method` | literal-string | `eth_getBalance` | RPC method; namespace eth: standard |
| samples/01-sample-minimal.json | `env.capabilities[]` | literal-string | `rpc` | capability the env claims to provide |
| samples/01-sample-minimal.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| samples/01-sample-minimal.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| samples/01-sample-minimal.json | `steps[2].method` | literal-string | `eth_getTransactionReceipt` | RPC method; namespace eth: standard |
| samples/02-sample-lifecycle.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| samples/02-sample-lifecycle.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| samples/02-sample-lifecycle.json | `requires[]` | literal-string | `process` | capability the case needs (rpc/ws/process/consensus) |
| samples/02-sample-lifecycle.json | `steps[3].on` | role | `node1` | target node role (endpoint selection) |
| samples/02-sample-lifecycle.json | `steps[4].on` | role | `node1` | target node role (endpoint selection) |
| samples/02-sample-lifecycle.json | `steps[5].on` | role | `en1` | target node role (endpoint selection) |
| samples/02-sample-lifecycle.json | `steps[6].on` | role | `node1` | target node role (endpoint selection) |
| samples/02-sample-lifecycle.json | `steps[7].on` | role | `en1` | target node role (endpoint selection) |
| samples/02-sample-lifecycle.json | `steps[8].on` | role | `en1` | target node role (endpoint selection) |
| samples/02-sample-lifecycle.json | `steps[10].on` | role | `en1` | target node role (endpoint selection) |
| stress/01-stress-block-time.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| stress/01-stress-block-time.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| stress/01-stress-block-time.json | `steps[1].on` | role | `node1` | target node role (endpoint selection) |
| stress/02-stress-tx-flood.json | `requires[]` | literal-string | `rpc` | capability the case needs (rpc/ws/process/consensus) |
| stress/02-stress-tx-flood.json | `requires[]` | literal-string | `consensus` | capability the case needs (rpc/ws/process/consensus) |
| stress/02-stress-tx-flood.json | `steps[2].on` | role | `node1` | target node role (endpoint selection) |

### 컨트랙트 주소·calldata

| 파일 | 필드 | 값 종류 | 값 | 의미 |
|---|---|---|---|---|
| basic/05-basic-tx-send.json | `steps[2].to` | binding | `$r` | address bound from an earlier step (deployed contract or created account) |
| basic/06-basic-txpool-propagation.json | `steps[2].to` | binding | `$r` | address bound from an earlier step (deployed contract or created account) |
| fault/04-fault-p2p-topology.json | `steps[6].to` | binding | `$r` | address bound from an earlier step (deployed contract or created account) |
| fault/06-fault-txpool-leader-change.json | `steps[2].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[1].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[2].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[3].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[4].params[0].to` | literal-address | `0x0000000000000000000000000000000000000100` | SYSTEM CONTRACT: P256VERIFY precompile (RIP-7212) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[5].params[0].to` | literal-address | `0x0000000000000000000000000000000000000100` | SYSTEM CONTRACT: P256VERIFY precompile (RIP-7212) |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `steps[0].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `steps[1].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `steps[3].address` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `steps[4].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `steps[5].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[0].selector` | literal-hex | `0xb03d36cd` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[1].data` | binding | `$refBalCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[2].data` | literal-string | `0x64e2a8fc00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[4].selector` | literal-hex | `0xe0a8f6f5` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[5].data` | binding | `$cancelData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[6].data` | binding | `$refBalCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[0].selector` | literal-hex | `0xb03d36cd` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[1].data` | binding | `$refBalCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[2].data` | literal-string | `0x64e2a8fc00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[4].selector` | literal-hex | `0xc8541fe0` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[5].data` | binding | `$disData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[6].data` | binding | `$disData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[7].data` | binding | `$disData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[8].selector` | literal-hex | `0x013cf08b` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[9].data` | binding | `$proposalsCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[9].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[10].data` | binding | `$refBalCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[10].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[0].selector` | literal-hex | `0xb03d36cd` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[1].data` | binding | `$refBalCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[2].address` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[3].data` | literal-string | `0x64e2a8fc00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[4].address` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[9].selector` | literal-hex | `0xe1b526b0` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[10].data` | binding | `$expData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[10].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[12].selector` | literal-hex | `0x013cf08b` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[13].data` | binding | `$proposalsCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[13].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[14].data` | binding | `$refBalCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[14].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[20].address` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[0].selector` | literal-hex | `0xb03d36cd` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[1].data` | binding | `$refBalCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[2].data` | literal-string | `0x64e2a8fc00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[4].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[5].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[6].selector` | literal-hex | `0x013cf08b` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[7].data` | binding | `$proposalsCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[8].data` | binding | `$refBalCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[8].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[0].selector` | literal-hex | `0xb03d36cd` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[1].data` | binding | `$refBalCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[2].data` | literal-string | `0x64e2a8fc00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[4].selector` | literal-hex | `0xe0a8f6f5` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[5].data` | binding | `$cancelData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[6].data` | binding | `$refBalCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[7].data` | literal-hex | `0x936834b9` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[8].data` | binding | `$refBalCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[8].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `steps[2].data` | literal-hex | `0x936834b9` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[0].data` | literal-string | `0x64e2a8fc00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[2].selector` | literal-hex | `0xe0a8f6f5` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[3].data` | binding | `$cancelData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[4].data` | literal-hex | `0x936834b9` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[5].data` | literal-hex | `0x936834b9` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[2].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[6].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `steps[0].to` | literal-address | `0x00000000000000000000000000000000C0FFEE0E` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `steps[0].to` | literal-address | `0x00000000000000000000000000000000C0FFEE0E` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `steps[0].to` | literal-address | `0x00000000000000000000000000000000C0FFEE0E` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `steps[1].params[0]` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `steps[0].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `steps[1].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `steps[2].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `steps[3].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `steps[0].params[0]` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `steps[1].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `steps[2].params[0]` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[0].data` | literal-string | `0x64e2a8fc00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[2].selector` | literal-hex | `0xe0a8f6f5` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[3].data` | binding | `$cancelData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[5].data` | literal-hex | `0x936834b9` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[10].address` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[11].address` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[2].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[3].selector` | literal-hex | `0x93a8bb99` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[4].data` | binding | `$authData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[6].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[7].data` | binding | `$authApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `steps[0].to` | literal-address | `0x00000000000000000000000000000000C0FFEE11` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[2].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[3].selector` | literal-hex | `0x93a8bb99` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[4].data` | binding | `$authData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[6].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[7].data` | binding | `$authApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `steps[0].selector` | literal-hex | `0xfe9fbb80` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `steps[1].data` | binding | `$isAuthData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `steps[0].selector` | literal-hex | `0xfe575a87` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `steps[1].data` | binding | `$isBlData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `env.genesis.overlay.alloc.0x90F79bf6EB2c4f870365E785982E1f101E93b906.balance` | literal-hex | `0xDE0B6B3A7640000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `env.genesis.overlay.alloc.0x90F79bf6EB2c4f870365E785982E1f101E93b906.extra` | literal-hex | `0x4000000000000000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `env.genesis.overlay.alloc.0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65.balance` | literal-hex | `0xDE0B6B3A7640000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `env.genesis.overlay.alloc.0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65.extra` | literal-hex | `0x8000000000000000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `env.genesis.overlay.alloc.0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc.balance` | literal-hex | `0xDE0B6B3A7640000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `env.genesis.overlay.alloc.0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc.extra` | literal-hex | `0xc000000000000000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[1].selector` | literal-hex | `0xfe9fbb80` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[2].data` | binding | `$authCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[3].selector` | literal-hex | `0xfe575a87` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[4].data` | binding | `$blCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[5].selector` | literal-hex | `0xfe9fbb80` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[6].data` | binding | `$dualAuth` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[7].selector` | literal-hex | `0xfe575a87` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[8].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[8].data` | binding | `$dualBl` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `env.genesis.overlay.alloc.0x14dC79964da2C08b23698B3D3cc7Ca32193d9955.balance` | literal-hex | `0xDE0B6B3A7640000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `env.genesis.overlay.alloc.0x14dC79964da2C08b23698B3D3cc7Ca32193d9955.extra` | literal-hex | `0x4000000000000000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `env.genesis.overlay.alloc.0x90F79bf6EB2c4f870365E785982E1f101E93b906.balance` | literal-hex | `0xDE0B6B3A7640000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `env.genesis.overlay.alloc.0x90F79bf6EB2c4f870365E785982E1f101E93b906.extra` | literal-hex | `0x4000000000000000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `steps[0].selector` | literal-hex | `0xfe9fbb80` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `steps[1].selector` | literal-hex | `0xfe9fbb80` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `steps[2].selector` | literal-hex | `0xfe9fbb80` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `steps[3].data` | binding | `$allocOnly` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `steps[4].data` | binding | `$paramOnly` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `steps[5].data` | binding | `$bothSides` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `steps[6].data` | literal-hex | `0x6a6debd7` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `steps[0].selector` | literal-hex | `0xfe9fbb80` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `steps[1].selector` | literal-hex | `0xfe575a87` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `steps[2].data` | binding | `$isAuthData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `steps[3].data` | binding | `$isBlData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `env.genesis.overlay.alloc.0x90F79bf6EB2c4f870365E785982E1f101E93b906.balance` | literal-hex | `0xDE0B6B3A7640000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `env.genesis.overlay.alloc.0x90F79bf6EB2c4f870365E785982E1f101E93b906.extra` | literal-hex | `0x4000000000000000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `env.genesis.overlay.alloc.0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65.balance` | literal-hex | `0xDE0B6B3A7640000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `env.genesis.overlay.alloc.0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65.extra` | literal-hex | `0x8000000000000000` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[4].selector` | literal-hex | `0xfe9fbb80` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[5].selector` | literal-hex | `0xfe575a87` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[6].data` | binding | `$isAuthCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[7].data` | binding | `$isBlackCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | `steps[0].data` | literal-hex | `0x6a6debd7` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | `steps[0].data` | literal-hex | `0x6a6debd7` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | `steps[0].data` | literal-hex | `0x6a6debd7` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | `steps[0].data` | literal-hex | `0x6a6debd7` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | `steps[0].data` | literal-hex | `0x6a6debd7` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | `steps[0].data` | literal-hex | `0x6a6debd7` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[2].selector` | literal-hex | `0x93a8bb99` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[3].data` | binding | `$authData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[5].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[6].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[7].selector` | literal-hex | `0xfe9fbb80` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[8].data` | binding | `$isAuthData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[8].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `steps[4].to` | literal-address | `0x00000000000000000000000000000000C0FFEE0E` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `steps[4].to` | literal-address | `0x00000000000000000000000000000000C0FFEE0E` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `steps[6].to` | literal-address | `0x00000000000000000000000000000000C0FFEE0E` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `steps[0].to` | literal-address | `0x00000000000000000000000000000000C0FFEE05` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `steps[0].to` | literal-address | `0x00000000000000000000000000000000C0FFEE06` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/api/05-transaction-count-increments.json | `steps[1].to` | literal-address | `0x00000000000000000000000000000000C0FFEE04` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `steps[0].address` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `steps[1].address` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `steps[2].address` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `steps[3].address` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `steps[4].address` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/api/10-estimate-gas-token-transfer.json | `steps[0].data` | literal-string | `0xa9059cbb00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/api/10-estimate-gas-token-transfer.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/api/22-token-total-supply-readable.json | `steps[0].data` | literal-hex | `0x18160ddd` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/api/22-token-total-supply-readable.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `steps[0].data` | literal-string | `0x095ea7b300000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `steps[3].address` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `steps[4].data` | literal-string | `0xdd62ed3e00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[2].selector` | literal-hex | `0x0d321273` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[3].data` | binding | `$addBlData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[5].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[6].data` | binding | `$addApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[7].to` | literal-address | `0x00000000000000000000000000000000C0FFEE10` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[0].data` | literal-string | `0x0d32127300000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[2].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[3].data` | binding | `$addApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[4].data` | literal-string | `0xfe575a8700000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[5].to` | literal-address | `0x00000000000000000000000000000000C0FFEE32` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[6].data` | literal-string | `0x3d4c045200000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[8].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[9].data` | binding | `$rmApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[9].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[2].to` | binding | `$sender` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[3].to` | binding | `$feePayer` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[4].selector` | literal-hex | `0x0d321273` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[5].data` | binding | `$addBlData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[7].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[8].data` | binding | `$addApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[8].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[9].to` | literal-address | `0x00000000000000000000000000000000C0FFEE13` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[0].data` | literal-string | `0x0d32127300000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[2].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[3].data` | binding | `$addApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[4].data` | literal-string | `0x3d4c045200000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[6].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[7].data` | binding | `$rmApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[11].data` | literal-string | `0xfe575a8700000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[11].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[12].address` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000000000` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000000001` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000000100` | SYSTEM CONTRACT: P256VERIFY precompile (RIP-7212) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000b00001` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000b00002` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000b00003` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json | `steps[0].data` | literal-string | `0xfe575a8700000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json | `steps[0].data` | literal-string | `0xfe9fbb8000000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[2].selector` | literal-hex | `0x93a8bb99` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[3].data` | binding | `$authData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[5].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[6].data` | binding | `$authApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[5].to` | literal-address | `0x00000000000000000000000000000000C0FFEE03` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `steps[4].to` | literal-address | `0x00000000000000000000000000000000C0FFEE0C` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `steps[3].to` | literal-address | `0x00000000000000000000000000000000C0FFEE0D` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[2].to` | literal-address | `0x00000000000000000000000000000000C0FFEE10` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[3].to` | literal-address | `0x00000000000000000000000000000000C0FFEE10` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[4].to` | literal-address | `0x00000000000000000000000000000000C0FFEE10` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[2].to` | literal-address | `0x00000000000000000000000000000000C0FFEE11` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[3].to` | literal-address | `0x00000000000000000000000000000000C0FFEE11` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[4].to` | literal-address | `0x00000000000000000000000000000000C0FFEE11` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `steps[0].to` | literal-address | `0x00000000000000000000000000000000C0FFEE10` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `steps[2].to` | literal-address | `0x00000000000000000000000000000000C0FFEE07` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `steps[2].to` | literal-address | `0x00000000000000000000000000000000C0FFEE10` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `steps[0].to` | literal-address | `0x00000000000000000000000000000000C0FFEE0B` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[2].to` | literal-address | `0x00000000000000000000000000000000C0FFEE12` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[3].to` | literal-address | `0x00000000000000000000000000000000C0FFEE12` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[4].to` | literal-address | `0x00000000000000000000000000000000C0FFEE12` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[2].to` | literal-address | `0x00000000000000000000000000000000C0FFEE13` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[3].to` | literal-address | `0x00000000000000000000000000000000C0FFEE13` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[4].to` | literal-address | `0x00000000000000000000000000000000C0FFEE13` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[2].to` | binding | `$sponsor` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `steps[0].do` | literal-string | `deployContract` | node-signed asset helper |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `steps[0].bytecode` | literal-string | `0x600a600c600039600a6000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `steps[2].data` | literal-string | `0x` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `steps[2].to` | binding | `$contract` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/ethereum/22-estimate-gas.json | `steps[0].data` | literal-string | `0x` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/ethereum/22-estimate-gas.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000000001` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `steps[0].data` | literal-string | `0x6080604052348015600e57…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `steps[4].data` | literal-hex | `0xa9cc4718` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `steps[4].to` | binding | `$addr` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `steps[0].data` | literal-string | `0x600580600b6000396000f3…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `steps[2].to` | binding | `$addr` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `steps[0].data` | literal-string | `0x600480600b6000396000f3…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `steps[2].to` | binding | `$addr` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `steps[1].to` | literal-address | `0x00000000000000000000000000000000C0FFEE01` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[0].data` | literal-string | `0x6027600c60003960276000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[3].to` | binding | `$addr` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `steps[0].data` | literal-string | `0x6027600c60003960276000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `steps[2].to` | binding | `$addr` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[2].to` | binding | `$sender` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[3].to` | binding | `$feePayer` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[4].to` | literal-address | `0x00000000000000000000000000000000C0FFEE02` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `steps[2].to` | literal-address | `0x00000000000000000000000000000000C0FFEE20` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `steps[2].to` | literal-address | `0x00000000000000000000000000000000C0FFEE21` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `steps[2].to` | binding | `$sender` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `steps[2].to` | literal-address | `0x00000000000000000000000000000000C0FFEE22` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `steps[2].to` | literal-address | `0x00000000000000000000000000000000C0FFEE23` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `steps[2].to` | binding | `$sender` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `steps[3].to` | literal-address | `0x00000000000000000000000000000000C0FFEE10` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json | `steps[0].address` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `steps[0].data` | literal-string | `0xa9059cbb00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `steps[3].address` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `steps[4].address` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/02-token-balance-readable.json | `steps[0].data` | literal-string | `0x70a0823100000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/02-token-balance-readable.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/02-token-balance-readable.json | `steps[1].data` | literal-hex | `0x18160ddd` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/02-token-balance-readable.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[3].to` | binding | `$owner` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[4].to` | binding | `$spender` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[5].selector` | literal-hex | `0x095ea7b3` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[6].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[7].selector` | literal-hex | `0x23b872dd` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[8].data` | binding | `$transferFromData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[8].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[9].selector` | literal-hex | `0x70a08231` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[14].data` | binding | `$balOfRecipient` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[14].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[2].data` | literal-string | `0x1e5e042600000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[4].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[5].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[10].address` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `steps[0].data` | literal-string | `0x64e2a8fc00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `steps[3].address` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[2].data` | literal-string | `0x1e5e042600000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[4].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[5].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[0].data` | literal-string | `0x64e2a8fc00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[2].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[3].data` | binding | `$burnApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[4].selector` | literal-hex | `0x013cf08b` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[5].data` | binding | `$proposalsCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[0].data` | literal-string | `0x1e5e042600000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[2].selector` | literal-hex | `0x0d61b519` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[3].data` | binding | `$execData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[4].selector` | literal-hex | `0x013cf08b` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[5].data` | binding | `$proposalsCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[6].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[7].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000001003` | SYSTEM CONTRACT: GovMinter (stablenet) |
| go-stablenet/regression/system-contracts/09-validator-metadata-readable.json | `steps[0].data` | literal-hex | `0x5890ef79` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/09-validator-metadata-readable.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[2].selector` | literal-hex | `0xeeaf6816` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[3].data` | binding | `$propData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[5].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[6].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[8].selector` | literal-hex | `0xeeaf6816` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[9].data` | binding | `$restoreData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[9].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[11].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[12].data` | binding | `$approveData2` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[12].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[18].address` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[0].selector` | literal-hex | `0xeeaf6816` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[1].data` | binding | `$propData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[6].selector` | literal-hex | `0xe1b526b0` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[7].data` | binding | `$expData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[8].selector` | literal-hex | `0x013cf08b` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[9].data` | binding | `$proposalsCall` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[9].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[0].data` | literal-string | `0x898420a900000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[2].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[3].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[6].data` | literal-string | `0x8a6db9c300000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[0].data` | literal-string | `0x898420a900000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[2].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[3].data` | binding | `$cfgApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[4].data` | literal-string | `0xaa271e1a00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[5].data` | literal-string | `0x9336411700000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[7].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[8].data` | binding | `$rmApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[8].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[12].data` | literal-string | `0xaa271e1a00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[12].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[1].selector` | literal-hex | `0x5c646aa6` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[2].data` | binding | `$addData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[4].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[5].data` | binding | `$addApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[6].data` | literal-hex | `0x6ad89315` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[7].selector` | literal-hex | `0x85752d03` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[8].data` | binding | `$isMemData1` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[8].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[9].data` | literal-hex | `0x1703a018` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[9].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[10].selector` | literal-hex | `0xbfbd7f4c` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[11].data` | binding | `$rmData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[11].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[13].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[14].data` | binding | `$rmApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[14].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[15].data` | binding | `$rmApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[15].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[16].data` | literal-hex | `0x6ad89315` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[16].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[17].selector` | literal-hex | `0x85752d03` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[18].data` | binding | `$isMemData2` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[18].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[19].data` | literal-hex | `0x1703a018` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[19].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `steps[2].data` | literal-string | `0x898420a900000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000001002` | SYSTEM CONTRACT: GovMasterMinter (stablenet) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[0].data` | literal-string | `0x0d32127300000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[2].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[3].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[7].data` | literal-string | `0xfe575a8700000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[8].address` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[0].data` | literal-string | `0x93a8bb9900000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[2].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[3].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[6].data` | literal-string | `0xfe9fbb8000000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[6].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[0].data` | literal-string | `0x93a8bb9900000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[2].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[3].data` | binding | `$addApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[4].data` | literal-string | `0xcf44550e00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[6].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[7].data` | binding | `$rmApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[11].data` | literal-string | `0xfe9fbb8000000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[11].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[12].address` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `steps[1].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `steps[2].data` | literal-string | `0xf9f92be400000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `steps[2].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[0].data` | literal-string | `0x93a8bb9900000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[2].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[3].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[3].to` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[7].data` | literal-string | `0xfe9fbb8000000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000B00003` | reserved low address (precompile/system range) literal |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[8].address` | literal-address | `0x0000000000000000000000000000000000001004` | SYSTEM CONTRACT: GovCouncil (stablenet) |
| go-stablenet/regression/system-contracts/25-token-metadata.json | `steps[0].data` | literal-hex | `0x06fdde03` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/25-token-metadata.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/25-token-metadata.json | `steps[1].data` | literal-hex | `0x95d89b41` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/25-token-metadata.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/28-minter-status-readable.json | `steps[0].data` | literal-string | `0xaa271e1a00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/28-minter-status-readable.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/system-contracts/28-minter-status-readable.json | `steps[1].data` | literal-string | `0x8a6db9c300000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/system-contracts/28-minter-status-readable.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000001000` | SYSTEM CONTRACT: NativeCoinAdapter (stablenet) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[0].selector` | literal-hex | `0x5c646aa6` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[1].data` | binding | `$propData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[3].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[4].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[5].data` | literal-string | `0x08ae4b0c00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[4].selector` | literal-hex | `0x5c646aa6` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[5].data` | binding | `$addData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[7].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[8].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[8].data` | binding | `$approveData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[0].selector` | literal-hex | `0x5c646aa6` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[1].data` | binding | `$addData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[3].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[4].data` | binding | `$addApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[4].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[5].data` | literal-string | `0x08ae4b0c00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[5].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[6].selector` | literal-hex | `0xbfbd7f4c` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[7].data` | binding | `$rmData` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[7].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[9].selector` | literal-hex | `0x98951b56` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[10].data` | binding | `$rmApprove` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[10].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[11].data` | literal-string | `0x08ae4b0c00000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[11].to` | literal-address | `0x0000000000000000000000000000000000001001` | SYSTEM CONTRACT: GovValidator (stablenet) |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[1].do` | literal-string | `deployContract` | node-signed asset helper |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[1].bytecode` | literal-string | `0x600580600b6000396000f3…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[3].to` | binding | `$rv` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/vocabulary/01-derived-address-and-checksum.json | `steps[2].bytecode` | literal-hex | `0x6001` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `steps[2].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/vocabulary/03-register-contract.json | `steps[1].do` | literal-string | `deployContract` | node-signed asset helper |
| go-stablenet/vocabulary/03-register-contract.json | `steps[1].bytecode` | literal-string | `0x600a600c600039600a6000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-stablenet/vocabulary/03-register-contract.json | `steps[2].do` | literal-string | `registerContract` | node-signed asset helper |
| go-stablenet/vocabulary/03-register-contract.json | `steps[2].to` | binding | `$c` | address bound from an earlier step (deployed contract or created account) |
| go-stablenet/vocabulary/03-register-contract.json | `steps[2].data` | literal-hex | `0x01` | calldata/bytecode fixture (EVM revision dependent) |
| go-wbft/accounts/01-secp256r1-precompile-valid.json | `steps[0].data` | literal-string | `0x4fc05e7c6fa9dcfc4d09b2…` | calldata/bytecode fixture (EVM revision dependent) |
| go-wbft/accounts/01-secp256r1-precompile-valid.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000000100` | SYSTEM CONTRACT: P256VERIFY precompile (RIP-7212) |
| go-wbft/accounts/02-secp256r1-precompile-invalid.json | `steps[0].data` | literal-string | `0x0000000000000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-wbft/accounts/02-secp256r1-precompile-invalid.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000000100` | SYSTEM CONTRACT: P256VERIFY precompile (RIP-7212) |
| go-wbft/accounts/03-secp256r1-precompile-short-input.json | `steps[0].data` | literal-string | `0x0000000000000000000000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-wbft/accounts/03-secp256r1-precompile-short-input.json | `steps[0].to` | literal-address | `0x0000000000000000000000000000000000000100` | SYSTEM CONTRACT: P256VERIFY precompile (RIP-7212) |
| go-wbft/accounts/03-secp256r1-precompile-short-input.json | `steps[1].data` | literal-string | `0x` | calldata/bytecode fixture (EVM revision dependent) |
| go-wbft/accounts/03-secp256r1-precompile-short-input.json | `steps[1].to` | literal-address | `0x0000000000000000000000000000000000000100` | SYSTEM CONTRACT: P256VERIFY precompile (RIP-7212) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `env.genesis.overlay.alloc.0xc17d493883eaa3b4cceb0f214b273392d562f9d8.balance` | literal-hex | `0x3635c9adc5dea00000` | genesis overlay/set value |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[2].to` | binding | `$r` | address bound from an earlier step (deployed contract or created account) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[4].do` | literal-string | `deployContract` | node-signed asset helper |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[4].bytecode` | literal-string | `0x600a600c600039600a6000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[6].to` | binding | `$c` | address bound from an earlier step (deployed contract or created account) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[6].data` | literal-string | `0x` | calldata/bytecode fixture (EVM revision dependent) |
| go-wbft/tx/02-wbft-insufficient-funds-rejected.json | `steps[2].to` | literal-address | `0x00000000000000000000000000000000C0FFEE07` | reserved low address (precompile/system range) literal |
| go-wbft/tx/03-wbft-revert-status-zero.json | `steps[0].data` | literal-string | `0x600580600b6000396000f3…` | calldata/bytecode fixture (EVM revision dependent) |
| go-wbft/tx/03-wbft-revert-status-zero.json | `steps[3].to` | binding | `$addr` | address bound from an earlier step (deployed contract or created account) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `env.genesis.overlay.alloc.0xc17d493883eaa3b4cceb0f214b273392d562f9d8.balance` | literal-hex | `0x3635c9adc5dea00000` | genesis overlay/set value |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[2].to` | binding | `$r` | address bound from an earlier step (deployed contract or created account) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[4].do` | literal-string | `deployContract` | node-signed asset helper |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[4].bytecode` | literal-string | `0x600a600c600039600a6000…` | calldata/bytecode fixture (EVM revision dependent) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[6].to` | binding | `$c` | address bound from an earlier step (deployed contract or created account) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[6].data` | literal-string | `0x` | calldata/bytecode fixture (EVM revision dependent) |
| go-wemix/tx/02-wemix-insufficient-funds-rejected.json | `steps[2].to` | literal-address | `0x00000000000000000000000000000000C0FFEE07` | reserved low address (precompile/system range) literal |
| go-wemix/tx/03-wemix-revert-status-zero.json | `steps[0].data` | literal-string | `0x600580600b6000396000f3…` | calldata/bytecode fixture (EVM revision dependent) |
| go-wemix/tx/03-wemix-revert-status-zero.json | `steps[3].to` | binding | `$addr` | address bound from an earlier step (deployed contract or created account) |
| samples/02-sample-lifecycle.json | `steps[3].to` | binding | `$acct` | address bound from an earlier step (deployed contract or created account) |

### Chain ID·client family

| 파일 | 필드 | 값 종류 | 값 | 의미 |
|---|---|---|---|---|
| basic/01-basic-consensus.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| basic/02-basic-peers.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| basic/03-basic-rpc-health.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| basic/04-basic-sync.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| basic/05-basic-tx-send.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| basic/06-basic-txpool-propagation.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| basic/07-basic-wbft-consensus.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| fault/01-fault-network-partition.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| fault/02-fault-node-crash.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| fault/03-fault-node-recover.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| fault/04-fault-p2p-topology.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| fault/05-fault-two-down.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| fault/06-fault-txpool-leader-change.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json | `steps[1].is` | literal-number | `0` | expected eth_chainId value |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/anzeon/06-basefee-minimum.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/anzeon/06-basefee-minimum.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/anzeon/07-basefee-maximum.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/anzeon/07-basefee-maximum.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/01-block-transactions-field.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/02-block-by-hash-consistency.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/05-transaction-count-increments.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/05-transaction-count-increments.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/07-gas-price-positive.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/09-fee-history-well-formed.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/10-estimate-gas-token-transfer.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/10-estimate-gas-token-transfer.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/11-node-address-returned.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/11-node-address-returned.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/12-validator-set-nonempty.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/12-validator-set-nonempty.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/12b-validator-set-count.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/12b-validator-set-count.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/14-wbft-extra-info-fields.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/14-wbft-extra-info-fields.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/15-istanbul-status-fields.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/15-istanbul-status-fields.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/16-is-validator-flags.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/16-is-validator-flags.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/18-txpool-status.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/19-txpool-content-well-formed.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/20-admin-peers-populated.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/20-admin-peers-populated.json | `applicableChains` | literal-string | `stablenet,wbft,wemix` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/22-token-total-supply-readable.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/22-token-total-supply-readable.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/api/24-chain-not-syncing.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/01-stablenet-chain-up.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/01-stablenet-chain-up.json | `steps[1].is` | literal-number | `0` | expected eth_chainId value |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/22-estimate-gas.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/27-genesis-balance.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/27-genesis-balance.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/29-logs-query-well-formed.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/30-chain-id.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/30-chain-id.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/30-chain-id.json | `steps[1].is` | literal-number | `0` | expected eth_chainId value |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `steps[1].is` | literal-number | `0` | expected eth_chainId value |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/02-token-balance-readable.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/02-token-balance-readable.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/09-validator-metadata-readable.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/09-validator-metadata-readable.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/25-token-metadata.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/25-token-metadata.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/system-contracts/28-minter-status-readable.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/system-contracts/28-minter-status-readable.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| go-stablenet/regression/wbft/14-stablenet-gastip-field.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/regression/wbft/14-stablenet-gastip-field.json | `applicableChains` | literal-string | `stablenet` | chain allowlist filter (not a support proof) |
| go-stablenet/topology/01-proxied-pn-routing.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/tx/01-negative-tx-revert.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/vocabulary/01-derived-address-and-checksum.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/vocabulary/03-metric-head-block.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-stablenet/vocabulary/03-register-contract.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wbft/accounts/01-secp256r1-precompile-valid.json | `env.chain` | literal-string | `wbft` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wbft/accounts/01-secp256r1-precompile-valid.json | `applicableChains` | literal-string | `wbft` | chain allowlist filter (not a support proof) |
| go-wbft/accounts/02-secp256r1-precompile-invalid.json | `env.chain` | literal-string | `wbft` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wbft/accounts/02-secp256r1-precompile-invalid.json | `applicableChains` | literal-string | `wbft` | chain allowlist filter (not a support proof) |
| go-wbft/accounts/03-secp256r1-precompile-short-input.json | `env.chain` | literal-string | `wbft` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wbft/accounts/03-secp256r1-precompile-short-input.json | `applicableChains` | literal-string | `wbft` | chain allowlist filter (not a support proof) |
| go-wbft/chain-up/01-wbft-chain-up.json | `env.chain` | literal-string | `wbft` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wbft/chain-up/01-wbft-chain-up.json | `steps[1].is` | literal-number | `0` | expected eth_chainId value |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `env.chain` | literal-string | `wbft` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `steps[1].is` | literal-number | `0` | expected eth_chainId value |
| go-wbft/consensus/01-e1-mixed-producers.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wbft/fault/01-wbft-node-crash.json | `env.chain` | literal-string | `wbft` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wbft/network/01-wbft-proxied-routing.json | `env.chain` | literal-string | `wbft` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `env.chain` | literal-string | `wbft` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wbft/tx/02-wbft-insufficient-funds-rejected.json | `env.chain` | literal-string | `wbft` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wbft/tx/03-wbft-revert-status-zero.json | `env.chain` | literal-string | `wbft` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wemix/chain-up/01-wemix-chain-up.json | `env.chain` | literal-string | `wemix` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wemix/chain-up/01-wemix-chain-up.json | `steps[1].is` | literal-number | `0` | expected eth_chainId value |
| go-wemix/chain-up/02-wemix-chain-up-15.json | `env.chain` | literal-string | `wemix` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wemix/chain-up/02-wemix-chain-up-15.json | `steps[1].is` | literal-number | `0` | expected eth_chainId value |
| go-wemix/fault/01-wemix-node-crash.json | `env.chain` | literal-string | `wemix` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `env.chain` | literal-string | `wbft` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `env.chain` | literal-string | `wemix` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `env.chain` | literal-string | `wemix` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wemix/tx/02-wemix-insufficient-funds-rejected.json | `env.chain` | literal-string | `wemix` | client family selection (fixes manifest chain_id/network_id defaults) |
| go-wemix/tx/03-wemix-revert-status-zero.json | `env.chain` | literal-string | `wemix` | client family selection (fixes manifest chain_id/network_id defaults) |
| remote/01-remote-rpc-health.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| remote/01-remote-rpc-health.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| remote/02-remote-chain-info.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| remote/02-remote-chain-info.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| remote/02-remote-chain-info.json | `steps[0].is` | literal-number | `0` | expected eth_chainId value |
| remote/03-remote-balance-check.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| remote/03-remote-balance-check.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| samples/01-sample-minimal.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| samples/01-sample-minimal.json | `applicableChains` | literal-string | `stablenet,wbft` | chain allowlist filter (not a support proof) |
| samples/02-sample-lifecycle.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| stress/01-stress-block-time.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |
| stress/02-stress-tx-flood.json | `env.chain` | literal-string | `stablenet` | client family selection (fixes manifest chain_id/network_id defaults) |

### 포크·합의 전용 RPC

| 파일 | 필드 | 값 종류 | 값 | 의미 |
|---|---|---|---|---|
| basic/07-basic-wbft-consensus.json | `steps[1].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| basic/07-basic-wbft-consensus.json | `steps[2].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `env.genesis.overlay.config.bohoBlock` | literal-number | `3` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `env.genesis.overlay.config.boho.systemContracts.govMinter.address` | literal-address | `0x0000000000000000000000000000000000001003` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `env.genesis.overlay.config.boho.systemContracts.govMinter.version` | literal-string | `v2` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `env.hardforks.boho` | literal-number | `10` | fork activation height override |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `env.hardforks.boho` | literal-number | `10` | fork activation height override |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[4].do` | literal-string | `signAuthorization` | EIP-7702 set-code path (fork gated; unsupported on go-wemix) |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `steps[5].do` | literal-string | `signAuthorization` | EIP-7702 set-code path (fork gated; unsupported on go-wemix) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `steps[4].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `env.genesis.overlay.config.anzeon.systemContracts.govCouncil.params.authorizedAccounts` | literal-string | `0x976EA74026E726554dB657fA54763abd0C3a0aa9,0x14dC79964da2C08` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `steps[2].genesisOverlay` | literal-string | `{"alloc": {"0x90F79bf6EB2c4f870365E785982E1f101E93b906": {"b` | per-step genesis overlay (swapNode) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `env.genesis.overlay.config.bohoBlock` | literal-number | `6` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `env.genesis.overlay.config.bohoBlock` | literal-number | `6` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `env.genesis.overlay.config.boho.systemContracts.govMinter.address` | literal-address | `0x0000000000000000000000000000000000001003` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `env.genesis.overlay.config.boho.systemContracts.govMinter.version` | literal-string | `v99` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | `env.genesis.overlay.config.anzeon.systemContracts.govCouncil.params.authorizedAccounts` | literal-string | `0xaaa,0xbbb,0xccc` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | `env.genesis.overlay.config.anzeon.systemContracts.govCouncil.params.authorizedAccounts` | literal-string | `0xaaa, 0xbbb, 0xccc` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | `env.genesis.overlay.config.anzeon.systemContracts.govCouncil.params.authorizedAccounts` | literal-string | ` 0xaaa , 0xbbb ` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | `env.genesis.overlay.config.anzeon.systemContracts.govCouncil.params.authorizedAccounts` | literal-string | `0xaaa,,0xbbb` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | `env.genesis.overlay.config.anzeon.systemContracts.govCouncil.params.authorizedAccounts` | literal-hex | `0xaaa` | genesis overlay/set value |
| go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | `env.genesis.overlay.config.anzeon.systemContracts.govCouncil.params.authorizedAccounts` | literal-string | `` | genesis overlay/set value |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `steps[1].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `steps[7].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[10].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `steps[2].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `steps[2].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `steps[2].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | `steps[3].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | `steps[2].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/11-node-address-returned.json | `steps[0].method` | literal-string | `istanbul_nodeAddress` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/12-validator-set-nonempty.json | `steps[0].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/12-validator-set-nonempty.json | `steps[1].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/12b-validator-set-count.json | `steps[1].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `steps[0].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `steps[1].method` | literal-string | `istanbul_getCommitSignersFromBlock` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `steps[2].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/14-wbft-extra-info-fields.json | `steps[1].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/14-wbft-extra-info-fields.json | `steps[2].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/15-istanbul-status-fields.json | `steps[0].method` | literal-string | `istanbul_status` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/15-istanbul-status-fields.json | `steps[1].method` | literal-string | `istanbul_status` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/15-istanbul-status-fields.json | `steps[2].method` | literal-string | `istanbul_status` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/15-istanbul-status-fields.json | `steps[3].method` | literal-string | `istanbul_status` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/api/16-is-validator-flags.json | `steps[0].method` | literal-string | `istanbul_isValidator` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/ethereum/01-stablenet-chain-up.json | `steps[3].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[2].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `steps[2].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `env.genesis.overlay.config.applepieBlock` | literal-number | `0` | genesis overlay/set value |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[3].do` | literal-string | `sendSetCode` | EIP-7702 set-code path (fork gated; unsupported on go-wemix) |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `steps[3].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `env.genesis.overlay.config.applepieBlock` | literal-number | `0` | genesis overlay/set value |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `env.genesis.overlay.config.applepieBlock` | literal-number | `0` | genesis overlay/set value |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `env.genesis.overlay.config.applepieBlock` | literal-number | `0` | genesis overlay/set value |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `env.genesis.overlay.config.applepieBlock` | literal-number | `0` | genesis overlay/set value |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `env.genesis.overlay.config.applepieBlock` | literal-number | `0` | genesis overlay/set value |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `env.genesis.overlay.config.applepieBlock` | literal-number | `0` | genesis overlay/set value |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `env.genesis.overlay.config.applepieBlock` | literal-number | `0` | genesis overlay/set value |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[1].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[7].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[13].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `steps[1].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `steps[2].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `steps[1].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `env.genesis.overlay.config.anzeon.wbft.epochLength` | literal-number | `10` | genesis overlay/set value |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[1].method` | literal-string | `istanbul_nodeAddress` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[2].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[3].method` | literal-string | `istanbul_isValidator` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[11].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[12].method` | literal-string | `istanbul_isValidator` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `steps[2].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `steps[3].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | `steps[1].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-stablenet/regression/wbft/14-stablenet-gastip-field.json | `steps[1].method` | literal-string | `istanbul_getWbftExtraInfo` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-wbft/chain-up/01-wbft-chain-up.json | `steps[3].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `steps[3].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-wbft/consensus/01-e1-mixed-producers.json | `steps[1].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `steps[3].method` | literal-string | `istanbul_getValidators` | RPC method; namespace istanbul: wbft-family only (go-wbft, go-stablenet) |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `env.genesis.overlay.config.brioche.firstHalvingBlock` | literal-number | `10` | genesis overlay/set value |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `env.genesis.overlay.config.brioche.halvingPeriod` | literal-number | `10` | genesis overlay/set value |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `env.genesis.overlay.config.brioche.finishRewardBlock` | literal-number | `1000` | genesis overlay/set value |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `env.genesis.overlay.config.brioche.halvingTimes` | literal-number | `4` | genesis overlay/set value |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `env.genesis.overlay.config.brioche.halvingRate` | literal-number | `50` | genesis overlay/set value |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `steps[1].method` | literal-string | `wemix_getBriocheBlockReward` | RPC method; namespace wemix: wemix namespace (go-wemix, go-wbft) |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `steps[2].method` | literal-string | `wemix_getBriocheBlockReward` | RPC method; namespace wemix: wemix namespace (go-wemix, go-wbft) |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `steps[3].method` | literal-string | `wemix_getBriocheBlockReward` | RPC method; namespace wemix: wemix namespace (go-wemix, go-wbft) |
| samples/02-sample-lifecycle.json | `env.genesis.overlay.config.bohoBlock` | literal-number | `6` | genesis overlay/set value |

### 수수료·가스 입력

| 파일 | 필드 | 값 종류 | 값 | 의미 |
|---|---|---|---|---|
| basic/05-basic-tx-send.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| basic/06-basic-txpool-propagation.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| fault/04-fault-p2p-topology.json | `steps[6].value` | literal-number | `1` | fee/gas/value input literal |
| fault/06-fault-txpool-leader-change.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[2].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[2].value` | literal-hex | `0xde0b6b3a7640000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `steps[5].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[2].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[2].value` | literal-hex | `0xde0b6b3a7640000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[5].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[6].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `steps[7].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[3].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[3].value` | literal-hex | `0xde0b6b3a7640000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[10].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[2].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[2].value` | literal-hex | `0xde0b6b3a7640000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `steps[5].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[2].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[2].value` | literal-hex | `0xde0b6b3a7640000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[5].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `steps[7].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `steps[1].value` | literal-number | `100000000000000000000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `steps[2].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[0].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[0].value` | literal-hex | `0xde0b6b3a7640000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[3].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[4].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `steps[5].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `steps[0].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `steps[0].gasPrice` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `steps[0].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `steps[0].accessList` | literal-string | `[]` | EIP-2930 access list input |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `steps[0].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `steps[0].gasPrice` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `steps[0].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `steps[0].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `steps[0].maxFeePerGas` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `steps[0].maxPriorityFeePerGas` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `steps[0].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json | `steps[2].expect` | literal-string | `baseFee` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[0].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[0].value` | literal-hex | `0xde0b6b3a7640000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[3].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `steps[5].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[2].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[2].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[4].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[7].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[8].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `steps[0].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `steps[0].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[1].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[2].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[2].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[4].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[7].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[8].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[1].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `steps[2].reason` | literal-string | `genesis` | expected rejection reason (client error string; may differ per client) |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `steps[3].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `steps[3].maxFeePerGas` | literal-number | `1000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `steps[3].maxPriorityFeePerGas` | binding | `$hightip` | fee/gas/value bound from a step |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `steps[3].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[1].value` | literal-number | `30000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[3].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[6].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[12].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[12].maxFeePerGas` | literal-number | `1000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[12].maxPriorityFeePerGas` | binding | `$customTip` | fee/gas/value bound from a step |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[12].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `steps[1].source` | literal-string | `baseFee` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `steps[2].fillPercent` | literal-number | `25` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `steps[3].expect` | literal-string | `baseFee` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `steps[1].source` | literal-string | `baseFee` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `steps[2].fillPercent` | literal-number | `10` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `steps[3].expect` | literal-string | `baseFee` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `steps[1].fillPercent` | literal-number | `25` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `steps[2].source` | literal-string | `baseFee` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `steps[4].expect` | literal-string | `baseFee` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/anzeon/06-basefee-minimum.json | `steps[0].expect` | literal-string | `baseFee` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/anzeon/07-basefee-maximum.json | `steps[0].expect` | literal-string | `baseFee` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `steps[4].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `steps[4].maxFeePerGas` | binding | `$feecap` | fee/gas/value bound from a step |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `steps[4].maxPriorityFeePerGas` | binding | `$tip` | fee/gas/value bound from a step |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `steps[4].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `steps[4].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `steps[4].maxFeePerGas` | binding | `$feecap` | fee/gas/value bound from a step |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `steps[4].maxPriorityFeePerGas` | binding | `$tip` | fee/gas/value bound from a step |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `steps[4].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `steps[6].gas` | binding | `$over` | fee/gas/value bound from a step |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `steps[6].maxFeePerGas` | binding | `$feecap` | fee/gas/value bound from a step |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `steps[6].maxPriorityFeePerGas` | binding | `$tip` | fee/gas/value bound from a step |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `steps[6].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `steps[0].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `steps[0].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `steps[0].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `steps[0].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/api/05-transaction-count-increments.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/api/05-transaction-count-increments.json | `steps[1].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/api/07-gas-price-positive.json | `steps[0].expect` | literal-string | `gasPrice` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | `steps[0].source` | literal-string | `gasPrice` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/api/10-estimate-gas-token-transfer.json | `steps[0].expect` | literal-string | `estimateGas` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `steps[0].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[1].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[3].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[6].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[7].reason` | literal-string | `blacklist` | expected rejection reason (client error string; may differ per client) |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `steps[7].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[0].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[3].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[5].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[5].reason` | literal-string | `blacklist` | expected rejection reason (client error string; may differ per client) |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[5].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[6].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `steps[9].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[2].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[2].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[3].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[3].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[5].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[8].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[9].reason` | literal-string | `blacklist` | expected rejection reason (client error string; may differ per client) |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `steps[9].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[0].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[3].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[4].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `steps[7].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `steps[0].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `steps[0].reason` | literal-string | `zero` | expected rejection reason (client error string; may differ per client) |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `steps[0].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[0].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[0].reason` | literal-string | `precompile` | expected rejection reason (client error string; may differ per client) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[0].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[1].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[1].reason` | literal-string | `precompile` | expected rejection reason (client error string; may differ per client) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[1].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[2].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[2].reason` | literal-string | `precompile` | expected rejection reason (client error string; may differ per client) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[3].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[3].reason` | literal-string | `precompile` | expected rejection reason (client error string; may differ per client) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[3].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[4].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[4].reason` | literal-string | `precompile` | expected rejection reason (client error string; may differ per client) |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `steps[4].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[1].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[3].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[6].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `steps[7].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[5].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[5].gasPrice` | binding | `$gp` | fee/gas/value bound from a step |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `steps[5].value` | literal-number | `1000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `steps[4].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `steps[4].maxFeePerGas` | binding | `$feecap` | fee/gas/value bound from a step |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `steps[4].maxPriorityFeePerGas` | binding | `$tip` | fee/gas/value bound from a step |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `steps[4].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `steps[0].source` | literal-string | `baseFee` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `steps[2].params[0].value` | literal-hex | `0x1` | fee/gas input in RPC params |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `steps[3].accessList` | literal-string | `"$createdList"` | EIP-2930 access list input |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `steps[3].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `steps[3].gasPrice` | binding | `$gp` | fee/gas/value bound from a step |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `steps[3].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[1].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[2].maxFeePerGas` | literal-number | `100000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[2].maxPriorityFeePerGas` | literal-number | `30000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[3].maxFeePerGas` | literal-number | `100000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[3].maxPriorityFeePerGas` | literal-number | `30000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[3].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[4].maxFeePerGas` | literal-number | `100000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[4].maxPriorityFeePerGas` | literal-number | `30000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[4].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[1].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[2].maxFeePerGas` | literal-number | `100000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[2].maxPriorityFeePerGas` | literal-number | `30000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[3].maxFeePerGas` | literal-number | `100000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[3].maxPriorityFeePerGas` | literal-number | `30000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[3].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[4].maxFeePerGas` | literal-number | `100000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[4].maxPriorityFeePerGas` | literal-number | `30000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[4].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `steps[0].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `steps[0].maxFeePerGas` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `steps[0].maxPriorityFeePerGas` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `steps[0].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `steps[2].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `steps[2].value` | binding | `$over` | fee/gas/value bound from a step |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `steps[2].gas` | binding | `$over` | fee/gas/value bound from a step |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `steps[0].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `steps[0].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[1].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[2].maxFeePerGas` | literal-number | `100000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[2].maxPriorityFeePerGas` | literal-number | `30000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[3].maxFeePerGas` | literal-number | `120000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[3].maxPriorityFeePerGas` | literal-number | `36000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[3].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[4].maxFeePerGas` | literal-number | `100000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[4].maxPriorityFeePerGas` | literal-number | `30000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[4].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[1].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[2].maxFeePerGas` | literal-number | `100000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[2].maxPriorityFeePerGas` | literal-number | `30000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[3].maxFeePerGas` | literal-number | `120000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[3].maxPriorityFeePerGas` | literal-number | `36000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[3].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[4].maxFeePerGas` | literal-number | `100000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[4].maxPriorityFeePerGas` | literal-number | `30000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[4].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[2].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[2].value` | literal-number | `30000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `steps[0].gas` | literal-number | `200000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/22-estimate-gas.json | `steps[0].expect` | literal-string | `estimateGas` | fee observation whose expected value is chain-policy dependent |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `steps[0].gas` | literal-number | `300000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `steps[0].gas` | literal-number | `200000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `steps[2].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `steps[0].gas` | literal-number | `200000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `steps[2].gas` | literal-number | `50000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `steps[1].value` | literal-number | `1000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[0].gas` | literal-number | `200000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[3].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `steps[0].gas` | literal-number | `200000` | fee/gas/value input literal |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `steps[2].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[2].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[2].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[3].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[3].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `steps[4].value` | literal-number | `1000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `steps[1].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `steps[1].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `steps[2].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `steps[2].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `steps[3].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `steps[1].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `steps[1].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `steps[2].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `steps[2].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `steps[3].value` | literal-number | `1` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `steps[0].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[3].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[3].value` | literal-number | `5000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[4].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `steps[4].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[2].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `steps[5].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `steps[0].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `steps[0].value` | literal-hex | `0xde0b6b3a7640000` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[2].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `steps[5].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[0].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[0].value` | literal-hex | `0x38d7ea4c68000` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `steps[3].gas` | literal-hex | `0xb71b0` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[0].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[3].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `steps[7].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[3].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[6].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[9].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[12].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[1].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[7].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[0].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `steps[3].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[0].gas` | literal-hex | `0x7a120` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[3].gas` | literal-hex | `0x7a120` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[5].gas` | literal-hex | `0x7a120` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `steps[8].gas` | literal-hex | `0x7a120` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[2].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[5].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[11].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[14].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `steps[15].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `steps[1].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[0].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `steps[3].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[0].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `steps[3].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[0].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[3].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[4].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `steps[7].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `steps[1].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `steps[1].value` | literal-number | `10000000000000000000` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[0].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `steps[3].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[1].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `steps[4].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[5].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[8].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[1].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[4].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[7].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `steps[10].gas` | literal-hex | `0x16e360` | fee/gas/value input literal |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[1].gas` | literal-number | `300000` | fee/gas/value input literal |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[3].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `steps[2].amount` | literal-number | `1000000000000000000` | fee/gas/value input literal |
| go-stablenet/vocabulary/03-register-contract.json | `steps[1].gas` | literal-number | `300000` | fee/gas/value input literal |
| go-stablenet/vocabulary/03-register-contract.json | `steps[2].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[4].gas` | literal-number | `300000` | fee/gas/value input literal |
| go-wbft/tx/02-wbft-insufficient-funds-rejected.json | `steps[2].value` | binding | `$over` | fee/gas/value bound from a step |
| go-wbft/tx/02-wbft-insufficient-funds-rejected.json | `steps[2].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-wbft/tx/03-wbft-revert-status-zero.json | `steps[0].gas` | literal-number | `200000` | fee/gas/value input literal |
| go-wbft/tx/03-wbft-revert-status-zero.json | `steps[3].gas` | literal-number | `100000` | fee/gas/value input literal |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[2].value` | literal-number | `1` | fee/gas/value input literal |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[4].gas` | literal-number | `300000` | fee/gas/value input literal |
| go-wemix/tx/02-wemix-insufficient-funds-rejected.json | `steps[2].value` | binding | `$over` | fee/gas/value bound from a step |
| go-wemix/tx/02-wemix-insufficient-funds-rejected.json | `steps[2].gas` | literal-number | `21000` | fee/gas/value input literal |
| go-wemix/tx/03-wemix-revert-status-zero.json | `steps[0].gas` | literal-number | `200000` | fee/gas/value input literal |
| go-wemix/tx/03-wemix-revert-status-zero.json | `steps[3].gas` | literal-number | `100000` | fee/gas/value input literal |
| samples/01-sample-minimal.json | `steps[1].gas` | binding | `$transferGas` | fee/gas/value bound from a step |
| samples/01-sample-minimal.json | `steps[1].value` | literal-number | `1000000000000000000` | fee/gas/value input literal |
| samples/02-sample-lifecycle.json | `steps[3].value` | literal-number | `1000000000000000000` | fee/gas/value input literal |
| samples/02-sample-lifecycle.json | `steps[3].gas` | literal-number | `21000` | fee/gas/value input literal |
| stress/02-stress-tx-flood.json | `steps[1].fillPercent` | literal-number | `30` | fee/gas/value input literal |

### 토폴로지·시간·프로세스 제어

| 파일 | 필드 | 값 종류 | 값 | 의미 |
|---|---|---|---|---|
| basic/01-basic-consensus.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| basic/01-basic-consensus.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| basic/01-basic-consensus.json | `steps[0].target` | literal-number | `20` | timing/block-count constant (block period dependent) |
| basic/01-basic-consensus.json | `steps[0].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| basic/01-basic-consensus.json | `steps[1].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| basic/01-basic-consensus.json | `steps[1].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| basic/02-basic-peers.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| basic/02-basic-peers.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| basic/02-basic-peers.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| basic/02-basic-peers.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| basic/02-basic-peers.json | `steps[1].expect` | literal-string | `peerCount` | peer graph shape |
| basic/02-basic-peers.json | `steps[2].expect` | literal-string | `peerCount` | peer graph shape |
| basic/02-basic-peers.json | `steps[3].expect` | literal-string | `peerCount` | peer graph shape |
| basic/02-basic-peers.json | `steps[4].expect` | literal-string | `peerCount` | peer graph shape |
| basic/02-basic-peers.json | `steps[5].expect` | literal-string | `peerCount` | peer graph shape |
| basic/03-basic-rpc-health.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| basic/03-basic-rpc-health.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| basic/03-basic-rpc-health.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| basic/03-basic-rpc-health.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| basic/04-basic-sync.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| basic/04-basic-sync.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| basic/04-basic-sync.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| basic/04-basic-sync.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| basic/05-basic-tx-send.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| basic/05-basic-tx-send.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| basic/05-basic-tx-send.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| basic/05-basic-tx-send.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| basic/06-basic-txpool-propagation.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| basic/06-basic-txpool-propagation.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| basic/06-basic-txpool-propagation.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| basic/06-basic-txpool-propagation.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| basic/07-basic-wbft-consensus.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| basic/07-basic-wbft-consensus.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| basic/07-basic-wbft-consensus.json | `steps[0].target` | literal-number | `5` | timing/block-count constant (block period dependent) |
| basic/07-basic-wbft-consensus.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| fault/01-fault-network-partition.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| fault/01-fault-network-partition.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| fault/01-fault-network-partition.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| fault/01-fault-network-partition.json | `steps[2].do` | literal-string | `partition` | process/peer control (needs owned nodes) |
| fault/01-fault-network-partition.json | `steps[3].expect` | literal-string | `peerCount` | peer graph shape |
| fault/01-fault-network-partition.json | `steps[4].expect` | literal-string | `peerCount` | peer graph shape |
| fault/01-fault-network-partition.json | `steps[5].do` | literal-string | `healPartition` | process/peer control (needs owned nodes) |
| fault/01-fault-network-partition.json | `steps[6].target` | literal-number | `8` | timing/block-count constant (block period dependent) |
| fault/01-fault-network-partition.json | `steps[6].timeout` | literal-string | `150s` | timing/block-count constant (block period dependent) |
| fault/02-fault-node-crash.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| fault/02-fault-node-crash.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| fault/02-fault-node-crash.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| fault/02-fault-node-crash.json | `steps[2].do` | literal-string | `stopNode` | process/peer control (needs owned nodes) |
| fault/02-fault-node-crash.json | `steps[3].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| fault/02-fault-node-crash.json | `steps[3].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| fault/02-fault-node-crash.json | `steps[4].do` | literal-string | `restartNode` | process/peer control (needs owned nodes) |
| fault/02-fault-node-crash.json | `steps[5].target` | literal-number | `8` | timing/block-count constant (block period dependent) |
| fault/02-fault-node-crash.json | `steps[5].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| fault/03-fault-node-recover.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| fault/03-fault-node-recover.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| fault/03-fault-node-recover.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| fault/03-fault-node-recover.json | `steps[1].do` | literal-string | `stopNode` | process/peer control (needs owned nodes) |
| fault/03-fault-node-recover.json | `steps[2].target` | literal-number | `12` | timing/block-count constant (block period dependent) |
| fault/03-fault-node-recover.json | `steps[2].timeout` | literal-string | `150s` | timing/block-count constant (block period dependent) |
| fault/03-fault-node-recover.json | `steps[3].do` | literal-string | `restartNode` | process/peer control (needs owned nodes) |
| fault/03-fault-node-recover.json | `steps[4].target` | literal-number | `12` | timing/block-count constant (block period dependent) |
| fault/03-fault-node-recover.json | `steps[4].timeout` | literal-string | `150s` | timing/block-count constant (block period dependent) |
| fault/04-fault-p2p-topology.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| fault/04-fault-p2p-topology.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| fault/04-fault-p2p-topology.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| fault/04-fault-p2p-topology.json | `steps[2].do` | literal-string | `partition` | process/peer control (needs owned nodes) |
| fault/04-fault-p2p-topology.json | `steps[3].expect` | literal-string | `peerCount` | peer graph shape |
| fault/04-fault-p2p-topology.json | `steps[4].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| fault/04-fault-p2p-topology.json | `steps[4].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| fault/04-fault-p2p-topology.json | `steps[8].do` | literal-string | `healPartition` | process/peer control (needs owned nodes) |
| fault/04-fault-p2p-topology.json | `steps[9].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| fault/04-fault-p2p-topology.json | `steps[9].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| fault/05-fault-two-down.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| fault/05-fault-two-down.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| fault/05-fault-two-down.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| fault/05-fault-two-down.json | `steps[2].do` | literal-string | `stopNode` | process/peer control (needs owned nodes) |
| fault/05-fault-two-down.json | `steps[3].do` | literal-string | `stopNode` | process/peer control (needs owned nodes) |
| fault/05-fault-two-down.json | `steps[4].within` | literal-string | `12s` | timing/block-count constant (block period dependent) |
| fault/05-fault-two-down.json | `steps[4].maxAdvance` | literal-number | `1` | timing/block-count constant (block period dependent) |
| fault/05-fault-two-down.json | `steps[5].do` | literal-string | `startNode` | process/peer control (needs owned nodes) |
| fault/05-fault-two-down.json | `steps[6].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| fault/05-fault-two-down.json | `steps[6].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| fault/05-fault-two-down.json | `steps[7].do` | literal-string | `startNode` | process/peer control (needs owned nodes) |
| fault/06-fault-txpool-leader-change.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| fault/06-fault-txpool-leader-change.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| fault/06-fault-txpool-leader-change.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| fault/06-fault-txpool-leader-change.json | `steps[3].do` | literal-string | `stopNode` | process/peer control (needs owned nodes) |
| fault/06-fault-txpool-leader-change.json | `steps[4].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| fault/06-fault-txpool-leader-change.json | `steps[4].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| fault/06-fault-txpool-leader-change.json | `steps[6].do` | literal-string | `startNode` | process/peer control (needs owned nodes) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[0].target` | literal-number | `6` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `steps[2].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `steps[2].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[8].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `steps[8].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[0].pollInterval` | literal-string | `1s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[0].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[3].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `steps[3].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `steps[0].pollInterval` | literal-string | `1s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `steps[0].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `env.topology.syncMode` | literal-string | `snap` | node count/role layout |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[10].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `steps[10].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `env.topology.syncMode` | literal-string | `snap` | node count/role layout |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[3].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `steps[3].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `env.topology.syncMode` | literal-string | `snap` | node count/role layout |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[10].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `steps[10].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[0].target` | literal-number | `1` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `steps[0].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `steps[2].do` | literal-string | `swapNode` | process/peer control (needs owned nodes) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `steps[3].do` | literal-string | `readNodeLog` | process/peer control (needs owned nodes) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `steps[4].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `steps[4].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[3].target` | literal-number | `9` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `steps[3].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[6].do` | literal-string | `swapNode` | process/peer control (needs owned nodes) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[7].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[7].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[12].do` | literal-string | `readNodeLog` | process/peer control (needs owned nodes) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `steps[2].do` | literal-string | `swapNode` | process/peer control (needs owned nodes) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `steps[3].do` | literal-string | `readNodeLog` | process/peer control (needs owned nodes) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `steps[4].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `steps[4].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `steps[0].target` | literal-number | `5` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `steps[2].do` | literal-string | `readNodeLog` | process/peer control (needs owned nodes) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `steps[3].timeout` | literal-string | `20s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `steps[3].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[8].pollInterval` | literal-string | `1s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `steps[8].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `steps[0].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `steps[2].do` | literal-string | `load` | process/peer control (needs owned nodes) |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `steps[2].blocks` | literal-number | `6` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `steps[2].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `steps[0].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `steps[2].do` | literal-string | `load` | process/peer control (needs owned nodes) |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `steps[2].blocks` | literal-number | `6` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `steps[2].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `steps[0].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `steps[1].do` | literal-string | `load` | process/peer control (needs owned nodes) |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `steps[1].blocks` | literal-number | `6` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `steps[1].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `steps[3].target` | literal-number | `30` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `steps[3].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/anzeon/06-basefee-minimum.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/anzeon/07-basefee-maximum.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/01-block-transactions-field.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/02-block-by-hash-consistency.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/05-transaction-count-increments.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/07-gas-price-positive.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/09-fee-history-well-formed.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/10-estimate-gas-token-transfer.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/11-node-address-returned.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/12-validator-set-nonempty.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/12b-validator-set-count.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/12b-validator-set-count.json | `steps[0].pollInterval` | literal-string | `1s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/api/12b-validator-set-count.json | `steps[0].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/14-wbft-extra-info-fields.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/15-istanbul-status-fields.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/16-is-validator-flags.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/18-txpool-status.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/19-txpool-content-well-formed.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/20-admin-peers-populated.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/22-token-total-supply-readable.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/api/24-chain-not-syncing.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/01-stablenet-chain-up.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/01-stablenet-chain-up.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/01-stablenet-chain-up.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[2].wait` | literal-bool | `False` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[3].wait` | literal-bool | `False` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[4].wait` | literal-bool | `False` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[5].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `steps[5].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[2].wait` | literal-bool | `False` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[3].wait` | literal-bool | `False` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[4].wait` | literal-bool | `False` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[5].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `steps[5].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[2].wait` | literal-bool | `False` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[3].wait` | literal-bool | `False` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[4].wait` | literal-bool | `False` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[5].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `steps[5].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[2].wait` | literal-bool | `False` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[3].wait` | literal-bool | `False` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[4].wait` | literal-bool | `False` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[5].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `steps[5].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[4].pollInterval` | literal-string | `1s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `steps[4].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/22-estimate-gas.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `steps[2].pollInterval` | literal-string | `1s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `steps[2].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/27-genesis-balance.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/29-logs-query-well-formed.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/30-chain-id.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `steps[0].count` | literal-number | `1` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `steps[0].timeout` | literal-string | `30s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[6].count` | literal-number | `1` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `steps[6].timeout` | literal-string | `30s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `env.topology.bp` | literal-number | `13` | node count/role layout |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `env.topology.pn` | literal-number | `1` | node count/role layout |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `steps[0].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/02-token-balance-readable.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/09-validator-metadata-readable.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[7].pollInterval` | literal-string | `1s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[7].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[13].pollInterval` | literal-string | `1s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `steps[13].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[5].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `steps[5].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/25-token-metadata.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/system-contracts/28-minter-status-readable.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `steps[0].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `steps[0].target` | literal-number | `140` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `steps[0].timeout` | literal-string | `240s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[9].target` | literal-number | `25` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `steps[9].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `steps[0].pollInterval` | literal-string | `1s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `steps[0].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/regression/wbft/14-stablenet-gastip-field.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/topology/01-proxied-pn-routing.json | `env.topology.bp` | literal-number | `3` | node count/role layout |
| go-stablenet/topology/01-proxied-pn-routing.json | `env.topology.pn` | literal-number | `1` | node count/role layout |
| go-stablenet/topology/01-proxied-pn-routing.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| go-stablenet/topology/01-proxied-pn-routing.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-stablenet/topology/01-proxied-pn-routing.json | `steps[0].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| go-stablenet/topology/01-proxied-pn-routing.json | `steps[1].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/tx/01-negative-tx-revert.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-stablenet/tx/01-negative-tx-revert.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/vocabulary/01-derived-address-and-checksum.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/vocabulary/01-derived-address-and-checksum.json | `steps[0].target` | literal-number | `1` | timing/block-count constant (block period dependent) |
| go-stablenet/vocabulary/01-derived-address-and-checksum.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `steps[0].target` | literal-number | `1` | timing/block-count constant (block period dependent) |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `steps[2].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-stablenet/vocabulary/03-metric-head-block.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/vocabulary/03-metric-head-block.json | `env.launch.all.metrics` | literal-bool | `True` | node launch flag |
| go-stablenet/vocabulary/03-metric-head-block.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-stablenet/vocabulary/03-metric-head-block.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-stablenet/vocabulary/03-register-contract.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-stablenet/vocabulary/03-register-contract.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-stablenet/vocabulary/03-register-contract.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-wbft/accounts/01-secp256r1-precompile-valid.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wbft/accounts/02-secp256r1-precompile-invalid.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wbft/accounts/03-secp256r1-precompile-short-input.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wbft/chain-up/01-wbft-chain-up.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wbft/chain-up/01-wbft-chain-up.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-wbft/chain-up/01-wbft-chain-up.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `env.topology.bp` | literal-number | `13` | node count/role layout |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `env.topology.pn` | literal-number | `1` | node count/role layout |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `steps[0].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| go-wbft/consensus/01-e1-mixed-producers.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| go-wbft/consensus/01-e1-mixed-producers.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-wbft/consensus/01-e1-mixed-producers.json | `steps[2].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| go-wbft/consensus/01-e1-mixed-producers.json | `steps[2].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-wbft/fault/01-wbft-node-crash.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wbft/fault/01-wbft-node-crash.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| go-wbft/fault/01-wbft-node-crash.json | `steps[0].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| go-wbft/fault/01-wbft-node-crash.json | `steps[1].do` | literal-string | `stopNode` | process/peer control (needs owned nodes) |
| go-wbft/fault/01-wbft-node-crash.json | `steps[2].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| go-wbft/fault/01-wbft-node-crash.json | `steps[2].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| go-wbft/fault/01-wbft-node-crash.json | `steps[3].do` | literal-string | `restartNode` | process/peer control (needs owned nodes) |
| go-wbft/network/01-wbft-proxied-routing.json | `env.topology.bp` | literal-number | `2` | node count/role layout |
| go-wbft/network/01-wbft-proxied-routing.json | `env.topology.pn` | literal-number | `1` | node count/role layout |
| go-wbft/network/01-wbft-proxied-routing.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[0].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[1].expect` | literal-string | `peerCount` | peer graph shape |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[2].expect` | literal-string | `peerCount` | peer graph shape |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[3].expect` | literal-string | `peerCount` | peer graph shape |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[4].expect` | literal-string | `peerCount` | peer graph shape |
| go-wbft/network/01-wbft-proxied-routing.json | `steps[5].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| go-wbft/tx/01-wbft-tx-and-contract.json | `steps[0].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| go-wbft/tx/02-wbft-insufficient-funds-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wbft/tx/02-wbft-insufficient-funds-rejected.json | `steps[3].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| go-wbft/tx/03-wbft-revert-status-zero.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wbft/tx/03-wbft-revert-status-zero.json | `steps[5].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| go-wemix/chain-up/01-wemix-chain-up.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wemix/chain-up/01-wemix-chain-up.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-wemix/chain-up/01-wemix-chain-up.json | `steps[0].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| go-wemix/chain-up/02-wemix-chain-up-15.json | `env.topology.bp` | literal-number | `13` | node count/role layout |
| go-wemix/chain-up/02-wemix-chain-up-15.json | `env.topology.en` | literal-number | `2` | node count/role layout |
| go-wemix/chain-up/02-wemix-chain-up-15.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-wemix/chain-up/02-wemix-chain-up-15.json | `steps[0].timeout` | literal-string | `300s` | timing/block-count constant (block period dependent) |
| go-wemix/fault/01-wemix-node-crash.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wemix/fault/01-wemix-node-crash.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| go-wemix/fault/01-wemix-node-crash.json | `steps[0].timeout` | literal-string | `300s` | timing/block-count constant (block period dependent) |
| go-wemix/fault/01-wemix-node-crash.json | `steps[1].do` | literal-string | `stopNode` | process/peer control (needs owned nodes) |
| go-wemix/fault/01-wemix-node-crash.json | `steps[2].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| go-wemix/fault/01-wemix-node-crash.json | `steps[2].pollInterval` | literal-string | `3s` | timing/block-count constant (block period dependent) |
| go-wemix/fault/01-wemix-node-crash.json | `steps[3].do` | literal-string | `restartNode` | process/peer control (needs owned nodes) |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `steps[0].target` | literal-number | `22` | timing/block-count constant (block period dependent) |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `steps[0].timeout` | literal-string | `240s` | timing/block-count constant (block period dependent) |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `steps[0].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| go-wemix/tx/01-wemix-tx-and-contract.json | `steps[0].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| go-wemix/tx/02-wemix-insufficient-funds-rejected.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wemix/tx/02-wemix-insufficient-funds-rejected.json | `steps[3].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| go-wemix/tx/03-wemix-revert-status-zero.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| go-wemix/tx/03-wemix-revert-status-zero.json | `steps[5].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| remote/01-remote-rpc-health.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| remote/02-remote-chain-info.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| remote/03-remote-balance-check.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| samples/01-sample-minimal.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| samples/02-sample-lifecycle.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| samples/02-sample-lifecycle.json | `env.topology.en` | literal-number | `1` | node count/role layout |
| samples/02-sample-lifecycle.json | `steps[0].target` | literal-number | `3` | timing/block-count constant (block period dependent) |
| samples/02-sample-lifecycle.json | `steps[0].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| samples/02-sample-lifecycle.json | `steps[5].do` | literal-string | `stopNode` | process/peer control (needs owned nodes) |
| samples/02-sample-lifecycle.json | `steps[6].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| samples/02-sample-lifecycle.json | `steps[6].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |
| samples/02-sample-lifecycle.json | `steps[7].do` | literal-string | `startNode` | process/peer control (needs owned nodes) |
| samples/02-sample-lifecycle.json | `steps[8].target` | literal-number | `8` | timing/block-count constant (block period dependent) |
| samples/02-sample-lifecycle.json | `steps[8].timeout` | literal-string | `120s` | timing/block-count constant (block period dependent) |
| samples/02-sample-lifecycle.json | `steps[10].do` | literal-string | `readNodeLog` | process/peer control (needs owned nodes) |
| samples/02-sample-lifecycle.json | `steps[10].maxBytes` | literal-number | `4096` | timing/block-count constant (block period dependent) |
| stress/01-stress-block-time.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| stress/01-stress-block-time.json | `steps[0].target` | literal-number | `20` | timing/block-count constant (block period dependent) |
| stress/01-stress-block-time.json | `steps[0].timeout` | literal-string | `180s` | timing/block-count constant (block period dependent) |
| stress/01-stress-block-time.json | `steps[1].blocks` | literal-number | `15` | timing/block-count constant (block period dependent) |
| stress/01-stress-block-time.json | `steps[1].maxSeconds` | literal-number | `60` | timing/block-count constant (block period dependent) |
| stress/02-stress-tx-flood.json | `env.topology.bp` | literal-number | `4` | node count/role layout |
| stress/02-stress-tx-flood.json | `steps[0].target` | literal-number | `2` | timing/block-count constant (block period dependent) |
| stress/02-stress-tx-flood.json | `steps[0].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| stress/02-stress-tx-flood.json | `steps[1].do` | literal-string | `load` | process/peer control (needs owned nodes) |
| stress/02-stress-tx-flood.json | `steps[1].blocks` | literal-number | `15` | timing/block-count constant (block period dependent) |
| stress/02-stress-tx-flood.json | `steps[1].timeout` | literal-string | `90s` | timing/block-count constant (block period dependent) |
| stress/02-stress-tx-flood.json | `steps[2].timeout` | literal-string | `60s` | timing/block-count constant (block period dependent) |
| stress/02-stress-tx-flood.json | `steps[2].pollInterval` | literal-string | `2s` | timing/block-count constant (block period dependent) |

### 바이너리

| 파일 | 필드 | 값 종류 | 값 | 의미 |
|---|---|---|---|---|
| basic/01-basic-consensus.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| basic/02-basic-peers.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| basic/03-basic-rpc-health.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| basic/04-basic-sync.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| basic/05-basic-tx-send.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| basic/06-basic-txpool-propagation.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| basic/07-basic-wbft-consensus.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| fault/01-fault-network-partition.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| fault/02-fault-node-crash.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| fault/03-fault-node-recover.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| fault/04-fault-p2p-topology.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| fault/05-fault-two-down.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| fault/06-fault-txpool-leader-change.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `env.binaries.upgrade` | binding | `${GSTABLE_UPGRADE_BIN:-gstable}` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | `steps[6].binary` | literal-string | `upgrade` | binary swapped into a node |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `env.binaries.mismatch` | binding | `${GSTABLE_MISMATCH_BIN:-gstable-genesis-mismatch}` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | `steps[2].binary` | literal-string | `mismatch` | binary swapped into a node |
| go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/anzeon/06-basefee-minimum.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/regression/anzeon/07-basefee-maximum.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/regression/api/01-block-transactions-field.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/02-block-by-hash-consistency.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/03-transaction-by-hash-fields.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/04-transaction-receipt-fields.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/05-transaction-count-increments.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/06-system-contracts-deployed.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/07-gas-price-positive.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/regression/api/09-fee-history-well-formed.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/10-estimate-gas-token-transfer.json | `env.binaries.default` | literal-string | `go-stablenet` | binary name/path resolved by the composer |
| go-stablenet/regression/api/11-node-address-returned.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/12-validator-set-nonempty.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/12b-validator-set-count.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/13-commit-signers-quorum.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/14-wbft-extra-info-fields.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/15-istanbul-status-fields.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/16-is-validator-flags.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/18-txpool-status.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/19-txpool-content-well-formed.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/20-admin-peers-populated.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/22-token-total-supply-readable.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/23-token-approve-sets-allowance.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/api/24-chain-not-syncing.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/01-stablenet-chain-up.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/08-legacy-transfer.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/10-access-list-tx.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/11-nonce-ordering.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/16-effective-gas-price.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/17-replacement-tx.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/18-set-code-delegation.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/19-contract-roundtrip.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/22-estimate-gas.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/27-genesis-balance.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/27b-value-transfer.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/29-logs-query-well-formed.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/30-chain-id.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | `env.binaries.default` | literal-string | `/data/chainbench/bin/gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/ethereum/36-contract-event-emitted.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/02-token-balance-readable.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/04-mint-transfer-event.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/05-burn-transfer-event.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/09-validator-metadata-readable.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/13-remove-minter-executes.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/25-token-metadata.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/system-contracts/28-minter-status-readable.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/wbft/01-block-period-one-second.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/wbft/02-wbft-seals-quorum.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/wbft/04-validator-add-member-executes.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/wbft/05-validator-remove-member-executes.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/wbft/11-prev-seals-quorum.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/regression/wbft/14-stablenet-gastip-field.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/topology/01-proxied-pn-routing.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/tx/01-negative-tx-revert.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/vocabulary/01-derived-address-and-checksum.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/vocabulary/02-faucet-funds-account.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/vocabulary/03-metric-head-block.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-stablenet/vocabulary/03-register-contract.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-wbft/accounts/01-secp256r1-precompile-valid.json | `env.binaries.default` | literal-string | `gwbft` | binary name/path resolved by the composer |
| go-wbft/accounts/02-secp256r1-precompile-invalid.json | `env.binaries.default` | literal-string | `gwbft` | binary name/path resolved by the composer |
| go-wbft/accounts/03-secp256r1-precompile-short-input.json | `env.binaries.default` | literal-string | `gwbft` | binary name/path resolved by the composer |
| go-wbft/chain-up/01-wbft-chain-up.json | `env.binaries.default` | binding | `${GWBFT_BIN:-gwbft}` | binary name/path resolved by the composer |
| go-wbft/chain-up/02-wbft-chain-up-15.json | `env.binaries.default` | literal-string | `/data/chainbench/bin/gwbft` | binary name/path resolved by the composer |
| go-wbft/consensus/01-e1-mixed-producers.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| go-wbft/fault/01-wbft-node-crash.json | `env.binaries.default` | literal-string | `gwbft` | binary name/path resolved by the composer |
| go-wbft/network/01-wbft-proxied-routing.json | `env.binaries.default` | literal-string | `gwbft` | binary name/path resolved by the composer |
| go-wbft/tx/01-wbft-tx-and-contract.json | `env.binaries.default` | literal-string | `gwbft` | binary name/path resolved by the composer |
| go-wbft/tx/02-wbft-insufficient-funds-rejected.json | `env.binaries.default` | literal-string | `gwbft` | binary name/path resolved by the composer |
| go-wbft/tx/03-wbft-revert-status-zero.json | `env.binaries.default` | literal-string | `gwbft` | binary name/path resolved by the composer |
| go-wemix/chain-up/01-wemix-chain-up.json | `env.binaries.default` | binding | `${GWEMIX_BIN:-gwemix}` | binary name/path resolved by the composer |
| go-wemix/chain-up/02-wemix-chain-up-15.json | `env.binaries.default` | literal-string | `/data/chainbench/bin/gwemix` | binary name/path resolved by the composer |
| go-wemix/fault/01-wemix-node-crash.json | `env.binaries.default` | literal-string | `gwemix` | binary name/path resolved by the composer |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `env.binaries.producer` | binding | `${GWEMIX_BIN:-gwemix}` | binary name/path resolved by the composer |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `env.binaries.validator` | binding | `${GWBFT_BIN:-gwbft}` | binary name/path resolved by the composer |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `env.upgrade.profile` | literal-string | `profiles/wemix-upgrade.yaml` | mixed-binary handoff declaration |
| go-wemix/handoff/01-wemix-wbft-handoff.json | `env.upgrade.template` | binding | `${GOWEMIX_TEMPLATE}` | mixed-binary handoff declaration |
| go-wemix/rpc/01-wemix-brioche-block-reward.json | `env.binaries.default` | binding | `${GWEMIX_BIN:-gwemix}` | binary name/path resolved by the composer |
| go-wemix/tx/01-wemix-tx-and-contract.json | `env.binaries.default` | binding | `${GWEMIX_BIN:-gwemix}` | binary name/path resolved by the composer |
| go-wemix/tx/02-wemix-insufficient-funds-rejected.json | `env.binaries.default` | literal-string | `gwemix` | binary name/path resolved by the composer |
| go-wemix/tx/03-wemix-revert-status-zero.json | `env.binaries.default` | literal-string | `gwemix` | binary name/path resolved by the composer |
| remote/01-remote-rpc-health.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| remote/02-remote-chain-info.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| remote/03-remote-balance-check.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| samples/01-sample-minimal.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| samples/02-sample-lifecycle.json | `env.binaries.default` | binding | `${GSTABLE_BIN:-gstable}` | binary name/path resolved by the composer |
| stress/01-stress-block-time.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |
| stress/02-stress-tx-flood.json | `env.binaries.default` | literal-string | `gstable` | binary name/path resolved by the composer |

