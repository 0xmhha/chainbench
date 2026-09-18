## Module sizes (top by LOC)

| module | files | loc | types | funcs | languages |
|---|---|---|---|---|---|
| wemix/bind | 8 | 23170 | 278 | 1549 | go |
| core | 29 | 11030 | 62 | 362 | go |
| les | 27 | 9788 | 111 | 404 | go |
| core/vm | 21 | 6277 | 52 | 265 | go |
| eth/downloader | 16 | 5975 | 37 | 171 | go |
| core/rawdb | 17 | 5501 | 26 | 302 | go |
| trie | 14 | 5140 | 36 | 241 | go |
| cmd/geth | 12 | 4887 | 10 | 88 | go |
| core/state/snapshot | 15 | 4667 | 35 | 168 | go |
| les/downloader | 10 | 4615 | 32 | 161 | go |
| core/types | 21 | 4247 | 51 | 293 | go |
| accounts/usbwallet/trezor | 5 | 4089 | 49 | 540 | go |
| eth/protocols/snap | 7 | 4043 | 37 | 91 | go |
| crypto/bls12381 | 14 | 3909 | 20 | 250 | go |
| metrics | 30 | 3859 | 61 | 400 | go |
| eth | 12 | 3752 | 29 | 186 | go |
| rpc | 18 | 3630 | 51 | 182 | go |
| wemix | 5 | 3562 | 11 | 101 | go |
| consensus/ethash | 7 | 3410 | 13 | 81 | go |
| cmd/utils | 6 | 3262 | 9 | 80 | go |
| eth/protocols/eth | 10 | 3152 | 61 | 151 | go |
| internal/ethapi | 6 | 3139 | 22 | 133 | go |
| p2p | 9 | 3133 | 41 | 139 | go |
| core/state | 10 | 2999 | 34 | 171 | go |
| p2p/discover | 7 | 2858 | 24 | 146 | go |

## Fan-in (most-imported internal modules)

| module | imported by N modules |
|---|---|
| common | 70 |
| log | 61 |
| core/types | 50 |
| crypto | 41 |
| common/math | 39 |
| rlp | 35 |
| common/hexutil | 33 |
| params | 30 |
| ethdb | 30 |
| metrics | 26 |
| rpc | 25 |
| event | 24 |
| core/rawdb | 21 |
| trie | 18 |
| core/state | 17 |
| p2p/enode | 17 |
| common/mclock | 16 |
| core | 16 |
| consensus | 15 |
| accounts | 14 |
| p2p/enr | 12 |
| core/vm | 11 |
| common/prque | 10 |
| node | 10 |
| p2p | 9 |

## Top supertypes / adopted interfaces

(no inheritance edges extracted — expected for Go/Rust-only trees)

## Totals

- modules: 120, files: 630, loc: 212568, types: 1936, type_edges: 0
