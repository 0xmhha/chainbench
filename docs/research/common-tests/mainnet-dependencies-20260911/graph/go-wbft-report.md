## Module sizes (top by LOC)

| module | files | loc | types | funcs | languages |
|---|---|---|---|---|---|
| wemixgov/bind | 8 | 23237 | 280 | 1551 | go |
| core | 24 | 8334 | 43 | 273 | go |
| core/vm | 22 | 6867 | 53 | 297 | go |
| core/rawdb | 21 | 6761 | 30 | 369 | go |
| eth/downloader | 17 | 6273 | 36 | 168 | go |
| core/types | 30 | 5868 | 61 | 387 | go |
| trie | 16 | 5029 | 41 | 244 | go |
| eth/protocols/snap | 8 | 4803 | 44 | 106 | go |
| core/state/snapshot | 13 | 4596 | 30 | 147 | go |
| rpc | 19 | 4365 | 60 | 218 | go |
| accounts/usbwallet/trezor | 5 | 4089 | 49 | 540 | go |
| eth | 17 | 4028 | 25 | 179 | go |
| crypto/bls12381 | 14 | 3906 | 20 | 250 | go |
| cmd/gwemix | 10 | 3824 | 4 | 71 | go |
| core/state | 11 | 3553 | 34 | 180 | go |
| p2p | 10 | 3378 | 39 | 142 | go |
| cmd/utils | 5 | 3324 | 2 | 63 | go |
| internal/ethapi | 6 | 3303 | 26 | 134 | go |
| core/txpool/legacypool | 4 | 3211 | 16 | 143 | go |
| consensus/wbft/core | 17 | 3201 | 16 | 106 | go |
| p2p/discover | 9 | 3173 | 26 | 157 | go |
| rlp | 8 | 3099 | 21 | 162 | go |
| metrics | 28 | 3003 | 47 | 236 | go |
| node | 11 | 2867 | 16 | 123 | go |
| triedb/pathdb | 10 | 2795 | 19 | 93 | go |

## Fan-in (most-imported internal modules)

| module | imported by N modules |
|---|---|
| log | 64 |
| core/types | 61 |
| common/math | 47 |
| crypto | 42 |
| params | 40 |
| rlp | 40 |
| common/hexutil | 36 |
| metrics | 30 |
| ethdb | 29 |
| event | 24 |
| rpc | 23 |
| core/rawdb | 22 |
| core/state | 21 |
| trie | 18 |
| consensus | 16 |
| accounts | 12 |
| p2p | 11 |
| common/lru | 11 |
| consensus/misc/eip1559 | 11 |
| p2p/enode | 11 |
| core/vm | 10 |
| consensus/wbft | 10 |
| common/mclock | 10 |
| p2p/enr | 9 |
| trie/trienode | 8 |

## Top supertypes / adopted interfaces

(no inheritance edges extracted — expected for Go/Rust-only trees)

## Totals

- modules: 135, files: 685, loc: 209312, types: 1824, type_edges: 0
