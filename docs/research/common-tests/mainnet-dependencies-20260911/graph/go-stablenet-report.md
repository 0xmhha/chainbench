## Module sizes (top by LOC)

| module | files | loc | types | funcs | languages |
|---|---|---|---|---|---|
| core | 23 | 8145 | 44 | 278 | go |
| core/vm | 23 | 7511 | 67 | 338 | go |
| core/rawdb | 21 | 6773 | 30 | 369 | go |
| eth/downloader | 17 | 6273 | 36 | 168 | go |
| core/types | 31 | 6063 | 64 | 403 | go |
| trie | 16 | 5040 | 41 | 244 | go |
| eth/protocols/snap | 8 | 4803 | 44 | 106 | go |
| core/state/snapshot | 13 | 4598 | 30 | 147 | go |
| rpc | 19 | 4365 | 60 | 218 | go |
| accounts/usbwallet/trezor | 5 | 4089 | 49 | 540 | go |
| cmd/gstable | 11 | 4027 | 5 | 76 | go |
| eth | 16 | 3907 | 22 | 175 | go |
| crypto/bls12381 | 14 | 3906 | 20 | 250 | go |
| core/state | 11 | 3706 | 35 | 199 | go |
| p2p | 10 | 3378 | 39 | 142 | go |
| internal/ethapi | 6 | 3326 | 26 | 133 | go |
| cmd/utils | 5 | 3324 | 2 | 63 | go |
| core/txpool/legacypool | 4 | 3229 | 16 | 142 | go |
| consensus/wbft/core | 17 | 3202 | 16 | 106 | go |
| p2p/discover | 9 | 3173 | 26 | 157 | go |
| rlp | 8 | 3099 | 21 | 162 | go |
| metrics | 28 | 3003 | 47 | 236 | go |
| node | 11 | 2852 | 16 | 122 | go |
| triedb/pathdb | 10 | 2795 | 19 | 93 | go |
| accounts/abi | 14 | 2554 | 10 | 84 | go |

## Fan-in (most-imported internal modules)

| module | imported by N modules |
|---|---|
| log | 62 |
| core/types | 58 |
| common/math | 42 |
| rlp | 39 |
| crypto | 39 |
| params | 37 |
| common/hexutil | 35 |
| metrics | 30 |
| ethdb | 28 |
| rpc | 21 |
| core/rawdb | 21 |
| event | 21 |
| core/state | 20 |
| trie | 17 |
| consensus | 14 |
| common/lru | 11 |
| consensus/misc/eip1559 | 11 |
| accounts | 11 |
| p2p | 10 |
| core/vm | 10 |
| common/mclock | 10 |
| p2p/enode | 10 |
| consensus/wbft | 9 |
| p2p/enr | 9 |
| trie/trienode | 8 |

## Top supertypes / adopted interfaces

(no inheritance edges extracted — expected for Go/Rust-only trees)

## Totals

- modules: 129, files: 668, loc: 184640, types: 1528, type_edges: 0
